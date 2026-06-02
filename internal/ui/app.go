package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"
	"smallFileSync/internal/storage"
	"smallFileSync/internal/util"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// View states
type viewState int

const (
	viewFileList viewState = iota
	viewAddFile
	viewSettings
	viewSyncResult
	viewSetDir
)

// App is the main Bubble Tea model.
type App struct {
	state    viewState
	width    int
	height   int
	quitting bool

	// Data
	fileList       []model.FileRecord
	localDirMap    map[string]string
	localStateMap  map[string]model.FileState
	settings       model.AppSettings
	uid            string
	lastSyncResult *model.SyncResult

	// Storage
	localStore  *storage.LocalStore
	webdavStore *storage.WebDAVStore

	// File list cursor
	cursor     int
	pageOffset int // first visible row index
	pageRows   int // how many rows fit on screen

	// Add file state
	addFileInputs   []textinput.Model
	addFileFocus    int
	addFilePath     string
	addFileStats    os.FileInfo
	addFileFeedback string
	addFileErr      bool

	// Settings state
	settingsInputs   []textinput.Model
	settingsFocus    int
	settingsFeedback string
	settingsErr      bool
	showPassword     bool

	// Sync state
	syncing       bool
	syncProgress  string
	autoSync      bool
	autoCountdown int

	// Confirm dialog
	showConfirm   bool
	confirmTitle  string
	confirmMsg    string
	confirmLabel  string
	confirmAction func() tea.Msg

	// Toast
	toast     string
	toastType string // success, warning, error
	toastAt   time.Time

	// Info board (sync result overlay)
	showInfoBoard bool
	infoContent   string

	// Help
	showHelp bool

	// SetDir state
	setDirTarget   string
	setDirInput    textinput.Model
	setDirFeedback string
}

// Messages
type syncDoneMsg struct {
	result model.SyncResult
}

type syncProgressMsg struct {
	text string
}

type autoSyncTickMsg struct{}

type toastMsg struct {
	text string
	typ  string
}

type clearToastMsg struct{}

type fileListRefreshMsg struct{}

// NewApp creates the main application model.
func NewApp() (*App, error) {
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
			// Use migrated uid so it matches the uTools data
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

	// Apply saved language preference
	if settings.Language != "" {
		i18n.SetLocale(i18n.Locale(settings.Language))
	}

	app := &App{
		state:          viewFileList,
		localStore:     localStore,
		webdavStore:    webdavStore,
		settings:       settings,
		uid:            uid,
		localDirMap:    dirMap,
		localStateMap:  stateMap,
		autoSync:       settings.AutoSync,
		autoCountdown:  60,
		pageRows:       10,
		addFileInputs:  make([]textinput.Model, 2),
		settingsInputs: make([]textinput.Model, 4),
	}

	// Init add file inputs
	app.addFileInputs[0] = textinput.New()
	app.addFileInputs[0].Placeholder = i18n.T("add_file.placeholder.path")
	app.addFileInputs[0].Width = 50

	app.addFileInputs[1] = textinput.New()
	app.addFileInputs[1].Placeholder = i18n.T("add_file.placeholder.note")
	app.addFileInputs[1].Width = 40

	// Init settings inputs
	app.settingsInputs[0] = textinput.New()
	app.settingsInputs[0].Placeholder = i18n.T("settings.placeholder.endpoint")
	app.settingsInputs[0].Width = 50
	app.settingsInputs[0].SetValue(settings.Storage.WebDAV.Endpoint)

	app.settingsInputs[1] = textinput.New()
	app.settingsInputs[1].Placeholder = i18n.T("settings.placeholder.username")
	app.settingsInputs[1].Width = 40
	app.settingsInputs[1].SetValue(settings.Storage.WebDAV.Username)

	app.settingsInputs[2] = textinput.New()
	app.settingsInputs[2].Placeholder = i18n.T("settings.placeholder.password")
	app.settingsInputs[2].EchoMode = textinput.EchoPassword
	app.settingsInputs[2].Width = 40
	app.settingsInputs[2].SetValue(settings.Storage.WebDAV.Password)

	app.settingsInputs[3] = textinput.New()
	app.settingsInputs[3].Placeholder = i18n.T("settings.placeholder.base_path")
	app.settingsInputs[3].Width = 40
	if settings.Storage.WebDAV.BasePath != "" {
		app.settingsInputs[3].SetValue(settings.Storage.WebDAV.BasePath)
	}

	return app, nil
}

// Init initializes the Bubble Tea program.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.loadFileList(),
		a.startAutoSyncTicker(),
	)
}

