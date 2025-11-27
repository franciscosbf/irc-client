package cmds

type Type int

func (t Type) toString() string {
	switch t {
	case Help:
		return "help"
	case Connect:
		return "connect"
	case Disconnect:
		return "disconnect"
	case Join:
		return "join"
	case Part:
		return "part"
	case Nick:
		return "nick"
	case Quit:
		return "quit"
	case Msg:
		fallthrough
	default:
		return ""
	}
}

const (
	Help Type = iota
	Connect
	Disconnect
	Join
	Part
	Nick
	Quit
	Msg
)

type Cmd interface {
	GetType() Type
}

type ConnectCmd struct {
	Host           string
	Nickname, Name string
}

func (ConnectCmd) GetType() Type {
	return Connect
}

type DisconnectCmd struct{}

func (DisconnectCmd) GetType() Type {
	return Disconnect
}

type JoinCmd struct {
	Tag string
}

func (JoinCmd) GetType() Type {
	return Join
}

type PartCmd struct {
	Tag string
}

func (PartCmd) GetType() Type {
	return Part
}

type NickCmd struct {
	Nickname string
}

func (NickCmd) GetType() Type {
	return Nick
}

type QuitCmd struct{}

func (QuitCmd) GetType() Type {
	return Quit
}

type MsgCmd struct {
	MsgContent string
}

func (MsgCmd) GetType() Type {
	return Msg
}

type HelpCmd struct{}

func (HelpCmd) GetType() Type {
	return Help
}

func (HelpCmd) HelpMsg() string {
	return `Available commands:
/help                              Shows this message
/connect <host> <nickname> <name>  Connects to a network
/disconnect                        Disconnects from a network
/join <channel>                    Connects to a channel in the network
/part <channel>                    Disconnects from a channel in the network
/nick <nickname>                   Changes your nickname in the network
/quit                              Closes the IRC Client
<bunch of text>                    Sends a message in the current channel`
}
