package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseResponseJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(strings.NewReader(`{
			"success": true,
			"message": "ok",
			"data": {"value": 1}
		}`)),
	}

	result, err := parseResponse(resp)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success=true, got false")
	}
	if result.Message != "ok" {
		t.Fatalf("expected message ok, got %q", result.Message)
	}
	if string(result.Data) != `{"value": 1}` {
		t.Fatalf("unexpected data payload: %s", string(result.Data))
	}
}

func TestParseResponsePlainTextFallback(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("plain error")),
	}

	result, err := parseResponse(resp)
	if err != nil {
		t.Fatalf("parseResponse returned error: %v", err)
	}

	if result.Success {
		t.Fatalf("expected success=false for HTTP 500")
	}
	if result.Message != "plain error" {
		t.Fatalf("expected plain text message, got %q", result.Message)
	}
}

func TestHealthSendsAuthHeader(t *testing.T) {
	t.Parallel()

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-TD-CLI-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"version":"0.1.0"}}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Token:      "secret-token",
	}

	if _, err := client.Health(); err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if gotHeader != "secret-token" {
		t.Fatalf("expected auth header to be sent, got %q", gotHeader)
	}
}

func TestCallSendsAuthHeader(t *testing.T) {
	t.Parallel()

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-TD-CLI-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{}}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Token:      "secret-token",
	}

	if _, err := client.Call("/status", map[string]string{"ping": "pong"}); err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if gotHeader != "secret-token" {
		t.Fatalf("expected auth header to be sent, got %q", gotHeader)
	}
}
