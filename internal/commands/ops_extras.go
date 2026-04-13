package commands

import (
	"encoding/json"
	"fmt"

	"github.com/0dot77/td-cli/internal/client"
)

func OpsRename(c *client.Client, path, newName string, jsonOutput bool) error {
	resp, err := c.Call("/ops/rename", map[string]interface{}{
		"path": path, "name": newName,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	if jsonOutput {
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Printf("Renamed to %s\n", newName)
	return nil
}

func OpsCopy(c *client.Client, src, parent, name string, nodeX, nodeY int, jsonOutput bool) error {
	payload := map[string]interface{}{
		"src": src, "parent": parent,
	}
	if name != "" {
		payload["name"] = name
	}
	if nodeX >= 0 {
		payload["nodeX"] = nodeX
	}
	if nodeY >= 0 {
		payload["nodeY"] = nodeY
	}
	resp, err := c.Call("/ops/copy", payload)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	if jsonOutput {
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Printf("Copied %s\n", src)
	return nil
}

func OpsMove(c *client.Client, src, parent string, jsonOutput bool) error {
	resp, err := c.Call("/ops/move", map[string]interface{}{
		"src": src, "parent": parent,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	if jsonOutput {
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Printf("Moved %s to %s\n", src, parent)
	return nil
}

func OpsClone(c *client.Client, src, parent, name string, nodeX, nodeY int, jsonOutput bool) error {
	payload := map[string]interface{}{
		"src": src, "parent": parent,
	}
	if name != "" {
		payload["name"] = name
	}
	if nodeX >= 0 {
		payload["nodeX"] = nodeX
	}
	if nodeY >= 0 {
		payload["nodeY"] = nodeY
	}
	resp, err := c.Call("/ops/clone", payload)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	if jsonOutput {
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	fmt.Printf("Cloned %s\n", src)
	return nil
}

func OpsSearch(c *client.Client, parent, pattern, family string, depth int, jsonOutput bool) error {
	resp, err := c.Call("/ops/search", map[string]interface{}{
		"parent": parent, "pattern": pattern,
		"family": family, "depth": depth,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	if jsonOutput {
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
		return nil
	}
	var data struct {
		Operators []struct {
			Path   string `json:"path"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Family string `json:"family"`
		} `json:"operators"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}
	if len(data.Operators) == 0 {
		fmt.Println("No operators found")
		return nil
	}
	for _, op := range data.Operators {
		fmt.Printf("  %-6s %-25s %s\n", op.Family, op.Name, op.Path)
	}
	fmt.Printf("\n%d result(s)\n", len(data.Operators))
	return nil
}
