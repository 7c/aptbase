package cmd

import (
	"testing"

	"github.com/7c/aptbase/internal/client"
)

func TestCountStr(t *testing.T) {
	if countStr(-1) != "?" {
		t.Errorf("countStr(-1) = %q, want ?", countStr(-1))
	}
	if countStr(0) != "0" {
		t.Errorf("countStr(0) = %q, want 0", countStr(0))
	}
	if countStr(12) != "12" {
		t.Errorf("countStr(12) = %q, want 12", countStr(12))
	}
}

func TestJoinOrDash(t *testing.T) {
	if joinOrDash(nil) != "-" {
		t.Error("empty should be -")
	}
	if got := joinOrDash([]string{"amd64", "arm64"}); got != "amd64,arm64" {
		t.Errorf("got %q", got)
	}
}

func TestActiveTasks(t *testing.T) {
	tasks := []client.Task{
		{ID: 1, State: client.TaskSucceeded},
		{ID: 2, State: client.TaskRunning},
		{ID: 3, State: client.TaskInit},
		{ID: 4, State: client.TaskFailed},
	}
	active := activeTasks(tasks)
	if len(active) != 2 {
		t.Fatalf("got %d active, want 2 (running + init)", len(active))
	}
	if active[0].ID != 2 || active[1].ID != 3 {
		t.Errorf("unexpected active tasks: %+v", active)
	}
}
