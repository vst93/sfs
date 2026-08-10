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

// NewLocalStore creates a LocalStore in the platform's user config directory.
// Older releases always used ~/.config; keep using that directory when it
// already contains SFS data so upgrades do not appear to lose configuration.
func NewLocalStore() (*LocalStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	legacyDir := filepath.Join(home, ".config", "small-filesync")
	configRoot, err := os.UserConfigDir()
	if err != nil || configRoot == "" {
		configRoot = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configRoot, "small-filesync")
	if filepath.Clean(dir) != filepath.Clean(legacyDir) && !hasLocalStoreData(dir) && hasLocalStoreData(legacyDir) {
		dir = legacyDir
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if err := os.Chmod(filepath.Join(dir, entry.Name()), 0o600); err != nil {
				return nil, err
			}
		}
	}
	return &LocalStore{dir: dir}, nil
}

func hasLocalStoreData(dir string) bool {
	for _, name := range []string{"settings.json", "uid"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// Dir returns the directory containing local SFS data.
func (s *LocalStore) Dir() string {
	return s.dir
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
	_ = writePrivateFile(path, []byte(uid))
	return uid
}

// SaveUID writes the machine UID to disk.
func (s *LocalStore) SaveUID(uid string) error {
	return writePrivateFile(filepath.Join(s.dir, "uid"), []byte(uid))
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
	return writePrivateFile(filepath.Join(s.dir, "settings.json"), data)
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
	return writePrivateFile(filepath.Join(s.dir, "dirmap_"+uid+".json"), data)
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
	return writePrivateFile(filepath.Join(s.dir, "filestate_"+uid+".json"), data)
}

func writePrivateFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
