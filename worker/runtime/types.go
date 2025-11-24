package runtime

type TaskSpec struct {
	Path string
	Args []string
}

type ContainerSpec struct {
	Image string
	Dir   string
	Envs  []string
}
