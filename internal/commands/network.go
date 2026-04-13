package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/0dot77/td-cli/internal/client"
)

// NetworkExport exports the network structure as a JSON snapshot.
func NetworkExport(c *client.Client, path string, outputFile string, depth int, includeDefaults bool, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path":            path,
		"depth":           depth,
		"includeDefaults": includeDefaults,
	}

	resp, err := c.Call("/network/export", payload)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	if outputFile != "" {
		out, _ := json.MarshalIndent(json.RawMessage(resp.Data), "", "  ")
		if err := os.WriteFile(outputFile, out, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		var meta struct {
			NodeCount int `json:"nodeCount"`
		}
		if err := json.Unmarshal(resp.Data, &meta); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
		fmt.Printf("Exported %d nodes to %s\n", meta.NodeCount, outputFile)
		return nil
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	// Pretty print summary
	var data struct {
		NodeCount int    `json:"nodeCount"`
		RootPath  string `json:"rootPath"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}
	fmt.Printf("Exported %d nodes from %s\n", data.NodeCount, data.RootPath)
	fmt.Println("Use -o <file> to save to a file")
	return nil
}

// NetworkImport recreates a network from a JSON snapshot file.
func NetworkImport(c *client.Client, filePath string, targetPath string, jsonOutput bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var snapshot json.RawMessage
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	payload := map[string]interface{}{
		"snapshot":   json.RawMessage(data),
		"targetPath": targetPath,
	}

	resp, err := c.Call("/network/import", payload)
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
