package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUserPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got, want := ExpandUserPath("~/notes.txt"), filepath.Join(home, "notes.txt"); got != want {
		t.Fatalf("ExpandUserPath() = %q, want %q", got, want)
	}
	if got := ExpandUserPath("~another/notes.txt"); got != "~another/notes.txt" {
		t.Fatalf("named-user path changed to %q", got)
	}
}

func TestNormalizeLocalPathResolvesRelativePath(t *testing.T) {
	workingDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDir) })

	got, err := NormalizeLocalPath(filepath.Join("config", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(workingDir, "config", "settings.json")
	if got != want {
		t.Fatalf("NormalizeLocalPath() = %q, want %q", got, want)
	}
}
