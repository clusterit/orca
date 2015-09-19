package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	bind       string
	port       int
	etcdConfig string
	etcdKey    string
	etcdCert   string
	etcdCa     string
)

var serve = &cobra.Command{
	Use:   "serve",
	Short: "Start the web listener for the endpoint an web UI",
	Long:  "Starts the web listener on the address specified by the options. Note that this is not the same as the publish address.",
	Run: func(cmd *cobra.Command, args []string) {
		listenaddress := fmt.Sprintf("%s:%d", bind, port)
		log.Error(start(listenaddress))
	},
}

func init() {
	serve.Flags().StringVarP(&bind, "bind", "b", "127.0.0.1", "bind address for the endpoint")
	serve.Flags().IntVarP(&port, "port", "p", 9011, "bin port for the endpoint")
	serve.Flags().StringVarP(&etcdConfig, "etcd", "e", "", "etcd cluster machine Url's. if empty use env ORCA_ETCD_MACHINES which is by default http://localhost:4001")
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
