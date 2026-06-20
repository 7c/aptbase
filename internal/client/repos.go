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

// RepoPackages lists package keys in a repository, optionally filtered by an
// aptly query (e.g. `nginx (>= 1.20)`).
func (c *Client) RepoPackages(name, query string) ([]string, error) {
	q := url.Values{}
	if query != "" {
		q.Set("q", query)
	}
	var keys []string
	if err := c.get("/api/repos/"+url.PathEscape(name)+"/packages", q, &keys); err != nil {
		return nil, err
	}
	return keys, nil
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
