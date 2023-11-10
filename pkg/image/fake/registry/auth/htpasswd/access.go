// Package htpasswd provides a simple authentication scheme that checks for the
// user credential hash in an htpasswd formatted file in a configuration-determined
// location.
//
// This authentication method MUST be used under TLS, as simple token-replay attack is possible.
//
// This code is adapted from https://github.com/distribution/distribution/blob/v2.8.3/registry/auth/htpasswd/access.go
package htpasswd

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/distribution/distribution/v3/registry/auth"
	"k8s.io/apimachinery/pkg/util/sets"
)

type accessController struct {
	realm              string
	htpasswd           *htpasswd
	allowedUsersByRepo map[string]sets.Set[string]
}

var _ auth.AccessController = &accessController{}
var ErrAuthorizationFailure = errors.New("authorization failure")

func newAccessController(options map[string]interface{}) (auth.AccessController, error) {
	realm, present := options["realm"]
	if _, ok := realm.(string); !present || !ok {
		return nil, fmt.Errorf(`"realm" must be set for htpasswd access controller`)
	}

	plainCredentials, present := options["credentials"]
	plainCredentialsMap, ok := plainCredentials.(map[string]string)
	if !present || !ok {
		return nil, fmt.Errorf(`"credentials" must be set for htpasswd access controller`)
	}

	// routesAllowedUsers is a map of routes to users that are allowed to access them
	// if a route is in the map, only users in its values are allowed to access the routes in the map
	// if a route is not in the map, all users are allowed to access it
	allowedUsersByRepo, present := options["allowedUsersByRepo"]
	allowedUsersByRepoMap, ok := allowedUsersByRepo.(map[string]sets.Set[string])
	if !present || !ok {
		return nil, fmt.Errorf(`"routesAllowedUsers" must be set for htpasswd access controller`)
	}
	return &accessController{
		realm:              realm.(string),
		htpasswd:           newHtpasswd(plainCredentialsMap),
		allowedUsersByRepo: allowedUsersByRepoMap,
	}, nil
}

func (ac *accessController) Authorized(req *http.Request, accessRecords ...auth.Access) (*auth.Grant, error) {
	// based on https://github.com/distribution/distribution/v3/registry/handlers/app.go, only one access record is
	// passed in. We use a for loop to cover the case where no access records are available (e.g., when the route is not
	// the one of an image manifest, like ready or healthz).
	var (
		pathRequireAuth bool
		allowedUsers    sets.Set[string]
	)
	for _, accessRecord := range accessRecords {
		allowedUsers, pathRequireAuth = ac.allowedUsersByRepo[accessRecord.Name]
	}
	if !pathRequireAuth {
		// We allow anonymous access to the registry if the repo is not in the allowedUsersByRepo map.
		// When no access records are available, this holds true.
		return &auth.Grant{User: auth.UserInfo{Name: "anonymous"}}, nil
	}
	// The accessRecord/path/repo is in the allowedUsersByRepo map. We check if the user is authenticated and authorized.
	username, password, authHeaderPresent := req.BasicAuth()
	if !authHeaderPresent || ac.htpasswd.authenticateUser(username, password) != nil {
		return nil, &challenge{
			realm: ac.realm,
			err:   auth.ErrAuthenticationFailure,
		}
	}
	// The user is authenticated. We check if the user is authorized.
	if !allowedUsers.Has(username) {
		return nil, &challenge{
			realm: ac.realm,
			err:   ErrAuthorizationFailure,
		}
	}
	// The user is authorized.
	return &auth.Grant{User: auth.UserInfo{Name: username}}, nil
}

// challenge implements the auth.Challenge interface.
type challenge struct {
	realm string
	err   error
}

var _ auth.Challenge = challenge{}

// SetHeaders sets the basic challenge header on the response.
func (ch challenge) SetHeaders(r *http.Request, w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", ch.realm))
}

func (ch challenge) Error() string {
	return fmt.Sprintf("basic authentication challenge for realm %q: %s", ch.realm, ch.err)
}

func init() {
	err := auth.Register("htpasswd_authorization", newAccessController)
	if err != nil {
		log.Fatal(err)
	}
}
