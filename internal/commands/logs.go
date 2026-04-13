package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0dot77/td-cli/internal/client"
)

type logEvent struct {
	Timestamp    float64 `json:"timestamp"`
	RequestID    string  `json:"requestId"`
	Route        string  `json:"route"`
	ProjectName  string  `json:"projectName"`
	ProjectPath  string  `json:"projectPath"`
	TargetPath   string  `json:"targetPath"`
	Success      bool    `json:"success"`
	DurationMS   float64 `json:"durationMs"`
	BackupID     string  `json:"backupId"`
	WarningCount int     `json:"warningCount"`
	Message      string  `json:"message"`
	Error        string  `json:"error"`
}

// LogsList lists recent audit log events in reverse chronological order.
func LogsList(c *client.Client, limit int, jsonOutput bool) error {
	return logsRequest(c, "/logs/list", limit, jsonOutput)
}

// LogsTail reads recent audit log events in chronological order.
func LogsTail(c *client.Client, limit int, jsonOutput bool) error {
	return logsRequest(c, "/logs/tail", limit, jsonOutput)
}

func logsRequest(c *client.Client, endpoint string, limit int, jsonOutput bool) error {
	payload := map[string]interface{}{}
	if limit > 0 {
		payload["limit"] = limit
	}

	resp, err := c.Call(endpoint, payload)
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
		Events []logEvent `json:"events"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse logs response: %w", err)
	}

	if len(data.Events) == 0 {
		fmt.Println("No log events found")
		return nil
	}

	for _, event := range data.Events {
		ts := time.Unix(int64(event.Timestamp), 0).Format(time.RFC3339)
		status := "ok"
		if !event.Success {
			status = "err"
		}
		fmt.Printf("%s %-4s %-18s %-24s %.3fms", ts, status, event.Route, event.TargetPath, event.DurationMS)
		if event.BackupID != "" {
			fmt.Printf(" backup=%s", event.BackupID)
		}
		if event.WarningCount > 0 {
			fmt.Printf(" warnings=%d", event.WarningCount)
		}
		fmt.Println()
		if event.Error != "" {
			fmt.Printf("  error: %s\n", event.Error)
		}
	}
	return nil
}
