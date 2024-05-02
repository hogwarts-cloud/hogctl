package validator

import (
	"fmt"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/models"
)

func Validate(instances []*models.Instance) error {
	for _, instance := range instances {
		if err := validateInstance(instance); err != nil {
			return fmt.Errorf("failed to validate instance: %w", err)
		}
	}

	return nil
}

func validateInstance(instance *models.Instance) error {
	return nil
}
