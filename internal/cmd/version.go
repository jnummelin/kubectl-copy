package cmd

import "runtime/debug"

// version is set at build time for releases, with
// -ldflags "-X github.com/jnummelin/kubectl-copy/internal/cmd.version=...".
var version string

// buildVersion returns the version to report: the release version if one was
// set at build time, else the module version that "go install" records, else
// "dev" for a plain local build.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
