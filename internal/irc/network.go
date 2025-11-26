package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	networkTlsPort        = 6697
	readerBufSize         = 520
	messagesBufSize       = 32
	dialConnectionTimeout = time.Second * 8
)

type Connection interface {
	read() ([]byte, bool, error)
	write(b []byte) error
	close()
}

type NetworkConnection struct {
	conn   net.Conn
	reader *bufio.Reader
}

func (nc *NetworkConnection) read() ([]byte, bool, error) {
	return nc.reader.ReadLine()
}

func (nc *NetworkConnection) write(b []byte) error {
	_, err := nc.conn.Write(b)

	return err
}

func (nc *NetworkConnection) close() {
	_ = nc.conn.Close()
}

func DialNetworkConnection(host string, port uint16) (*NetworkConnection, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(int(port)))

	var (
		conn net.Conn
		err  error
	)
	netDialer := net.Dialer{
		Timeout: dialConnectionTimeout,
	}
	if port == networkTlsPort {
		tlsDialer := tls.Dialer{
			NetDialer: &netDialer,
		}
		conn, err = tlsDialer.Dial("tcp4", addr)
	} else {
		conn, err = netDialer.Dial("tcp4", addr)
	}
	if err != nil {
		return nil, err
	}

	return &NetworkConnection{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, readerBufSize),
	}, nil
}

type NetworkChannel struct {
	tag        string
	closed     atomic.Bool
	noMoreMsgs chan struct{}
	msgs       chan ChannelMessage
	network    *Network
	users      map[string]struct{}
}

func (nc *NetworkChannel) signalNoMoreMsgs() {
	select {
	case nc.noMoreMsgs <- struct{}{}:
	default:
	}
}

func (nc *NetworkChannel) stopReceivingMsgs() {
	nc.closed.Store(true)

	nc.signalNoMoreMsgs()
}

func (nc *NetworkChannel) GetTag() string {
	return nc.tag
}

func (nc *NetworkChannel) SendMessage(content string) error {
	privMsg := privMessage{
		target:  nc.tag,
		content: content,
	}
	return nc.network.conn.write(privMsg.encode())
}

func (nc *NetworkChannel) ReceiveMessage() (ChannelMessage, bool) {
	if nc.closed.Load() {
		return ChannelMessage{}, false
	}

	select {
	case <-nc.noMoreMsgs:
		return ChannelMessage{}, false
	case msg := <-nc.msgs:
		return msg, true
	}
}

func (nc *NetworkChannel) Part() error {
	if !nc.closed.CompareAndSwap(false, true) {
		return nil
	}

	nc.signalNoMoreMsgs()

	partMsg := partMessage{
		channelTag: nc.tag,
	}
	err := nc.network.conn.write(partMsg.encode())

	nc.network.removeChannel(nc.tag)

	return err
}

type Network struct {
	registered atomic.Bool

	nmx      sync.Mutex
	nickname string

	listenerStarted bool
	conn            Connection
	msgs            chan NetworkMessage

	cmx           sync.Mutex
	channels      map[string]*NetworkChannel
	usersChannels map[string]map[string]*NetworkChannel
}

func (n *Network) closeAndCleanup() {
	n.conn.close()

	n.cmx.Lock()
	defer n.cmx.Unlock()

	n.channels = nil
	n.usersChannels = nil
}

func (n *Network) hasNickname(nickname string) bool {
	n.nmx.Lock()
	defer n.nmx.Unlock()

	return n.nickname == nickname
}

func (n *Network) setNickname(nickname string) {
	n.nmx.Lock()
	defer n.nmx.Unlock()

	n.nickname = nickname
}

func (n *Network) replaceNickname(oldNickName, newNickname string) bool {
	n.nmx.Lock()
	defer n.nmx.Unlock()

	if oldNickName != n.nickname {
		return false
	}

	n.nickname = newNickname

	return true
}

func (n *Network) getNickname() string {
	n.nmx.Lock()
	defer n.nmx.Unlock()

	return n.nickname
}

