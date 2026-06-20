package client

import (
	"net/url"
	"strings"
)

// ListPublished returns all published repositories.
func (c *Client) ListPublished() ([]PublishedRepo, error) {
	var pubs []PublishedRepo
	if err := c.get("/api/publish", nil, &pubs); err != nil {
		return nil, err
	}
	return pubs, nil
}

// UpdatePublished re-publishes a (prefix, distribution) publication
// asynchronously, returning the task to wait on. Used after adding packages to
// a local repo so the changes go live.
func (c *Client) UpdatePublished(prefix, distribution string, req UpdatePublishRequest) (*Task, error) {
	q := url.Values{}
	q.Set("_async", "1")
	path := "/api/publish/" + encodePrefix(prefix) + "/" + url.PathEscape(distribution)
	var task Task
	if err := c.put(path, q, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// encodePrefix encodes a publish prefix for use in an API path following
// aptly's convention: an empty prefix becomes ".", "_" is doubled, and "/" is
// replaced with "_".
func encodePrefix(prefix string) string {
	if prefix == "" || prefix == "." {
		return ":."
	}
	encoded := strings.ReplaceAll(prefix, "_", "__")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	return encoded
}
