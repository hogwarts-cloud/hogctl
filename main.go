package main

import (
	"embed"
	"fmt"
	"os"
	"os/user"
	"text/template"

	"github.com/hogwarts-cloud/hogctl/config"
	"github.com/hogwarts-cloud/hogctl/internal/apply"
	"github.com/hogwarts-cloud/hogctl/internal/backup"
	"github.com/hogwarts-cloud/hogctl/internal/executor"
	"github.com/hogwarts-cloud/hogctl/internal/incus"
	"github.com/hogwarts-cloud/hogctl/internal/mail"
	"github.com/hogwarts-cloud/hogctl/internal/validate"
	incusServer "github.com/lxc/incus/client"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed templates
	templatesFS embed.FS

	clusterConfigPath   string
	instancesConfigPath string
)

var root = &cobra.Command{
	Use:   "hogctl",
	Short: "Utility for Hogwarts Cloud",
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the configuration from directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		clusterConfig := config.LoadCluster(clusterConfigPath)
		instancesConfig := config.LoadInstances(instancesConfigPath)

		server, err := incusServer.ConnectIncusUnix("", nil)
		if err != nil {
			return fmt.Errorf("failed to connect to Incus server: %w", err)
		}

		incusTemplates, err := template.ParseFS(templatesFS, "templates/incus/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse incus templates: %w", err)
		}

		mailTemplates, err := template.ParseFS(templatesFS, "templates/mail/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse mail templates: %w", err)
		}

		incusConfig := incus.Config{
			Server:    server,
			Cluster:   clusterConfig.Cluster,
			Templates: incusTemplates,
		}

		incus := incus.New(incusConfig)

		currentUser, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}

		senderName := fmt.Sprintf("%s@%s", currentUser.Username, hostname)

		mailSender := mail.NewSender(clusterConfig.Cluster.Mail.Server, senderName)

		applyConfig := apply.Config{
			Incus:         incus,
			MailSender:    mailSender,
			Domain:        clusterConfig.Cluster.Domain,
			CIDR:          clusterConfig.Cluster.Network.CIDR,
			OccupiedIPs:   clusterConfig.Cluster.Network.OccupiedIPs,
			MailTemplates: mailTemplates,
		}

		applyCmd := apply.NewCmd(applyConfig)

		launchedInstancesInfo, err := applyCmd.Run(cmd.Context(), instancesConfig.Instances)
		if err != nil {
			return fmt.Errorf("failed to apply instances: %w", err)
		}

		data, err := yaml.Marshal(launchedInstancesInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal to yaml: %w", err)
		}

		fmt.Println(string(data))

		return nil
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup instances located on the same host",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		clusterConfig := config.LoadCluster(clusterConfigPath)

		server, err := incusServer.ConnectIncusUnix("", nil)
		if err != nil {
			return fmt.Errorf("failed to connect to Incus server: %w", err)
		}

		incusConfig := incus.Config{
			Server:  server,
			Cluster: clusterConfig.Cluster,
		}

		incus := incus.New(incusConfig)

		executor := executor.New()

		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}

		backupConfig := backup.Config{
			Incus:     incus,
			Executor:  executor,
			Hostname:  hostname,
			Directory: clusterConfig.Cluster.Backup.Dir,
		}

		backupCmd := backup.NewCmd(backupConfig)

		if err := backupCmd.Run(cmd.Context()); err != nil {
			return fmt.Errorf("failed to backup instances: %w", err)
		}

		return nil
	},
}

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Notify users that the expiration date is approaching",
	RunE: func(cmd *cobra.Command, args []string) error {
		// cmd.SilenceUsage = true
		// cfg := config.Load(configPath)

		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration from directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		clusterConfig := config.LoadCluster(clusterConfigPath)
		instancesConfig := config.LoadInstances(instancesConfigPath)

		validateCmd := validate.NewCmd(clusterConfig.Cluster.Flavors, clusterConfig.Cluster.Domain)

		if err := validateCmd.Run(instancesConfig.Instances); err != nil {
			return fmt.Errorf("failed to validate instances: %w", err)
		}

		return nil
	},
}

func init() {
	root.PersistentFlags().StringVar(&clusterConfigPath, "cluster-config", "/etc/hogctl/cluster.yaml", "Path to cluster config")
	root.PersistentFlags().StringVar(&instancesConfigPath, "instances-config", "/etc/hogctl/instances.yaml", "Path to instances config")
	root.AddCommand(applyCmd, backupCmd, notifyCmd, validateCmd)
}

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
