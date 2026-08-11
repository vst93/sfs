package ui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"smallFileSync/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFileListRefreshIgnoresStaleResult(t *testing.T) {
	app := &App{
		refreshSeq:     2,
		fileList:       []model.FileRecord{{ID: "current", FileName: "current.txt"}},
		localDirMap:    map[string]string{"current": "/current"},
		localStateMap:  map[string]model.FileState{},
		probeCache:     map[string]localProbe{},
		fileStateCache: map[string]model.FileStatus{},
		updateDone:     true,
	}

	_, cmd := app.Update(fileListRefreshMsg{
		seq:         1,
		fileList:    []model.FileRecord{{ID: "stale", FileName: "stale.txt"}},
		localDirMap: map[string]string{"stale": "/stale"},
	})
	if cmd != nil {
		t.Fatal("stale refresh returned a command")
	}
	if len(app.fileList) != 1 || app.fileList[0].ID != "current" {
		t.Fatalf("stale refresh replaced current list: %#v", app.fileList)
	}
	if app.localDirMap["current"] != "/current" {
		t.Fatalf("stale refresh replaced local directory map: %#v", app.localDirMap)
	}
}

func TestFileListRefreshKeepsListOnRemoteError(t *testing.T) {
	app := &App{
		refreshSeq:     1,
		fileList:       []model.FileRecord{{ID: "current", FileName: "current.txt"}},
		localDirMap:    map[string]string{"current": "/old"},
		localStateMap:  map[string]model.FileState{},
		probeCache:     map[string]localProbe{},
		fileStateCache: map[string]model.FileStatus{},
		updateDone:     true,
	}

	_, cmd := app.Update(fileListRefreshMsg{
		seq:           1,
		localDirMap:   map[string]string{"current": "/new"},
		localStateMap: map[string]model.FileState{},
		err:           errors.New("offline"),
	})
	if cmd == nil {
		t.Fatal("remote error did not return a toast command")
	}
	if len(app.fileList) != 1 || app.fileList[0].ID != "current" {
		t.Fatalf("remote error cleared the current list: %#v", app.fileList)
	}
	if app.localDirMap["current"] != "/new" {
		t.Fatalf("local mapping was not refreshed: %#v", app.localDirMap)
	}
}

func TestSyncWorkerAppliesResultOnUpdateLoop(t *testing.T) {
	app := &App{
		state:          viewFileList,
		fileList:       []model.FileRecord{{ID: "unbound", FileName: "settings.json"}},
		localDirMap:    map[string]string{},
		localStateMap:  map[string]model.FileState{},
		probeCache:     map[string]localProbe{},
		fileStateCache: map[string]model.FileStatus{},
	}

	start := app.runSync("", "", true)
	if !app.syncing || app.lastSyncResult == nil {
		t.Fatal("runSync did not initialize sync state on the update loop")
	}
	stepMsg, ok := start().(syncStepMsg)
	if !ok {
		t.Fatalf("start command returned %T, want syncStepMsg", start())
	}
	_, step := app.Update(stepMsg)
	resultMsg, ok := step().(syncStepDoneMsg)
	if !ok {
		t.Fatalf("step command returned %T, want syncStepDoneMsg", step())
	}
	if app.lastSyncResult.Summary.Checked != 0 {
		t.Fatal("background worker mutated sync summary before its result was applied")
	}
	app.Update(resultMsg)
	if app.lastSyncResult.Summary.Checked != 1 || app.lastSyncResult.Summary.Unbound != 1 {
		t.Fatalf("unexpected summary: %#v", app.lastSyncResult.Summary)
	}
}

func TestConfirmedSyncDoesNotStartBeforeConfirmation(t *testing.T) {
	app := &App{
		fileList:       []model.FileRecord{{ID: "file-1", FileName: "settings.json"}},
		localDirMap:    map[string]string{"file-1": t.TempDir()},
		localStateMap:  map[string]model.FileState{},
		probeCache:     map[string]localProbe{},
		fileStateCache: map[string]model.FileStatus{},
	}
	app.handleForceDownload()
	if app.syncing {
		t.Fatal("sync started while the confirmation dialog was still open")
	}
	msg, ok := app.confirmAction().(startSyncMsg)
	if !ok {
		t.Fatalf("confirmation returned %T, want startSyncMsg", app.confirmAction())
	}
	app.Update(msg)
	if !app.syncing {
		t.Fatal("sync did not start after confirmation was applied")
	}
}

func TestAddStartsAtCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDir) })

	app := &App{
		settings: model.AppSettings{Storage: model.StorageConfig{WebDAV: model.WebDAVConfig{Endpoint: "https://x", Username: "u", Password: "p"}}},
		addFileInputs: []textinput.Model{
			textinput.New(),
			textinput.New(),
		},
	}
	_, cmd := app.handleFileListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatalf("unexpected command: %v", cmd)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := wd + string(filepath.Separator)
	if got := app.addFileInputs[0].Value(); got != want {
		t.Fatalf("add path default = %q, want %q", got, want)
	}
}

func TestPathCompletionCyclesSiblings(t *testing.T) {
	dir := t.TempDir()
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDir) })
	for _, name := range []string{"alpha.json", "beta.json", "beta.log"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	app := &App{addFileInputs: []textinput.Model{textinput.New(), textinput.New()}}
	app.addFileInputs[0].SetValue("b")
	app.completePathInput()
	if got := app.addFileInputs[0].Value(); got != "beta." {
		t.Fatalf("first completion = %q, want shared prefix %q", got, "beta.")
	}
	assertCursorAtEnd(t, app.addFileInputs[0])
	app.completePathInput()
	if got := app.addFileInputs[0].Value(); got != "beta.json" {
		t.Fatalf("second completion = %q, want beta.json", got)
	}
	assertCursorAtEnd(t, app.addFileInputs[0])
	app.completePathInput()
	if got := app.addFileInputs[0].Value(); got != "beta.log" {
		t.Fatalf("third completion = %q, want beta.log", got)
	}
	assertCursorAtEnd(t, app.addFileInputs[0])
}

func TestPathCompletionAppendsDirectorySeparator(t *testing.T) {
	dir := t.TempDir()
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDir) })
	if err := os.Mkdir(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	app := &App{addFileInputs: []textinput.Model{textinput.New(), textinput.New()}}
	app.addFileInputs[0].SetValue("notes")
	app.completePathInput()
	want := "notes" + string(filepath.Separator)
	if got := app.addFileInputs[0].Value(); got != want {
		t.Fatalf("directory completion = %q, want %q", got, want)
	}
	assertCursorAtEnd(t, app.addFileInputs[0])
}

func assertCursorAtEnd(t *testing.T, m textinput.Model) {
	t.Helper()
	want := utf8.RuneCountInString(m.Value())
	if got := m.Position(); got != want {
		t.Fatalf("cursor position = %d, want end %d (%q)", got, want, m.Value())
	}
}

func TestAddNoteTabIsInert(t *testing.T) {
	app := &App{
		addFileInputs: []textinput.Model{textinput.New(), textinput.New()},
		addFileFocus:  1,
	}
	app.addFileInputs[1].SetValue("note text")
	_, cmd := app.handleAddFileKey(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("tab in note field returned a command: %v", cmd)
	}
	if app.addFileFocus != 1 {
		t.Fatalf("tab moved focus to %d, want 1", app.addFileFocus)
	}
	if got := app.addFileInputs[1].Value(); got != "note text" {
		t.Fatalf("tab modified note value to %q", got)
	}
}
