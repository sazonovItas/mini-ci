package configutil

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/iamolegga/enviper"
	"github.com/spf13/viper"
)

type (
	Option interface {
		Apply(cfg *enviper.Enviper)
	}

	optionFunc func(cfg *enviper.Enviper)
)

func (of optionFunc) Apply(cfg *enviper.Enviper) {
	of(cfg)
}

func WithConfigType(t string) optionFunc {
	return func(cfg *enviper.Enviper) {
		cfg.SetConfigType(t)
	}
}

func WithEnvs(envPrefix string) optionFunc {
	return func(cfg *enviper.Enviper) {
		cfg.SetEnvPrefix(envPrefix)
		cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		cfg.AutomaticEnv()
	}
}

func Load(cfg any, defaultConfig []byte, options ...Option) error {
	viperCfg := enviper.New(viper.New())
	for _, option := range options {
		option.Apply(viperCfg)
	}

	if defaultConfig != nil {
		if err := viperCfg.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
			return err
		}
	}

	if err := viperCfg.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
