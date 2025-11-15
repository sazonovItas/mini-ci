package runtime

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/defaults"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/sazonovItas/mini-ci/worker/runtime/idgen"
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

type RootFSManager interface {
	PullImage(ctx context.Context, imageRef string) (image containerd.Image, err error)
}

type Network interface {
	Add(ctx context.Context, id string, task containerd.Task) error
	Remove(ctx context.Context, id string, task containerd.Task) error
	Cleanup(ctx context.Context, id string) error
}

type Runtime struct {
	client        *containerd.Client
	network       Network
	killer        Killer
	ioManager     IOManager
	rootfsManager RootFSManager
}

type RuntimeOpt func(r *Runtime)

func NewRuntime(client *containerd.Client, opts ...RuntimeOpt) (*Runtime, error) {
	r := &Runtime{
		client: client,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.client == nil {
		return nil, errdefs.ErrInvalidArgument.WithMessage("nil client")
	}

	if r.network == nil {
		network, err := NewCNINetwork(
			WithCNINetworkConfig(defaultCNINetworkConfig),
		)
		if err != nil {
			return nil, err
		}

		r.network = network
	}

	if r.killer == nil {
		r.killer = NewKiller()
	}

	if r.ioManager == nil {
		r.ioManager = NewIOManager()
	}

	if r.rootfsManager == nil {
		r.rootfsManager = NewRootFSManager(
			r.client,
			r.client.ImageService(),
			r.client.SnapshotService(defaults.DefaultSnapshotter),
		)
	}

	return r, nil
}

func (r *Runtime) Create(
	ctx context.Context,
	spec ContainerSpec,
	taskSpec TaskSpec,
	taskIO TaskIO,
) (*Container, error) {
	if spec.ID == "" {
		spec.ID = idgen.ID()
	}

	container, err := r.createContainer(ctx, spec, taskSpec, taskIO)
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
	taskIO TaskIO,
) (*Container, error) {
	image, err := r.rootfsManager.PullImage(ctx, spec.Image)
	if err != nil {
		return nil, fmt.Errorf("getting or pulling image: %w", err)
	}

	ociOpts := r.ociSpecOpts(image, taskSpec)

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

func (r *Runtime) ociSpecOpts(image containerd.Image, taskSpec TaskSpec) []oci.SpecOpts {
	ociOpts := []oci.SpecOpts{
		oci.WithDefaultUnixDevices,
		oci.WithDefaultPathEnv,
		oci.WithImageConfig(image),
		oci.WithEnv(taskSpec.Envs),
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

func (r *Runtime) Destroy(ctx context.Context, id string) error {
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

		if err := container.Delete(ctx); err != nil {
			return fmt.Errorf("delete container: %w", err)
		}

		return nil
	}

	if err := r.killer.Kill(ctx, task, true); err != nil {
		return fmt.Errorf("kill task gracefully: %w", err)
	}

	if err := r.network.Remove(ctx, container.ID(), task); err != nil {
		return fmt.Errorf("removing network: %w", err)
	}

	if err := r.network.Cleanup(ctx, container.ID()); err != nil {
		return fmt.Errorf("cleaning up network: %w", err)
	}

	_, err = task.Delete(ctx, containerd.WithProcessKill)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if err := container.Delete(ctx); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	return nil
}
