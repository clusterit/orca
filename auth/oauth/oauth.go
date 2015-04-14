package oauth

import (
	"fmt"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/rest"
	"github.com/clusterit/orca/users"
	"gopkg.in/emicklei/go-restful.v1"
)

const (
	oauthPath = "/oauth"
)

type OauthRegistration struct {
	Network        string `json:"network"`
	ClientId       string `json:"clientid"`
	ClientSecrect  string `json:"clientsecret"`
	Scopes         string `json:"scopes"`
	AuthUrl        string `json:"auth_url"`
	AccessTokenUrl string `json:"accesstoken_url"`
	UserinfoUrl    string `json:"userinfo_url"`
	PathEmail      string `json:"pathemail"`
	PathName       string `json:"pathname"`
	PathPicture    string `json:"pathpicture"`
	PathCover      string `json:"pathcover"`
}

type OAuthRegistry interface {
	Create(network string, clientid, clientsecrect, scopes, authurl, accessurl, userinfourl, pathemail, pathname, pathpicture, pathcover string) (*OauthRegistration, error)
	Delete(network string) (*OauthRegistration, error)
	Get(network string) (*OauthRegistration, error)
	GetAll() ([]OauthRegistration, error)
}

type AuthRegService struct {
	Auth     auth.Auther
	Users    users.Users
	Registry OAuthRegistry
}

type oauthApp struct {
	cc      *etcd.Cluster
	persist etcd.Persister
}

func New(cc *etcd.Cluster) (OAuthRegistry, error) {
	pers, err := cc.NewJsonPersister(oauthPath)
	if err != nil {
		return nil, err
	}
	return &oauthApp{cc, pers}, nil
}

func (a *oauthApp) Get(network string) (*OauthRegistration, error) {
	var res OauthRegistration
	return &res, a.persist.Get(network, &res)
}

func (a *oauthApp) Create(network, clientid, clientsecret, scopes, authurl, accessurl, userinfourl, pathemail, pathname, pathpicture, pathcover string) (*OauthRegistration, error) {
	if network == "" {
		return nil, fmt.Errorf("empty network not allowed")
	}
	reg := OauthRegistration{
		Network:        network,
		ClientId:       clientid,
		ClientSecrect:  clientsecret,
		Scopes:         scopes,
		AuthUrl:        authurl,
		AccessTokenUrl: accessurl,
		UserinfoUrl:    userinfourl,
		PathEmail:      pathemail,
		PathName:       pathname,
		PathPicture:    pathpicture,
		PathCover:      pathcover,
	}
	// if this is a known network and there are empty fields, fill them ...
	reg = fillDefaults(network, reg)
	a.persist.Put(network, reg)
	return &reg, nil
}

func (a *oauthApp) Delete(network string) (*OauthRegistration, error) {
	var res OauthRegistration
	if err := a.persist.Get(network, &res); err != nil {
		return nil, err
	}
	return &res, a.persist.Remove(network)
}

func (a *oauthApp) GetAll() ([]OauthRegistration, error) {
	var res []OauthRegistration
	return res, a.persist.GetAll(true, false, &res)
}

func (t *AuthRegService) Shutdown() {
}

func (t *AuthRegService) Register(root string, c *restful.Container) {

	mgr := users.CheckUser(t.Auth, t.Users, users.ManagerRoles)

	ws := new(restful.WebService)
	ws.
		Path(root + "/authregistry").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("/").To(mgr(t.createReg)).
		Doc("create a oauth registration").
		Operation("createReg").
		Reads(OauthRegistration{}).
		Writes(OauthRegistration{}))
	ws.Route(ws.GET("/").To(mgr(t.getAllRegs)).
		Doc("get all registered oauth registrations").
		Operation("getAllRegs").
		Returns(200, "OK", []OauthRegistration{}))
	ws.Route(ws.DELETE("/{network}").To(mgr(t.deleteReg)).
		Doc("delete the registry for the given network").
		Param(ws.PathParameter("network", "the network name of the registry").DataType("string")).
		Operation("deleteReg").
		Returns(200, "OK", OauthRegistration{}))

	c.Add(ws)
}

func (t *AuthRegService) createReg(me *users.User, request *restful.Request, response *restful.Response) {
	var reg OauthRegistration
	if err := request.ReadEntity(&reg); err != nil {
		rest.HandleError(err, response)
		return
	}
	rest.HandleEntity(t.Registry.Create(
		reg.Network,
		reg.ClientId,
		reg.ClientSecrect,
		reg.Scopes,
		reg.AuthUrl,
		reg.AccessTokenUrl,
		reg.UserinfoUrl,
		reg.PathEmail,
		reg.PathName,
		reg.PathPicture,
		reg.PathCover))(request, response)
}

func (t *AuthRegService) deleteReg(me *users.User, request *restful.Request, response *restful.Response) {
	network := request.PathParameter("network")
	rest.HandleEntity(t.Registry.Delete(network))(request, response)
}

func (t *AuthRegService) getAllRegs(me *users.User, request *restful.Request, response *restful.Response) {
	rest.HandleEntity(t.Registry.GetAll())(request, response)
}
