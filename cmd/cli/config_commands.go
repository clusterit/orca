package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
)

var (
	loglevel  string
	timecheck string
	keyfile   string
)

var zones = &cobra.Command{
	Use:   "zones",
	Short: "list all zones",
	Long:  "list all current zones",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		stg, err := c.zones()
		if err != nil {
			fmt.Printf("%s\n", err)
		} else {
			for _, s := range stg {
				fmt.Printf("%s\n", s)
			}
		}
	},
}

var gateway = &cobra.Command{
	Use:   "gateway [zone]",
	Short: "show or update gateway config of zone",
	Long:  "dumps the current configuration of the gateway in the given zone or updates the values if options are set",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		zone := args[0]
		gw, err := c.getGateway(zone)
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
		update := false
		if !isNone(loglevel) && loglevel != gw.LogLevel {
			gw.LogLevel = loglevel
			update = true
		}
		if !isNone(timecheck) && gw.CheckAllow != isTrue(timecheck) {
			gw.CheckAllow = isTrue(timecheck)
			update = true
		}
		if keyfile != "" {
			kf, err := ioutil.ReadFile(keyfile)
			if err != nil {
				fmt.Printf("%s\n", err)
				return
			}
			gw.HostKey = string(kf)
			update = true
		}
		if update {
			if err := c.putGateway(zone, *gw); err != nil {
				fmt.Printf("%s\n", err)
				return
			}
		} else {
			fmt.Printf("LogLevel: %s\n", gw.LogLevel)
			fmt.Printf("Timecheck: %v\n", gw.CheckAllow)
			fmt.Printf("%s\n", gw.HostKey)
		}
	},
}

func initConfig() {
	gateway.Flags().StringVarP(&loglevel, "loglevel", "l", "", "the loglevel to set")
	gateway.Flags().StringVarP(&timecheck, "timecheck", "t", "", "update the CheckAllow field [true/false]")
	gateway.Flags().StringVarP(&keyfile, "keyfile", "k", "", "the keyfile for the host key")
}

func isNone(s string) bool {
	return s == ""
}

func isTrue(s string) bool {
	return strings.ToLower(s) == "true"
}
