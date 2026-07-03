# Q&A memory — proposals

**Status:** exploration round complete (design + probes, ~$0.10 spent — estimated, P2 only; P1/P3 ran at $0 — of the ~$5–15 envelope) —
proposals for Joe's decision, NO build this round.
**Ask (Joe, 2026-07-03):** remember question+answer pairs; link Q→A and answer→contributing
notes; use accumulated contribution links as a graded usage signal that outscales binary
activation; crystallize answer-reasoning not already in a note; build a compounding knowledge
graph. Plus: did we ever act on "persisted reasoning helps lesser models"?
**Evidence:** plan `docs/superpowers/plans/2026-07-03-qa-memory-exploration.md` (Gate-A-approved,
5 settled decisions D1–D5); probes committed at `dev/eval/qa/` (scripts 22db4cbf, results
bd91c5ff). Every number labeled **measured / estimated / projected**.

## The factual answer first: PARTIALLY, and the shipped half is the reasoning half

The "persisted reasoning helps lesser models" finding was **measured and half-shipped**:
- **Measured** (vault note 135, 2026-06-28): sonnet + recalled memory fully matched opus across
  apply-conventions (15/15), recency-supersession (3/3), and emergent abduction (6/6); sonnet
  cold failed. "Memory democratizes reasoning."
