package basic

import (
	"encoding/base64"
	"strings"
)

type Authenticator struct {
}

func New() *Authenticator {
	return &Authenticator{}
}

func (a *Authenticator) Check(header, user, pass string) bool {
	credentials := user + ":" + pass
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(credentials))
	return strings.Contains(header, "Authorization: Basic "+encodedCredentials)
}

func (a *Authenticator) UnauthorizedHeader() string {
	return "HTTP/1.1 401 Unauthorized\r\nWWW-Authenticate: Basic realm=\"DefaultRealm\""
}
