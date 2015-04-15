package basic

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/clusterit/orca/auth"
)

type basicAuther struct {
	verifyCert bool
	httpUrl    string
}

// taken from the stdlib
func parseBasicAuth(token string) (username, password string, ok bool) {
	if !strings.HasPrefix(token, "Basic ") {
		return
	}
	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(token, "Basic "))
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func (g *basicAuther) check(url string, user, pwd string) error {
	if url == "" { // assume no check wanted
		return nil
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !g.verifyCert},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pwd)
	rsp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode >= 200 && rsp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("[%s] : %s", url, rsp.Status)
}

func (g *basicAuther) Get(token string) (*auth.AuthUser, error) {
	// if there is no correct Basic-Auth-Header we simply use "" als user/password
	// to check against the backend. if this succeeds, we assume that there is no
	// auth needed
	u, p, _ := parseBasicAuth(token)
	err := g.check(g.httpUrl, u, p)
	if err == nil {
		u := auth.AuthUser{
			Uid:  u,
			Name: u,
		}
		return &u, nil
	}
	return nil, err
}

func (g *basicAuther) Create(network, authToken, redirectUrl string) (string, auth.Token, *auth.AuthUser, error) {
	u, e := g.Get(authToken)
	return authToken, nil, u, e
}

func NewAuther(url string, verifyCert bool) auth.Auther {
	return &basicAuther{httpUrl: url, verifyCert: verifyCert}
}
