package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clusterit/orca/auth"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/emicklei/go-restful.v1"
)

const (
	jsonType = "application/json"
)

var (
	usermap = map[string]User{
		"myid":    User{Id: "myid", Name: "myname", Roles: UserRoles},
		"adminid": User{Id: "adminid", Name: "admin name", Roles: ManagerRoles},
		"user1":   User{Id: "user1", Name: "User 1", Roles: UserRoles},
		"user2":   User{Id: "user2", Name: "User 2", Roles: UserRoles, Aliases: []string{"user2@google"}},
	}
	client http.Client
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
	create               func(string, string, string, Roles) (*User, error)
	addalias             func(string, string, string) (*User, error)
	byidtoken            func(string) (*User, error)
	get                  func(string) (*User, error)
	getall               func() ([]User, error)
	removealias          func(string, string, string) (*User, error)
	delete               func(string) (*User, error)
	update               func(string, string, Roles) (*User, error)
	newidtoken           func(string) (*User, error)
	permit               func(Allowance, uint64) error
	getbykey             func(string) (*User, *Key, error)
	addkey               func(string, string, string, string) (*Key, error)
	removekey            func(string, string) (*Key, error)
	setautologinafter2fa func(string, int) (*User, error)
	checkandallowtoken   func(string, string, int) error
	checktoken           func(string, string) error
}

func (m *mockusers) Create(network, id, name string, rolzs Roles) (*User, error) {
	return m.create(network, id, name, rolzs)
}
func (m *mockusers) AddAlias(id, network, alias string) (*User, error) {
	return m.addalias(id, network, alias)
}
func (m *mockusers) NewIdToken(uid string) (*User, error) {
	return m.newidtoken(uid)
}
func (m *mockusers) ByIdToken(idtok string) (*User, error) {
	return m.byidtoken(idtok)
}
func (m *mockusers) RemoveAlias(id, network, alias string) (*User, error) {
	return m.removealias(id, network, alias)
}
func (m *mockusers) GetAll() ([]User, error) {
	return m.getall()
}
func (m *mockusers) Get(id string) (*User, error) {
	return m.get(id)
}
func (m *mockusers) AddKey(uid, kid string, pubkey string, fp string) (*Key, error) {
	return m.addkey(uid, kid, pubkey, fp)
}
func (m *mockusers) RemoveKey(uid, kid string) (*Key, error) {
	return m.removekey(uid, kid)
}
func (m *mockusers) Update(uid, username string, rolz Roles) (*User, error) {
	return m.update(uid, username, rolz)
}
func (m *mockusers) Permit(a Allowance, ttlSecs uint64) error {
	return m.permit(a, ttlSecs)
}
func (m *mockusers) Delete(uid string) (*User, error) {
	return m.delete(uid)
}
func (m *mockusers) GetByKey(pubkey string) (*User, *Key, error) {
	return m.getbykey(pubkey)
}
func (m *mockusers) Create2FAToken(domain, uid string) (string, error) {
	return "", nil
}
func (m *mockusers) SetAutologinAfter2FA(uid string, duration int) (*User, error) {
	return m.setautologinafter2fa(uid, duration)
}
func (m *mockusers) Use2FAToken(uid string, use bool) error {
	return nil
}
func (m *mockusers) CheckToken(uid, token string) error {
	return m.checktoken(uid, token)
}
func (m *mockusers) CheckAndAllowToken(uid, token string, maxAllowance int) error {
	return m.checkandallowtoken(uid, token, maxAllowance)
}
func (m *mockusers) Close() error {
	return nil
}

