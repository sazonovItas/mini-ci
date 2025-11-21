package logging

import (
	"bufio"
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
	"github.com/muesli/cancelreader"
	"github.com/sazonovItas/mini-ci/worker/runtime/filesystem"
)

type Logger interface {
	Process(id string, stdout <-chan string, stderr <-chan string) error
}

type LogIO struct {
	Stdout io.Reader
	Stderr io.Reader
}

func ProcessLogs(parentCtx context.Context, container containerd.Container, logIO LogIO, loggers ...Logger) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	stdoutR, err := cancelreader.NewReader(logIO.Stdout)
	if err != nil {
		return err
	}

	stderrR, err := cancelreader.NewReader(logIO.Stderr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		stdoutR.Cancel()
		stderrR.Cancel()
	}()

	pipeStdoutR, pipeStdoutW := io.Pipe()
	pipeStderrR, pipeStderrW := io.Pipe()
	copyStream := func(reader io.Reader, writer *io.PipeWriter) {
		buf := make([]byte, 32<<10)
		_, err := io.CopyBuffer(writer, reader, buf)
		if err != nil {
			// TODO: add logs
		}
	}

	go copyStream(stdoutR, pipeStdoutW)
	go copyStream(stderrR, pipeStderrW)

	var wg sync.WaitGroup
	processLogFunc := func(reader io.Reader, chs ...chan string) {
		defer func() {
			for _, ch := range chs {
				close(ch)
			}
		}()

		r := bufio.NewReader(reader)

		var (
			s   string
			err error
		)

		for err == nil {
			s, err = r.ReadString('\n')
			if s != "" {
				for _, ch := range chs {
					ch <- s
				}
			}

			if err != nil && !errors.Is(err, io.EOF) {
				// TODO: add logs
			}
		}
	}

	stdouts := make([]chan string, len(loggers))
	stderrs := make([]chan string, len(loggers))
	for i := range len(loggers) {
		stdouts[i] = make(chan string, 10000)
		stderrs[i] = make(chan string, 10000)

		wg.Go(func() {
			if err := loggers[i].Process(container.ID(), stdouts[i], stderrs[i]); err != nil {
				// TODO: add logs
			}
		})
	}

	wg.Go(func() { processLogFunc(pipeStdoutR, stdouts...) })
	wg.Go(func() { processLogFunc(pipeStderrR, stderrs...) })

	go func() {
		defer pipeStdoutW.Close()
		defer pipeStderrW.Close()

		exitch, err := getContainerWait(ctx, container)
		if err != nil {
			// TODO: add logs
			return
		}

		<-exitch
	}()

	wg.Wait()

	return nil
}

func getContainerWait(ctx context.Context, container containerd.Container) (<-chan containerd.ExitStatus, error) {
	task, err := container.Task(ctx, nil)
	if err == nil {
		return task.Wait(ctx)
	}
	if !errdefs.IsNotFound(err) {
		return nil, err
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// TODO: add normal error from the errdefs
			return nil, errors.New("timed out waiting for container task to start")
		case <-ticker.C:
			task, err = container.Task(ctx, nil)
			if err != nil {
				if errdefs.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			return task.Wait(ctx)
		}
	}
}

func waitForLogger(dataStore, id string) error {
	return filesystem.WithLock(getLockPath(dataStore, id), func() error {
		return nil
	})
}

func getLockPath(dataStore, id string) string {
	return filepath.Join(dataStore, "containers", id, "log-lock")
}
