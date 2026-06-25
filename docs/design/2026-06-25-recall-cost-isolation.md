# Isolating recall's cost — where do the ~350s and the tokens go?

**Date:** 2026-06-25 · **Status:** plan (Gate-A pending) · **Relates to:**
`2026-06-25-memory-cost-reanchor.md` (this is the "unbundle/decompose recall" step it called for).

**Question:** the warm op's ~350 s recall phase is the wall-time premium. *What inside recall* spends
that time and those tokens — so we know whether it's trimmable (the "faster" axis) and confirm recall's
token shape.

## Already measured (binary-side, direct — this env)

| recall sub-step | wall-time | tokens |
|---|---|---|
| Step 0.5 sweep (`engram ingest --auto`) | ~1.5 s | ~0 (subprocess) |
| Step 2 query (`engram query`, 10 phrases) | ~3.4 s | **payload ~49 K tokens** (197 KB) |
| → payload composition | — | **363 chunk items + 2 notes, 3 clusters** |

**The binary is ~5 s — NOT the time sink.** The payload the agent must *read* is ~49 K tokens,
overwhelmingly **raw chunks** (363:2 vs notes). That is recall's dominant *input-token* load, paid every
invocation. (Consistent with note 79: capping payload moved $ ~nothing — input tokens are cheap — but it
is still a large thing to read.)

## To measure (agent-side — the ~345 s remainder)

The ~350 s is the **agent**, not the binary: reading the 49 K payload + the multi-step procedure. Decompose
it per recall step from a real full-recall transcript (per-message **timestamps** → wall-time;
**`usage`** → input/output tokens; harness `token_usage_for_session` helpers parse this).

**Steps to attribute** (markers in the transcript: the `engram` tool calls + the skill's step language):

| Step | Activity | Expect |
|---|---|---|
| 0 + 1 | upfront plan + 10 phrases | small output |
| 2 | read the ~49 K query payload | large **input** tokens |
| 2.5 | per-cluster: `engram show` candidates (input) + coverage-judge (output) + writes (`amend`/`learn`) | suspected dominant slice (round-3) |
| 2.6 / 2.7 | cross-cluster link + `activate` | small |
| 3 | synthesis | output |

**Data source (pick the cheaper valid one):**
1. **Preferred — decompose an existing harness full-recall transcript** (the build-session JSONL whose
   `recall_s ≈ 350 s` was recorded in round-3). Free, and it's the *actual* 350 s. Check availability
   first (transcripts may be pruned).
2. **Fallback — one fresh isolated `/recall`** via `claude -p` over a **copied** vault + chunk index (no
   pollution of the real vault), capturing the JSONL. Representative of recall over the big chunk index;
   note it may be <350 s here (smaller vault → less crystallization) — the per-step **shape** is what we
   need, not the absolute.

## Output

A per-step **time + token** table for one real recall → the dominant slice(s) named, with the
trimmable-vs-inherent call for each (e.g. "reading 49 K of chunks = trimmable by returning fewer items;
coverage-judge reasoning = inherent unless steps are cut").

## Validity

- Real timestamps + `usage` from the JSONL; binary times measured directly (above).
- Isolate vault/chunks on any fresh run (no real-vault writes).
- **n = 1 is directional** — name the dominant slice, don't over-precision the seconds.
