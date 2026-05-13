# Plan: engram#612 — `engram starting-points` subcommand

**Branch:** `opencode-plugin` (continuing the in-flight body of work — Luhmann utilities and promote machinery already live here).

**Goal:** Add `engram starting-points [--vault <path>]` that emits one wikilink per line: every MOC, plus the in-degree winner (Luhmann tie-break) of each MOC-less connected component. Then update the `recall` skill to consume it.

---

## Spec clarifications (decisions that #612 leaves implicit)

| Question | Decision |
|---|---|
| Broken wikilinks (target file doesn't exist) | Drop. Don't create phantom nodes. Edge ignored. |
| Self-links (note linking to itself) | Drop. Don't count toward in-degree. |
| Duplicate wikilinks within one note | Count as one edge, not many. |
| Output ordering | Globally sorted by Luhmann tree-order across all starting points (MOCs and component winners interleaved). Determinism over presentation. |
| Wikilink resolution key | Filename basename minus `.md` (e.g. `[[9o1.2026-05-10.cross-cutting-finds-asymmetry]]` → `9o1.2026-05-10.cross-cutting-finds-asymmetry.md`). Not the Luhmann ID. |
| Graph model | Undirected for component detection, directed for in-degree counting. |
| Empty vault / missing subdir | Empty vault: empty output, exit 0. Missing subdir (e.g. no `Fleeting/`): treat as empty, no error. |
| Vault path source | `--vault <path>` flag, env `ENGRAM_VAULT_PATH`, no hardcoded default (error if neither provided — vault location is user-specific, no sensible default exists). |

These will be encoded as test cases (RED phase) before implementation.

---

## Architecture

### New package: `internal/luhmann/`

Pure utilities, extracted from `internal/cli/luhmann.go` so non-CLI packages can use them.

**Move:**
- `ParseID(id string) ([]string, error)` (was `parseLuhmannID`)
- `Less(a, b string) bool` (was `luhmannLess`)
- `SortIDs(ids []string)` (was `sortLuhmannIDs`)

**Keep in `internal/cli/luhmann.go`:**
- `nextChild`, `nextLuhmannID`, `nextSibling`, `nextTopLevel`, `directChildSegments`, `maxDigitSeg`, `maxLetterSeg`, `nextLetter` — these are promote-specific.
- The CLI file's calls to the moved functions become `luhmann.ParseID`, `luhmann.Less`, `luhmann.SortIDs`.

**Tests:** existing `luhmann_test.go` tests for the moved functions go to `internal/luhmann/luhmann_test.go`. Promote-specific tests stay.

### New package: `internal/vaultgraph/`

Pure logic (no I/O). All filesystem reads injected via `VaultFS` interface.

```
internal/vaultgraph/
├── vaultgraph.go       // public API: StartingPoints(VaultFS, vaultPath) ([]string, error)
├── parser.go           // filename + wikilink body parsing
├── graph.go            // graph type, builder, in-degree, components (union-find)
├── selector.go         // per-component starting-point selection logic
├── *_test.go           // blackbox tests in package vaultgraph_test
```

**Public API:**
```go
type VaultFS interface {
    ReadDir(path string) ([]DirEntry, error)
    ReadFile(path string) ([]byte, error)
}

type DirEntry struct {
    Name  string
    IsDir bool
}

// StartingPoints returns one canonical wikilink per starting point,
// globally sorted by Luhmann tree order.
func StartingPoints(fs VaultFS, vaultPath string) ([]string, error)
```

**Internal types:**
```go
type note struct {
    basename string  // "9o1.2026-05-10.cross-cutting-finds-asymmetry"
    luhmann  string  // "9o1" (empty for fleetings)
    isMOC    bool    // true if file lives in MOCs/
    outgoing []string // basenames this note links to (deduped, self-link removed, broken-removed)
}
```

### CLI adapter: `internal/cli/startingpoints.go`

```go
type StartingPointsArgs struct {
    VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=path to vault directory"`
}

func runStartingPoints(ctx context.Context, a StartingPointsArgs, stdout io.Writer) error {
    if a.VaultPath == "" { return errVaultPathRequired }
    points, err := vaultgraph.StartingPoints(&osVaultFS{}, a.VaultPath)
    if err != nil { return fmt.Errorf("starting-points: %w", err) }
    for _, p := range points {
        fmt.Fprintln(stdout, "[["+p+"]]")
    }
    return nil
}