func (a *App) loadFileList() tea.Cmd {
	return func() tea.Msg {
		if !a.isStorageConfigured() {
			return fileListRefreshMsg{}
		}
		list, err := a.webdavStore.GetFileList()
		if err != nil {
			return toastMsg{text: fmt.Sprintf(i18n.T("error.load_file_list"), err.Error()), typ: "error"}
		}
		var normalized []model.FileRecord
		for _, r := range list {
			if r.ID != "" && r.FileName != "" {
				normalized = append(normalized, model.NormalizeFileRecord(r))
			}
		}
		a.fileList = normalized
		return fileListRefreshMsg{}
	}
}

func (a *App) startAutoSyncTicker() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return autoSyncTickMsg{}
	})
}

func (a *App) isStorageConfigured() bool {
	w := a.settings.Storage.WebDAV
	return w.Endpoint != "" && w.Username != "" && w.Password != ""
}

func (a *App) showToast(text, typ string) tea.Cmd {
	return func() tea.Msg {
		return toastMsg{text: text, typ: typ}
	}
}

// Update handles messages and user input.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// header ~4 lines, footer ~2 lines, table header 1 line => reserve ~10 lines
		a.pageRows = max(3, msg.Height-10)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case syncDoneMsg:
		a.syncing = false
		a.syncProgress = ""
		result := msg.result
		a.lastSyncResult = &result
		a.showInfoBoard = true
		a.infoContent = a.renderSyncResult(result)
		return a, nil

	case syncProgressMsg:
		a.syncProgress = msg.text
		return a, nil

	case autoSyncTickMsg:
		if a.autoSync && !a.syncing {
			a.autoCountdown--
			if a.autoCountdown <= 0 {
				a.autoCountdown = 60
				return a, a.runSync("", "", true)
			}
		}
		return a, a.startAutoSyncTicker()

	case toastMsg:
		a.toast = msg.text
		a.toastType = msg.typ
		a.toastAt = time.Now()
		return a, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearToastMsg{}
		})

	case clearToastMsg:
		a.toast = ""
		return a, nil

	case fileListRefreshMsg:
		if a.cursor >= len(a.fileList) {
			a.cursor = max(0, len(a.fileList)-1)
		}
		a.moveCursor(a.cursor) // re-clamp pageOffset
		return a, nil
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Confirm dialog takes priority
	if a.showConfirm {
		return a.handleConfirmKey(msg)
	}

	// Help overlay
	if a.showHelp {
		if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
			a.showHelp = false
		}
		return a, nil
	}

	// Info board overlay
	if a.showInfoBoard {
		if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
			a.showInfoBoard = false
		}
		return a, nil
	}

	// Global keys
	switch msg.String() {
	case "ctrl+c":
		a.quitting = true
		return a, tea.Quit
	}

	// View-specific keys
	switch a.state {
	case viewFileList:
		return a.handleFileListKey(msg)
	case viewAddFile:
		return a.handleAddFileKey(msg)
	case viewSettings:
		return a.handleSettingsKey(msg)
	case viewSetDir:
		return a.handleSetDirKey(msg)
	}

	return a, nil
}

