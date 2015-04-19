package users

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"code.google.com/p/rsc/qr"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/rest"
	"gopkg.in/emicklei/go-restful.v1"
)

type UsersService struct {
	Auth     auth.Auther
	Provider Users
}

type CheckedUser func(f UserFunction) restful.RouteFunction

func CheckUser(a auth.Auther, u Users, rlz Roles) CheckedUser {
	return func(f UserFunction) restful.RouteFunction {
		return HasRoles(f, a, u, rlz)
	}
}

func roles(sroles []string) Roles {
	var r Roles
	for _, s := range sroles {
		switch s {
		case string(RoleUser):
			r = append(r, RoleUser)
		case string(RoleManager):
			r = append(r, RoleManager)
		}
	}
	return r
}

func (t *UsersService) Shutdown() error {
	return t.Provider.Close()
}

func (t *UsersService) Register(root string, c *restful.Container) {
	manager := CheckUser(t.Auth, t.Provider, ManagerRoles)
	userRoles := CheckUser(t.Auth, t.Provider, UserRoles)

	ws := new(restful.WebService)
	ws.
		Path(root + "users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("/{network}").To(manager(t.createUser)).
		Doc("create a user").
		Operation("createUser").
		Param(ws.PathParameter("network", "identifier the provider for the user").DataType("string")).
		Reads(User{}).
		Writes(User{}))
	ws.Route(ws.PUT("/alias/{user-id}/{network}/{alias}").To(userRoles(t.addAlias)).
		Doc("add an alias to user").
		Operation("add Alias").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("network", "identifier the provider for the alias").DataType("string")).
		Param(ws.PathParameter("alias", "identifier of the alias").DataType("string")).
		Writes(User{}))
	ws.Route(ws.DELETE("/alias/{user-id}/{network}/{alias}").To(userRoles(t.removeAlias)).
		Doc("remove an alias from a user").
		Operation("remove Alias").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("network", "identifier the provider for the alias").DataType("string")).
		Param(ws.PathParameter("alias", "identifier of the alias").DataType("string")).
		Writes(User{}))
	ws.Route(ws.GET("/").To(manager(t.getAll)).
		Doc("get all registered users").
		Operation("getAll").
		Returns(200, "OK", []User{}))
	ws.Route(ws.GET("/me").To(userRoles(t.getUser)).
		Doc("retrieves the current authenticated user").
		Operation("getUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.DELETE("/{user-id}").To(userRoles(t.deleteUser)).
		Doc("deletes the given user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Operation("deleteUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/{user-id}").To(userRoles(t.updateUser)).
		Doc("updates the given user's name").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.QueryParameter("name", "new name of the user").DataType("string")).
		Param(ws.QueryParameter("role", "a role of the user").DataType("string").AllowMultiple(true)).
		Operation("updateUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/{user-id}/permit/{duration}").To(manager(t.permitUser)).
		Doc("permits the user to login the next 'duration' seconds").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("duration", "time in seconds to allow logins").DataType("string")).
		Operation("permitUser").
		Returns(200, "OK", Allowance{}))
	ws.Route(ws.POST("/{zone}/pubkey").To(t.getUserByKey).
		Doc("retrieves the user with the embedded public key").
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("getUserByKey").
		Reads("").
		Returns(200, "OK", User{}))
	ws.Route(ws.PUT("/{user-id}/{key-id}/{zone}/pubkey").To(userRoles(t.addUserKey)).
		Doc("add the given key to the users list of public keys").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("addUserKey").
		Reads("").
		Returns(200, "OK", Key{}))
	ws.Route(ws.DELETE("/{user-id}/{key-id}/{zone}/pubkey").To(userRoles(t.deleteUserKey)).
		Doc("delete the given key to the users list of public keys").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("deleteUserKey").
		Returns(200, "OK", Key{}))
	ws.Route(ws.GET("/2fatoken").To(userRoles(t.gen2FAtoken)).
		Doc("generates a 2FA token for the current user and returns an PNG encoded image with the secret").
		Operation("gen2FAtoken").
		Reads("").
		Returns(200, "OK", []byte{}))
	ws.Route(ws.PATCH("/2fa/{usage}/{token}").To(userRoles(t.use2fa)).
		Doc("stores a flag if the user wants 2fa").
		Param(ws.PathParameter("usage", "enables or disables 2fa").DataType("string")).
		Param(ws.PathParameter("token", "the token to validate the request").DataType("string")).
		Operation("use2fa").
		Reads("").
		Returns(200, "OK", User{}))
	ws.Route(ws.POST("/parsekey").To(userRoles(t.parseKey)).
		Doc("retrieves the user with the embedded public key").
		Operation("parseKey").
		Reads("").
		Returns(200, "OK", Key{}))

	c.Add(ws)
}

