# C4 Diagram Skill — Design

**Date:** 2026-04-25
**Status:** Approved (brainstorming complete)
**Owner:** Joe

## Purpose

A Claude Code skill that generates and maintains C4 architecture diagrams for the engram codebase. The skill encodes (a) what "good" C4 looks like per c4model.com, (b) how to ground each diagram in current code, and (c) how to reconcile code reality with intended architecture from docs, commit history, and session memory.

## Scope

- **Levels supported:** L1 (System Context), L2 (Container), L3 (Component), L4 (Code as property/invariant ledger).
- **Diagram format:** Mermaid (`flowchart` for L1–L3 with consistent classDef styling; tables for L4 ledger). Renders natively on GitHub.
- **File location:** `architecture/c4/c[level]-[name].md`.
- **No index file** — files cross-link to parent and children directly.

## Skill Identity

- **Name:** `c4`
- **Slash command:** `/c4`
- **Skill directory:** `skills/c4/`
- **Trigger description (frontmatter `description` field):** Use when generating, updating, or reviewing C4 architecture diagrams under `architecture/c4/`. Triggers on "/c4", "create C4 diagram", "update C4", "add component to L3", "regenerate context diagram", or any request to add/modify architecture diagrams in C4 form.

## Sub-actions

Dispatched by intent under one `/c4` umbrella:

| Sub-action | Purpose |
|---|---|
| `create <level> <name>` | Generate a new diagram file at the named level. |
| `update <name>` | Modify an existing diagram, then propose updates to other affected layers. |
| `review <name>` | Read-only drift report for a single diagram. |
| `audit` | Sweep every file under `architecture/c4/`, produce a roll-up drift report. No edits. |

## Per-Level File Structure

Every `c[N]-<name>.md` file follows the same shape:

1. **Front-matter header**
   - `level` (1–4)
   - `name` (the `<name>` portion of the filename)
   - `parent` (relative link to the L(N-1) file this expands; absent for L1)
   - `children` (relative links to L(N+1) files that expand parts of this; may be empty)
   - `last_reviewed_commit` (git sha at last successful create/update/review)
2. **Mermaid diagram block** — for L1–L3 only.
3. **Element catalog** — for L1–L3. Table: name, type, responsibility, code-pointer (L3) or system-of-record (L1/L2).
4. **Relationships** — for L1–L3. Table: from, to, description, protocol/medium.
5. **Property ledger** — for L4 only. Table: Property, Statement (universally-quantified), Enforced at (`file:line` link), Tested at (`test_file:line` or **⚠ UNTESTED** callout), Notes.
6. **Drift notes** *(only when present)* — bulleted list of code-vs-intent gaps the user chose to defer at last edit. Each entry: date, brief description, reason for deferring.
7. **Cross-links** — explicit "Parent: [path]" and "Refines this container: [path]" lines.

## L4 Property Ledger

L4 replaces the traditional UML/code diagram with a property/invariant ledger.

- Each row states what the code **fundamentally guarantees, across all possible inputs** — universally-quantified statements ("for all X, …").
- Every property has an **Enforced at** pointer to the code that establishes the guarantee.
- Every property has a **Tested at** pointer to the test that validates it, OR an explicit **⚠ UNTESTED** callout.
- The skill never invents a test pointer. If no test exists, the row is flagged untested.
- Format details and examples live in `references/property-ledger-format.md`.

## Mermaid Conventions (L1–L3)

Since mermaid has no native C4 shapes, the skill enforces a project-wide convention:

- Person/actor: stadium `([Name])` with `:::person`
- External system: rounded `(Name)` with `:::external`
- Internal container: rectangle `[Name]` with `:::container`
- Internal component: subgraph inside container with `:::component`
- Standardized `classDef` block at the top of every diagram for GitHub mermaid compatibility.

Full convention reference: `references/mermaid-conventions.md`.

## Workflow per Sub-action

### `/c4 create <level> <name>`

1. Read `architecture/c4/` to learn what already exists.
2. If `level > 1`, read the parent-level file. The new diagram must refine some element of its parent.
3. **Shallow scan** the repo: `ls`, top-level dirs, `cmd/`, `internal/` package list, README.
4. For L3/L4, **deep-read** the specific packages/files in scope.
5. Read **intent sources**:
   - `CLAUDE.md` (project + user-global)
   - `docs/` tree
   - `git log --format=full` (scoped to files in play for L3/L4; repo-wide recent N for L1/L2). Commit bodies are first-class evidence of *why*.
   - Recent session memory via `engram recall --query "<topic>"`
