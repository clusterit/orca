package main

import "github.com/spf13/viper"

func main() {
	viper.SetEnvPrefix("orca")
	viper.SetDefault("etcd_machines", "http://localhost:4001")
	viper.AutomaticEnv()

	root.AddCommand(serve)
	root.Execute()
}
