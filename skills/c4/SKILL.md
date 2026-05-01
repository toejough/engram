---
name: c4
description: Use when generating, updating, or reviewing C4 architecture diagrams under architecture/c4/. Triggers on "/c4", "create C4 diagram", "update C4", "add component to L3", "regenerate context diagram", or any request to add/modify architecture diagrams in C4 form.
---

# C4 Diagram Skill

Generate and maintain C4 architecture diagrams under `architecture/c4/`. L1–L3 are
SNMPR-style node-and-relationship diagrams + element catalog. L4 is a property/invariant
ledger plus a strict call diagram.

## Sub-actions

The user invokes `/c4 <sub-action> [args]`.

| Sub-action | What it does |
|---|---|
| `create <level> <name>` | Draft a new diagram; require explicit user approval before writing. |
| `update <name>` | Modify an existing diagram; propose updates per affected layer. |
| `audit [<name>]` | Read-only drift report. With `<name>`, scoped to one file. Without, sweeps `architecture/c4/` and produces a roll-up. No edits in either mode. |

## Non-Negotiable Rules

These rules apply at every level. Level-specific rules live in *Level-Specific Sections*.

1. **Ask, don't guess, on code/intent conflict.** When code reality and intent (docs / commit
   bodies / session memory) disagree, STOP. Present both views. Ask the user. Deferred gaps
   and skipped propagation proposals become issues via the available issue-filing skill —
   never inline drift annotations. (Legacy `## Drift Notes` sections are read-only; surface
   them as input to the issue-filing decision.)
2. **Never invent pointers.** L4 properties without tests are **⚠ UNTESTED**. L3 components
   without `source` are flagged. L1/L2 elements with a path-like `source` that doesn't resolve
   are flagged. Don't fabricate file:line references at any level. **Every catalog row carries
   a *where*** — `source` at L1/L2/L3 (path or descriptive identifier), `enforced_at` +
   `tested_at` at L4 properties. Tier 2 of Two-Tier Extraction (Rule 7) MUST verify every
   *where* by reading the source it points to.
3. **Never edit a non-target file without per-file approval.** Propagation is by proposal +
   approve/skip/defer, not silent edit. Idempotent rebuilds of auto-generated sections
   (mermaid block, catalog, cross-links — anything `c4-l*-build` regenerates) are propagation,
   not edits, and don't require approval. JSON edits to non-target files (renames, carry-over
   additions) DO require per-file `[a]pply`/`[s]kip`/`[d]efer`.
4. **Use the project mermaid convention.** classDef block + `:::person/external/container/component`
   classes + level-scoped IDs (`S<n>` at L1, `N<n>` at L2, `M<n>` at L3, `P<n>` for L4
   properties, `R<n>` for runtime-call edges). Cross-doc references use the full
   hyphen-separated path (`S2-N3-M5-P1`). Mismatches between diagram and catalog/relationships
   tables are reported as drift findings by `audit`. Full ID-namespace details and shape
   syntax: `references/mermaid-conventions.md`.
5. **Cross-link in the file body.** No index file. Each file names its parent and children
   directly with relative paths.
6. **Pre-render every diagram to SVG via `targ c4-render` and commit both `.mmd` and `.svg`.**
   GitHub's Mermaid renderer doesn't support the ELK layout engine; pre-rendering is
   required for clean R-edge layout.
7. **Two-Tier Extraction.** Every "scan source for findings" task — at every level, for the
   *what* (identification) and the *where* (location) — runs Tier 1 (Haiku sub-agent,
   wide-scan recall) → Tier 2 (you, precision verification by re-reading source). Single-pass
   authoring is forbidden; Tier 1 output is signal, never the final spec. Per-level
   enumeration lists, the iron rules, and the dispatch template:
   `references/two-tier-extraction.md`.

## Notation

Workflow lines are single messages: `LLM → X:` outbound, `LLM ← X:` inbound,
`LLM:` internal cognition, `if/loop/else` control flow. Actors: **User**, **Subagent**
(Haiku Tier 1), **targ** (`c4-l*-build`, `c4-render`, `c4-audit`, `c4-l1-externals`,
`c4-history`, `engram recall`), **FS**, **VCS**.

**Memory consultation.** Wherever a workflow says `consult memories`, you query whatever
memory plugin is installed for prior false positives, conflict resolutions, and
granularity preferences in this domain. Same step everywhere it appears.

## Workflow: `create <level> <name>`

Same shape for L1–L4. Per-level discovery, spec schema, and build target tabled below.

