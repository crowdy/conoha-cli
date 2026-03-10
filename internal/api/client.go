package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/crowdy/conoha-cli/internal/config"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

// UserAgent is the User-Agent header value sent with all requests.
var UserAgent = "crowdy/conoha-cli/dev"

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
)

// Client is the base HTTP client for ConoHa API.
type Client struct {
	HTTP     *http.Client
	Region   string
	Token    string
	TenantID string
}

// NewClient creates a new API client.
func NewClient(region, token, tenantID string) *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: defaultTimeout},
		Region:   region,
		Token:    token,
		TenantID: tenantID,
	}
}

// intServiceMap maps external service names to internal API path segments.
var intServiceMap = map[string]string{
	"image":      "image-service",
	"networking": "network",
}

// BaseURL returns the service endpoint URL.
// If CONOHA_ENDPOINT is set, it overrides the default URL.
// If CONOHA_ENDPOINT_MODE=int, the service name is appended as a path segment
// (with remapping for services that differ between ext and int APIs).
func (c *Client) BaseURL(service string) string {
	if ep := os.Getenv(config.EnvEndpoint); ep != "" {
		if os.Getenv(config.EnvEndpointMode) == "int" {
			if mapped, ok := intServiceMap[service]; ok {
				service = mapped
			}
			return ep + "/" + service
		}
		return ep
	}
	return fmt.Sprintf("https://%s.%s.conoha.io", service, c.Region)
}

// Do executes an HTTP request with auth headers and error handling.
// Note: retries only work for requests without a body (GET/DELETE).
// POST/PUT with a body are not retried because the body is consumed on first attempt.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)
	if c.Token != "" {
		req.Header.Set("X-Auth-Token", c.Token)
	}
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Read request body for debug logging
	var reqBody []byte
	if debugLevel >= DebugAPI && req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}
	debugLogRequest(req, reqBody)

	start := time.Now()
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.HTTP.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return nil, &cerrors.NetworkError{Err: err}
			}
			// Only retry if body is nil (GET/DELETE) or body supports seeking
			if req.Body != nil {
				return nil, &cerrors.NetworkError{Err: err}
			}
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		// Retry on 429 or 5xx (only for bodyless requests)
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			if attempt < maxRetries && req.Body == nil {
				resp.Body.Close()
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		}
		break
	}
	elapsed := time.Since(start)

	if resp == nil {
		return nil, &cerrors.NetworkError{Err: fmt.Errorf("no response after retries")}
	}

	// Debug log response
	if debugLevel >= DebugAPI {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		debugLogResponse(resp, elapsed, respBody)
	} else {
		debugLogResponse(resp, elapsed, nil)
	}

	if resp.StatusCode >= 400 {
		return resp, parseAPIError(resp)
	}

	return resp, nil
}

// Request creates and executes a request, returning the response.
func (c *Client) Request(method, url string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	return c.Do(req)
}

// Get performs a GET request and decodes the response into result.
func (c *Client) Get(url string, result any) error {
	resp, err := c.Request(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Post performs a POST request and decodes the response into result.
func (c *Client) Post(url string, body, result any) (*http.Response, error) {
	resp, err := c.Request(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// Put performs a PUT request.
func (c *Client) Put(url string, body, result any) error {
	resp, err := c.Request(http.MethodPut, url, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Delete performs a DELETE request.
func (c *Client) Delete(url string) error {
	resp, err := c.Request(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// parseAPIError reads the response body and returns an APIError.
func parseAPIError(resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	apiErr := &cerrors.APIError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
	}

	// Try to parse structured error
	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		apiErr.Code = errResp.Error.Code
		apiErr.Message = errResp.Error.Message
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &cerrors.AuthError{Message: apiErr.Message}
	}
	if resp.StatusCode == 404 {
		return &cerrors.NotFoundError{Resource: "resource", ID: ""}
	}

	return apiErr
}
