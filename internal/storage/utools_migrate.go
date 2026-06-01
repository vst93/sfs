package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

// UtoolsMigrate reads data from the uTools LevelDB and imports into the local store.
// It returns the imported uid, settings, dirMap, fileStateMap, and error.
func UtoolsMigrate() (uid string, settings model.AppSettings, dirMap map[string]string, fileStateMap map[string]model.FileState, err error) {
	dbPath := findUtoolsDBPath()
	if dbPath == "" {
		return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.db_not_found"))
	}

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		// DB is locked if uTools is running - try to copy it first
		tmpPath := filepath.Join(os.TempDir(), "smallfs-utools-db-copy")
		if copyErr := copyLevelDBDir(dbPath, tmpPath); copyErr != nil {
			return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.db_locked"), err)
		}
		db, err = leveldb.OpenFile(tmpPath, nil)
		if err != nil {
			return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.db_copy_failed"), err)
		}
		defer os.RemoveAll(tmpPath)
	}
	defer db.Close()

	// Scan by-sequence entries for our plugin data
	pluginPrefix := "" // Will auto-detect
	entries := scanBySequence(db)

	// Find the plugin prefix (znp15rl3 or similar)
	for _, e := range entries {
		if id, ok := e["_id"].(string); ok {
			if strings.HasSuffix(id, "/uid") && !strings.HasPrefix(id, "dev_") {
				parts := strings.SplitN(id, "/", 2)
				if len(parts) == 2 {
					pluginPrefix = parts[0]
					break
				}
			}
		}
	}

	if pluginPrefix == "" {
		// Try dev_ prefix
		for _, e := range entries {
			if id, ok := e["_id"].(string); ok {
				if strings.HasSuffix(id, "/uid") && strings.HasPrefix(id, "dev_") {
					parts := strings.SplitN(id, "/", 2)
					if len(parts) == 2 {
						pluginPrefix = parts[0]
						break
					}
				}
			}
		}
	}

	if pluginPrefix == "" {
		return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.plugin_not_found"))
	}

	// Extract uid
	uidKey := pluginPrefix + "/uid"
	uidEntry := findByID(entries, uidKey)
	if uidEntry == nil {
		return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.uid_not_found"))
	}
	if v, ok := uidEntry["value"].(string); ok {
		uid = v
	} else {
		return "", model.DefaultSettings(), nil, nil, fmt.Errorf(i18n.T("migrate.uid_format_error"))
	}

	// Extract settings
	settingsKey := pluginPrefix + "/settings"
	settingsEntry := findByID(entries, settingsKey)
	settings = model.DefaultSettings()
	if settingsEntry != nil {
		if v, ok := settingsEntry["value"].(string); ok {
			var s model.AppSettings
			if json.Unmarshal([]byte(v), &s) == nil {
				settings = s
			}
		} else if v, ok := settingsEntry["value"].(map[string]interface{}); ok {
			b, _ := json.Marshal(v)
			var s model.AppSettings
			if json.Unmarshal(b, &s) == nil {
				settings = s
			}
		}
	}

	// Extract all fileLocalDirMap entries for this plugin
	dirMap = make(map[string]string)
	fileStateMap = make(map[string]model.FileState)
	for _, e := range entries {
		id, _ := e["_id"].(string)
		if !strings.HasPrefix(id, pluginPrefix+"/") {
			continue
		}

		// fileLocalDirMap_<uid>
		if strings.HasPrefix(id, pluginPrefix+"/fileLocalDirMap_") {
			mapUID := strings.TrimPrefix(id, pluginPrefix+"/fileLocalDirMap_")
			if mapUID == uid {
				if val, ok := e["value"].(map[string]interface{}); ok {
					for k, v := range val {
						if s, ok := v.(string); ok {
							dirMap[k] = s
						}
					}
				} else if valStr, ok := e["value"].(string); ok {
					var m map[string]string
					if json.Unmarshal([]byte(valStr), &m) == nil {
						for k, v := range m {
							dirMap[k] = v
						}
					}
				}
			}
		}

		// localDirMap_<uid> (legacy format)
		if strings.HasPrefix(id, pluginPrefix+"/localDirMap_") {
			mapUID := strings.TrimPrefix(id, pluginPrefix+"/localDirMap_")
			if mapUID == uid {
				if val, ok := e["value"].(map[string]interface{}); ok {
					for k, v := range val {
						if s, ok := v.(string); ok {
							dirMap[k] = s
						}
					}
				} else if valStr, ok := e["value"].(string); ok {
					var m map[string]string
					if json.Unmarshal([]byte(valStr), &m) == nil {
						for k, v := range m {
							dirMap[k] = v
						}
					}
				}
			}
		}

		// fileLocalState_<uid>
		if strings.HasPrefix(id, pluginPrefix+"/fileLocalState_") || strings.HasPrefix(id, pluginPrefix+"/localFileState_") {
			var stateUID string
			if strings.HasPrefix(id, pluginPrefix+"/fileLocalState_") {
				stateUID = strings.TrimPrefix(id, pluginPrefix+"/fileLocalState_")
			} else {
				stateUID = strings.TrimPrefix(id, pluginPrefix+"/localFileState_")
			}
			if stateUID == uid {
				if val, ok := e["value"].(map[string]interface{}); ok {
					b, _ := json.Marshal(val)
					json.Unmarshal(b, &fileStateMap)
				} else if valStr, ok := e["value"].(string); ok {
					json.Unmarshal([]byte(valStr), &fileStateMap)
				}
			}
		}
	}

	return uid, settings, dirMap, fileStateMap, nil
}

