package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/clusterit/orca/timebuffer"

	"github.com/hashicorp/logutils"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	"github.com/clusterit/orca/users"

	"github.com/spf13/viper"

	"golang.org/x/crypto/ssh"
)

const (
	timeoutFor2FAinSeconds = 30
)

var (
	fetcher       UserFetcher
	configuration *config.Gateway
	sshConfig     ssh.ServerConfig
	configer      config.Configer
	zone          string
	lock          sync.Mutex
	revision      = "latest"
)

func initGateway() {
	viper.SetEnvPrefix(common.OrcaPrefix)
	viper.AutomaticEnv()
	viper.SetDefault("bind", ":2022")
	viper.SetDefault("etcd_machines", "http://localhost:4001")

	viper.SetDefault("zone", "intranet")

	zone = viper.GetString("zone")
	etcds := strings.Split(viper.GetString("etcd_machines"), ",")
	etcdKey := viper.GetString("etcd_key")
	etcdCert := viper.GetString("etcd_cert")
	etcdCa := viper.GetString("etcd_ca")

	cc, err := etcd.InitTLS(etcds, etcdKey, etcdCert, etcdCa)
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
		PasswordCallback:  pwdCallback,
		ServerVersion:     fmt.Sprintf("SSH-2.0-orca_%s", revision),
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

func checkAllowed(sessid []byte, u *users.User) error {
	if u.Use2FA {
		// if the users has 2FA, check the allowance field if another
		// check is needed
		if u.Allowance == nil || u.Allowance.Until.Before(time.Now()) {
			timebuffer.Put(string(sessid), u, timeoutFor2FAinSeconds)
			return fmt.Errorf("2FA enabled, next password check")
		}

		if u.Allowance != nil && u.Allowance.Until.After(time.Now().Add(time.Duration(configuration.MaxAutologin2FA)*time.Second)) {
			timebuffer.Put(string(sessid), u, timeoutFor2FAinSeconds)
			return fmt.Errorf("allowance too long, next password check")
		}
		// there is a successfull allowance
		return nil
	}
	if configuration.Force2FA {
		return fmt.Errorf("you must use 2fa!")
	}
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
	if err := checkAllowed(conn.SessionID(), usr); err != nil {
		Log(logging.Debug, "remote: %s: not allowed to login for user '%s': %s", conn.RemoteAddr().String(), conn.User(), err)
		return nil, err
	}
	Log(logging.Info, "remote: %s: login by %+v", conn.RemoteAddr().String(), usr)
	return &ssh.Permissions{Extensions: map[string]string{
		"user_id": usr.Id}}, nil
}

func pwdCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	keyusr := timebuffer.Get(string(conn.SessionID()))
	if keyusr == nil {
		Log(logging.Debug, "remote: %s: no key auth happend before OTP check '%s'", conn.RemoteAddr().String(), conn.User())
		return nil, fmt.Errorf("no key auth happend before OTP check")
	}
	usr := keyusr.(*users.User)
	ttl := usr.AutologinAfter2FA
	if ttl > configuration.MaxAutologin2FA {
		ttl = configuration.MaxAutologin2FA
	}
	err := fetcher.CheckToken(zone, usr.Id, string(password), ttl)
	if err != nil {
		Log(logging.Debug, "remote: %s: wrong token for user '%s': '%s'", conn.RemoteAddr().String(), conn.User(), err)
		return nil, err
	}

	return &ssh.Permissions{Extensions: map[string]string{
		"user_id": string(usr.Id)}}, nil
}

func main() {
	initGateway()
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
