package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

// ToolSchema describes a single tool's interface for AI agent discovery.
type ToolSchema struct {
	Name        string          `json:"name"`
	Route       string          `json:"route"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`
}

// ToolParameter describes a parameter for a tool.
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// ToolsList queries the TD server for all registered tool schemas.
func ToolsList(c *client.Client, jsonOutput bool) error {
	resp, err := c.Call("/tools/list", map[string]interface{}{})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("server error: %s", resp.Message)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	var data struct {
		Tools []ToolSchema `json:"tools"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse tools response: %w", err)
	}

	fmt.Printf("Available tools (%d):\n\n", len(data.Tools))
	for _, t := range data.Tools {
		fmt.Printf("  %-20s %s\n", t.Name, t.Description)
		for _, p := range t.Parameters {
			req := ""
			if p.Required {
				req = " (required)"
			}
			fmt.Printf("    %-18s %-10s %s%s\n", p.Name, p.Type, p.Description, req)
		}
		fmt.Println()
	}

	return nil
}
