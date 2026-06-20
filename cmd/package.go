package cmd

import (
	"fmt"
	"sort"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"pkg"},
	Short:   "List, search, and inspect packages",
	Long:    "List or search packages across local repositories (aptly query syntax) and show package details.",
}

// Shared flags for `package list` and `package search`.
var (
	pkgRepos   []string
	pkgLatest  bool
	pkgDetails bool
	pkgArchs   []string
	pkgLimit   int
	pkgSort    string
	pkgKeys    bool
)

func addPackageQueryFlags(c *cobra.Command) {
	f := c.Flags()
	f.StringArrayVar(&pkgRepos, "repo", nil, "repo to query (repeatable; default: configured repos)")
	f.BoolVar(&pkgLatest, "latest", false, "only the newest version of each package")
	f.BoolVar(&pkgDetails, "details", false, "show full records (version, arch, size, maintainer)")
	f.StringArrayVar(&pkgArchs, "arch", nil, "filter by architecture (repeatable)")
	f.IntVar(&pkgLimit, "limit", 0, "max packages to show per repo (0 = all)")
	f.StringVar(&pkgSort, "sort", "name", "sort order: name or version")
	f.BoolVar(&pkgKeys, "keys", false, "print raw aptly package keys instead of a table")
}

var packageListCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List packages in repositories",
	Long: `List packages in the target repositories (--repo, else configured repos).

An optional aptly query narrows the listing; with no query, everything is
listed. By default packages are shown as a NAME / VERSION / ARCH table grouped
by repo.`,
	Example: `  aptbase package list
  aptbase package list --latest
  aptbase package list --details --repo app-stable
  aptbase package list 'nginx' --arch amd64 --limit 20
  aptbase package list --keys`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPackageQuery(firstArg(args))
	},
}

var packageSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search packages in repositories by aptly query",
	Long: `Search packages using aptly's query syntax.

aptly has no global package index, so the search runs against each target repo
(--repo, else the configured 'repos'). Supports the same options as 'list'.`,
	Example: `  aptbase package search 'nginx'
  aptbase package search 'nginx (>= 1.20)' --repo app-stable
  aptbase package search 'Name (% myapp*)' --latest --keys`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPackageQuery(args[0])
	},
}

// runPackageQuery is the shared engine for `list` and `search`.
func runPackageQuery(query string) error {
	if pkgDetails && pkgKeys {
		return fmt.Errorf("choose only one of --details or --keys")
	}
	if pkgSort != "name" && pkgSort != "version" {
		return fmt.Errorf("invalid --sort %q (want name or version)", pkgSort)
	}

	repos := pkgRepos
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
	q := client.PackageQuery{Query: query, MaximumVersion: pkgLatest}

	return forEachServer(set, func(srv target.Server) error {
		if pkgDetails {
			return packageDetails(srv, repos, q)
		}
		return packageKeys(srv, repos, q)
	})
}

// packageKeys handles the default parsed-table output and the --keys raw output.
func packageKeys(srv target.Server, repos []string, q client.PackageQuery) error {
	type repoResult struct {
		Repo     string        `json:"repo"`
		Packages []pkgPreview  `json:"packages,omitempty"`
		Keys     []string      `json:"keys,omitempty"`
	}
	var results []repoResult
	total := 0

	for _, repo := range repos {
		keys, err := srv.Client.RepoPackageKeys(repo, q)
		if err != nil {
			return err
		}
		entries, matched := selectEntries(keys)
		total += len(entries)

		rr := repoResult{Repo: repo}
		for _, e := range entries {
			if pkgKeys {
				rr.Keys = append(rr.Keys, e.Key)
			} else {
				rr.Packages = append(rr.Packages, e.pkgPreview)
			}
		}
		results = append(results, rr)

		if !settings.JSON && len(entries) > 0 {
			ui.Heading(repo)
			if pkgKeys {
				for _, e := range entries {
					ui.Info("%s", e.Key)
				}
			} else {
				rows := make([][]string, 0, len(entries))
				for _, e := range entries {
					rows = append(rows, []string{e.Name, e.Version, e.Arch})
				}
				ui.Table([]string{"NAME", "VERSION", "ARCH"}, rows)
			}
			if matched > len(entries) {
				ui.Dim("showing %d of %d (--limit)", len(entries), matched)
			}
		}
	}

	if settings.JSON {
		return render.JSON(results)
	}
	if total == 0 {
		ui.Dim("no matches")
	}
	return nil
}

