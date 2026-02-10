# Memory Optimization Verification Plan (ISSUE-177 through ISSUE-182)

All commands below have been tested against the installed binary.

## ISSUE-182: Duplicate header fix

```bash
projctl memory promote --review    # Run twice
grep -c "## Promoted Learnings" ~/.claude/CLAUDE.md  # Should be 1
```

The `appendToClaudeMD()` function now checks for an existing `## Promoted Learnings` section and appends entries into it instead of creating a new header each time.

## ISSUE-180: Session-aware ACT-R scoring

Session-aware scoring is internal — it automatically weights cross-session retrievals higher (1.5x multiplier when memories are retrieved across sessions >30min apart). There is no separate CLI output for this.

**How to verify it's working:** Query the same topic from different projects over multiple sessions (>30min apart). Over time, memories retrieved across sessions will rank higher than single-session memories with the same retrieval count.

```bash
projctl memory query "tdd" --project proj-a
# ... use projctl in a different session later ...
projctl memory query "tdd" --project proj-b
```

Cross-session memories will naturally float to the top of query results over time. The effect compounds — the more sessions a memory surfaces in, the stronger its score.

## ISSUE-181: Hybrid search (BM25 + vector)

```bash
# Use -v to confirm search method
projctl memory query "calculateSimilarity" -v
```

Verbose output shows:
- `Search method: hybrid (vector+BM25)` — confirms hybrid path is active
- `BM25 available: true` — FTS5 is enabled via `sqlite_fts5` build tag (included in `targ install`)

## ISSUE-178: Primacy ordering in context-inject

```bash
# Create a correction and a regular learning
projctl memory learn -m "Never use git checkout -- ." -t correction
projctl memory learn -m "prefer gomega for assertions"

# Verify corrections appear first
projctl memory context-inject
```

Corrections surface first in the output, before regular learnings. Verbose query also shows the type tag:

```bash
projctl memory query "git" -v
# Output: 1. (0.48) [correction] Never use git checkout -- .
```

## ISSUE-177: CLAUDE.md consolidation

```bash
projctl memory consolidate --claude-md
```

Shows proposals for:
- **Redundant entries**: items that exist in both CLAUDE.md and the memory DB (similarity >0.9)
- **Promotion candidates**: memories with 5+ retrievals across 3+ projects

## ISSUE-179: Pattern synthesis

```bash
projctl memory consolidate --synthesize
```

Reports `Patterns identified: N` — clusters similar memories and identifies recurring themes. Needs 3+ similar memories on the same topic to form a pattern (controlled by `--min-cluster-size`).

## Bug fixes included

### `--transcript` flag (extract-session)
The `--transcript` and `--memory-root` flags were broken due to incorrect targ struct tags (`targ:"--flag"` instead of `targ:"flag,name=flag,desc=..."`). Fixed in extract-session and context-inject.

### `--type` flag (learn)
The `--type` / `-t` flag was missing from the CLI despite being supported internally. Now wired through so corrections can be created from the command line.

### `--project` flag (query)
The `--project` / `-p` flag was missing from the query CLI args. Now wired through for retrieval tracking.

## Ongoing signal

The strongest "it's working" signal is that **context-inject output improves over time**:
- Corrections surface first (primacy ordering)
- Exact keyword matches appear (hybrid search, when BM25 available)
- Cross-session learnings rank higher (session-aware ACT-R)
- Duplicate/redundant entries get cleaned up (consolidation)

Run `projctl memory context-inject` periodically and compare the quality of what gets injected.