func (a *App) handleFileListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		a.quitting = true
		return a, tea.Quit

	// ── Navigation ──
	case "up", "k":
		a.moveCursor(a.cursor - 1)
	case "down", "j":
		a.moveCursor(a.cursor + 1)
	case "pgup", "left", "h":
		a.moveCursor(a.cursor - a.pageRows)
	case "pgdown", "right", "l":
		a.moveCursor(a.cursor + a.pageRows)
	case "g":
		a.moveCursor(0)
	case "G":
		a.moveCursor(len(a.fileList) - 1)

	// ── File operations ──
	case "a":
		if !a.isStorageConfigured() {
			return a, a.showToast(i18n.T("warn.configure_storage_s"), "warning")
		}
		a.state = viewAddFile
		a.addFileFocus = 0
		a.addFilePath = ""
		a.addFileStats = nil
		a.addFileFeedback = ""
		a.addFileInputs[0].SetValue("")
		a.addFileInputs[1].SetValue("")
		a.addFileInputs[0].Focus()
		return a, nil

	case "s":
		a.state = viewSettings
		a.settingsFocus = 0
		a.settingsFeedback = ""
		a.settingsInputs[0].SetValue(a.settings.Storage.WebDAV.Endpoint)
		a.settingsInputs[1].SetValue(a.settings.Storage.WebDAV.Username)
		a.settingsInputs[2].SetValue(a.settings.Storage.WebDAV.Password)
		basePath := a.settings.Storage.WebDAV.BasePath
		if basePath == "" {
			basePath = "small-file-sync"
		}
		a.settingsInputs[3].SetValue(basePath)
		a.settingsInputs[0].Focus()
		return a, nil

	case "enter":
		if len(a.fileList) == 0 {
			return a, nil
		}
		return a.handlePrimaryAction()

	case "u":
		// u = Upload (force)
		if len(a.fileList) == 0 {
			return a, nil
		}
		return a.handleForceUpload()

	case "d":
		// d = Download (force)
		if len(a.fileList) == 0 {
			return a, nil
		}
		return a.handleForceDownload()

	case "x":
		// x = delete record (vim-style)
		if len(a.fileList) == 0 {
			return a, nil
		}
		return a.handleDelete()

	case "e":
		// e = Edit directory
		if len(a.fileList) > 0 {
			return a.handleSetDir()
		}

	// ── Sync actions ──
	case "y":
		if !a.isStorageConfigured() {
			return a, a.showToast(i18n.T("warn.configure_storage"), "warning")
		}
		if a.syncing {
			return a, a.showToast(i18n.T("warn.sync_in_progress"), "warning")
		}
		return a, a.runSync("", "", false)

	case "o":
		// o = tOggle auto sync
		a.autoSync = !a.autoSync
		a.settings.AutoSync = a.autoSync
		_ = a.localStore.SaveSettings(a.settings)
		if a.autoSync {
			a.autoCountdown = 60
			return a, a.showToast(i18n.T("auto_sync.enabled"), "success")
		}
		return a, a.showToast(i18n.T("auto_sync.disabled"), "warning")

	case "r":
		if !a.isStorageConfigured() {
			return a, a.showToast(i18n.T("warn.configure_storage"), "warning")
		}
		return a, tea.Batch(a.loadFileList(), a.showToast(i18n.T("refreshing"), "success"))

	// ── General ──
	case "L":
		newLocale := i18n.ToggleLocale()
		a.settings.Language = string(newLocale)
		_ = a.localStore.SaveSettings(a.settings)
		return a, a.showToast(fmt.Sprintf("Language: %s", i18n.LocaleName(newLocale)), "success")

	case "ctrl+y":
		// Copy associated directory or full file path to clipboard
		if len(a.fileList) == 0 {
			return a, nil
		}
		item := a.fileList[a.cursor]
		if dir := a.localDirMap[item.ID]; dir != "" {
			fullPath := dir + util.FileSeparator() + item.FileName
			_ = copyToClipboard(fullPath)
			return a, a.showToast(fmt.Sprintf(i18n.T("toast.copied_path"), fullPath), "success")
		}
		return a, a.showToast(i18n.T("toast.no_dir"), "warning")

	case "?":
		a.showHelp = true
	}

	return a, nil
}

func (a *App) handlePrimaryAction() (tea.Model, tea.Cmd) {
	item := a.fileList[a.cursor]
	state := a.computeFileState(item)

	switch state.Key {
	case "unbound":
		return a.handleSetDir()
	case "initial_upload":
		return a, a.runSync(item.ID, "force_upload", false)
	case "matched":
		return a, a.runSync(item.ID, "force_download", false)
	case "download", "missing":
		return a, a.runSync(item.ID, "force_download", false)
	case "conflict":
		return a, a.runSync(item.ID, "force_upload", false)
	default: // pending_upload
		return a, a.runSync(item.ID, "force_upload", false)
	}
}

func (a *App) handleForceUpload() (tea.Model, tea.Cmd) {
	item := a.fileList[a.cursor]
	state := a.computeFileState(item)
	if state.Key == "initial_upload" {
		return a, a.runSync(item.ID, "force_upload", false)
	}
	a.showConfirm = true
	a.confirmTitle = i18n.T("confirm.cloud_overwrite.title")
	a.confirmMsg = i18n.T("confirm.cloud_overwrite.msg")
	a.confirmLabel = i18n.T("confirm.cloud_overwrite.action")
	a.confirmAction = func() tea.Msg {
		return a.doSync(item.ID, "force_upload", false)
	}
	return a, nil
}

func (a *App) handleForceDownload() (tea.Model, tea.Cmd) {
	item := a.fileList[a.cursor]
	a.showConfirm = true
	a.confirmTitle = i18n.T("confirm.local_overwrite.title")
	a.confirmMsg = i18n.T("confirm.local_overwrite.msg")
	a.confirmLabel = i18n.T("confirm.local_overwrite.action")
	a.confirmAction = func() tea.Msg {
		return a.doSync(item.ID, "force_download", false)
	}
	return a, nil
}

func (a *App) handleDelete() (tea.Model, tea.Cmd) {
	item := a.fileList[a.cursor]
	a.showConfirm = true
	a.confirmTitle = i18n.T("confirm.delete.title")
	a.confirmMsg = fmt.Sprintf(i18n.T("confirm.delete.msg"), item.FileName)
	a.confirmLabel = i18n.T("confirm.delete.action")
	a.confirmAction = func() tea.Msg {
		return a.doDelete(item.ID)
	}
	return a, nil
}

