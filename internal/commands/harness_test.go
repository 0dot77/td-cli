package commands

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/0dot77/td-cli/internal/protocol"
)

func TestRenderHarnessCapabilitiesIncludesCoreSections(t *testing.T) {
	var data protocol.HarnessCapabilitiesData
	data.Runtime.ProjectName = "Example.toe"
	data.Runtime.TDVersion = "2023.10000"
	data.Runtime.TDBuild = "10000"
	data.Connector.Name = "TDCliServer"
	data.Connector.Version = "0.2.0"
	data.Connector.ProtocolVersion = 1
	data.Tools.Routes = []string{"/harness/observe", "/harness/apply"}
	data.Support.Families = map[string][]string{"TOP": {"media/*"}, "POP": {"pop/*"}}
	data.Support.Observe = true
	data.Support.Verify = true
	data.Support.Rollback = true

	resp := &protocol.Response{Success: true, Data: mustMarshalJSON(t, data)}

	out := captureStdout(t, func() {
		if err := renderHarnessCapabilities(resp); err != nil {
			t.Fatalf("renderHarnessCapabilities() error = %v", err)
		}
	})

	for _, want := range []string{
		"Harness capabilities: Example.toe",
		"Connector: TDCliServer v0.2.0",
		"Routes:",
		"Families:",
		"Features:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderHarnessHistoryShowsEntryStatusAndDetails(t *testing.T) {
	resp := &protocol.Response{
		Success: true,
		Data: mustMarshalJSON(t, protocol.HarnessHistoryData{
			Iterations: []protocol.HarnessHistoryEntry{
				{
					CreatedAt:  1712900000,
					Status:     "applied",
					TargetPath: "/project1",
					Goal:       "stabilize output",
					RecordPath: "/tmp/run.json",
				},
			},
		}),
	}

	out := captureStdout(t, func() {
		if err := renderHarnessHistory(resp); err != nil {
			t.Fatalf("renderHarnessHistory() error = %v", err)
		}
	})

	for _, want := range []string{
		"Harness history",
		"/project1",
		"APPLIED",
		"goal=stabilize output",
		"record=/tmp/run.json",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestHistoryStatusUsesUppercaseStatus(t *testing.T) {
	got := historyStatus(protocol.HarnessHistoryEntry{Status: "rolled-back"})
	if got != "ROLLED_BACK" {
		t.Fatalf("historyStatus() = %q, want %q", got, "ROLLED_BACK")
	}
}

func mustMarshalJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	return string(data)
}
