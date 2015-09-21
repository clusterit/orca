package main

import (
	"net/http"
	"os"

	"github.com/GeertJohan/go.rice"
	"github.com/clusterit/orca/cmd/manage/endpoints"
	"github.com/emicklei/go-restful"
	"github.com/spf13/cobra"
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
	cnt := restful.NewContainer()
	restservice := endpoints.NewUserService(kit, mustServiceBackend(), pt)
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
