package main

import (
	"embed"
	"fmt"
	"os"
	"text/template"

	"github.com/hogwarts-cloud/hogctl/config"
	"github.com/hogwarts-cloud/hogctl/internal/applier"
	"github.com/hogwarts-cloud/hogctl/internal/incus"
	"github.com/hogwarts-cloud/hogctl/internal/mailer"
	"github.com/hogwarts-cloud/hogctl/internal/validator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed templates
	templatesFS embed.FS

	configPath string
)

var root = &cobra.Command{
	Use:   "hogctl",
	Short: "Utility for Hogwarts Cloud",
}

var validate = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration from directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cfg := config.Load(configPath)

		validator := validator.New(cfg.Cluster.Flavors)

		if err := validator.Validate(cfg.Instances); err != nil {
			return fmt.Errorf("failed to validate instances: %w", err)
		}

		return nil
	},
}

var apply = &cobra.Command{
	Use:   "apply",
	Short: "Apply the configuration from directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cfg := config.Load(configPath)

		incusTemplates, err := template.ParseFS(templatesFS, "templates/incus/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse incus templates: %w", err)
		}

		mailTemplates, err := template.ParseFS(templatesFS, "templates/mail/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse mail templates: %w", err)
		}

		incus, err := incus.New(cfg.Cluster, incusTemplates)
		if err != nil {
			return fmt.Errorf("failed to create incus client: %w", err)
		}

		mailer := mailer.New(cfg.Cluster.Mail.Server)

		applierCfg := applier.Config{
			Domain:        cfg.Cluster.Domain,
			CIDR:          cfg.Cluster.Network.CIDR,
			OccupiedIPs:   cfg.Cluster.Network.OccupiedIPs,
			MailTemplates: mailTemplates,
		}

		applier := applier.New(incus, mailer, applierCfg)

		launchedInstancesInfo, err := applier.Apply(cmd.Context(), cfg.Instances)
		if err != nil {
			return fmt.Errorf("failed to apply instances: %w", err)
		}

		message, err := yaml.Marshal(launchedInstancesInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal to yaml: %w", err)
		}

		fmt.Println(string(message))

		return nil
	},
}

func init() {
	root.PersistentFlags().StringVar(&configPath, "config", "", "Path to config directory")
	root.MarkPersistentFlagRequired("config")
	root.AddCommand(validate, apply)
}

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