```
LLM      → FS:       read architecture/c4/, CLAUDE.md, docs/
LLM      ← FS:       existing diagrams, intent
if level > 1:
  LLM    → FS:       read parent c<level-1>-*.json
  LLM    ← FS:       parent IDs, element names
LLM      → targ:     <per-level discovery target — see table>
LLM      ← targ:     candidates JSON (externals, commit metadata, ...)
LLM      → Subagent: Tier 1 — wide-scan source for candidates (Rule 7)
LLM      ← Subagent: candidates (each with the WHAT and the WHERE)
LLM:                 Tier 2 — verify each candidate by re-reading source
                     prune false positives, merge duplicates, locate file:line
LLM:                 consult memories
if code/intent conflict:
  LLM:               consult memories
  LLM    → User:     present both views (and prior resolution if any), ask
  LLM    ← User:     resolution (or → file as issue)
LLM      → FS:       author architecture/c4/c<level>-<name>.json
LLM      → targ:     c4-l<level>-build --input <spec> --noconfirm
LLM      → targ:     c4-render
LLM      → targ:     c4-audit --file <spec>
LLM      ← targ:     findings (must be zero)
LLM:                 run Propagation Discipline sweep (below)
LLM      → User:     show rendered markdown
LLM      ← User:     approve
LLM      → VCS:      stage + commit .json + .md + .mmd + .svg
```

Per-level discovery surface and build target (per-level rules in *Level-Specific Sections*):

| Level | Tier 1 discovery surface | Build target |
|---|---|---|
| 1 | `targ c4-l1-externals --root . --packages ./...` + `targ c4-history --since 90d --limit 50` | `targ c4-l1-build` |
| 2 | repo top + in-scope L1 element's source surface | `targ c4-l2-build` |
| 3 | packages/files inside the focus container | `targ c4-l3-build` |
| 4 | focus component's source | `targ c4-l4-build` |

For the JSON spec shape per level — field names, types, required vs optional,
validation rules — see `references/spec-schemas.md`. That reference is the source of
truth for authoring. The build target validates and rejects malformed input with a
clear error.

## Workflow: `update <name>`

```
LLM      ← User:     change request
LLM      → FS:       read target c<level>-<name>.json + parent + children
LLM      ← FS:       linked specs
LLM      → FS:       re-read affected packages
LLM      ← FS:       current code
LLM:                 judge — does the change touch identification (new/renamed/removed
                     elements that need re-discovery, vs purely a description tweak)?
if yes:
  LLM    → Subagent: Tier 1 — re-discover (Rule 7)
  LLM    ← Subagent: candidates
  LLM:               consult memories
  LLM:               Tier 2 — verify
if new code/intent conflict:
  LLM:               consult memories
  LLM    → User:     present (with prior resolution if any), ask
  LLM    ← User:     resolution (or → file as issue)
LLM:                 consult memories
LLM:                 classify change (see classification cheat sheet below)
LLM      → FS:       edit target spec
LLM      → targ:     c4-l<level>-build --input <spec> --noconfirm
LLM      → targ:     c4-render
LLM      → targ:     c4-audit --file <spec>
LLM      ← targ:     findings
LLM:                 run Propagation Discipline sweep (below)
LLM      → User:     present target diff + per-affected-file proposed changes
                     (each = unified diff + one-line reason)
loop per propagation proposal:
  LLM    ← User:     [a]pply | [s]kip | [d]efer
  if apply:
    LLM  → FS:       edit affected spec
    LLM  → targ:     rebuild affected
  if defer:
    LLM  → issues:   file gap as issue (via available issue-filing skill)
LLM      → User:     present final diff
LLM      ← User:     approve commit
LLM      → VCS:      stage + commit
```

Change-classification cheat sheet (drives what propagation must touch):

- **Renamed element** → parent's `from_parent` carry-over and every child's `from_parent`
  carry-over need the same rename.
- **Removed element** → parent's `from_parent` reference is orphaned, any child whose
  `focus.id` matches the removed ID is invalidated.
- **New element** → parent's catalog should add a corresponding entry, an L(N+1) child
  can be scaffolded.
- **Changed responsibility/relationship** → parent's matching prose may drift, children
  whose `from_parent` element previously had a different responsibility need a re-read.
- **L3 source change** → audit's `source_path_unresolved` catches dead paths on next audit.

## Workflow: `audit [<name>]`

Read-only. With `<name>` = scoped to one spec, without = sweep `architecture/c4/`. No
edits in either mode.

```
if <name> given:
  LLM    → targ:     c4-audit --file architecture/c4/c<level>-<name>.json
  LLM    ← targ:     findings
else (sweep):
  LLM    → FS:       list architecture/c4/c*-*.json
  LLM    ← FS:       file list
  loop per spec:
    LLM  → targ:     c4-audit --file <spec>
    LLM  ← targ:     findings for <spec>
LLM:                 roll up findings (per file or aggregate)
LLM      → User:     drift report
```

The audit takes the `.json` spec only — passing `.md` is rejected with a hint. Finding
IDs include: `spec_invalid`, `source_path_unresolved` (L1/L2/L3),
`property_link_unresolved` (L4), `l4_carryover` (L4↔L3 element parity),
`rendered_markdown_missing` / `rendered_markdown_stale` (byte-compare against fresh emit).

## Propagation Discipline

A C4 set is interconnected. Whenever you create or rename/remove an element another
diagram references, propagate. Skipping = drift on the next audit.

