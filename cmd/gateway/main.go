package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/logutils"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	"github.com/clusterit/orca/users"

	"github.com/spf13/viper"

	"golang.org/x/crypto/ssh"
)

var (
	fetcher       UserFetcher
	configuration *config.Gateway
	sshConfig     ssh.ServerConfig
	configer      config.Configer
	zone          string
	lock          sync.Mutex
	revision      string
)

func init() {
	viper.SetEnvPrefix(cmd.OrcaPrefix)
	viper.AutomaticEnv()
	viper.SetDefault("bind", ":2222")
	viper.SetDefault("etcd", "http://localhost:4001")
	viper.SetDefault("zone", "intranet")

	zone = viper.GetString("zone")
	etcds := strings.Split(viper.GetString("etcd"), ",")
	cc, err := etcd.Init(etcds)
	if err != nil {
		panic(err)
	}
	fetcher, err = NewHttpFetcher(cc)
	if err != nil {
		panic(err)
	}
	cfger, err := config.New(cc)
	if err != nil {
		panic(err)
	}
	configer = cfger

}

func initWithConfig(gw *config.Gateway) error {
	lock.Lock()
	defer lock.Unlock()

	signer, err := ssh.ParsePrivateKey([]byte(gw.HostKey))
	if err != nil {
		return err
	}
	filter := &logutils.LevelFilter{
		Levels:   logging.Levels,
		MinLevel: logging.ByName(gw.LogLevel),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	sshConfig = ssh.ServerConfig{
		PublicKeyCallback: keyAuth,
	}
	sshConfig.AddHostKey(signer)
	configuration = gw
	return nil
}

func initWithSettings(zone string) error {
	cfg, _, err := cmd.ForceZone(configer, zone, true, true)
	if err != nil {
		return err
	}
	initWithConfig(cfg)

	go func() {
		ngw, stp, err := configer.Gateway(zone)
		if err != nil {
			Log(logging.Error, "cannot create watcher for gateway config: %s")
			return
		}
		for gw := range ngw {
			Log(logging.Debug, "new gateway config: %#v", gw)
			initWithConfig(&gw)
		}
		close(stp)
	}()
	return nil
}

func checkAllowed(u *users.User) error {
	if !configuration.CheckAllow {
		return nil
	}
	if u.Allowance == nil {
		return fmt.Errorf("please activate your account")
	}
	if u.Allowance.Until.Before(time.Now()) {
		return fmt.Errorf("your activation timed out")
	}

	return nil
}

func keyAuth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	pubk := string(ssh.MarshalAuthorizedKey(key))
	usr, err := fetcher.UserByKey(zone, strings.TrimSpace(pubk))
	if err != nil {
		Log(logging.Debug, "remote: %s: cannot fetch key for user '%s': %s", conn.RemoteAddr().String(), conn.User(), err)
		return nil, err
	}
	if err := checkAllowed(usr); err != nil {
		Log(logging.Debug, "remote: %s: not allowed to login for user '%s': %s", conn.RemoteAddr().String(), conn.User(), err)
		return nil, err
	}
	Log(logging.Info, "remote: %s: login by %+v", conn.RemoteAddr().String(), usr)
	return &ssh.Permissions{Extensions: map[string]string{
		"user_id": usr.Id}}, nil
}

func main() {
	initWithSettings(zone)

	bind := viper.GetString("bind")
	socket, err := net.Listen("tcp", bind)
	if err != nil {
		panic(err)
	}
	Log(logging.Info, "gateway listens on %#v ...", socket.Addr().String())
	for {

		tcpConn, err := socket.Accept()
		if err != nil {
			Log(logging.Error, "failed to accept incoming connection (%s)", err)
			continue
		}

		_, err = NewSession(ssh.NewServerConn(tcpConn, &sshConfig))

		if err != nil {
			Log(logging.Error, "failed to handshake (%s)", err)
			continue
		}

	}
}
