package ui

import tea "github.com/charmbracelet/bubbletea"

func Run() error {
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
	program := tea.NewProgram(initialModel(), options...)

	_, err := program.Run()
	return err
}
