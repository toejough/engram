---
name: c4
description: Use when generating, updating, or reviewing C4 architecture diagrams under architecture/c4/. Triggers on "/c4", "create C4 diagram", "update C4", "add component to L3", "regenerate context diagram", or any request to add/modify architecture diagrams in C4 form.
---

# C4 Diagram Skill

Generate and maintain C4 architecture diagrams under `architecture/c4/`. Diagrams live at
`architecture/c4/c[level]-[name].md`. L1–L3 use mermaid + element catalog + relationships table.
L4 is a property/invariant ledger linking to enforcing code and validating tests.

## Sub-actions

Dispatch by intent. The user invokes `/c4 <sub-action> [args]`.

| Sub-action | What it does |
|---|---|
| `create <level> <name>` | Draft a new diagram file; require explicit user approval before writing. |
| `update <name>` | Modify an existing diagram, then propose updates per affected layer. |
| `review <name>` | Read-only drift report. No edits. |
| `audit` | Sweep `architecture/c4/`, produce a roll-up drift report. No edits. |

## Non-Negotiable Rules

1. **Ask, don't guess, on code/intent conflict.** When code reality and intent (docs / commit
   bodies / session memory) disagree, STOP drafting. Present both views. Ask the user which is
   correct. Record the resolution in the file (drift note) or proceed per their answer.
2. **Never invent test pointers** (L4). If a property has no test, mark it **⚠ UNTESTED**.
3. **Never edit a non-target file without per-file approval.** Propagation is by proposal +
   user approve/skip/defer, not by silent edit.
4. **Every diagram uses the project mermaid convention.** classDef block at top, `:::person /
   :::external / :::container / :::component` classes. See `references/mermaid-conventions.md`.
5. **Cross-link in the file body.** No index file. Each file names its parent and children
   directly with relative paths.
