package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/clusterit/orca/auth"

	"github.com/dgrijalva/jwt-go"
)

var (
	// registry of backends: google, facebook, ...
	// the client sends which one to use
	backends = make(map[string]AuthBackend)
)

// A backend must parse a token and return an AuthUser
type AuthBackend interface {
	Get(token string) (*auth.AuthUser, error)
}

// store the keys in memory ... security issue?
type jwtAuthorizer struct {
	privKey *rsa.PrivateKey
}

// register a backend service for the given network name
func RegisterBackend(name string, be AuthBackend) {
	backends[name] = be
}

// The one and only Auther.  Please create these keypair with openssl or
// something else. Another option is to let orca generate them.
func NewAuther(key *rsa.PrivateKey) auth.Auther {
	return &jwtAuthorizer{privKey: key}
}

func (ja *jwtAuthorizer) parse(value string) (*jwt.Token, error) {
	return jwt.Parse(value, func(token *jwt.Token) (interface{}, error) {
		return ja.privKey.Public(), nil
	})
}

// Create a JWT token for the given authToken inside the given network.
// There must be a registered backend for the network. This backend is used
// to query the AuthUser and this user is wrapped in the JWT token.
func (ja *jwtAuthorizer) Create(network, authToken string) (string, *auth.AuthUser, error) {
	auth, err := backends[network].Get(authToken)
	if err != nil {
		return "", nil, err
	}
	// create a signer for rsa 256
	t := jwt.New(jwt.GetSigningMethod("RS256"))

	t.Claims["AccessToken"] = "orca"
	t.Claims["user"] = *auth
	t.Claims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	tok, err := t.SignedString(ja.privKey)
	return tok, auth, err
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
