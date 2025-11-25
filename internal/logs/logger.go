package logs

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	file *os.File
}

func (l Logger) Close() {
	l.file.Close()
}

func Setup(logFilename string) (Logger, error) {
	file, err := os.OpenFile(logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return Logger{}, fmt.Errorf("failed to open log file %s: %v", logFilename, err)
	}

	log.SetOutput(file)

	return Logger{
		file: file,
	}, nil
}
