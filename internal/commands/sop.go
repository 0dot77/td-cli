package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
)

func SopInfo(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/sop/info", map[string]interface{}{"path": path})
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
		Name      string `json:"name"`
		Type      string `json:"type"`
		NumPoints int    `json:"numPoints"`
		NumPrims  int    `json:"numPrims"`
		NumVerts  int    `json:"numVerts"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("SOP: %s (%s)\n", data.Name, data.Type)
	fmt.Printf("  Points: %d  Prims: %d  Verts: %d\n", data.NumPoints, data.NumPrims, data.NumVerts)
	return nil
}

func SopPoints(c *client.Client, path string, start, limit int, jsonOutput bool) error {
	resp, err := c.Call("/sop/points", map[string]interface{}{
		"path": path, "start": start, "limit": limit,
	})
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
		TotalPoints int `json:"totalPoints"`
		Count       int `json:"count"`
		Points      []struct {
			Index int     `json:"index"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
			Z     float64 `json:"z"`
		} `json:"points"`
	}
	json.Unmarshal(resp.Data, &data)
	fmt.Printf("Points %d-%d of %d:\n", data.Points[0].Index, data.Points[len(data.Points)-1].Index, data.TotalPoints)
	for _, p := range data.Points {
		fmt.Printf("  [%d] %.4f %.4f %.4f\n", p.Index, p.X, p.Y, p.Z)
	}
	return nil
}

func SopAttribs(c *client.Client, path string, jsonOutput bool) error {
	resp, err := c.Call("/sop/attribs", map[string]interface{}{"path": path})
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
		PointAttributes     []string `json:"pointAttributes"`
		PrimitiveAttributes []string `json:"primitiveAttributes"`
		VertexAttributes    []string `json:"vertexAttributes"`
	}
	json.Unmarshal(resp.Data, &data)
	if len(data.PointAttributes) > 0 {
		fmt.Printf("  Point:  %v\n", data.PointAttributes)
	}
	if len(data.PrimitiveAttributes) > 0 {
		fmt.Printf("  Prim:   %v\n", data.PrimitiveAttributes)
	}
	if len(data.VertexAttributes) > 0 {
		fmt.Printf("  Vert:   %v\n", data.VertexAttributes)
	}
	return nil
}
