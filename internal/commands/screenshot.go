package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/0dot77/td-cli/internal/client"
)

type screenshotResult struct {
	Image  string `json:"image"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Screenshot captures a TOP output as PNG.
func Screenshot(c *client.Client, path, outputFile string, jsonOutput bool) error {
	payload := map[string]string{}
	if path != "" {
		payload["path"] = path
	}

	resp, err := c.Call("/screenshot", payload)
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

	var result screenshotResult
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	if outputFile != "" {
		data, err := base64.StdEncoding.DecodeString(result.Image)
		if err != nil {
			return fmt.Errorf("failed to decode image: %w", err)
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Screenshot saved to %s (%dx%d)\n", outputFile, result.Width, result.Height)
	} else {
		fmt.Print(result.Image)
	}

	return nil
}