func (n *Network) removeUser(nickname string) []*NetworkChannel {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	userChannels, ok := n.usersChannels[nickname]
	if !ok {
		return nil
	}

	delete(n.usersChannels, nickname)

	channels := []*NetworkChannel{}
	for _, channel := range userChannels {
		delete(channel.users, nickname)
		channels = append(channels, channel)
	}

	return channels
}

func (n *Network) addChannel(tag string, channel *NetworkChannel) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	n.channels[tag] = channel
}

func (n *Network) removeChannel(tag string) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channel, ok := n.channels[tag]
	if !ok {
		return
	}

	delete(n.channels, tag)

	for nickname := range channel.users {
		delete(n.usersChannels, nickname)
	}
}

func (n *Network) getChannel(tag string) (*NetworkChannel, bool) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channel, ok := n.channels[tag]
	return channel, ok
}

func (n *Network) getChannels() []*NetworkChannel {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channels := []*NetworkChannel{}
	for _, channel := range n.channels {
		channels = append(channels, channel)
	}

	return channels
}

func (n *Network) addChannelUsers(nicknames []string, tag string) (*NetworkChannel, bool) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channel, ok := n.channels[tag]
	if !ok {
		return nil, false
	}

	for _, nickname := range nicknames {
		channel.users[nickname] = struct{}{}

		userChannels, ok := n.usersChannels[nickname]
		if !ok {
			userChannels = map[string]*NetworkChannel{}
			n.usersChannels[nickname] = userChannels
		}
		userChannels[tag] = channel
	}

	return channel, true
}

func (n *Network) removeChannelUser(nickname, tag string) (*NetworkChannel, bool) {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channel, ok := n.channels[tag]
	if !ok {
		return nil, false
	}

	if userChannels, ok := n.usersChannels[nickname]; ok {
		delete(userChannels, tag)
	}

	delete(channel.users, nickname)

	return channel, true
}

func (n *Network) replaceUser(oldNickName, newNickname string) []*NetworkChannel {
	n.cmx.Lock()
	defer n.cmx.Unlock()

	channels := []*NetworkChannel{}

	userChannels, ok := n.usersChannels[oldNickName]
	if ok {
		for _, channel := range userChannels {
			delete(channel.users, oldNickName)
			channel.users[newNickname] = struct{}{}
			channels = append(channels, channel)
		}
	} else {
		userChannels = map[string]*NetworkChannel{}
	}
	n.usersChannels[newNickname] = userChannels

	delete(n.usersChannels, oldNickName)

	return channels
}

func (n *Network) fetchMessage() (message, error) {
	raw, truncated, err := n.conn.read()
	if truncated {
		return nil, errors.New("message was truncated due to its size")
	} else if err != nil {
		return nil, err
	}

	return decodeMessage(raw), nil
}

