package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0dot77/td-cli/internal/protocol"
)

const (
	instanceDir    = ".td-cli/instances"
	staleThreshold = 5 * time.Second
)

// ScanInstances reads all heartbeat files and returns active (non-stale) instances.
func ScanInstances() ([]protocol.Instance, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home directory: %w", err)
	}

	dir := filepath.Join(home, instanceDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read instances directory: %w", err)
	}

	var instances []protocol.Instance
	now := float64(time.Now().Unix())

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var inst protocol.Instance
		if err := json.Unmarshal(data, &inst); err != nil {
			continue
		}

		// Skip stale instances (timestamp older than threshold)
		if now-inst.Timestamp > staleThreshold.Seconds() {
			// Clean up stale file
			os.Remove(filepath.Join(dir, entry.Name()))
			continue
		}

		instances = append(instances, inst)
	}

	return instances, nil
}

// FindInstance resolves a single instance based on options.
// Priority: port override > project path match > single instance auto-select.
func FindInstance(port int, projectPath string) (*protocol.Instance, error) {
	if port > 0 {
		// Direct port override — don't scan files
		return &protocol.Instance{Port: port}, nil
	}

	instances, err := ScanInstances()
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no running TouchDesigner instances found\nMake sure TDCliServer is installed and running in your TD project")
	}

	if projectPath != "" {
		for i := range instances {
			if instances[i].ProjectPath == projectPath {
				return &instances[i], nil
			}
		}
		return nil, fmt.Errorf("no TD instance found for project: %s", projectPath)
	}

	if len(instances) == 1 {
		return &instances[0], nil
	}

	// Multiple instances — ask user to specify
	msg := fmt.Sprintf("multiple TD instances running (%d), specify with --port or --project:\n", len(instances))
	for _, inst := range instances {
		msg += fmt.Sprintf("  port %d  %s  (%s)\n", inst.Port, inst.ProjectName, inst.ProjectPath)
	}
	return nil, fmt.Errorf("%s", msg)
}
