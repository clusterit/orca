package etcdstore

import (
	"strings"

	"gopkg.in/errgo.v1"

	orcaerr "github.com/clusterit/orca/errors"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/user"
	"github.com/clusterit/orca/utils"
	etcderr "github.com/coreos/etcd/error"
	goetcd "github.com/coreos/go-etcd/etcd"
)

const (
	basePath   = "/data"
	usersPath  = basePath + "/users"
	aliasPath  = basePath + "/alias"
	keysPath   = basePath + "/keys"
	permitPath = basePath + "/permit"
	twofaPath  = basePath + "/2fa"
	idtoksPath = basePath + "/idtoks"
)

type etcdUsers struct {
	up     etcd.Persister
	kp     etcd.Persister
	pm     etcd.Persister
	al     etcd.Persister
	twofa  etcd.Persister
	idtoks etcd.Persister

	// used for testing of 2FA
	scratchCodes []int
}

// New returns a new users service implemented with a etcd cluster
func New(cl *etcd.Cluster) (user.Users, error) {
	up, e := cl.JSONPersister(usersPath)
	if e != nil {
		return nil, e
	}
	kp, e := cl.JSONPersister(keysPath)
	if e != nil {
		return nil, e
	}
	pm, e := cl.JSONPersister(permitPath)
	if e != nil {
		return nil, e
	}
	al, e := cl.JSONPersister(aliasPath)
	if e != nil {
		return nil, e
	}
	twofa, e := cl.JSONPersister(twofaPath)
	if e != nil {
		return nil, e
	}
	idtoks, e := cl.JSONPersister(idtoksPath)
	if e != nil {
		return nil, e
	}
	return &etcdUsers{up: up, kp: kp, pm: pm, al: al, twofa: twofa, idtoks: idtoks}, nil
}

func (eu *etcdUsers) key(k *user.Key) string {
	return strings.Replace(k.Fingerprint, ":", "", -1)
}

func uid(net, id string) string {
	return utils.NetworkUID(net, id)
}

func (eu *etcdUsers) RemoveAlias(id, network, alias string) (*user.User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, e
	}
	auid := uid(network, alias)
	u.Aliases = remove(auid, u.Aliases)
	if err := eu.al.Remove(auid); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.ID, u)
}

func (eu *etcdUsers) AddAlias(id, network, alias string) (*user.User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, errgo.Mask(e)
	}
	auid := uid(network, alias)
	u.Aliases = insert(auid, u.Aliases)
	if err := eu.al.Put(auid, u.ID); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.ID, u)
}

func (eu *etcdUsers) Create(network, alias, name string, rlz ...user.Role) (*user.User, error) {
	usrid := uid(network, alias)
	u, e := eu.Get(usrid)
	if e != nil {
		internalid := utils.GenerateUUID()
		u = &user.User{ID: internalid, Name: name, Roles: rlz, Aliases: []string{usrid}}
		// generate an alias for internalid too
		if err := eu.al.Put(internalid, internalid); err != nil {
			return nil, err
		}
	} else {
		// 'alias@network' already exists, do an update
		u.Name = name
		u.Roles = rlz
		u.Aliases = insert(usrid, u.Aliases)
	}
	if err := eu.al.Put(usrid, u.ID); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.ID, u)
}

func (eu *etcdUsers) Find(network, alias string) (*user.User, error) {
	usrid := uid(network, alias)
	return eu.Get(usrid)
}

func (eu *etcdUsers) GetAll() ([]user.User, error) {
	var res []user.User
	return res, eu.up.GetAll(true, false, &res)
}

func (eu *etcdUsers) Get(uid string) (*user.User, error) {
	var u user.User
	var realid string
	// we have an alias for our intenal id too, so the
	// next lookup must always succeed if the user exists
	if err := eu.al.Get(uid, &realid); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, orcaerr.NotFound(cerr, "'%s' not found", uid)
			}
		}
		return nil, errgo.Mask(err)
	}
	if err := eu.up.Get(realid, &u); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, orcaerr.NotFound(cerr, "'%s' not found", realid)
			}
		}
		return nil, errgo.Mask(err)
	}
	return &u, nil
}

func (eu *etcdUsers) AddKey(uid, kid string, pubkey string, fp string) (*user.Key, error) {
	k := user.Key{ID: kid, Fingerprint: fp, Value: pubkey}
	u, err := eu.Get(uid)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	uid = u.ID
	var found *user.Key
	for _, k := range u.Keys {
		if k.Fingerprint == fp {
			found = &k
			break
		}
	}
	if found != nil {
		// key with same FP already exists, do nothing
		return found, nil
	}
	u.Keys = append(u.Keys, k)
	if err := eu.up.Put(uid, &u); err != nil {
		return nil, errgo.Mask(err)
	}
	if err := eu.kp.Put(eu.key(&k), uid); err != nil {
		// we should put the old user back ... i'm too lazy now
		return nil, errgo.Mask(err)
	}
	return &k, nil
}

func (eu *etcdUsers) RemoveKey(uid, kid string) (*user.Key, error) {
	u, err := eu.Get(uid)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	uid = u.ID
	var newkeys []user.Key
	var found user.Key

	for i, k := range u.Keys {
		if k.ID != kid {
			newkeys = append(newkeys, u.Keys[i])
		} else {
			found = k
		}
	}
	u.Keys = newkeys
	if err := eu.kp.Remove(eu.key(&found)); err != nil {
		return nil, errgo.Mask(err)
	}
	return &found, eu.up.Put(uid, &u)
}

func (eu *etcdUsers) Update(uid, username string, rolz user.Roles) (*user.User, error) {
	u, err := eu.Get(uid)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	u.Name = username
	u.Roles = rolz
	return u, eu.up.Put(u.ID, u)
}

func (eu *etcdUsers) Delete(uid string) (*user.User, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return nil, errgo.Mask(e)
	}
	uid = u.ID
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, errgo.Mask(err)
	}
	for _, k := range u.Keys {
		if err := eu.kp.Remove(eu.key(&k)); err != nil {
			return nil, errgo.Mask(err)
		}
	}
	eu.pm.Remove(uid)
	return u, eu.up.Remove(uid)
}

func insert(s string, ar []string) []string {
	m := make(map[string]bool)
	for _, a := range ar {
		m[a] = true
	}
	m[s] = true
	var res []string
	for k := range m {
		res = append(res, k)
	}
	return res
}

func remove(s string, ar []string) []string {
	m := make(map[string]bool)
	for _, a := range ar {
		m[a] = true
	}
	delete(m, s)
	var res []string
	for k := range m {
		res = append(res, k)
	}
	return res
}
