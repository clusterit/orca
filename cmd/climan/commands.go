package main

import (
	"strings"

	"github.com/clusterit/orca/cmd"

	"github.com/spf13/cobra"
)

var climan = &cobra.Command{Use: "climan"}

func init() {
	var cmdServe = &cobra.Command{
		Use:   "serve",
		Short: "Starts the cli manager to listen on the given address",
		Long:  "Start the cli manager service on the given address. The CLI manager uses the given auth Urls for authentication",
		Run: func(cm *cobra.Command, args []string) {
			publish = cmd.PublishAddress(publish, listen)
			cmi, err := NewCLIManager(strings.Split(etcdConfig, ","))
			if err != nil {
				panic(err)
			}
			cmi.Start()
			defer cmi.Stop()
		},
	}

	var cmdManagers = &cobra.Command{
		Use:   "manager",
		Short: "Set the manager userids",
		Long:  "Set the manager userids in the configuration backend to enable bootstrapping",
		Run: func(cmd *cobra.Command, args []string) {
			managers = args
			_, err := NewCLIManager(strings.Split(etcdConfig, ","))
			if err != nil {
				panic(err)
			}
		},
	}

	climan.PersistentFlags().StringVarP(&etcdConfig, "etcd", "e", "http://localhost:4001", "etcd cluster machine Url's")
	climan.PersistentFlags().StringVarP(&publish, "publish", "p", "self", "self published http address. if empty don't publish, the value 'self' will be replace with the currnent listen address")
	climan.PersistentFlags().StringVarP(&zone, "zone", "z", "intranet", "use this zone as a subtree in the etcd backbone")
	cmdServe.Flags().StringVarP(&listen, "listen", "l", ":9010", "listen address for cli manager")
	cmdServe.Flags().VarP(&managers, "managers", "m", "comma separated list of userids with manager role")
	cmdServe.Flags().StringVarP(&authUrl, "authurl", "u", "", "An URL with Basic Auth to check user access")
	cmdServe.Flags().BoolVarP(&verifyCert, "certcheck", "c", true, "verify the certificate of the auth-url")

	climan.AddCommand(cmdServe, cmdManagers)
}
