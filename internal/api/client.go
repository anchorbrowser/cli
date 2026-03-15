package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.anchorbrowser.io"

var ErrDryRun = errors.New("dry-run request emitted")

// Options configures API client behavior.
type Options struct {
	BaseURL string
	Timeout time.Duration
	DryRun  bool
	Verbose bool
	Out     io.Writer
}

// RequestError represents non-2xx API responses.
type RequestError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Body       any    `json:"body,omitempty"`
}

func (e *RequestError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("api request failed: status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api request failed: status %d", e.StatusCode)
}

// Client performs authenticated requests against AnchorBrowser API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	dryRun     bool
	verbose    bool
	out        io.Writer
}

func New(options Options) *Client {
	base := strings.TrimSpace(options.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	out := options.Out
	if out == nil {
		out = os.Stdout
	}
	return &Client{
		baseURL: strings.TrimRight(base, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		dryRun:  options.DryRun,
		verbose: options.Verbose,
		out:     out,
	}
}

func (c *Client) JSON(ctx context.Context, apiKey, method, path string, query url.Values, body any) (any, error) {
	var payload io.Reader
	var rawBody []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		rawBody = encoded
		payload = bytes.NewReader(encoded)
	}

	endpoint, err := c.makeURL(path, query)
	if err != nil {
		return nil, err
	}
	if c.dryRun {
		return nil, c.emitDryRun(method, endpoint, rawBody, map[string]string{"content-type": "application/json"})
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("anchor-api-key", apiKey)
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}

	if c.verbose {
		fmt.Fprintf(c.out, "request: %s %s\n", method, endpoint)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	decoded, err := decodeJSONBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, buildAPIError(resp.StatusCode, decoded)
	}
	return decoded, nil
}

func (c *Client) Binary(ctx context.Context, apiKey, method, path string, query url.Values, body any) ([]byte, error) {
	var payload io.Reader
	var rawBody []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		rawBody = encoded
		payload = bytes.NewReader(encoded)
	}

	endpoint, err := c.makeURL(path, query)
	if err != nil {
		return nil, err
	}
	if c.dryRun {
		return nil, c.emitDryRun(method, endpoint, rawBody, map[string]string{"content-type": "application/json"})
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("anchor-api-key", apiKey)
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		decoded, _ := decodeJSONBytes(data)
		if decoded == nil {
			decoded = string(data)
		}
		return nil, buildAPIError(resp.StatusCode, decoded)
	}
	return data, nil
}

func (c *Client) MultipartFile(ctx context.Context, apiKey, method, path string, query url.Values, fileField, filePath string) (any, error) {
	endpoint, err := c.makeURL(path, query)
	if err != nil {
		return nil, err
	}

	if c.dryRun {
		return nil, c.emitDryRun(method, endpoint, nil, map[string]string{"multipart_file_field": fileField, "multipart_file_path": filePath})
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create multipart field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file to multipart body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("anchor-api-key", apiKey)
	req.Header.Set("content-type", writer.FormDataContentType())
	req.Header.Set("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	decoded, err := decodeJSONBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, buildAPIError(resp.StatusCode, decoded)
	}
	return decoded, nil
}

func (c *Client) makeURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", c.baseURL, err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

func (c *Client) emitDryRun(method, endpoint string, body []byte, headers map[string]string) error {
	payload := map[string]any{
		"dry_run": true,
		"method":  method,
		"url":     endpoint,
	}
	if len(headers) > 0 {
		payload["headers"] = headers
	}
	if len(body) > 0 {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err == nil {
			payload["body"] = decoded
		} else {
			payload["body"] = string(body)
		}
	}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(c.out, string(encoded))
	return ErrDryRun
}

func decodeJSONBody(r io.Reader) (any, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	decoded, err := decodeJSONBytes(data)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodeJSONBytes(data []byte) (any, error) {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func buildAPIError(statusCode int, decoded any) error {
	msg := ""
	if m, ok := decoded.(map[string]any); ok {
		msg = findMessage(m)
	}
	return &RequestError{StatusCode: statusCode, Message: msg, Body: decoded}
}

func findMessage(data map[string]any) string {
	if v, ok := data["message"].(string); ok {
		return v
	}
	if errRaw, ok := data["error"]; ok {
		if s, ok := errRaw.(string); ok {
			return s
		}
		if m, ok := errRaw.(map[string]any); ok {
			if msg, ok := m["message"].(string); ok {
				return msg
			}
		}
	}
	return ""
}
