package chatslist

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/franciscosbf/irc-client/internal/ui/components/chat"
)

type channel struct {
	tag string
}

func (c channel) Title() string {
	return c.tag
}

func (c channel) Description() string {
	return ""
}

func (c channel) FilterValue() string {
	return ""
}

type Model struct {
	list list.Model
}

func (m *Model) SetChats(chat []chat.Model) {
	items := make([]list.Item, len(chat))
	for i, model := range chat {
		items[i] = channel{
			tag: model.GetTag(),
		}
	}
	m.list.SetItems(items)
}

func (m *Model) SetSelectedChat(index int) {
	m.list.Select(index)
}

func (m *Model) GetSelectedChat() string {
	return m.list.SelectedItem().(channel).tag
}

func (m *Model) GetWidth() int {
	return m.list.Width()
}

func (m *Model) SetSize(width int, height int) {
	m.list.SetSize(width, height)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return m.list.View()
}

func InitialModel() Model {
	m := Model{}

	listDelegate := list.NewDefaultDelegate()
	listDelegate.ShowDescription = false
	listDelegate.SetSpacing(0)
	listDelegate.Styles.NormalTitle = lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(2).
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})
	listDelegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(1).
		Border(lipgloss.Border{Left: "|"}, false, false, false, true).
		Foreground(lipgloss.Color("#bde1e4"))

	m.list = list.New([]list.Item{}, listDelegate, 0, 0)
	m.list.SetShowTitle(false)
	m.list.SetShowFilter(false)
	m.list.SetShowStatusBar(false)
	m.list.SetShowHelp(false)
	m.list.Styles.PaginationStyle = lipgloss.Style{}

	return m
}
