package runtime

type TaskSpec struct {
	Command []string
	Args    []string
}

type ContainerSpec struct {
	Image  string
	Cwd    string
	Env    []string
	Mounts []MountSpec
}

type MountSpec struct {
	Src      string
	Dst      string
	Readonly bool
}
