package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"smallFileSync/internal/model"
	"smallFileSync/internal/storage"
)

func newTestLocalStore(t *testing.T) *storage.LocalStore {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(root, "AppData", "Roaming"))
	}
	store, err := storage.NewLocalStore()
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestHandleSetDirAllowsUnbindAndClearsBaseline(t *testing.T) {
	store := newTestLocalStore(t)
	server := &Server{
		localStore:  store,
		uid:         "test-machine",
		fileList:    []model.FileRecord{{ID: "file-1", FileName: "settings.json"}},
		localDirMap: map[string]string{"file-1": t.TempDir()},
		stateMap:    map[string]model.FileState{"file-1": {MD5: "old-baseline"}},
	}
	if err := store.SaveLocalDirMap(server.uid, server.localDirMap); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFileStateMap(server.uid, server.stateMap); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/files/dir", bytes.NewBufferString(`{"id":"file-1","dir":""}`))
	recorder := httptest.NewRecorder()
	server.handleSetDir(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, ok := server.localDirMap["file-1"]; ok {
		t.Fatalf("directory mapping was not removed: %#v", server.localDirMap)
	}
	if _, ok := server.stateMap["file-1"]; ok {
		t.Fatalf("sync baseline was not removed: %#v", server.stateMap)
	}
	if _, ok := store.GetLocalDirMap(server.uid)["file-1"]; ok {
		t.Fatal("persisted directory mapping was not removed")
	}
	if _, ok := store.GetFileStateMap(server.uid)["file-1"]; ok {
		t.Fatal("persisted sync baseline was not removed")
	}
}

func TestHandleSetDirRejectsUnknownRecord(t *testing.T) {
	server := &Server{
		localStore:  newTestLocalStore(t),
		uid:         "test-machine",
		localDirMap: map[string]string{},
		stateMap:    map[string]model.FileState{},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/files/dir", bytes.NewBufferString(`{"id":"missing","dir":""}`))
	recorder := httptest.NewRecorder()
	server.handleSetDir(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestSetLocalDirClearsBaselineWhenPathChanges(t *testing.T) {
	store := newTestLocalStore(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()
	server := &Server{
		localStore:  store,
		uid:         "test-machine",
		localDirMap: map[string]string{"file-1": oldDir},
		stateMap:    map[string]model.FileState{"file-1": {MD5: "old-baseline"}},
	}
	if err := server.setLocalDirLocked("file-1", newDir); err != nil {
		t.Fatal(err)
	}
	if server.localDirMap["file-1"] != newDir {
		t.Fatalf("directory = %q, want %q", server.localDirMap["file-1"], newDir)
	}
	if _, ok := server.stateMap["file-1"]; ok {
		t.Fatal("old sync baseline survived a path change")
	}
}
