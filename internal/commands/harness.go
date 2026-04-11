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

const (
	harnessCapabilitiesRoute = "/harness/capabilities"
	harnessObserveRoute      = "/harness/observe"
	harnessVerifyRoute       = "/harness/verify"
	harnessApplyRoute        = "/harness/apply"
	harnessRollbackRoute     = "/harness/rollback"
	harnessHistoryRoute      = "/harness/history"
)

// HarnessCapabilities queries the harness capabilities surface.
func HarnessCapabilities(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessCapabilitiesRoute, payload, jsonOutput, renderHarnessCapabilities)
}

// HarnessObserve requests an agent-oriented observation snapshot.
func HarnessObserve(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessObserveRoute, payload, jsonOutput, renderHarnessObserve)
}

// HarnessVerify requests verification evidence from the harness.
func HarnessVerify(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessVerifyRoute, payload, jsonOutput, renderHarnessVerify)
}

// HarnessApply sends an apply/dry-run request to the harness.
func HarnessApply(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessApplyRoute, payload, jsonOutput, renderHarnessApply)
}

// HarnessRollback requests rollback of a prior harness change.
func HarnessRollback(c *client.Client, payload map[string]interface{}, jsonOutput bool) error {
	return harnessCall(c, harnessRollbackRoute, payload, jsonOutput, renderHarnessRollback)
}

// HarnessHistory requests recent harness activity.
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

	title := strings.TrimSpace(data.Project)
	if title == "" {
		title = "active session"
	}
	fmt.Printf("Harness capabilities: %s\n", title)

	if data.TDVersion != "" || data.TDBuild != "" {
		fmt.Printf("  TD: %s", data.TDVersion)
		if data.TDBuild != "" {
			fmt.Printf(" (build %s)", data.TDBuild)
		}
		fmt.Println()
	}

	if connector := renderConnector(data.ConnectorName, data.ConnectorVersion); connector != "" {
		fmt.Printf("  Connector: %s\n", connector)
	}

	if data.ProtocolVersion != 0 {
		fmt.Printf("  Protocol: v%d\n", data.ProtocolVersion)
	}

	printStringSection("Routes", sortedCopy(data.SupportedRoutes))
	printStringSection("Families", sortedCopy(data.SupportedFamilies))
	printStringSection("Features", sortedCopy(data.Features))
	printStringSection("Constraints", data.Constraints)
	printStringSection("Notes", data.Notes)
	printStringSection("Warnings", data.Warnings)

	if allStringSlicesEmpty(data.SupportedRoutes, data.SupportedFamilies, data.Features, data.Constraints, data.Notes, data.Warnings) && strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  %s\n", resp.Message)
	}

	return nil
}

func renderHarnessObserve(resp *protocol.Response) error {
	var data protocol.HarnessObserveData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("Harness observe")
	if data.Scope != "" {
		fmt.Printf("  Scope: %s\n", data.Scope)
	}
	if data.Goal != "" {
		fmt.Printf("  Goal: %s\n", data.Goal)
	}
	if data.Summary != "" {
		fmt.Printf("  Summary: %s\n", data.Summary)
	} else if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	if data.RequestID != "" {
		fmt.Printf("  Request: %s\n", data.RequestID)
	}
	if data.RollbackHandle != "" {
		fmt.Printf("  Rollback: %s\n", data.RollbackHandle)
	}
	if data.Checkpoint != nil {
		fmt.Printf("  Checkpoint: %s\n", checkpointLabel(*data.Checkpoint))
	}

	printTargets("Outputs", data.Outputs)
	printTargets("Viewers", data.Viewers)
	printChanges("Changes", data.Changes)
	printMetrics("Hotspots", data.Hotspots)
	printStringSection("Warnings", data.Warnings)
	printStringSection("Errors", data.Errors)
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
	if data.Scope != "" {
		fmt.Printf("  Scope: %s\n", data.Scope)
	}
	if data.Goal != "" {
		fmt.Printf("  Goal: %s\n", data.Goal)
	}
	if data.Summary != "" {
		fmt.Printf("  Summary: %s\n", data.Summary)
	} else if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	if data.Score != 0 {
		fmt.Printf("  Score: %.2f\n", data.Score)
	}
	if data.RequestID != "" {
		fmt.Printf("  Request: %s\n", data.RequestID)
	}
	if data.RollbackHandle != "" {
		fmt.Printf("  Rollback: %s\n", data.RollbackHandle)
	}

	printCheckResults(data.Checks)
	printEvidence(data.Evidence)
	printStringSection("Warnings", data.Warnings)
	printStringSection("Errors", data.Errors)
	return nil
}

