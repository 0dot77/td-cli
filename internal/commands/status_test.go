package commands

import (
	"strings"
	"testing"

	"github.com/td-cli/td-cli/internal/protocol"
)

func TestFormatConnectorLineIncludesMode(t *testing.T) {
	health := protocol.HealthData{
		ConnectorName:        "TDCliServer",
		ConnectorVersion:     "0.1.0",
		ConnectorInstallMode: "tox",
	}

	got := formatConnectorLine(health)
	want := "TDCliServer v0.1.0 (tox)"
	if got != want {
		t.Fatalf("formatConnectorLine() = %q, want %q", got, want)
	}
}

func TestCompatibilityWarningForProtocolMismatch(t *testing.T) {
	health := protocol.HealthData{
		ProtocolVersion: protocol.CurrentProtocolVersion + 1,
	}

	got := compatibilityWarning(health)
	if got == "" {
		t.Fatal("compatibilityWarning() = empty, want warning")
	}
	if !strings.Contains(got, "Update the TDCliServer TOX") {
		t.Fatalf("compatibilityWarning() = %q, want TOX update guidance", got)
	}
}

func TestCompatibilityWarningEmptyWhenProtocolsMatch(t *testing.T) {
	health := protocol.HealthData{
		ProtocolVersion: protocol.CurrentProtocolVersion,
	}

	if got := compatibilityWarning(health); got != "" {
		t.Fatalf("compatibilityWarning() = %q, want empty string", got)
	}
}
