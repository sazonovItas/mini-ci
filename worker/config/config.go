package config

type Config struct {
	Runtime RuntimeConfig
}

type RuntimeConfig struct {
	Address string
	CNI     CNIConfig
	Storage StorageConfig
}

type CNIConfig struct {
	PluginDir string
}

type StorageConfig struct {
	Path string
}
