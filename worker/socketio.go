package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/sazonovItas/mini-ci/core/events/eventq"
	"github.com/zishang520/socket.io/clients/engine/v3/transports"
	"github.com/zishang520/socket.io/clients/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

const (
	queueDiscardTimeout = 250 * time.Millisecond
)

type SocketIORunner struct {
	name      string
	namespace string
	eventName string

	manager *socket.Manager

	sendq eventq.Queue[events.Event]
	recvq eventq.Queue[events.Event]

	ctx    context.Context
	cancel context.CancelFunc
}

func NewSocketIORunner(name, address, namespace, eventName string) *SocketIORunner {
	opts := socket.DefaultOptions()
	opts.SetUpgrade(true)
	opts.SetAutoConnect(true)
	opts.SetReconnection(true)
	opts.SetReconnectionAttempts(10)
	opts.SetTransports(types.NewSet(transports.WebSocket, transports.Polling))

	manager := socket.NewManager(address, opts)

	return &SocketIORunner{
		name:      name,
		namespace: namespace,
		eventName: eventName,
		manager:   manager,
		recvq:     eventq.New(queueDiscardTimeout, func(events.Event) {}),
		sendq:     eventq.New(queueDiscardTimeout, func(events.Event) {}),
	}
}

func (r *SocketIORunner) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	socket := r.manager.Socket(r.namespace, nil)

	if err := r.registerEvents(socket); err != nil {
		return err
	}

	go r.startSender(r.ctx, socket)

	return nil
}

func (r *SocketIORunner) Stop(ctx context.Context) error {
	r.cancel()

	r.recvq.Shutdown()
	r.sendq.Shutdown()

	return nil
}

func (r *SocketIORunner) Publish(_ context.Context, event events.Event) error {
	r.sendq.Publish(event)
	return nil
}

func (r *SocketIORunner) Events() (<-chan events.Event, io.Closer) {
	return r.recvq.Subscribe()
}

func (r *SocketIORunner) registerEvents(socket *socket.Socket) (err error) {
	err = socket.On(types.EventName(r.eventName), func(msgs ...any) {
		if len(msgs) == 0 {
			return
		}

		jsonMessage, err := json.Marshal(msgs[0])
		if err != nil {
			log.G(r.ctx).WithError(err).Error("failed to marshal recieved message")
			return
		}

		var message events.Message
		if err := json.Unmarshal(jsonMessage, &message); err != nil {
			log.G(r.ctx).WithError(err).Error("failed to unmarshal recieved message into event")
			return
		}

		r.recvq.Publish(message.Event)
	})
	if err != nil {
		return fmt.Errorf("failed register event listener on %s: %w", r.eventName, err)
	}

	return nil
}

func (r *SocketIORunner) startSender(ctx context.Context, socket *socket.Socket) {
	const requeueEventTimeout = 1 * time.Second

	defer socket.Disconnect()

	evch, closer := r.sendq.Subscribe()
	defer func() {
		_ = closer.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-evch:
			msg := events.Message{Event: event}
			if err := socket.Emit(r.eventName, msg); err != nil {
				log.G(ctx).WithError(err).Errorf("failed to send event %s", event.Type())
				go r.requeueEventAfter(ctx, event, requeueEventTimeout)
			}
		}
	}
}

func (r *SocketIORunner) requeueEventAfter(
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
