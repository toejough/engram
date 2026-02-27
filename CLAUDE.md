# Engram

Self-correcting memory for LLM agents. Measures impact, not just frequency — memories that don't improve outcomes get diagnosed and fixed.

## Active Work

When user says "continue", "resume", or similar without other context:
1. Read `docs/state.toml` for current cursor position and next action
2. Read `docs/prompt.md` for full process instructions
3. Resume from the cursor's `next_action` — do NOT ask "what would you like to work on?"
4. Announce what layer/group you're in and what you're about to do

State persistence (write-ahead): After each substantive interaction (node transition, decision made, flag set/cleared), immediately update `docs/state.toml`. Do not defer to session end. See the specification-layers skill for the TOML format.

## Process: Depth-First Tree Traversal

See the specification-layers skill for the full model. Key points:
- Tree of group nodes within layers, walked depth-first left-to-right
- Group and prioritize at EVERY layer (not just UC)
- `dirty` and `unsatisfiable` flags on nodes drive cursor behavior
- Refactor the ENTIRE layer only when a change is absorbed (not on escalation)
- Escalate without refactoring — rise until something absorbs

## Design Principles

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges.
- **Pure Go, no CGO:** TF-IDF instead of ONNX. External embedding API if vector similarity needed.
- **Plugin form factor:** Hooks, skills, CLAUDE.md management, Go binary for computation.
- **Test hard-to-test code by refactoring for DI**, not by writing integration tests around I/O.
- **Content quality > mechanical sophistication.** Measure impact, not just frequency.
