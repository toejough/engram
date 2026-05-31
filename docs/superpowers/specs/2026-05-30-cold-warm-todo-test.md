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

---

## Results (2026-05-30)

### Cold/warm convergence (n=1)
- Cold phase: 5 review rounds drip-feeding 18 requirements. Round-1 spec-match ~11%; reached ~13/18 working by R4; the R5 DI refactor broke the build (undefined Todo). 62 turns, $3.64.
- End-of-session `learn` to seed the vault **failed: "Prompt is too long"** — a 5-round session can't be resumed for bulk capture. Only 3 incidental notes got written by earlier auto-fires.
- Warm round-1 ≈ cold round-1 (~11%, 23 vs 24 turns) — recall fired and the executor adopted the ONE captured note (a DI `pathFn` trick), but the 18 requirements were never in the vault.
- **Finding:** recall works; the binding constraint is **learn-capture** (no reliable per-round fire; bulk end-of-session capture exceeds context). Capture must be incremental + small-context.

### Capture-only eval — 5 learn variants (judged on written notes, not retrieval)
- baseline ~8/19, requirement-capture ~17/19, chunked ~17/19. baseline collapses features to keywords ("priority", "undo") with no load-bearing detail; it captures architecture/gotchas well but treats stated *requirements* as keyword asides.
- Two gaps: **set-completeness** (#3, the full command set as a unit — captured as individual items, never as "the complete set"), and #15 (never actually fed → source artifact, not a learn failure).

### Learn × Recall matrix (3 learn-vaults × 2 recall strategies; real warm builds)
specMatch /18 (all 6 compiled + passed tests):

| LEARN \ RECALL | passive | directed |
| --- | --- | --- |
| baseline | 8 (39t/$1.07) | 14 (34t/$1.54) |
| requirement-capture | 15 (31t/$0.97) | 17 (40t/$1.82) |
| consolidated | 18 (31t/$1.22) | 18 (33t/$1.47) |

- **Learn dominates** (passive column: 8→15→18, a 10-pt swing). Recall is second-order (0–6 pts, mostly baseline rescue).
- **Asymmetric interaction:** directed recall rescues a weak vault (+6) but does nothing for a good one (+0). Learn quality and recall effort are **substitutes**.
- Best $/correctness: requirement-capture × passive. Max correctness cheaply: consolidated × passive. Both winners are **passive** — directed recall adds cost faster than coverage on decent vaults.
- `surfacing ≠ application`: a recalled requirement (color) was still dropped by the executor.

### Conclusion
The highest-leverage engram change is to **LEARN-capture**, not recall and not cross-vector linking (#637): capture stated requirements explicitly **with their load-bearing detail**, **consolidated** into dense topic-notes (not atomized past the recall cutoff), **enumerate complete sets**, and always write a **dedicated mistake note** — performed **incrementally per round** (bulk end-of-session capture fails on context length). This **contradicts the current learn SKILL.md "Atomicity — one idea per permanent" bar**; the eval says, for retrievable + applicable capture, dense-and-complete beats atomic. Recall is fine as-is (passive); don't over-invest there.

---

## Isolated episode-only recall result (2026-05-31) — VALIDATED

After fixing recording (clean per-arc episodes; harness-injection + task-notification stripped) and retrieval (episodes embedded by their `situation`, not whole body), re-ran the test in ISOLATION: a vault containing *only* 3 requirement-bearing episodes of a cold todo build, then a warm build recalling them.

| | spec-match | rounds | turns | cost |
| --- | --- | --- | --- | --- |
| COLD (empty vault) | ~2/18 at r1 → ~17/18 after 4 review rounds | 4 | 61 | $4.34 |
| WARM (3 episodes recalled) | ~17/18 in round 1 (compiles, tests pass) | 1 | 29 | $0.95 |

**Memory delivered faster + cheaper + more correct (~4.5x cheaper).** Honest controls: vault was episodes-only (no fact/feedback confound); episodes were raw transcript evidence, not distilled answers; recall surfaced them at cosine 1.0 via situation-embedding; recall fired and the agent re-applied the requirements. The earlier "warm worse than cold" was a *capture* failure (empty seed), not inherent memory overhead.

Caveat: n=1, and the warm task is the SAME todo build — proves "don't re-pay to rediscover known requirements," not yet generalization to a different task. Transfer test (a different but conventions-sharing app) is next.

## Generalization + accumulation (2026-05-31)

Transfer to a different app, conventions rubric (17 items): the todo episodes carry the opinionated architecture + cross-cutting patterns; status/priority/due are todo-specific.

| build | score | turns | cost |
| --- | --- | --- | --- |
| bookmarks COLD | 4/17 (0/6 architecture) | 23 | $0.66 |
| bookmarks WARM (+todo) | 10/17 (6/6 architecture) | 17 | $0.58 |
| contacts COLD | 3/17 | 20 | $0.53 |
| contacts +todo | 14/17 | 17 | $0.57 |
| contacts +todo+bookmarks | 12/17 | 22 | $0.69 |

- **Architecture transfers cleanly to a different app** (bookmarks 0/6→6/6; contacts cold→+todo strongly). Feature transfer is partial (domain-specific features don't auto-graft).
- **Accumulation (2nd app's memories) did NOT help and cost more:** contacts +todo = 14/17, +todo+bookmarks = 12/17 at +21% cost. The 2nd (partial) example reinforced the shared architecture but DILUTED features unique to the first (lost exactly stats/color, which bookmarks lacked), and recall injected more (cost up). Memory quality/curation > quantity. Vault note Permanent/253.
