package runtime

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
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
	Init(id string) error
	Mounts(id string) []specs.Mount
	Add(ctx context.Context, id string, task containerd.Task) error
	Remove(ctx context.Context, id string, task containerd.Task) error
	Cleanup(id string) error
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

	if r.network == nil {
		r.network, err = network.NewNetwork(network.WithContainerStore(r.dataStore))
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

func (r *Runtime) Create(
	ctx context.Context,
	spec ContainerSpec,
	taskSpec TaskSpec,
	taskIO TaskIO,
) (*Container, error) {
	spec.ID = idgen.ID()

	if err := r.network.Init(spec.ID); err != nil {
		return nil, err
	}

	container, err := r.createContainer(ctx, spec, taskSpec)
	if err != nil {
		return nil, err
	}

	task, err := container.newTask(ctx, taskIO)
	if err != nil {
		return nil, fmt.Errorf("creating init task: %w", err)
	}

	if err := r.network.Add(ctx, container.ID(), task); err != nil {
		return nil, fmt.Errorf("adding task to network: %w", err)
	}

	return container, nil
}

func (r *Runtime) createContainer(
	ctx context.Context,
	spec ContainerSpec,
	taskSpec TaskSpec,
) (*Container, error) {
	image, err := r.client.Pull(ctx, spec.Image)
	if err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}

	ociOpts := r.ociSpecOpts(spec.ID, image, taskSpec)

	ctr, err := r.client.NewContainer(
		ctx,
		spec.ID,
		containerd.WithNewSnapshot(spec.ID, image),
		containerd.WithImageConfigLabels(image),
		containerd.WithNewSpec(ociOpts...),
	)
	if err != nil {
		return nil, err
	}

	container := NewContainer(ctr, r.ioManager, r.killer)

	return container, nil
}

func (r *Runtime) ociSpecOpts(id string, image containerd.Image, taskSpec TaskSpec) []oci.SpecOpts {
	ociOpts := []oci.SpecOpts{
		oci.WithDefaultUnixDevices,
		oci.WithDefaultPathEnv,
		oci.WithImageConfig(image),
		oci.WithEnv(taskSpec.Envs),
		oci.WithMounts(r.network.Mounts(id)),
	}

	if taskSpec.Dir != "" {
		ociOpts = append(ociOpts, oci.WithProcessCwd(taskSpec.Dir))
	}

	if taskSpec.Path != "" {
		args := []string{taskSpec.Path}
		if len(taskSpec.Args) != 0 {
			args = append(args, taskSpec.Args...)
		}

		ociOpts = append(ociOpts, oci.WithProcessArgs(args...))
	}

	if taskSpec.User != nil {
		ociOpts = append(
			ociOpts,
			oci.WithUIDGID(
				uint32(taskSpec.User.UID),
				uint32(taskSpec.User.GID),
			),
		)
	}

	return ociOpts
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

		if err = r.cleanup(id); err != nil {
			return err
		}

		return nil
	}

	if err = r.killer.Kill(ctx, task, true); err != nil {
		return fmt.Errorf("kill task gracefully: %w", err)
	}

	if err = r.network.Remove(ctx, container.ID(), task); err != nil {
		return fmt.Errorf("removing network: %w", err)
	}

	_, err = task.Delete(ctx, containerd.WithProcessKill)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if err = container.Delete(ctx); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	if err = r.cleanup(id); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) cleanup(id string) error {
	if err := r.network.Cleanup(id); err != nil {
		return err
	}

	if err := r.dataStore.Cleanup(id); err != nil {
		return err
	}

	return nil
}
