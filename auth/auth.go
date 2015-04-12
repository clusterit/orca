package auth

import (
	"net/http"

	"github.com/clusterit/orca/rest"

	"gopkg.in/emicklei/go-restful.v1"
)

// An AuthUser is a Uid and a Name. The BackgroundUrl
// and the ThumbnailUrl is optional an can be empty
type AuthUser struct {
	Uid           string `json:"uid"`
	Name          string `json:"name"`
	BackgroundUrl string `json:"backgroundurl"`
	ThumbnailUrl  string `json:"thumbnail"`
}

// A Auther creates an AuthUser from a network and an access_token
// for this network.
type Auther interface {
	// Return a JWT token out of a given auth'd user
	Create(network, authToken string) (string, *AuthUser, error)
	// Read the User out of the JWT token
	Get(token string) (*AuthUser, error)
}

// Pull the "Authorization" header from the request and check if the token
// can be parsed. If true, delegate to the wrapped function, otherwise
// send a unauthorized.
func Authed(wrap AuthedFunction, ath Auther) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("Authorization")
		a, err := ath.Get(token)
		if err != nil {
			response.WriteError(http.StatusUnauthorized, rest.JsonError(err.Error()))
			return
		}

		wrap(a, request, response)
	}
}

// Signature for a restful function which needs an authenticated user
type AuthedFunction func(ath *AuthUser, request *restful.Request, response *restful.Response)

// The rest service
type AutherService struct {
	Auth Auther
}

// Shutdown the Auther
func (t *AutherService) Shutdown() error {
	return nil
}

// Rest interface description
func (t *AutherService) Register(root string, c *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path(root + "/auth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/").To(t.createToken).
		Consumes("application/x-www-form-urlencoded").
		Doc("create a new auth token").
		Param(ws.FormParameter("network", "the auth network").DataType("string")).
		Param(ws.FormParameter("access_token", "the access token for the user data").DataType("string")).
		Returns(200, "OK", AuthUser{}).
		Operation("createToken"))
	ws.Route(ws.GET("/user").To(t.getAuth).
		Doc("get the authenticated user data").
		Operation("getAuth").
		Returns(200, "OK", AuthUser{}))

	c.Add(ws)

}

// REST function to create a JWT token. take the network and the toke out of the request
// and delegate the call to the inside service
func (t *AutherService) createToken(rq *restful.Request, rsp *restful.Response) {
	network := rq.Request.FormValue("network")
	accessToken := rq.Request.FormValue("access_token")
	tok, auth, err := t.Auth.Create(network, accessToken)
	if err != nil {
		rsp.WriteError(http.StatusInternalServerError, err)
		return
	}
	rsp.Header().Add("Authorization", tok)
	rsp.WriteEntity(*auth)
}

// Get the AuthUser from the JWT token.
func (t *AutherService) getAuth(rq *restful.Request, rsp *restful.Response) {
	token := rq.HeaderParameter("Authorization")

	res, err := t.Auth.Get(token)
	if err != nil {
		rsp.WriteError(http.StatusInternalServerError, err)
		return
	}
	rsp.WriteEntity(res)
}
