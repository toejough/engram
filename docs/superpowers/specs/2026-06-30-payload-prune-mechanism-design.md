# Payload-prune mechanism — design + smoke validation

**Ask (Joe, 2026-06-30):** brainstorm the payload-prune *mechanism* (how to drop the carried recall payload from
the build context after Step 3, keeping only the synthesis), pick an option, and validate it with quick smoke
tests. This is **brainstorm + smoke-validate**, NOT the production build.

## Problem (verified, notes 100/95)

recall's `engram query` payload (~97 KB) is **carried** in the build context: the `$METER` harness's Phase-2
build **`--resume`s the recall session** (`dev/eval/cumulative/harness.py:716`), so every build round re-sends the
payload as `cache_read` → the **~$1/op warm-over-cold premium** (note 95: "the resumed recall context is re-read
as cache every build turn"). The payload *bytes* are cheap once (note 100 — payload-size caps move *time*, not
dollars); **carrying** them across N build rounds is the dollar cost. The *value* — Step 3's "apply these as
requirements" synthesis — is a few hundred tokens.

## The lever, and the honest challenge

payload-prune = after Step 3, the build carries **only the synthesis**, not the raw payload. It is the **only
verified $ lever** (note 100; everything else — payload-size caps, whole-op/split haiku, cutting phrases — was
refuted). **Honest challenge (notes 77/91):** recall is dollar-light; the ~$1 is *small* vs the $2–4 build, and
dropping the payload could **cost rebuild rounds** if the synthesis under-captures (note 107 — a payload cut is
not a net win unless you measure the downstream behavior). So the smoke must show the $1 is **real AND
capability-safe**, or it dies cheaply.

## Mechanisms considered

| # | Mechanism | Role | Notes |
|---|---|---|---|
| 1 | **Synthesis-injection** — build starts in a *fresh* session with the Step-3 synthesis injected; no resume of the payload session | eval/harness form + simplest production form | the payload never enters the build context |
| 2 | **Subagent-isolated recall** — recall runs in a subagent, returns only the synthesis to the parent build | production form (route-aligned) | shifts recall off its current inline design; same payload isolation as (1) |
| 3 | **Context-edit / compaction of the resumed session** — strip the payload tool-output before build rounds | **REJECTED** | Claude Code has no targeted tool-output removal; auto-compaction is untargeted — uncertain/unbuildable |

For **cost**, (1) ≈ (2): both isolate the payload and carry only the synthesis, so the smoke measures what either
production form would deliver.

## Chosen approach

**Validate the lever via mechanism (1)** — the synthesis-injection harness variant — the cheapest de-risk that
*also* measures the production payoff. Build the production subagent (2) **only if** the smoke validates.
Rejected (3) as an uncertain mechanism. (Measure-before-build, per the #661→#663 discipline.)

## Smoke test (measure-first; the de-risk)

A variant of `dev/eval/cumulative/harness.py` Phase 2, run on a small set of apps (smoke n; reuse the existing
`recall_cost`/`build_cost` `$METER` split, schema v5, note 102):

- **Arm A — Carried (current):** Phase 2 `do_build(build_msg, resume_sid=recall_sid)` — resumes the recall
  session (payload carried across all build rounds).
- **Arm B — Pruned:** extract the recall call's Step-3 synthesis text from `recall_res`; Phase 2
  `do_build(synthesis_text + build_msg, resume_sid=None)` — a **fresh** build session carrying only the synthesis.

**Measure both axes (the challenge demands it):**
1. **$ (the premise):** `build_cost(B)` vs `build_cost(A)` — is the ~$1 real *and recoverable* by pruning?
2. **Capability (note 107):** build **rounds-to-converge** + success rate — does synthesis-only hold, or does
   dropping the payload force rebuild rounds (which would negate the saving and/or fail the build)?

**Verdict-gate:** a **net win iff** `build_cost(B) < build_cost(A)` by a margin **above the noise floor**
(size noise from a same-arm contrast — note 96) **AND** capability(B) ≈ capability(A) (no extra rounds, no extra
failures). If B saves $ but costs rounds → **not a win** (synthesis under-captures). If the $ delta is below
noise → **underpowered, not a tie** (note 96) — report as "can't distinguish at this n", don't crown it.

**Honest bounds to report:** smoke n is small; the warm-vs-cold $1 (note 95) was on *easy* builds where memory is
net-negative anyway — so the smoke measures the *recoverable* slice of the premium on this harness/model, not a
universal figure. Spend: a few `$METER` cells (~$ per cell × A/B × apps) — estimate + report actual.

## What this does NOT do

This validates the lever + the isolation premise. If the smoke is GREEN, the **production mechanism**
(subagent-isolated recall, or a please/build-flow synthesis-injection) is a **separate** brainstorm→plan→build —
because shipping it touches recall's inline architecture (the crystallization happens inline today) and the
please/build resume flow.

## Spec self-review
- **Placeholders:** none — the smoke arms, measures, and verdict-gate are concrete.
- **Consistency:** the chosen mechanism (1) and the smoke arm B are the same isolation; (2) is the deferred
  production form; (3) rejected with a reason.
- **Scope:** single focused validation (one harness variant + a verdict). Production build is explicitly deferred.
- **Ambiguity:** the verdict-gate names both the $ margin (above noise) and the capability condition, and the
  underpowered case — no "tie" crowning.
