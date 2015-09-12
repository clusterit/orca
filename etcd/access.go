package etcd

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"sync"
	"time"

	"gopkg.in/errgo.v1"

	log "github.com/Sirupsen/logrus"
	orcaerr "github.com/clusterit/orca/errors"

	etcderr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

const (
	orcaManagePath  = "/orca/manage"
	orcaPersistPath = "/orca"
)

// A Cluster implementation backed by etcd
type Cluster struct {
	client *etcd.Client
}

// A Configurator supports operations on a subtree inside of etcd
type Configurator interface {
	Register(pt, value string, ttl int) error
	Unregister(pt string)
	GetValues(pt string) ([]string, error)
}

// Init creates the cluster by using the etcd-members
func Init(machines []string) (*Cluster, error) {
	client := etcd.NewClient(machines)
	return &Cluster{client}, nil
}

// InitTLS creates the cluster with TLS and client cert
func InitTLS(machines []string, key, cert, cacert string) (*Cluster, error) {
	if cert == "" && key == "" && cacert == "" {
		log.Info("connect to etcd without TLS: %s", machines)
		return Init(machines)
	}
	log.Info("connect to etcd with TLS: %s", machines)
	client, err := etcd.NewTLSClient(machines, cert, key, cacert)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ETCD via TLS: %s", err)
	}
	return &Cluster{client}, nil
}

type pathConfigurator struct {
	basepath string
	cc       *Cluster
	stop     map[string]chan bool
	mux      sync.Mutex
}

// NewConfigurator creates a configurator for a subtree path.
func (cc *Cluster) NewConfigurator(base string) (Configurator, error) {
	_, err := cc.client.Get(base, false, false)
	if err != nil {
		_, err = cc.client.CreateDir(base, 0)
		if err != nil {
			return nil, err
		}
	}

	pc := pathConfigurator{basepath: base, cc: cc}
	pc.stop = make(map[string]chan bool)
	return &pc, nil
}

// NewManager returns the Configurator for the manage path
func (cc *Cluster) NewManager() (Configurator, error) {
	return cc.NewConfigurator(orcaManagePath)
}

// Register a value inside the subtree of the configuration. The pt
// must be an absolute path which is appended to the base path of
// the subtree. The value is stored in this tree and refresh every
// ttl-5 seconds. If ttl is lower than 10 seconds its value is increased
// by 10 seconds.
func (pc *pathConfigurator) Register(pt, value string, ttl int) error {
	pc.mux.Lock()
	defer pc.mux.Unlock()

	stopchan := make(chan bool)
	path := pc.basepath + pt
	pc.stop[path] = stopchan

	if ttl < 10 {
		ttl = ttl + 5
	}
	rsp, err := pc.cc.client.AddChild(path, value, uint64(ttl))
	if err != nil {
		return err
	}
	key := rsp.Node.Key
	go func() {
		t := time.Tick(time.Second * (time.Duration(ttl - 10)))
		for {
			select {
			case <-t:
				pc.cc.client.Update(key, value, uint64(ttl))
			case <-stopchan:
				pc.cc.client.Delete(key, true)
				return
			}
		}
	}()
	return nil
}

// Stops the auto-updating of the given path
func (pc *pathConfigurator) Unregister(pt string) {
	pc.mux.Lock()
	defer pc.mux.Unlock()
	path := pc.basepath + pt
	close(pc.stop[path])
	delete(pc.stop, path)
}

// GetValues retrieves all registered values at path pt.
func (pc *pathConfigurator) GetValues(pt string) ([]string, error) {
	path := pc.basepath + pt
	rsp, err := pc.cc.client.Get(path, false, false)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, n := range rsp.Node.Nodes {
		res = append(res, n.Value)
	}
	return res, nil
}

// A Persister can read/write values from/to etcd
type Persister interface {
	Path(s string) string
	Put(k string, v interface{}) error
	PutTTL(k string, ttl uint64, v interface{}) error
	Get(k string, v interface{}) error
	GetAll(sorted, recursive bool, v interface{}) error
	Remove(k string) error
	RemoveDir(k string) error
	Chdir(p string) Persister
	Ls(p string) ([]string, error)
	RawClient() *etcd.Client
}

