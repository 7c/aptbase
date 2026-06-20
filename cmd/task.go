package cmd

import (
	"fmt"
	"strconv"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Inspect asynchronous aptly tasks",
	Long:  "List tasks, show a task's state, wait for completion, or print a task's output.",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks on each server",
	Args:  cobra.NoArgs,
	Example: `  aptbase task list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			tasks, err := srv.Client.ListTasks()
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(tasks)
			}
			if len(tasks) == 0 {
				ui.Dim("no tasks")
				return nil
			}
			rows := make([][]string, 0, len(tasks))
			for _, t := range tasks {
				rows = append(rows, []string{strconv.Itoa(t.ID), t.Name, t.StateString()})
			}
			ui.Table([]string{"ID", "NAME", "STATE"}, rows)
			return nil
		})
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a task's current state",
	Args:  cobra.ExactArgs(1),
	Example: `  aptbase task show 7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task id %q", args[0])
		}
		return onSingleServer(func(srv target.Server) error {
			task, err := srv.Client.GetTask(id)
			if err != nil {
				return err
			}
			if settings.JSON {
				return render.JSON(task)
			}
			ui.KeyValues([][2]string{
				{"ID", strconv.Itoa(task.ID)},
				{"Name", task.Name},
				{"State", task.StateString()},
			})
			return nil
		})
	},
}

var taskWaitCmd = &cobra.Command{
	Use:   "wait <id>",
	Short: "Wait for a task to finish, streaming its output",
	Args:  cobra.ExactArgs(1),
	Example: `  aptbase task wait 7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task id %q", args[0])
		}
		return onSingleServer(func(srv target.Server) error {
			return runTask(srv, &client.Task{ID: id, Name: "task " + args[0]}, "waiting")
		})
	},
}

var taskOutputCmd = &cobra.Command{
	Use:   "output <id>",
	Short: "Print a task's accumulated output",
	Args:  cobra.ExactArgs(1),
	Example: `  aptbase task output 7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task id %q", args[0])
		}
		return onSingleServer(func(srv target.Server) error {
			out, err := srv.Client.TaskOutput(id)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		})
	},
}

// onSingleServer runs fn against the first configured server. Task IDs are
// server-local, so task subcommands act on one server (use --api/--server to
// pick which).
func onSingleServer(fn func(target.Server) error) error {
	set, err := resolveTargets()
	if err != nil {
		return err
	}
	srv := set.Servers[0]
	if len(set.Servers) > 1 {
		ui.Warn("multiple servers configured; using %s (task IDs are server-local)", srv.URL)
	}
	return fn(srv)
}

func init() {
	taskCmd.AddCommand(taskListCmd, taskShowCmd, taskWaitCmd, taskOutputCmd)
	rootCmd.AddCommand(taskCmd)
}
