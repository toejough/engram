# projctl Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-18

## Active Technologies
- SQLite (embeddings.db) — no schema changes needed (014-skill-gen-compliance)
- Go 1.25+ (module: `github.com/toejough/projctl`) + go-sqlite3, sqlite-vec, onnxruntime_go (E5-small-v2), gomega (testing) (015-continuous-eval-memory)
- SQLite (embeddings.db) — WAL mode, busy_timeout=5000ms (015-continuous-eval-memory)

- Go 1.21+ + database/sql (SQLite via go-sqlite3), encoding/json, strings, gomega (testing) (014-skill-gen-compliance)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.21+

## Code Style

Go 1.21+: Follow standard conventions

## Recent Changes
- 015-continuous-eval-memory: Added Go 1.25+ (module: `github.com/toejough/projctl`) + go-sqlite3, sqlite-vec, onnxruntime_go (E5-small-v2), gomega (testing)
- 014-skill-gen-compliance: Added Go 1.21+ + database/sql (SQLite via go-sqlite3), encoding/json, strings, gomega (testing)

- 014-skill-gen-compliance: Added Go 1.21+ + database/sql (SQLite via go-sqlite3), encoding/json, strings, gomega (testing)

<!-- MANUAL ADDITIONS START -->

## Active Work: Memory System Rebuild

When user says "continue", "resume", or similar without other context:
1. Read `docs/rebuild/state.md` for current phase and next action
2. Read `docs/rebuild/prompt.md` for full process instructions (especially the current phase section)
3. Resume from the "Next Action" in state.md — do NOT ask "what would you like to work on?"
4. Announce what phase you're in and what you're about to do

State persistence (belt and suspenders):
- **Write-ahead (primary):** After each substantive rebuild interaction (phase transition, decision made, interview question answered), update `docs/rebuild/state.md` with current phase, specific next action, context files, and session summary. The next action must be concrete enough that a fresh session can start immediately.
- **Stop hook (safety net):** A Stop agent hook in `.claude/settings.local.json` infers the current phase from artifact file existence and updates state.md if it's stale. This catches cases where write-ahead was missed.

## Coverage Workflow

When a function is hard to test, that's a signal to refactor it for DI — not to write integration tests around it. If achieving coverage requires real I/O (filesystem, network, ONNX runtime, LLM APIs), the function needs dependency injection, not a more elaborate test setup. Never write a test that depends on external state like `~/.claude/models/` existing.

<!-- MANUAL ADDITIONS END -->