func (a *App) handleSetDir() (tea.Model, tea.Cmd) {
	if len(a.fileList) == 0 {
		return a, nil
	}
	item := a.fileList[a.cursor]
	a.state = viewSetDir
	a.setDirTarget = item.ID
	a.setDirInput = textinput.New()
	a.setDirInput.Placeholder = i18n.T("set_dir.placeholder")
	a.setDirInput.Width = 60
	if dir := a.localDirMap[item.ID]; dir != "" {
		a.setDirInput.SetValue(dir)
	} else {
		cwd, _ := os.Getwd()
		a.setDirInput.SetValue(cwd)
	}
	a.setDirInput.Focus()
	a.setDirFeedback = ""
	return a, nil
}

func (a *App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		a.showConfirm = false
		if a.confirmAction != nil {
			return a, a.confirmAction
		}
		return a, nil
	case "n", "N", "esc":
		a.showConfirm = false
		return a, nil
	}
	return a, nil
}

func (a *App) handleAddFileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = viewFileList
		return a, nil

	case "tab":
		a.addFileFocus = (a.addFileFocus + 1) % len(a.addFileInputs)
		for i := range a.addFileInputs {
			if i == a.addFileFocus {
				a.addFileInputs[i].Focus()
			} else {
				a.addFileInputs[i].Blur()
			}
		}
		return a, nil

	case "enter":
		return a.submitAddFile(false)

	case "ctrl+s":
		return a.submitAddFile(true)

	case "ctrl+y":
		val := a.addFileInputs[a.addFileFocus].Value()
		if val != "" {
			_ = copyToClipboard(val)
			return a, a.showToast(i18n.T("toast.copied"), "success")
		}
		return a, nil

	default:
		var cmd tea.Cmd
		a.addFileInputs[a.addFileFocus], cmd = a.addFileInputs[a.addFileFocus].Update(msg)
		// Validate path on every keystroke for input[0]
		if a.addFileFocus == 0 {
			a.validateAddFilePath()
		}
		return a, cmd
	}
}

func (a *App) validateAddFilePath() {
	path := strings.TrimSpace(a.addFileInputs[0].Value())
	if path == "" {
		a.addFilePath = ""
		a.addFileStats = nil
		return
	}
	// Expand ~
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	info, err := os.Stat(path)
	if err != nil {
		a.addFilePath = path
		a.addFileStats = nil
		return
	}
	if info.IsDir() {
		a.addFilePath = path
		a.addFileStats = nil
		return
	}
	if info.Size() > 200*1024*1024 {
		a.addFilePath = path
		a.addFileStats = nil
		return
	}
	a.addFilePath = path
	a.addFileStats = info
}

func (a *App) submitAddFile(continueAdding bool) (tea.Model, tea.Cmd) {
	if a.addFilePath == "" || a.addFileStats == nil {
		a.addFileFeedback = WarningText.Render(i18n.T("add_file.error.invalid_path"))
		a.addFileErr = true
		return a, nil
	}
	if a.addFileStats.IsDir() {
		a.addFileFeedback = WarningText.Render(i18n.T("add_file.error.is_dir"))
		a.addFileErr = true
		return a, nil
	}
	if a.addFileStats.Size() > 200*1024*1024 {
		a.addFileFeedback = WarningText.Render(i18n.T("add_file.error.too_large"))
		a.addFileErr = true
		return a, nil
	}

	fileName := filepath.Base(a.addFilePath)
	dirPath := filepath.Dir(a.addFilePath)
	note := strings.TrimSpace(a.addFileInputs[1].Value())

	// Check for duplicates
	for _, item := range a.fileList {
		if item.FileName == fileName && a.localDirMap[item.ID] == dirPath {
			a.addFileFeedback = WarningText.Render(i18n.T("add_file.error.duplicate"))
			a.addFileErr = true
			return a, nil
		}
	}

	// Create record
	newID := util.GenerateUID()
	record := model.NormalizeFileRecord(model.FileRecord{
		ID:       newID,
		FileName: fileName,
		Note:     note,
	})

	// Save to local dir map
	a.localDirMap[newID] = dirPath
	if err := a.localStore.SaveLocalDirMap(a.uid, a.localDirMap); err != nil {
		a.addFileFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("add_file.error.save_dir"), err.Error()))
		a.addFileErr = true
		return a, nil
	}

	// Save to remote file list
	a.fileList = append(a.fileList, record)
	if err := a.webdavStore.SaveFileList(a.fileList); err != nil {
		// Rollback
		a.fileList = a.fileList[:len(a.fileList)-1]
		delete(a.localDirMap, newID)
		_ = a.localStore.SaveLocalDirMap(a.uid, a.localDirMap)
		a.addFileFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("add_file.error.save"), err.Error()))
		a.addFileErr = true
		return a, nil
	}

	if continueAdding {
		a.addFileInputs[0].SetValue("")
		a.addFileInputs[1].SetValue("")
		a.addFilePath = ""
		a.addFileStats = nil
		a.addFileFeedback = SuccessText.Render(fmt.Sprintf(i18n.T("add_file.added_continue"), fileName))
		a.addFileErr = false
		a.addFileInputs[0].Focus()
		return a, nil
	}

	a.state = viewFileList
	return a, a.showToast(fmt.Sprintf(i18n.T("add_file.added"), fileName), "success")
}

