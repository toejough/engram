# Vocab lifecycle liveness — investigation plan

> **For agentic workers:** investigate-and-propose ask. The deliverable is a PROPOSALS doc Joe
> reviews — NO build this round. Measurements are free/cheap; no production changes. (Amended
> post-Gate-A: O0 recalibration-only added; O4 pre-judgment stripped; the detection/action reframe
> scoped to O2 only; measurement labels honestied; trigger grid pinned with joint replay.)

**Ask (Joe, 2026-07-03, condensed):** think end-to-end about keeping the vocabulary live and
effective. Learn isn't the only note writer — if vocab checks must follow every write, recall needs
them too. Joe's lean: learn is the sensible check site (to avoid re-inflating recall's cost), but
the actual impact is unknown. Investigate a few options balancing liveness/effectiveness vs time/$,
and present proposals.

## Facts, labeled honestly (2026-07-03)

**Measured:**
- `engram vocab stats`: 22 ms, ~237 tokens of output. It already flags `[hub]`/`[orphan]` per term
  and vault-wide untagged-rate — but emits NO verdict line; threshold-comparison output is NET-NEW
  binary work (~50–100 LOC incl. tests; code-verified against `vocab_commands.go:267`).
- Vault: 141 memory notes; growth 10/37/75/19 per week (week 4 partial — 4 days in).
- Untagged today: 3.6%. Write-time threshold hooks: NONE exist today (verified — learn/amend's
  assignment functions `applyVocabAssignmentAfterLearn`/`...AfterAmend` have no threshold logic).
- Payload flag feasibility: a top-level `refit_pending` omitempty field ≈5 tokens when set
  (verified against the queryPayload struct; mechanically trivial).

**Estimated (not field-verifiable):** note-origin split ≈ learn 55 / recall-2.5 ≈46 / synthesis 27 /
eval-other 13 (sums to 141; the `source:` field does not reliably distinguish recall-2.5 writes from
learn writes — keyword heuristic). Direction is safe: **recall writes roughly half the vault's
notes.** `resituate` is a further write site (rewrites situation + sidecar — check whether it
re-assigns vocab; include in the coverage matrix).

**Projected (from growth data, not a script run):** the documented +30%-growth refit trigger
(baseline: `2026-07-02-vocab-notes-and-linking-replacement.md:86` — the shipped trigger set is
(a) untagged-rate >10% of last 25 writes, (b) any term >25% of vault, (c) vault grew >30% since
last refit) would have fired **~8 refits in 3 weeks** at the observed growth (~$16 + tag churn +
gate runs). Percentage triggers run hot on a small base. Step 1 replaces this projection with a
real replay.

## The write-site coverage question (Joe's core concern, stated precisely)

Recall writes ~half the notes (2.5C amends/learns + Step 4 synthesis + any resituate use), so any
design whose DETECTION only runs at learn-time has a staleness window = all vocab-relevant events
between learn runs. Whether that window matters depends on trigger speed (all current triggers are
slow-moving aggregates). **The "detection persists so the write site stops mattering" claim is the
O2 DESIGN, not a property of O1** — O1's window must be modeled honestly, not assumed away
(Gate-A correction).

## Options to model — the full set, none pre-judged

- **O0 — Recalibrate only, placement unchanged:** keep today's reality (stats on demand, no
  automatic check anywhere) but fix the trigger set + document a human/agent cadence. Zero
  implementation. Models the "is the trigger the whole problem?" hypothesis.
- **O1 — Learn-anchored stateless:** learn's sweep runs `vocab stats` (with the NEW verdict line);
  a skill conditional keyed to the observable verdict acts (refit flow / propose). Recall untouched.
  Staleness window = vocab events between learn runs (model it: heavy /please day ≈ 10+ recall
  writes before the closing learn).
