package ui

import (
	"fmt"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/model"
	"smallFileSync/internal/util"
	"strings"
	"time"

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
	}
	if a.toast != "" {
		body.WriteString("\n")
		a.renderToast(&body)
	}

	bodyStr := body.String()
	bodyStr = strings.TrimRight(bodyStr, "\n")
	bodyLines := 0
	if bodyStr != "" {
		bodyLines = strings.Count(bodyStr, "\n") + 1
	}
	available := a.height - 1 // 1 line for bottom bar
	padding := available - bodyLines
	if padding > 0 {
		bodyStr += strings.Repeat("\n", padding)
	}

	return bodyStr + "\n" + a.renderBottomBar()
}

// ── Toast ───────────────────────────────────────────────────────────────────

func (a *App) renderToast(b *strings.Builder) {
	var icon, label string
	var clr lipgloss.Color
	switch a.toastType {
	case "warning":
		icon, label, clr = "⚠", i18n.T("common.warning"), colorWarning
	case "error":
		icon, label, clr = "✗", i18n.T("common.error"), colorDanger
	default:
		icon, label, clr = "✓", i18n.T("common.success"), colorSuccess
	}
	tag := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(clr).
		Padding(0, 1).
		Render(" " + icon + " " + label + " ")
	msg := lipgloss.NewStyle().Foreground(clr).Render(a.toast)
	b.WriteString("  " + tag + " " + msg)
}

// ── Bottom bar ──────────────────────────────────────────────────────────────

func (a *App) renderBottomBar() string {
	storage := i18n.T("bottom.storage_unconfigured")
	if a.isStorageConfigured() {
		storage = "WebDAV"
	}

	right := storage
	if a.syncing {
		right = i18n.T("bottom.syncing")
	} else if a.autoSync {
		right = fmt.Sprintf(i18n.T("bottom.auto_sync"), a.autoCountdown, storage)
	}
	// Show update badge if new version available
	if a.updateDone && a.updateResult != nil && a.updateResult.HasUpdate {
		right = styleWarning.Render(fmt.Sprintf("↑ %s", a.updateResult.LatestVersion)) + "  " + right
	}

	parts := []string{
		i18n.T("bottom.navigate"),
		i18n.T("bottom.sync"),
		i18n.T("bottom.upload"),
		i18n.T("bottom.download"),
		i18n.T("bottom.delete"),
		i18n.T("bottom.note"),
		i18n.T("bottom.dir"),
		i18n.T("bottom.add"),
		i18n.T("bottom.settings"),
		i18n.T("bottom.sync_all"),
		i18n.T("bottom.auto"),
		i18n.T("bottom.refresh"),
		i18n.T("bottom.help"),
		i18n.T("bottom.lang"),
		i18n.T("bottom.quit"),
	}
	left := " " + strings.Join(parts, " · ") + " "

	rightStr := styleMuted.Render(right)

	gap := a.width - lipgloss.Width(left) - lipgloss.Width(rightStr)
	if gap < 1 {
		gap = 1
	}

	barText := left + strings.Repeat(" ", gap) + rightStr

	return lipgloss.NewStyle().Foreground(colorBarText).Render(barText)
}

// ── File list ───────────────────────────────────────────────────────────────

