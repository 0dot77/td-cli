package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0dot77/td-cli/internal/client"
)

type backupRecord struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	CreatedAt   float64 `json:"createdAt"`
	Project     string  `json:"projectName"`
	ProjectPath string  `json:"projectPath"`
	Path        string  `json:"path"`
	TargetPath  string  `json:"targetPath"`
}

// BackupList lists recent backup artifacts.
func BackupList(c *client.Client, limit int, jsonOutput bool) error {
	payload := map[string]interface{}{}
	if limit > 0 {
		payload["limit"] = limit
	}

	resp, err := c.Call("/backup/list", payload)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	var data struct {
		Backups []backupRecord `json:"backups"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse backup list: %w", err)
	}

	if len(data.Backups) == 0 {
		fmt.Println("No backups found")
		return nil
	}

	fmt.Printf("Backups (%d):\n", len(data.Backups))
	for _, backup := range data.Backups {
		createdAt := time.Unix(int64(backup.CreatedAt), 0).Format(time.RFC3339)
		fmt.Printf("  %-28s %-16s %-20s %s\n", backup.ID, backup.Kind, backup.TargetPath, createdAt)
	}
	return nil
}

// BackupRestore restores a backup artifact by id.
func BackupRestore(c *client.Client, id string, jsonOutput bool) error {
	resp, err := c.Call("/backup/restore", map[string]string{"id": id})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	var data map[string]interface{}
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	fmt.Println(resp.Message)
	if id, ok := data["backupId"]; ok {
		fmt.Printf("  Backup: %v\n", id)
	}
	if path, ok := data["restoredPath"]; ok {
		fmt.Printf("  Restored Path: %v\n", path)
	}
	if kind, ok := data["restoredKind"]; ok {
		fmt.Printf("  Kind: %v\n", kind)
	}
	if warnings, ok := data["warningCount"]; ok {
		fmt.Printf("  Warnings: %v\n", warnings)
	}
	return nil
}
