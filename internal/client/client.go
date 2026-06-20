// Package client is a typed Go client for the aptly REST API.
//
// It decodes aptly's JSON error envelope into friendly errors, supports
// optional HTTP Basic auth that is triggered only when the server answers 401,
// and wraps aptly's asynchronous task API for live progress reporting.
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/7c/aptbase/internal/debug"
)

func now() time.Time                  { return time.Now() }
func since(t time.Time) time.Duration { return time.Since(t).Round(time.Millisecond) }

// Client talks to a single aptly API server.
type Client struct {
	baseURL string
	http    *http.Client
	user    string
	pass    string
	hasAuth bool // credentials are known (configured or prompted)
	prompt  Prompter
}

// Options configures a Client.
type Options struct {
	BaseURL  string
	User     string
	Password string
	HasAuth  bool // true when a password was supplied up front
	Insecure bool
	Timeout  time.Duration
	Prompt   Prompter // used on 401 when no credentials are known
}

// New builds a Client from Options.
func New(opts Options) *Client {
	transport := &http.Transport{}
	if opts.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		http:    &http.Client{Timeout: timeout, Transport: transport},
		user:    opts.User,
		pass:    opts.Password,
		hasAuth: opts.HasAuth,
		prompt:  opts.Prompt,
	}
}

// BaseURL returns the server's base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// APIError represents a non-2xx response from aptly.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("aptly returned HTTP %d", e.Status)
	}
	return fmt.Sprintf("aptly error (HTTP %d): %s", e.Status, e.Message)
}

// get/post/put/delete are thin helpers over do.
func (c *Client) get(path string, query url.Values, out any) error {
	return c.do(http.MethodGet, path, query, nil, out)
}

func (c *Client) post(path string, query url.Values, body, out any) error {
	return c.do(http.MethodPost, path, query, body, out)
}

func (c *Client) put(path string, query url.Values, body, out any) error {
	return c.do(http.MethodPut, path, query, body, out)
}

func (c *Client) delete(path string, query url.Values, out any) error {
	return c.do(http.MethodDelete, path, query, nil, out)
}

// do performs a JSON request, retrying once with credentials if the server
// issues a 401 challenge and a Prompter is available.
func (c *Client) do(method, path string, query url.Values, body, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		bodyBytes = b
	}

	resp, err := c.send(method, path, query, "application/json", bodyBytes)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized && !c.hasAuth && c.prompt != nil {
		realm := parseRealm(resp.Header.Get("WWW-Authenticate"))
		resp.Body.Close()
		user, pass, perr := c.prompt.Credentials(realm, c.user)
		if perr != nil {
			return perr
		}
		c.user, c.pass, c.hasAuth = user, pass, true
		if resp, err = c.send(method, path, query, "application/json", bodyBytes); err != nil {
			return err
		}
	}
	return decode(resp, out)
}

// send issues a single HTTP request with the given raw body.
func (c *Client) send(method, path string, query url.Values, contentType string, body []byte) (*http.Response, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if contentType != "" && body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if c.hasAuth {
		req.SetBasicAuth(c.user, c.pass)
	}
	debug.Logf("→ %s %s (auth=%s, body=%d bytes)", method, u, authState(c), len(body))
	if len(body) > 0 {
		debug.Logf("  request body: %s", debug.Redact(body))
	}
	start := now()
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Logf("✗ %s %s failed after %s: %v", method, u, since(start), err)
		return nil, fmt.Errorf("connecting to %s: %w", c.baseURL, err)
	}
	debug.Logf("← %d %s (%s)", resp.StatusCode, u, since(start))
	return resp, nil
}

// authState renders whether and as whom a request is authenticated, never the
// password itself.
func authState(c *Client) string {
	if !c.hasAuth {
		return "n"
	}
	return "y user=" + c.user
}

// decode reads the response, mapping non-2xx into an APIError and otherwise
// JSON-decoding the body into out (when non-nil).
func decode(resp *http.Response, out any) error {
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		debug.Logf("  error body: %s", debug.Redact(data))
		return &APIError{Status: resp.StatusCode, Message: extractError(data)}
	}
	debug.Logf("  response: %d bytes", len(data))
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// extractError pulls a human message out of aptly's error body, which may be
// {"error":"..."}, [{"error":"..."}], or plain text.
func extractError(data []byte) string {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return ""
	}
	var obj struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.Error != "" {
		return obj.Error
	}
	var arr []struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 && arr[0].Error != "" {
		return arr[0].Error
	}
	return trimmed
}

// parseRealm extracts the realm from a WWW-Authenticate header, if present.
func parseRealm(header string) string {
	const marker = `realm="`
	i := strings.Index(header, marker)
	if i < 0 {
		return ""
	}
	rest := header[i+len(marker):]
	if j := strings.Index(rest, `"`); j >= 0 {
		return rest[:j]
	}
	return ""
}

// uploadFiles posts the given local files to POST /api/files/:dir as multipart
// form data (field name "file"), returning the server-side file list.
func (c *Client) uploadFiles(dir string, paths []string) ([]string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", p, err)
		}
		fw, err := mw.CreateFormFile("file", filepath.Base(p))
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("preparing upload for %s: %w", p, err)
		}
		if _, err := io.Copy(fw, f); err != nil {
			f.Close()
			return nil, fmt.Errorf("reading %s: %w", p, err)
		}
		f.Close()
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("finalizing upload: %w", err)
	}

	rawBody := buf.Bytes()
	contentType := mw.FormDataContentType()
	path := "/api/files/" + url.PathEscape(dir)

	resp, err := c.sendRaw(http.MethodPost, path, contentType, rawBody)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized && !c.hasAuth && c.prompt != nil {
		realm := parseRealm(resp.Header.Get("WWW-Authenticate"))
		resp.Body.Close()
		user, pass, perr := c.prompt.Credentials(realm, c.user)
		if perr != nil {
			return nil, perr
		}
		c.user, c.pass, c.hasAuth = user, pass, true
		if resp, err = c.sendRaw(http.MethodPost, path, contentType, rawBody); err != nil {
			return nil, err
		}
	}

	var uploaded []string
	if err := decode(resp, &uploaded); err != nil {
		return nil, err
	}
	return uploaded, nil
}

// sendRaw issues a request with a raw (already-encoded) body and content type.
func (c *Client) sendRaw(method, path, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	if c.hasAuth {
		req.SetBasicAuth(c.user, c.pass)
	}
	debug.Logf("→ %s %s (auth=%s, upload=%d bytes, %s)", method, c.baseURL+path, authState(c), len(body), contentType)
	start := now()
	resp, err := c.http.Do(req)
	if err != nil {
		debug.Logf("✗ %s %s failed after %s: %v", method, c.baseURL+path, since(start), err)
		return nil, fmt.Errorf("connecting to %s: %w", c.baseURL, err)
	}
	debug.Logf("← %d %s (%s)", resp.StatusCode, c.baseURL+path, since(start))
	return resp, nil
}
