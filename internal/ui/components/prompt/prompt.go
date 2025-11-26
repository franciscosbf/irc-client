package prompt

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const maxPromptInput = 300

var suppressedKeys = key.NewBinding(key.WithKeys(
	"alt+j", "alt+k", "alt+b", "alt+n", "alt+p", "alt+t",
))

func trimRight(input string) string {
	return strings.TrimRight(input, " \t\n")
}

func Blink() tea.Cmd {
	return textinput.Blink
}

type Model struct {
	history        []string
	historyMatches map[string]struct{}
	historyPos     int
	input          textinput.Model
}

func (m *Model) SetWidth(width int) {
	m.input.Width = width
}

func (m Model) GetPromptWidth() int {
	return len(m.input.Prompt)
}

func (m *Model) GetInputAndResetIt() string {
	input := m.input.Value()
	input = trimRight(input)

	if _, ok := m.historyMatches[input]; !ok {
		m.history = append(m.history, input)
		m.historyPos = len(m.history)
		m.historyMatches[input] = struct{}{}
	}

	m.input.Reset()

	return input
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, suppressedKeys) {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyDown:
			if m.historyPos == len(m.history) {
				break
			}
			m.historyPos = min(len(m.history), m.historyPos+1)
			if m.historyPos < len(m.history) {
				m.input.SetValue(m.history[m.historyPos])
			} else {
				m.input.SetValue("")
			}
		case tea.KeyUp:
			if m.historyPos == len(m.history) && m.input.Value() != "" {
				break
			}
			m.historyPos = max(-1, m.historyPos-1)
			if m.historyPos >= 0 {
				m.input.SetValue(m.history[m.historyPos])
			}
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

	m.historyMatches = map[string]struct{}{}
	m.historyPos = -1
	m.input = textinput.New()
	m.input.Focus()
	m.input.CharLimit = maxPromptInput
	m.input.Prompt = "$ "
	m.input.KeyMap.LineEnd.SetEnabled(false)

	return m
}
