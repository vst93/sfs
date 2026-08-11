package update

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "vst93"
	repoName  = "sfs"

	// BrewFormula is the fully-qualified formula used for package-managed updates.
	BrewFormula = "vst93/tap/sfs"
)

// Release represents a GitHub release.
type Release struct {
	TagName    string  `json:"tag_name"`
	Prerelease bool    `json:"prerelease"`
	Draft      bool    `json:"draft"`
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

// PreparedUpdate is a downloaded and extracted update waiting to be installed.
// Cleanup must be called after the update has either been applied or abandoned.
type PreparedUpdate struct {
	BinaryPath string
	TempDir    string
}

// Cleanup removes the staged update files.
func (p PreparedUpdate) Cleanup() {
	if p.TempDir != "" {
		_ = os.RemoveAll(p.TempDir)
	}
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
	exePath, err := CurrentExecutable()
	if err != nil {
		return false
	}
	return IsBrewPath(exePath)
}

// IsBrewPath reports whether a resolved executable path belongs to Homebrew.
func IsBrewPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(lower, "/homebrew/") ||
		strings.Contains(lower, "/.linuxbrew/") ||
		strings.Contains(lower, "/cellar/")
}

// BrewExecutable locates brew, including when sfs is launched with a restricted PATH.
func BrewExecutable() (string, error) {
	if path, err := exec.LookPath("brew"); err == nil {
		return path, nil
	}

	exePath, err := CurrentExecutable()
	if err == nil {
		lower := strings.ToLower(filepath.ToSlash(exePath))
		if idx := strings.Index(lower, "/cellar/"); idx >= 0 {
			candidate := filepath.Join(exePath[:idx], "bin", "brew")
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("brew executable not found")
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
	return evaluateRelease(release, currentVersion, result.IsBrew)
}

func evaluateRelease(release Release, currentVersion string, isBrew bool) CheckResult {
	result := CheckResult{IsBrew: isBrew}

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

	// Homebrew owns its installation. It does not need a platform archive in the
	// release because brew will fetch the formula's configured artifact itself.
	if isBrew {
		result.HasUpdate = true
		return result
	}

	// Find the asset matching the current platform.
	assetName := PlatformAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.HasUpdate = true
			result.DownloadURL = asset.BrowserDownloadURL
			return result
		}
	}

	// No matching binary for this platform — ignore silently.
	return result
}

// ProgressCallback is called during download with bytes downloaded and total size.
// If total is unknown, it will be -1.
type ProgressCallback func(downloaded, total int64)

// PrepareUpdate downloads and extracts the release into a user-owned temp dir.
func PrepareUpdate(downloadURL string, progress ProgressCallback) (PreparedUpdate, error) {
	if strings.TrimSpace(downloadURL) == "" {
		return PreparedUpdate{}, fmt.Errorf("download URL is empty")
	}

	tmpDir, err := os.MkdirTemp("", "sfs-update-*")
	if err != nil {
		return PreparedUpdate{}, fmt.Errorf("create temp dir: %w", err)
	}
	prepared := PreparedUpdate{TempDir: tmpDir}
	failed := true
	defer func() {
		if failed {
			prepared.Cleanup()
		}
	}()

	zipPath := filepath.Join(tmpDir, "update.zip")
	if err := downloadFile(downloadURL, zipPath, progress); err != nil {
		return PreparedUpdate{}, fmt.Errorf("download: %w", err)
	}

	newBinaryPath, err := extractBinary(zipPath, tmpDir)
	if err != nil {
		return PreparedUpdate{}, fmt.Errorf("extract: %w", err)
	}
	prepared.BinaryPath = newBinaryPath
	failed = false
	return prepared, nil
}

// CurrentExecutable returns the resolved path of the running binary.
func CurrentExecutable() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return exePath, nil
}

// NeedsElevation reports whether replacing target requires elevated privileges.
// Directory write access is what controls an atomic rename on Unix.
func NeedsElevation(target string) (bool, error) {
	probe, err := os.CreateTemp(filepath.Dir(target), ".sfs-update-probe-*")
	if err == nil {
		name := probe.Name()
		_ = probe.Close()
		_ = os.Remove(name)
		return false, nil
	}
	if os.IsPermission(err) {
		return true, nil
	}
	return false, fmt.Errorf("check update permission: %w", err)
}

// ApplyCurrentBinary replaces the running executable with a prepared binary.
func ApplyCurrentBinary(source string) error {
	target, err := CurrentExecutable()
	if err != nil {
		return err
	}
	return ReplaceBinary(source, target)
}

// ReplaceBinary atomically replaces target and rolls back if installation fails.
func ReplaceBinary(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open staged binary: %w", err)
	}
	defer input.Close()
	if info, statErr := input.Stat(); statErr != nil || !info.Mode().IsRegular() {
		if statErr != nil {
			return fmt.Errorf("stat staged binary: %w", statErr)
		}
		return fmt.Errorf("staged update is not a regular file")
	}

	targetDir := filepath.Dir(target)
	staged, err := os.CreateTemp(targetDir, ".sfs-update-*")
	if err != nil {
		return fmt.Errorf("stage binary beside target: %w", err)
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)

	if _, err := io.Copy(staged, input); err != nil {
		_ = staged.Close()
		return fmt.Errorf("copy staged binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := staged.Chmod(0755); err != nil {
			_ = staged.Close()
			return fmt.Errorf("chmod staged binary: %w", err)
		}
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return fmt.Errorf("sync staged binary: %w", err)
	}
	if err := staged.Close(); err != nil {
		return fmt.Errorf("close staged binary: %w", err)
	}

	backupPath := target + ".old"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale backup: %w", err)
	}
	if err := os.Rename(target, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}

	if err := os.Rename(stagedPath, target); err != nil {
		_ = os.Rename(backupPath, target)
		return fmt.Errorf("replace binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, 0755); err != nil {
			_ = os.Remove(target)
			_ = os.Rename(backupPath, target)
			return fmt.Errorf("chmod: %w", err)
		}
	}

	_ = os.Remove(backupPath)
	return nil
}

// DownloadAndUpdate downloads and applies an update without elevation.
func DownloadAndUpdate(downloadURL string, progress ProgressCallback) error {
	prepared, err := PrepareUpdate(downloadURL, progress)
	if err != nil {
		return err
	}
	defer prepared.Cleanup()
	return ApplyCurrentBinary(prepared.BinaryPath)
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
