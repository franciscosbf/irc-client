package ui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/franciscosbf/irc-client/internal/cmds"
	"github.com/franciscosbf/irc-client/internal/irc"
	"github.com/franciscosbf/irc-client/internal/ui/components/chat"
	"github.com/franciscosbf/irc-client/internal/ui/components/chats"
	"github.com/franciscosbf/irc-client/internal/ui/components/prompt"
	"github.com/franciscosbf/irc-client/internal/ui/components/textsliding"
)

const (
	progName        = "IRC Client"
	quitMsg         = "C'est la vie"
	slidingInterval = 250 * time.Millisecond
	networkChat     = 0
)

var paddedBorderStyle = lipgloss.NewStyle().
	Padding(1, 1, 1, 1)

var roundedBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder())

var slidingStyle = lipgloss.NewStyle().
	Bold(true).
	PaddingLeft(1).
	Foreground(lipgloss.Color("#bde1e4"))

type networkMsg struct {
	msg    irc.NetworkMessage
	isOpen bool
}

type channelMsg struct {
	channel *irc.NetworkChannel
	msg     irc.ChannelMessage
	isOpen  bool
}

func networkMsgCmd(network *irc.Network) tea.Cmd {
	return func() tea.Msg {
		msg, ok := network.ReceiveMessage()
		return networkMsg{
			msg:    msg,
			isOpen: ok,
		}
	}
}

func channelMsgCmd(channel *irc.NetworkChannel) tea.Cmd {
	return func() tea.Msg {
		msg, ok := channel.ReceiveMessage()
		return channelMsg{
			channel: channel,
			msg:     msg,
			isOpen:  ok,
		}
	}
}

type modeledChannel struct {
	index   int
	channel *irc.NetworkChannel
}

type model struct {
	network         *irc.Network
	modeledChannels map[string]modeledChannel

	chats      chats.Model
	chat       []chat.Model
	activeChat int
	prompt     prompt.Model
	sliding    textsliding.Model
}

func (m *model) resetChats() {
	m.activeChat = networkChat
	m.chat = m.chat[:1]
	m.chats.SetChats(m.chat)
	m.chats.SetSelectedChat(0)
}

func (m *model) quitCurrentNetwork() bool {
	if m.network == nil {
		return false
	}

	if err := m.network.Quit(quitMsg); err != nil {
		log.Printf("Got error when quitting network: %v\n", err)
		return false
	}

	m.resetChats()
	m.network = nil

	return true
}

