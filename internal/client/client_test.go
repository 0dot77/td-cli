package client

import (
	"io"
	"net/http"
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
