package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func TimelineInfo(c *client.Client, jsonOutput bool) error {
	resp, err := c.Call("/timeline/info", map[string]interface{}{})
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
		CurrentTime  float64 `json:"currentTime"`
		Start        float64 `json:"start"`
		End          float64 `json:"end"`
		Rate         float64 `json:"rate"`
		IsPlaying    bool    `json:"isPlaying"`
		PlayMode     string  `json:"playMode"`
		CuePoint     float64 `json:"cuePoint"`
		IsCueEnabled bool    `json:"isCueEnabled"`
		Signaled     bool    `json:"signaled"`
	}
	json.Unmarshal(resp.Data, &data)
	state := "paused"
	if data.IsPlaying {
		state = "playing"
	}
	fmt.Printf("Timeline [%s] %.4f / %.1f-%.1f  rate=%.2f\n", state, data.CurrentTime, data.Start, data.End, data.Rate)
	fmt.Printf("  Play mode: %s  Cue: %.4f (enabled=%v)\n", data.PlayMode, data.CuePoint, data.IsCueEnabled)
	if data.Signaled {
		fmt.Println("  Signaled: true")
	}
	return nil
}

func TimelinePlay(c *client.Client, jsonOutput bool) error {
	resp, err := c.Call("/timeline/play", map[string]interface{}{})
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
	fmt.Println("Timeline playing")
	return nil
}

func TimelinePause(c *client.Client, jsonOutput bool) error {
	resp, err := c.Call("/timeline/pause", map[string]interface{}{})
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
	fmt.Println("Timeline paused")
	return nil
}

func TimelineSeek(c *client.Client, time float64, jsonOutput bool) error {
	resp, err := c.Call("/timeline/seek", map[string]interface{}{"time": time})
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
	fmt.Printf("Seeked to %.4f\n", time)
	return nil
}

func TimelineRange(c *client.Client, start, end float64, jsonOutput bool) error {
	resp, err := c.Call("/timeline/range", map[string]interface{}{"start": start, "end": end})
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
	fmt.Printf("Timeline range set: %.1f - %.1f\n", start, end)
	return nil
}

func TimelineRate(c *client.Client, rate float64, jsonOutput bool) error {
	resp, err := c.Call("/timeline/rate", map[string]interface{}{"rate": rate})
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
	fmt.Printf("Timeline rate set: %.2f\n", rate)
	return nil
}

func CookNode(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/cook/node", map[string]interface{}{"path": path})
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
		Path     string  `json:"path"`
		CookTime float64 `json:"cookTime"`
		Cooked   bool    `json:"cooked"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Cooked %s (%.2fms)\n", data.Path, data.CookTime)
	return nil
}

func CookNetwork(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/cook/network", map[string]interface{}{"path": path})
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
		Path        string  `json:"path"`
		NodesCooked int     `json:"nodesCooked"`
		TotalTime   float64 `json:"totalTime"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Cooked network %s: %d nodes in %.2fms\n", data.Path, data.NodesCooked, data.TotalTime)
	return nil
}

func UiNavigate(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/ui/navigate", map[string]interface{}{"path": path})
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
	fmt.Printf("Navigated to %s\n", path)
	return nil
}

func UiSelect(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/ui/select", map[string]interface{}{"path": path})
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
	fmt.Printf("Selected %s\n", path)
	return nil
}

func UiPulse(c *client.Client, path, name string, jsonOutput bool) error {
	resp, err := c.Call("/ui/pulse", map[string]interface{}{"path": path, "name": name})
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
	fmt.Printf("Pulsed %s.%s\n", path, name)
	return nil
}
