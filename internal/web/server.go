package web

import (
	"encoding/json"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"smallFileSync/internal/model"
	"smallFileSync/internal/storage"
	"smallFileSync/internal/util"
	"strings"
	"sync"
	"time"
)

// Server holds the web server state.
type Server struct {
	localStore  *storage.LocalStore
	webdavStore *storage.WebDAVStore
	settings    model.AppSettings
	uid         string
	localDirMap map[string]string
	stateMap    map[string]model.FileState
	fileList    []model.FileRecord
	staticFS    embed.FS

	mu sync.RWMutex
}

// NewServer creates a new web server.
func NewServer(staticFS embed.FS) (*Server, error) {
	localStore, err := storage.NewLocalStore()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local store: %w", err)
	}

	uid := localStore.UID()
	settings := localStore.GetSettings()
	dirMap := localStore.GetLocalDirMap(uid)
	stateMap := localStore.GetFileStateMap(uid)

	// Try to migrate from uTools if local data is empty
	if len(dirMap) == 0 {
		migUID, migSettings, migDirMap, migStateMap, migErr := storage.UtoolsMigrate()
		if migErr == nil && len(migDirMap) > 0 {
			uid = migUID
			localStore.SaveUID(uid)
			settings = migSettings
			localStore.SaveSettings(settings)
			dirMap = migDirMap
			localStore.SaveLocalDirMap(uid, dirMap)
			stateMap = migStateMap
			localStore.SaveFileStateMap(uid, stateMap)
		}
	}

	webdavStore := storage.NewWebDAVStore(settings.Storage.WebDAV)

	s := &Server{
		localStore:  localStore,
		webdavStore: webdavStore,
		settings:    settings,
		uid:         uid,
		localDirMap: dirMap,
		stateMap:    stateMap,
		staticFS:    staticFS,
	}

	// Load file list from remote if configured
	if s.isStorageConfigured() {
		list, err := webdavStore.GetFileList()
		if err == nil {
			var normalized []model.FileRecord
			for _, r := range list {
				if r.ID != "" && r.FileName != "" {
					normalized = append(normalized, model.NormalizeFileRecord(r))
				}
			}
			s.fileList = normalized
		}
	}

	return s, nil
}

func (s *Server) isStorageConfigured() bool {
	w := s.settings.Storage.WebDAV
	return w.Endpoint != "" && w.Username != "" && w.Password != ""
}

// fileListItem is the JSON representation sent to the web frontend.
type fileListItem struct {
	ID             string  `json:"id"`
	FileName       string  `json:"fileName"`
	Note           string  `json:"note"`
	FileMD5        string  `json:"fileMd5"`
	LastChangeTime int64   `json:"lastChangeTime"`
	LastUploadTime int64   `json:"lastUploadTime"`
	LastUploadUser string  `json:"lastUploadUser"`
	Size           float64 `json:"size"`
	Status         string  `json:"status"`
	StatusDetail   string  `json:"statusDetail"`
	LocalDir       string  `json:"localDir"`
	LocalPath      string  `json:"localPath"`
	HasRemote      bool    `json:"hasRemote"`
	HasLocal       bool    `json:"hasLocal"`
}

