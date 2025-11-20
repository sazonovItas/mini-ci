package runtime

import "io"

type TaskIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type ContainerSpec struct {
	Envs  []string
	Image string
	Dir   string
}

type TaskSpec struct {
	Path string
	Args []string
}
