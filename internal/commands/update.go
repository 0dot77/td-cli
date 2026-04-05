package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	githubRepo   = "td-cli/td-cli"
	updateAPIURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
)

// GitHubRelease is the subset of GitHub's release API we need.
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset is a release asset.
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update checks GitHub Releases for a newer version and performs a transactional binary replacement.
func Update(currentVersion string, jsonOutput bool) error {
	fmt.Println("Checking for updates...")

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", updateAPIURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
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
		out, _ := json.MarshalIndent(map[string]interface{}{
			"currentVersion": currentVersion,
			"latestVersion":  latestVersion,
			"upToDate":       latestVersion == currentVersion,
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if latestVersion == currentVersion {
		fmt.Printf("Already up to date (v%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("New version available: v%s (current: v%s)\n", latestVersion, currentVersion)

	// Find the right asset for this OS/arch
	assetName := getAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (expected %s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	fmt.Printf("Downloading %s...\n", assetName)

	// Download the new binary
	dlResp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", dlResp.StatusCode)
	}

	newBinary, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read download: %w", err)
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

func getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	switch {
	case os == "windows" && arch == "amd64":
		return "td-cli-windows-amd64.exe"
	case os == "darwin" && arch == "amd64":
		return "td-cli-darwin-amd64"
	case os == "darwin" && arch == "arm64":
		return "td-cli-darwin-arm64"
	case os == "linux" && arch == "amd64":
		return "td-cli-linux-amd64"
	default:
		return fmt.Sprintf("td-cli-%s-%s", os, arch)
	}
}
