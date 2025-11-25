package textsliding

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type Model struct {
	text       string
	pos        int
	width      int
	windowSize int
	interval   time.Duration
}

func (m *Model) addjustWindowSize() {
	m.windowSize = max(m.width, len(m.text)+1)
	m.pos = m.windowSize - 1
}

func (m Model) Sliding() tea.Cmd {
	return tickCmd(m.interval)
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.addjustWindowSize()
}

func (m *Model) SetText(text string) {
	m.text = text
	m.addjustWindowSize()
}

func (m Model) Init() tea.Cmd {
	return m.Sliding()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.pos--
		if m.pos < 0 {
			m.pos = m.windowSize - 1
		}

		return m, m.Sliding()
	}

	return m, nil
}

func (m Model) View() string {
	if m.windowSize == 0 {
		return ""
	}

	window := make([]rune, m.windowSize)
	for i := range m.windowSize {
		window[i] = ' '
	}
	text := []rune(m.text)

	start := m.pos
	end := m.pos + len(m.text)
	remainder := min(m.windowSize, end)

	copy(window[start:remainder], text[:remainder-start])
	copy(window[:end-remainder], text[remainder-start:])

	return string(window[:m.width])
}

func InitialModel(text string, interval time.Duration) Model {
	return Model{
		text:       text,
		interval:   interval,
		windowSize: len(text),
	}
}
