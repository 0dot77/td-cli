package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/0dot77/td-cli/internal/client"
)

type snapshotData struct {
	Nodes []snapshotNode `json:"nodes"`
}

type snapshotNode struct {
	Path       string                       `json:"path"`
	Name       string                       `json:"name"`
	Type       string                       `json:"type"`
	Family     string                       `json:"family"`
	Parameters map[string]snapshotParam     `json:"parameters"`
	Inputs     []snapshotInput              `json:"inputs"`
}

type snapshotParam struct {
	Value   string `json:"value"`
	Default string `json:"default"`
}

type snapshotInput struct {
	Index      int    `json:"index"`
	SourcePath string `json:"sourcePath"`
}

type diffResult struct {
	Added    []string   `json:"added"`
	Removed  []string   `json:"removed"`
	Modified []nodeDiff `json:"modified"`
}

type nodeDiff struct {
	Path          string      `json:"path"`
	ParamsChanged []paramDiff `json:"paramsChanged,omitempty"`
	InputsChanged bool        `json:"inputsChanged,omitempty"`
}

type paramDiff struct {
	Name     string `json:"name"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// DiffFiles compares two network snapshot files.
func DiffFiles(file1, file2 string, jsonOutput bool) error {
	snap1, err := loadSnapshot(file1)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", file1, err)
	}
	snap2, err := loadSnapshot(file2)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", file2, err)
	}

	result := computeDiff(snap1, snap2)
	return printDiff(result, file1, file2, jsonOutput)
}

// DiffLive compares current TD state against a saved snapshot.
func DiffLive(c *client.Client, snapshotFile string, path string, jsonOutput bool) error {
	snap1, err := loadSnapshot(snapshotFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", snapshotFile, err)
	}

	// Get live state
	payload := map[string]interface{}{
		"path":  path,
		"depth": 10,
	}
	resp, err := c.Call("/network/export", payload)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}

	var snap2 snapshotData
	if err := json.Unmarshal(resp.Data, &snap2); err != nil {
		return fmt.Errorf("failed to parse live data: %w", err)
	}

	result := computeDiff(snap1, snap2)
	return printDiff(result, snapshotFile, "live", jsonOutput)
}

func loadSnapshot(path string) (snapshotData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshotData{}, err
	}
	var snap snapshotData
	if err := json.Unmarshal(data, &snap); err != nil {
		return snapshotData{}, err
	}
	return snap, nil
}

func computeDiff(a, b snapshotData) diffResult {
	aMap := make(map[string]snapshotNode)
	for _, n := range a.Nodes {
		aMap[n.Path] = n
	}
	bMap := make(map[string]snapshotNode)
	for _, n := range b.Nodes {
		bMap[n.Path] = n
	}

	var result diffResult

	// Removed: in A but not B
	for path := range aMap {
		if _, ok := bMap[path]; !ok {
			result.Removed = append(result.Removed, path)
		}
	}

	// Added: in B but not A
	for path := range bMap {
		if _, ok := aMap[path]; !ok {
			result.Added = append(result.Added, path)
		}
	}

	// Modified: in both, check params and inputs
	for path, nodeA := range aMap {
		nodeB, ok := bMap[path]
		if !ok {
			continue
		}

		var diff nodeDiff
		diff.Path = path

		// Compare parameters
		allParams := make(map[string]bool)
		for k := range nodeA.Parameters {
			allParams[k] = true
		}
		for k := range nodeB.Parameters {
			allParams[k] = true
		}

		for param := range allParams {
			valA := ""
			valB := ""
			if p, ok := nodeA.Parameters[param]; ok {
				valA = p.Value
			}
			if p, ok := nodeB.Parameters[param]; ok {
				valB = p.Value
			}
			if valA != valB {
				diff.ParamsChanged = append(diff.ParamsChanged, paramDiff{
					Name:     param,
					OldValue: valA,
					NewValue: valB,
				})
			}
		}

		// Compare inputs
		inputsA, _ := json.Marshal(nodeA.Inputs)
		inputsB, _ := json.Marshal(nodeB.Inputs)
		if string(inputsA) != string(inputsB) {
			diff.InputsChanged = true
		}

		if len(diff.ParamsChanged) > 0 || diff.InputsChanged {
			result.Modified = append(result.Modified, diff)
		}
	}

	return result
}

func printDiff(result diffResult, nameA, nameB string, jsonOutput bool) error {
	if jsonOutput {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	total := len(result.Added) + len(result.Removed) + len(result.Modified)
	if total == 0 {
		fmt.Println("No differences found")
		return nil
	}

	fmt.Printf("Diff: %s vs %s\n\n", nameA, nameB)

	if len(result.Added) > 0 {
		fmt.Printf("Added (%d):\n", len(result.Added))
		for _, p := range result.Added {
			fmt.Printf("  + %s\n", p)
		}
		fmt.Println()
	}

	if len(result.Removed) > 0 {
		fmt.Printf("Removed (%d):\n", len(result.Removed))
		for _, p := range result.Removed {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println()
	}

	if len(result.Modified) > 0 {
		fmt.Printf("Modified (%d):\n", len(result.Modified))
		for _, m := range result.Modified {
			fmt.Printf("  ~ %s\n", m.Path)
			for _, p := range m.ParamsChanged {
				fmt.Printf("      %s: %s -> %s\n", p.Name, p.OldValue, p.NewValue)
			}
			if m.InputsChanged {
				fmt.Println("      (connections changed)")
			}
		}
	}

	return nil
}
