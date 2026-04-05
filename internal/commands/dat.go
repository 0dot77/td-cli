package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/td-cli/td-cli/internal/client"
)

type datReadResult struct {
	Content  *string    `json:"content"`
	Table    [][]string `json:"table"`
	NumRows  int        `json:"numRows"`
	NumCols  int        `json:"numCols"`
	IsTable  bool       `json:"isTable"`
}

// DatRead reads DAT content.
func DatRead(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/dat/read", map[string]string{"path": path})
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

	var result datReadResult
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &result)
	}

	if result.IsTable && result.Table != nil {
		for _, row := range result.Table {
			fmt.Println(strings.Join(row, "\t"))
		}
	} else if result.Content != nil {
		fmt.Print(*result.Content)
	}

	return nil
}

// DatWrite writes content to a DAT.
func DatWrite(c *client.Client, path, content, filePath string, jsonOutput bool) error {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}
		content = string(data)
	}

	if content == "" {
		return fmt.Errorf("no content provided")
	}

	payload := map[string]interface{}{
		"path":    path,
		"content": content,
	}

	resp, err := c.Call("/dat/write", payload)
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
