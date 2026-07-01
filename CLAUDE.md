# Engram

Persistent memory for LLM agents, backed by an agent-memory zettelkasten vault. Two skills — `recall` and `learn` — read from and write to the vault on demand. A third skill, `please`, orchestrates end-to-end work by sequencing recall, learn, and other available skills around a user's `<ask>`, with adversarial review gates over the plan, refactors, docs, and outward prose. A fourth skill, `route`, encodes the delegate-everything doctrine: it guides subagent selection (agent type, model, effort) rather than doing object-level work itself. `please` consults it when assigning gate reviewers.

## Core Principles

- **Verify, don't guess.** Always read source code and documentation before guessing at formats, paths, or API contracts. Never assume directory structures or payload fields — verify first.

## Directory Structure

```
engram/
├── cmd/engram/        # CLI binary entry point
├── internal/          # Non-public implementation
│   ├── chunk/         # Splits memory sources (stripped transcripts, markdown) into embedding-sized chunks for the auto-ingested vector space (pure string logic, no I/O)
│   ├── cli/           # CLI command wiring (targ targets)
│   ├── cluster/       # k-means clustering with silhouette-based auto-K, for recall clustering
│   ├── context/       # Transcript processing for LLM agents
│   ├── debuglog/      # Tail-friendly debug logger
│   ├── embed/         # Embedder interface + Hugot/GoMLX backend, sidecar I/O, state classification
│   ├── luhmann/       # Luhmann zettelkasten ID parsing/sorting
│   ├── transcript/    # Claude Code session transcript reading
│   ├── update/        # `engram update` subcommand
│   └── vaultgraph/    # Wikilink graph analysis of the vault
├── skills/            # Source for the recall, learn, please, and route skills
├── commands/          # Source for OpenCode slash commands
├── guidance/          # Source for the deployable recall-firing guidance (activated via CLAUDE.md `@import`)
├── dev/               # Build tooling (targ definitions, linter configs)
└── docs/              # Active design docs, research prompts, and C4 architecture diagrams
```

## Key Files

- `cmd/engram/main.go` — CLI entry point
- `internal/cli/targets.go` — Subcommand wiring
- `skills/{learn,recall,please,route}/SKILL.md` — Skill definitions
- `dev/targs.go` — Build targets (targ definitions)
- `docs/architecture/c1-system-context.md` — L1 C4 system context diagram + sequence diagrams for the four key flows (recall, learn, please, update)

## Design Principles

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges.
- **Pure Go, no CGO.** External API only for LLM operations. Embedder runs through GoMLX's `simplego` backend (CGO not required).
- **Skills + binary:** Skills for behavior (learn, recall), slim Go binary for computation.
- **Embed-on-write:** Every note gets a sibling `.vec.json` sidecar on `engram learn`. The bundled MiniLM-L6 model (`minilm-l6-v2@384`) is `go:embed`-ed into the binary from `internal/embed/assets/model/` (git-lfs tracked). Sidecars are stamped with the model_id so future swaps require explicit `engram embed apply --force`.
- **Test hard-to-test code by refactoring for DI**, not by writing integration tests around I/O.
- **Test categorization:** Unit tests verify business logic via DI + mocks (imptest). Integration tests verify wiring of thin I/O wrappers with real dependencies. If a function has business logic AND I/O, refactor to separate them — don't write an integration test around the whole thing.

## Code Quality

- **Use `targ` for all test/lint/check operations** — NEVER run `go test`, `go vet` directly
  - Tests: `targ test`
  - Lint + coverage: `targ check-full`
  - Install the binary: `go install ./cmd/engram` (there is no `targ build` target — targ covers test/lint/check, not binary install)
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
