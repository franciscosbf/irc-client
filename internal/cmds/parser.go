package cmds

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type InvalidCmdErr struct {
	CmdType Type
	Reason  string
}

var specialChs = []rune{'[', ']', '\\', '`', '_', '^', '{', '|', '}'}

func (e InvalidCmdErr) Error() string {
	return fmt.Sprintf("Invalid command %s: %s", e.CmdType, e.Reason)
}

func trimRight(input string) string {
	return strings.TrimRight(input, " \t")
}

func cut(input string) (string, string) {
	before, after, _ := strings.Cut(input, " ")
	return before, after
}

func splitNArgs(args string, nArgs int) []string {
	return strings.SplitN(args, " ", nArgs)
}

func isNicknameValid(nickname string) bool {
	if len(nickname) > 9 {
		return false
	}

	for _, r := range nickname {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			slices.Contains(specialChs, r) ||
			(r >= 0x5B || r <= 0x60) ||
			(r >= 0x7B || r <= 0x7D) ||
			r == '_') {
			return false
		}
	}

	return true
}

func isNameValid(name string) bool {
	for _, r := range name {
		if r > unicode.MaxASCII {
			return false
		}
	}

	return true
}

func isChannelTagValid(channel string) bool {
	if !strings.HasPrefix(channel, "#") {
		return false
	}

	channel = channel[1:]

	if len(channel) > 50 {
		return false
	}

	for _, r := range channel {
		if r > unicode.MaxASCII || r == ' ' || r == ',' || r == 7 {
			return false
		}
	}

	return true
}

func Parse(input string) (Cmd, error) {
	input = trimRight(input)

	if !strings.HasPrefix(input, "/") {
		return MsgCmd{
			MsgContent: input,
		}, nil
	}

	cmdType, args := cut(input[1:])
	switch Type(cmdType) {
	case Help:
		if len(args) != 0 {
			return nil, InvalidCmdErr{
				CmdType: Help,
				Reason:  "command doesn't have arguments",
			}
		}
		return HelpCmd{}, nil
	case Connect:
		args := splitNArgs(args, 3)
		if len(args) < 3 {
			return nil, InvalidCmdErr{
				CmdType: Connect,
				Reason:  "expecting arguments <address> <nickname> <name>",
			}
		}
		host, rawPort, err := net.SplitHostPort(args[0])
		if err != nil {
			return nil, InvalidCmdErr{
				CmdType: Connect,
				Reason:  "invalid address argument",
			}
		}
		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil {
			return nil, InvalidCmdErr{
				CmdType: Connect,
				Reason:  "invalid address port",
			}
		}
		if !isNicknameValid(args[1]) {
			return nil, InvalidCmdErr{
				CmdType: Connect,
				Reason:  "invalid nickname",
			}
		}
		nickname := args[1]
		if !isNameValid(args[1]) {
			return nil, InvalidCmdErr{
				CmdType: Connect,
				Reason:  "invalid name",
			}
		}
		name := args[2]
		return ConnectCmd{
			Host:     host,
			Port:     uint16(port),
			Nickname: nickname,
			Name:     name,
		}, nil
	case Disconnect:
		if len(args) != 0 {
			return nil, InvalidCmdErr{
				CmdType: Disconnect,
				Reason:  "command doesn't have arguments",
			}
		}
		return DisconnectCmd{}, nil
	case Join, Part:
		var cmd Cmd
		if Type(cmdType) == Join {
			cmd = JoinCmd{
				Tag: args,
			}
		} else {
			cmd = PartCmd{
				Tag: args,
			}
		}
		if args == "" {
			return nil, InvalidCmdErr{
				CmdType: cmd.GetType(),
				Reason:  "expecting argument <channel>",
			}
		}
		if !isChannelTagValid(args) {
			return nil, InvalidCmdErr{
				CmdType: cmd.GetType(),
				Reason:  "invalid channel",
			}
		}
		return cmd, nil
	case Nick:
		if args == "" {
			return nil, InvalidCmdErr{
				CmdType: Nick,
				Reason:  "expecting argument <nickname>",
			}
		}
		if !isNicknameValid(args) {
			return nil, InvalidCmdErr{
				CmdType: Nick,
				Reason:  "invalid nickname",
			}
		}
		nickname := args
		return NickCmd{
			Nickname: nickname,
		}, nil
	case Quit:
		if len(args) != 0 {
			return nil, InvalidCmdErr{
				CmdType: Quit,
				Reason:  "command doesn't have arguments",
			}
		}
		return QuitCmd{}, nil
	}

	return nil, fmt.Errorf(
		"Unknown command: %v. "+
			"Type /help and check the available commands",
		input,
	)
}
