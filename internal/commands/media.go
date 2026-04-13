package commands

import (
	"encoding/json"
	"fmt"

	"github.com/0dot77/td-cli/internal/client"
)

func MediaInfo(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/media/info", map[string]interface{}{"path": path})
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
		Path      string  `json:"path"`
		Type      string  `json:"type"`
		Width     int     `json:"width"`
		Height    int     `json:"height"`
		Duration  float64 `json:"duration"`
		FrameRate float64 `json:"frameRate"`
		Codec     string  `json:"codec"`
		FilePath  string  `json:"filePath"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}
	fmt.Printf("Media: %s (%s)\n", data.Path, data.Type)
	if data.Width > 0 {
		fmt.Printf("  Resolution: %dx%d\n", data.Width, data.Height)
	}
	if data.Duration > 0 {
		fmt.Printf("  Duration: %.2fs  Rate: %.2ffps\n", data.Duration, data.FrameRate)
	}
	if data.Codec != "" {
		fmt.Printf("  Codec: %s\n", data.Codec)
	}
	if data.FilePath != "" {
		fmt.Printf("  File: %s\n", data.FilePath)
	}
	return nil
}

func MediaExport(c *client.Client, path, outputFile string, jsonOutput bool) error {
	resp, err := c.Call("/media/export", map[string]interface{}{"path": path, "outputFile": outputFile})
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
		OutputPath string  `json:"outputPath"`
		Duration   float64 `json:"duration"`
		FrameCount int     `json:"frameCount"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}
	fmt.Printf("Exported: %s (%d frames, %.2fs)\n", data.OutputPath, data.FrameCount, data.Duration)
	return nil
}

func MediaRecord(c *client.Client, path string, start, end float64, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if start > 0 {
		payload["start"] = start
	}
	if end > 0 {
		payload["end"] = end
	}
	resp, err := c.Call("/media/record", payload)
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
	fmt.Println("Recording started")
	return nil
}

func MediaSnapshot(c *client.Client, path, outputFile string, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if outputFile != "" {
		payload["outputFile"] = outputFile
	}
	resp, err := c.Call("/media/snapshot", payload)
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
		OutputPath string `json:"outputPath"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}
	fmt.Printf("Snapshot: %s\n", data.OutputPath)
	return nil
}
