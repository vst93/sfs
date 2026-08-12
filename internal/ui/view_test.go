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
		for _, width := range []int{40, 62, 90, 120} {
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

func TestFileListMetadataColumnsAlign(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	i18n.SetLocale(i18n.En)
	t.Cleanup(func() { i18n.SetLocale(i18n.Zh) })

	app := &App{width: 110, fileList: make([]model.FileRecord, 120)}
	stamp := int64(1700000000000)
	state := model.FileStatus{Key: "matched"}
	short := ansi.Strip(app.fileLine(1, model.FileRecord{FileName: "a.txt", Size: 12, LastUploadTime: stamp, Note: "short"}, state, false))
	long := ansi.Strip(app.fileLine(120, model.FileRecord{FileName: "a-very-long-file-name-that-needs-truncation.json", Size: 2048, LastUploadTime: stamp, Note: "long"}, state, false))
	header := ansi.Strip(app.fileListHeader())
	updated := time.UnixMilli(stamp).Format("01-02 15:04")

	for _, text := range []string{"Synced", updated} {
		shortCol := displayColumnBefore(t, short, text)
		longCol := displayColumnBefore(t, long, text)
		if shortCol != longCol {
			t.Errorf("%q column differs: short=%d long=%d\nshort: %q\nlong:  %q", text, shortCol, longCol, short, long)
		}
	}
	if got, want := displayColumnBefore(t, header, "Status"), displayColumnBefore(t, short, "Synced"); got != want {
		t.Errorf("status header column = %d, row column = %d", got, want)
	}
	if got, want := displayColumnBefore(t, header, "Updated"), displayColumnBefore(t, short, updated); got != want {
		t.Errorf("updated header column = %d, row column = %d", got, want)
	}
}

func TestFileListColumnsFitSupportedDesktopWidths(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	for _, locale := range []i18n.Locale{i18n.Zh, i18n.En} {
		i18n.SetLocale(locale)
		for _, width := range []int{62, 72, 91, 92, 110, 140} {
			app := &App{width: width, fileList: make([]model.FileRecord, 1000)}
			item := model.FileRecord{
				FileName:       "配置文件-with-a-very-long-filename.json",
				Size:           128 * 1024,
				LastUploadTime: 1700000000000,
				Note:           "shared editor configuration",
			}
			for _, line := range []string{
				app.fileListHeader(),
				app.fileLine(1000, item, model.FileStatus{Key: "initial_upload"}, false),
			} {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("locale=%s width=%d rendered=%d: %q", locale, width, got, ansi.Strip(line))
				}
			}
		}
	}
	i18n.SetLocale(i18n.Zh)
}

func displayColumnBefore(t *testing.T, line, text string) int {
	t.Helper()
	idx := strings.Index(line, text)
	if idx < 0 {
		t.Fatalf("%q not found in %q", text, line)
	}
	return lipgloss.Width(line[:idx])
}

func TestSelectedFileLineUsesPortableFocusStyleAndForeground(t *testing.T) {
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
	accentPrefix := strings.SplitN(lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("selected"), "selected", 2)[0]
	if !strings.HasPrefix(line, accentPrefix) {
		t.Fatalf("selected row does not use the primary foreground accent: %q", line)
	}
}

func TestAdaptiveForegroundsSupportLightAndDarkTerminals(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	lipgloss.SetHasDarkBackground(false)
	lightMuted := styleMuted.Render("muted")
	wantLightMuted := lipgloss.NewStyle().Foreground(lipgloss.Color("#58677A")).Render("muted")
	if lightMuted != wantLightMuted {
		t.Fatalf("light terminal muted color = %q, want %q", lightMuted, wantLightMuted)
	}
	lightSelected := (&App{width: 40}).fileLine(1, model.FileRecord{FileName: "settings.json"}, model.FileStatus{Key: "matched"}, true)
	wantLightAccent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0F766E")).Render("selected")
	if !strings.Contains(lightSelected, strings.SplitN(wantLightAccent, "selected", 2)[0]) {
		t.Fatalf("light terminal selected row does not use the light accent: %q", lightSelected)
	}

	lipgloss.SetHasDarkBackground(true)
	darkMuted := styleMuted.Render("muted")
	wantDarkMuted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8")).Render("muted")
	if darkMuted != wantDarkMuted {
		t.Fatalf("dark terminal muted color = %q, want %q", darkMuted, wantDarkMuted)
	}
	darkSelected := (&App{width: 40}).fileLine(1, model.FileRecord{FileName: "settings.json"}, model.FileStatus{Key: "matched"}, true)
	wantDarkAccent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2DD4BF")).Render("selected")
	if !strings.Contains(darkSelected, strings.SplitN(wantDarkAccent, "selected", 2)[0]) {
		t.Fatalf("dark terminal selected row does not use the dark accent: %q", darkSelected)
	}
}

