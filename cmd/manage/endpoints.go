package main

import (
	"net/http"

	"github.com/GeertJohan/go.rice"
)

func webRegisterURLMapping(mux *http.ServeMux) {
	mux.HandleFunc("/authed/", kit.Handle(authed))
	registerOAuth(mux)
	mux.Handle("/", http.FileServer(rice.MustFindBox("app/public").HTTPBox()))
}
