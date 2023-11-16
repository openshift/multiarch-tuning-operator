package htpasswd

//This code is adapted from https://github.com/distribution/distribution/blob/v2.8.3/registry/auth/htpasswd/htpasswd.go

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/distribution/distribution/v3/registry/auth"
)

// htpasswd holds a map entries that stores the credentials. Only bcrypt hash entries are supported.
type htpasswd struct {
	entries map[string][]byte // maps username to password byte slice.
}

// AuthenticateUser checks a given user:password credential against the
// receiving HTPasswd's file. If the check passes, nil is returned.
func (htpasswd *htpasswd) authenticateUser(username string, password string) error {
	credentials, ok := htpasswd.entries[username]
	if !ok {
		return auth.ErrAuthenticationFailure
	}

	err := bcrypt.CompareHashAndPassword(credentials, []byte(password))
	if err != nil {
		return auth.ErrAuthenticationFailure
	}

	return nil
}

func newHtpasswd(userPwEntries map[string]string) *htpasswd {
	entries := make(map[string][]byte, len(userPwEntries))
	for user, pw := range userPwEntries {
		entries[user], _ = bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	}
	return &htpasswd{
		entries: entries,
	}
}
