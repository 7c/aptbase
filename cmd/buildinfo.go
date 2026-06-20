package cmd

import "runtime/debug"

// init enriches build metadata from the embedded Go build info. This makes
// `go install github.com/7c/aptbase@v1.2.3` report a real version and commit
// even though it does not run the Makefile's -ldflags injection.
func init() {
	enrichBuildInfo()
	rootCmd.Version = version
}

func enrichBuildInfo() {
	// If -ldflags already set everything, keep those values.
	if version != "dev" && commit != "none" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if commit == "none" && s.Value != "" {
				commit = s.Value
				if len(commit) > 12 {
					commit = commit[:12]
				}
			}
		case "vcs.time":
			if date == "unknown" && s.Value != "" {
				date = s.Value
			}
		}
	}
}
