package jwt

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/auth/oauth"
	"golang.org/x/oauth2"

	"github.com/dgrijalva/jwt-go"
)

var (
	// registry of backends: google, facebook, ...
	// the client sends which one to use
	backends = make(map[string]AuthBackend)
	arIndex  = regexp.MustCompile(`\[\d\]`)
)

// A backend must parse a token and return an AuthUser
type AuthBackend interface {
	Get(token string) (*auth.AuthUser, error)
}

// store the keys in memory ... security issue?
type jwtAuthorizer struct {
	privKey      *rsa.PrivateKey
	authRegistry oauth.OAuthRegistry
}

// register a backend service for the given network name
func RegisterBackend(name string, be AuthBackend) {
	backends[name] = be
}

// The one and only Auther.  Please create these keypair with openssl or
// something else. Another option is to let orca generate them.
func NewAuther(key *rsa.PrivateKey, registry oauth.OAuthRegistry) auth.Auther {
	return &jwtAuthorizer{privKey: key, authRegistry: registry}
}

func (ja *jwtAuthorizer) parse(value string) (*jwt.Token, error) {
	return jwt.Parse(value, func(token *jwt.Token) (interface{}, error) {
		return ja.privKey.Public(), nil
	})
}

// Create a JWT token for the given authCode inside the given network.
// There must be a registered backend for the network. This backend is used
// to query the AuthUser and this user is wrapped in the JWT token.
func (ja *jwtAuthorizer) Create(network, authCode, redirectUrl string) (string, string, *auth.AuthUser, error) {
	//auth, err := backends[network].Get(authToken)
	auth, oauthtok, err := ja.auth(network, authCode, redirectUrl)
	if err != nil {
		return "", "", nil, err
	}

	// create a signer for rsa 256
	t := jwt.New(jwt.GetSigningMethod("RS256"))

	t.Claims["AccessToken"] = "orca"
	t.Claims["user"] = *auth
	t.Claims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	tok, err := t.SignedString(ja.privKey)
	return tok, oauthtok, auth, err
}

// Pull out the AuthUser from the JWT token.
func (ja *jwtAuthorizer) Get(token string) (*auth.AuthUser, error) {
	t, err := ja.parse(token)
	if err != nil {
		return nil, fmt.Errorf("jwt token cannot be parsed: %s", err)
	}
	ath := t.Claims["user"].(map[string]interface{})
	var a auth.AuthUser
	a.Uid = ath["uid"].(string)
	a.Name = ath["name"].(string)
	a.BackgroundUrl = ath["backgroundurl"].(string)
	a.ThumbnailUrl = ath["thumbnail"].(string)
	return &a, nil
}

func (ja *jwtAuthorizer) auth(network, code, redirectUrl string) (*auth.AuthUser, string, error) {
	reg, err := ja.authRegistry.Get(network)
	if err != nil {
		return nil, "", err
	}
	conf := &oauth2.Config{
		ClientID:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		Scopes:       strings.Split(reg.Scopes, ","),
		RedirectURL:  redirectUrl,
		Endpoint: oauth2.Endpoint{
			AuthURL:  reg.AuthUrl,
			TokenURL: reg.AccessTokenUrl,
		},
	}
	tok, err := conf.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, "", err
	}
	tokval, err := json.Marshal(tok)
	if err != nil {
		return nil, "", err
	}
	client := conf.Client(oauth2.NoContext, tok)
	rsp, err := client.Get(reg.UserinfoUrl)
	if err != nil {
		return nil, "", err
	}
	defer rsp.Body.Close()

	dat, err := parse(rsp.Body)
	if err != nil {
		return nil, "", err
	}

	var res auth.AuthUser
	log.Printf("data: %#v", dat)
	v, err := getValue(reg.PathId, dat)
	if err != nil {
		return nil, "", fmt.Errorf("cannot get email: %s", err)
	} else {
		res.Uid = v
	}
	v, err = getValue(reg.PathName, dat)
	if err != nil {
		return nil, "", fmt.Errorf("cannot get name: %s", err)
	} else {
		res.Name = v
	}
	if reg.PathCover != "" {
		v, err = getValue(reg.PathCover, dat)
		if err != nil {
			return nil, "", fmt.Errorf("cannot get cover: %s", err)
		} else {
			res.BackgroundUrl = v
		}
	}
	if reg.PathPicture != "" {
		v, err = getValue(reg.PathPicture, dat)
		if err != nil {
			return nil, "", fmt.Errorf("cannot get picture: %s", err)
		} else {
			res.ThumbnailUrl = v
		}
	}
	return &res, string(tokval), nil
}

func parse(r io.Reader) (map[string]interface{}, error) {
	m := make(map[string]interface{})

	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func getValue(path string, data map[string]interface{}) (string, error) {
	target := data
	var res string
	parts := strings.Split(path, ".")
	for idx, p := range parts {
		val, err := getSimpleValue(p, target)
		if err != nil {
			return "", err
		}
		if idx < len(parts)-1 {
			target = val.(map[string]interface{})
		} else {
			res = val.(string)
		}
	}
	return res, nil
}

func getSimpleValue(v string, data map[string]interface{}) (interface{}, error) {
	loc := arIndex.FindStringIndex(v)
	if loc != nil {
		index64, err := strconv.ParseInt(v[loc[0]+1:loc[1]-1], 10, 0)
		if err != nil {
			return nil, err
		}
		indx := int(index64)
		key := v[0:loc[0]]
		res := data[key].([]interface{})
		return res[indx], nil
	}
	log.Printf("return value %v\n", data[v])
	return data[v], nil
}
