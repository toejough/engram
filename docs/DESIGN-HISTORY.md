# Engram — Design History

A consolidated record of the decisions that shaped engram, in the order they
were made, with the *why* for each and a `git log`-recoverable path to the
original artifact. It exists so a future maintainer can understand **why the
system is shaped as it is** without spelunking deleted plans.

The detailed originals (research logs, per-decision specs, eval result tables,
implementation plans) were deleted in the issue-649 cleanup but remain in git
history — search by the dated filenames cited below, e.g.
`git log --all --oneline -- docs/superpowers/research/2026-05-22-tiered-memory-research-log.md`.

The **active** design home is `docs/architecture/` (C4 diagrams c1/c2/c3 +
`adr.md`). This file is history, not current truth — where the two disagree,
the architecture docs win.

---

## 1. Tiered-memory research (2026-05 — the founding model)

**Decision: a four-tier "lopsided hourglass" — L0 provenance index, L1 stripped
session segments, L2 atomic facts, L3 synthesis — append-only at L0–L2,
regenerated at L3.**

The motivating failure: every Permanent note distilled a session into a
principle, but the session itself was invisible to recall ("you did it but you
don't remember"). The information lived in `~/.claude/projects/` JSONL; only the
absence of an index made it unrecoverable. The redesign exists to stop
discarding that substrate while preserving the MOC craft rules (LLM-voiced
framing prose, no global index, every wikilink in an explaining sentence) and
preserving contradictions rather than smoothing them.

**Constraints locked here (still binding):** pure Go / no CGO (rules out FAISS,
ONNX-runtime, sqlite-vss); no LLM calls from the binary (read-time stays offline
and snappy; embedding at write time is fine); L0–L2 append-only, L3+
regenerated; Luhmann IDs retained at L2 as a *secondary* retrieval signal only.

A cognitive-science review (`2026-05-22-human-memory-literature-summary.md`)
pressure-tested the model and flagged two things engram lacked and later built
toward: forgetting is *adaptive inhibition*, not decay (motivates demote /
recency-decay rather than pure append-only), and **there was no working-memory
analog** — effective retrieval combines recency + importance + relevance, not
relevance alone (Generative Agents/ACT-R). This finding directly seeded the
2026-06 short-term-memory work below.

The question board (`2026-05-22-tiered-memory-research-log.md`) resolved the
shape: F1 added **episodes** as a third Permanent kind; F2 chose the embedder
(below); F4 **dropped MOCs as a write kind** and migrated them to facts/feedback;
F5 dropped a link-as-we-go sidecar (embeddings already cluster); F6/F9.1 put
**subgraph clustering at query time** (3-hop wikilink BFS → auto-k-means by
silhouette → in-degree hubs) but kept **synthesis in the consuming skill, not in
the binary** — engram stays mechanical/offline.

- Founding design: `docs/superpowers/specs/2026-05-14-tiered-memory-design.md`
  (+ its brief `2026-05-12-tiered-memory-research-prompt.md`)
- Research log / decision board: `docs/superpowers/research/2026-05-22-tiered-memory-research-log.md`
- Cognitive-science grounding: `docs/superpowers/research/2026-05-22-human-memory-literature-summary.md`

## 2. Embedder choice + the v2 query spike (2026-05-23/24)

**Decision: bundle a sentence-transformer in the binary and run it pure-Go via
Hugot + GoMLX `simplego`; dual-vector sidecars; semantic `engram query`.**

The pure-Go/no-CGO constraint drove this. Arctic-xs was the first pick with
MiniLM-L6-v2 as the fallback; the spike's UAT showed Arctic-xs fell back, so
**MiniLM-L6-v2@384 is the shipped bundled model** (`go:embed`-ed). Each note
gets a sibling `.vec.json` sidecar (the on-disk contract `internal/embed`'s
`Sidecar` type still implements verbatim); the spike defined the snake-case
sidecar keys that are now a frozen file format.

- Embedder options: `docs/superpowers/research/2026-05-23-embedder-options.md`,
  `2026-05-24-simplego-model-compat.md`
- Query/sidecar spike spec: `docs/superpowers/research/2026-05-24-engram-query-spike.md`
  and `docs/plans/2026-05-24-engram-query-spike.md`
- MOC migration procedure: `docs/superpowers/research/2026-05-24-moc-migration-procedure.md`

## 3. v2 shipped + v3 field-query research (2026-05-25/26)

**Decision: v2 is the embeddings-driven retrieval rewrite — episode kind,
subgraph clustering, single-call `/recall`.**

The v2 roadmap closed with: the spike pipeline, the F4 MOC migration (25 MOCs →
~39 facts/feedback; `engram learn moc` retired), the **episode kind** as L1
evidence (`engram learn episode` with narrative-voice discipline + transcript
provenance), F6/F9.1 subgraph clustering in `engram query`, and a `/recall`
SKILL.md rewrite from a manual cascade to a single `engram query` call with a
per-cluster synthesis gate. v3 field-query / signal-based retrieval (recency,
same-project, same-code-area) was researched as a *substrate a later milestone
might need* — explicitly not built until evidence demanded it.

- v2 execution roadmap: `docs/superpowers/research/2026-05-25-v2-execution-roadmap.md`
- Episode kind spec: `docs/superpowers/research/2026-05-25-episode-kind-spec.md`
- Subgraph-clustering spec: `docs/superpowers/research/2026-05-25-f6-f91-spec.md`
- Multi-phrase query + L1 fix + v3 field research:
  `docs/superpowers/plans/2026-05-26-multi-phrase-query.md`,
  `docs/superpowers/research/2026-05-26-l1-episode-fix-spec.md`,
  `docs/superpowers/research/2026-05-26-v3-field-query-research.md`

## 4. L3 synthesis tier + tier-capped retrieval (2026-05-31)

**Decision: add L3 ADR-style notes that synthesize an L2 cluster into a
scenario-discoverable standard; `engram query` returns only the top tier
present.**

L2 only surfaces if you query its keywords — but an agent about to make a
mistake doesn't know the lesson exists. L3 generalizes a cluster into a crisp
standard discoverable from the *scenario* the agent is in, surfaced before they
act. Down-links let the agent chase L3 → L2 → L1 evidence by choice. This
synthesis-at-learn-time model (ADR-0005) was **later superseded** by lazy-L2
recall-time synthesis (§7).

- L3 design: `docs/superpowers/specs/2026-05-31-l3-synthesis-tier-design.md`
- Tier binary/synthesis/validation plans:
  `docs/superpowers/plans/2026-05-31-l3-tier-{binary-foundation,synthesis-flow,validation-chain}.md`

## 5. The memory-eval harness (2026-05-29 → 2026-06-08, the empirical turn)

**Decision: stop debating retrieval design and *measure* it — an end-to-end
behavioral eval where real headless Claude Code agents drive build tasks with
the actual shipping skills, scored differentially across config arms.** No
deterministic gold-set; score what the agent actually *does* (cost/efficiency +
behavioral failure-modes like repeating a known lesson). Harness lives in `dev/`,
invoked `targ eval`; the engram binary + skills are external artifacts under
test.

**The harness immediately reshaped the value question.** Run 1's calibration
gate fired RED — no floor-vs-baseline delta — because the vault's discriminating
value is **engram-domain-bound** (created *during* engram's own build); measuring
it on portable greenfield tasks was a category mismatch, and codified conventions
(targ, AI-Used, DI) live in static `CLAUDE.md`/rules loaded by *all* arms, so
they can't isolate the vault's contribution. This forced the **self-seeding
cold-vs-warm** redesign: let the agent generate its own memories from its own
mistakes, then measure whether replay helps (memory effect = cold − warm).

**The pivotal early finding (n=1 probe):** memory was net-*negative* on an easy
task — warm took ~2× turns and ~2.5× cost. Memory pays off only when the
**avoided dead-end cost exceeds recall's own overhead**; on an easy task with no
expensive stumble, surfacing + processing recalled notes is pure tax. This is the
single most important correction to "memory always helps."

**Key eval conclusions (the v2 cumulative matrix, 2026-06-08/10, n=5, 3 models):**
- **Memory cuts CONVENTION restatement far more than FEATURE restatement** —
  the transferable-vs-app-specific gap (haiku 30% vs 9%; sonnet 55% vs 2%; opus
  58% vs 28%) is the real signal. Memory is a **capability amplifier, not an
  equalizer**: it helps stronger models *more*, widening the gap.
- **Memory's win is say-once (front-loaded correctness), not cost.** It does
  **not** reduce time/tokens/$ on a short horizon — recall + richer learn cost
  more per build. The cost case is "teach each convention once, ever," amortized
  over many convention-sharing apps; a 3-app chain *understates* warm. On a
  one-off or two, **cold is the cheaper reliable floor**.
- **For strong models, tier choice was flat** (L1/L2/L3 spread was n=5 noise);
  the decision that matters is cold-vs-warm, not which tier. `l2.l2` was picked
  as the *never-worst-across-capability* config (it rescued haiku to 80%
  completion), a robustness tiebreak, not a measured tier victory.

- Harness design: `docs/superpowers/specs/2026-05-29-memory-eval-harness-design.md`
  (+ plan `docs/superpowers/plans/2026-05-29-memory-eval-harness-m1.md`)
- Cold-warm tests + validation program: `docs/superpowers/specs/2026-05-30-{cold-warm-todo-test,memory-validation-program}.md`,
  `2026-06-05-cold-warm-postfix-rerun.md`
- Cumulative-accumulation eval + results: `docs/superpowers/specs/2026-06-01-{cumulative-accumulation-eval,final-memory-eval}.md`,
  `2026-06-02-cumulative-accumulation-results.md`, `2026-06-06-cumulative-accumulation-v2-brief.md`
- Eval result tables (deleted in Unit F; conclusions folded in above):
  `dev/eval/cumulative/results-{table,v2,lazy-l2-opus,real-skill-haiku,real-skill-opus}.md`

## 6. Memory-system rigor + invariants (2026-06-04)

**Decision: codify the as-built system's invariants and verified defects as
first-class artifacts** — a built-vs-docs / fix-plan rigor pass that produced the
**memory-invariants** spec and the **memory-system-rigor** effort.

These two specs are the exception to this cleanup: **they remain live** and are
linked from the architecture docs (`adr.md`, `c2`, `c3`). They are not historical
— see `docs/superpowers/specs/2026-06-04-memory-invariants.md` and
`docs/superpowers/specs/2026-06-04-memory-system-rigor.md` (+ the fixes plan
`docs/superpowers/plans/2026-06-04-memory-fixes.md`, historical).

## 7. Lazy, compositional L2 synthesis (2026-06-09 → 2026-06-18, the current model)

**Decision: generate L2 lazily, at recall, judged by the agent — supersedes
eager/proactive L2 synthesis and the L3-ADR-at-learn-time model.**

Hypothesis: an L2 is worth generating once it *proves relevant* (a real recall
demands it) → fewer, only-relevant L2s; and because synthesis draws on clusters
that include existing L2s, the layer becomes **recursive/compositional**. The
real-skill eval confirmed it for opus: lazy **matched eager on first-attempt
convention quality at ~15% lower cost and ~⅓ the L2 volume** (5 vs 13 notes);
for haiku the eval couldn't separate the two under convergence/capture noise, but
lazy was no worse and far leaner. Eager's extra L2 volume bought nothing
measurable and is a retrieval-precision + vault-bloat liability.

Two lessons drove the converged v2 model (after five adversarial review gates):
- **Chunks do not belong in the vault *note* model.** Every attempt to make them
  vault files collided with Obsidian-parity, the scanner, embed/check, and
  Luhmann IDs. So chunks stay in the **chunk index** (append-only history);
  chunk-grounding is **frontmatter provenance, not a graph edge**; note↔note
  links remain wikilinks (the Obsidian graph). **The episode layer was dropped**
  — chunks are the episodic layer.
- **Cosine cannot decide coverage.** A distilled L2 is a semantic abstraction,
  never the vector-centroid of its sources, so a cosine threshold systematically
  misfires. **Coverage is agent-judged** (covered / near / absent); cosine only
  nominates **top-K `candidate_l2s`** by centroid cosine. Nomination = recall
  (surface the true cover generously); precision = the agent.

Seven locked decisions (D1–D7): D1 one clustering over matched chunks+notes;
D2 `engram amend` (in-place relation-merge + provenance-merge + field-replace,
reusing the `resituate`/`migrate-links` DI; `learn` stays create-only, gains
`--chunk-source`); D3 exclude `Related to:` from the embed/`ContentHash` source
so link edits are cheap; D4 chunks-in-index-as-provenance (episodes dropped);
D5 append-only chunk history + per-chunk `IngestedAt` recency; D6 recency as a
first-class distillation weight (recent authoritative on conflict, old
uncontradicted retained; cross-cluster supersession a known gap); D7 agent-judged
coverage over top-K nominees. Recall writes are **blocking/inline**, superseding
the older fire-and-forget model.

This pass also retired the `transcript`/`episode`/`migrate-episodes` binary
surface (issue 649) — see ADR-0006/0008/0009 (Superseded) in `docs/architecture/adr.md`.

- Lazy-L2 design (the v2 model): `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md`
- Lazy-L2 plans: `docs/superpowers/plans/2026-06-10-lazy-l2-synthesis.md`,
  `2026-06-18-lazy-l2-synthesis-build.md`, `2026-06-11-real-skill-harness-rebuild.md`
- Phase-0 source review: `docs/superpowers/specs/2026-06-04-phase0-source-review.md`

## 8. Short-term memory: recency for L2 + usefulness activation (2026-06-16/17)

**Decision: recency in ranking is the gap — add note decay, a `LastUsed`
usefulness signal, and recency-weighted recall.**

The trigger: a fresh agent ran `/please`, looked up issues *it had authored
minutes earlier*, and remarked at how well-written they were — having forgotten
it was the author. The framing: **provenance falls out of recency** — surface the
recent raw transcript narration and re-reading it *is* the agent remembering.
The verification pass narrowed the symptom to **purely a ranking problem**:
`engram query` had zero recency (pure cosine); capture self-heals (recall's Step
0.5 re-ingests before querying); provenance needs no separate mechanism. Ten
approaches were evaluated against the code (CONTENDER/PARK); the contenders were a
recency-decay re-rank, guaranteed-recent inclusion, cursor replay, and a
SessionStart boot-hook.

This led to the **usefulness-activation** model: an additive `LastUsed` sidecar
field (no schema bump, hash-excluded → no re-embed); `engram query` emits a
read-only `activated` flag; a new `engram activate` command bumps `LastUsed`; L2
notes now **decay** with a most-recently-used floor band, mirroring chunks
(ACT-R feedback loop — regularly-useful L2s stay fresh, never-retrieved ones
decay out). This reversed v1's "recency on chunks only, notes pure cosine."
A related fix corrected the `engram activate` vault-path resolution.

- Approaches brainstorm: `docs/superpowers/research/2026-06-16-short-term-memory-brainstorm.md`
- Plans: `docs/superpowers/plans/2026-06-17-short-term-memory.md`,
  `2026-06-17-usefulness-activation-crystallization.md`,
  `2026-06-17-fix-activate-vault-path.md`
- Vault connectivity analysis: `docs/superpowers/research/2026-06-17-vault-connectivity-analysis.md`

## 9. `please` adversarial review gates + the `route` skill (2026-06-12/14)

**Decision: the `please` workflow stops trusting its own LLM-generated
artifacts — specs/plans, refactors, doc updates, and outward prose each pass an
adversarial review gate; and a `route` skill encodes a delegate-everything
doctrine `please` consults when staffing those gates.**

`please` gained four gates (A: plan, B: each REFACTOR, C: touched docs, D:
outward prose). Reviewers are fresh-context subagents, one per angle
(ask-alignment, code-alignment, docs/diagrams, clarity), each runs `/recall`
first, is prompted to *refute*, and is argued to consensus (escalate to the user
after ~2 deadlocked rounds). An angle whose subject is absent is skipped out loud
("N/A"); "the artifact is small" is never a skip.

The `route` skill makes the **delegate-everything** doctrine explicit: the
top-level agent only orchestrates (routes, decomposes, synthesizes) and does no
object-level work itself; easy work goes to a cheap model (haiku, not skipped),
complex work is decomposed before dispatch, deep thinking goes to a fresh
focused subagent, and every dispatched subagent recalls first. The router is
*advisory* — Claude Code can't let a skill change the main-loop model — so its
output is a decision the orchestrator encodes into its `Agent(...)` call. `please`'s
gate model-pins became router-overridable defaults. The rubric deliberately
aligns with `audit.md`'s model-selection doctrine so the repo has one routing
policy, not two.

- Adversarial-review design + plans: `docs/superpowers/specs/2026-06-12-please-adversarial-review-design.md`,
  `docs/superpowers/plans/2026-06-12-please-{adversarial-review-gates,anti-sycophantic-lean,skill-generalization}.md`
- Recall empty-vault create-band plan: `docs/superpowers/plans/2026-06-12-recall-empty-vault-create-band.md`
- Route skill design + plan: `docs/superpowers/specs/2026-06-14-route-skill-design.md`,
  `docs/superpowers/plans/2026-06-14-route-skill.md`
