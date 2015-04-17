package auth

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/clusterit/orca/rest"

	"gopkg.in/emicklei/go-restful.v1"
)

// An AuthUser is a Uid and a Name. The BackgroundUrl
// and the ThumbnailUrl is optional an can be empty
type AuthUser struct {
	Network       string `json:"network"`
	Uid           string `json:"uid"`
	Name          string `json:"name"`
	BackgroundUrl string `json:"backgroundurl"`
	ThumbnailUrl  string `json:"thumbnail"`
}

type Token map[string]string

// A Auther creates an AuthUser from a network and an access_token
// for this network.
type Auther interface {
	// Return a JWT token out of a given auth'd user
	Create(network, authCode, redirectUrl string) (string, Token, *AuthUser, error)
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
		Path(root + "auth").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/oauth").To(t.createTokenFromCode).
		Consumes("application/x-www-form-urlencoded").
		Doc("create a new access token").
		Param(ws.FormParameter("state", "the state field of the client").DataType("string")).
		Param(ws.FormParameter("code", "the code sent from the oauth provider").DataType("string")).
		Param(ws.FormParameter("redirect_uri", "the redirect URI").DataType("string")).
		Returns(200, "OK", AuthUser{}).
		Operation("createTokenFromCode"))
	ws.Route(ws.GET("/user").To(t.getAuth).
		Doc("get the authenticated user data").
		Operation("getAuth").
		Returns(200, "OK", AuthUser{}))

	c.Add(ws)

}

func (t *AutherService) createTokenFromCode(rq *restful.Request, rsp *restful.Response) {
	parms := rq.Request.URL.Query()
	state := rq.QueryParameter("state")
	code := rq.QueryParameter("code")
	redirUri := rq.QueryParameter("redirect_uri")
	res := make(map[string]interface{})
	if err := json.Unmarshal([]byte(state), &res); err != nil {
		rsp.WriteError(http.StatusInternalServerError, err)
		return
	}
	network := res["network"].(string)
	tk, oauthtk, _, err := t.Auth.Create(network, code, parms.Get("redirect_uri"))
	if err != nil {
		rsp.WriteError(http.StatusInternalServerError, err)
		return
	}
	redirVals := make(url.Values)
	for k, v := range oauthtk {
		redirVals.Add(k, v)
	}
	redirVals.Add("orca", tk)
	redir := redirUri + "?" + redirVals.Encode() + "&state=" + state
	http.Redirect(rsp.ResponseWriter, rq.Request, redir, http.StatusTemporaryRedirect)
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