// a jsonpersistor puts the values as json in etcd
type jsonPersister struct {
	basepath string
	cc       *Cluster
}

// JSONPersister creates a new persister at the given basepath.
func (cc *Cluster) JSONPersister(pt string) (Persister, error) {
	bp := orcaPersistPath + pt
	_, err := cc.client.Get(bp, false, false)
	if err != nil {
		_, err = cc.client.CreateDir(bp, 0)
		if err != nil {
			return nil, errgo.Mask(err)
		}
	}
	return &jsonPersister{basepath: bp, cc: cc}, nil
}

func (jp *jsonPersister) path(k string) string {
	return jp.basepath + "/" + k
}

// Return a new Persister with a new basepath
func (jp *jsonPersister) Chdir(p string) Persister {
	return &jsonPersister{basepath: path.Join(jp.basepath, p), cc: jp.cc}
}

// Returns the full path inside the orac universum :-)
func (jp *jsonPersister) Path(s string) string {
	return jp.path(s)
}

// Read all entries in the given path
func (jp *jsonPersister) Ls(p string) ([]string, error) {
	rsp, err := jp.cc.client.Get(jp.path(p), false, false)
	if err != nil {
		if cerr, ok := err.(*etcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, orcaerr.NotFound(cerr, "cannot ls at '%s'", p)
			}
		}

		return nil, errgo.Mask(err)
	}
	var res []string
	ptlen := len(jp.path(p))
	for _, n := range rsp.Node.Nodes {
		k := n.Key[ptlen:]
		res = append(res, k)
	}
	return res, nil
}

// Return the raw etcd client to do funny things.
func (jp *jsonPersister) RawClient() *etcd.Client {
	return jp.cc.client
}

// Put the value v at the position k.
func (jp *jsonPersister) Put(k string, v interface{}) error {
	return jp.PutTTL(k, 0, v)
}

// PutTTL puts the value v at the position k with a given ttl.
func (jp *jsonPersister) PutTTL(k string, ttl uint64, v interface{}) error {
	b, e := json.Marshal(v)
	if e != nil {
		return errgo.Mask(e)
	}
	_, e = jp.cc.client.Set(jp.path(k), string(b), ttl)
	return errgo.Mask(e)
}

// Get the value at position k.
func (jp *jsonPersister) Get(k string, v interface{}) error {
	n, e := jp.cc.client.Get(jp.path(k), false, false)
	if e != nil {
		if cerr, ok := e.(*etcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return orcaerr.NotFound(cerr, "cannot get at '%s'", k)
			}
		}

		return errgo.Mask(e)
	}
	return json.Unmarshal([]byte(n.Node.Value), v)
}

// Get all values inside the current context. The res must be a pointer
// to an array of the corrent result type.
func (jp *jsonPersister) GetAll(sorted, recursive bool, res interface{}) error {
	ptr := reflect.ValueOf(res)
	targ := reflect.Indirect(ptr)

	arType := targ.Type().Elem()

	vals, e := jp.cc.client.Get(jp.path(""), sorted, recursive)
	if e != nil {
		return errgo.Mask(e)
	}

	for _, n := range vals.Node.Nodes {
		nval := reflect.New(arType)
		err := json.Unmarshal([]byte(n.Value), nval.Interface())
		if err != nil {
			return errgo.Mask(err)
		}
		targ.Set(reflect.Append(targ, reflect.Indirect(nval)))
	}
	return nil
}

// Remove the value at position k.
func (jp *jsonPersister) Remove(k string) error {
	_, e := jp.cc.client.Delete(jp.path(k), false)
	return errgo.Mask(e)
}

func (jp *jsonPersister) RemoveDir(k string) error {
	_, e := jp.cc.client.RawDelete(jp.path(k), true, true)
	return errgo.Mask(e)
}