func renderHarnessApply(resp *protocol.Response) error {
	var data protocol.HarnessApplyData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	status := "blocked"
	switch {
	case data.DryRun:
		status = "dry-run"
	case data.Applied:
		status = "applied"
	}
	fmt.Printf("Harness apply: %s\n", status)
	if data.Scope != "" {
		fmt.Printf("  Scope: %s\n", data.Scope)
	}
	if data.Goal != "" {
		fmt.Printf("  Goal: %s\n", data.Goal)
	}
	if data.Summary != "" {
		fmt.Printf("  Summary: %s\n", data.Summary)
	} else if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	if data.RequestID != "" {
		fmt.Printf("  Request: %s\n", data.RequestID)
	}
	if data.RollbackHandle != "" {
		fmt.Printf("  Rollback: %s\n", data.RollbackHandle)
	}
	if data.Checkpoint != nil {
		fmt.Printf("  Checkpoint: %s\n", checkpointLabel(*data.Checkpoint))
	}

	printChanges("Changes", data.Changes)
	printStringSection("Warnings", data.Warnings)
	printStringSection("Errors", data.Errors)
	return nil
}

func renderHarnessRollback(resp *protocol.Response) error {
	var data protocol.HarnessRollbackData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	status := "not-restored"
	if data.Restored {
		status = "restored"
	}
	fmt.Printf("Harness rollback: %s\n", status)
	if data.Scope != "" {
		fmt.Printf("  Scope: %s\n", data.Scope)
	}
	if data.Handle != "" {
		fmt.Printf("  Handle: %s\n", data.Handle)
	}
	if data.RequestID != "" {
		fmt.Printf("  Request: %s\n", data.RequestID)
	}
	if data.Summary != "" {
		fmt.Printf("  Summary: %s\n", data.Summary)
	} else if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	printStringSection("Warnings", data.Warnings)
	printStringSection("Errors", data.Errors)
	return nil
}

func renderHarnessHistory(resp *protocol.Response) error {
	var data protocol.HarnessHistoryData
	if err := decodeHarnessData(resp.Data, &data); err != nil {
		return err
	}

	fmt.Println("Harness history")
	if data.Scope != "" {
		fmt.Printf("  Scope: %s\n", data.Scope)
	}
	if data.Summary != "" {
		fmt.Printf("  Summary: %s\n", data.Summary)
	} else if strings.TrimSpace(resp.Message) != "" {
		fmt.Printf("  Summary: %s\n", resp.Message)
	}
	if data.Cursor != "" {
		fmt.Printf("  Cursor: %s\n", data.Cursor)
	}

	if len(data.Entries) == 0 {
		fmt.Println("  No entries")
		return nil
	}

	fmt.Println("Entries:")
	for _, entry := range data.Entries {
		fmt.Printf("  %s  %s  %s\n", formatHarnessTimestamp(entry.Timestamp), historyStatus(entry), historyHeadline(entry))
		if detail := historyDetail(entry); detail != "" {
			fmt.Printf("    %s\n", detail)
		}
	}
	return nil
}

func decodeHarnessData(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("failed to parse harness response: %w", err)
	}
	return nil
}

func renderConnector(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	switch {
	case name == "" && version == "":
		return ""
	case name == "":
		return version
	case version == "":
		return name
	default:
		return fmt.Sprintf("%s v%s", name, version)
	}
}

func sortedCopy(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := append([]string(nil), items...)
	sort.Strings(out)
	return out
}

func allStringSlicesEmpty(groups ...[]string) bool {
	for _, group := range groups {
		if len(group) > 0 {
			return false
		}
	}
	return true
}

func printStringSection(title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		fmt.Printf("  - %s\n", item)
	}
}

func printTargets(title string, items []protocol.HarnessTarget) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		label := firstNonEmpty(item.Label, item.Path, item.Role, "unnamed target")
		var parts []string
		if item.Path != "" && item.Path != label {
			parts = append(parts, item.Path)
		}
		if item.Type != "" {
			parts = append(parts, item.Type)
		}
		if item.Family != "" {
			parts = append(parts, item.Family)
		}
		if item.Role != "" && item.Role != label {
			parts = append(parts, "role="+item.Role)
		}
		if item.Active {
			parts = append(parts, "active")
		}
		fmt.Printf("  - %s", label)
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
	}
}

func printChanges(title string, items []protocol.HarnessChange) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		label := firstNonEmpty(item.Summary, item.Path, item.Target, "change")
		var prefix string
		if item.Action != "" {
			prefix = item.Action + ": "
		}
		var details []string
		if item.Path != "" && item.Path != label {
			details = append(details, item.Path)
		}
		if item.Target != "" && item.Target != label && item.Target != item.Path {
			details = append(details, item.Target)
		}
		if item.Family != "" {
			details = append(details, item.Family)
		}
		fmt.Printf("  - %s%s", prefix, label)
		if len(details) > 0 {
			fmt.Printf(" (%s)", strings.Join(details, ", "))
		}
		fmt.Println()
	}
}