// packageDetails handles --details: full records as a wide table or JSON.
func packageDetails(srv target.Server, repos []string, q client.PackageQuery) error {
	type repoResult struct {
		Repo     string           `json:"repo"`
		Packages []map[string]any `json:"packages"`
	}
	var results []repoResult
	total := 0

	for _, repo := range repos {
		records, err := srv.Client.RepoPackageDetails(repo, q)
		if err != nil {
			return err
		}
		records = filterSortLimitRecords(records)
		total += len(records)
		results = append(results, repoResult{Repo: repo, Packages: records})

		if !settings.JSON && len(records) > 0 {
			ui.Heading(repo)
			rows := make([][]string, 0, len(records))
			for _, r := range records {
				rows = append(rows, []string{
					getStr(r, "Package"),
					getStr(r, "Version"),
					getStr(r, "Architecture"),
					getStr(r, "Size"),
					getStr(r, "Maintainer"),
				})
			}
			ui.Table([]string{"NAME", "VERSION", "ARCH", "SIZE", "MAINTAINER"}, rows)
		}
	}

	if settings.JSON {
		return render.JSON(results)
	}
	if total == 0 {
		ui.Dim("no matches")
	}
	return nil
}

// pkgEntry is a parsed package key plus its original raw key.
type pkgEntry struct {
	Key string
	pkgPreview
}

// selectEntries parses keys, applies --arch and (client-side) --latest, sorts
// per --sort, and caps to --limit. It returns the (possibly truncated) entries
// and the number that matched before the limit.
func selectEntries(keys []string) (entries []pkgEntry, matched int) {
	for _, k := range keys {
		e := pkgEntry{Key: k, pkgPreview: parsePkgKey(k)}
		if archMatch(e.Arch, pkgArchs) {
			entries = append(entries, e)
		}
	}
	if pkgLatest {
		entries = latestEntries(entries)
	}
	sort.Slice(entries, func(i, j int) bool {
		return lessPkg(entries[i].Name, entries[i].Version, entries[j].Name, entries[j].Version, pkgSort)
	})
	matched = len(entries)
	if pkgLimit > 0 && len(entries) > pkgLimit {
		entries = entries[:pkgLimit]
	}
	return entries, matched
}

// latestEntries keeps only the highest version of each (name, arch).
func latestEntries(entries []pkgEntry) []pkgEntry {
	best := map[string]int{} // name\x00arch -> index into out
	var out []pkgEntry
	for _, e := range entries {
		key := e.Name + "\x00" + e.Arch
		if idx, ok := best[key]; ok {
			if compareVersions(e.Version, out[idx].Version) > 0 {
				out[idx] = e
			}
			continue
		}
		best[key] = len(out)
		out = append(out, e)
	}
	return out
}

// filterSortLimitRecords applies arch filter, --latest, sort, and limit to
// detail records.
func filterSortLimitRecords(records []map[string]any) []map[string]any {
	filtered := records[:0:0]
	for _, r := range records {
		if archMatch(getStr(r, "Architecture"), pkgArchs) {
			filtered = append(filtered, r)
		}
	}
	if pkgLatest {
		filtered = latestRecords(filtered)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return lessPkg(
			getStr(filtered[i], "Package"), getStr(filtered[i], "Version"),
			getStr(filtered[j], "Package"), getStr(filtered[j], "Version"), pkgSort)
	})
	if pkgLimit > 0 && len(filtered) > pkgLimit {
		filtered = filtered[:pkgLimit]
	}
	return filtered
}

// latestRecords keeps only the highest version of each (Package, Architecture).
func latestRecords(records []map[string]any) []map[string]any {
	best := map[string]int{}
	var out []map[string]any
	for _, r := range records {
		key := getStr(r, "Package") + "\x00" + getStr(r, "Architecture")
		if idx, ok := best[key]; ok {
			if compareVersions(getStr(r, "Version"), getStr(out[idx], "Version")) > 0 {
				out[idx] = r
			}
			continue
		}
		best[key] = len(out)
		out = append(out, r)
	}
	return out
}

// lessPkg orders by name asc then version desc, or by version desc then name
// asc when sortBy == "version". Versions compare with Debian semantics.
func lessPkg(ni, vi, nj, vj, sortBy string) bool {
	if sortBy == "version" {
		if c := compareVersions(vi, vj); c != 0 {
			return c > 0
		}
		return ni < nj
	}
	if ni != nj {
		return ni < nj
	}
	return compareVersions(vi, vj) > 0
}

// archMatch reports whether arch passes the --arch filter (empty = allow all).
func archMatch(arch string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == arch {
			return true
		}
	}
	return false
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return toStr(v)
	}
	return ""
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
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
	addPackageQueryFlags(packageListCmd)
	addPackageQueryFlags(packageSearchCmd)
	packageCmd.AddCommand(packageListCmd, packageSearchCmd, packageShowCmd)
	rootCmd.AddCommand(packageCmd)
}