func (n *Network) StartListener() {
	if n.listenerStarted {
		return
	}
	n.listenerStarted = true

	go func() {
		defer func() {
			for _, channel := range n.getChannels() {
				channel.stopReceivingMsgs()
			}

			n.closeAndCleanup()

			close(n.msgs)
		}()

		var (
			msg message
			err error
		)
		for {
			msg, err = n.fetchMessage()
			if err != nil {
				if err != io.EOF && !errors.Is(err, net.ErrClosed) {
					log.Printf("failed to read message from network: %v\n", err)
				}
				return
			}

			switch cmsg := msg.(type) {
			case replyMessage:
				switch cmsg.code {
				case rpl_WELCOME:
					n.registered.Store(true)
					n.setNickname(cmsg.target)
					fallthrough
				case
					rpl_YOURHOST,
					rpl_CREATED,
					rpl_MYINFO,
					rpl_BOUNCE,
					rpl_LUSERCLIENT,
					rpl_LUSEROP,
					rpl_LUSERUNKNOWN,
					rpl_LUSERCHANNELS,
					rpl_LUSERME,
					rpl_ADMINME,
					rpl_LUSERS,
					rpl_GUSERS,
					rpl_AWAY,
					rpl_MOTDSTART,
					rpl_MOTD,
					rpl_DHOST,
					err_NOMOTD,
					err_NICKCOLLISION,
					err_NOTREGISTERED,
					err_ALREADYREGISTRED:
					cmsg.content = strings.TrimPrefix(cmsg.content, n.getNickname())
					cmsg.content = strings.TrimPrefix(cmsg.content, " :")
					n.msgs <- NetworkMessage{
						Content: cmsg.content,
					}
				case err_NOSUCHCHANNEL, err_NOTONCHANNEL:
					n.msgs <- NetworkMessage{
						Content: cmsg.content,
					}
				case err_ERRONEUSNICKNAME:
					nickname := n.getNickname()
					n.msgs <- NetworkMessage{
						Content: "Nickname " + nickname + " is invalid",
					}
				case err_NICKNAMEINUSE:
					nickname, _, _ := strings.Cut(cmsg.content, " :")
					n.msgs <- NetworkMessage{
						Content: nickname + " is already in use",
					}
				case rpl_TOPIC, err_INVITEONLYCHAN, err_BANNEDFROMCHAN, err_CANNOTSENDTOCHAN:
					parts := strings.SplitN(cmsg.content, " :", 2)
					tag := parts[0]
					topic := parts[1]
					channel, ok := n.getChannel(tag)
					if !ok {
						break
					}
					channel.msgs <- ChannelMessage{
						Content: topic,
					}
				case rpl_NAMREPLY:
					parts := strings.Split(cmsg.content, " :")
					tag := strings.Split(parts[0], " ")[1]
					nicknames := []string{}
					for nickname := range strings.SplitSeq(parts[1], " ") {
						nicknames = append(nicknames, strings.TrimLeft(nickname, "@+"))
					}
					n.addChannelUsers(nicknames, tag)
				case rpl_ENDOFNAMES, rpl_ENDOFMOTD, rpl_TOPICWHOTIME:
				case err_RESTRICTED:
					n.msgs <- NetworkMessage{
						Content: cmsg.content,
					}
					return
				default:
					log.Printf(
						"unknown reply -> %s\n", cmsg.getUnparsed())
				}
			case quitMessage:
				uorigin := cmsg.origin.(userOrigin)
				nickname := uorigin.nickname
				for _, channel := range n.removeUser(nickname) {
					channel.msgs <- ChannelMessage{
						Content: nickname + " has quit",
					}
				}
			case kickMessage:
				tag := cmsg.channelTag
				nickname := cmsg.nickname
				channel, ok := n.removeChannelUser(nickname, tag)
				if !ok {
					break
				}
				var msgContent string
				if n.hasNickname(nickname) {
					msgContent = "You have been kicked from the channel"
				} else {
					msgContent = nickname + " has been kicked from the channel"
				}
				if cmsg.reason != "" {
					msgContent += ". Reason: " + cmsg.reason
				}
				channel.msgs <- ChannelMessage{
					Content: msgContent,
				}
			case joinMessage:
				uorigin := cmsg.origin.(userOrigin)
				nickname := uorigin.nickname
				tag := strings.TrimLeft(cmsg.channelTag, ":")
				channel, ok := n.addChannelUsers([]string{nickname}, tag)
				if !ok {
					break
				}
				var msgContent string
				if n.hasNickname(nickname) {
					msgContent = "You have joined " + tag
				} else {
					msgContent = nickname + " has joined " + tag
				}
				channel.msgs <- ChannelMessage{
					Content: msgContent,
				}
			case partMessage:
				uorigin := cmsg.origin.(userOrigin)
				nickname := uorigin.nickname
				tag := strings.TrimLeft(cmsg.channelTag, ":")
				channel, ok := n.removeChannelUser(nickname, tag)
				if !ok {
					break
				}
				if n.hasNickname(nickname) {
					break
				}
				channel.msgs <- ChannelMessage{
					Content: nickname + " has left " + tag,
				}
			case nickMessage:
				uorigin := cmsg.origin.(userOrigin)
				oldNickName := uorigin.nickname
				newNickname := cmsg.nickname
				var msgContent string
				if n.replaceNickname(oldNickName, newNickname) {
					msgContent = fmt.Sprintf("You're now known as %s", newNickname)
					n.msgs <- NetworkMessage{
						Content: msgContent,
					}
				} else {
					msgContent = fmt.Sprintf("%s changed his nickname to %s", oldNickName, newNickname)
				}
				for _, channel := range n.replaceUser(oldNickName, newNickname) {
					channel.msgs <- ChannelMessage{
						Content: msgContent,
					}
				}
			case privMessage:
				uorigin, ok := cmsg.origin.(userOrigin)
				if !ok {
					break
				}
				if !strings.HasPrefix(cmsg.target, "#") {
					break
				}
				channel, ok := n.getChannel(cmsg.target)
				if !ok {
					break
				}
				channel.msgs <- ChannelMessage{
					Sender:  uorigin.nickname,
					Content: cmsg.content,
				}
			case pingMessage:
				sorigin := cmsg.origin.(serverOrigin)
				pongMsg := pongMessage{
					server: sorigin.servername,
				}
				if err := n.conn.write(pongMsg.encode()); err != nil {
					log.Printf("failed to send pong: %v\n", err)
					return
				}
			case noticeMessage:
				n.msgs <- NetworkMessage{
					Content: cmsg.content,
				}
			case errorMessage:
				_, after, found := strings.Cut(cmsg.content, " :")
				if found {
					cmsg.content = after
				}
				n.msgs <- NetworkMessage{
					Content: "ERROR " + cmsg.content,
				}
				return
			case modeMessage:
				if strings.HasPrefix(cmsg.target, "#") {
					break
				}
				n.msgs <- NetworkMessage{
					Content: "Your modes are " + cmsg.modes,
				}
			case unknownMessage:
				log.Printf("unknown message -> %s\n", msg.getUnparsed())
			}
		}
	}()
}

