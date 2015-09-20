package endpoints

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ulrichSchreiner/authkit"

	"github.com/emicklei/go-restful"
)

// An AuthFunction is like a RouteFunction but it has an additional
// AuthContext as a parameter
type AuthFunction func(*authkit.AuthContext, *restful.Request, *restful.Response)

func authed(kit *authkit.Authkit, af AuthFunction) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		ctx, err := kit.Context(request.Request)
		if err != nil {
			log.Printf("no valid auth context found: %s", err)
		} else {
			log.Printf("current authenticated user: %#v", ctx.User)
		}
		af(ctx, request, response)
	}
}
