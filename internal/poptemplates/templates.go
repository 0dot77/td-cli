package poptemplates

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed data/*.py
var templateFS embed.FS

type Template struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var templates = map[string]Template{
	"audio-reactive": {
		Key:         "audio-reactive",
		Name:        "POP Audio Reactive Surface",
		Description: "Build a POP-based audio reactive scene with a preview container and TOP output.",
	},
}

func List() []Template {
	keys := make([]string, 0, len(templates))
	for key := range templates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]Template, 0, len(keys))
	for _, key := range keys {
		result = append(result, templates[key])
	}
	return result
}

func Render(key string, root string, name string) (Template, string, error) {
	canonical := canonicalKey(key)
	tmpl, ok := templates[canonical]
	if !ok {
		return Template{}, "", fmt.Errorf("unknown POP template: %s", key)
	}
	if root == "" {
		root = "/project1"
	}
	if name == "" {
		name = "pop_audio_visual"
	}

	raw, err := templateFS.ReadFile("data/" + canonical + ".py")
	if err != nil {
		return Template{}, "", fmt.Errorf("cannot read POP template %q: %w", canonical, err)
	}

	replacer := strings.NewReplacer(
		"__ROOT_PATH__", pythonString(root),
		"__BASE_NAME__", pythonString(name),
		"__PREVIEW_NAME__", pythonString(name+"_preview"),
	)

	return tmpl, replacer.Replace(string(raw)), nil
}

func canonicalKey(key string) string {
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "", "audio-reactive", "audio_reactive", "av":
		return "audio-reactive"
	default:
		return key
	}
}

func pythonString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
