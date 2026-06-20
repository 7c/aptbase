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

func TestParsePkgKey(t *testing.T) {
	got := parsePkgKey("Pamd64 nginx 1.20.1-1 a1b2c3")
	if got.Name != "nginx" || got.Version != "1.20.1-1" || got.Arch != "amd64" {
		t.Errorf("parsePkgKey = %+v", got)
	}
	if fb := parsePkgKey("garbage"); fb.Name != "garbage" {
		t.Errorf("fallback name = %q", fb.Name)
	}
}

func TestPreviewPackages(t *testing.T) {
	keys := []string{
		"Pamd64 nginx 1.18.0 h1",
		"Pamd64 nginx 1.20.1 h2",
		"Pamd64 app 0.1.0 h3",
	}
	if previewPackages(keys, 0) != nil {
		t.Error("limit 0 should return nil")
	}
	all := previewPackages(keys, -1)
	if len(all) != 3 {
		t.Fatalf("limit <0 should return all, got %d", len(all))
	}
	// Sorted by name asc, then version desc: app first, then nginx newest first.
	if all[0].Name != "app" || all[1].Version != "1.20.1" {
		t.Errorf("unexpected order: %+v", all)
	}
	if top := previewPackages(keys, 2); len(top) != 2 {
		t.Errorf("limit 2 should cap at 2, got %d", len(top))
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