func (s *Server) buildFileListItem(item model.FileRecord) fileListItem {
	localDir := s.localDirMap[item.ID]
	localPath := ""
	hasLocal := false
	if localDir != "" {
		localPath = localDir + util.FileSeparator() + item.FileName
		_, err := os.Stat(localPath)
		hasLocal = err == nil
	}

	hasRemote := item.LastUploadTime > 0 && item.FileMD5 != "" && item.StorageKey() != ""

	status := "pending_upload"
	statusDetail := "本地内容较新，可上传覆盖云端"
	if localDir == "" {
		status = "unbound"
		statusDetail = "请先为当前设备设置本地目录"
	} else if !hasRemote {
		status = "initial_upload"
		statusDetail = "请执行首次上传"
	} else if !hasLocal {
		status = "missing"
		statusDetail = "本地文件缺失，可从云端恢复"
	} else {
		probe := s.probeLocal(localPath)
		if probe.OK && probe.MD5 == item.FileMD5 {
			status = "matched"
			statusDetail = "本地与云端一致"
		} else if probe.OK {
			localState := s.stateMap[item.ID]
			localChanged := localState.MD5 != "" && localState.MD5 != probe.MD5
			remoteChanged := localState.MD5 != "" && localState.MD5 != item.FileMD5
			if localChanged && remoteChanged {
				status = "conflict"
				statusDetail = "本地与云端都已变化，请手动选择"
			} else if remoteChanged && !localChanged {
				status = "download"
				statusDetail = "云端版本较新，可下载覆盖本地"
			} else {
				status = "pending_upload"
				statusDetail = "本地内容较新，可上传覆盖云端"
			}
		} else {
			status = "missing"
			statusDetail = "本地文件缺失，可从云端恢复"
		}
	}

	return fileListItem{
		ID:             item.ID,
		FileName:       item.FileName,
		Note:           item.Note,
		FileMD5:        item.FileMD5,
		LastChangeTime: item.LastChangeTime,
		LastUploadTime: item.LastUploadTime,
		LastUploadUser: item.LastUploadUser,
		Size:           item.Size,
		Status:         status,
		StatusDetail:   statusDetail,
		LocalDir:       localDir,
		LocalPath:      localPath,
		HasRemote:      hasRemote,
		HasLocal:       hasLocal,
	}
}

type localProbe struct {
	OK    bool
	MD5   string
	Mtime int64
	Size  int64
}

func (s *Server) probeLocal(path string) localProbe {
	if path == "" {
		return localProbe{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return localProbe{}
	}
	mtime := info.ModTime().UnixMilli()
	md5Hash, err := util.CalculateFileMD5(path)
	if err != nil {
		return localProbe{}
	}
	return localProbe{OK: true, MD5: md5Hash, Mtime: mtime, Size: info.Size()}
}

// Start launches the web server.
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/files/add", s.handleAddFile)
	mux.HandleFunc("/api/files/delete", s.handleDeleteFile)
	mux.HandleFunc("/api/files/note", s.handleUpdateNote)
	mux.HandleFunc("/api/files/dir", s.handleSetDir)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/sync/single", s.handleSyncSingle)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/test", s.handleTestConnection)

	// Static files from embed
	sub, err := fs.Sub(s.staticFS, "internal/web/static")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	fsHandler := http.FileServer(http.FS(sub))
	mux.Handle("/", fsHandler)

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Port occupied, try auto-find a free port
		fmt.Printf("Port %d is occupied, finding an available port...\n", port)
		ln, err = net.Listen("tcp", ":0")
		if err != nil {
			return fmt.Errorf("failed to find an available port: %w", err)
		}
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port

	url := fmt.Sprintf("http://localhost:%d", actualPort)
	fmt.Printf("SFS Web mode started at %s\n", url)

	// Try to open browser automatically
	go openBrowser(url)

	return http.Serve(ln, mux)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}

// ── API Handlers ────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{
		"appName":    model.AppFullName,
		"appVersion": model.AppVersion,
		"uid":        s.uid,
	})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "Method not allowed")
		return
	}

	s.mu.Lock()
	// Refresh file list from remote if configured
	if s.isStorageConfigured() {
		list, err := s.webdavStore.GetFileList()
		if err == nil {
			var normalized []model.FileRecord
			for _, r := range list {
				if r.ID != "" && r.FileName != "" {
					normalized = append(normalized, model.NormalizeFileRecord(r))
				}
			}
			s.fileList = normalized
		}
	}
	s.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]fileListItem, 0, len(s.fileList))
	for _, item := range s.fileList {
		items = append(items, s.buildFileListItem(item))
	}

	summary := map[string]int{"total": 0, "matched": 0, "pending": 0, "unbound": 0}
	for _, item := range items {
		summary["total"]++
		switch item.Status {
		case "matched":
			summary["matched"]++
		case "unbound":
			summary["unbound"]++
			summary["pending"]++
		default:
			summary["pending"]++
		}
	}

	writeJSON(w, 200, map[string]interface{}{
		"files":   items,
		"storage": s.isStorageConfigured(),
		"summary": summary,
	})
}

