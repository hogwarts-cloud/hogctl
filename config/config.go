package config

import (
	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type ClusterConfig struct {
	Cluster models.Cluster
}

type InstancesConfig struct {
	Instances []models.Instance
}

func LoadCluster(path string) ClusterConfig {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	cluster := ClusterConfig{}

	if err := viper.Unmarshal(&cluster, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToIPHookFunc(),
			mapstructure.StringToIPNetHookFunc(),
		))); err != nil {
		panic(err)
	}

	return cluster
}

func LoadInstances(path string) InstancesConfig {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	instances := InstancesConfig{}

	if err := viper.Unmarshal(&instances, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeHookFunc("02-01-2006"),
		))); err != nil {
		panic(err)
	}

	return instances
}
