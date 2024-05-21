package applier

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

type IncusProvider interface {
	GetLaunchedInstances(ctx context.Context) ([]models.InstanceInfo, error)
	GetInstanceIP(ctx context.Context, instance string) (net.IP, error)
	LaunchInstance(ctx context.Context, instance models.LaunchConfig) error
	DeleteInstance(ctx context.Context, instance string) error
}

type Mailer interface {
	SendMail(recipient, subject, text string) error
}

type Config struct {
	Domain        string
	CIDR          net.IPNet
	OccupiedIPs   []net.IP
	MailTemplates *template.Template
}

type Applier struct {
	incus         IncusProvider
	mailer        Mailer
	domain        string
	cidr          net.IPNet
	occupiedIPs   []net.IP
	mailTemplates *template.Template
}

func (a *Applier) Apply(ctx context.Context, targetInstances []models.Instance) (models.ApplyResult, error) {
	launchedInstances, err := a.incus.GetLaunchedInstances(ctx)
	if err != nil {
		return models.ApplyResult{}, fmt.Errorf("failed to get existing instances: %w", err)
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
				Name:  targetInstance.Name,
				Email: targetInstance.User.Email,
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

func (a *Applier) deleteInstances(ctx context.Context, instances []models.InstanceInfo) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	for _, instance := range instances {
		instance := instance

		eg.Go(func() error { return a.deleteInstance(ctx, instance) })
	}

	return eg.Wait()
}

func (a *Applier) deleteInstance(ctx context.Context, instance models.InstanceInfo) error {
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

	if err := a.mailer.SendMail(instance.Email, SubjectInstanceDeleted, buf.String()); err != nil {
		return fmt.Errorf("failed to mail about delete instance: %w", err)
	}

	return nil
}

func (a *Applier) launchInstances(ctx context.Context, instances []models.Instance, occupiedInstancesIPs []net.IP) ([]models.InstanceInfo, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	ips, err := network.GetAvailableIPs(len(instances), a.cidr, append(a.occupiedIPs, occupiedInstancesIPs...))
	if err != nil {
		return nil, fmt.Errorf("failed to get available ips: %w", err)
	}

	ipIndex := 0

	instancesInfo := make([]models.InstanceInfo, 0, len(instances))
	for _, instance := range instances {
		ip := ips[ipIndex]
		ipIndex++

		instanceNetworkInfo := models.InstanceNetworkInfo{
			IP:            ip,
			ForwardedPort: network.GeneratePortByIP(ip),
		}

		instanceInfo := models.InstanceInfo{
			Name:                instance.Name,
			Email:               instance.User.Email,
			InstanceNetworkInfo: instanceNetworkInfo,
		}

		launchConfig := models.LaunchConfig{
			Instance:            instance,
			InstanceNetworkInfo: instanceNetworkInfo,
		}

		instancesInfo = append(instancesInfo, instanceInfo)

		eg.Go(func() error { return a.launchInstance(ctx, launchConfig) })
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return instancesInfo, nil
}

func (a *Applier) launchInstance(ctx context.Context, instance models.LaunchConfig) error {
	if err := a.incus.LaunchInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	buf := &strings.Builder{}
	if err := a.mailTemplates.ExecuteTemplate(
		buf,
		InstanceCreatedTemplate+constants.TemplateExtension,
		map[string]any{"Name": instance.Name, "Domain": a.domain, "Port": instance.ForwardedPort},
	); err != nil {
		return fmt.Errorf("failed to execute instance created template: %w", err)
	}

	if err := a.mailer.SendMail(instance.User.Email, SubjectInstanceCreated, buf.String()); err != nil {
		return fmt.Errorf("failed to mail about launch: %w", err)
	}

	return nil
}

func New(incus IncusProvider, notifier Mailer, cfg Config) *Applier {
	return &Applier{
		incus:         incus,
		mailer:        notifier,
		domain:        cfg.Domain,
		cidr:          cfg.CIDR,
		occupiedIPs:   cfg.OccupiedIPs,
		mailTemplates: cfg.MailTemplates,
	}
}
