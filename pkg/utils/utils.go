package utils

import (
	"net"
)

func GetAllNetworkIPs(network net.IPNet) []net.IP {
	var ips []net.IP
	ip := network.IP

	for ip := ip.Mask(network.Mask); network.Contains(ip); incIP(ip) {
		ips = append(ips, dupIP(ip))
	}

	return ips[1 : len(ips)-1]
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func dupIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}
