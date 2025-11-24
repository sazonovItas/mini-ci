package runtime

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sazonovItas/mini-ci/worker/runtime/logging"
)

type Container struct {
	killer    Killer
	ioManager IOManager
	container containerd.Container
}

func NewContainer(
	container containerd.Container,
	ioManager IOManager,
	killer Killer,
) *Container {
	return &Container{
		killer:    killer,
		ioManager: ioManager,
		container: container,
	}
}

func (c *Container) ID() string {
	return c.container.ID()
}

func (c *Container) Container() containerd.Container {
	return c.container
}

func (c *Container) NewTask(ctx context.Context, spec TaskSpec) error {
	ociSpec, err := c.container.Spec(ctx)
	if err != nil {
		return fmt.Errorf("container spec: %w", err)
	}

	err = c.container.Update(
		ctx,
		containerd.UpdateContainerOpts(
			containerd.WithSpec(ociSpec, taskOCISpecOpts(spec)...),
		),
	)
	if err != nil {
		return fmt.Errorf("update oci spec: %w", err)
	}

	taskIO := c.ioManager.TaskIO(c.ID())
	ioCreator := c.ioManager.Creator(c.ID(), cio.NewCreator(containerCIO(taskIO, false)...))

	task, err := c.container.NewTask(ctx, ioCreator)
	if err != nil {
		return fmt.Errorf("create new task: %w", err)
	}

	if err := task.CloseIO(ctx, containerd.WithStdinCloser); err != nil {
		return fmt.Errorf("close stdin stream: %w", err)
	}

	return nil
}

func (c *Container) Start(ctx context.Context, loggers ...logging.Logger) (*Task, error) {
	task, err := c.container.Task(ctx, cio.Load)
	if err != nil {
		return nil, fmt.Errorf("retrieve task: %w", err)
	}

	taskIO := c.ioManager.TaskIO(c.ID())
	logIO := logging.LogIO{Stdout: taskIO.StdoutR, Stderr: taskIO.StderrR}

	go func() {
		if err := logging.ProcessLogs(ctx, c.Container(), logIO, loggers...); err != nil {
			log.G(ctx).WithError(err).Error("failed to process task logs")
		}
	}()

	if err := task.Start(ctx); err != nil {
		return nil, fmt.Errorf("task start: %w", err)
	}

	taskWaitStatus, err := task.Wait(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("task wait: %w", err)
	}

	return NewTask(task, taskWaitStatus), nil
}

func (c *Container) Stop(ctx context.Context, kill bool) error {
	task, err := c.container.Task(ctx, cio.Load)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("retrieve task: %w", err)
	}

	if err := c.killer.Kill(ctx, task, !kill); err != nil {
		return fmt.Errorf("kill task: %w", err)
	}

	return nil
}

func containerCIO(taskIO TaskIO, tty bool) []cio.Opt {
	if !tty {
		return []cio.Opt{
			cio.WithStreams(
				nil,
				taskIO.StdoutW,
				taskIO.StderrW,
			),
		}
	}

	opts := []cio.Opt{
		cio.WithStreams(
			nil,
			taskIO.StdoutW,
			taskIO.StderrW,
		),
		cio.WithTerminal,
	}

	return opts
}

func containerOCISpecOpts(
	image containerd.Image,
	spec ContainerSpec,
	netNsPath string,
	mounts []specs.Mount,
) []oci.SpecOpts {
	opts := []oci.SpecOpts{
		oci.WithDefaultPathEnv,
		oci.WithDefaultUnixDevices,
		oci.WithImageConfig(image),
		oci.WithEnv(spec.Envs),
		oci.WithMounts(mounts),
		oci.WithLinuxNamespace(
			specs.LinuxNamespace{
				Type: specs.NetworkNamespace,
				Path: netNsPath,
			},
		),
	}

	if spec.Dir != "" {
		opts = append(opts, oci.WithProcessCwd(spec.Dir))
	}

	return opts
}

func taskOCISpecOpts(spec TaskSpec) []oci.SpecOpts {
	var opts []oci.SpecOpts
	if spec.Path != "" {
		args := []string{spec.Path}
		if len(spec.Args) != 0 {
			args = append(args, spec.Args...)
		}

		opts = append(opts, oci.WithProcessArgs(args...))
	}

	return opts
}
