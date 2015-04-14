package main

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/clusterit/orca/cmd"
	"github.com/howeyc/gopass"
	"github.com/jmcvetta/napping"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	jsonType = "application/json"
)

// options
var (
	serviceUrl string
	user       string
	debug      bool
	password   bool
	revision   string
)

type cli struct {
	server   string
	user     string
	session  napping.Session
	password string
}

func (c *cli) url(u string) string {
	return c.server + u
}

func (c *cli) rq(m, u string, pl interface{}) *napping.Request {
	rq := &napping.Request{Method: m, Url: c.url(u), Header: &http.Header{}}
	if c.user != "" {
		rq.Userinfo = url.UserPassword(c.user, c.password)
	}
	rq.Header.Add("Content-Type", jsonType)
	rq.Header.Add("Accept", jsonType)
	if pl != nil {
		rq.Payload = pl
	}
	return rq
}

func newCli() *cli {
	user := viper.GetString("user")
	passwd := viper.GetString("password")
	if user != "" && passwd == "" && password {
		fmt.Print("Password: ")
		passwd = string(gopass.GetPasswdMasked())
	}
	sess := napping.Session{Log: debug}
	return &cli{server: serviceUrl, user: user, password: passwd, session: sess}
}

func main() {

	var cli = &cobra.Command{Use: "cli"}
	cli.PersistentFlags().StringVarP(&serviceUrl, "service", "s", "http://localhost:9010", "the service url of climan")
	cli.PersistentFlags().StringVarP(&user, "user", "u", "", "the username to use for the connection")
	cli.PersistentFlags().BoolVarP(&password, "password", "p", false, "prompt for a password if set. Environment variable ORCA_PASSWORD overwrites this")
	cli.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output of the HTTP flow")

	cli.AddCommand(usercmd, keycmd, zones, gateway, oauth)

	viper.SetEnvPrefix(cmd.OrcaPrefix)
	viper.AutomaticEnv()
	viper.BindPFlag("user", cli.Flag("user"))
	cli.Execute()

}
