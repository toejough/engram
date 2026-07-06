# Baseline pressure test — autonomous learn: capture self-discovered kinds too

Reusable RED/GREEN scenario input for autonomous (no-user-prompt) self-fire. Autonomous fire is NOT
narrowed to user-attributed kinds — all four Step-2 kinds apply, including self-discovered ones —
kind 3 reversals and kind 4b self-validated bets — which need no user turn. A plain discovered fact
(matching no kind) and one-off trivial fixes stay unwritten, left to the chunk index.

## Scenario

A coding agent finished a cycle of autonomous work (grinding an implementation plan), all tests
green, just committed. NO user prompt — the learn skill self-fires. The session contained, in order:

1. **User correction** (kind 1) — earlier the user said "don't hand-roll the kerning lookup — call
   `glyphd.KernPair()` like the rest of the pipeline."
2. **Self-discovered reversal** (kind 3) — the agent had committed "cache every glyph in one
   in-process LRU map" to its plan, discovered while implementing that the CJK working set thrashed
   it, and switched to a two-tier slab allocator that held. Nobody said the first approach was wrong.
3. **Self-validated bet** (kind 4b) — unsure whether to shard rasterization by font-family or
   glyph-range, the agent bet on glyph-range, implemented it, and the throughput benchmark then
   passed at the 4000 glyphs/sec target.
4. **Plain self-discovered fact** (no kind) — "the `glyphd build` command exits 0 even when a
   subfont fails to embed; grep stderr for `WARN embed`." Neither a reversal nor a bet.
5. **Typo fix** in a comment ("recieve" → "receive").

## Expected current-skill behavior (GREEN)

- **Step 1 — sweep:** `engram ingest --auto` runs first.
- **Step 2 — scan all four kinds (autonomous fire is not narrowed):**
  - Item 1 (user correction) → WRITE `feedback` (kind 1).
  - Item 2 (self-discovered reversal) → WRITE `feedback` (kind 3). Nobody SAID the correction —
    self-discovered reversals qualify; autonomous fire captures them.
  - Item 3 (self-validated bet) → WRITE `feedback` (kind 4b). A bet acted on and confirmed by an
    observable outcome (the benchmark passing); autonomous fire captures it.
  - Item 4 (plain discovered fact) → DO NOT WRITE. It matches none of the four kinds — not a
    reversal, not a bet — so it stays in the chunk index (Step 1). The skip is "no kind matches,"
    NOT the stale "autonomous mode only does corrections" rationale that #673 removed.
  - Item 5 (typo fix) → DO NOT WRITE. One-off, not a behavior lesson.
- **Result:** 3 writes (items 1, 2, 3), 0 for the plain fact, 0 for the typo. No `engram transcript`
  calls, no episode notes, no session summarizing.

## RED (pre-kind-3 skill) vs GREEN (current skill)

- **RED:** the pre-kind-3 two-kind skill (corrections + save-requests only) writes ONLY item 1;
  items 2 & 3 fall under no kind the pre-kind-3 skill recognizes, so a self-discovered reversal and
  a self-validated bet fired autonomously EVAPORATE. (Measured 2026-07-06, sonnet, 5 headless
  fresh-process reps: item 1 written 5/5; items 2 & 3 written 0/5.)
- **GREEN:** the current four-kind skill writes items 1, 2, 3 and skips items 4 & 5. (Measured
  2026-07-06, same 5-rep run: items 1+2+3 written 5/5; items 4 & 5 written 0/5.)

Items 2 & 3 are the discriminating cases: they prove autonomous fire captures the self-discovered
kinds (option a, #673) and would catch a regression that re-narrowed autonomous mode to two kinds.

## Failure modes that must FAIL this test

- NOT writing item 2 (self-discovered reversal) or item 3 (self-validated bet) — the whole point of
  option (a): autonomous fire is not narrowed to user-attributed kinds.
- Writing item 4 (plain fact) or item 5 (typo) — over-capture; neither is one of the four kinds.
- Skipping item 1 (the genuine correction).
- Running `engram transcript`, writing episode notes, or applying removed three-gate logic.
