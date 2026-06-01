package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color palette ────────────────────────────────────────────────────────────

var (
	colorPrimary   = lipgloss.Color("#4A6CF7")
	colorSuccess   = lipgloss.Color("#027a48")
	colorWarning   = lipgloss.Color("#b54708")
	colorDanger    = lipgloss.Color("#d92d20")
	colorMuted     = lipgloss.Color("#667085")
	colorBarText   = lipgloss.Color("#9ca3af")
	colorHighlight = lipgloss.Color("#e0e7ff")
	colorDim       = lipgloss.Color("#4b5563")
	colorBorder    = lipgloss.Color("#6b7280")
)

// ── Reusable style primitives ────────────────────────────────────────────────

var (
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning)
	styleDanger  = lipgloss.NewStyle().Foreground(colorDanger)
	stylePrimary = lipgloss.NewStyle().Foreground(colorPrimary)
)

// ── Public text styles (used by other files) ────────────────────────────────

var (
	WarningText = lipgloss.NewStyle().Foreground(colorWarning)
	ErrorText   = lipgloss.NewStyle().Foreground(colorDanger)
	SuccessText = lipgloss.NewStyle().Foreground(colorSuccess)
)

// ── Separator ───────────────────────────────────────────────────────────────

func separator(width int) string {
	if width < 2 {
		width = 2
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("─", width))
}

// ── Styled button (pill badge) ─────────────────────────────────────────────

func pillBtn(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render("[ " + label + " ]")
}

func pillBtnHighlight(label string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(colorPrimary).
		Padding(0, 1).
		Render(" " + label + " ")
}
