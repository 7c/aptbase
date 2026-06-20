package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultTemplate is the annotated config.ini written by `aptbase config new`.
const DefaultTemplate = `# aptbase configuration
#
# Values here are overridden by APTBASE_* environment variables, which are in
# turn overridden by command-line flags (last value wins). List values accept
# whitespace- or comma-separated entries.

[default]
# One or more aptly API base URLs. Commands fan out to all of them unless
# narrowed with --api or --server.
api           = http://localhost:8080

# Basic-auth username and password. Both are optional:
#   - Omit 'password' and aptbase prompts for it (hidden) only if the server
#     answers 401 — best for interactive use.
#   - Set 'password' here (or APTBASE_PASSWORD, or --password) to authenticate
#     without a prompt — best for automation/CI.
# If you store the password here, restrict the file: chmod 600 this config.
user          =
# password    = changeme

# Default distributions used by publish/deploy when -d/--distribution is omitted.
distributions = noble jammy focal

# Default local repos used when the <repo> argument is omitted.
repos         = app-stable

# Default publish prefix.
prefix        = .

# TLS / networking.
insecure      = false
timeout       = 60s

# Output defaults.
json          = false
no-color      = false

# Example of a named server, selected with: aptbase --server staging ...
# [server:staging]
# api           = http://staging:8080
# distributions = noble
`

// WriteDefault writes the annotated default template to path. It refuses to
// overwrite an existing file unless force is true, creating parent directories
// as needed.
func WriteDefault(path string, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(DefaultTemplate), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
