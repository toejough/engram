# C4 L3 Targets + Cross-Diagram ID Registry — Design

**Date:** 2026-04-26
**Status:** Approved (brainstorming complete)
**Owner:** Joe
**Predecessors:**
- `docs/superpowers/specs/2026-04-25-c4-diagram-skill-design.md` (the c4 skill)
- `docs/superpowers/specs/2026-04-25-c4-l1-targets-design.md` (L1 targets)
- `dev/c4_l2.go` (L2 build target, shipped)

## Purpose

Two coupled goals:

1. **Add deterministic build/audit support for C4 Level 3 (Component) diagrams** so `/c4 create 3` mirrors the L1/L2 workflow: spec JSON in, canonical markdown out, structural audit on the markdown.
2. **Enforce a single E-ID namespace across all C4 diagrams** in the repo. Today the existing diagrams already drift (L1 `E3 = "Claude (Anthropic API)"` vs L2 `E3 = "Claude Code"`; L1 `E6 = "Local filesystem"` vs L2 `E10 = "Engram memory store"`). The hand-written L3 files I wrote on 2026-04-25 collide on focus IDs (`E6` and `E8` reused inside their own diagrams). Without a global namespace this gets worse as L3/L4 expand.

The unifying principle: **the JSON specs are the source of truth; the registry is a compute-on-demand projection of them, with no separate persisted state.**

## Scope (in)

- One new build target: `c4-l3-build` (JSON L3 spec → canonical L3 markdown).
- One new read-only target: `c4-registry` (walks `architecture/c4/c*.json`, emits unified ID/name view + conflict findings).
- Audit generalization in `c4-audit`:
  - `validRefinedByFile` keyed off the file's own `level` (`level=N` → children must match `^c{N+1}-`).
  - New finding `code_pointer_unresolved` for L3 catalog code-pointer paths that don't resolve on disk.
  - Always-on registry cross-check: derive the registry on each audit run and emit `id_name_drift` / `name_id_split` findings for the audited file.
- One-shot reconciliation of existing L1/L2 diagrams to a single namespace before any of the above ships in workflows.

## Scope (out)

- L4 targets (property ledgers). Future spec.
- Auto-assignment of E-IDs by the build (explicit IDs only — agent picks from registry view).
- Auto-discovery of L3 components (Tier 2 work — `c4-l3-components`, `c4-l3-edges`. Tracked as follow-ups, not in this spec).
- Auto-rewriting of cross-file propagation (`children`, "Refined by") — agent still applies edits with user approval.
- Anything in the engram binary or `internal/`.

## File Layout

```
dev/
├── c4.go                    # extend: parameterize validRefinedByFile, add cross-check helpers
├── c4_l2.go                 # unchanged
├── c4_l3.go                 # NEW: L3Spec + c4L3Build + emit helpers
├── c4_registry.go           # NEW: registry derivation + c4Registry target
├── c4_test.go               # extend: audit cross-check + level-parameterized child prefix
├── c4_l3_test.go            # NEW
├── c4_registry_test.go      # NEW
└── testdata/c4/
    ├── valid_l3.json
    ├── valid_l3.md          # golden
    ├── invalid_l3_*.json    # build fail-fast fixtures
    ├── audit_l3_*.md        # audit fixtures
    └── registry_*/           # multi-file fixtures (c1+c2+c3 sets)
```

`package dev`, `//go:build targ`. Same conventions as existing `c4.go`.

## Target: `c4-registry`

### Purpose

Walk every JSON spec under `architecture/c4/`, parse each as the appropriate spec type (`L1Spec` / `L2Spec` / `L3Spec`), and emit a single unified projection of element IDs and names plus any cross-spec inconsistencies.

The agent calls this **before drafting a new spec** so it knows which IDs are taken and which names already have IDs.

### Flags

- `--dir PATH` (default `architecture/c4`): directory to scan.

### Output (stdout, JSON)

