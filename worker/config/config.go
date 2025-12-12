package config

import (
	_ "embed"
)

//go:embed default.yaml
var DefaultConfig []byte

type Config struct {
	Name     string         `yaml:"name" mapstructure:"name"`
	Runtime  RuntimeConfig  `yaml:"runtime" mapstructure:"runtime"`
	SocketIO SocketIOConfig `yaml:"socket_io" mapstructure:"socket_io"`
}

type RuntimeConfig struct {
	Address       string `yaml:"address" mapstructure:"address"`
	Snapshotter   string `yaml:"snapshotter" mapstructure:"snapshotter"`
	DataStorePath string `yaml:"data_store_path" mapstructure:"data_store_path"`
}

type SocketIOConfig struct {
	URL       string `yaml:"url" mapstructure:"url"`
	Namespace string `yaml:"namespace" mapstructure:"namespace"`
	EventName string `yaml:"event_name" mapstructure:"event_name"`
}
