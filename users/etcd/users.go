package etcd

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/clusterit/orca/logging"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/etcd"
	. "github.com/clusterit/orca/users"
	etcderr "github.com/coreos/etcd/error"
	goetcd "github.com/coreos/go-etcd/etcd"
	"github.com/dgryski/dgoogauth"
)

const (
	usersPath  = "/users"
	aliasPath  = "/alias"
	keysPath   = "/keys"
	permitPath = "/permit"
	twofaPath  = "/2fa"
)

var (
	logger = logging.Simple()
)

type etcdUsers struct {
	up    etcd.Persister
	kp    etcd.Persister
	pm    etcd.Persister
	al    etcd.Persister
	twofa etcd.Persister
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
	al, e := cl.NewJsonPersister("/data" + aliasPath)
	if e != nil {
		return nil, e
	}
	twofa, e := cl.NewJsonPersister("/data" + twofaPath)
	if e != nil {
		return nil, e
	}
	return &etcdUsers{up: up, kp: kp, pm: pm, al: al, twofa: twofa}, nil
}

func (eu *etcdUsers) key(k *Key) string {
	return strings.Replace(k.Fingerprint, ":", "", -1)
}

func uid(net, id string) string {
	return common.NetworkUser(net, id)
}

func (eu *etcdUsers) RemoveAlias(id, network, alias string) (*User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, e
	}
	auid := uid(network, alias)
	u.Aliases = remove(auid, u.Aliases)
	if err := eu.al.Remove(auid); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) AddAlias(id, network, alias string) (*User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, e
	}
	auid := uid(network, alias)
	u.Aliases = insert(auid, u.Aliases)
	if err := eu.al.Put(auid, u.Id); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) Create(network, id, name string, rlz Roles) (*User, error) {
	usrid := id
	if network != "" {
		usrid = uid(network, id)
	}
	u, e := eu.Get(usrid)
	if e != nil {
		internalid := common.GenerateUUID()
		u = &User{Id: internalid, Name: name, Roles: rlz, Aliases: []string{usrid}}
		// generate an alias for internalid too
		if err := eu.al.Put(internalid, internalid); err != nil {
			return nil, err
		}
	} else {
		u.Name = name
		u.Roles = rlz
		u.Allowance = nil
		u.Aliases = insert(usrid, u.Aliases)
	}
	if err := eu.al.Put(usrid, u.Id); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) GetAll() ([]User, error) {
	var res []User
	return res, eu.up.GetAll(true, false, &res)
}

func (eu *etcdUsers) Get(id string) (*User, error) {
	var u User
	var a Allowance
	var realid string
	// we have an alias for our intenal id too, so the
	// next lookup must always succeed if the user exists
	if err := eu.al.Get(id, &realid); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, common.ErrNotFound
			}
		}
		return nil, err
	}
	if err := eu.up.Get(realid, &u); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, common.ErrNotFound
			}
		}
		return nil, err
	}
	if err := eu.pm.Get(realid, &a); err == nil {
		u.Allowance = &a
	} else {
		u.Allowance = nil
	}
	return &u, nil
}

func (eu *etcdUsers) GetByKey(zone, pubkey string) (*User, *Key, error) {
	var (
		u   User
		uid string
	)
	pk, err := ParseKey(pubkey)
	if err != nil {
		return nil, nil, err
	}
	if err := eu.kp.Get(eu.key(pk), &uid); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, nil, common.ErrNotFound
			}
		}
		return nil, nil, err
	}
	if err := eu.up.Get(uid, &u); err != nil {
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
			} else {
				u.Allowance = nil
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
	if err := eu.kp.Put(eu.key(&k), uid); err != nil {
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
	if ttlSecs == 0 {
		logger.Infof("remove allowance for %s", a.Uid)
		return eu.pm.Remove(a.Uid)
	}
	a.Until = time.Now().UTC().Add(time.Second * time.Duration(ttlSecs))
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

func (eu *etcdUsers) Create2FAToken(zone, domain, uid string) (string, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return "", e
	}
	sec := make([]byte, 6)
	_, err := rand.Read(sec)
	if err != nil {
		return "", err
	}
	encodedSecret := base32.StdEncoding.EncodeToString(sec)
	if err := eu.twofa.Put(uid, encodedSecret); err != nil {
		return "", err
	}
	auth_string := "otpauth://totp/" + u.Name + "@" + domain + "?secret=" + encodedSecret + "&issuer=orca"
	return auth_string, nil
}

func (eu *etcdUsers) CheckAndAllowToken(zone, uid, token string, maxAllowance int) error {
	if err := eu.CheckToken(zone, uid, token); err != nil {
		return err
	}
	u, e := eu.Get(uid)
	if e != nil {
		return e
	}
	permit := u.AutologinAfter2FA
	if maxAllowance < permit {
		permit = maxAllowance
	}
	if permit > 0 {
		a := Allowance{
			GrantedBy: uid,
			Uid:       uid,
			Until:     time.Now(), // will be set in the Permit function
		}
		return eu.Permit(a, uint64(maxAllowance))
	}
	return nil
}

func (eu *etcdUsers) CheckToken(zone, uid, token string) error {
	var secret string
	if err := eu.twofa.Get(uid, &secret); err != nil {
		return err
	}

	otpc := &dgoogauth.OTPConfig{
		Secret:      secret,
		WindowSize:  3,
		HotpCounter: 0,
	}
	ok, err := otpc.Authenticate(token)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid token")
	}
	return nil
}

func (eu *etcdUsers) Use2FAToken(zone, uid string, use bool) error {
	u, e := eu.Get(uid)
	if e != nil {
		return e
	}
	u.Use2FA = use
	if !use {
		eu.twofa.Remove(uid)
	}
	return eu.up.Put(uid, u)
}

func (eu *etcdUsers) SetAutologinAfter2FA(zone, uid string, duration int) (*User, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return nil, e
	}
	u.AutologinAfter2FA = duration
	return u, eu.up.Put(uid, u)
}

func (eu *etcdUsers) Close() error {
	return nil
}

func insert(s string, ar []string) []string {
	m := make(map[string]bool)
	for _, a := range ar {
		m[a] = true
	}
	m[s] = true
	var res []string
	for k, _ := range m {
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
	for k, _ := range m {
		res = append(res, k)
	}
	return res
}
