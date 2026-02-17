package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/spf13/cobra"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9-]+`)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import an existing markdown file as a specflow artifact",
		Long: `Reads a markdown file and imports it into the specflow project.

If the file has YAML frontmatter, its fields are used.
If not, metadata is generated from the filename.

The --type flag determines the artifact type:
  story (default), doc, epic, initiative, decision`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			epicSlug, _ := cmd.Flags().GetString("epic")
			entityType, _ := cmd.Flags().GetString("type")

			data, err := os.ReadFile(filePath) //nolint:gosec // user-provided path is intentional
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			slug := slugFromFilename(filePath)
			if err := models.ValidateSlug(slug); err != nil {
				return fmt.Errorf("generated slug %q is invalid: %w (use a simpler filename)", slug, err)
			}

			switch entityType {
			case "story":
				return importStory(data, slug, epicSlug)
			case "doc":
				return importDoc(data, slug, epicSlug)
			case "epic":
				return importEpic(data, slug)
			case "initiative":
				return importInitiative(data, slug)
			case "decision":
				return importDecision(data, slug)
			default:
				return fmt.Errorf("unsupported type %q: must be one of [story, doc, epic, initiative, decision]", entityType)
			}
		},
	}

	cmd.Flags().String("type", "story", "Artifact type: story, doc, epic, initiative, decision")
	cmd.Flags().String("epic", "", "Parent epic slug (for stories and docs)")

	return cmd
}

func slugFromFilename(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonSlugChars.ReplaceAllString(slug, "")
	// Collapse multiple hyphens.
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}

func importStory(data []byte, slug, epicSlug string) error {
	var st models.Story
	body, err := store.Parse(data, &st)
	if err != nil {
		// No frontmatter — treat entire content as body.
		st.Title = slug
		body = string(data)
	}
	st.Slug = slug
	st.Body = body
	if epicSlug != "" && st.Epic == "" {
		st.Epic = epicSlug
	}

	if err := appStore.CreateStory(&st); err != nil {
		return err
	}
	fmt.Printf("Imported story %q (%s)\n", st.Title, st.Slug)
	return nil
}

func importDoc(data []byte, slug, epicSlug string) error {
	var d models.Document
	body, err := store.Parse(data, &d)
	if err != nil {
		d.Title = slug
		d.Type = models.DocTypePRD
		body = string(data)
	}
	d.Slug = slug
	d.Body = body
	if epicSlug != "" && d.Epic == "" {
		d.Epic = epicSlug
	}

	if err := appStore.CreateDoc(&d); err != nil {
		return err
	}
	fmt.Printf("Imported doc %q (%s)\n", d.Title, d.Slug)
	return nil
}

func importEpic(data []byte, slug string) error {
	var e models.Epic
	body, err := store.Parse(data, &e)
	if err != nil {
		e.Title = slug
		body = string(data)
	}
	e.Slug = slug
	e.Body = body

	if err := appStore.CreateEpic(&e); err != nil {
		return err
	}
	fmt.Printf("Imported epic %q (%s)\n", e.Title, e.Slug)
	return nil
}

func importInitiative(data []byte, slug string) error {
	var i models.Initiative
	body, err := store.Parse(data, &i)
	if err != nil {
		i.Title = slug
		body = string(data)
	}
	i.Slug = slug
	i.Body = body

	if err := appStore.CreateInitiative(&i); err != nil {
		return err
	}
	fmt.Printf("Imported initiative %q (%s)\n", i.Title, i.Slug)
	return nil
}

func importDecision(data []byte, slug string) error {
	var d models.Decision
	body, err := store.Parse(data, &d)
	if err != nil {
		d.Title = slug
		body = string(data)
	}
	d.Slug = slug
	d.Body = body

	if err := appStore.CreateDecision(&d); err != nil {
		return err
	}
	fmt.Printf("Imported decision %q (%s)\n", d.Title, d.Slug)
	return nil
}
