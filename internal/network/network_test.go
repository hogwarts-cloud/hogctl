package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetAvailableIPs(t *testing.T) {
	testCases := []struct {
		name        string
		count       int
		network     net.IPNet
		occupiedIPs []net.IP
		expected    []net.IP
		wantErr     bool
		err         error
	}{
		{
			name:    "happy path",
			count:   3,
			network: net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(29, 32)},
			occupiedIPs: []net.IP{
				net.ParseIP("192.168.1.1").To4(),
				net.ParseIP("192.168.1.2").To4(),
			},
			expected: []net.IP{
				net.ParseIP("192.168.1.3").To4(),
				net.ParseIP("192.168.1.4").To4(),
				net.ParseIP("192.168.1.5").To4(),
			},
			wantErr: false,
		},
		{
			name:    "too few available ips",
			count:   5,
			network: net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(29, 32)},
			occupiedIPs: []net.IP{
				net.ParseIP("192.168.1.1").To4(),
				net.ParseIP("192.168.1.2").To4(),
				net.ParseIP("192.168.1.3").To4(),
				net.ParseIP("192.168.1.4").To4(),
			},
			wantErr: true,
			err:     ErrTooFewAvailableIPs,
		},
	}

	for _, tc := range testCases {
		actual, err := GetAvailableIPs(tc.count, tc.network, tc.occupiedIPs)
		if tc.wantErr {
			assert.ErrorIs(t, err, tc.err)
		} else {
			assert.ElementsMatch(t, tc.expected, actual)
		}
	}
}

func Test_GeneratePortByIP(t *testing.T) {
	testCases := []struct {
		ip       net.IP
		expected int
	}{
		{
			ip:       net.ParseIP("192.168.0.1"),
			expected: 62001,
		},
		{
			ip:       net.ParseIP("10.96.17.31"),
			expected: 62031,
		},
	}

	for _, tc := range testCases {
		actual := GeneratePortByIP(tc.ip)
		assert.Equal(t, tc.expected, actual)
	}
}
