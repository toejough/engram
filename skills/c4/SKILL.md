---
name: c4
description: Use when generating, updating, or reviewing C4 architecture diagrams under architecture/c4/. Triggers on "/c4", "create C4 diagram", "update C4", "add component to L3", "regenerate context diagram", or any request to add/modify architecture diagrams in C4 form.
---

# C4 Diagram Skill

Generate and maintain C4 architecture diagrams under `architecture/c4/`. L1–L3 are
SNMPR-style node-and-relationship diagrams + element catalog. L4 is a property/invariant
ledger plus a strict call diagram and a derived wiring diagram.

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
   bodies / session memory) disagree, STOP. Present both views. Ask the user. Record the
   resolution as a drift note or proceed per their answer.
2. **Never invent pointers.** L4 properties without tests are **⚠ UNTESTED**. L3 components
   without `source` are flagged. L1/L2 elements with a path-like `source` that doesn't resolve
   are flagged. Don't fabricate file:line references at any level. **Every catalog row carries
   a *where*** — `source` at L1/L2/L3 (path or descriptive identifier), `enforced_at` +
   `tested_at` at L4 properties, `wired_by_*` + `wrapped_entity_id` at L4 manifest. Tier 2 of
   Two-Tier Extraction (Rule 7) MUST verify every *where* by reading the source it points to.
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
7. **Two-Tier Extraction Discipline applies to every "scan source for findings" task at every level.**
   Single-pass authoring is forbidden. Tier 1 (Haiku-class sub-agent) wide-scans the source
   and enumerates every plausible candidate — Tier 1 owns recall and biases toward too many.
   Tier 2 (Sonnet-class+ — typically you, the orchestrator) verifies each candidate against
   source, prunes false positives, merges near-duplicates, splits conflated rows, locates
   exact file:line, marks genuinely unverifiable items as untested/unfound — Tier 2 owns
   precision. **Every extraction has two sub-tasks: the *what* (identification) and the
   *where* (location). Both must go through Tier 1 → Tier 2; Tier 2 must open every *where*
   pointer and verify it against source.** Per level:

   | Level | What (identify) | Where (locate) |
   |---|---|---|
   | L1 | external systems + in-scope system | each element's `source` |
   | L2 | containers under the in-scope L1 element | each element's `source` |
   | L3 | components inside the focus container | each component's `source` (file:line path) |
   | L4 | properties + manifest seams + R-edge property-tag assignments | each property's `enforced_at` + `tested_at`; each manifest row's `wired_by_*` + `wrapped_entity_id` |

   Also applies to any future "mine source artifacts for findings" task. Tier 1 output is
   **signal**, never the final artifact — never write `c<level>-<name>.json` directly from
   Tier 1. Tier 2 must read source to verify, not just trust Tier 1's claims. Full
   discipline (per-level enumeration lists, empirical baseline, untested-pointer rule):
   `references/two-tier-extraction.md`.

## Workflow: `create <level> <name>`

Universal procedure — same shape for L1–L4. Per-level discovery, schema, build target, and
specifics are tabled below.

1. Read `architecture/c4/`. Note what exists.
2. If `level > 1`, read the parent file. The new diagram MUST refine an element of its parent.
3. **Run discovery via Two-Tier Extraction** (Rule 7). Tier 1 dispatch → wide candidate list.
   Discovery target per level table.
4. Read intent: `CLAUDE.md` (project + user-global), `docs/`, `git log --format=full` (scoped
   for L3/L4 to packages in play; recent N=50 repo-wide for L1/L2 — commit bodies are
   first-class evidence of *why*), and `engram recall --query "<topic>"`.
5. **Tier 2 verify.** For each Tier 1 candidate, read source, prune/merge/split, locate
   exact file:line. **If conflict** between code and intent: stop, present, ask. Record
   resolutions.
6. Author `architecture/c4/c<level>-<name>.json` per the level's spec schema.
7. Run the level's build target.
8. Run `targ c4-render` to (re)generate any new/stale `.svg`.
9. Run `targ c4-audit --file architecture/c4/c<level>-<name>.json`. Zero findings. The
   audit takes the `.json` spec only (the source of truth); rendered `.md`/`.mmd`/`.svg`
   files are mechanical artifacts and the audit checks them only for staleness via
   byte-compare against a fresh emit.
10. Run *Propagation Discipline* sweep (below).
11. Show the rendered markdown to the user. On approval, commit `.json`, `.md`, `.mmd`, `.svg`.

