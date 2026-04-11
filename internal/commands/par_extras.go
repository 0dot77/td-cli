package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func ParPulse(c *client.Client, path, name string, jsonOutput bool) error {
	resp, err := c.Call("/par/pulse", map[string]interface{}{
		"path": path, "name": name,
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
	fmt.Printf("Pulsed %s.%s\n", path, name)
	return nil
}

func ParReset(c *client.Client, path string, names []string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path": path,
	}
	if len(names) > 0 {
		payload["names"] = names
	}
	resp, err := c.Call("/par/reset", payload)
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
	fmt.Printf("Reset parameters on %s\n", path)
	return nil
}

func ParExpr(c *client.Client, path, name, expression string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path": path, "name": name,
	}
	if expression != "" {
		payload["expression"] = expression
	}
	resp, err := c.Call("/par/expr", payload)
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
		Expression string `json:"expression"`
		Value      string `json:"value"`
	}
	json.Unmarshal(resp.Data, &data)
	if expression != "" {
		fmt.Printf("Set expression: %s = %s (value: %s)\n", name, data.Expression, data.Value)
	} else {
		fmt.Printf("%s.%s expr: %s (value: %s)\n", path, name, data.Expression, data.Value)
	}
	return nil
}

func ParExport(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/par/export", map[string]interface{}{
		"path": path,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	out, _ := json.MarshalIndent(resp.Data, "", "  ")
	fmt.Println(string(out))
	return nil
}

func ParImport(c *client.Client, path string, params []interface{}, jsonOutput bool) error {
	resp, err := c.Call("/par/import", map[string]interface{}{
		"path": path, "params": params,
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
		Applied int `json:"applied"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Imported %d parameters to %s\n", data.Applied, path)
	return nil
}
