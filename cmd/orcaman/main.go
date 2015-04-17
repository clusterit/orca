package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	"github.com/clusterit/orca/users"
	"gopkg.in/emicklei/go-restful.v1"

	"github.com/clusterit/orca/auth"
	_ "github.com/clusterit/orca/auth/google"
	"github.com/clusterit/orca/auth/oauth"
	uetcd "github.com/clusterit/orca/users/etcd"

	"github.com/spf13/cobra"
)

const (
	webRoot = "/remote/api/"
	cliRoot = "/api/"
)

var (
	etcdConfig string
	listen     string
	publish    string
	zone       string
	logger     = logging.Simple()
	revision   string
	root       = &cobra.Command{Use: "orcaman"}
	useweb     bool
	usecli     bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Orca",
	Long:  `Orca's build version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Orca service manager, revision '%s'\n", revision)
	},
}

var cmdAdmins = &cobra.Command{
	Use:   "admins [# admin-ids ...]",
	Short: "Set the admin userids",
	Long:  "Set the admin userids in the configuration backend to enable bootstrapping",
	Run: func(cm *cobra.Command, args []string) {
		m, err := newRest(strings.Split(etcdConfig, ","), "", "")
		if err != nil {
			panic(err)
		}
		m.setAdmins(args...)
	},
}

var serve = &cobra.Command{
	Use:   "serve",
	Short: "Starts the manager to listen on the given address",
	Long:  "Start the manager service on the given address. ",
	Run: func(cm *cobra.Command, args []string) {
		var managers []*restmanager
		log.Printf("%v, %v", usecli, useweb)
		if usecli {
			cmi, err := NewCli(strings.Split(etcdConfig, ","), cmd.PublishAddress(publish, listen, cliRoot))
			if err != nil {
				panic(err)
			}
			managers = append(managers, cmi)
		}
		if useweb {
			wm, err := NewWeb(strings.Split(etcdConfig, ","), cmd.PublishAddress(publish, listen, webRoot))
			if err != nil {
				panic(err)
			}
			managers = append(managers, wm)
		}
		start(listen, managers)
	},
}

func start(listenAddress string, rm []*restmanager) error {
	mux := http.NewServeMux()
	srv := http.Server{
		Addr:    listenAddress,
		Handler: mux,
	}
	for _, r := range rm {
		if err := r.initWithZone(zone); err == nil {
			r.serveAndPublish(mux)
		} else {
			logger.Errorf("cannot init %s: %s", r.rootUrl, err)
		}
	}
	logger.Infof("start listening on %s", srv.Addr)
	// todo: add TLS
	return srv.ListenAndServe()

}

type restmanager struct {
	publishUrl     string
	rootUrl        string
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

	initAuther         func(string, config.ManagerConfig, oauth.OAuthRegistry) (auth.Auther, error)
	switchSettings     func(config.ManagerConfig, oauth.OAuthRegistry) (auth.Auther, error)
	registerUrlMapping func(*http.ServeMux)
}

func newRest(etcds []string, publishurl string, rooturl string) (*restmanager, error) {
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
	oauther, err := oauth.New(cc)
	if err != nil {
		return nil, err
	}
	rm := &restmanager{cluster: cc,
		configer:   cfger,
		userimpl:   userimpl,
		oauthreg:   oauther,
		publishUrl: publishurl,
		rootUrl:    rooturl,
	}
	return rm, nil
}

func (rm *restmanager) initWithZone(zone string) error {
	_, cfg, err := cmd.ForceZone(rm.configer, zone, true, true)
	if err != nil {
		return err
	}
	auth, err := rm.initAuther(zone, *cfg, rm.oauthreg)
	if err != nil {
		return err
	}

	rm.authimpl = auth

	go func() {
		mgr, stp, err := rm.configer.ManagerConfig(zone)
		if err != nil {
			logger.Errorf("cannot create watcher for manger config config: %s", err)
			return
		}
		for m := range mgr {
			logger.Debugf("new manager config: Key:%s", m.Key)
			auth, err := rm.switchSettings(m, rm.oauthreg)
			if err == nil {
				rm.authimpl = auth
				rm.usersService.Auth = auth
				rm.configService.Auth = auth
			}
		}
		close(stp)
	}()
	return nil
}

func (rm *restmanager) stop() {
	rm.autherService.Shutdown()
	rm.usersService.Shutdown()
	rm.configService.Shutdown()
	rm.authregService.Shutdown()
}

func (rm *restmanager) register(rootpath string) *restful.Container {
	c := restful.NewContainer()
	rm.autherService = &auth.AutherService{Auth: rm.authimpl}
	rm.autherService.Register(rootpath, c)

	rm.usersService = &users.UsersService{Auth: rm.authimpl, Provider: rm.userimpl}
	rm.usersService.Register(rootpath, c)

	rm.configService = &config.ConfigService{Auth: rm.authimpl, Users: rm.userimpl, Config: rm.configer, Zone: zone}
	rm.configService.Register(rootpath, c)

	rm.authregService = &oauth.AuthRegService{Auth: rm.authimpl, Users: rm.userimpl, Registry: rm.oauthreg}
	rm.authregService.Register(rootpath, c)

	rm.wsContainer = c
	return c
	//rm.ServeAndPublish(rootpath)
}

func (rm *restmanager) serveAndPublish(mux *http.ServeMux) {
	c := rm.register(rm.rootUrl)
	mux.Handle(rm.rootUrl, c)
	rm.registerUrlMapping(mux)
	man, e := rm.cluster.NewManager()
	if e != nil {
		panic(e)
	}
	if publish != "" {
		man.Register("/"+cmd.ManagerService, rm.publishUrl, 20)
	}
}

func (rm *restmanager) setAdmins(admins ...string) {
	if len(admins) > 0 {
		for _, m := range admins {
			aliasedName := strings.Split(m, ":")
			u, e := rm.userimpl.Get(m)
			if e != nil {
				_, err := rm.userimpl.Create(aliasedName[0], aliasedName[1], aliasedName[1], users.ManagerRoles)
				if err != nil {
					logger.Errorf("cannot create manager user '%s': %s", m, err)
					continue
				}
			} else {
				if _, err := rm.userimpl.Update(u.Id, u.Name, users.ManagerRoles); err != nil {
					logger.Errorf("cannot update manager user '%s': %s", u.Id, err)
				}
			}
		}
	}
}

func main() {
	root.PersistentFlags().StringVarP(&etcdConfig, "etcd", "e", "http://localhost:4001", "etcd cluster machine Url's")
	root.PersistentFlags().StringVarP(&publish, "publish", "p", "self", "self published http address. if empty don't publish, the value 'self' will be replace with the currnent listen address")
	root.PersistentFlags().StringVarP(&zone, "zone", "z", "intranet", "use this zone as a subtree in the etcd backbone")
	root.PersistentFlags().StringVarP(&listen, "listen", "l", ":9011", "listen address for the endpoint")
	root.PersistentFlags().BoolVar(&useweb, "useweb", true, "start a web UI with oauth")
	root.PersistentFlags().BoolVar(&usecli, "usecli", true, "start a CLI with basic auth")
	root.AddCommand(cmdAdmins, versionCmd, serve)
	root.Execute()
}
