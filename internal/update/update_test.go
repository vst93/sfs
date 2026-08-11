package update

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.1.9", 1},
		{"1.0", "1.0.0", 0},
		{"0.9.9", "1.0.0", -1},
		{"1.10.0", "1.9.9", 1},
	}
	for _, test := range tests {
		if got := CompareVersions(test.a, test.b); got != test.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
		}
	}
}

func TestIsBrewPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/Cellar/sfs/1.2.0/bin/sfs", true},
		{"/usr/local/Cellar/sfs/1.2.0/bin/sfs", true},
		{"/home/linuxbrew/.linuxbrew/Cellar/sfs/1.2.0/bin/sfs", true},
		{"/usr/local/bin/sfs", false},
		{"/home/user/bin/sfs", false},
	}
	for _, test := range tests {
		if got := IsBrewPath(test.path); got != test.want {
			t.Errorf("IsBrewPath(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestEvaluateReleaseForBrewDoesNotRequirePlatformAsset(t *testing.T) {
	release := Release{TagName: "v1.2.0"}
	result := evaluateRelease(release, "1.1.0", true)
	if !result.HasUpdate {
		t.Fatal("Homebrew update was hidden because the release had no platform archive")
	}
	if result.DownloadURL != "" {
		t.Fatalf("Homebrew update should not use a release archive: %q", result.DownloadURL)
	}
}

func TestEvaluateReleaseForDirectInstallRequiresPlatformAsset(t *testing.T) {
	release := Release{TagName: "v1.2.0"}
	if result := evaluateRelease(release, "1.1.0", false); result.HasUpdate {
		t.Fatal("direct update was offered without a matching platform archive")
	}

	release.Assets = []Asset{{Name: PlatformAssetName(), BrowserDownloadURL: "https://example.test/sfs.zip"}}
	result := evaluateRelease(release, "1.1.0", false)
	if !result.HasUpdate || result.DownloadURL == "" {
		t.Fatalf("matching direct update was not selected: %+v", result)
	}
}

func TestNeedsElevationWritableDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sfs")
	needsElevation, err := NeedsElevation(target)
	if err != nil {
		t.Fatal(err)
	}
	if needsElevation {
		t.Fatal("writable directory unexpectedly requires elevation")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "downloaded-sfs")
	target := filepath.Join(dir, "sfs")
	if err := os.WriteFile(source, []byte("new binary"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceBinary(source, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new binary" {
		t.Fatalf("target content = %q, want new binary", got)
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Fatalf("backup was not cleaned up: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0755 {
			t.Fatalf("target mode = %o, want 755", info.Mode().Perm())
		}
	}
}
