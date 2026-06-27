# Recall-trigger patterns + 10 proposals

> **EXTERNAL-VALIDITY CAVEAT — READ FIRST (the result does not auto-generalize).**
> This taxonomy is mined from **one corpus that is 100% engram eval / memory-system work.**
> **37 of 59 (63%) TRIGGER moments are eval-design vocabulary** (eval, harness, fixture, probe, A/B,
> cold/warm, distractor, RED baseline), and the Tier-C spot-check of non-engram logs found
> **0 real corrections** (`non-engram-spot-check`: the sampled logs were synthetic probe harnesses, not
> real work). So every "signal" below is over-fit to **eval-design + engram code loci** (recency-band
> wiring, nilaway/modernize lint, `engram query` ranking). It will **not** auto-transfer to ordinary
> coding. As a tell: the genuinely cross-project triggers in Joe's curated memory index —
> **verify-don't-guess on external API/path/format contracts, worktree-return, batch-parallel-writes** —
> are **absent as failure moments here**, and the one that does recur (real-binary-verify, note 54) shows
> up only in engram-CLI-specific clothing. Treat this as a map of *this project's* recall gaps, not a
> general trigger library.

---

## 0. Headline verdict

| Quantity | Value | Meaning |
|---|---|---|
| Real corrections in corpus | **82** | = 59 TRIGGER + 23 CAPTURE |
| TRIGGER (memory existed, not recalled/applied) | **59 (72%)** | the *most* the trigger lever can ever reach |
| **CAPTURE (NO memory existed — unreachable by ANY trigger)** | **23 (28%)** | **hard ceiling: triggering ships nothing here; needs write-side capture (P6)** |
| Naive trigger "recall before **every tool call**" | **8674 / 59 ≈ 147×** over-fire | **FAIL** (×190s/fire) |
| Naive trigger "recall on **every user turn**" | **635 / 59 ≈ 11×** over-fire | **FAIL** (×190s/fire) |
| Dominant addressable sub-cluster: self-recommendation | **15 / 59 (25%)** | **NOT a retrieval miss** — note is in-context, outranked at synthesis (note 82). Firing recall more does **not** fix it; needs re-rank (#655) + gate (#656). |

**Headline (one sentence):** Triggering can reach **at most 72%** of corrections — **28% had no memory at
all** and are a write-side problem, not a trigger problem — and within that 72% the two naive "fire more"
triggers both blow the over-fire bound while the largest addressable cluster (self-recommendation, 25%)
is an **application/ranking** failure that firing recall *more* cannot fix; the only clean recall-firing
win is a **narrow eval-design task-type trigger (~7×, DERIVED)**, and the rest of the real value lives on
**free surfaces** (hooks, static docs, a binary re-rank) that the over-fire bound does not constrain.

### 0.1 The cost model (stated once, applied uniformly below)

> **memory-tax(trigger) = over-fire-ratio × per-fire-cost.** Two regimes, and which one a proposal is in
> decides whether the ratio matters:
>
> - **Recall-firing triggers** — per-fire-cost ≈ **190s** (the full recall body; per
>   `recall-cost-isolation.md`, 2026-06-25). At 190s/fire, Joe's **~10× over-fire bound binds hard**:
>   11×-on-user-turn and 147×-on-tool-call are *fatal*; only ≤ ~10× is viable.
> - **Free-fire triggers** — a PreToolUse/PostToolUse **hook**, an always-loaded **static doc**, or a
>   **binary re-rank** costs ≈ **0s** per fire. The ratio does **not** bind: a **43× hook (P4) is fine**,
>   a 380× static doc is fine. The only cost is attention/noise, not wall-time.
>
> This single model explains every rating: it is why **11×-recall is fatal but 43×-hook is fine.**

### 0.2 The fire-unit (stated once)

> A recall trigger **fires at TASK INITIATION** — a user turn / new task context where skill selection
> happens — **not at every assistant turn or every tool call.** All over-fire ratios below are computed at
> this unit, **except** hook proposals (fire at a tool event) and static-doc proposals (always-present;
> "fire" once per context). Every per-proposal number is tagged **DERIVED** (computed from a measured
> denominator) or **ESTIMATE** (behavioral signal, no clean denominator → wide range, and the rating's
> confidence is **lowered**).

---

## 1. The moment corpus + signal taxonomy

> **Provenance / data trail.** The classified moments and the deterministic extractor are committed at
> [`2026-06-27-recall-trigger-data/`](2026-06-27-recall-trigger-data/): `extract_moments.py` reproduces
> the denominators (8674 tool calls, 586/635 user turns, the 551→78 candidate filter, the tool/git/skill
> counts) from the engram session logs; `moments_all.json` (83) + `moments_trigger.json` (59) are the
> classified moments with `summary`/`source`/`signal_category`/`klass`/`preceding_cue`/`lesson` — the
> long list **in substance** and the audit trail for every count below. Denominators are deterministic;
> classifications are LLM judgment (Tier-A exhaustive over 52 feedback notes, Tier-B over 78 candidates),
> adversarially critiqued (see the doc's git history).

**Lead table — signal split of the 59 TRIGGER moments** (DERIVED from `moments_trigger.json`):

| signal_category | count | share | eval-vocab? | addressable by | fire-unit |
|---|---|---|---|---|---|
| task-type | **27** | 46% | mostly (eval-design) | P1 (eval slice ~12), static docs, re-rank | task initiation |
| self-recommendation | **15** | 25% | partly | **P5 re-rank + P2 gate** (~13), P3 probe (~2) | recommend-boundary |
| tool-imminent | **8** | 14% | some | P7 go.md (5), P4 hook (1), P10 hook (1), behavioral (1) | tool event |
| phrase-intent | **4** | 7% | some | P2 gate (reviewer-claim), P6 capture (note 67) | task initiation |
| step-boundary | **4** | 7% | some | P9 done-claim (PARK) | assistant turn |
| ask-keyed | **1** | 2% | no | MEMORY.md (note 35, already indexed as feedback_deliver_full_diverse_set) | task initiation |
| **total** | **59** | 100% | **37 (63%)** | — | — |

**Moment list by cluster (REAL source IDs from `moments_trigger.json` — vault notes by number, raw logs
by `jsonl#line`).**

- **task-type (27):** notes 101, 104, 24, 33, 36 (×2: `…scoredchunk-conversion` + `…note-recency-decay`),
  39, 42, 44, 45, 60, 62, 63, 68, 69, 70, 72, 75, 83, 86, 89; raw logs
  `58649fd7…jsonl#144`, `a19e7b75…jsonl#966`, `a19e7b75…jsonl#5589`, `c1195a11…jsonl#1272`,
  `d498a354…jsonl#1468`, `d498a354…jsonl#5853`.
- **self-recommendation (15):** notes 105, 107, 108, 73, 74, 76, 78, 80, 81, 84, 87, 92, 96, 97; raw log
  `d498a354…jsonl#6534`.
- **tool-imminent (8):** notes 106 (binary install), 25 (git restore), 37, 43, 57, 58, 59 (go-lint), 94
  (verify-metric).
- **phrase-intent (4):** notes 29, 61, 67, 71.
- **step-boundary (4):** notes 54, 64, 66, 90.
- **ask-keyed (1):** note 35.

**CAPTURE (23, the 28% ceiling — `memory_existed: "n"`, sampled IDs):** `58649fd7…jsonl#552`,
`842c4589…jsonl#136`, `95570838…jsonl#76`, `95570838…jsonl#2627`, `95570838…jsonl#3117`,
`a19e7b75…jsonl#6184`, `a19e7b75…jsonl#6504`, `a19e7b75…jsonl#9710`, `a19e7b75…jsonl#12168`,
`a19e7b75…jsonl#12573`, `c1195a11…jsonl#2979`, `d498a354…jsonl#2825/#2836/#3492/#8185/#10353`,
`ee8329d2…jsonl#269/#768/#4989/#5017/#5029/#7990/#11580`. **No trigger reaches these** — see P6.

---

## 2. Pattern taxonomy — coverage, over-fire, bound flag

**Lead table** (over-fire at the fire-unit from §0.2; per-fire cost from §0.1):

| signal / candidate trigger | over-fire | unit | D/E | per-fire cost | bound? |
|---|---|---|---|---|---|
| every tool call | **147×** (8674/59) | tool call | DERIVED | 190s | **FAIL** |
| every user turn | **11×** (635/59) | user turn | DERIVED | 190s | **FAIL** |
| eval-design vocab → recall (P1) | **~7×** (85/12) | task init | DERIVED | 190s (→ probe via P3) | **PASS-ish** (≤10×) |
| done-claim → verify-recall (P9) | **~380×** (1141/3) | asst turn | DERIVED | 190s | **FAIL → PARK** |
| destructive-git → hook (P4) | **43×** (43/1) | tool event | DERIVED | ~0s | **PASS (free)** |
| self-recommendation → re-rank+gate (P2/P5) | n/a (in-context) | recommend | ESTIMATE | re-rank ~0s; gate « 190s | application fix, not a "fire" |
| go-lint convention → go.md (P7) | always-loaded | context | DERIVED denom | ~0s | **PASS (free)** |

**Denominators used** (DERIVED, this corpus): 8674 tool calls; 635 user turns (586 genuine); 85 eval-vocab
turns; ~1141 done-claim assistant turns; .md edits 944; SKILL.md 105; git commit 369; git destructive 43;
git push 43; Agent dispatch 567; Skill 145.

**Key taxonomy finding (note 82, the honest re-attribution):** the 25% self-recommendation cluster is
**APPLICATION**, not retrieval — the relevant note is *already in context* but **outranked by chunks at
synthesis**. So its coverage must **not** be banked to any "fire recall more" trigger. It splits:
**~13/15** are fixed by **re-rank (#655) + reconcile gate (#656)**; only **~2/15** are genuine retrieval
misses a cheap recall-probe could catch.

---

## 3. The 10 proposals

**Lead table — the set** (rating per §0.1 cost model):

| id | name | solution-surface | coverage (of 59) | over-fire (unit, D/E) | per-fire cost | rating |
|---|---|---|---|---|---|---|
| P1 | eval-design recall trigger | skill-body | ~12 (20%) | ~7× (task init, **D**) | 190s→probe | **CONTENDER** (low conf) |
| P2 | self-rec reconcile gate | please-gate | ~13 (22%, shared w/ P5) | ~2–15× (recommend, **E**) | « 190s | **CONTENDER** (low conf) |
| P3 | two-speed recall quick-probe | split-skill / two-speed | enabler for P1+P2 | inherits caller's | decide cheap / exec deferred | **CONTENDER** |
| P4 | destructive-git PreToolUse hook | binary/hook | 1 (note 25) | 43× (tool event, **D**) | ~0s | **CONTENDER (KEEP)** |
| P5 | note-negation re-rank (#655) | re-rank (binary) | ~13 (22%, shared w/ P2) | n/a (modifies existing recall) | ~0s | **CONTENDER** |
| P6 | capture-on-correction | write-side / learn | reaches the **28% CAPTURE**, not TRIGGER | ~14% base-rate (**E**) | learn-side | **CONTENDER** (honest: not triggering) |
| P7 | go-lint conventions → `.claude/rules/go.md` | static-doc | 5 (notes 37/43/57/58/59) | always-loaded (**D** denom) | ~0s | **PARK** (real win, wrong surface) |
| P8 | cadence + results-table rule → CLAUDE.md | CLAUDE.md-pointer | 3 (notes 45/`c1195a11#1272`/`a19e7b75#5589`) | always-loaded | ~0s | **PARK** (real win, wrong surface) |
| P9 | done-claim verification trigger | skill-body / Stop-hook | 4 (step-boundary) | **380×** (asst turn, **D**) | 190s | **PARK** |
| P10 | binary-staleness PostToolUse hook | binary/hook | 1 (note 106) | rare (tool event, **E**) | ~0s | **CONTENDER (free)** |

Surfaces spanned: skill-body (P1), please-gate (P2), split-skill/two-speed (P3), binary/hook (P4, P10),
re-rank (P5), write-side (P6), static-doc/CLAUDE.md-pointer (P7, P8), done-claim trigger (P9). **All eight
required surfaces present**; the duplicated surfaces (hook, static-doc) are the two cheapest regimes.

---

### P1 — Eval-design recall trigger

| field | value |
|---|---|
| problem-pattern | task-type is the largest cluster (27/59); its dominant, recurring sub-cluster is **eval design** (37/59 eval-vocab). Agent repeatedly designs an eval (distractor ages, terminal literalness, retrieval-vs-synthesis metric) without recalling the note that already warns against the exact trap. |
| trigger-signal | eval-design vocabulary at **task initiation** (eval, harness, fixture, probe, A/B, cold/warm, distractor, RED baseline, planted, half-life). |
| solution-surface | **frontmatter-description (primary)** — recall's `description:` governs which tasks auto-invoke it (note 100: firing is *description*-gated, not body-gated), so add eval-design phrasings to the trigger description ("designing/costing/running an eval", "writing a harness/fixture/probe", "A/B or RED-baseline", "cold/warm comparison"). **skill-body (secondary)** — add an eval-design task-type to the recall trigger list. The description edit alone may suffice for selection; a body routing line is needed only if the description match proves too coarse (test both, §test-sketch). |
| coverage | **~12/59 (20%)** — the genuine eval-design recall-fixable slice (subset of the 37 eval-vocab moments; e.g. notes 42, 44, 75, 83, 86, 104). |
| over-fire | **85 eval-vocab turns / ~12 genuine ≈ 7×.** Fire-unit = task initiation. **DERIVED.** Under the ~10× bound — but *only* with P3 (else per-fire = 190s and 7× is borderline-fatal). |
| trigger-cost | **decide** = cheap vocab/classifier match (sub-1k-token); **execute** = full 190s recall body (until P3 makes it a quick-probe). |
| test-sketch | Cold pilot: present an eval-design task-init turn whose ask body omits the obvious vocab; assert the trigger still fires and a non-eval task-init turn does not (precision both ways). |
| prior-art | notes 42/44/75/83/86 (eval-design traps), note 104 (free retrieval-probe gate); test infra #654 (C7 anti-amnesia harness pattern reusable for cold-fire pilots). |
| **rating** | **CONTENDER, confidence lowered.** 7× passes the bound, decide-cost is cheap — but (a) it is **viable only paired with P3** (190s/fire otherwise), and (b) external validity: "eval-vocab" over-fits this corpus and will not transfer. |

---

### P2 — Self-recommendation reconcile gate (the genuine fix, split per note 82)

| field | value |
|---|---|
| problem-pattern | self-recommendation (15/59, 25%): the agent invents/re-proposes a lever or challenge late in a task; the disproving note **is already in context** but is **outranked by chunks at synthesis** (note 82) — so this is **APPLICATION, not retrieval.** Firing recall again does **not** fix it. |
| trigger-signal | the agent is about to **emit a recommendation / lever / challenge** (recommend-boundary), especially late in a `/please`. |
| solution-surface | **please-gate** — a reconcile step that re-states each proposed lever as an artifact keyed to **prior-outcome terms** ("already tried / rolled back / −14%") before it can ship. |
| coverage | **the application share: ~13/15** of self-recommendation (shared accounting with P5). The **other ~2/15** (note genuinely absent from context) are *not* P2's — they go to a tiny recall-probe via **P3** at the recommend-boundary. **No coverage is double-banked.** |
| over-fire | Fire-unit = recommendation-emission within `/please` (not every turn). **ESTIMATE** — behavioral, no clean denominator; plausible **~2–15×** of genuinely-needed gate passes. |
| trigger-cost | **decide** = the gate reads the in-flight recommendation (cheap, no new recall); **execute** = a reconcile/re-rank reasoning pass — **« 190s**, no full recall fire. |
| test-sketch | Harness where a known-failed lever (haiku-recall-tier, −14%) sits in-context but outranked; assert the gate surfaces and reconciles it before the recommendation ships (current skill RED ≈ 0). |
| prior-art | **#656** (please: gate analytical recommendations as artifacts keyed to prior-outcome terms), **#655** (reconcile-proposals + re-entry), **#654** (C7 anti-amnesia RED harness); notes 81, 87, 89. |
| **rating** | **CONTENDER, confidence lowered.** It targets the *true* failure mode (application/ranking), but its over-fire is an ESTIMATE and note 89 warns vague reconcile instructions only halved the failure — it must **name the rationalization** to work. |

---

### P3 — Two-speed recall quick-probe (decouples decide-cost from execute-cost)

| field | value |
|---|---|
| problem-pattern | The draft's frontmatter-enum idea **conflated DECISION cost with EXECUTION cost**: matching a classifier enum is cheap, but recall today has **no two-speed path** — any fire runs the full **190s** body. So a "cheap to decide" trigger still pays 190s to execute, which is what makes 7× borderline and 11× fatal. |
| trigger-signal | not a standalone trigger — a **mode** any caller (P1, P2's ~2/15 probe) selects: "I only need a fast surfaced+rank check, not full synthesis." |
| solution-surface | **split-skill / two-speed** — a quick-probe path (Step-0/1 surface + rank, skip Step-2 paging / Step-2.5 synthesis) vs the full body. |
| coverage | **enabler** — it does not cover moments itself; it **lowers per-fire-cost**, which is what moves P1 from "borderline-fatal at 190s" to "viable at 7×," and gives P2 its cheap recommend-boundary probe. |
| over-fire | inherits the caller's ratio; the point is the **per-fire-cost** term, not the ratio. |
| trigger-cost | **decide** = classifier match (cheap); **execute** = quick-probe (Step-2 paging is the ~43–63% dominant cost per `recall-cost-isolation.md`, so deferring it is the win). |
| test-sketch | Measure quick-probe wall-time vs full body on the same query; assert quick-probe surfaces the right note's rank without the Step-2 page-in, at a fraction of 190s. |
| prior-art | **#657** (recall: ship safe procedure-time cuts — O2/L2/L3a sweep, C7-gated); `recall-cost-isolation.md`; note 78 (split at the seam, no bespoke infra). |
| **rating** | **CONTENDER.** It is the **precondition** that makes any recall-firing trigger (P1) honest. Relabelled per critique: it fixes EXECUTION cost, which the frontmatter enum never did. |

---

### P4 — Destructive-git PreToolUse hook (KEEP)

| field | value |
|---|---|
| problem-pattern | note 25: `git restore`/`checkout` run on user-deleted files, treating deliberate deletion as data loss. A deterministic, observable tool event. |
| trigger-signal | a `git restore` / `git checkout -- ` / `clean -fd` about to run (PreToolUse on Bash). |
| solution-surface | **binary/hook** — a PreToolUse hook that pauses and asks "these deletions weren't yours — intentional?". |
| coverage | **1** named moment (note 25); class of destructive-git in general. |
| over-fire | **43 destructive-git events / 1 genuine ≈ 43×.** Fire-unit = tool event. **DERIVED.** |
| trigger-cost | **decide** = regex on the command (sub-ms); **execute** = ~0s (a prompt), **not** a recall fire. |
| test-sketch | Fire a `git restore` on a path the agent did not delete; assert the hook interposes before execution. |
| prior-art | note 25; aligns with Joe's global rule "never run destructive git without status/diff first" (new hook). |
| **rating** | **CONTENDER (KEEP).** 43× would be fatal for a recall trigger but is **fine on a free hook** — exactly the §0.1 point. This is the clearest cheap win. |

---

### P5 — Note-negation re-rank (#655) — the missing ranking fix

| field | value |
|---|---|
| problem-pattern | **No draft proposal fixed ranking** — they only made recall fire more. But note 82 says the self-recommendation misses are **notes outranked by chunks at synthesis.** The fix is in the **binary's ranking**, not the trigger. |
| trigger-signal | none — this runs **inside every existing recall**; it re-ranks so a directly-relevant note (esp. a negation/"already-tried" note) can **override** the chunk flood instead of being buried. |
| solution-surface | **re-rank (binary)** — note-negation override in `engram query` candidate ranking. |
| coverage | **~13/15** of self-recommendation (shared with P2; P5 = retrieval/ranking layer, P2 = synthesis-gate layer — they stack, not double-count). |
| over-fire | **n/a** — it adds **no new fires**; it modifies the output of recalls already happening. |
| trigger-cost | **~0s** — a ranking tweak in the binary; no extra LLM round-trip. |
| test-sketch | Vault with a negation note + many co-topical chunks for the same query; assert the negation note ranks above the chunk flood (current RED: note buried). |
| prior-art | **#655** (note-negation override + reconcile + re-entry); notes 81, 82; pairs with **#652** (recency-weighted centroid nomination) as adjacent ranking work. |
| **rating** | **CONTENDER.** Free, no over-fire, and it attacks the *actual* mechanism (ranking) the draft missed. Strongest of the recall-side proposals on cost grounds. |

---

### P6 — Capture-on-correction (write-side — honestly NOT triggering)

| field | value |
|---|---|
| problem-pattern | **28% of corrections (23 CAPTURE) had NO memory at all** — unreachable by **any** trigger. Triggering ships nothing for them. This is the hard ceiling in §0. |
| trigger-signal | a **user correction** event ("why are you…", "that won't scale", "stop", "per what caveat?") — the write side, not the read side. |
| solution-surface | **write-side / learn** — capture each correction as a reproducible note at the moment it happens (turn a correction into a guard, per note 67). |
| coverage | reaches the **23 CAPTURE moments** that the entire trigger lever cannot — but **this is the learn side, stated plainly: it is not a recall trigger.** |
| over-fire | Fire-unit = detected correction. **ESTIMATE** — ~82 corrections / 635 turns ≈ **~13% base rate**; precision depends on correction-detection quality (note 67's "make it non-recurring"). |
| trigger-cost | learn-side write; no recall fire. |
| test-sketch | Replay a session with a known novel correction (e.g. `d498a354…#8185` "why are you writing to one file?"); assert a note is captured that would TRIGGER next time. |
| prior-art | note 67 (corrections→reproducible tests); **#635** (learn: explicit-request triggers); the 23 CAPTURE moments. |
| **rating** | **CONTENDER (with the honest caveat).** It is the **only** lever that moves the 28% ceiling, but it belongs to **learn**, not triggering — included because the headline ceiling makes it indispensable, not because it is a trigger. |

---

### P7 — Go-lint conventions → `.claude/rules/go.md` (PARK: real win, wrong surface)

| field | value |
|---|---|
| problem-pattern | 5/8 tool-imminent moments are deterministic Go-lint conventions: modernize/`slices.Backward` (58), funlen/wsl scope-lift (43), `bytes.Split` trailing-newline (57) + nil-guard (59), unused-field early-consumer (37). |
| trigger-signal | editing engram `*.go` — **already** the glob scope of `.claude/rules/go.md`. |
| solution-surface | **static-doc** — extend the existing `.claude/rules/go.md` (globs `*.go`, already holds nilaway + line-length rules) with notes 58/43/57. |
| coverage | **5/59.** |
| over-fire | **always-loaded** for `*.go` (the file's glob). The "fire" is free; the denominator (944 .md edits, 105 SKILL.md, etc.) does not gate it. **DERIVED denom.** |
| trigger-cost | ~0s — a few always-present lines; no recall fire. |
| test-sketch | Write an engram `*.go` file with a backward for-loop; assert the in-scope rule already steers to `slices.Backward` without any recall. |
| prior-art | notes 37/43/57/58/59; existing `.claude/rules/go.md`. |
| **rating** | **PARK (real win, wrong surface for *triggering*).** These are genuine recurring failures, but the fix is a **static lint rule**, not a recall trigger — kept exactly as the critique asked. |

---

### P8 — Cadence default + results-table rule → CLAUDE.md (PARK: real win, wrong surface)

| field | value |
|---|---|
| problem-pattern | recurring **behavioral** conventions: recency/decay defaults assumed daily cadence vs Joe's monthly cycle (notes 45, `c1195a11…#1272`); eval results narrated in prose instead of a labeled table (`a19e7b75…#5589`). |
| trigger-signal | "about to set a time-default" / "about to present eval results" — both already covered by always-loaded memory. |
| solution-surface | **CLAUDE.md-pointer / static-doc** — these are already partly in Joe's memory index (table-rule, cadence); consolidate as standing rules, not triggers. |
| coverage | **3/59.** |
| over-fire | always-loaded; free. |
| trigger-cost | ~0s. |
| test-sketch | Ask for eval results; assert the standing rule produces a labeled criteria table unprompted (no recall fire needed). |
| prior-art | notes 45/`c1195a11#1272` (cadence), `a19e7b75#5589` (table) — both already in `MEMORY.md`. |
| **rating** | **PARK (real win, wrong surface).** Behavioral conventions belong in always-loaded CLAUDE.md/memory, not in a per-fire recall trigger. |

---

### P9 — Done-claim verification trigger (PARK: 380× over-fire)

| field | value |
|---|---|
| problem-pattern | step-boundary (4/59): declaring a removal/CLI/feature "done" before verifying (notes 54, 64, 66, 90). A real recurring failure. |
| trigger-signal | a **done-claim** in an assistant turn ("complete", "done", "passing", "shipped"). |
| solution-surface | **skill-body / Stop-hook** — fire a verification recall on every done-claim. |
| coverage | **4/59.** |
| over-fire | **~1141 done-claim assistant turns / 3 genuine ≈ 380×.** Fire-unit = assistant turn. **DERIVED.** |
| trigger-cost | **decide** = phrase match (cheap); **execute** = 190s recall — and 380× of them. |
| test-sketch | Count done-claims vs genuine premature-done failures in a session; assert the ratio (it is ~380×). |
| prior-art | notes 54/64/66/90; superpowers `verification-before-completion`; loosely **#638**. |
| **rating** | **PARK.** 380× × 190s is catastrophic. Demonstrates the §0.1 model in reverse: a real pattern that is **fatal as a recall trigger**; if pursued, it must be a free Stop-hook checklist, never a recall fire. |

---

### P10 — Binary-staleness PostToolUse hook (free)

| field | value |
|---|---|
| problem-pattern | note 106: ran `targ build` (no such target); a stale binary on PATH made measurements read as unchanged — a near-mis-conclusion. |
| trigger-signal | a `targ build` (or other non-existent target) invocation, or a stale-mtime engram binary — a deterministic tool event. |
| solution-surface | **binary/hook** — PostToolUse hook on Bash matching `targ build`, emitting "no `targ build` target; install with `go install ./cmd/engram`." |
| coverage | **1/59** (note 106); class of stale-binary measurement bugs. |
| over-fire | rare (only on the matched command). Fire-unit = tool event. **ESTIMATE** (no `targ build` denominator measured — flagged). |
| trigger-cost | ~0s — a hook print; no recall fire. |
| test-sketch | Run `targ build`; assert the hook prints the `go install ./cmd/engram` correction. |
| prior-art | note 106; engram CLAUDE.md ("there is no `targ build` target"). |
| **rating** | **CONTENDER (free).** Deterministic, free, and prevents a silent measurement-invalidation bug. Cheap second hook alongside P4. |

---

## 4. Recommendation (within the set — no member deleted)

**Ship order, by cost regime (cheapest-first), all 10 retained:**

1. **Free deterministic surfaces first — P4, P10, P7, P8.** Zero per-fire cost, the over-fire bound does
   not bind them, and they capture **9 TRIGGER moments** plus the whole go-lint/cadence/table recurrence
   for ~no wall-time. P7/P8 are **PARK *as triggers*** but **real wins on their proper static surface** —
   land them there.
2. **The ranking + application fix next — P5 then P2.** P5 (#655 note-negation re-rank) is free, adds no
   fires, and attacks the **actual** self-recommendation mechanism (note 82) the draft missed; P2 (#656
   gate) stacks the synthesis-side reconcile on top. Together they address the **25% cluster that "fire
   recall more" cannot.** Gate both behind the **#654 C7 RED harness** confirming ≈0 baseline first.
3. **The enabler, then the one viable recall trigger — P3 then P1.** P3 (#657 two-speed) must land before
   P1, because P1's 7× is only honest once a fire is a quick-probe, not 190s. P1 is the **only**
   recall-firing trigger that clears the bound — and only for the narrow, over-fit eval-design slice.
4. **Write-side to lift the ceiling — P6.** The single lever that touches the **28%** triggering can never
   reach; sequence it independently on the learn side.
5. **P9 stays PARK** — documented as the cautionary 380× case, pursued (if ever) only as a free Stop-hook
   checklist, never a recall fire.

**Bottom line:** the high-leverage moves are **not** "fire recall more" — they are **free surfaces (hooks,
static docs), a free binary re-rank (#655), and a synthesis gate (#656)**, with exactly **one** narrow
recall-firing trigger (P1) that is viable only after the two-speed split (P3). And none of it moves the
**28% no-memory ceiling**, which is P6's write-side job.
