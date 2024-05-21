package models

import "net"

type Cluster struct {
	Hosts   []Host
	Flavors []Flavor
	Storage Storage
	Image   string
	Domain  string
	Network Network
	Mail    Mail
}

type Host struct {
	Name      string
	Resources HostResources
}

type HostResources struct {
	CPU    int
	Memory int
	Disk   int
}

type Flavor struct {
	Name      string
	Resources FlavorResources
}

type FlavorResources struct {
	CPU    int
	Memory int
}

type Storage struct {
	Pool string
}

type Network struct {
	Bridge      string
	NIC         string
	CIDR        net.IPNet
	Gateway     net.IP
	Nameservers []net.IP
	OccupiedIPs []net.IP
}

type Mail struct {
	Server string
}

type ClusterInfo struct {
	Instances []InstanceInfo
	IPs       []net.IP
}

type InstanceInfo struct {
	Name  string
	Email string
}
