package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractError(t *testing.T) {
	cases := map[string]string{
		`{"error":"boom"}`:    "boom",
		`[{"error":"boom"}]`:  "boom",
		`plain text problem`:  "plain text problem",
		``:                    "",
	}
	for body, want := range cases {
		if got := extractError([]byte(body)); got != want {
			t.Errorf("extractError(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestAPIErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"repo not found"}`))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL})
	_, err := c.ListRepos()
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != http.StatusBadRequest || apiErr.Message != "repo not found" {
		t.Errorf("got %+v", apiErr)
	}
}

func TestQueryEncoding(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL})
	if _, err := c.RepoPackages("app", "nginx (>= 1.20)"); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "nginx (>= 1.20)" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestPrefixEncoding(t *testing.T) {
	cases := map[string]string{
		"":         ":.",
		".":        ":.",
		"debian":   "debian",
		"a/b":      "a_b",
		"a_b":      "a__b",
		"deb/a_b":  "deb_a__b",
	}
	for in, want := range cases {
		if got := encodePrefix(in); got != want {
			t.Errorf("encodePrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAsyncAddSetsFlag(t *testing.T) {
	var gotAsync string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAsync = r.URL.Query().Get("_async")
		w.Write([]byte(`{"ID":5,"Name":"add","State":1}`))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL})
	task, err := c.AddPackagesFromDir("app", "dir-1")
	if err != nil {
		t.Fatal(err)
	}
	if gotAsync != "1" {
		t.Errorf("_async = %q, want 1", gotAsync)
	}
	if task.ID != 5 {
		t.Errorf("task.ID = %d", task.ID)
	}
}

// stubPrompter supplies fixed credentials without touching a terminal.
type stubPrompter struct {
	user, pass string
	calls      int
}

func (s *stubPrompter) Credentials(realm, defaultUser string) (string, string, error) {
	s.calls++
	return s.user, s.pass, nil
}

func TestAuthPromptOn401(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="aptly"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		authHeader = r.Header.Get("Authorization")
		w.Write([]byte(`{"Version":"1.5.0"}`))
	}))
	defer srv.Close()

	prompt := &stubPrompter{user: "deploy", pass: "secret"}
	c := New(Options{BaseURL: srv.URL, Prompt: prompt})
	v, err := c.Version()
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "1.5.0" {
		t.Errorf("version = %q", v.Version)
	}
	if prompt.calls != 1 {
		t.Errorf("prompt called %d times, want 1", prompt.calls)
	}
	if authHeader == "" {
		t.Error("retry should carry Authorization header")
	}
}

func TestNoPromptWhenPreauthed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"1.5.0"}`))
	}))
	defer srv.Close()

	prompt := &stubPrompter{user: "x", pass: "y"}
	c := New(Options{BaseURL: srv.URL, User: "u", Password: "p", HasAuth: true, Prompt: prompt})
	if _, err := c.Version(); err != nil {
		t.Fatal(err)
	}
	if prompt.calls != 0 {
		t.Errorf("prompt should not be called when preauthed, got %d", prompt.calls)
	}
}
