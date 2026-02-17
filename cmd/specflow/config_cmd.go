package main

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage project configuration",
		Long:  "Get, set, or list specflow project configuration values.",
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigLsCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			val, err := appConfig.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := appConfig.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := config.Save(appStore.ConfigFile(), appConfig); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			fmt.Printf("%s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func newConfigLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all config values",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			data, err := yaml.Marshal(appConfig)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			fmt.Print(string(data))
			return nil
		},
	}
}
