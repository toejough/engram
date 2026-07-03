# P1 Follow-Up — Delivery Consequence of QA Pollution

> **Joe's challenge:** The 3/48 top-5 displacements were measured (pollution is real), but was the
> DELIVERY consequence ever measured? Were the displaced notes the TARGET notes?
>
> **Scope extension (coordinator relay, mid-run):** For all 13 QA appearances, record per-appearance
> topical relevance and whether the case's target was still delivered. Split the 3 displacements into
> "displaced BY topically relevant A-note, target still delivered" vs "target lost or displaced by
> off-topic note."
>
> **Method:** Re-ran arm0 (copy vault, no QA) and arm1 (copy vault + 5 embedded synthetic pairs,
> current binary) against the preserved work-dir vaults (same vaults as the original P1 probe —
> embeddings intact). Captured ALL items[] at any rank. 48 cases × 2 arms, 8-way parallel.
> Live vault not touched.

---

## 1. Target Delivery — Per-Arm Recovery

| Metric | Arm 0 (no QA) | Arm 1 (+QA nodes) |
|---|---|---|
| Target in output at any rank | 15 / 48 | 15 / 48 |
| Target in top-5 (fact/feedback) | 2 / 48 | 2 / 48 |
| Delivery losses (arm0 delivered, arm1 did not) | — | **0** |

**Headline:** Arm 1 lost zero target deliveries. The 3 top-5 displacements carried zero delivery
consequence: the displaced note was never a target, and the affected targets were either absent from
both arms or rank-pressured but still present.

---

## 2. Three Displacement Cases Dissected

All 3 displacements originate from the same underlying query (`P1-Q01-n10`). The 3 case rows share
the query but have different target notes (72, 85, 122). The query ran once; all 3 rows reflect the
same item ranking.

| Field | Value |
|---|---|
| Case rows | P1-Q01-n10 × 3 (targets: 72, 85, 122) |
| Query topic | Link-value exploration — do wikilinks make engram recall pay off? |
| Note displaced from arm0 top-5 | `83.2026-06-24.eval-end-metric-conflates-retrieval-vs-synthesis.md` |
| Note(s) that displaced it | `qa.2026-07-03.usage-signal-design.q.md` at rank 4 (Q-note) |
| Was displaced note (83) any target? | **No** — it is incidental; none of targets 72, 85, 122 |
| Classification | Displaced BY off-topic Q-note; NOT by a topically relevant A-note |

### Per-target disposition in Q01-n10

| Target note | Arm 0 rank | Arm 1 rank | Delivery consequence |
|---|---|---|---|
| 72.…multiphrase-recall-subsumes-graph-expansion.md | 26 | 31 | Rank pressure (+5), still delivered |
| 85.…anti-amnesia-miss-not-toy-reproducible.md | not in output | not in output | None — absent from both arms |
| 122.…scope-prior-negative-evidence-before-parking-a-direction.md | not in output | not in output | None — absent from both arms |

**Summary of 3 displacement cases:**

| Category | Count |
|---|---|
| Displaced BY topically relevant A-note, target still delivered | 0 |
| Displaced BY off-topic note (Q-note or off-topic A-note), target still delivered | 3 |
| Displaced BY any note, target LOST | 0 |

---

## 3. All 13 QA Appearances — Topical Relevance + Target Delivery

The 13 appearances span 7 case rows and 3 unique query/phrase sets. Since P1-Q01-n3 runs the same
phrase set for 3 target rows, and P1-Q01-n10 likewise, relevance is identical across rows sharing
the same query.

| Appearance # | QA note | Case rows | Query topic (condensed) | QA note kind | Topical relevance | Case target in arm1? |
|---|---|---|---|---|---|---|
| 1–3 | `qa.usage-signal-design.a.md` (rank 6) | P1-Q01-n3 ×3 (targets 34, 69, 82) | Wikilink graph value; link traversal for recall | A-note: contribution in-degree usage signal for vault retention triage | **Off-topic** — usage frequency tracking is unrelated to link graph traversal value | No (all 3 targets absent from both arms) |
| 4–6 | `qa.usage-signal-design.q.md` (rank 4) | P1-Q01-n10 ×3 (targets 122, 72, 85) | Same | Q-note: "What mechanism tracks which vault notes are most useful for retention?" | **Off-topic** — retention/triage vs graph expansion for recall are separate concerns | 72 at rank 31 (still delivered); 85, 122 absent |
| 7–9 | `qa.usage-signal-design.a.md` (rank 18) | P1-Q01-n10 ×3 (targets 122, 72, 85) | Same | A-note: contribution in-degree signal | **Off-topic** — same as above, below top-5 | Same as above |
| 10–12 | `qa.qa-exclusion-seam.a.md` (rank 26) | P1-Q01-n10 ×3 (targets 122, 72, 85) | Same | A-note: why QA notes must be excluded from main cosine set; dedicated Q-channel design | **Off-topic** — QA kind exclusion is adjacent to retrieval design but specifically concerns QA node routing, not link graph expansion | Same as above |
| 13 | `qa.observable-attribution.a.md` (rank 8) | P2-B5 (target 69) | Persisting reasoned conclusions as synthesis notes; linking synthesis notes to contributing inputs for auditability | A-note: attribution must be cite-derived via wikilinks in the written answer body | **Relevant** — B5 asks about "linking synthesis notes to their contributing inputs with typed relationships for auditability"; the A-note articulates exactly this principle (observable wikilink-derived attribution) | No (target 69 absent from both arms) |

