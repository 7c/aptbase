package cmd

import (
	"fmt"
	"strings"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Inspect and update published repositories",
	Long:  "List publications and re-publish (update) distributions after repository changes.",
}

var publishListCmd = &cobra.Command{
	Use:   "list",
	Short: "List published repositories",
	Example: `  aptbase publish list
  aptbase publish list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			pubs, err := srv.Client.ListPublished()
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(pubs)
			}
			if len(pubs) == 0 {
				ui.Dim("nothing published")
				return nil
			}
			rows := make([][]string, 0, len(pubs))
			for _, p := range pubs {
				rows = append(rows, []string{
					prefixOrRoot(p.Prefix),
					p.Distribution,
					p.SourceKind,
					sourceNames(p.Sources),
					strings.Join(p.Architectures, ","),
				})
			}
			ui.Table([]string{"PREFIX", "DISTRIBUTION", "KIND", "SOURCES", "ARCH"}, rows)
			return nil
		})
	},
}

var (
	publishUpdateGpgKey   string
	publishUpdateKeyring  string
	publishUpdatePass     string
	publishUpdateSkip     bool
	publishUpdateBatch    bool
	publishUpdateForce    bool
)

var publishUpdateCmd = &cobra.Command{
	Use:   "update [distribution...]",
	Short: "Re-publish distributions so repository changes go live",
	Long: `Re-publish one or more distributions at the configured prefix.

If no distribution is given, the configured default 'distributions' are used.
This runs as an async aptly task with streamed progress.`,
	Example: `  aptbase publish update noble
  aptbase publish update            # uses configured distributions
  aptbase publish update noble jammy --gpg-key DEADBEEF --batch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dists := args
		if len(dists) == 0 {
			var err error
			if dists, err = distributionsFor(); err != nil {
				return err
			}
		}
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		req := client.UpdatePublishRequest{
			Signing: client.Signing{
				Skip:       publishUpdateSkip,
				GpgKey:     publishUpdateGpgKey,
				Keyring:    publishUpdateKeyring,
				Passphrase: publishUpdatePass,
				Batch:      publishUpdateBatch,
			},
			ForceOverwrite: publishUpdateForce,
		}
		return forEachServer(set, func(srv target.Server) error {
			for _, dist := range dists {
				task, err := srv.Client.UpdatePublished(settings.Prefix, dist, req)
				if err != nil {
					return err
				}
				if err := runTask(srv, task, fmt.Sprintf("publish %s/%s", settings.Prefix, dist)); err != nil {
					return err
				}
				ui.Success("published %s/%s", settings.Prefix, dist)
			}
			return nil
		})
	},
}

func prefixOrRoot(p string) string {
	if p == "" || p == "." {
		return "(root)"
	}
	return p
}

func sourceNames(sources []client.PublishedSource) string {
	names := make([]string, 0, len(sources))
	for _, s := range sources {
		names = append(names, s.Component+":"+s.Name)
	}
	return strings.Join(names, ",")
}

func init() {
	f := publishUpdateCmd.Flags()
	f.StringVar(&publishUpdateGpgKey, "gpg-key", "", "GPG key ID to sign with")
	f.StringVar(&publishUpdateKeyring, "keyring", "", "GPG keyring file to use")
	f.StringVar(&publishUpdatePass, "passphrase", "", "GPG key passphrase")
	f.BoolVar(&publishUpdateSkip, "skip-signing", false, "publish without signing")
	f.BoolVar(&publishUpdateBatch, "batch", false, "run GPG in batch mode")
	f.BoolVar(&publishUpdateForce, "force-overwrite", false, "overwrite existing published files")

	publishCmd.AddCommand(publishListCmd, publishUpdateCmd)
	rootCmd.AddCommand(publishCmd)
}
