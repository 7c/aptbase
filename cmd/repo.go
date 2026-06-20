package cmd

import (
	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage local repositories on remote aptly servers",
	Long: `List, inspect, create, edit, and delete local repositories, and add or
deploy packages to them.

The flagship workflow is 'aptbase repo deploy', which uploads, adds, publishes,
and verifies a package in one command.`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local repositories",
	Long:  "List all local repositories on each configured server.",
	Example: `  aptbase repo list
  aptbase --api http://localhost:8080 repo list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			repos, err := srv.Client.ListRepos()
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(repos)
			}
			if len(repos) == 0 {
				ui.Dim("no local repositories")
				return nil
			}
			rows := make([][]string, 0, len(repos))
			for _, r := range repos {
				rows = append(rows, []string{r.Name, r.DefaultDistribution, r.DefaultComponent, r.Comment})
			}
			ui.Table([]string{"NAME", "DISTRIBUTION", "COMPONENT", "COMMENT"}, rows)
			return nil
		})
	},
}

var repoShowCmd = &cobra.Command{
	Use:   "show <repo>",
	Short: "Show details of a local repository",
	Example: `  aptbase repo show app-stable`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		name := args[0]
		return forEachServer(set, func(srv target.Server) error {
			repo, err := srv.Client.ShowRepo(name)
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(repo)
			}
			ui.KeyValues([][2]string{
				{"Name", repo.Name},
				{"Distribution", repo.DefaultDistribution},
				{"Component", repo.DefaultComponent},
				{"Comment", repo.Comment},
			})
			return nil
		})
	},
}

var (
	repoCreateComment string
	repoCreateDist    string
	repoCreateComp    string
)

var repoCreateCmd = &cobra.Command{
	Use:   "create <repo>",
	Short: "Create a local repository",
	Example: `  aptbase repo create app-stable --distribution noble --component main
  aptbase repo create app-edge -d jammy --comment "edge builds"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		dist := repoCreateDist
		if dist == "" && len(settings.Distributions) > 0 {
			dist = settings.Distributions[0]
		}
		req := client.CreateRepoRequest{
			Name:                args[0],
			Comment:             repoCreateComment,
			DefaultDistribution: dist,
			DefaultComponent:    repoCreateComp,
		}
		return forEachServer(set, func(srv target.Server) error {
			repo, err := srv.Client.CreateRepo(req)
			if err != nil {
				return err
			}
			ui.Success("created repo %q", repo.Name)
			return nil
		})
	},
}

var (
	repoEditComment string
	repoEditDist    string
	repoEditComp    string
)

var repoEditCmd = &cobra.Command{
	Use:   "edit <repo>",
	Short: "Edit a local repository's defaults",
	Example: `  aptbase repo edit app-stable --distribution noble
  aptbase repo edit app-stable --comment "stable channel"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		req := client.EditRepoRequest{
			Comment:             repoEditComment,
			DefaultDistribution: repoEditDist,
			DefaultComponent:    repoEditComp,
		}
		return forEachServer(set, func(srv target.Server) error {
			repo, err := srv.Client.EditRepo(args[0], req)
			if err != nil {
				return err
			}
			ui.Success("updated repo %q", repo.Name)
			return nil
		})
	},
}

var repoDropForce bool

var repoDropCmd = &cobra.Command{
	Use:     "drop <repo>",
	Aliases: []string{"delete", "rm"},
	Short:   "Delete a local repository",
	Long:    "Delete a local repository. Use --force to drop one still referenced by a publication or snapshot.",
	Example: `  aptbase repo drop app-edge
  aptbase repo drop app-edge --force --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		if !confirm("Delete repository " + args[0] + " on all targets?") {
			ui.Dim("aborted")
			return nil
		}
		return forEachServer(set, func(srv target.Server) error {
			if err := srv.Client.DeleteRepo(args[0], repoDropForce); err != nil {
				return err
			}
			ui.Success("dropped repo %q", args[0])
			return nil
		})
	},
}

var repoPackagesQuery string

var repoPackagesCmd = &cobra.Command{
	Use:   "packages [repo]",
	Short: "List packages in a repository",
	Long: `List package keys in a repository, optionally filtered by an aptly query.

If repo is omitted, the first configured default repo is used.`,
	Example: `  aptbase repo packages app-stable
  aptbase repo packages app-stable -q 'nginx (>= 1.20)'`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		repo, err := singleRepo(args)
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			keys, err := srv.Client.RepoPackages(repo, repoPackagesQuery)
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(keys)
			}
			if len(keys) == 0 {
				ui.Dim("no packages match")
				return nil
			}
			for _, k := range keys {
				ui.Info("%s", k)
			}
			ui.Dim("%d package(s)", len(keys))
			return nil
		})
	},
}

// forEachServer runs fn against each server, printing a heading per server when
// more than one is configured, and returning an aggregate error.
func forEachServer(set *target.Set, fn func(target.Server) error) error {
	multi := len(set.Servers) > 1
	results := set.ForEachServer(func(srv target.Server) error {
		if multi && !settings.JSON {
			ui.Heading(srv.URL)
		}
		if err := fn(srv); err != nil {
			ui.Error("%s: %s", srv.URL, err.Error())
			return err
		}
		return nil
	})
	return target.AggregateError(results)
}

// singleRepo resolves a single repo from args or the first configured default.
func singleRepo(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if len(settings.Repos) > 0 {
		return settings.Repos[0], nil
	}
	return "", errNoRepo
}

func init() {
	repoCreateCmd.Flags().StringVar(&repoCreateComment, "comment", "", "repository comment")
	repoCreateCmd.Flags().StringVar(&repoCreateDist, "distribution", "", "default distribution")
	repoCreateCmd.Flags().StringVar(&repoCreateComp, "component", "", "default component")

	repoEditCmd.Flags().StringVar(&repoEditComment, "comment", "", "repository comment")
	repoEditCmd.Flags().StringVar(&repoEditDist, "distribution", "", "default distribution")
	repoEditCmd.Flags().StringVar(&repoEditComp, "component", "", "default component")

	repoDropCmd.Flags().BoolVar(&repoDropForce, "force", false, "force deletion even if referenced")

	repoPackagesCmd.Flags().StringVarP(&repoPackagesQuery, "query", "q", "", "aptly package query filter")

	repoCmd.AddCommand(repoListCmd, repoShowCmd, repoCreateCmd, repoEditCmd, repoDropCmd, repoPackagesCmd)
	rootCmd.AddCommand(repoCmd)
}
