package target

import (
	"errors"
	"testing"
	"time"

	"github.com/7c/aptbase/internal/config"
)

func TestResolveBuildsClientPerAPI(t *testing.T) {
	s := &config.Settings{
		APIs:          []string{"http://a:8080", "http://b:8080"},
		Distributions: []string{"noble"},
		Repos:         []string{"app"},
		Timeout:       30 * time.Second,
	}
	set, err := Resolve(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(set.Servers))
	}
	if set.Servers[0].URL != "http://a:8080" {
		t.Errorf("server[0] = %q", set.Servers[0].URL)
	}
}

func TestResolveRequiresAPI(t *testing.T) {
	if _, err := Resolve(&config.Settings{}, nil); err == nil {
		t.Fatal("expected error when no API configured")
	}
}

func TestForEachServerAggregates(t *testing.T) {
	s := &config.Settings{APIs: []string{"http://a", "http://b", "http://c"}}
	set, _ := Resolve(s, nil)

	results := set.ForEachServer(func(srv Server) error {
		if srv.URL == "http://b" {
			return errors.New("down")
		}
		return nil
	})
	if len(results) != 3 {
		t.Fatalf("got %d results", len(results))
	}
	err := AggregateError(results)
	if err == nil {
		t.Fatal("expected aggregate error")
	}
}

func TestAggregateErrorNilOnSuccess(t *testing.T) {
	results := []Result{{URL: "a"}, {URL: "b"}}
	if err := AggregateError(results); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
