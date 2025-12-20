package config

import (
	_ "embed"
)

//go:embed default.yaml
var DefaultConfig []byte

type Config struct {
	API      APIConfig      `yaml:"api" mapstructure:"api"`
	WorkerIO WorkerIOConfig `yaml:"worker_io" mapstructure:"worker_io"`
	Postgres PostgresConfig `yaml:"postgres" mapstructure:"postgres"`
}

type APIConfig struct {
	Address          string `yaml:"address" mapstructure:"address"`
	SocketIOEndpoint string `yaml:"socket_io_endpoint" mapstructure:"socket_io_endpoint"`
}

type WorkerIOConfig struct {
	Addresss string `yaml:"address" mapstructure:"address"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
}

type PostgresConfig struct {
	URI string `yaml:"uri" mapstructure:"uri"`
}
