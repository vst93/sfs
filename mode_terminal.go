package main

import (
	"fmt"
	"os"
	"smallFileSync/internal/i18n"
	"smallFileSync/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func startTerminalMode() {
	locale := i18n.DetectLocale()
	i18n.SetLocale(locale)

	app, err := ui.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("init.failed")+"\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, i18n.T("run.failed")+"\n", err)
		os.Exit(1)
	}
}
