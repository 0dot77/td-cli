package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

type opInfo struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Family string `json:"family"`
	NodeX  int    `json:"nodeX"`
	NodeY  int    `json:"nodeY"`
}

type opsListResult struct {
	Operators []opInfo `json:"operators"`
}

// OpsList lists operators at a path.
func OpsList(c *client.Client, path string, depth int, family string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path":  path,
		"depth": depth,
	}
	if family != "" {
		payload["family"] = family
	}

	resp, err := c.Call("/ops/list", payload)
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

	var result opsListResult
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &result)
	}

	if len(result.Operators) == 0 {
		fmt.Println("No operators found")
		return nil
	}

	for _, op := range result.Operators {
		fmt.Printf("  %-6s %-20s %s\n", op.Family, op.Type, op.Path)
	}
	fmt.Printf("\n%d operator(s)\n", len(result.Operators))
	return nil
}

// OpsCreate creates a new operator.
func OpsCreate(c *client.Client, opType, parent, name string, nodeX, nodeY int, jsonOutput bool) error {
	payload := map[string]interface{}{
		"type":   opType,
		"parent": parent,
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

	resp, err := c.Call("/ops/create", payload)
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

	var info opInfo
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &info)
	}

	fmt.Printf("Created %s at %s\n", info.Type, info.Path)
	return nil
}

// OpsDelete deletes an operator.
func OpsDelete(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/ops/delete", map[string]string{"path": path})
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

// OpsInfo gets detailed info about an operator.
func OpsInfo(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/ops/info", map[string]string{"path": path})
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

	// Parse and display structured info
	var info map[string]interface{}
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &info)
	}

	fmt.Printf("Operator: %s\n", info["path"])
	fmt.Printf("  Type:    %s\n", info["type"])
	fmt.Printf("  Family:  %s\n", info["family"])
	fmt.Printf("  Comment: %s\n", info["comment"])

	if inputs, ok := info["inputs"].([]interface{}); ok && len(inputs) > 0 {
		fmt.Println("  Inputs:")
		for _, in := range inputs {
			m := in.(map[string]interface{})
			fmt.Printf("    [%v] %s\n", m["index"], m["path"])
		}
	}

	if outputs, ok := info["outputs"].([]interface{}); ok && len(outputs) > 0 {
		fmt.Println("  Outputs:")
		for _, out := range outputs {
			m := out.(map[string]interface{})
			fmt.Printf("    [%v] %s\n", m["index"], m["path"])
		}
	}

	if params, ok := info["parameters"].([]interface{}); ok {
		fmt.Printf("  Parameters: (%d)\n", len(params))
		for _, p := range params {
			m := p.(map[string]interface{})
			fmt.Printf("    %-20s = %s\n", m["name"], m["value"])
		}
	}

	return nil
}
