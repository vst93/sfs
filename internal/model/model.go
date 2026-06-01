package model

import "time"

// AppName is the short display name.
const AppName = "SFS"

// AppFullName is the full display name shown in titles.
const AppFullName = "SFS — SmallFileSync"

// AppVersion is the current application version.
const AppVersion = "0.1.0"

// FileRecord represents a synced file entry (compatible with legacy uTools plugin).
type FileRecord struct {
	ID              string  `json:"id"`
	FileName        string  `json:"fileName"`
	Note            string  `json:"note"`
	FileMD5         string  `json:"fileMd5"`
	LastChangeTime  int64   `json:"lastChangeTime"`
	LastUploadTime  int64   `json:"lastUploadTime"`
	LastUploadUser  string  `json:"lastUploadUser"`
	Size            float64 `json:"size"` // KB
	FileID          string  `json:"fileId"`
	LocalDirPending bool    `json:"localDirPending,omitempty"`
}

// NormalizeFileRecord sanitizes a file record (matching legacy behavior).
func NormalizeFileRecord(r FileRecord) FileRecord {
	if r.Note == "" {
		r.Note = ""
	}
	if r.LastUploadUser == "" {
		r.LastUploadUser = ""
	}
	if r.FileMD5 == "" {
		r.FileMD5 = ""
	}
	if r.FileID == "" {
		r.FileID = ""
	}
	// When never uploaded, clear derived fields (matches legacy behavior)
	if r.LastUploadTime == 0 {
		r.LastUploadUser = ""
		r.FileMD5 = ""
		r.Size = 0
		r.FileID = ""
	}
	return r
}

// WebDAVConfig holds WebDAV connection parameters.
type WebDAVConfig struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	BasePath string `json:"basePath"`
}

// StorageConfig holds the storage backend configuration.
type StorageConfig struct {
	Type   string      `json:"type"`
	WebDAV WebDAVConfig `json:"webdav"`
}

// AppSettings holds all application settings.
type AppSettings struct {
	Language string         `json:"language,omitempty"`
	AutoSync bool          `json:"autoSync"`
	Storage  StorageConfig `json:"storage"`
}

// DefaultSettings returns the default application settings.
func DefaultSettings() AppSettings {
	return AppSettings{
		AutoSync: false,
		Storage: StorageConfig{
			Type: "webdav",
			WebDAV: WebDAVConfig{
				Endpoint: "",
				Username: "",
				Password: "",
				BasePath: "small-file-sync",
			},
		},
	}
}

// FileState represents the local baseline state of a synced file.
type FileState struct {
	MD5          string `json:"md5"`
	MtimeMs      int64  `json:"mtimeMs"`
	LastSyncTime int64  `json:"lastSyncTime"`
}

// FileStatus describes the current sync status of a file.
type FileStatus struct {
	Key    string // matched, pending_upload, download, conflict, missing, unbound, initial_upload, pending_binding
	Text   string // Display text
	Detail string // Explanation
}

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	IsAuto    bool
	StartedAt time.Time
	Summary   SyncSummary
	Details   []SyncDetail
}

// SyncSummary holds aggregate sync counts.
type SyncSummary struct {
	Checked    int
	Uploaded   int
	Downloaded int
	Skipped    int
	Failed     int
	Unbound    int
}

// SyncDetail describes a single file's sync outcome.
type SyncDetail struct {
	FileName string
	Action   string
	Status   string // 成功 or 失败
	Reason   string
}
