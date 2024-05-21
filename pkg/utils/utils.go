package utils

import (
	"errors"
	"net"
)

var ErrTooFewAvailableIPs = errors.New("too few available ips")

func GetAvailableIPs(count int, network net.IPNet, occupiedIPs []net.IP) ([]net.IP, error) {
	var ips []net.IP

loop:
	for _, ip := range getAllNetworkIPs(network) {
		for _, occupiedIP := range occupiedIPs {
			if occupiedIP.Equal(ip) {
				continue loop
			}
		}

		ips = append(ips, ip)

		if len(ips) == count {
			return ips, nil
		}
	}

	return nil, ErrTooFewAvailableIPs
}

func getAllNetworkIPs(network net.IPNet) []net.IP {
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
