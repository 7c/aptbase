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
  3. refreshes every publication that sources the repo so the package goes
     live — the publication prefix and distribution are discovered from the
     server, and
  4. verifies the package is present in the repo.

Repos default to the configured 'repos' (override positionally or with --repo,
which is repeatable for multi-repo releases). By default every distribution the
repo is published to is refreshed; narrow that with -d/--distribution (or the
configured 'distributions'). It fans out across every configured server, so one
command can release everywhere.`,
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
		// Distributions are an optional filter; the publication prefix is
		// auto-discovered from the server (see deployToServer).
		dists := settings.Distributions
		set, err := resolveTargets()
		if err != nil {
			return err
		}

		ui.Info("deploying %d package(s) → repos %v across %d server(s)%s",
			len(files), repos, len(set.Servers), distNote(dists))

		return forEachServer(set, func(srv target.Server) error {
			return deployToServer(srv, repos, dists, files)
		})
	},
}

// distNote renders the optional distribution filter for the deploy banner.
func distNote(dists []string) string {
	if len(dists) == 0 {
		return ", all published distributions"
	}
	return fmt.Sprintf(", distributions %v", dists)
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

// publishTargetsForRepo returns the publications on srv that source the given
// local repo, optionally filtered to the requested distributions. The prefix is
// taken from the publication itself, so deploy refreshes exactly what is live —
// it does not guess the prefix from config.
func publishTargetsForRepo(srv target.Server, repo string, dists []string) ([]client.PublishedRepo, error) {
	pubs, err := srv.Client.ListPublished()
	if err != nil {
		return nil, err
	}
	return filterPublications(pubs, repo, dists), nil
}

// filterPublications keeps local publications sourcing repo, optionally limited
// to the given distributions.
func filterPublications(pubs []client.PublishedRepo, repo string, dists []string) []client.PublishedRepo {
	distSet := map[string]bool{}
	for _, d := range dists {
		distSet[d] = true
	}
	var out []client.PublishedRepo
	for _, p := range pubs {
		if p.SourceKind != "local" || !publicationSourcesRepo(p, repo) {
			continue
		}
		if len(distSet) > 0 && !distSet[p.Distribution] {
			continue
		}
		out = append(out, p)
	}
	return out
}

// publicationSourcesRepo reports whether a publication's sources include repo.
func publicationSourcesRepo(p client.PublishedRepo, repo string) bool {
	for _, s := range p.Sources {
		if s.Name == repo {
			return true
		}
	}
	return false
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

	for _, repo := range repos {
		targets, err := publishTargetsForRepo(srv, repo, dists)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return fmt.Errorf("repo %q is not published%s on %s — create the publication first (e.g. aptly publish repo) or adjust --distribution",
				repo, distNote(dists), srv.URL)
		}
		for _, p := range targets {
			task, err := srv.Client.UpdatePublished(p.Prefix, p.Distribution, req)
			if err != nil {
				return err
			}
			label := fmt.Sprintf("publish %s/%s", prefixOrRoot(p.Prefix), p.Distribution)
			if err := runTask(srv, task, label); err != nil {
				return err
			}
			ui.Success("published %s/%s", prefixOrRoot(p.Prefix), p.Distribution)
		}
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
