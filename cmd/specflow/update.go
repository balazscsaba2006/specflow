package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	repoOwner = "balazscsaba2006"
	repoName  = "specflow"
)

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update specflow to the latest release",
		Long:  "Downloads the latest release from GitHub and replaces the current binary.",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Find the current binary path.
			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding executable path: %w", err)
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return fmt.Errorf("resolving symlinks: %w", err)
			}

			// Fetch latest release info.
			release, err := fetchLatestRelease()
			if err != nil {
				return fmt.Errorf("fetching latest release: %w", err)
			}

			latestVersion := strings.TrimPrefix(release.TagName, "v")
			if latestVersion == version {
				fmt.Printf("Already up to date (v%s)\n", version)
				return nil
			}

			// Find the matching asset.
			assetName := buildAssetName(latestVersion)
			var downloadURL string
			for _, a := range release.Assets {
				if a.Name == assetName {
					downloadURL = a.BrowserDownloadURL
					break
				}
			}
			if downloadURL == "" {
				return fmt.Errorf("no release asset found for %s/%s (expected %s)", runtime.GOOS, runtime.GOARCH, assetName)
			}

			fmt.Printf("Updating v%s → v%s\n", version, latestVersion)
			fmt.Printf("Downloading %s...\n", assetName)

			// Download the archive.
			archiveData, err := downloadAsset(downloadURL)
			if err != nil {
				return fmt.Errorf("downloading release: %w", err)
			}

			// Extract the binary from the archive.
			binary, err := extractBinary(assetName, archiveData)
			if err != nil {
				return fmt.Errorf("extracting binary: %w", err)
			}

			// Replace the current binary atomically.
			if err := replaceBinary(execPath, binary); err != nil {
				return fmt.Errorf("replacing binary: %w", err)
			}

			fmt.Printf("Updated to v%s\n", latestVersion)
			fmt.Println()
			fmt.Println("Run 'specflow sync' in your projects to update the Claude Code skill.")
			return nil
		},
	}
}

func fetchLatestRelease() (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url) //nolint:gosec // static URL
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func buildAssetName(ver string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("specflow_%s_%s_%s.%s", ver, runtime.GOOS, runtime.GOARCH, ext)
}

func downloadAsset(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:gosec // URL from GitHub API
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func extractBinary(assetName string, data []byte) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(data)
	}
	return extractFromTarGz(data)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == "specflow" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("specflow binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == "specflow" || base == "specflow.exe" {
			return readZipFile(f)
		}
	}
	return nil, fmt.Errorf("specflow binary not found in archive")
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func replaceBinary(execPath string, newBinary []byte) error {
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "specflow-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(newBinary); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath) //nolint:gosec // tmpPath from os.CreateTemp, not user input
		return err
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil { //nolint:gosec // binary must be executable
		os.Remove(tmpPath) //nolint:gosec // tmpPath from os.CreateTemp
		return err
	}

	if err := os.Rename(tmpPath, execPath); err != nil { //nolint:gosec // execPath from os.Executable
		os.Remove(tmpPath) //nolint:gosec // tmpPath from os.CreateTemp
		return err
	}

	return nil
}
