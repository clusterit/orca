// A backend for Google-accounts. You must register an application with
// Google's cloud console and specifiy the environment variables
// GOOGLE_CLIENTID and GOOGLE_CLIENTSECRET. This module then makes
// oauth2 requests to google to query the data from the user.
package google

import (
	"encoding/json"
	"os"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/jwt"

	"golang.org/x/oauth2"
)

import "log"

const (
	googleNetwork = "google"
	userinfoUrl   = "https://www.googleapis.com/plus/v1/people/me"
)

type googleBackend struct {
	config *oauth2.Config
}

type email struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}
type image struct {
	Url string `json:"url"`
}
type cover struct {
	CoverPhoto *image `json:"coverPhoto"`
}
type userinfo struct {
	EMails   []email `json:"emails"`
	Id       string  `json:"id"`
	Name     string  `json:"displayName"`
	Language string  `json:"language"`
	Image    *image  `json:"image"`
	Cover    *cover  `json:"cover"`
	Gender   string  `json:"gender"`
}

func init() {
	gid := os.Getenv("GOOGLE_CLIENTID")
	gcs := os.Getenv("GOOGLE_CLIENTSECRET")
	if gid == "" || gcs == "" {
		log.Printf("[DEBUG] no backend for google configured")
		return
	}
	conf := &oauth2.Config{
		ClientID:     gid,
		ClientSecret: gcs,
		Scopes:       []string{"openid", "email", "profile", "plus.me"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://accounts.google.com/o/oauth2/token",
		},
	}
	gb := &googleBackend{config: conf}
	jwt.RegisterBackend(googleNetwork, gb)
}

func (g *googleBackend) Get(token string) (*auth.AuthUser, error) {
	tok := &oauth2.Token{AccessToken: token}
	client := g.config.Client(oauth2.NoContext, tok)
	rsp, err := client.Get(userinfoUrl)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	var u userinfo
	if err := json.NewDecoder(rsp.Body).Decode(&u); err != nil {
		return nil, err
	}
	tu := ""
	if u.Image != nil {
		tu = u.Image.Url
	}
	bi := ""
	if u.Cover != nil && u.Cover.CoverPhoto != nil {
		bi = u.Cover.CoverPhoto.Url
	}
	return &auth.AuthUser{Uid: u.EMails[0].Value, Name: u.Name, ThumbnailUrl: tu, BackgroundUrl: bi}, nil
}
