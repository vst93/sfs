package ui

import (
	"errors"
	"testing"

	"smallFileSync/internal/model"
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
