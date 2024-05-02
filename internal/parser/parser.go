package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/models"
	"gopkg.in/yaml.v3"
)

const YAMLExtension = ".yaml"

func Parse(path string) ([]*models.Instance, error) {
	instances := make([]*models.Instance, 0)

	err := filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() || filepath.Ext(path) != YAMLExtension {
			return nil
		}

		instance, err := parseInstance(path)
		if err != nil {
			return fmt.Errorf("failed to parse instance: %w", err)
		}

		instances = append(instances, instance)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walking through directory: %w", err)
	}

	return instances, nil
}

func parseInstance(path string) (*models.Instance, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	instance := new(models.Instance)
	if err := yaml.Unmarshal(content, instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	return instance, nil
}
