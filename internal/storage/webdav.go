package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// GetFileList reads the remote fileList.
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

// GetFile reads a single file from remote (Base64-encoded).
func (s *WebDAVStore) GetFile(key string) ([]byte, error) {
	url := s.buildURL("file_" + key)
	resp, err := s.doRequest("GET", url, nil, "")
	if err != nil {
		return nil, fmt.Errorf("WebDAV read file failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("WebDAV read file failed: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode file data: %w", err)
	}
	return decoded, nil
}

// SaveFile writes a single file to remote (Base64-encoded).
func (s *WebDAVStore) SaveFile(key string, data []byte) error {
	url := s.buildURL("file_" + key)
	if err := s.ensureParentCollections(url); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	resp, err := s.doRequest("PUT", url, strings.NewReader(encoded), "application/octet-stream")
	if err != nil {
		return fmt.Errorf("WebDAV write file failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("WebDAV write file failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// RemoveFile deletes a single file from remote.
func (s *WebDAVStore) RemoveFile(key string) error {
	url := s.buildURL("file_" + key)
	resp, err := s.doRequest("DELETE", url, nil, "")
	if err != nil {
		return fmt.Errorf("WebDAV delete file failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("WebDAV delete file failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

// HealthCheck verifies the WebDAV connection.
func (s *WebDAVStore) HealthCheck() (bool, string) {
	if strings.TrimSpace(s.config.Endpoint) == "" {
		return false, i18n.T("webdav.endpoint_required")
	}
	if s.config.Username == "" {
		return false, i18n.T("webdav.username_required")
	}
	probeKey := "__healthcheck__"
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

// SaveFileDataToStorage saves a local file to remote storage.
// Returns the file storage key, or an error.
func SaveFileDataToStorage(store *WebDAVStore, filePath string, id string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf(i18n.T("webdav.read_local_failed"), err)
	}
	if len(data) > 10*1024*1024 {
		return "", fmt.Errorf(i18n.T("webdav.file_too_large"))
	}
	fileKey := "file_" + id
	if err := store.SaveFile(fileKey, data); err != nil {
		return "", fmt.Errorf(i18n.T("webdav.write_storage"), err)
	}
	return fileKey, nil
}

// SaveFileDataToLocal downloads a file from remote storage to a local path.
func SaveFileDataToLocal(store *WebDAVStore, localPath string, fileKey string) error {
	data, err := store.GetFile(fileKey)
	if err != nil {
		return fmt.Errorf(i18n.T("webdav.remote_empty")+": %w", err)
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

// HasStoredFileData checks if a file exists on remote.
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

// DeleteStoredFileData deletes a file from remote storage.
func DeleteStoredFileData(store *WebDAVStore, fileKey string) error {
	return store.RemoveFile(fileKey)
}
