# Memory System Rebuild

This repo is being rebuilt from scratch as a Claude Code plugin for self-correcting LLM agent memory.

## Active Work

When user says "continue", "resume", or similar without other context:
1. Read `docs/state.md` for current phase and next action
2. Read `docs/prompt.md` for full process instructions
3. Resume from the "Next Action" in state.md — do NOT ask "what would you like to work on?"
4. Announce what phase you're in and what you're about to do

State persistence (belt and suspenders):
- **Write-ahead (primary):** After each substantive rebuild interaction (phase transition, decision made, interview question answered), update `docs/state.md` with current phase, specific next action, context files, and session summary. The next action must be concrete enough that a fresh session can start immediately.
- **Stop hook (safety net):** A Stop agent hook in `.claude/settings.local.json` infers the current phase from artifact file existence and updates state.md if it's stale.

## Process: Depth-First Vertical

See `docs/state.md` for the full process description. Key points:
- Group and prioritize at every layer (UC → REQ → DES/ARCH → Tests → Impl)
- Refactor the ENTIRE current layer before descending
- Dirty-mark descendants of changed nodes; resolve on visit
- Final depth-first sweep as safety net

## Design Principles

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges.
- **Pure Go, no CGO:** TF-IDF instead of ONNX. External embedding API if vector similarity needed.
- **Plugin form factor:** Hooks, skills, CLAUDE.md management, Go binary for computation.
- **Test hard-to-test code by refactoring for DI**, not by writing integration tests around I/O.
- **Content quality > mechanical sophistication.** Measure impact, not just frequency.
