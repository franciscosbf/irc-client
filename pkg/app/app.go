package app

import (
	"github.com/franciscosbf/irc-client/internal/logs"
	"github.com/franciscosbf/irc-client/internal/ui"
)

func Run(logFilename string) error {
	logger, err := logs.Setup(logFilename)
	if err != nil {
		return err
	}
	defer logger.Close()

	return ui.Run()
}
