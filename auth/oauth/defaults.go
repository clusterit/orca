package oauth

const (
	googleNetwork = "google"
)

var (
	defaultBackends = map[string]OauthRegistration{
		googleNetwork: OauthRegistration{
			Scopes:         "openid,email,profile,plus.me",
			AuthUrl:        "https://accounts.google.com/o/oauth2/auth",
			AccessTokenUrl: "https://accounts.google.com/o/oauth2/token",
			UserinfoUrl:    "https://www.googleapis.com/plus/v1/people/me",
			PathEmail:      "emails[0].value",
			PathName:       "displayName",
			PathPicture:    "image.url",
			PathCover:      "cover.coverPhoto.url",
		},
	}
)

func getDefaults(backend string) OauthRegistration {
	res, _ := defaultBackends[backend]
	return res
}

func fillDefaults(backend string, reg OauthRegistration) OauthRegistration {
	def := getDefaults(backend)
	if reg.Scopes == "" {
		reg.Scopes = def.Scopes
	}
	if reg.AuthUrl == "" {
		reg.AuthUrl = def.AuthUrl
	}
	if reg.AccessTokenUrl == "" {
		reg.AccessTokenUrl = def.AccessTokenUrl
	}
	if reg.UserinfoUrl == "" {
		reg.UserinfoUrl = def.UserinfoUrl
	}
	if reg.PathEmail == "" {
		reg.PathEmail = def.PathEmail
	}
	if reg.PathName == "" {
		reg.PathName = def.PathName
	}
	if reg.PathPicture == "" {
		reg.PathPicture = def.PathPicture
	}
	if reg.PathCover == "" {
		reg.PathCover = def.PathCover
	}
	return reg
}
