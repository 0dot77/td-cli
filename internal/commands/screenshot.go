package commands

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/0dot77/td-cli/internal/client"
)

type screenshotResult struct {
	Image  string `json:"image"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Screenshot captures a TOP output as PNG. When opaque is true, alpha is forced to 255.
func Screenshot(c *client.Client, path, outputFile string, opaque, jsonOutput bool) error {
	payload := map[string]string{}
	if path != "" {
		payload["path"] = path
	}

	resp, err := c.Call("/screenshot", payload)
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

	var result screenshotResult
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	data, err := base64.StdEncoding.DecodeString(result.Image)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	if opaque {
		flattened, err := flattenAlpha(data)
		if err != nil {
			return fmt.Errorf("failed to flatten alpha: %w", err)
		}
		data = flattened
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Screenshot saved to %s (%dx%d)\n", outputFile, result.Width, result.Height)
	} else {
		if opaque {
			fmt.Print(base64.StdEncoding.EncodeToString(data))
		} else {
			fmt.Print(result.Image)
		}
	}

	return nil
}

func flattenAlpha(pngBytes []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	if src, ok := img.(*image.NRGBA); ok {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				si := src.PixOffset(x, y)
				di := out.PixOffset(x, y)
				out.Pix[di+0] = src.Pix[si+0]
				out.Pix[di+1] = src.Pix[si+1]
				out.Pix[di+2] = src.Pix[si+2]
				out.Pix[di+3] = 255
			}
		}
	} else {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				di := out.PixOffset(x, y)
				out.Pix[di+0] = c.R
				out.Pix[di+1] = c.G
				out.Pix[di+2] = c.B
				out.Pix[di+3] = 255
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}
