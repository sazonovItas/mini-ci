package runtime

type ContainerSpec struct {
	ID    string
	Image string
}

type TaskSpec struct {
	Path string
	Args []string
	Envs []string
	Dir  string
	User *UserSpec
}

type UserSpec struct {
	UID int
	GID int
}