func (m model) Init() tea.Cmd {
	return tea.Batch(prompt.Blink(), m.sliding.Sliding())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		chatsCmd       tea.Cmd
		activeChatCmd  tea.Cmd
		promptCmd      tea.Cmd
		slidingCmd     tea.Cmd
		additionalCmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		leftSlice := int(float64(msg.Width) * 0.1)
		rightSlice := msg.Width - leftSlice
		mod := func(x int) int {
			if x < 0 {
				return -x
			} else {
				return x
			}
		}
		m.chats.SetSize(mod(leftSlice-2), mod(msg.Height-3))
		m.sliding.SetWidth(mod(leftSlice - 3))
		m.chat[m.activeChat].SetSize(mod(rightSlice-2), mod(msg.Height-3))
		m.prompt.SetWidth(rightSlice)
		m.chat[m.activeChat].GoToBottom()
	case tea.KeyMsg:
		switch msg.String() {
		case "alt+j":
			m.chat[m.activeChat].ScrollOneLineDown()
		case "alt+k":
			m.chat[m.activeChat].ScrollOneLineUp()
		case "alt+b":
			m.chat[m.activeChat].GoToBottom()
		case "alt+n":
			m.activeChat = min(len(m.chat)-1, m.activeChat+1)
			m.chats.SetSelectedChat(m.activeChat)
		case "alt+p":
			m.activeChat = max(0, m.activeChat-1)
			m.chats.SetSelectedChat(m.activeChat)
		case "enter":
			input := m.prompt.GetInput()
			m.prompt.ResetContent()
			cmd, err := cmds.Parse(input)
			if err != nil {
				m.chat[m.activeChat].AddMsg(err.Error())
			} else {
				switch cmd := cmd.(type) {
				case cmds.HelpCmd:
					m.chat[m.activeChat].AddMsg(cmd.HelpMsg())
				case cmds.ConnectCmd:
					m.quitCurrentNetwork()
					conn, err := irc.DialNetworkConnection(cmd.Host, cmd.Port)
					if err != nil {
						m.chat[networkChat].AddMsg("Failed to dial connection")
						break
					}
					network := irc.NewNetwork(conn, cmd.Nickname, cmd.Name)
					network.StartListener()
					if err := network.Register(); err != nil {
						m.chat[networkChat].AddMsg("Failed to send connection registration")
						break
					}
					additionalCmds = append(additionalCmds, networkMsgCmd(network))
					m.network = network
				case cmds.DisconnectCmd:
					if !m.quitCurrentNetwork() {
						m.chat[networkChat].AddMsg("No current network")
					}
				case cmds.JoinCmd:
					// TODO: handle
				case cmds.PartCmd:
					// TODO: handle
				case cmds.NickCmd:
					if m.network != nil {
						if err := m.network.ChangeNickname(cmd.Nickname); err != nil {
							m.chat[networkChat].AddMsg("Failed to issue nickname change")
						}
					} else {
						m.chat[networkChat].AddMsg("No current network")
					}
				case cmds.MsgCmd:
					if m.activeChat == 0 {
						break
					} else if m.network == nil {
						m.chat[networkChat].AddMsg("No current network")
					} else {
						// TODO: handle
					}
				}
			}
			m.chat[m.activeChat].GoToBottom()
		case "ctrl+c", "esc":
			if m.network != nil {
				m.quitCurrentNetwork()
			}
			return m, tea.Quit
		}
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.chat[m.activeChat].WheelScrollUp()
		case tea.MouseButtonWheelDown:
			m.chat[m.activeChat].WheelScrollDown()
		}
	case networkMsg:
		if !msg.isOpen {
			m.chat[networkChat].AddMsg("Disconnected from the network")
			m.resetChats()
			break
		}
		m.chat[networkChat].AddMsg(msg.msg.Content)
		additionalCmds = append(additionalCmds, networkMsgCmd(m.network))
		if !m.chat[networkChat].AtBottom() {
			m.chat[networkChat].GoToBottom()
		}
	case channelMsg:
		if !msg.isOpen {
			break
		}
		chatChannel, ok := m.modeledChannels[msg.channel.GetTag()]
		if !ok {
			break
		}
		m.chat[chatChannel.index].AddMsg(msg.msg.Sender + " " + msg.msg.Content)
		additionalCmds = append(additionalCmds, channelMsgCmd(chatChannel.channel))
		if chatChannel.index == m.activeChat && !m.chat[chatChannel.index].AtBottom() {
			m.chat[m.activeChat].GoToBottom()
		}
	}

	m.prompt, promptCmd = m.prompt.Update(msg)
	m.chats, chatsCmd = m.chats.Update(msg)
	m.chat[m.activeChat], activeChatCmd = m.chat[m.activeChat].Update(msg)
	m.sliding, slidingCmd = m.sliding.Update(msg)

	cmds := tea.Batch(
		append([]tea.Cmd{
			chatsCmd,
			activeChatCmd,
			promptCmd,
			slidingCmd,
		}, additionalCmds...)...,
	)
	return m, cmds
}

func (m model) View() string {
	chats := paddedBorderStyle.Render(lipgloss.PlaceHorizontal(
		m.chats.GetWidth(), lipgloss.Left, m.chats.View()))
	sliding := slidingStyle.Render(m.sliding.View())
	activeChat := roundedBorderStyle.Render(m.chat[m.activeChat].View())
	prompt := m.prompt.View()

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, chats, sliding),
		lipgloss.JoinVertical(lipgloss.Left, activeChat, prompt))
}

func initialModel() model {
	m := model{}

	m.activeChat = networkChat
	m.chat = []chat.Model{chat.InitialModel("network")}
	m.chats = chats.InitialModel()
	m.chats.SetChats(m.chat)
	m.prompt = prompt.InitialModel()
	m.sliding = textsliding.InitialModel(progName, slidingInterval)

	m.resetChats()

	return m
}
