// Package secrets flags paths that look like credentials before they are staged.
package secrets

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`(?i)(^|/)(id_(rsa|dsa|ecdsa|ed25519)|\.env(\..*)?|\.netrc|\.pgpass|credentials|[^/]*\.(pem|key|p12|pfx)|[^/]*(token|secret|password)[^/]*|\.gnupg/.*)$`)

var safeSuffix = []string{".pub", ".example", ".sample", ".md"}

// Looks reports whether p looks like a secret (private key, token, .env, ...).
func Looks(p string) bool {
	for _, s := range safeSuffix {
		if strings.HasSuffix(p, s) {
			return false
		}
	}
	return re.MatchString(p)
}
