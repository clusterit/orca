package cmd

import (
	"strings"

	"github.com/clusterit/orca/logging"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/config"
)

const (
	OrcaPrefix     = "orca"
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

func ForceZone(cfger config.Configer, zone string, createGateway, createMc bool) (*config.Gateway, *config.ManagerConfig, error) {
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
	var myMc *config.ManagerConfig
	var myGateway *config.Gateway
	if createGateway {
		gw, err := cfger.GetGateway(zone)
		if common.IsNotFound(err) {
			gw, err := config.GenerateGateway()
			if err != nil {
				return nil, nil, err
			}
			logger.Debugf("create a default gatway setting")
			if err = cfger.PutGateway(zone, *gw); err != nil {
				return nil, nil, err
			}
			myGateway = gw
		} else if err != nil {
			return nil, nil, err
		} else {
			myGateway = gw
		}
	}
	if createMc {
		mcf, err := cfger.GetManagerConfig(zone)
		if common.IsNotFound(err) {
			mcf, err := config.GenerateManagerConfig()
			if err != nil {
				return nil, nil, err
			}
			logger.Debugf("create a default ManagerConfig setting")
			if err = cfger.PutManagerConfig(zone, *mcf); err != nil {
				return nil, nil, err
			}
			myMc = mcf
		} else if err != nil {
			return nil, nil, err
		} else {
			myMc = mcf
		}
	}

	return myGateway, myMc, nil
}
