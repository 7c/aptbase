package cmd

import (
	"fmt"

	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Check connectivity and auth against each configured server",
	Long: `Verify that every configured aptly API server is reachable and that
authentication (if required) works, printing each server's aptly version.

This is the first thing to run when setting up a new server or config: it
exercises the URL, TLS, and any 401-triggered Basic auth.`,
	Example: `  aptbase --api http://localhost:8080 ping
  aptbase --server staging ping`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}

		type result struct {
			URL     string `json:"url"`
			OK      bool   `json:"ok"`
			Version string `json:"version,omitempty"`
			Error   string `json:"error,omitempty"`
		}
		var results []result

		for _, srv := range set.Servers {
			v, verr := srv.Client.Version()
			r := result{URL: srv.URL, OK: verr == nil}
			if verr != nil {
				r.Error = verr.Error()
			} else {
				r.Version = v.Version
			}
			results = append(results, r)
		}

		if settings.JSON {
			return render.JSON(results)
		}

		failed := 0
		for _, r := range results {
			if r.OK {
				ui.Success("%s → aptly %s", r.URL, r.Version)
			} else {
				ui.Error("%s → %s", r.URL, r.Error)
				failed++
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d of %d server(s) unreachable", failed, len(results))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
