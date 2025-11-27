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
	quitMsg          = "C'est la vie"
	slidingInterval  = 250 * time.Millisecond
	networkChatIndex = 0
	timeFormat       = "15:04"
)

var notConnectedSlidingText = "Not connected"

var paddedBorderStyle = lipgloss.NewStyle().
	Padding(1, 1, 1, 1)

var roundedBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder())

var slidingStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#2fabb5", Dark: "#bde1e4"}).
	PaddingLeft(1)

var appMsgStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var nickNameStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var timeStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.AdaptiveColor{Light: "#3c3c3c", Dark: "#a8a8a8"})

func currentTime() string {
	return time.Now().Format(timeFormat)
}

type networkMsg struct {
	network *irc.Network
	msg     irc.NetworkMessage
	isOpen  bool
}

type channelMsg struct {
	network *irc.Network
	channel *irc.NetworkChannel
	msg     irc.ChannelMessage
	isOpen  bool
}

type connectionMsg struct {
	cmd  cmds.ConnectCmd
	conn *irc.NetworkConnection
	err  error
}

func networkMsgCmd(network *irc.Network) tea.Cmd {
	return func() tea.Msg {
		msg, ok := network.ReceiveMessage()
		return networkMsg{
			network: network,
			msg:     msg,
			isOpen:  ok,
		}
	}
}

func channelMsgCmd(network *irc.Network, channel *irc.NetworkChannel) tea.Cmd {
	return func() tea.Msg {
		msg, ok := channel.ReceiveMessage()
		return channelMsg{
			network: network,
			channel: channel,
			msg:     msg,
			isOpen:  ok,
		}
	}
}

func connectionMsgCmd(cmd cmds.ConnectCmd) tea.Cmd {
	return func() tea.Msg {
		conn, err := irc.DialNetworkConnection(cmd.Host, cmd.Port)
		return connectionMsg{
			cmd:  cmd,
			conn: conn,
			err:  err,
		}
	}
}

type connDialup struct {
	inProcess bool
	host      string
}

func (cd *connDialup) register(host string) {
	*cd = connDialup{
		inProcess: true,
		host:      host,
	}
}

func (cd *connDialup) unregister() {
	*cd = connDialup{}
}

type modeledChannel struct {
	index   int
	channel *irc.NetworkChannel
}

type model struct {
	connDialup      connDialup
	network         *irc.Network
	modeledChannels map[string]modeledChannel
	chatsList       chatslist.Model
	chats           []chat.Model
	prevActiveChat  int
	activeChatIndex int
	prompt          prompt.Model
	sliding         textsliding.Model
}

func (m *model) setActiveChat(index int) {
	m.activeChatIndex = index
	m.prevActiveChat = m.activeChatIndex
}

func (m *model) toggleActiveChatWithNetworkChat() {
	if m.activeChatIndex == networkChatIndex {
		m.activeChatIndex = m.prevActiveChat
	} else {
		m.prevActiveChat = m.activeChatIndex
		m.activeChatIndex = networkChatIndex
	}
	m.chatsList.SetSelectedChat(m.activeChatIndex)
}

func (m *model) disconnectFromNetwork() bool {
	host, disconnected := m.quitCurrentNetwork()

	if disconnected {
		m.addAppMsg("Disconnected from the network " + host)
	}

	return disconnected
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
	m.chats[m.activeChatIndex].SetSize(mod(rightSlice-2), mod(height-3))
	m.prompt.SetWidth(rightSlice)

	if m.chats[m.activeChatIndex].PastBottom() {
		m.chats[m.activeChatIndex].GoToBottom()
	}
}

func (m *model) goToPreviousChat() {
	m.setActiveChat(max(0, m.activeChatIndex-1))
	m.chatsList.SetSelectedChat(m.activeChatIndex)
}

func (m *model) goToNextChat() {
	m.setActiveChat(min(len(m.chats)-1, m.activeChatIndex+1))
	m.chatsList.SetSelectedChat(m.activeChatIndex)
}

func (m *model) onHelpCmd(cmd cmds.HelpCmd) {
	m.addAppMsg(cmd.HelpMsg())
}

func (m *model) onQuitCmd() {
	m.quitCurrentNetwork()
}

func (m *model) onConnectCmd(cmd cmds.ConnectCmd) tea.Cmd {
	if m.network != nil {
		m.addAppMsg("You must first disconnect from the current network")
		return nil
	}

	m.connDialup.register(cmd.Host)

	return connectionMsgCmd(cmd)
}

func (m *model) onDisconnectCmd() {
	m.disconnectFromNetwork()

	m.sliding.SetText(notConnectedSlidingText)
}

