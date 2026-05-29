package commands

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0dot77/td-cli/internal/client"
)

func TestDoctorLiveChecksExerciseCoreLiveRoutes(t *testing.T) {
	seen := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path]++
		switch r.URL.Path {
		case "/exec":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"stdout":"","stderr":"","result":""}}`))
		case "/screenshot":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":{"image":"cG5n","width":1,"height":1}}`))
		case "/ui/navigate":
			_, _ = w.Write([]byte(`{"success":true,"message":"Navigated","data":{}}`))
		case "/harness/observe":
			_, _ = w.Write([]byte(`{"success":true,"message":"Observed","data":{"graph":{"nodeCount":1,"connectionCount":0},"issues":{"issueCount":0}}}`))
		default:
			t.Fatalf("unexpected live route: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := &client.Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	checks := doctorLiveChecks(c)
	if len(checks) != 4 {
		t.Fatalf("doctorLiveChecks returned %d checks, want 4", len(checks))
	}
	for _, check := range checks {
		if check.Status != "ok" {
			t.Fatalf("check %s status = %s, want ok (%s)", check.Name, check.Status, check.Message)
		}
	}
	for _, path := range []string{"/exec", "/screenshot", "/ui/navigate", "/harness/observe"} {
		if seen[path] == 0 {
			t.Fatalf("route %s was not called", path)
		}
	}
}
