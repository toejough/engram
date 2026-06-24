# Reducing the token & $ cost of using engram — 5 options

**Date:** 2026-06-24 · **Status:** brainstorm (options to evaluate, not decisions)

## Why this exists

The opus C1–C6 sweep measured engram's two cost axes as **negative**: a memory-warm run is
**slower and costlier** than a cold run, because the recall+learn cycle adds agent-side overhead.

| Axis | cold | warm (memory) | Δ |
|---|---|---|---|
| C1 wall time | 10 min | 21 min | **+11 min** |
| C2 tokens | 2.2 M | 6.8 M | **+4.6 M** |
| C2 $ | $3.30 | $7.89 | **+$4.59** |

Memory's *value* (C3/C4/C5/C6) is real and large on idiosyncratic content — but it is bought with
tokens. These options attack the **cost** side without surrendering that value.

> The cost is **agent tokens**, not the binary. `engram query`/`ingest` are local, zero-LLM, and
> effectively free. Every dollar above is the LLM reading payloads and executing skill steps.

## Measured cost drivers (from a live recall this session)

A single `/recall` on a normal ask produced:

- **A 5 965-line query payload — 383 items carried with full content** (`--limit 20`, but cluster
  *members* carry full chunk text regardless of the limit). The agent reads all of it.
- **A second full-content source: the recency channel** appends up to **200 newest chunks** with full
  text (`recentFillChunks = 200`), in `items[]` but NOT cluster members — a large slice of those 383.
- **`engram show` fan-out** — Step 2.5 issues one blocking `engram show` per candidate note *not
  already surfaced in `items[]`* (0 on this recall — chunk-only clusters). The skill already skips
  `show` for surfaced members, so the residual fan-out is modest, but non-zero on note-rich recalls.
- **A fixed 10-phrase query** → top-30 matches per phrase, unioned to ~300 matched items.
- **The recall `SKILL.md` (285 lines) reloaded on every invocation.**
- **The full cycle runs even when memory is silent** — this very recall surfaced 0 note-clusters
  and crystallized 0 lessons, yet ran the entire 7-step procedure.

---

## The 5 options

Each: the mechanism, the driver it attacks, expected savings on **both** axes, risk, effort, rating.
"Expected savings" are estimates to be validated, not measured results.

### Option 1 — Cap full-content in the query payload (content budget)  · **CONTENDER**

**Mechanism.** `engram query` returns full content only for each cluster's *representative* + the
top-`--limit` direct hits; all other members **and the ~200 recency-channel chunks** return path +
score + a one-line snippet. Add a `--content-budget` (chars/items) the agent can dial.

- **Driver:** the 383-full-item / 5 965-line payload (cluster members **+ the 200-chunk recency
  channel**) — the largest single sink. The mechanism must trim both, not just cluster members.
- **Tokens:** large win — a recall payload could drop from ~40–60 K tokens to ~5–10 K (~75–85%).
- **$ / time:** proportional; highest-leverage because it hits *every* recall.
- **Risk:** low–medium. The agent loses the ability to read every member inline; it must
  `engram show` a member it wants in full (rare — representatives usually suffice). Validate that
  coverage/synthesis quality (C3/C6) holds with snippet-only members.
- **Effort:** medium (Go change in the query payload builder + a skill note).

### Option 2 — Inline candidate-note content; delete the `engram show` round-trips  · **CONTENDER**

**Mechanism.** `engram query` already knows each `candidate_l2`'s content — emit it inline in the
payload so Step 2.5 needs **zero** `engram show` calls. Pairs naturally with Option 1 (representatives
inline, members as snippets).

- **Driver:** the per-candidate blocking `engram show` fan-out (K calls/recall on note-rich asks).
- **Tokens:** small–medium (removes redundant call framing + lets the agent skip re-reading).
- **$ / time:** removes K sequential round-trips per recall → tangible wall-time win on note-rich
  recalls; modest token win.
- **Risk:** low — it's the same content, delivered once instead of fetched N times.
- **Effort:** low–medium (Go: attach candidate content to clusters).

### Option 3 — Run recall/learn on a cheaper model than the main task  · **CONTENDER**

**Mechanism.** Execute the recall and learn skill steps (read payload, judge coverage, write notes)
on haiku/sonnet via a delegated subagent, returning only the Step-3 synthesis to the opus main loop.
Recall is mechanical curation; spending opus tokens to read 383 items is the waste.