```json
{
  "schema_version": "1",
  "scanned_dir": "architecture/c4",
  "scanned_files": ["c1-engram-system.json", "c2-engram-plugin.json"],
  "elements": [
    {
      "id": "E2",
      "names": ["Engram", "Engram plugin"],
      "appears_in": [
        {"file": "c1-engram-system.json", "name": "Engram"},
        {"file": "c2-engram-plugin.json", "name": "Engram plugin"}
      ]
    }
  ],
  "names_to_ids": [
    {
      "name_pattern": "Anthropic",
      "ids": ["E3", "E5"],
      "appears_in": [
        {"file": "c1-engram-system.json", "id": "E3", "name": "Claude (Anthropic API)"},
        {"file": "c2-engram-plugin.json", "id": "E5", "name": "Anthropic API"}
      ]
    }
  ],
  "conflicts": [
    {
      "kind": "id_name_drift",
      "id": "E3",
      "detail": "E3 has different names across files",
      "evidence": ["c1-engram-system.json: 'Claude (Anthropic API)'", "c2-engram-plugin.json: 'Claude Code'"]
    },
    {
      "kind": "name_id_split",
      "name_pattern": "Anthropic",
      "detail": "Likely-same element appears under different IDs",
      "evidence": ["E3 'Claude (Anthropic API)' in c1", "E5 'Anthropic API' in c2"]
    }
  ]
}
```

### Conflict kinds

| Kind | Trigger |
|---|---|
| `id_name_drift` | Same `E<n>` ID resolves to different element names across files. Detected by exact-name comparison after trimming. |
| `name_id_split` | Two distinct E-IDs share a substring-matched name pattern across files. Heuristic, **report only** — the LLM/human reconciles. Substring length ≥ 5 chars to avoid noise. |
| `id_collision_within_file` | One spec file declares two elements with the same explicit `E<n>`. Hard error in the spec, but registry still emits the finding. |

### Behavior

- Reads all `*.json` under `--dir`, ignores any with `schema_version != "1"`.
- Dispatches per `level` field: parse with `L1Spec`, `L2Spec`, or `L3Spec`.
- Builds two indices:
  - `id → set of (file, name)` for ID-name drift detection.
  - `normalized-name token → set of (file, id, name)` for name-id splits. Tokenization: lowercase, split on non-alphanumeric, drop tokens shorter than 5 chars.
- Sorts elements by ID for stable output.
- Returns nonzero exit only on parse / I/O failures. Conflicts surface in output but don't fail the target — the agent and the build/audit decide what to do.

### Errors

- Cannot read `--dir` → exit 1.
- Spec file with malformed JSON → log to stderr, skip the file, continue.
- Spec file with unrecognized `level` → log to stderr, skip.

## Target: `c4-l3-build`

### Purpose

Render canonical L3 markdown from a JSON spec, validating against the cross-diagram registry.

### Flags (mirrors `c4-l1-build`/`c4-l2-build`)

- `--input PATH` (required): path to L3 JSON spec.
- `--check`: verify existing `.md` matches generated; non-zero on diff.
- `--noconfirm`: overwrite existing `.md` without prompting.

### Spec schema (`L3Spec`)

```json
{
  "schema_version": "1",
  "level": 3,
  "name": "engram-cli-binary",
  "parent": "c2-engram-plugin.md",
  "preamble": "Refines L2's E8 engram CLI binary into nine internal components…",
  "focus": {
    "id": "E8",
    "name": "engram CLI binary"
  },
  "elements": [
    {
      "id": "E14",
      "name": "main.go",
      "kind": "component",
      "subtitle": "process entry",
      "responsibility": "Process entry. Calls cli.SetupSignalHandling and forwards cli.Targets into targ.Main.",
      "code_pointer": "../../cmd/engram/main.go"
    },
    {
      "id": "E1",
      "from_parent": true,
      "name": "Developer",
      "kind": "person",
      "responsibility": "Engineer who triggers slash-commands."
    }
  ],
  "relationships": [
    {"from": "Developer", "to": "main.go", "description": "execs the binary as a subprocess (Bash tool)", "protocol": "Subprocess exec"}
  ],
  "drift_notes": [],
  "cross_links": {"refined_by": []}
}
```