| Level | Discovery (Tier 1 input) | Spec schema | Build target | Specifics |
|---|---|---|---|---|
| 1 | `targ c4-l1-externals --root . --packages ./...` + `targ c4-history --since 90d --limit 50` | `L1Spec` (see `dev/c4_l1.go`) | `targ c4-l1-build` | *L1 specifics* |
| 2 | Manual scan + Tier 1 sub-agent over the in-scope L1 element's source surface | `L2Spec` (see `dev/c4_l2.go`) | `targ c4-l2-build` | *L2 specifics* |
| 3 | Tier 1 sub-agent over packages/files in scope | `L3Spec` (see `dev/c4_l3.go`) | `targ c4-l3-build` | *L3 specifics* |
| 4 | Tier 1 sub-agent over the focus component's source | `L4Spec` (see `dev/c4_l4.go`) | `targ c4-l4-build` | *L4 specifics* |

## Workflow: `update <name>`

1. Read the target file and its parent + children (per the file's front-matter `parent` and
   `children` fields).
2. Take the user's requested change.
3. Re-ground in code (steps 3–5 of `create`, scoped to affected packages — Two-Tier
   Extraction still applies to any re-discovery work).
4. Resolve any new code/intent conflicts via ask-the-user.
5. Draft the new diagram + catalog state.
6. **Classify the change** so propagation knows what to do:
   - **Renamed element** → parent's `from_parent` carry-over and every child's `from_parent`
     carry-over need the same rename. Mermaid edges using the old name need rewriting.
   - **Removed element** → parent's `from_parent` reference is orphaned; any child whose
     `focus.id` matches the removed ID is invalidated.
   - **New element** → parent's catalog should add a corresponding entry; an L(N+1) child can
     be scaffolded.
   - **Changed responsibility/relationship** → parent's matching prose may drift; children
     whose `from_parent` element previously had a different responsibility need a re-read.
   - **L3 source change** → the audit's `source_path_unresolved` finding catches dead
     paths automatically on next audit.
7. Edit JSON, run the level's build target, run `targ c4-render`, run `targ c4-audit`.
8. Run *Propagation Discipline* sweep.
9. Present, in order: the target-layer diff, then per-affected-layer proposed change. Each
   proposed edit is a unified diff with a one-line reason.
10. For each proposal, the user picks `[a]pply`, `[s]kip`, or `[d]efer`. Apply approved
    edits. Persist deferred ones as drift notes in the target file.

## Workflow: `audit [<name>]`

Read-only. With `<name>`: steps 1–4 of `update` against that file, then output a report —
drift findings, missing cross-links, broken code/test pointers, untested L4 properties added
since last audit, AND **ID-mismatch findings** (diagram nodes/edges whose IDs don't match
catalog/relationships rows, or vice versa). Without `<name>`: loop the same over every file
in `architecture/c4/` and produce a roll-up. No edits in either mode.

## Propagation Discipline

A C4 set is interconnected. Whenever you create or rename/remove an element another
diagram references, propagate. Skipping = drift on the next audit.

**Required sweep after any create or update:**

1. **Update parent's `cross_links.refined_by`** in the parent JSON; rerun the parent's build.
   This step is required even when the parent already has unrelated drift in `refined_by` —
   fix it now or capture as a Drift Note. (The front-matter `children` field is currently
   hard-stamped `[]` by the build target and is not the propagation surface — work through
   `cross_links.refined_by` only.)
2. **Rebuild siblings.** For any peer at the same level whose `parent` matches the changed
   file, rerun the build target so its auto-generated "Siblings:" cross-link section
   refreshes. Idempotent — only auto-generated sections diff.
3. **Walk children.** For every child of the modified file, check whether its `from_parent`
   carry-overs still match the parent's element names and IDs. Rebuild any whose
   carry-overs drift.
4. **Sweep audit.** Run `targ c4-audit --file <spec>.json` on every modified spec. Zero
   findings. Capture intentional gaps as Drift Notes. The audit reads the JSON spec; do
   not pass a rendered `.md` (it will be rejected with a hint).

### Common rationalizations to reject

