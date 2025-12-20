package runner

import (
	"context"
	"net/http"
	"sync"

	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/core/events"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

type EventsGateway struct {
	subscriber events.Subscriber
	ioServer   *socket.Server

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewEventGateway(subscriber events.Subscriber) *EventsGateway {
	opts := socket.DefaultServerOptions()
	opts.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	ioServer := socket.NewServer(nil, opts)

	gateway := &EventsGateway{
		subscriber: subscriber,
		ioServer:   ioServer,
	}

	_ = ioServer.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		log.G(context.TODO()).Debugf("User connected: %s", client.Id())
	})

	return gateway
}

func (u *EventsGateway) Handler() http.Handler {
	return u.ioServer.ServeHandler(nil)
}

func (u *EventsGateway) Start(ctx context.Context) {
	u.ctx, u.cancel = context.WithCancel(ctx)
	u.wg.Go(func() { u.streamEvents(u.ctx) })
}

func (u *EventsGateway) Stop(ctx context.Context) {
	u.cancel()
	u.ioServer.Close(nil)
	u.wg.Wait()
}

func (u *EventsGateway) streamEvents(ctx context.Context) {
	evch, errs := u.subscriber.Subscribe(ctx, events.WithEventTypes(
		events.EventTypeBuildStatus,
		events.EventTypeJobStatus,
		events.EventTypeTaskStatus,
		events.EventTypeTaskLog,
	))

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errs:
			if err != nil {
				log.G(ctx).WithError(err).Error("UserGateway bus error")
			}
		case event, ok := <-evch:
			if !ok {
				return
			}
			u.broadcast(event)
		}
	}
}

func (u *EventsGateway) broadcast(event events.Event) {
	msg := events.Message{Event: event}

	switch e := event.(type) {
	case events.BuildStatus:
		u.ioServer.Emit("workflow:status", msg)
		u.ioServer.Emit("workflow:"+e.WorkflowID+":build:status", msg)

	case events.JobStatus:
		u.ioServer.Emit("build:"+e.BuildID+":job:status", msg)

	case events.TaskStatus:
		u.ioServer.Emit("job:"+e.JobID+":task:status", msg)

	case events.TaskLog:
		u.ioServer.Emit("task:"+e.Origin().ID+":logs", msg)
	}
}
