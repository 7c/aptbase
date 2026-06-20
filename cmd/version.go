package cmd

import (
	"runtime"

	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build info (and remote aptly versions if configured)",
	Long: `Print the local aptbase build: version, git commit, build date, and Go runtime.

When one or more API servers are configured, it also queries each server's
aptly version (GET /api/version). Use 'aptbase --version' for just the version
string.`,
	Example: `  aptbase version
  aptbase --version
  aptbase --api http://localhost:8080 version`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if settings.JSON {
			return versionJSON()
		}

		ui.Info("aptbase %s", version)
		ui.KeyValues([][2]string{
			{"commit", commit},
			{"built", date},
			{"go", runtime.Version()},
		})

		set, err := resolveTargets()
		if err != nil {
			return nil // no servers configured: local build info is enough
		}
		ui.Heading("Remote aptly servers")
		for _, srv := range set.Servers {
			v, verr := srv.Client.Version()
			if verr != nil {
				ui.Warn("%s: %s", srv.URL, verr.Error())
				continue
			}
			ui.Dim("%s → aptly %s", srv.URL, v.Version)
		}
		return nil
	},
}

func versionJSON() error {
	out := map[string]any{
		"aptbase": version,
		"commit":  commit,
		"built":   date,
		"go":      runtime.Version(),
	}
	if set, err := resolveTargets(); err == nil {
		remotes := map[string]string{}
		for _, srv := range set.Servers {
			if v, verr := srv.Client.Version(); verr == nil {
				remotes[srv.URL] = v.Version
			} else {
				remotes[srv.URL] = "error: " + verr.Error()
			}
		}
		if len(remotes) > 0 {
			out["remotes"] = remotes
		}
	}
	return render.JSON(out)
}
