package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/KirillSachkov/gvardia/internal/config"
)

// Run launches the cockpit and blocks until the user quits.
func Run(cfg config.Config) error {
	p := tea.NewProgram(New(cfg))
	_, err := p.Run()
	return err
}