func (a *App) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = viewFileList
		return a, nil

	case "tab":
		a.settingsFocus = (a.settingsFocus + 1) % len(a.settingsInputs)
		for i := range a.settingsInputs {
			if i == a.settingsFocus {
				a.settingsInputs[i].Focus()
			} else {
				a.settingsInputs[i].Blur()
			}
		}
		return a, nil

	case "enter":
		return a.saveSettings()

	case "t":
		return a.testConnection()

	case "p":
		// Toggle password visibility for the password field (index 2)
		if a.settingsFocus == 2 {
			a.showPassword = !a.showPassword
			if a.showPassword {
				a.settingsInputs[2].EchoMode = textinput.EchoNormal
			} else {
				a.settingsInputs[2].EchoMode = textinput.EchoPassword
			}
		}
		return a, nil

	case "ctrl+y":
		val := a.settingsInputs[a.settingsFocus].Value()
		if val != "" {
			_ = copyToClipboard(val)
			return a, a.showToast(i18n.T("toast.copied"), "success")
		}
		return a, nil

	default:
		var cmd tea.Cmd
		a.settingsInputs[a.settingsFocus], cmd = a.settingsInputs[a.settingsFocus].Update(msg)
		return a, cmd
	}
}

func (a *App) handleSetDirKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.state = viewFileList
		return a, nil

	case "enter":
		dirPath := strings.TrimSpace(a.setDirInput.Value())
		if dirPath == "" {
			a.setDirFeedback = WarningText.Render(i18n.T("set_dir.error.empty"))
			return a, nil
		}
		if strings.HasPrefix(dirPath, "~") {
			home, _ := os.UserHomeDir()
			dirPath = filepath.Join(home, dirPath[1:])
		}
		info, err := os.Stat(dirPath)
		if err != nil {
			a.setDirFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("set_dir.error.not_exist"), dirPath))
			return a, nil
		}
		if !info.IsDir() {
			a.setDirFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("set_dir.error.not_dir"), dirPath))
			return a, nil
		}

		a.localDirMap[a.setDirTarget] = dirPath
		if err := a.localStore.SaveLocalDirMap(a.uid, a.localDirMap); err != nil {
			a.setDirFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("error.save_failed"), err.Error()))
			return a, nil
		}

		a.state = viewFileList
		return a, a.showToast(fmt.Sprintf(i18n.T("set_dir.saved"), dirPath), "success")

	case "ctrl+y":
		val := a.setDirInput.Value()
		if val != "" {
			_ = copyToClipboard(val)
			return a, a.showToast(i18n.T("toast.copied"), "success")
		}
		return a, nil

	default:
		var cmd tea.Cmd
		a.setDirInput, cmd = a.setDirInput.Update(msg)
		return a, cmd
	}
}

func (a *App) testConnection() (tea.Model, tea.Cmd) {
	config := a.collectSettings().Storage.WebDAV
	testStore := storage.NewWebDAVStore(config)
	ok, msg := testStore.HealthCheck()
	if ok {
		return a, a.showToast(i18n.T("settings.test_success"), "success")
	}
	return a, a.showToast(fmt.Sprintf(i18n.T("settings.test_failed"), msg), "error")
}

func (a *App) saveSettings() (tea.Model, tea.Cmd) {
	settings := a.collectSettings()
	w := settings.Storage.WebDAV
	if w.Endpoint == "" || w.Username == "" || w.Password == "" {
		a.settingsFeedback = WarningText.Render(i18n.T("settings.error.required"))
		a.settingsErr = true
		return a, nil
	}

	// Test connection first
	testStore := storage.NewWebDAVStore(w)
	ok, msg := testStore.HealthCheck()
	if !ok {
		a.settingsFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("settings.test_failed"), msg))
		a.settingsErr = true
		return a, nil
	}

	a.settings = settings
	a.webdavStore = testStore
	if err := a.localStore.SaveSettings(settings); err != nil {
		a.settingsFeedback = ErrorText.Render(fmt.Sprintf(i18n.T("error.save_failed"), err.Error()))
		a.settingsErr = true
		return a, nil
	}

	a.state = viewFileList
	return a, tea.Batch(a.loadFileList(), a.showToast(i18n.T("settings.saved"), "success"))
}

