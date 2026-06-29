# Real-vault glance-vs-deep recall cost — de-risk for #662

> **Decision (Joe, 2026-06-29).** Before building the depth dial (#662), measure the one thing #661 couldn't:
> the glance-vs-deep recall **wall-time** saving on a **real-scale** vault. The glance/deep split's only payoff
> is per-fire recall cost/latency; #661 measured **1.2× (≈ within rep noise)** but **on a 5-note vault where
> deep recall is already cheap** — so that number can't answer the real-vault question. This measurement gates
> the *glance-mode split*. (It does **not** gate the cost-independent work — see Decision rule.)

## Why #661's Phase-4 number can't answer this

#661 timed recall on a **5-note, empty-chunk** vault → deep recall did almost no work (~35s) → glance saved
~nothing (1.2×, and one glance rep was *slower* than deep — within noise). The deep tax is **not** a binary cost
(the `engram query` binary is ~5s even on the real index) and **not** mere payload bytes — it is **agent-side**:
on a large index Step 2.5 has the agent run `engram show-chunk`/`engram show` round-trips over **every chunk
member + candidate note, per cluster**, then crystallize-writes (2.5C/2.6/Step-4). On the live index that's
~**228 chunk items across ~10 clusters** (measured) — the iterative paging + reasoning + write loop that is the
~190s tax (notes 91/93; "the **procedure** is the tax, not payload"). A 5-note vault has ~0 of this. So the
saving must be measured where the loop is real.

## Method

1. **Per-rep fresh isolated copy** of the live vault + chunk index: `cp -R $HOME/.local/share/engram/vault` and
   `…/engram/chunks` (incl. `manifest.json`) to temp dirs **— a fresh pair per rep** (deep's rep-1
   crystallization must not bias rep-2 retrieval; `--auto` mutates the copy). `cp -R` is ~1.3s total (measured).
   **Copy only while no background ingest is running** (the live chunks dir churns from hooks → torn-manifest
   risk). Redirect recall via `ENGRAM_VAULT_PATH` + `ENGRAM_CHUNKS_DIR` (the binary resolves these;
   `targets.go:131-132`), and keep `ENGRAM_TRANSCRIPT_DIR` pointed at the isolated cfg dir so Step-0.5
   `engram ingest --auto` sweeps only the tiny cfg transcript (seconds), never the live sources. **Assert both
   env vars are set+non-empty before each launch** — else recall falls back to the **live** vault and pollutes
   it.
2. **Recall-only timing**, glance cfg vs deep cfg (the validated #661 cfgs — persist them from session scratchpad
   into `dev/eval/` first, they are ephemeral). A recall-only prompt (invoke `/recall`, report what surfaced,
   write **no** code). **N ≥ 5 reps** each. Adapt the #661 `phase4_cost.py` (also persisted): swap its
   `seed_c3`/empty-chunks for the per-rep live copies.
3. **Instrument per arm** (the payload-floor guard — without this a thin-payload run repeats #661's error): for
   each run record **wall-time (s)**, **billed $**, **matched-set size**, **chunk-item count**, **cluster
   count**, **turns**. Measure wall-time AND $ as separate axes (note 84). $ is mostly cache-cheap on reads
   (note 100) so **wall-time is the likely value** — but deep's crystallization emits **output tokens** (≫
   cache-read price), so the $ gap may grow at scale; do not pre-judge $ negligible.

## Payload floor (validity gate)

A **"don't build"** verdict is only trustworthy if **deep actually reproduced real scale** on the chosen task —
target ≈ the measured live figures: matched-set ~300, **~228 chunk items, ~10 clusters**, deep wall ≈ the ~190s
real tax. Choose a task that maximizes payload (a Go build → dense conventions + many chunk clusters) and
**frame it as the dial's densest / near-best case**: then a *marginal* result generalizes *a fortiori* (if even
the densest payload barely saves, lighter ones save less), while a *material* result at ≥2× is a valid existence
proof. Report the instrumented floor next to the verdict.

## Decision rule (single partition, noise-floored, note-140-mapped)

The wall-time **ratio (deep/glance) is literally the firing-ceiling relaxation factor** (note 140: glance relaxes
the over-fire ceiling *proportionally* to its lower per-fire cost; `tax = over-fire × per-fire-cost`, note 109).
Let Δ = deep_wall − glance_wall, and `spread` = the larger within-arm rep range (note 96 — a gap below the rep
spread is *"can't distinguish,"* not a tie).

- **BUILD the glance/deep split** iff **Δ > spread AND ratio ≥ ~2×** AND deep hit the payload floor — the dial
  buys a real ≥2× firing-headroom on a real vault.
- **DON'T build the split** iff Δ ≤ spread, or ratio ≪ 2× with the floor met — the per-fire saving is
  immaterial; the glance-mode complexity isn't worth it.
- **The win measured here is per-fire recall cost / firing-frequency headroom — NOT end-to-end op cost** (notes
  77/95: the build loop dominates end-to-end $; this gate does not claim an end-to-end win).

**Cost-INDEPENDENT carve-out (proceeds regardless of the verdict):** #662 also owns (a) the **C5 recency-apply
fix** — a *quality* bug #661 found even under deep (honored 4/5), not gated by cost — and (b) **#657's O2/L2
safe cuts**, which speed deep recall unconditionally. "Don't build the split" parks the *glance mode*, not these.
If BUILD: the glance-mode ship must also carry **C5-type recency cues → deep routing** (#661: glance fails C5).

## Output (pre-committed table)

| arm | wall-time (s, mean ± range) | $ (mean) | matched-set | chunk-items | clusters | turns |
|---|---|---|---|---|---|---|
| deep | … | … | … | … | … | … |
| glance | … | … | … | … | … | … |
| **Δ / ratio** | … | … | — | — | — | — |

Verdict line: BUILD / DON'T-build the split + the relaxation factor + whether the payload floor was met.

## Bounds

- One **densest-case** task; N≥5 controls rep noise, not task-to-task payload variance (the densest-case framing
  makes a marginal verdict generalize). Delivery is **not** re-measured (#661 settled: glance delivers
  C3/C4i/C6, fails C5).
