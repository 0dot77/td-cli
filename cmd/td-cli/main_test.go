package main

import "testing"

func TestParseExecArgsInlineCode(t *testing.T) {
	code, filePath := parseExecArgs([]string{"print('hello')", "print('world')"})

	if got, want := code, "print('hello') print('world')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
	if filePath != "" {
		t.Fatalf("expected empty filePath, got %q", filePath)
	}
}

func TestParseExecArgsFileAndTrailingCode(t *testing.T) {
	code, filePath := parseExecArgs([]string{"-f", "script.py", "print('ignored?')"})

	if got, want := filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if got, want := code, "print('ignored?')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
}

func TestParseExecArgsFileOnly(t *testing.T) {
	code, filePath := parseExecArgs([]string{"-f", "script.py"})

	if got, want := filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if code != "" {
		t.Fatalf("expected empty code, got %q", code)
	}
}

func TestParseExecArgsInlineCodeBeforeAndAfterFile(t *testing.T) {
	code, filePath := parseExecArgs([]string{"print('before')", "-f", "script.py", "print('after')"})

	if got, want := filePath, "script.py"; got != want {
		t.Fatalf("parseExecArgs filePath = %q, want %q", got, want)
	}
	if got, want := code, "print('before') print('after')"; got != want {
		t.Fatalf("parseExecArgs code = %q, want %q", got, want)
	}
}
