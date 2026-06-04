# Memory-System Fix Plan (Phase 7)

**Goal:** fix every confirmed defect from the rigor effort (operator: "fix all"), gated by a new
executable invariant checker, then re-run the eval from scratch.

**Source of truth:** [invariants](2026-06-04-memory-invariants.md) · [rigor plan](2026-06-04-memory-system-rigor.md) ·
ADRs + C4 in `docs/architecture/`. Every fix is TDD (RED via the checker or a unit test → GREEN → REFACTOR).

> **For checkpoint 2:** §A (strategy/order) and §B (per-fix approach) are for approval. §D lists the
> **5 design decisions that need an operator answer before Phase 8 implements them.** The mechanical
> FAIL fixes (§B tier 1) need no decision — they can start on approval.

---

## A. Strategy & ordering

1. **Build the tripwire first — `engram check` (§C).** It operationalizes the VC-class invariants as
   a `targ` gate. It is the RED for most correctness fixes: the check reports the defect, the fix makes
   it pass. Its absence (D7) is the root cause of this whole effort, so it ships first.
2. **Then the high-impact FAIL fixes the operator named:** G0 (empty graph) and T1a (tier leak).
3. **Then the rest of the FAIL set:** E4, M2-segments, M4, M5, E5, G5.
4. **Property tests** (K1, M1–M3, M6–M8, C1, R1) land alongside their packages.
5. **Design-decision fixes** (§D) after the operator answers — INV-S1, INV-S2, L3 sparsity, and the
   G0/G5 approach picks.
6. **Re-run the eval from scratch** (operator's stated next step) once `engram check` is green.

Worktrees per fix where files would collide; ff-only merges after review (per CLAUDE.md).

---

## B. Per-fix approach (TDD)

### Tier 1 — FAIL-class correctness

**G0 — link-form resolution (the headline; 155/183 edges dropped). [D1 LOCKED: full basenames — Obsidian convention]**
- *Approach:* `learn` resolves a relation's id (`--relation "105|…"`) to the target's full basename and
  writes `[[105.<date>.<slug>]]` — which resolves in **both Obsidian and engram**. The resolver
  (`BuildGraph`) stays basename-only, matching Obsidian, so the Obsidian graph == the engram graph. The
  skill keeps passing ids; the binary does the id→basename lookup at write. Plus a one-time **migration**
  rewriting the 151 existing bare-id links to full basenames.
- *RED:* `engram learn fact --relation "105|x"` writes `[[105.<date>.<slug>]]` (not `[[105]]`); the
  migration rewrites all 151; `engram check` resolution check → **179 edges** (verified: 0 dangling).
- *Files:* `internal/cli/learn.go` (write-side resolve) + a migration step; `internal/vaultgraph` unchanged.

**T1a — tier isolation across ALL channels (live leak: 44 non-L3 under `--tier L3`).**
- *Approach:* extend `applyTierFilter` so when `--tier` is set it constrains `clusters[].members`,
  `nearest_l3`, and `hubs` too — not just `items`. Absent `--tier`, unchanged (blended). §6b uses
  un-tiered queries, so synthesis is unaffected.
- *RED:* `engram query --tier L3` returns only L3 across every channel (unit/integration).
- *Files:* `internal/cli/query.go`.

**E4 — staleness hash must cover the embedded text.**
- *Approach:* compute `ContentHash` over `Text(raw)` (what is actually embedded) rather than
  `ExtractBody`. Then for episodes the hash covers `situation`, for facts it covers the body — always
  aligned. Bumps the hash for the **64 episodes only** (`Text==ExtractBody` for facts → all 106 facts
  unchanged); one `engram embed apply --stale` pass restamps exactly those 64.
- *RED:* editing an episode's `situation` changes its `content_hash` (unit, `internal/embed`).
- *Files:* `internal/embed/hash.go`.

**M2-segments — marker over-advance on the segments path.**
- *Approach:* give `SegmentsFrom` a truncation signal (a `Partial` on its result, mirroring
  `ReadFrom`), and gate `emitSegments` to hold the marker at the last fully-included row on truncation.
- *RED:* a budget-truncated `SegmentsFrom` does not advance the marker past the last read row (property test).
- *Files:* `internal/transcript/transcript.go`, `internal/cli/transcript.go`.

**M4 — model-homogeneity guard (no silent recall-emptying on a model swap).**
- *Approach:* when `loadCompatibleSidecars` drops sidecars on a `model_id` mismatch, surface a
  warning (and a distinct error if it drops *all* of them) instead of silent empty recall.
- *RED:* a vault with mixed `model_id`s yields a warning/error, not a silent empty result.
- *Files:* `internal/cli/query.go`.

**M5 — `situation` required on fact/feedback.**
- *Approach:* mark `--situation` required for fact and feedback args (today required only for episode).
- *RED:* `engram learn fact` without `--situation` exits non-zero (targets test).
- *Files:* `internal/cli/targets.go`.

**E5 — empty-episode-situation guard.**
- *Approach:* make an empty episode `situation` a hard error at learn time (no silent body fallback in
  `embed.Text`); the checker also flags any existing empty-situation episode.
- *RED:* `engram learn episode` with empty `--situation` errors; `engram check` flags empties.
- *Files:* `internal/cli/targets.go` (validation) + `engram check`.

**G5 — false edges from episode transcripts. [D2 LOCKED: code fence + parser skips fences]**
- *Approach:* wrap the episode transcript in a fenced code block (under `## Transcript`); engram's
  wikilink parser **skips fenced code regions, exactly as Obsidian does** — so neither tool turns
  transcript `[[x]]` into edges, and the two graphs match. Authored relations live under `## Related`
  (outside any fence) and still resolve. (Pairs with the D6 episode-format work.)
- *RED (both directions):* `[[105]]` inside an episode's fenced transcript → **no** edge (engram and
  Obsidian agree); `[[105]]` under `## Related` → **edge**.
