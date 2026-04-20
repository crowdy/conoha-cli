package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Executor runs a shell command on a remote host.
// stdin is streamed to the command; stdout receives its output.
// Implementations should surface process-level failures as errors
// (exit code != 0 on the remote is NOT a process failure — curl returns 0
// even on HTTP errors because we pass -f off and parse the status ourselves).
type Executor interface {
	Run(cmd string, stdin io.Reader, stdout io.Writer) error
}

// Client speaks the conoha-proxy Admin API via an Executor.
type Client struct {
	exec Executor
	sock string
}

// NewClient constructs a Client with the given executor and socket path.
func NewClient(exec Executor, sock string) *Client {
	return &Client{exec: exec, sock: sock}
}

// Get returns a single service by name.
func (c *Client) Get(name string) (*Service, error) {
	body, err := c.call("GET", "/v1/services/"+name, nil)
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}
	return &s, nil
}

// List returns all registered services.
func (c *Client) List() ([]Service, error) {
	body, err := c.call("GET", "/v1/services", nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Services []Service `json:"services"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	return wrap.Services, nil
}

// Upsert creates or replaces a service.
func (c *Client) Upsert(req UpsertRequest) (*Service, error) {
	return c.postService("/v1/services", req)
}

// Deploy probes the new target and swaps on success.
func (c *Client) Deploy(name string, req DeployRequest) (*Service, error) {
	return c.postService("/v1/services/"+name+"/deploy", req)
}

// Rollback swaps active and draining targets within the drain window.
func (c *Client) Rollback(name string, drainMs int) (*Service, error) {
	return c.postService("/v1/services/"+name+"/rollback", RollbackRequest{DrainMs: drainMs})
}

// Delete removes the service; the drain window (if any) is discarded.
func (c *Client) Delete(name string) error {
	_, err := c.call("DELETE", "/v1/services/"+name, nil)
	return err
}

func (c *Client) postService(path string, body interface{}) (*Service, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	out, err := c.call("POST", path, data)
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}
	return &s, nil
}

// call synthesizes a curl invocation, runs it, splits status from body,
// and converts 4xx/5xx into typed errors via ParseAPIError.
func (c *Client) call(method, path string, body []byte) ([]byte, error) {
	parts := []string{
		"curl", "-sS",
		"--unix-socket", c.sock,
		"-X", method,
		"-w", `"\nHTTPSTATUS:%{http_code}"`,
	}
	if body != nil {
		parts = append(parts, "-H", "'Content-Type: application/json'", "--data-binary", "@-")
	}
	parts = append(parts, "'http://admin"+path+"'")
	cmd := strings.Join(parts, " ")

	var buf bytes.Buffer
	var stdin io.Reader
	if body != nil {
		stdin = bytes.NewReader(body)
	}
	if err := c.exec.Run(cmd, stdin, &buf); err != nil {
		return nil, fmt.Errorf("exec curl: %w", err)
	}
	respBody, status, err := splitStatus(buf.Bytes())
	if err != nil {
		return nil, err
	}
	if apiErr := ParseAPIError(status, respBody); apiErr != nil {
		return nil, apiErr
	}
	return respBody, nil
}

// splitStatus separates the trailing "\nHTTPSTATUS:NNN" line from the body.
// Accepts the body as-is if the tag is missing (maps to status 0 → error).
func splitStatus(raw []byte) (body []byte, status int, err error) {
	tag := []byte("\nHTTPSTATUS:")
	i := bytes.LastIndex(raw, tag)
	if i < 0 {
		return nil, 0, fmt.Errorf("missing HTTPSTATUS tag in curl output: %q", string(raw))
	}
	body = raw[:i]
	statusStr := strings.TrimSpace(string(raw[i+len(tag):]))
	n, convErr := strconv.Atoi(statusStr)
	if convErr != nil {
		return nil, 0, fmt.Errorf("parse HTTPSTATUS %q: %w", statusStr, convErr)
	}
	return body, n, nil
}
