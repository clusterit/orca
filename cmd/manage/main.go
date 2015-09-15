package main

import (
	"log"
	"net/http"
	"os"

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

func authed(ac *authkit.AuthContext, w http.ResponseWriter, rq *http.Request) {
	log.Printf("user: %#v", ac.User)
	for k, v := range ac.Claims {
		log.Printf(" - vals[%s] = %s\n", k, v)
	}
}

func main() {
	root.AddCommand(serve)
	root.Execute()
}
