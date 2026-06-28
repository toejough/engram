# Crystallization audit — are we writing the *right* notes?

> **Question (Joe, 2026-06-28).** The note-floor (Track 1) makes a good note surface. But "we're clustering
> and refining notes from the semantic cluster — are we deriving the notes that most helpfully answer our
> questions?" This audits the *write* side: do our crystallized notes answer real questions, and is the
> cluster-driven path as good as the correction-driven one? Data trail: `2026-06-28-crystallization-audit-data/`.

## Verdict

**Two real gaps, both upstream of retrieval.** (1) **Quality** — the **cluster-driven** crystallization path
(recall Step 2.5: distill a note from a semantic cluster) produces notes **~half as question-useful** as the
**correction-driven** path (learn Step 2): 40% vs 79% are both question-shaped and useful. (2) **Coverage** —
against real failure situations, only **2% are fully answered** by an existing note (30% partial, 68%
uncovered). The fix both point at is **question-shaped crystallization**: derive the situation handle (and the
lesson) from the *anticipated question / the failure it prevents*, not from the cluster's shared topic. This is
the next investigation; Track 3 only measures that it's warranted (implementing it is out of scope here).

## Method

| Item | Value |
|---|---|
| Notes audited | **98 vault notes**, judged blind to their crystallization path |
| Path classification | from `source:` frontmatter — **correction-driven 63** (`session … context/correction`), **cluster-driven 24** (`synthesized from chunk cluster`), synthesis-derived 8, other 3 |
| Note-audit judgment (haiku) | per note: *question-shaped* (situation phrased as a future task's moment, vs the cluster's topic), *content-quality*, *future-task-useful* |
| Coverage judgment (haiku) | 40 sampled real failure situations (from the 137-failure corpus) → notes-only isolation query → does a surfaced note *answer* the need? |

## Finding 1 — Quality: the cluster path under-produces (the path-split)

The three left columns are the Method's three judgment dimensions (`content-quality` shown as its
*distinct-actionable* share); the last is derived. Counts are over non-empty-body notes: **58 correction-driven,
15 cluster-driven, 8 synthesis-derived**.

| path | question-shaped | content: distinct-actionable | useful (y) | **question-shaped AND useful** |
|---|--:|--:|--:|--:|
| **correction-driven** (learn Step 2) | 83% | 90% | 90% | **79%** |
| **cluster-driven** (recall Step 2.5) | 62% | 47% | 47% | **40%** |
| synthesis-derived | 62% | 75% | 38%* | — |

(*distinct-actionable* and *useful=y* land on the same percentages because the same notes scored well on both —
a coincidence of the data, not one column restating the other.) Cluster-driven notes are **53% narrow-impl or
vague-aggregation** (vs 10% for correction-driven), and 2 of 15
were judged *not* useful for any future task (vs 0 of 58 correction-driven). **Why** (note 68,
`aggregation-is-not-emergent-synthesis`): clustering groups *co-topical* items, so the distilled note is pitched
at the shared **topic** the cluster is about, not at a future **question** an agent would phrase. The
correction-driven path is better precisely because the learn skill writes the situation "the way a future task
would describe it." (*synthesis-derived "useful=y" is low because many are meta/project-strategy notes; not the
core contrast.*)

## Finding 2 — Coverage: the vault under-answers the real question space

| coverage of 40 real failure situations | count | share |
|---|--:|--:|
| **covered** (a note delivers the needed lesson) | 1 | **2%** |
| **partial** (a note is related but misses the actionable point) | 12 | 30% |
| **uncovered** (no surfaced note answers the need) | 27 | **68%** |

**Read honestly:** the corpus *is* the failure situations — by construction, places where memory didn't help —
so low coverage is partly a **capture gap** (these lessons aren't crystallized yet), which is exactly what the
failure-mining pipeline (`2026-06-28-failure-eval-material.md`) was built to fill. The **30% partial** is the
sharper signal: a *related* note exists but **misses the actionable point** — a quality/precision gap that
question-shaped crystallization would close.

## What this means (and where it sits)

- The **floor (Track 1, shipped)** ensures a good note *surfaces*. This audit says we also need the notes to be
  *worth* surfacing (quality) and to *exist* (coverage).
- **Next investigation — question-shaped crystallization:** when writing a note (especially on the cluster
  path), derive the `situation` handle from the **question/failure it answers**, not the cluster topic; consider
  routing cluster-driven candidates through the same question-shaping the learn (correction) path uses. This
  fixes Finding 1 directly and, paired with failure-mining capture, closes Finding 2's partial+uncovered gap.
- **Relationship to note 68:** the deeper limit is that engram does aggregation, not relational synthesis — the
  cluster path is structurally topic-shaped. Question-shaping is the cheap near-term lever; the vaultgraph
  relational substrate (note 68) is the longer arc.

## Honest limits
- **N for cluster-driven is modest** (~15–16 non-empty-body notes) — the 40%-vs-79% split is a clear but
  *suggestive* signal, not a tight estimate. Re-run on more cluster-driven notes before treating the exact
  percentages as fixed.
- **Coverage corpus is failure-biased by construction** — it measures the vault against the *hard* cases
  (where memory already failed), so 2% covered is a floor, not the vault's coverage of routine work.
- **Agent-judged** (haiku). The note-audit was **blind to path**: the judging input gave each note's
  `situation` + body + `type` only — the `source:` frontmatter (which names the crystallization path) was
  withheld — so the path correlation in Finding 1 is not judge bias. Still a single-judge pass — directional,
  not adjudicated.
- Notes 116–119 had empty bodies in the audit input (written after the probe inventory); excluded from
  content-quality stats.
