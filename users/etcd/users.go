package etcd

import (
	"strings"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/etcd"
	. "github.com/clusterit/orca/users"
	etcderr "github.com/coreos/etcd/error"
	goetcd "github.com/coreos/go-etcd/etcd"
)

const (
	usersPath  = "/users"
	keysPath   = "/keys"
	permitPath = "/permit"
)

type etcdUsers struct {
	up etcd.Persister
	kp etcd.Persister
	pm etcd.Persister
}

func New(cl *etcd.Cluster) (Users, error) {
	up, e := cl.NewJsonPersister("/data" + usersPath)
	if e != nil {
		return nil, e
	}
	kp, e := cl.NewJsonPersister("/data" + keysPath)
	if e != nil {
		return nil, e
	}
	pm, e := cl.NewJsonPersister("/data" + permitPath)
	if e != nil {
		return nil, e
	}
	return &etcdUsers{up: up, kp: kp, pm: pm}, nil
}

func (eu *etcdUsers) key(k *Key) string {
	return strings.Replace(k.Fingerprint, ":", "", -1)
}

func (eu *etcdUsers) Create(id, name string, rlz Roles) (*User, error) {
	u, e := eu.Get(id)
	if e != nil {
		u = &User{Id: id, Name: name, Roles: rlz}
	} else {
		u.Name = name
		u.Roles = rlz
		u.Allowance = nil
	}
	return u, eu.up.Put(id, u)
}

func (eu *etcdUsers) GetAll() ([]User, error) {
	var res []User
	return res, eu.up.GetAll(true, false, &res)
}

func (eu *etcdUsers) Get(id string) (*User, error) {
	var u User
	var a Allowance
	if err := eu.up.Get(id, &u); err != nil {
		return nil, err
	}
	if err := eu.pm.Get(id, &a); err == nil {
		u.Allowance = &a
	}
	return &u, nil
}

func (eu *etcdUsers) GetByKey(zone, pubkey string) (*User, *Key, error) {
	var u User
	pk, err := ParseKey(pubkey)
	if err != nil {
		return nil, nil, err
	}
	if err := eu.kp.Get(eu.key(pk), &u); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, nil, common.ErrNotFound
			}
		}
		return nil, nil, err
	}
	for _, k := range u.Keys {
		if pk.Value == k.Value {
			var a Allowance
			if err := eu.pm.Get(u.Id, &a); err == nil {
				u.Allowance = &a
			}
			return &u, &k, nil
		}
	}
	return nil, nil, common.ErrNotFound
}

func (eu *etcdUsers) AddKey(zone, uid, kid string, pubkey string, fp string) (*Key, error) {
	k := Key{Id: kid, Fingerprint: fp, Value: pubkey}
	var u User
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, err
	}
	u.Keys = append(u.Keys, k)
	if err := eu.up.Put(uid, &u); err != nil {
		return nil, err
	}
	if err := eu.kp.Put(eu.key(&k), &u); err != nil {
		// we should put the old user back ... i'm too lazy now
		return nil, err
	}
	return &k, nil
}

func (eu *etcdUsers) RemoveKey(zone, uid, kid string) (*Key, error) {
	var u User
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, err
	}
	var newkeys []Key
	var found Key

	for i, k := range u.Keys {
		if k.Id != kid {
			newkeys = append(newkeys, u.Keys[i])
		} else {
			found = k
		}
	}
	u.Keys = newkeys
	if err := eu.kp.Remove(eu.key(&found)); err != nil {
		return nil, err
	}
	return &found, eu.up.Put(uid, &u)
}

func (eu *etcdUsers) Update(uid, username string, rolz Roles) (*User, error) {
	var u User
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, err
	}
	u.Name = username
	u.Roles = rolz
	return &u, eu.up.Put(uid, &u)
}

func (eu *etcdUsers) Permit(a Allowance, ttlSecs uint64) error {
	return eu.pm.PutTtl(a.Uid, ttlSecs, &a)
}

func (eu *etcdUsers) Delete(uid string) (*User, error) {
	var u User
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, err
	}
	for _, k := range u.Keys {
		if err := eu.kp.Remove(eu.key(&k)); err != nil {
			return nil, err
		}
	}
	eu.pm.Remove(uid)
	return &u, eu.up.Remove(uid)
}

func (eu *etcdUsers) Close() error {
	return nil
}
