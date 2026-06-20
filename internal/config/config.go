// Package config resolves aptbase settings from layered sources.
//
// Resolution order, where a later layer overrides an earlier one (last wins):
//
//	built-in defaults
//	  → /etc/aptbase.ini
//	  → ~/aptbase.ini
//	  → APTBASE_* environment variables
//	  → command-line flags (only when explicitly set)
//
// When --config (or APTBASE_CONFIG) names an explicit file, only that file is
// read instead of the two well-known locations.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
)

// Setting keys. These double as INI keys and CLI flag names.
const (
	KeyAPI           = "api"
	KeyServer        = "server"
	KeyUser          = "user"
	KeyPassword      = "password"
	KeyDistributions = "distributions"
	KeyRepos         = "repos"
	KeyPrefix        = "prefix"
	KeyInsecure      = "insecure"
	KeyTimeout       = "timeout"
	KeyJSON          = "json"
	KeyNoColor       = "no-color"
	KeyYes           = "yes"
)

// SystemConfigPath is the default system-wide config location.
const SystemConfigPath = "/etc/aptbase.ini"

// Settings holds the fully resolved configuration for a single invocation.
type Settings struct {
	APIs          []string
	Server        string
	User          string
	Password      string
	HasPassword   bool // whether a password was supplied (vs. prompt-on-401)
	Distributions []string
	Repos         []string
	Prefix        string
	Insecure      bool
	Timeout       time.Duration
	JSON          bool
	NoColor       bool
	Yes           bool

	sources map[string]string // key -> human-readable source label
}

// Source returns where the value for key was resolved from (e.g. "flag",
// "/etc/aptbase.ini", "env:APTBASE_API", "default").
func (s *Settings) Source(key string) string {
	if src, ok := s.sources[key]; ok {
		return src
	}
	return "default"
}

func defaults() *Settings {
	return &Settings{
		Prefix:  ".",
		Timeout: 60 * time.Second,
		sources: map[string]string{},
	}
}

// Defaults returns the built-in default settings (the base layer before any
// config file, environment, or flag is applied).
func Defaults() *Settings { return defaults() }

var listSplitter = regexp.MustCompile(`[\s,]+`)

// parseList splits a whitespace- or comma-separated value into trimmed,
// non-empty tokens.
func parseList(v string) []string {
	out := []string{}
	for _, tok := range listSplitter.Split(strings.TrimSpace(v), -1) {
		if tok != "" {
			out = append(out, tok)
		}
	}
	return out
}

// userConfigPath returns ~/aptbase.ini (empty if home unknown).
func userConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "aptbase.ini")
}

// Resolve builds Settings from all layers. flags is the root command's
// persistent flag set; only flags reported as Changed are applied.
func Resolve(flags *pflag.FlagSet) (*Settings, error) {
	s := defaults()

	// Server selection (picks the [server:NAME] INI section) comes only from
	// flag or env, since it determines which config section to read.
	server := firstNonEmpty(flagValue(flags, KeyServer), os.Getenv("APTBASE_SERVER"))

	explicit := firstNonEmpty(flagValue(flags, "config"), os.Getenv("APTBASE_CONFIG"))
	paths := configPaths(explicit)
	for _, path := range paths {
		if err := s.loadFile(path, server); err != nil {
			return nil, err
		}
	}

	s.applyEnv()

	if err := s.applyFlags(flags); err != nil {
		return nil, err
	}

	if server != "" {
		s.Server = server
	}
	return s, nil
}

// configPaths returns the ordered list of config files to read.
func configPaths(explicit string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	paths := []string{SystemConfigPath}
	if up := userConfigPath(); up != "" {
		paths = append(paths, up)
	}
	return paths
}

// loadFile merges a single INI file: the [default] section first, then the
// selected [server:NAME] section on top. Missing files are ignored.
func (s *Settings) loadFile(path, server string) error {
	if _, err := os.Stat(path); err != nil {
		return nil // absent files are not an error
	}
	f, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("reading config %s: %w", path, err)
	}
	s.applySection(f.Section("default"), path)
	if server != "" {
		name := "server:" + server
		if f.HasSection(name) {
			s.applySection(f.Section(name), path)
		}
	}
	return nil
}

func (s *Settings) applySection(sec *ini.Section, src string) {
	if sec.HasKey(KeyAPI) {
		s.APIs = parseList(sec.Key(KeyAPI).String())
		s.sources[KeyAPI] = src
	}
	if sec.HasKey(KeyUser) {
		s.User = sec.Key(KeyUser).String()
		s.sources[KeyUser] = src
	}
	if sec.HasKey(KeyPassword) {
		s.Password = sec.Key(KeyPassword).String()
		s.HasPassword = true
		s.sources[KeyPassword] = src
	}
	if sec.HasKey(KeyDistributions) {
		s.Distributions = parseList(sec.Key(KeyDistributions).String())
		s.sources[KeyDistributions] = src
	}
	if sec.HasKey(KeyRepos) {
		s.Repos = parseList(sec.Key(KeyRepos).String())
		s.sources[KeyRepos] = src
	}
	if sec.HasKey(KeyPrefix) {
		s.Prefix = sec.Key(KeyPrefix).String()
		s.sources[KeyPrefix] = src
	}
	if sec.HasKey(KeyInsecure) {
		s.Insecure, _ = sec.Key(KeyInsecure).Bool()
		s.sources[KeyInsecure] = src
	}
	if sec.HasKey(KeyTimeout) {
		if d, err := time.ParseDuration(sec.Key(KeyTimeout).String()); err == nil {
			s.Timeout = d
			s.sources[KeyTimeout] = src
		}
	}
	if sec.HasKey(KeyJSON) {
		s.JSON, _ = sec.Key(KeyJSON).Bool()
		s.sources[KeyJSON] = src
	}
	if sec.HasKey(KeyNoColor) {
		s.NoColor, _ = sec.Key(KeyNoColor).Bool()
		s.sources[KeyNoColor] = src
	}
	if sec.HasKey(KeyYes) {
		s.Yes, _ = sec.Key(KeyYes).Bool()
		s.sources[KeyYes] = src
	}
}

