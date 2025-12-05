# IRC Client

A perpetual beta implementation etched into my machine until the day it's finally
wiped. Only `#...` channels are supported and private messages aren't in the features
list unless I decide to come back to this project. I only implemented the necessary
components described in the [RFC 2812](https://datatracker.ietf.org/doc/html/rfc2812)
and some unofficial reply codes.

## Supported Keybinds

- `Alt+h/Alt+j/Alt+k/Alt+l` - scroll left/up/down/right in the current chat
- `Alt+b` - go to the bottom of the chat
- `Alt+p` -  go to chat above
- `Alt+n` - go to the chat bellow
- `Alt+t` - toggle between network chat and the current channel chat
- `Enter` - issue a command
- `Ctrl+c/Esc` - exit the IRC client

There are others that can be used in the commands prompt (shipped by default in the
framework that implements this feature). Feel free to explore which ones are as the
list is a bit long.

## Supported commands

```
/help                              Shows this message
/connect <host> <nickname> <name>  Connects to a network
/disconnect                        Disconnects from a network
/join <channel>                    Connects to a channel in the network
/part <channel>                    Disconnects from a channel in the network
/nick <nickname>                   Changes your nickname in the network
/quit                              Closes the IRC Client
<bunch of text>                    Sends a message in the current channel`
```
