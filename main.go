package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"
	"github.com/sazonovItas/mini-ci/worker/runtime"
	"github.com/sazonovItas/mini-ci/worker/runtime/logging"
)

func init() {
	if err := log.SetFormat(log.TextFormat); err != nil {
		panic(err)
	}

	if err := log.SetLevel(log.DebugLevel.String()); err != nil {
		panic(err)
	}
}

func main() {
	ctx := context.Background()

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("test"))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = client.Close()
	}()
	r, err := runtime.New(client, runtime.WithSnapshotter("fuse-overlayfs"))
	if err != nil {
		panic(err)
	}

	jsonLogger := logging.NewJSONLogger("/var/lib/minici")

	containerSpec := runtime.ContainerConfig{
		Image: "docker.io/library/centos:centos7",
		Mounts: []runtime.MountConfig{
			{
				Src: "/var/lib",
				Dst: "/test",
			},
		},
	}

	taskSpec := runtime.TaskConfig{
		Command: []string{"sh", "-c"},
		Args: []string{
			`
				set -xeu

				ls -la /test

				cat /etc/resolv.conf

				dnf update && dnf install curl

				curl -L archlinux.org
			`,
		},
	}

	container, err := r.Create(ctx, containerSpec)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := r.Destroy(context.Background(), container.ID()); err != nil {
			fmt.Println(err.Error())
		}
	}()

	if err := container.NewTask(ctx, taskSpec); err != nil {
		panic(err)
	}

	task, err := container.Start(ctx, jsonLogger, stdoutLogger{})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := container.Stop(context.Background(), true); err != nil {
			fmt.Println(err.Error())
		}
	}()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	exitStatus, err := task.WaitExitStatus(ctx)
	if err != nil {
		fmt.Printf("proc finished error: %s\n", err.Error())
	} else {
		fmt.Printf("proc finished with status %d\n", exitStatus)
	}

	<-ctx.Done()
}

type stdoutLogger struct{}

func (l stdoutLogger) Process(id string, stdout <-chan string, stderr <-chan string) error {
	var (
		isErrClosed = false
		isOutClosed = false
	)

	w := bufio.NewWriter(os.Stdout)

	for !isOutClosed || !isErrClosed {
		select {
		case log, ok := <-stdout:
			if !ok {
				isOutClosed = true
				break
			}

			_, _ = w.WriteString(log)
			_ = w.Flush()
		case log, ok := <-stderr:
			if !ok {
				isErrClosed = true
				break
			}

			_, _ = w.WriteString(log)
			_ = w.Flush()
		}
	}

	return nil
}
