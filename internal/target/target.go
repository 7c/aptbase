// Package target resolves the set of aptly servers (and default repos /
// distributions) an invocation acts on, and runs work across servers with an
// aggregated result.
package target

import (
	"errors"
	"fmt"

	"github.com/7c/aptbase/internal/client"
	"github.com/7c/aptbase/internal/config"
)

// Server pairs a resolved API URL with its client.
type Server struct {
	URL    string
	Client *client.Client
}

// Set is the resolved fan-out scope for a command.
type Set struct {
	Servers       []Server
	Repos         []string
	Distributions []string
}

// Resolve builds one client per configured API URL. prompt is used for
// 401-triggered authentication when no password was supplied.
func Resolve(s *config.Settings, prompt client.Prompter) (*Set, error) {
	if len(s.APIs) == 0 {
		return nil, errors.New("no aptly API URL configured; set --api or 'api' in config.ini")
	}
	servers := make([]Server, 0, len(s.APIs))
	for _, api := range s.APIs {
		c := client.New(client.Options{
			BaseURL:  api,
			User:     s.User,
			Password: s.Password,
			HasAuth:  s.HasPassword,
			Insecure: s.Insecure,
			Timeout:  s.Timeout,
			Prompt:   prompt,
		})
		servers = append(servers, Server{URL: api, Client: c})
	}
	return &Set{
		Servers:       servers,
		Repos:         s.Repos,
		Distributions: s.Distributions,
	}, nil
}

// Result captures the outcome of running work against one server.
type Result struct {
	URL string
	Err error
}

// OK reports whether the work succeeded.
func (r Result) OK() bool { return r.Err == nil }

// ForEachServer runs fn against every server sequentially, collecting results.
// A failure on one server does not stop the others.
func (set *Set) ForEachServer(fn func(Server) error) []Result {
	results := make([]Result, 0, len(set.Servers))
	for _, srv := range set.Servers {
		results = append(results, Result{URL: srv.URL, Err: fn(srv)})
	}
	return results
}

// AggregateError returns a combined error if any result failed, else nil.
func AggregateError(results []Result) error {
	var failed []string
	for _, r := range results {
		if !r.OK() {
			failed = append(failed, r.URL)
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return fmt.Errorf("%d of %d server(s) failed: %v", len(failed), len(results), failed)
}
