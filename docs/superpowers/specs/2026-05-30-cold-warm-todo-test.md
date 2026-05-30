# Cold/warm convergence test — secret todo spec + orchestration

Date: 2026-05-30. A dead-end-rich, domain-matched instantiation of the
self-seeding cold-vs-warm memory test (see
`2026-05-29-memory-eval-harness-design.md` Run-1 findings and vault notes
241/242/244). The executor must **rediscover** these hidden requirements
through review rounds — the spec is NEVER shared with it.

## Experiment shape

- **Executor:** headless `claude -p` (sonnet) with recall+learn skills +
  engram binary, `ENGRAM_VAULT_PATH` = the phase vault,
  `CLAUDE_CONFIG_DIR` = arm config (keychain creds + skills + binary).
  Workspace = a fresh `/tmp` dir. In-phase iteration via `claude
  --resume <session-id> -p "<review feedback>"`.
- **Vague prompt (identical both phases):** "Build a command-line todo
  application in Go." Plus standard recall/learn usage + the
  non-interactive marker workaround (`engram transcript --mark --from
  all`) and an explicit `/learn` at the end of each round.
- **Reviewer (me):** run the app, score against the secret rubric below,
  return **spec-free** feedback (describe what's wrong / wanted, never
  hand over the spec).
- **Cold phase:** empty vault. Iterate ≤5 rounds to ~80% match. Each
  round the executor `/learn`s the feedback → seeds the vault.
- **Warm phase:** fresh session, vault = cold's populated vault. recall
  surfaces prior requirements → richer round-1 build. Iterate ≤5 rounds.
- **Metrics (both):** round-1 spec-match %, and rounds + turns + cost to
  reach the threshold. Compare cold vs warm.

## Secret rubric — 18 items (~80% = 15)

The executor is scored on how many of these it satisfies. It is told
NONE of them; they leak only as review feedback, one cluster at a time.

### Features (12)
1. Task fields: id, title, status (`todo`/`doing`/`done`), priority
   (`low`/`med`/`high`), tags ([]string), created + completed
   timestamps, optional due date.
2. Subcommands present: `add`, `list`, `start <id>` (→ doing), `done
   <id>`, `rm <id>`, `edit <id>`, `tag <id> <tag>`, `search <q>`,
   `archive`, `stats`, `undo`.
3. Default `list` sort: doing-first, then priority desc, then due-date
   asc.
4. Tag filtering: `list --tag <t>`.
5. Search: substring match across title + tags.
6. Archive: `archive` moves done tasks to a separate archive store;
   `list --archived` shows them.
7. Stats: counts by status + completion rate + overdue count.
8. Due dates with overdue highlighting in `list`.
9. Persistence at `$XDG_DATA_HOME/todo/tasks.json` (fallback
   `~/.local/share/todo/...`), **atomic write** (temp file + rename).
10. `list --json` machine output; default is an aligned human table.
11. `undo` reverts the last mutating command (single-level snapshot).
12. Colorized output for priority/overdue, auto-disabled when output is
    not a TTY or `NO_COLOR` is set.

### Architecture opinions (6)
13. DI everywhere: no direct `os`/filesystem/clock calls in core logic;
    inject `Store` (load/save), `Clock`, and an output `io.Writer`. Wire
    only in `main`.
14. Pure core / thin shell: command logic operates on an in-memory
    `TaskList` value; I/O strictly at the edges.
15. Stdlib-only, single binary, hand-rolled subcommand dispatch (no
    cobra / no third-party CLI or color deps).
16. Table-driven unit tests using an in-memory `Store` + a fake `Clock`
    — no real files or wall-clock in unit tests.
17. Sentinel + wrapped errors (`ErrNotFound`, `fmt.Errorf("...: %w")`).
18. No global mutable state; the app is constructed from its deps.

## Review-feedback drip plan (spec-free, clustered)

To make rediscovery genuinely costly, feedback reveals requirements a
cluster at a time, as a reviewer would:
- R1 reaction → core data model + missing subcommands + priority/status.
- R2 → sorting, tags, search, due dates/overdue.
- R3 → archive, stats, undo, json output, color/NO_COLOR.
- R4 → architecture: DI, pure core, stdlib-only, atomic writes.
- R5 → tests (table-driven + fakes), sentinel errors, no-globals.

(Order may compress if the executor volunteers items early.)

## Hypotheses
- **H1 (priming):** warm round-1 spec-match % > cold round-1 (recall
  front-loads requirements).
- **H2 (convergence):** warm reaches threshold in fewer rounds/turns/cost
  than cold.
- **H_null (overhead):** per note 241, recall overhead may still swamp
  savings — warm could match or underperform. That is a valid result.
