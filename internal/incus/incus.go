package incus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/hogwarts-cloud/hogctl/internal/network"
	"github.com/hogwarts-cloud/hogctl/pkg/constants"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

const (
	AddressFamily          = "inet"
	EmailKey               = "user.email"
	CloudInitNetworkConfig = "cloud-init.network-config"
	CloudInitUserData      = "cloud-init.user-data"
)

var (
	ErrNoSuchAddressFamily = errors.New("no such address family")
)

type Incus struct {
	server    incus.InstanceServer
	cluster   models.Cluster
	templates *template.Template
}

func (i *Incus) GetLaunchedInstances(ctx context.Context) ([]models.InstanceInfo, error) {
	incusInstances, err := i.server.GetInstancesFull(api.InstanceTypeContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances full: %w", err)
	}

	launchedInstances := make([]models.InstanceInfo, 0, len(incusInstances))
	for _, instance := range incusInstances {
		ip, err := getInstanceIPFromState(instance.State, i.cluster.Network.NIC)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance ip from state: %w", err)
		}

		info := models.InstanceInfo{
			Name:  instance.Name,
			Email: instance.Config[EmailKey],
			InstanceNetworkInfo: models.InstanceNetworkInfo{
				IP:            ip,
				ForwardedPort: network.GeneratePortByIP(ip),
			},
		}

		launchedInstances = append(launchedInstances, info)
	}

	return launchedInstances, nil
}

func (i *Incus) GetInstanceIP(ctx context.Context, instance string) (net.IP, error) {
	instanceState, _, err := i.server.GetInstanceState(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance state: %w", err)
	}

	ip, err := getInstanceIPFromState(instanceState, i.cluster.Network.NIC)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance ip from state: %w", err)
	}

	return ip, nil
}

func (i *Incus) LaunchInstance(ctx context.Context, instance models.LaunchConfig) error {
	networkConfigBuf := &strings.Builder{}
	mask, _ := i.cluster.Network.CIDR.Mask.Size()
	if err := i.templates.ExecuteTemplate(
		networkConfigBuf,
		CloudInitNetworkConfig+constants.TemplateExtension,
		map[string]any{"Network": i.cluster.Network, "IP": instance.IP.String(), "Mask": mask},
	); err != nil {
		return fmt.Errorf("failed to execute network config template: %w", err)
	}

	userDataBuf := &strings.Builder{}
	if err := i.templates.ExecuteTemplate(
		userDataBuf,
		CloudInitUserData+constants.TemplateExtension,
		instance.User,
	); err != nil {
		return fmt.Errorf("failed to execute user data template: %w", err)
	}

	var cpu, memory int
	for _, availableFlavor := range i.cluster.Flavors {
		if instance.Resources.Flavor == availableFlavor.Name {
			cpu = availableFlavor.Resources.CPU
			memory = availableFlavor.Resources.Memory
			break
		}
	}

	op, err := i.server.CreateInstance(api.InstancesPost{
		InstancePut: api.InstancePut{
			Config: map[string]string{
				EmailKey:               instance.User.Email,
				CloudInitNetworkConfig: networkConfigBuf.String(),
				CloudInitUserData:      userDataBuf.String(),
				"limits.cpu":           fmt.Sprintf("%d", cpu),
				"limits.memory":        fmt.Sprintf("%dGB", memory),
			},
			Devices: map[string]map[string]string{
				i.cluster.Network.NIC: {
					"type":    "nic",
					"nictype": "bridged",
					"name":    i.cluster.Network.NIC,
					"parent":  i.cluster.Network.Bridge,
				},
				"root": {
					"type": "disk",
					"path": "/",
					"pool": i.cluster.Storage.Pool,
					"size": fmt.Sprintf("%dGB", instance.Resources.Disk),
				},
			},
		},
		Name:   instance.Name,
		Source: api.InstanceSource{Type: "image", Alias: i.cluster.Image},
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

func (i *Incus) DeleteInstance(ctx context.Context, instance string) error {
	if err := i.stopInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	op, err := i.server.DeleteInstance(instance)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait delete instance operation: %w", err)
	}

	return nil
}

func (i *Incus) stopInstance(ctx context.Context, instance string) error {
	op, err := i.server.UpdateInstanceState(instance, api.InstanceStatePut{
		Action: "stop",
	}, "")
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait stop instance operation: %w", err)
	}

	return nil
}

func New(cluster models.Cluster, templates *template.Template) (*Incus, error) {
	server, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Incus: %w", err)
	}

	return &Incus{
		server:    server,
		cluster:   cluster,
		templates: templates,
	}, nil
}

func getInstanceIPFromState(state *api.InstanceState, nic string) (net.IP, error) {
	for _, address := range state.Network[nic].Addresses {
		if address.Family == AddressFamily {
			return net.ParseIP(address.Address), nil
		}
	}

	return nil, ErrNoSuchAddressFamily
}