Required sweep after any `create` or `update`:

```
if level > 1:
  LLM    → FS:       open parent c<level-1> spec
  LLM    → User:     propose update to parent's `cross_links.refined_by`
  LLM    ← User:     [a]pply | [s]kip | [d]efer
  if apply:
    LLM  → FS:       edit parent spec
    LLM  → targ:     c4-l<level-1>-build (rebuild parent's auto sections)
LLM      → FS:       list architecture/c4/c<level>-*.json
LLM      ← FS:       sibling peer files
LLM:                 filter to peers whose `parent` matches the changed file
loop per sibling peer:
  LLM    → targ:     c4-l<level>-build (idempotent — no JSON edit, no approval)
LLM      → FS:       list architecture/c4/c<child-level>-*.json
LLM      ← FS:       candidate children
LLM:                 filter to children whose `parent` matches the modified file
loop per child:
  LLM    → FS:       check `from_parent` carry-overs vs current parent
  if drift:
    LLM  → User:     propose carry-over update
    LLM  ← User:     [a]pply | [s]kip | [d]efer
    if apply:
      LLM → FS:      edit child spec
      LLM → targ:    c4-l<child-level>-build
LLM      → targ:     c4-audit --file <each-modified-spec>
LLM      ← targ:     findings (target zero, or filed as issues)
```

Idempotent rebuilds of auto-generated sections (mermaid block, catalog, cross-links —
anything `c4-l*-build` regenerates) are propagation, not edits, and don't require
approval. JSON edits to non-target specs DO. (Rule 3.) The front-matter `children` field
is hard-stamped `[]` by the build target — work through `cross_links.refined_by` only.

### Common rationalizations to reject

| Excuse | Reality |
|---|---|
| "Pre-existing drift, not caused by my change" | Touching this set means leaving known drift creates audit findings on the next run. Fix it as part of your change, or file it as an issue. |
| "The change is too small to need propagation" | Every C4 file cross-references peers. Skipping the sweep means `c4-audit` surfaces conflicts later when context is gone. |
| "Rebuilding peers feels like editing files I shouldn't" | Idempotent rebuilds of auto-generated sections are propagation; see Rule 3. |
| "I'll catch it in the next audit" | The next audit may be in a different session, after rationale is lost. Propagate now. |
| "Single-pass extraction is good enough this time" | Rule 7. Single-pass is forbidden. Tier 1 → Tier 2, every time. |

## Level-Specific Sections

### L1 specifics

- L1 has no parent — skip step 2 of `create`.
- `targ c4-l1-externals` produces JSON of external-system candidates (HTTP calls, filesystem
  boundaries, subprocess invocations, env reads). The Tier 1 sub-agent enumerates from this;
  Tier 2 picks which become diagram externals.

### L2 specifics

- L2 refines a single in-scope L1 container into multiple L2 containers (binary entry
  points, on-disk stores, hooks, skills, etc.). The L2Spec marks the in-scope element via a
  flag; `c4-l2-build` validates that exactly one in-scope element exists.
- IDs are `N<n>`, scoped within this diagram (start at `N1`).
- `from_parent` neighbors carry IDs/names from the L1 spec.

### L3 specifics

- L3 has a `focus` field naming the L2 container being refined. `focus.id`/`focus.name` must
  match an in-scope element of the parent L2.
- IDs are `M<n>`, scoped within this diagram (start at `M1`).
- Each `kind: "component"` element requires `source` (a repo-relative path).
  `from_parent: true` neighbors carry IDs/names from peer specs.
- Audit catches dead `source` paths via `source_path_unresolved` (same finding fires at L1/L2
  whenever a path-like `source` value doesn't resolve).

### L4 specifics

Read `references/property-ledger-format.md` (row format + untested-property discipline)
and `references/mermaid-conventions.md` (call diagram, R-edge property tags) before
authoring.

- **One strict C4 call diagram.** SNMPR-style nodes, `R<n>` runtime-call edges only — no
  D-edges, no port nodes, no `W`/`A` namespaces.
- **Externals required on the call diagram.** Every external system the focus crosses to
  must appear as a node with at least one R-edge from the focus.
- **Tier 1 enumerates** properties, call-diagram nodes, and R-edge property tags;
  never invent test pointers or externals not actually crossed.
- **`property_link_unresolved`** audit finding catches dead enforced/tested paths.

## References (load on demand)

| File | Use when |
|---|---|
| `references/c4-principles.md` | learning the 4 abstractions / 4 levels / common pitfalls |
| `references/mermaid-conventions.md` | authoring diagrams (classDef, shapes, ID namespace, L4 call diagram, render setup) |
| `references/property-ledger-format.md` | authoring L4 (row format, untested-property discipline) |
| `references/two-tier-extraction.md` | dispatching Tier 1, verifying Tier 2 (per-level enumeration, dispatch template) |
| `references/spec-schemas.md` | finding the canonical Go-struct schema for a level |
| `references/templates/c<1-4>-template.md` | scaffolding a new spec |

