# Engram validation — experiment log

Living tracker for the cumulative cross-app memory eval (`dev/eval/cumulative/`). One row per
**run**; runs are grouped by **variant** (the memory-activation mechanism under test). Update this
file as runs complete — record the data path, the headline amortized result, and the retro verdict.

> **Verify, don't guess.** Numbers here are copied from `aggregate.py --root <data path>` output or
> direct per-op JSON reads — never from memory. If a row says "running", it has no trusted result yet.

---

## Target axes (what every run measures)

| Axis | Question | Primary metric (unit) | Exercised by current chain? |
|---|---|---|---|
| **C1** | faster? | active time = recall+build+learn (s); build time (s) | ✅ |
| **C2** | cheaper? | cost (USD); tokens (Mtok) | ✅ |
| **C3** | fewer interactions? | convention restatements (count); feedback rounds (rounds) | ✅ |
| **C4** | adapts as standards change over time? | (separate probe — not in the 3-app chain) | ❌ not yet |
| **C5** | uses recent history? | recency probe (separate) | ❌ not yet |
| **C6** | emergent synthesis / compounding lessons? | crystallize-new-note + adversarial judge (separate fixtures) | ❌ not yet |

**Reporting rule (learned 2026-06-21):** report the **amortized** view — separate the app1 **seed**
(one-time memory write, no prior memory to recall) from the **payback** apps (2..N). Chain totals
smear the seed cost across every app and misread it as a per-app penalty. Memory's economics are
"pay once to seed, win on every subsequent app."

---

## Variant registry (the memory-activation mechanism under test)

What differs between variants is **how the agent is prompted to use memory** — the recall/learn
machinery and the binary are identical. Cold (no-memory) is the shared baseline in every run.

| ID | Variant | Manipulation | Hypothesis |
|---|---|---|---|
| **V0** | **Baseline** (current) | Build prompt embeds an explicit per-op directive: "before writing code, INVOKE /recall"; after convergence, "INVOKE /learn". | Memory cuts C3 and (amortized) C1/C2 on payback apps. |
| **V1** | **please-driven** | Every instruction/correction is issued via **/please <ask>** — the please workflow orchestrates recall→plan→build→learn with its review gates, instead of a hand-embedded recall/learn directive. | Please's structured recall + gates lift reliability of memory use and first-pass quality vs. a bare directive. |
| **V2** | **CLAUDE.md advice** | Remove the explicit per-op recall/learn directive; instead place ambient recall/learn guidance in **CLAUDE.md** and rely on the agent to self-trigger. | Ambient advice is weaker than an explicit directive — tests whether memory still gets used without per-prompt nudging. |
| **V3** | **Combination** | please-driven **and** CLAUDE.md advice together. | Best-case activation: structured workflow + ambient reinforcement. |

> Assumptions to confirm with Joe: (1) V1 wraps both the initial build instruction and each
> feedback correction in /please; (2) V2 strips the build-prompt recall/learn lines entirely (pure
> ambient); (3) all variants keep cold as the no-memory control. Adjust the manipulations above if
> these don't match intent.

---

## Run log

| # | Variant | Model | n | stall patience | engram SHA | data path | status | headline (payback: conv / cost / build time) | retro |
|---|---|---|--:|--:|---|---|---|---|---|
| 1 | V0 | haiku | 1 | 2 | ce80225c | `/tmp/cummatrix-n1` | done (pilot) | conv −38%* / — / — (n=1, high variance) | clean; advance to n=5 |
| 2 | V0 | haiku | 5 | 2 | ce80225c | `/tmp/cummatrix-n5` | done | conv −17% / cost −11% / build −23% | real & significant (p<0.001); seed-vs-payback fix found |
| 3 | V0 | haiku | 5 | 3 | (current) | `/tmp/cummatrix-n5p3` | done | conv payback −10% (within noise); cost sign-FLIPPED vs #2 | n=5 underpowered; cost noise-dominated |
| 4 | V0 | haiku | 10 | 3 | (current) | `/tmp/cummatrix-n10more` | **CONTAMINATED — discard** | — | server outage produced 5/20 degraded warm builds (1-round, <$0.40 no-ops scoring phantom conv=0); inflated effect to −5/−6, not trustworthy |

