package runner

import (
	"context"
	"encoding/json"
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
	requeueTimeout      = 1 * time.Second
)

type WorkerIOConfig struct {
	Address  string
	Endpoint string
}

type WorkerIO struct {
	bus events.Bus

	ioServer   *socket.Server
	httpServer *HTTPServer

	recvq eventq.Queue[events.Event]
	sendq eventq.Queue[events.Event]

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorkerIO(bus events.Bus, cfg WorkerIOConfig) *WorkerIO {
	opts := socket.DefaultServerOptions()
	opts.SetPingTimeout(20 * time.Second)
	opts.SetPingInterval(25 * time.Second)
	opts.SetMaxHttpBufferSize(1e6)

	ioServer := socket.NewServer(nil, opts)

	handler := http.NewServeMux()
	handler.Handle(cfg.Endpoint, ioServer.ServeHandler(nil))
	httpServer := NewHTTPServer(cfg.Address, handler)

	runner := &WorkerIO{
		ioServer:   ioServer,
		httpServer: httpServer,
		recvq:      eventq.New(queueDiscardTimeout, func(events.Event) {}),
		sendq:      eventq.New(queueDiscardTimeout, func(events.Event) {}),
	}

	_ = ioServer.
		On("connection", func(workers ...any) {
			log.G(runner.ctx).Debug("worker connected")

			worker := workers[0].(*socket.Socket)
			workerCtx, workerCancel := context.WithCancel(runner.ctx)

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
				workerCancel()
			})

			runner.wg.Go(func() { runner.startSender(workerCtx, worker) })
		})

	return runner
}

func (r *WorkerIO) Start(ctx context.Context) {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.wg.Go(func() { r.startReceiver(ctx) })
	r.wg.Go(func() { r.startBusForwarder(ctx) })

	r.httpServer.Start(r.ctx)
}

func (r *WorkerIO) Stop(ctx context.Context) {
	r.cancel()
	r.sendq.Shutdown()
	r.recvq.Shutdown()

	r.ioServer.Close(func(err error) {
		if err != nil {
			log.G(ctx).WithError(err).Error("failed to close worker socket io")
		}
	})

	r.httpServer.Stop(ctx)

	r.wg.Wait()
}

func (r *WorkerIO) Events() (<-chan events.Event, io.Closer) {
	return r.recvq.Subscribe()
}

func (r *WorkerIO) Publish(_ context.Context, event events.Event) error {
	r.sendq.Publish(event)
	return nil
}

func (r *WorkerIO) startBusForwarder(ctx context.Context) {
	evch, errs := r.bus.Subscribe(
		ctx,
		events.WithEventTypes(
			events.EventTypeInitContainerFinish,
			events.EventTypeScriptStart,
			events.EventTypeScriptAbort,
			events.EventTypeCleanupContainer,
		),
	)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				log.G(ctx).Debug("worker io bus channel is closed")
				return
			}

			_ = r.Publish(ctx, event)

		case err := <-errs:
			if err != nil {
				log.G(ctx).WithError(err).Error("failed to listen on bus")
			}
		}
	}
}

func (r *WorkerIO) startReceiver(ctx context.Context) {
	evch, closer := r.Events()
	defer func() {
		_ = closer.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-evch:
			if !ok {
				log.G(ctx).Debug("worker io receiver channel is closed")
				return
			}

			if err := r.bus.Publish(ctx, event); err != nil {
				log.G(ctx).WithError(err).Error("failed to publish worker message to bus")
				go events.PublishAfter(ctx, r.bus, event, requeueTimeout)
			}
		}
	}
}

func (r *WorkerIO) startSender(ctx context.Context, worker *socket.Socket) {
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
				log.G(ctx).Debug("worker io sender channel is closed")
				return
			}

			if err := worker.Emit(workerEventName, events.Message{Event: event}); err != nil {
				log.G(ctx).WithError(err).Error("failed to send message to worker")
				go events.PublishAfter(ctx, r, event, requeueTimeout)
			}
		}
	}
}
