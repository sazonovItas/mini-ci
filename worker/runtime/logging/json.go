package logging

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

type jsonLogEntry struct {
	Log    string    `json:"log,omitempty"`
	Stream string    `json:"stream,omitempty"`
	Time   time.Time `json:"time,omitzero"`
}

func newJsonLogEntry(log, stream string) jsonLogEntry {
	return jsonLogEntry{
		Log:    log,
		Stream: stream,
		Time:   time.Now().UTC(),
	}
}

type jsonLogger struct {
	dataStore string
}

func NewJSONLogger(dataStore string) *jsonLogger {
	return &jsonLogger{
		dataStore: dataStore,
	}
}

func (l jsonLogger) Process(id string, stdout <-chan string, stderr <-chan string) error {
	var (
		isErrClosed = false
		isOutClosed = false
	)

	file, err := os.OpenFile(
		l.getLogFilePath(id),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		os.FileMode(0o600),
	)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	for !isOutClosed || !isErrClosed {
		select {
		case log, ok := <-stdout:
			if !ok {
				isOutClosed = true
				break
			}

			if err := l.writeLog(w, log, "stdout"); err != nil {
				return err
			}
		case log, ok := <-stderr:
			if !ok {
				isErrClosed = true
				break
			}

			if err := l.writeLog(w, log, "stderr"); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l jsonLogger) writeLog(w io.Writer, log, stream string) error {
	entry := newJsonLogEntry(log, stream)

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (l jsonLogger) getLogFilePath(id string) string {
	return filepath.Join(l.dataStore, "containers", id, "log.json")
}
