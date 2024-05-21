package config

import (
	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Config struct {
	Cluster   models.Cluster
	Instances []models.Instance
}

func Load(path string) Config {
	viper.AddConfigPath(path)
	viper.SetConfigType("yaml")

	for _, config := range []string{"cluster", "instances"} {
		viper.SetConfigName(config)
		if err := viper.MergeInConfig(); err != nil {
			panic(err)
		}
	}

	cfg := Config{}

	if err := viper.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeHookFunc("02-01-2006"),
			mapstructure.StringToIPHookFunc(),
			mapstructure.StringToIPNetHookFunc(),
		))); err != nil {
		panic(err)
	}

	return cfg
}
