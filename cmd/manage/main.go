package main

import (
	"strings"

	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/user"
	"github.com/clusterit/orca/user/backend/etcdstore"
	"github.com/spf13/viper"
)

func mustServiceBackend() user.Users {
	b, e := createServiceBackend()
	if e != nil {
		panic(e)
	}
	return b
}

func createServiceBackend() (user.Users, error) {
	if etcds == "" {
		etcds = viper.GetString("etcd_machines")
	}
	if etcdKey == "" {
		etcdKey = viper.GetString("etcd_key")
	}
	if etcdCert == "" {
		etcdCert = viper.GetString("etcd_cert")
	}
	if etcdCa == "" {
		etcdCa = viper.GetString("etcd_ca")
	}
	cc, err := etcd.InitTLS(strings.Split(etcds, ","), etcdKey, etcdCert, etcdCa)
	if err != nil {
		return nil, err
	}
	return etcdstore.New(cc)
}

func main() {
	viper.SetEnvPrefix("orca")
	viper.SetDefault("etcd_machines", "http://localhost:4001")
	viper.AutomaticEnv()

	userCommand.AddCommand(useradd)

	root.AddCommand(serve, userCommand)
	root.Execute()
}
