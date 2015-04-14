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

func init() {
	oauth.AddCommand(oauthList)
}
