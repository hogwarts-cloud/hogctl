package network

import (
	"errors"
	"net"

	"github.com/hogwarts-cloud/hogctl/pkg/utils"
)

var ErrTooFewAvailableIPs = errors.New("too few available ips")

func GetAvailableIPs(count int, network net.IPNet, occupiedIPs []net.IP) ([]net.IP, error) {
	var ips []net.IP

loop:
	for _, ip := range utils.GetAllNetworkIPs(network) {
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

func GeneratePortByIP(ip net.IP) int {
	return 62000 + int(ip.To4()[3])
}
