package commands

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/td-cli/td-cli/internal/protocol"
)

func TestRenderHarnessCapabilitiesIncludesCoreSections(t *testing.T) {
	resp := &protocol.Response{
		Success: true,
		Data: mustMarshalJSON(t, protocol.HarnessCapabilitiesData{
			Project:           "Example.toe",
			TDVersion:         "2023.10000",
			TDBuild:           "10000",
			ConnectorName:     "TDCliServer",
			ConnectorVersion:  "0.2.0",
			ProtocolVersion:   1,
			SupportedRoutes:   []string{"/harness/observe", "/harness/apply"},
			SupportedFamilies: []string{"TOP", "POP"},
			Features:          []string{"rollback", "history"},
			Warnings:          []string{"experimental"},
		}),
	}

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
		"Warnings:",
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
			Scope: "/project1",
			Entries: []protocol.HarnessHistoryEntry{
				{
					Timestamp:      1712900000,
					Route:          "/harness/apply",
					Scope:          "/project1",
					Goal:           "stabilize output",
					RequestID:      "req-1",
					RollbackHandle: "ckpt-1",
					Applied:        true,
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
		"/harness/apply",
		"APPLIED",
		"request=req-1",
		"rollback=ckpt-1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestHistoryStatusPrefersDryRun(t *testing.T) {
	got := historyStatus(protocol.HarnessHistoryEntry{
		Applied: true,
		DryRun:  true,
	})

	if got != "DRY-RUN" {
		t.Fatalf("historyStatus() = %q, want %q", got, "DRY-RUN")
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
