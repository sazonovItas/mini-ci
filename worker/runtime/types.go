package runtime

type TaskConfig struct {
	Command []string
	Args    []string
}

type ContainerConfig struct {
	Image  string
	Cwd    string
	Env    []string
	Mounts []MountConfig
}

type MountConfig struct {
	Src      string
	Dst      string
	Readonly bool
}
