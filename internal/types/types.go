package types

import "runtime/debug"

// Version is set at build time via ldflags. Falls back to build info or "dev".
var Version = "dev"

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
}
