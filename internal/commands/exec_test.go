package commands

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0dot77/td-cli/internal/client"
)

func TestExecWritesScreenshotToRequestedOutputFile(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "preview.png")
	c := newExecTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/exec":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"stdout":"","stderr":"","result":""}}`))
		case "/screenshot":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"image":"cG5n","width":2,"height":3}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	if err := Exec(c, "print('ok')", "", false, "", false, "/project1/out", outputFile); err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}

	got, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("expected screenshot file to be written: %v", err)
	}
	if string(got) != "png" {
		t.Fatalf("screenshot file = %q, want %q", string(got), "png")
	}
}

func TestExecVerifyStrictReturnsErrorWhenVerifyReportsIssues(t *testing.T) {
	c := newExecTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/exec":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"stdout":"","stderr":"","result":""}}`))
		case "/harness/observe":
			_, _ = w.Write([]byte(`{
				"success": true,
				"message": "observed",
				"data": {
					"graph": {"nodeCount": 2, "connectionCount": 1},
					"issues": {
						"issueCount": 1,
						"targetErrors": [],
						"targetWarnings": [],
						"nodes": [
							{"path": "/project1/bad", "errors": ["NameError"], "warnings": []}
						]
					}
				}
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	err := Exec(c, "print('ok')", "", false, "/project1", true, "", "")
	if err == nil {
		t.Fatalf("Exec returned nil error, want strict verify failure")
	}
	if !strings.Contains(err.Error(), "verify strict failed") {
		t.Fatalf("Exec error = %q, want verify strict failure", err.Error())
	}
}

func newExecTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &client.Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
}
