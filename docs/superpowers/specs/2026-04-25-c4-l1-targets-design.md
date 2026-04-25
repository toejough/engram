# C4 L1 `dev/` Targets — Design

**Date:** 2026-04-25
**Status:** Approved (brainstorming complete)
**Owner:** Joe
**Predecessor:** `docs/superpowers/specs/2026-04-25-c4-diagram-skill-design.md` (the c4 skill itself, already shipped)

## Purpose

Add deterministic build/audit tooling for **C4 Level 1 (System Context) diagrams** so the `/c4 create 1` skill workflow can offload everything mechanical and the LLM only does judgment work (naming, prose, externals selection, conflict resolution).

## Scope (in)

- Four `targ` targets, all registered in a single `dev/c4.go` file via an `init()` function:
  1. `c4-l1-externals` — discover external-system candidates by walking the repo with Go AST analysis
  2. `c4-history` — wrap `git log` and emit structured JSON of recent commits + bodies
  3. `c4-l1-build` — accept a JSON spec describing an L1 diagram and emit canonical markdown
  4. `c4-audit` — structural validation of an L1 markdown file (rule 6 + front-matter + cross-link sanity)

## Scope (out)

- L2/L3/L4 targets (containers, components, property ledgers). Future work, separate specs.
- Anything in the engram binary or `internal/` — this is build tooling only.
- Auto-generation of the diagram from code alone (the LLM still picks/names/prose-writes).
- Mermaid PNG/SVG rendering (mermaid-cli is optional and only used for syntactic validation).

## File Layout

```
dev/
├── c4.go                          # all four target registrations + helpers
├── c4_test.go                     # unit tests
└── testdata/c4/
    ├── valid_l1.json              # canonical L1 input fixture
    ├── valid_l1.md                # expected canonical L1 output (golden)
    ├── invalid_*.json             # build fail-fast fixtures
    ├── audit_clean.md             # passes audit
    ├── audit_dirty.md             # multiple findings expected
    └── …
```

No new top-level packages; `package dev` with `//go:build targ` matching the existing convention in `dev/targs.go`.

## Targ Standards (must follow)

- File header: `//go:build targ` + `package dev`
- Registration via `targ.Register(targ.Targ(fn).Name("…").Description("…"))` inside `init()`
- Handler signature: `func(ctx context.Context) error`
- Non-nil error → targ converts to exit code 1; never call `os.Exit`
- JSON output to `stdout` (let callers pipe); diagnostic logs to `stderr`
- Per-target flags via `flag.FlagSet` scoped to the target, NOT global
- Composability via Unix pipes; no hardcoded "call X then Y" between targets
- Tests under `dev/c4_test.go`; fixtures under `dev/testdata/c4/`

## Target 1: `c4-l1-externals`

### Purpose
Walk the repo with Go AST analysis and emit a structured list of external-system candidates the LLM can consider when drafting an L1 diagram.

### Flags
- `--root PATH` (default `.`): module root to scan
- `--packages SPEC` (default `./...`): packages.Load pattern
- `--include-tests` (default false): include `_test.go` files in scan

### Output (stdout, JSON)
```json
{
  "schema_version": "1",
  "scanned_packages": ["github.com/toejough/engram/cmd/engram", "..."],
  "findings": [
    {
      "kind": "http_call",
      "target": "https://api.anthropic.com",
      "source": "internal/anthropic/client.go:42",
      "evidence": "http.NewRequestWithContext(ctx, \"POST\", anthropicAPIURL, …)"
    },
    {
      "kind": "fs_path",
      "target": "$XDG_DATA_HOME/engram/memory/feedback",
      "source": "internal/memory/store.go:18",
      "evidence": "filepath.Join(xdg.DataHome, \"engram\", \"memory\", \"feedback\")"
    },
    {
      "kind": "exec",
      "target": "git",
      "source": "internal/cli/dispatch.go:201",
      "evidence": "exec.Command(\"git\", \"log\", \"--format=…\")"
    },
    {
      "kind": "env_read",
      "target": "ANTHROPIC_API_KEY",
      "source": "internal/anthropic/client.go:30",
      "evidence": "os.Getenv(\"ANTHROPIC_API_KEY\")"
    }
  ]
}
```