// envBindings maps env var name -> setting key for source labelling.
var envBindings = map[string]string{
	"APTBASE_API":           KeyAPI,
	"APTBASE_USER":          KeyUser,
	"APTBASE_PASSWORD":      KeyPassword,
	"APTBASE_DISTRIBUTIONS": KeyDistributions,
	"APTBASE_REPOS":         KeyRepos,
	"APTBASE_PREFIX":        KeyPrefix,
	"APTBASE_INSECURE":      KeyInsecure,
	"APTBASE_TIMEOUT":       KeyTimeout,
	"APTBASE_JSON":          KeyJSON,
	"APTBASE_NO_COLOR":      KeyNoColor,
	"APTBASE_YES":           KeyYes,
}

func (s *Settings) applyEnv() {
	for env, key := range envBindings {
		v, ok := os.LookupEnv(env)
		if !ok {
			continue
		}
		src := "env:" + env
		switch key {
		case KeyAPI:
			s.APIs = parseList(v)
		case KeyUser:
			s.User = v
		case KeyPassword:
			s.Password = v
			s.HasPassword = true
		case KeyDistributions:
			s.Distributions = parseList(v)
		case KeyRepos:
			s.Repos = parseList(v)
		case KeyPrefix:
			s.Prefix = v
		case KeyInsecure:
			s.Insecure = parseBool(v)
		case KeyTimeout:
			if d, err := time.ParseDuration(v); err == nil {
				s.Timeout = d
			} else {
				continue
			}
		case KeyJSON:
			s.JSON = parseBool(v)
		case KeyNoColor:
			s.NoColor = parseBool(v)
		case KeyYes:
			s.Yes = parseBool(v)
		}
		s.sources[key] = src
	}
}

// applyFlags overlays CLI flags, but only those the user explicitly set.
func (s *Settings) applyFlags(flags *pflag.FlagSet) error {
	const src = "flag"
	if changed(flags, KeyAPI) {
		v, _ := flags.GetStringArray(KeyAPI)
		// Each entry may itself be a list (e.g. --api "a b"); flatten.
		var all []string
		for _, item := range v {
			all = append(all, parseList(item)...)
		}
		s.APIs = all
		s.sources[KeyAPI] = src
	}
	if changed(flags, KeyUser) {
		s.User, _ = flags.GetString(KeyUser)
		s.sources[KeyUser] = src
	}
	if changed(flags, KeyPassword) {
		s.Password, _ = flags.GetString(KeyPassword)
		s.HasPassword = true
		s.sources[KeyPassword] = src
	}
	if changed(flags, KeyDistributions) {
		v, _ := flags.GetStringArray(KeyDistributions)
		var all []string
		for _, item := range v {
			all = append(all, parseList(item)...)
		}
		s.Distributions = all
		s.sources[KeyDistributions] = src
	}
	if changed(flags, KeyRepos) {
		v, _ := flags.GetStringArray(KeyRepos)
		var all []string
		for _, item := range v {
			all = append(all, parseList(item)...)
		}
		s.Repos = all
		s.sources[KeyRepos] = src
	}
	if changed(flags, KeyPrefix) {
		s.Prefix, _ = flags.GetString(KeyPrefix)
		s.sources[KeyPrefix] = src
	}
	if changed(flags, KeyInsecure) {
		s.Insecure, _ = flags.GetBool(KeyInsecure)
		s.sources[KeyInsecure] = src
	}
	if changed(flags, KeyTimeout) {
		s.Timeout, _ = flags.GetDuration(KeyTimeout)
		s.sources[KeyTimeout] = src
	}
	if changed(flags, KeyJSON) {
		s.JSON, _ = flags.GetBool(KeyJSON)
		s.sources[KeyJSON] = src
	}
	if changed(flags, KeyNoColor) {
		s.NoColor, _ = flags.GetBool(KeyNoColor)
		s.sources[KeyNoColor] = src
	}
	if changed(flags, KeyYes) {
		s.Yes, _ = flags.GetBool(KeyYes)
		s.sources[KeyYes] = src
	}
	return nil
}

func parseBool(v string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(v))
	return b
}

func changed(flags *pflag.FlagSet, name string) bool {
	if flags == nil {
		return false
	}
	f := flags.Lookup(name)
	return f != nil && f.Changed
}

// flagValue returns a string flag's current value if the flag exists.
func flagValue(flags *pflag.FlagSet, name string) string {
	if flags == nil {
		return ""
	}
	if f := flags.Lookup(name); f != nil && f.Changed {
		v, _ := flags.GetString(name)
		return v
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