func (a *App) computeVisibleRows() int {
	used := 4
	if a.toast != "" {
		used++
	}
	n := a.height - used
	if n < 2 {
		n = 2
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

	// ── Top header ──
	_, matched, pending, unbound := a.countStats()
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(" " + model.AppFullName))
	b.WriteString(styleMuted.Render("  v" + model.AppVersion))
	b.WriteString("\n\n")

	// Stats line
	statsParts := []string{}
	statsParts = append(statsParts, fmt.Sprintf(i18n.T("file_list.files_count"), total))
	statsParts = append(statsParts, styleSuccess.Render(fmt.Sprintf(i18n.T("file_list.stats.matched"), matched)))
	statsParts = append(statsParts, styleWarning.Render(fmt.Sprintf(i18n.T("file_list.stats.pending"), pending)))
	if unbound > 0 {
		statsParts = append(statsParts, styleMuted.Render(fmt.Sprintf(i18n.T("file_list.stats.unbound"), unbound)))
	}
	b.WriteString("  " + strings.Join(statsParts, "  "))
	b.WriteString("\n")

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
		}
	}

	// ── Page indicator ──
	if total > a.pageRows {
		pages := (total + a.pageRows - 1) / a.pageRows
		curPage := 0
		if a.pageRows > 0 {
			curPage = a.pageOffset / a.pageRows
		}
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(fmt.Sprintf(i18n.T("file_list.page_indicator"), curPage+1, pages)))
	}

	return b.String()
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
	var stColor lipgloss.Color
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

	// ── Right part ──
	rightParts := []string{}
	if item.Size > 0 {
		if item.Size >= 1024 {
			rightParts = append(rightParts, fmt.Sprintf("%.1fM", item.Size/1024))
		} else {
			rightParts = append(rightParts, fmt.Sprintf("%.0fK", item.Size))
		}
	}
	if item.LastUploadTime > 0 {
		rightParts = append(rightParts, time.UnixMilli(item.LastUploadTime).Format("01-02 15:04"))
	}
	if item.Note != "" {
		rightParts = append(rightParts, item.Note)
	}
	rightPlain := strings.Join(rightParts, "  ")

	// ── Layout calc ──
	cursorPlain := "   "
	if selected {
		cursorPlain = " ▸ "
	}
	idxStr := fmt.Sprintf("%d", idx)
	prefixW := lipgloss.Width(cursorPlain) + 1 + lipgloss.Width(idxStr) + 1
	rightW := lipgloss.Width(rightPlain)
	nameW := a.width - prefixW - rightW - 6
	if nameW > 35 {
		nameW = 35
	}
	if nameW < 4 {
		nameW = 4
	}
	name := item.FileName
	if lipgloss.Width(name) > nameW {
		runes := []rune(name)
		if len(runes) > nameW-1 {
			name = string(runes[:nameW-1]) + "…"
		}
	}
	pad := nameW - lipgloss.Width(name)
	if pad < 0 {
		pad = 0
	}

	// ── Styled ──
	idxS := lipgloss.NewStyle().Foreground(colorMuted).Width(3).Align(lipgloss.Right).Render(idxStr)

	nameS := name
	if selected {
		nameS = lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(name)
	}

	rightS := styleMuted.Render(rightPlain)

	cursorS := "   "
	if selected {
		cursorS = lipgloss.NewStyle().Foreground(colorPrimary).Render(" ▸ ")
	}

	return cursorS + idxS + " " +
		lipgloss.NewStyle().Foreground(stColor).Render(stIcon) + " " +
		nameS + strings.Repeat(" ", pad) + "  " +
		lipgloss.NewStyle().Foreground(stColor).Width(12).Render(stText) +
		rightS
}

func (a *App) detailLine(item model.FileRecord, state model.FileStatus) string {
	localDir := a.localDirMap[item.ID]
	fullPath := ""
	if localDir != "" {
		fullPath = localDir + "/" + item.FileName
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

	line := "       " + strings.Join(parts, " · ")
	if len(line) > a.width {
		line = line[:a.width-1] + "…"
	}
	return styleMuted.Render(line)
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
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(" " + model.AppFullName))
	b.WriteString(styleMuted.Render("  v" + model.AppVersion))
	b.WriteString("\n\n\n")
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
	}
	return b.String()
}

// ─── Add file ────────────────────────────────────────────────────────────────

