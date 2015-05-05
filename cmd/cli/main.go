package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/clusterit/orca/common"
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
	debug      bool
	unsecure   bool
	revision   string
	usertoken  string
)

type cli struct {
	server  string
	session napping.Session
}

func (c *cli) url(u string) string {
	return c.server + u
}

func (c *cli) rq(m, u string, pl interface{}) *napping.Request {
	rq := &napping.Request{Method: m, Url: c.url(u), Header: &http.Header{}}
	rq.Header.Add("Content-Type", jsonType)
	rq.Header.Add("Accept", jsonType)
	rq.Header.Add("X-Orca-Token", usertoken)
	if pl != nil {
		rq.Payload = pl
	}
	return rq
}

func newCli() *cli {
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: unsecure},
	}
	client := &http.Client{Transport: tr}
	sess := napping.Session{Log: debug, Client: client}
	return &cli{server: serviceUrl, session: sess}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Long:  "Print the version number of Orca client",
	Short: `Orca's build version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Orca service client, revision '%s'\n", revision)
	},
}
var whoami = &cobra.Command{
	Use:   "whoami",
	Long:  "Print my userdata",
	Short: `Query my data from orca`,
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		me, err := c.me()
		exitWhenError(err)
		dumpValue(me)
	},
}
var permit = &cobra.Command{
	Use:   "permit [#duration in secs]",
	Long:  "permit the current user to login via the gateway for the given time",
	Short: `permit the current user to do a ssh login`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
			os.Exit(1)
		}
		dur, err := strconv.ParseInt(args[0], 10, 0)
		exitWhenError(err)
		c := newCli()

		a, err := c.permit(int(dur))
		exitWhenError(err)

		dumpValue(a)
	},
}

func main() {

	var cli = &cobra.Command{Use: "cli"}
	cli.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug output of the HTTP flow")
	cli.PersistentFlags().BoolVarP(&unsecure, "unsecure", "u", false, "do not verify the SSL cert of the remote service (use only for selfsigned certs)")

	cli.AddCommand(whoami, permit, usercmd, keycmd, zones, gateway, cluster, oauthCmd, versionCmd)

	viper.SetEnvPrefix(common.OrcaPrefix)
	viper.AutomaticEnv()
	serviceUrl = viper.GetString("service")
	usertoken = viper.GetString("token")

	run := true
	if serviceUrl == "" {
		fmt.Printf("please set the envirnment ORCA_SERVICE to the URL of a running orca manager")
		run = false
	}
	if usertoken == "" {
		fmt.Printf("please set the environment ORCA_TOKEN to your ID-Token which is displayed in the webapp. This token identifies you!")
		run = false
	}
	if run {
		cli.Execute()
	} else {
		os.Exit(1)
	}
}

func dumpValue(v interface{}) {
	res, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		fmt.Printf("cannot dump %#v as JSON: %s\n", v, e)
	} else {
		fmt.Printf("%s\n", string(res))
	}
}
