package runalyze

import (
	"crypto/tls"
	"testing"
)

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name         string
		insecure     bool
		wantInsecure bool
	}{
		{"secure by default", false, false},
		{"insecure when opted in", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := buildTLSConfig(tt.insecure)
			if cfg.MinVersion != tls.VersionTLS12 {
				t.Errorf("MinVersion = %x, want %x", cfg.MinVersion, tls.VersionTLS12)
			}
			if cfg.InsecureSkipVerify != tt.wantInsecure {
				t.Errorf("InsecureSkipVerify = %v, want %v", cfg.InsecureSkipVerify, tt.wantInsecure)
			}
		})
	}
}

func TestInsecureTLSEnabled_FromEnv(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
	}

	for _, c := range cases {
		t.Run("env="+c.val, func(t *testing.T) {
			t.Setenv("SW_INSECURE_TLS", c.val)
			if got := insecureTLSEnabled(); got != c.want {
				t.Errorf("insecureTLSEnabled() = %v for %q, want %v", got, c.val, c.want)
			}
		})
	}
}
