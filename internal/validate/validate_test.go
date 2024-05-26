package validate

import (
	"testing"
	"time"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		instances []models.Instance
		wantErr   bool
		err       error
	}{
		{
			name: "happy path",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin",
						Email:     "admin@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMt4RmHplan7NCJJtZEque5vBjvgeAYMncR45lJKG/mL admin@fedora",
					},
					ExpirationDate: time.Now().Add(time.Hour),
				},
				{
					Name: "u2",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin2",
						Email:     "admin2@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEn0DLc0I+Lmmgjey59zn4AJfrf/o0BCoEMXKK8yOc2v admin2@fedora",
					},
					ExpirationDate: time.Now().Add(2 * time.Hour),
				},
				{
					Name: "u3",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin3",
						Email:     "admin3@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJYEv69d14nkNq0ooosOFJVbWbqiKlrVWjub78RDHQ+k admin3@fedora",
					},
					ExpirationDate: time.Now().Add(3 * time.Hour),
				},
			},
			wantErr: false,
		},
		{
			name:      "empty instances list",
			instances: make([]models.Instance, 0),
			wantErr:   true,
			err:       ErrEmptyInstancesList,
		},
		{
			name: "empty instance name",
			instances: []models.Instance{
				{
					Name: "",
				},
			},
			wantErr: true,
			err:     ErrEmptyInstanceName,
		},
		{
			name: "invalid instance name",
			instances: []models.Instance{
				{
					Name: "aba&caba",
				},
			},
			wantErr: true,
			err:     ErrInvalidInstanceName,
		},
		{
			name: "invalid instance name",
			instances: []models.Instance{
				{
					Name: "aba@caba",
				},
			},
			wantErr: true,
			err:     ErrInvalidInstanceName,
		},
		{
			name: "invalid instance name 2",
			instances: []models.Instance{
				{
					Name: "aba.caba",
				},
			},
			wantErr: true,
			err:     ErrInvalidInstanceName,
		},
		{
			name: "instance name too big",
			instances: []models.Instance{
				{
					Name: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
				},
			},
			wantErr: true,
			err:     ErrInstanceNameTooBig,
		},
		{
			name: "invalid flavor",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "medium",
					},
				},
			},
			wantErr: true,
			err:     ErrInvalidFlavor,
		},
		{
			name: "non positive disk size",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   -1,
					},
				},
			},
			wantErr: true,
			err:     ErrNonPositiveDiskSize,
		},
		{
			name: "empty user name",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name: "",
					},
				},
			},
			wantErr: true,
			err:     ErrEmptyUserName,
		},
		{
			name: "user name too big",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
					},
				},
			},
			wantErr: true,
			err:     ErrUserNameTooBig,
		},
		{
			name: "invalid email",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:  "admin",
						Email: "abacaba",
					},
				},
			},
			wantErr: true,
			err:     ErrInvalidEmail,
		},
		{
			name: "invalid public key",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin",
						Email:     "admin@admin.com",
						PublicKey: "abacaba",
					},
				},
			},
			wantErr: true,
			err:     ErrInvalidPublicKey,
		},
		{
			name: "expired instance",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin",
						Email:     "admin@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMt4RmHplan7NCJJtZEque5vBjvgeAYMncR45lJKG/mL admin@fedora",
					},
					ExpirationDate: time.Now().Add(-time.Hour),
				},
			},
			wantErr: true,
			err:     ErrExpiredInstance,
		},
		{
			name: "found duplicated instance names",
			instances: []models.Instance{
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin",
						Email:     "admin@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMt4RmHplan7NCJJtZEque5vBjvgeAYMncR45lJKG/mL admin@fedora",
					},
					ExpirationDate: time.Now().Add(time.Hour),
				},
				{
					Name: "u1",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin2",
						Email:     "admin2@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEn0DLc0I+Lmmgjey59zn4AJfrf/o0BCoEMXKK8yOc2v admin2@fedora",
					},
					ExpirationDate: time.Now().Add(2 * time.Hour),
				},
				{
					Name: "u3",
					Resources: models.InstanceResources{
						Flavor: "micro",
						Disk:   20,
					},
					User: models.User{
						Name:      "admin3",
						Email:     "admin3@admin.com",
						PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJYEv69d14nkNq0ooosOFJVbWbqiKlrVWjub78RDHQ+k admin3@fedora",
					},
					ExpirationDate: time.Now().Add(3 * time.Hour),
				},
			},
			wantErr: true,
			err:     ErrFoundDuplicatedInstanceNames,
		},
	}

	flavors := []models.Flavor{
		{
			Name: "micro",
			Resources: models.FlavorResources{
				CPU:    1,
				Memory: 1,
			},
		},
	}

	domain := "vm.urgu.org"

	validator := NewCmd(flavors, domain)

	for _, tc := range testCases {
		err := validator.Run(tc.instances)
		if tc.wantErr {
			assert.ErrorIs(t, err, tc.err)
		} else {
			assert.Nil(t, err)
		}
	}
}