func (m *model) onNickCmd(cmd cmds.NickCmd) {
	if err := m.network.ChangeNickname(cmd.Nickname); err != nil {
		m.addAppMsg("Failed to request chaning nickname to " + cmd.Nickname)
	}
}

func (m *model) onJoinCmd(cmd cmds.JoinCmd) tea.Cmd {
	if _, ok := m.modeledChannels[cmd.Tag]; ok {
		m.addAppMsg("Already in channel " + cmd.Tag)
	} else if channel, err := m.network.JoinChannel(cmd.Tag); err == nil {
		prevActiveChat := m.chats[m.activeChatIndex]

		m.setActiveChat(len(m.chats))

		m.chats = append(m.chats, chat.InitialModel(cmd.Tag))

		m.modeledChannels[cmd.Tag] = modeledChannel{
			index:   m.activeChatIndex,
			channel: channel,
		}

		m.chats[m.activeChatIndex].SetSize(prevActiveChat.GetWidth(), prevActiveChat.GetHeight())
		m.chatsList.SetChats(m.chats)
		m.chatsList.SetSelectedChat(m.activeChatIndex)

		return channelMsgCmd(m.network, channel)

	} else {
		m.addAppMsg("Failed to join channel " + cmd.Tag)
	}

	return nil
}

func (m *model) onPartCmd(cmd cmds.PartCmd) {
	if chatChannel, ok := m.modeledChannels[cmd.Tag]; ok {
		delete(m.modeledChannels, chatChannel.channel.GetTag())

		m.chats = append(m.chats[:chatChannel.index], m.chats[chatChannel.index+1:]...)
		m.chatsList.SetChats(m.chats)

		if m.activeChatIndex >= chatChannel.index {
			for i := m.activeChatIndex; i < len(m.chats); i++ {
				modeledChannel := m.modeledChannels[m.chats[i].GetTag()]
				modeledChannel.index--
				m.modeledChannels[m.chats[i].GetTag()] = modeledChannel
			}
			m.setActiveChat(m.activeChatIndex - 1)
		}
		m.chatsList.SetSelectedChat(m.activeChatIndex)

		if err := chatChannel.channel.Part(); err == nil {
		} else {
			m.addAppMsg("Failed to part channel " + cmd.Tag)
		}

		return
	}

	m.addAppMsg("Not in channel " + cmd.Tag)
}

func (m *model) onMsgCmd(cmd cmds.MsgCmd) {
	if m.activeChatIndex == networkChatIndex {
		return
	}

	modeledChannel := m.modeledChannels[m.chats[m.activeChatIndex].GetTag()]
	if err := modeledChannel.channel.SendMessage(cmd.MsgContent); err == nil {
		m.addChannelMsg(m.activeChatIndex, irc.ChannelMessage{
			Sender:  m.network.GetNickname(),
			Content: cmd.MsgContent,
		})
		return
	}

	m.addAppMsg("Failed to send message to channel " + modeledChannel.channel.GetTag())
}

func (m *model) interpretUserInput() (teaCmd tea.Cmd, exit bool) {
	input := m.prompt.GetInputAndResetIt()
	if input == "" {
		return
	}

	cmd, err := cmds.Parse(input)
	if err != nil {
		m.addAppMsg(err.Error())
		return
	}

	switch cmd := cmd.(type) {
	case cmds.HelpCmd:
		m.onHelpCmd(cmd)
	case cmds.QuitCmd:
		m.onQuitCmd()
		exit = true
	default:
		if m.connDialup.inProcess {
			m.addAppMsg("Still waiting to connect to " + m.connDialup.host)
		} else if cmd.GetType() == cmds.Connect {
			teaCmd = m.onConnectCmd(cmd.(cmds.ConnectCmd))
		} else if m.network == nil {
			if cmd.GetType() != cmds.Msg {
				m.addAppMsg("No current network")
			}
		} else {
			switch cmd := cmd.(type) {
			case cmds.DisconnectCmd:
				m.onDisconnectCmd()
			case cmds.NickCmd:
				m.onNickCmd(cmd)
			default:
				if !m.network.IsRegistered() {
					if cmd.GetType() != cmds.Msg {
						m.addAppMsg("Wait until user registration is complete")
					}
					break
				}
				switch cmd := cmd.(type) {
				case cmds.JoinCmd:
					teaCmd = m.onJoinCmd(cmd)
				case cmds.PartCmd:
					m.onPartCmd(cmd)
				case cmds.MsgCmd:
					m.onMsgCmd(cmd)
				}
			}
		}
	}

	if cmd.GetType() != cmds.Msg {
		m.prompt.AddLastInputToHistory()
	}

	return
}