- **Driver:** opus reading large payloads — the dominant **$** term (haiku ≈ 1/10–1/15 opus $/token).
- **$:** large win — the bulk of recall tokens move to a ~10× cheaper model.
- **Tokens:** unchanged in count (possibly higher), but **$** drops sharply — this is a $-axis play,
  not a token-axis one. Report both; do not conflate.
- **Risk:** medium — coverage/recency judgments (Step 2.5-B/C) on a weaker model may degrade note
  quality. **Validate where it bites:** the C4 supersession + C5 recency tasks, where a weak judge
  would mis-rank — not a quality-blind metric.
- **Effort:** medium (orchestration: recall-as-subagent with a model override).

### Option 4 — Reduce phrase width (10 → agent-chosen 3–5)  · **PARK**

**Mechanism.** Let recall emit 3–5 task-shaped phrases instead of a fixed 10, shrinking the matched
union and everything downstream.

- **Driver:** the 10-phrase fan → ~300-item union.
- **Tokens / $ / time:** rough ~30–50% fewer matched items (3–5 vs 10 phrases) *before* O1, but
  **dominated by Option 1** — capping content makes payload size far less sensitive to phrase count,
  shrinking this to a marginal few-K tokens once O1 is deployed (so O4 is re-evaluated *after* O1).
- **Risk:** medium — fewer phrases = lower recall; C5 recency depends on a topically-distant phrase
  catching R. Could silently drop the very hits memory exists to surface.
- **Effort:** low (skill edit) — but **only test after O1**, since O1 removes most of its upside.

### Option 5 — Conditional recall: a cheap gate before the full cycle  · **PARK**

**Mechanism.** A lightweight pre-check (one cheap query; inspect top cosine + whether any note
matches) decides whether to run the full clustering+show+synthesis procedure. Below a floor, or on a
trivially-scoped ask, skip to a one-line "memory silent" and proceed.

- **Driver:** the full 7-step cycle running when memory is silent (e.g. this session's recall).
- **Tokens / $ / time:** on a *silent* recall, near-total (skip the whole ~40–60 K payload read);
  zero benefit when memory is rich. Net savings scale with how often recalls come back silent.
- **Risk:** higher — a mis-tuned gate skips recall *exactly* when a subtle-but-present memory would
  have helped (the failure mode the recall skill's red-flags table was built to prevent). Cuts
  against the "always recall" discipline; needs a conservative floor + measurement of false skips.
- **Effort:** medium (skill + a gate heuristic; careful calibration).

---

## Ratings & sequence

| # | Option | $ axis | token axis | risk | effort | rating |
|---|--------|--------|-----------|------|--------|--------|
| 1 | Cap payload content | ↓↓ | ↓↓↓ | low–med | med | **CONTENDER** |
| 2 | Inline notes, drop `engram show` | ↓ | ↓ | low | low–med | **CONTENDER** |
| 3 | Cheaper model for recall/learn | ↓↓↓ | ~ | med | med | **CONTENDER** |
| 4 | Fewer phrases | ↓ | ↓ | med | low | PARK (after O1) |
| 5 | Conditional recall gate | ↓ | ↓ | higher | med | PARK |

*Arrows = expected magnitude, unvalidated: ↓ small · ↓↓ medium · ↓↓↓ large. `~` = no change on that axis.*

**Recommendation (next step, not part of the brainstorm):** start with **O1 + O2** (one coherent payload change — biggest
token win, lowest risk), then **O3** for the **$** axis (orthogonal — moves the same work to a
cheaper model). O4/O5 are parked: O4 is largely subsumed by O1; O5 trades the most cost-cutting
potential on silent recalls against the highest risk of skipping useful memory.

## How to validate (next step, not part of the brainstorm — non-negotiable when building any option)

- Report **both axes** every time ($ and tokens, plus wall time) — never collapse to one number.
  Setup/seed cost is separated from per-recall payback.
- For any "cheaper variant is just as good" claim (O3, O4, O5), **test where the no-cost baseline
  fails** — the C4 supersession and C5 recency tasks — not a quality-blind metric. A tie on a metric
  blind to the cut means "can't distinguish," not "the cut was free."
- A/B each option against the current recall on the existing C3/C4/C5/C6 harnesses: cost must drop
  **and** the criterion must hold.
