package main

import (
	"net/http"

	"github.com/GeertJohan/go.rice"
)

func webRegisterUrlMapping(mux *http.ServeMux) {
	mux.Handle("/", http.FileServer(rice.MustFindBox("app").HTTPBox()))
}
