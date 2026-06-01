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

## Exact memory-vs-no-memory cost to the same bar (2026-05-31)

Apples-to-apples on the contacts build, to ~15-16/17 conventions:

| path | rounds | turns | cost | autonomous |
| --- | --- | --- | --- | --- |
| memory (generic-actionable learn + as-reqs recall) | 1 | 27 | $0.82 | yes (one shot) |
| no memory (review loop) | 4 | 56 | $3.53 | no (human feeds requirements each round) |

Memory is ~4.3x cheaper and ~2x fewer turns, and reaches the bar autonomously vs four human-review rounds. No-memory rounds escalate ($0.53/0.53/0.82/1.65) because each `--resume` re-sends the growing transcript; memory avoids that with a single primed build. (n=1; the 5x5 matrix found generic-actionable+as-reqs = 15/17 @ $0.82.)

## Layer-isolation experiment — L1 vs L2 vs L3, convergence cost (2026-05-31)

Six arms differing only in vault contents (todo-derived memory; contacts test
build), measuring rounds / human-review-interactions / turns / cost to reach the
architecture bar (arch >= 9/10). Vaults: cold(empty), l1(3 episodes),
l2(7 facts), l3(2 distilled), l1l2, l1l2l3.

**Corrected result (after a scorer fix — see below):** all six arms reach the
same ~9/10 architecture. Memory's value is *autonomous convergence*, not
sufficiency:

| arm | round-1 arch | converged | human interactions | cost-to-converge |
| --- | --- | --- | --- | --- |
| cold | ~4 | round 2 | 1 | ~$1.91 |
| l1 (episodes) | 8 | round 2 | 1 | $0.93 |
| **l2 (facts)** | **9** | **round 1** | **0** | **$0.55** |
| l3 (distilled) | 8 | round 2 | 1 | $1.39 |
| l1l2 | 9 | round 1 | 0 | $0.67 |
| l1l2l3 | 9 | round 1 | 0 | $0.86 |

- **L2 specific-facts is the best single layer** (autonomous, cheapest). L1
  episodes and L3 distilled each need one polish round (they drop the
  `--json`/color details L2 carries — *distillation loses specifics*); L3 is the
  priciest converged-in-2 arm ($1.39). Stacking (l1l2/l1l2l3) is autonomous but
  costs more than L2 alone — adding layers adds cost, not convergence.
- **No-memory (cold) reaches the same bar** but at ~3.5x L2's cost + 1 human round.

**METHODOLOGY LESSON (load-bearing): the first scorer was name-biased.** Its DI
check required the literal word `Store` in the interface name — which is the
vocabulary the L2/L3 *notes* prescribe. Memory arms copy `Store` and passed;
cold chose `Repository` (a synonym) and was scored "no DI", producing a false
"cold never converges (5 rounds, capped)" headline. A heuristic scorer that keys
on vocabulary drawn from the thing-under-test will systematically favor the
memory arms. Name-agnostic re-scoring (detect the *pattern*: any persistence
interface + injection) put cold at 9/10 — it had built clean DI all along.

**Caveats:** n=1 throughout; the 0-vs-1-round splits and layer cost-ordering hinge
on 1–2 rubric items (json/color); cold's convergence round inferred from rounds
3–5 being "already done" no-ops; circularity (rubric = the notes' own content, so
this is memory-vs-review-channel, not emergent transfer). Treat the layer ranking
as suggestive pending a clean re-run (name-agnostic scorer + n>=2 on anchors).

**Bookmarks/accumulation status:** bookmarks memory exists only as a raw L1
episode (`bm-vault`/`combo-vault`); no bookmarks-derived L2/L3 was ever distilled.
The prior "contacts+todo+bookmarks = 12/17 (diluted vs +todo 14)" was L1-episode
accumulation. An *accumulated L3* (bookmarks lessons distilled into the existing
L3 notes) has never been built or tested — it is the next experiment.

## Accumulation by layer — +bookmarks (2026-05-31)

Distilled the bookmarks build into the existing todo-derived L2/L3 via a learn
run. Result: **100% elaboration, 0 new notes** — the architecture recurs; the only
deltas are *added actionable detail* (chiefly the L3 DI principle generalizing to
"inject ANY side-effect — browser-open/exec/net — not just Store/Clock/Writer").
Built accumulated vaults (l1bm/l2bm/l3bm/l1l2l3bm), tested vs todo-only on contacts,
round-1 autonomous, n=2, name-agnostic scorer:

| layer | todo-only arch | +bookmarks arch | read |
| --- | --- | --- | --- |
| L1 | 10.0 | 9.0 | no benefit (noise) |
| L2 | 10.0 | 10.0 | no benefit |
| L3 | 9.0 | 8.0 | no benefit — the −1.0 is ONE outlier build (missed test-isolation + NO_COLOR, both unrelated to bookmarks); identical 5-file DI structure |
| all | 9.0 | 8.5 | no benefit |

**Accumulating a second source gave NO benefit at any layer; nothing improved.**
The accumulated L3 did not score highest — equal-to-marginally-lower, within n=2
noise. The earlier "richest L3 won (15/17)" was **todo-only generic-actionable**
distillation — the win was distillation *quality* (dense on-domain detail), not
multi-source accumulation. Accumulation didn't help because the bookmarks deltas
are **off-domain for contacts** (side-effect/browser-open injection contacts never
needs), so the enriched note is longer, not more useful.

**Caveats:** n=2, differences <=1 pt (mostly noise — robust signal is only
"memory helps a lot; +2nd-source adds nothing"). By design this cannot show
accumulation HELPING — the 2nd source is off-domain for the target; a fair test
needs a target whose needs match the accumulated lessons (e.g. an app that needs
side-effect injection, where the bookmarks-enriched L3 would be on-domain).