- *Files:* `internal/vaultgraph/scanner.go` (fence-aware parse), `internal/cli/learn.go` (write fenced
  transcripts + sections), + 64-episode migration. Variable-length fences for transcripts containing ```.

### Tier 3 — property tests (no behavior change)
- **K1** vault write-lock, **M1/M2/M3** marker forward-progress (promote unit→property), **M6**
  idempotency, **M7** monotonicity, **M8** luhmann uniqueness, **C1** clustering determinism,
  **R1** recall-mirror. rapid tests in the owning packages.

---

## C. The tripwire — `engram check`

A new read-only subcommand over the vault. Implements the VC-class invariants (graph G0/G1/G5,
tier T1a, embed E1/E4/E5, provenance P1, plus model M4, situation M5, id M8). Exits non-zero on any
**FAIL-class** violation; reports WARN-class non-blocking. Wired as a `targ check-full` gate so a
regression fails CI. UTF-8-robust parsing (the BSD-grep skipped-7-notes lesson). Output: per-invariant
PASS/FAIL + counts, so it doubles as the built-vs-docs report on every run.

---

## C2. Scope of "fix all" — nothing silent

- **Also fixed:** U1 (`update` idempotence) gets an integration test in `internal/update` (re-run with
  identical source = copy-equivalent no-op; sentinels on missing-go / no-harness / missing-skills).
- **Consciously deferred** (tracked, not dropped): P1's single missing provenance (episode 126's
  transcript path) — a data cleanup, not a code bug; R2 graceful-degradation (a guideline, not a
  checkable invariant); L3-1 match-stability (folds into D5's synthesis work).

## D. Design decisions needed before Phase 8 (checkpoint 2)

| # | Decision | Options | Recommendation |
|---|---|---|---|
| **D1** | G0 fix locus | (a) resolver normalizes bare-id→basename; (b) `learn` writes full basenames (Obsidian convention) | **LOCKED (operator): (b) full basenames.** Verified Obsidian resolves only full filenames/aliases, NOT bare `[[105]]` — so full basenames make the graph work in Obsidian AND engram. `learn` resolves the relation id→basename at write; migrate the 151 existing links; resolver stays basename-only (Obsidian-consistent); skill still passes ids. |
| **D2** | G5 — false edges from episode transcripts | (a) code-fence transcripts + engram parser **skips fenced regions, mirroring Obsidian**; (b) parse only the `Related` section | **LOCKED (operator): (a) code fence.** Both tools ignore links in fences → engram graph == Obsidian graph. Pairs with the episode-format expansion (**D6**). Cost: 64-episode migration. |
| **D6** | Episode note format (operator-requested) | `## Summary` (narrative + boundary rationale) · `## Transcript` (fenced) · `## Related` (relations + **preceding-episode links**) | **LOCKED (operator): (i) local chain** — preceding links = episodes *active* at this one's start (started before it, not ended by it) + the single most-recent prior if none overlap. Binary auto-computes from transcript-range start times; links are full basenames (D1); summary is skill-authored (writing-skills TDD); re-introduces episode summaries *alongside* the fenced transcript. Migration backfills fences + preceding-links on the 64 episodes (summaries left blank — can't auto-generate). |
| **D3** | INV-S1 — skill reads/edits vault directly | (a) return member content in payload; (b) keep paths-only, agent reads on judgment | **LOCKED (operator): keep paths-only.** The §3a agent-read is **intended** — the agent decides per-cluster whether the abstract signal is worth investigating; don't force-load content. INV-S1's read-half is accepted design (not a defect); only the §6b *edit* is fixed — via D4's `engram resituate`. |
| **D4** | INV-S2 — fact stores `situation` twice | (a) `engram resituate` rewrites BOTH copies + re-embeds, `engram check` asserts they match; (b) embed only the frontmatter situation (drops subject/predicate/object signal) | **LOCKED (operator): (a).** Keeps facts' rich body embedding; gives §6b a safe sync-preserving edit path (also closes INV-S1's §6b write-half). |
| **D5** | L3 sparsity (1 L3 from 106 L2) | **LOCKED (operator 2026-06-04):** per new/changed fact, §6b runs all the fact's seeds, **unions** their matches, clusters that union **once**; AutoK (k=2–7, sil ≥0.10) only decides *how many* L3s the matches split into, and **K=0 means one cluster** (never "nothing"). **Drop the ≥6-member floor** for synthesis. Each cluster → centroid → `nearest_l3` ≥0.9 → update-or-create. ≥1 candidate every time. | Binary: a synthesis-clustering path (cluster the *merged* hit set, K=0→single cluster) distinct from recall's per-phrase clustering; skill §6b unions a fact's seeds into one synthesis query. G0's fix enriches the unions. |

D3/D4/D5 are skill+binary changes with real behavior impact — they want an operator answer. D1/D2 have
clear recommendations; confirm or override.

**Any fix that edits `recall`/`learn` SKILL.md (D3, D4, parts of D5) MUST go through
`superpowers:writing-skills` — RED baseline behavior test → GREEN → pressure tests — per the project
rule; it is not an ordinary code edit.** D5 is now traced + locked: K=0 means AutoK found no ≥2-way
separation (it never returns *one* cluster — a coherent blob of related matches scores silhouette
<0.10), compounded by G0's empty graph (BFS reaches nothing) and per-phrase clustering. The locked
fix makes K=0 mean *one cluster* over the unioned seed matches, so synthesis always has a candidate.

---

## E. After the fixes
Re-run the cold/warm + tier-regime eval **from scratch** (operator's instruction) once `engram check`
is green and `--tier` isolation is real — only then does any tier-regime comparison mean anything.

> Phase 8 will expand each §B/§C item into bite-sized TDD steps (superpowers:writing-plans /
> subagent-driven-development) once §D is decided.
