package commands

import (
	"encoding/json"
	"fmt"

	"github.com/0dot77/td-cli/internal/client"
)

// ProjectInfo displays project metadata.
func ProjectInfo(c *client.Client, jsonOutput bool) error {
	resp, err := c.Call("/project/info", map[string]string{})
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

	var info map[string]interface{}
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &info); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	fmt.Println("Project Info:")
	fmt.Printf("  Name:      %v\n", info["name"])
	fmt.Printf("  Folder:    %v\n", info["folder"])
	fmt.Printf("  TD:        %v (build %v)\n", info["tdVersion"], info["tdBuild"])
	fmt.Printf("  FPS:       %v\n", info["fps"])
	fmt.Printf("  Real Time: %v\n", info["realTime"])
	fmt.Printf("  Frame:     %v\n", info["timelineFrame"])
	return nil
}

// ProjectSave saves the project.
func ProjectSave(c *client.Client, path string, jsonOutput bool) error {
	payload := map[string]interface{}{}
	if path != "" {
		payload["path"] = path
	}

	resp, err := c.Call("/project/save", payload)
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