6. **If code/intent conflict surfaces:** stop. Present each conflict with both views. Ask the user which is correct. Record the resolution.
7. Draft the diagram body (mermaid + catalog + relationships, OR property ledger for L4).
8. Show full draft to the user, get approval, then write the file.
9. After write: scan the parent file for cross-links that need updating; present those edits as propagation proposals (see below).

### `/c4 update <name>`

1. Read the target file and its parent + children.
2. Take the user's requested change.
3. Re-ground in code (steps 3–5 above, scoped to affected packages).
4. Resolve any new code/intent conflicts via ask-the-user.
5. Draft the new diagram + catalog state.
6. Compute propagation: walk parent (does it still describe a real refinement?) and every child file (do they still describe real elements of this layer?). For each affected layer, draft the proposed change.
7. **Present, in order:** target-layer diff, then per-affected-layer proposed change. User approves each independently.
8. Apply approved changes. Rejected/deferred changes persist as drift notes in the target file.

### `/c4 review <name>`

Read-only execution of update steps 1–4. Produces a report. No edits.

### `/c4 audit`

Loop `review` over every file in `architecture/c4/`. Produce a roll-up report.

## Code Grounding Strategy

- **L1/L2:** shallow scan (package tree, external boundaries, repo top-level). Cheap.
- **L3/L4:** narrow deep-read of specific packages/files in scope.
- No persistent manifest cache. The repo and `git log` are the source of truth; a manifest would be a second source that drifts.

## Conflict Resolution Rule

When code and intent disagree, the skill stops drafting and asks the user which is correct. After resolution:

- If the user picks code: intent source becomes a drift note (with reference to the doc/commit that disagreed).
- If the user picks intent: a drift note is added describing the code that needs to change to match. The skill does NOT edit code from the C4 skill — drift notes are the only output.
- If the user defers: a drift note records both views and the deferral.

## Propagation Rule

When `update` modifies layer N:

| Change | Propagation behavior |
|---|---|
| Renamed/removed element | Check L(N-1) box that refers to it; check every L(N+1) file whose `parent` is this file. |
| New element | Propose adding it to L(N-1) parent's catalog as a refinement target; offer to scaffold L(N+1) child file. |
| Changed responsibility / relationship | Check parent for consistency; check children for orphaned assumptions. |
| Code-pointer change at L3/L4 | Re-verify L4 properties still link to live code; flag any broken test pointer. |

Each proposed edit is shown as a unified diff with a one-line reason. User picks `[a]pply`, `[s]kip`, or `[d]efer`. The skill never edits a non-target file without explicit per-file approval. Deferred changes become drift notes.

## Skill Directory Layout

```
skills/c4/
├── SKILL.md                          # principles inline, dispatch flow, non-negotiable rules
└── references/
    ├── c4-principles.md              # 4 levels, abstractions, common pitfalls (distilled from c4model.com)
    ├── mermaid-conventions.md        # classDef block, shapes, GitHub mermaid quirks
    ├── property-ledger-format.md     # L4 invariant statement style + examples
    └── templates/
        ├── c1-template.md
        ├── c2-template.md
        ├── c3-template.md
        └── c4-template.md
```

SKILL.md is lean — only what every invocation must see. References load on demand when the skill consults them (progressive disclosure).

## Testing & Verification

Per project rule, all SKILL.md authoring uses `superpowers:writing-skills` (TDD discipline).

- **RED — baseline behavioral test:** before writing the skill, capture how a fresh Claude session responds to "create a C4 L1 diagram for engram" without the skill loaded. Likely produces a generic/inconsistent diagram with no drift handling and no property ledger. Record that as the baseline.
- **GREEN:** author SKILL.md + references; verify a fresh session given the same prompt now:
  1. reaches for the skill;
  2. produces a file at `architecture/c4/c1-<name>.md`;
  3. follows the mermaid + catalog + relationships shape;
  4. asks before resolving code/intent conflicts;
  5. proposes propagation when used in update mode.
- **Pressure tests** (at least three):
  1. **Code/intent conflict** — docs say "X talks to Y" but code shows X never imports Y. Skill must surface the conflict and ask, not silently pick one.
  2. **Update propagation** — rename a container at L2; skill must propose updates to the L1 parent and every L3 child file referencing it.
  3. **L4 untested property** — feed code with a real invariant that has no test; skill must mark it **⚠ UNTESTED**, not omit it and not fabricate a test link.

A "Verification" section in SKILL.md's footer lists these so the skill can be self-audited later.

## Out of Scope

- Auto-fixing code to match intent (skill emits drift notes only, never edits non-diagram files).
- Diagram rendering tooling beyond mermaid (no Structurizr, no PlantUML).
- Deployment / dynamic / system-landscape diagrams (can be added later as a separate skill or extension).
- A persistent manifest cache.