func (a *App) collectSettings() model.AppSettings {
	return model.AppSettings{
		AutoSync: a.autoSync,
		Storage: model.StorageConfig{
			Type: "webdav",
			WebDAV: model.WebDAVConfig{
				Endpoint: strings.TrimSpace(a.settingsInputs[0].Value()),
				Username: strings.TrimSpace(a.settingsInputs[1].Value()),
				Password: a.settingsInputs[2].Value(),
				BasePath: strings.TrimSpace(a.settingsInputs[3].Value()),
			},
		},
	}
}

// Sync logic

func (a *App) runSync(fileID, syncType string, isAuto bool) tea.Cmd {
	return func() tea.Msg {
		return a.doSync(fileID, syncType, isAuto)
	}
}

// findFileIndex returns the index of the file with the given ID, or -1.
func (a *App) findFileIndex(id string) int {
	for i, item := range a.fileList {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func (a *App) doSync(fileID, syncType string, isAuto bool) tea.Msg {
	a.syncing = true

	// Wrap with a 5-minute timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result := model.SyncResult{
		IsAuto:    isAuto,
		StartedAt: time.Now(),
	}

	// For single-file sync, use direct index lookup
	var items []model.FileRecord
	if fileID != "" {
		idx := a.findFileIndex(fileID)
		if idx >= 0 {
			items = []model.FileRecord{a.fileList[idx]}
		}
	} else {
		items = a.fileList
	}

	for _, item := range items {
		// Check timeout
		select {
		case <-ctx.Done():
			result.Details = append(result.Details, model.SyncDetail{
				FileName: "...", Action: i18n.T("sync.action.skip"), Status: i18n.T("common.failure"),
				Reason: "sync timeout (5 minutes exceeded)",
			})
			return syncDoneMsg{result: result}
		default:
		}

		result.Summary.Checked++
		state := a.computeFileState(item)

		if !a.hasLocalDir(item.ID) {
			result.Summary.Unbound++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.unprocessed"), Status: i18n.T("common.failure"), Reason: state.Detail,
			})
			continue
		}

		localPath := a.localPath(item)
		probe := a.probeLocal(localPath)
		hasRemote := item.LastUploadTime > 0 && item.FileMD5 != "" && item.StorageKey() != "" && storage.HasStoredFileData(a.webdavStore, item.StorageKey())

		// Initial upload
		if state.Key == "initial_upload" {
			if syncType == "force_download" {
				result.Summary.Failed++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.download"), Status: i18n.T("common.failure"), Reason: i18n.T("sync.reason.cloud_not_available"),
				})
				continue
			}
			if !probe.ok {
				result.Summary.Failed++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.failure"), Reason: i18n.T("sync.reason.local_read_failed"),
				})
				continue
			}
			uploadResult := a.uploadFile(item, localPath, probe.md5)
			if uploadResult.success {
				result.Summary.Uploaded++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.success"), Reason: i18n.T("sync.reason.first_upload"),
				})
			} else {
				result.Summary.Failed++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.failure"), Reason: uploadResult.message,
				})
			}
			continue
		}

		// Missing local, has remote -> restore
		if (!probe.ok || probe.md5 == "") && hasRemote {
			if err := storage.SaveFileDataToLocal(a.webdavStore, localPath, item.StorageKey()); err != nil {
				result.Summary.Failed++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.download"), Status: i18n.T("common.failure"), Reason: err.Error(),
				})
				continue
			}
			a.commitCurrentLocalState(item.ID, localPath)
			result.Summary.Downloaded++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.download"), Status: i18n.T("common.success"), Reason: i18n.T("sync.reason.local_restored"),
			})
			continue
		}

		// Skip if matched (not forced)
		if syncType != "force_download" && probe.ok && probe.md5 == item.FileMD5 {
			result.Summary.Skipped++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.skip"), Status: i18n.T("common.success"), Reason: i18n.T("sync.reason.already_synced"),
			})
			continue
		}

		// No local, no remote
		if !probe.ok && !hasRemote {
			result.Summary.Failed++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.unprocessed"), Status: i18n.T("common.failure"), Reason: i18n.T("sync.reason.both_missing"),
			})
			continue
		}

		// Conflict (auto skip)
		if state.Key == "conflict" && syncType != "force_upload" && syncType != "force_download" {
			result.Summary.Skipped++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.skip"), Status: i18n.T("common.success"), Reason: i18n.T("sync.reason.conflict_manual"),
			})
			continue
		}

		// Decide direction
		shouldDownload := syncType == "force_download" || state.Key == "download" ||
			(syncType != "force_upload" && item.LastChangeTime > 0 && probe.mtime > 0 && probe.mtime < item.LastChangeTime)

		if shouldDownload {
			if err := storage.SaveFileDataToLocal(a.webdavStore, localPath, item.StorageKey()); err != nil {
				result.Summary.Failed++
				result.Details = append(result.Details, model.SyncDetail{
					FileName: item.FileName, Action: i18n.T("sync.action.download"), Status: i18n.T("common.failure"), Reason: err.Error(),
				})
				continue
			}
			a.commitCurrentLocalState(item.ID, localPath)
			result.Summary.Downloaded++
			reason := i18n.T("sync.reason.cloud_newer")
			if syncType == "force_download" {
				reason = i18n.T("sync.reason.overwrite_local")
			}
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.download"), Status: i18n.T("common.success"), Reason: reason,
			})
			continue
		}

		// Upload
		if !probe.ok {
			result.Summary.Failed++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.failure"), Reason: i18n.T("sync.reason.local_read_failed"),
			})
			continue
		}
		uploadResult := a.uploadFile(item, localPath, probe.md5)
		if uploadResult.success {
			result.Summary.Uploaded++
			reason := i18n.T("sync.reason.local_newer")
			if syncType == "force_upload" {
				reason = i18n.T("sync.reason.overwrite_cloud")
			}
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.success"), Reason: reason,
			})
		} else {
			result.Summary.Failed++
			result.Details = append(result.Details, model.SyncDetail{
				FileName: item.FileName, Action: i18n.T("sync.action.upload"), Status: i18n.T("common.failure"), Reason: uploadResult.message,
			})
		}
	}

	a.syncing = false
	_ = a.loadFileList()()
	return syncDoneMsg{result: result}
}

