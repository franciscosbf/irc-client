package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/franciscosbf/irc-client/pkg/app"
)

func main() {
	var logFilename string

	flag.StringVar(&logFilename, "log", "irc-chat.log", "log filename")
	flag.Parse()

	if err := app.Run(logFilename); err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
	}

	// conn, err := irc.DialNetworkConnection("irc.freenode.net", 6667)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// network := irc.NewNetwork(conn, "crazylol", "lolamol")
	// network.StartListener()
	// if err := network.Register(); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// go func() {
	// 	time.Sleep(time.Second * 10)
	// 	network.ChangeNickname("boda")
	// 	channel, err := network.JoinChannel("#gamesurge")
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	//
	// 	if err := channel.Part(); err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	//
	// 	for {
	// 		if msg, ok := channel.ReceiveMessage(); ok {
	// 			fmt.Println(msg.Content)
	// 		} else {
	// 			return
	// 		}
	// 	}
	// }()
	//
	// for {
	// 	if msg, ok := network.ReceiveMessage(); ok {
	// 		fmt.Println(msg.Content)
	// 	} else {
	// 		return
	// 	}
	// }
}