func (a *App) renderAddFile() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(i18n.T("add_file.title")))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Width(6)

	// Path field
	if a.addFileFocus == 0 {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorPrimary).Render("▸ ") + labelStyle.Render(i18n.T("add_file.label.path")) + ": ")
	} else {
		b.WriteString("    " + labelStyle.Render(i18n.T("add_file.label.path")) + ": ")
	}
	b.WriteString(a.addFileInputs[0].View())
	b.WriteString("\n")
	if a.addFileStats != nil {
		b.WriteString("        " + styleSuccess.Render(fmt.Sprintf(i18n.T("add_file.path_valid"), a.addFileStats.Name(), float64(a.addFileStats.Size())/1024)))
	} else if a.addFilePath != "" {
		b.WriteString("        " + styleDanger.Render(i18n.T("add_file.path_invalid")))
	}
	b.WriteString("\n\n")

	// Note field
	if a.addFileFocus == 1 {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorPrimary).Render("▸ ") + labelStyle.Render(i18n.T("add_file.label.note")) + ": ")
	} else {
		b.WriteString("    " + labelStyle.Render(i18n.T("add_file.label.note")) + ": ")
	}
	b.WriteString(a.addFileInputs[1].View())
	if a.addFileFeedback != "" {
		b.WriteString("\n\n  " + a.addFileFeedback)
	}
	b.WriteString("\n\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(i18n.T("add_file.hint")))
	return b.String()
}

// ─── Set directory ───────────────────────────────────────────────────────────

func (a *App) renderSetDir() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(i18n.T("set_dir.title")))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n\n")
	b.WriteString("  " + i18n.T("set_dir.label") + ": ")
	b.WriteString(a.setDirInput.View())
	if a.setDirFeedback != "" {
		b.WriteString("\n\n  " + a.setDirFeedback)
	}
	b.WriteString("\n\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(i18n.T("set_dir.hint")))
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
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(i18n.T("settings.title")))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Width(12)
	for i, label := range labels {
		if a.settingsFocus == i {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorPrimary).Render("▸ ") + labelStyle.Render(label) + ": ")
		} else {
			b.WriteString("    " + labelStyle.Render(label) + ": ")
		}
		b.WriteString(a.settingsInputs[i].View())
		if i == 2 && a.settingsFocus == 2 {
			if a.showPassword {
				b.WriteString("  " + styleMuted.Render(i18n.T("settings.password_hide")))
			} else {
				b.WriteString("  " + styleMuted.Render(i18n.T("settings.password_show")))
			}
		}
		if i == 3 {
			b.WriteString("\n          " + styleMuted.Render(i18n.T("settings.base_path_default")))
		}
		b.WriteString("\n\n")
	}
	if a.settingsFeedback != "" {
		b.WriteString("  " + a.settingsFeedback + "\n\n")
	}
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(i18n.T("settings.hint")))
	return b.String()
}

// ─── Confirm (full-page view) ───────────────────────────────────────────────

func (a *App) renderConfirmView() string {
	var b strings.Builder

	// Title
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorDanger).Render("  ⚠  " + a.confirmTitle))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n\n")

	// Message body
	b.WriteString("  " + a.confirmMsg)
	b.WriteString("\n\n")

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

	b.WriteString("\n\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  " + i18n.T("confirm.hint_close")))

	return b.String()
}

// ─── Help (full-page view) ──────────────────────────────────────────────────

