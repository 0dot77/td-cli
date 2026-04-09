package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal", a: "0.1.0", b: "0.1.0", want: 0},
		{name: "patch less", a: "0.1.0", b: "0.1.1", want: -1},
		{name: "minor greater", a: "0.2.0", b: "0.1.9", want: 1},
		{name: "missing patch treated as zero", a: "1.2", b: "1.2.0", want: 0},
		{name: "major less", a: "1.9.9", b: "2.0.0", want: -1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := compareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVerifyChecksumSuccess(t *testing.T) {
	t.Parallel()

	binary := []byte("td-cli-binary")
	sum := sha256.Sum256(binary)
	expected := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  td-cli-darwin-arm64\n", expected)
	}))
	defer server.Close()

	if err := verifyChecksum(server.Client(), server.URL, "td-cli-darwin-arm64", binary); err != nil {
		t.Fatalf("verifyChecksum returned error: %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	t.Parallel()

	binary := []byte("td-cli-binary")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "deadbeef  td-cli-darwin-arm64")
	}))
	defer server.Close()

	err := verifyChecksum(server.Client(), server.URL, "td-cli-darwin-arm64", binary)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestVerifyChecksumMissingEntry(t *testing.T) {
	t.Parallel()

	binary := []byte("td-cli-binary")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "deadbeef  some-other-file")
	}))
	defer server.Close()

	err := verifyChecksum(server.Client(), server.URL, "td-cli-darwin-arm64", binary)
	if err == nil {
		t.Fatal("expected missing checksum entry error")
	}
}
