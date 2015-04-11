package oauth

import (
	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/etcd"
)

type OauthRegistration struct {
	ClientId       string `json:"clientid"`
	ClientSecrect  string `json:"clientsecret"`
	AuthUrl        string `json:"auth_url"`
	AccessTokenUrl string `json:"accesstoken_url"`
}

type Oauther interface {
	Create(network string, clientid, clientsecrect, authurl, accessurl string) (*OauthRegistration, error)
	Delete(network string) error
	Get(network, token string) (*auth.AuthUser, error)
}

type oauthApp struct {
	cc *etcd.Cluster
}

func New(cc *etcd.Cluster) Oauther {
	return &oauthApp{cc}
}

func (a *oauthApp) Get(network, token string) (*auth.AuthUser, error) {
	return nil, nil
}

func (a *oauthApp) Create(network, clientid, clientsecrect, authurl, accessurl string) (*OauthRegistration, error) {
	return nil, nil
}

func (a *oauthApp) Delete(network string) error {
	return nil
}
