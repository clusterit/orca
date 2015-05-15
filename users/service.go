package users

import (
	"net/http"
	"strconv"
	"time"

	"code.google.com/p/rsc/qr"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/rest"
	"gopkg.in/emicklei/go-restful.v1"
)

type UsersService struct {
	Auth     auth.Auther
	Provider Users
	Config   config.Configer
}

type CheckedUser func(f UserFunction) restful.RouteFunction

func CheckUser(a auth.Auther, u Users, rlz Roles, cfg config.Configer) CheckedUser {
	return func(f UserFunction) restful.RouteFunction {
		return HasRoles(f, a, u, rlz, cfg)
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
	manager := CheckUser(t.Auth, t.Provider, ManagerRoles, nil)
	userRoles := CheckUser(t.Auth, t.Provider, UserRoles, nil)
	// the next rolechecker would create the user if he does not exist
	userRolesAutoCreate := CheckUser(t.Auth, t.Provider, UserRoles, t.Config)

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
	ws.Route(ws.PUT("/alias/{network}/{alias}").To(userRoles(t.addAlias)).
		Doc("add an alias to user").
		Operation("add Alias").
		Param(ws.PathParameter("network", "identifier the provider for the alias").DataType("string")).
		Param(ws.PathParameter("alias", "identifier of the alias").DataType("string")).
		Writes(User{}))
	ws.Route(ws.DELETE("/alias/{network}/{alias}").To(userRoles(t.removeAlias)).
		Doc("remove an alias from a user").
		Operation("remove Alias").
		Param(ws.PathParameter("network", "identifier the provider for the alias").DataType("string")).
		Param(ws.PathParameter("alias", "identifier of the alias").DataType("string")).
		Writes(User{}))
	ws.Route(ws.GET("/").To(manager(t.getAll)).
		Doc("get all registered users").
		Operation("getAll").
		Returns(200, "OK", []User{}))
	ws.Route(ws.GET("/me").To(userRolesAutoCreate(t.getUser)).
		Doc("retrieves the current authenticated user").
		Operation("getUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.DELETE("/{user-id}").To(manager(t.deleteUser)).
		Doc("deletes the given user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Operation("deleteUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/{user-id}").To(manager(t.updateUser)).
		Doc("updates the given user's name").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.QueryParameter("name", "new name of the user").DataType("string")).
		Param(ws.QueryParameter("role", "a role of the user").DataType("string").AllowMultiple(true)).
		Operation("updateUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/idtoken").To(userRoles(t.updateUserIdToken)).
		Doc("generate a new id-token for the current user").
		Operation("updateUserIdToken").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/permit/{duration}").To(userRoles(t.permitUser)).
		Doc("permits the user to login the next 'duration' seconds").
		Param(ws.PathParameter("duration", "time in seconds to allow logins").DataType("string")).
		Operation("permitUser").
		Returns(200, "OK", Allowance{}))
	ws.Route(ws.POST("/pubkey").To(t.getUserByKey).
		Doc("retrieves the user with the embedded public key").
		Operation("getUserByKey").
		Reads("").
		Returns(200, "OK", User{}))
	ws.Route(ws.PUT("/{key-id}/pubkey").To(userRoles(t.addUserKey)).
		Doc("add the given key to the users list of public keys").
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Operation("addUserKey").
		Reads("").
		Returns(200, "OK", Key{}))
	ws.Route(ws.DELETE("/{key-id}/pubkey").To(userRoles(t.deleteUserKey)).
		Doc("delete the given key from the users list of public keys").
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Operation("deleteUserKey").
		Returns(200, "OK", Key{}))
	ws.Route(ws.GET("/2fatoken").To(userRoles(t.gen2FAtoken)).
		Doc("generates a 2FA token for the current user and returns an PNG encoded image with the secret").
		Operation("gen2FAtoken").
		Reads("").
		Returns(200, "OK", []byte{}))
	ws.Route(ws.GET("/{user-id}/{token}/check").To(t.checkToken).
		Doc("checks a 2FA token for the given user-id and permits an autologin within the user configured time only if there is a maxtime parameter; without this parameter, this is only a one-time grant").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("token", "the token to validate the request").DataType("string")).
		Param(ws.QueryParameter("maxtime", "the maximum number of seconds for the autologin").DataType("int")).
		Operation("checkToken").
		Reads("").
		Returns(200, "OK", ""))
	ws.Route(ws.PATCH("/2fa/{usage}/{token}").To(userRoles(t.use2fa)).
		Doc("stores a flag if the user wants 2fa").
		Param(ws.PathParameter("usage", "enables or disables 2fa").DataType("string")).
		Param(ws.PathParameter("token", "the token to validate the request").DataType("string")).
		Operation("use2fa").
		Reads("").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/autologin2fa/{duration}").To(userRoles(t.autologin2fa)).
		Doc("updates the duration for which a 2FA is not necessary").
		Param(ws.PathParameter("duration", "the duration in seconds within a new OTP is not requred").DataType("int")).
		Operation("autologin2fa").
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
	if allowed(me, uid, response) {
		rest.HandleEntity(t.Provider.Delete(uid))(request, response)
	}
}

func (t *UsersService) updateUser(me *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	name := request.QueryParameter("name")
	rlz := request.Request.Form["role"]
	if allowed(me, uid, response) {
		rest.HandleEntity(t.Provider.Update(uid, name, roles(rlz)))(request, response)
	}
}

func (t *UsersService) updateUserIdToken(me *User, request *restful.Request, response *restful.Response) {
	rest.HandleEntity(t.Provider.NewIdToken(me.Id))(request, response)
}

func (t *UsersService) permitUser(me *User, request *restful.Request, response *restful.Response) {
	dur := request.PathParameter("duration")
	dr, err := strconv.ParseInt(dur, 10, 64)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	until := time.Now().UTC().Add(time.Second * time.Duration(dr))
	a := Allowance{GrantedBy: me.Id, Uid: me.Id, Until: until}
	err = t.Provider.Permit(a, uint64(dr))
	rest.HandleEntity(a, err)(request, response)
}

func (t *UsersService) getUserByKey(request *restful.Request, response *restful.Response) {
	var pubk string
	err := request.ReadEntity(&pubk)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	u, _, e := t.Provider.GetByKey(pubk)
	rest.HandleEntity(u, e)(request, response)
}

func (t *UsersService) addUserKey(me *User, request *restful.Request, response *restful.Response) {
	kid := request.PathParameter("key-id")
	var pubk string
	err := request.ReadEntity(&pubk)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	_, _, e := t.Provider.GetByKey(string(pubk))
	if e == nil {
		response.WriteError(http.StatusInternalServerError, rest.JsonError("Key already exists"))
		return
	}

	k, err := AsKey(t.Provider, me.Id, kid, string(pubk))
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(k)
}

func (t *UsersService) deleteUserKey(me *User, request *restful.Request, response *restful.Response) {
	kid := request.PathParameter("key-id")
	k, err := t.Provider.RemoveKey(me.Id, kid)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	response.WriteEntity(k)
}

func (t *UsersService) addAlias(me *User, request *restful.Request, response *restful.Response) {
	alias := request.PathParameter("alias")
	network := request.PathParameter("network")
	rest.HandleEntity(t.Provider.AddAlias(me.Id, network, alias))(request, response)
}

func (t *UsersService) removeAlias(me *User, request *restful.Request, response *restful.Response) {
	alias := request.PathParameter("alias")
	network := request.PathParameter("network")
	rest.HandleEntity(t.Provider.RemoveAlias(me.Id, network, alias))(request, response)
}

func (t *UsersService) use2fa(me *User, request *restful.Request, response *restful.Response) {
	usage := request.PathParameter("usage")
	token := request.PathParameter("token")
	usg, err := strconv.ParseBool(usage)
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	if usg {
		// only check the token if we enable the 2FA
		if err := t.Provider.CheckToken(me.Id, token); err != nil {
			rest.HandleError(err, response)
			return
		}
	}
	if err := t.Provider.Use2FAToken(me.Id, usg); err != nil {
		rest.HandleError(err, response)
		return
	}

	me.Use2FA = usg
	response.WriteEntity(me)
}

func (t *UsersService) checkToken(request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	token := request.PathParameter("token")
	maxtime := request.QueryParameter("maxtime")
	if maxtime != "" {
		maxt, e := strconv.ParseInt(maxtime, 10, 0)
		if e != nil {
			rest.HandleError(e, response)
			return
		}
		if err := t.Provider.CheckAndAllowToken(uid, token, int(maxt)); err != nil {
			response.WriteError(http.StatusForbidden, rest.JsonError(err.Error()))
			return
		}
	} else {
		if err := t.Provider.CheckToken(uid, token); err != nil {
			response.WriteError(http.StatusForbidden, rest.JsonError(err.Error()))
			return
		}
		response.WriteEntity("")
	}
}

func (t *UsersService) gen2FAtoken(me *User, request *restful.Request, response *restful.Response) {
	cluster, e := t.Config.Cluster()
	if e != nil {
		rest.HandleError(e, response)
		return
	}
	sec, e := t.Provider.Create2FAToken(cluster.Name, me.Id)
	if e != nil {
		rest.HandleError(e, response)
		return
	}
	code, e := qr.Encode(sec, qr.L)
	if e != nil {
		rest.HandleError(e, response)
		return
	}
	response.WriteEntity(code.PNG())
}

func (t *UsersService) autologin2fa(me *User, request *restful.Request, response *restful.Response) {
	dur := request.PathParameter("duration")
	duration, err := strconv.ParseInt(dur, 10, 0)
	if err != nil {
		rest.HandleError(err, response)
		return
	}

	// when changin the autologin-value, remove all current
	// allowed-blocks.
	a := Allowance{GrantedBy: me.Id, Uid: me.Id, Until: time.Now()}
	// ignore error here. if there is no current allow-instance we get a notfound here
	t.Provider.Permit(a, 0)

	rest.HandleEntity(t.Provider.SetAutologinAfter2FA(me.Id, int(duration)))(request, response)
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
