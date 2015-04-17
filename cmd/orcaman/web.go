package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/jwt"
	"github.com/clusterit/orca/auth/oauth"
	"github.com/clusterit/orca/config"
)

func webInitZone(zone string, cfg config.ManagerConfig, reg oauth.OAuthRegistry) (auth.Auther, error) {
	blk, _ := pem.Decode([]byte(cfg.Key))
	jwtPk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}

	return jwt.NewAuther(jwtPk, reg), nil
}

func webSwitchSettings(cfg config.ManagerConfig, reg oauth.OAuthRegistry) (auth.Auther, error) {
	blk, _ := pem.Decode([]byte(cfg.Key))
	jwtPk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		return nil, err
	}
	return jwt.NewAuther(jwtPk, reg), nil
}

func webRegisterUrlMapping(mux *http.ServeMux) {
	mux.Handle("/", http.FileServer(rice.MustFindBox("app").HTTPBox()))
}

func NewWeb(etcds []string, publishurl string) (*restmanager, error) {
	rm, err := newRest(etcds, publishurl, webRoot)
	if err != nil {
		return nil, err
	}
	rm.initAuther = webInitZone
	rm.switchSettings = webSwitchSettings
	rm.registerUrlMapping = webRegisterUrlMapping
	return rm, nil
}