// findUtoolsDBPath returns the path to the uTools database directory.
func findUtoolsDBPath() string {
	home, _ := os.UserHomeDir()
	var base string
	switch runtime.GOOS {
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support", "uTools", "database")
	case "windows":
		base = filepath.Join(os.Getenv("APPDATA"), "uTools", "database")
	default:
		base = filepath.Join(home, ".config", "utools", "database")
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return ""
	}

	// Find the plugin's database (not "default")
	for _, e := range entries {
		if e.IsDir() && e.Name() != "default" {
			fullPath := filepath.Join(base, e.Name())
			// Quick check if it has MANIFEST file (valid LevelDB)
			matches, _ := filepath.Glob(filepath.Join(fullPath, "MANIFEST-*"))
			if len(matches) > 0 {
				return fullPath
			}
		}
	}
	return ""
}

// scanBySequence scans all entries and returns parsed JSON maps for our plugin.
func scanBySequence(db *leveldb.DB) []map[string]interface{} {
	var results []map[string]interface{}
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		val := iter.Value()

		var entry map[string]interface{}
		if json.Unmarshal(val, &entry) != nil {
			continue
		}
		id, ok := entry["_id"].(string)
		if !ok || id == "" {
			continue
		}
		if strings.Contains(id, "znp15rl3") ||
			strings.Contains(id, "localDirMap") ||
			strings.Contains(id, "fileLocalDirMap") ||
			strings.Contains(id, "localFileState") ||
			strings.Contains(id, "fileLocalState") ||
			strings.Contains(id, "fileList") {
			results = append(results, entry)
		}
	}
	return results
}

// findByID finds the latest (non-deleted) entry with the given _id.
func findByID(entries []map[string]interface{}, id string) map[string]interface{} {
	var latest map[string]interface{}
	for _, e := range entries {
		if eid, ok := e["_id"].(string); ok && eid == id {
			if deleted, ok := e["_deleted"].(bool); ok && deleted {
				continue
			}
			latest = e
		}
	}
	return latest
}

// copyLevelDBDir copies a LevelDB directory to a temp location.
func copyLevelDBDir(src, dst string) error {
	os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

