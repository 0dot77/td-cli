package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/td-cli/td-cli/internal/client"
)

func TableRows(c *client.Client, path string, start, end int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if start > 0 {
		payload["start"] = start
	}
	if end >= 0 {
		payload["end"] = end
	}
	resp, err := c.Call("/table/rows", payload)
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
		NumRows int        `json:"numRows"`
		NumCols int        `json:"numCols"`
		Rows    [][]string `json:"rows"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Table: %d rows x %d cols\n", data.NumRows, data.NumCols)
	for i, row := range data.Rows {
		fmt.Printf("  [%d] %v\n", i+start, row)
	}
	return nil
}

func TableCell(c *client.Client, path string, row, col int, value string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path": path, "row": row, "col": col,
	}
	if value != "" {
		payload["value"] = value
	}
	resp, err := c.Call("/table/cell", payload)
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
		Row   int    `json:"row"`
		Col   int    `json:"col"`
		Value string `json:"value"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("[%d,%d] = %s\n", data.Row, data.Col, data.Value)
	return nil
}

func TableAppend(c *client.Client, path, mode string, values []string, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path, "mode": mode}
	if len(values) > 0 {
		payload["values"] = values
	}
	resp, err := c.Call("/table/append", payload)
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
	if mode == "col" {
		var data struct {
			NumCols int `json:"numCols"`
		}
		json.Unmarshal(resp.Data, &data)
		fmt.Printf("Col appended (%d cols)\n", data.NumCols)
	} else {
		var data struct {
			NumRows int `json:"numRows"`
		}
		json.Unmarshal(resp.Data, &data)
		fmt.Printf("Row appended (%d rows)\n", data.NumRows)
	}
	return nil
}

func TableDelete(c *client.Client, path, mode string, index int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path, "mode": mode}
	if index >= 0 {
		payload["index"] = index
	}
	resp, err := c.Call("/table/delete", payload)
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
	fmt.Printf("Deleted %s\n", mode)
	return nil
}

func ParseTableCoords(args []string) (row, col int, value string) {
	row, col = 0, 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--value":
			if i+1 < len(args) {
				value = args[i+1]
				i++
			}
		default:
			n, err := strconv.Atoi(args[i])
			if err == nil {
				if row == 0 {
					row = n
				} else if col == 0 {
					col = n
				}
			}
		}
	}
	return
}
