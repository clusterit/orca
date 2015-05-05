package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"time"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/logging"
	cetcd "github.com/coreos/go-etcd/etcd"
	"golang.org/x/crypto/ssh"
)

type Gateway struct {
	DefaultHost     string   `json:"defaulthost"`
	Force2FA        bool     `json:"force2fa"`
	HostKey         string   `json:"hostkey"`
	LogLevel        string   `json:"loglevel"`
	CheckAllow      bool     `json:"checkAllow"`
	MaxAutologin2FA int      `json:"maxautologin2fa"`
	AllowedCidrs    []string `json:"allowedcidrs"`
	DeniedCidrs     []string `json:"deniedcidrs"`
	AllowDeny       bool     `json:"allowdeny"`
}

type NewGateway <-chan Gateway

type Stop chan bool

type ClusterConfig struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	SelfRegister bool   `json:"selfregister"`
}

type NewClusterConfig <-chan ClusterConfig

type Configer interface {
	Cluster() (*ClusterConfig, error)
	UpdateCluster(ClusterConfig) (*ClusterConfig, error)
	ClusterConfig() (NewClusterConfig, Stop, error)
	Zones() ([]string, error)
	CreateZone(zone string) error
	DropZone(zone string) error
	PutGateway(zone string, gw Gateway) error
	GetGateway(zone string) (*Gateway, error)
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

func (e *etcdConfig) Cluster() (*ClusterConfig, error) {
	var result ClusterConfig
	return &result, e.persister.Get("/cluster", &result)
}

func (e *etcdConfig) UpdateCluster(cc ClusterConfig) (*ClusterConfig, error) {
	_, err := ssh.ParsePrivateKey([]byte(cc.Key))
	if err != nil {
		return nil, err
	}
	return &cc, e.persister.Put("/cluster", cc)
}

func (e *etcdConfig) Zones() ([]string, error) {
	return e.persister.Ls("/zones")
}

func (e *etcdConfig) CreateZone(zone string) error {
	return e.persister.Put(e.pt(zone, "created"), time.Now())
}

func (e *etcdConfig) DropZone(zone string) error {
	return e.persister.RemoveDir(e.pt(zone, ""))
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

func (e *etcdConfig) ClusterConfig() (NewClusterConfig, Stop, error) {
	cchan := make(chan ClusterConfig)
	stop := make(Stop)
	etcrsp := make(chan *cetcd.Response)
	go func() {
		path := e.persister.Path("/cluster")
		e.persister.RawClient().Watch(path, 0, false, etcrsp, stop)
	}()
	go func() {
		for {
			select {
			case r := <-etcrsp:
				if r != nil && r.Node != nil {
					val := []byte(r.Node.Value)
					var cc ClusterConfig
					if err := json.Unmarshal(val, &cc); err == nil {
						cchan <- cc
					}
				}
			case <-stop:
				return
			}
		}
	}()
	return cchan, stop, nil
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
				if r != nil && r.Node != nil {
					val := []byte(r.Node.Value)
					var gw Gateway
					if err := json.Unmarshal(val, &gw); err == nil {
						gwchan <- gw
					}
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
		HostKey:      string(pem.EncodeToMemory(&data)),
		LogLevel:     logging.Debug,
		CheckAllow:   true,
		AllowDeny:    true,
		AllowedCidrs: []string{"0.0.0.0/0"},
		DeniedCidrs:  []string{"127.0.0.1/8"},
	}, nil
}

func GenerateCluster(name string, selfreg bool) (*ClusterConfig, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	data := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)}
	return &ClusterConfig{
		Key:          string(pem.EncodeToMemory(&data)),
		Name:         name,
		SelfRegister: selfreg,
	}, nil
}

func InitZone(cfger Configer, zone string, createGateway bool) (*Gateway, error) {
	var myGateway *Gateway
	if createGateway {
		gw, err := cfger.GetGateway(zone)
		if common.IsNotFound(err) {
			gw, err := GenerateGateway()
			if err != nil {
				return nil, err
			}
			if err = cfger.PutGateway(zone, *gw); err != nil {
				return nil, err
			}
			myGateway = gw
		} else if err != nil {
			return nil, err
		} else {
			myGateway = gw
		}
	}

	return myGateway, nil
}
