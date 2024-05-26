package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"gopkg.in/yaml.v3"
)

const (
	InstanceBackup = "instance.tar.gz"
	ConfigBackup   = "config.yaml"
)

//go:generate mockgen -source backup.go -destination mocks/incus_provider.go -package mocks
type IncusProvider interface {
	GetLaunchedInstances(ctx context.Context) ([]models.InstanceInfo, error)
	UpdateInstanceState(ctx context.Context, instance string, state models.InstanceState) error
	CreateSnapshot(ctx context.Context, instance string) (string, error)
	DeleteSnapshot(ctx context.Context, instance, snapshot string) error
	CreateImageFromSnapshot(ctx context.Context, instance string, snapshot string) (string, error)
	ExportImage(ctx context.Context, fingerprint string, path string) error
	DeleteImage(ctx context.Context, fingerprint string) error
	GetInstanceRecoveryInfo(ctx context.Context, instance string) (models.RecoveryInfo, error)
	GetClusterMemberNames(ctx context.Context) ([]string, error)
}

//go:generate mockgen -source backup.go -destination mocks/executor.go -package mocks
type Executor interface {
	Execute(command string, args []string) error
}

type Config struct {
	Incus     IncusProvider
	Executor  Executor
	Hostname  string
	Directory string
}

type BackupCmd struct {
	incus     IncusProvider
	executor  Executor
	hostname  string
	directory string
}

func (b *BackupCmd) Run(ctx context.Context) error {
	launchedInstances, err := b.incus.GetLaunchedInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to get launched instances: %w", err)
	}

	for _, instance := range launchedInstances {
		if instance.Location != b.hostname {
			continue
		}

		directory := filepath.Join(b.directory, instance.Name) + "/"

		if err := os.MkdirAll(directory, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		instanceBackupPath := filepath.Join(directory, InstanceBackup)
		configBackupPath := filepath.Join(directory, ConfigBackup)

		if err := b.backupInstance(ctx, instance.Name, instanceBackupPath); err != nil {
			return fmt.Errorf("failed to backup instance: %w", err)
		}

		if err := b.backupConfig(ctx, instance.Name, configBackupPath); err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}

		clusterMembers, err := b.incus.GetClusterMemberNames(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cluster member names")
		}

		command := "rsync"
		for _, member := range clusterMembers {
			if member == b.hostname {
				continue
			}

			args := []string{"-r", "--mkpath", directory, fmt.Sprintf("%s:%s", member, directory)}

			if err := b.executor.Execute(command, args); err != nil {
				return fmt.Errorf("failed to execute rsync: %w", err)
			}
		}

		if err := os.RemoveAll(directory); err != nil {
			return fmt.Errorf("failed to delete backup directory: %w", err)
		}
	}

	return nil
}

func (b *BackupCmd) backupInstance(ctx context.Context, instance, path string) error {
	if err := b.incus.UpdateInstanceState(ctx, instance, models.StoppedState); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	snapshot, err := b.incus.CreateSnapshot(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	if err := b.incus.UpdateInstanceState(ctx, instance, models.RunningState); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fingerprint, err := b.incus.CreateImageFromSnapshot(ctx, instance, snapshot)
	if err != nil {
		return fmt.Errorf("failed to create image from snapshot: %w", err)
	}

	if err := b.incus.DeleteSnapshot(ctx, instance, snapshot); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	if err := b.incus.ExportImage(ctx, fingerprint, path); err != nil {
		return fmt.Errorf("failed to export image: %w", err)
	}

	if err := b.incus.DeleteImage(ctx, fingerprint); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

func (b *BackupCmd) backupConfig(ctx context.Context, instance, path string) error {
	recovery, err := b.incus.GetInstanceRecoveryInfo(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to get instance recovery info: %w", err)
	}

	data, err := yaml.Marshal(recovery)
	if err != nil {
		return fmt.Errorf("failed to marshal recovery info: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func NewCmd(config Config) *BackupCmd {
	return &BackupCmd{
		incus:     config.Incus,
		executor:  config.Executor,
		hostname:  config.Hostname,
		directory: config.Directory,
	}
}
