package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"time"

	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	cetcd "github.com/coreos/go-etcd/etcd"
	"golang.org/x/crypto/ssh"
)

type ManagerConfig struct {
	Key        string `json:"key"`
	AuthUrl    string `json:"authUrl"`
	VerifyCert bool   `json:"verifyCert"`
}

type NewManagerConfig <-chan ManagerConfig

type Gateway struct {
	Use2FA     bool   `json:"use2fa"`
	HostKey    string `json:"hostkey"`
	LogLevel   string `json:"loglevel"`
	CheckAllow bool   `json:"checkAllow"`
}

type NewGateway <-chan Gateway

type Stop chan bool

type Configer interface {
	Zones() ([]string, error)
	CreateZone(zone string) error
	DropZone(zone string) error
	PutManagerConfig(zone string, mgrs ManagerConfig) error
	GetManagerConfig(zone string) (*ManagerConfig, error)
	PutGateway(zone string, gw Gateway) error
	GetGateway(zone string) (*Gateway, error)
	ManagerConfig(zone string) (NewManagerConfig, Stop, error)
	Gateway(zone string) (NewGateway, Stop, error)
}

type etcdConfig struct {
	persister etcd.Persister
}

func New(cl *etcd.Cluster) (Configer, error) {
	p, e := cl.NewJsonPersister("")
	if e != nil {
		return nil, e
	}
	return &etcdConfig{persister: p}, nil
}

func (e *etcdConfig) pt(s, k string) string {
	return "/zones/" + s + "/" + k
}

func (e *etcdConfig) Zones() ([]string, error) {
	return e.persister.Ls("/zones")
}

func (e *etcdConfig) CreateZone(zone string) error {
	return e.persister.Put(e.pt(zone, "created"), time.Now())
}

func (e *etcdConfig) DropZone(zone string) error {
	return e.persister.Remove(e.pt(zone, ""))
}

func (e *etcdConfig) PutManagerConfig(zone string, mc ManagerConfig) error {
	_, err := ssh.ParsePrivateKey([]byte(mc.Key))
	if err != nil {
		return err
	}
	return e.persister.Put(e.pt(zone, "mgrConfig"), mc)
}

func (e *etcdConfig) GetManagerConfig(zone string) (*ManagerConfig, error) {
	var result ManagerConfig
	return &result, e.persister.Get(e.pt(zone, "mgrConfig"), &result)
}

func (e *etcdConfig) PutGateway(zone string, gw Gateway) error {
	_, err := ssh.ParsePrivateKey([]byte(gw.HostKey))
	if err != nil {
		return err
	}
	return e.persister.Put(e.pt(zone, "gateway"), gw)
}

func (e *etcdConfig) GetGateway(zone string) (*Gateway, error) {
	var result Gateway
	return &result, e.persister.Get(e.pt(zone, "gateway"), &result)
}

func (e *etcdConfig) ManagerConfig(zone string) (NewManagerConfig, Stop, error) {
	mgrchan := make(chan ManagerConfig)
	stop := make(Stop)
	etcrsp := make(chan *cetcd.Response)
	go func() {
		path := e.persister.Path(e.pt(zone, "mgrConfig"))
		e.persister.RawClient().Watch(path, 0, false, etcrsp, stop)
	}()
	go func() {
		for {
			select {
			case r := <-etcrsp:
				val := []byte(r.Node.Value)
				var mg ManagerConfig
				if err := json.Unmarshal(val, &mg); err == nil {
					mgrchan <- mg
				}
			case <-stop:
				return
			}
		}
	}()
	return mgrchan, stop, nil
}

func (e *etcdConfig) Gateway(zone string) (NewGateway, Stop, error) {
	gwchan := make(chan Gateway)
	stop := make(Stop)
	etcrsp := make(chan *cetcd.Response)
	go func() {
		path := e.persister.Path(e.pt(zone, "gateway"))
		e.persister.RawClient().Watch(path, 0, false, etcrsp, stop)
	}()
	go func() {
		for {
			select {
			case r := <-etcrsp:
				val := []byte(r.Node.Value)
				var gw Gateway
				if err := json.Unmarshal(val, &gw); err == nil {
					gwchan <- gw
				}
			case <-stop:
				return
			}
		}
	}()
	return gwchan, stop, nil
}

func GenerateGateway() (*Gateway, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	data := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)}
	return &Gateway{
		HostKey:    string(pem.EncodeToMemory(&data)),
		LogLevel:   logging.Debug,
		CheckAllow: true,
	}, nil
}

func GenerateManagerConfig() (*ManagerConfig, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	data := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)}
	return &ManagerConfig{
		Key: string(pem.EncodeToMemory(&data)),
	}, nil
}
