package ui

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/franciscosbf/irc-client/internal/cmds"
	"github.com/franciscosbf/irc-client/internal/irc"
	"github.com/franciscosbf/irc-client/internal/ui/components/chat"
	"github.com/franciscosbf/irc-client/internal/ui/components/chatslist"
	"github.com/franciscosbf/irc-client/internal/ui/components/prompt"
	"github.com/franciscosbf/irc-client/internal/ui/components/textsliding"
)

const (
	quitMsg         = "C'est la vie"
	slidingInterval = 250 * time.Millisecond
	networkChat     = 0
)

var notConnectedSlidingText = "Not connected"

var paddedBorderStyle = lipgloss.NewStyle().
	Padding(1, 1, 1, 1)

var roundedBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder())

var slidingStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#bde1e4")).
	PaddingLeft(1)

var appMsgStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var nickNameStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

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
	chatsList       chatslist.Model
	chats           []chat.Model
	prevActiveChat  int
	activeChat      int
	prompt          prompt.Model
	sliding         textsliding.Model
}

func (m *model) setActiveChat(index int) {
	m.activeChat = index
	m.prevActiveChat = m.activeChat
}

func (m *model) toggleActiveChatWithNetworkChat() {
	if m.activeChat == networkChat {
		m.activeChat = m.prevActiveChat
	} else {
		m.prevActiveChat = m.activeChat
		m.activeChat = networkChat
	}
	m.chatsList.SetSelectedChat(m.activeChat)
}

func (m *model) addaptToWindowSize(width, height int) {
	leftSlice := int(float64(width) * 0.12)
	rightSlice := width - leftSlice
	mod := func(x int) int {
		if x < 0 {
			return -x
		} else {
			return x
		}
	}
	m.chatsList.SetSize(mod(leftSlice-2), mod(height-3))
	m.sliding.SetWidth(mod(leftSlice - 3))
	m.chats[m.activeChat].SetSize(mod(rightSlice-2), mod(height-3))
	m.prompt.SetWidth(rightSlice)
	if m.chats[m.activeChat].PastBottom() {
		m.chats[m.activeChat].GoToBottom()
	}
}

func (m *model) goToPreviousChat() {
	m.setActiveChat(max(0, m.activeChat-1))
	m.chatsList.SetSelectedChat(m.activeChat)
}

func (m *model) goToNextChat() {
	m.setActiveChat(min(len(m.chats)-1, m.activeChat+1))
	m.chatsList.SetSelectedChat(m.activeChat)
}

