package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color palette ────────────────────────────────────────────────────────────

var (
	colorPrimary    = adaptiveColor("#0F766E", "#2DD4BF", "30", "43", "6", "6")
	colorSuccess    = adaptiveColor("#15803D", "#4ADE80", "28", "78", "2", "2")
	colorWarning    = adaptiveColor("#A16207", "#FBBF24", "136", "220", "3", "3")
	colorDanger     = adaptiveColor("#BE123C", "#FB7185", "161", "204", "1", "1")
	colorMuted      = adaptiveColor("#58677A", "#94A3B8", "59", "145", "8", "8")
	colorBarText    = adaptiveColor("#334155", "#CBD5E1", "59", "252", "0", "7")
	colorSelectedBg = adaptiveColor("#DDF3F0", "#164E4A", "195", "23", "7", "0")
	colorDim        = adaptiveColor("#7C8799", "#64748B", "102", "102", "8", "8")
	colorBorder     = adaptiveColor("#64748B", "#64748B", "59", "102", "8", "8")
	colorOnAccent   = adaptiveColor("#FFFFFF", "#071B1A", "15", "16", "15", "0")
)

func adaptiveColor(light, dark, light256, dark256, lightANSI, darkANSI string) lipgloss.CompleteAdaptiveColor {
	return lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: light, ANSI256: light256, ANSI: lightANSI},
		Dark:  lipgloss.CompleteColor{TrueColor: dark, ANSI256: dark256, ANSI: darkANSI},
	}
}

// ── Reusable style primitives ────────────────────────────────────────────────

var (
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleDim     = lipgloss.NewStyle().Foreground(colorDim)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning)
	styleDanger  = lipgloss.NewStyle().Foreground(colorDanger)
	stylePrimary = lipgloss.NewStyle().Foreground(colorPrimary)
	// Strong text inherits the terminal's foreground. This stays readable even
	// when a terminal reports the wrong light/dark background mode.
	styleStrong = lipgloss.NewStyle().Bold(true)
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
	return lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("-", width))
}

// ── Styled button (pill badge) ─────────────────────────────────────────────

func pillBtn(label string, color lipgloss.TerminalColor) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render("[ " + label + " ]")
}

func pillBtnHighlight(label string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(colorOnAccent).
		Background(colorPrimary).
		Padding(0, 1).
		Render(" " + label + " ")
}
