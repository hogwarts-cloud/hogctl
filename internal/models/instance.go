package models

import (
	"net"
	"time"
)

type Instance struct {
	Name           string
	Resources      InstanceResources
	User           User
	ExpirationDate time.Time
}

func (i Instance) IsExpired() bool {
	return time.Now().After(i.ExpirationDate)
}

type InstanceResources struct {
	Flavor string
	Disk   int
}

type User struct {
	Name      string
	Email     string
	PublicKey string
}

type InstanceNetworkInfo struct {
	IP            net.IP `yaml:"ip"`
	ForwardedPort int    `yaml:"port"`
}

type LaunchConfig struct {
	Instance
	InstanceNetworkInfo
}

type InstanceInfo struct {
	Name                string `yaml:"name"`
	Email               string `yaml:"email"`
	InstanceNetworkInfo `yaml:"network"`
}

type ApplyResult struct {
	Launched []InstanceInfo `yaml:"launched"`
	Deleted  []InstanceInfo `yaml:"deleted"`
}
