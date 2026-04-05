package shaders

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Shader represents a GLSL shader template.
type Shader struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	GLSL        string    `json:"glsl"`
	Uniforms    []Uniform `json:"uniforms"`
	Inputs      []string  `json:"inputs"`
	Tags        []string  `json:"tags"`
}

// Uniform describes a shader uniform parameter.
type Uniform struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Expression  string      `json:"expression,omitempty"`
	Description string      `json:"description"`
}

// ShaderSummary is a brief listing entry.
type ShaderSummary struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

var shaderMap map[string]Shader

func init() {
	shaderMap = make(map[string]Shader)
	json.Unmarshal(shadersJSON, &shaderMap)
}

// List returns all shaders, optionally filtered by category.
func List(category string) []ShaderSummary {
	var results []ShaderSummary
	for key, s := range shaderMap {
		if category != "" && !strings.EqualFold(s.Category, category) {
			continue
		}
		results = append(results, ShaderSummary{
			Key:         key,
			Name:        s.Name,
			Category:    s.Category,
			Description: s.Description,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Category != results[j].Category {
			return results[i].Category < results[j].Category
		}
		return results[i].Key < results[j].Key
	})
	return results
}

// Get returns a shader by key.
func Get(key string) *Shader {
	s, ok := shaderMap[key]
	if !ok {
		return nil
	}
	return &s
}

// Search finds shaders matching a query string.
func Search(query string, limit int) []ShaderSummary {
	query = strings.ToLower(query)
	var results []ShaderSummary
	for key, s := range shaderMap {
		score := 0
		if strings.Contains(strings.ToLower(key), query) {
			score += 3
		}
		if strings.Contains(strings.ToLower(s.Name), query) {
			score += 2
		}
		if strings.Contains(strings.ToLower(s.Description), query) {
			score += 1
		}
		for _, tag := range s.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				score += 2
			}
		}
		if score > 0 {
			results = append(results, ShaderSummary{
				Key:         key,
				Name:        s.Name,
				Category:    s.Category,
				Description: s.Description,
			})
		}
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// FormatShader formats a shader for CLI display.
func FormatShader(key string, s *Shader) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (%s)\n", s.Name, s.Category))
	b.WriteString(fmt.Sprintf("%s\n\n", s.Description))

	if len(s.Uniforms) > 0 {
		b.WriteString("Uniforms:\n")
		for _, u := range s.Uniforms {
			expr := ""
			if u.Expression != "" {
				expr = fmt.Sprintf(" (expr: %s)", u.Expression)
			}
			b.WriteString(fmt.Sprintf("  %-15s %-8s default=%-6v %s%s\n",
				u.Name, u.Type, u.Default, u.Description, expr))
		}
		b.WriteString("\n")
	}

	if len(s.Inputs) > 0 {
		b.WriteString("Inputs:\n")
		for _, inp := range s.Inputs {
			b.WriteString(fmt.Sprintf("  - %s\n", inp))
		}
		b.WriteString("\n")
	}

	b.WriteString("GLSL:\n")
	b.WriteString(s.GLSL)
	b.WriteString("\n")

	return b.String()
}
