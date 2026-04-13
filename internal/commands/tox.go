package commands

import (
	"encoding/json"
	"fmt"

	"github.com/0dot77/td-cli/internal/client"
)

// ToxExport exports a COMP as a .tox file.
func ToxExport(c *client.Client, compPath, outputPath string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path":   compPath,
		"output": outputPath,
	}

	resp, err := c.Call("/tox/export", payload)
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

	var result struct {
		Output string `json:"output"`
		Size   int64  `json:"size"`
	}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}

	fmt.Printf("Exported to %s (%d bytes)\n", result.Output, result.Size)
	return nil
}

// ToxImport imports a .tox file into a parent COMP.
func ToxImport(c *client.Client, toxPath, parentPath, name string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"toxPath":    toxPath,
		"parentPath": parentPath,
	}
	if name != "" {
		payload["name"] = name
	}

	resp, err := c.Call("/tox/import", payload)
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
