package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func BatchExec(c *client.Client, commands []map[string]interface{}, jsonOutput bool) error {
	resp, err := c.Call("/batch/exec", map[string]interface{}{"commands": commands})
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
		Total    int     `json:"total"`
		Success  int     `json:"success"`
		Failed   int     `json:"failed"`
		Duration float64 `json:"duration"`
		Results  []struct {
			Route   string `json:"route"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		} `json:"results"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Batch: %d/%d succeeded (%.2fms)\n", data.Success, data.Total, data.Duration)
	for i, r := range data.Results {
		status := "OK"
		if !r.Success {
			status = "FAIL: " + r.Message
		}
		fmt.Printf("  [%d] %s — %s\n", i+1, r.Route, status)
	}
	return nil
}

func BatchParSet(c *client.Client, sets []map[string]interface{}, jsonOutput bool) error {
	resp, err := c.Call("/batch/parset", map[string]interface{}{"sets": sets})
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
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Batch par set: %d/%d succeeded\n", data.Success, data.Total)
	return nil
}
