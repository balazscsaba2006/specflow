# specflow — Implementation Plan

## Table of Contents

1. [Project Vision](#1-project-vision)
2. [Background Research](#2-background-research)
3. [Architecture Overview](#3-architecture-overview)
4. [Storage Layout](#4-storage-layout)
5. [Data Models](#5-data-models)
6. [CLI Commands](#6-cli-commands)
7. [MCP Tools](#7-mcp-tools)
8. [Context Builder](#8-context-builder)
9. [Document Templates](#9-document-templates)
10. [Hard Questions System](#10-hard-questions-system)
11. [Review Prompt System](#11-review-prompt-system)
12. [Features](#12-features)
13. [Go Module Structure](#13-go-module-structure)
14. [Dependencies](#14-dependencies)
15. [Build Phases](#15-build-phases)
16. [Phase Implementation Details](#16-phase-implementation-details)

---

## 1. Project Vision

### What is specflow?

A personal **spec-driven development CLI** that acts as a structured memory and context layer for Claude Code. It sits between the human (architect/PM/tech lead) and Claude Code (the AI agent), providing:

1. **Structured artifact management** — initiatives, epics, stories, specs, decisions stored as git-friendly markdown
2. **Rich context assembly** — layered prompt building from project state for Claude Code
3. **Progress tracking** — status rollup across the full hierarchy
4. **Verification support** — plan-vs-implementation comparison data
5. **Working style enforcement** — behavioral instructions embedded in MCP tool descriptions

### Core Principle

**specflow has ZERO AI logic.** No Anthropic API key. No prompt engineering in Go code. No LLM calls. Claude Code IS the AI. specflow manages structured state and assembles context. All intelligence comes from Claude Code interacting through MCP tools.

### Architecture Pattern

```
┌──────────────────────────────────────────────────┐
│                    Human (CLI)                     │
│   specflow epic new, specflow status, etc.        │
└───────────────┬──────────────────────────────────┘
                │
                ▼
┌──────────────────────────────────────────────────┐
│              specflow (Go binary)                  │
│                                                    │
│   CLI mode          │    MCP server mode (stdio)  │
│   (human ops)       │    (Claude Code ops)        │
│                     │                              │
│   ┌─────────────────┴────────────────────────┐   │
│   │            Shared Core                    │   │
│   │  Store · Context Builder · Git · HardQ   │   │
│   └──────────────────────────────────────────┘   │
│                     │                              │
│   ┌─────────────────┴────────────────────────┐   │
│   │        Filesystem Store (.specflow/)      │   │
│   └──────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
                │ MCP (stdio)
                ▼
┌──────────────────────────────────────────────────┐
│                 Claude Code                        │
│                                                    │
│  Reads context → implements code → writes results │
└──────────────────────────────────────────────────┘
```

### The Hierarchy

```
Project (.specflow/)
  └── Initiative (optional — groups epics toward a strategic goal)
       └── Epic (optional — a shippable feature/capability)
            └── Story (the atomic work unit)
```

Everything is optional upward:
- `initiative > epic > story` — full hierarchy
- `epic > story` — no initiative
- Standalone story — no epic, no initiative

---

## 2. Background Research

### Traycer.ai — What We Took

Traycer is a VS Code extension + cloud backend that orchestrates AI coding agents with a Plan → Execute → Verify loop.

**Concepts adopted:**
- Plan → Execute → Verify lifecycle per story
- Phase decomposition (ordered groups of stories)
- Verification system (comparing implementation against plan with severity categories: Critical/Major/Minor)
- Execution tracking (git ref before/after, diff capture)
- Context carrying forward between phases (completed work informs next phase)
- YOLO-style autonomous execution (Claude Code loops through stories)

**Concepts rejected:**
- VS Code extension UI (we're CLI + MCP native)
- Multi-model orchestration (Claude Code handles model selection)
- Cloud backend / credit system (local only)
- Agent abstraction layer (we only support Claude Code)
- Team collaboration features (personal tool)

### ChatPRD.ai — What We Took

ChatPRD is an AI-powered PRD generation platform with structured templates and CPO-level document review.

**Concepts adopted:**
- Structured PRD templates with well-chosen sections (Problem, Users, Goals, Scope, Risks, Open Questions)
- "Open Questions" as a first-class concept that propagates up the hierarchy
- AI-as-reviewer coaching — challenging assumptions, finding gaps, asking hard questions
- Multiple document types (PRD, tech spec, API spec, design spec, ADR)
- "What If" list as a standard section in every PRD and tech spec
- CPO-level review prompts embedded in templates

**Concepts rejected:**
- SaaS product / team features
- Integration with Jira, Confluence, Slack, etc.
- Custom AI personas / fine-tuning
- Prototyping tool integration (v0, Lovable, etc.)

### NestERP CLAUDE.md — What We Took

The CLAUDE.md from the NestERP project defines an excellent working style contract between human and AI.

**Patterns embedded in specflow:**
- **Persona:** "Sparring partner who challenges thinking, catches blind spots, helps move fast without cutting corners"
- **Before writing code:** Discuss trade-offs, flag implications, present options
- **After writing code:** Summarize decisions, "what if" list, flag missing tests
- **Architecture decisions:** Devil's advocate by default, hard questions (10x scale, failure mode, migration path)
- **Code review:** Principal engineer — direct, not diplomatic. Focus on correctness, edge cases, naming
- **Anti-patterns:** Block accidental complexity, call out gold-plating, call out bikeshedding
- **Fast/careful mode:** Match the user's pace — skip ceremony when prototyping, slow down for production

**How these are embedded:**
1. specflow's own CLAUDE.md — adapted for Go project conventions
2. MCP tool descriptions — behavioral instructions that tell Claude Code HOW to interact when using each tool
3. Review prompt templates — coaching prompts that embed the working style
4. Hard questions system — deterministic question templates triggered by entity type

---

## 3. Architecture Overview

### Two Modes, One Binary

```bash
specflow [command]     # CLI mode — human creates/manages artifacts
specflow mcp           # MCP mode — Claude Code reads/writes via stdio
```

Both modes share the same core:
- `store/` — filesystem CRUD
- `context/` — context assembly
- `git/` — git operations
- `hardq/` — hard questions templates

### Data Flow

**Human → CLI:**
```
Human types: specflow epic new "Purchase Orders"
  → CLI creates .specflow/epics/purchase-orders/epic.md
  → CLI opens $EDITOR for human to write description
  → Human saves and closes editor
  → CLI confirms creation
```

**Claude Code → MCP:**
```
Human tells Claude: "What should I work on next?"
  → Claude calls sf_story_next()
  → specflow reads all stories, checks blocked_by, returns next
  → Claude calls sf_context_build("s_003")
  → specflow assembles: CLAUDE.md + epic context + specs + plan + file refs
  → Claude reads the context, implements the code
  → Claude calls sf_execution_start("s_003")
  → specflow records git ref baseline
  → Claude finishes implementation
  → Claude calls sf_execution_complete("x_01...")
  → specflow captures git ref after + diff
  → Claude calls sf_verify_save("s_003", findings)
  → specflow stores verification
  → Claude calls sf_story_update("s_003", status="done")
```

---

## 4. Storage Layout

```
.specflow/
├── config.yaml                              # Project configuration
├── log.jsonl                                # Activity log (append-only)
├── templates/                               # User-customizable templates (override defaults)
│   ├── prd.md.tmpl
│   ├── tech-spec.md.tmpl
│   ├── api-spec.md.tmpl
│   ├── design-spec.md.tmpl
│   ├── adr.md.tmpl
│   ├── one-pager.md.tmpl
│   ├── story.md.tmpl
│   ├── epic.md.tmpl
│   ├── initiative.md.tmpl
│   ├── review-prd.md.tmpl                  # Review/coaching prompt templates
│   ├── review-tech-spec.md.tmpl
│   ├── review-story.md.tmpl
│   └── decompose.md.tmpl
├── initiatives/
│   └── {slug}/
│       └── initiative.md                    # Frontmatter + description
├── epics/
│   └── {slug}/
│       ├── epic.md                          # Frontmatter + description
│       ├── docs/                            # Specs/PRDs/ADRs scoped to this epic
│       │   ├── prd.md
│       │   ├── tech-spec.md
│       │   └── adr-001-some-decision.md
│       └── stories/
│           ├── 001-create-model.md
│           ├── 002-create-repo.md
│           └── 003-create-actions.md
├── stories/                                 # Standalone stories (no epic)
│   ├── fix-timezone-bug.md
│   └── upgrade-dependency.md
├── docs/                                    # Project-level docs (no epic scope)
│   ├── adr-001-multi-tenancy.md
│   └── api-spec-v1.md
├── decisions/                               # Lightweight project-level decision log
│   ├── 001-use-go-for-cli.md
│   └── 002-filesystem-over-sqlite.md
└── executions/                              # Flat, referenced by story ID
    └── {story-id}/
        └── {exec-id}/
            ├── plan.md                      # Implementation plan
            ├── verification.md              # Verification findings
            └── meta.yaml                    # Git refs, timestamps, status
```

### Key Design Decisions

1. **Epics own their stories and docs** — colocated in the epic directory for easy browsing
2. **Standalone stories** live in a top-level `stories/` directory
3. **Executions are flat** — indexed by story ID, not nested under epics (a story might move between epics)
4. **Log is append-only JSONL** — one JSON object per line, easy to parse, grep, tail
5. **Templates are overridable** — user copies from defaults to `.specflow/templates/` to customize

---

## 5. Data Models

### 5.1 Initiative

```yaml
---
id: i_01JMXYZ123456
slug: nesterp-go-live
title: "NestERP Go-Live Readiness"
status: active                    # active | completed | on_hold | archived
created: 2026-02-17T10:00:00Z
updated: 2026-02-17T10:00:00Z
epics:                            # Ordered list of epic slugs
  - inventory-purchase-orders
  - e2e-tests
  - docs-polish
goal: "Production-ready NestERP with all P0 features complete"
success_criteria:
  - "All 15 modules with admin UI"
  - "E2E tests for critical paths"
  - "Documentation complete"
open_questions:
  - "Do we need load testing before go-live?"
  - "What's the rollback strategy for first tenant?"
---
# NestERP Go-Live Readiness

Strategic initiative to bring NestERP to production readiness.
This encompasses all remaining P0 work across modules...
```

**Status transitions:** `active` → `completed` | `on_hold` | `archived`

### 5.2 Epic

```yaml
---
id: e_01JMXYZ234567
slug: inventory-purchase-orders
title: "Inventory Module — Purchase Orders"
status: active                    # draft | active | completed | on_hold | archived
initiative: nesterp-go-live       # Optional — slug of parent initiative (null if standalone)
created: 2026-02-17T10:00:00Z
updated: 2026-02-17T10:00:00Z
phases:                           # Ordered list of phases, each with story IDs
  - label: "Models & Migrations"
    stories: [001-create-po-model, 002-create-poi-model]
  - label: "Actions & Repositories"
    stories: [003-po-repository, 004-po-actions]
  - label: "Livewire UI"
    stories: [005-po-list-create, 006-po-detail-flow, 007-goods-receipt-ui]
open_questions: []
decisions:
  - "Use PurchaseOrderStatus enum, not string column (2026-02-17)"
---
# Purchase Order Management

Full purchase order lifecycle for the Inventory module:
supplier management, PO creation, approval workflow, goods receiving...
```

**Status transitions:** `draft` → `active` → `completed` | `on_hold` | `archived`

### 5.3 Story

```yaml
---
id: s_01JMXYZ345678
slug: 001-create-po-model
title: "Create PurchaseOrder Model & Migration"
status: planned                   # draft | planned | in_progress | verifying | done | blocked
priority: high                    # critical | high | medium | low
epic: inventory-purchase-orders   # Optional — slug of parent epic (null if standalone)
blocked_by: []                    # Story slugs (not IDs) for readability
labels: [backend, model]
acceptance:
  - "PurchaseOrder model with HasUuids, SoftDeletes, BelongsToTenant"
  - "PurchaseOrderStatus enum (draft/submitted/approved/ordered/received/closed/cancelled)"
  - "Migration with all required columns"
  - "Factory generates valid test data"
  - "Repository interface + Eloquent implementation"
  - "Repository feature tests pass"
doc_refs: [prd, tech-spec]        # Slugs of docs in the epic's docs/ directory
open_questions: []
assumptions: []                   # Populated during/after execution
created: 2026-02-17T10:00:00Z
updated: 2026-02-17T10:00:00Z
---
# Create PurchaseOrder Model & Migration

Implement the core PurchaseOrder model following the existing NestERP patterns.
Reference AwbRequest model for similar structure...
```

**Status transitions:** `draft` → `planned` → `in_progress` → `verifying` → `done`
**Also:** Any status → `blocked` (when blocked_by is non-empty and blocker is not done)

### 5.4 Document (PRD / Tech Spec / API Spec / Design Spec / ADR / One-Pager)

```yaml
---
id: d_01JMXYZ456789
slug: prd
type: prd                         # prd | tech-spec | api-spec | design-spec | adr | one-pager
title: "PRD: Purchase Order Management"
status: approved                  # draft | review | approved | superseded
epic: inventory-purchase-orders   # Optional — null if project-level doc
created: 2026-02-17T10:00:00Z
updated: 2026-02-17T10:00:00Z
open_questions:
  - "Should POs support multi-currency?"
  - "What approval threshold requires manager sign-off?"
---
# PRD: Purchase Order Management

## Problem Statement
Currently, inventory replenishment is managed outside the system...

## Target Users
- Warehouse managers placing orders with suppliers
- Finance team approving purchase orders above threshold

## Goals & Success Metrics
| Goal | Metric | Target |
|------|--------|--------|
| Reduce manual PO creation time | Time to create PO | < 2 min |
| Improve receiving accuracy | Discrepancy rate | < 1% |

## Scope
### In Scope
- PO creation, approval workflow, receiving
- Supplier management
### Out of Scope
- Automated reorder points (future epic)
- Supplier portal

## User Stories
- As a warehouse manager, I want to create a PO from a supplier catalog...
- As a finance manager, I want to approve POs above EUR 5000...

## Requirements
### Functional
- ...
### Non-Functional
- PO search < 200ms for 10K records
- Audit trail for all status changes

## Constraints
- Must follow existing Action + Repository pattern
- PHPStan Level 8 clean
- HasUuids, SoftDeletes, BelongsToTenant on all models

## Risks & Mitigations
| Risk | Impact | Mitigation |
|------|--------|------------|
| Complex approval chains | Delays implementation | Start with single-level approval |

## What If (Fragility List)
- What if multi-currency is needed later? → Currency column is nullable, default EUR
- What if approval chains become multi-level? → Status enum supports it, approval logic is in one Action
- What if POs need to link to invoices? → foreign key is ready, Finance integration is separate epic

## Open Questions
- Should POs support multi-currency?
- What approval threshold requires manager sign-off?

## Dependencies
- Catalog module (products referenced by PO items)
- Finance module (invoice linking, post go-live)
```

### 5.5 Decision (Lightweight ADR)

```yaml
---
id: dec_01JMXYZ567890
slug: 001-use-go-for-cli
date: 2026-02-17
title: "Use Go for specflow CLI"
status: accepted                  # proposed | accepted | superseded | deprecated
context_refs: []                  # Epic/doc slugs this decision relates to
---
## Context
Need a language for the specflow CLI tool. Options: Go, Rust, TypeScript/Node, Python.

## Decision
Use Go. Single static binary, fast compilation, excellent stdlib for CLI/filesystem work,
good MCP SDK available (mark3labs/mcp-go). Cross-platform without CGO.

## Consequences
- Fast binary startup (<10ms)
- Easy cross-platform distribution
- No runtime dependencies
- Team (me) already proficient in Go
- Slightly more verbose than Python for string manipulation
```

### 5.6 Plan (generated by Claude Code, stored by specflow)

```yaml
---
id: p_01JMXYZ678901
story: 001-create-po-model        # Story slug
status: approved                  # draft | approved | executing | verified
created: 2026-02-17T10:30:00Z
git_ref_baseline: abc1234def5678  # Commit SHA when plan was created/approved
estimated_files: 6
---
# Implementation Plan for: Create PurchaseOrder Model & Migration

## Step 1: Create PurchaseOrderStatus Enum
**File:** `modules/Inventory/app/Enums/PurchaseOrderStatus.php`
**Action:** create
**Pattern reference:** `modules/Connectors/app/Enums/CourierType.php`
- String-backed enum with cases: draft, submitted, approved, ordered, partially_received, received, closed, cancelled
- Add label() method returning human-readable names
- Add color() method for UI badge colors

## Step 2: Create PurchaseOrder Model
**File:** `modules/Inventory/app/Models/PurchaseOrder.php`
**Action:** create
**Pattern reference:** `modules/Connectors/app/Models/AwbRequest.php`
- Use HasUuids, HasFactory, SoftDeletes, BelongsToTenant traits
- Fillable: supplier_id, po_number, status, expected_delivery_date, notes, total_amount, currency
- Casts: status → PurchaseOrderStatus, expected_delivery_date → date, total_amount → decimal:2
- Relations: belongsTo(Supplier), hasMany(PurchaseOrderItem)

## Step 3: Create Migration
**File:** `modules/Inventory/database/migrations/YYYY_MM_DD_HHMMSS_create_purchase_orders_table.php`
**Action:** create (via artisan)
- Columns: id (uuid), tenant_id, supplier_id (foreign), po_number (unique per tenant), status, expected_delivery_date, notes, total_amount, currency, timestamps, soft_deletes
- Indexes: tenant_id + status composite, tenant_id + po_number unique

## Step 4: Create Factory
**File:** `modules/Inventory/database/factories/PurchaseOrderFactory.php`
**Action:** create
- Generate valid test data for all fields
- Define states for each PurchaseOrderStatus

## Step 5: Create Repository Interface
**File:** `modules/Inventory/app/Contracts/PurchaseOrderRepositoryInterface.php`
**Action:** create
- Methods: create, update, find, findByPoNumber, paginate, findByStatus

## Step 6: Create Eloquent Repository
**File:** `modules/Inventory/app/Repositories/EloquentPurchaseOrderRepository.php`
**Action:** create
- Implement all interface methods
- Register binding in InventoryServiceProvider

## Step 7: Create Repository Tests
**File:** `modules/Inventory/tests/Feature/Repositories/PurchaseOrderRepositoryTest.php`
**Action:** create
- Test each repository method with real database
- Use RefreshDatabase trait
```

### 5.7 Execution Metadata

```yaml
# .specflow/executions/{story-slug}/{exec-id}/meta.yaml
id: x_01JMXYZ789012
story: 001-create-po-model
plan: p_01JMXYZ678901
status: completed                 # started | completed | failed
started_at: 2026-02-17T10:35:00Z
completed_at: 2026-02-17T10:50:00Z
git_ref_before: abc1234def5678
git_ref_after: 9876fedc5432ba10
files_changed:
  - path: modules/Inventory/app/Enums/PurchaseOrderStatus.php
    action: added
  - path: modules/Inventory/app/Models/PurchaseOrder.php
    action: added
  - path: modules/Inventory/database/migrations/2026_02_17_103500_create_purchase_orders_table.php
    action: added
  - path: modules/Inventory/database/factories/PurchaseOrderFactory.php
    action: added
  - path: modules/Inventory/app/Contracts/PurchaseOrderRepositoryInterface.php
    action: added
  - path: modules/Inventory/app/Repositories/EloquentPurchaseOrderRepository.php
    action: added
```

### 5.8 Verification

```yaml
---
id: v_01JMXYZ890123
execution: x_01JMXYZ789012
story: 001-create-po-model
result: partial                   # pass | fail | partial
created: 2026-02-17T10:55:00Z
stats:
  critical: 0
  major: 1
  minor: 2
findings:
  - severity: major
    category: missing
    file: ~
    description: "Repository feature test not created — acceptance criteria requires it"
    suggestion: "Create modules/Inventory/tests/Feature/Repositories/PurchaseOrderRepositoryTest.php following existing patterns"
  - severity: minor
    category: clarity
    file: modules/Inventory/app/Enums/PurchaseOrderStatus.php
    description: "Enum missing color() method mentioned in plan step 1"
    suggestion: "Add color() method returning Tailwind badge colors per status"
  - severity: minor
    category: quality
    file: modules/Inventory/app/Models/PurchaseOrder.php
    description: "total_amount cast as decimal:2 but migration uses decimal(12,2) — verify precision matches"
    suggestion: "Both are consistent, this is fine. Marking as minor for awareness."
---
# Verification Report

## Summary
Implementation is mostly complete. 5 of 6 acceptance criteria met.
Missing: Repository feature tests (major gap).

## Acceptance Criteria Check
- [x] PurchaseOrder model with HasUuids, SoftDeletes, BelongsToTenant
- [x] PurchaseOrderStatus enum with all cases
- [x] Migration with required columns
- [x] Factory with valid test data
- [x] Repository interface + Eloquent implementation
- [ ] Repository feature tests — NOT CREATED

## Assumptions Recorded
- Currency defaults to EUR (multi-currency deferred)
- PO number format: PO-{YYYYMMDD}-{sequence}
```

### 5.9 Activity Log Entry

```json
{"ts":"2026-02-17T10:35:00Z","type":"story.status_changed","entity":"s_001","from":"planned","to":"in_progress","epic":"inventory-purchase-orders"}
{"ts":"2026-02-17T10:35:01Z","type":"execution.started","entity":"x_01JMXYZ789012","story":"s_001","git_ref":"abc1234"}
{"ts":"2026-02-17T10:50:00Z","type":"execution.completed","entity":"x_01JMXYZ789012","story":"s_001","git_ref":"9876fedc","files_changed":6}
{"ts":"2026-02-17T10:55:00Z","type":"verification.saved","entity":"v_01JMXYZ890123","story":"s_001","result":"partial","critical":0,"major":1,"minor":2}
{"ts":"2026-02-17T11:00:00Z","type":"story.status_changed","entity":"s_001","from":"in_progress","to":"done","epic":"inventory-purchase-orders"}
```

---

## 6. CLI Commands

### 6.1 Setup

```bash
specflow init                                     # Create .specflow/ in current directory
specflow init --with-claude                       # Also add MCP config to .claude/settings.json
specflow config set <key> <value>                 # Set config value
specflow config get <key>                         # Get config value
specflow config ls                                # List all config
```

`specflow init --with-claude` adds this to `.claude/settings.json`:
```json
{
  "mcpServers": {
    "specflow": {
      "command": "specflow",
      "args": ["mcp"]
    }
  }
}
```

### 6.2 Initiative Management

```bash
specflow initiative new <slug>                    # Create + open in $EDITOR
specflow initiative ls                            # Table: slug | status | epics | progress
specflow initiative show <slug>                   # Full detail with epic breakdown
specflow initiative edit <slug>                   # Open in $EDITOR
specflow initiative set <slug> status <status>    # Update status
specflow initiative archive <slug>                # Set status to archived
```

### 6.3 Epic Management

```bash
specflow epic new <slug> [--initiative=<slug>]    # Create + open in $EDITOR
specflow epic ls [--initiative=<slug>]            # Table with filters
specflow epic show <slug>                         # Full detail with phase/story breakdown
specflow epic edit <slug>                         # Open in $EDITOR
specflow epic set <slug> <field> <value>          # Update field
specflow epic archive <slug>                      # Set status to archived
```

### 6.4 Story Management

```bash
specflow story new <slug> [--epic=<slug>]         # Create + open in $EDITOR
specflow story ls [--epic=<slug>] [--status=X] [--label=X] [--blocked]  # Table with filters
specflow story show <slug>                        # Full detail with acceptance criteria, plan, verification
specflow story edit <slug>                        # Open in $EDITOR
specflow story set <slug> <field> <value>         # Update field (status, priority, labels, etc.)
specflow story next [--epic=<slug>]               # Next actionable story (smart ordering)
```

### 6.5 Document Management

```bash
specflow doc new <slug> --type=<type> [--epic=<slug>]  # Create from template + open in $EDITOR
specflow doc ls [--epic=<slug>] [--type=<type>]        # List documents
specflow doc show <slug> [--epic=<slug>]               # Render to terminal
specflow doc edit <slug> [--epic=<slug>]               # Open in $EDITOR
```

Document types: `prd`, `tech-spec`, `api-spec`, `design-spec`, `adr`, `one-pager`

### 6.6 Decision Management

```bash
specflow decision new <slug>                      # Create + open in $EDITOR
specflow decision ls                              # List all decisions
specflow decision show <slug>                     # Render to terminal
```

### 6.7 Status & Reporting

```bash
specflow status                                   # Full project status rollup
specflow status <slug>                            # Status for specific initiative/epic/story (auto-detect type)
specflow questions                                # All open questions grouped by source
specflow questions [--initiative=X] [--epic=X]    # Scoped open questions
specflow blocked                                  # All blocked stories with their blockers
specflow next [--epic=<slug>]                     # Alias for: specflow story next
specflow assumptions [--epic=<slug>]              # All recorded assumptions
specflow log [--last=N]                           # Recent activity timeline
specflow scope-check <epic-slug>                  # Compare stories against PRD scope
specflow diff-check <epic-slug>                   # Detect spec-vs-story drift
```

### 6.8 Context & Review (read-only, mainly for debugging)

```bash
specflow context <story-slug>                     # Preview assembled context
specflow context --export <story-slug>            # Export to stdout for manual use
```

### 6.9 Utility

```bash
specflow search <query>                           # Full-text search across all artifacts
specflow import <file> [--epic=<slug>]            # Import existing markdown as artifact
specflow mode [fast|careful]                      # Toggle or show current mode
```

### 6.10 MCP Server

```bash
specflow mcp                                      # Start MCP server on stdio
```

---

## 7. MCP Tools

### 7.1 Read Tools

#### `sf_status`
```
description: |
  Returns project-wide status rollup. Shows all initiatives with epic progress,
  standalone epics, standalone stories, counts of blocked items and open questions.
  Use this when the user asks "what's the status?" or "where are we?"

input:
  scope: string (optional) — initiative slug, epic slug, or story slug to narrow scope

returns: Formatted markdown with progress bars and story counts per status
```

#### `sf_initiative_show`
```
description: |
  Returns full initiative detail with all epics and their progress.

input:
  slug: string (required)

returns: Initiative metadata + epic breakdown with % complete
```

#### `sf_epic_show`
```
description: |
  Returns full epic detail with phase map, story statuses, linked docs, and open questions.

input:
  slug: string (required)

returns: Epic metadata + phase breakdown with story statuses
```

#### `sf_story_show`
```
description: |
  Returns full story detail including acceptance criteria, linked doc refs,
  current plan, latest verification, and recorded assumptions.

input:
  slug: string (required)

returns: Complete story with all metadata and linked artifacts
```

#### `sf_story_next`
```
description: |
  Returns the next recommended story to work on. Considers: status (planned first),
  phase order (earlier phases first), priority (critical > high > medium > low),
  blocked_by resolution (only unblocked stories), and currently in-progress count.

input:
  epic: string (optional) — scope to specific epic

returns: Story summary with context on why it's recommended next
```

#### `sf_story_ls`
```
description: |
  Lists stories with optional filters. Returns compact table format.

input:
  epic: string (optional)
  status: string (optional)
  label: string (optional)
  blocked: boolean (optional) — if true, only show blocked stories

returns: Table of stories with slug, title, status, priority, epic, labels
```

#### `sf_doc_read`
```
description: |
  Reads a document (PRD, tech spec, ADR, etc.) by slug. If the doc is scoped
  to an epic, provide the epic slug as well.

input:
  slug: string (required)
  epic: string (optional) — required for epic-scoped docs

returns: Full document content (frontmatter + body)
```

#### `sf_plan_read`
```
description: |
  Reads the current implementation plan for a story. Returns null if no plan exists.

input:
  story: string (required) — story slug

returns: Plan content or indication that no plan exists
```

#### `sf_verify_read`
```
description: |
  Reads the latest verification for a story.

input:
  story: string (required) — story slug

returns: Verification findings or indication that no verification exists
```

#### `sf_context_build`
```
description: |
  THE CORE TOOL. Builds full, layered execution context for a story.
  Returns everything Claude Code needs to implement the story:
  project conventions, epic context, specs, plan, referenced files, open questions.

  ALWAYS call this before implementing a story. The context includes decisions
  already made — don't re-litigate them. It also includes open questions that
  might affect implementation — surface these to the user.

input:
  story: string (required) — story slug

returns: Multi-section markdown with full assembled context (see Context Builder section)
```

#### `sf_questions`
```
description: |
  Returns all open questions across the project, grouped by source entity.
  Use this to surface unresolved decisions before starting work.

input:
  initiative: string (optional)
  epic: string (optional)

returns: Grouped list of open questions with source entity references
```

#### `sf_blocked`
```
description: |
  Returns all blocked stories with their blockers and blocker statuses.

returns: Table of blocked stories and what's blocking them
```

#### `sf_decisions`
```
description: |
  Returns the decision log, optionally scoped.

input:
  epic: string (optional)

returns: List of decisions with date, title, status
```

#### `sf_log`
```
description: |
  Returns recent activity timeline from the append-only log.

input:
  last: number (optional, default 20) — number of entries

returns: Formatted activity timeline
```

#### `sf_diff`
```
description: |
  Returns git diff for a story's execution. Defaults to diff between
  execution start and current HEAD.

input:
  story: string (optional) — story slug (uses latest execution's baseline)
  refs: string (optional) — explicit git ref range (e.g., "abc123..HEAD")

returns: Git diff content
```

#### `sf_assumptions`
```
description: |
  Returns all recorded assumptions across the project, optionally scoped.

input:
  epic: string (optional)
  story: string (optional)

returns: Grouped list of assumptions with source stories
```

#### `sf_scope_check`
```
description: |
  Compares current stories against the PRD's scope definition.
  Flags stories that aren't traceable to PRD user stories or are
  explicitly marked as out-of-scope in the PRD.

input:
  epic: string (required)

returns: Scope comparison with flagged stories
```

#### `sf_diff_check`
```
description: |
  Detects drift between specs and their stories. Checks if spec
  documents were updated more recently than their referencing stories,
  indicating potential drift.

input:
  epic: string (required)

returns: List of potentially drifted stories with changed spec sections
```

#### `sf_hard_questions`
```
description: |
  Returns contextual hard questions for any entity based on its type.
  These are deterministic template-based questions, not AI-generated.
  Use these to challenge thinking before finalizing an artifact.

input:
  entity: string (required) — any entity slug (initiative, epic, story, doc)

returns: List of hard questions relevant to the entity type
```

#### `sf_review_prompt`
```
description: |
  Assembles a coaching/review prompt for a document. Returns a structured
  prompt that tells you (Claude Code) how to review the document — what to
  focus on, what to challenge, what tone to use.

input:
  doc: string (required) — document slug
  epic: string (optional) — for epic-scoped docs

returns: Review prompt with document content embedded
```

### 7.2 Write Tools

#### `sf_initiative_create`
```
description: |
  Creates a new initiative. Before creating, challenge the scope:
  - Is this actually one initiative or multiple?
  - What's the success criteria? If you can't measure it, push back.
  - What happens if this takes 3x longer than expected?
  - What's the minimum viable version?
  - What dependencies does this create across the project?

  Ask these questions conversationally before writing the initiative.
  Be direct, not diplomatic. If the scope is too vague, say so.

input:
  slug: string (required)
  title: string (required)
  goal: string (required)
  success_criteria: string[] (required)
  body: string (optional) — markdown description
  open_questions: string[] (optional)
```

#### `sf_epic_create`
```
description: |
  Creates a new epic within an optional initiative. Before creating:
  - Does this epic have a clear boundary? Where does it end?
  - What's the failure mode if we ship half of this?
  - Is this over-engineered for current needs or under-engineered for where we're heading?
  - Flag architectural implications the user might be overlooking
  - If there are multiple valid approaches, present options with trade-offs

  An epic should be shippable independently. If it's not, it might need
  to be split or reconsidered.

input:
  slug: string (required)
  title: string (required)
  initiative: string (optional) — parent initiative slug
  phases: Phase[] (optional) — initial phase map
  body: string (optional) — markdown description
  open_questions: string[] (optional)
```

#### `sf_story_create`
```
description: |
  Creates a story (atomic work unit). Before creating:
  - Is this actually one story or should it be split?
  - Are the acceptance criteria specific and testable?
  - Does this story have clear "done" criteria?
  - If the acceptance criteria need a paragraph to explain, the story is too big.

  Stories can be standalone, under an epic, or under an initiative>epic.
  Require: title, acceptance criteria. Everything else is optional.

input:
  slug: string (required)
  title: string (required)
  epic: string (optional) — parent epic slug
  priority: string (optional, default "medium")
  acceptance: string[] (required) — list of acceptance criteria
  labels: string[] (optional)
  blocked_by: string[] (optional) — story slugs
  doc_refs: string[] (optional) — doc slugs referenced by this story
  body: string (optional) — markdown description
  open_questions: string[] (optional)
```

#### `sf_story_update`
```
description: |
  Updates story fields. Most commonly used to update status.
  Validates state transitions (e.g., can't go from draft to done directly).

input:
  slug: string (required)
  status: string (optional)
  priority: string (optional)
  labels: string[] (optional) — replaces existing labels
  blocked_by: string[] (optional) — replaces existing blocked_by
  assumptions: string[] (optional) — APPENDS to existing assumptions
  open_questions: string[] (optional) — APPENDS to existing
```

#### `sf_doc_write`
```
description: |
  Creates or updates a document. When creating a PRD:
  - Start with the PROBLEM, not the solution. Push back if the user jumps to solutions.
  - Success metrics must be measurable. "Improve UX" is not a metric.
  - The "What If" section is mandatory — what breaks if requirements change?
  - Open questions are a feature, not a bug. Capture what you don't know.
  - Challenge scope: is this MVP or gold-plating?
  - Flag risks the user hasn't mentioned.

  When creating a tech spec:
  - Ask the hard questions: What at 10x scale? Failure mode? Migration path?
  - Flag when a pattern choice has non-obvious downstream consequences.
  - Constraints section must be explicit — implicit constraints cause drift.

  Act as a CPO/principal engineer reviewing the document. Be direct about gaps.

input:
  slug: string (required)
  type: string (required) — prd | tech-spec | api-spec | design-spec | adr | one-pager
  title: string (required)
  epic: string (optional) — parent epic slug
  body: string (required) — full markdown content
  open_questions: string[] (optional)
```

#### `sf_decision_record`
```
description: |
  Records a decision. Use this when a choice has been made during planning
  or implementation that future work should know about. Keep decisions concise.

input:
  slug: string (required)
  title: string (required)
  context: string (required) — why this decision was needed
  decision: string (required) — what was decided
  consequences: string (required) — what follows from this decision
  context_refs: string[] (optional) — related epic/doc slugs
```

#### `sf_plan_save`
```
description: |
  Saves an implementation plan for a story. Captures current git ref as baseline.
  Plans should include file-level detail: which files to create/modify,
  what pattern to follow, and reference files for each step.

input:
  story: string (required) — story slug
  content: string (required) — full plan markdown
  status: string (optional, default "draft") — draft | approved
```

#### `sf_execution_start`
```
description: |
  Starts execution tracking for a story. Records current git ref as "before".
  Call this BEFORE starting to implement a story. Also sets story status
  to in_progress if it isn't already.

input:
  story: string (required) — story slug

returns: execution ID for use with sf_execution_complete
```

#### `sf_execution_complete`
```
description: |
  Completes execution tracking. Records current git ref as "after",
  captures the diff, lists changed files. Call this AFTER finishing
  implementation (code written, tests run).

input:
  execution_id: string (required)

returns: Summary of changes (files changed, diff stats)
```

#### `sf_verify_save`
```
description: |
  Saves verification results after comparing implementation against plan.
  Verification should check:
  - Were all acceptance criteria met?
  - Were all planned files actually touched?
  - Were any unexpected files modified?
  - Are there assumptions baked in that should be documented?
  - What will break if requirements change? (the "what if" list)

  Be a principal engineer — direct, not diplomatic. If something is
  fine but the user would regret it in 6 months, flag it now.

  Category values: missing | bug | performance | security | clarity | quality

input:
  story: string (required)
  result: string (required) — pass | fail | partial
  findings: Finding[] (required) — array of {severity, category, file, description, suggestion}
  summary: string (required) — brief overall summary
  acceptance_check: AcceptanceCheck[] (optional) — array of {criteria, met (bool)}
  assumptions: string[] (optional) — assumptions discovered during verification
```

#### `sf_question_resolve`
```
description: |
  Marks an open question as resolved with an answer. Removes it from the
  open_questions list and records the resolution in the activity log.

input:
  entity: string (required) — entity slug containing the question
  question: string (required) — the question text (must match exactly)
  answer: string (required) — the resolution/answer
```

---

## 8. Context Builder

The context builder is the core value proposition of specflow. When Claude Code calls `sf_context_build`, it assembles a multi-layered context document:

### Layer Architecture

```
┌─────────────────────────────────────────────────────┐
│ LAYER 1: Project Conventions                        │
│  ├─ CLAUDE.md from the consuming project            │
│  ├─ AGENTS.md if it exists                          │
│  └─ .specflow/config.yaml project-specific rules    │
├─────────────────────────────────────────────────────┤
│ LAYER 2: Initiative/Epic Context (awareness)        │
│  ├─ Initiative goal + success criteria (if exists)  │
│  ├─ Epic description + phase map                    │
│  ├─ Completed stories (title + summary only)        │
│  ├─ In-progress stories (title + what's happening)  │
│  └─ Decisions made so far                           │
├─────────────────────────────────────────────────────┤
│ LAYER 3: Spec Requirements                          │
│  ├─ All docs referenced by this story (full content)│
│  └─ Acceptance criteria extracted and highlighted   │
├─────────────────────────────────────────────────────┤
│ LAYER 4: Implementation Plan                        │
│  ├─ Approved plan with file-level detail            │
│  └─ "No plan yet" prompt if plan doesn't exist      │
├─────────────────────────────────────────────────────┤
│ LAYER 5: Referenced Files                           │
│  ├─ Files mentioned in plan (current content)       │
│  ├─ Pattern exemplars from config                   │
│  └─ Files created by completed predecessor stories  │
├─────────────────────────────────────────────────────┤
│ LAYER 6: Open Items                                 │
│  ├─ Open questions that might affect implementation │
│  ├─ Assumptions from related stories                │
│  └─ Blockers (should be empty if story is ready)    │
└─────────────────────────────────────────────────────┘
```

### Context Assembly Algorithm

```
function BuildContext(storySlug):
  story = store.LoadStory(storySlug)

  // Layer 1: Project conventions
  claudeMd = readFileIfExists("CLAUDE.md")
  agentsMd = readFileIfExists("AGENTS.md")
  config = store.LoadConfig()

  // Layer 2: Epic/initiative context
  epic = store.LoadEpic(story.epic) if story.epic
  initiative = store.LoadInitiative(epic.initiative) if epic?.initiative
  siblingStories = store.LoadStoriesForEpic(story.epic)
  completedStories = filter(siblingStories, status == "done")
  decisions = store.LoadDecisions(epic.slug) if epic

  // Layer 3: Specs
  docs = [store.LoadDoc(ref, story.epic) for ref in story.doc_refs]

  // Layer 4: Plan
  plan = store.LoadPlan(story.slug)

  // Layer 5: Referenced files
  referencedFiles = extractFileRefs(plan) + config.pattern_exemplars
  fileContents = [readFileIfExists(path) for path in referencedFiles]

  // Layer 6: Open items
  allQuestions = collectOpenQuestions(story, docs, epic, initiative)
  allAssumptions = collectAssumptions(completedStories)

  // Assemble
  return renderTemplate("context.md.tmpl", all_of_the_above)
```

### File Reference Resolution

The context builder reads files mentioned in:
1. Plan step `file` or `pattern_reference` fields
2. `config.yaml` `pattern_exemplars` list (global patterns to always include)
3. Story `doc_refs` that reference project files

Files that don't exist are noted as "planned but not yet created."

### Config: Pattern Exemplars

```yaml
# .specflow/config.yaml
pattern_exemplars:
  repository: "modules/Connectors/app/Contracts/CourierConnectionRepositoryInterface.php"
  action: "modules/Warehouse/app/Actions/GenerateAWBAction.php"
  model: "modules/Connectors/app/Models/AwbRequest.php"
  livewire: "modules/Connectors/app/Http/Livewire/Couriers/CourierSetup.php"
  factory: "modules/Connectors/database/factories/AwbRequestFactory.php"
  feature_test: "modules/Connectors/tests/Feature/Repositories/CourierConnectionRepositoryTest.php"
  unit_test: "modules/Connectors/tests/Unit/Actions/GenerateAWBActionTest.php"
```

These are project-specific exemplar files that the context builder always includes when relevant.

---

## 9. Document Templates

### 9.1 PRD Template

```markdown
# PRD: {{title}}

## Problem Statement
What problem are we solving? Who is affected? What evidence do we have?

## Target Users
Who benefits from this? List user types and their relationship to the problem.

## Goals & Success Metrics
| Goal | Metric | Target |
|------|--------|--------|
| | | |

## Scope
### In Scope
- ...

### Out of Scope
- ...

## User Stories
- As a [user type], I want [action], so that [benefit]

## Requirements
### Functional
- ...

### Non-Functional
- ...

## Constraints
Technical, business, or regulatory constraints that bound the solution.

## Risks & Mitigations
| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| | | | |

## What If (Fragility List)
What breaks if requirements change? What are we betting on that might not hold?
- What if X changes? → Impact: ...
- What if Y is needed later? → Impact: ...

## Open Questions
Things we don't know yet that could affect the solution.
- ?

## Dependencies
What must exist before or alongside this work?
- ...
```

### 9.2 Tech Spec Template

```markdown
# Tech Spec: {{title}}

## Context
Why this change? What's the current state and why is it insufficient?

## Architecture
How does this fit into the existing system? What modules/components are affected?

## Data Model
Schema changes, new models, relationship changes.

## API Changes
New or modified endpoints, contracts, breaking changes.

## Implementation Approach
High-level approach and key design decisions.

## Constraints
- ...

## Testing Strategy
What tests are needed? Unit, integration, E2E?

## Rollback Plan
How do we undo this if it goes wrong?

## What If (Fragility List)
- What if X changes? → ...
- What if this needs to scale 10x? → ...

## Open Questions
- ?

## Dependencies
- ...
```

### 9.3 ADR Template

```markdown
## Context
What is the issue that we're seeing that is motivating this decision?

## Decision
What is the change that we're proposing and/or doing?

## Consequences
What becomes easier or more difficult to do because of this change?
```

### 9.4 Story Template

```markdown
# {{title}}

Implementation details, context, and notes for this story.
Reference existing code patterns and document any non-obvious decisions.
```

### 9.5 One-Pager Template

```markdown
# {{title}}

## Problem
One paragraph describing the problem.

## Proposed Solution
One paragraph describing the approach.

## Key Metrics
How we'll know this worked.

## Timeline
High-level timeline.

## Open Questions
- ?
```

---

## 10. Hard Questions System

Hard questions are deterministic, template-based questions generated for each entity type. No AI needed — they're static templates that apply universally. The `sf_hard_questions` MCP tool returns them.

### Initiative Hard Questions
```
- What's the minimum viable version of this initiative?
- What happens if this takes 3x longer than planned?
- Which epics are P0 (must-have) vs P1 (nice-to-have)?
- What's the failure mode if we ship 60% of this?
- What external dependencies could block this?
- What do we need to learn before we can plan this properly?
- How will we know this initiative is "done"?
```

### Epic Hard Questions
```
- What's the boundary of this epic? Where exactly does it end?
- What happens at 10x scale?
- What's the migration path from current state to this?
- What coupling does this introduce between modules/systems?
- Is this over-engineered for now or under-engineered for later?
- What's the rollback plan if this goes wrong?
- Can this be shipped incrementally, or is it all-or-nothing?
- What's the simplest version that validates the core hypothesis?
```

### Story Hard Questions
```
- Is this actually one story or multiple?
- What edge cases aren't covered by the acceptance criteria?
- What assumptions are baked into this story?
- What will break if requirements change after this ships?
- Is this testable without manual verification?
- What's the blast radius if this implementation is wrong?
```

### PRD Hard Questions
```
- Is the problem validated with evidence or assumed?
- Are the success metrics actually measurable with current tooling?
- What's explicitly out of scope, and is that the right boundary?
- Who loses if we build this? What's the second-order effect?
- What's the simplest version that validates the hypothesis?
- Are there regulatory or compliance implications not mentioned?
```

### Tech Spec Hard Questions
```
- What happens at 10x the expected load?
- What's the failure mode? What breaks first?
- What's the migration path from current state?
- Are there coupling decisions that will be expensive to reverse?
- Is the testing strategy sufficient? What's untested?
- What's the operational overhead of this approach?
```

---

## 11. Review Prompt System

Review prompts are templates that `sf_review_prompt` assembles. They embed the working style from the CLAUDE.md persona.

### PRD Review Prompt

```markdown
Review the following PRD as a senior product architect.

Focus on:
- Is the problem statement clear and evidence-based, or is it assumed?
- Are success metrics measurable and realistic?
- Is the scope right-sized? Flag gold-plating AND under-specification.
- Are there gaps in the user stories? Missing edge cases? Missing personas?
- Are the constraints sufficient or do they leave dangerous ambiguity?
- Is the "What If" list honest? What scenarios are missing?
- Are the open questions the RIGHT questions, or are they dodging harder ones?
- Would you approve this PRD to move to tech spec? If not, what's blocking?

Be direct, not diplomatic. If something is fine but would cause regret
in 6 months, say so now. If the scope is too vague to implement, block it.

---

{{doc_content}}
```

### Tech Spec Review Prompt

```markdown
Review the following technical specification as a principal engineer.

Ask the hard questions:
- What happens at 10x the expected load?
- What's the failure mode? What breaks first?
- What's the migration path from current state?
- Are there coupling decisions that will be expensive to reverse?
- Is this over-engineered for the current scale?
- Is this under-engineered for where we're heading?
- Are there non-obvious downstream consequences of pattern choices?
- Is the testing strategy sufficient? What's untested?

Focus on: correctness, edge cases, performance cliffs, security, naming.
Don't nitpick formatting. If something is fine but fragile, flag it.

---

{{doc_content}}
```

### Story Decomposition Prompt

```markdown
Decompose the following spec into implementable stories.

Rules:
- Each story must be independently testable
- Each story must have clear, specific acceptance criteria
- If acceptance criteria need a paragraph, the story is too big — split it
- Order stories by dependency (what must exist before what)
- Group into phases where stories in the same phase have no inter-dependencies
- Flag any trade-offs in the decomposition
- Identify blocked_by relationships explicitly
- Each story should be completable in a single Claude Code session

Present options if there are multiple valid decomposition strategies.
Don't pick for me — show the trade-offs.

---

{{spec_content}}
```

---

## 12. Features

### 12.1 Fast / Careful Mode

```yaml
# .specflow/config.yaml
mode: careful    # careful | fast
```

| Aspect | Fast Mode | Careful Mode |
|--------|-----------|--------------|
| Story creation | Title + acceptance criteria only | Full template with all fields |
| Plan save | Auto-set to approved | Requires explicit status=approved |
| Verification | Only on explicit request | Prompted after every execution_complete |
| Hard questions | Suppressed from MCP responses | Always included in context |
| Doc requirements | Stories can start without PRD | PRD required before epic goes active |

Toggle: `specflow mode fast` / `specflow mode careful`

### 12.2 Assumptions Tracking

Assumptions are strings recorded on stories during or after execution. They surface in:
- `specflow assumptions` — full project view
- `sf_assumptions` MCP tool — for Claude Code to check before implementing related stories
- Context builder — Layer 6 includes assumptions from completed predecessor stories

Assumptions are **append-only** via MCP (can only add, not remove through `sf_story_update`). Manual editing via CLI/editor can remove them.

### 12.3 Scope Check

`specflow scope-check <epic>` compares stories against the PRD:
1. Reads the PRD's "In Scope" and "Out of Scope" sections
2. Reads the PRD's "User Stories" section
3. Compares each story's title + description against PRD content
4. Flags stories with no PRD traceability
5. Flags stories that match "Out of Scope" items

This is a **text comparison** (substring matching on key terms), not AI analysis. Good enough to catch obvious scope creep.

### 12.4 Drift Detection

`specflow diff-check <epic>` detects spec-vs-story drift:
1. Reads all docs in the epic with their `updated` timestamps
2. Reads all stories with their `updated` timestamps and `doc_refs`
3. If a doc was updated AFTER a story that references it, flags potential drift
4. Reports which doc sections changed (by comparing content hashes of markdown headers)

### 12.5 Activity Log

Every state change is appended to `.specflow/log.jsonl`:
- Story status changes
- Execution start/complete
- Verification saves
- Document creates/updates
- Decision records

Format: one JSON object per line, each with `ts`, `type`, `entity`, and type-specific fields.

`specflow log --last=20` renders the most recent entries as a formatted timeline.

### 12.6 Search

`specflow search <query>` does full-text search across all markdown files in `.specflow/`:
- Searches frontmatter fields AND body content
- Returns: entity type, slug, matching line with context
- Implementation: simple file walking + string matching (no index needed for personal scale)

---

## 13. Go Module Structure

```
specflow/
├── cmd/
│   └── specflow/
│       ├── main.go                          # Cobra root command setup
│       ├── init.go                          # specflow init
│       ├── config.go                        # specflow config [set|get|ls]
│       ├── initiative.go                    # specflow initiative [new|ls|show|edit|set|archive]
│       ├── epic.go                          # specflow epic [new|ls|show|edit|set|archive]
│       ├── story.go                         # specflow story [new|ls|show|edit|set|next]
│       ├── doc.go                           # specflow doc [new|ls|show|edit]
│       ├── decision.go                      # specflow decision [new|ls|show]
│       ├── status.go                        # specflow status [slug]
│       ├── questions.go                     # specflow questions
│       ├── blocked.go                       # specflow blocked
│       ├── assumptions.go                   # specflow assumptions
│       ├── log.go                           # specflow log
│       ├── search.go                        # specflow search
│       ├── scope_check.go                   # specflow scope-check
│       ├── diff_check.go                    # specflow diff-check
│       ├── context.go                       # specflow context [slug]
│       ├── mode.go                          # specflow mode [fast|careful]
│       ├── import.go                        # specflow import
│       └── mcp.go                           # specflow mcp
├── internal/
│   ├── config/
│   │   ├── config.go                        # Config struct + loader
│   │   └── config_test.go
│   ├── store/
│   │   ├── store.go                         # Store interface + constructor
│   │   ├── frontmatter.go                   # Frontmatter parse/write helpers
│   │   ├── initiative.go                    # Initiative CRUD
│   │   ├── initiative_test.go
│   │   ├── epic.go                          # Epic CRUD
│   │   ├── epic_test.go
│   │   ├── story.go                         # Story CRUD + state machine validation
│   │   ├── story_test.go
│   │   ├── doc.go                           # Document CRUD
│   │   ├── doc_test.go
│   │   ├── decision.go                      # Decision CRUD
│   │   ├── decision_test.go
│   │   ├── plan.go                          # Plan CRUD
│   │   ├── plan_test.go
│   │   ├── execution.go                     # Execution lifecycle
│   │   ├── execution_test.go
│   │   ├── verification.go                  # Verification CRUD
│   │   ├── verification_test.go
│   │   ├── log.go                           # Activity log (append + read)
│   │   └── log_test.go
│   ├── context/
│   │   ├── builder.go                       # 6-layer context assembler
│   │   ├── builder_test.go                  # Golden file tests
│   │   └── fileref.go                       # File reference resolution
│   ├── git/
│   │   ├── git.go                           # Git operations (shell out)
│   │   └── git_test.go
│   ├── mcp/
│   │   ├── server.go                        # MCP stdio server setup
│   │   ├── tools_read.go                    # All read tool handlers
│   │   ├── tools_write.go                   # All write tool handlers
│   │   └── server_test.go
│   ├── hardq/
│   │   ├── questions.go                     # Hard question templates per entity type
│   │   └── questions_test.go
│   ├── models/
│   │   ├── initiative.go                    # Initiative struct
│   │   ├── epic.go                          # Epic struct
│   │   ├── story.go                         # Story struct + status validation
│   │   ├── doc.go                           # Document struct
│   │   ├── decision.go                      # Decision struct
│   │   ├── plan.go                          # Plan struct
│   │   ├── execution.go                     # Execution struct
│   │   ├── verification.go                  # Verification struct + Finding struct
│   │   └── log_entry.go                     # LogEntry struct
│   └── ui/
│       ├── render.go                        # Terminal rendering helpers
│       ├── table.go                         # Table output (lipgloss)
│       ├── markdown.go                      # Markdown rendering (glamour)
│       └── progress.go                      # Progress bar rendering
├── templates/                               # go:embed default templates
│   ├── initiative.md.tmpl
│   ├── epic.md.tmpl
│   ├── story.md.tmpl
│   ├── prd.md.tmpl
│   ├── tech-spec.md.tmpl
│   ├── api-spec.md.tmpl
│   ├── design-spec.md.tmpl
│   ├── adr.md.tmpl
│   ├── one-pager.md.tmpl
│   ├── context.md.tmpl                      # Context builder output template
│   ├── review-prd.md.tmpl                   # PRD review prompt
│   ├── review-tech-spec.md.tmpl             # Tech spec review prompt
│   └── decompose.md.tmpl                    # Spec decomposition prompt
├── testdata/                                # Test fixtures
│   ├── sample-project/                      # Complete .specflow/ fixture
│   └── golden/                              # Golden file test expectations
├── go.mod
├── go.sum
├── Makefile
└── .goreleaser.yaml                         # Cross-platform binary releases
```

---

## 14. Dependencies

```go
module github.com/yourusername/specflow

go 1.24

require (
    github.com/spf13/cobra v1.8.1              // CLI framework
    github.com/adrg/frontmatter v0.2.0         // YAML frontmatter parsing
    github.com/charmbracelet/lipgloss v1.0.0   // Terminal styling
    github.com/charmbracelet/glamour v0.8.0    // Markdown terminal rendering
    github.com/mark3labs/mcp-go v0.25.0        // MCP server SDK
    github.com/oklog/ulid/v2 v2.1.0            // ULID generation (time-sortable IDs)
    gopkg.in/yaml.v3 v3.0.1                    // YAML marshal/unmarshal
)
```

**Zero CGO. ~7 direct dependencies. Single static binary.**

---

## 15. Build Phases

| Phase | Name | Scope | Effort | Depends On |
|-------|------|-------|--------|------------|
| **1** | Models & Store | All data model structs + filesystem CRUD + frontmatter parsing + activity log | ~8h | — |
| **2** | CLI Foundation | Cobra setup + init + config + all CRUD commands (initiative, epic, story, doc, decision) + $EDITOR integration | ~8h | Phase 1 |
| **3** | Status & Reporting | status, questions, blocked, next, assumptions, log, search CLI commands | ~5h | Phase 2 |
| **4** | Context Builder | 6-layer context assembly + file reference resolution + context CLI command | ~5h | Phase 1 |
| **5** | MCP Server | All read + write tools over stdio + behavioral descriptions | ~6h | Phase 1, 4 |
| **6** | Git Integration | Diff, rev-parse, execution lifecycle (start/complete with ref tracking) | ~3h | Phase 1 |
| **7** | Hard Questions & Review Prompts | Template engine + all question sets + review prompts | ~3h | Phase 1 |
| **8** | Scope Check & Drift Detection | scope-check + diff-check commands and MCP tools | ~3h | Phase 1 |
| **9** | UI Polish | Terminal tables, markdown rendering, progress bars, colored output | ~3h | Phase 3 |
| **10** | Templates & Mode | Embedded default templates + fast/careful mode toggle | ~2h | Phase 1 |
| **11** | Import & Migration | Import existing markdown files as specflow artifacts | ~2h | Phase 1, 2 |
| **12** | Release | Makefile, goreleaser, cross-platform binaries, README | ~2h | All |

**Total: ~50 hours**

**Critical path:** Phase 1 → 2 → 3 gives you a usable CLI (~21h)
**Full loop:** Add Phase 4 → 5 → 6 for Claude Code integration (~35h)
**Complete:** All phases for the polished tool (~50h)

---

## 16. Phase Implementation Details

### Phase 1: Models & Store (~8h)

**Goal:** All data model structs defined, filesystem CRUD working, tests passing.

**Files to create:**
1. `internal/models/*.go` — All model structs with YAML tags
2. `internal/store/frontmatter.go` — Generic frontmatter parse/write (read file → split YAML/body → unmarshal)
3. `internal/store/store.go` — Store struct with root path, constructor, path helpers
4. `internal/store/initiative.go` — Create, Load, Save, List, Delete for initiatives
5. `internal/store/epic.go` — Same for epics (+ LoadStoriesForEpic)
6. `internal/store/story.go` — Same for stories (+ status transition validation, LoadStandalone, LoadForEpic)
7. `internal/store/doc.go` — Same for docs (project-level + epic-scoped)
8. `internal/store/decision.go` — Same for decisions
9. `internal/store/plan.go` — Plan CRUD (under executions/{story}/)
10. `internal/store/execution.go` — Execution CRUD (meta.yaml)
11. `internal/store/verification.go` — Verification CRUD
12. `internal/store/log.go` — Append log entry, read last N entries

**Key decisions:**
- Store struct holds root `.specflow/` path
- All methods take/return model structs
- Frontmatter roundtrip must preserve body formatting
- Story status transitions validated: draft→planned→in_progress→verifying→done, any→blocked
- IDs generated via ULID on create
- Slugs validated: lowercase, alphanumeric + hyphens only

**Tests:** Table-driven tests for each CRUD operation. Use `t.TempDir()` for isolated filesystem.

### Phase 2: CLI Foundation (~8h)

**Goal:** All CRUD commands working, $EDITOR integration for creating/editing artifacts.

**Files to create:**
1. `cmd/specflow/main.go` — Root cobra command, version flag
2. `cmd/specflow/init.go` — Create `.specflow/` directory structure
3. `cmd/specflow/config.go` — Config get/set/ls
4. `cmd/specflow/initiative.go` — initiative new/ls/show/edit/set/archive
5. `cmd/specflow/epic.go` — epic new/ls/show/edit/set/archive
6. `cmd/specflow/story.go` — story new/ls/show/edit/set/next
7. `cmd/specflow/doc.go` — doc new/ls/show/edit
8. `cmd/specflow/decision.go` — decision new/ls/show
9. `internal/config/config.go` — Config struct + file loading (project + global)

**Key decisions:**
- `$EDITOR` integration: write template to temp file, open editor, read back on close
- For `new` commands: pre-populate frontmatter from flags, body from template
- For `edit` commands: read existing file, open in editor, save back
- `ls` commands output simple ASCII tables (lipgloss in Phase 9)
- `show` commands render markdown to terminal (glamour in Phase 9, plain text for now)
- `next` command logic: filter by status==planned && blocked_by all done, sort by phase order then priority

**init --with-claude adds to .claude/settings.json:**
```json
{
  "mcpServers": {
    "specflow": {
      "command": "specflow",
      "args": ["mcp"]
    }
  }
}
```

### Phase 3: Status & Reporting (~5h)

**Goal:** Status rollup, open questions, blocked view, assumptions, activity log, search.

**Commands:**
1. `status` — Aggregate: count stories by status per epic, compute % complete, show phase progress
2. `status <slug>` — Auto-detect entity type (check initiatives/ then epics/ then stories/), show appropriate detail
3. `questions` — Walk all entities, collect open_questions, group by source
4. `blocked` — Filter stories where blocked_by contains non-done stories
5. `assumptions` — Walk all stories, collect assumptions, group by epic
6. `log` — Read last N entries from log.jsonl, format as timeline
7. `search` — Walk all .md files in .specflow/, grep for query, return matches with context

### Phase 4: Context Builder (~5h)

**Goal:** `sf_context_build` assembles full layered context from project state.

**Files to create:**
1. `internal/context/builder.go` — Main BuildContext function
2. `internal/context/fileref.go` — Extract file paths from plan markdown, resolve against project root

**Algorithm:**
1. Load story → follow epic ref → follow initiative ref
2. Load all docs referenced by story
3. Load plan if exists
4. Load completed sibling stories (title + summary only)
5. Load decisions for the epic
6. Collect open questions from all layers
7. Collect assumptions from completed stories
8. Read CLAUDE.md and AGENTS.md from project root
9. Read pattern exemplar files from config
10. Read files referenced in plan
11. Render through `context.md.tmpl` template

**Tests:** Golden file tests — create a fixture `.specflow/` directory in `testdata/sample-project/`, run BuildContext, compare output against `testdata/golden/context-*.md`.

### Phase 5: MCP Server (~6h)

**Goal:** Full MCP server over stdio with all read/write tools.

**Files to create:**
1. `internal/mcp/server.go` — Server setup, tool registration, stdio transport
2. `internal/mcp/tools_read.go` — All `sf_*` read tool handlers
3. `internal/mcp/tools_write.go` — All `sf_*` write tool handlers
4. `cmd/specflow/mcp.go` — Cobra command that starts the MCP server

**Key decisions:**
- Each tool handler is a function that takes MCP input → calls store/context → returns MCP text content
- Tool descriptions include behavioral instructions (the long descriptions from Section 7)
- Write tools log activity entries after successful operations
- Error handling: return MCP error responses with helpful messages

**Testing:** Integration test that starts MCP server in a goroutine, sends tool calls via stdio pipe, validates responses.

### Phase 6: Git Integration (~3h)

**Goal:** Git diff, ref tracking, execution lifecycle.

**Files to create:**
1. `internal/git/git.go` — Functions: CurrentRef, Diff(from, to), Status, FileChanges(from, to)

All functions shell out to `git` via `os/exec`. No git library dependency.

**Integration with store:**
- `sf_execution_start` calls `git.CurrentRef()` → stores as `git_ref_before`
- `sf_execution_complete` calls `git.CurrentRef()` → stores as `git_ref_after`, calls `git.FileChanges()` → stores changed files
- `sf_diff` calls `git.Diff(before, after)` → returns diff content

### Phase 7: Hard Questions & Review Prompts (~3h)

**Files to create:**
1. `internal/hardq/questions.go` — Map of entity type → question list
2. Templates: `review-prd.md.tmpl`, `review-tech-spec.md.tmpl`, `decompose.md.tmpl`

**MCP tools:** `sf_hard_questions`, `sf_review_prompt`

### Phase 8: Scope Check & Drift Detection (~3h)

1. `scope-check`: Parse PRD's In Scope/Out of Scope/User Stories sections, compare against story titles/descriptions
2. `diff-check`: Compare `updated` timestamps of docs vs stories that reference them

### Phase 9: UI Polish (~3h)

Replace plain text output with:
- lipgloss styled tables for `ls` commands
- glamour markdown rendering for `show` commands
- Progress bars for `status` output (using lipgloss)
- Colored status badges (green=done, yellow=in_progress, red=blocked, gray=draft)

### Phase 10: Templates & Mode (~2h)

1. Embed all templates via `go:embed`
2. Template override: check `.specflow/templates/` first, fall back to embedded
3. `specflow mode` command: reads/writes `mode` field in config
4. Mode affects: plan auto-approval, verification prompting, hard questions inclusion, doc requirements

### Phase 11: Import & Migration (~2h)

`specflow import <file> [--epic=<slug>] [--type=story|doc]`
- Reads markdown file
- Attempts to parse frontmatter (if present, use it; if not, generate from filename)
- Places in correct location based on flags
- Generates ID and slug

### Phase 12: Release (~2h)

1. Makefile with targets: build, test, lint, install
2. `.goreleaser.yaml` for cross-platform binaries (darwin/linux/windows × amd64/arm64)
3. README.md with installation, quickstart, usage

---

## Config File Reference

### Project config: `.specflow/config.yaml`

```yaml
# Mode: careful (full ceremony) or fast (minimal friction)
mode: careful

# Conventions file to include in context (relative to project root)
conventions_file: CLAUDE.md

# Additional context files
agents_file: AGENTS.md              # Optional

# Pattern exemplar files — included in context when relevant
pattern_exemplars:
  repository: "path/to/exemplar/repo.go"
  service: "path/to/exemplar/service.go"
  model: "path/to/exemplar/model.go"
  test: "path/to/exemplar/test.go"

# Default priority for new stories
default_priority: medium

# Default labels for new stories
default_labels: []
```

### Global config: `~/.specflow/config.yaml`

```yaml
# Editor for creating/editing artifacts (falls back to $EDITOR)
editor: nvim

# Default mode for new projects
default_mode: careful
```

---

## Status Summary

| Phase | Status |
|-------|--------|
| Phase 1: Models & Store | pending |
| Phase 2: CLI Foundation | pending |
| Phase 3: Status & Reporting | pending |
| Phase 4: Context Builder | pending |
| Phase 5: MCP Server | pending |
| Phase 6: Git Integration | pending |
| Phase 7: Hard Questions & Review | pending |
| Phase 8: Scope Check & Drift | pending |
| Phase 9: UI Polish | pending |
| Phase 10: Templates & Mode | pending |
| Phase 11: Import & Migration | pending |
| Phase 12: Release | pending |