### Detection rules

| `kind` | What to detect | AST pattern |
|---|---|---|
| `http_call` | Network endpoints contacted | `CallExpr` with `Sel.Name ∈ {NewRequest, NewRequestWithContext, Get, Post, Put, Delete}` from `net/http`; URL argument extracted from string literal or constant lookup |
| `fs_path` | User-config / data directory boundaries | `CallExpr` with `Sel.Name ∈ {UserHomeDir, UserConfigDir, UserCacheDir}` from `os`; calls into `github.com/adrg/xdg` package; string literals beginning with `~/`, `/etc/`, `/var/`, `/tmp/` passed to `os.Open*`/`os.WriteFile`/`os.Create*` |
| `exec` | Subprocess invocations | `CallExpr` with `Sel.Name = Command` from `os/exec`; first string arg literal extracted |
| `env_read` | Configuration environment variables | `CallExpr` with `Sel.Name ∈ {Getenv, LookupEnv}` from `os`; key argument extracted |

### Behavior

- Loads packages with `packages.Config{Mode: NeedSyntax|NeedTypes|NeedImports|NeedFiles}`
- Walks each file's `*ast.File` syntax tree with `ast.Inspect`
- Resolves caller package via type info (so `http.NewRequest` from `net/http` is detected, not just any function literally named `NewRequest`)
- Sorts findings deterministically by `(kind, source, target)` so output is stable across runs
- Logs scan progress to stderr: package name as scanned, finding count summary
- Empty repo / no findings → exits 0 with `{"findings": []}`

### Errors

- Cannot load packages → return error (exit 1)
- AST parse failure for one file → log to stderr, continue with other files; counts toward summary

## Target 2: `c4-history`

### Purpose
Wrap `git log` and emit structured JSON of commit metadata + bodies for use as intent input.

### Flags
- `--paths PATH...` (repeatable): only commits touching these paths
- `--since DURATION` (default unset): cutoff (e.g., `90d`, `2026-01-01`)
- `--limit N` (default 50): max commits returned
- `--grep PATTERN` (default unset): commit-message regex filter

### Output (stdout, JSON only — no markdown mode)

```json
{
  "schema_version": "1",
  "filters": {
    "paths": ["cmd/engram"],
    "since": "30d",
    "limit": 20,
    "grep": ""
  },
  "commits": [
    {
      "sha": "df51bc93",
      "date": "2026-04-25T13:46:10Z",
      "author": "joe",
      "subject": "test(c4): pressure test 3 PASS - L4 untested property handling",
      "body": "Subagent generated property ledger…",
      "files_changed": [
        {"path": "skills/c4/tests/pressure-untested-property.md", "status": "A"}
      ]
    }
  ]
}
```

### Implementation
- Shell out to `git log` with format `--format=%H%x09%aI%x09%an%x09%s%x00%B%x00`
- `--name-status` for files, separator `\0` between commits, robust to bodies with newlines/quotes
- Parse the output line-by-line in Go (no `jq`); state-object parser per line of NUL-delimited records
- Validates `git` is on PATH at handler start; clear error if not

### Errors
- `git` not found → return error (exit 1)
- `git log` non-zero exit → return error wrapping stderr (exit 1)

## Target 3: `c4-l1-build`

### Purpose
Read a JSON spec describing an L1 diagram and emit canonical markdown next to the input file. Single source of truth: rule-6 violations are structurally impossible because IDs/anchors/click directives are derived from the same JSON.

### Flags
- `--input PATH` (required): path to JSON spec
- `--check` (default false): verify the existing `.md` matches what the builder would produce; exit 1 with diff on stderr if not. (Use case: pre-commit / CI gate that source and rendered markdown stay in sync.)
- `--no-confirm` (default false): skip prompt before overwriting existing `.md`

