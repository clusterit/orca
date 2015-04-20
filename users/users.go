package users

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Role string
type Roles []Role

const (
	RoleUser    Role = "USER"
	RoleManager      = "MANAGER"
)

var (
	UserRoles    = []Role{RoleUser}
	ManagerRoles = []Role{RoleUser, RoleManager}
)

type User struct {
	Id                string     `json:"id"`
	Name              string     `json:"name"`
	Keys              []Key      `json:"keys"`
	Roles             Roles      `json:"roles"`
	Aliases           []string   `json:"aliases"`
	Use2FA            bool       `json:"use2fa"`
	AutologinAfter2FA int        `json:"autologinafter2FA"`
	Allowance         *Allowance `json:"allowance,omitempty"`
}

type Key struct {
	Id          string `json:"id"`
	Value       string `json:"value"`
	Fingerprint string `json:"fingerprint"`
}

type Allowance struct {
	GrantedBy string    `json:"grantedBy"`
	Uid       string    `json:"uid"`
	Until     time.Time `json:"until"`
}

type Users interface {
	Create(network, id, name string, rolzs Roles) (*User, error)
	AddAlias(id, network, alias string) (*User, error)
	RemoveAlias(id, network, alias string) (*User, error)
	GetAll() ([]User, error)
	Get(id string) (*User, error)
	AddKey(zone string, uid, kid string, pubkey string, fp string) (*Key, error)
	RemoveKey(zone string, uid, kid string) (*Key, error)
	Update(uid, username string, rolz Roles) (*User, error)
	Permit(a Allowance, ttlSecs uint64) error
	Delete(uid string) (*User, error)
	GetByKey(zone string, pubkey string) (*User, *Key, error)
	Create2FAToken(zone, uid string) (string, error)
	SetAutologinAfter2FA(zone, uid string, duration int) (*User, error)
	Use2FAToken(zone, uid string, use bool) error
	CheckToken(zone, uid, token string) error
	CheckAndAllowToken(zone, uid, token string, maxAllowance int) error
	Close() error
}

func (rlz Roles) String() string {
	sr := make([]string, len(rlz))
	for i, r := range rlz {
		sr[i] = string(r)
	}
	return strings.Join(sr, ",")
}

func (rlz Roles) Has(r Role) bool {
	for _, rl := range rlz {
		if rl == r {
			return true
		}
	}
	return false
}

func Fingerprint(k ssh.PublicKey) string {
	hash := md5.Sum(k.Marshal())
	return strings.Replace(fmt.Sprintf("% x", hash), " ", ":", -1)
}

func ParseKey(pubkey string) (*Key, error) {
	pk, c, _, _, err := ssh.ParseAuthorizedKey([]byte(pubkey))
	if err != nil {
		return nil, err
	}
	fp := Fingerprint(pk)
	k := Key{Id: c, Value: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk))), Fingerprint: fp}
	return &k, nil
}

func AsKey(usrs Users, zone, uid, kid, pubkey string) (*Key, error) {
	k, err := ParseKey(pubkey)
	if err != nil {
		return nil, err
	}
	if kid != "" {
		k.Id = kid
	}
	return usrs.AddKey(zone, uid, k.Id, k.Value, k.Fingerprint)
}
