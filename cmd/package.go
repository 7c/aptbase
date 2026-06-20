package cmd

import (
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"pkg"},
	Short:   "Search for and inspect packages",
	Long:    "Search packages across repositories (aptly query syntax) and show package details.",
}

var packageSearchRepos []string

var packageSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search packages in repositories by aptly query",
	Long: `Search packages using aptly's query syntax.

aptly has no global package index, so the search runs against each target repo:
the repos given with --repo, else the configured default 'repos'.`,
	Example: `  aptbase package search 'nginx'
  aptbase package search 'nginx (>= 1.20)' --repo app-stable
  aptbase --api http://localhost:8080 pkg search 'Name (% myapp*)'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		repos := packageSearchRepos
		if len(repos) == 0 {
			if len(settings.Repos) == 0 {
				return errNoRepo
			}
			repos = settings.Repos
		}
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			type hit struct {
				Repo string   `json:"repo"`
				Keys []string `json:"keys"`
			}
			var hits []hit
			for _, repo := range repos {
				keys, err := srv.Client.RepoPackages(repo, query)
				if err != nil {
					return err
				}
				hits = append(hits, hit{Repo: repo, Keys: keys})
			}
			if settings.JSON {
				return render.JSON(hits)
			}
			total := 0
			for _, h := range hits {
				if len(h.Keys) == 0 {
					continue
				}
				ui.Heading(h.Repo)
				for _, k := range h.Keys {
					ui.Info("%s", k)
					total++
				}
			}
			if total == 0 {
				ui.Dim("no matches")
			} else {
				ui.Dim("%d match(es)", total)
			}
			return nil
		})
	},
}

var packageShowCmd = &cobra.Command{
	Use:   "show <key>",
	Short: "Show details for a package key",
	Long:  "Show the full detail map for a package key (as returned by aptly).",
	Example: `  aptbase package show 'Pamd64 nginx 1.20.1-1 abc123'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			detail, err := srv.Client.ShowPackage(args[0])
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(detail)
			}
			pairs := make([][2]string, 0, len(detail))
			for k, v := range detail {
				pairs = append(pairs, [2]string{k, toStr(v)})
			}
			ui.KeyValues(sortPairs(pairs))
			return nil
		})
	},
}

func init() {
	packageSearchCmd.Flags().StringArrayVar(&packageSearchRepos, "repo", nil, "repo to search (repeatable; default: configured repos)")
	packageCmd.AddCommand(packageSearchCmd, packageShowCmd)
	rootCmd.AddCommand(packageCmd)
}
