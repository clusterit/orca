package etcd

import (
	"encoding/json"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/clusterit/orca/common"

	etcderr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

const (
	orcaManagerPath = "/orca/manager"
	orcaPersistPath = "/orca"
)

// a cluster implementation backed by etcd
type Cluster struct {
	client *etcd.Client
}

// A configurator supports operations on a subtree inside of etcd
type Configurator interface {
	Register(pt, value string, ttl int) error
	Unregister(pt string)
	GetValues(pt string) ([]string, error)
}

// Create the cluster by using the etcd-members
func Init(machines []string) (*Cluster, error) {
	client := etcd.NewClient(machines)
	return &Cluster{client}, nil
}

type pathConfigurator struct {
	basepath string
	cc       *Cluster
	stop     map[string]chan bool
	mux      sync.Mutex
}

// create a configurator for a subtree path.
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

// Returns the Configurator for the manager path
func (cc *Cluster) NewManager() (Configurator, error) {
	return cc.NewConfigurator(orcaManagerPath)
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

// Retrieve all registered values at path pt.
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

// A persister can read/write values from/to etcd
type Persister interface {
	Path(s string) string
	Put(k string, v interface{}) error
	PutTtl(k string, ttl uint64, v interface{}) error
	Get(k string, v interface{}) error
	GetAll(sorted, recursive bool, v interface{}) error
	Remove(k string) error
	Chdir(p string) Persister
	Ls(p string) ([]string, error)
	RawClient() *etcd.Client
}

// a jsonpersistor puts the values as json in etcd
type jsonPersister struct {
	basepath string
	cc       *Cluster
}

// Create a new JsonPersister at the given basepath.
func (cc *Cluster) NewJsonPersister(pt string) (Persister, error) {
	bp := orcaPersistPath + pt
	_, err := cc.client.Get(bp, false, false)
	if err != nil {
		_, err = cc.client.CreateDir(bp, 0)
		if err != nil {
			return nil, err
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
				return nil, common.ErrNotFound
			}
		}

		return nil, err
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
	return jp.PutTtl(k, 0, v)
}

// Put the value v at the position k with a given ttl.
func (jp *jsonPersister) PutTtl(k string, ttl uint64, v interface{}) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = jp.cc.client.Set(jp.path(k), string(b), ttl)
	return e
}

// Get the value at position k.
func (jp *jsonPersister) Get(k string, v interface{}) error {
	n, e := jp.cc.client.Get(jp.path(k), false, false)
	if e != nil {
		if cerr, ok := e.(*etcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return common.ErrNotFound
			}
		}

		return e
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
		return e
	}

	for _, n := range vals.Node.Nodes {
		nval := reflect.New(arType)
		err := json.Unmarshal([]byte(n.Value), nval.Interface())
		if err != nil {
			return err
		}
		targ.Set(reflect.Append(targ, reflect.Indirect(nval)))
	}
	return nil
}

// Remove the value at position k.
func (jp *jsonPersister) Remove(k string) error {
	_, e := jp.cc.client.Delete(jp.path(k), false)
	return e
}
