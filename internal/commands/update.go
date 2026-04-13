package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	githubRepo       = "0dot77/td-cli"
	updateAPIURL     = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	maxBinarySize    = 100 * 1024 * 1024 // 100 MB max download
	githubDomainHTTP = "https://github.com/"
	githubDomainDL   = "https://objects.githubusercontent.com/"
)

// GitHubRelease is the subset of GitHub's release API we need.
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset is a release asset.
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update checks GitHub Releases for a newer version and performs a transactional binary replacement.
func Update(currentVersion string, jsonOutput bool) error {
	fmt.Println("Checking for updates...")

	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", updateAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	if jsonOutput {
		cmp := compareVersions(currentVersion, latestVersion)
		out, _ := json.MarshalIndent(map[string]interface{}{
			"currentVersion": currentVersion,
			"latestVersion":  latestVersion,
			"upToDate":       cmp >= 0,
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if compareVersions(currentVersion, latestVersion) >= 0 {
		fmt.Printf("Already up to date (v%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("New version available: v%s (current: v%s)\n", latestVersion, currentVersion)

	// Find the right asset for this OS/arch
	assetName := getAssetName()
	var downloadURL string
	var checksumAssetURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumAssetURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (expected %s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	// Validate download URL points to GitHub
	if !strings.HasPrefix(downloadURL, githubDomainHTTP) && !strings.HasPrefix(downloadURL, githubDomainDL) {
		return fmt.Errorf("download URL does not point to GitHub: %s", downloadURL)
	}

	fmt.Printf("Downloading %s...\n", assetName)

	// Download the new binary with size limit
	dlResp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", dlResp.StatusCode)
	}

	newBinary, err := io.ReadAll(io.LimitReader(dlResp.Body, maxBinarySize))
	if err != nil {
		return fmt.Errorf("failed to read download: %w", err)
	}

	// Verify checksum if checksums.txt is available
	if checksumAssetURL != "" {
		if err := verifyChecksum(httpClient, checksumAssetURL, assetName, newBinary); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("Checksum verified.")
	}

	// Transactional replacement: rename current -> .bak, write new, cleanup
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	backupPath := execPath + ".bak"

	// Remove any leftover backup
	os.Remove(backupPath)

	// Rename current binary to backup
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Write new binary
	if err := os.WriteFile(execPath, newBinary, 0755); err != nil {
		// Restore from backup
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	fmt.Printf("Updated to v%s\n", latestVersion)
	return nil
}

// verifyChecksum downloads checksums.txt and verifies the binary's SHA256.
func verifyChecksum(httpClient *http.Client, checksumURL, assetName string, binary []byte) error {
	resp, err := httpClient.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse checksums.txt: each line is "sha256hash  filename"
	actualHash := sha256.Sum256(binary)
	actualHex := hex.EncodeToString(actualHash[:])

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			if !strings.EqualFold(parts[0], actualHex) {
				return fmt.Errorf("expected %s, got %s", parts[0], actualHex)
			}
			return nil
		}
	}

	return fmt.Errorf("no checksum entry found for %s", assetName)
}

// compareVersions compares two semver-like version strings (e.g., "0.1.0" vs "0.2.0").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func getAssetName() string {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	switch {
	case goos == "windows" && arch == "amd64":
		return "td-cli-windows-amd64.exe"
	case goos == "darwin" && arch == "amd64":
		return "td-cli-darwin-amd64"
	case goos == "darwin" && arch == "arm64":
		return "td-cli-darwin-arm64"
	case goos == "linux" && arch == "amd64":
		return "td-cli-linux-amd64"
	default:
		return fmt.Sprintf("td-cli-%s-%s", goos, arch)
	}
}
