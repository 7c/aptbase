package client

// Version returns the remote aptly version (GET /api/version).
func (c *Client) Version() (*Version, error) {
	var v Version
	if err := c.get("/api/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
