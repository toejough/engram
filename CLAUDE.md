# Engram

Self-correcting memory for LLM agents. Measures impact, not just frequency — memories that don't improve outcomes get diagnosed and fixed.

## Active Work

When user says "continue", "resume", or similar without other context:
1. Read `docs/state.toml` for current cursor position and next action
2. Read `docs/prompt.md` for full process instructions
3. Resume from the cursor's `next_action` — do NOT ask "what would you like to work on?"
4. Announce what layer/group you're in and what you're about to do

State persistence (write-ahead): After each substantive interaction (node transition, decision made, flag set/cleared), immediately update `docs/state.toml`. Do not defer to session end. See the specification-layers skill for the TOML format.

**Context preservation:** During long sessions, before context approaches limits, write a brief `docs/session-context.md` capturing: (a) constraints discovered this session, (b) patterns/workarounds found, (c) what's been tried and failed.

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
- **Test categorization:** Unit tests verify business logic via DI + mocks (imptest). Integration tests verify wiring of thin I/O wrappers with real dependencies. If a function has business logic AND I/O, refactor to separate them — don't write an integration test around the whole thing.

## Code Quality

- **Use `targ` for all build/test/check operations** — NEVER run `go test`, `go vet`, `go build` directly
  - Tests: `targ test`
  - Lint + coverage: `targ check`
  - Build: `targ build`
  - Don't reverse-engineer targ's behavior — treat it as a black box
- When `targ check` reports a failure, assume there are MORE failures behind it — run repeatedly until fully clean
- Minimal code that solves the problem — don't over-engineer
- Use `make([]T, 0, capacity)` when size is known

## Nilaway + Gomega Compatibility

nilaway doesn't recognize gomega assertions as nil guards. Required patterns:
- After `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` before accessing values
- Use `g.Expect(err).To(MatchError(...))` instead of `err.Error()`
- Add explicit nil guards before field access on pointers
- For test helpers returning `(*T, error)`, nil-check the pointer after asserting no error

## Known Model Defaults to Avoid

Generate code that passes linters on first commit:
- Name constants instead of magic numbers: `const maxRetries = 3`, not bare `3`
- Descriptive variable names: `memory`, `pattern`, `score` — not `m`, `p`, `s`
- Wrap errors with context: `fmt.Errorf("finding similar: %w", err)` not bare `return err`
- Use sentinel errors: `var ErrNotFound = errors.New(...)` not inline `fmt.Errorf(...)`
- Add `t.Parallel()` to every test and subtest (with no shared mutable state)
- Use `http.NewRequestWithContext` not `http.Get`
- Use `crypto/rand` not `math/rand`
- Line length under 120 chars
