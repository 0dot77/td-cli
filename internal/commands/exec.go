package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0dot77/td-cli/internal/client"
)

type execResult struct {
	Result string `json:"result"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// Exec executes Python code in TouchDesigner.
func Exec(c *client.Client, code string, filePath string, jsonOutput bool, verifyPath string, screenshotPath string) error {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}
		code = string(data)
	}

	if code == "" {
		return fmt.Errorf("no code provided (use td-cli exec \"<code>\" or td-cli exec -f <file>)")
	}

	resp, err := c.Call("/exec", map[string]string{"code": code})
	if err != nil {
		return err
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("execution error: %s", resp.Message)
	}

	var result execResult
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("failed to parse response data: %w", err)
		}
	}

	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Result != "" {
		fmt.Println(result.Result)
	}
	if result.Stderr != "" {
		fmt.Fprintf(os.Stderr, "%s", result.Stderr)
	}

	// Capture screenshot after successful execution if requested.
	if screenshotPath != "" {
		screenshotResp, err := c.Call("/screenshot", map[string]string{"path": screenshotPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Screenshot warning: %s\n", err)
			return nil
		}
		if !screenshotResp.Success {
			fmt.Fprintf(os.Stderr, "Screenshot warning: %s\n", screenshotResp.Message)
			return nil
		}

		var ssResult screenshotResult
		if screenshotResp.Data != nil {
			if err := json.Unmarshal(screenshotResp.Data, &ssResult); err != nil {
				fmt.Fprintf(os.Stderr, "Screenshot warning: failed to parse response data: %s\n", err)
				return nil
			}
		}

		imgData, err := base64.StdEncoding.DecodeString(ssResult.Image)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Screenshot warning: failed to decode image: %s\n", err)
			return nil
		}

		outPath := filepath.Join(".tmp", "preview.png")
		if err := os.MkdirAll(".tmp", 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Screenshot warning: failed to create .tmp dir: %s\n", err)
			return nil
		}
		if err := os.WriteFile(outPath, imgData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Screenshot warning: failed to write file: %s\n", err)
			return nil
		}
		if ssResult.Width > 0 && ssResult.Height > 0 {
			fmt.Printf("Screenshot saved to .tmp/preview.png (%dx%d)\n", ssResult.Width, ssResult.Height)
		} else {
			fmt.Printf("Screenshot saved to .tmp/preview.png\n")
		}
	}

	// Verify node graph at the target path after successful execution.
	if verifyPath != "" {
		verifyResp, verifyErr := c.Call("/harness/observe", map[string]interface{}{
			"path":  verifyPath,
			"depth": 1,
		})
		if verifyErr != nil {
			fmt.Fprintf(os.Stderr, "Verify warning: %s\n", verifyErr)
		} else if !verifyResp.Success {
			fmt.Fprintf(os.Stderr, "Verify warning: %s\n", verifyResp.Message)
		} else if verifyResp.Data != nil {
			var verifyData struct {
				Graph struct {
					NodeCount       int `json:"nodeCount"`
					ConnectionCount int `json:"connectionCount"`
				} `json:"graph"`
				Issues struct {
					IssueCount     int `json:"issueCount"`
					TargetErrors   []string `json:"targetErrors"`
					TargetWarnings []string `json:"targetWarnings"`
					Nodes          []struct {
						Path     string   `json:"path"`
						Errors   []string `json:"errors"`
						Warnings []string `json:"warnings"`
					} `json:"nodes"`
				} `json:"issues"`
			}
			if jsonErr := json.Unmarshal(verifyResp.Data, &verifyData); jsonErr == nil {
				ic := verifyData.Issues.IssueCount
				if ic > 0 {
					fmt.Printf("Verify: %s — %d nodes, %d connections, %d issues ⚠\n",
						verifyPath, verifyData.Graph.NodeCount, verifyData.Graph.ConnectionCount, ic)
					for _, e := range verifyData.Issues.TargetErrors {
						fmt.Printf("  ✗ %s\n", e)
					}
					for _, w := range verifyData.Issues.TargetWarnings {
						fmt.Printf("  ⚠ %s\n", w)
					}
					for _, n := range verifyData.Issues.Nodes {
						for _, e := range n.Errors {
							fmt.Printf("  ✗ %s: %s\n", n.Path, e)
						}
						for _, w := range n.Warnings {
							fmt.Printf("  ⚠ %s: %s\n", n.Path, w)
						}
					}
				} else {
					fmt.Printf("Verify: %s — %d nodes, %d connections ✓\n",
						verifyPath, verifyData.Graph.NodeCount, verifyData.Graph.ConnectionCount)
				}
			}
		}
	}

	return nil
}
