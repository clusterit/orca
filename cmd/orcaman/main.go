package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/config"
	configservice "github.com/clusterit/orca/config/service"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	"github.com/clusterit/orca/users"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/emicklei/go-restful.v1"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/oauth"
	uetcd "github.com/clusterit/orca/users/etcd"

	"github.com/spf13/cobra"
)

const (
	webRoot = "/remote/api/"
	cliRoot = "/api/"
)

var (
	etcdConfig   string
	etcdKey      string
	etcdCert     string
	etcdCa       string
	listen       string
	clilisten    string
	publish      string
	zone         string
	logger       = logging.Simple()
	revision     string
	root         = &cobra.Command{Use: "orcaman"}
	useweb       bool
	usecli       bool
	providerType string
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
		cc, cfger, err := connect(etcdConfig, etcdKey, etcdCert, etcdCa)
		if err != nil {
			panic(err)
		}
		m, err := newRest(cc, cfger, "", "")
		if err != nil {
			panic(err)
		}
		_, _, err = cmd.ForceZone(cfger, zone, true)
		if err != nil {
			panic(err)
		}
		m.setAdmins(args...)
	},
}

var provider = &cobra.Command{
	Use:   "provider [# network] [# clientid] [# clientsecret]",
	Short: "create a default provider network for oauth",
	Long:  "configure a default oauth provider for authentication",
	Run: func(cm *cobra.Command, args []string) {
		cc, cfger, err := connect(etcdConfig, etcdKey, etcdCert, etcdCa)
		if err != nil {
			panic(err)
		}
		m, err := newRest(cc, cfger, "", "")
		if err != nil {
			panic(err)
		}
		_, _, err = cmd.ForceZone(cfger, zone, true)
		if err != nil {
			panic(err)
		}
		if len(args) < 3 {
			cm.Help()
			os.Exit(1)
		}
		if reg, err := m.setProvider(providerType, args[0], args[1], args[2]); err != nil {
			panic(err)
		} else {
			spew.Dump(reg)
		}
	},
}

var serve = &cobra.Command{
	Use:   "serve",
	Short: "Starts the manager to listen on the given address",
	Long:  "Start the manager service on the given address. ",
	Run: func(*cobra.Command, []string) {
		serveAll()
	},
}

func serveAll() {
	var managers []*restmanager
	cc, cfger, err := connect(etcdConfig, etcdKey, etcdCert, etcdCa)
	if err != nil {
		panic(err)
	}
	var cmi, wm *restmanager
	if usecli {
		cmi, err = NewCli(cc, cfger, cmd.PublishAddress(publish, listen, cliRoot))
		if err != nil {
			panic(err)
		}
		managers = append(managers, cmi)
	}
	if useweb {
		wm, err = NewWeb(cc, cfger, cmd.PublishAddress(publish, listen, webRoot))
		if err != nil {
			panic(err)
		}
		managers = append(managers, wm)
	}
	go func() {
		if err := fetchZoneData(cfger, zone, managers); err != nil {
			panic(err)
		} else {
			// this should never happen, but if it happens, we end
			logger.Errorf("zone data fetcher ended, ending manager")
			os.Exit(1)
		}
	}()
	var wg sync.WaitGroup
	if clilisten == "" {
		wg.Add(1)
		go func() {
			start(listen, managers...)
			wg.Done()
		}()
	} else {
		wg.Add(1)
		go func() {
			start(listen, []*restmanager{wm}...)
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			start(clilisten, []*restmanager{cmi}...)
			wg.Done()
		}()
	}
	wg.Wait()
}

func connect(etcds string, etcdKey, etcdCert, etcdCaCert string) (*etcd.Cluster, config.Configer, error) {
	if etcds == "" {
		etcds = viper.GetString("etcd_machines")
	}
	if etcdKey == "" {
		etcdKey = viper.GetString("etcd_key")
	}
	if etcdCert == "" {
		etcdCert = viper.GetString("etcd_cert")
	}
	if etcdCa == "" {
		etcdCa = viper.GetString("etcd_ca")
	}
	cc, err := etcd.InitTLS(strings.Split(etcds, ","), etcdKey, etcdCert, etcdCaCert)
	if err != nil {
		return nil, nil, err
	}
	cfger, err := config.New(cc)
	if err != nil {
		return nil, nil, err
	}
	return cc, cfger, nil
}

