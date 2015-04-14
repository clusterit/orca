package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/basic"
	"github.com/clusterit/orca/auth/oauth"
	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	"github.com/clusterit/orca/users"
	uetcd "github.com/clusterit/orca/users/etcd"

	"gopkg.in/emicklei/go-restful.v1"
)

const (
	rootPath = "/api"
)

// options
var (
	etcdConfig string
	listen     string
	authUrl    string
	publish    string
	zone       string
	verifyCert bool
	managers   uids
	logger     = logging.Simple()
	revision   string
)

type uids []string

func (u *uids) String() string {
	return strings.Join(*u, ",")
}

func (u *uids) Type() string {
	return "uids"
}

func (u *uids) Set(value string) error {
	if len(*u) > 0 {
		return errors.New("uids flag already set")
	}
	for _, usr := range strings.Split(value, ",") {
		*u = append(*u, usr)
	}
	return nil
}

type climanager struct {
	cluster        *etcd.Cluster
	userimpl       users.Users
	authimpl       auth.Auther
	configer       config.Configer
	oauthreg       oauth.OAuthRegistry
	autherService  *auth.AutherService
	configService  *config.ConfigService
	usersService   *users.UsersService
	wsContainer    *restful.Container
	authregService *oauth.AuthRegService
}

func (cm *climanager) Stop() {
	cm.autherService.Shutdown()
	cm.usersService.Shutdown()
}

func (cm *climanager) Start() {
	c := restful.NewContainer()
	cm.autherService = &auth.AutherService{Auth: cm.authimpl}
	cm.autherService.Register(rootPath, c)

	cm.usersService = &users.UsersService{Auth: cm.authimpl, Provider: cm.userimpl}
	cm.usersService.Register(rootPath, c)

	cm.configService = &config.ConfigService{Auth: cm.authimpl, Users: cm.userimpl, Config: cm.configer, Zone: zone}
	cm.configService.Register(rootPath, c)

	cm.authregService = &oauth.AuthRegService{Auth: cm.authimpl, Users: cm.userimpl, Registry: cm.oauthreg}
	cm.authregService.Register(rootPath, c)

	cm.wsContainer = c
	cm.ServeAndPublish()
}

func (cm *climanager) initWithZone(zone string) error {
	_, cfg, err := cmd.ForceZone(cm.configer, zone, true, true)
	if err != nil {
		return err
	}
	cm.authimpl = basic.NewAuther(cfg.AuthUrl, cfg.VerifyCert)

	go func() {
		mgr, stp, err := cm.configer.ManagerConfig(zone)
		if err != nil {
			logger.Errorf("cannot create watcher for manger config config: %s", err)
			return
		}
		for m := range mgr {
			logger.Debugf("new manager config: AuthURL:%s, VerifyCert: %v", m.AuthUrl, m.VerifyCert)
			cm.switchSettings(m)
		}
		close(stp)
	}()
	return nil
}

func (cm *climanager) switchSettings(cfg config.ManagerConfig) {
	cm.authimpl = basic.NewAuther(cfg.AuthUrl, cfg.VerifyCert)
	cm.usersService.Auth = cm.authimpl
	cm.configService.Auth = cm.authimpl
	cm.authregService.Auth = cm.authimpl
}

func (cm *climanager) ServeAndPublish() {
	man, e := cm.cluster.NewManager()
	if e != nil {
		panic(e)
	}
	if publish != "" {
		man.Register("/"+cmd.ManagerService, publish, 20)
	}
	logger.Infof("start listening on %s", listen)
	server := &http.Server{Addr: listen, Handler: cm.wsContainer}
	logger.Errorf("%s", server.ListenAndServe())
}

func NewCLIManager(etcds []string) (*climanager, error) {
	cc, err := etcd.Init(etcds)
	if err != nil {
		return nil, err
	}
	userimpl, err := uetcd.New(cc)
	if err != nil {
		return nil, err
	}

	cfger, err := config.New(cc)
	if err != nil {
		return nil, err
	}

	reg, err := oauth.New(cc)
	if err != nil {
		return nil, err
	}

	if len(managers) > 0 {
		for _, m := range managers {
			u, e := userimpl.Get(m)
			if e != nil {
				_, err := userimpl.Create(m, m, users.ManagerRoles)
				if err != nil {
					logger.Errorf("cannot create manager user '%s': %s", m, err)
					continue
				}
			} else {
				if _, err := userimpl.Update(u.Id, u.Name, users.ManagerRoles); err != nil {
					logger.Errorf("cannot update manager user '%s': %s", u.Id, err)
				}
			}
		}
	}
	climan := &climanager{cluster: cc,
		userimpl: userimpl,
		configer: cfger,
		oauthreg: reg,
	}
	climan.initWithZone(zone)
	return climan, nil
}

func main() {
	climan.Execute()
}