func newUsers() Users {
	var userimpl mockusers
	userimpl.byidtoken = func(tok string) (*User, error) {
		u := usermap[tok]
		return &u, nil
	}
	userimpl.get = userimpl.byidtoken
	userimpl.getall = func() ([]User, error) {
		var res []User
		for _, u := range usermap {
			res = append(res, u)
		}
		return res, nil
	}
	userimpl.create = func(nt, id, nam string, r Roles) (*User, error) {
		return &User{Id: id + "-added", Name: nam, Roles: r}, nil
	}
	userimpl.addalias = func(id, netw, alias string) (*User, error) {
		u := usermap[id]
		u.Aliases = append(u.Aliases, alias+"@"+netw)
		return &u, nil
	}
	userimpl.removealias = func(id, netw, alias string) (*User, error) {
		u := usermap[id]
		u.Aliases = []string{}
		return &u, nil
	}
	userimpl.delete = func(id string) (*User, error) {
		u, ok := usermap[id]
		if !ok {
			return nil, fmt.Errorf("unknown userid %s", id)
		}
		return &User{Id: id + "-removed", Name: u.Name, Roles: u.Roles}, nil
	}
	userimpl.update = func(id, name string, r Roles) (*User, error) {
		_, ok := usermap[id]
		if !ok {
			return nil, fmt.Errorf("unknown userid %s", id)
		}
		return &User{Id: id, Name: name, Roles: r}, nil
	}
	userimpl.newidtoken = func(uid string) (*User, error) {
		u, ok := usermap[uid]
		if !ok {
			return nil, fmt.Errorf("unknown userid %s", uid)
		}
		u.IdToken = "anewtoken"
		return &u, nil
	}
	userimpl.permit = func(a Allowance, ttl uint64) error {
		return nil
	}
	userimpl.checktoken = func(uid, token string) error {
		if token == "wrongtoken" {
			return fmt.Errorf("illegal token")
		}
		return nil
	}
	userimpl.checkandallowtoken = func(uid, token string, ttl int) error {
		if token == "wrongtoken" {
			return fmt.Errorf("illegal token")
		}
		return nil
	}
	userimpl.getbykey = func(pubkey string) (*User, *Key, error) {
		if testpk_pubkey == pubkey {
			k, e := ParseKey(pubkey)
			u := usermap["myid"]
			return &u, k, e
		}
		return nil, nil, fmt.Errorf("unknown key")
	}
	userimpl.addkey = func(uid, kid, pubk, fp string) (*Key, error) {
		k, e := ParseKey(pubk)
		k.Id = kid + "-added"
		return k, e
	}
	userimpl.removekey = func(uid, kid string) (*Key, error) {
		if kid == "toremove" {
			k, e := ParseKey(testpk_pubkey)
			k.Id = kid + "-removed"
			return k, e
		}
		return nil, fmt.Errorf("wrong key id")
	}
	userimpl.setautologinafter2fa = func(uid string, duration int) (*User, error) {
		u, ok := usermap[uid]
		if !ok {
			return nil, fmt.Errorf("unknown userid %s", uid)
		}
		u.AutologinAfter2FA = duration
		return &u, nil
	}
	return &userimpl
}

func createRequest(ts *httptest.Server, meth, url string, uid string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer

	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	rq, err := http.NewRequest(meth, ts.URL+url, &buf)
	if err != nil {
		return nil, err
	}
	rq.Header.Add("Content-Type", jsonType)
	rq.Header.Add("Accept", jsonType)
	rq.Header.Add("X-Orca-Token", uid)

	return client.Do(rq)
}

