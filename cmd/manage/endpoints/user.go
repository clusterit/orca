package endpoints

import (
	"github.com/clusterit/orca/user"
	"github.com/emicklei/go-restful"
	"github.com/ulrichSchreiner/authkit"
)

type userService struct {
	backend user.Users
	kit     *authkit.Authkit
}

// NewUserService creates a new rest service with the given service
// as the backend on the basepath given by 'pt'.
func NewUserService(kit *authkit.Authkit, srv user.Users, pt string) *restful.WebService {
	s := userService{srv, kit}
	return s.register(pt)
}

func (u userService) register(pt string) *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(pt + "/users").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(u.findAllUsers).
		// docs
		Doc("get all users").
		Operation("findAllUsers").
		Returns(200, "OK", []user.User{}))

	ws.Route(ws.GET("/{network}/{alias}").To(authed(u.kit, u.findUser)).
		Doc("find a user").
		Operation("findUser").
		Param(ws.PathParameter("network", "name of the network").DataType("string")).
		Param(ws.PathParameter("alias", "alias of the user").DataType("string")).
		Writes(user.User{}))

	ws.Route(ws.GET("/{user-id}").To(u.getUser).
		// docs
		Doc("get a user").
		Operation("findUser").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Writes(user.User{}))

	return ws
}

func (u userService) findAllUsers(request *restful.Request, response *restful.Response) {
}

func (u userService) findUser(ctx *authkit.AuthContext, request *restful.Request, response *restful.Response) {
	network := request.PathParameter("network")
	alias := request.PathParameter("alias")

	usr, e := u.backend.Find(network, alias)
	if e != nil {
		panic(e)
	}
	response.WriteEntity(usr)
}

func (u userService) getUser(request *restful.Request, response *restful.Response) {
}
