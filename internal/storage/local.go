package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"smallFileSync/internal/model"
	"smallFileSync/internal/util"
	"strings"
)

// LocalStore manages local persistence (replacing utools.dbStorage).
type LocalStore struct {
	dir string
}

// NewLocalStore creates a LocalStore using ~/.config/small-filesync/.
func NewLocalStore() (*LocalStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".config", "small-filesync")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{dir: dir}, nil
}

// UID returns the machine UID, creating one if it doesn't exist.
func (s *LocalStore) UID() string {
	path := filepath.Join(s.dir, "uid")
	data, err := os.ReadFile(path)
	if err == nil {
		uid := strings.TrimSpace(string(data))
		if uid != "" {
			return uid
		}
	}
	uid := util.GenerateUID()
	os.WriteFile(path, []byte(uid), 0o644)
	return uid
}

// SaveUID writes the machine UID to disk.
func (s *LocalStore) SaveUID(uid string) error {
	return os.WriteFile(filepath.Join(s.dir, "uid"), []byte(uid), 0o644)
}

// GetSettings reads the application settings.
func (s *LocalStore) GetSettings() model.AppSettings {
	path := filepath.Join(s.dir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return model.DefaultSettings()
	}
	var settings model.AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return model.DefaultSettings()
	}
	return settings
}

// SaveSettings writes the application settings.
func (s *LocalStore) SaveSettings(settings model.AppSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "settings.json"), data, 0o644)
}

// GetLocalDirMap reads the local directory mapping for the given UID.
func (s *LocalStore) GetLocalDirMap(uid string) map[string]string {
	path := filepath.Join(s.dir, "dirmap_"+uid+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}
	}
	return m
}

// SaveLocalDirMap writes the local directory mapping for the given UID.
func (s *LocalStore) SaveLocalDirMap(uid string, m map[string]string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "dirmap_"+uid+".json"), data, 0o644)
}

// GetFileStateMap reads the local file state map for the given UID.
func (s *LocalStore) GetFileStateMap(uid string) map[string]model.FileState {
	path := filepath.Join(s.dir, "filestate_"+uid+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]model.FileState{}
	}
	var m map[string]model.FileState
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]model.FileState{}
	}
	return m
}

// SaveFileStateMap writes the local file state map for the given UID.
func (s *LocalStore) SaveFileStateMap(uid string, m map[string]model.FileState) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "filestate_"+uid+".json"), data, 0o644)
}
