package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color palette ────────────────────────────────────────────────────────────

var (
	colorPrimary   = adaptiveColor("#0F766E", "#2DD4BF", "30", "43", "6", "6")
	colorSuccess   = adaptiveColor("#15803D", "#4ADE80", "28", "78", "2", "2")
	colorWarning   = adaptiveColor("#A16207", "#FBBF24", "136", "220", "3", "3")
	colorDanger    = adaptiveColor("#BE123C", "#FB7185", "161", "204", "1", "1")
	colorMuted     = adaptiveColor("#64748B", "#94A3B8", "102", "145", "8", "8")
	colorBarText   = adaptiveColor("#334155", "#CBD5E1", "59", "252", "0", "7")
	colorHighlight = adaptiveColor("#0F172A", "#F8FAFC", "16", "231", "0", "15")
	colorDim       = adaptiveColor("#CBD5E1", "#334155", "252", "59", "7", "8")
	colorBorder    = adaptiveColor("#94A3B8", "#475569", "145", "102", "8", "8")
	colorOnAccent  = adaptiveColor("#FFFFFF", "#071B1A", "15", "16", "15", "0")
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
	return lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("-", width))
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
