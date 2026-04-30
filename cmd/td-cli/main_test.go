package main

import "testing"

func TestParseExecArgsInlineCode(t *testing.T) {
	opts := parseExecArgs([]string{"print('hello')", "print('world')"})

	if got, want := opts.code, "print('hello') print('world')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
	if opts.filePath != "" {
		t.Fatalf("expected empty filePath, got %q", opts.filePath)
	}
}

func TestParseExecArgsFileAndTrailingCode(t *testing.T) {
	opts := parseExecArgs([]string{"-f", "script.py", "print('ignored?')"})

	if got, want := opts.filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if got, want := opts.code, "print('ignored?')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
}

func TestParseExecArgsFileOnly(t *testing.T) {
	opts := parseExecArgs([]string{"-f", "script.py"})

	if got, want := opts.filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if opts.code != "" {
		t.Fatalf("expected empty code, got %q", opts.code)
	}
}

func TestParseExecArgsInlineCodeBeforeAndAfterFile(t *testing.T) {
	opts := parseExecArgs([]string{"print('before')", "-f", "script.py", "print('after')"})

	if got, want := opts.filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if got, want := opts.code, "print('before') print('after')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
}

func TestParseExecArgsScreenshotOutputAndStrictVerify(t *testing.T) {
	opts := parseExecArgs([]string{
		"-f", "scene.py",
		"--verify", "/project1/scene",
		"--verify-strict",
		"--screenshot", "/project1/scene/out",
		"-o", "/tmp/preview.png",
	})

	if got, want := opts.filePath, "scene.py"; got != want {
		t.Fatalf("filePath = %q, want %q", got, want)
	}
	if got, want := opts.verifyPath, "/project1/scene"; got != want {
		t.Fatalf("verifyPath = %q, want %q", got, want)
	}
	if !opts.verifyStrict {
		t.Fatalf("verifyStrict = false, want true")
	}
	if got, want := opts.screenshotPath, "/project1/scene/out"; got != want {
		t.Fatalf("screenshotPath = %q, want %q", got, want)
	}
	if got, want := opts.screenshotOutput, "/tmp/preview.png"; got != want {
		t.Fatalf("screenshotOutput = %q, want %q", got, want)
	}
}
