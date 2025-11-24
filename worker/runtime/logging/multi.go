package logging

import "sync"

type multiLogger struct {
	loggers []Logger
}

func NewMultiLogger(loggers ...Logger) *multiLogger {
	return &multiLogger{loggers: loggers}
}

func (l multiLogger) Process(id string, stdout <-chan string, stderr <-chan string) error {
	var wg sync.WaitGroup

	stdouts := make([]chan string, len(l.loggers))
	stderrs := make([]chan string, len(l.loggers))
	for i, logger := range l.loggers {
		stdouts[i], stderrs[i] = make(chan string, 100), make(chan string, 100)
		wg.Go(func() { _ = logger.Process(id, stdouts[i], stderrs[i]) })
	}

	var (
		isErrClosed = false
		isOutClosed = false
	)

	for !isOutClosed || !isErrClosed {
		select {
		case log, ok := <-stdout:
			if !ok {
				isOutClosed = true
				break
			}

			for _, outch := range stdouts {
				outch <- log
			}
		case log, ok := <-stderr:
			if !ok {
				isErrClosed = true
				break
			}

			for _, errch := range stderrs {
				errch <- log
			}
		}
	}

	wg.Wait()

	return nil
}
