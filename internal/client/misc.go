package client

// Authenticated reports whether the client is currently sending Basic-auth
// credentials (because they were configured or supplied via a 401 prompt).
func (c *Client) Authenticated() bool { return c.hasAuth }

// Version returns the remote aptly version (GET /api/version).
func (c *Client) Version() (*Version, error) {
	var v Version
	if err := c.get("/api/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
