package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	create      func(string, string, string, Roles) (*User, error)
	addalias    func(string, string, string) (*User, error)
	byidtoken   func(string) (*User, error)
	get         func(string) (*User, error)
	getall      func() ([]User, error)
	removealias func(string, string, string) (*User, error)
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
		return &User{Id: id, Name: nam, Roles: r}, nil
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
			So(toCreate.Id, ShouldEqual, resuser.Id)
			So(toCreate.Name, ShouldEqual, resuser.Name)
			So(toCreate.Roles, ShouldResemble, resuser.Roles)
		})
		Convey("create an alias", func() {
			res, err := createRequest(ts, "PUT", "/api/users/alias/google/userid", "myid", nil)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["myid"]
			So(u.Id, ShouldEqual, resuser.Id)
			So(u.Name, ShouldEqual, resuser.Name)
			So(u.Roles, ShouldResemble, resuser.Roles)
			So(resuser.Aliases, ShouldContain, "userid@google")
		})
		Convey("delete an alias", func() {
			res, err := createRequest(ts, "DELETE", "/api/users/alias/google/user2", "user2", nil)
			So(err, ShouldBeNil)
			var resuser User
			err = json.NewDecoder(res.Body).Decode(&resuser)
			So(err, ShouldBeNil)
			u := usermap["user2"]
			So(u.Id, ShouldEqual, resuser.Id)
			So(u.Name, ShouldEqual, resuser.Name)
			So(u.Roles, ShouldResemble, resuser.Roles)
			So(u.Aliases, ShouldContain, "user2@google")
			So(resuser.Aliases, ShouldNotContain, "user2@google")
		})
	})
}
