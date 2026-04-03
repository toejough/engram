# Engram

Self-correcting memory for LLM agents. Measures impact, not just frequency — memories that don't improve outcomes get diagnosed and fixed.

## Design Principles

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges.
- **Pure Go, no CGO:** TF-IDF instead of ONNX. External embedding API if vector similarity needed.
- **Plugin form factor:** Skills for behavior (engram-agent, use-engram-chat-as, recall), slim Go binary for computation (recall, show).
- **Test hard-to-test code by refactoring for DI**, not by writing integration tests around I/O.
- **Content quality > mechanical sophistication.** Measure impact, not just frequency.
- **Test categorization:** Unit tests verify business logic via DI + mocks (imptest). Integration tests verify wiring of thin I/O wrappers with real dependencies. If a function has business logic AND I/O, refactor to separate them — don't write an integration test around the whole thing.

## Code Quality

- **Use `targ` for all build/test/check operations** — NEVER run `go test`, `go vet`, `go build` directly
  - Tests: `targ test`
  - Lint + coverage: `targ check-full`
  - Build: `targ build`
  - Don't reverse-engineer targ's behavior — treat it as a black box
- Use `targ check-full` to get ALL errors at once — `targ check` stops early and you'll play whack-a-mole
- Minimal code that solves the problem — don't over-engineer
- Use `make([]T, 0, capacity)` when size is known
