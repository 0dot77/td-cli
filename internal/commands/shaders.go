package commands

import (
	"encoding/json"
	"fmt"

	"github.com/td-cli/td-cli/internal/client"
	"github.com/td-cli/td-cli/internal/shaders"
)

// ShadersList lists available shader templates (offline, no TD needed).
func ShadersList(category string, jsonOutput bool) error {
	results := shaders.List(category)

	if jsonOutput {
		out, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if len(results) == 0 {
		fmt.Println("No shaders found")
		if category != "" {
			fmt.Printf("Category '%s' has no templates. Use: generative, postfx, utility, raymarching\n", category)
		}
		return nil
	}

	fmt.Printf("Shader templates (%d):\n\n", len(results))
	currentCat := ""
	for _, s := range results {
		if s.Category != currentCat {
			currentCat = s.Category
			fmt.Printf("  [%s]\n", currentCat)
		}
		fmt.Printf("    %-25s %s\n", s.Key, s.Description)
	}
	fmt.Println("\nUse 'td-cli shaders get <name>' for details")
	return nil
}

// ShadersGet prints a specific shader template (offline, no TD needed).
func ShadersGet(name string, jsonOutput bool) error {
	s := shaders.Get(name)
	if s == nil {
		// Try search
		results := shaders.Search(name, 5)
		if len(results) > 0 {
			fmt.Printf("Shader '%s' not found. Did you mean:\n", name)
			for _, r := range results {
				fmt.Printf("  %-25s %s\n", r.Key, r.Name)
			}
			return nil
		}
		return fmt.Errorf("shader not found: %s", name)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	fmt.Print(shaders.FormatShader(name, s))
	return nil
}

// ShadersApply sends a shader template to TD and applies it to a GLSL TOP.
func ShadersApply(c *client.Client, name string, targetPath string, jsonOutput bool) error {
	s := shaders.Get(name)
	if s == nil {
		return fmt.Errorf("shader not found: %s", name)
	}

	// Build uniform list for the handler
	uniforms := make([]map[string]interface{}, len(s.Uniforms))
	for i, u := range s.Uniforms {
		uni := map[string]interface{}{
			"name":    u.Name,
			"type":    u.Type,
			"default": u.Default,
		}
		if u.Expression != "" {
			uni["expression"] = u.Expression
		}
		uniforms[i] = uni
	}

	payload := map[string]interface{}{
		"path":     targetPath,
		"glsl":     s.GLSL,
		"uniforms": uniforms,
	}

	resp, err := c.Call("/shaders/apply", payload)
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

	var result struct {
		CompileWarnings string `json:"compileWarnings"`
	}
	json.Unmarshal(resp.Data, &result)

	fmt.Printf("Applied '%s' to %s\n", s.Name, targetPath)
	if result.CompileWarnings != "" {
		fmt.Printf("Warning: %s\n", result.CompileWarnings)
	}
	return nil
}
