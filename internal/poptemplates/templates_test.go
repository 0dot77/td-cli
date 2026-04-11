package poptemplates

import (
	"strings"
	"testing"
)

func TestRenderAudioReactiveTemplate(t *testing.T) {
	tmpl, script, err := Render("av", "/project99", "club")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if tmpl.Key != "audio-reactive" {
		t.Fatalf("template key = %q, want %q", tmpl.Key, "audio-reactive")
	}
	for _, fragment := range []string{
		`ROOT_PATH = "/project99"`,
		`BASE_NAME = "club"`,
		`PREVIEW_NAME = "club_preview"`,
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("rendered script missing fragment %q", fragment)
		}
	}
	if strings.Contains(script, "__ROOT_PATH__") || strings.Contains(script, "__BASE_NAME__") || strings.Contains(script, "__PREVIEW_NAME__") {
		t.Fatalf("rendered script still contains placeholders")
	}
}

func TestListIncludesAudioReactiveTemplate(t *testing.T) {
	templates := List()
	if len(templates) == 0 {
		t.Fatal("List returned no templates")
	}
	if templates[0].Key != "audio-reactive" {
		t.Fatalf("first template = %q, want %q", templates[0].Key, "audio-reactive")
	}
}
