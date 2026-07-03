# Vocab trigger replay results

Run date: 2026-07-03
Script: `dev/eval/vocab/trigger_replay.py`
Vault: `~/.local/share/engram/vault` (141 memory notes)
Command: `python3 dev/eval/vocab/trigger_replay.py ~/.local/share/engram/vault`

## Verbatim output

```
======================================================================
VOCAB TRIGGER REPLAY — 2026-07-03
Vault: /Users/joe/.local/share/engram/vault
======================================================================

[1] Loading frontmatter created: dates (primary source)...
    Found 141 notes across 19 unique dates

[2] Running git log --diff-filter=A cross-check...
    Found 783 additions in git history

[3] Reconciling sources...

Reconciliation report:
  Primary source           : frontmatter created: field
  Notes on disk (current)  : 141 (measured)
  Notes ever added in git  : 783 (measured)
  Notes in BOTH            : 2 (measured)
  On disk, NOT in git      : 139 untracked notes — all notes 26–162 were
                             never committed to git; frontmatter dates are authoritative
  In git, NOT on disk      : 781 legacy/purged entries —
                             all Permanent/*.md, _legacy/*.md, and date-only files from
                             the pre-migration vault; not part of the current note series
  Event series             : 2026-06-12 → 2026-07-03
                             (22 calendar days)

======================================================================
UNREPLAYABLE: untagged/hub axes — tags backfilled at migration 2026-07-03.
  Trigger (a): untagged-rate >10% of last 25 writes — no historical tag data.
  Trigger (b): any term >25% of vault — no historical term-membership data.
  Both axes require per-note tag history that does not exist.
  Analytical modeling only (see proposals doc).
======================================================================

[4] Note-write event series (by created: date, all notes):

Daily note-write events (non-zero days only)
============================================
+-------------------+---------------------+--------------------+
| date (YYYY-MM-DD) | notes added (count) | cumulative (count) |
+-------------------+---------------------+--------------------+
| 2026-06-12        | 6                   | 6                  |
| 2026-06-14        | 4                   | 10                 |
| 2026-06-16        | 2                   | 12                 |
| 2026-06-17        | 22                  | 34                 |
| 2026-06-18        | 1                   | 35                 |
| 2026-06-19        | 4                   | 39                 |
| 2026-06-20        | 6                   | 45                 |
| 2026-06-22        | 2                   | 47                 |
| 2026-06-23        | 8                   | 55                 |
| 2026-06-24        | 13                  | 68                 |
| 2026-06-25        | 9                   | 77                 |
| 2026-06-26        | 6                   | 83                 |
| 2026-06-27        | 6                   | 89                 |
| 2026-06-28        | 26                  | 115                |
| 2026-06-29        | 7                   | 122                |
| 2026-06-30        | 6                   | 128                |
| 2026-07-01        | 6                   | 134                |
| 2026-07-02        | 4                   | 138                |
| 2026-07-03        | 3                   | 141                |
+-------------------+---------------------+--------------------+

[5] Joint trigger replay: absolute growth × min-interval
    Fires when: (notes written since last refit ≥ threshold) AND (days since last refit ≥ interval)


Joint absolute-growth trigger grid
==================================
+--------------------------+---------------------+------------+------------------------------------+-------------------+----------------+
| growth threshold (notes) | min interval (days) | fire count | fire dates                         | median gap (days) | min gap (days) |
+--------------------------+---------------------+------------+------------------------------------+-------------------+----------------+
| 30                       | 7                   | 3          | 2026-06-19, 2026-06-26, 2026-07-03 | 7.0               | 7              |
| 30                       | 14                  | 1          | 2026-06-26                         | —                 | —              |
| 30                       | 30                  | 0          | —                                  | —                 | —              |
| 40                       | 7                   | 2          | 2026-06-22, 2026-06-29             | 7.0               | 7              |
| 40                       | 14                  | 1          | 2026-06-26                         | —                 | —              |
| 40                       | 30                  | 0          | —                                  | —                 | —              |
| 50                       | 7                   | 2          | 2026-06-24, 2026-07-01             | 7.0               | 7              |
| 50                       | 14                  | 1          | 2026-06-26                         | —                 | —              |
| 50                       | 30                  | 0          | —                                  | —                 | —              |
+--------------------------+---------------------+------------+------------------------------------+-------------------+----------------+

[6] Conjunct co-occurrence analysis (per plan note 161)
    For each grid cell: was growth-only ever pending? interval-only ever pending? did they JOINTLY fire?


Conjunct co-occurrence table
============================
+----------------+-----------------+-------------+-------------------------------------------+-------------------------------------------+-------------------------+
| growth (notes) | interval (days) | joint fires | growth-ready but interval-blocked (count) | interval-ready but growth-blocked (count) | conjuncts ever co-fire? |
+----------------+-----------------+-------------+-------------------------------------------+-------------------------------------------+-------------------------+
| 30             | 7               | 3           | 6                                         | 0                                         | YES                     |
| 30             | 14              | 1           | 13                                        | 0                                         | YES                     |
| 30             | 30              | 0           | 15                                        | 0                                         | NO — never co-fire      |
| 40             | 7               | 2           | 2                                         | 3                                         | YES                     |
| 40             | 14              | 1           | 8                                         | 0                                         | YES                     |
| 40             | 30              | 0           | 12                                        | 0                                         | NO — never co-fire      |
| 50             | 7               | 2           | 2                                         | 5                                         | YES                     |
| 50             | 14              | 1           | 5                                         | 0                                         | YES                     |
| 50             | 30              | 0           | 10                                        | 0                                         | NO — never co-fire      |
+--------------------------+---------------------+------------+------------------------------------+-------------------+----------------+

[7] Shipped relative-growth baseline trigger: vault grew >30% since last refit


Relative-growth (shipped c) baseline trigger
============================================
+---------------------------------------+------------+------------------------------------------------------------------------+-------------------+----------------+
| variant                               | fire count | fire dates                                                             | median gap (days) | min gap (days) |
+---------------------------------------+------------+------------------------------------------------------------------------+-------------------+----------------+
| 30%-growth, no min-interval (shipped) | 6          | 2026-06-14, 2026-06-17, 2026-06-20, 2026-06-24, 2026-06-27, 2026-06-29 | 3.0               | 2              |
| 30%-growth + 7d min-interval          | 3          | 2026-06-19, 2026-06-26, 2026-07-03                                     | 7.0               | 7              |
| 30%-growth + 14d min-interval         | 1          | 2026-06-26                                                             | —                 | —              |
+---------------------------------------+------------+------------------------------------------------------------------------+-------------------+----------------+

[8] SUMMARY NOTES

1. Event-series authority: frontmatter created: dates are the ONLY authoritative source.
   All 139 notes 26–162 are untracked in git (never committed). Notes 24–25 are in git
   (added 2026-06-12) but their frontmatter dates agree (2026-06-12). Git cross-check
   confirms zero discrepancy for the 2 tracked notes.

2. Git-only legacy entries: 781 files (Permanent/*.md, _legacy/*.md, date-only .md)
   appear in git history but NOT on disk — all are pre-migration vault artifacts, not
   current memory notes. They are correctly excluded from the event series.

3. Relative-growth trigger runs hot on small bases: early in the series,
   even a handful of notes constitutes >30% growth. The 7d floor suppresses
   this significantly.

4. Per note 161: any grid cell with joint_fires = 0 means the two conjuncts
   (growth AND interval) NEVER co-occurred simultaneously in this history —
   that combination is vacuous and should be excluded from the proposal's
   candidate set.

5. UNREPLAYABLE axes: untagged-rate and hub-share cannot be replayed.
   Model analytically: in a new-domain influx scenario, assume 6 notes/day
   for 7 days = 42 notes, all untagged = 100% untagged-rate; the (a) trigger
   would fire on day 1 (>10% of last 25 = 3 untagged). This is a pure
   stress-test model, not a historical replay.
```

