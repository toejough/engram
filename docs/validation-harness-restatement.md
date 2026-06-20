# Validation Harness — Restatement & Interrogation

*Working document. Restates the ask, maps it against the harness that already exists, and surfaces the inconsistencies/gaps to resolve before any plan or run.*

## 1. What I believe you want

A validation harness that gives **fast, cheap, high-confidence** evidence about whether the engram memory system actually delivers value, tested against **the real shipped skills + binary** (not a stand-in), across **Anthropic models**, with **n=5** trials, and **grouped so each run teaches the most per dollar**. Concretely, six capability claims to measure:

| # | Claim | Plain meaning |
|---|-------|---------------|
| C1 | **Faster** | Memory reduces time/turns to a correct result |
| C2 | **Cheaper** | Memory reduces tokens / $ per task |
| C3 | **Fewer human interactions** | Human restates fewer conventions / intervenes less |
| C4 | **Learns from changes over time** | When a standard changes, later runs adopt the new standard |
| C5 | **Remembers recent history** | Recent work is surfaced and reused |
| C6 | **Compounding web of lessons not directly taught** | Lessons transfer/compose beyond what was explicitly taught |

Cross-cutting requirements: real artifacts only; model comparison; n=5 confidence; cost-efficient grouping; learn as much as possible as cheaply as possible.

## 2. Critical context you may not have: a harness already exists

`dev/eval/cumulative/` is a substantial, already-built cumulative harness. It runs a 3-app accumulation chain (`notes → links → feeds`), cold-vs-warm arms, **real-skill arms that invoke the actual `/recall` + `/learn` skills and the real `engram` binary end-to-end** (`real.lazy/eager/auto/autol2`), sweeps **haiku/sonnet/opus**, at **n=5**, with a **name-agnostic structural scorer** (detects patterns, not vocabulary — already guards against the scorer-bias failure mode). It already produces cost/token tables.

**So the realistic ask is "extend + modernize + re-validate," not "build from scratch."** This changes everything downstream.

## 3. Where your six claims stand against the existing harness

> **Stale-results rule (per your correction).** The existing harness's *prior runs* were produced on the **pre-`recall-v2` architecture** and are **void** for deciding the current system's behavior. We re-validate **all six axes fresh** against the shipping artifacts. The table below states only (a) what the harness *infrastructure* can measure today and (b) what instrumentation is *missing* — **not** any prior result, which must not anchor next steps.

| Claim | Infra status (not results) | What's needed |
|-------|----------------------------|---------------|
| C1 Faster | Captures turns/wall-clock per op | Re-run fresh; report with CI |
| C2 Cheaper | Captures tokens → cost per op | Re-run fresh; report with CI |
| C3 Fewer interactions | Has the intervention metric (convention restatements) | Re-run fresh on real-skill arms |
| C4 Standards-change-over-time | **No mechanism.** Single-SHA snapshot; no standard reversed mid-chain. | New variant: reverse a taught standard between apps |
| C5 Recent history | Ingests + counts chunks, but no recency-surfacing score | New probe: was a recent lesson preferentially surfaced/used? |
| C6 Compounding / not-taught | Only a keyword-substring proxy + an L2→L2 link count | Real semantic/transfer judge |

**The frontier:** C4, C5, C6 have **no real measurement** yet — these are where the design effort goes. C1/C2/C3 have infrastructure but need a clean fresh run on the shipping architecture. Every number we report will be freshly produced; no prior finding is carried forward.

## 4. Validity issues in the existing harness (worth fixing regardless)

- **Legacy arms test a dead architecture.** The 7 `l1/l2/l3` regimes encode the pre-`recall-v2` tiered model (L1 episodes / L3 ADRs) that **no longer ships** (recall-v2 phase 6 removed the L1/L3 surface, collapsed to notes+chunks). They also use **closed-loop scaffolding** (the harness feeds the agent the convention labels it should "learn"), which confounds agent autonomy. → Retire them; make the real-skill, chunk-based arms the spine.
- **Learn-capture scoring is a keyword proxy, not semantic** (`harness.py:360`). C6 needs a real semantic/transfer judge.
- **No variance/significance reporting.** n=5 means are printed with no CI/error bars. Small effects (C1/C2) can sit below the noise floor and read as "tie" when they're really "underpowered."
- **Must run isolated.** Eval runs must point at an isolated `ENGRAM_CHUNKS_DIR` + `ENGRAM_VAULT_PATH` (now reinforced by the just-shipped ingest-hygiene prevention) so they never read from or pollute your prod vault.

## 5. Design principles I'm carrying in (from crystallized lessons)

- Run the **component under test** for real (agent invokes the skill; never hand-inline `engram query`).
- **Fail loud** on missing eval inputs; never silently fall back to a strawman baseline.
- **Size noise from the same contrast** (warm-vs-warm) and call a sub-noise gap *underpowered*, not a tie.
- **Test where the no-memory baseline fails**, not just average outcomes — a tie on a memory-blind task ≠ "memory optional."
- **No spend cap mid-run**; estimate + confirm cost up front, then let it finish.
- **Calibrate time/decay defaults to your real cadence** for any "over time" test.

## 6. Grouping for cheap learning (my current thinking)

One accumulation chain can yield **multiple** capability signals at once if instrumented well: a single `notes→links→feeds` warm run already exposes C3 (interventions), C2 (tokens/cost), and — with added probes — C5 (was a recent lesson surfaced?) and C6 (did a never-re-taught convention transfer to app3?). C4 needs a *dedicated* variant (reverse a standard between apps and watch for adoption). So: instrument the existing chain to emit 4 of 6 metrics per run, add one C4 variant, and reserve model×trial expansion for the axes that actually move.

## 7. Open questions (interrogation)

See the 4 questions posed alongside this document. Beyond those, smaller gaps I'll assume unless you say otherwise:
- **Model role:** vary the *agent-under-test* model (haiku/sonnet/opus); keep the model that internally runs `/recall`/`/learn` matched to it. (Add Fable-5? Vary them independently? — costs more.)
- **n=5 + CI:** keep n=5 but add variance/CI reporting and flag any effect below the warm-vs-warm noise floor as underpowered rather than concluding.
- **Cost:** I will compute the exact matrix cost and confirm with you before any paid run.