type uploadResult struct {
	success bool
	message string
}

func (a *App) uploadFile(item model.FileRecord, localPath, localMD5 string) uploadResult {
	var completed bool
	var fileKey string

	defer func() {
		if !completed && fileKey != "" {
			// Rollback: restore record and remove remote file
			for i := range a.fileList {
				if a.fileList[i].ID == item.ID {
					a.fileList[i].FileID = item.FileID
					a.fileList[i].FileIds = item.FileIds
					a.fileList[i].FileMD5 = item.FileMD5
					a.fileList[i].LastUploadTime = item.LastUploadTime
					a.fileList[i].LastUploadUser = item.LastUploadUser
					a.fileList[i].LastChangeTime = item.LastChangeTime
					a.fileList[i].Size = item.Size
					break
				}
			}
			_ = a.webdavStore.RemoveFile(fileKey)
		}
	}()

	// Upload as a single whole file
	var err error
	fileKey, err = storage.SaveFileDataToStorage(a.webdavStore, localPath, item.ID)
	if err != nil {
		return uploadResult{false, err.Error()}
	}

	// Verify
	ok, msg := storage.VerifyStoredFileData(a.webdavStore, fileKey, localMD5)
	if !ok {
		return uploadResult{false, msg}
	}

	// Re-read local info
	info, err := os.Stat(localPath)
	if err != nil {
		return uploadResult{false, i18n.T("sync.reason.upload_read_failed")}
	}
	newMD5, _ := util.CalculateFileMD5(localPath)

	// Update record
	for i := range a.fileList {
		if a.fileList[i].ID == item.ID {
			a.fileList[i].FileID = fileKey
			a.fileList[i].FileIds = nil
			a.fileList[i].FileMD5 = newMD5
			a.fileList[i].LastUploadTime = time.Now().UnixMilli()
			a.fileList[i].LastUploadUser = util.CurrentUsername()
			a.fileList[i].LastChangeTime = info.ModTime().UnixMilli()
			a.fileList[i].Size = float64(info.Size()) / 1024
			break
		}
	}

	// Save file list
	if err := a.webdavStore.SaveFileList(a.fileList); err != nil {
		return uploadResult{false, i18n.T("sync.reason.metadata_failed")}
	}

	// Update local state
	a.commitCurrentLocalState(item.ID, localPath)
	completed = true
	return uploadResult{true, ""}
}

