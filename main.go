package main

import (
	"context"
	"fmt"
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
	r, err := runtime.NewRuntime(client)
	if err != nil {
		panic(err)
	}

	jsonLogger := logging.NewJSONLogger("/var/lib/minici")

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

				apk update && apk add curl

				curl -L archlinux.org

				ip addr
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

	task, err := container.Start(ctx, jsonLogger)
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

	taskSpec = runtime.TaskSpec{
		Path: "sh",
		Args: []string{
			"-c",
			`
				set -xeu

				echo "Hello, second task"

				ip addr
			`,
		},
	}

	if err := container.NewTask(ctx, taskSpec); err != nil {
		panic(err)
	}

	task, err = container.Start(ctx, jsonLogger)
	if err != nil {
		panic(err)
	}

	exitStatus, err = task.WaitExitStatus(ctx)
	if err != nil {
		fmt.Printf("proc finished error: %s\n", err.Error())
	} else {
		fmt.Printf("proc finished with status %d\n", exitStatus)
	}

	<-ctx.Done()
}
