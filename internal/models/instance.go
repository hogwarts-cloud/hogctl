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

type LaunchConfig struct {
	Instance
	InstanceNetworkInfo
}

type InstanceInfo struct {
	Name                string `yaml:"name"`
	Location            string `yaml:"-"`
	InstanceUserInfo    `yaml:"user"`
	InstanceNetworkInfo `yaml:"network"`
}

type InstanceUserInfo struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type InstanceNetworkInfo struct {
	IP            net.IP `yaml:"ip"`
	ForwardedPort int    `yaml:"port"`
}

type ApplyResult struct {
	Launched []InstanceInfo `yaml:"launched"`
	Deleted  []InstanceInfo `yaml:"deleted"`
}

type InstanceState int

const (
	RunningState InstanceState = iota
	StoppedState
)

func (s InstanceState) String() string {
	switch s {
	case RunningState:
		return "start"
	case StoppedState:
		return "stop"
	}
	return ""
}

type RecoveryInfo struct {
	Config  map[string]string            `yaml:"config"`
	Devices map[string]map[string]string `yaml:"devices"`
}
