package cmd

import (
	"strings"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name    string
		info    versionInfo
		wantHas []string
	}{
		{
			name: "clean build with all fields",
			info: versionInfo{
				version: "v1.2.3",
				commit:  "abcdef1234567890",
				date:    "2026-04-13T20:00:00Z",
				dirty:   false,
				goOS:    "linux",
				goArch:  "amd64",
				goVer:   "go1.24.0",
			},
			wantHas: []string{"syncwich v1.2.3", "abcdef123456", "2026-04-13T20:00:00Z", "linux/amd64", "go1.24.0"},
		},
		{
			name: "dirty build shows -dirty",
			info: versionInfo{
				version: "devel",
				commit:  "deadbeefcafebabe",
				dirty:   true,
				goOS:    "darwin",
				goArch:  "arm64",
				goVer:   "go1.24.0",
			},
			wantHas: []string{"deadbeefcafe-dirty"},
		},
		{
			name: "short commit not truncated",
			info: versionInfo{
				version: "devel",
				commit:  "abc123",
				goOS:    "linux",
				goArch:  "amd64",
				goVer:   "go1.24.0",
			},
			wantHas: []string{"(abc123"},
		},
		{
			name: "missing commit still renders",
			info: versionInfo{
				version: "devel",
				goOS:    "linux",
				goArch:  "amd64",
				goVer:   "go1.24.0",
			},
			wantHas: []string{"syncwich devel", "linux/amd64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.info)
			for _, needle := range tt.wantHas {
				if !strings.Contains(got, needle) {
					t.Errorf("formatVersion output missing %q\n\nOutput:\n%s", needle, got)
				}
			}
		})
	}
}