type osVaultFS struct{}
func (*osVaultFS) ReadDir(p string) ([]vaultgraph.DirEntry, error) { /* os.ReadDir */ }
func (*osVaultFS) ReadFile(p string) ([]byte, error)                { /* os.ReadFile */ }
```

Wired in `targets.go` alongside other subcommands.

---

## TDD slicing

Each slice = one RED → GREEN → REFACTOR → `targ check-full` cycle, one commit per slice.

### S1: Extract `internal/luhmann/` package
- Move ParseID/Less/SortIDs (rename to exported); update CLI callers; verify `targ test` and `targ check-full` clean.
- No new behavior; pure refactor.

### S2: Wikilink body parser (pure, no I/O)
- `parseWikilinks(body []byte) []string` — regex `\[\[([^\]\n]+)\]\]`, dedupe.
- RED: example tests for body with 0/1/many links, duplicates, multiline, links inside code fences (decision: count code-fenced links — too many edge cases to special-case; vault discipline doesn't put wikilinks in code).
- Property test: result is always a deduped subset of substrings between `[[` and `]]` markers.

### S3: Filename → note metadata parser
- `parseFilename(filename, dir string) (basename, luhmann string, isMOC bool, ok bool)`.
- Skips non-`.md` files. Extracts Luhmann ID via regex (or via reused promote.go logic; verify reusability).
- Tests: MOC filename, Permanent filename, Fleeting filename (no Luhmann), filename without Luhmann prefix, non-md file.

### S4: VaultFS interface + scanner
- `scanVault(fs VaultFS, vaultPath string) ([]note, error)` — reads MOCs/, Permanent/, Fleeting/; returns notes with parsed body wikilinks.
- Tests use a fake `VaultFS` with in-memory entries.
- Edge cases: missing subdir, empty vault, file read errors propagate.

### S5: Graph builder
- `buildGraph(notes []note) graph` — produces adjacency, drops broken/self/duplicate edges.
- Tests: constructed `[]note` fixtures producing known graphs. Property: `len(edges) ≤ Σ len(outgoing)`.

### S6: Connected components (union-find)
- `components(g graph) [][]string` — undirected components.
- Tests: isolated nodes, two-node component, fully-connected, mixed.
- Property: number of components = nodes − unique-edges-in-spanning-forest.

### S7: Per-component starting-point selection
- `selectStartingPoints(comp []string, g graph) []string` — MOCs first; otherwise highest in-degree, Luhmann tiebreak (earliest wins), all-on-tie if Luhmann can't break.
- Tests: MOC component (single MOC, multi MOC), MOC-less with clear winner, MOC-less with Luhmann tie, MOC-less with unbreakable tie (e.g., two fleetings with no IDs and same in-degree).
- Property: always emits ≥1 starting point per non-empty component.

### S8: Top-level `StartingPoints` composition + global sort
- Composes S2–S7. Final output sorted via `luhmann.SortIDs` adapted for full basenames (sort by the Luhmann prefix; nodes without a Luhmann ID sort last, lexically among themselves).
- Tests: end-to-end fixture vault → expected output list.
- Property: idempotent (running twice on same input yields same output); deterministic across runs.

### S9: `osVaultFS` adapter
- Real os.ReadDir / os.ReadFile wrappers. Thin; covered indirectly by smoke test, no unit tests (per project convention for adapters).

### S10: CLI wiring (`runStartingPoints`, `StartingPointsArgs`, targets.go entry)
- Mirrors `runShow` / `runList` shape.
- Unit test: `runStartingPoints` with a fake VaultFS, captures stdout, verifies output format.

### S11: End-to-end smoke test
- Builds binary or uses `executeForTest`. Fixture vault under `testdata/` with: 1 multi-MOC component, 1 single-MOC component, 1 MOC-less component with Luhmann winner, 1 MOC-less with tiebreak, 1 MOC-less with unbreakable tie. Asserts exact stdout.

---

## Verification (Phase 6)

After S11 passes, run binary against real vault:
```
go run ./cmd/engram starting-points --vault /Users/joe/repos/personal/agent-memory
```
Expected: every MOC ([[5.2026-05-09.llm-rationalization-patterns]], [[7.2026-05-09.zettelkasten-as-agent-memory]], etc.) plus the today's-orphans-cluster anchor (likely a node with high in-degree like [[14]] or [[16]] or whichever wins). Visually traverse 2-3 emitted starting points to confirm they reach the missing parents (9o, 10c, 1c, etc.) within a couple of wikilink hops.

---

## Skill update (Phase 7)

`skills/recall/SKILL.md` (symlinked into `~/.claude/skills/recall/`) — edited via `superpowers:writing-skills` (TDD: baseline test → edit → pressure test).

Changes:
1. Replace the "read MOCs and orphans" cascade entry with: "Shell out to `engram starting-points --vault <path>`. The binary returns one wikilink per starting point — every MOC plus one anchor per MOC-unreachable component. Read each emitted note. Traverse outgoing wikilinks adaptively from there."
2. Remove the line "orphans have no outgoing links to follow" (line 93 surrounding paragraph).
3. Remove the entire "Use Luhmann sibling proximity as a free signal" rule (line 97 paragraph) — Luhmann adjacency is encoded in wikilinks; traversal handles it.
4. Update glossary/preamble where needed: rename "orphan" to "MOC-unreachable note" or just drop the term.

This is a separate commit in the skills directory (not part of the engram PR).

---

## Risks & open questions

- **Skill lives in this repo now.** `skills/recall/` is symlinked into `~/.claude/skills/recall/`, so the skill update lands in the same engram PR as the binary work — no separate skills-repo commit needed.
- **Existing `extractLuhmannFromFilename` in promote.go** is similar to the filename parser I'd write for vaultgraph. Can be lifted/shared via the new `internal/luhmann/` package if the code reads cleanly that way; otherwise duplicate the small regex.
- **Relationship to in-flight opencode-plugin work.** Building on the existing branch is correct (per user). When this branch eventually merges, both #612 and the existing promote work land together.

---

## Estimated work

11 slices × ~30-60 min each (RED+GREEN+REFACTOR+commit). Half a day if subagents run in parallel where independent (e.g., S2 and S3 can be parallel; S5/S6/S7 are sequential).

Skill update + verification: another 30-60 min.
