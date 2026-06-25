# Isolating recall's cost — where do the time and tokens go?

**Date:** 2026-06-25 · **Status:** plan v2 (revised per Gate A — a load-bearing premise was wrong) ·
**Relates to:** `2026-06-25-memory-cost-reanchor.md` (this corrects the recall-cost number that doc relied on).

## Gate-A finding that reframes everything: "recall ≈ 350 s" was never recall

The harness field `recall_s` (~350 s) does **not** measure recall. `harness.py:673-675` brackets
`res = do_build(prompt)` — and that prompt (`harness.py:145,159`) is *"Before writing ANY code, invoke
/recall … then build the app and make tests pass."* So **`recall_s` = round 1 = /recall + the entire first
code-build**; `build_s` is rounds 2+. **Recall's standalone cost has never been measured** — the "~350 s
recall" in the round-3 findings and the re-anchor is a mislabel. Consequences:

- The re-anchor's "recall's 350 s alone exceeds the cold build (288 s) ⇒ faster is structurally
  impossible" is **unfounded** — recall is some *fraction* of 350 s, likely well under the cold build.
- The warm-vs-cold phase accounting (recall 350 / build 204 / learn 61) is **not clean** — `recall_s`
  and `build_s` are round-1 vs rounds-2+, not recall vs build. The whole "overhead swamps the build gain"
  arithmetic needs redoing once recall is isolated.

**This measurement's real job:** produce the *first* recall-only cost, and re-state warm-vs-cold on
comparable terms.

## Already measured (binary-side, direct — but report as a RANGE)

The `engram query` binary is fast and its payload is phrase-dependent (Gate A: it does not reproduce as a
constant):

| sub-step | wall-time | payload |
|---|---|---|
| `engram ingest --auto` (sweep) | ~1.5 s | — |
| `engram query` (varies by phrases) | ~3–4 s | **~125–244 KB ≈ 31–61 K tokens; ~250–440 chunk items, ~1–2 notes** |

So the binary is **~5 s — not the time sink**, and the payload is a large, chunk-dominated *input-token*
load (tens of K), but **a range, not a fixed 49 K** (byte→token ≈ ÷4; vault is growing). Tie any single
figure to the *actual phrases a measured recall used*, recoverable from its transcript.

## To measure — recall-ONLY, per step

**Primary data source (recall-only by construction):** one fresh isolated `/recall` run via `claude -p`
over a **copied** vault + chunk index (no real-vault writes), prompted to run recall for a representative
task and **stop after Step 3** (no build). Its JSONL is recall and nothing else.

*(Alternative if reusing a build transcript: recall = session start → the **last `engram` tool call before
the first code-write** (Edit/Write); everything after is build, not recall. Do NOT attribute post-code
messages to "Step 3 synthesis." — Gate A R5.)*

**Segmentation is NEW code, not a reused helper.** `harness.token_usage_for_session` only *sums* a whole
session (no step-awareness — Gate A R2). Write a small parser: walk the JSONL messages in order; bucket
each message's `message.usage` (input/output/cache) and `timestamp`-delta into a step window keyed by the
`engram` tool-call boundaries + skill step language. Markers verified present in real transcripts
(`timestamp` + `message.usage` on most lines; `engram query`/`show`/`activate` calls visible).

| Step window | Boundary marker | Expect |
|---|---|---|
| 0 + 1 | session start → first `engram query` | small output (plan + 10 phrases) |
| 2 | the `engram query` call → next engram call | **large input** (read the payload) |
| 2.5 | `engram show`/`amend`/`learn`/`activate` span | per-cluster judging (in+out) + writes |
| 3 | last engram call → end (or first code-write) | synthesis output |

## Output

A per-step **time + token** table for one recall-only run → the dominant slice named (trimmable vs
inherent), **and** a corrected warm-vs-cold statement using recall-only (not `recall_s`).

## Validity

- Real `timestamp` + `message.usage` from the JSONL; binary times measured directly.
- Isolate vault/chunks on the fresh run (no real-vault writes).
- **n = 1 is directional** — name the dominant slice; don't over-precision the seconds.
- **Keep the axes separate (note 84):** decomposing recall *time* does not auto-deliver a *dollar* cut —
  recall is time-heavy but dollar-light; call out any per-step finding that crosses axes.
- This measurement may **revise the re-anchor's conclusion** (if recall-only ≪ 350 s, "faster" is no
  longer structurally blocked). Update the re-anchor + notes 91 accordingly once measured.

## Results (measured 2026-06-25)

Two recall-only decompositions (n=2: one fresh isolated `claude -p` run, one clean in-session sub-slice),
both **~190 s** — directional, in agreement. Real vault untouched (isolated copy; git clean).

**Recall-only TOTAL ≈ 190 s — NOT 350 s.** The `recall_s` field (~350 s) was round-1 = recall **+ the
first full code build**; recall is ~**55%** of it. Crucially, **recall (~190 s) is *below* the cold build
(~288 s)** — so the re-anchor's "recall's 350 s exceeds the cold build ⇒ faster is structurally
impossible" is **false**; faster is not structurally blocked.

| recall step | wall-time | % of recall | the work |
|---|---|---|---|
| 0+1 — plan + 10 phrases | ~32 s | ~17% | cold-session system-prompt inflated; ~0 warm |
| **2 — query + read/page payload** | **~82–118 s** | **~43–63%** | **dominant.** binary `engram query` ~3.5 s; the rest is the agent **paging a ~200 KB payload** (too big for one read → re-queries/pages ~8×) |
| 2.5 — per-cluster judge + writes | ~17–63 s | ~9–33% | coverage-judge reasoning + `engram show`/writes |
| 2.7/3 — activate + synthesis | ~15–17 s | ~8% | small |

**Binary `engram query` (direct, 3 phrasings):** ~2–3.5 s; payload **~141–237 KB ≈ 35–59 K tokens,
~370–410 chunks + 1–9 notes**. Fast binary, large chunk-dominated payload.

**Tokens / $:** billed ~$1.84 (cold `claude -p`, pays full system-prompt once). The bulk is `cache_read`
(~1.35 M, billed 0.1×) — i.e. the payload. Recall is **time-heavy, dollar-light** (confirms note 84 /
the re-anchor's corrected $-asymmetry).

### The dominant, trimmable slice

**Step 2 (read the query payload) is ~50% of recall, and it is partly trimmable:** the binary is 3.5 s;
the ~80–115 s is the agent wrestling a 200 KB payload that won't fit one tool read (→ ~8 re-queries/pages).
**Capping the payload / returning a clusters-first or pre-sliced view so it's consumable in one read would
cut Step 2 substantially.** This is a **TIME** cut (note 84: the payload is mostly cheap `cache_read`, so
it barely moves $ — consistent with note 79's "payload cap moved $ ~nothing"; the cost it moves is *time*,
not dollars). The query-phrasing/clustering itself is inherent (you must read what surfaced).

**Caveats:** n=2, directional — don't over-precision the seconds; both runs agree on ~190 s and on Step 2
as dominant. Step 0+1's tokens are cold-session-inflated (full system-prompt on turn 1); in a warm session
that's ~0. Parser + artifacts in the session scratchpad (`decompose.py`).
