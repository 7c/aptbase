package client

// Snapshot is a point-in-time snapshot (GET /api/snapshots).
type Snapshot struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	CreatedAt   string `json:"CreatedAt"`
}

// ListSnapshots returns all snapshots.
func (c *Client) ListSnapshots() ([]Snapshot, error) {
	var snapshots []Snapshot
	if err := c.get("/api/snapshots", nil, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}
