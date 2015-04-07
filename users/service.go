package users

import (
	"net/http"
	"strconv"
	"time"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/rest"
	"gopkg.in/emicklei/go-restful.v1"
)

type UsersService struct {
	Auth     auth.Auther
	Provider Users
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

func (t *UsersService) af(f auth.AuthedFunction) restful.RouteFunction {
	return auth.Authed(f, t.Auth)
}

func (t *UsersService) uf(f UserFunction, rlz Roles) restful.RouteFunction {
	return HasRoles(f, t.Auth, t.Provider, rlz)
}

func (t *UsersService) Register(c *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("").To(t.uf(t.createUser, ManagerRoles)).
		Doc("create a user").
		Operation("createUser").
		Reads(User{}).
		Writes(User{}))
	ws.Route(ws.GET("/").To(t.uf(t.getAll, ManagerRoles)).
		Doc("get all registered users").
		Operation("getAll").
		Returns(200, "OK", []User{}))
	ws.Route(ws.GET("/{user-id}").To(t.uf(t.getUser, UserRoles)).
		Doc("retrieves the given user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Operation("getUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.DELETE("/{user-id}").To(t.uf(t.deleteUser, UserRoles)).
		Doc("deletes the given user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Operation("deleteUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/{user-id}").To(t.uf(t.updateUser, UserRoles)).
		Doc("updates the given user's name").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.QueryParameter("name", "new name of the user").DataType("string")).
		Param(ws.QueryParameter("role", "a role of the user").DataType("string").AllowMultiple(true)).
		Operation("updateUser").
		Returns(200, "OK", User{}))
	ws.Route(ws.PATCH("/{user-id}/permit/{duration}").To(t.uf(t.permitUser, ManagerRoles)).
		Doc("permits the user to login the next 'duration' seconds").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.QueryParameter("duration", "time in seconds to allow logins").DataType("string")).
		Operation("permitUser").
		Returns(200, "OK", Allowance{}))
	ws.Route(ws.POST("/{zone}/pubkey").To(t.getUserByKey).
		Doc("retrieves the user with the embedded public key").
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("getUserByKey").
		Reads("").
		Returns(200, "OK", User{}))
	ws.Route(ws.PUT("/{user-id}/{key-id}/{zone}/pubkey").To(t.uf(t.addUserKey, UserRoles)).
		Doc("add the given key to the users list of public keys").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("addUserKey").
		Reads("").
		Returns(200, "OK", Key{}))
	ws.Route(ws.DELETE("/{user-id}/{key-id}/{zone}/pubkey").To(t.uf(t.deleteUserKey, UserRoles)).
		Doc("delete the given key to the users list of public keys").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Param(ws.PathParameter("key-id", "the key-id of the new key").DataType("string")).
		Param(ws.PathParameter("zone", "the zone where to search the user in").DataType("string")).
		Operation("deleteUserKey").
		Returns(200, "OK", Key{}))
	ws.Route(ws.POST("/parsekey").To(t.uf(t.parseKey, UserRoles)).
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
	res, err := t.Provider.Create(u.Id, u.Name, u.Roles)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(res)
}

func (t *UsersService) getAll(me *User, request *restful.Request, response *restful.Response) {
	rest.HandleEntity(t.Provider.GetAll())(request, response)
}

func (t *UsersService) getUser(u *User, request *restful.Request, response *restful.Response) {
	uid := request.PathParameter("user-id")
	if !allowed(u, uid, response) {
		return
	}
	rest.HandleEntity(t.Provider.Get(uid))(request, response)
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
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(k)
}

func (t *UsersService) parseKey(me *User, request *restful.Request, response *restful.Response) {
	var pubk string
	err := request.ReadEntity(&pubk)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	rest.HandleEntity(ParseKey(string(pubk)))(request, response)
}
