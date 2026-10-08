package main

import "runtime/debug"

// version is set at build time with -ldflags "-X main.version=vX.Y.Z".
var version string

// buildVersion returns the release version: the ldflags value if set, else
// the module version recorded by "go install", else "dev" for a local build.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
