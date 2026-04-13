package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0dot77/td-cli/internal/protocol"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory error = %v", err)
		}
	})
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}

func TestWriteCLAUDEMDWithoutHealthCreatesGenericGuidance(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)

	if err := writeCLAUDEMD(nil, 0); err != nil {
		t.Fatalf("writeCLAUDEMD() error = %v", err)
	}

	content := readFileForTest(t, "CLAUDE.md")
	for _, want := range []string{
		"# TouchDesigner Project - Agent Integration",
		"You do NOT need to ask the user to use td-cli",
		"td-cli context",
		"Treat `TDCliServer` as an installed connector",
		"`--project <path>`",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("CLAUDE.md missing %q", want)
		}
	}
	if strings.Contains(content, "## Project:") {
		t.Fatalf("CLAUDE.md unexpectedly included project metadata:\n%s", content)
	}
}

func TestWriteCLAUDEMDWithHealthIncludesProjectMetadata(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)

	health := &protocol.HealthData{
		Project:   "demo.toe",
		TDVersion: "2023.12340",
		TDBuild:   "12340",
	}

	if err := writeCLAUDEMD(health, 9981); err != nil {
		t.Fatalf("writeCLAUDEMD() error = %v", err)
	}

	content := readFileForTest(t, "CLAUDE.md")
	for _, want := range []string{
		"## Project: demo.toe",
		"- TD Version: 2023.12340 (build 12340)",
		"- Port: 9981",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("CLAUDE.md missing %q", want)
		}
	}
}

func TestWriteAGENTSMDWithoutHealthOmitsAutoDetectedInstance(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)

	if err := writeAGENTSMD(nil, 0); err != nil {
		t.Fatalf("writeAGENTSMD() error = %v", err)
	}

	content := readFileForTest(t, "AGENTS.md")
	for _, want := range []string{
		"# TouchDesigner Agent Configuration",
		"td-cli status",
		"- Always use absolute paths: /project1/...",
		"- The connector (TDCliServer) is not to be modified",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("AGENTS.md missing %q", want)
		}
	}
	if strings.Contains(content, "## Auto-Detected Instance") {
		t.Fatalf("AGENTS.md unexpectedly included auto-detected instance block:\n%s", content)
	}
}

func TestWriteAGENTSMDWithHealthIncludesAutoDetectedInstance(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)

	health := &protocol.HealthData{
		Project:   "/project1",
		TDVersion: "2023.12340",
		TDBuild:   "12340",
	}

	if err := writeAGENTSMD(health, 9500); err != nil {
		t.Fatalf("writeAGENTSMD() error = %v", err)
	}

	content := readFileForTest(t, "AGENTS.md")
	for _, want := range []string{
		"## Auto-Detected Instance",
		"- Port: 9500",
		"- Project: /project1",
		"- TD Version: 2023.12340 (build 12340)",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("AGENTS.md missing %q", want)
		}
	}
}

func TestWriteClaudeCommandsCreatesContextCommandFile(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)

	if err := writeClaudeCommands(); err != nil {
		t.Fatalf("writeClaudeCommands() error = %v", err)
	}

	content := readFileForTest(t, filepath.Join(".claude", "commands", "td-context.md"))
	for _, want := range []string{
		"description: Get TouchDesigner project context and connection status",
		"Run: td-cli context",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("td-context.md missing %q", want)
		}
	}
}

func TestClaudeMDContentIncludesHarnessPreview(t *testing.T) {
	t.Parallel()

	health := &protocol.HealthData{
		Project:   "Demo.toe",
		TDVersion: "2023.12345",
		TDBuild:   "12345",
	}

	content := claudeMDContent(health, 9500)
	for _, want := range []string{
		"## Harness Loop",
		"td-cli harness capabilities",
		"td-cli harness apply /project1 --file patch.json",
		"td-cli harness rollback 1712900000-harness",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("claudeMDContent() missing %q", want)
		}
	}
}

func TestAgentsMDContentIncludesHarnessPreview(t *testing.T) {
	t.Parallel()

	content := agentsMDContent(nil, 0)
	for _, want := range []string{
		"## Harness Loop",
		"td-cli harness observe /project1 --depth 2",
		"td-cli harness history",
		"td-cli harness rollback 1712900000-harness",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("agentsMDContent() missing %q", want)
		}
	}
}