func (s *Server) handleAddFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isStorageConfigured() {
		writeError(w, 400, "请先配置存储设置")
		return
	}

	req.FilePath = strings.TrimSpace(req.FilePath)
	if req.FilePath == "" {
		writeError(w, 400, "文件路径不能为空")
		return
	}

	// Expand ~
	if strings.HasPrefix(req.FilePath, "~") {
		home, _ := os.UserHomeDir()
		req.FilePath = filepath.Join(home, req.FilePath[1:])
	}

	info, err := os.Stat(req.FilePath)
	if err != nil {
		writeError(w, 400, "文件不存在或不可读")
		return
	}
	if info.IsDir() {
		writeError(w, 400, "不支持添加目录")
		return
	}
	if info.Size() > 200*1024*1024 {
		writeError(w, 400, "文件大于200MB")
		return
	}

	fileName := filepath.Base(req.FilePath)
	dirPath := filepath.Dir(req.FilePath)
	note := strings.TrimSpace(req.Note)

	// Check duplicates
	for _, item := range s.fileList {
		if item.FileName == fileName && s.localDirMap[item.ID] == dirPath {
			writeError(w, 400, "当前目录下同名文件已存在同步记录")
			return
		}
	}

	newID := util.GenerateUID()
	record := model.NormalizeFileRecord(model.FileRecord{
		ID:       newID,
		FileName: fileName,
		Note:     note,
	})

	s.localDirMap[newID] = dirPath
	if err := s.localStore.SaveLocalDirMap(s.uid, s.localDirMap); err != nil {
		writeError(w, 500, "保存本地目录失败")
		return
	}

	s.fileList = append(s.fileList, record)
	if err := s.webdavStore.SaveFileList(s.fileList); err != nil {
		s.fileList = s.fileList[:len(s.fileList)-1]
		delete(s.localDirMap, newID)
		_ = s.localStore.SaveLocalDirMap(s.uid, s.localDirMap)
		writeError(w, 500, "保存元数据失败: "+err.Error())
		return
	}

	writeJSON(w, 200, map[string]string{"id": newID, "message": "已添加"})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, item := range s.fileList {
		if item.ID == req.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		writeError(w, 404, "未找到记录")
		return
	}

	item := s.fileList[idx]
	_ = s.webdavStore.RemoveFile(item.StorageKey())

	s.fileList = append(s.fileList[:idx], s.fileList[idx+1:]...)
	if err := s.webdavStore.SaveFileList(s.fileList); err != nil {
		writeError(w, 500, "保存文件列表失败: "+err.Error())
		return
	}

	delete(s.localDirMap, item.ID)
	delete(s.stateMap, item.ID)
	_ = s.localStore.SaveLocalDirMap(s.uid, s.localDirMap)
	_ = s.localStore.SaveFileStateMap(s.uid, s.stateMap)

	writeJSON(w, 200, map[string]string{"message": "已删除"})
}

func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		ID   string `json:"id"`
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.fileList {
		if s.fileList[i].ID == req.ID {
			s.fileList[i].Note = strings.TrimSpace(req.Note)
			if err := s.webdavStore.SaveFileList(s.fileList); err != nil {
				writeError(w, 500, "保存失败")
				return
			}
			writeJSON(w, 200, map[string]string{"message": "已更新"})
			return
		}
	}
	writeError(w, 404, "未找到记录")
}

