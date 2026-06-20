package client

import "net/url"

// UploadFiles uploads local package files into the named upload directory,
// returning the server-side file list.
func (c *Client) UploadFiles(dir string, paths []string) ([]string, error) {
	return c.uploadFiles(dir, paths)
}

// ListUploadDirs lists the existing upload directories.
func (c *Client) ListUploadDirs() ([]string, error) {
	var dirs []string
	if err := c.get("/api/files", nil, &dirs); err != nil {
		return nil, err
	}
	return dirs, nil
}

// ListUploadedFiles lists files within an upload directory.
func (c *Client) ListUploadedFiles(dir string) ([]string, error) {
	var files []string
	if err := c.get("/api/files/"+url.PathEscape(dir), nil, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// DeleteUploadDir removes an upload directory and its contents.
func (c *Client) DeleteUploadDir(dir string) error {
	return c.delete("/api/files/"+url.PathEscape(dir), nil, nil)
}
