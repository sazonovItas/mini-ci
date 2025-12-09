package config

import (
	_ "embed"
)

//go:embed default.yaml
var DefaultConfig []byte

type Config struct {
	Runtime  RuntimeConfig  `yaml:"runtime" mapstructure:"runtime"`
	SocketIO SocketIOConfig `yaml:"socket_io" mapstructure:"socket_io"`
}

type RuntimeConfig struct {
	Address       string `yaml:"address" mapstructure:"address"`
	Snapshotter   string `yaml:"snapshotter" mapstructure:"snapshotter"`
	DataStorePath string `yaml:"data_store_path" mapstructure:"data_store_path"`
}

type SocketIOConfig struct {
	Address        string `yaml:"address" mapstructure:"address"`
	Endpoint       string `yaml:"endpoint" mapstructure:"endpoint"`
	EventNamespace string `yaml:"event_namespace" mapstructure:"event_namespace"`
}
