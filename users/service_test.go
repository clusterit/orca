package users

import (
	"log"
	"testing"

	"github.com/clusterit/orca/auth"
	. "github.com/smartystreets/goconvey/convey"
)

type mockauther struct {
	token string
	oauth auth.Token
	usr   *auth.AuthUser
	err   error
}

func (m *mockauther) Create(network, authCode, redirectUrl string) (string, auth.Token, *auth.AuthUser, error) {
	return m.token, m.oauth, m.usr, m.err
}

func (m *mockauther) Get(token string) (*auth.AuthUser, error) {
	return m.usr, m.err
}

func newAuther(t string, o auth.Token, u *auth.AuthUser, e error) auth.Auther {
	return &mockauther{t, o, u, e}
}

type mockusers struct {
	create   func(string, string, string, Roles) (*User, error)
	addalias func(string, string, string) (*User, error)
}

func (m *mockusers) Create(network, id, name string, rolzs Roles) (*User, error) {
	return m.create(network, id, name, rolzs)
}
func (m *mockusers) AddAlias(id, network, alias string) (*User, error) {
	return m.addalias(id, network, alias)
}
func (m *mockusers) NewIdToken(uid string) (*User, error) {
	return nil, nil
}
func (m *mockusers) ByIdToken(idtok string) (*User, error) {
	return nil, nil
}
func (m *mockusers) RemoveAlias(id, network, alias string) (*User, error) {
	return nil, nil
}
func (m *mockusers) GetAll() ([]User, error) {
	return nil, nil
}
func (m *mockusers) Get(id string) (*User, error) {
	return nil, nil
}
func (m *mockusers) AddKey(uid, kid string, pubkey string, fp string) (*Key, error) {
	return nil, nil
}
func (m *mockusers) RemoveKey(uid, kid string) (*Key, error) {
	return nil, nil
}
func (m *mockusers) Update(uid, username string, rolz Roles) (*User, error) {
	return nil, nil
}
func (m *mockusers) Permit(a Allowance, ttlSecs uint64) error {
	return nil
}
func (m *mockusers) Delete(uid string) (*User, error) {
	return nil, nil
}
func (m *mockusers) GetByKey(pubkey string) (*User, *Key, error) {
	return nil, nil, nil
}
func (m *mockusers) Create2FAToken(domain, uid string) (string, error) {
	return "", nil
}
func (m *mockusers) SetAutologinAfter2FA(uid string, duration int) (*User, error) {
	return nil, nil
}
func (m *mockusers) Use2FAToken(uid string, use bool) error {
	return nil
}
func (m *mockusers) CheckToken(uid, token string) error {
	return nil
}
func (m *mockusers) CheckAndAllowToken(uid, token string, maxAllowance int) error {
	return nil
}
func (m *mockusers) Close() error {
	return nil
}

func TestUser(t *testing.T) {
	Convey("create a mock for users backend", t, func() {
		var userimpl mockusers
		authuser := auth.AuthUser{}
		authimpl := newAuther("token", make(auth.Token), &authuser, nil)
		service := UsersService{Auth: authimpl}
		Convey("create a user with", func() {
			log.Printf("%#v, %#v", userimpl, service)
		})
	})
}
