package commands

import (
	"encoding/json"
	"fmt"

	"github.com/0dot77/td-cli/internal/client"
)

type paramInfo struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Default string `json:"default"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
}

type parGetResult struct {
	Parameters []paramInfo `json:"parameters"`
}

// ParGet gets parameters of an operator.
func ParGet(c *client.Client, path string, names []string, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if len(names) > 0 {
		payload["names"] = names
	}

	resp, err := c.Call("/par/get", payload)
	if err != nil {
		return err
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	var result parGetResult
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	for _, p := range result.Parameters {
		fmt.Printf("  %-25s = %-15s (default: %s)\n", p.Name, p.Value, p.Default)
	}
	return nil
}

// ParSet sets parameters on an operator.
func ParSet(c *client.Client, path string, params map[string]interface{}, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path":   path,
		"params": params,
	}

	resp, err := c.Call("/par/set", payload)
	if err != nil {
		return err
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	fmt.Println(resp.Message)
	return nil
}