---

## Reconciliation notes (not in script output)

### Weekly note-write counts (replayed from frontmatter, measured)

| week | date range | notes written (count) |
|---|---|---|
| Wk 1 | 2026-06-12 – 2026-06-18 | 35 |
| Wk 2 | 2026-06-19 – 2026-06-25 | 42 |
| Wk 3 | 2026-06-26 – 2026-07-02 | 61 |
| Wk 4 (partial, 4 days) | 2026-07-03 | 3 |

Plan's pre-replay projection ("growth 10/37/75/19 per week") differs from replayed actuals (35/42/61/3). The plan's numbers appear to use different week boundaries or counted only certain note types. Use the replayed actuals.

### 30d-interval cells vacuous for a different reason

All three 30d-interval cells fire 0 times NOT because the growth threshold is never reached, but because the HISTORY IS ONLY 22 DAYS LONG. The growth-ready-but-interval-blocked counts (10–15) confirm that growth thresholds WERE crossed repeatedly — the 30d wait can never complete in a 22-day history. These cells cannot be evaluated from current history; they need 30+ days of data to characterize.

### Git-untracked notes (structural finding)

The vault's git repo is used as a backup/checkpoint mechanism, not for continuous note tracking. Notes 26–162 (and their .vec.json sidecars) are explicitly untracked. The git log cross-check is therefore structurally limited for this vault: it can only confirm the 2 tracked notes (24, 25) agree with their frontmatter dates, which they do. All other notes rely entirely on frontmatter.

