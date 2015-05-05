package cmd

import (
	"strings"

	"github.com/clusterit/orca/logging"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/config"
)

const (
	ManagerService = "/userFetchService"
)

var (
	logger = logging.Simple()
)

func PublishAddress(pub, listen, path string) string {
	if pub == "self" {
		addr := strings.Split(listen, ":")
		if addr[0] != "" {
			pub = "http://" + addr[0] + ":" + addr[1]
		} else {
			pub = "http://localhost:" + addr[1]
		}
	}
	return pub + path
}

func ForceZone(cfger config.Configer, zone string, createGateway bool) (*config.Gateway, *config.ClusterConfig, error) {
	cfg, err := cfger.Cluster()
	if common.IsNotFound(err) {
		logger.Debugf("no clusterconfig existing, creating config 'local corp.'")
		confg, err := config.GenerateCluster("local", false)
		if err != nil {
			return nil, nil, err
		}
		if _, err := cfger.UpdateCluster(*confg); err != nil {
			return nil, nil, err
		}
	}
	zns, err := cfger.Zones()
	if common.IsNotFound(err) {
		logger.Debugf("no zones existing, creating zone '%s'.", zone)
		if err = cfger.CreateZone(zone); err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		// search zones if this zone already exist
		found := false
		for _, s := range zns {
			if s == zone {
				found = true
			}
		}
		if !found {
			logger.Debugf("zone not found, creating zone '%s'.", zone)
			if err = cfger.CreateZone(zone); err != nil {
				return nil, nil, err
			}
		}
	}
	gw, err := config.InitZone(cfger, zone, createGateway)
	return gw, cfg, err
}
