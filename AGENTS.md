# Engram

Persistent memory for LLM agents, backed by an agent-memory zettelkasten vault. Six skills read from and write to the vault on demand: `recall`, `learn`, `please`, `route`, `curate`, and `write-memory`. Each skill ships in `agent-instructions/skills/` via `engram update`. Additionally, `engram update`/`engram register-skills` (`--accept`/`--decline`/`--dry-run`/`--adopt`) offer a companion vault runbook note for every skill, command and Pi prompt template in the default folders — not only engram's six: Claude user skills and commands (symlinks followed), claude.ai-synced skills, Pi user and agents skills, Pi-configured skill/prompt paths and packages, enabled plugin skills and commands, and project-local Claude sources plus Pi sources for trusted projects. Each note is identified by a source-qualified skill key (`claude:route`, `claude:cmd:audit`, `superpowers:brainstorming`, `commit:cmd:commit`, `project:github.com/toejough/engram:cmd:opsx:apply`), with a slug derived from the key and `skill_key`/`skill_source` recorded on the note; every key is source-qualified; engram's own skills are keyed by the folder they are found in (e.g. `claude:route`, `pi:route`), identical copies collapse, and every preamble names the note's `skill_source`. Removal offers are limited to sources that were read successfully. Declines are remembered by hash, keyed by skill key, in vault-root `skill-registrations.json`; nothing is written without a terminal, and offers are answered by key, key-prefix pattern, or `@scope`. New/refreshed notes are pending offers the `curate` skill reviews. `recall` and `learn` invoke `write-memory` natively as a skill at their write sites, not reached by basename/wikilink. `please` is one note (its former three sub-runbooks merged into the single note/skill body). The runbook concept itself (captured runbooks with triggers, the shim follow-frame) remains unchanged — only the "skills are replaced by runbooks" direction is reversed.

## Core Principles