6. **Every diagram element and edge carries an ID.** Each catalog row gets `E1, E2, …`, each
   relationships row gets `R1, R2, …`, and the same IDs appear inside the mermaid node labels
   and edge labels. Every node also gets a `click NODE href "#anchor"` directive that links to
   the catalog row's anchor. Catalog and relationships rows carry HTML anchors
   (`<a id="e1-…"></a>`) so the links resolve. Mismatches (a node ID with no catalog row, or a
   catalog row whose ID isn't on the diagram) are reported as drift findings by `review` and
   `audit`. See `references/mermaid-conventions.md` for the exact pattern. *(L4 ledgers use
   `P1, P2, …` for properties; no diagram IDs needed since L4 has no diagram.)*

## Workflow: `create <level> <name>`

1. Read `architecture/c4/`. Note what already exists.
2. If `level > 1`, read the parent-level file. The new diagram MUST refine an element of its
   parent.
3. Shallow-scan the repo: `ls`, top-level dirs, `cmd/`, `internal/` package list, `README.md`.
4. For L3/L4, deep-read the specific packages/files in scope.
5. Read intent sources:
   - `CLAUDE.md` (project-level + user-global)
   - `docs/` tree
   - `git log --format=full` (scoped to files in play for L3/L4; recent N=50 repo-wide for L1/L2).
     Commit bodies are first-class evidence of *why*. Quote them when explaining a relationship,
     a deprecation, or a drift note.
   - Recent session memory: `engram recall --query "<topic>"`
6. **If conflict:** stop, present each conflict with both views, ask the user. Record resolutions.
7. Open `references/templates/c<level>-template.md` and fill it in.
8. For L1–L3, follow `references/mermaid-conventions.md`. For L4, follow
   `references/property-ledger-format.md`.
9. Show full draft to user, get approval, write the file at `architecture/c4/c<level>-<name>.md`.
10. After write: scan parent file for cross-link updates. Present those as propagation proposals
    (see Workflow: update). Apply approved ones; skip/defer the rest.

## Workflow: `create 1 <name>` (L1 specifics)

For L1, the workflow uses the `dev/c4-*` targets so mechanical work is offloaded
and only judgment remains in the LLM:

1. Read `architecture/c4/`. Note what already exists.
2. (L1: skip parent — there is no L0.)
3. **Run `targ c4-l1-externals --root . --packages ./...`** — capture JSON listing
   external-system candidates (HTTP calls, filesystem boundaries, subprocess
   invocations, env reads). The LLM picks which ones become diagram externals.
4. (L3/L4 only — skip for L1.)
5. **Run `targ c4-history --since 90d --limit 50`** — capture structured JSON of
   recent commits + bodies for intent context. Also read `CLAUDE.md`, `docs/`,
   and `engram recall --query "<topic>"` for additional intent.
6. **If conflict:** stop, present, ask the user. Record resolutions.
7. **Author `architecture/c4/c1-<name>.json`** filling in the L1Spec schema (see
   the design spec at `docs/superpowers/specs/2026-04-15-c4-l1-targets-design.md`).
8. **Run `targ c4-l1-build --input architecture/c4/c1-<name>.json --noconfirm`**
   to emit the markdown.
9. **Run `targ c4-audit --file architecture/c4/c1-<name>.md`** to verify zero
   findings. If any: revise the JSON and rebuild.
10. Show user the rendered markdown for approval; commit both `.json` and
    `.md`.
11. **After write: run the Propagation Discipline sweep** (see below). For
    L1, the relevant propagation is downward: any L2/L3 file with a
    `from_parent` carry-over from L1 should be checked for stale names and
    rebuilt if needed.

For `update`: edit the JSON, rerun `targ c4-l1-build`, rerun
`targ c4-audit`, and run the Propagation Discipline sweep before presenting
the diff for approval.

## Workflow: `create 3 <name>` (L3 specifics)

L3 mirrors the L1 pattern: the LLM authors a JSON spec; `c4-l3-build` emits
canonical markdown; `c4-audit` validates it. The differences from L1 are
parent linkage (an L3 must refine a specific element of its parent L2) and
component identification (each component has a code pointer).

1. **Read `architecture/c4/`** and the parent L2 file. The L3's `focus.id` and
   `focus.name` must match an in-scope element of the parent.
2. **Run `targ c4-registry --dir architecture/c4`** to learn which E-IDs are
   already taken across the existing diagrams. Pick component IDs that are
   free; pick from_parent IDs that already exist with the correct names.
3. **Deep-read the specific packages/files in scope.** Use the L1 externals
   target if you need a focused scan: `targ c4-l1-externals --root . --packages ./internal/<pkg>/...`
   gives a per-package list of HTTP/fs/exec/env calls that may surface as
   external-system edges.
4. **Read intent sources** (CLAUDE.md, docs/, recent commits via
   `targ c4-history`, and `engram recall --query "<topic>"`).
5. **If conflict** between code and intent: stop, present, ask. Record
   resolutions as drift notes.
6. **Author `architecture/c4/c3-<name>.json`** per the `L3Spec` schema (see
   the design spec at
   `docs/superpowers/specs/2026-04-26-c4-l3-and-registry-design.md`):
   - `focus`: `{ id, name, responsibility }` — the parent L2 container being
     refined. The L3 file gets a catalog row for the focus, even though the
     parent owns the canonical definition.
   - `elements`: every element has an explicit E-ID (no auto-assignment).
     Components live inside the focus subgraph (`kind: "component"`,
     `code_pointer` required); carry-over neighbors (people, externals,
     containers) live outside (`from_parent: true`, `code_pointer` forbidden).
   - `relationships`: `from`/`to` reference element names (or focus name).
7. **Run `targ c4-l3-build --input architecture/c4/c3-<name>.json --noconfirm`**
   to emit the markdown. The build performs registry validation: any
   from_parent element disagreeing with peer specs on name, or any new
   component reusing an existing E-ID under a different name, fails fast with
   a clear error.
8. **Run `targ c4-audit --file architecture/c4/c3-<name>.md`** to verify zero
   findings. The audit's always-on registry cross-check catches drift between
   the rendered markdown and peer JSONs; the level-aware child-prefix check
   catches stale `Refined by` entries; the L3-only code-pointer audit
   verifies every catalog code-pointer link resolves on disk.
9. **Show user the rendered markdown for approval**; commit both `.json` and
   `.md`.
10. **After write: run the Propagation Discipline sweep** (see below). For an
    L3 create that means, at minimum: rebuild any existing c3-*.md siblings
    so their Cross-links pick up the new file, update the parent c2 file's
    `children` list and "Refined by:" section, and rerun `targ c4-audit` +
    `targ c4-registry` on the full set.

For `update`: edit the JSON, rerun `targ c4-l3-build`, rerun
`targ c4-audit`, and run the Propagation Discipline sweep before presenting
the diff for approval.

## Workflow: `update <name>`

1. Read the target file and its parent + children (per the file's front-matter `parent` and
   `children` fields).
2. Take the user's requested change.
3. Re-ground in code (steps 3–5 of `create`, scoped to affected packages).
4. Resolve any new code/intent conflicts via ask-the-user.
5. Draft the new diagram + catalog state.
6. Classify the change so propagation knows what to do:
   - **Renamed element** → parent's matching `from_parent` element + every child's `from_parent`
     carry-over need the new name; the parent's mermaid edges using the old name need rewriting.
   - **Removed element** → parent's `from_parent` reference becomes orphaned; every child whose
     `focus.id` matches the removed ID is invalidated and must be rewritten or deleted.
   - **New element** → parent's catalog should add a corresponding entry if appropriate; an
     L(N+1) child can be scaffolded.
   - **Changed responsibility / relationship** → parent's matching prose may drift; children
     whose `from_parent` element previously had a different responsibility need a re-read.
   - **L3 code-pointer change** → the audit's `code_pointer_unresolved` finding catches dead
     paths automatically on next audit.
7. **Run the Propagation Discipline sweep** (see below) to apply the classified change to
   every affected file.
8. Present, in order: the target-layer diff, then per-affected-layer proposed change. Each
   proposed edit is a unified diff with a one-line reason.
9. For each proposal, the user picks `[a]pply`, `[s]kip`, or `[d]efer`. Apply approved edits.
   Persist deferred ones as drift notes in the target file.

## Workflow: `review <name>`

Steps 1–4 of `update`, read-only. Output a report: drift findings, missing cross-links, broken
code/test pointers, untested L4 properties added since last review, AND **ID-mismatch findings**
(diagram nodes/edges whose IDs don't match catalog/relationships rows, or vice versa). No edits.

## Workflow: `audit`

Loop `review` over every file in `architecture/c4/`. Produce a roll-up report.

## Propagation Discipline

Every C4 file is part of an interconnected set: a parent at level N-1,
siblings at level N, children at level N+1. When you create or update any
file, treat the change as potentially affecting all three directions and run
this sweep before declaring done.

The sweep is **always-on** — invoked from every `create`, `update`, and L1/L3
build workflow above. Skipping it lets drift accumulate quietly until the
next audit run surfaces it.

1. **Registry first.** Run `targ c4-registry --dir architecture/c4` and read
   the conflict list. Empty is the goal. Any conflict (`id_name_drift`,
   `name_id_split`, `id_collision_within_file`) is either a real bug to fix or
   an intentional gap that needs a Drift Note in the relevant file.

2. **Update the parent.** If the change introduced a new file or renamed an
   element that the parent carries:
   - Edit the parent JSON's front-matter `children` list to include or
     remove the affected child filename.
   - Edit the parent JSON's `cross_links.refined_by` array if the updated
     child is the parent's first/new refinement target.
   - If a from_parent element's name no longer matches what the parent
     declares, propose either updating the parent's name or the child's
     carry-over.
   Rebuild the parent (`targ c4-l<N-1>-build --input ... --noconfirm`).

3. **Rebuild siblings.** If you created or renamed a file at level N, every
   other file at level N whose `parent` matches yours has a stale Cross-links
   "Siblings:" section. Rerun `targ c4-l<N>-build` for each one — the build
   re-emits deterministically, so only the affected line will diff. Without
   this step, sibling lists drift the moment a peer is added.

4. **Notify children.** For every child of the modified file (front-matter
   `children` array), check whether the child's `from_parent` carry-overs
   still match the parent's element names and IDs. Propose updates as needed.
   If an element was deleted at the parent level, any child whose
   `focus.id` matched the deleted ID is orphaned and must be rewritten or
   deleted.

5. **Audit each touched file.** `targ c4-audit --file <each>` confirms the
   level-aware child-prefix check, the L3 code-pointer check, and the
   always-on registry cross-check are all clean.

6. **Present propagation proposals** as unified diffs with one-line reasons.
   For each, the user picks `[a]pply`, `[s]kip`, or `[d]efer`. Apply
   approved edits. Persist deferred ones as Drift Notes in the target file.

The aim: the architecture set always represents a consistent, audited
snapshot. Any edit that doesn't propagate creates drift the registry will
surface on the next audit run — and by then it's harder to reconstruct the
intent that should drive the fix.

## Drift Notes

When the user defers a propagation proposal or chooses to record a code/intent gap rather than
resolve it, append to a `## Drift Notes` section at the bottom of the target file:

```markdown
## Drift Notes

- **YYYY-MM-DD** — <one-line description>. Reason: <why deferred>.
```

Drift notes never silently disappear. They persist until a future `update` resolves them.

## References (load on demand)

- `references/c4-principles.md` — the 4 abstractions, 4 levels, common pitfalls.
- `references/mermaid-conventions.md` — classDef block + shape conventions + GitHub quirks.
- `references/property-ledger-format.md` — L4 row format + untested-property discipline.
- `references/templates/c<1-4>-template.md` — per-level starter scaffolds.

## Verification

This skill was verified with:

1. **Behavioral RED/GREEN test** — `tests/baseline-output-no-skill.md` vs.
   `tests/baseline-output-with-skill.md`. The skill-loaded run must produce a file at
   `architecture/c4/c1-<name>.md` with mermaid classDef block, element catalog, relationships
   table, and explicit cross-links to L2.
2. **Pressure test 1 — code/intent conflict** (`tests/pressure-conflict.md`): given docs that
   disagree with code, the skill must surface the conflict and ask, not silently pick one.
3. **Pressure test 2 — propagation** (`tests/pressure-propagation.md`): renaming a container at
   L2 must trigger proposed updates to the L1 parent and every L3 child file referencing it.
4. **Pressure test 3 — L4 untested property** (`tests/pressure-untested-property.md`): given a
   real invariant with no test, the skill must mark it **⚠ UNTESTED**, not omit it and not
   fabricate a test link.
