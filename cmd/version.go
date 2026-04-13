package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, commit, and build metadata",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(formatVersion(readVersionInfo()))
	},
}

// versionInfo bundles every field the version output cares about so the
// formatting function can be unit-tested without depending on the ambient
// build environment.
type versionInfo struct {
	version string
	commit  string
	date    string
	dirty   bool
	goOS    string
	goArch  string
	goVer   string
}

func readVersionInfo() versionInfo {
	v := versionInfo{
		version: "devel",
		goOS:    runtime.GOOS,
		goArch:  runtime.GOARCH,
		goVer:   runtime.Version(),
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return v
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		v.version = info.Main.Version
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			v.commit = s.Value
		case "vcs.time":
			v.date = s.Value
		case "vcs.modified":
			v.dirty = s.Value == "true"
		}
	}

	return v
}

func formatVersion(v versionInfo) string {
	commit := v.commit
	if len(commit) > 12 {
		commit = commit[:12]
	}
	if v.dirty {
		commit += "-dirty"
	}

	inner := commit
	if v.date != "" {
		if inner != "" {
			inner += ", "
		}
		inner += "built " + v.date
	}
	inner += fmt.Sprintf(", %s/%s, %s", v.goOS, v.goArch, v.goVer)

	return fmt.Sprintf("syncwich %s (%s)", v.version, inner)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
