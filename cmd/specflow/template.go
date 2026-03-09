package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/balazscsaba2006/specflow/templates"
	"github.com/spf13/cobra"
)

// docTypeShortNames maps short names used in CLI to embedded template file names.
// Entity types (story, epic, etc.) map directly; doc types get a "doc_" prefix.
var docTypeShortNames = map[string]string{
	"prd":         "doc_prd",
	"tech-spec":   "doc_tech-spec",
	"api-spec":    "doc_api-spec",
	"design-spec": "doc_design-spec",
	"adr":         "doc_adr",
	"one-pager":   "doc_one-pager",
	"generic":     "doc_generic",
}

// resolveTemplateName converts a user-facing short name to the embedded template name.
func resolveTemplateName(name string) string {
	if mapped, ok := docTypeShortNames[name]; ok {
		return mapped
	}
	return name
}

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tmpl"},
		Short:   "Manage templates",
		Long:    "List, override, and reset specflow templates.",
	}

	cmd.AddCommand(newTemplateLsCmd())
	cmd.AddCommand(newTemplateOverrideCmd())
	cmd.AddCommand(newTemplateResetCmd())

	return cmd
}

func newTemplateLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all available templates",
		RunE: func(_ *cobra.Command, _ []string) error {
			names := templates.List()
			root := appStore.Root()

			fmt.Printf("%-20s %s\n", "TEMPLATE", "OVERRIDE")
			fmt.Printf("%-20s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 8))
			for _, name := range names {
				if name == "skill" {
					continue
				}
				override := ""
				if templates.HasOverride(root, name) {
					override = "yes"
				}
				fmt.Printf("%-20s %s\n", name, override)
			}
			return nil
		},
	}
}

func newTemplateOverrideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "override <name>",
		Short: "Copy an embedded template to .specflow/templates/ for customization",
		Long: `Copies the default embedded template to the project override path.

For doc types, use the short name (e.g., "tech-spec" instead of "doc_tech-spec").
For entity types, use the name directly (e.g., "story", "epic", "initiative").

Examples:
  specflow template override tech-spec    # → .specflow/templates/doc_tech-spec.md
  specflow template override story        # → .specflow/templates/story.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveTemplateName(args[0])
			if name == "skill" {
				return fmt.Errorf("skill template is not overridable — use 'specflow sync' to update it")
			}
			root := appStore.Root()

			// Load the embedded default (ignore any existing override).
			content, err := templates.LoadEmbedded(name)
			if err != nil {
				return fmt.Errorf("unknown template %q", args[0])
			}

			dest := templates.OverridePath(root, name)
			if err := os.WriteFile(dest, []byte(content), 0o600); err != nil {
				return fmt.Errorf("writing override: %w", err)
			}

			fmt.Printf("Copied %s → %s\n", name, dest)
			return nil
		},
	}
}

func newTemplateResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <name>",
		Short: "Delete a template override, reverting to the embedded default",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := resolveTemplateName(args[0])
			if name == "skill" {
				return fmt.Errorf("skill template is not overridable — use 'specflow sync' to update it")
			}
			root := appStore.Root()

			path := templates.OverridePath(root, name)
			if err := os.Remove(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no override exists for %q", args[0])
				}
				return fmt.Errorf("removing override: %w", err)
			}

			fmt.Printf("Removed override for %s (reverted to embedded default)\n", name)
			return nil
		},
	}
}