func (m *model) interpretNetworkMsg(msg networkMsg) tea.Cmd {
	if m.network != msg.network {
		return nil
	}

	if !msg.isOpen {
		m.addAppMsg("Disconnected from the network " + msg.network.GetHost())

		m.resetChats()

		m.sliding.SetText(notConnectedSlidingText)

		return nil
	}

	m.addNetworkMsg(msg.msg)

	return networkMsgCmd(m.network)
}

func (m *model) interpretChannelMsg(msg channelMsg) tea.Cmd {
	if m.network != msg.network {
		return nil
	}

	chatChannel, ok := m.modeledChannels[msg.channel.GetTag()]

	if !ok || !msg.isOpen {
		return nil
	}

	m.addChannelMsg(chatChannel.index, msg.msg)

	return channelMsgCmd(m.network, chatChannel.channel)
}

func (m *model) resetChats() {
	m.setActiveChat(networkChatIndex)

	m.chats = m.chats[:1]

	m.chatsList.SetChats(m.chats)
	m.chatsList.SetSelectedChat(m.activeChatIndex)
}

func (m *model) addMsg(chatIndex int, msg string) {
	atBottom := m.chats[chatIndex].AtBottom()

	time := timeStyle.Render(currentTime())

	m.chats[chatIndex].AddMsg(time + " " + msg)

	if atBottom || chatIndex != m.activeChatIndex {
		m.chats[chatIndex].GoToBottom()
	}
}

func (m *model) addAppMsg(msg string) {
	m.addMsg(networkChatIndex, appMsgStyle.Render(msg))
}

func (m *model) addNetworkMsg(msg irc.NetworkMessage) {
	m.addMsg(networkChatIndex, msg.Content)
}

func (m *model) addChannelMsg(chatIndex int, msg irc.ChannelMessage) {
	var msgContent string
	if msg.Sender != "" {
		msgContent = nickNameStyle.Render(msg.Sender) + " " + msg.Content
	} else {
		msgContent = msg.Content
	}
	m.addMsg(chatIndex, msgContent)
}

func (m *model) quitCurrentNetwork() (string, bool) {
	if m.network == nil {
		return "", true
	}

	if err := m.network.Quit(quitMsg); err != nil {
		log.Printf("Error when quitting network: %v\n", err)
	}

	m.resetChats()
	host := m.network.GetHost()
	m.network = nil

	return host, true
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
		case "alt+h":
			m.chats[m.activeChatIndex].ScrollOneColumnLeft()
		case "alt+j":
			m.chats[m.activeChatIndex].ScrollOneLineDown()
		case "alt+k":
			m.chats[m.activeChatIndex].ScrollOneLineUp()
		case "alt+l":
			m.chats[m.activeChatIndex].ScrollOneColumnRight()
		case "alt+b":
			m.chats[m.activeChatIndex].GoToBottom()
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
				m.chats[m.activeChatIndex].WheelScrollUp()
			}
		case tea.MouseButtonWheelDown:
			if msg.Alt {
				m.goToNextChat()
			} else {
				m.chats[m.activeChatIndex].WheelScrollDown()
			}
		}
	case connectionMsg:
		m.connDialup.unregister()
		if msg.err != nil {
			m.addAppMsg("Failed to dial connection")
			break
		}
		network := irc.NewNetwork(msg.conn)
		network.StartListener()
		if err := network.Register(msg.cmd.Nickname, msg.cmd.Name); err != nil {
			m.addAppMsg("Failed to send connection registration")
			break
		}
		m.network = network
		m.sliding.SetText("Connected to network " + msg.cmd.Host)
		appendAdditionalCmd(networkMsgCmd(network))
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
	m.chats[m.activeChatIndex], activeChatCmd = m.chats[m.activeChatIndex].Update(msg)
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
	activeChat := roundedBorderStyle.Render(m.chats[m.activeChatIndex].View())
	prompt := m.prompt.View()

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, chats, sliding),
		lipgloss.JoinVertical(lipgloss.Left, activeChat, prompt))
}

func initialModel() model {
	m := model{}

	m.modeledChannels = map[string]modeledChannel{}
	m.activeChatIndex = networkChatIndex
	m.chats = []chat.Model{chat.InitialModel("network")}
	m.chatsList = chatslist.InitialModel()
	m.chatsList.SetChats(m.chats)
	m.prompt = prompt.InitialModel()
	m.sliding = textsliding.InitialModel(notConnectedSlidingText, slidingInterval)

	m.resetChats()

	return m
}