### Behavior
- Output path derived: `<input-without-ext>.md`. Example: `architecture/c4/c1-engram-system.json` → `architecture/c4/c1-engram-system.md`.
- If output exists and differs from generated content, prompt user before overwriting unless `--no-confirm`.
- Front-matter `last_reviewed_commit` is computed at build time via `git rev-parse --short HEAD`; the JSON does NOT carry it (the field is build-time, not source-of-truth).

### Input JSON schema (v1)

```json
{
  "schema_version": "1",
  "level": 1,
  "name": "engram-system",
  "parent": null,
  "preamble": "Engram is a Claude Code plugin that gives the agent persistent, query-ranked memory. This diagram shows who and what Engram interacts with at the system boundary; it deliberately hides the CLI binary, hooks, on-disk stores, and skills (those live at L2).",
  "elements": [
    {
      "name": "Joe",
      "kind": "person",
      "subtitle": "developer using Claude Code",
      "responsibility": "Developer who triggers /prepare, /recall, /remember, /learn, /migrate and authors the work that produces memories",
      "system_of_record": "Human, at a Claude Code session"
    },
    {
      "name": "Engram plugin",
      "kind": "container",
      "is_system": true,
      "subtitle": null,
      "responsibility": "Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries",
      "system_of_record": "This repository (github.com/toejough/engram)"
    },
    {
      "name": "Claude Code",
      "kind": "external",
      "subtitle": "agent harness",
      "responsibility": "Agent harness that loads the plugin, dispatches skills, fires hooks, …",
      "system_of_record": "Anthropic Claude Code CLI"
    }
  ],
  "relationships": [
    {
      "from": "Joe",
      "to": "Claude Code",
      "description": "Invokes slash-commands and writes prompts that trigger skill auto-invocation",
      "protocol": "Claude Code CLI / TTY",
      "bidirectional": false
    }
  ],
  "drift_notes": [],
  "cross_links": {
    "refined_by": [
      { "file": "c2-engram-containers.md", "note": "decomposes Engram plugin into …" }
    ]
  }
}
```

### Schema validation (fail-fast)

A single bad field aborts the build with a clear error message. Validators:

- `schema_version == "1"` (else: unknown schema version)
- `level == 1` (this target is L1-only; reject L2+)
- `name` matches `^[a-z][a-z0-9-]*$`; fail on capitals/spaces/underscores
- `parent == null` (L1 has no parent)
- `preamble` non-empty
- `len(elements) ≥ 2`; each element's `kind` must be one of `person`, `external`, `container`
- Exactly one element has `is_system: true`, and that element's `kind` must be `container`
- All element names are unique (case-sensitive)
- Element fields `name` and `responsibility` non-empty; `system_of_record` non-empty; `subtitle` may be `null`
- `len(relationships) ≥ 1`
- For each relationship: `from` and `to` must reference an element name (the system is just the element with `is_system: true`)
- `bidirectional` is a boolean; default false if absent
- `cross_links.refined_by[].file` matches `^c2-[a-z0-9-]+\.md$`

Failure mode: print error to stderr with JSON path of the bad field, exit 1. No partial output.

### Slug + ID assignment (deterministic)

- Each element in `elements` gets `E<i+1>` in the order they appear (so first element is E1, second E2, etc.). The element with `is_system: true` gets whatever index it sits at — author-controlled placement, no special-casing in the numbering.
- Each relationship in `relationships` gets `R<i+1>`.
- Slug rule: lowercase, replace non-`[a-z0-9]` runs with `-`, trim `-` from ends. (E.g., `Engram plugin` → `engram-plugin`; `Claude Code memory surfaces` → `claude-code-memory-surfaces`.)
- Anchor: `e<n>-<slug(name)>` and `r<n>-<slug(from)>-<slug(to)>`.
- If two anchors collide (same slug from two distinct names — rare but possible), append `-2`, `-3`, … in source order. Builder logs the collision to stderr.

