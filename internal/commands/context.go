package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/protocol"
)

func Context(c *client.Client, jsonOutput bool) error {
	resp, err := c.Health()
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("server error: %s", resp.Message)
	}

	var health protocol.HealthData
	if err := json.Unmarshal(resp.Data, &health); err != nil {
		return fmt.Errorf("failed to parse health data: %w", err)
	}

	opsResp, err := c.Call("/ops/list", map[string]interface{}{
		"path": "/", "depth": 2,
	})
	if err != nil {
		return fmt.Errorf("failed to list operators: %w", err)
	}

	networkSummary := make(map[string]int)
	opCount := 0
	if opsResp.Success && opsResp.Data != nil {
		var opsData struct {
			Operators []struct {
				Family string `json:"family"`
			} `json:"operators"`
		}
		if err := json.Unmarshal(opsResp.Data, &opsData); err == nil {
			opCount = len(opsData.Operators)
			for _, op := range opsData.Operators {
				networkSummary[op.Family]++
			}
		}
	}

	logResp, _ := c.Call("/logs/list", map[string]interface{}{"limit": 5})
	var recentLogs []interface{}
	if logResp != nil && logResp.Success && logResp.Data != nil {
		var logData struct {
			Events []interface{} `json:"events"`
		}
		if err := json.Unmarshal(logResp.Data, &logData); err == nil {
			if len(logData.Events) > 5 {
				recentLogs = logData.Events[:5]
			} else {
				recentLogs = logData.Events
			}
		}
	}

	if jsonOutput {
		result := map[string]interface{}{
			"connected":      true,
			"project":        health.Project,
			"port":           c.Port(),
			"tdVersion":      health.TDVersion,
			"tdBuild":        health.TDBuild,
			"totalOperators": opCount,
			"byFamily":       networkSummary,
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Println("TouchDesigner: Connected")
	fmt.Printf("  Project: %s\n", health.Project)
	fmt.Printf("  Port: %d | TD: %s (build %s)\n", c.Port(), health.TDVersion, health.TDBuild)

	if opCount > 0 {
		fmt.Printf("\nNetwork (/):\n  %d operators\n", opCount)
		families := []string{"TOP", "CHOP", "SOP", "POP", "DAT", "COMP", "MAT"}
		parts := []string{}
		for _, f := range families {
			if v, ok := networkSummary[f]; ok {
				parts = append(parts, fmt.Sprintf("%d %s", v, f))
			}
		}
		if len(parts) > 0 {
			fmt.Printf("  %s\n", joinWith(parts, "  "))
		}
	}

	if len(recentLogs) > 0 {
		fmt.Printf("\nRecent activity:\n")
		for _, ev := range recentLogs {
			if m, ok := ev.(map[string]interface{}); ok {
				action := "?"
				if a, ok := m["action"].(string); ok {
					action = a
				}
				timestamp := ""
				if ts, ok := m["timestamp"].(float64); ok {
					elapsed := time.Since(time.Unix(int64(ts), 0))
					timestamp = formatDuration(elapsed)
				}
				target := ""
				if t, ok := m["target"].(string); ok {
					target = t
				}
				fmt.Printf("  [%s ago] %s %s\n", timestamp, action, target)
			}
		}
	}

	return nil
}

func joinWith(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