---

## Cost model (Task 2)

All numbers labeled per the plan: `measured` / `estimated` / `projected` / `replayed`.

### Per-fire overhead

| option | description | tokens added/learn fire | tokens added/recall fire | $/learn fire (cache) | $/recall fire (cache) |
|---|---|---|---|---|---|
| O0 | Recalibrate only | 0 | 0 | $0 | $0 |
| O1 | Learn-anchored stateless | ~237 (vocab stats output) | 0 | ~$0.00036 (estimated; cache_read at $1.5/MTok) | $0 |
| O2 | Binary-persistent flag | ~0 (in-process check) | ~5 (if flag set) | ~$0 | ~$0.000008 |
| O3 | Scheduled out-of-band | 0 | 0 | $0 | $0 |
| O4 | Recall-side checks | 0 | ~237 (vocab stats output) | $0 | ~$0.00036 (estimated; cache_read) |

Source for 237 tokens: `measured` — `engram vocab stats` output clocked at ~237 tokens (plan ground fact).
Source for token pricing: `estimated` — claude-sonnet-4-6 cache_read $1.5/MTok (no direct session meter; using Anthropic pricing).
Source for in-process flag: `measured` — the `refit_pending` field is an omitempty struct field read from vocab.centroids.json; no LLM call (plan: "mechanically trivial").

### Action cost when tripped (refit)

| action | cost | label | source |
|---|---|---|---|
| `engram vocab refit` LLM judgment | ~$1–2 | estimated | No standalone metered run exists. Derived from: 25 terms × ~1 LLM call × ~1.5K tokens/call = 37.5K tokens at $15/MTok (opus) ≈ $0.56, plus orchestrator overhead; plan itself says "~$2 when tripped" |
| C3–C6 trap regression smoke (if automated) | $3.09 | measured | docs/design/2026-07-03-vocab-notes-build-results.md:line 47 |
| Total action cost (refit + gates) | ~$4–5 | estimated | Above two rows summed |
| `engram vocab refit` standalone (no gates) | ~$1–2 | estimated | As above |

### Fire frequencies

| metric | value | label | source |
|---|---|---|---|
| Learn writes/week (observed) | ~18 | estimated | 55 learn-origin notes / 3.1 weeks (plan note-origin split; `source:` field heuristic) |
| Recall-2.5 writes/week (observed) | ~24 | estimated | 73 recall-origin notes / 3.1 weeks (amend + learn via recall-2.5C + Step 4) |
| Regular /please calls/week | ~3 | estimated | Session transcript dates (non-build days: Jun 8/10/12/15/19/20/24/25 = 8 days in ~2.5 weeks ≈ 3/week) |
| Learn runs/week (regular cadence) | ~6 | estimated | 3 /please × 2 learn runs/please (plan assumption) |
| Recall runs/week (regular cadence) | ~6 | estimated | 3 /please × 2 recall fires/please (plan assumption) |
| Heavy /please day recall writes before closing learn | 10+ | estimated | plan stated "heavy /please day ≈ 10+ recall writes before the closing learn" |

### Staleness window (events, not time)

