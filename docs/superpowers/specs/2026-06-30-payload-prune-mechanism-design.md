# Payload-prune smoke-test plan — validate the isolation premise (production deferred)

**Ask (Joe, 2026-06-30):** brainstorm the payload-prune *mechanism*, pick an option, and validate it with quick
smoke tests. This is **brainstorm + smoke-validate the premise**, NOT a production build (deferred — see end).

## Problem (verified, notes 100/95)

recall's `engram query` payload (~97 KB, post lazy-chunks) is **carried** in the build context: the `$METER`
harness's Phase-2 build **`--resume`s the recall session** (`dev/eval/cumulative/harness.py:716`, verified), so
every build round re-sends the payload as `cache_read` → the **~$1/op warm-over-cold premium** (note 95: "the
resumed recall context is re-read as cache every build turn"). The payload *bytes* are cheap once (note 100 —
size caps move *time*, not dollars); **carrying** them across N build rounds is the dollar cost. The *value* —
Step 3's "apply these as requirements" synthesis — is a few hundred tokens.

## The lever, and the honest challenge

payload-prune = after Step 3, the build carries **only the synthesis**, not the raw payload. It's the **only
verified $ lever** (note 100; size-caps, whole-op/split haiku, phrase-cuts all refuted). **Honest challenge
(notes 77/91):** recall is dollar-light; the ~$1 is *small* vs the $2–4 build, and dropping the payload could
**cost rebuild rounds** if the synthesis under-captures (note 107). So the smoke must show the $1 is **real AND
capability-safe**, on the current model — or it dies cheaply.

## Mechanisms considered (with tradeoffs)

| # | Mechanism | Cost effect | Capability risk | Production-architecture cost | Verdict |
|---|---|---|---|---|---|
| 1 | **Synthesis-injection** — build starts in a fresh session with the Step-3 synthesis injected; no resume | drops the carried payload (the full saving) | synthesis must fully capture what the build needs (it's the only thing carried) | **smoke proxy only** — as a *product* it still needs recall to emit + the build flow to inject (touches recall's inline design) | **chosen for the smoke** (cheapest faithful proxy of the saving) |
| 2 | **Subagent-isolated recall** — recall runs in a subagent, returns only the synthesis to the parent build | same payload isolation as (1) | same, + the subagent↔parent return path could reformat/truncate the synthesis | the real product form; shifts recall off inline crystallization | **deferred** to a separate build IF the smoke validates |
| 3 | **Context-edit / compaction of the resumed session** | could drop the payload mid-session | — | Claude Code has **no targeted tool-output removal**; auto-compaction is untargeted | **REJECTED** (uncertain/unbuildable) |

**(1) and (2) deliver the same cost isolation** (carry only the synthesis). They differ in *how the synthesis
reaches the build*: (1) injects it into a fresh session's first message; (2) returns it from a subagent. So the
smoke via (1) validates the **isolation premise** — *does carrying only the synthesis save the $ and hold
capability?* — which is the load-bearing question for BOTH. It does **not** prove (2)'s specific return-path
fidelity; that's a risk (2)'s production build must re-check.

## Chosen approach

