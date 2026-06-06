package update

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "vst93"
	repoName  = "sfs"
)

// Release represents a GitHub release.
type Release struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
	Assets     []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckResult holds the result of an update check.
type CheckResult struct {
	HasUpdate     bool
	LatestVersion string
	DownloadURL   string
	IsBrew        bool
	Error         error
}

// CompareVersions compares two semver-like version strings (e.g. "0.2.0" vs "0.1.1").
// Returns 1 if a > b, -1 if a < b, 0 if equal.
func CompareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		aNum, bNum := 0, 0
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &aNum)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bNum)
		}
		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
	}
	return 0
}

// IsBrewInstall checks if the current binary was installed via Homebrew.
func IsBrewInstall() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		realPath = exePath
	}
	lower := strings.ToLower(realPath)
	return strings.Contains(lower, "/homebrew/") ||
		strings.Contains(lower, "/.linuxbrew/") ||
		strings.Contains(lower, "/Cellar/")
}

// PlatformAssetName returns the expected release asset filename for the current platform.
func PlatformAssetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("sfs-%s-%s.zip", osName, arch)
}

// CheckLatestRelease fetches the latest non-prerelease release from GitHub
// and checks whether an update is available.
func CheckLatestRelease(currentVersion string) CheckResult {
	result := CheckResult{
		IsBrew: IsBrewInstall(),
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	// 404 means no releases exist at all — silently ignore.
	if resp.StatusCode == http.StatusNotFound {
		return result
	}
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		return result
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		result.Error = err
		return result
	}

	// Safety: skip pre-releases and drafts (API /latest should already filter, but be explicit).
	if release.Prerelease || release.Draft {
		return result
	}

	// Strip leading 'v' if present.
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	result.LatestVersion = latestVersion

	// Compare versions — only update if strictly newer.
	if CompareVersions(latestVersion, currentVersion) <= 0 {
		return result
	}

	// Find the asset matching the current platform.
	assetName := PlatformAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.HasUpdate = true
			// For brew installs, don't set DownloadURL — user should use brew upgrade
			if !result.IsBrew {
				result.DownloadURL = asset.BrowserDownloadURL
			}
			return result
		}
	}

	// No matching binary for this platform — ignore silently.
	return result
}

// ProgressCallback is called during download with bytes downloaded and total size.
// If total is unknown, it will be -1.
type ProgressCallback func(downloaded, total int64)

// DownloadAndUpdate downloads the release zip for the current platform and
// replaces the running binary. On Unix it renames the old binary to .old first.
// The progress callback is optional and can be nil.
func DownloadAndUpdate(downloadURL string, progress ProgressCallback) error {
	// 1. Create temp directory for the download.
	tmpDir, err := os.MkdirTemp("", "sfs-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Download the zip file.
	zipPath := filepath.Join(tmpDir, "update.zip")
	if err := downloadFile(downloadURL, zipPath, progress); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// 3. Extract the binary from the zip.
	newBinaryPath, err := extractBinary(zipPath, tmpDir)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// 4. Locate the current executable.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	// 5. Backup the current binary (rename to .old).
	backupPath := exePath + ".old"
	_ = os.Remove(backupPath) // remove stale backup if any
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}

	// 6. Move the new binary into place.
	if err := os.Rename(newBinaryPath, exePath); err != nil {
		// Attempt rollback.
		_ = os.Rename(backupPath, exePath)
		return fmt.Errorf("replace binary: %w", err)
	}

	// 7. Ensure executable permission on Unix.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(exePath, 0755); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
	}

	// 8. Best-effort cleanup of the .old file after a short delay.
	go func() {
		time.Sleep(5 * time.Second)
		_ = os.Remove(backupPath)
	}()

	return nil
}

// downloadFile downloads url to the given local path.
func downloadFile(url, dest string, progress ProgressCallback) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if progress != nil {
		total := resp.ContentLength
		reader := &progressReader{Reader: resp.Body, total: total, callback: progress}
		_, err = io.Copy(f, reader)
	} else {
		_, err = io.Copy(f, resp.Body)
	}
	return err
}

// progressReader wraps an io.Reader and reports progress.
type progressReader struct {
	Reader   io.Reader
	total    int64
	current  int64
	callback ProgressCallback
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		r.current += int64(n)
		r.callback(r.current, r.total)
	}
	return n, err
}

// extractBinary finds and extracts the sfs binary from a zip archive.
func extractBinary(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	binaryName := "sfs"
	if runtime.GOOS == "windows" {
		binaryName = "sfs.exe"
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		// Match exact binary name or prefixed name like sfs-darwin-arm64.
		if base == binaryName || strings.HasPrefix(base, "sfs-") {
			dest := filepath.Join(destDir, binaryName)
			if err := extractFile(f, dest); err != nil {
				return "", err
			}
			if runtime.GOOS != "windows" {
				_ = os.Chmod(dest, 0755)
			}
			return dest, nil
		}
	}

	return "", fmt.Errorf("binary %s not found inside zip", binaryName)
}

func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
