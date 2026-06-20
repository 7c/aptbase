package client

import (
	"net/url"
)

// ListRepos returns all local repositories.
func (c *Client) ListRepos() ([]Repo, error) {
	var repos []Repo
	if err := c.get("/api/repos", nil, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// ShowRepo returns details for a single local repository.
func (c *Client) ShowRepo(name string) (*Repo, error) {
	var repo Repo
	if err := c.get("/api/repos/"+url.PathEscape(name), nil, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// CreateRepo creates a new local repository.
func (c *Client) CreateRepo(req CreateRepoRequest) (*Repo, error) {
	var repo Repo
	if err := c.post("/api/repos", nil, req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// EditRepo updates a local repository's configuration.
func (c *Client) EditRepo(name string, req EditRepoRequest) (*Repo, error) {
	var repo Repo
	if err := c.put("/api/repos/"+url.PathEscape(name), nil, req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// DeleteRepo removes a local repository. When force is true, it is removed even
// if referenced by published repositories or snapshots.
func (c *Client) DeleteRepo(name string, force bool) error {
	q := url.Values{}
	if force {
		q.Set("force", "1")
	}
	return c.delete("/api/repos/"+url.PathEscape(name), q, nil)
}

// PackageQuery holds options for listing/searching packages in a repository.
type PackageQuery struct {
	Query          string // aptly query, e.g. `nginx (>= 1.20)` (empty lists all)
	MaximumVersion bool   // only the newest version of each package
	WithDeps       bool   // include packages that satisfy dependencies
}

// values builds the query string shared by the key and detail endpoints.
func (q PackageQuery) values(details bool) url.Values {
	v := url.Values{}
	if q.Query != "" {
		v.Set("q", q.Query)
	}
	if q.MaximumVersion {
		v.Set("maximumVersion", "1")
	}
	if q.WithDeps {
		v.Set("withDeps", "1")
	}
	if details {
		v.Set("format", "details")
	}
	return v
}

// RepoPackages lists package keys in a repository, optionally filtered by an
// aptly query (e.g. `nginx (>= 1.20)`).
func (c *Client) RepoPackages(name, query string) ([]string, error) {
	return c.RepoPackageKeys(name, PackageQuery{Query: query})
}

// RepoPackageKeys lists package keys in a repository with full query options.
func (c *Client) RepoPackageKeys(name string, q PackageQuery) ([]string, error) {
	var keys []string
	if err := c.get("/api/repos/"+url.PathEscape(name)+"/packages", q.values(false), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// RepoPackageDetails lists full package records (aptly format=details) in a
// repository. Each record is a map of Debian control fields (Package, Version,
// Architecture, Maintainer, Filename, Size, ...).
func (c *Client) RepoPackageDetails(name string, q PackageQuery) ([]map[string]any, error) {
	var records []map[string]any
	if err := c.get("/api/repos/"+url.PathEscape(name)+"/packages", q.values(true), &records); err != nil {
		return nil, err
	}
	return records, nil
}

// AddPackagesFromDir adds previously uploaded files (in upload dir) to a
// repository asynchronously, returning the task to wait on.
func (c *Client) AddPackagesFromDir(repo, dir string) (*Task, error) {
	q := url.Values{}
	q.Set("_async", "1")
	var task Task
	path := "/api/repos/" + url.PathEscape(repo) + "/file/" + url.PathEscape(dir)
	if err := c.post(path, q, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}
