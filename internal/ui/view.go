package ui

import (
	"fmt"
	"path/filepath"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func (a *App) View() string {
	if a.quitting {
		return ""
	}

	var body strings.Builder
	switch a.state {
	case viewFileList:
		body.WriteString(a.renderFileList())
	case viewAddFile:
		body.WriteString(a.renderAddFile())
	case viewSettings:
		body.WriteString(a.renderSettings())
	case viewSetDir:
		body.WriteString(a.renderSetDir())
	case viewSyncResult:
		body.WriteString(a.renderSyncResultView())
	case viewConfirm:
		body.WriteString(a.renderConfirmView())
	case viewHelp:
		body.WriteString(a.renderHelpView())
	case viewNote:
		body.WriteString(a.renderNoteView())
	case viewExportConfig:
		body.WriteString(a.renderExportConfigView())
	}
	bodyStr := body.String()
	bodyStr = strings.TrimRight(bodyStr, "\n")

	toastStr := ""
	if a.toast != "" {
		var toast strings.Builder
		a.renderToast(&toast)
		toastStr = toast.String()
	}

	available := max(0, a.height-1) // reserve the last line for shortcuts
	bodyAvailable := available
	if toastStr != "" && bodyAvailable > 0 {
		bodyAvailable--
	}
	bodyStr = a.fitBodyToViewport(bodyStr, bodyAvailable)
	bodyLines := lineCount(bodyStr)
	padding := bodyAvailable - bodyLines
	if padding > 0 {
		bodyStr += strings.Repeat("\n", padding)
	}

	parts := make([]string, 0, 3)
	if bodyAvailable > 0 {
		parts = append(parts, bodyStr)
	}
	if toastStr != "" && available > 0 {
		parts = append(parts, toastStr)
	}
	parts = append(parts, a.renderBottomBar())
	return strings.Join(parts, "\n")
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func (a *App) fitBodyToViewport(body string, height int) string {
	if body == "" || height <= 0 {
		return ""
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= height {
		a.scrollOffset = 0
		return body
	}
	start := 0
	if a.state != viewFileList {
		start = min(a.scrollOffset, len(lines)-height)
		a.scrollOffset = start
	}
	return strings.Join(lines[start:start+height], "\n")
}

// ── Shared layout helpers ───────────────────────────────────────────────────

// compact reports whether the terminal is narrow enough to require the
// mobile/portrait layout (e.g. Termux).
func (a *App) compact() bool {
	return a.width < 62
}

func (a *App) contentWidth() int {
	return max(2, a.width-4)
}

func (a *App) viewHeader(title string) string {
	title = truncateText(strings.TrimSpace(title), max(1, a.contentWidth()-2))
	left := "  " + styleStrong.Render(title)
	if a.compact() {
		return left + "\n" + separator(a.contentWidth()) + "\n"
	}
	tag := styleMuted.Render("SFS  v" + model.AppVersion)
	gap := max(2, a.width-lipgloss.Width(left)-lipgloss.Width(tag)-2)
	return left + strings.Repeat(" ", gap) + tag + "\n" + separator(a.contentWidth()) + "\n"
}

func (a *App) viewFooter(hint string) string {
	if a.compact() {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n" + separator(a.contentWidth()) + "\n")
	for _, line := range wrapText(hint, max(1, a.contentWidth()-2)) {
		b.WriteString("  " + line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (a *App) inputWidth(labelW int) int {
	if a.compact() {
		// textinput adds its prompt and cursor around Width.
		return max(4, a.width-7)
	}
	return max(6, min(64, a.width-8-labelW-2))
}

func (a *App) fieldLine(label string, input textinput.Model, focused bool, labelW int) string {
	if a.compact() {
		marker := "  "
		labelStyle := styleMuted
		if focused {
			marker = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("> ")
			labelStyle = styleStrong
		}
		return "  " + marker + labelStyle.Render(label) + "\n    " + input.View()
	}
	labelS := lipgloss.NewStyle().Foreground(colorMuted).Width(labelW).Render(label)
	if focused {
		return "  " + lipgloss.NewStyle().Foreground(colorPrimary).Render("> ") + labelS + ": " + input.View()
	}
	return "    " + labelS + ": " + input.View()
}

// ── Toast ───────────────────────────────────────────────────────────────────

func (a *App) renderToast(b *strings.Builder) {
	var icon, label string
	var clr lipgloss.TerminalColor
	switch a.toastType {
	case "warning":
		icon, label, clr = "!", i18n.T("common.warning"), colorWarning
	case "error":
		icon, label, clr = "x", i18n.T("common.error"), colorDanger
	default:
		icon, label, clr = "+", i18n.T("common.success"), colorSuccess
	}
	tag := lipgloss.NewStyle().
		Bold(true).
		Foreground(clr).
		Render("[" + icon + "] " + label + ":")
	msgW := max(1, a.width-lipgloss.Width(tag)-3)
	msg := lipgloss.NewStyle().Foreground(clr).Render(truncateText(a.toast, msgW))
	b.WriteString(" " + tag + " " + msg)
}

// ── Bottom bar ──────────────────────────────────────────────────────────────

func (a *App) renderBottomBar() string {
	storage := i18n.T("bottom.storage_unconfigured")
	if a.isStorageConfigured() {
		storage = "WebDAV"
	}

	// Build right side: auto-sync status + storage
	rightParts := []string{}
	if a.syncing {
		rightParts = append(rightParts, stylePrimary.Render(i18n.T("bottom.syncing")))
	} else if a.autoSync {
		rightParts = append(rightParts, a.renderAutoSyncCountdown())
	}
	rightParts = append(rightParts, styleMuted.Render(storage))
	right := strings.Join(rightParts, "  ·  ")

	// Show update badge if new version available
	if a.updateDone && a.updateResult != nil && a.updateResult.HasUpdate {
		right = styleWarning.Render(fmt.Sprintf("UP %s", a.updateResult.LatestVersion)) + "  " + right
	}

	if a.state != viewFileList {
		left := renderShortcutSet(a.contextShortcuts())
		if lipgloss.Width(left) > max(0, a.width) {
			left = renderShortcutSet([]string{a.contextFallbackShortcut()})
		}
		return lipgloss.NewStyle().Foreground(colorBarText).Width(max(0, a.width)).Render(left)
	}

	tiers := [][]string{
		{i18n.T("bottom.navigate"), i18n.T("bottom.action"), i18n.T("bottom.upload"), i18n.T("bottom.download"), i18n.T("bottom.add"), i18n.T("bottom.settings"), i18n.T("bottom.sync_all"), i18n.T("bottom.update"), i18n.T("bottom.help"), i18n.T("bottom.quit")},
		{i18n.T("bottom.navigate"), i18n.T("bottom.action"), i18n.T("bottom.add"), i18n.T("bottom.settings"), i18n.T("bottom.sync_all"), i18n.T("bottom.update"), i18n.T("bottom.help"), i18n.T("bottom.quit")},
		{i18n.T("bottom.navigate"), i18n.T("bottom.action"), i18n.T("bottom.add"), i18n.T("bottom.settings"), i18n.T("bottom.help"), i18n.T("bottom.quit")},
		{i18n.T("bottom.action"), i18n.T("bottom.note"), i18n.T("bottom.help"), i18n.T("bottom.quit")},
		{i18n.T("bottom.action"), i18n.T("bottom.note"), i18n.T("bottom.quit")},
		{i18n.T("bottom.action"), i18n.T("bottom.quit")},
	}
	left := renderShortcutSet(tiers[len(tiers)-1])
	for _, tier := range tiers {
		candidate := renderShortcutSet(tier)
		if lipgloss.Width(candidate) <= max(0, a.width-1) {
			left = candidate
			break
		}
	}

	rightStr := right

	gap := a.width - lipgloss.Width(left) - lipgloss.Width(rightStr) - 1
	if gap < 2 {
		rightStr = ""
		gap = max(0, a.width-lipgloss.Width(left))
	}

	barText := left + strings.Repeat(" ", gap) + rightStr

	return lipgloss.NewStyle().Foreground(colorBarText).Width(max(0, a.width)).Render(barText)
}

func (a *App) contextShortcuts() []string {
	switch a.state {
	case viewAddFile:
		return []string{"Tab " + i18n.T("common.next"), "Enter " + i18n.T("common.confirm"), "Esc " + i18n.T("common.back")}
	case viewSettings:
		return []string{"Tab " + i18n.T("common.next"), "Enter " + i18n.T("common.save"), "Esc " + i18n.T("common.back")}
	case viewSetDir:
		return []string{"Enter " + i18n.T("common.confirm"), "Esc " + i18n.T("common.back")}
	case viewConfirm:
		return []string{"Tab " + i18n.T("common.select"), "Enter " + i18n.T("common.confirm"), "Esc " + i18n.T("common.cancel")}
	case viewExportConfig:
		return []string{i18n.T("bottom.navigate"), "Esc " + i18n.T("common.close")}
	case viewHelp, viewNote, viewSyncResult:
		return []string{i18n.T("bottom.navigate"), "Esc " + i18n.T("common.back")}
	default:
		return []string{"Esc " + i18n.T("common.back")}
	}
}

func (a *App) contextFallbackShortcut() string {
	switch a.state {
	case viewConfirm:
		return "Esc " + i18n.T("common.cancel")
	case viewExportConfig:
		return "Esc " + i18n.T("common.close")
	default:
		return "Esc " + i18n.T("common.back")
	}
}

func renderShortcutSet(parts []string) string {
	styled := make([]string, 0, len(parts))
	for _, part := range parts {
		styled = append(styled, renderShortcut(part))
	}
	return " " + strings.Join(styled, " ") + " "
}

func renderShortcut(shortcut string) string {
	parts := strings.SplitN(shortcut, " ", 2)
	if len(parts) != 2 {
		return shortcut
	}
	key := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(parts[0])
	label := lipgloss.NewStyle().Foreground(colorBarText).Render(parts[1])
	return key + " " + label
}

// renderAutoSyncCountdown returns a colored countdown display.
func (a *App) renderAutoSyncCountdown() string {
	countdown := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(fmt.Sprintf("%ds", a.autoCountdown))
	if a.autoCountdown <= 5 {
		countdown = lipgloss.NewStyle().Bold(true).Foreground(colorWarning).Render(fmt.Sprintf("%ds", a.autoCountdown))
	}
	if a.autoCountdown <= 2 {
		countdown = lipgloss.NewStyle().Bold(true).Foreground(colorDanger).Render(fmt.Sprintf("%ds", a.autoCountdown))
	}
	return fmt.Sprintf("AUTO %s", countdown)
}

// ── File list ───────────────────────────────────────────────────────────────

func (a *App) computeVisibleRows() int {
	used := 6 // title rail, separator, selected-row detail, and bottom bar
	if !a.compact() {
		used++ // column header
	}
	if a.compact() && a.selectedItemHasNote() {
		used++ // selected-row note preview
	}
	if a.toast != "" {
		used++
	}
	n := a.height - used
	if len(a.fileList) > n {
		n-- // page indicator
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (a *App) renderFileList() string {
	total := len(a.fileList)
	a.pageRows = a.computeVisibleRows()
	if total == 0 {
		return a.renderEmpty()
	}
	a.clampPage()
	start := a.pageOffset
	end := start + a.pageRows
	if end > total {
		end = total
	}

	var b strings.Builder

	// ── Top header and status rail ──
	_, matched, pending, unbound := a.countStats()
	title := " " + styleStrong.Render("SFS")
	if !a.compact() {
		title += styleMuted.Render("  SMALL FILE SYNC")
	}
	title += styleMuted.Render("  v" + model.AppVersion)
	b.WriteString(title)
	b.WriteString("\n\n")

	// ── Status rail: stats (left) · auto-sync (right) ──
	chipStyle := lipgloss.NewStyle().Foreground(colorMuted)
	chipStyleSuccess := lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	chipStyleWarning := lipgloss.NewStyle().Bold(true).Foreground(colorWarning)

	// Left: file stats
	type statPart struct {
		text  string
		style lipgloss.Style
	}
	stats := []statPart{
		{fmt.Sprintf(i18n.T("file_list.files_count"), total), styleStrong},
		{"● " + fmt.Sprintf(i18n.T("file_list.stats.matched_short"), matched), chipStyleSuccess},
		{"● " + fmt.Sprintf(i18n.T("file_list.stats.pending_short"), pending), chipStyleWarning},
	}
	if unbound > 0 && !a.compact() {
		stats = append(stats, statPart{"● " + fmt.Sprintf(i18n.T("file_list.stats.unbound_short"), unbound), chipStyle})
	}
	renderStats := func(parts []statPart) string {
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			out = append(out, part.style.Render(part.text))
		}
		return strings.Join(out, "  ")
	}
	left := renderStats(stats)
	for len(stats) > 1 && lipgloss.Width(left) > max(2, a.width-3) {
		stats = stats[:len(stats)-1]
		left = renderStats(stats)
	}

	// Right: auto-sync status
	right := ""
	if a.autoSync {
		autoLabel := i18n.T("file_list.auto_sync_on")
		if a.syncing {
			autoLabel = i18n.T("file_list.syncing")
		}
		countdownStr := fmt.Sprintf("%ds", a.autoCountdown)
		if a.autoCountdown <= 5 {
			countdownStr = styleWarning.Render(countdownStr)
		} else {
			countdownStr = stylePrimary.Render(countdownStr)
		}
		right = chipStyle.Render("AUTO ") + autoLabel + "  " + countdownStr
		if !a.lastSyncTime.IsZero() {
			right += chipStyle.Render(fmt.Sprintf("  ·  %s %s", i18n.T("file_list.last_sync"), a.lastSyncTime.Format("15:04:05")))
		}
	}

	// Hide the clock before allowing it to collide with the status summary.
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	if leftW+rightW+5 > a.width {
		right = ""
		rightW = 0
	}
	gap := a.width - leftW - rightW - 3
	if gap < 1 {
		gap = 1
	}
	b.WriteString(" " + left + strings.Repeat(" ", gap) + right + " ")
	b.WriteString("\n")
	b.WriteString(" " + separator(max(2, a.width-2)))
	b.WriteString("\n")
	if !a.compact() {
		b.WriteString(a.fileListHeader())
		b.WriteString("\n")
	}

	// ── List rows ──
	for i := start; i < end; i++ {
		item := a.fileList[i]
		state := a.computeFileState(item)
		selected := i == a.cursor
		b.WriteString(a.fileLine(i+1, item, state, selected))
		b.WriteString("\n")
		if selected {
			b.WriteString(a.detailLine(item, state))
			b.WriteString("\n")
			if a.compact() && strings.TrimSpace(item.Note) != "" {
				b.WriteString(a.notePreviewLine(item.Note))
				b.WriteString("\n")
			}
		}
	}

	// ── Page indicator ──
	if total > a.pageRows {
		pages := (total + a.pageRows - 1) / a.pageRows
		curPage := 0
		if a.pageRows > 0 {
			curPage = a.pageOffset / a.pageRows
		}
		b.WriteString(styleMuted.Render(fmt.Sprintf(i18n.T("file_list.page_indicator"), curPage+1, pages)))
	}

	return b.String()
}

func (a *App) selectedItemHasNote() bool {
	return a.cursor >= 0 && a.cursor < len(a.fileList) && strings.TrimSpace(a.fileList[a.cursor].Note) != ""
}

func (a *App) notePreviewLine(note string) string {
	label := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(i18n.T("note.label") + ": ")
	available := max(1, a.width-5-lipgloss.Width(i18n.T("note.label")+": "))
	return "     " + label + styleMuted.Render(truncateText(strings.TrimSpace(note), available))
}

func (a *App) clampPage() {
	total := len(a.fileList)
	if a.cursor < a.pageOffset {
		a.pageOffset = a.cursor
	}
	if a.cursor >= a.pageOffset+a.pageRows {
		a.pageOffset = a.cursor - a.pageRows + 1
	}
	if a.pageOffset < 0 {
		a.pageOffset = 0
	}
	if a.pageRows > 0 && a.pageOffset > total-a.pageRows {
		a.pageOffset = max(0, total-a.pageRows)
	}
}

func (a *App) fileLine(idx int, item model.FileRecord, state model.FileStatus, selected bool) string {
	// ── Status ──
	var stText string
	var stColor lipgloss.TerminalColor
	var stIcon string
	switch state.Key {
	case "matched":
		stText, stColor, stIcon = i18n.T("status.matched"), colorSuccess, "✔"
	case "pending_upload":
		stText, stColor, stIcon = i18n.T("status.pending_upload"), colorWarning, "↑"
	case "download":
		stText, stColor, stIcon = i18n.T("status.download"), colorPrimary, "↓"
	case "initial_upload":
		stText, stColor, stIcon = i18n.T("status.initial_upload"), colorWarning, "★"
	case "missing":
		stText, stColor, stIcon = i18n.T("status.missing"), colorDanger, "✕"
	case "conflict":
		stText, stColor, stIcon = i18n.T("status.conflict"), colorDanger, "!"
	case "unbound":
		stText, stColor, stIcon = i18n.T("status.unbound"), colorMuted, "○"
	default:
		stText, stColor, stIcon = "", colorMuted, " "
	}

	cursorS := "  "
	if selected {
		cursorS = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("> ")
	}
	iconStyle := lipgloss.NewStyle().Foreground(stColor)
	if selected {
		iconStyle = iconStyle.Bold(true)
	}

	if a.compact() {
		name := truncateText(item.FileName, max(4, a.width-5))
		nameStyle := lipgloss.NewStyle()
		if selected {
			nameStyle = nameStyle.Bold(true).Foreground(colorPrimary)
		}
		return cursorS + iconStyle.Render(stIcon) + " " + nameStyle.Render(name)
	}

	cols := a.fileListColumns()
	idxS := lipgloss.NewStyle().Foreground(colorMuted).Width(cols.indexW).Align(lipgloss.Right).Render(fmt.Sprintf("%d", idx))
	nameStyle := lipgloss.NewStyle().Width(cols.nameW)
	statusStyle := lipgloss.NewStyle().Foreground(stColor).Width(cols.statusW)
	metaStyle := lipgloss.NewStyle().Foreground(colorMuted)
	timeStyle := lipgloss.NewStyle().Foreground(colorDim)
	if selected {
		nameStyle = nameStyle.Bold(true).Foreground(colorPrimary)
		statusStyle = statusStyle.Bold(true).Foreground(colorPrimary)
		metaStyle = metaStyle.Bold(true).Foreground(colorPrimary)
		timeStyle = timeStyle.Bold(true).Foreground(colorPrimary)
	}

	size := "-"
	if item.Size > 0 {
		size = formatKB(item.Size)
	}
	updated := "-"
	if item.LastUploadTime > 0 {
		updated = time.UnixMilli(item.LastUploadTime).Format("01-02 15:04")
	}

	line := cursorS + idxS + " " + iconStyle.Render(stIcon) + " "
	line += nameStyle.Render(truncateText(item.FileName, cols.nameW))
	line += columnGap + statusStyle.Render(truncateText(stText, cols.statusW))
	line += columnGap + metaStyle.Width(cols.sizeW).Align(lipgloss.Right).Render(truncateText(size, cols.sizeW))
	line += columnGap + timeStyle.Width(cols.updatedW).Render(updated)
	if cols.noteW > 0 {
		line += columnGap + metaStyle.Width(cols.noteW).Render(truncateText(item.Note, cols.noteW))
	}
	return line
}

const columnGap = "  "

type fileListColumnLayout struct {
	indexW   int
	nameW    int
	statusW  int
	sizeW    int
	updatedW int
	noteW    int
}

func (a *App) fileListColumns() fileListColumnLayout {
	cols := fileListColumnLayout{
		indexW:   max(2, lipgloss.Width(fmt.Sprintf("%d", len(a.fileList)))),
		statusW:  8,
		sizeW:    8,
		updatedW: 11,
	}
	prefixW := 2 + cols.indexW + 3 // cursor, index gap, icon, and icon gap
	fixedW := prefixW + cols.statusW + cols.sizeW + cols.updatedW + 3*lipgloss.Width(columnGap)
	cols.nameW = max(12, min(36, a.width-1-fixedW))
	if a.width >= 92 {
		cols.noteW = max(0, a.width-1-fixedW-cols.nameW-lipgloss.Width(columnGap))
	}
	return cols
}

func (a *App) fileListHeader() string {
	cols := a.fileListColumns()
	prefixW := 2 + cols.indexW + 3
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
	line := strings.Repeat(" ", prefixW)
	line += headerStyle.Width(cols.nameW).Render(i18n.T("file_list.header_file"))
	line += columnGap + headerStyle.Width(cols.statusW).Render(i18n.T("file_list.header_status"))
	line += columnGap + headerStyle.Width(cols.sizeW).Align(lipgloss.Right).Render(i18n.T("file_list.header_size"))
	line += columnGap + headerStyle.Width(cols.updatedW).Render(i18n.T("file_list.header_updated"))
	if cols.noteW > 0 {
		line += columnGap + headerStyle.Width(cols.noteW).Render(i18n.T("file_list.header_note"))
	}
	return line
}

func (a *App) detailLine(item model.FileRecord, state model.FileStatus) string {
	localDir := a.localDirMap[item.ID]
	fullPath := ""
	if localDir != "" {
		fullPath = filepath.Join(localDir, item.FileName)
	}

	parts := []string{}
	if fullPath != "" {
		parts = append(parts, fullPath)
	} else {
		parts = append(parts, i18n.T("file_list.no_dir"))
	}
	if item.LastUploadUser != "" {
		parts = append(parts, item.LastUploadUser)
	}
	parts = append(parts, state.Detail)

	indent := "       "
	if a.compact() {
		indent = "     "
	} else {
		indent = strings.Repeat(" ", 2+a.fileListColumns().indexW+3)
	}
	line := truncateText(indent+strings.Join(parts, " · "), max(1, a.width-1))
	// Pass selected flag from call site — we always show detail only for the selected row,
	// so use a distinct soft-blue so it doesn't clash with the white-highlighted filename.
	return lipgloss.NewStyle().Foreground(colorMuted).Render(line)
}

func truncateText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	var b strings.Builder
	for _, r := range s {
		candidate := b.String() + string(r)
		if lipgloss.Width(candidate)+3 > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "..."
}

func (a *App) countStats() (total, matched, pending, unbound int) {
	for _, item := range a.fileList {
		total++
		s := a.computeFileState(item)
		switch s.Key {
		case "matched":
			matched++
		case "unbound":
			unbound++
			pending++
		default:
			pending++
		}
	}
	return
}

// ─── Empty ───────────────────────────────────────────────────────────────────

func (a *App) renderEmpty() string {
	var b strings.Builder
	title := " " + styleStrong.Render("SFS")
	if a.compact() {
		title += styleMuted.Render("  v" + model.AppVersion)
	} else {
		title += styleMuted.Render("  SMALL FILE SYNC  /  v" + model.AppVersion)
	}
	b.WriteString(title + "\n" + separator(max(2, a.width-2)) + "\n\n")
	if !a.isStorageConfigured() {
		b.WriteString(i18n.T("empty.no_storage") + "\n\n")
		b.WriteString(i18n.T("empty.press"))
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("s"))
		b.WriteString(i18n.T("empty.configure"))
	} else {
		b.WriteString(i18n.T("empty.no_files") + "\n\n")
		b.WriteString(i18n.T("empty.press"))
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("a"))
		b.WriteString(i18n.T("empty.add_hint"))
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(i18n.T("empty.drag_hint")))
	}
	return b.String()
}

// ─── Add file ────────────────────────────────────────────────────────────────

func (a *App) renderAddFile() string {
	var b strings.Builder

	labelW := 6
	inputW := a.inputWidth(labelW)
	a.addFileInputs[0].Width = inputW
	a.addFileInputs[1].Width = inputW

	if a.addFileEditMode {
		// ── Edit mode: directory + note, no path validation ──
		b.WriteString(a.viewHeader(i18n.T("edit_file.title")))
		b.WriteString(responsiveGap(a.compact()))
		b.WriteString(a.fieldLine(i18n.T("add_file.label.path"), a.addFileInputs[0], a.addFileFocus == 0, labelW))
		b.WriteString(responsiveGap(a.compact()))
		b.WriteString(a.fieldLine(i18n.T("add_file.label.note"), a.addFileInputs[1], a.addFileFocus == 1, labelW))
		if a.addFileFeedback != "" {
			b.WriteString("\n\n  " + a.addFileFeedback)
		}
		b.WriteString(a.viewFooter(i18n.T("edit_file.hint")))
		return b.String()
	}

	// ── Add mode: path + note with validation ──
	b.WriteString(a.viewHeader(i18n.T("add_file.title")))
	b.WriteString(responsiveGap(a.compact()))
	b.WriteString(a.fieldLine(i18n.T("add_file.label.path"), a.addFileInputs[0], a.addFileFocus == 0, labelW))
	b.WriteString("\n")
	if a.addFileStats != nil {
		validMsg := fmt.Sprintf(i18n.T("add_file.path_valid"), a.addFileStats.Name(), formatBytes(a.addFileStats.Size()))
		indent := "        "
		if a.compact() {
			indent = "    "
		}
		b.WriteString("\n" + indent + styleSuccess.Render(truncateText(validMsg, max(1, a.width-lipgloss.Width(indent)-2))))
		b.WriteString(responsiveGap(a.compact()))
	} else if a.addFilePath != "" {
		indent := "        "
		if a.compact() {
			indent = "    "
		}
		b.WriteString("\n" + indent + styleDanger.Render(truncateText(i18n.T("add_file.path_invalid"), max(1, a.width-lipgloss.Width(indent)-2))))
		b.WriteString(responsiveGap(a.compact()))
	} else {
		b.WriteString(responsiveGap(a.compact()))
	}
	b.WriteString(a.fieldLine(i18n.T("add_file.label.note"), a.addFileInputs[1], a.addFileFocus == 1, labelW))
	if a.addFileFeedback != "" {
		b.WriteString("\n\n  " + a.addFileFeedback)
	}
	b.WriteString(a.viewFooter(i18n.T("add_file.hint")))
	return b.String()
}

// ─── Set directory ───────────────────────────────────────────────────────────

func (a *App) renderSetDir() string {
	labelW := 8
	a.setDirInput.Width = a.inputWidth(labelW)
	var b strings.Builder
	b.WriteString(a.viewHeader(i18n.T("set_dir.title")))
	b.WriteString(responsiveGap(a.compact()))
	b.WriteString(a.fieldLine(i18n.T("set_dir.label"), a.setDirInput, true, labelW))
	if a.setDirFeedback != "" {
		b.WriteString("\n\n  " + a.setDirFeedback)
	}
	b.WriteString(a.viewFooter(i18n.T("set_dir.hint")))
	return b.String()
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (a *App) renderSettings() string {
	labels := []string{
		i18n.T("settings.label.endpoint"),
		i18n.T("settings.label.username"),
		i18n.T("settings.label.password"),
		i18n.T("settings.label.base_path"),
	}
	labelW := 12
	if a.compact() {
		labelW = 10
	}
	for i := range a.settingsInputs {
		a.settingsInputs[i].Width = a.inputWidth(labelW)
	}
	var b strings.Builder
	b.WriteString(a.viewHeader(i18n.T("settings.title")))
	b.WriteString(responsiveGap(a.compact()))
	for i, label := range labels {
		b.WriteString(a.fieldLine(label, a.settingsInputs[i], a.settingsFocus == i, labelW))
		if i == 2 && a.settingsFocus == 2 {
			passwordHint := styleMuted.Render(i18n.T("settings.password_hide"))
			if !a.showPassword {
				passwordHint = styleMuted.Render(i18n.T("settings.password_show"))
			}
			indent := "          "
			if a.compact() {
				indent = "    "
			}
			b.WriteString("\n" + indent + passwordHint)
		}
		if i == 3 && !a.compact() {
			b.WriteString("\n          " + styleMuted.Render(i18n.T("settings.base_path_default")))
		}
		b.WriteString(responsiveGap(a.compact()))
	}
	if a.settingsFeedback != "" {
		b.WriteString("  " + a.settingsFeedback + "\n\n")
	}
	b.WriteString(a.viewFooter(i18n.T("settings.hint")))
	return b.String()
}

func responsiveGap(compact bool) string {
	// One empty line keeps form groups readable at every terminal width.
	return "\n\n"
}

// ─── Confirm (full-page view) ───────────────────────────────────────────────

func (a *App) renderConfirmView() string {
	var b strings.Builder

	// Title
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorDanger).Render("  " + a.confirmTitle))
	b.WriteString("\n" + separator(a.contentWidth()) + "\n\n")

	// Message body
	for _, line := range wrapText(a.confirmMsg, max(1, a.width-4)) {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")

	// Action buttons
	if a.confirmAction != nil {
		var confirmBtn, cancelBtn string
		if a.confirmFocus == 0 {
			confirmBtn = pillBtnHighlight(fmt.Sprintf("y %s", a.confirmLabel))
			cancelBtn = pillBtn(fmt.Sprintf("n %s", i18n.T("common.cancel")), colorMuted)
		} else {
			confirmBtn = pillBtn(fmt.Sprintf("y %s", a.confirmLabel), colorDanger)
			cancelBtn = pillBtnHighlight(fmt.Sprintf("n %s", i18n.T("common.cancel")))
		}
		b.WriteString("  " + confirmBtn + "   " + cancelBtn + "\n")
	} else {
		b.WriteString("  " + pillBtnHighlight(i18n.T("common.close")))
	}

	b.WriteString(a.viewFooter(i18n.T("confirm.hint_close")))
	return b.String()
}

// ─── Help (full-page view) ──────────────────────────────────────────────────

func (a *App) renderHelpView() string {
	var b strings.Builder
	b.WriteString(a.viewHeader(i18n.T("help.title")))

	sections := []struct {
		title string
		items []string
	}{
		{i18n.T("help.nav"), []string{i18n.T("help.nav.up"), i18n.T("help.nav.down"), i18n.T("help.nav.page_up"), i18n.T("help.nav.page_down"), i18n.T("help.nav.first_last")}},
		{i18n.T("help.ops"), []string{i18n.T("help.ops.execute"), i18n.T("help.ops.upload"), i18n.T("help.ops.download"), i18n.T("help.ops.delete"), i18n.T("help.ops.note"), i18n.T("help.ops.set_dir")}},
		{i18n.T("help.features"), []string{i18n.T("help.features.add"), i18n.T("help.features.settings"), i18n.T("help.features.sync_all"), i18n.T("help.features.auto_sync"), i18n.T("help.features.refresh"), i18n.T("help.features.update"), i18n.T("help.features.lang"), i18n.T("help.features.quit")}},
		{i18n.T("help.general"), []string{i18n.T("help.general.copy"), i18n.T("help.general.password")}},
	}

	sectionTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	keyW := 14
	if a.compact() {
		keyW = 12
	}
	keyStyle := lipgloss.NewStyle().Foreground(colorBarText).Width(keyW)
	actionStyle := lipgloss.NewStyle().Foreground(colorMuted)
	actionW := max(1, a.width-8-keyW)

	for si, sec := range sections {
		if si > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  " + sectionTitleStyle.Render(sec.title))
		for _, item := range sec.items {
			parts := strings.SplitN(item, "  ", 2)
			if len(parts) == 2 {
				b.WriteString("\n    " + keyStyle.Render(parts[0]) + actionStyle.Render(truncateText(parts[1], actionW)))
			} else {
				b.WriteString("\n    " + keyStyle.Render(truncateText(item, actionW)))
			}
		}
	}

	b.WriteString(a.viewFooter(i18n.T("help.hint_close")))
	return b.String()
}

// ─── Note (full-page view) ──────────────────────────────────────────────────

func (a *App) renderNoteView() string {
	var b strings.Builder
	item := a.fileList[a.cursor]
	state := a.computeFileState(item)

	// ── Title ──
	b.WriteString(a.viewHeader("[NOTE] " + item.FileName))
	if a.compact() {
		b.WriteString("\n  " + styleStrong.Render(i18n.T("note.label")) + "\n")
		note := strings.TrimSpace(item.Note)
		if note == "" {
			note = i18n.T("note.empty")
		}
		for _, line := range wrapText(note, max(1, a.width-4)) {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n" + separator(a.contentWidth()) + "\n")
	}

	// ── Status line ──
	statusParts := []string{
		styleMuted.Render(i18n.T("note.status")+": ") + state.Text,
	}
	if item.Size > 0 {
		statusParts = append(statusParts, styleMuted.Render(i18n.T("note.size")+": ")+formatKB(item.Size))
	}
	if item.LastUploadTime > 0 {
		statusParts = append(statusParts, styleMuted.Render(i18n.T("note.uploaded")+": ")+time.UnixMilli(item.LastUploadTime).Format("2006-01-02 15:04"))
	}
	if item.LastUploadUser != "" {
		statusParts = append(statusParts, styleMuted.Render(i18n.T("note.user")+": ")+item.LastUploadUser)
	}
	b.WriteString("\n")
	if a.compact() {
		for _, p := range statusParts {
			b.WriteString("  " + p + "\n")
		}
	} else {
		b.WriteString("  " + strings.Join(statusParts, "  ") + "\n")
	}

	// ── Local path ──
	b.WriteString("\n")
	dir := a.localDirMap[item.ID]
	pathVal := filepath.Join(dir, item.FileName)
	if dir == "" {
		pathVal = i18n.T("file_list.no_dir")
	}
	b.WriteString("  " + styleMuted.Render(i18n.T("note.path")+": ") + truncateText(pathVal, max(1, a.width-6)))
	b.WriteString("\n\n")

	// ── Note ──
	if !a.compact() {
		note := item.Note
		if note == "" {
			note = i18n.T("note.empty")
		}
		b.WriteString(separator(a.contentWidth()))
		b.WriteString("\n")
		for _, line := range wrapText(note, max(1, a.width-4)) {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString(a.viewFooter(i18n.T("note.hint_close")))

	return b.String()
}

// ─── Export config view ──────────────────────────────────────────────────────

func (a *App) renderExportConfigView() string {
	var b strings.Builder

	// ── Title ──
	b.WriteString(a.viewHeader(i18n.T("export.title")))
	b.WriteString("\n")

	// ── Copied hint ──
	b.WriteString("  " + styleSuccess.Render(i18n.T("export.copied_hint")))
	b.WriteString("\n")

	// ── Temp file fallback ──
	if a.exportTempFile != "" {
		b.WriteString("  " + styleMuted.Render(i18n.T("export.temp_file")+": "+a.exportTempFile))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// ── Command box ──
	b.WriteString("  " + styleMuted.Render(i18n.T("export.command_label")+":"))
	b.WriteString("\n")
	boxWidth := min(72, max(8, a.width-10))
	b.WriteString("    " + strings.Repeat("\u2500", boxWidth) + "\n")
	wrapped := wrapString(a.exportCommand, boxWidth)
	for _, line := range wrapped {
		b.WriteString("    " + line + "\n")
	}
	b.WriteString("    " + strings.Repeat("\u2500", boxWidth) + "\n")
	b.WriteString("\n")

	// ── Instruction ──
	for _, line := range wrapText(i18n.T("export.instruction"), max(1, a.width-4)) {
		b.WriteString("  " + styleMuted.Render(line) + "\n")
	}
	b.WriteString("\n")

	// ── Security warning ──
	for _, line := range wrapText(i18n.T("export.security_warning"), max(1, a.width-4)) {
		b.WriteString("  " + styleDanger.Render(line) + "\n")
	}
	b.WriteString(a.viewFooter(i18n.T("export.close_hint")))

	return b.String()
}

// wrapText splits a string into display-width-aware lines no longer than width.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	cur := ""
	curW := 0
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if curW+w > width && cur != "" {
			lines = append(lines, strings.TrimRight(cur, " "))
			cur = ""
			curW = 0
		}
		cur += string(r)
		curW += w
	}
	if cur != "" {
		lines = append(lines, strings.TrimRight(cur, " "))
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// wrapString splits a long string into lines no longer than width,
// breaking only at spaces (or at width if no space is found).
func wrapString(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	runes := []rune(s)
	for len(runes) > width {
		breakAt := width
		for i := width; i >= 0; i-- {
			if runes[i] == ' ' {
				breakAt = i
				break
			}
		}
		lines = append(lines, string(runes[:breakAt]))
		runes = runes[breakAt:]
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	return lines
}

// ─── Sync result (full-page view) ────────────────────────────────────────────

func (a *App) renderSyncResultView() string {
	result := a.lastSyncResult
	if result == nil {
		return ""
	}
	return a.renderSyncResult(*result)
}

func (a *App) renderSyncResult(result model.SyncResult) string {
	s := result.Summary

	var b strings.Builder

	// ── Title ──
	title := i18n.T("sync.result.title")
	if result.IsAuto {
		title += " " + i18n.T("sync.result.auto")
	}
	b.WriteString(a.viewHeader(title))

	// ── Progress bar (when still syncing) ──
	if a.syncing && len(a.syncItems) > 0 {
		total := len(a.syncItems)
		done := a.syncIndex
		if done > total {
			done = total
		}
		barWidth := min(40, max(4, a.width-10))
		filled := 0
		if total > 0 {
			filled = done * barWidth / total
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		b.WriteString("\n" + fmt.Sprintf("  [%s] %d/%d", bar, done, total))
		if done < total {
			curItem := a.syncItems[done]
			b.WriteString("\n  " + styleMuted.Render(i18n.T("sync.processing")+": ") + truncateText(curItem.FileName, max(1, a.width-6)))
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString("\n")
	}

	// ── Summary row ──
	b.WriteString("  ")
	summaryItems := []string{}
	if s.Checked > 0 {
		summaryItems = append(summaryItems, fmt.Sprintf(i18n.T("sync.result.checked"), s.Checked))
	}
	if s.Uploaded > 0 {
		summaryItems = append(summaryItems, styleSuccess.Render(fmt.Sprintf(i18n.T("sync.result.uploaded"), s.Uploaded)))
	}
	if s.Downloaded > 0 {
		summaryItems = append(summaryItems, styleSuccess.Render(fmt.Sprintf(i18n.T("sync.result.downloaded"), s.Downloaded)))
	}
	if s.Skipped > 0 {
		summaryItems = append(summaryItems, styleMuted.Render(fmt.Sprintf(i18n.T("sync.result.skipped"), s.Skipped)))
	}
	if s.Failed > 0 {
		summaryItems = append(summaryItems, styleDanger.Render(fmt.Sprintf(i18n.T("sync.result.failed"), s.Failed)))
	}
	if s.Unbound > 0 {
		summaryItems = append(summaryItems, styleMuted.Render(fmt.Sprintf(i18n.T("sync.result.unbound"), s.Unbound)))
	}
	summarySep := "  "
	if a.compact() {
		summarySep = " "
	}
	summaryLine := strings.Join(summaryItems, summarySep)
	if lipgloss.Width(summaryLine) > max(1, a.width-2) {
		summaryLine = truncateText(summaryLine, max(1, a.width-2))
	}
	b.WriteString(summaryLine)
	b.WriteString("\n")

	// ── Detail table ──
	if len(result.Details) > 0 {
		b.WriteString("\n")
		sepW := max(2, a.width-8)
		b.WriteString("  " + separator(sepW))
		b.WriteString("\n")

		// Sort: failures first
		sorted := make([]model.SyncDetail, len(result.Details))
		copy(sorted, result.Details)
		failLabel := i18n.T("common.failure")
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].Status == failLabel && sorted[i].Status != failLabel {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		compact := a.compact()
		fileW := 14
		actionW := 8
		if compact {
			fileW, actionW = 10, 6
		}
		reasonW := max(1, a.width-8-fileW-actionW-6)
		hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
		fileHdr := hdrStyle.Width(fileW).Render(i18n.T("sync.result.header_file"))
		actionHdr := hdrStyle.Width(actionW).Render(i18n.T("sync.result.header_action"))
		resultHdr := hdrStyle.Width(reasonW).Render(i18n.T("sync.result.header_result"))
		b.WriteString("  " + fileHdr + "  " + actionHdr + "  " + resultHdr)
		b.WriteString("\n")
		b.WriteString("  " + separator(sepW))
		b.WriteString("\n")

		cellStyle := lipgloss.NewStyle().Foreground(colorBarText)
		for _, d := range sorted {
			mark := styleSuccess.Render("+")
			if d.Status == failLabel {
				mark = styleDanger.Render("x")
			}
			reasonStyle := styleMuted
			if d.Status == failLabel {
				reasonStyle = styleDanger
			}
			fileName := truncateText(d.FileName, fileW)
			action := truncateText(d.Action, actionW)
			reason := truncateText(d.Reason, reasonW)
			b.WriteString("  " + mark + "  " +
				cellStyle.Width(fileW).Render(fileName) + "  " +
				cellStyle.Width(actionW).Render(action) + "  " +
				reasonStyle.Render(reason) + "\n")
		}
		b.WriteString("  " + separator(sepW))
	}

	b.WriteString(a.viewFooter(func() string {
		if a.syncing {
			return i18n.T("sync.result.syncing_hint")
		}
		return i18n.T("sync.result.close_hint")
	}()))

	return b.String()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// formatBytes formats a byte count into a human-readable string.
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// formatKB formats a size stored in KB into a human-readable string.
func formatKB(kb float64) string {
	const (
		MB = 1024.0
		GB = MB * 1024
	)
	switch {
	case kb >= GB:
		return fmt.Sprintf("%.1fGB", kb/GB)
	case kb >= MB:
		return fmt.Sprintf("%.1fMB", kb/MB)
	default:
		if kb < 1 {
			return "<1KB"
		}
		return fmt.Sprintf("%.0fKB", kb)
	}
}
