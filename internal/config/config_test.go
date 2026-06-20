package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

// newFlags builds a flag set mirroring the root command's persistent flags.
func newFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.StringArray(KeyAPI, nil, "")
	fs.String(KeyServer, "", "")
	fs.String(KeyUser, "", "")
	fs.String(KeyPassword, "", "")
	fs.String("config", "", "")
	fs.StringArray(KeyDistributions, nil, "")
	fs.String(KeyPrefix, ".", "")
	fs.Bool(KeyInsecure, false, "")
	fs.Duration(KeyTimeout, 60*time.Second, "")
	fs.Bool(KeyJSON, false, "")
	fs.Bool(KeyNoColor, false, "")
	fs.Bool(KeyYes, false, "")
	return fs
}

func writeINI(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseList(t *testing.T) {
	cases := map[string][]string{
		"noble jammy focal":  {"noble", "jammy", "focal"},
		"noble, jammy,focal": {"noble", "jammy", "focal"},
		"  a   b  ":          {"a", "b"},
		"":                   {},
	}
	for in, want := range cases {
		if got := parseList(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseList(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDefaults(t *testing.T) {
	fs := newFlags()
	t.Setenv("APTBASE_API", "") // ensure no leakage
	os.Unsetenv("APTBASE_API")
	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if s.Prefix != "." {
		t.Errorf("default prefix = %q, want .", s.Prefix)
	}
	if s.Timeout != 60*time.Second {
		t.Errorf("default timeout = %v, want 60s", s.Timeout)
	}
}

func TestFileLayer(t *testing.T) {
	path := writeINI(t, `
[default]
api = http://a:8080 http://b:8080
user = deploy
distributions = noble jammy
repos = app-stable app-edge
timeout = 30s
insecure = true
`)
	fs := newFlags()
	_ = fs.Set("config", path)

	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.APIs, []string{"http://a:8080", "http://b:8080"}) {
		t.Errorf("APIs = %v", s.APIs)
	}
	if !reflect.DeepEqual(s.Distributions, []string{"noble", "jammy"}) {
		t.Errorf("Distributions = %v", s.Distributions)
	}
	if s.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", s.Timeout)
	}
	if !s.Insecure {
		t.Error("Insecure should be true")
	}
	if s.Source(KeyAPI) != path {
		t.Errorf("source(api) = %q, want %q", s.Source(KeyAPI), path)
	}
}

func TestFlagOverridesFile(t *testing.T) {
	path := writeINI(t, "[default]\napi = http://file:8080\ndistributions = noble jammy\n")
	fs := newFlags()
	_ = fs.Set("config", path)
	_ = fs.Set(KeyDistributions, "focal")

	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Distributions, []string{"focal"}) {
		t.Errorf("Distributions = %v, want [focal] (flag wins)", s.Distributions)
	}
	if s.Source(KeyDistributions) != "flag" {
		t.Errorf("source = %q, want flag", s.Source(KeyDistributions))
	}
	// api untouched by flag -> stays from file
	if s.Source(KeyAPI) != path {
		t.Errorf("api source = %q, want file", s.Source(KeyAPI))
	}
}

func TestEnvOverridesFileButNotFlag(t *testing.T) {
	path := writeINI(t, "[default]\nprefix = fromfile\n")
	t.Setenv("APTBASE_PREFIX", "fromenv")

	fs := newFlags()
	_ = fs.Set("config", path)
	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if s.Prefix != "fromenv" {
		t.Errorf("Prefix = %q, want fromenv", s.Prefix)
	}

	fs2 := newFlags()
	_ = fs2.Set("config", path)
	_ = fs2.Set(KeyPrefix, "fromflag")
	s2, err := Resolve(fs2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Prefix != "fromflag" {
		t.Errorf("Prefix = %q, want fromflag (flag beats env)", s2.Prefix)
	}
}

func TestServerSection(t *testing.T) {
	path := writeINI(t, `
[default]
api = http://default:8080
distributions = noble jammy focal

[server:staging]
api = http://staging:8080
distributions = noble
`)
	fs := newFlags()
	_ = fs.Set("config", path)
	_ = fs.Set(KeyServer, "staging")

	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.APIs, []string{"http://staging:8080"}) {
		t.Errorf("APIs = %v, want staging", s.APIs)
	}
	if !reflect.DeepEqual(s.Distributions, []string{"noble"}) {
		t.Errorf("Distributions = %v, want [noble]", s.Distributions)
	}
	if s.Server != "staging" {
		t.Errorf("Server = %q", s.Server)
	}
}

func TestPasswordPromptDefault(t *testing.T) {
	fs := newFlags()
	s, err := Resolve(fs)
	if err != nil {
		t.Fatal(err)
	}
	if s.HasPassword {
		t.Error("HasPassword should be false when no password supplied")
	}
}
