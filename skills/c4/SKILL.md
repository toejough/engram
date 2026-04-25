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
10. Show user the rendered markdown for approval; commit both `.json` and `.md`.

For `update`: edit the JSON, rerun `targ c4-l1-build`, rerun `targ c4-audit`,
and present the diff.

## Workflow: `update <name>`

1. Read the target file and its parent + children (per the file's front-matter `parent` and
   `children` fields).
2. Take the user's requested change.
3. Re-ground in code (steps 3–5 of `create`, scoped to affected packages).
4. Resolve any new code/intent conflicts via ask-the-user.
5. Draft the new diagram + catalog state.
6. Compute propagation:
   - Renamed/removed element → check L(N-1) box and every child file's `parent` reference.
   - New element → propose adding to L(N-1) parent's catalog; offer to scaffold an L(N+1) child.
   - Changed responsibility/relationship → check parent for consistency; check children for
     orphaned assumptions.
   - L3/L4 code-pointer change → re-verify L4 properties still link to live code; flag broken
     test pointers.
7. Present, in order: the target-layer diff, then per-affected-layer proposed change. Each
   proposed edit is a unified diff with a one-line reason.
8. For each proposal, the user picks `[a]pply`, `[s]kip`, or `[d]efer`. Apply approved edits.
   Persist deferred ones as drift notes in the target file.

## Workflow: `review <name>`

Steps 1–4 of `update`, read-only. Output a report: drift findings, missing cross-links, broken
code/test pointers, untested L4 properties added since last review, AND **ID-mismatch findings**
(diagram nodes/edges whose IDs don't match catalog/relationships rows, or vice versa). No edits.

## Workflow: `audit`

Loop `review` over every file in `architecture/c4/`. Produce a roll-up report.

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
