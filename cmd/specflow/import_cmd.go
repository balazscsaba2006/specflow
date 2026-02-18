package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
		Use:   "import [file]",
		Short: "Import an existing markdown file or GitHub issue as a specflow artifact",
		Long: `Reads a markdown file and imports it into the specflow project.

If the file has YAML frontmatter, its fields are used.
If not, metadata is generated from the filename.

The --type flag determines the artifact type:
  story (default), doc, epic, initiative, decision

Use --github to import from a GitHub issue:
  specflow import --github owner/repo#123
  specflow import --github https://github.com/owner/repo/issues/123`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")
			ghRef, _ := cmd.Flags().GetString("github")

			if ghRef != "" {
				return importGitHubIssue(ghRef, epicSlug)
			}

			if len(args) == 0 {
				return fmt.Errorf("either a file path or --github flag is required")
			}

			filePath := args[0]
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
	cmd.Flags().String("github", "", "GitHub issue reference (owner/repo#123 or full URL)")

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

// ghIssueURLRegexp matches GitHub issue URLs like https://github.com/owner/repo/issues/123
var ghIssueURLRegexp = regexp.MustCompile(`github\.com/([^/]+/[^/]+)/issues/(\d+)`)

// ghIssueRefRegexp matches owner/repo#123 format.
var ghIssueRefRegexp = regexp.MustCompile(`^([^#]+)#(\d+)$`)

// ghIssue holds the JSON fields we need from gh issue view.
type ghIssue struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Number int    `json:"number"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	URL string `json:"url"`
}

func importGitHubIssue(ref, epicSlug string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI is required for GitHub import but was not found in PATH.\nInstall it: https://cli.github.com")
	}

	// Normalize ref to owner/repo#number format for gh CLI.
	issueRef := normalizeGitHubRef(ref)
	if issueRef == "" {
		return fmt.Errorf("invalid GitHub issue reference %q: use owner/repo#123 or a full issue URL", ref)
	}

	// Parse repo and number for the gh command.
	parts := ghIssueRefRegexp.FindStringSubmatch(issueRef)
	if parts == nil {
		return fmt.Errorf("could not parse issue reference %q", issueRef)
	}
	repo, number := parts[1], parts[2]

	out, err := exec.Command("gh", "issue", "view", number, "--repo", repo, "--json", "title,body,number,labels,url").Output() //nolint:gosec // user-provided repo/number is intentional
	if err != nil {
		return fmt.Errorf("fetching GitHub issue: %w\nMake sure you're authenticated with: gh auth login", err)
	}

	var issue ghIssue
	if err := json.Unmarshal(out, &issue); err != nil {
		return fmt.Errorf("parsing GitHub issue response: %w", err)
	}

	// Build slug from title.
	slug := slugFromFilename(issue.Title + ".md")
	if err := models.ValidateSlug(slug); err != nil {
		// Fall back to repo-number format.
		slug = slugFromFilename(fmt.Sprintf("%s-%d.md", strings.ReplaceAll(repo, "/", "-"), issue.Number))
	}

	// Map labels.
	var labels []string
	for _, l := range issue.Labels {
		labels = append(labels, l.Name)
	}

	body := issue.Body
	if issue.URL != "" {
		body = fmt.Sprintf("Imported from: %s\n\n%s", issue.URL, body)
	}

	st := &models.Story{
		Slug:   slug,
		Title:  issue.Title,
		Epic:   epicSlug,
		Labels: labels,
		Body:   body,
	}

	if err := appStore.CreateStory(st); err != nil {
		return err
	}

	fmt.Printf("Imported GitHub issue #%d as story %q (%s)\n", issue.Number, st.Title, st.Slug)
	return nil
}

func normalizeGitHubRef(ref string) string {
	// Try URL format first.
	if m := ghIssueURLRegexp.FindStringSubmatch(ref); m != nil {
		return m[1] + "#" + m[2]
	}
	// Try owner/repo#number format.
	if ghIssueRefRegexp.MatchString(ref) {
		return ref
	}
	return ""
}
