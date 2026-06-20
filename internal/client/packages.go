package client

import (
	"encoding/json"
	"net/url"
)

// ShowPackage returns the raw detail map for a package key (e.g.
// "Pamd64 nginx 1.20.1-1 abc..."). The shape varies by package, so it is
// returned as a generic map.
func (c *Client) ShowPackage(key string) (map[string]any, error) {
	var raw json.RawMessage
	if err := c.get("/api/packages/"+url.PathEscape(key), nil, &raw); err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}
