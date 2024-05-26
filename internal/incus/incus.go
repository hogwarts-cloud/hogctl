package incus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
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
	UserNameKey            = "user.name"
	UserEmailKey           = "user.email"
	CloudInitNetworkConfig = "cloud-init.network-config"
	CloudInitUserData      = "cloud-init.user-data"
	InstanceArchive        = "instance.tar.gz"
)

var (
	ErrNoSuchAddressFamily = errors.New("no such address family")
)

//go:generate mockgen -source incus.go -destination mocks/incus_server_provider.go -package mocks
type IncusServerProvider interface {
	GetInstancesFull(instanceType api.InstanceType) ([]api.InstanceFull, error)
	GetInstanceState(instance string) (*api.InstanceState, string, error)
	UpdateInstanceState(instance string, state api.InstanceStatePut, ETag string) (incus.Operation, error)
	CreateInstance(instance api.InstancesPost) (incus.Operation, error)
	DeleteInstance(instance string) (incus.Operation, error)
	CreateInstanceSnapshot(instance string, snapshot api.InstanceSnapshotsPost) (incus.Operation, error)
	DeleteInstanceSnapshot(instance string, snapshot string) (incus.Operation, error)
	CreateImage(image api.ImagesPost, args *incus.ImageCreateArgs) (incus.Operation, error)
	DeleteImage(fingerprint string) (incus.Operation, error)
	GetImageFile(fingerprint string, req incus.ImageFileRequest) (*incus.ImageFileResponse, error)
	GetInstance(instance string) (*api.Instance, string, error)
	GetClusterMemberNames() ([]string, error)
}

type Config struct {
	Server    IncusServerProvider
	Cluster   models.Cluster
	Templates *template.Template
}

type Incus struct {
	server    IncusServerProvider
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
			Name:     instance.Name,
			Location: instance.Location,
			InstanceUserInfo: models.InstanceUserInfo{
				Name:  instance.Config[UserNameKey],
				Email: instance.Config[UserEmailKey],
			},
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

func (i *Incus) LaunchInstance(ctx context.Context, instance models.LaunchConfig, async bool) error {
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
				UserNameKey:            instance.User.Name,
				UserEmailKey:           instance.User.Email,
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

	if !async {
		if err := op.WaitContext(ctx); err != nil {
			return fmt.Errorf("failed to wait create instance operation: %w", err)
		}
	}

	return nil
}

func (i *Incus) DeleteInstance(ctx context.Context, instance string) error {
	if err := i.UpdateInstanceState(ctx, instance, models.StoppedState); err != nil {
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

func (i *Incus) UpdateInstanceState(ctx context.Context, instance string, state models.InstanceState) error {
	op, err := i.server.UpdateInstanceState(instance, api.InstanceStatePut{Action: state.String()}, "")
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait stop instance operation: %w", err)
	}

	return nil
}

func (i *Incus) CreateSnapshot(ctx context.Context, instance string) (string, error) {
	snapshot := fmt.Sprintf("%s-snapshot", instance)

	op, err := i.server.CreateInstanceSnapshot(
		instance,
		api.InstanceSnapshotsPost{Name: snapshot},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create instance snapshot: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return "", fmt.Errorf("failed to wait create instance snapshot operation: %w", err)
	}

	return snapshot, nil
}

func (i *Incus) DeleteSnapshot(ctx context.Context, instance, snapshot string) error {
	op, err := i.server.DeleteInstanceSnapshot(instance, snapshot)
	if err != nil {
		return fmt.Errorf("failed to delete instance snapshot: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait delete instance snapshot operation: %w", err)
	}

	return nil
}

func (i *Incus) CreateImageFromSnapshot(ctx context.Context, instance string, snapshot string) (string, error) {
	alias := fmt.Sprintf("%s-backup", instance)
	name := fmt.Sprintf("%s/%s", instance, snapshot)

	op, err := i.server.CreateImage(
		api.ImagesPost{
			Source:  &api.ImagesPostSource{Type: "snapshot", Name: name},
			Aliases: []api.ImageAlias{{Name: alias}},
		}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create image: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return "", fmt.Errorf("failed to wait create image operation: %w", err)
	}

	fingerprint := op.Get().Metadata["fingerprint"].(string)

	return fingerprint, nil
}

func (i *Incus) ExportImage(ctx context.Context, fingerprint string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = i.server.GetImageFile(
		fingerprint,
		incus.ImageFileRequest{
			MetaFile: io.WriteSeeker(file),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get image file: %w", err)
	}

	return nil
}

func (i *Incus) DeleteImage(ctx context.Context, fingerprint string) error {
	op, err := i.server.DeleteImage(fingerprint)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if err := op.WaitContext(ctx); err != nil {
		return fmt.Errorf("failed to wait delete image operation: %w", err)
	}

	return nil
}

func (i *Incus) GetInstanceRecoveryInfo(ctx context.Context, instance string) (models.RecoveryInfo, error) {
	incusInstance, _, err := i.server.GetInstance(instance)
	if err != nil {
		return models.RecoveryInfo{}, fmt.Errorf("failed to get instance: %w", err)
	}

	info := models.RecoveryInfo{
		Config:  incusInstance.Config,
		Devices: incusInstance.Devices,
	}

	return info, nil
}

func (i *Incus) GetClusterMemberNames(ctx context.Context) ([]string, error) {
	return i.server.GetClusterMemberNames()
}

func New(config Config) *Incus {
	return &Incus{
		server:    config.Server,
		cluster:   config.Cluster,
		templates: config.Templates,
	}
}

func getInstanceIPFromState(state *api.InstanceState, nic string) (net.IP, error) {
	for _, address := range state.Network[nic].Addresses {
		if address.Family == AddressFamily {
			return net.ParseIP(address.Address), nil
		}
	}

	return nil, ErrNoSuchAddressFamily
}