func TestUserServices(t *testing.T) {
	Convey("create a mock for users backend", t, func() {
		userimpl := newUsers()
		authuser := auth.AuthUser{}
		authimpl := newAuther("token", make(auth.Token), &authuser, nil)
		service := UsersService{Auth: authimpl, Provider: userimpl}
		c := restful.NewContainer()
		service.Register("/api/", c)
		ts := httptest.NewServer(c)
		defer ts.Close()

		Convey("try to read all users als normal user", func() {
			res, err := createRequest(ts, "GET", "/api/users", "myid", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusForbidden)
			Convey("and now as a manager", func() {
				res, err := createRequest(ts, "GET", "/api/users", "adminid", nil)
				So(err, ShouldBeNil)
				So(res.StatusCode, ShouldEqual, http.StatusOK)
				var resuser []User
				err = json.NewDecoder(res.Body).Decode(&resuser)
				So(err, ShouldBeNil)
				So(len(resuser), ShouldEqual, len(usermap))
			})
		})
		Convey("create a new user ", func() {
			toCreate := User{Id: "newid", Name: "newname", Roles: UserRoles}
			res, err := createRequest(ts, "PUT", "/api/users/mysocialnet", "adminid", toCreate)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			So(resuser.Id, ShouldEqual, toCreate.Id+"-added")
			So(resuser.Name, ShouldEqual, toCreate.Name)
			So(resuser.Roles, ShouldResemble, toCreate.Roles)
		})
		Convey("delete a user ", func() {
			res, err := createRequest(ts, "DELETE", "/api/users/myid", "adminid", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			So("myid-removed", ShouldEqual, resuser.Id)
		})
		Convey("delete a user as a normal user", func() {
			res, err := createRequest(ts, "DELETE", "/api/users/myid", "myid", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusForbidden)
		})
		Convey("update a user ", func() {
			res, err := createRequest(ts, "PATCH", "/api/users/myid?name=john&role=USER&role=MANAGER", "adminid", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			So(resuser.Id, ShouldEqual, "myid")
			So(resuser.Name, ShouldEqual, "john")
			So(resuser.Roles, ShouldResemble, Roles(ManagerRoles))
		})
		Convey("create an alias", func() {
			res, err := createRequest(ts, "PUT", "/api/users/alias/google/userid", "myid", nil)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["myid"]
			So(resuser.Id, ShouldEqual, u.Id)
			So(resuser.Name, ShouldEqual, u.Name)
			So(resuser.Roles, ShouldResemble, u.Roles)
			So(resuser.Aliases, ShouldContain, "userid@google")
		})
		Convey("delete an alias", func() {
			res, err := createRequest(ts, "DELETE", "/api/users/alias/google/user2", "user2", nil)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["user2"]
			So(resuser.Id, ShouldEqual, u.Id)
			So(resuser.Name, ShouldEqual, u.Name)
			So(resuser.Roles, ShouldResemble, u.Roles)
			So(resuser.Aliases, ShouldNotContain, "user2@google")
		})
		Convey("check if /me returns the current user", func() {
			res, err := createRequest(ts, "GET", "/api/users/me", "user2", nil)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["user2"]
			So(resuser.Id, ShouldEqual, u.Id)
			So(resuser.Name, ShouldEqual, u.Name)
			So(resuser.Roles, ShouldResemble, u.Roles)
			So(resuser.Aliases, ShouldResemble, u.Aliases)
		})
		Convey("generate a new idToken", func() {
			res, err := createRequest(ts, "PATCH", "/api/users/idtoken", "user2", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["user2"]
			So(resuser.Id, ShouldEqual, u.Id)
			So(resuser.Name, ShouldEqual, u.Name)
			So(resuser.Roles, ShouldResemble, u.Roles)
			So(resuser.IdToken, ShouldEqual, "anewtoken")
		})
		Convey("grant a allowance for a specific time", func() {
			now := time.Now()
			res, err := createRequest(ts, "PATCH", "/api/users/permit/300", "user2", nil)
			So(err, ShouldBeNil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var a Allowance
			err = json.NewDecoder(res.Body).Decode(&a)
			So(err, ShouldBeNil)
			So(a.GrantedBy, ShouldEqual, "user2")
			So(a.Uid, ShouldEqual, "user2")
			So(a.Until, ShouldHappenBetween, now.Add(300*time.Second), now.Add(302*time.Second))
		})
		Convey("query user by a public key", func() {
			res, err := createRequest(ts, "POST", "/api/users/pubkey", "user2", testpk_pubkey)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["myid"]
			So(resuser.Id, ShouldEqual, u.Id)
			So(resuser.Name, ShouldEqual, u.Name)
			So(resuser.Roles, ShouldResemble, u.Roles)
			So(resuser.Aliases, ShouldResemble, u.Aliases)
		})
		Convey("add a public key", func() {
			res, err := createRequest(ts, "PUT", "/api/users/newkeyid/pubkey", "user2", testpk2_pubkey)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var k Key
			err = json.NewDecoder(res.Body).Decode(&k)
			So(err, ShouldBeNil)
			So(k.Id, ShouldEqual, "newkeyid-added")
			So(k.Fingerprint, ShouldEqual, testpk_fp)
		})
		Convey("remove a public key", func() {
			res, err := createRequest(ts, "DELETE", "/api/users/toremove/pubkey", "user2", testpk_pubkey)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var k Key
			err = json.NewDecoder(res.Body).Decode(&k)
			So(err, ShouldBeNil)
			So(k.Id, ShouldEqual, "toremove-removed")
			So(k.Fingerprint, ShouldEqual, testpk_fp)
		})
		Convey("check if autologinafter2fa calls the corresponding function", func() {
			res, err := createRequest(ts, "PATCH", "/api/users/autologin2fa/30", "user2", nil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			var u User
			err = json.NewDecoder(res.Body).Decode(&u)
			So(err, ShouldBeNil)
			So(u.AutologinAfter2FA, ShouldEqual, 30)
		})
		Convey("checktokens", func() {
			res, _ := createRequest(ts, "GET", "/api/users/user2/token/check", "user2", nil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			res, _ = createRequest(ts, "GET", "/api/users/user2/wrongtoken/check", "user2", nil)
			So(res.StatusCode, ShouldEqual, http.StatusForbidden)
			res, _ = createRequest(ts, "GET", "/api/users/user2/token/check?maxtime=100", "user2", nil)
			So(res.StatusCode, ShouldEqual, http.StatusOK)
			res, _ = createRequest(ts, "GET", "/api/users/user2/token/check?maxtime=a00", "user2", nil)
			So(res.StatusCode, ShouldEqual, http.StatusInternalServerError)
		})
	})
}
