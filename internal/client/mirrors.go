package client

// Mirror is a remote repository mirror (GET /api/mirrors).
type Mirror struct {
	Name          string   `json:"Name"`
	ArchiveRoot   string   `json:"ArchiveRoot"`
	Distribution  string   `json:"Distribution"`
	Components    []string `json:"Components"`
	Architectures []string `json:"Architectures"`
}

// ListMirrors returns all configured mirrors.
func (c *Client) ListMirrors() ([]Mirror, error) {
	var mirrors []Mirror
	if err := c.get("/api/mirrors", nil, &mirrors); err != nil {
		return nil, err
	}
	return mirrors, nil
}
