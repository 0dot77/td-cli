package commands

import (
	"encoding/json"
	"fmt"

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
		json.Unmarshal(resp.Data, &health)
	}

	fmt.Println("Connected to TouchDesigner")
	fmt.Printf("  Project:    %s\n", health.Project)
	fmt.Printf("  TD Version: %s (build %s)\n", health.TDVersion, health.TDBuild)
	fmt.Printf("  Server:     %s v%s\n", "td-cli", health.Version)
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
		fmt.Printf("  %-20s  port:%-5d  pid:%-6d  %s\n",
			inst.ProjectName, inst.Port, inst.PID, inst.ProjectPath)
	}
}
