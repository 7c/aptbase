package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/target"
)

// prompter returns the Prompter used for 401-triggered auth. In --json
// (non-interactive intent) runs it still attempts to prompt, but the terminal
// prompter fails clearly when no TTY is attached.
func prompter() client.Prompter {
	return client.TerminalPrompter{}
}

// resolveTargets builds the fan-out set of servers from the resolved settings.
func resolveTargets() (*target.Set, error) {
	return target.Resolve(settings, prompter())
}

// confirm asks the user to confirm a destructive action unless --yes was given.
// It returns true when the action should proceed.
func confirm(prompt string) bool {
	if settings.Yes {
		return true
	}
	cFlag.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
