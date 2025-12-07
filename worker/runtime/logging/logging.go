package logging

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/muesli/cancelreader"
)

type Logger interface {
	Process(id string, stdout <-chan string, stderr <-chan string) error
}

type LogIO struct {
	Stdout io.Reader
	Stderr io.Reader
}

func ProcessLogs(
	parentCtx context.Context,
	container containerd.Container,
	logIO LogIO,
	loggers ...Logger,
) error {
	if len(loggers) == 0 {
		return nil
	}

	logger := loggers[0]
	if len(loggers) > 1 {
		logger = NewMultiLogger(loggers...)
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	return processLogs(ctx, container, logIO, logger)
}

func processLogs(
	ctx context.Context,
	container containerd.Container,
	logIO LogIO,
	logger Logger,
) error {
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
		if err != nil && !errors.Is(err, cancelreader.ErrCanceled) {
			log.G(ctx).WithError(err).Error("failed to copy stream")
		}
	}

	go copyStream(stdoutR, pipeStdoutW)
	go copyStream(stderrR, pipeStderrW)

	var wg sync.WaitGroup
	processLogFunc := func(reader io.Reader, ch chan string) {
		defer close(ch)

		r := bufio.NewReader(reader)

		var (
			s   string
			err error
		)

		for err == nil {
			s, err = r.ReadString('\n')
			if len(s) > 0 {
				ch <- s
			}

			if err != nil && !errors.Is(err, io.EOF) {
				log.G(ctx).WithError(err).Error("failed to read log")
			}
		}
	}

	stdout := make(chan string, 100)
	stderr := make(chan string, 100)
	wg.Go(func() {
		if err := logger.Process(container.ID(), stdout, stderr); err != nil {
			log.G(ctx).WithError(err).Error("logger failed to process logs")
		}
	})

	wg.Go(func() { processLogFunc(pipeStdoutR, stdout) })
	wg.Go(func() { processLogFunc(pipeStderrR, stderr) })

	go func() {
		defer func() {
			_ = pipeStdoutW.Close()
		}()
		defer func() {
			_ = pipeStderrW.Close()
		}()

		exitch, err := getContainerWait(ctx, container)
		if err != nil {
			log.G(ctx).WithError(err).Error("failed to get container wait")
			return
		}

		<-exitch
	}()

	wg.Wait()

	return nil
}

func getContainerWait(
	ctx context.Context,
	container containerd.Container,
) (<-chan containerd.ExitStatus, error) {
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
