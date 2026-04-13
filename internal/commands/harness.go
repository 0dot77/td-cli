package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/0dot77/td-cli/internal/client"
	"github.com/0dot77/td-cli/internal/protocol"
)

const (
	harnessCapabilitiesRoute = "/harness/capabilities"
	harnessObserveRoute      = "/harness/observe"
	harnessVerifyRoute       = "/harness/verify"
	harnessApplyRoute        = "/harness/apply"
	harnessRollbackRoute     = "/harness/rollback"
	harnessHistoryRoute      = "/harness/history"
)

func HarnessCapabilities(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessCapabilitiesRoute, payload, jsonOutput, renderHarnessCapabilities)
}

func HarnessObserve(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessObserveRoute, payload, jsonOutput, renderHarnessObserve)
}

func HarnessVerify(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessVerifyRoute, payload, jsonOutput, renderHarnessVerify)
}

func HarnessApply(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessApplyRoute, payload, jsonOutput, renderHarnessApply)
}

func HarnessRollback(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessRollbackRoute, payload, jsonOutput, renderHarnessRollback)
}

func HarnessHistory(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessHistoryRoute, payload, jsonOutput, renderHarnessHistory)
}

func harnessCall(c *client.Client, endpoint string, payload map[string]interface{}, jsonOutput bool, render func(*protocol.Response) error) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}

	resp, err := c.Call(endpoint, payload)
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
	return render(resp)
}

func renderHarnessCapabilities(resp *protocol.Response) error {
	var data protocol.HarnessCapabilitiesData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	project := data.Runtime.ProjectName
	if project == "" {
		project = "active session"
	}
	fmt.Printf("Harness capabilities: %s\n", project)
	if data.Runtime.TDVersion != "" {
		fmt.Printf("  TD: %s", data.Runtime.TDVersion)
		if data.Runtime.TDBuild != "" {
			fmt.Printf(" (build %s)", data.Runtime.TDBuild)
		}
		fmt.Println()
	}
	if connector := renderConnector(data.Connector.Name, data.Connector.Version); connector != "" {
		fmt.Printf("  Connector: %s\n", connector)
	}
	if data.Connector.ProtocolVersion != 0 {
		fmt.Printf("  Protocol: v%d\n", data.Connector.ProtocolVersion)
	}
	if data.Runtime.HarnessRoot != "" {
		fmt.Printf("  Harness Root: %s\n", data.Runtime.HarnessRoot)
	}

	printStringSection("Routes", sortedCopy(data.Tools.Routes))
	if len(data.Support.Families) > 0 {
		fmt.Println("Families:")
		families := make([]string, 0, len(data.Support.Families))
		for family := range data.Support.Families {
			families = append(families, family)
		}
		sort.Strings(families)
		for _, family := range families {
			fmt.Printf("  %s: %s\n", family, strings.Join(data.Support.Families[family], ", "))
		}
	}

	features := make([]string, 0, 4)
	if data.Support.Observe {
		features = append(features, "observe")
	}
	if data.Support.Verify {
		features = append(features, "verify")
	}
	if data.Support.Rollback {
		features = append(features, "rollback")
	}
	if data.Support.History {
		features = append(features, "history")
	}
	if len(data.Support.BatchRoutes) > 0 {
		features = append(features, "batch:"+strings.Join(data.Support.BatchRoutes, ","))
	}
	printStringSection("Features", features)
	return nil
}

func renderHarnessObserve(resp *protocol.Response) error {
	var data protocol.HarnessObserveData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("Harness observe")
	if data.Path != "" {
		fmt.Printf("  Scope: %s\n", data.Path)
	}
	fmt.Printf("  Graph: %d nodes, %d connections\n", data.Graph.NodeCount, data.Graph.ConnectionCount)
	printStringSection("Data Flow", data.Graph.DataFlow)

	if len(data.Outputs) > 0 {
		fmt.Println("Outputs:")
		for _, output := range data.Outputs {
			fmt.Printf("  %s (%s)\n", stringField(output, "path"), stringField(output, "type"))
		}
	}

	if issues := intField(data.Issues, "issueCount"); issues > 0 {
		fmt.Printf("  Issues: %d\n", issues)
	}
	if len(data.RecentActivity) > 0 {
		fmt.Println("Recent Activity:")
		for _, event := range data.RecentActivity {
			fmt.Printf("  %s  %s  %s\n",
				formatHarnessTimestamp(numberField(event, "timestamp")),
				stringField(event, "route"),
				stringField(event, "message"),
			)
		}
	}
	return nil
}

