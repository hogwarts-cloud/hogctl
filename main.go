package main

import (
	"fmt"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/deployer"
	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/incus"
	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/parser"
	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/validator"
	"github.com/spf13/cobra"
)

var (
	path string
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

		instances, err := parser.Parse(path)
		if err != nil {
			return fmt.Errorf("failed to parse instances: %w", err)
		}

		if err := validator.Validate(instances); err != nil {
			return fmt.Errorf("failed to validate instances: %w", err)
		}

		return nil
	},
}

var deploy = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the configuration from directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		instances, err := parser.Parse(path)
		if err != nil {
			return fmt.Errorf("failed to parse instances: %w", err)
		}

		incus, err := incus.New()
		if err != nil {
			return fmt.Errorf("failed to create incus client: %w", err)
		}

		deployer := deployer.New(incus)

		if err := deployer.Deploy(cmd.Context(), instances); err != nil {
			return fmt.Errorf("failed to deploy instances: %w", err)
		}

		return nil
	},
}

func init() {
	root.PersistentFlags().StringVar(&path, "path", "", "Path to instances directory")
	root.MarkPersistentFlagRequired("path")
	root.AddCommand(validate, deploy)
}

func main() {
	root.Execute()
}