| 5 | V0 | haiku | 10 | 3 | (current) | `/tmp/cummatrix-n10b` | done (clean) | conv payback −33% (gap −4.5); 0 degenerate | replaces contaminated #4 |

**POOLED n=20 (clean: #2 n5 + #3 n5p3 + #5 n10b; #4 discarded), C3 convention metric (round-1,
patience-invariant):** payback gap **−3.15 (−23%), z≈−4.41, permutation p=0.0001** — CONFIRMED
significant, direction unanimous (all 3 apps negative, warm < cold every app). **Magnitude
unstable batch-to-batch** (per-run gaps −2.4 / −1.2 / −4.5) — warm has high variance (±2.89 vs
cold ±1.36) because recall is probabilistic (sometimes eliminates a convention entirely, sometimes
misses). Quote as "~15–25%, significant", not a precise point. **C1/C2 (cost/time) remain
noise-dominated at n=20 — not a defensible finding. C4/C5/C6 not exercised.**

\* n=1 magnitude was a high-variance draw; n=5 settled C3 at ~−17% per-app (−12% cold-anchored chain).

### Run 6 — V0 · sonnet · n=1 (`/tmp/cummatrix-sonnet-n1`) — DIRECTIONAL ONLY

Clean (6/6, 0 rate_limited, 0 degenerate, $13.36). Convention payback (links+feeds): cold 12 →
warm **1** (gap −11, ~92%). Warm round-1 arch climbs 9→9→**10/10** across the chain (cold 3→4→4) —
genuine memory front-loading, not phantom builds (all warm ≥$1.69, recall fired, real scored apps).
**NOT a finding:** −92% is pinned to the metric floor (warm payback = {1,0}); the "grows with model
strength" read illegitimately compares sonnet n=1 vs haiku n=20. Retro verdict: DIRECTIONAL-NEEDS-N.
**Next:** sonnet n≥5 (target 10), then permutation-test sonnet warm-payback distribution vs haiku
n=20 — only then can model-strength scaling be promoted to a finding.

| # | Variant | Model | n | patience | data path | status | headline (C3 payback) | retro |
|---|---|---|--:|--:|---|---|---|---|
| 6 | V0 | sonnet | 1 | 3 | (folded into n5) | superseded | conv payback 12→1 (~92%, FLOOR-PINNED) | directional only; needs n≥5 |
| 7 | V0 | sonnet | 5 | 3 | `/tmp/cummatrix-sonnet-n5` | done (clean) | conv payback 12.60→4.80 (−62%, p=0.008) | n=1's −92% regressed to −62% as predicted; reportable |

**CROSS-MODEL FINDING (haiku n=20 vs sonnet n=5):** cold payback ~identical (13.55 vs 12.60 — same
burden without memory); warm payback 10.40 vs 4.80. Memory cut: **haiku −23% vs sonnet −62%**,
permutation test on warm-payback distributions **p=0.0020**. → *Memory's value grows with model
strength* — a stronger model applies recalled conventions more reliably on the first draft (sonnet
warm round-1 arch front-loads to 10/10). Caveat: sonnet n=5, high warm variance (±2.93); n=10 would
tighten the −62% point. Direction is significant.

### Run 8 — V0 · opus · n=5 (`/tmp/cummatrix-opus-n5`) — SURPRISING: oracle saturation

Clean (30/30, 0 true no-ops; note: 2 builds tripped the cost<$0.40 degenerate filter but were
FALSE POSITIVES — real arch-10/10 one-shot builds. Correct no-op discriminator = score=None OR
turns≤3, NOT cost). Convention payback: opus cold **3.80±5.19** / warm 6.00±2.68 — memory shows NO
benefit (within noise; the +58% is the t1=14 outlier, not real).

| # | Variant | Model | n | data path | status | conv payback | note |
|---|---|---|--:|---|---|---|---|
| 8 | V0 | opus | 5 | `/tmp/cummatrix-opus-n5` | done (clean) | cold 3.80 / warm 6.00 (+58%, NOISE) | opus one-shots clean cold → oracle floors |

