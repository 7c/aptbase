package cmd

import (
	"fmt"
	"strings"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var (
	deployRepos      []string
	deployGpgKey     string
	deployKeyring    string
	deployPassphrase string
	deploySkipSign   bool
	deployBatch      bool
	deployForce      bool
	deployNoVerify   bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [repo] <file.deb...>",
	Short: "Shortcut: upload, add, publish, and verify a package in one command",
	Long: `Deploy is the flagship release shortcut, collapsing several steps into one.
For each target server it:

  1. uploads the .deb file(s),
  2. adds them to the target repo(s),
  3. re-publishes each target distribution so the package goes live, and
  4. verifies the package is present in the repo.

Repos default to the configured 'repos' (override positionally or with --repo,
which is repeatable for multi-repo releases); distributions default to the
configured 'distributions' (override with -d). It fans out across every
configured server, so one command can release everywhere.`,
	Example: `  # Single repo, two distributions
  aptbase deploy app-stable ./app_1.2.3_amd64.deb -d noble -d jammy

  # Override the repo(s) with --repo (repeatable)
  aptbase deploy ./app.deb --repo app-stable --repo app-edge -d noble

  # Use config defaults for repo/distributions/servers
  aptbase deploy ./app_1.2.3_amd64.deb

  # Signed publish
  aptbase deploy app-stable ./app.deb -d noble --gpg-key DEADBEEF --batch

  # Unsigned (lab) publish, no confirmation prompts
  aptbase deploy app-stable ./app.deb -d noble --skip-signing --yes`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoArg, files, err := splitRepoAndFiles(args)
		if err != nil {
			return err
		}
		repos, err := reposForDeploy(repoArg)
		if err != nil {
			return err
		}
		dists, err := distributionsFor()
		if err != nil {
			return err
		}
		set, err := resolveTargets()
		if err != nil {
			return err
		}

		ui.Info("deploying %d package(s) → repos %v, distributions %v, %d server(s)",
			len(files), repos, dists, len(set.Servers))

		return forEachServer(set, func(srv target.Server) error {
			return deployToServer(srv, repos, dists, files)
		})
	},
}

// reposForDeploy resolves the target repos: --repo wins (repeatable), else the
// positional repo argument, else the configured default repos. The --repo flag
// and a positional repo cannot both be given.
func reposForDeploy(repoArg string) ([]string, error) {
	if len(deployRepos) > 0 {
		if repoArg != "" {
			return nil, fmt.Errorf("specify the repo positionally or with --repo, not both")
		}
		return deployRepos, nil
	}
	return reposFor(repoArg)
}

// deployToServer runs the full add → publish → verify pipeline on one server.
func deployToServer(srv target.Server, repos, dists, files []string) error {
	if err := addToServer(srv, repos, files); err != nil {
		return err
	}

	signing := client.Signing{
		Skip:       deploySkipSign,
		GpgKey:     deployGpgKey,
		Keyring:    deployKeyring,
		Passphrase: deployPassphrase,
		Batch:      deployBatch,
	}
	req := client.UpdatePublishRequest{Signing: signing, ForceOverwrite: deployForce}

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

	if deployNoVerify {
		return nil
	}
	return verifyDeploy(srv, repos, files)
}

// verifyDeploy confirms each uploaded package is present in each target repo.
func verifyDeploy(srv target.Server, repos, files []string) error {
	var missing []string
	for _, repo := range repos {
		keys, err := srv.Client.RepoPackages(repo, "")
		if err != nil {
			return err
		}
		for _, f := range files {
			info := parseDeb(f)
			if !packagePresent(keys, info) {
				missing = append(missing, fmt.Sprintf("%s in %s", info.File, repo))
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("verification failed; not found: %s", strings.Join(missing, ", "))
	}
	ui.Success("verified %d package(s) present", len(files))
	return nil
}

// packagePresent reports whether a package key matching the .deb is present.
func packagePresent(keys []string, info debInfo) bool {
	for _, k := range keys {
		if !strings.Contains(k, info.Name) {
			continue
		}
		if info.Version == "" || strings.Contains(k, info.Version) {
			return true
		}
	}
	return false
}

func init() {
	f := deployCmd.Flags()
	f.StringArrayVar(&deployRepos, "repo", nil, "target repo (repeatable; overrides positional/config)")
	f.StringVar(&deployGpgKey, "gpg-key", "", "GPG key ID to sign the published repo")
	f.StringVar(&deployKeyring, "keyring", "", "GPG keyring file to use")
	f.StringVar(&deployPassphrase, "passphrase", "", "GPG key passphrase")
	f.BoolVar(&deploySkipSign, "skip-signing", false, "publish without signing")
	f.BoolVar(&deployBatch, "batch", false, "run GPG in batch (non-interactive) mode")
	f.BoolVar(&deployForce, "force-overwrite", false, "overwrite existing published files")
	f.BoolVar(&deployNoVerify, "no-verify", false, "skip post-publish verification")

	rootCmd.AddCommand(deployCmd)
}
