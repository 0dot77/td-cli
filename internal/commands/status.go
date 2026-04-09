package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/protocol"
)

// Status checks connection to a TD instance and prints info.
func Status(c *client.Client, jsonOutput bool) error {
	resp, err := c.Health()
	if err != nil {
		return err
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("server error: %s", resp.Message)
	}

	var health protocol.HealthData
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &health); err != nil {
			return fmt.Errorf("failed to parse health data: %w", err)
		}
	}

	fmt.Println("Connected to TouchDesigner")
	fmt.Printf("  Project:    %s\n", health.Project)
	fmt.Printf("  TD Version: %s (build %s)\n", health.TDVersion, health.TDBuild)
	fmt.Printf("  Server:     %s v%s\n", "td-cli", health.Version)
	if connectorLine := formatConnectorLine(health); connectorLine != "" {
		fmt.Printf("  Connector:  %s\n", connectorLine)
	}
	if warning := compatibilityWarning(health); warning != "" {
		fmt.Printf("  Warning:    %s\n", warning)
	}
	return nil
}

// Instances lists all running TD instances.
func Instances(instances []protocol.Instance, jsonOutput bool) {
	if jsonOutput {
		out, _ := json.MarshalIndent(instances, "", "  ")
		fmt.Println(string(out))
		return
	}

	if len(instances) == 0 {
		fmt.Println("No running TouchDesigner instances found")
		return
	}

	fmt.Printf("Found %d instance(s):\n", len(instances))
	for _, inst := range instances {
		state := inst.State
		if state == "" {
			state = "unknown"
		}
		connector := ""
		if inst.ConnectorVersion != "" {
			connector = fmt.Sprintf("  connector:%s", formatInstanceConnector(inst))
		}
		fmt.Printf("  %-20s  %-12s  port:%-5d  pid:%-6d  %s%s\n",
			inst.ProjectName, "["+state+"]", inst.Port, inst.PID, inst.ProjectPath, connector)
	}
}

func formatConnectorLine(health protocol.HealthData) string {
	if health.ConnectorVersion == "" {
		return ""
	}

	label := strings.TrimSpace(health.ConnectorName)
	if label == "" {
		label = "TDCliServer"
	}

	line := fmt.Sprintf("%s v%s", label, health.ConnectorVersion)
	if mode := strings.TrimSpace(health.ConnectorInstallMode); mode != "" {
		line += fmt.Sprintf(" (%s)", mode)
	}

	return line
}

func formatInstanceConnector(inst protocol.Instance) string {
	label := strings.TrimSpace(inst.ConnectorName)
	if label == "" {
		label = "TDCliServer"
	}

	line := fmt.Sprintf("%s@%s", label, inst.ConnectorVersion)
	if mode := strings.TrimSpace(inst.ConnectorInstallMode); mode != "" {
		line += fmt.Sprintf(" (%s)", mode)
	}

	return line
}

func compatibilityWarning(health protocol.HealthData) string {
	if health.ProtocolVersion == 0 || health.ProtocolVersion == protocol.CurrentProtocolVersion {
		return ""
	}

	return fmt.Sprintf(
		"CLI protocol v%d expects a matching connector, but this project reports protocol v%d. Update the TDCliServer TOX before using AI-driven edits.",
		protocol.CurrentProtocolVersion,
		health.ProtocolVersion,
	)
}
