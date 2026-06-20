package cmd

import (
	"fmt"
	"strings"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
)

// runTask streams a task's output (indented) and returns an error if it failed.
func runTask(srv target.Server, task *client.Task, label string) error {
	ui.Dim("  %s (task %d)…", label, task.ID)
	final, err := srv.Client.StreamTask(task.ID, printIndented)
	if err != nil {
		return err
	}
	if final.Failed() {
		out, _ := srv.Client.TaskOutput(task.ID)
		return client.TaskError(final, out)
	}
	return nil
}

// printIndented writes a task output chunk with a two-space indent per line.
func printIndented(chunk string) {
	for _, line := range strings.Split(strings.TrimRight(chunk, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Printf("    %s\n", line)
	}
}