func (s *Server) handleSetDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		ID  string `json:"id"`
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dirPath := strings.TrimSpace(req.Dir)
	if dirPath == "" {
		writeError(w, 400, "目录路径不能为空")
		return
	}

	if strings.HasPrefix(dirPath, "~") {
		home, _ := os.UserHomeDir()
		dirPath = filepath.Join(home, dirPath[1:])
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		writeError(w, 400, "路径不存在")
		return
	}
	if !info.IsDir() {
		writeError(w, 400, "不是目录")
		return
	}

	s.localDirMap[req.ID] = dirPath
	if err := s.localStore.SaveLocalDirMap(s.uid, s.localDirMap); err != nil {
		writeError(w, 500, "保存失败")
		return
	}

	writeJSON(w, 200, map[string]string{"message": "目录已设置"})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		SyncType string `json:"syncType"`
		IsAuto   bool   `json:"isAuto"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isStorageConfigured() {
		writeError(w, 400, "请先配置存储设置")
		return
	}

	result := s.doSyncAll(req.SyncType, req.IsAuto)
	writeJSON(w, 200, result)
}

func (s *Server) handleSyncSingle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req struct {
		ID       string `json:"id"`
		SyncType string `json:"syncType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isStorageConfigured() {
		writeError(w, 400, "请先配置存储设置")
		return
	}

	idx := -1
	for i, item := range s.fileList {
		if item.ID == req.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		writeError(w, 404, "未找到记录")
		return
	}

	detail := s.doSyncItem(idx, req.SyncType)
	writeJSON(w, 200, detail)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.mu.RLock()
		defer s.mu.RUnlock()
		writeJSON(w, 200, map[string]interface{}{
			"settings":  s.settings,
			"hasStorage": s.isStorageConfigured(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			AutoSync bool               `json:"autoSync"`
			Storage  model.StorageConfig `json:"storage"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "Invalid request body")
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		s.settings.AutoSync = req.AutoSync
		s.settings.Storage = req.Storage
		s.webdavStore = storage.NewWebDAVStore(req.Storage.WebDAV)

		if err := s.localStore.SaveSettings(s.settings); err != nil {
			writeError(w, 500, "保存设置失败")
			return
		}

		writeJSON(w, 200, map[string]string{"message": "设置已保存"})
		return
	}

	writeError(w, 405, "Method not allowed")
}

func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "Method not allowed")
		return
	}

	var req model.WebDAVConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	testStore := storage.NewWebDAVStore(req)
	ok, msg := testStore.HealthCheck()
	writeJSON(w, 200, map[string]interface{}{
		"success": ok,
		"message": msg,
	})
}

// ── Sync logic ──────────────────────────────────────────────────────────────

type syncResult struct {
	IsAuto  bool         `json:"isAuto"`
	Summary syncSummary  `json:"summary"`
	Details []syncDetail `json:"details"`
}

type syncSummary struct {
	Checked    int `json:"checked"`
	Uploaded   int `json:"uploaded"`
	Downloaded int `json:"downloaded"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
	Unbound    int `json:"unbound"`
}

type syncDetail struct {
	FileName string `json:"fileName"`
	Action   string `json:"action"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

func (s *Server) doSyncAll(syncType string, isAuto bool) syncResult {
	result := syncResult{
		IsAuto:  isAuto,
		Summary: syncSummary{},
	}

	for i := range s.fileList {
		detail := s.doSyncItem(i, syncType)
		result.Summary.Checked++
		result.Details = append(result.Details, detail)

		if detail.Action == "未处理" {
			result.Summary.Unbound++
		} else if detail.Status == "成功" {
			switch detail.Action {
			case "上传":
				result.Summary.Uploaded++
			case "下载":
				result.Summary.Downloaded++
			case "跳过":
				result.Summary.Skipped++
			}
		} else {
			result.Summary.Failed++
		}
	}

	return result
}

func (s *Server) doSyncItem(idx int, syncType string) syncDetail {
	item := s.fileList[idx]
	localDir := s.localDirMap[item.ID]

	if localDir == "" {
		return syncDetail{
			FileName: item.FileName, Action: "未处理", Status: "失败", Reason: "未设置本地目录",
		}
	}

	localPath := localDir + util.FileSeparator() + item.FileName
	probe := s.probeLocal(localPath)
	hasRemote := item.LastUploadTime > 0 && item.FileMD5 != "" && item.StorageKey() != ""

	// Initial upload
	if !hasRemote {
		if syncType == "force_download" {
			return syncDetail{FileName: item.FileName, Action: "下载", Status: "失败", Reason: "尚未首次上传，暂无可下载内容"}
		}
		if !probe.OK {
			return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "读取本地文件失败"}
		}
		if probe.Size > 200*1024*1024 {
			return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "文件大于200MB"}
		}
		return s.doUpload(&item, localPath, probe.MD5, "首次上传成功")
	}

	// Missing local, has remote -> download
	if (!probe.OK || probe.MD5 == "") && hasRemote {
		if err := storage.SaveFileDataToLocal(s.webdavStore, localPath, item.StorageKey()); err != nil {
			return syncDetail{FileName: item.FileName, Action: "下载", Status: "失败", Reason: err.Error()}
		}
		s.commitLocalState(item.ID, localPath)
		return syncDetail{FileName: item.FileName, Action: "下载", Status: "成功", Reason: "已恢复到本地"}
	}

	// Skip if matched (not forced)
	if syncType != "force_download" && probe.OK && probe.MD5 == item.FileMD5 {
		return syncDetail{FileName: item.FileName, Action: "跳过", Status: "成功", Reason: "本地与云端一致"}
	}

	// Both missing
	if !probe.OK && !hasRemote {
		return syncDetail{FileName: item.FileName, Action: "未处理", Status: "失败", Reason: "本地文件缺失，且云端无内容"}
	}

	// Conflict check (only for auto sync)
	if syncType == "" {
		localState := s.stateMap[item.ID]
		localChanged := localState.MD5 != "" && localState.MD5 != probe.MD5
		remoteChanged := localState.MD5 != "" && localState.MD5 != item.FileMD5
		if localChanged && remoteChanged {
			return syncDetail{FileName: item.FileName, Action: "跳过", Status: "成功", Reason: "本地与云端同时修改，请手动处理"}
		}
	}

	// Decide direction
	shouldDownload := syncType == "force_download"
	if !shouldDownload && syncType != "force_upload" {
		localState := s.stateMap[item.ID]
		localChanged := localState.MD5 != "" && localState.MD5 != probe.MD5
		remoteChanged := localState.MD5 != "" && localState.MD5 != item.FileMD5
		if remoteChanged && !localChanged {
			shouldDownload = true
		}
	}

	if shouldDownload {
		if err := storage.SaveFileDataToLocal(s.webdavStore, localPath, item.StorageKey()); err != nil {
			return syncDetail{FileName: item.FileName, Action: "下载", Status: "失败", Reason: err.Error()}
		}
		s.commitLocalState(item.ID, localPath)
		reason := "云端较新"
		if syncType == "force_download" {
			reason = "已覆盖本地文件"
		}
		return syncDetail{FileName: item.FileName, Action: "下载", Status: "成功", Reason: reason}
	}

	// Upload
	if !probe.OK {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "读取本地文件失败"}
	}
	if probe.Size > 200*1024*1024 {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "文件大于200MB"}
	}

	reason := "本地较新"
	if syncType == "force_upload" {
		reason = "已覆盖云端文件"
	}
	return s.doUpload(&item, localPath, probe.MD5, reason)
}

func (s *Server) doUpload(item *model.FileRecord, localPath, localMD5, successReason string) syncDetail {
	fileKey, err := storage.SaveFileDataToStorage(s.webdavStore, localPath, item.ID)
	if err != nil {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: err.Error()}
	}

	ok, msg := storage.VerifyStoredFileData(s.webdavStore, fileKey, localMD5)
	if !ok {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: msg}
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "上传后读取文件失败"}
	}
	newMD5, _ := util.CalculateFileMD5(localPath)

	for i := range s.fileList {
		if s.fileList[i].ID == item.ID {
			s.fileList[i].FileID = fileKey
			s.fileList[i].FileIds = nil
			s.fileList[i].FileMD5 = newMD5
			s.fileList[i].LastUploadTime = time.Now().UnixMilli()
			s.fileList[i].LastUploadUser = util.CurrentUsername()
			s.fileList[i].LastChangeTime = info.ModTime().UnixMilli()
			s.fileList[i].Size = float64(info.Size()) / 1024
			break
		}
	}

	if err := s.webdavStore.SaveFileList(s.fileList); err != nil {
		return syncDetail{FileName: item.FileName, Action: "上传", Status: "失败", Reason: "保存元数据失败"}
	}

	s.commitLocalState(item.ID, localPath)
	return syncDetail{FileName: item.FileName, Action: "上传", Status: "成功", Reason: successReason}
}

func (s *Server) commitLocalState(fileID, localPath string) {
	info, err := os.Stat(localPath)
	if err != nil {
		return
	}
	md5Hash, err := util.CalculateFileMD5(localPath)
	if err != nil {
		return
	}
	s.stateMap[fileID] = model.FileState{
		MD5:          md5Hash,
		MtimeMs:      info.ModTime().UnixMilli(),
		LastSyncTime: time.Now().UnixMilli(),
	}
	_ = s.localStore.SaveFileStateMap(s.uid, s.stateMap)
}