**KEY FINDING — memory benefit is NON-MONOTONIC / oracle-saturation:** opus cold per-trial conv
[14,3,0,1,1] — trials 3-5 opus produces fully convention-compliant apps UNAIDED (arch 10/10,
conv=0). The n=1 (trial 1) was an unlucky hard draw (arch 3-4/10), which falsely made memory look
huge. Cross-model: haiku cold 13.6 (−23%), sonnet cold 12.6 (−62%), opus cold 3.8 (no effect).
**RETRO VERDICT (confirmed): "non-monotonic, peaks at sonnet" REFUTED — it's ORACLE SATURATION.**
Cold-flooring rate (conv≤1): opus **70%** (7/10), sonnet 0%, haiku 2%. When opus one-shots conv=0
cold, the metric physically can't register memory value. The +58% rides on the t1=14 outlier;
strip it → opus cold median 1 (floored). Paired: the only opus trial with real cold burden (t1=14)
had warm crush it (−9, memory-helps signature). **Measured benefit tracks COLD CONVENTION-BURDEN,
not model strength** — opus has ~0 cold burden on these easy CRUD apps, so its benefit is
UNMEASURED, not zero.

**CAN report:** sonnet −62%, haiku −24% (where cold burden exists). **CANNOT report:** any opus
memory claim, or a model-strength curve. C2 cost: opus warm +160-260%, no measurable payback (but
payback is unmeasurable here anyway).

**REQUIRED before any opus/strong-model claim — harder test cases (see issue):**
1. Idiosyncratic NON-DEFAULT conventions (bespoke error format, mandated internal helper over
   stdlib, forbidden-but-tempting stdlib call) — not LLM house-style (DI/wrapped-errors) opus
   applies unprompted. Memory carries ARBITRARY LOCAL convention, not universal good taste.
2. Conventions that CONFLICT with the model's prior ("do X the unusual way") — cold fails, warm
   recalls the local rule.
3. Longer accumulating chains (app4/app5) stacking conventions beyond one-shot range.
4. Pre-flight SATURATION GATE in harness: require cold median conv ≥ threshold per model; if a
   model floors cold (>10% builds at conv≤1) declare benefit UNMEASURABLE, don't report a number.

**Follow-up (2026-06-22): [OPUS-TRAP-CATALOG.md](OPUS-TRAP-CATALOG.md)** — instead of inventing
synthetic hard conventions, mined the user's session history (38 opus transcripts + vault feedback
notes + CLAUDE.md/go.md rules) for real opus correction-traps. Yields buildable, deterministically-
checkable, high-cold-falls-in exercises (idiosyncratic local code conventions: `slices.Backward`,
nilaway guards, `crypto/rand`, `AI-Used` trailer, `targ` not `go test`) that opus re-commits cold.
Each exercise = one tiny Go task + a one-line grep/lint check, with a cheap cold-confirm-first
protocol that doubles as the per-case saturation gate. Spec only; no runs yet.