func renderHarnessVerify(resp *protocol.Response) error {
	var data protocol.HarnessVerifyData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	status := "FAIL"
	if data.Passed {
		status = "PASS"
	}
	fmt.Printf("Harness verify: %s\n", status)
	fmt.Printf("  Path: %s\n", data.Path)
	fmt.Printf("  Assertions: %d/%d passed\n", data.PassedCount, data.AssertionCount)
	if len(data.Assertions) > 0 {
		fmt.Println("Checks:")
		for _, check := range data.Assertions {
			result := "FAIL"
			if boolField(check, "passed") {
				result = "PASS"
			}
			fmt.Printf("  [%s] %s\n", result, stringField(check, "kind"))
			if actual, ok := check["actual"]; ok && actual != nil {
				fmt.Printf("    actual=%v\n", actual)
			}
			if details, ok := check["details"].([]interface{}); ok && len(details) > 0 {
				for _, detail := range details {
					fmt.Printf("    %v\n", detail)
				}
			}
		}
	}
	return nil
}

func renderHarnessApply(resp *protocol.Response) error {
	var data protocol.HarnessApplyData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Printf("Harness apply: %s\n", data.Status)
	fmt.Printf("  Scope: %s\n", data.TargetPath)
	if data.RollbackID != "" {
		fmt.Printf("  Rollback: %s\n", data.RollbackID)
	}
	if data.RecordPath != "" {
		fmt.Printf("  Record: %s\n", data.RecordPath)
	}
	if len(data.Results) > 0 {
		fmt.Println("Operations:")
		for _, result := range data.Results {
			state := "FAIL"
			if boolField(result, "success") {
				state = "OK"
			}
			fmt.Printf("  [%s] %s\n", state, stringField(result, "route"))
			if msg := stringField(result, "message"); msg != "" {
				fmt.Printf("    %s\n", msg)
			}
		}
	}
	return nil
}

func renderHarnessRollback(resp *protocol.Response) error {
	var data map[string]interface{}
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("Harness rollback")
	if id := stringField(data, "rollbackId"); id != "" {
		fmt.Printf("  Rollback: %s\n", id)
	}
	if path := stringField(data, "recordPath"); path != "" {
		fmt.Printf("  Record: %s\n", path)
	}
	if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	return nil
}

func renderHarnessHistory(resp *protocol.Response) error {
	var data protocol.HarnessHistoryData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("Harness history")
	for _, entry := range data.Iterations {
		fmt.Printf("  %s  %s  %s\n",
			formatHarnessTimestamp(entry.CreatedAt),
			historyStatus(entry),
			entry.TargetPath,
		)
		if entry.Goal != "" {
			fmt.Printf("    goal=%s\n", entry.Goal)
		}
		if entry.RecordPath != "" {
			fmt.Printf("    record=%s\n", entry.RecordPath)
		}
	}
	return nil
}

func decodeHarnessData(raw json.RawMessage, target interface{}) error {
	if raw == nil {
		return fmt.Errorf("missing harness response data")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("failed to parse harness response: %w", err)
	}
	return nil
}

func renderConnector(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" && version == "" {
		return ""
	}
	if name == "" {
		return version
	}
	if version == "" {
		return name
	}
	return fmt.Sprintf("%s v%s", name, version)
}

func sortedCopy(items []string) []string {
	cloned := append([]string(nil), items...)
	sort.Strings(cloned)
	return cloned
}

func printStringSection(title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		fmt.Printf("  %s\n", item)
	}
}

func stringField(data map[string]interface{}, key string) string {
	value, _ := data[key]
	text, _ := value.(string)
	return text
}

func boolField(data map[string]interface{}, key string) bool {
	value, _ := data[key]
	flag, _ := value.(bool)
	return flag
}

func numberField(data map[string]interface{}, key string) float64 {
	value, _ := data[key]
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

func intField(data map[string]interface{}, key string) int {
	return int(numberField(data, key))
}

func historyStatus(entry protocol.HarnessHistoryEntry) string {
	status := strings.TrimSpace(entry.Status)
	if status == "" {
		return "UNKNOWN"
	}
	return strings.ToUpper(strings.ReplaceAll(status, "-", "_"))
}

func formatHarnessTimestamp(ts float64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(int64(ts), 0).Format(time.RFC3339)
}
