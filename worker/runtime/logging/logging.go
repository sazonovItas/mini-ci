package logging

import (
	"io"
	"time"
)

type LogIO struct {
	Stdout io.Reader
	Stderr io.Reader
}

type LogEntry struct {
	Log    string    `json:"log,omitempty"`
	Stream string    `json:"stream,omitempty"`
	Time   time.Time `json:"time,omitempty"`
}

type Logger interface {
	Process(stdout <-chan string, stdin <-chan string) error
}

func NewLogEntry(log, stream string) LogEntry {
	return LogEntry{
		Log:    log,
		Stream: stream,
		Time:   time.Now().UTC(),
	}
}
