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
}
