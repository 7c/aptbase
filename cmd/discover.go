package cmd

import (
	"sort"
	"strconv"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var discoverNoCounts bool

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Probe an aptly server and print a detailed overview",
	Long: `Discover inspects each configured aptly server and prints a rich overview:
its version and auth status, a summary count of repositories, mirrors,
snapshots, publications and tasks, and detailed tables for each.

It is a fast way to understand an unfamiliar server, audit what is live, or
sanity-check a deploy target. By default it also counts packages per local
repository (one query each); use --no-counts to skip that on large servers.`,
	Example: `  aptbase --api http://aptbase:8080 discover
  aptbase --api http://aptbase:8080 discover --no-counts
  aptbase --api http://aptbase:8080 discover --json | jq '.repos'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}

		var overviews []*overview
		results := set.ForEachServer(func(srv target.Server) error {
			ov, derr := discoverServer(srv)
			if derr != nil {
				if !settings.JSON {
					ui.Heading(srv.URL)
					ui.Error("%s", derr.Error())
				}
				return derr
			}
			overviews = append(overviews, ov)
			if !settings.JSON {
				ov.print()
			}
			return nil
		})

		if settings.JSON {
			if err := render.JSON(overviews); err != nil {
				return err
			}
		}
		return target.AggregateError(results)
	},
}

// overview is the gathered snapshot of one aptly server.
type overview struct {
	URL           string                 `json:"url"`
	Version       string                 `json:"version"`
	Authenticated bool                   `json:"authenticated"`
	Repos         []repoInfo             `json:"repos"`
	Mirrors       []client.Mirror        `json:"mirrors"`
	Snapshots     []client.Snapshot      `json:"snapshots"`
	Publications  []client.PublishedRepo `json:"publications"`
	Tasks         []client.Task          `json:"tasks"`
}

// repoInfo is a repository plus its package count (-1 when unknown).
type repoInfo struct {
	client.Repo
	Packages int `json:"packages"`
}

func discoverServer(srv target.Server) (*overview, error) {
	v, err := srv.Client.Version()
	if err != nil {
		return nil, err
	}
	ov := &overview{URL: srv.URL, Version: v.Version}

	repos, err := srv.Client.ListRepos()
	if err != nil {
		return nil, err
	}
	for _, r := range repos {
		count := -1
		if !discoverNoCounts {
			if keys, perr := srv.Client.RepoPackages(r.Name, ""); perr == nil {
				count = len(keys)
			}
		}
		ov.Repos = append(ov.Repos, repoInfo{Repo: r, Packages: count})
	}

	// Mirrors, snapshots, publications and tasks are best-effort: a server may
	// have none, and older deployments may not expose every endpoint.
	if mirrors, merr := srv.Client.ListMirrors(); merr == nil {
		ov.Mirrors = mirrors
	}
	if snaps, serr := srv.Client.ListSnapshots(); serr == nil {
		ov.Snapshots = snaps
	}
	if pubs, perr := srv.Client.ListPublished(); perr == nil {
		ov.Publications = pubs
	}
	if tasks, terr := srv.Client.ListTasks(); terr == nil {
		ov.Tasks = tasks
	}

	// Authenticated reflects whether credentials were sent on the calls above.
	ov.Authenticated = srv.Client.Authenticated()
	return ov, nil
}

func (ov *overview) print() {
	ui.Heading(ov.URL)
	auth := "none"
	if ov.Authenticated {
		auth = "basic"
	}
	ui.Dim("aptly %s  •  auth: %s", ov.Version, auth)

	ui.Label("Summary")
	ui.KeyValues([][2]string{
		{"repositories", strconv.Itoa(len(ov.Repos))},
		{"mirrors", strconv.Itoa(len(ov.Mirrors))},
		{"snapshots", strconv.Itoa(len(ov.Snapshots))},
		{"publications", strconv.Itoa(len(ov.Publications))},
		{"tasks", strconv.Itoa(len(ov.Tasks))},
	})

	if len(ov.Repos) > 0 {
		ui.Label("Local repositories")
		rows := make([][]string, 0, len(ov.Repos))
		for _, r := range ov.Repos {
			rows = append(rows, []string{r.Name, r.DefaultDistribution, r.DefaultComponent, countStr(r.Packages), r.Comment})
		}
		ui.Table([]string{"NAME", "DIST", "COMPONENT", "PACKAGES", "COMMENT"}, rows)
	}

	if len(ov.Mirrors) > 0 {
		ui.Label("Mirrors")
		rows := make([][]string, 0, len(ov.Mirrors))
		for _, m := range ov.Mirrors {
			rows = append(rows, []string{m.Name, m.ArchiveRoot, m.Distribution, joinOrDash(m.Components)})
		}
		ui.Table([]string{"NAME", "ARCHIVE", "DIST", "COMPONENTS"}, rows)
	}

	if len(ov.Snapshots) > 0 {
		ui.Label("Snapshots")
		snaps := append([]client.Snapshot(nil), ov.Snapshots...)
		sort.Slice(snaps, func(i, j int) bool { return snaps[i].CreatedAt > snaps[j].CreatedAt })
		rows := make([][]string, 0, len(snaps))
		for _, s := range snaps {
			rows = append(rows, []string{s.Name, s.CreatedAt, s.Description})
		}
		ui.Table([]string{"NAME", "CREATED", "DESCRIPTION"}, rows)
	}

	if len(ov.Publications) > 0 {
		ui.Label("Publications")
		rows := make([][]string, 0, len(ov.Publications))
		for _, p := range ov.Publications {
			rows = append(rows, []string{
				prefixOrRoot(p.Prefix),
				p.Distribution,
				p.SourceKind,
				sourceNames(p.Sources),
				joinOrDash(p.Architectures),
			})
		}
		ui.Table([]string{"PREFIX", "DIST", "KIND", "SOURCES", "ARCH"}, rows)
	}

	if active := activeTasks(ov.Tasks); len(active) > 0 {
		ui.Label("Active tasks")
		rows := make([][]string, 0, len(active))
		for _, t := range active {
			rows = append(rows, []string{strconv.Itoa(t.ID), t.Name, t.StateString()})
		}
		ui.Table([]string{"ID", "NAME", "STATE"}, rows)
	}
}

func countStr(n int) string {
	if n < 0 {
		return "?"
	}
	return strconv.Itoa(n)
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	out := items[0]
	for _, s := range items[1:] {
		out += "," + s
	}
	return out
}

func activeTasks(tasks []client.Task) []client.Task {
	var active []client.Task
	for _, t := range tasks {
		if !t.Done() {
			active = append(active, t)
		}
	}
	return active
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverNoCounts, "no-counts", false, "skip counting packages per repository")
	rootCmd.AddCommand(discoverCmd)
}