| Excuse | Reality |
|---|---|
| "Pre-existing drift, not caused by my change" | Touching this set means leaving known drift creates audit findings on the next run. Fix it as part of your change, or capture it as a Drift Note. |
| "The change is too small to need propagation" | Every C4 file cross-references peers. Skipping the sweep means `c4-audit` surfaces conflicts later when context is gone. |
| "Rebuilding peers feels like editing files I shouldn't" | Idempotent rebuilds of auto-generated sections are propagation; see Rule 3. |
| "I'll catch it in the next audit" | The next audit may be in a different session, after rationale is lost. Propagate now. |
| "Single-pass extraction is good enough this time" | Rule 7. Single-pass is forbidden. Tier 1 → Tier 2, every time. |

## Drift Notes

When the user defers a propagation proposal or chooses to record a code/intent gap rather
than resolve it, append to a `## Drift Notes` section at the bottom of the target file:

```markdown
## Drift Notes

- **YYYY-MM-DD** — <one-line description>. Reason: <why deferred>.
```

Drift notes never silently disappear. They persist until a future `update` resolves them.

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

L4 is the most complex level and has its own diagrams, schemas, and conventions. Read
`references/property-ledger-format.md` (manifest + DI Wires schemas) and
`references/mermaid-conventions.md` (call + wiring diagrams, R-edge property tags, build-time
validation) before authoring. A complete worked example lives at
`references/worked-example-c4-recall.md`.

- **Two diagrams.** A strict C4 call diagram (`<name>.mmd`) and a wiring diagram
  (`<name>-wiring.mmd`). The call diagram has SNMPR-style nodes and `R<n>` runtime-call
  edges only — no D-edges, no port nodes, no `W`/`A` namespaces. The wiring diagram is
  **derived** from the dependency manifest by grouping rows by `(wired_by_id,
  wrapped_entity_id)`. The L4 builder enforces strict alignment: every manifest
  `wrapped_entity_id` must match a node on the call diagram.
- **Externals required on the call diagram.** Every external system the focus crosses to
  via DI (filesystem, OS, network, Anthropic API, Claude Code, etc.) must appear as a node
  with at least one R-edge from the focus.
- **R-edge property tags.** Each R-edge label may end with the P-IDs the call realizes:
  `R8: ... [P3, P4, P9, P10]`. Use range notation for contiguous P-runs (`[P5–P8]`).
- **Two-Tier Extraction (Rule 7) applies with extra rigor.** Tier 1 enumerates four
  candidate types: properties, call-diagram nodes, manifest rows, R-edge property tags.
  Tier 2 verifies each against source — never invent test pointers, never invent externals
  not actually crossed. See `references/two-tier-extraction.md` for the full per-tier
  enumeration lists.
- **`property_link_unresolved` audit finding** catches dead enforced/tested paths.

## References (load on demand)

- `references/c4-principles.md` — the 4 abstractions, 4 levels, common pitfalls.
- `references/mermaid-conventions.md` — classDef + shape conventions + ID namespace + L4
  call/wiring diagrams + GitHub quirks + render setup.
- `references/property-ledger-format.md` — L4 row format + Dependency Manifest + DI Wires
  schemas + untested-property discipline.
- `references/two-tier-extraction.md` — Tier 1/Tier 2 discipline, per-level enumeration
  lists, empirical baseline.
- `references/worked-example-c4-recall.md` — c4-recall walk-through (call + wiring +
  manifest).
- `references/templates/c<1-4>-template.md` — per-level starter scaffolds.

## Verification

This skill was verified with:

1. **Behavioral RED/GREEN test** — `tests/baseline-output-no-skill.md` vs.
   `tests/baseline-output-with-skill.md`. The skill-loaded run must produce a file at
   `architecture/c4/c1-<name>.md` with mermaid classDef block, element catalog,
   relationships table, and explicit cross-links to L2.
2. **Pressure test 1 — code/intent conflict** (`tests/pressure-conflict.md`): given docs
   that disagree with code, the skill must surface the conflict and ask, not silently pick one.
3. **Pressure test 2 — propagation** (`tests/pressure-propagation.md`): renaming a
   container at L2 must trigger proposed updates to the L1 parent and every L3 child file
   referencing it.
4. **Pressure test 3 — L4 untested property** (`tests/pressure-untested-property.md`):
   given a real invariant with no test, the skill must mark it **⚠ UNTESTED**, not omit it
   and not fabricate a test link.
5. **Pressure test 4 — Two-Tier Extraction at L3** (`tests/pressure-two-tier-l3.md`): given
   an L3 component-identification task, the skill must dispatch a Tier 1 sub-agent before
   Tier 2 verification, not single-pass identify.
