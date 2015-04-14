package main

import "github.com/spf13/cobra"

var ()

var oauth = &cobra.Command{
	Use:   "oauth",
	Short: "show, register and unregister oauth providers",
	Long:  "show, register and unregister oauth providers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var oauthList = &cobra.Command{
	Use:   "list",
	Short: "list all registered oauth providers",
	Long:  "list all registered oauth providers",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var oauthPut = &cobra.Command{
	Use:   "put [# network] [# clientid] [# clientsecret]",
	Short: "add a new oauth provider with the given data",
	Long:  "some predefined networks only require the clientid/secret, others need more data so use the flags to add more data",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	oauth.AddCommand(oauthList, oauthPut)
}
