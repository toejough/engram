# Vocab notes + linking replacement — build results (2026-07-03)

> Slices 1–5 are complete and merged. This doc records what shipped, the gate numbers,
> one honest incident, and the spend.

## What shipped (slices 1–5)

| Slice | Scope | Key commits |
|---|---|---|
| 1 · Binary substrate | Exclusion filters (`type: vocab` at `applyFloorAndCap`/`capWithNoteFloor` + pre-clustering); term-note model; write-time cosine assigner (centroid two-pass, floor 0.35, plain top-3 — the close-3rd rider was swept and dropped in slice 2); dual-channel `Vocab:` writer (body line + `vocab:` frontmatter, idempotent replace-whole); `vocab:` field added to typed frontmatter structs; `--supersedes` flag + frontmatter + inverse maintenance; `--relation` removal pass (~200 LOC, 6 functions, `migrateRelationLinks`/`migrate-links` retired, `RelatedSectionMarker` consumers audited); `outbound_links` payload field removed | `537d4a1a` + fixes `722b43ff` (2026-07-02) |
| 2 · Vocab commands | `engram vocab bootstrap` (seeds 25 term-notes from `dev/eval/links/fabrics/l6.json`, embeds, tags all existing notes, generates `vocab.index.md`, idempotent); `engram vocab propose` (LLM-gated: no-overlap + ≤20% attachment); `engram vocab stats`; `engram vocab refit` (LLM-judged: merge/split/rename + member-link rewrite + major-version bump); assignment wiring into learn/amend (term vectors created by bootstrap); assignment tuning sweep completed | `43f55a9a`, `12d72074`, `7a107e51` + fixes `f824c85e` (2026-07-02/03) |
| 3 · Query integration | Tag-match nomination in `candidate_l2s` (budget fields `tag_nominations_added`/`dropped`, cap 40/cluster); superseded-note ride-along (+1 note per hit); gated: C3–C6 smoke GREEN, 38 zero-miss replays, S2 recovery probe | `d426e53c` + fixes `c115004b`, `6251ebb2` (2026-07-03) |
| 4 · Skill rewrite | recall SKILL.md: Step 2.6 deleted, Step 2.5C/4 rewritten (no `--relation`, `--supersedes` only when correcting), overview + mode refs updated; learn SKILL.md: `--supersedes` + auto-vocab language; writing-skills TDD (headless RED/GREEN) | `22850e56` (2026-07-03) |
| 5 · Migration | Live vault backup verified (file count equal); `engram vocab bootstrap` on live vault; 84 relation edges inventoried and classified (criteria: supersession only if rationale states refutes/narrows/updates/corrects/scopes) → 7 supersession edges = 6 unique relationships typed; 76 dropped; 1 dangling; all archived in `docs/design/artifacts/2026-07-02-retired-relation-rationales.md`; "Related to:" sections stripped from all note bodies; vocab.index.md generated; `engram check` GREEN | `e3a8c3fc` (2026-07-03); vault not in git |

## Gate numbers

### Assignment sweep (slice 2, bootstrapped copy vault, 48-case miss population; 2026-07-03)

The original pre-registered bar (recovery ≥ 60% AND median pool ≤ 40) proved JOINTLY UNREACHABLE and
was found miscalibrated — its cited 64.6% baseline had been measured at pool 44, so the bar's two
halves never co-occurred in any measurement. The slice-2 executor surfaced the FAIL; Joe re-anchored
the gate to the delivery-validated operating point (the link-exploration S3 eval proved +17.3pp
delivery at 54.2% recovery / ~30 pool): **PASS = recovery ≥ 54.2% AND median pool ≤ 40.**

Final sweep over the shipped mechanism arm (centroid two-pass), floor ∈ {0.25, 0.30, 0.35, 0.40} ×
K ∈ {K2+rider, K3 = plain top-3} — full table in `dev/eval/links/sweep_s2_results.json`:

| config (arm \| floor \| K) | recovery (% of 48 cases) | median pool (notes) | verdict vs re-anchored bar |
|---|---|---|---|
| twopass \| 0.35 \| K3 | 56.2 (27/48) | 31.0 | **PASS — SELECTED** (tie on recovery → higher floor) |
| twopass \| 0.30 \| K3 | 56.2 (27/48) | 33.0 | PASS |
| twopass \| 0.25 \| K3 | 56.2 (27/48) | 39.5 | PASS |
| twopass \| 0.40 \| K3 | 50.0 (24/48) | 44.0 | FAIL (both halves) |
| twopass \| 0.25–0.40 \| K2+rider | 41.7–50.0 | 24.0–33.0 | FAIL (recovery) |

