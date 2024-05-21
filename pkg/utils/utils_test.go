package utils

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetAllNetworkIPs(t *testing.T) {
	testCases := []struct {
		network  net.IPNet
		expected []net.IP
	}{
		{
			network: net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(29, 32)},
			expected: []net.IP{
				net.ParseIP("192.168.1.1").To4(),
				net.ParseIP("192.168.1.2").To4(),
				net.ParseIP("192.168.1.3").To4(),
				net.ParseIP("192.168.1.4").To4(),
				net.ParseIP("192.168.1.5").To4(),
				net.ParseIP("192.168.1.6").To4(),
			},
		},
	}

	for _, tc := range testCases {
		actual := GetAllNetworkIPs(tc.network)
		assert.ElementsMatch(t, tc.expected, actual)
	}
}

func Test_incIP(t *testing.T) {
	testCases := []struct {
		ip       net.IP
		expected net.IP
	}{
		{
			ip:       net.ParseIP("192.168.1.0"),
			expected: net.ParseIP("192.168.1.1"),
		},
		{
			ip:       net.ParseIP("10.0.0.255"),
			expected: net.ParseIP("10.0.1.0"),
		},
	}

	for _, tc := range testCases {
		incIP(tc.ip)
		assert.Equal(t, tc.expected, tc.ip)
	}
}
