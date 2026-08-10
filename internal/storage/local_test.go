package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWritePrivateFileRestrictsExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateFile(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("permissions = %o, want 600", got)
	}
}

func TestHasLocalStoreData(t *testing.T) {
	dir := t.TempDir()
	if hasLocalStoreData(dir) {
		t.Fatal("empty directory reported as containing local store data")
	}
	if err := os.WriteFile(filepath.Join(dir, "uid"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !hasLocalStoreData(dir) {
		t.Fatal("uid file was not detected")
	}
}