- **Shipped:** model-tier routing in the `route` skill (the cost lever), and deep recall's
  Step 4 (sound conclusions persisted with certainty labels + chunk provenance — the
  don't-re-derive lever).
- **Never built** (this exploration's territory): Q&A pairs as memory objects, answer→note
  provenance edges, and any graded usage signal. Ad-hoc Q&A outside a recall bracket still
  evaporates unless a lesson happens to be crystallized later.

## What the probes proved (and honestly could not)

| Probe | Claim tested | Result (pre-registered branch) | Label |
|---|---|---|---|
| P1 Arm 1 | un-excluded QA notes pollute retrieval | **Rank intrusion, zero measured delivery cost**: QA notes surfaced in 7/48 cases (13 appearances), 3/48 top-5 displacements — but the delivery follow-up (below) found 0/48 target losses | measured |
| P1 delivery follow-up (Joe's challenge, commit 2465139f) | did the intrusion COST anything at delivery? | **No**: target recovery identical both arms (15/48 any-rank, 2/48 top-5, 0 losses). Of 13 appearances: the 1 topically RELEVANT one was an A-note (rank 8, attribution query); the only displacer was an OFF-TOPIC Q-note (rank 4); 12/13 off-topic overall, driven by one query family × synthetic-pair topic bleed (n=5 pairs — direction, not magnitude) | measured |
| P1 Arm 2 | the four-point exclusion eliminates it | **PASS**: 0/48 QA appearances, 0/48 disturbances, 0 QA nominations (probe-only patched binary, all four seam points) | measured |
| P1 Arm V | Q-wording wins in a dedicated q-space (the D5 channel premise) | **BORDERLINE 7/10** paraphrases ranked their own Q note first → channel viable, larger-n check required before the channel builds (pre-registered consequence) | measured |
| P2 | cite-derived beats free-listed attribution | Fired branches (verbatim): **"BOTH<20%"** + **"RECALL-BORDERLINE"** — but DEGENERATE: only 3 historical events, all 0%/0% (historical Step-4 writes carry `--chunk-source`, never `[[basename]]` citations — no cite-derived events exist to score); see results-2026-07-03.md | measured (degenerate) |
| P3 | usage counts would have spread (top-10% ≥3× median) | Fired branch (verbatim): **"FAIL: distribution flat (CV=0.000 < 0.5)"** — from 3 OLD-format citations only; 0 vault-format contribution citations exist in the corpus, so the signal is unevaluable on real data | measured (null) |

**Interpretation correction (Joe's challenge, 2026-07-03):** the original "pollution is real"
headline judged rank shifts, not delivery — the follow-up shows the displacement cost nothing
measured, and the appearance pattern splits exactly along note kind: the one useful surfacing
was an A-note (a synthesis note in structure), the harm-shaped one an off-topic Q-note. That
asymmetry re-settled D5 (below).

**The honest reframe P2/P3 force** (the fired branches above are technically valid but carry
no evidential weight — 0/0 rates over 3 degenerate events, a flat CV over 3 old-format
citations): the attribution-fidelity and signal-spread questions cannot
be answered from history, because the capture mechanism doesn't exist yet — there is nothing to
measure until Q&A capture ships. They convert from pre-build evidence into **post-ship
validation gates**. This inverts the natural instinct ("validate before building"): here the
minimal capture build IS the instrument that makes validation possible.

### P1 per-arm detail (mandated form: arms side by side, deltas vs Arm 0)

| Metric (unit) | Arm 0 baseline | Arm 1 current binary + QA | Δ1 vs 0 | Arm 2 four-point patched | Δ2 vs 0 |
|---|---|---|---|---|---|
| QA notes appearing in items[] (cases of 48) | 0 | 7 | +7 | 0 | 0 |
| QA total appearances (count) | 0 | 13 | +13 | 0 | 0 |
| Top-5 displacements (cases of 48) | — | 3 | +3 | 0 | 0 |
| tag_nominations_total, NOT QA-specific (count) | — | 1,528 | n/a* | 1,471 | n/a* |

*The nominations field counts ALL nominations, not QA-specific ones (p1_results.json), and has
no Arm-0 measurement — the meaningful contrast is Arm 1 vs Arm 2 (1,528 − 1,471 = +57); the +57
is consistent with un-excluded vocab-tagged QA notes participating in nomination, but
the field cannot isolate them — the gate metric is QA presence in items[], which is 0 under the
four-point patch. Arm V (separate metric): 7/10 paraphrases ranked their own Q note first
(pre-registered bands: PASS ≥8, BORDERLINE 6–7, FAIL <6 → **BORDERLINE**).

### Design options per dimension (full set, honest ratings — from the gated plan)

| Dim | Option | Rating | Rationale | Build cost (est.) |
|---|---|---|---|---|
| A node shape | A1 unified note | OVERRIDDEN (D4) | containment demotes the Q to metadata; Joe wants a first-class Q node | — |
| A node shape | A2 split Q/A notes | **SETTLED (D4)** | Q is a graph node; q-space matching premise (Arm V); post-follow-up: the split is also what makes D5′ asymmetric participation possible | in round 1 |
| B edge channels | B1 dual-channel (frontmatter + body wikilinks) | **CONTENDER** | vocab precedent; InDegreeIn reads body edges; Obsidian-visible | in round 1 |
| B edge channels | B2 frontmatter only | PARK | loses graph/Obsidian leverage; stats-time frontmatter scans | — |
| C capture moments | Step 4 + /please close + ad-hoc `learn qa` | **CONTENDER (all three)** | the D2 observable-bar moments | 2 skill TDD cycles ~$2–4 + binary ~$10–30 |
| D count derivation | D-rt derived at read time | **CONTENDER** | count IS the graph state; no drift | first BuildGraph wiring, small |
| D count derivation | D-ps persisted counter | PARK | redundant state, drift on QA deletion | — |
| E triage consumer | E1 `engram usage report` | **CONTENDER (deferred to round 3)** | sorted actionable view | small, gated on P3' |
| E triage consumer | E2 inside vocab stats | PARK | keeps concerns separate | — |
| E triage consumer | E3 Obsidian only | PARK | loses the sorted triage view | — |

## The design (settled by D1–D5 + probe evidence)

- **Split Q/A notes** (D4, Joe's call): `qa.<date>.<slug>.q.md` (`type: qa-question`, body = the
  verbatim question) + `qa.<date>.<slug>.a.md` (`type: qa-answer`, body = the answer), typed
  inverse edges `answered_by`/`answers` as full basenames.
- **Machine-written contributor edges** (immune by construction to G0 — the vault's known
  bare-id link-resolution defect, catalogued in `docs/architecture/c2-containers.md`): the
  binary writes `Contributors: [[<full-basename>]], ...` on the A note, the same pattern as the
  machine-written `Vocab:` body line shipped with the vocab build — single writer,
  replace-whole idempotency. The BodyText/ContentHash exclusion is a BUILD REQUIREMENT (the
  round-1 build must add `Contributors:` to `stripMachineLines`, same as Vocab:/Supersedes: —
  not yet in code); what IS code-verified today: the vaultgraph scanner parses raw file bodies
  (scanner.go ParseWikilinks), so machine-written lines register as graph edges. Bare-id links
  are forbidden; they don't
  resolve (measured, link census in adr.md: 151/183 link-instances bare-id, 138/171 notes
  orphaned).
- **Four-point exclusion seam** (probe-validated): qa kinds excluded at pre-clustering,
  floor/cap, nomination gate, AND the TermIndex builder — the fourth point Gate A caught; the
  probe confirmed the four-point patch holds and the three-point version would have leaked.
- **Attribution is cite-derived** (D2): contributors = the full-basename wikilinks actually
  written in the answer text, auto-extracted into a `--contributors` flag — never free-listed
  (measured confabulation risk, notes 145/148/162).
- **Usage signal derived, not stored** : `vaultgraph.InDegreeIn(note, qaAnswerSet)` at report
  time. Honest cost note: `BuildGraph` has zero production callers today; the usage report is
  its first production wiring (`ScanVault → BuildGraph → InDegreeIn`) — small but net-new.
- **D5′ — ASYMMETRIC participation (re-settled by Joe 2026-07-03 after the delivery
  follow-up, superseding the original full-exclusion D5):** A-notes COMPETE in the main matched
  set — they are synthesis notes (pre-reasoned conclusions, the artifact class the set already
  values) with provenance and a question handle; the exclusion seam applies to Q-NOTES ONLY
  (`type: qa-question` — the qanchor-indicted kind, and the only displacer in the follow-up
  data). Q-notes are reached via the q-space channel (deferred pending Arm V larger-n), and a
  surfaced Q-note must deliver its A via a new `answered_by` RIDE-ALONG (supersession
  ride-along precedent; does not exist yet — round-3 build item with the channel). Build notes:
  the four-point seam covers one kind; A-notes count as vocab member notes (they carry real
  tags and compete like content); Q-notes are excluded from member counting like vocab notes.

## Recommended build sequence (each round gated on the prior)

**Round 1 — capture (the instrument):** `engram learn qa` (writes the pair + machine lines +
auto-vocab on the A note) + the four-point exclusion for `qa-question` ONLY (D5′ — A-notes
compete) + a `qa.` prefix audit of the filename-scan loops (Q-notes excluded from vocab member
counting; A-notes counted like content notes) + capture blocks in the learn and please skills
at the D2 moments (each a writing-skills TDD cycle). The `answered_by` ride-along is round 3
(with the q-channel).
Estimated ~$12–34 TOTAL (skill TDD ~$2–4 + binary work ~$10–30 — plan Dim C; not metered).
**What it delivers immediately:** no reasoning lost at substantive-answer moments; the graph
Joe can see in Obsidian; the corpus P2'/P3' need.

**Round 2 — validate (post-ship gates, cheap):** after ~2 weeks / ≥20 captured pairs:
P2' attribution fidelity on REAL captured events (same pre-registered branch set); P3'
distribution (same spread bar: top-10% ≥3× median); Arm V at larger n (≥30 paraphrases) to
settle the BORDERLINE. Estimated ~$5–10.

**Round 3 — consume (only what round 2 licenses):** the D5 Q-channel (if Arm V-large passes)
and `engram usage report` (if P3' shows spread) — retention/triage only, per D1.

**Ranking A/B sketch (deferred per D1, design falsifier on record):** arms = warm recall vs
warm recall + usage-count boost in ranking; population = the 48-case miss set + trap suites;
metric = knowledge DELIVERY (not item rank — note 119); falsified if delivery does not improve
≥2σ while collateral stays 0. Not scheduled; exists so round-3 ranking ambitions have a
pre-registered bar.

## What this does NOT do (bounds)

- No retrieval-time traversal or re-ranking anywhere (three measured A/B negatives, note 73).
- The usage signal cannot demote/prune anything by itself — retention decisions stay Joe-visible
  (D1: triage, not automation).
- Q&A capture does not fire on trivial exchanges — the D2 observable bar (cites ≥1 note OR
  crystallizes new reasoning) gates volume; expected rate ≈ 2–6 pairs per working day
  (projected from this week's cadence: ~2–3 /please closes + ~2–3 deep recalls or substantive
  ad-hoc answers per active day; P3' measures the real rate), so round 2's ≥20-pair floor lands
  in roughly 1–2 working weeks.
- The Q-channel and the `answered_by` ride-along are NOT built in round 1; until then, past
  answers reach you three ways: A-notes competing in the main matched set (D5′), Obsidian, and
  contributor links from the notes they used.
- The D5′ asymmetry rests on n=5 synthetic pairs (direction, not magnitude); round 2's
  validation re-checks A-note behavior at real corpus scale before round 3 builds on it.

## Decisions needed from Joe

0. ~~Re-settle D5~~ — DONE 2026-07-03: **asymmetric (D5′)**, chosen after the delivery
   follow-up (commit 2465139f).
1. **Green-light round 1** (the capture build) as scoped above, now with the one-kind seam?
2. **Arm V larger-n**: run it inside round 1 (adds ~$1–2, settles the channel question sooner)
   or defer to round 2 as sequenced?
3. **Naming**: `contributors` as the frontmatter key and `engram learn qa` as the subcommand —
   any preference otherwise? (Gate A flagged the name as unconfirmed with you.)
