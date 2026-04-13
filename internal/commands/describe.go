package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0dot77/td-cli/internal/client"
)

type describeResult struct {
	Path        string              `json:"path"`
	NodeCount   int                 `json:"nodeCount"`
	Families    map[string]int      `json:"families"`
	Nodes       []describeNode      `json:"nodes"`
	Connections []describeEdge      `json:"connections"`
	DataFlow    []string            `json:"dataFlow"`
}

type describeNode struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Family    string            `json:"family"`
	KeyParams map[string]string `json:"keyParams"`
}

type describeEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	FromIndex int    `json:"fromIndex"`
}

// Describe generates an AI-friendly description of a TD network.
func Describe(c *client.Client, path string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path": path,
	}

	resp, err := c.Call("/network/describe", payload)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	var result describeResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Header
	fmt.Printf("Network: %s (%d nodes)\n\n", result.Path, result.NodeCount)

	// Family breakdown
	fmt.Println("Families:")
	for fam, count := range result.Families {
		fmt.Printf("  %-8s %d\n", fam, count)
	}

	// Data flow
	if len(result.DataFlow) > 0 {
		fmt.Println("\nData Flow:")
		for _, chain := range result.DataFlow {
			fmt.Printf("  %s\n", chain)
		}
	}

	// Nodes with modified parameters
	fmt.Println("\nNodes:")
	for _, n := range result.Nodes {
		paramStr := ""
		if len(n.KeyParams) > 0 {
			parts := make([]string, 0, len(n.KeyParams))
			for k, v := range n.KeyParams {
				if len(v) > 30 {
					v = v[:30] + "..."
				}
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
			paramStr = " [" + strings.Join(parts, ", ") + "]"
		}
		fmt.Printf("  %-6s %-25s (%s)%s\n", n.Family, n.Name, n.Type, paramStr)
	}

	return nil
}
