package main

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// DualOutputHook is a logrus hook to send logs to multiple destinations
type DualOutputHook struct {
	Outputs   []io.Writer
	Formatter log.Formatter
	LogLevels []log.Level
}

// Fire handles a log event and sends it to multiple outputs
func (h *DualOutputHook) Fire(entry *log.Entry) error {
	line, err := h.Formatter.Format(entry)
	if err != nil {
		return err
	}

	for _, output := range h.Outputs {
		if _, err := output.Write(line); err != nil {
			return err
		}
	}
	return nil
}

// Levels returns the log levels that this hook handles
func (h *DualOutputHook) Levels() []log.Level {
	return h.LogLevels
}

// setupLogs sets up log output to both stdout (with color) and a log file (without color)
func setupLogs() {
	consoleFormatter := &log.TextFormatter{
		FullTimestamp: true,
		DisableColors: false,
	}

	fileFormatter := &log.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	}

	if os.Getenv("ENABLE_FILE_LOGGING") == "1" {

		timestamp := time.Now().Format("20060102-150405")
		logFile, err := os.OpenFile(fmt.Sprintf("reaper-%s.log", timestamp), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}

		dualOutputHook := &DualOutputHook{
			Outputs:   []io.Writer{logFile},                                                                      // Dual outputs
			Formatter: fileFormatter,                                                                             // Default formatter with color
			LogLevels: []log.Level{log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, log.FatalLevel}, // All levels
		}

		log.AddHook(dualOutputHook)

	}

	log.SetFormatter(consoleFormatter) // Colorize console output

	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "WARN":
	case "WARNING":
		log.SetLevel(log.WarnLevel)
	case "ERR":
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func Fail(msg string) {
	log.Fatalf(msg)
	os.Exit(1)
}
