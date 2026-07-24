# Plan: #697 top-level folder reorg + #698 FEATURES.md plain-language rewrite

Cycle: /please, 2026-07-24. Cross-check ran first: no LEDGER row, vault note, or issue
comment records a prior disposition for either issue — both genuinely open.

## Ask (verbatim scope)

- **#697:** move all agent-specific language (`commands/`, `skills/`, `guidance/`) into a new
  top-level folder `agent-instructions/` (keep the three as subfolders). Goal: top level
  sensible to a human — `commands` reads like `cmd`, `guidance` reads like `docs`.
- **#698:** "plan language" sweep of `docs/FEATURES.md` — segments that reference cycle
  internals without antecedents ("the split", "chunk-index I/O load", "honoring a
  recently-updated standard") must become self-contained plain English.

Out of scope: untracked top-level clutter (`engram` binary, `feeds.json`, `links.json`,
`notes.json`, `subscriptions.json`, `coverage.out` — none are git-tracked); any deploy-target
change (`~/.claude/skills` etc. stay as-is); OpenCode harness behavior.

## Unit A — #697 reorg (first)

1. **RED:** update `internal/update` + `internal/cli` tests that model the repo layout
   (`internal/cli/invariants_u1_test.go:338-342`, `internal/update/update_test.go:227`) to
   expect `agent-instructions/{skills,commands,guidance}`; run `targ test` — fails.
2. **GREEN:** `git mv commands skills guidance` into `agent-instructions/`; point
   `internal/update/update.go:208-210` (`filepath.Join(source.Root, ...)`) at the new
   subpaths; `targ test` passes.
3. **REFACTOR + doc scrub:** apply the disposition list below; re-grep after apply
   (`\b(skills|commands|guidance)/` + `agent-instructions` + godoc comments) until clean.
4. **Verify with the real binary:** `go install ./cmd/engram`, then `engram update --dry-run`
   and `engram update` from a non-repo cwd — and assert **nonzero op counts for each of
   skills, commands, AND guidance** in the dry-run report (planCommandCopies and
   planGuidanceCopies silently return nil on a missing source dir — update.go:744/786 — so
   "it ran without error" does NOT prove the commands/guidance joins are right).
   `targ check-full` green.

### Doc-surface disposition list (author-grepped: `\b(skills|commands|guidance)/`, godoc comments included)

| File | Refs | Disposition |
| --- | --- | --- |
| `internal/update/update.go:208-210` | 3 | update — the load-bearing source joins |
| `internal/update/update.go` godoc/comment lines (2, 23, 582-583) | ~4 | update where they name repo-source paths; keep where they name deploy targets |
| `internal/cli/invariants_u1_test.go` (~12), `internal/update/update_test.go` (~35), `internal/update/runner_test.go` (~18), `internal/cli/update_deps_test.go` (~3) | ~68 | update repo-layout fixtures; deploy-target (`.claude/...`) fixtures keep |
| `internal/cli/update.go:290,362` | 2 | update — `fmt.Fprintf` user-facing output naming source paths |
| `CLAUDE.md` (4), `README.md` (6+), `docs/README.md` (2), `docs/GLOSSARY.md` (6), `docs/FEATURES.md` (3 folder names on line 69), `docs/ROADMAP.md` (3) | ~24 | update path refs to `agent-instructions/...` (per-line counts are indicative; the re-grep after apply is the completeness check, not these counts) |
| `docs/architecture/c1-system-context.md` (5), `c2-containers.md` (5), `c3-components.md` (2), `adr.md` (1) | 13 | update path refs AND diagram labels naming the folders |
| `dev/eval/LEDGER.md`, `dev/eval/cumulative/**` READMEs/fixtures, `docs/superpowers/plans/*` (prior cycles) | many | keep — vintage-stamped historical records; paths were valid at vintage. Executor re-checks each for a live-path use and updates only those (note 383: keep-verdicts must not create newly-misleading text). **Live vs historical test:** a ref a future runner must FOLLOW to operate the harness today (e.g. a cumulative README's "edit `skills/recall/SKILL.md` to select the arm" instruction) = live → update; a ref describing what a past cycle measured/edited (e.g. a LEDGER evidence column citing `skills/please/SKILL.md:81-92` at its vintage) = historical → keep |
| `skills/*/tests/*.md` internal relative refs | few | move with the dir; executor greps inside `agent-instructions/` post-move for now-broken relative paths |
| `dev/eval/adapters_test.go` | 2 | keep — refs are deploy-target (`cfgDir/skills`), not repo layout |
| `.gitignore:36-37` (`skills/*/result.toml`, `skills/*/test-coverage.md`) | 2 | update — anchored patterns stop matching after the move; re-root under `agent-instructions/` |
| `.claude/` project files (`skills/commit.md`, `skills/engram-go-conventions.md`, `commands/commit.md`, `rules/go.md`) | 0 | keep — scanned (`grep -rE '\b(skills|commands|guidance)/' .claude/`), no repo-path refs found; executor re-runs the same grep post-move as confirmation |

## Unit B — #698 FEATURES.md rewrite (after A, so path refs are final)

1. **RED (non-code analogue):** dispatch a fresh-context reader agent over
   `docs/FEATURES.md` (249 lines) with no repo context: list every segment it cannot parse
   self-containedly (unresolved referents, plan language). Baseline list recorded in the
   cycle scratchpad. Issue's two named examples must appear or the probe is under-sensitive.
2. **GREEN:** rewrite flagged segments in plain English — name every referent explicitly
   (e.g. "the timing breakdown showed that reading the on-disk chunk index, not running the
   embedding model, is what makes recall's query slow"). Keep the entry-per-capability
   structure and `why:`/`validation:` pointers intact.
3. **Verify:** re-run the fresh-reader probe — zero unparseable segments; spot-check that no
   factual claim drifted from its LEDGER/ADR source during rewrite.

## Gates

- Gate A (this plan): ask-alignment, code-alignment, docs/diagrams-alignment (verifies the
  disposition list independently), clarity/standards.
- Gate B: design-fit after each unit's refactor.
- Gate C: every touched doc (relevance + clarity/cohesion).
- Gate D: commit messages + issue-close text.

Commits: one per unit (`refactor: move agent instruction sources under agent-instructions/`,
`docs: rewrite FEATURES.md in plain language`), `AI-Used: [claude]` trailer, ff-only main.
