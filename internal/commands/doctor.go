package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/0dot77/td-cli/internal/client"
	"github.com/0dot77/td-cli/internal/discovery"
	"github.com/0dot77/td-cli/internal/protocol"
)

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", "fail"
	Message string `json:"message"`
}

// Doctor runs diagnostic checks and reports setup health.
// portOverride and projectPath correspond to --port and --project global flags.
func Doctor(version string, portOverride int, projectPath string, jsonOutput bool) error {
	var checks []doctorCheck

	// 1. System info
	checks = append(checks, doctorCheck{
		Name:    "system",
		Status:  "ok",
		Message: fmt.Sprintf("td-cli v%s  %s/%s", version, runtime.GOOS, runtime.GOARCH),
	})

	// 2. Home directory writable
	home, err := os.UserHomeDir()
	if err != nil {
		checks = append(checks, doctorCheck{
			Name:    "home_directory",
			Status:  "fail",
			Message: fmt.Sprintf("cannot determine home directory: %s", err),
		})
	} else {
		testFile := filepath.Join(home, ".td-cli", ".doctor-test")
		os.MkdirAll(filepath.Join(home, ".td-cli"), 0755)
		if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
			checks = append(checks, doctorCheck{
				Name:    "home_directory",
				Status:  "fail",
				Message: fmt.Sprintf("~/.td-cli is not writable: %s", err),
			})
		} else {
			os.Remove(testFile)
			checks = append(checks, doctorCheck{
				Name:    "home_directory",
				Status:  "ok",
				Message: fmt.Sprintf("%s/.td-cli is writable", home),
			})
		}
	}

	// 3. Instances directory
	instancesDir := filepath.Join(home, ".td-cli", "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		if os.IsNotExist(err) {
			checks = append(checks, doctorCheck{
				Name:    "instances_dir",
				Status:  "warn",
				Message: "~/.td-cli/instances/ does not exist (no TD heartbeat received yet)",
			})
		} else {
			checks = append(checks, doctorCheck{
				Name:    "instances_dir",
				Status:  "fail",
				Message: fmt.Sprintf("cannot read instances directory: %s", err),
			})
		}
	} else {
		jsonCount := 0
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				jsonCount++
			}
		}
		if jsonCount == 0 {
			checks = append(checks, doctorCheck{
				Name:    "instances_dir",
				Status:  "warn",
				Message: "~/.td-cli/instances/ exists but contains no heartbeat files",
			})
		} else {
			checks = append(checks, doctorCheck{
				Name:    "instances_dir",
				Status:  "ok",
				Message: fmt.Sprintf("~/.td-cli/instances/ has %d heartbeat file(s)", jsonCount),
			})
		}
	}

	// 4. Instance discovery (active, non-stale)
	instances, err := discovery.ScanInstances()
	if err != nil {
		checks = append(checks, doctorCheck{
			Name:    "discovery",
			Status:  "fail",
			Message: fmt.Sprintf("instance scan failed: %s", err),
		})
	} else if len(instances) == 0 {
		checks = append(checks, doctorCheck{
			Name:    "discovery",
			Status:  "warn",
			Message: "no active TD instances found (heartbeat files may be stale or TD is not running)",
		})
	} else {
		for _, inst := range instances {
			state := inst.State
			if state == "" {
				state = "unknown"
			}
			checks = append(checks, doctorCheck{
				Name:   "discovery",
				Status: "ok",
				Message: fmt.Sprintf("found instance: %s on port %d [%s] (pid %d)",
					inst.ProjectName, inst.Port, state, inst.PID),
			})
		}
	}

	// 5. Port reachability
	// Priority: explicit --port > --project match > discovered instances > default 9500
	portsToCheck := []int{}
	if portOverride > 0 {
		portsToCheck = []int{portOverride}
	} else if projectPath != "" {
		for _, inst := range instances {
			if inst.ProjectPath == projectPath {
				portsToCheck = append(portsToCheck, inst.Port)
			}
		}
	}
	if len(portsToCheck) == 0 {
		for _, inst := range instances {
			portsToCheck = append(portsToCheck, inst.Port)
		}
	}
	if len(portsToCheck) == 0 {
		portsToCheck = []int{9500}
	}

	for _, p := range portsToCheck {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 2*time.Second)
		if err != nil {
			checks = append(checks, doctorCheck{
				Name:    "port",
				Status:  "fail",
				Message: fmt.Sprintf("port %d is not reachable — TD webserver may be inactive", p),
			})
		} else {
			conn.Close()
			checks = append(checks, doctorCheck{
				Name:    "port",
				Status:  "ok",
				Message: fmt.Sprintf("port %d is reachable", p),
			})
		}
	}

	// 6. Health endpoint
	var connectedHealth *protocol.HealthData
	for _, p := range portsToCheck {
		c := client.New(p, 5*time.Second)
		resp, err := c.Health()
		if err != nil {
			checks = append(checks, doctorCheck{
				Name:    "health",
				Status:  "fail",
				Message: fmt.Sprintf("port %d: health check failed — %s", p, err),
			})
			continue
		}
		if !resp.Success {
			checks = append(checks, doctorCheck{
				Name:    "health",
				Status:  "fail",
				Message: fmt.Sprintf("port %d: server error — %s", p, resp.Message),
			})
			continue
		}
		var health protocol.HealthData
		if err := json.Unmarshal(resp.Data, &health); err != nil {
			checks = append(checks, doctorCheck{
				Name:    "health",
				Status:  "warn",
				Message: fmt.Sprintf("port %d: connected but could not parse health data", p),
			})
			continue
		}
		connectedHealth = &health
		checks = append(checks, doctorCheck{
			Name:   "health",
			Status: "ok",
			Message: fmt.Sprintf("port %d: %s — TD %s (build %s), connector %s v%s",
				p, health.Project, health.TDVersion, health.TDBuild,
				health.ConnectorName, health.ConnectorVersion),
		})
	}

	// 7. Protocol version compatibility
	if connectedHealth != nil && connectedHealth.ProtocolVersion > 0 {
		if connectedHealth.ProtocolVersion == protocol.CurrentProtocolVersion {
			checks = append(checks, doctorCheck{
				Name:    "protocol",
				Status:  "ok",
				Message: fmt.Sprintf("protocol version %d matches", connectedHealth.ProtocolVersion),
			})
		} else {
			checks = append(checks, doctorCheck{
				Name:    "protocol",
				Status:  "warn",
				Message: fmt.Sprintf("CLI expects protocol v%d but connector reports v%d — update TDCliServer.tox", protocol.CurrentProtocolVersion, connectedHealth.ProtocolVersion),
			})
		}
	}

	// 8. Auth token
	token := os.Getenv("TD_CLI_TOKEN")
	if token == "" {
		checks = append(checks, doctorCheck{
			Name:    "auth",
			Status:  "ok",
			Message: "TD_CLI_TOKEN not set (authentication disabled — fine for local use)",
		})
	} else {
		checks = append(checks, doctorCheck{
			Name:    "auth",
			Status:  "ok",
			Message: "TD_CLI_TOKEN is set (authentication enabled)",
		})
	}

	// Output
	if jsonOutput {
		out, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Println("td-cli doctor")
	fmt.Println("=============")
	fmt.Println()

	hasFailure := false
	hasWarning := false
	for _, ch := range checks {
		var icon string
		switch ch.Status {
		case "ok":
			icon = "  OK"
		case "warn":
			icon = "WARN"
			hasWarning = true
		case "fail":
			icon = "FAIL"
			hasFailure = true
		}
		fmt.Printf("  [%s] %s\n", icon, ch.Message)
	}

	fmt.Println()
	if hasFailure {
		fmt.Println("Some checks failed. Common fixes:")
		fmt.Println("  1. Is TouchDesigner running with a project open?")
		fmt.Println("  2. Is TDCliServer.tox imported into the root network?")
		fmt.Println("  3. Is webserver1 inside TDCliServer set to Active?")
		fmt.Println("  4. Check port with: td-cli instances")
	} else if hasWarning {
		fmt.Println("Warnings found but no critical failures.")
	} else {
		fmt.Println("All checks passed.")
	}

	return nil
}
