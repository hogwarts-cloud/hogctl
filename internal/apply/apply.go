package apply

import (
	"context"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/hogwarts-cloud/hogctl/internal/network"
	"github.com/hogwarts-cloud/hogctl/pkg/constants"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

const (
	MaxConcurrentRequests   = 3
	SubjectInstanceCreated  = "Your instance in Hogwarts Cloud has been created"
	InstanceCreatedTemplate = "created"
	SubjectInstanceDeleted  = "Your instance in Hogwarts Cloud has been deleted"
	InstanceDeletedTemplate = "deleted"
)

//go:generate mockgen -source apply.go -destination mocks/incus_provider.go -package mocks
type IncusProvider interface {
	GetLaunchedInstances(ctx context.Context) ([]models.InstanceInfo, error)
	GetInstanceIP(ctx context.Context, instance string) (net.IP, error)
	LaunchInstance(ctx context.Context, instance models.LaunchConfig, async bool) error
	DeleteInstance(ctx context.Context, instance string) error
}

//go:generate mockgen -source apply.go -destination mocks/mail_sender.go -package mocks
type MailSender interface {
	Send(recipient, subject, text string) error
}

type Config struct {
	Incus         IncusProvider
	MailSender    MailSender
	Domain        string
	CIDR          net.IPNet
	OccupiedIPs   []net.IP
	MailTemplates *template.Template
}

type ApplyCmd struct {
	incus         IncusProvider
	mailSender    MailSender
	domain        string
	cidr          net.IPNet
	occupiedIPs   []net.IP
	mailTemplates *template.Template
}

func (a *ApplyCmd) Run(ctx context.Context, targetInstances []models.Instance) (models.ApplyResult, error) {
	launchedInstances, err := a.incus.GetLaunchedInstances(ctx)
	if err != nil {
		return models.ApplyResult{}, fmt.Errorf("failed to get launched instances: %w", err)
	}

	instancesToDelete := make([]models.InstanceInfo, 0)
	for _, launchedInstance := range launchedInstances {
		targetInstancesContains := lo.ContainsBy(targetInstances, func(targetInstance models.Instance) bool {
			return targetInstance.Name == launchedInstance.Name
		})

		if !targetInstancesContains {
			instancesToDelete = append(instancesToDelete, launchedInstance)
		}
	}

	instancesToLaunch := make([]models.Instance, 0)
	for _, targetInstance := range targetInstances {
		launchedInstancesContains := lo.ContainsBy(launchedInstances, func(launchedInstance models.InstanceInfo) bool {
			return targetInstance.Name == launchedInstance.Name
		})

		if targetInstance.IsExpired() {
			ip, err := a.incus.GetInstanceIP(ctx, targetInstance.Name)
			if err != nil {
				return models.ApplyResult{}, fmt.Errorf("failed to get instance ip: %w", err)
			}

			info := models.InstanceInfo{
				Name: targetInstance.Name,
				InstanceUserInfo: models.InstanceUserInfo{
					Name:  targetInstance.User.Name,
					Email: targetInstance.User.Email,
				},
				InstanceNetworkInfo: models.InstanceNetworkInfo{
					IP:            ip,
					ForwardedPort: network.GeneratePortByIP(ip),
				},
			}

			instancesToDelete = append(instancesToDelete, info)
			continue
		}

		if !launchedInstancesContains {
			instancesToLaunch = append(instancesToLaunch, targetInstance)
		}
	}

	if err := a.deleteInstances(ctx, instancesToDelete); err != nil {
		return models.ApplyResult{}, fmt.Errorf("failed to delete instances: %w", err)
	}

	occupiedIPs := lo.Map(launchedInstances, func(launchedInstance models.InstanceInfo, _ int) net.IP {
		return launchedInstance.IP
	})

	newlyLaunchedInstances, err := a.launchInstances(ctx, instancesToLaunch, occupiedIPs)
	if err != nil {
		return models.ApplyResult{}, fmt.Errorf("failed to launch instances: %w", err)
	}

	result := models.ApplyResult{
		Launched: newlyLaunchedInstances,
		Deleted:  instancesToDelete,
	}

	return result, nil
}

func (a *ApplyCmd) deleteInstances(ctx context.Context, instances []models.InstanceInfo) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	for _, instance := range instances {
		instance := instance

		eg.Go(func() error { return a.deleteInstance(ctx, instance) })
	}

	return eg.Wait()
}

func (a *ApplyCmd) deleteInstance(ctx context.Context, instance models.InstanceInfo) error {
	if err := a.incus.DeleteInstance(ctx, instance.Name); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	buf := &strings.Builder{}
	if err := a.mailTemplates.ExecuteTemplate(
		buf,
		InstanceDeletedTemplate+constants.TemplateExtension,
		instance,
	); err != nil {
		return fmt.Errorf("failed to execute instance deleted template: %w", err)
	}

	// if err := a.mailSender.Send(instance.Email, SubjectInstanceDeleted, buf.String()); err != nil {
	// 	return fmt.Errorf("failed to mail about delete instance: %w", err)
	// }

	return nil
}

func (a *ApplyCmd) launchInstances(ctx context.Context, instances []models.Instance, occupiedInstancesIPs []net.IP) ([]models.InstanceInfo, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	ips, err := network.GetAvailableIPs(len(instances), a.cidr, append(a.occupiedIPs, occupiedInstancesIPs...))
	if err != nil {
		return nil, fmt.Errorf("failed to get available ips: %w", err)
	}

	ipIndex := 0

	instancesInfo := make([]models.InstanceInfo, 0, len(instances))
	for _, instance := range instances {
		ip := ips[ipIndex]
		ipIndex++

		instanceUserInfo := models.InstanceUserInfo{
			Name:  instance.User.Name,
			Email: instance.User.Email,
		}

		instanceNetworkInfo := models.InstanceNetworkInfo{
			IP:            ip,
			ForwardedPort: network.GeneratePortByIP(ip),
		}

		instanceInfo := models.InstanceInfo{
			Name:                instance.Name,
			InstanceUserInfo:    instanceUserInfo,
			InstanceNetworkInfo: instanceNetworkInfo,
		}

		launchConfig := models.LaunchConfig{
			Instance:            instance,
			InstanceNetworkInfo: instanceNetworkInfo,
		}

		instancesInfo = append(instancesInfo, instanceInfo)

		if err := a.launchInstance(ctx, launchConfig); err != nil {
			return nil, fmt.Errorf("failed to launch instance: %w", err)
		}
	}

	return instancesInfo, nil
}

func (a *ApplyCmd) launchInstance(ctx context.Context, instance models.LaunchConfig) error {
	if err := a.incus.LaunchInstance(ctx, instance, true); err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	buf := &strings.Builder{}
	if err := a.mailTemplates.ExecuteTemplate(
		buf,
		InstanceCreatedTemplate+constants.TemplateExtension,
		map[string]any{"Name": instance.Name, "Domain": a.domain, "Port": instance.ForwardedPort, "User": instance.User.Name},
	); err != nil {
		return fmt.Errorf("failed to execute instance created template: %w", err)
	}

	// if err := a.mailSender.Send(instance.User.Email, SubjectInstanceCreated, buf.String()); err != nil {
	// 	return fmt.Errorf("failed to mail about launch: %w", err)
	// }

	return nil
}

func NewCmd(config Config) *ApplyCmd {
	return &ApplyCmd{
		incus:         config.Incus,
		mailSender:    config.MailSender,
		domain:        config.Domain,
		cidr:          config.CIDR,
		occupiedIPs:   config.OccupiedIPs,
		mailTemplates: config.MailTemplates,
	}
}
