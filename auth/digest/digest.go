package digest

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Authenticator struct {
}

var (
	startTimestamp = time.Now().UnixNano()
)

func New() *Authenticator {
	return &Authenticator{}
}

func (a *Authenticator) Check(header, user, pass string) bool {
	lines := strings.Split(header, "\n")
	if len(lines) == 0 {
		fmt.Println("Error getting header first line")
		return false
	}
	requestLine := lines[0]

	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		fmt.Println("Malformed request line")
		return false
	}

	method := parts[0]
	requestURI := parts[1]

	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", user, "DefaultRealm", pass)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, requestURI)))
	expectedResponse := md5.Sum([]byte(fmt.Sprintf("%x:%s:%x", ha1, generateDailyNonce(), ha2)))
	// Authorization: Digest username="user", realm="DefaultRealm", nonce="178bdf22eafc1836", uri="/", response="a7c53b8d0bf4b66360499198a388fbce"
	return strings.Contains(header, `response="`+hex.EncodeToString(expectedResponse[:])+`"`)
}

func (a *Authenticator) UnauthorizedHeader() string {
	return fmt.Sprintf("HTTP/1.1 401 Unauthorized\r\nWWW-Authenticate: Digest realm=\"DefaultRealm\", nonce=\"%s\"", generateDailyNonce())
}

func generateDailyNonce() string {
	currentDate := time.Now().Format("2006-01-02")            // change nonce every day
	data := fmt.Sprintf("%s:%d", currentDate, startTimestamp) // mix with start timestamp
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
