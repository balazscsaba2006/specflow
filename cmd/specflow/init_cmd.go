package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/balazscsaba2006/specflow/internal/config"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

// installSkillGlobal writes the specflow skill to ~/.claude/skills/specflow/SKILL.md.
// Uses the embedded template directly — no user override support for the skill.
func installSkillGlobal() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	content, err := templates.LoadEmbedded("skill")
	if err != nil {
		return fmt.Errorf("loading skill template: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "specflow")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	return os.WriteFile(skillPath, []byte(content), 0o600)
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new specflow project",
		Long:  "Creates the .specflow/ directory structure, configures .mcp.json, and installs the Claude Code skill globally.",
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

			// Write default config, respecting global default_mode if set.
			cfg := config.DefaultConfig()
			if gc, err := config.LoadGlobal(); err == nil && gc.DefaultMode != "" {
				if gc.DefaultMode == "careful" || gc.DefaultMode == "fast" {
					cfg.Mode = gc.DefaultMode
				}
			}
			if err := config.Save(s.ConfigFile(), cfg); err != nil {
				return fmt.Errorf("writing default config: %w", err)
			}

			fmt.Println("Initialized specflow project in", cwd)

			if err := setupMCPConfig(cwd); err != nil {
				return fmt.Errorf("setting up MCP config: %w", err)
			}
			fmt.Println("Added specflow MCP server to .mcp.json")

			if err := installSkillGlobal(); err != nil {
				return fmt.Errorf("installing skill: %w", err)
			}
			fmt.Println("Installed specflow skill to ~/.claude/skills/specflow/SKILL.md")

			return nil
		},
	}
}

// setupMCPConfig creates or updates .mcp.json with the specflow MCP server entry.
func setupMCPConfig(projectRoot string) error {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")

	// Read existing config or start fresh.
	mcpConfig := make(map[string]interface{})
	data, readErr := os.ReadFile(mcpPath)
	if readErr == nil {
		if unmarshalErr := json.Unmarshal(data, &mcpConfig); unmarshalErr != nil {
			return fmt.Errorf("parsing existing .mcp.json: %w", unmarshalErr)
		}
	} else if !os.IsNotExist(readErr) {
		return fmt.Errorf("reading .mcp.json: %w", readErr)
	}

	// Merge specflow into mcpServers.
	mcpServers, ok := mcpConfig["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}
	mcpServers["specflow"] = map[string]interface{}{
		"command": "specflow",
		"args":    []string{"mcp"},
	}
	mcpConfig["mcpServers"] = mcpServers

	// Write back with indentation.
	out, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling .mcp.json: %w", err)
	}

	return os.WriteFile(mcpPath, append(out, '\n'), 0o600)
}
