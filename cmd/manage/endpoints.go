package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/GeertJohan/go.rice"
	"github.com/clusterit/orca/cmd/manage/endpoints"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/user/backend/etcdstore"
	"github.com/emicklei/go-restful"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ulrichSchreiner/authkit"
)

var (
	kit  = authkit.Must("/authkit")
	root = &cobra.Command{}
)

func registerOAuth(mux *http.ServeMux) {
	kit.Add(authkit.Instance(authkit.Google, os.Getenv("GOOGLE_CLIENTID"), os.Getenv("GOOGLE_CLIENTSECRET")))
	kit.Register(mux)
}

func createServiceContainer(pt string) (*restful.Container, error) {
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
	cc, err := etcd.InitTLS(strings.Split(etcds, ","), etcdKey, etcdCert, etcdCa)
	if err != nil {
		return nil, err
	}
	backend, err := etcdstore.New(cc)
	if err != nil {
		return nil, err
	}
	cnt := restful.NewContainer()
	restservice := endpoints.NewUserService(kit, backend, pt)
	cnt.Add(restservice)
	return cnt, nil

}

func webRegisterURLMapping(mux *http.ServeMux) {
	registerOAuth(mux)
	cnt, err := createServiceContainer("/api/v1")
	if err != nil {
		panic(err)
	}
	mux.Handle("/api/v1/", cnt)
	mux.Handle("/", http.FileServer(rice.MustFindBox("app/public").HTTPBox()))
}
