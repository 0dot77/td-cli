package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/td-cli/td-cli/internal/protocol"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
	port       int
}

func New(port int, timeout time.Duration) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Token: strings.TrimSpace(os.Getenv("TD_CLI_TOKEN")),
		port:  port,
	}
}

func (c *Client) Port() int {
	return c.port
}

// Health checks the server health endpoint.
func (c *Client) Health() (*protocol.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.applyAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to TD at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

// Call sends a POST request to the given endpoint with a JSON body.
func (c *Client) Call(endpoint string, payload interface{}) (*protocol.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Token == "" {
		return
	}
	req.Header.Set("X-TD-CLI-Token", c.Token)
}

func parseResponse(resp *http.Response) (*protocol.Response, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result protocol.Response
	if err := json.Unmarshal(data, &result); err != nil {
		// Fall back to plain text
		return &protocol.Response{
			Success: resp.StatusCode == 200,
			Message: string(data),
		}, nil
	}

	return &result, nil
}
