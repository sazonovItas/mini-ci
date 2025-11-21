package runtime

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/errdefs"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
	"github.com/sazonovItas/mini-ci/worker/runtime/logging"
	"github.com/sazonovItas/mini-ci/worker/runtime/network"
)

type Killer interface {
	Kill(ctx context.Context, task containerd.Task, gracefully bool) (err error)
}

type IOManager interface {
	Creator(containerID string, creator cio.Creator) (wrapCreator cio.Creator)
	Attach(containerID string, attach cio.Attach) (wrapAttach cio.Attach)
	Get(containerID string) (io cio.IO, exists bool)
	Delete(containerID string)
}

type DataStore interface {
	Set(content []byte, id string, keys ...string) error
	Get(id string, keys ...string) (content []byte, err error)
	Location(id string, keys ...string) (path string)
	Cleanup(id string) error
}

type Network interface {
	Setup(ctx context.Context, id string) (network.NetworkConfig, error)
	Load(ctx context.Context, id string) (network.NetworkConfig, error)
	Cleanup(ctx context.Context, id string) error
	Mounts(id string) []specs.Mount
}

const (
	defaultDataStorePath = "/var/lib/minici"
)

type Runtime struct {
	client *containerd.Client

	dataStore DataStore
	ioManager IOManager
	killer    Killer
	network   Network
	logger    logging.Logger

	dataStorePath string
}

type RuntimeOpt func(r *Runtime)

func NewRuntime(client *containerd.Client, opts ...RuntimeOpt) (r *Runtime, err error) {
	r = &Runtime{client: client}
	for _, opt := range opts {
		opt(r)
	}

	if r.client == nil {
		return nil, errdefs.ErrInvalidArgument.WithMessage("nil client")
	}

	if r.dataStorePath == "" {
		r.dataStorePath = defaultDataStorePath
	}

	if r.dataStore == nil {
		r.dataStore, err = NewDataStore(r.dataStorePath)
		if err != nil {
			return nil, fmt.Errorf("new data store: %w", err)
		}
	}

	if r.logger == nil {
		r.logger = logging.NewJSONLogger(r.dataStorePath)
	}

	if r.network == nil {
		r.network, err = network.NewNetwork(defaultDataStorePath)
		if err != nil {
			return nil, err
		}
	}

	if r.killer == nil {
		r.killer = NewKiller()
	}

	if r.ioManager == nil {
		r.ioManager = NewIOManager()
	}

	return r, nil
}

func (r *Runtime) Create(ctx context.Context, spec ContainerSpec) (*Container, error) {
	id := idgen.ID()

	netConfig, err := r.network.Setup(ctx, id)
	if err != nil {
		return nil, err
	}

	container, err := r.createContainer(ctx, id, spec, netConfig)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (r *Runtime) createContainer(
	ctx context.Context,
	id string,
	spec ContainerSpec,
	netConfig network.NetworkConfig,
) (*Container, error) {
	image, err := r.client.Pull(ctx, spec.Image)
	if err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}

	ociOpts := containerOciSpecOpts(image, spec, netConfig.NetNsPath, r.network.Mounts(id))

	ctr, err := r.client.NewContainer(
		ctx, id,
		containerd.WithNewSnapshot(id, image),
		containerd.WithImageConfigLabels(image),
		containerd.WithNewSpec(ociOpts...),
	)
	if err != nil {
		return nil, err
	}

	container := NewContainer(ctr, r.ioManager, r.killer)

	return container, nil
}

func (r *Runtime) Container(ctx context.Context, id string) (*Container, error) {
	if id == "" {
		return nil, ErrMissingContainerID
	}

	container, err := r.client.LoadContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	return NewContainer(container, r.ioManager, r.killer), nil
}

func (r *Runtime) Containers(ctx context.Context, filters ...string) ([]*Container, error) {
	ctrs, err := r.client.Containers(ctx, filters...)
	if err != nil {
		return nil, err
	}

	containers := make([]*Container, 0, len(ctrs))
	for _, ctr := range ctrs {
		containers = append(containers, NewContainer(ctr, r.ioManager, r.killer))
	}

	return containers, nil
}

func (r *Runtime) Destroy(ctx context.Context, id string) (err error) {
	r.ioManager.Delete(id)

	container, err := r.client.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("get container: %w", err)
	}

	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("lookup task: %w", err)
		}

		if err = container.Delete(ctx); err != nil {
			return fmt.Errorf("delete container: %w", err)
		}

		if err = r.cleanup(ctx, id); err != nil {
			return err
		}

		return nil
	}

	if err = r.killer.Kill(ctx, task, true); err != nil {
		return fmt.Errorf("kill task gracefully: %w", err)
	}

	_, err = task.Delete(ctx, containerd.WithProcessKill)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if err = container.Delete(ctx); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	if err = r.cleanup(ctx, id); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) cleanup(ctx context.Context, id string) error {
	if err := r.network.Cleanup(ctx, id); err != nil {
		return err
	}

	if err := r.dataStore.Cleanup(id); err != nil {
		return err
	}

	return nil
}