| option | staleness window | description |
|---|---|---|
| O0 | unbounded / manual | no automatic check; only catches what a human/agent manually checks |
| O1 | ~10+ writes (heavy /please day) | all vocab-relevant events between recall writes and the closing learn run |
| O2 | 0 events (detection side) | flag is set immediately at each write via in-process hook |
| O3 | ~30–42 writes (7d window) | up to 7 days of writes at ~6 notes/day before the weekly job runs |
| O4 | near-0 (1–2 events) | recall skill fires before its own write path; at most 1 write per recall skill fire before the next check |

### $/month across growth scenarios

Fire rate basis: `replayed` (absolute 30/7 grid cell = 3 fires in 22 days ≈ 4 fires/month).
Refit action cost: ~$4–5 per fire (refit + gates, `estimated`).
Check overhead: `estimated` from token pricing above.

| option | growth scenario | check overhead/month | refit fires/month | action cost/month | total $/month range | sensitivity driver |
|---|---|---|---|---|---|---|
| O0 | any | $0 | 0 (manual) | $0 | $0 | — |
| O1 | Wk1 (35 notes/wk) | ~$0.009 | ~4 | ~$16–20 | ~$16–20 | refit fire rate |
| O1 | Wk2 (42 notes/wk) | ~$0.009 | ~4 | ~$16–20 | ~$16–20 | refit fire rate |
| O1 | Wk3 (61 notes/wk) | ~$0.009 | ~4 | ~$16–20 | ~$16–20 | refit fire rate |
| O1 | steady-40/wk | ~$0.009 | ~4 | ~$16–20 | ~$16–20 | refit fire rate |
| O1 | idle week | ~$0.009 | 0 | $0 | ~$0.01 | check overhead only |
| O2 | any active | ~$0 | same as O1 | ~$16–20 | ~$16–20 | refit fire rate |
| O2 | idle | ~$0 | 0 | $0 | $0 | — |
| O3 | 30+ notes in 7d | $0 | 4/month | ~$16–20 + gates | ~$16–20 | refit + gate cost |
| O3 | idle | $0 | 0 | $0 | $0 | — |
| O4 | any active | ~$0.009 (recall-side) | same as O1 | ~$16–20 | ~$16–20 | refit fire rate |
| O4 | idle | ~$0.009 | 0 | $0 | ~$0.01 | check overhead only |

Note: the refit ACTION cost dominates by 3–4 orders of magnitude over the check overhead. The sensitivity driver in all non-idle scenarios is the refit fire rate, not which option you pick (O1/O2/O4 are cost-equivalent when they use the same trigger). The check overhead difference between options is ~$0.01/month — negligible.

### Implementation cost

| option | binary LOC (net-new) | files changed | skill edits | writing-skills TDD cost | infra | notes |
|---|---|---|---|---|---|---|
| O0 | 0 | none | none | $0 | none | recalibrate trigger docs only |
| O1 | ~50–100 | vocab_commands.go (verdict line in `printStatsReport`) | learn SKILL.md (add stats conditional) | ~$1–2 | none | verdict line is the only binary work |
| O2 | ~100–150 | vocab_commands.go (verdict line) + learn.go + amend.go (hook sites × 2) + vocab.centroids.json schema | learn SKILL.md | ~$1–2 | none | ring-buffer variant (windowed untagged) adds ~60–100 LOC more — flagged as HEAVIEST sub-piece |
| O3 | ~50–100 | vocab_commands.go (verdict line only) | none | $0 | launchd/cron (external) | NO in-repo scheduling infra exists; requires OS-level setup; autonomy decision is a gate criterion per plan |
| O4 | ~50–100 (same as O1; verdict line reused) | vocab_commands.go | recall SKILL.md (add stats conditional; deep+glance paths both need updating) | ~$1–2 | none | recall SKILL.md is longer/more complex than learn SKILL.md; both deep and glance paths must be handled |

Source: vocab_commands.go `printStatsReport` at line 1018–1051 (`measured`; 33 LOC function body, verdict line = ~20–50 LOC addition depending on threshold struct). learn.go:199 (`applyVocabAssignmentAfterLearn`), amend.go:285 (`applyVocabAssignmentAfterAmend`) — both confirmed as the hook points (`measured`).

---

## Coverage matrix (Task 3)