func (a *App) renderHelpView() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(i18n.T("help.title")))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")

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
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d1d5db")).Width(14)
	actionStyle := lipgloss.NewStyle().Foreground(colorMuted)

	for si, sec := range sections {
		if si > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  " + sectionTitleStyle.Render(sec.title))
		for _, item := range sec.items {
			parts := strings.SplitN(item, "  ", 2)
			if len(parts) == 2 {
				b.WriteString("\n    " + keyStyle.Render(parts[0]) + actionStyle.Render(parts[1]))
			} else {
				b.WriteString("\n    " + keyStyle.Render(item))
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  " + i18n.T("help.hint_close")))

	return b.String()
}

// ─── Note (full-page view) ──────────────────────────────────────────────────

func (a *App) renderNoteView() string {
	var b strings.Builder
	item := a.fileList[a.cursor]
	state := a.computeFileState(item)

	// ── Title ──
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render("  📝  " + item.FileName))
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")

	// ── Status line ──
	b.WriteString("\n")
	b.WriteString("  " + styleMuted.Render(i18n.T("note.status")+": ") + state.Text)
	if item.Size > 0 {
		var sizeStr string
		if item.Size >= 1024 {
			sizeStr = fmt.Sprintf("%.1fM", item.Size/1024)
		} else {
			sizeStr = fmt.Sprintf("%.0fK", item.Size)
		}
		b.WriteString("  " + styleMuted.Render(i18n.T("note.size")+": ") + sizeStr)
	}
	if item.LastUploadTime > 0 {
		b.WriteString("  " + styleMuted.Render(i18n.T("note.uploaded")+": ") + time.UnixMilli(item.LastUploadTime).Format("2006-01-02 15:04"))
	}
	if item.LastUploadUser != "" {
		b.WriteString("  " + styleMuted.Render(i18n.T("note.user")+": ") + item.LastUploadUser)
	}
	b.WriteString("\n")

	// ── Local path ──
	b.WriteString("\n")
	dir := a.localDirMap[item.ID]
	if dir != "" {
		b.WriteString("  " + styleMuted.Render(i18n.T("note.path")+": ") + dir + util.FileSeparator() + item.FileName)
	} else {
		b.WriteString("  " + styleMuted.Render(i18n.T("note.path")+": ") + i18n.T("file_list.no_dir"))
	}
	b.WriteString("\n")

	// ── Note ──
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString("\n")
	if item.Note != "" {
		b.WriteString("  " + item.Note)
	} else {
		b.WriteString("  " + styleMuted.Render(i18n.T("note.empty")))
	}
	b.WriteString("\n\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  " + i18n.T("note.hint_close")))

	return b.String()
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
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Render(i18n.T("sync.result.title")))
	if result.IsAuto {
		b.WriteString(styleMuted.Render(i18n.T("sync.result.auto")))
	}
	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")

	// ── Progress bar (when still syncing) ──
	if a.syncing && len(a.syncItems) > 0 {
		total := len(a.syncItems)
		done := a.syncIndex
		if done > total {
			done = total
		}
		barWidth := min(40, a.width-10)
		if barWidth < 4 {
			barWidth = 4
		}
		filled := 0
		if total > 0 {
			filled = done * barWidth / total
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		progress := fmt.Sprintf("  [%s] %d/%d", bar, done, total)
		b.WriteString("\n" + progress)
		// Show current file being processed
		if done < total {
			curItem := a.syncItems[done]
			b.WriteString("\n  " + styleMuted.Render(i18n.T("sync.processing")+": ") + truncate(curItem.FileName, 30))
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
	b.WriteString(strings.Join(summaryItems, "  "))
	b.WriteString("\n")

	// ── Detail table ──
	if len(result.Details) > 0 {
		b.WriteString("\n")
		b.WriteString("  " + separator(a.width-8))
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

		// Table header
		hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
		b.WriteString("  " + hdrStyle.Render(fmt.Sprintf("%-4s  %-14s  %-8s  %s", "", i18n.T("sync.result.header_file"), i18n.T("sync.result.header_action"), i18n.T("sync.result.header_result"))))
		b.WriteString("\n")
		b.WriteString("  " + separator(a.width-8))
		b.WriteString("\n")

		for _, d := range sorted {
			mark := styleSuccess.Render("✔")
			if d.Status == failLabel {
				mark = styleDanger.Render("✕")
			}
			reasonStyle := styleMuted
			if d.Status == failLabel {
				reasonStyle = styleDanger
			}
			fileName := truncate(d.FileName, 14)
			b.WriteString(fmt.Sprintf("  %s  %-14s  %-8s  %s\n",
				mark,
				lipgloss.NewStyle().Render(fileName),
				lipgloss.NewStyle().Render(d.Action),
				reasonStyle.Render(d.Reason),
			))
		}
		b.WriteString("  " + separator(a.width-8))
	}

	b.WriteString("\n")
	b.WriteString(separator(a.width - 4))
	b.WriteString("\n")
	if a.syncing {
		b.WriteString(styleMuted.Render("  " + i18n.T("sync.result.syncing_hint")))
	} else {
		b.WriteString(styleMuted.Render("  " + i18n.T("sync.result.close_hint")))
	}

	return b.String()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
