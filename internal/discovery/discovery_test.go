package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0dot77/td-cli/internal/protocol"
)

func TestScanInstancesFiltersStaleAndCleansFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := filepath.Join(os.Getenv("HOME"), instanceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create instance dir: %v", err)
	}

	activePath := filepath.Join(dir, "active.json")
	stalePath := filepath.Join(dir, "stale.json")

	writeInstanceFile(t, activePath, protocol.Instance{
		ProjectName: "active",
		ProjectPath: "/show/active.toe",
		Port:        9500,
		Timestamp:   float64(time.Now().Unix()),
	})
	writeInstanceFile(t, stalePath, protocol.Instance{
		ProjectName: "stale",
		ProjectPath: "/show/stale.toe",
		Port:        9501,
		Timestamp:   float64(time.Now().Add(-staleThreshold - time.Second).Unix()),
	})

	instances, err := ScanInstances()
	if err != nil {
		t.Fatalf("ScanInstances returned error: %v", err)
	}

	if len(instances) != 1 {
		t.Fatalf("expected 1 active instance, got %d", len(instances))
	}
	if instances[0].ProjectName != "active" {
		t.Fatalf("unexpected instance returned: %+v", instances[0])
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale heartbeat file to be removed, stat err=%v", err)
	}
}

func TestFindInstanceByProjectPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := filepath.Join(os.Getenv("HOME"), instanceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create instance dir: %v", err)
	}

	writeInstanceFile(t, filepath.Join(dir, "one.json"), protocol.Instance{
		ProjectName: "one",
		ProjectPath: "/show/one.toe",
		Port:        9500,
		Timestamp:   float64(time.Now().Unix()),
	})
	writeInstanceFile(t, filepath.Join(dir, "two.json"), protocol.Instance{
		ProjectName: "two",
		ProjectPath: "/show/two.toe",
		Port:        9501,
		Timestamp:   float64(time.Now().Unix()),
	})

	inst, err := FindInstance(0, "/show/two.toe")
	if err != nil {
		t.Fatalf("FindInstance returned error: %v", err)
	}
	if inst.Port != 9501 {
		t.Fatalf("expected port 9501, got %d", inst.Port)
	}
}

func TestFindInstanceMultipleRequiresSelection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := filepath.Join(os.Getenv("HOME"), instanceDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create instance dir: %v", err)
	}

	writeInstanceFile(t, filepath.Join(dir, "one.json"), protocol.Instance{
		ProjectName: "one",
		ProjectPath: "/show/one.toe",
		Port:        9500,
		Timestamp:   float64(time.Now().Unix()),
	})
	writeInstanceFile(t, filepath.Join(dir, "two.json"), protocol.Instance{
		ProjectName: "two",
		ProjectPath: "/show/two.toe",
		Port:        9501,
		Timestamp:   float64(time.Now().Unix()),
	})

	_, err := FindInstance(0, "")
	if err == nil {
		t.Fatal("expected error when multiple instances are present")
	}
	if !strings.Contains(err.Error(), "multiple TD instances running") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeInstanceFile(t *testing.T, path string, inst protocol.Instance) {
	t.Helper()

	data, err := json.Marshal(inst)
	if err != nil {
		t.Fatalf("failed to marshal instance: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write instance file: %v", err)
	}
}
