package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	loglevel      string
	timecheck     string
	keyfile       string
	defaulthost   string
	force2fa      string
	maxtimeout2fa int
	allowdeny     string
	allowedcidrs  string
	deniedcidrs   string
	name          string
	selfregister  string
)

var zones = &cobra.Command{
	Use:   "zones",
	Short: "list all zones",
	Long:  "list all current zones",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		stg, err := c.zones()
		exitWhenError(err)
		for _, s := range stg {
			fmt.Printf("%s\n", s)
		}
	},
}

var gateway = &cobra.Command{
	Use:   "gateway [zone]",
	Short: "show or update gateway config of zone",
	Long:  "dumps the current configuration of the gateway in the given zone or updates the values if options are set",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		if len(args) < 1 {
			cmd.Usage()
			os.Exit(1)
		}
		zone := args[0]
		gw, err := c.getGateway(zone)
		exitWhenError(err)
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
			exitWhenError(err)
			gw.HostKey = string(kf)
			update = true
		}
		if defaulthost != "" {
			gw.DefaultHost = defaulthost
			update = true
		}
		if force2fa != "" {
			gw.Force2FA = isTrue(force2fa)
			update = true
		}
		if maxtimeout2fa >= 0 {
			gw.MaxAutologin2FA = maxtimeout2fa
			update = true
		}
		if allowdeny != "" {
			gw.AllowDeny = allowdeny == "allow"
			update = true
		}
		if allowedcidrs != "" {
			gw.AllowedCidrs = strings.Split(allowedcidrs, ",")
			update = true
		}
		if deniedcidrs != "" {
			gw.DeniedCidrs = strings.Split(deniedcidrs, ",")
			update = true
		}
		if update {
			if err := c.putGateway(zone, *gw); err != nil {
				fmt.Printf("%s\n", err)
				return
			}
		} else {
			dumpValue(gw)
		}
	},
}

var cluster = &cobra.Command{
	Use:   "cluster ",
	Short: "show or update cluster config",
	Long:  "dumps the current configuration of the cluster in the given zone or updates the values if options are set",
	Run: func(cmd *cobra.Command, args []string) {
		c := newCli()
		cc, err := c.getCluster()
		exitWhenError(err)
		update := false
		if keyfile != "" {
			kf, err := ioutil.ReadFile(keyfile)
			exitWhenError(err)
			cc.Key = string(kf)
			update = true
		}
		if name != "" {
			cc.Name = name
			update = true
		}
		if selfregister != "" {
			cc.SelfRegister = isTrue(selfregister)
			update = true
		}
		if update {
			if err := c.putCluster(*cc); err != nil {
				fmt.Printf("%s\n", err)
				return
			}
		} else {
			dumpValue(*cc)
		}
	},
}

func init() {
	gateway.Flags().StringVar(&loglevel, "loglevel", "", "the loglevel to set")
	gateway.Flags().StringVar(&timecheck, "timecheck", "", "update the CheckAllow field [true/false]")
	gateway.Flags().StringVar(&keyfile, "keyfile", "", "the keyfile for the host key")
	gateway.Flags().StringVar(&defaulthost, "defaulthost", "", "the default host for the gateway")
	gateway.Flags().StringVar(&force2fa, "force2fa", "", "force 2fa for this zone [true/false]")
	gateway.Flags().IntVar(&maxtimeout2fa, "maxtimeout2fa", -1, "maximum timeout in seconds for a successful 2fa. use -1 to leave it unchanged")
	gateway.Flags().StringVar(&allowdeny, "allowdeny", "", "use 'allow' for allow/deny, 'deny' for deny/allow")
	gateway.Flags().StringVar(&allowedcidrs, "allowedcidrs", "", "a comma seperated list of allowed cidrs")
	gateway.Flags().StringVar(&deniedcidrs, "deniedcidrs", "", "a comma seperated list of denied cidrs")

	cluster.Flags().StringVar(&keyfile, "keyfile", "", "the keyfile for the host key")
	cluster.Flags().StringVar(&name, "name", "", "the name of the cluster")
	cluster.Flags().StringVar(&selfregister, "selfregister", "", "use selfregister for this cluster [true/false]")
}

func isNone(s string) bool {
	return s == ""
}

func isTrue(s string) bool {
	return strings.ToLower(s) == "true"
}
