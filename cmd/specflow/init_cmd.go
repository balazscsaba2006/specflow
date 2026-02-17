package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var withClaude bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new specflow project",
		Long:  "Creates the .specflow/ directory structure and default config in the current directory.",
		RunE: func(_ *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			root := filepath.Join(cwd, ".specflow")

			// Check if already initialized.
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				return fmt.Errorf(".specflow/ already exists in %s", cwd)
			}

			// Create directory structure.
			s := store.New(root)
			if err := s.Init(); err != nil {
				return fmt.Errorf("creating directory structure: %w", err)
			}

			// Write default config.
			cfg := config.DefaultConfig()
			if err := config.Save(s.ConfigFile(), cfg); err != nil {
				return fmt.Errorf("writing default config: %w", err)
			}

			fmt.Println("Initialized specflow project in", cwd)

			if withClaude {
				if err := setupClaudeSettings(cwd); err != nil {
					return fmt.Errorf("setting up Claude settings: %w", err)
				}
				fmt.Println("Added specflow MCP server to .claude/settings.json")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&withClaude, "with-claude", false, "Also configure .claude/settings.json with specflow MCP server")

	return cmd
}

// setupClaudeSettings creates or updates .claude/settings.json with the specflow MCP server entry.
func setupClaudeSettings(projectRoot string) error {
	claudeDir := filepath.Join(projectRoot, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Ensure .claude/ directory exists.
	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		return fmt.Errorf("creating .claude directory: %w", err)
	}

	// Read existing settings or start fresh.
	settings := make(map[string]interface{})
	data, readErr := os.ReadFile(settingsPath)
	if readErr == nil {
		if unmarshalErr := json.Unmarshal(data, &settings); unmarshalErr != nil {
			return fmt.Errorf("parsing existing settings.json: %w", unmarshalErr)
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("reading settings.json: %w", readErr)
	}

	// Merge specflow into mcpServers.
	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}
	mcpServers["specflow"] = map[string]interface{}{
		"command": "specflow",
		"args":    []string{"mcp"},
	}
	settings["mcpServers"] = mcpServers

	// Write back with indentation.
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings.json: %w", err)
	}

	return os.WriteFile(settingsPath, append(out, '\n'), 0o600)
}
