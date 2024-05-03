package deployer

import (
	"context"
	"fmt"
	"slices"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/models"
	"golang.org/x/sync/errgroup"
)

const (
	MaxConcurrentRequests = 3
)

type IncusProvider interface {
	GetInstanceNames(ctx context.Context) ([]string, error)
	LaunchInstance(ctx context.Context, instance *models.Instance) error
	DeleteInstance(ctx context.Context, instanceName string) error
}

type Deployer struct {
	incus IncusProvider
}

func (d *Deployer) Deploy(ctx context.Context, targetInstances []*models.Instance) error {
	existingInstancesNames, err := d.incus.GetInstanceNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing instances names: %w", err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(MaxConcurrentRequests)

	for _, existingInstanceName := range existingInstancesNames {
		existingInstanceName := existingInstanceName

		if slices.ContainsFunc(targetInstances, func(targetInstance *models.Instance) bool {
			return targetInstance.Name == existingInstanceName
		}) {
			continue
		}

		eg.Go(func() error {
			if err := d.incus.DeleteInstance(ctx, existingInstanceName); err != nil {
				return fmt.Errorf("failed to delete instance: %w", err)
			}
			return nil
		})
	}

	for _, targetInstance := range targetInstances {
		targetInstance := targetInstance

		if slices.ContainsFunc(existingInstancesNames, func(existingInstanceName string) bool {
			return targetInstance.Name == existingInstanceName
		}) {
			continue
		}

		eg.Go(func() error {
			if err := d.incus.LaunchInstance(ctx, targetInstance); err != nil {
				return fmt.Errorf("failed to launch instance: %w", err)
			}
			return nil
		})
	}

	return eg.Wait()
}

func New(incus IncusProvider) *Deployer {
	return &Deployer{incus: incus}
}
