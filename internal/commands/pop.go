package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func PopInfo(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/pop/info", map[string]interface{}{"path": path})
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
		Name            string   `json:"name"`
		Type            string   `json:"type"`
		NumPoints       int      `json:"numPoints"`
		NumPrims        int      `json:"numPrims"`
		NumVerts        int      `json:"numVerts"`
		Dimension       string   `json:"dimension"`
		PointAttributes []string `json:"pointAttributes"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("POP: %s (%s)\n", data.Name, data.Type)
	fmt.Printf("  Points: %d  Prims: %d  Verts: %d\n", data.NumPoints, data.NumPrims, data.NumVerts)
	if data.Dimension != "" {
		fmt.Printf("  Dimension: %s\n", data.Dimension)
	}
	if len(data.PointAttributes) > 0 {
		fmt.Printf("  Attributes: %v\n", data.PointAttributes)
	}
	return nil
}

func PopPoints(c *client.Client, path, attr string, start, count int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if attr != "" {
		payload["attribute"] = attr
	}
	if start > 0 {
		payload["start"] = start
	}
	if count > 0 {
		payload["count"] = count
	}
	resp, err := c.Call("/pop/points", payload)
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
	printPopData(resp.Data)
	return nil
}

func PopPrims(c *client.Client, path, attr string, start, count int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if attr != "" {
		payload["attribute"] = attr
	}
	if start > 0 {
		payload["start"] = start
	}
	if count > 0 {
		payload["count"] = count
	}
	resp, err := c.Call("/pop/prims", payload)
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
	printPopData(resp.Data)
	return nil
}

func PopVerts(c *client.Client, path, attr string, start, count int, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if attr != "" {
		payload["attribute"] = attr
	}
	if start > 0 {
		payload["start"] = start
	}
	if count > 0 {
		payload["count"] = count
	}
	resp, err := c.Call("/pop/verts", payload)
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
	printPopData(resp.Data)
	return nil
}

func PopBounds(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/pop/bounds", map[string]interface{}{"path": path})
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
		MinX, MinY, MinZ          float64 `json:"minX,minY,minZ"`
		MaxX, MaxY, MaxZ          float64 `json:"maxX,maxY,maxZ"`
		CenterX, CenterY, CenterZ float64 `json:"centerX,centerY,centerZ"`
		SizeX, SizeY, SizeZ       float64 `json:"sizeX,sizeY,sizeZ"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Bounds:\n")
	fmt.Printf("  Min:    %.4f %.4f %.4f\n", data.MinX, data.MinY, data.MinZ)
	fmt.Printf("  Max:    %.4f %.4f %.4f\n", data.MaxX, data.MaxY, data.MaxZ)
	fmt.Printf("  Center: %.4f %.4f %.4f\n", data.CenterX, data.CenterY, data.CenterZ)
	fmt.Printf("  Size:   %.4f %.4f %.4f\n", data.SizeX, data.SizeY, data.SizeZ)
	return nil
}

func PopAttributes(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/pop/attributes", map[string]interface{}{"path": path})
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
		PointAttributes []string `json:"pointAttributes"`
		PrimAttributes  []string `json:"primAttributes"`
		VertAttributes  []string `json:"vertAttributes"`
	}
	json.Unmarshal(resp.Data, &data)
	if len(data.PointAttributes) > 0 {
		fmt.Printf("  Point: %v\n", data.PointAttributes)
	}
	if len(data.PrimAttributes) > 0 {
		fmt.Printf("  Prim:  %v\n", data.PrimAttributes)
	}
	if len(data.VertAttributes) > 0 {
		fmt.Printf("  Vert:  %v\n", data.VertAttributes)
	}
	return nil
}

func PopSave(c *client.Client, path, filepath string, jsonOutput bool) error {
	payload := map[string]interface{}{"path": path}
	if filepath != "" {
		payload["filepath"] = filepath
	}
	resp, err := c.Call("/pop/save", payload)
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
		Filepath string `json:"filepath"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Saved to: %s\n", data.Filepath)
	return nil
}

func printPopData(raw json.RawMessage) {
	var data struct {
		Attribute string      `json:"attribute"`
		Start     int         `json:"start"`
		Count     int         `json:"count"`
		Values    interface{} `json:"values"`
	}
	json.Unmarshal(raw, &data)
	fmt.Printf("%s (start=%d, count=%d)\n", data.Attribute, data.Start, data.Count)
	switch v := data.Values.(type) {
	case []interface{}:
		n := len(v)
		if n > 10 {
			fmt.Printf("  [%v ... %v] (%d values)\n", v[:3], v[n-3:], n)
		} else {
			fmt.Printf("  %v\n", v)
		}
	default:
		fmt.Printf("  %v\n", v)
	}
}