func printMetrics(title string, items []protocol.HarnessMetric) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		label := firstNonEmpty(item.Name, item.Target, item.Summary, "metric")
		var parts []string
		if item.Value != 0 {
			parts = append(parts, fmt.Sprintf("%.2f%s", item.Value, item.Unit))
		} else if item.Unit != "" {
			parts = append(parts, item.Unit)
		}
		if item.Severity != "" {
			parts = append(parts, "severity="+item.Severity)
		}
		if item.Target != "" && item.Target != label {
			parts = append(parts, item.Target)
		}
		if item.Summary != "" && item.Summary != label {
			parts = append(parts, item.Summary)
		}
		fmt.Printf("  - %s", label)
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
	}
}

func printCheckResults(items []protocol.HarnessCheckResult) {
	if len(items) == 0 {
		return
	}
	fmt.Println("Checks:")
	for _, item := range items {
		status := strings.ToUpper(firstNonEmpty(item.Status, boolStatus(item.Passed, "pass", "fail")))
		label := firstNonEmpty(item.Name, item.Target, item.Kind, "check")
		var parts []string
		if item.Kind != "" && item.Kind != label {
			parts = append(parts, item.Kind)
		}
		if item.Target != "" && item.Target != label {
			parts = append(parts, item.Target)
		}
		if item.Message != "" {
			parts = append(parts, item.Message)
		}
		fmt.Printf("  - [%s] %s", status, label)
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
	}
}

func printEvidence(items []protocol.HarnessEvidence) {
	if len(items) == 0 {
		return
	}
	fmt.Println("Evidence:")
	for _, item := range items {
		label := firstNonEmpty(item.Label, item.Path, item.Kind, "evidence")
		var parts []string
		if item.Kind != "" && item.Kind != label {
			parts = append(parts, item.Kind)
		}
		if item.Path != "" && item.Path != label {
			parts = append(parts, item.Path)
		}
		if item.Value != "" {
			parts = append(parts, item.Value)
		}
		if item.Summary != "" {
			parts = append(parts, item.Summary)
		}
		fmt.Printf("  - %s", label)
		if len(parts) > 0 {
			fmt.Printf(" (%s)", strings.Join(parts, ", "))
		}
		fmt.Println()
	}
}

func checkpointLabel(checkpoint protocol.HarnessCheckpoint) string {
	handle := firstNonEmpty(checkpoint.Handle, checkpoint.ID, "checkpoint")
	if checkpoint.CreatedAt == 0 {
		return handle
	}
	return fmt.Sprintf("%s @ %s", handle, formatHarnessTimestamp(checkpoint.CreatedAt))
}

func historyStatus(entry protocol.HarnessHistoryEntry) string {
	switch {
	case entry.Restored:
		return "RESTORED"
	case entry.Passed:
		return "PASS"
	case entry.Applied:
		if entry.DryRun {
			return "DRY-RUN"
		}
		return "APPLIED"
	case entry.Success:
		return "OK"
	default:
		return "FAIL"
	}
}

func historyHeadline(entry protocol.HarnessHistoryEntry) string {
	parts := []string{firstNonEmpty(entry.Route, "unknown-route")}
	if entry.Scope != "" {
		parts = append(parts, entry.Scope)
	}
	if entry.Goal != "" {
		parts = append(parts, entry.Goal)
	} else if entry.Summary != "" {
		parts = append(parts, entry.Summary)
	}
	return strings.Join(parts, "  ")
}

func historyDetail(entry protocol.HarnessHistoryEntry) string {
	var parts []string
	if entry.RequestID != "" {
		parts = append(parts, "request="+entry.RequestID)
	}
	if entry.RollbackHandle != "" {
		parts = append(parts, "rollback="+entry.RollbackHandle)
	}
	if len(entry.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("warnings=%d", len(entry.Warnings)))
	}
	if len(entry.Errors) > 0 {
		parts = append(parts, fmt.Sprintf("errors=%d", len(entry.Errors)))
	}
	return strings.Join(parts, "  ")
}

func formatHarnessTimestamp(ts float64) string {
	if ts == 0 {
		return "unknown-time"
	}
	return time.Unix(int64(ts), 0).Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boolStatus(value bool, whenTrue, whenFalse string) string {
	if value {
		return whenTrue
	}
	return whenFalse
}
