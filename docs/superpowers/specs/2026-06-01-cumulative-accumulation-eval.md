# Cumulative cross-app accumulation eval — design

Date: 2026-06-01

## The question
Does **accumulating** memory across apps reduce the cost (round-1 conformance, human
review rounds, $, turns, time) of reaching **feature-complete** on a *new* app — and does
it depend on **tier-isolated vs blended recall**? Prior runs answered "no clear benefit"
but were confounded (off-domain apps, tier-isolated only, learn on/off, auto-scorer bias).
This run pre-resolves every methodological caveat as a *design choice*.

## Why prior runs went flat: saturation
todo / bookmarks / contacts are **parallel** — they teach the *same* architecture, which
saturates after one app, and differ only in *off-domain features*. So stacking a 2nd app's
memory adds nothing that transfers. To make accumulation *able* to pay off, the apps must
be **cumulative**: app3 genuinely needs what app1 AND app2 each *uniquely* taught.

## The cumulative app trilogy (fixed order — each builds on the last)

Three CLI apps in Go. Shared **baseline architecture** (saturates at app1) + two stacked
capability layers:

- **app1 `notes`** — introduces **Layer α: tag + search subsystem.** Normalized (lowercased,
  deduped) tags; `tag`/`untag <id> <tag>`; `list --tag <t>`; `search <q>` = case-insensitive
  substring across **all** text fields.
- **app2 `links`** (bookmark mgr) — reuses α; introduces **Layer β: validation + dedup +
  portability subsystem.** Typed sentinel validation (`ErrInvalid`; input must parse as a
  URL); **dedup by a normalized natural key** (canonical key, not raw string); **import/export**
  (`export [--json] <file>` atomic; `import <file>` merges by key, last-write-wins);
  **lookup by natural key** (`get <key>` — a stable secondary index beyond numeric ID).
- **app3 `feeds`** (reading list — TARGET) — native: subscribe to a feed URL, mark items
  read/unread, `refresh` (timestamp stub), list unread-first. **Needs:** baseline arch +
  **α** (tag/search feeds) + **β** (validate feed URL, dedup by canonical URL, import/export
  subscription list, get-by-URL) + native.

### Why this makes accumulation measurable
`feeds`'s hidden spec is **bucketed**: ARCH (shared/saturating) · α (from app1) · β (from
app2) · NATIVE. The accumulation signal is **localized to the β bucket**:

| feeds built with… | ARCH | α (tags/search) | β (validate/dedup/import) | NATIVE |
|---|---|---|---|---|
| cold (no memory) | low | low | low | derive |
| +notes memory | lit (saturated) | **lit** | low | derive |
| +notes+links memory | lit | lit | **lit** ← accumulation signal | derive |

If β lights up **only** when app2's memory is present, accumulation is real. β items are
deliberately **non-default** decisions (canonical-key dedup w/ merge rule, atomic export,
import-merge-by-key) a cold builder won't reproduce by accident — same dynamic as cold
todo skipping atomic writes until told.

## Variables (the matrix)
- **Recall regime:** tier-isolated **L2** (strongest single tier) vs **blended** (all tiers).
  Expandable to L1-iso / L3-iso if desired.
- **Accumulation stage:** cold / +notes / +notes+links.
- **Target:** `feeds`. Priors `notes`, `links` built once (blended recall, converged,
  learn-all-tiers) to generate the vault. Order is fixed (cumulative), so no cyclic orders.

Core matrix = 2 regimes × 3 stages = **6 feeds convergence loops + 2 prior builds = 8 loops.**
(Full = 4 regimes × 3 stages + 2 = 14 loops.)

## Pre-resolved caveats (the point of this run)
- **Scorer bias** → a **frozen, name-agnostic, deterministic scorer**, not an LLM judge:
  arch items via structural pattern-detection over the code (any persistence interface +
  injection + sentinel + atomic + parallel-fake tests — detect the *pattern*, never a
  vocabulary token); feature items via **behavioral checks that run the built binary**
  (e.g. `export` then `import` actually round-trips & merges). Transparent and re-runnable.
- **Tier-isolated weakness** → now a measured **variable** (L2-iso vs blended), not a confound.
- **Learn on/off** → **always-learn** every stage, all tiers.
- **Converged ≠ 100%** → push to **feature-complete** (all buckets pass, not a ≥12/15 bar).
- **Saturation / off-domain** → **cumulative apps**; accumulation signal isolated to β.
- **Isolation** → headless `claude -p`, minimal cfgs, `/tmp` cwd (proven cold this session).

## Convergence harness (per cell)
1. Headless build (cold cfg, or warm cfg + `ENGRAM_VAULT_PATH` + recall regime).
2. Deterministic scorer → list of failed bucketed items.
3. If failures & rounds < cap (8): emit each failed item's **pre-written user-symptom
   phrasing** (gap-level, never the fix, never the spec) → resume builder → goto 2.
4. Converged when all buckets pass (or cap hit). Always-learn at the end.
5. Record per round: bucketed score, turns, $, time; per cell: round-1 conformance,
   rounds-to-feature-complete, human-equivalent review rounds, total $/turns/time.

## Remaining honest caveat
- **n.** Core matrix is n=1 per cell. Key cells (e.g. feeds +notes vs +notes+links, blended)
  can be repeated for n=2–3 if the β-bucket delta is within noise. Flagged, not hidden.

## Headline the run will produce
A bucketed conformance × cost table across {regime} × {stage}, with the **β-bucket delta
between +notes and +notes+links** as the accumulation verdict, and the **L2-iso vs blended**
gap as the recall-regime verdict — all on a deterministic name-agnostic scorer.