### Markdown emission

The builder uses `text/template` with a fixed template inlined as a Go string (not a separate file — must work from the binary regardless of cwd). Output structure mirrors the canonical c1 file we already have:

1. Front-matter: `level`, `name`, `parent`, `children` (always `[]` for new L1; reserved for future), `last_reviewed_commit`
2. `# C1 — <name-of-the-is-system-element> (System Context)`
3. `preamble` paragraph
4. Mermaid block: classDef + one node per element (shape per `kind`: `person` → stadium `([Name])`, `external` → rounded `(Name)`, `container` → rectangle `[Name]`) + relationship edges (`-->` for unidirectional, `<-->` for bidirectional) + class assignments + click directives. The `is_system: true` flag is informational metadata only — rendering still uses `:::container` per the project mermaid convention. (A future v2 could add a distinct `:::system` style.)
5. `## Element Catalog` table with anchored ID column. The `is_system: true` element gets the `Type` "The system in scope" instead of "Container" in the table; all other containers are "Container".
6. `## Relationships` table with anchored ID column
7. `## Cross-links` section (Parent always `none (L1 is the root).`; Refined by from `cross_links.refined_by`)
8. `## Drift Notes` section, only emitted if `drift_notes` is non-empty

### Idempotence + diff property

`c4-l1-build INPUT.json && c4-l1-build INPUT.json` produces a byte-identical `.md` (modulo `last_reviewed_commit` if HEAD moved). Tested.

### Errors
- Invalid JSON → exit 1 with line/col + message
- Schema validation failure → exit 1 with field path + reason
- Cannot read input file → exit 1
- Cannot write output and `--check` not set → exit 1
- `--check` mode: if generated content != existing file content, print unified diff to stderr, exit 1

## Target 4: `c4-audit`

### Purpose
Structural audit of an L1 markdown file. Catches drift introduced by manual edits or by a broken builder.

### Flags
- `--file PATH` (required): markdown file to audit
- `--strict` (default false): treat info-level findings as errors too (reserved for future use; v1 has no info-level findings)

### Behavior

Reports **all** findings (no fail-fast) and exits 1 if any finding exists. Exits 0 only if the file is fully clean.

### Findings (all rule-6 + structural)

| Finding | Detection |
|---|---|
| `front_matter_missing` | No YAML front-matter block at top of file |
| `front_matter_field_missing` | Required field absent: level, name, parent, children, last_reviewed_commit |
| `level_invalid` | level not in {1,2,3,4} (v1 only checks structural; semantic L1 checks happen if level==1) |
| `name_filename_mismatch` | front-matter name != filename slug |
| `last_reviewed_commit_invalid` | SHA doesn't resolve via `git rev-parse` |
| `parent_missing` | (L≥2) front-matter parent file doesn't exist |
| `child_missing` | front-matter children entry doesn't exist |
| `mermaid_block_missing` | No fenced ` ```mermaid ` block |
| `classdef_missing` | classDef block absent or doesn't define `person`, `external`, `container` |
| `node_id_missing` | Mermaid node label without `E\d+` prefix |
| `edge_id_missing` | Mermaid edge label without `R\d+:` prefix |
| `node_orphan` | Node ID `En` has no matching catalog row |
| `catalog_orphan` | Catalog row ID `En` doesn't appear in any node label |
| `edge_orphan` | Edge ID `Rn` has no matching relationships row |
| `relationships_orphan` | Relationships row ID `Rn` doesn't appear in any edge label |
| `click_missing` | Node has no `click NODE href "#anchor"` directive |
| `click_target_unresolved` | `click` href points to a non-existent anchor |
| `anchor_missing` | Catalog or relationships row lacks `<a id="…"></a>` |
| `mermaid_render_failed` | If `mmdc` is on PATH, the block fails to parse |

### Output (stdout, plain text by default; `--json` for structured)

```
architecture/c4/c1-engram-system.md: 2 findings

