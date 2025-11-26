package irc

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	rpl_WELCOME       = 1
	rpl_YOURHOST      = 2
	rpl_CREATED       = 3
	rpl_MYINFO        = 4
	rpl_BOUNCE        = 5
	rpl_LUSERCLIENT   = 251
	rpl_LUSEROP       = 252
	rpl_LUSERUNKNOWN  = 253
	rpl_LUSERCHANNELS = 254
	rpl_LUSERME       = 255
	rpl_ADMINME       = 256
	rpl_LUSERS        = 265
	rpl_GUSERS        = 266
	rpl_AWAY          = 301
	rpl_TOPIC         = 332
	rpl_TOPICWHOTIME  = 333
	rpl_NAMREPLY      = 353
	rpl_ENDOFNAMES    = 366
	rpl_MOTDSTART     = 375
	rpl_MOTD          = 372
	rpl_ENDOFMOTD     = 376
	rpl_DHOST         = 396

	err_NOSUCHCHANNEL    = 403
	err_NOMOTD           = 422
	err_ERRONEUSNICKNAME = 432
	err_NICKNAMEINUSE    = 433
	err_NICKCOLLISION    = 436
	err_NOTONCHANNEL     = 442
	err_NOTREGISTERED    = 451
	err_ALREADYREGISTRED = 462
	err_INVITEONLYCHAN   = 473
	err_BANNEDFROMCHAN   = 474
	err_RESTRICTED       = 484
	err_CANNOTSENDTOCHAN = 404
)

type message interface {
	getSender() string
	getUnparsed() string
}

type origin interface {
	getSender() string
}

type withoutOrigin struct{}

func (withoutOrigin) getSender() string {
	return "unspecified sender"
}

type serverOrigin struct {
	servername string
}

func (o serverOrigin) getSender() string {
	return o.servername
}

type userOrigin struct {
	nickname, identifier string
}

func (o userOrigin) getSender() string {
	return o.nickname + "!" + o.identifier
}

type baseMessage struct {
	origin   origin
	original string
}

func (m baseMessage) getSender() string {
	return m.origin.getSender()
}

func (m baseMessage) getUnparsed() string {
	return m.original
}

type nickMessage struct {
	baseMessage

	nickname string
}

func (m nickMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "NICK %s\r\n", m.nickname)
}

type userMessage struct {
	baseMessage

	user     string
	realname string
}

func (m userMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "USER %s 0 * :%s\r\n", m.user, m.realname)
}

type joinMessage struct {
	baseMessage

	channelTag string
}

func (m joinMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "JOIN %s\r\n", m.channelTag)
}

type partMessage struct {
	baseMessage

	channelTag string
}

func (m partMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "PART %s\r\n", m.channelTag)
}

type privMessage struct {
	baseMessage

	target, content string
}

func (m privMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "PRIVMSG %s :%s\r\n", m.target, m.content)
}

type pongMessage struct {
	baseMessage

	server string
}

func (m pongMessage) encode() []byte {
	return fmt.Appendf([]byte{}, "PONG :%s\r\n", m.server)
}

type quitMessage struct {
	baseMessage

	content string
}

func (m quitMessage) encode() []byte {
	raw := fmt.Appendf([]byte{}, "QUIT :%s\r\n", m.content)
	return raw
}

type replyMessage struct {
	baseMessage

	target  string
	code    uint16
	content string
}

type noticeMessage struct {
	baseMessage

	content string
}

type pingMessage struct {
	baseMessage
}

type kickMessage struct {
	baseMessage

	channelTag, nickname, reason string
}

type errorMessage struct {
	baseMessage

	content string
}

type modeMessage struct {
	baseMessage

	target, modes string
}

type unknownMessage struct {
	baseMessage
}

func decodeMessage(raw []byte) message {
	sraw := string(raw)
	baseMsg := baseMessage{
		origin:   withoutOrigin{},
		original: sraw,
	}

	var parts []string
	if raw[0] == ':' {
		parts = strings.SplitN(sraw[1:], " ", 3)
		var origin origin
		if nickname, identifier, found := strings.Cut(parts[0], "!"); found {
			origin = userOrigin{
				nickname:   nickname,
				identifier: identifier,
			}
		} else {
			origin = serverOrigin{
				servername: parts[0],
			}
		}
		baseMsg.origin = origin
		parts = parts[1:]
	} else {
		parts = strings.SplitN(sraw, " ", 2)
	}

	if len(parts[0]) == 3 {
		if code, err := strconv.ParseUint(parts[0], 10, 16); err == nil {
			parts = strings.SplitN(parts[1], " ", 2)
			parts[1] = strings.TrimPrefix(parts[1], ":")
			return replyMessage{
				baseMessage: baseMsg,
				target:      parts[0],
				code:        uint16(code),
				content:     parts[1],
			}
		}
	}

	var msg message
	switch parts[0] {
	case "NICK":
		parts[1] = strings.TrimPrefix(parts[1], ":")
		msg = nickMessage{
			baseMessage: baseMsg,
			nickname:    parts[1],
		}
	case "JOIN":
		msg = joinMessage{
			baseMessage: baseMsg,
			channelTag:  parts[1],
		}
	case "PART":
		msg = partMessage{
			baseMessage: baseMsg,
			channelTag:  parts[1],
		}
	case "PRIVMSG":
		parts = strings.SplitN(parts[1], " ", 2)
		parts[1] = strings.TrimPrefix(parts[1], ":")
		msg = privMessage{
			baseMessage: baseMsg,
			target:      parts[0],
			content:     parts[1],
		}
	case "QUIT":
		msg = quitMessage{
			baseMessage: baseMsg,
		}
	case "NOTICE":
		parts = strings.SplitN(parts[1], " ", 2)
		parts[1] = strings.TrimPrefix(parts[1], ":")
		msg = noticeMessage{
			baseMessage: baseMsg,
			content:     parts[1],
		}
	case "KICK":
		parts = strings.SplitN(parts[1], " ", 3)
		kickMsg := kickMessage{
			baseMessage: baseMsg,
			channelTag:  parts[0],
			nickname:    parts[1],
		}
		if len(parts) == 3 {
			kickMsg.reason = parts[2][1:]
		}
		msg = kickMsg
	case "PING":
		baseMsg.origin = serverOrigin{
			servername: parts[1][1:],
		}
		msg = pingMessage{
			baseMessage: baseMsg,
		}
	case "ERROR":
		parts[1] = strings.TrimPrefix(parts[1], ":")
		msg = errorMessage{
			baseMessage: baseMsg,
			content:     parts[1],
		}
	case "MODE":
		parts = strings.SplitN(parts[1], " ", 2)
		parts[1] = strings.TrimPrefix(parts[1], ":")
		msg = modeMessage{
			baseMessage: baseMsg,
			target:      parts[0],
			modes:       parts[1],
		}
	default:
		msg = unknownMessage{
			baseMessage: baseMsg,
		}
	}

	return msg
}

type NetworkMessage struct {
	Content string
}

type ChannelMessage struct {
	Sender  string
	Content string
}