### Planned next
- [ ] V0 sonnet n=5 (after run #3 retro passes) — ~$110
- [ ] V0 opus n=5 — ~$400
- [ ] V1 (please-driven) — model/n TBD
- [ ] V2 (CLAUDE.md advice) — model/n TBD
- [ ] V3 (combination) — model/n TBD
- [ ] Wire C4 / C5 / C6 probes into a run (currently separate/unexercised)

---

## Results detail

### Run #2 — V0 · haiku · n=5 · patience=2 (`/tmp/cummatrix-n5`)

Amortized economics (warm vs cold; negative = memory better):

| segment | conv (count) | rounds | build (s) | cost (USD) | tokens (Mtok) | total active time (s) |
|---|--:|--:|--:|--:|--:|--:|
| seed (app1) | −17% | +0% | −20% | **+81%** | **+136%** | +4% |
| payback (2–3) | **−17%** | **−6%** | **−23%** | **−11%** | **−13%** | **−5%** |

Read: app1 is a one-time investment (warm pays +81% cost / +136% tokens to *write* memory); apps
2–3 are net wins on every axis. C3 cut significant (permutation p=0.0004 all / p=0.0067
converged-only); cold convention count is a near-point-mass at 7.0/app.

### Run #3 — V0 · haiku · n=5 · patience=3 (`/tmp/cummatrix-n5p3`)

_Running. C3 is stall-invariant (won't move from #2); C1/C2 + convergence refresh under the
recalibrated stall. Fill in from `aggregate.py --root /tmp/cummatrix-n5p3` when complete._

---

## Decisions & lessons (chronological)

- **2026-06-21** — Recalibrated stall early-stop patience 2→3 (it cut 14/30 builds with budget to
  spare; original motivation was the cmd/-layout build bug, since fixed). Reporting fixes: per-build
  convergence (not the all-3-product), and the amortized seed-vs-payback table. validate.py 57/57.
- **2026-06-21** — Headline must be the **cold-anchored chain (~−12%)** or the **amortized payback
  view**, not the per-op 3× number (~−17%) which credits memory for the un-seeded app1.
- **2026-06-21** — Two independent n=5 draws (#2 p=2, #3 p=3) showed **C1/C2 (cost/time) are
  noise-dominated** — cost gap flipped sign (#2 −0.32 within noise → #3 +1.35) because build cost is
  round-count-dominated (2–8 rounds/build). Retracted run #2's "warm cheaper −11%" — single-draw
  over-confidence. Only C3 (conventions) has signal, and even it needs pooling/more n.
- **2026-06-21** — `convention_statements` confirmed **round-1-based** (harness.py:814 = round-1
  convention failures, the say-once metric) — NOT round-count-inflated. Its variance is genuine
  first-draft variance (cold isn't a fixed 7/app point-mass — the model sometimes gets a convention
  right unaided). Patience-invariant, so poolable across runs.
- **2026-06-21** — Attribution note: warm seed (app1, empty memory) still beats cold (6.2 vs 7.0),
  so part of the benefit is the recall-instruction *priming*, not memory content. Joe's call: the
  product-level cold-vs-warm comparison is the right question (the recall step IS part of engram) —
  no warm-empty control arm needed; bump n instead.
- **2026-06-21** — Parallelism: dropped the cold path's artificial DAG dependency (cold writes no
  vault — its 3 apps share nothing, so chaining them was needless serialization). Peak parallelism
  rises from 10 (regime×trial chains) to ~20 (15 independent cold ops + 5 warm chain-frontiers).
  Warm stays sequential (app2 recalls app1's learned notes — a real data dependency). Run with
  `--workers 20`; >20 buys nothing (only ~20 ops ever ready at once).
- **2026-06-21** — Op flow confirmed: per app, **instruct(+recall-first) → build/iterate → learn**;
  across the chain `learn(N)+ingest(N) → recall(N+1)`. `recall_s` is round-1 wall (recall + first
  draft), not isolated recall latency.
- **2026-06-23** — **Slice 1: cross-cluster linking (recall Step 2.6) — BUILT, end-to-end proven.**
  Harness `dev/eval/traps/cake.py`. The cake RED baseline **falsified the design premise**: k-means
  groups by shared *property*, not req-vs-mech domain, and the current skill already forms cross-note
  links — but imprecisely (**4/9 correct means-ends, 5 spurious flood**). Reframed slice 1 around
  **precision** (user-approved). Added Step 2.6 precision gate (directed relations + 1:1 shared key +
  hub test, default DROP) governing within- and cross-cluster linking. Final (opus, n=3–4):
  **8–9/9 correct means-ends, 0 flood; control cake+git 0 cross-links; analogy 0.** Precision is the
  whole game — symmetric part-whole/abstraction relations disabled as the flood vector.
- **2026-06-23** — **Slice 2: graph-expanded retrieval (`query.go` BFS) — BUILT, end-to-end proven.**
  Slice 1 fixed the *write* gap (precise edges); slice 2 fixes the *retrieval* gap — the bridge note
  cosine never sees because it is embedding-distant from the query. **Graph expansion surfaces bridges
  cosine missed:** the transitive bridge `sugar-provides-sweetness` is **absent** from
  `clusters[].members` cosine-only but **present** with BFS expansion (proof
  `dev/eval/traps/graphexpand.py`, reference-based, deterministic). Wired `vaultgraph.BFSWithCap` into
  `engram query`: expand the cosine-matched seeds 1–2 wikilink hops (`--graph-expand-hops`, default 2;
  negative disables) before clustering, reusing slice-1's body `Related to:` edges (verified
  `ParseWikilinks`-traversable). Link-density caveat: bridges surface only where a slice-1 edge exists.