func (m *model) interpretUserInput() (teaCmd tea.Cmd, exit bool) {
	input := m.prompt.GetInput()
	m.prompt.ResetContent()
	cmd, err := cmds.Parse(input)
	if err != nil {
		m.addAppMsg(err.Error())
		return
	}
	if cmd, ok := cmd.(cmds.HelpCmd); ok {
		m.addAppMsg(cmd.HelpMsg())
		return
	}
	switch cmd := cmd.(type) {
	case cmds.ConnectCmd:
		m.quitCurrentNetwork()
		conn, err := irc.DialNetworkConnection(cmd.Host, cmd.Port)
		if err != nil {
			m.addAppMsg("Failed to dial connection")
			break
		}
		network := irc.NewNetwork(conn)
		network.StartListener()
		if err := network.Register(cmd.Nickname, cmd.Name); err != nil {
			m.addAppMsg("Failed to send connection registration")
			break
		}
		m.network = network
		m.sliding.SetText("Connected to " + cmd.Host)
		teaCmd = networkMsgCmd(network)
	case cmds.DisconnectCmd:
		if !m.quitCurrentNetwork() {
			m.addAppMsg("No current network")
		} else {
			m.sliding.SetText(notConnectedSlidingText)
		}
	case cmds.JoinCmd:
		if m.network != nil {
			if !m.network.IsRegistered() {
				m.addAppMsg("Server is still registering user")
			} else if _, ok := m.modeledChannels[cmd.Tag]; ok {
				m.addAppMsg("Already in channel " + cmd.Tag)
			} else if channel, err := m.network.JoinChannel(cmd.Tag); err == nil {
				prevActiveChat := m.chats[m.activeChat]
				m.setActiveChat(len(m.chats))
				m.chats = append(m.chats, chat.InitialModel(cmd.Tag))
				m.modeledChannels[cmd.Tag] = modeledChannel{
					index:   m.activeChat,
					channel: channel,
				}
				m.chats[m.activeChat].SetSize(prevActiveChat.GetWidth(), prevActiveChat.GetHeight())
				m.chatsList.SetChats(m.chats)
				m.chatsList.SetSelectedChat(m.activeChat)
				teaCmd = channelMsgCmd(channel)
			} else {
				m.addAppMsg("Failed to join channel " + cmd.Tag)
			}
		} else {
			m.addAppMsg("No current network")
		}
	case cmds.PartCmd:
		if m.network != nil {
			if !m.network.IsRegistered() {
				m.addAppMsg("Server is still registering user")
			} else if chatChannel, ok := m.modeledChannels[cmd.Tag]; ok {
				delete(m.modeledChannels, chatChannel.channel.GetTag())
				m.chats = append(m.chats[:chatChannel.index], m.chats[chatChannel.index+1:]...)
				m.chatsList.SetChats(m.chats)
				if m.activeChat >= chatChannel.index {
					for i := m.activeChat; i < len(m.chats); i++ {
						modeledChannel := m.modeledChannels[m.chats[i].GetTag()]
						modeledChannel.index--
						m.modeledChannels[m.chats[i].GetTag()] = modeledChannel
					}
					m.setActiveChat(m.activeChat - 1)
				}
				m.chatsList.SetSelectedChat(m.activeChat)
				if err := chatChannel.channel.Part(); err == nil {
				} else {
					m.addAppMsg("Failed to part channel " + cmd.Tag)
				}
			} else {
				m.addAppMsg("Not in channel " + cmd.Tag)
			}
		} else {
			m.addAppMsg("No current network")
		}
	case cmds.NickCmd:
		if m.network != nil {
			if err := m.network.ChangeNickname(cmd.Nickname); err != nil {
				m.addAppMsg("Failed to issue nickname change to " + cmd.Nickname)
			}
		} else {
			m.addAppMsg("No current network")
		}
	case cmds.QuitCmd:
		m.quitCurrentNetwork()
		exit = true
	case cmds.MsgCmd:
		if m.activeChat == networkChat {
			break
		} else if m.network != nil {
			modeledChannel := m.modeledChannels[m.chats[m.activeChat].GetTag()]
			if err := modeledChannel.channel.SendMessage(cmd.MsgContent); err == nil {
				m.addChannelMsg(m.activeChat, irc.ChannelMessage{
					Sender:  m.network.GetNickname(),
					Content: cmd.MsgContent,
				})
			} else {
				m.addAppMsg("Failed to send message")
			}
		} else {
			m.addAppMsg("No current network")
		}
	}

	return
}

func (m *model) interpretNetworkMsg(msg networkMsg) tea.Cmd {
	if !msg.isOpen {
		m.addAppMsg("Disconnected from the network")
		m.resetChats()
		m.sliding.SetText(notConnectedSlidingText)
		return nil
	}
	m.addNetworkMsg(msg.msg)
	return networkMsgCmd(m.network)
}

func (m *model) interpretChannelMsg(msg channelMsg) tea.Cmd {
	chatChannel, ok := m.modeledChannels[msg.channel.GetTag()]
	if !ok || !msg.isOpen {
		return nil
	}
	m.addChannelMsg(chatChannel.index, msg.msg)
	return channelMsgCmd(chatChannel.channel)
}

func (m *model) resetChats() {
	m.setActiveChat(networkChat)
	m.chats = m.chats[:1]
	m.chatsList.SetChats(m.chats)
	m.chatsList.SetSelectedChat(m.activeChat)
}

func (m *model) addAppMsg(msg string) {
	m.chats[networkChat].AddMsg(appMsgStyle.Render(msg))
	m.chats[networkChat].GoToBottom()
}

