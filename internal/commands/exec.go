package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/td-cli/td-cli/internal/client"
)

type execResult struct {
	Result string `json:"result"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// Exec executes Python code in TouchDesigner.
func Exec(c *client.Client, code string, filePath string, jsonOutput bool) error {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}
		code = string(data)
	}

	if code == "" {
		return fmt.Errorf("no code provided (use td-cli exec \"<code>\" or td-cli exec -f <file>)")
	}

	resp, err := c.Call("/exec", map[string]string{"code": code})
	if err != nil {
		return err
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("execution error: %s", resp.Message)
	}

	var result execResult
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &result)
	}

	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Result != "" {
		fmt.Println(result.Result)
	}
	if result.Stderr != "" {
		fmt.Fprintf(os.Stderr, "%s", result.Stderr)
	}

	return nil
}
