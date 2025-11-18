package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/sazonovItas/mini-ci/worker/runtime"
)

func main() {
	ctx := context.Background()

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("test"))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = client.Close()
	}()

	r, err := runtime.NewRuntime(client)
	if err != nil {
		panic(err)
	}

	containerSpec := runtime.ContainerSpec{
		Image: "docker.io/library/alpine:3.22",
	}

	taskSpec := runtime.TaskSpec{
		Path: "sh",
		Args: []string{
			"-c",
			`
				set -xeu

				ping -c 2 archlinux.org

				ping -c 2 localhost

				apk update && apk add curl

				curl -Lf archlinux.org
			`,
		},
	}

	taskIO := runtime.TaskIO{
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}

	container, err := r.Create(ctx, containerSpec, taskSpec, taskIO)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := r.Destroy(context.Background(), container.ID()); err != nil {
			fmt.Println(err.Error())
		}
	}()

	task, err := container.Start(ctx)
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

	go func() {
		exitStatus, err := task.WaitExitStatus(ctx)
		if err != nil {
			fmt.Printf("proc finished error: %s\n", err.Error())
		} else {
			fmt.Printf("proc finished with status %d\n", exitStatus)
		}
	}()

	<-ctx.Done()
}