func (n *Network) Register(nickname, realname string) error {
	nickMsg := nickMessage{
		nickname: nickname,
	}
	if err := n.conn.write(nickMsg.encode()); err != nil {
		return err
	}

	userMsg := userMessage{
		realname: realname,
	}
	if currUser, err := user.Current(); err == nil {
		userMsg.user = currUser.Username
	} else {
		userMsg.user = nickname
	}
	if err := n.conn.write(userMsg.encode()); err != nil {
		return err
	}

	return nil
}

func (n *Network) IsRegistered() bool {
	return n.registered.Load()
}

func (n *Network) GetNickname() string {
	return n.getNickname()
}

func (n *Network) ReceiveMessage() (NetworkMessage, bool) {
	msg, ok := <-n.msgs
	return msg, ok
}

func (n *Network) JoinChannel(tag string) (*NetworkChannel, error) {
	if _, ok := n.getChannel(tag); ok {
		return nil, fmt.Errorf("already connected to %s", tag)
	}

	channel := &NetworkChannel{
		tag:        tag,
		noMoreMsgs: make(chan struct{}, 1),
		msgs:       make(chan ChannelMessage, messagesBufSize),
		network:    n,
		users:      map[string]struct{}{},
	}

	joinMsg := joinMessage{
		channelTag: tag,
	}
	if err := n.conn.write(joinMsg.encode()); err != nil {
		return nil, err
	}

	n.addChannel(tag, channel)

	return channel, nil
}

func (n *Network) ChangeNickname(newNickname string) error {
	if n.hasNickname(newNickname) {
		return nil
	}

	nickMsg := nickMessage{
		nickname: newNickname,
	}
	return n.conn.write(nickMsg.encode())
}

func (n *Network) Quit(message string) error {
	if !n.listenerStarted {
		close(n.msgs)
	}

	quitMsg := quitMessage{
		content: message,
	}
	if err := n.conn.write(quitMsg.encode()); err != nil {
		return err
	}

	n.closeAndCleanup()

	return nil
}

func NewNetwork(conn Connection) *Network {
	return &Network{
		conn:          conn,
		channels:      map[string]*NetworkChannel{},
		usersChannels: map[string]map[string]*NetworkChannel{},
		msgs:          make(chan NetworkMessage, messagesBufSize),
	}
}
