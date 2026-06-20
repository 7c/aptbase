package client

import (
	"fmt"
	"strings"
	"time"
)

// taskPollInterval is how often StreamTask polls for new output.
const taskPollInterval = 600 * time.Millisecond

// ListTasks returns all known tasks.
func (c *Client) ListTasks() ([]Task, error) {
	var tasks []Task
	if err := c.get("/api/tasks", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTask returns a task's current state.
func (c *Client) GetTask(id int) (*Task, error) {
	var task Task
	if err := c.get(fmt.Sprintf("/api/tasks/%d", id), nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// WaitTask blocks server-side until the task reaches a terminal state.
func (c *Client) WaitTask(id int) (*Task, error) {
	var task Task
	if err := c.get(fmt.Sprintf("/api/tasks/%d/wait", id), nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// TaskOutput returns the accumulated output text of a task.
func (c *Client) TaskOutput(id int) (string, error) {
	var out string
	if err := c.get(fmt.Sprintf("/api/tasks/%d/output", id), nil, &out); err != nil {
		return "", err
	}
	return out, nil
}

// StreamTask polls a task until completion, invoking onChunk with each newly
// appended slice of output for live progress. It returns the final task state.
func (c *Client) StreamTask(id int, onChunk func(string)) (*Task, error) {
	var seen int
	for {
		task, err := c.GetTask(id)
		if err != nil {
			return nil, err
		}
		if onChunk != nil {
			if out, oerr := c.TaskOutput(id); oerr == nil && len(out) > seen {
				onChunk(out[seen:])
				seen = len(out)
			}
		}
		if task.Done() {
			return task, nil
		}
		time.Sleep(taskPollInterval)
	}
}

// TaskError builds an error describing a failed task, including its tail output.
func TaskError(task *Task, output string) error {
	msg := strings.TrimSpace(output)
	if msg == "" {
		return fmt.Errorf("task %d (%s) failed", task.ID, task.Name)
	}
	return fmt.Errorf("task %d (%s) failed: %s", task.ID, task.Name, msg)
}
