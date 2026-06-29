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

---

## Result (2026-06-29) — BUILD the glance/deep split

Recall-only, glance cfg vs deep cfg, **per-rep fresh copy of the live vault** (~120 notes / ~3.4k chunks),
N=5, densest-case Go task. **Payload floor met:** the copy's 10-phrase query returned **167 chunk items /
~120KB** (real scale — deep did the agent-side show-chunk/crystallize loop, 8–12 turns).
Data: `…-661-data/realvault_cost_results.json` + `realvault_cost.py`.

| arm | wall-time (s, mean[range]) | $ (mean) | turns |
|---|---|---|---|
| **deep** | 94 [82–130] | 0.78 | 10 |
| **glance** | 42 [35–61] | 0.42 | 7 |
| **Δ / ratio** | **−52s / 2.23×** | **−46%** | −3 |

**Verdict: BUILD.** Glance is **2.23× faster** and **~46% cheaper** per fire on a real vault — and the
**wall-time** distributions are non-overlapping (every glance run < every deep run), a firmer signal than the barely-
cleared Δ(52s) > spread(49s) noise check (note 96). The wall-time ratio IS the firing-ceiling relaxation factor
(note 140): the dial buys ~2.2× firing-headroom. The **$ gap grew with scale** (7% on #661's tiny vault → 46%
here) — deep's crystallization writes emit output tokens glance skips (as predicted). This **reverses** the
skeptical prior: #661's 1.2× was a small-vault artifact; the real value is material. Measure-first earned its
keep — it confirmed the value rather than killing the dial on a misleading number.

**Honest bounds:** deep is **~94s, not the ~190s premise** — so the absolute per-fire saving is a moderate
~52s, not ~150s (the ~190s may have been a heavier task / pre-`--lazy-chunks`). This is the **densest-case**
task; lighter tasks save less. The win is **per-fire recall latency/headroom, not end-to-end op cost** (notes
77/95 — the build loop still dominates end-to-end). The cost-independent **C5 recency-apply fix + #657 O2/L2
cuts** proceed regardless.

**→ #662 greenlit:** build glance/deep modes (3-phrase glance + skip write-side), land O2/L2, route C5-type
recency cues to deep (or fix recent-channel apply), trap-gate GREEN before/after.
