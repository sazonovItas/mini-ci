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
	recvDiscardTImeout = 250 * time.Millisecond
	sendDiscardTimeout = 250 * time.Millisecond
)

type SocketIORunner struct {
	endpoint       string
	eventNamespace string

	manager *socket.Manager

	sendq eventq.Queue[events.Event]
	recvq eventq.Queue[events.Event]

	ctx    context.Context
	cancel context.CancelFunc
}

func NewSocketIORunner(address string, endpoint string, eventNamespace string) *SocketIORunner {
	opts := socket.DefaultOptions()
	opts.SetTimeout(60 * time.Second)
	opts.SetUpgrade(true)
	opts.SetAutoConnect(true)
	opts.SetReconnection(true)
	opts.SetReconnectionAttempts(5)
	opts.SetTransports(types.NewSet(transports.WebSocket))

	manager := socket.NewManager(address, opts)

	return &SocketIORunner{
		endpoint:       endpoint,
		eventNamespace: eventNamespace,
		manager:        manager,
		recvq:          eventq.New(recvDiscardTImeout, func(events.Event) {}),
		sendq:          eventq.New(sendDiscardTimeout, func(events.Event) {}),
	}
}

func (r *SocketIORunner) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	socket := r.manager.Socket(r.endpoint, nil)

	err := socket.On(types.EventName(r.eventNamespace), func(msgs ...any) {
		if len(msgs) == 0 {
			return
		}

		jsonMessage, err := json.Marshal(msgs[0])
		if err != nil {
			log.G(ctx).WithError(err).Error("failed to marshal recieved message")
			return
		}

		var message events.Message
		if err := json.Unmarshal(jsonMessage, &message); err != nil {
			log.G(ctx).WithError(err).Error("failed to unmarshal recieved message into event")
			return
		}

		r.recvq.Publish(message.Event)
	})
	if err != nil {
		return fmt.Errorf("failed register listener on %s", r.eventNamespace)
	}

	go r.startSender(r.ctx, socket)

	return nil
}

func (r *SocketIORunner) Stop(ctx context.Context) error {
	r.recvq.Shutdown()
	r.sendq.Shutdown()

	r.cancel()

	return nil
}

func (r *SocketIORunner) Send(event events.Event) {
	r.sendq.Publish(event)
}

func (r *SocketIORunner) Events() (<-chan events.Event, io.Closer) {
	return r.recvq.Subscribe()
}

func (r *SocketIORunner) startSender(ctx context.Context, socket *socket.Socket) {
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
			if err := socket.Emit(r.eventNamespace, msg); err != nil {
				log.G(ctx).WithError(err).Errorf("failed to send event %s", event.Type())
				go r.requeueSendEvent(ctx, event)
			}
		}
	}
}

func (r *SocketIORunner) requeueSendEvent(ctx context.Context, event events.Event) {
	const requeueTimeout = 1 * time.Second

	select {
	case <-ctx.Done():
	case <-time.After(requeueTimeout):
		r.sendq.Publish(event)
	}
}
