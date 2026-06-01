package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"
	"smallFileSync/internal/util"
	"strings"
	"time"
)

// WebDAVStore implements remote storage via WebDAV.
type WebDAVStore struct {
	config model.WebDAVConfig
	client *http.Client
}

// NewWebDAVStore creates a new WebDAV store.
func NewWebDAVStore(config model.WebDAVConfig) *WebDAVStore {
	return &WebDAVStore{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// rootURL returns the root WebDAV URL.
func (s *WebDAVStore) rootURL() string {
	endpoint := strings.TrimRight(strings.TrimSpace(s.config.Endpoint), "/")
	basePath := util.NormalizeStorageBasePath(s.config.BasePath)
	if basePath == "" {
		return endpoint
	}
	return endpoint + "/" + basePath
}

// buildURL builds a URL for a given key.
func (s *WebDAVStore) buildURL(key string) string {
	root := s.rootURL()
	if key == "fileList" {
		return root + "/meta/fileList.json"
	}
	return root + "/data/" + key
}

// authHeader returns the Basic auth header.
func (s *WebDAVStore) authHeader() string {
	cred := s.config.Username + ":" + s.config.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(cred))
}

// doRequest performs an HTTP request.
func (s *WebDAVStore) doRequest(method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", s.authHeader())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return s.client.Do(req)
}

// ensureParentCollections creates parent directories.
func (s *WebDAVStore) ensureParentCollections(targetURL string) error {
	endpoint := strings.TrimRight(strings.TrimSpace(s.config.Endpoint), "/")
	normalized := strings.TrimRight(targetURL, "/")
	if !strings.HasPrefix(normalized, endpoint) {
		return fmt.Errorf("invalid target URL")
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(normalized[len(endpoint):], "/"), "/")
	parts := strings.Split(rel, "/")
	if len(parts) <= 1 {
		return nil
	}
	current := endpoint
	for _, part := range parts[:len(parts)-1] {
		current = current + "/" + part
		// Try PROPFIND first
		resp, err := s.doRequest("PROPFIND", current, nil, "")
		if err == nil && (resp.StatusCode == 200 || resp.StatusCode == 207) {
			resp.Body.Close()
			continue
		}
		if resp != nil {
			resp.Body.Close()
		}
		// Try MKCOL
		resp, err = s.doRequest("MKCOL", current, nil, "")
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 && resp.StatusCode != 405 && resp.StatusCode != 301 && resp.StatusCode != 302 {
			return fmt.Errorf("failed to create directory: HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

// GetFileList reads the remote fileList and migrates legacy records.
func (s *WebDAVStore) GetFileList() ([]model.FileRecord, error) {
	url := s.buildURL("fileList")
	resp, err := s.doRequest("GET", url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("WebDAV read failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return []model.FileRecord{}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("WebDAV read failed: HTTP %d", resp.StatusCode)
	}
	var list []model.FileRecord
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode fileList: %w", err)
	}
	// Migrate legacy records that have FileIds but no FileID
	for i := range list {
		list[i].MigrateFromLegacy()
	}
	return list, nil
}

// SaveFileList writes the fileList to remote.
func (s *WebDAVStore) SaveFileList(list []model.FileRecord) error {
	url := s.buildURL("fileList")
	if err := s.ensureParentCollections(url); err != nil {
		return err
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	resp, err := s.doRequest("PUT", url, strings.NewReader(string(data)), "application/octet-stream")
	if err != nil {
		return fmt.Errorf("WebDAV write failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("WebDAV write failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// GetFile reads a whole file from remote storage.
func (s *WebDAVStore) GetFile(key string) ([]byte, error) {
	var result []byte
	err := withRetry(2, func() error {
		url := s.buildURL("file_" + key)
		resp, e := s.doRequest("GET", url, nil, "")
		if e != nil {
			return fmt.Errorf("WebDAV read failed: %w", e)
		}
		defer resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("WebDAV read failed: HTTP %d", resp.StatusCode)
		}
		body, e := io.ReadAll(resp.Body)
		if e != nil {
			return e
		}
		decoded, e := base64.StdEncoding.DecodeString(string(body))
		if e != nil {
			return fmt.Errorf("failed to decode data: %w", e)
		}
		result = decoded
		return nil
	})
	return result, err
}

// randString returns a random string of lowercase letters and digits.
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// isRetryableError returns true for network/timeout errors that should be retried.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Check for context deadline or common network errors
	if err == context.DeadlineExceeded || err == io.ErrUnexpectedEOF {
		return true
	}
	return strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "reset by peer")
}

// withRetry retries fn up to maxRetries times with exponential backoff.
// Only retries on network/timeout errors, not on HTTP 4xx.
func withRetry(maxRetries int, fn func() error) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)
		}
		err = fn()
		if err == nil || !isRetryableError(err) {
			return err
		}
	}
	return err
}

// SaveFile writes a whole file to remote storage.
func (s *WebDAVStore) SaveFile(key string, data []byte) error {
	err := withRetry(2, func() error {
		url := s.buildURL("file_" + key)
		if e := s.ensureParentCollections(url); e != nil {
			return e
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		resp, e := s.doRequest("PUT", url, strings.NewReader(encoded), "application/octet-stream")
		if e != nil {
			return fmt.Errorf("WebDAV write failed: %w", e)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("WebDAV write failed: HTTP %d", resp.StatusCode)
		}
		return nil
	})
	return err
}

// RemoveFile deletes a file from remote storage.
func (s *WebDAVStore) RemoveFile(key string) error {
	url := s.buildURL("file_" + key)
	resp, err := s.doRequest("DELETE", url, nil, "")
	if err != nil {
		return fmt.Errorf("WebDAV delete failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("WebDAV delete failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// HealthCheck verifies the WebDAV connection using a single probe file.
func (s *WebDAVStore) HealthCheck() (bool, string) {
	if strings.TrimSpace(s.config.Endpoint) == "" {
		return false, i18n.T("webdav.endpoint_required")
	}
	if s.config.Username == "" {
		return false, i18n.T("webdav.username_required")
	}
	probeKey := fmt.Sprintf("__healthcheck_%s__", randString(6))
	probeValue := fmt.Sprintf("%d", time.Now().UnixNano())
	url := s.buildURL(probeKey)
	if err := s.ensureParentCollections(url); err != nil {
		return false, err.Error()
	}
	if err := s.SaveFile(probeKey, []byte(probeValue)); err != nil {
		return false, err.Error()
	}
	got, err := s.GetFile(probeKey)
	if err != nil {
		return false, err.Error()
	}
	_ = s.RemoveFile(probeKey)
	if string(got) != probeValue {
		return false, i18n.T("webdav.verify_failed")
	}
	return true, i18n.T("webdav.connect_success")
}

// BuildFileDataStorageKey builds the storage key for file data.
func BuildFileDataStorageKey(id string) string {
	return "file_" + id
}

// SaveFileDataToStorage reads a local file and uploads it as a single WebDAV object.
// Returns the storage key ("file_<id>") on success.
func SaveFileDataToStorage(store *WebDAVStore, filePath string, id string) (string, error) {
	var data []byte
	var err error
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		data, err = os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf(i18n.T("webdav.read_local_failed"), err)
		}
		if len(data) > 10*1024*1024 {
			return "", fmt.Errorf(i18n.T("webdav.file_too_large"))
		}
		fileKey := "file_" + id
		if err := store.SaveFile(fileKey, data); err != nil {
			if isRetryableError(err) && attempt < 2 {
				continue
			}
			return "", fmt.Errorf(i18n.T("webdav.write_storage"), err)
		}
		return fileKey, nil
	}
	return "", fmt.Errorf(i18n.T("webdav.write_storage"), err)
}

// SaveFileDataToLocal downloads a whole file from remote storage and writes it to a local path.
func SaveFileDataToLocal(store *WebDAVStore, localPath string, fileKey string) error {
	var data []byte
	var err error
	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		data, err = store.GetFile(fileKey)
		if err != nil {
			if isRetryableError(err) && attempt < 2 {
				continue
			}
			return fmt.Errorf(i18n.T("webdav.remote_empty")+": %w", err)
		}
		break
	}
	if data == nil {
		return fmt.Errorf(i18n.T("webdav.remote_empty"))
	}
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf(i18n.T("webdav.create_dir_failed"), err)
	}
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return fmt.Errorf(i18n.T("webdav.write_local"), err)
	}
	return nil
}

// HasStoredFileData checks if the file exists on remote storage.
func HasStoredFileData(store *WebDAVStore, fileKey string) bool {
	data, err := store.GetFile(fileKey)
	return err == nil && data != nil
}

// VerifyStoredFileData verifies remote data matches expected MD5.
func VerifyStoredFileData(store *WebDAVStore, fileKey string, expectedMD5 string) (bool, string) {
	data, err := store.GetFile(fileKey)
	if err != nil || data == nil {
		return false, i18n.T("webdav.readback_empty")
	}
	hash, _ := util.CalculateFileMD5FromBytes(data)
	if expectedMD5 != "" && hash != expectedMD5 {
		return false, i18n.T("webdav.readback_verify")
	}
	return true, ""
}
