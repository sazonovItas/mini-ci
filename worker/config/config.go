package config

import (
	_ "embed"
)

//go:embed default.yaml
var DefaultConfig string

type Config struct {
	Runtime RuntimeConfig `yaml:"runtime" mapstructure:"runtime"`
}

type RuntimeConfig struct {
	Address       string `yaml:"address" mapstructure:"address"`
	Snapshotter   string `yaml:"snapshotter" mapstructure:"snapshotter"`
	DataStorePath string `yaml:"data_store_path" mapstructure:"data_store_path"`
}