**Validate the isolation premise via the synthesis-injection proxy (1).** A GREEN smoke means the premise holds
(dropping the payload saves $ without a capability hit) → it justifies building the production form (2). It is
**not** authorization to ship (1) inline. A RED smoke kills the lever cheaply. (Measure-before-build, per
#661→#663.)

## Smoke test (measure-first; the de-risk)

A variant of `dev/eval/cumulative/harness.py`, reusing the `recall_cost`/`build_cost` `$METER` split (schema v5;
`build_cost` = sum of build-round costs, excludes recall — verified `harness.py:657-662`).

**Buildability fix (Gate-A code review):** the current `recall_only_prompt` prints a *one-line* summary, so
`recall_res["result"]` does NOT hold the Step-3 synthesis. Add **`recall_synthesis_prompt(app)`** — instructs the
agent to emit the **full Step-3 block** (the count line, per-action bullets, and the "Apply these as
requirements:" list) as its final message. Run **one** recall per app with this prompt; it yields BOTH the
session (for Arm A to resume) AND the synthesis text in `recall_res["result"]` (for Arm B to inject) — so the two
arms share the *identical* recall and differ only in carry-vs-drop.

- **Arm A — Carried (current):** `do_build(build_msg, resume_sid=recall_sid)` — resumes the recall session
  (payload carried across all build rounds).
- **Arm B — Pruned:** `do_build(recall_res["result"] + build_prompt(..., include_recall=False), resume_sid=None)`
  — a fresh build session with only the synthesis text injected (`include_recall=False` avoids a second recall —
  verified `harness.py:148`).

**Fixtures:** model **opus-4.8[1m]** (the real model — re-measure, don't inherit the old warm-vs-cold figure,
note 146); **n = 3 apps**, chosen toward **multi-round** builds (the rounds/under-capture risk only surfaces when
a build takes >2 rounds — note 95: easy 2-round CRUD has zero rebuild waste to expose). 1 recall + 2 builds
(A,B) per app.

**Measures + verdict-gate:**
1. **$ (premise):** `build_cost(B)` vs `build_cost(A)`. Net-win needs B cheaper by a margin **above the noise
   floor** (size the floor from a same-arm A-vs-A contrast — note 96).
2. **Capability (note 107):** `rounds_to_converge` + `completed`/success. Tolerance: **B ≈ A** = rounds within
   **±1** AND success within the same-arm noise (A 3/3 vs B 3/3 = tied; A 3/3 vs B 1/3 = regression).
3. **Verdict:** net win **iff** ($ cheaper above noise) **AND** (capability not regressed). If B saves $ but
   costs rounds → **not a win** (under-capture). If the $ delta is below noise → **underpowered, "can't
   distinguish at this n,"** NOT a tie (note 96).

**Results template (labeled, with units — Joe's standing requirement):**

| Metric | Unit | Arm A (carried) | Arm B (pruned) | Δ | vs noise | sub-verdict |
|---|---|---|---|---|---|---|
| build_cost | USD | _fill_ | _fill_ | _fill_ | above/below | ✓/✗ |
| rounds_to_converge | rounds | _fill_ | _fill_ | _fill_ | within ±1 | ✓/✗ |
| success | n/N | _fill_ | _fill_ | _fill_ | within noise | ✓/✗ |
| **Net win** | — | — | — | — | — | win / not-a-win / underpowered |

**Spend estimate (note 101 — derive before launching):** ~$7/warm cell (prior runs) → per app ≈ 1 recall (~$0.5)
+ 2 builds (~$3 each) ≈ ~$6.5; ×3 apps ≈ **~$20** (estimate; report actual; no cap, but confirm before the run).

**Honest bounds:** small n; the ~$1 (note 95) surfaced on *easy* builds that converged cold (memory
net-negative there), so the smoke measures the *recoverable* slice on this model/harness, not a universal figure;
and (1) is a **proxy** for the production form (2) — it validates the isolation premise, not (2)'s return-path.

## Smoke results (2026-06-30) — GREEN (WIN)

Ran the synthesis-injection proxy on **all three** harness apps (the matrix defines exactly these — feeds/8
checks, links/7, notes/4; "pick the 3 hardest" reduces to "use all 3"), model `claude-opus-4-8`, one recall +
two builds (A carried, B pruned) per app. Every synthesis was substantial (4375 / 3620 / 3896 chars — the
hardened `<500`-char abort never tripped), so no arm ran on a strawman. Actual spend **$24.16** (recall $4.51 +
Arm A $12.26 + Arm B $7.39), ~57 min — matching the ~$20 estimate.

| Metric | Unit | Arm A (carried) | Arm B (pruned) | Δ | vs noise | sub-verdict |
|---|---|---|---|---|---|---|
| build_cost (total) | USD | 12.26 | 7.39 | **−4.86 (−40%)** | ≫ $0.50 stand-in | ✓ |
| rounds_to_converge (total) | rounds | 6 | 6 | +0 | within ±1 | ✓ |
| success | n/N | 3/3 | 3/3 | 0 | tied | ✓ |
| **Net win** | — | — | — | — | — | **WIN** |

Per-app (B cheaper every time, identical rounds, identical success, all converged):

| App | checks | build_cost A | build_cost B | Δ | rds A→B | ok A/B |
|---|---|---|---|---|---|---|
| feeds | 8 | $4.50 | $2.49 | −$2.01 (−45%) | 2→2 | ✓/✓ |
| links | 7 | $3.94 | $3.04 | −$0.90 (−23%) | 2→2 | ✓/✓ |
| notes | 4 | $3.81 | $1.86 | −$1.96 (−51%) | 2→2 | ✓/✓ |

**Verdict: the lever is validated.** Pruning the payload saved **~40% of build_cost (~$1.6/app)** with **zero
capability regression** — identical rounds (2/2/2), identical success (3/3), same final convergence + arch 10/10
on every app, so the synthesis fully captured what the build needed (no under-capture, note 107). The premise
(note 95's ~$1/op) **held — if anything an underestimate** on these apps. The saving is **mechanistic, not
noise**: the gap shows in *every* build round (e.g. feeds round-2 A $2.35 vs B $1.14), consistent with the ~97 KB
payload re-reading as `cache_read` each turn — not stochastic variance.

**Honest bound (note 96):** n=1 per app; I did **not** measure an empirical same-arm A-vs-A noise floor, so
strictly this is *large-consistent-mechanism-explained*, not *noise-floor-validated*. But the effect (−$4.86,
unanimous 3/3, ~10× a $0.50 stand-in threshold, capability tied) is well beyond plausible per-build variance and
has a clean causal mechanism — **not** an underpowered tie. A same-arm replicate or higher n would make it
conclusive; it is not required to justify the next step.

Also observed (does not affect the arms — both share the identical recall): despite the stop-writes instruction,
2/3 recalls still performed a vault write (notes crystallized 1 note + 1 link-enrich into the **temp** vault copy;
the live vault was verified untouched). Noted for the production form, where recall's inline crystallization is
the thing being moved off the build path.

## What this does NOT do

It validates the lever + isolation premise. If GREEN, the **production mechanism** (subagent-isolated recall, or
a please/build synthesis-injection) is a **separate** brainstorm→plan→build — it touches recall's inline
crystallization and the please/build resume flow. **GREEN reached (above) → that production build is now
justified**, as its own effort — this smoke is not authorization to ship the proxy (1) inline.

## Spec self-review
- **Placeholders:** none — arms, the `recall_synthesis_prompt` fix, measures, tolerances, verdict-gate, and the
  results table are concrete.
- **Consistency:** (1) chosen as the smoke proxy; (2) deferred production; (3) rejected — and (1)≈(2) for cost is
  stated, with the proxy-gap named.
- **Scope:** single validation (one harness variant + verdict); production build explicitly deferred.
- **Ambiguity:** the verdict-gate names the $ margin (above noise), the capability tolerance (±1 round, success
  within noise), and the underpowered case — no false-win crowning.