### Schema rules (validated by `loadAndValidateL3Spec`)

- `schema_version == "1"`, `level == 3`.
- `name` matches `^[a-z][a-z0-9-]*$` and equals filename slug minus the `c3-` prefix.
- `parent` non-empty; resolved relative to the spec dir, must exist.
- `preamble` non-empty.
- `focus.id` matches `^E\d+`; `focus.name` non-empty.
- **`focus` does NOT appear in `elements`.** It is rendered as the wrapping subgraph AND as a catalog row, but the spec keeps these separate so we don't accidentally double-list it.
- Every `element.id` is explicit (no auto-assignment) and matches `^E\d+`. No within-file duplicates.
- Every `element.kind ∈ {person, external, container, component}`.
- `from_parent: true` requires `kind ∈ {person, external, container}` (a carry-over neighbor).
- `kind == "component"` requires `code_pointer` (relative path) and `from_parent` to be unset/false.
- `kind != "component"` forbids `code_pointer`.
- `relationships[*].from` and `to` reference either an element name or `focus.name`.

### Registry validation (cross-file)

After in-file validation, `c4-l3-build` derives the registry over all *other* JSON specs in the same directory and enforces:

- For every element with `from_parent: true`: its `(id, name)` pair must equal an existing registry entry. Mismatch on either → fail with `name_drift` or `id_drift` error citing the conflicting file.
- For every component element (kind `component`): its `id` must NOT exist in the registry under a different name. If the registry already has the same `id` with the same `name`, that's accepted (idempotent re-run after a rename). If the same `name` exists under a different `id`, fail with `name_id_split` error.
- `focus` is treated as a `from_parent`-style reference and validated the same way.

### Output (canonical L3 markdown)

```markdown
---
level: 3
name: engram-cli-binary
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: <SHA from `git rev-parse --short HEAD`>
---

# C3 — engram CLI binary (Component)

> Container in focus: **E8 · engram CLI binary** from [c2-engram-plugin.md](c2-engram-plugin.md).

<preamble>

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    <neighbor nodes outside the focus subgraph, one per from_parent element>

    subgraph e8 [E8 · engram CLI binary]
        <component nodes only>
    end

    <edges with R<n> labels>

    <class statements>
    <click directives, including a click for e8 → focus catalog row anchor>
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e8-engram-cli-binary"></a>E8 | engram CLI binary | Container in focus | <focus.responsibility from spec, defaulted from parent if not provided> | <code_pointer if any, else em-dash> |
| <a id="e14-main-go"></a>E14 | main.go | Component | <responsibility> | [../../cmd/engram/main.go](../../cmd/engram/main.go) |
| <a id="e1-developer"></a>E1 | Developer | Person | <responsibility> | — |

## Relationships

<same shape as L1/L2>

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **E8 · engram CLI binary**)
- Siblings: <auto-discovered: any other c3-*.md whose parent matches this one>
- Refined by: <from spec.cross_links.refined_by, or "(none yet)">

## Drift Notes
<from spec.drift_notes, formatted as in L1/L2>
```

### Sibling auto-discovery

The build scans `architecture/c4/c3-*.md` and, for each one whose front-matter `parent` matches the input spec's `parent`, emits a sibling cross-link. This is read-only (sibling files aren't edited).

### Catalog column

