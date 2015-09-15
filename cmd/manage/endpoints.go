package main

import (
	"net/http"

	"github.com/GeertJohan/go.rice"
)

func webRegisterUrlMapping(mux *http.ServeMux) {
	mux.HandleFunc("/authed/", kit.Handle(authed))
	mux.Handle("/", http.FileServer(rice.MustFindBox("app/public").HTTPBox()))
}
