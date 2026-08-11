package ui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestTruncateTextPreservesUnicodeAndWidth(t *testing.T) {
	got := truncateText("配置文件-very-long-name.json", 12)
	if !utf8.ValidString(got) {
		t.Fatalf("truncateText returned invalid UTF-8: %q", got)
	}
	if width := lipgloss.Width(got); width > 12 {
		t.Fatalf("truncateText width = %d, want <= 12: %q", width, got)
	}
}

func TestFileLineFitsTerminal(t *testing.T) {
	item := model.FileRecord{
		FileName:       "配置文件-with-a-very-long-filename.json",
		Size:           128 * 1024,
		LastUploadTime: 1700000000000,
		Note:           "shared editor configuration",
	}
	state := model.FileStatus{Key: "pending_upload"}

	profiles := []termenv.Profile{termenv.Ascii, termenv.ANSI, termenv.ANSI256, termenv.TrueColor}
	for _, profile := range profiles {
		lipgloss.SetColorProfile(profile)
		for _, width := range []int{40, 72, 120} {
			app := &App{width: width}
			for _, selected := range []bool{false, true} {
				line := app.fileLine(12, item, state, selected)
				if got := lipgloss.Width(line); got > width {
					t.Errorf("fileLine width = %d, terminal = %d, profile = %v, selected = %v", got, width, profile, selected)
				}
			}
		}
	}
}

func TestSelectedFileLineUsesPortableFocusStyle(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	app := &App{width: 80}
	line := app.fileLine(1, model.FileRecord{FileName: "settings.json"}, model.FileStatus{Key: "matched"}, true)
	plain := ansi.Strip(line)
	if !strings.HasPrefix(plain, "> ") {
		t.Fatalf("selected row does not start with the portable focus cursor: %q", plain)
	}
	if strings.Contains(line, "\x1b[48;") {
		t.Fatalf("selected row contains a background color sequence: %q", line)
	}
}

func TestBottomBarFitsChineseAndEnglish(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	locales := []i18n.Locale{i18n.Zh, i18n.En}
	states := []viewState{viewFileList, viewAddFile, viewSettings, viewConfirm, viewHelp, viewExportConfig}
	for _, locale := range locales {
		i18n.SetLocale(locale)
		for _, state := range states {
			for _, width := range []int{28, 48, 72, 100, 140} {
				app := &App{width: width, state: state}
				if got := lipgloss.Width(app.renderBottomBar()); got > width {
					t.Errorf("bottom bar width = %d, terminal = %d, locale = %s, state = %d", got, width, locale, state)
				}
			}
		}
	}
	i18n.SetLocale(i18n.Zh)
}

func TestViewsFitNarrowTerminal(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	fileList := []model.FileRecord{
		{ID: "1", FileName: "配置文件-with-很长-filename-而且-notice.json", Size: 128 * 1024, Note: "shared config for team", LastUploadTime: 1700000000000},
		{ID: "2", FileName: "settings.json", Size: 12, Note: "editor settings"},
	}
	input := func(v string) textinput.Model {
		m := textinput.New()
		m.SetValue(v)
		return m
	}
	newApp := func(width int) *App {
		a := &App{
			width:          width,
			height:         26,
			state:          viewFileList,
			fileList:       fileList,
			localDirMap:    map[string]string{},
			localStateMap:  map[string]model.FileState{},
			probeCache:     map[string]localProbe{},
			fileStateCache: map[string]model.FileStatus{},
			cursor:         0,
			pageRows:       8,
			autoSync:       true,
			autoCountdown:  60,
			lastSyncTime:   time.Now(),
			exportCommand:  "sfs import --endpoint https://dav.example.com/dav --username user --password secret --base-path small-file-sync",
			lastSyncResult: &model.SyncResult{
				Summary: model.SyncSummary{Checked: 12, Uploaded: 2, Downloaded: 1, Failed: 1, Unbound: 1},
				Details: []model.SyncDetail{
					{FileName: "配置文件-很长.json", Action: "UP", Status: i18n.T("common.failure"), Reason: "网络错误或磁盘空间不足"},
					{FileName: "settings.json", Action: "SKIP", Status: "ok", Reason: "no change"},
				},
			},
		}
		a.addFileInputs = []textinput.Model{input("/sdcard/Download/"), input("note text")}
		a.settingsInputs = []textinput.Model{input("value"), input("value"), input("value"), input("value")}
		a.setDirInput = input("/sdcard/Download")
		return a
	}

	cases := []struct {
		name  string
		state viewState
		setup func(a *App)
	}{
		{name: "file list", state: viewFileList},
		{name: "add file", state: viewAddFile},
		{name: "settings", state: viewSettings, setup: func(a *App) { a.settingsFocus = 2 }},
		{name: "set dir", state: viewSetDir},
		{name: "sync result", state: viewSyncResult},
		{name: "confirm", state: viewConfirm, setup: func(a *App) {
			a.confirmTitle = "更新到 v0.3.0（当前 v0.2.0）"
			a.confirmMsg = i18n.T("update.brew_confirm")
			a.confirmLabel = i18n.T("update.action")
			a.confirmAction = func() tea.Msg { return doUpdateMsg{} }
		}},
		{name: "help", state: viewHelp},
		{name: "note", state: viewNote},
		{name: "export", state: viewExportConfig},
	}

	for _, tc := range cases {
		for _, width := range []int{36, 48, 72, 100} {
			for _, locale := range []i18n.Locale{i18n.Zh, i18n.En} {
				i18n.SetLocale(locale)
				a := newApp(width)
				a.state = tc.state
				if tc.setup != nil {
					tc.setup(a)
				}
				out := a.View()
				for n, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
					if got := lipgloss.Width(line); got > width {
						t.Errorf("%s (locale=%s, width=%d) line %d width=%d: %q", tc.name, locale, width, n, got, ansi.Strip(line))
					}
				}
			}
		}
	}
	i18n.SetLocale(i18n.Zh)
}
