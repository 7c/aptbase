package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// debInfo is the package identity parsed from a .deb filename.
type debInfo struct {
	File    string
	Name    string
	Version string
	Arch    string
}

// splitRepoAndFiles separates an optional leading repo argument from the .deb
// file arguments. The repo is present only if the first argument is not itself
// a .deb path.
func splitRepoAndFiles(args []string) (repoArg string, files []string, err error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("need at least one .deb file")
	}
	if !strings.HasSuffix(args[0], ".deb") {
		repoArg = args[0]
		files = args[1:]
	} else {
		files = args
	}
	if len(files) == 0 {
		return "", nil, fmt.Errorf("need at least one .deb file")
	}
	for _, f := range files {
		if !strings.HasSuffix(f, ".deb") {
			return "", nil, fmt.Errorf("%q is not a .deb file", f)
		}
		if _, statErr := os.Stat(f); statErr != nil {
			return "", nil, fmt.Errorf("%s: %w", f, statErr)
		}
	}
	return repoArg, files, nil
}

// reposFor resolves the target repos: the explicit repo argument if given, else
// the configured default repos.
func reposFor(repoArg string) ([]string, error) {
	if repoArg != "" {
		return []string{repoArg}, nil
	}
	if len(settings.Repos) > 0 {
		return settings.Repos, nil
	}
	return nil, errNoRepo
}

// distributionsFor resolves the target distributions, requiring at least one.
func distributionsFor() ([]string, error) {
	if len(settings.Distributions) > 0 {
		return settings.Distributions, nil
	}
	return nil, errNoDist
}

// parseDeb derives package identity from a Debian package filename of the form
// name_version_arch.deb. Returns best-effort values when the format differs.
func parseDeb(path string) debInfo {
	base := strings.TrimSuffix(filepath.Base(path), ".deb")
	info := debInfo{File: filepath.Base(path)}
	parts := strings.Split(base, "_")
	switch len(parts) {
	case 3:
		info.Name, info.Version, info.Arch = parts[0], parts[1], parts[2]
	case 2:
		info.Name, info.Version = parts[0], parts[1]
	default:
		info.Name = base
	}
	return info
}

// newUploadDir returns a unique upload directory name for this invocation.
func newUploadDir() string {
	return fmt.Sprintf("aptbase-%d-%d", os.Getpid(), time.Now().UnixNano())
}
