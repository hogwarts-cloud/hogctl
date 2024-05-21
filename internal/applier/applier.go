package applier

import (
	"context"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/hogwarts-cloud/hogctl/pkg/constants"
	"github.com/hogwarts-cloud/hogctl/pkg/utils"
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
	GetClusterInfo(ctx context.Context) (models.ClusterInfo, error)
	LaunchInstance(ctx context.Context, instance models.LaunchConfig) error
	DeleteInstance(ctx context.Context, instance string) error
}

type Mailer interface {
	Mail(recipient, subject, text string) error
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

func (a *Applier) Apply(ctx context.Context, targetInstances []models.Instance) ([]models.LaunchedInstanceInfo, error) {
	clusterInfo, err := a.incus.GetClusterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing instances: %w", err)
	}

	instancesToDelete := make([]models.InstanceInfo, 0)
	for _, existingInstance := range clusterInfo.Instances {
		targetInstancesContains := lo.ContainsBy(targetInstances, func(targetInstance models.Instance) bool {
			return targetInstance.Name == existingInstance.Name
		})

		if !targetInstancesContains {
			instancesToDelete = append(instancesToDelete, existingInstance)
		}
	}

	instancesToLaunch := make([]models.Instance, 0)
	for _, targetInstance := range targetInstances {
		existingsInstancesContains := lo.ContainsBy(clusterInfo.Instances, func(existingInstance models.InstanceInfo) bool {
			return targetInstance.Name == existingInstance.Name
		})

		if targetInstance.IsExpired() {
			info := models.InstanceInfo{
				Name:  targetInstance.Name,
				Email: targetInstance.User.Email,
			}

			instancesToDelete = append(instancesToDelete, info)
			continue
		}

		if !existingsInstancesContains {
			instancesToLaunch = append(instancesToLaunch, targetInstance)
		}
	}

	if err := a.deleteInstances(ctx, instancesToDelete); err != nil {
		return nil, fmt.Errorf("failed to delete instances: %w", err)
	}

	launchConfigs, err := a.launchInstances(ctx, instancesToLaunch, clusterInfo.IPs)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instances: %w", err)
	}

	return launchConfigs, nil
}

func (a *Applier) deleteInstances(ctx context.Context, instances []models.InstanceInfo) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	for _, instance := range instances {
		instance := instance

		eg.Go(func() error {
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

			if err := a.mailer.Mail(instance.Email, SubjectInstanceDeleted, buf.String()); err != nil {
				return fmt.Errorf("failed to mail about delete instance: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (a *Applier) launchInstances(ctx context.Context, instances []models.Instance, occupiedInstancesIPs []net.IP) ([]models.LaunchedInstanceInfo, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	ips, err := utils.GetAvailableIPs(len(instances), a.cidr, append(a.occupiedIPs, occupiedInstancesIPs...))
	if err != nil {
		return nil, fmt.Errorf("failed to get available ips: %w", err)
	}

	ipIndex := 0

	launchedInstancesInfo := make([]models.LaunchedInstanceInfo, 0, len(instances))
	for _, instance := range instances {
		ip := ips[ipIndex]
		ipIndex++

		launchConfig := models.LaunchConfig{
			Instance: instance,
			IP:       ip,
		}

		launchedInstanceInfo := models.LaunchedInstanceInfo{
			Name: instance.Name,
			IP:   ip,
			Port: 62000 + int(ip.To4()[3]),
		}

		launchedInstancesInfo = append(launchedInstancesInfo, launchedInstanceInfo)

		eg.Go(func() error {
			if err := a.incus.LaunchInstance(ctx, launchConfig); err != nil {
				return fmt.Errorf("failed to launch instance: %w", err)
			}

			buf := &strings.Builder{}
			if err := a.mailTemplates.ExecuteTemplate(
				buf,
				InstanceCreatedTemplate+constants.TemplateExtension,
				map[string]any{"Name": launchedInstanceInfo.Name, "Domain": a.domain, "Port": launchedInstanceInfo.Port},
			); err != nil {
				return fmt.Errorf("failed to execute instance created template: %w", err)
			}

			if err := a.mailer.Mail(launchConfig.User.Email, SubjectInstanceCreated, buf.String()); err != nil {
				return fmt.Errorf("failed to mail about launch: %w", err)
			}

			return nil
		})

	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return launchedInstancesInfo, nil
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
