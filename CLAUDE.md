# Engram

Persistent memory for LLM agents, backed by an agent-memory zettelkasten vault. Two skills — `recall` and `learn` — read from and write to the vault on demand.

## Core Principles

- **Verify, don't guess.** Always read source code and documentation before guessing at formats, paths, or API contracts. Never assume directory structures or payload fields — verify first.

## Directory Structure

```
engram/
├── cmd/engram/        # CLI binary entry point
├── internal/          # Non-public implementation
│   ├── cli/           # CLI command wiring (targ targets)
│   ├── context/       # Transcript processing for LLM agents
│   ├── debuglog/      # Tail-friendly debug logger
│   ├── luhmann/       # Luhmann zettelkasten ID parsing/sorting
│   ├── tokenresolver/ # API token resolution
│   ├── transcript/    # Claude Code session transcript reading
│   ├── update/        # `engram update` subcommand
│   └── vaultgraph/    # Wikilink graph analysis of the vault
├── skills/            # Claude Code skill definitions (learn, recall)
├── opencode/          # OpenCode plugin (commands, skills)
├── dev/               # Build tooling (targ definitions, linter configs)
└── docs/              # Active design docs and research prompts
```

## Key Files

- `cmd/engram/main.go` — CLI entry point
- `internal/cli/targets.go` — Subcommand wiring
- `skills/{learn,recall}/SKILL.md` — Skill definitions
- `dev/targs.go` — Build targets (targ definitions)

## Design Principles

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges.
- **Pure Go, no CGO.** External API only for LLM operations.
- **Plugin form factor:** Skills for behavior (learn, recall), slim Go binary for computation.
- **Test hard-to-test code by refactoring for DI**, not by writing integration tests around I/O.
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

## Issue Workflow (Default)

When working through issues, the preferred default is:
1. **Parallel planning** — spawn planners for independent issues simultaneously
2. **Plan review with argumentation** — reviewer challenges each plan, argue to resolution
3. **Parallel implementation** — executors implement approved plans (worktrees for file conflicts)
4. **Implementation review with argumentation** — reviewer validates each implementation, argue to resolution
5. Use as many relevant skills as possible at each stage (brainstorming, writing-plans, TDD, etc.)
6. **Keep interacting agents alive together** — don't kill an executor until its reviewer ACKs (or argument resolves). Don't kill a planner until its reviewer signs off. Agents that need to argue must both be alive.

## Worktree & Merge Rules

- **Review before merge** — implementation review (with argumentation) must complete before any merge.
- **ff-only merges only** — always `git merge --ff-only`. No merge commits.
- **Rebase on main before merge** — after review passes, rebase the branch on main and re-test before merging.
- **Rebase loop** — if another branch merged ahead of you, rebase on the new main and re-test again before retrying the merge.
- **Never push without review** — all worktree work must be reviewed before pushing.

## Skill Editing

- **ALWAYS use the `superpowers:writing-skills` skill when editing any SKILL.md file.** No exceptions.
- This enforces TDD: baseline behavior test (RED), update skill (GREEN), verify behavioral change.
- Run pressure tests before marking skill edits complete.
