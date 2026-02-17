package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/spf13/cobra"
)

// Shared state set by PersistentPreRun.
var (
	appStore  *store.Store
	appConfig config.Config
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "specflow",
		Short: "Spec-driven development CLI",
		Long:  "specflow — a structured memory and context layer for Claude Code.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip store init for commands that don't need it.
			if cmd.Name() == "init" || cmd.Name() == "version" {
				return nil
			}
			return initStoreAndConfig()
		},
		SilenceUsage: true,
	}

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newInitiativeCmd())
	cmd.AddCommand(newEpicCmd())
	cmd.AddCommand(newStoryCmd())
	cmd.AddCommand(newDocCmd())
	cmd.AddCommand(newDecisionCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newQuestionsCmd())
	cmd.AddCommand(newBlockedCmd())
	cmd.AddCommand(newAssumptionsCmd())
	cmd.AddCommand(newLogCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newMCPCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("specflow %s (%s) built %s\n", version, commit, date)
		},
	}
}

func initStoreAndConfig() error {
	root, err := findSpecflowRoot()
	if err != nil {
		return err
	}
	appStore = store.New(root)
	appConfig, err = config.Load(appStore.ConfigFile())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	return nil
}

// findSpecflowRoot walks up from cwd looking for .specflow/ directory.
func findSpecflowRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ".specflow")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a specflow project (no .specflow/ directory found)\nRun 'specflow init' to create one")
}