func TestStrongTextInheritsTerminalForeground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	for _, dark := range []bool{false, true} {
		lipgloss.SetHasDarkBackground(dark)
		got := styleStrong.Render("SFS")
		if strings.Contains(got, "\x1b[38;") {
			t.Fatalf("strong text sets an explicit foreground (dark=%v): %q", dark, got)
		}
		if ansi.Strip(got) != "SFS" {
			t.Fatalf("strong text content = %q, want SFS", ansi.Strip(got))
		}
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
	newApp := func(width, height int) *App {
		a := &App{
			width:          width,
			height:         height,
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
		for _, size := range []struct{ width, height int }{{36, 16}, {40, 20}, {48, 26}, {72, 26}, {100, 32}} {
			for _, locale := range []i18n.Locale{i18n.Zh, i18n.En} {
				i18n.SetLocale(locale)
				a := newApp(size.width, size.height)
				a.state = tc.state
				if tc.setup != nil {
					tc.setup(a)
				}
				out := a.View()
				lines := strings.Split(out, "\n")
				if len(lines) > size.height {
					t.Errorf("%s (locale=%s, size=%dx%d) rendered %d lines", tc.name, locale, size.width, size.height, len(lines))
				}
				for n, line := range lines {
					if got := lipgloss.Width(line); got > size.width {
						t.Errorf("%s (locale=%s, size=%dx%d) line %d width=%d: %q", tc.name, locale, size.width, size.height, n, got, ansi.Strip(line))
					}
				}
			}
		}
	}
	i18n.SetLocale(i18n.Zh)
}

func TestSecondaryViewScrollKeepsBottomBarVisible(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	a := &App{width: 36, height: 12, state: viewHelp}
	before := a.View()
	a.handleHelpKey(tea.KeyMsg{Type: tea.KeyDown})
	after := a.View()
	if before == after {
		t.Fatal("help view did not move after scrolling")
	}
	if got := len(strings.Split(after, "\n")); got != a.height {
		t.Fatalf("scrolled view height = %d, want %d", got, a.height)
	}
	last := strings.Split(after, "\n")[a.height-1]
	if !strings.Contains(ansi.Strip(last), "Esc") {
		t.Fatalf("bottom bar was not kept visible after scrolling: %q", ansi.Strip(last))
	}
}

func TestCompactLayoutKeepsVerticalBreathingRoom(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	input := func(value string) textinput.Model {
		m := textinput.New()
		m.SetValue(value)
		return m
	}
	a := &App{
		width:          40,
		height:         20,
		state:          viewSettings,
		settingsInputs: []textinput.Model{input("endpoint"), input("user"), input("password"), input("path")},
	}
	plain := ansi.Strip(a.View())
	lines := strings.Split(plain, "\n")
	endpointLine := -1
	for i, line := range lines {
		if strings.Contains(line, "endpoint") {
			endpointLine = i
			break
		}
	}
	if endpointLine < 0 || endpointLine+1 >= len(lines) || strings.TrimSpace(lines[endpointLine+1]) != "" {
		t.Fatalf("compact fields are missing a blank separator line: %q", plain)
	}

	a.state = viewFileList
	a.fileList = []model.FileRecord{{ID: "1", FileName: "settings.json"}}
	a.localDirMap = map[string]string{}
	a.localStateMap = map[string]model.FileState{}
	a.probeCache = map[string]localProbe{}
	a.fileStateCache = map[string]model.FileStatus{}
	plain = ansi.Strip(a.View())
	if !strings.Contains(plain, "v"+model.AppVersion+"\n\n") {
		t.Fatalf("compact list header is missing breathing room: %q", plain)
	}
	if !strings.Contains(plain, strings.Repeat("-", a.width-2)) {
		t.Fatalf("compact list is missing the status/list divider: %q", plain)
	}
}
