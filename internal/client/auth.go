package client

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Prompter supplies Basic-auth credentials when the server issues a 401
// challenge and none were preconfigured.
type Prompter interface {
	// Credentials returns the username and password to authenticate with.
	// defaultUser is the configured user (may be empty).
	Credentials(realm, defaultUser string) (user, pass string, err error)
}

// TerminalPrompter reads credentials interactively from the controlling
// terminal, reading the password without echo.
type TerminalPrompter struct{}

// Credentials implements Prompter against the terminal. It fails clearly when
// no interactive terminal is attached (e.g. in scripts or --json runs) rather
// than blocking.
func (TerminalPrompter) Credentials(realm, defaultUser string) (string, string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", "", fmt.Errorf("authentication required%s but no interactive terminal is available; pass --user/--password", realmSuffix(realm))
	}

	user := defaultUser
	if user == "" {
		fmt.Fprintf(os.Stderr, "Username%s: ", realmSuffix(realm))
		var entered string
		if _, err := fmt.Fscanln(os.Stdin, &entered); err != nil && err.Error() != "unexpected newline" {
			return "", "", fmt.Errorf("reading username: %w", err)
		}
		user = strings.TrimSpace(entered)
	}

	fmt.Fprintf(os.Stderr, "Password for %s%s: ", user, realmSuffix(realm))
	pw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", "", fmt.Errorf("reading password: %w", err)
	}
	return user, string(pw), nil
}

func realmSuffix(realm string) string {
	if realm == "" {
		return ""
	}
	return fmt.Sprintf(" (%s)", realm)
}
