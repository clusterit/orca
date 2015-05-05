package main

import (
	"net/http"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/oauth"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/etcd"
)

func cliInitZone(zone string, cfg config.ClusterConfig, reg oauth.AuthRegistry) (auth.Auther, error) {
	return nil, nil
}

func cliSwitchSettings(cfg config.ClusterConfig, reg oauth.AuthRegistry) (auth.Auther, error) {
	return nil, nil
}

func cliRegisterUrlMapping(mux *http.ServeMux) {
}

func NewCli(cc *etcd.Cluster, cfg config.Configer, publishurl string) (*restmanager, error) {
	rm, err := newRest(cc, cfg, publishurl, cliRoot)
	if err != nil {
		return nil, err
	}
	rm.initAuther = cliInitZone
	rm.switchSettings = cliSwitchSettings
	rm.registerUrlMapping = cliRegisterUrlMapping
	return rm, nil
}