func (a *App) doDelete(id string) tea.Msg {
	var targetIdx = -1
	for i, item := range a.fileList {
		if item.ID == id {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return toastMsg{text: i18n.T("delete.not_found"), typ: "warning"}
	}

	item := a.fileList[targetIdx]

	// Remove remote file
	_ = a.webdavStore.RemoveFile(item.StorageKey())

	// Remove from list
	a.fileList = append(a.fileList[:targetIdx], a.fileList[targetIdx+1:]...)
	if err := a.webdavStore.SaveFileList(a.fileList); err != nil {
		return toastMsg{text: fmt.Sprintf(i18n.T("delete.failed"), err.Error()), typ: "error"}
	}

	// Clean up local mappings
	delete(a.localDirMap, id)
	delete(a.localStateMap, id)
	_ = a.localStore.SaveLocalDirMap(a.uid, a.localDirMap)
	_ = a.localStore.SaveFileStateMap(a.uid, a.localStateMap)

	if a.cursor >= len(a.fileList) {
		a.cursor = max(0, len(a.fileList)-1)
	}
	return toastMsg{text: fmt.Sprintf(i18n.T("delete.success"), item.FileName), typ: "success"}
}

// Helper functions

func (a *App) hasLocalDir(id string) bool {
	_, ok := a.localDirMap[id]
	return ok
}

func (a *App) localPath(item model.FileRecord) string {
	dir := a.localDirMap[item.ID]
	if dir == "" {
		return ""
	}
	return dir + util.FileSeparator() + item.FileName
}

type localProbe struct {
	ok    bool
	md5   string
	mtime int64
	size  int64
}

func (a *App) probeLocal(path string) localProbe {
	if path == "" {
		return localProbe{}
	}
	info, err := os.Stat(path)
	if err != nil {
		return localProbe{}
	}
	md5Hash, err := util.CalculateFileMD5(path)
	if err != nil {
		return localProbe{}
	}
	return localProbe{
		ok:    true,
		md5:   md5Hash,
		mtime: info.ModTime().UnixMilli(),
		size:  info.Size(),
	}
}

func (a *App) commitCurrentLocalState(fileID, localPath string) {
	info, err := os.Stat(localPath)
	if err != nil {
		return
	}
	md5Hash, err := util.CalculateFileMD5(localPath)
	if err != nil {
		return
	}
	a.localStateMap[fileID] = model.FileState{
		MD5:          md5Hash,
		MtimeMs:      info.ModTime().UnixMilli(),
		LastSyncTime: time.Now().UnixMilli(),
	}
	_ = a.localStore.SaveFileStateMap(a.uid, a.localStateMap)
}

func (a *App) computeFileState(item model.FileRecord) model.FileStatus {
	if !a.hasLocalDir(item.ID) {
		return model.FileStatus{
			Key:    "unbound",
			Text:   i18n.T("status.unbound"),
			Detail: i18n.T("status.unbound.detail"),
		}
	}

	probe := a.probeLocal(a.localPath(item))
	hasRemote := item.LastUploadTime > 0 && item.FileMD5 != "" && item.StorageKey() != ""

	if !hasRemote {
		return model.FileStatus{
			Key:    "initial_upload",
			Text:   i18n.T("status.initial_upload"),
			Detail: i18n.T("status.initial_upload.detail"),
		}
	}

	if !probe.ok {
		return model.FileStatus{
			Key:    "missing",
			Text:   i18n.T("status.missing"),
			Detail: i18n.T("status.missing.detail"),
		}
	}

	if probe.md5 == item.FileMD5 {
		return model.FileStatus{
			Key:    "matched",
			Text:   i18n.T("status.matched"),
			Detail: i18n.T("status.matched.detail"),
		}
	}

	localState := a.localStateMap[item.ID]
	localChanged := localState.MD5 != "" && localState.MD5 != probe.md5
	remoteChanged := localState.MD5 != "" && localState.MD5 != item.FileMD5

	if localChanged && remoteChanged {
		return model.FileStatus{
			Key:    "conflict",
			Text:   i18n.T("status.conflict"),
			Detail: i18n.T("status.conflict.detail"),
		}
	}

	if remoteChanged && !localChanged {
		return model.FileStatus{
			Key:    "download",
			Text:   i18n.T("status.download"),
			Detail: i18n.T("status.download.detail"),
		}
	}

	if localChanged && !remoteChanged {
		return model.FileStatus{
			Key:    "pending_upload",
			Text:   i18n.T("status.pending_upload"),
			Detail: i18n.T("status.pending_upload.detail"),
		}
	}

	// Fallback: compare timestamps
	if item.LastChangeTime > 0 && probe.mtime > 0 && probe.mtime < item.LastChangeTime {
		return model.FileStatus{
			Key:    "download",
			Text:   i18n.T("status.download"),
			Detail: i18n.T("status.download.detail"),
		}
	}

	return model.FileStatus{
		Key:    "pending_upload",
		Text:   i18n.T("status.pending_upload"),
		Detail: i18n.T("status.pending_upload.detail"),
	}
}

// openFileInManager opens a file path in the system file manager.
func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func openFileInManager(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filepath.Dir(filePath))
	case "windows":
		cmd = exec.Command("explorer", "/select,", filePath)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// moveCursor moves the cursor to the given target index and keeps it visible.
func (a *App) moveCursor(target int) {
	total := len(a.fileList)
	if total == 0 {
		a.cursor = 0
		a.pageOffset = 0
		return
	}
	if target < 0 {
		target = 0
	}
	if target >= total {
		target = total - 1
	}
	a.cursor = target
	// Keep cursor in the visible window
	if a.cursor < a.pageOffset {
		a.pageOffset = a.cursor
	}
	if a.cursor >= a.pageOffset+a.pageRows {
		a.pageOffset = a.cursor - a.pageRows + 1
	}
	if a.pageOffset < 0 {
		a.pageOffset = 0
	}
}
