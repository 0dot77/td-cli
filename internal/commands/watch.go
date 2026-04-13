package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/0dot77/td-cli/internal/client"
)

type monitorResult struct {
	FPS       float64       `json:"fps"`
	ActualFPS float64       `json:"actualFps"`
	Frame     int           `json:"frame"`
	Seconds   float64       `json:"seconds"`
	RealTime  bool          `json:"realTime"`
	Children  []childMetric `json:"children"`
}

type childMetric struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Family      string  `json:"family"`
	CookTime    float64 `json:"cookTime,omitempty"`
	CPUCookTime float64 `json:"cpuCookTime,omitempty"`
	Errors      string  `json:"errors,omitempty"`
	Warnings    string  `json:"warnings,omitempty"`
}

// Watch runs a continuous monitoring loop.
func Watch(c *client.Client, path string, interval time.Duration, jsonOutput bool) error {
	// Handle Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	if !jsonOutput {
		fmt.Printf("Monitoring %s (interval: %s, Ctrl+C to stop)\n\n", path, interval)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			fmt.Println("\nStopped")
			return nil
		case <-ticker.C:
			if err := watchOnce(c, path, jsonOutput); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
			}
		}
	}
}

func watchOnce(c *client.Client, path string, jsonOutput bool) error {
	payload := map[string]interface{}{
		"path": path,
	}

	resp, err := c.Call("/monitor", payload)
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

	var result monitorResult
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Clear screen (ANSI)
	fmt.Print("\033[2J\033[H")

	// Header
	rt := "ON"
	if !result.RealTime {
		rt = "OFF"
	}
	fmt.Printf("td-cli watch | FPS: %.0f/%.0f | Frame: %d | Time: %.1fs | RealTime: %s\n",
		result.ActualFPS, result.FPS, result.Frame, result.Seconds, rt)
	fmt.Println(strings.Repeat("-", 70))

	// Children table
	if len(result.Children) == 0 {
		fmt.Println("  (no operators)")
		return nil
	}

	fmt.Printf("  %-6s %-20s %-15s %8s  %s\n", "Family", "Name", "Type", "Cook(ms)", "Status")
	fmt.Println(strings.Repeat("-", 70))

	for _, ch := range result.Children {
		status := ""
		if ch.Errors != "" {
			status = "ERR: " + ch.Errors
		} else if ch.Warnings != "" {
			status = "WARN: " + ch.Warnings
		}
		if len(status) > 30 {
			status = status[:30] + "..."
		}

		cookStr := ""
		if ch.CookTime > 0 {
			cookStr = fmt.Sprintf("%.3f", ch.CookTime)
		}

		fmt.Printf("  %-6s %-20s %-15s %8s  %s\n",
			ch.Family, ch.Name, ch.Type, cookStr, status)
	}

	return nil
}
