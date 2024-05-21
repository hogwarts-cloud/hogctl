package incus

import (
	"net"
	"testing"

	"github.com/lxc/incus/shared/api"
	"github.com/stretchr/testify/assert"
)

func Test_stopInstance(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
	})
}

func Test_getInstanceIPFromState(t *testing.T) {
	testCases := []struct {
		name     string
		state    *api.InstanceState
		nic      string
		expected net.IP
		wantErr  bool
		err      error
	}{
		{
			name: "happy path",
			state: &api.InstanceState{
				Network: map[string]api.InstanceStateNetwork{
					"veth0": {
						Addresses: []api.InstanceStateNetworkAddress{
							{
								Family:  AddressFamily,
								Address: "192.168.0.100",
							},
						},
					},
				},
			},
			nic:      "veth0",
			expected: net.ParseIP("192.168.0.100"),
			wantErr:  false,
		},
		{
			name: "no such address family",
			state: &api.InstanceState{
				Network: map[string]api.InstanceStateNetwork{
					"veth0": {
						Addresses: make([]api.InstanceStateNetworkAddress, 0),
					},
				},
			},
			nic:     "veth0",
			wantErr: true,
			err:     ErrNoSuchAddressFamily,
		},
	}

	for _, tc := range testCases {
		actual, err := getInstanceIPFromState(tc.state, tc.nic)
		if tc.wantErr {
			assert.ErrorIs(t, err, tc.err)
		} else {
			assert.ElementsMatch(t, tc.expected, actual)
		}
	}
}
