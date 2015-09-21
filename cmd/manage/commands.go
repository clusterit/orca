package main

import (
	"fmt"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/clusterit/orca/user"
	"github.com/spf13/cobra"
)

var (
	bind     string
	port     int
	etcds    string
	etcdKey  string
	etcdCert string
	etcdCa   string
)

func checkAndFail(condition bool, msg string, par ...interface{}) {
	if !condition {
		fmt.Printf(msg+"\n", par...)
		os.Exit(1)
	}
}

func checkErrorAndFail(e error, par ...interface{}) {
	if e != nil {
		fmt.Printf(e.Error()+"\n", par...)
		os.Exit(1)
	}
}

var (
	serve = &cobra.Command{
		Use:   "serve",
		Short: "Start the web listener for the endpoint an web UI",
		Long:  "Starts the web listener on the address specified by the options. Note that this is not the same as the publish address.",
		Run: func(cmd *cobra.Command, args []string) {
			listenaddress := fmt.Sprintf("%s:%d", bind, port)
			log.Error(start(listenaddress))
		},
	}

	userCommand = &cobra.Command{Use: "user"}
	useradd     = &cobra.Command{
		Use:   "add [#userid] [#network] [#fullname]",
		Short: "Add a new user to the backend",
		Long:  "Add a new user to the backend. The user must have an alias and a network, aka 'user@gmail.com@google'.",
		Run: func(cmd *cobra.Command, args []string) {
			checkAndFail(len(args) > 2, "you must specify a userid, a network and a fullname")
			uid := args[0]
			netw := args[1]
			name := args[2]
			backend := mustServiceBackend()
			_, e := backend.Create(netw, uid, name, user.ManagerRole)
			checkErrorAndFail(e)
		},
	}
)

func init() {
	serve.Flags().StringVarP(&bind, "bind", "b", "127.0.0.1", "bind address for the endpoint")
	serve.Flags().IntVarP(&port, "port", "p", 9011, "bin port for the endpoint")
	serve.Flags().StringVarP(&etcds, "etcd", "e", "", "etcd cluster machine Url's. if empty use env ORCA_ETCD_MACHINES which is by default http://localhost:4001")
	serve.Flags().StringVar(&etcdKey, "etcdkey", "", "the client key for this etcd member if using TLS. if empty use ORCA_ETCD_KEY.")
	serve.Flags().StringVar(&etcdCert, "etcdcert", "", "the client cert for this etcd member if using TLS. if empty use ORCA_ETCD_CERT.")
	serve.Flags().StringVar(&etcdCa, "etcdca", "", "the ca for this etcd member if using TLS. if empty use ORCA_ETCD_CA.")

}

func start(listenAddress string) error {
	mux := http.NewServeMux()
	srv := http.Server{
		Addr:    listenAddress,
		Handler: mux,
	}
	webRegisterURLMapping(mux)
	log.Infof("start listening on %s", srv.Addr)
	// todo: add TLS
	return srv.ListenAndServe()
}
