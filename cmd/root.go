// Package cmd holds the cobra command tree for aptbase.
package cmd

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/7c/aptbase/internal/config"
	"github.com/7c/aptbase/internal/debug"
	"github.com/7c/aptbase/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// redactArgs masks the value of --password (in both "--password x" and
// "--password=x" forms) so the startup args line is safe to paste in bug reports.
func redactArgs(args []string) []string {
	out := make([]string, len(args))
	maskNext := false
	for i, a := range args {
		switch {
		case maskNext:
			out[i] = "***"
			maskNext = false
		case a == "--password":
			out[i] = a
			maskNext = true
		case strings.HasPrefix(a, "--password="):
			out[i] = "--password=***"
		default:
			out[i] = a
		}
	}
	return out
}

// envTrue reports whether an environment variable is set to a truthy value.
func envTrue(name string) bool {
	v, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	b, _ := strconv.ParseBool(strings.TrimSpace(v))
	return b
}

// Build metadata, injected at build time via -ldflags (see Makefile).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// settings holds the resolved configuration for the current invocation. It is
// populated in PersistentPreRunE before any subcommand runs.
var settings *config.Settings

var rootCmd = &cobra.Command{
	Use:   "aptbase",
	Short: "Human-friendly CLI for remote aptly servers",
	Long: `aptbase is a friendly adapter over the aptly REST API.

It drives one or more remote aptly servers (--api) to manage local repositories,
upload and publish packages, and inspect what is live — with colored output,
tables, live task progress, and a config file so you are not retyping URLs and
credentials.

Configuration is layered (last wins):
  defaults  →  /etc/aptbase.ini  →  ~/aptbase.ini
            →  APTBASE_* env vars  →  command-line flags

Run 'aptbase config new' to scaffold a config file and 'aptbase config list' to
see every resolved setting and where it came from.`,
	Example: `  # Check connectivity and remote version
  aptbase --api http://localhost:8080 ping

  # Release a package end to end (upload + add + publish + verify)
  aptbase deploy app-stable ./app_1.2.3_amd64.deb -d noble -d jammy

  # Using config.ini defaults for api/repos/distributions
  aptbase deploy ./app_1.2.3_amd64.deb`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Enable debug before resolving config so the resolution itself is traced.
		dbg, _ := cmd.Flags().GetBool(config.KeyDebug)
		debug.Enabled = dbg || envTrue("APTBASE_DEBUG")
		debug.Logf("aptbase %s  args=%q", version, redactArgs(os.Args[1:]))

		s, err := config.Resolve(cmd.Flags())
		if err != nil {
			return err
		}
		settings = s
		if s.NoColor {
			color.NoColor = true
		}
		// config.ini / env may also enable debug; never turn it back off.
		debug.Enabled = debug.Enabled || s.Debug
		return nil
	},
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("aptbase {{.Version}}\n")

	pf := rootCmd.PersistentFlags()
	pf.StringArray(config.KeyAPI, nil, "aptly API base URL (repeatable; fans out to all)")
	pf.String(config.KeyServer, "", "named [server:NAME] config section to use")
	pf.String(config.KeyUser, "", "HTTP Basic auth username")
	pf.String(config.KeyPassword, "", "HTTP Basic auth password (prompted on 401 if omitted)")
	pf.String("config", "", "explicit config file path")
	pf.StringArrayP(config.KeyDistributions, "d", nil, "target distribution (repeatable)")
	pf.String(config.KeyPrefix, ".", "publish prefix")
	pf.Bool(config.KeyJSON, false, "output JSON instead of human-readable tables")
	pf.Bool(config.KeyNoColor, false, "disable colored output")
	pf.Bool(config.KeyInsecure, false, "skip TLS certificate verification")
	pf.Duration(config.KeyTimeout, 60*time.Second, "per-request timeout")
	pf.Bool(config.KeyYes, false, "assume yes; skip destructive-action confirmations")
	pf.Bool(config.KeyDebug, false, "print debug-level diagnostics to stderr (for bug reports)")

	rootCmd.AddCommand(versionCmd)
}