**Shipped defaults: floor 0.35, plain top-3 (K3), centroid two-pass.** K3 beat K2+rider at every
floor in the shipped arm; the close-3rd rider was dropped. Real-binary confirmation (parsing the
actual `vocab:` frontmatter written by `engram vocab bootstrap` on a fresh copy): 56.2% (27/48) @
median pool 31.0 — exact parity with the model.

### Slice-3 gates (2026-07-03, bootstrapped copy vault, worktree binary)

| Gate | Criterion | Result |
|---|---|---|
| S2 recovery on the REAL binary's emitted candidates | ≥ 54.2% AND median added candidates ≤ 40 | **77.1% (37/48) @ 28.0 — PASS** (the binary's per-cluster nomination outperforms the flat python model) |
| C3–C6 trap smoke (worktree binary on PATH) | GREEN | **GREEN** ($3.09) |
| No-regression | 0 baseline top-5 disturbances | **PASS via structural adjudication** — vault drift (notes 154–159 postdate replays.json) invalidated the recorded-replay comparison; Gate B traced the code and confirmed nomination is append-to-candidates-only and ride-along was a no-op on the then-fabric, so ranked output is structurally unchanged; invariants (no phantom ride_along items, valid YAML on all 38 replays) held at 0 violations |

### Live-vault verification (slice 5, post-migration, 2026-07-03)

| Check | Result |
|---|---|
| S2 recovery probe (proxy over live `vocab:` frontmatter) | **56.2% (27/48) @ median pool 31.0 — PASS** |
| `engram check` | PASS (3 pre-existing prose-wikilink dangles in notes 33/142, unrelated to the migration) |
| `engram embed status` | 0 stale (hash exclusion of machine lines held through the strip) |
| Spot queries (3) | candidate_l2s + `tag_nominations_added` budget field emitted; `outbound_links` gone |
| Supersessions written | 6 typed relationships (3 narrows, 3 updates) from 84 inventoried edges (7 supersession edges — the 120↔153 pair was reciprocal; 76 dropped; 1 dangling); rationales archived; 41 notes stripped |

## Eval-arm contamination incident (2026-07-02, honest record)

During the **link-value exploration's S3 delivery eval** (the study that motivated this build — not
this build's own slices), the delivery-arm `claude -p` processes (opus, `bypassPermissions`, temp
working directories) were handed payloads whose note contents included absolute paths into the real
repo. Several arms followed those paths out of their temp cwd and **performed the tasks they were
asked to merely plan**: between 10:19 and 12:54 on 2026-07-02 they edited 16 `internal/` files
(relations-merge, ingest, transcript-stripping work) and wrote two untracked files (a
"memory-system-value-retrospective" doc — literally the Q00 eval case executed for real — and a
test file). The side-effects were discovered 2026-07-03 when the dirty tree blocked this build's
merge, diagnosed by mtime + content correlation with the eval cases, and **discarded after archiving
a full recovery patch + tarball** (`~/.claude/jobs/9e790e0a/tmp/eval-arm-sideeffects.patch`,
`eval-arm-untracked.tgz`). The live vault was never touched. Lesson (vault note pending): eval arms
must not run with bypassPermissions while real repo paths are reachable from their payload content;
harness hardening is queued as a followup.

## Spend (this build; the motivating link-value exploration has its own ledger: $222.93, see `docs/design/2026-07-02-link-value-exploration.md`)

| Item | Cost (direct metered API) |
|---|---|
| C3–C6 trap smoke, slice 3 (worktree binary) | $3.09 |
| Skill rewrite headless RED/GREEN ×4 runs, slice 4 | $2.85 |
| Bootstrap / sweeps / probes (binary + python, LLM-free by design) | $0 |
| Edge classification, slice 5 (executor judged the 84 rationales directly) | $0 direct |
| **Direct metered total** | **~$6** |

Orchestrator/executor/reviewer agent tokens are not separately metered here. Direct API spend came
in well under the plan's $10–30 estimate because the two-phase LLM-as-files architecture kept the
binary LLM-free and the executors did the judgment work in-context.

## What remains

- **Obsidian acceptance** (Joe signs off): graph view shows ~25 term nodes as visible hubs
  (each ≥ 2 member spokes; `eval-methodology` ≈ 33 spokes); `vocab.index.md` links to every term
  note; clicking a term note's backlinks lists its members; a Dataview
  `TABLE ... WHERE contains(vocab, "eval-methodology")` returns the member set.
- **Refit lifecycle live**: `engram vocab stats` → monitor untagged-rate; `engram vocab refit`
  fires when untagged-rate > 10% of last 25 writes, OR any term > 25% of vault, OR vault grew
  > 30% since last refit.
- **Supersession re-smoke**: L5×T5 mechanism proven, delivery underpowered (n too small). Re-eval
  after the fabric grows (target: ≥ 10 supersession edges in the live vault).
