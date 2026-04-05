package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/td-cli/td-cli/internal/protocol"
)

// Client communicates with a TouchDesigner instance via HTTP.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New creates a Client for the given port with the specified timeout.
func New(port int, timeout time.Duration) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Health checks the server health endpoint.
func (c *Client) Health() (*protocol.Response, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
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

	resp, err := c.HTTPClient.Post(
		c.BaseURL+endpoint,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp)
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