func (m *model) goToBottomIfNotActiveChat(index int) {
	if index != m.activeChat {
		return
	}

	m.chats[index].GoToBottom()
}

func (m *model) addNetworkMsg(msg irc.NetworkMessage) {
	m.chats[networkChat].AddMsg(msg.Content)

	m.goToBottomIfNotActiveChat(networkChat)
}

func (m *model) addChannelMsg(index int, msg irc.ChannelMessage) {
	var msgContent string
	if msg.Sender != "" {
		msgContent = nickNameStyle.Render(msg.Sender) + " " + msg.Content
	} else {
		msgContent = msg.Content
	}
	m.chats[index].AddMsg(msgContent)

	m.goToBottomIfNotActiveChat(index)
}

func (m *model) quitCurrentNetwork() bool {
	if m.network == nil {
		return false
	}

	if err := m.network.Quit(quitMsg); err != nil {
		log.Printf("Got error when quitting network: %v\n", err)
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
		chatsListCmd   tea.Cmd
		activeChatCmd  tea.Cmd
		promptCmd      tea.Cmd
		slidingCmd     tea.Cmd
		additionalCmds []tea.Cmd
	)
	appendAdditionalCmd := func(cmd tea.Cmd) {
		additionalCmds = append(additionalCmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.addaptToWindowSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "alt+j":
			m.chats[m.activeChat].ScrollOneLineDown()
		case "alt+k":
			m.chats[m.activeChat].ScrollOneLineUp()
		case "alt+b":
			m.chats[m.activeChat].GoToBottom()
		case "alt+p":
			m.goToPreviousChat()
		case "alt+n":
			m.goToNextChat()
		case "alt+t":
			m.toggleActiveChatWithNetworkChat()
		case "enter":
			if cmd, exit := m.interpretUserInput(); exit {
				return m, tea.Quit
			} else if cmd != nil {
				appendAdditionalCmd(cmd)
			}
		case "ctrl+c", "esc":
			m.quitCurrentNetwork()
			return m, tea.Quit
		}
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if msg.Alt {
				m.goToPreviousChat()
			} else {
				m.chats[m.activeChat].WheelScrollUp()
			}
		case tea.MouseButtonWheelDown:
			if msg.Alt {
				m.goToNextChat()
			} else {
				m.chats[m.activeChat].WheelScrollDown()
			}
		}
	case networkMsg:
		if cmd := m.interpretNetworkMsg(msg); cmd != nil {
			appendAdditionalCmd(cmd)
		}
	case channelMsg:
		if cmd := m.interpretChannelMsg(msg); cmd != nil {
			appendAdditionalCmd(cmd)
		}
	}

	m.prompt, promptCmd = m.prompt.Update(msg)
	m.chatsList, chatsListCmd = m.chatsList.Update(msg)
	m.chats[m.activeChat], activeChatCmd = m.chats[m.activeChat].Update(msg)
	m.sliding, slidingCmd = m.sliding.Update(msg)

	cmds := tea.Batch(
		append([]tea.Cmd{
			chatsListCmd,
			activeChatCmd,
			promptCmd,
			slidingCmd,
		}, additionalCmds...)...,
	)
	return m, cmds
}

func (m model) View() string {
	chats := paddedBorderStyle.Render(lipgloss.PlaceHorizontal(
		m.chatsList.GetWidth(), lipgloss.Left, m.chatsList.View()))
	sliding := slidingStyle.Render(m.sliding.View())
	activeChat := roundedBorderStyle.Render(m.chats[m.activeChat].View())
	prompt := m.prompt.View()

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, chats, sliding),
		lipgloss.JoinVertical(lipgloss.Left, activeChat, prompt))
}

func initialModel() model {
	m := model{}

	m.modeledChannels = map[string]modeledChannel{}
	m.activeChat = networkChat
	m.chats = []chat.Model{chat.InitialModel("network")}
	m.chatsList = chatslist.InitialModel()
	m.chatsList.SetChats(m.chats)
	m.prompt = prompt.InitialModel()
	m.sliding = textsliding.InitialModel(notConnectedSlidingText, slidingInterval)

	m.resetChats()

	return m
}
