// Command increaseversion bumps the semantic version in version.txt — the
// single source of truth for aptbase's build version (see docs/version.md).
//
// Usage:
//
//	go run ./tools/increaseversion.go [patch|minor|major]
//
// It defaults to a patch bump. Run it at the end of a successful release so the
// next build gets a fresh version, then tag the commit to match (e.g.
// `git tag v$(cat version.txt)`) so `go install ...@latest` reports the same
// version.
//
// This lives under tools/ in package main, so it is never compiled into the
// aptbase binary and adds no dependency to consumers.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const path = "version.txt"

func main() {
	part := "patch"
	if len(os.Args) > 1 {
		part = os.Args[1]
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		fail("reading %s: %v", path, err)
	}
	major, minor, patch, err := parse(strings.TrimSpace(string(raw)))
	if err != nil {
		fail("%v", err)
	}

	switch part {
	case "major":
		major, minor, patch = major+1, 0, 0
	case "minor":
		minor, patch = minor+1, 0
	case "patch":
		patch++
	default:
		fail("unknown part %q (want major, minor, or patch)", part)
	}

	next := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if err := os.WriteFile(path, []byte(next+"\n"), 0o644); err != nil {
		fail("writing %s: %v", path, err)
	}
	fmt.Println(next)
}

// parse splits a strict MAJOR.MINOR.PATCH string into its numeric components.
func parse(s string) (int, int, int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver %q (want MAJOR.MINOR.PATCH)", s)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, 0, 0, fmt.Errorf("invalid semver %q (want MAJOR.MINOR.PATCH)", s)
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], nil
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "increaseversion: "+format+"\n", a...)
	os.Exit(1)
}