- **Verify, don't guess.** Always read source code and documentation before guessing at formats, paths, or API contracts. Never assume directory structures or payload fields — verify first.
- **Follow the documentation workflow before implementation.** For every behavior change, including code, tooling, configuration, and agent instructions, read and follow the required implementation workflow in [docs/AGENTS.md](docs/AGENTS.md#required-implementation-workflow), even when no documentation edits were initially planned.


## Directory Structure

```
engram/
├── cmd/engram/        # CLI binary entry point — single-statement main() + checker-thin per-group functions supplying raw I/O primitives (targ check-thin-api-enforced)
├── internal/          # Non-public implementation
│   ├── chunk/         # Splits memory sources (stripped transcripts, markdown) into embedding-sized chunks for the auto-ingested vector space (pure string logic, no I/O)
│   ├── cli/           # CLI command wiring (targ targets) + composition root: cli.NewDeps builds every production adapter from injected cli.Primitives
│   ├── cluster/       # k-means clustering with silhouette-based auto-K, for recall clustering
│   ├── context/       # Transcript processing for LLM agents
│   ├── debuglog/      # Tail-friendly debug logger
│   ├── embed/         # Embedder interface + Hugot/GoMLX backend, sidecar I/O, state classification
│   ├── luhmann/       # Luhmann zettelkasten ID parsing/sorting
│   ├── transcript/    # Claude Code session transcript reading
│   ├── update/        # `engram update` subcommand
│   └── vaultgraph/    # Wikilink graph analysis of the vault
├── agent-instructions/
│   ├── skills/        # Source for all six skills — recall, learn, please, route, curate, write-memory
│   └── guidance/      # Source for the deployable ambient guidance docs — recall-firing (`recall.md`), delegation-firing (`delegate.md`), learn-firing (`learn.md`), and runbook-follow-frame (`shim.md`) — synced by `engram update --with-guidance` to canonical paths in `~/.claude/engram/guidance/` (Claude Code) and `~/.pi/agent/engram/guidance/` (Pi); compat symlinks at old flat paths keep existing `@import` lines in CLAUDE.md / AGENTS.md resolving (792a11m in docs/specs/product/install-and-update.md); activated via `@import`
├── dev/               # Build tooling (targ definitions, linter configs)
└── docs/              # Organized per docs/AGENTS.md: changes/, specs/, the authoring guides, GLOSSARY.md, ROADMAP.md, images/
```

## Key Files

- `cmd/engram/main.go` — CLI entry point (wiring-only: single-statement main() composing `cli.Primitives` from checker-thin per-group functions of raw capability references; `targ check-thin-api`-enforced)
- `internal/cli/primitives.go` — Composition root (`cli.Primitives` + `cli.NewDeps`, which builds every production adapter from the injected primitives)
- `internal/cli/targets.go` — Subcommand wiring
- `agent-instructions/skills/{learn,recall,please,route,curate,write-memory}/SKILL.md` — Skill definitions for all six skills (each may also carry a companion `skill-claude-<name>` vault runbook note offered by `engram update`/`engram register-skills`)
- `dev/targs.go` — Build targets (targ definitions)
- `docs/AGENTS.md` — documentation conventions and the index of authoring guides; start here for any `docs/` work
- `docs/GLOSSARY.md`, `docs/ROADMAP.md` — project terms; planned, parked and shipped work
- `docs/specs/product/`, `docs/specs/technical/` — the canonical behavior record: one file per capability and per component

## Design Principles

Authority: the technical specs (792b) and the rationale under their requirements — the bullets below are the agent-facing summary.

- **DI everywhere:** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O through injected interfaces. Wire at the edges. Lint-enforced (depguard/forbidigo + `targ check-thin-api`, #700). The boundary is two-sided: `internal/` holds interfaces and ALL logic (adapter composition, error wrapping, lifecycle — built by `cli.NewDeps` from injected `cli.Primitives`) but imports no I/O packages; `cmd/engram` is wiring-only (a single-statement `main()` composing `cli.Primitives` from checker-thin per-group functions of raw capability references such as `os.ReadFile` and `time.Now`, zero orchestration, coverage-exempt). A new I/O capability needs a `Primitives` field plus internal composition. Seeded `math/rand/v2` stays legal.
- **Pure Go, no CGO.** External API only for LLM operations. Embedder runs through GoMLX's `simplego` backend (CGO not required).
- **Skills + binary:** Skills for behavior (recall, learn, please, route, curate, write-memory), slim Go binary for computation.
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
- If `check-nils-for-fail` or `check-coverage-for-fail` fail flakily (e.g. `signal: killed`, or coverage numbers that look corrupted/inconsistent across identical runs) in a long-lived sandbox, check `df -h /tmp` and `du -sh /tmp/engram-e2e-bin* 2>/dev/null` first — `/tmp` here is often a RAM-backed tmpfs, and a leaked `engram-e2e-bin*` dir from a prior `go test` run can fill it. `GOMEMLIMIT`/`GOGC` tuning is not the fix (confirmed wrong; `internal/cli/testbinary_test.go`'s `TestMain` already cleans this up on a normal exit) — don't reach for it.
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

## Agent skills

### Issue tracker

Issues and specs live as GitHub issues at github.com/toejough/engram. Use the `gh` CLI for all operations; it infers the repo from `git remote -v`.

- Create: `gh issue create --title "..." --body "..."` (heredoc for multi-line bodies). Read: `gh issue view <number> --comments`. List: `gh issue list --state open --json number,title,body,labels,comments --jq '...'` with `--label`/`--state` filters.
- Comment: `gh issue comment <number> --body "..."`. Labels: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`. Close: `gh issue close <number> --comment "..."`.
- "Publish to the issue tracker" means create a GitHub issue. "Fetch the relevant ticket" means `gh issue view <number> --comments`.
- **PRs as a request surface: no.** Set it to `yes` if external PRs become feature requests; then PRs run through the same labels and states via `gh pr view|diff|list|comment|edit|close`, keeping only authors with association `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR` or `NONE`. Issues and PRs share one number space, so resolve a bare `#42` with `gh pr view 42` and fall back to `gh issue view 42`.

Wayfinding (`/wayfinder`): the **map** is one issue labelled `wayfinder:map` holding Notes / Decisions-so-far / Fog; a **child ticket** is an issue linked as a GitHub sub-issue (`gh api` on the sub-issues endpoint; without sub-issues, a task list in the map body plus `Part of #<map>` atop the child), labelled `wayfinder:<research|prototype|grilling|task>`.

- **Blocking** uses GitHub's native issue dependencies: `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`, where the id is the blocker's numeric database id (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`), not its number or `node_id`. Without dependencies, put `Blocked by: #<n>, #<n>` atop the child body. A ticket is unblocked when every blocker is closed.
- **Frontier**: the map's open children minus any with an open blocker or an assignee; first in map order wins. **Claim**: `gh issue edit <n> --add-assignee @me`, the session's first write. **Resolve**: comment the answer, close the issue, then append a context pointer to the map's Decisions-so-far.

### Triage labels

The five canonical triage roles map one-to-one to label strings in this repo's tracker (label string equals role name): `needs-triage` (maintainer must evaluate), `needs-info` (waiting on reporter), `ready-for-agent` (fully specified, ready for an AFK agent), `ready-for-human` (requires human implementation), `wontfix` (will not be actioned). `wontfix` already exists on GitHub; `triage` creates the other four on first use.

### Domain docs

Single-context repo with its own documentation system, not the `CONTEXT.md`/`docs/adr/` default. Before exploring, read `docs/AGENTS.md` (conventions and the index of guides), the relevant capability and component files of the behavior record (`docs/specs/product/` and `docs/specs/technical/`), `docs/GLOSSARY.md`, and this file. If a file does not exist, proceed silently.

- Use the glossary's vocabulary when naming a domain concept in an issue title, refactor proposal, hypothesis or test name. If the concept is missing, either you are inventing language the project does not use or the glossary has a gap worth noting.
- If your output contradicts a requirement or its rationale in the specs, surface it explicitly ("Contradicts 792b8c1, but worth reopening because...") rather than silently overriding. A behavior change goes through a change under `docs/changes/` per `docs/SPECIFICATION-LIFECYCLE-GUIDE.md`.
