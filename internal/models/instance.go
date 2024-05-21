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

type LaunchConfig struct {
	Instance
	IP net.IP
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

type LaunchedInstanceInfo struct {
	Name string
	IP   net.IP
	Port int
}