[1] node_id_missing line 19: node "engram[Engram plugin]" lacks E<n> prefix
[2] click_target_unresolved line 36: click engram href "#e2-engram-plugin" but no <a id="e2-engram-plugin"> exists

exit 1
```

`--json` form:
```json
{
  "schema_version": "1",
  "file": "architecture/c4/c1-engram-system.md",
  "findings": [
    {"id": "node_id_missing", "line": 19, "detail": "..."},
    {"id": "click_target_unresolved", "line": 36, "detail": "..."}
  ]
}
```

### Errors
- Cannot read file → exit 1 (treated as a finding bundle of one)
- `git rev-parse` not available → skip the `last_reviewed_commit_invalid` check, log a warning to stderr, continue (don't fail the audit just because git is missing)

## Skill Workflow After Targets Land

The c4 skill's `create 1` workflow gets sharpened to:

1. `targ c4-l1-externals` → JSON of external candidates
2. `targ c4-history --since 90d` → JSON of recent commit bodies
3. Read CLAUDE.md, `docs/`, `engram recall` (judgment input)
4. LLM produces `architecture/c4/c1-<name>.json` (with all judgment fields filled)
5. `targ c4-l1-build --input architecture/c4/c1-<name>.json` → emits the `.md`
6. `targ c4-audit --file architecture/c4/c1-<name>.md` → trivially passes
7. Show user the markdown for approval; commit both `.json` and `.md`

The SKILL.md update for this is a follow-on task, not part of this spec.

## Testing

- `dev/c4_test.go` covers each target.
- `c4-l1-build`: golden-file tests using `dev/testdata/c4/valid_l1.json` ↔ `valid_l1.md`. Idempotence test (run twice, expect byte-identical output modulo SHA).
- `c4-l1-build` schema validation: one fixture per failure mode, expect specific error message.
- `c4-audit`: paired fixtures `audit_clean.md` (zero findings) and `audit_dirty.md` (every finding type represented at least once). Verify exit code and finding list.
- `c4-l1-externals`: synthetic fixture under `dev/testdata/c4/scanrepo/` with a tiny Go module containing one `http.NewRequest`, one `os.Getenv`, etc.; verify all four `kind`s detected.
- `c4-history`: dispatch a temp git repo via `t.TempDir() + git init + git commit` (real git, no stubs) — gives a real-binary integration test of the parsing path
- All tests follow engram's TDD discipline (TestT<N>_… naming via `TARG_BASELINE_PATTERN`).

## Linter Compliance

- Cyclomatic complexity ≤ 10 per function (cyclop)
- Variable names long enough (varnamelen)
- Tagged switches on equality (staticcheck QF1002)
- Modern Go: `bytes.Cut`, `strings.SplitSeq`, `strings.CutPrefix`
- Sentinel errors via `var Err…` not inline
- Errors wrapped with context

## Out of Scope (Restated)

- L2/L3/L4 build/audit targets
- Property-ledger generation
- Anything in `internal/` or the engram binary
- mermaid rendering to image formats
- Auto-extraction of element names/responsibilities from code (LLM judgment work stays the LLM's)

## Acceptance Criteria

- [ ] `dev/c4.go` exists with four targets registered
- [ ] `targ c4-l1-externals` produces deterministic JSON for the engram repo with at least one `http_call`, one `fs_path`, one `env_read` finding
- [ ] `targ c4-history --since 30d` produces JSON of recent commits including bodies
- [ ] `targ c4-l1-build` round-trips: build a JSON describing the existing `c1-engram-system.md`, generate, and `c4-audit` the result with zero findings
- [ ] `targ c4-audit architecture/c4/c1-engram-system.md` passes (zero findings) on the current canonical file
- [ ] `targ c4-audit` against a hand-broken copy reports each broken-rule kind with line numbers
- [ ] `dev/c4_test.go` covers each target with golden + failure-mode fixtures
- [ ] `targ check-full` clean
