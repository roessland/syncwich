package runalyze

import (
	"crypto/tls"
	"os"
	"strings"
)

// buildTLSConfig returns the TLS config the client should use. When insecure
// is true, certificate verification is disabled — useful for routing traffic
// through a local interception proxy (mitmproxy, Charles) while debugging.
func buildTLSConfig(insecure bool) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	}
}

// insecureTLSEnabled reports whether SW_INSECURE_TLS is set to a truthy value.
// Accepts "1", "true" (any case). Anything else is treated as false.
func insecureTLSEnabled() bool {
	switch strings.ToLower(os.Getenv("SW_INSECURE_TLS")) {
	case "1", "true":
		return true
	}
	return false
}