- **O2 — Binary-persistent flag:** threshold evaluation hooked into BOTH assignment call sites
  (learn + amend — recall's writes flow through amend/learn; verified hook points exist), verdict
  persisted as `refit_pending` + reason in **`vocab.centroids.json`** (code-recommended home:
  machine-maintained, survives `embed apply`, already versioned; index frontmatter would be
  clobbered by regen). Learn's sweep surfaces + acts; flag clears when the action runs. Optional
  visibility: the ≈5-token payload field. NOTE (code-verified): the windowed "last-25-writes
  untagged" trigger needs a write-history ring buffer — the HEAVIEST sub-piece; the vault-wide
  untagged-rate variant is nearly free. Model both.
- **O3 — Scheduled autonomous maintenance:** a weekly headless agent runs stats → refit-if-tripped
  → regression gates. **Named tension (ROADMAP:99):** Joe rejected HOOKS for recall — but that
  rejection was about procedural inflation during agent reasoning (over-fire 147–380× at ~190s/fire);
  O3 is background GC in isolation, ~$0 when healthy, ~$2 when tripped — distinguishable, but it IS
  a scheduling dependency (harness-side; none exists in-repo, verified) and re-opens an
  autonomy-adjacent decision → **Joe's explicit sign-off is a gate criterion for O3.** Precedent:
  compound-engineering fires its researcher inside `/ce-plan` — automatic within a workflow, manual
  at workflow level (`2026-07-02-research-followups.md:129–145`); O1/O2 are that pattern, O3 goes
  beyond it.
- **O4 — Recall-side checks:** recall's skill gains the same observable-verdict conditional as
  learn. Modeled with real numbers like every other option — per-recall-fire overhead (added
  steps/tokens × fire frequency), against the recall-economics constraints (140/141/144). The
  RATING follows the modeling.

**Trigger recalibration (cross-cutting AND standalone as O0):** keep (a-vault-wide untagged
variant) and (b) hub >25% (justified: both are utility signals that never fired falsely in the
projection); recalibrate (c): candidates = absolute growth ∈ {30, 40, 50} notes × minimum interval
∈ {7, 14, 30} days × untagged ∈ {vault-wide >8%, windowed >10%/25 (heavy — needs ring buffer)}.
**Replay the full candidate grid JOINTLY against vault history** (per note 161: verify conjuncts
co-occur; report per-conjunction fire counts AND which subsets never co-fire) — a real script this
time, committed with the proposals.

## Cost model (the deliverable's comparison table — one row per option incl. O0 and O4)

Columns: added tokens/steps per learn fire · per recall fire · staleness window (events, not just
time) · autonomy (fires without a human/workflow?) · agent-reliability class (observable predicate
vs judgment — note 145) · implementation cost (binary LOC / skill edits / infra) · $ per month
across growth scenarios {each observed week, steady-40/week} reported as a range + sensitivity.

## Steps

0. Write-site audit: confirm resituate's vocab behavior; finalize the coverage matrix (who writes,
   what flows through the assignment hooks).
1. Build + run the trigger-replay script (real, committed): full joint grid vs vault history.
2. Fill the cost model; stress the three common cases end-to-end: a heavy /please day (10+ recall
   writes, 2 learn runs), an idle week, a new-domain influx (untagged climbing).
3. Draft the proposals (O0–O4 + recommendation + honest bounds + what each does NOT catch) →
   `docs/design/2026-07-03-vocab-lifecycle-proposals.md`; note the ROADMAP Track-A integration the
   winning option implies (deliverable line, executed on Joe's go).
4. Gate C; commit (Gate D); PRESENT to Joe and STOP.

## Constraints
- No production changes; measurements/replays read-only against the vault. Labeled tables with
  units. Every claim carries its label (measured / estimated / projected / replayed).
- The recommendation must respect: observable-predicate reliability (145), recall economics
  (140/141/144), relocation≠reduction (108 — stated plainly: the CHECK is ~free everywhere; only
  the ACTION costs, and it fires conditionally), fire-unit pinning (109), joint-conjunct
  calibration (161).
