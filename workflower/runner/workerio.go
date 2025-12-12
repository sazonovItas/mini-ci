package runner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/eventq"
	"github.com/zishang520/socket.io/servers/socket/v3"
)

const (
	workerEventName     = "message"
	queueDiscardTimeout = 250 * time.Millisecond
)

type WorkerIORunner struct {
	ioServer   *socket.Server
	httpServer *http.Server

	recvq eventq.Queue[events.Event]
	sendq eventq.Queue[events.Event]

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorkerIORunner(address, endpoint string) *WorkerIORunner {
	opts := socket.DefaultServerOptions()
	opts.SetPingTimeout(20 * time.Second)
	opts.SetPingInterval(25 * time.Second)
	opts.SetMaxHttpBufferSize(1e6)

	ioServer := socket.NewServer(nil, opts)

	handler := http.NewServeMux()
	handler.Handle(endpoint, ioServer.ServeHandler(nil))
	httpServer := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 250 * time.Millisecond,
	}

	runner := &WorkerIORunner{
		ioServer:   ioServer,
		httpServer: httpServer,
		recvq:      eventq.New(queueDiscardTimeout, func(events.Event) {}),
		sendq:      eventq.New(queueDiscardTimeout, func(events.Event) {}),
	}

	_ = ioServer.
		On("connection", func(workers ...any) {
			log.G(runner.ctx).Debug("worker connected")

			worker := workers[0].(*socket.Socket)
			workerCtx, cancel := context.WithCancel(runner.ctx)

			_ = worker.On(workerEventName, func(msgs ...any) {
				if len(msgs) == 0 {
					return
				}

				jsonMessage, err := json.Marshal(msgs[0])
				if err != nil {
					log.G(workerCtx).WithError(err).Error("failed to marshal recieved message")
					return
				}

				var message events.Message
				if err := json.Unmarshal(jsonMessage, &message); err != nil {
					log.G(workerCtx).WithError(err).Error("failed to unmarshal recieved message into event")
					return
				}

				runner.recvq.Publish(message.Event)
			})

			_ = worker.On("disconnect", func(msgs ...any) {
				log.G(runner.ctx).Debug("worker disconnected")
				cancel()
			})

			runner.wg.Go(func() { runner.startSender(workerCtx, worker) })
		})

	return runner
}

func (r *WorkerIORunner) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.wg.Go(func() {
		if err := r.httpServer.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.G(r.ctx).WithError(err).Error("failed listen and serve http server")
			}
		}
	})

	return nil
}

func (r *WorkerIORunner) Stop(ctx context.Context) error {
	r.cancel()
	r.sendq.Shutdown()
	r.recvq.Shutdown()

	r.ioServer.Close(func(err error) {
		if err != nil {
			log.G(ctx).WithError(err).Error("failed to close worker socket io")
		}
	})

	if err := r.httpServer.Shutdown(ctx); err != nil {
		log.G(ctx).WithError(err).Error("failed to shutdown worker http server")
	}

	r.wg.Wait()

	return nil
}

func (r *WorkerIORunner) Events() (<-chan events.Event, io.Closer) {
	return r.recvq.Subscribe()
}

func (r *WorkerIORunner) Publish(_ context.Context, event events.Event) error {
	r.sendq.Publish(event)
	return nil
}

func (r *WorkerIORunner) startSender(ctx context.Context, worker *socket.Socket) {
	const requeueTimeout = 1 * time.Second

	evch, closer := r.sendq.Subscribe()
	defer func() {
		_ = closer.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				return
			}

			if err := worker.Emit(workerEventName, events.Message{Event: event}); err != nil {
				log.G(ctx).WithError(err).Error("failed to send message to worker")
				go r.requeueEventAfter(ctx, event, requeueTimeout)
			}
		}
	}
}

func (r *WorkerIORunner) requeueEventAfter(
	ctx context.Context,
	event events.Event,
	timeout time.Duration,
) {
	select {
	case <-ctx.Done():
	case <-time.After(timeout):
		r.sendq.Publish(event)
	}
}