### Relevance split

| Category | Appearances | Note types |
|---|---|---|
| Relevant (topically matched) | 1 / 13 | 1 A-note (observable-attribution.a, P2-B5) |
| Off-topic | 12 / 13 | 2 A-notes + 1 Q-note for Q01 link-value query |

---

## 4. Rank Pressure Without Delivery Loss

One case where a QA note outranked the target while both were present in arm1:

| Case | Target | QA note | Arm 0 target rank | Arm 1 target rank | QA rank (arm1) |
|---|---|---|---|---|---|
| P1-Q01-n10 | `72.…multiphrase-recall-subsumes-graph-expansion.md` | `qa.usage-signal-design.q.md` | 26 | 31 | 4 |

Score also shifted (0.401 arm0 → 0.321 arm1), consistent with cluster recomposition from the added
QA nodes. Target still in output at both arms — no delivery loss, just rank degradation.

---

## 5. Cases Where Target Appeared in Output (Any Rank)

Both arms return the same 15 target notes from 15 case rows (out of 48):

| Case ID | Kind | Target note | Arm 0 rank | Arm 1 rank |
|---|---|---|---|---|
| P1-Q00-n10 | P1 | 96.…adversarial-verify-overturns-favorable-nuance.md | 20 | 20 |
| P1-Q01-n10 | P1 | 72.…multiphrase-recall-subsumes-graph-expansion.md | 26 | 31 |
| P1-Q02-n10 | P1 | 70.…red-baseline-can-falsify-the-premise.md | 16 | 16 |
| P1-Q03-n10 | P1 | 41.…funlen-wsl-tension-at-scope-boundary.md | 15 | 15 |
| P1-Q03-n10 | P1 | 43.…funlen-wsl-scope-lift-inline-instead-of-var.md | 16 | 16 |
| P1-Q03-n10 | P1 | 58.…engram-modernize-linter-slices-backward.md | 27 | 27 |
| P1-Q03-n10 | P1 | 59.…nilaway-bytes-split-nil-slice-indexing.md | 10 | 10 |
| P1-Q04-n10 | P1 | 37.…unused-struct-fields-need-early-consumer-not-nolint.md | 26 | 26 |
| P1-Q12-n10 | P1 | 59.…nilaway-bytes-split-nil-slice-indexing.md | 24 | 24 |
| P1-Q12-n10 | P1 | 70.…red-baseline-can-falsify-the-premise.md | 16 | 16 |
| P1-Q22-n10 | P1 | 59.…nilaway-bytes-split-nil-slice-indexing.md | 12 | 12 |
| P2-B9 | P2 | 74.…persisting-synthesis-differs-from-in-session-reasoning.md | 10 | 10 |
| P3-P3-5 | P3 | 136.…route-by-capability-tier-not-model-name.md | 2 | 2 |
| P3-P3-6 | P3 | 144.…under-firing-is-the-recall-risk-not-over-firing.md | 6 | 6 |
| P3-P3-7 | P3 | 136.…route-by-capability-tier-not-model-name.md | 2 | 2 |

All 15 recovered at identical ranks in both arms, except P1-Q01-n10 target 72 (+5 rank in arm1).

---

## 6. Interpretation Frame

Joe's challenge raised the question of whether QA A-notes are "pollution" or legitimate
synthesis-note competitors in the main cosine set. The per-appearance evidence supports BOTH readings:

**Pollution reading (12/13 appearances):** All Q01 appearances are off-topic. The three QA notes
surfaced for a "do wikilinks make recall pay off?" query had no semantic overlap with link graph
traversal. The one top-5 displacement (note 83) and the rank-pressure on target 72 were caused by
an off-topic Q-note (`usage-signal-design.q.md`) at rank 4 — a question note competing against
content notes is the exact failure mode the QA exclusion seam was designed to prevent.

**Legitimate competitor reading (1/13 appearances):** The observable-attribution A-note in P2-B5
was topically matched to a synthesis-attribution auditability query. This supports the argument that
A-notes (pre-reasoned conclusions) can legitimately surface for the right queries — but this
appearance produced zero delivery harm (target absent from both arms regardless).

**Net verdict:** Arm 1 delivered zero fewer targets than arm 0. The 3 displacements were incidental
(non-target note 83 displaced by an off-topic Q-note). 12/13 QA appearances were off-topic, driven
by topic-bleed between the usage-signal notes and the link-value query domain. The one relevant
appearance (observable-attribution A-note in B5) was below top-5 (rank 8) and caused no harm.

---

## Deviations

1. **Original p1_results.json stored aggregate data only.** Per-case data came from the preserved
   WORK_DIR (`/private/var/folders/…/qa-probe-XXXXXX.5zZ8iEGBiq`), which the P1 probe script
   left intact ("WORK_DIR preserved… remove manually"). Both vault copies (noqa + QA-embedded)
   were reused verbatim — no re-embedding.
2. **Original arm0/arm1 JSONs stored only top5_notes (rank≤5, kind=fact|feedback).** This follow-up
   re-ran all queries and captured full items[] at any rank, enabling the "any rank" delivery metric.
3. **All 48 cases run in 8-way parallel threads** (~3 min total). The sequential run timed out at
   5 min; the QA-affected queries were pre-analyzed first to confirm approach.
4. **Coordinator scope extension received mid-run** (topical relevance per QA appearance). Addressed
   within the same run without re-running any queries — all needed context was available from the
   full items data.
