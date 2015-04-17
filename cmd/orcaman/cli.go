package main

import (
	"net/http"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/basic"
	"github.com/clusterit/orca/auth/oauth"
	"github.com/clusterit/orca/config"
)

func cliInitZone(zone string, cfg config.ManagerConfig, reg oauth.OAuthRegistry) (auth.Auther, error) {
	return basic.NewAuther(cfg.AuthUrl, cfg.VerifyCert), nil
}

func cliSwitchSettings(cfg config.ManagerConfig, reg oauth.OAuthRegistry) (auth.Auther, error) {
	return basic.NewAuther(cfg.AuthUrl, cfg.VerifyCert), nil
}

func cliRegisterUrlMapping(mux *http.ServeMux) {
}

func NewCli(etcds []string, publishurl string) (*restmanager, error) {
	rm, err := newRest(etcds, publishurl, cliRoot)
	if err != nil {
		return nil, err
	}
	rm.initAuther = cliInitZone
	rm.switchSettings = cliSwitchSettings
	rm.registerUrlMapping = cliRegisterUrlMapping
	return rm, nil
}
