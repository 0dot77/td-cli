package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func ChopInfo(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/chop/info", map[string]interface{}{"path": path})
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
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		NumChannels int     `json:"numChannels"`
		NumSamples  int     `json:"numSamples"`
		SampleRate  float64 `json:"sampleRate"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("CHOP: %s (%s)\n", data.Name, data.Type)
	fmt.Printf("  Channels: %d  Samples: %d  Rate: %.1f\n", data.NumChannels, data.NumSamples, data.SampleRate)
	return nil
}

func ChopChannels(c *client.Client, path string, start, count int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if start > 0 {
		payload["start"] = start
	}
	if count > 0 {
		payload["count"] = count
	}
	resp, err := c.Call("/chop/channels", payload)
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
		Channels []struct {
			Name   string    `json:"name"`
			Values []float64 `json:"values"`
		} `json:"channels"`
	}
	json.Unmarshal(resp.Data, &data)
	for _, ch := range data.Channels {
		n := len(ch.Values)
		if n > 5 {
			fmt.Printf("  %s: [%v ... %v] (%d samples)\n", ch.Name, ch.Values[:3], ch.Values[n-2:], n)
		} else {
			fmt.Printf("  %s: %v\n", ch.Name, ch.Values)
		}
	}
	return nil
}

func ChopSample(c *client.Client, path, channel string, index int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path, "index": index}
	if channel != "" {
		payload["channel"] = channel
	}
	resp, err := c.Call("/chop/sample", payload)
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
		Channel string  `json:"channel"`
		Index   int     `json:"index"`
		Value   float64 `json:"value"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("%s[%d] = %v\n", data.Channel, data.Index, data.Value)
	return nil
}
