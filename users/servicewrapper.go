package users

import (
	"net/http"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/rest"
	"gopkg.in/emicklei/go-restful.v1"
)

// Signature for a restful function which needs an authorized user
type UserFunction func(usr *User, request *restful.Request, response *restful.Response)

// Check if the user which is identified by the "Authorization" header
// has as least one of the given roles.
func HasRoles(wrap UserFunction, ath auth.Auther, usrs Users, rlz Roles) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		token := request.HeaderParameter("Authorization")

		a, err := ath.Get(token)
		if err != nil {
			response.WriteError(http.StatusUnauthorized, rest.JsonError(err.Error()))
			return
		}

		hasroles, u, err := hasAuthorizedRoles(a.Network, a.Uid, usrs, rlz)
		if err != nil || !hasroles {
			response.WriteError(http.StatusForbidden, rest.JsonError("not allowed"))
			return
		}
		wrap(u, request, response)
	}
}

// Query the user with the given uid from the users and checks of the user has
// at least one of the given roles. Returns true if the user has one of
// the given roles, otherwise false. Note: A return value of false does not
// imply an error!
func hasAuthorizedRoles(network, uid string, usrs Users, rlz Roles) (bool, *User, error) {
	u, err := usrs.Get(common.NetworkUser(network, uid))
	if err != nil {
		return false, nil, err
	}
	for _, r := range rlz {
		if !hasRole(r, u.Roles) {
			return false, nil, nil
		}
	}
	return true, u, nil
}

// Helper function to check if a role is in a role list
func hasRole(r Role, rlz Roles) bool {
	for _, rl := range rlz {
		if r == rl {
			return true
		}
	}
	return false
}