| write path | flows through assignment hook? | call site | vocab staleness consequence |
|---|---|---|---|
| `engram learn` (direct/from recall Step 2.5C absent) | YES | `applyVocabAssignmentAfterLearn` at learn.go:199 | note gets vocab tags assigned at write time; staleness = 0 |
| `engram amend` (recall Step 2.5C near/covered, or standalone) | YES | `applyVocabAssignmentAfterAmend` at amend.go:285 | note gets vocab tags re-assigned at amend time; staleness = 0 |
| recall Step 4 synthesis (`engram learn`) | YES | same as `engram learn` above (learn.go:199) | same — new synthesis note tagged at creation |
| `engram resituate` | NO — BYPASSED | zero calls to `applyVocab*`/`AssignVocabTerms` in resituate.go (code-verified, Gate A) | re-embedded sidecar may carry stale vocab tags relative to new embedding; tag drift silently accumulates |
| `engram` supersedes handling | NO | supersedes.go has no vocab calls; it only parses/writes the `supersedes:` frontmatter field | no staleness from supersedes writes; tags on the superseding note are set by the learn/amend call that created it |
| `engram vocab bootstrap` (migration) | YES (bulk backfill via `assignTermsToAllNotes`) | vocab_commands.go:414 `assignTermsToAllNotes` | all notes re-tagged at bootstrap time — this is the migration path, not a regular write path |

Source: all "NO" cells confirmed by `grep -n "applyVocab\|AssignVocab\|vocab\|Vocab" internal/cli/resituate.go` returning 0 matches and `grep -n "func\|vocab" internal/cli/supersedes.go` showing no vocab assignment calls. All "YES" cells confirmed by reading learn.go:196–225 and amend.go:282–311.

---

## Stress-case paragraphs (Task 2, plan Step 2)

### Case 1: Heavy /please day (10+ recall writes, 2 learn runs)

During a heavy /please day, the agent fires recall 2–4 times (each potentially writing 3–5 notes via Step 2.5C amend/learn and Step 4 learn), with a closing learn run at the end. Total new notes in a day: 10–26 (observed Jun 17: 22 notes, Jun 28: 26 notes). Under O1, the vocab check fires only at the 2 learn runs — so 10+ recall-origin writes accumulate between the opening and closing learn, all with no liveness check. If a new domain was introduced (say, 8 recall-origin notes all untagged), O1 detects this at the closing learn only. The staleness window = ~10 writes = roughly 4–6 hours of lag. Under O2, detection fires at every write: the 8th untagged write would set the `refit_pending` flag, and the closing learn's sweep surfaces it immediately. Under O4, the recall skill's own stats check fires 2–4 times during the day; each fire would catch untagged growth before the recall's write path adds more stale notes. Cost delta vs O0: all options cost ~$1–2 for one refit when tripped; the difference is detection speed and how many stale notes accumulate before detection.

### Case 2: Idle week (0 notes)

No notes are written; no triggers can fire. All options cost $0 in refit/action costs. O1 and O4 still pay the per-run check overhead (~$0.009/week — negligible). O2's binary flag stays clear. O3's scheduled job runs and reports healthy ($0 when not tripped). The liveness difference between options is zero: no writes means no vocab drift. This case does NOT favor one option over another — it is cost-neutral across all options.

### Case 3: New-domain influx (untagged climbing)

Suppose 6 new-domain notes/day arrive for 7 days (42 notes in 7 days — realistic for a big eval or new-feature build). All are in a domain with no existing vocab coverage, so they are potentially untagged or poorly tagged. Under the absolute-growth trigger at threshold 30 + 7d interval: after 30 notes in 5 days, growth condition fires; the 7d interval is NOT met yet (5 < 7), so the trigger is blocked — detection waits for day 7. Under O1, the closing learn on day 7 finally fires the verdict. Under O2, the `refit_pending` flag is set on the 30th write (day 5), and the next learn's sweep surfaces it. Under O4, any recall run after day 5 surfaces the pending flag. The untagged-rate trigger (a) — the "right" trigger for this scenario — would catch it much faster: after 3 untagged writes (10% of last 25), it fires on day 1. But trigger (a) requires either the windowed ring-buffer (O2's heaviest sub-piece) or a vault-wide proxy. UNREPLAYABLE from history; this is analytical modeling only.
