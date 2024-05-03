package incus

import (
	"context"
	"fmt"
	"os"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/models"
	client "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	"github.com/samber/lo"
)

var (
	UserData = `#cloud-config
users:
- name: %s
  sudo: ALL=(ALL) NOPASSWD:ALL
  shell: /bin/bash
  ssh_authorized_keys:
  - %s`
	NetworkConfig = `version: 2
ethernets:
  eth0:
    addresses:
    - %s/24
    gateway4: 10.10.10.1
    nameservers:
      addresses:
      - 10.10.10.1`
)

type Client struct {
	client client.InstanceServer
}

func (c *Client) GetInstanceNames(ctx context.Context) ([]string, error) {
	instances, err := c.client.GetInstances(api.InstanceTypeContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	return lo.Map(instances, func(instance api.Instance, _ int) string {
		return instance.Name
	}), nil
}

func (c *Client) LaunchInstance(ctx context.Context, instance *models.Instance) error {
	op, err := c.client.CreateInstance(api.InstancesPost{
		InstancePut: api.InstancePut{
			Config: map[string]string{
				"limits.cpu": instance.Resources.Flavor.CPU(),
				// "limits.memory":             instance.Flavor.Memory(), // wtf
				"cloud-init.user-data":      fmt.Sprintf(UserData, instance.User.Name, instance.User.PublicKey),
				"user.email":                instance.User.Email,
				"cloud-init.network-config": fmt.Sprintf(NetworkConfig, "10.10.10.100"),
			},
			Devices: map[string]map[string]string{
				"eth0": {"type": "nic", "name": "eth0", "network": "incusbr0"},
				"root": {"type": "disk", "path": "/", "pool": "incuslvm", "size": fmt.Sprintf("%dGB", instance.Resources.Disk)},
			},
		},
		Name:   instance.Name,
		Source: api.InstanceSource{Type: "image", Alias: "hogwarts-cloud-image"},
		Type:   api.InstanceTypeContainer,
		Start:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait create instance operation: %w", err)
	}

	return nil
}

func (c *Client) DeleteInstance(ctx context.Context, instanceName string) error {
	op, err := c.client.DeleteInstance(instanceName)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait delete instance operation: %w", err)
	}

	return nil
}

func New() (*Client, error) {
	clientCert, err := os.ReadFile("client.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read client certificate: %w", err)
	}

	clientKey, err := os.ReadFile("client.key")
	if err != nil {
		return nil, fmt.Errorf("failed to read client key: %w", err)
	}

	ip := "62.84.113.137"

	client, err := client.ConnectIncus(fmt.Sprintf("https://%s:8443", ip), &client.ConnectionArgs{
		TLSClientCert:      string(clientCert),
		TLSClientKey:       string(clientKey),
		InsecureSkipVerify: true,
	})
	// client, err := client.ConnectIncusUnix("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Incus: %w", err)
	}

	return &Client{client: client}, nil
}