func allowed(me *User, uid string, rsp *restful.Response) bool {
	if me.Id != uid && !me.Roles.Has(RoleManager) {
		rsp.WriteError(http.StatusForbidden, rest.JsonError("not allowed"))
		return false
	}
	return true
}

func (t *UsersService) createUser(me *User, request *restful.Request, response *restful.Response) {
	var u User
	err := request.ReadEntity(&u)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	network := request.PathParameter("network")
	if network == common.OrcaPrefix {
		// we have an internal ID, do update
		usr, e := t.Provider.Get(u.Id)
		if e != nil {
			rest.HandleError(e, response)
			return
		}
		u.Id = usr.Id
		network = ""
	}
	res, err := t.Provider.Create(network, u.Id, u.Name, u.Roles)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	response.WriteEntity(res)
}

func (t *UsersService) getAll(me *User, request *restful.Request, response *restful.Response) {
	rest.HandleEntity(t.Provider.GetAll())(request, response)
}

func (t *UsersService) getUser(u *User, request *restful.Request, response *restful.Response) {
	response.WriteEntity(*u)
}

func (t *UsersService) deleteUser(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	if !allowed(me, uid, response) {
		return
	}
	rest.HandleEntity(t.Provider.Delete(uid))(request, response)
}

func (t *UsersService) updateUser(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	name := request.QueryParameter("name")
	rlz := request.Request.Form["roles"]
	if !allowed(me, uid, response) {
		return
	}
	rest.HandleEntity(t.Provider.Update(uid, name, roles(rlz)))(request, response)
}

func (t *UsersService) permitUser(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	dur := request.PathParameter("duration")
	dr, err := strconv.ParseInt(dur, 10, 64)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	until := time.Now().UTC().Add(time.Second * time.Duration(dr))
	a := Allowance{GrantedBy: me.Id, Uid: uid, Until: until}
	err = t.Provider.Permit(a, uint64(dr))
	rest.HandleEntity(a, err)(request, response)
}

func (t *UsersService) getUserByKey(request *restful.Request, response *restful.Response) {
	var pubk string
	zone := request.PathParameter("zone")
	err := request.ReadEntity(&pubk)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	u, _, e := t.Provider.GetByKey(zone, pubk)
	rest.HandleEntity(u, e)(request, response)
}

func (t *UsersService) addUserKey(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	kid := request.PathParameter("key-id")
	zone := request.PathParameter("zone")
	if !allowed(me, uid, response) {
		return
	}
	var pubk string
	err := request.ReadEntity(&pubk)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	_, _, e := t.Provider.GetByKey(zone, string(pubk))
	if e == nil {
		response.WriteError(http.StatusInternalServerError, rest.JsonError("Key already exists"))
		return
	}

	k, err := AsKey(t.Provider, zone, uid, kid, string(pubk))
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(k)
}

func (t *UsersService) deleteUserKey(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	kid := request.PathParameter("key-id")
	zone := request.PathParameter("zone")
	if !allowed(me, uid, response) {
		return
	}
	k, err := t.Provider.RemoveKey(zone, uid, kid)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	response.WriteEntity(k)
}

func (t *UsersService) addAlias(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	alias := request.PathParameter("alias")
	network := request.PathParameter("network")
	rest.HandleEntity(t.Provider.AddAlias(uid, network, alias))(request, response)
}

func (t *UsersService) removeAlias(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	alias := request.PathParameter("alias")
	network := request.PathParameter("network")
	rest.HandleEntity(t.Provider.RemoveAlias(uid, network, alias))(request, response)
}

func (t *UsersService) use2fa(me *User, request *restful.Request, response *restful.Response) {
	usage := request.PathParameter("usage")
	token := request.PathParameter("token")
	usg, err := strconv.ParseBool(usage)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	if err := t.Provider.CheckToken(me.Id, token); err != nil {
		rest.HandleError(err, response)
		return
	}
	if err := t.Provider.Use2FAToken(me.Id, usg); err != nil {
		rest.HandleError(err, response)
		return
	}

	response.WriteEntity(me)
}

func (t *UsersService) gen2FAtoken(me *User, request *restful.Request, response *restful.Response) {
	sec, e := t.Provider.Create2FAToken(me.Id)
	if e != nil {
		rest.HandleError(e, response)
		return
	}
	// todo: add some company/zone/... data to support more installations
	auth_string := "otpauth://totp/orca:" + url.QueryEscape(me.Name) + "?secret=" + sec + "&issuer=orca"
	code, e := qr.Encode(auth_string, qr.L)
	if e != nil {
		rest.HandleError(e, response)
		return
	}
	response.WriteEntity(code.PNG())
}

func (t *UsersService) parseKey(me *User, request *restful.Request, response *restful.Response) {
	var pubk string
	err := request.ReadEntity(&pubk)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	rest.HandleEntity(ParseKey(string(pubk)))(request, response)
}
