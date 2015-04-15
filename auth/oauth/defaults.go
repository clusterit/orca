package oauth

const (
	googleNetwork = "google"
	githubNetwork = "github"
)

var (
	defaultBackends = map[string]OauthRegistration{
		googleNetwork: OauthRegistration{
			Scopes:         "https://www.googleapis.com/auth/userinfo.email,https://www.googleapis.com/auth/userinfo.profile,https://www.googleapis.com/auth/plus.me",
			AuthUrl:        "https://accounts.google.com/o/oauth2/auth",
			AccessTokenUrl: "https://accounts.google.com/o/oauth2/token",
			UserinfoUrl:    "https://www.googleapis.com/plus/v1/people/me",
			PathId:         "emails[0].value",
			PathName:       "displayName",
			PathPicture:    "image.url",
			PathCover:      "cover.coverPhoto.url",
		},
		githubNetwork: OauthRegistration{
			Scopes:         "user:email",
			AuthUrl:        "https://github.com/login/oauth/authorize",
			AccessTokenUrl: "https://github.com/login/oauth/access_token",
			UserinfoUrl:    "https://api.github.com/user",
			PathId:         "login",
			PathName:       "name",
			PathPicture:    "avatar_url",
			PathCover:      "",
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
	if reg.PathId == "" {
		reg.PathId = def.PathId
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
