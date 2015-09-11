package user

import (
	"strings"
	"time"

	"github.com/clusterit/orca/utils"

	"golang.org/x/crypto/ssh"
)

const (
	// UserRole defines a normal user
	UserRole Role = "USER"
	// ManagerRole is a manager
	ManagerRole = "MANAGER"
)

type (
	// Role is a role a user can have
	Role string
	// Roles is an array of roles
	Roles []Role

	// A User can login via the SSH gateway or via the webapp
	User struct {
		ID                string     `json:"id"`
		Name              string     `json:"name"`
		Keys              []Key      `json:"keys"`
		Roles             Roles      `json:"roles"`
		Aliases           []string   `json:"aliases"`
		Use2FA            bool       `json:"use2fa"`
		AutologinAfter2FA int        `json:"autologinafter2FA"`
		Allowance         *Allowance `json:"allowance,omitempty"`
		IDToken           string     `json:"idtoken"`
	}

	// A Key is a public part of a key pair
	Key struct {
		ID          string `json:"id"`
		Value       string `json:"value"`
		Fingerprint string `json:"fingerprint"`
	}

	// An Allowance grants a user for a specific time to log in
	Allowance struct {
		GrantedBy string    `json:"grantedBy"`
		UID       string    `json:"uid"`
		Until     time.Time `json:"until"`
	}
)

// String returns a commaseparated string which contains all roles
func (rlz Roles) String() string {
	sr := make([]string, len(rlz))
	for i, r := range rlz {
		sr[i] = string(r)
	}
	return strings.Join(sr, ",")
}

// Has checks if a specific role is contained in the roles
func (rlz Roles) Has(r Role) bool {
	for _, rl := range rlz {
		if rl == r {
			return true
		}
	}
	return false
}

// ParseKey parses a given string given in the format of a "authorized_key"
// file. If the key can be parsed it will be returned, otherwise an Error
// will be returned
func ParseKey(pubkey string) (*Key, error) {
	pk, c, _, _, err := ssh.ParseAuthorizedKey([]byte(pubkey))
	if err != nil {
		return nil, err
	}
	fp := utils.Fingerprint(pk)
	k := Key{ID: c, Value: strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pk))), Fingerprint: fp}
	return &k, nil
}

/*
func AsKey(usrs Users, uid, kid, pubkey string) (*Key, error) {
	k, err := ParseKey(pubkey)
	if err != nil {
		return nil, err
	}
	if kid != "" {
		k.Id = kid
	}
	return usrs.AddKey(uid, k.Id, k.Value, k.Fingerprint)
}
*/
