package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/protocol"
)

type contextOpEntry struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Family string `json:"family"`
}

func Context(c *client.Client, depth int, jsonOutput bool) error {
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
		"path": "/", "depth": depth,
	})
	if err != nil {
		return fmt.Errorf("failed to list operators: %w", err)
	}

	networkSummary := make(map[string]int)
	opCount := 0
	var operators []contextOpEntry
	if opsResp.Success && opsResp.Data != nil {
		var opsData struct {
			Operators []contextOpEntry `json:"operators"`
		}
		if err := json.Unmarshal(opsResp.Data, &opsData); err == nil {
			operators = opsData.Operators
			opCount = len(operators)
			for _, op := range operators {
				networkSummary[op.Family]++
			}
		}
	}

	// Fetch recent logs
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

	// Fetch harness history (best-effort, ignore errors)
	var harnessEntries []protocol.HarnessHistoryEntry
	harnessResp, _ := c.Call("/harness/history", map[string]interface{}{"limit": 3})
	if harnessResp != nil && harnessResp.Success && harnessResp.Data != nil {
		var histData protocol.HarnessHistoryData
		if err := json.Unmarshal(harnessResp.Data, &histData); err == nil {
			harnessEntries = histData.Iterations
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
			"operators":      operators,
		}
		if len(harnessEntries) > 0 {
			result["harnessHistory"] = harnessEntries
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	// === Header ===
	fmt.Println("=== TouchDesigner Context ===")
	fmt.Printf("Project: %s\n", health.Project)
	fmt.Printf("Port: %d | TD %s (build %s)\n", c.Port(), health.TDVersion, health.TDBuild)

	// === Operator summary ===
	if opCount > 0 {
		families := []string{"TOP", "CHOP", "SOP", "POP", "DAT", "COMP", "MAT"}
		parts := []string{}
		for _, f := range families {
			if v, ok := networkSummary[f]; ok {
				parts = append(parts, fmt.Sprintf("%d %s", v, f))
			}
		}
		// Include any families not in the standard list
		for f, v := range networkSummary {
			found := false
			for _, std := range families {
				if f == std {
					found = true
					break
				}
			}
			if !found {
				parts = append(parts, fmt.Sprintf("%d %s", v, f))
			}
		}
		fmt.Printf("Operators: %d total (%s)\n", opCount, strings.Join(parts, ", "))
	} else {
		fmt.Println("Operators: 0")
	}

	// === Network tree (depths 1-2) ===
	if len(operators) > 0 {
		fmt.Println()
		fmt.Println("Network tree:")
		printNetworkTree(operators)
	}

	// === Recent activity ===
	if len(recentLogs) > 0 {
		fmt.Println()
		fmt.Println("Recent activity:")
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

	// === Harness status ===
	if len(harnessEntries) > 0 {
		fmt.Println()
		fmt.Printf("Harness: %d recent iteration(s)\n", len(harnessEntries))
		for _, entry := range harnessEntries {
			ts := ""
			if entry.CreatedAt > 0 {
				elapsed := time.Since(time.Unix(int64(entry.CreatedAt), 0))
				ts = formatDuration(elapsed)
			}
			status := strings.ToLower(strings.ReplaceAll(entry.Status, "-", "_"))
			target := entry.TargetPath
			goal := ""
			if entry.Goal != "" {
				goal = fmt.Sprintf(" %q", entry.Goal)
			}
			if ts != "" {
				fmt.Printf("  [%s ago] %s", ts, status)
			} else {
				fmt.Printf("  %s", status)
			}
			if target != "" {
				fmt.Printf(" -> %s", target)
			}
			if goal != "" {
				fmt.Printf(" %s", goal)
			}
			fmt.Println()
		}
	}

	return nil
}

// printNetworkTree renders a concise operator tree for depths 1-2.
// It groups operators by parent path and shows them indented.
func printNetworkTree(operators []contextOpEntry) {
	// Separate into depth levels based on slash count
	// / = depth 0, /project1 = depth 1, /project1/foo = depth 2
	type treeNode struct {
		path   string
		opType string
		depth  int
	}

	var nodes []treeNode
	for _, op := range operators {
		p := strings.TrimSuffix(op.Path, "/")
		if p == "" {
			continue // skip root
		}
		d := strings.Count(p, "/")
		nodes = append(nodes, treeNode{path: p, opType: op.Type, depth: d})
	}

	// Sort by path for consistent output
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].path < nodes[j].path
	})

	// Only show depths 1 and 2
	for _, n := range nodes {
		if n.depth < 1 || n.depth > 2 {
			continue
		}
		name := n.path[strings.LastIndex(n.path, "/")+1:]
		indent := strings.Repeat("  ", n.depth)
		typeSuffix := ""
		if n.opType != "" {
			typeSuffix = fmt.Sprintf(" (%s)", n.opType)
		}
		fmt.Printf("%s%s%s\n", indent, name, typeSuffix)
	}
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
