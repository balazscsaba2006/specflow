package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/balazscsaba2006/specflow/internal/models"
	"github.com/balazscsaba2006/specflow/internal/store"
	"github.com/balazscsaba2006/specflow/internal/ui"
	"github.com/spf13/cobra"
)

// docTemplates maps doc types to their editor templates.
var docTemplates = map[string]string{
	models.DocTypePRD: `---
title: "PRD: "
type: prd
status: draft
epic: ""
open_questions: []
---
# PRD:

## Problem

What problem are we solving?

## Users

Who is affected?

## Goals

What does success look like?

## Scope

What is in scope? What is explicitly out of scope?

## Requirements

### Functional Requirements

### Non-Functional Requirements

## Risks

## What If

## Open Questions
`,
	models.DocTypeADR: `---
title: "ADR: "
type: adr
status: draft
epic: ""
open_questions: []
---
# ADR:

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?
`,
}

const docGenericTemplate = `---
title: ""
type: %s
status: draft
epic: ""
open_questions: []
---
# Title

## Overview

## Details
`

func docTemplate(docType string) string {
	if tmpl, ok := docTemplates[docType]; ok {
		return tmpl
	}
	return fmt.Sprintf(docGenericTemplate, docType)
}

func newDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doc",
		Aliases: []string{"d"},
		Short:   "Manage documents",
		Long:    "Create, list, show, and edit documents (PRDs, tech specs, ADRs, etc.).",
	}

	cmd.AddCommand(newDocNewCmd())
	cmd.AddCommand(newDocLsCmd())
	cmd.AddCommand(newDocShowCmd())
	cmd.AddCommand(newDocEditCmd())

	return cmd
}

func newDocNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Create a new document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := models.ValidateSlug(slug); err != nil {
				return err
			}

			docType, _ := cmd.Flags().GetString("type")
			if docType == "" {
				return fmt.Errorf("--type is required: must be one of [%s]", strings.Join(models.ValidDocTypes, ", "))
			}
			if !slices.Contains(models.ValidDocTypes, docType) {
				return fmt.Errorf("invalid doc type %q: must be one of [%s]", docType, strings.Join(models.ValidDocTypes, ", "))
			}

			epicSlug, _ := cmd.Flags().GetString("epic")

			tmpl := docTemplate(docType)
			if epicSlug != "" {
				tmpl = strings.Replace(tmpl, `epic: ""`, fmt.Sprintf("epic: %q", epicSlug), 1)
			}

			edited, err := openInEditor(tmpl)
			if err != nil {
				return fmt.Errorf("editing doc: %w", err)
			}

			var d models.Document
			body, err := store.Parse([]byte(edited), &d)
			if err != nil {
				return fmt.Errorf("parsing doc: %w", err)
			}
			d.Slug = slug
			d.Body = body

			if err := appStore.CreateDoc(&d); err != nil {
				return err
			}

			fmt.Printf("Created doc %q (%s)\n", d.Title, d.Slug)
			return nil
		},
	}

	cmd.Flags().String("type", "", "Document type (required): "+strings.Join(models.ValidDocTypes, ", "))
	cmd.Flags().String("epic", "", "Parent epic slug")

	return cmd
}

func newDocLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List documents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")
			typeFilter, _ := cmd.Flags().GetString("type")

			docs, err := appStore.ListDocs(epicSlug)
			if err != nil {
				return err
			}

			if typeFilter != "" {
				var filtered []*models.Document
				for _, d := range docs {
					if d.Type == typeFilter {
						filtered = append(filtered, d)
					}
				}
				docs = filtered
			}

			headers := []string{"SLUG", "TYPE", "TITLE", "STATUS", "EPIC"}
			rows := make([][]string, len(docs))
			for idx, d := range docs {
				rows[idx] = []string{d.Slug, d.Type, d.Title, ui.StatusBadge(d.Status), d.Epic}
			}

			printTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Filter by epic slug")
	cmd.Flags().String("type", "", "Filter by document type")

	return cmd
}

func newDocShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show document details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicSlug, _ := cmd.Flags().GetString("epic")

			d, err := appStore.LoadDoc(args[0], epicSlug)
			if err != nil {
				return err
			}

			fmt.Printf("%s  %s\n", ui.Label("ID:"), d.ID)
			fmt.Printf("%s  %s\n", ui.Label("Slug:"), d.Slug)
			fmt.Printf("%s  %s\n", ui.Label("Type:"), d.Type)
			fmt.Printf("%s  %s\n", ui.Label("Title:"), d.Title)
			fmt.Printf("%s  %s\n", ui.Label("Status:"), ui.StatusBadge(d.Status))
			fmt.Printf("%s  %s\n", ui.Label("Epic:"), d.Epic)
			fmt.Printf("%s  %s\n", ui.Label("Created:"), d.Created.Format("2006-01-02 15:04:05"))
			fmt.Printf("%s  %s\n", ui.Label("Updated:"), d.Updated.Format("2006-01-02 15:04:05"))

			if len(d.OpenQuestions) > 0 {
				fmt.Println("Open Questions:")
				for _, q := range d.OpenQuestions {
					fmt.Printf("  - %s\n", q)
				}
			}
			if d.Body != "" {
				fmt.Printf("\n%s\n", d.Body)
			}

			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic slug (for epic-scoped docs)")

	return cmd
}

func newDocEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <slug>",
		Short: "Edit a document in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			epicSlug, _ := cmd.Flags().GetString("epic")

			d, err := appStore.LoadDoc(slug, epicSlug)
			if err != nil {
				return err
			}

			data, err := store.Marshal(d, d.Body)
			if err != nil {
				return fmt.Errorf("marshaling doc: %w", err)
			}

			edited, err := openInEditor(string(data))
			if err != nil {
				return fmt.Errorf("editing doc: %w", err)
			}

			var updated models.Document
			body, err := store.Parse([]byte(edited), &updated)
			if err != nil {
				return fmt.Errorf("parsing doc: %w", err)
			}

			// Preserve immutable fields from the original.
			updated.ID = d.ID
			updated.Slug = d.Slug
			updated.Created = d.Created
			updated.Body = body

			if err := appStore.SaveDoc(&updated); err != nil {
				return err
			}

			fmt.Printf("Updated doc %q\n", updated.Slug)
			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic slug (for epic-scoped docs)")

	return cmd
}
