package cmd

import "testing"

func entries(keys ...string) []pkgEntry {
	out := make([]pkgEntry, 0, len(keys))
	for _, k := range keys {
		out = append(out, pkgEntry{Key: k, pkgPreview: parsePkgKey(k)})
	}
	return out
}

func TestLatestEntries(t *testing.T) {
	in := entries(
		"Pamd64 app 0.0.9 h1",
		"Pamd64 app 0.0.10 h2", // newest of amd64 (numeric > 0.0.9)
		"Parm64 app 0.0.9 h3",  // separate arch kept
	)
	out := latestEntries(in)
	if len(out) != 2 {
		t.Fatalf("got %d, want 2 (one per arch)", len(out))
	}
	for _, e := range out {
		if e.Arch == "amd64" && e.Version != "0.0.10" {
			t.Errorf("amd64 latest = %s, want 0.0.10", e.Version)
		}
	}
}

func TestArchMatch(t *testing.T) {
	if !archMatch("amd64", nil) {
		t.Error("empty filter should match all")
	}
	if !archMatch("amd64", []string{"arm64", "amd64"}) {
		t.Error("should match listed arch")
	}
	if archMatch("amd64", []string{"arm64"}) {
		t.Error("should not match unlisted arch")
	}
}

func TestLessPkg(t *testing.T) {
	// name sort: app before zoo
	if !lessPkg("app", "1.0", "zoo", "1.0", "name") {
		t.Error("app should sort before zoo")
	}
	// same name: newer version first (desc)
	if !lessPkg("app", "0.0.10", "app", "0.0.9", "name") {
		t.Error("0.0.10 should sort before 0.0.9 (version desc)")
	}
	// version sort: higher version first regardless of name
	if !lessPkg("zoo", "2.0", "app", "1.0", "version") {
		t.Error("2.0 should sort before 1.0 under version sort")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.0.9", "0.0.10", -1},
		{"0.0.10", "0.0.9", 1},
		{"1.0", "1.0", 0},
		{"1.2.3-1", "1.2.3-2", -1},
		{"1.0~rc1", "1.0", -1}, // tilde sorts before release
		{"2.0", "10.0", -1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
