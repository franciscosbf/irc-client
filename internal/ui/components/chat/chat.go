package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	tag      string
	msgs     []string
	viewport viewport.Model
}

func (m Model) GetTag() string {
	return m.tag
}

func (m Model) GetWidth() int {
	return m.viewport.Width
}

func (m Model) GetHeight() int {
	return m.viewport.Height
}

func (m *Model) AddMsg(msg string) {
	m.msgs = append(m.msgs, msg)

	joinedMsgs := strings.Join(m.msgs, "\n")
	renderedMsgs := lipgloss.NewStyle().
		Width(m.viewport.Width).
		Render(joinedMsgs)
	m.viewport.SetContent(renderedMsgs)
}

func (m *Model) ScrollOneLineUp() {
	m.viewport.ScrollUp(1)
}

func (m *Model) ScrollOneLineDown() {
	m.viewport.ScrollDown(1)
}

func (m *Model) ScrollOneColumnLeft() {
	m.viewport.ScrollLeft(1)
}

func (m *Model) ScrollOneColumnRight() {
	m.viewport.ScrollRight(1)
}

func (m *Model) WheelScrollUp() {
	m.viewport.ScrollUp(m.viewport.MouseWheelDelta)
}

func (m *Model) WheelScrollDown() {
	m.viewport.ScrollDown(m.viewport.MouseWheelDelta)
}

func (m Model) AtBottom() bool {
	return m.viewport.AtBottom()
}

func (m Model) PastBottom() bool {
	return m.viewport.PastBottom()
}

func (m *Model) GoToBottom() {
	m.viewport.GotoBottom()
}

func (m *Model) SetSize(width int, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return m.viewport.View()
}

func InitialModel(tag string) Model {
	m := Model{}

	m.tag = tag
	m.viewport = viewport.New(0, 0)

	return m
}
