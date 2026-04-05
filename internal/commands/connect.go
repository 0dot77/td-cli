package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

// Connect wires two operators together.
func Connect(c *client.Client, src, dst string, srcIndex, dstIndex int, jsonOutput bool) error {
	payload := map[string]interface{}{
		"src":      src,
		"dst":      dst,
		"srcIndex": srcIndex,
		"dstIndex": dstIndex,
	}

	resp, err := c.Call("/connect", payload)
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

// Disconnect removes a wire between two operators.
func Disconnect(c *client.Client, src, dst string, jsonOutput bool) error {
	payload := map[string]string{
		"src": src,
		"dst": dst,
	}

	resp, err := c.Call("/disconnect", payload)
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