func start(listenAddress string, rm ...*restmanager) error {
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

func fetchZoneData(cfger config.Configer, zone string, rms []*restmanager) error {
	ccf, stp, err := cfger.ClusterConfig()
	if err != nil {
		return err
	}
	for c := range ccf {
		logger.Debugf("new cluster config: Key:%s", c.Key)
		for _, rm := range rms {
			auth, err := rm.switchSettings(c, rm.oauthreg)
			if err == nil {
				rm.authimpl = auth
				if rm.usersService != nil {
					rm.usersService.Auth = auth
				}
				if rm.configService != nil {
					rm.configService.Auth = auth
				}
			}
		}
	}
	close(stp)
	return nil
}

type restmanager struct {
	publishUrl     string
	rootUrl        string
	cluster        *etcd.Cluster
	userimpl       users.Users
	authimpl       auth.Auther
	configer       config.Configer
	oauthreg       oauth.AuthRegistry
	autherService  *auth.AutherService
	configService  *configservice.ConfigService
	usersService   *users.UsersService
	wsContainer    *restful.Container
	authregService *oauth.AuthRegService

	initAuther         func(string, config.ClusterConfig, oauth.AuthRegistry) (auth.Auther, error)
	switchSettings     func(config.ClusterConfig, oauth.AuthRegistry) (auth.Auther, error)
	registerUrlMapping func(*http.ServeMux)
}

func newRest(cc *etcd.Cluster, cfg config.Configer, publishurl string, rooturl string) (*restmanager, error) {
	userimpl, err := uetcd.New(cc)
	if err != nil {
		return nil, err
	}

	oauther, err := oauth.New(cc)
	if err != nil {
		return nil, err
	}
	rm := &restmanager{cluster: cc,
		userimpl:   userimpl,
		oauthreg:   oauther,
		publishUrl: publishurl,
		configer:   cfg,
		rootUrl:    rooturl,
	}
	return rm, nil
}

func (rm *restmanager) initWithZone(zone string) error {
	_, clust, err := cmd.ForceZone(rm.configer, zone, true)
	if err != nil {
		return err
	}
	auth, err := rm.initAuther(zone, *clust, rm.oauthreg)
	if err != nil {
		return err
	}

	rm.authimpl = auth

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

	rm.usersService = &users.UsersService{Auth: rm.authimpl, Provider: rm.userimpl, Config: rm.configer}
	rm.usersService.Register(rootpath, c)

	rm.configService = &configservice.ConfigService{Auth: rm.authimpl, Users: rm.userimpl, Config: rm.configer, Zone: zone}
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
			if len(aliasedName) < 2 {
				fmt.Printf("prefix the useralias with the provider, aka: google:user.name@gmail.com, ignore: %s\n", m)
				continue
			}
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

func (rm *restmanager) setProvider(tp, network, clientid, clientsecret string) (*oauth.AuthRegistration, error) {
	return rm.oauthreg.Create(oauth.ProviderType(tp), network, clientid, clientsecret, "", "", "", "", "", "", "", "")
}

func main() {
	root.PersistentFlags().StringVarP(&etcdConfig, "etcd", "e", "", "etcd cluster machine Url's. if empty use env ORCA_ETCD_MACHINES which is by default http://localhost:4001")
	root.PersistentFlags().StringVar(&etcdKey, "etcdkey", "", "the client key for this etcd member if using TLS. if empty use ORCA_ETCD_KEY.")
	root.PersistentFlags().StringVar(&etcdCert, "etcdcert", "", "the client cert for this etcd member if using TLS. if empty use ORCA_ETCD_CERT.")
	root.PersistentFlags().StringVar(&etcdCa, "etcdca", "", "the ca for this etcd member if using TLS. if empty use ORCA_ETCD_CA.")
	root.PersistentFlags().StringVarP(&publish, "publish", "p", "self", "self published http address. if empty don't publish, the value 'self' will be replace with the currnent listen address")
	root.PersistentFlags().StringVarP(&zone, "zone", "z", "intranet", "use this zone as a subtree in the etcd backbone")
	root.PersistentFlags().StringVarP(&listen, "listen", "l", ":9011", "listen address for the endpoint")
	root.PersistentFlags().StringVar(&clilisten, "clilisten", "", "listen address for the cli endpoint. if empty use the 'listen' address")
	root.PersistentFlags().BoolVar(&useweb, "useweb", true, "start a web UI with oauth")
	root.PersistentFlags().BoolVar(&usecli, "usecli", true, "start a CLI with token auth")
	provider.Flags().StringVar(&providerType, "providertype", "oauth", "type of the new provider")

	root.AddCommand(cmdAdmins, versionCmd, serve, provider)
	viper.SetEnvPrefix("orca")
	viper.SetDefault("etcd_machines", "http://localhost:4001")
	viper.AutomaticEnv()
	root.Execute()
}