L3 catalog uses **Code Pointer** in column 5 (instead of L1/L2's "System of Record"). For carry-over neighbors and the focus, where there's no code pointer, the cell renders an em-dash (`—`).

### Mermaid emission rules

- Mermaid variable names: lowercase E-ID (`e1`, `e8`, `e14`).
- Focus subgraph: `subgraph <lowercase id> [E<n> · <name>]`.
- Components inside the focus subgraph; everything else outside.
- Component shape: rectangle (`[…]`), class `:::component`.
- Component label: `E<n> · <name>` plus `<br/><subtitle>` if subtitle is set.
- Click directives for **every** node, including the focus subgraph (it now has a catalog row, so the click resolves).

### Errors

- `--input` empty → exit 1.
- Schema validation failure → exit 1 with the offending field cited.
- Registry conflict → exit 1 with the conflicting file + ID/name cited.
- `git rev-parse --short HEAD` failure → exit 1.

## `c4-audit` extensions

### 1. Level-parameterized child prefix

Today: `validRefinedByFile = ^c2-[a-z0-9-]+\.md$` (hard-coded for L1).

Change: derive expected child prefix from the audited file's own `level`:

| Level | Expected child prefix |
|---|---|
| 1 | `^c2-` |
| 2 | `^c3-` |
| 3 | `^c4-` |
| 4 | (no children allowed) |

A `children: ["…"]` or `Refined by` entry that violates the prefix becomes finding `child_prefix_invalid`.

### 2. Code-pointer resolution (L3 only)

For each catalog row in an L3 file with a markdown link in the **Code Pointer** column, resolve the link target relative to the markdown file's directory and `os.Stat` it. Missing file → finding `code_pointer_unresolved` (same severity as `parent_missing`).

### 3. Always-on registry cross-check

On every `c4-audit` invocation, derive the registry from `architecture/c4/c*.json` (excluding the audited file's own JSON when one exists with the matching slug). Then, for the audited markdown:

- Parse out every E-ID + accompanying name from the catalog rows.
- Cross-check each `(id, name)` pair against the registry view.
- Mismatch on name (same ID, different name) → `id_name_drift` finding citing the registry file.
- Same name under a different ID elsewhere → `name_id_split` finding.

If the audited file has no registry-side counterpart (e.g., spec deleted but markdown still on disk), emit a single `registry_orphan` finding rather than per-row noise.

### 4. Subgraph orphan handling

The audit's existing orphan check treats subgraphs as nodes (per `mermaidSubgraphRe`). With L3's focus-has-a-catalog-row rule, no exemption is needed: the focus subgraph's `E<n>` label resolves to its own catalog row like any other node.

## L1/L2 Reconciliation (one-shot, lands first)

Before shipping `c4-l3-build` and the new audit checks, reconcile the existing L1/L2 diagrams to a single namespace so the registry comes online with zero conflicts.

### Reconciliation rules

- Anything appearing at multiple levels keeps its **first-issued ID** (lowest level wins).
- Where L2 introduced a different ID for an existing L1 element (e.g., `Anthropic API` becoming `E5` in L2), the L2 spec is rewritten to use the L1 ID.
- Where L2 renamed an L1 element (e.g., `Local filesystem` → `Engram memory store`), the rename is preserved but the L1 ID is reused (L1 spec also gets the rename if the new name is more accurate, otherwise both L1's old name and L2's new name are reflected as a Drift Note).
- Where L2 introduced new elements with no L1 counterpart, IDs above the L1 maximum are kept; gaps are not backfilled.

### Concrete reconciliation list (current state, predicted)

The registry's first run will report at least:

1. **L1 `E3 "Claude (Anthropic API)"` ↔ L2 `E5 "Anthropic API"`.** Same external. Reuse `E3` in L2 for `Anthropic API`. L1 retains `E3` but the name is updated to `Anthropic API` (drop the parenthetical — `Claude` is the model brand, `Anthropic API` is the system).
2. **L2 `E3 "Claude Code"` is a new element** not present in L1. The L1 spec didn't model the Claude Code harness as a separate external. Renumber L2 `Claude Code` to whatever L1's lowest unused ID is. (Current L1 max is `E6`, so `E7` works.)
3. **L1 `E6 "Local filesystem"` ↔ L2 `E10 "Engram memory store"`.** Same store, renamed. Reuse `E6`. The rename is justified (the L2 name is more specific). Drop L2's `E10`.
4. **L1 `E4 "Git"` and `E5 "Skill catalog"`.** L2 doesn't model these because they fold into the engram-plugin scope. No conflict; left as-is.
5. **L2 `E6 Skills`, `E7 Hooks`, `E8 engram CLI binary`** are L2-introduced internal containers. Above L1 max — kept.

After reconciliation, the canonical namespace through L2 is:

| ID | Element | Introduced at |
|---|---|---|
| E1 | Developer (was "User" in L1) | L1 |
| E2 | Engram plugin | L1 |
| E3 | Anthropic API | L1 |
| E4 | Git | L1 |
| E5 | Skill catalog | L1 |
| E6 | Engram memory store (was "Local filesystem") | L1 |
| E7 | Claude Code | L2 |
| E8 | Claude Code memory surfaces | L2 |
| E9 | Skills | L2 |
| E10 | Hooks | L2 |
| E11 | engram CLI binary | L2 |

Note: the L2 spec's current numbering will shift. The L2 `.md` regenerates from JSON. The hand-written L3 files referencing L2 IDs (`E6`, `E7`, `E8`) become invalid and have to be regenerated under `c4-l3-build` against the new namespace anyway.

### Reconciliation deliverables (in order)

1. **Update `c1-engram-system.json`** — rename `E1 User → Developer`, rename `E3 Claude (Anthropic API) → Anthropic API`, rename `E6 Local filesystem → Engram memory store`. Rebuild markdown.
2. **Update `c2-engram-plugin.json`** — drop redundant E-IDs, renumber `Claude Code` and the internal containers per the table above. Rebuild markdown.
3. **Delete the three hand-written L3 markdown files** — `c3-skills.md`, `c3-hooks.md`, `c3-engram-cli-binary.md`. They'll be regenerated under `c4-l3-build` once it lands. (The hand-written content has the focus-collision bug; not worth porting.)

The reconciliation lands as a single PR. The follow-up PRs add `c4-registry`, `c4-l3-build`, and the audit extensions.

## Implementation Order

1. **Reconciliation** of c1/c2 specs + delete the broken hand-written L3 files. (No new code; data-only PR + golden-fixture refresh.)
2. **`c4-registry` target** — derives projection, emits findings, no callers yet.
3. **`c4-audit` cross-check** — wire `c4-registry` derivation into audit, add new finding kinds. Verify zero findings on the post-reconciliation set.
4. **`c4-audit` level-parameterized child prefix + L3 code-pointer check.**
5. **`c4-l3-build`** — the main feature.
6. **Regenerate c3-skills, c3-hooks, c3-engram-cli-binary** under `c4-l3-build` against the canonical namespace.
7. **Update `skills/c4/SKILL.md`** to invoke `c4-registry` before drafting and to invoke `c4-l3-build` instead of hand-authoring L3 markdown.

## Testing

Per-target unit tests under `dev/c4_test.go` / `dev/c4_l3_test.go` / `dev/c4_registry_test.go`:

- **Registry:** golden fixture sets under `dev/testdata/c4/registry_*/` containing 2–4 spec JSONs each. Verify projection output, conflict detection (`id_name_drift`, `name_id_split`, `id_collision_within_file`), and graceful handling of malformed files.
- **L3 build:** golden `valid_l3.json` → `valid_l3.md`. Fail-fast cases: missing focus, focus duplicated in elements, component missing code_pointer, neighbor with code_pointer, registry conflicts.
- **Audit cross-check:** fixtures with deliberate `id_name_drift` and `name_id_split` against a known registry set; verify findings.
- **Audit child-prefix:** L1 file with a `c3-` child reference, L2 file with a `c2-` reference; verify findings.
- **Audit code-pointer:** L3 fixture with a missing file path in the catalog; verify `code_pointer_unresolved` finding.

## Open Questions / Non-blocking

- Whether to ship `c4-l3-components` and `c4-l3-edges` (Tier 2) before `c4-l3-build` is widely used. Defer to follow-up spec; not blocking.
- Whether to flag a `c4-l4-build` design now to keep momentum. Out of scope here.
- Naming: `code_pointer` vs `source_pointer` vs `path` for the L3 catalog column. Going with `code_pointer` to match the existing template (`skills/c4/references/templates/c3-template.md`) and reduce churn.
