package prompt

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const maxPromptInput = 300

func Blink() tea.Cmd {
	return textinput.Blink
}

type Model struct {
	input textinput.Model
}

func (m *Model) SetWidth(width int) {
	m.input.Width = width
}

func (m Model) GetInput() string {
	return m.input.Value()
}

func (m Model) GetPromptWidth() int {
	return len(m.input.Prompt)
}

func (m *Model) ResetContent() {
	m.input.Reset()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "alt+j", "alt+k", "alt+b", "alt+n", "alt+p":
			return m, nil
		default:
		}
	}

	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	return m.input.View()
}

func InitialModel() Model {
	m := Model{}

	m.input = textinput.New()
	m.input.Focus()
	m.input.CharLimit = maxPromptInput
	m.input.Prompt = "$ "
	m.input.KeyMap.LineEnd.SetEnabled(false)

	return m
}
