package main

import (
	"os"

	"github.com/clusterit/orca/auth/oauth"
	"github.com/davecgh/go-spew/spew"

	"github.com/spf13/cobra"
)

var (
	loginscope     string
	scopes         string
	authUrl        string
	accessTokenUrl string
	userinfoUrl    string
	pathid         string
	pathname       string
	pathpicture    string
	pathcover      string
)

var oauthCmd = &cobra.Command{
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
		c := newCli()
		res, err := c.listOauthProviders()
		exitWhenError(err)
		for _, r := range res {
			spew.Dump(r)
		}
	},
}

var oauthPut = &cobra.Command{
	Use:   "put [# network] [# clientid] [# clientsecret]",
	Short: "add a new oauth provider with the given data",
	Long:  "some predefined networks only require the clientid/secret, others need more data so use the flags to add more data",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 3 {
			cmd.Usage()
			os.Exit(1)
		}
		reg := oauth.OauthRegistration{
			Network:        args[0],
			ClientId:       args[1],
			ClientSecret:   args[2],
			Scopes:         scopes,
			AuthUrl:        authUrl,
			AccessTokenUrl: accessTokenUrl,
			UserinfoUrl:    userinfoUrl,
			PathId:         pathid,
			PathName:       pathname,
			PathPicture:    pathpicture,
			PathCover:      pathcover,
		}
		c := newCli()
		exitWhenError(c.putProvider(reg))
	},
}

var oauthDelete = &cobra.Command{
	Use:   "delete [# network]",
	Short: "delete the provider for the given network",
	Long:  "delete the provider for the given network",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
			os.Exit(1)
		}
		c := newCli()
		exitWhenError(c.delProvider(args[0]))
	},
}

func init() {
	oauthPut.Flags().StringVar(&loginscope, "loginscope", "", "the scopes to use on the client for authentication")
	oauthPut.Flags().StringVar(&scopes, "scopes", "", "the scopes to use on the server")
	oauthPut.Flags().StringVar(&authUrl, "authurl", "", "the authorization url")
	oauthPut.Flags().StringVar(&accessTokenUrl, "accesstokenurl", "", "the url to get an access token")
	oauthPut.Flags().StringVar(&userinfoUrl, "userinfourl", "", "the url to get the user account information")
	oauthPut.Flags().StringVar(&pathid, "id", "", "a json pathspec for the id value")
	oauthPut.Flags().StringVar(&pathname, "name", "", "a json pathspec for the name value")
	oauthPut.Flags().StringVar(&pathpicture, "picture", "", "a json pathspec for the picture value")
	oauthPut.Flags().StringVar(&pathcover, "cover", "", "a json pathspec for the cover value")
	oauthCmd.AddCommand(oauthList, oauthPut, oauthDelete)
}
