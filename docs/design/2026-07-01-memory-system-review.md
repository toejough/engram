# Memory-system review — goals, evidence, landscape, and what's next

**Ask (Joe, 2026-07-01):** review our conversational history, our docs, what we've built, and competing
ways to capture, distill, recall, refine, and build upon memories in an LLM ecosystem; report how well
or poorly we've achieved our goals and what else to explore.

**Method.** Five parallel research passes: (1a) goals + build trail from README/ROADMAP/21 dated design
docs/git; (1b) eval-verdicts table from the vault ledger (notes 99, 95, 98, 91, 103, 135, 73, 76, 68,
82, 119, 120, 149–154) with numbers taken from the design docs; (1c) conversational history — direct
chunk-index queries + a 6-transcript sample (2026-06-08 → 2026-07-01); (2a) external systems survey;
(2b) external techniques survey. External comparison is **architectural, not benchmarked** — we did not
run competitors. Superseded results are shown post-correction. **Measurement vintage:** the system
changed under the evals — payload cuts 06-27, note-floor 06-28, glance 06-29, decision-moments
guidance 07-01 — so each number binds the system state at its date; rows whose vintage affects the
conclusion carry an explicit **Vintage** flag (the pre-guidance 77% timing baseline, the pre-cuts
C1/C2 magnitudes, note-82's pre-floor evidence). Transcript corpus caveat: session logs
are a retained store; nothing predates 2026-06-08, so the earliest transcript is not founding intent.

---

## 1. Goals — what we set out to do

The founding mission, stated in the earliest available transcript and never elevated to a user-facing
doc until now (transcript-only, 1c):

> **"When I give a correction, I don't want to have to give it again."** — the say-once goal
> (2026-06-08, `ee8329d2` — also: "The user should not have to repeat lessons; the LLM should not
> waste cycles re-exploring known-bad patterns or re-searching for information it already learned.")

The dated goal trail, with evolution:

| # | Goal | First stated | Evolution |
|---|---|---|---|
| G0 | **Say-once** — a correction given once is applied thereafter | 2026-06-08 (transcript; `OPUS-TRAP-CATALOG` north star) | Never retracted; the whole project serves it |
| G1 | Persistent memory via a zettelkasten vault + linked concepts across time/project/subject axes | 2026-06-08 (issue #637 ask; README) | Original ask was **link enrichment for unanticipated queries** ("we can't enumerate all possible queries at learning time"); pivoted user-steered mid-session (2026-06-08) into *measure whether memory helps* — the eval harness era |
| G2 | Memory makes builds **cheaper and faster, net** | 2026-06-24 (`2026-06-25-memory-cost-reanchor.md`) | Refuted on easy builds 2026-06-25; re-anchored to "achievable on harder builds — unproven." The deeper framing (transcript-only, 2026-06-25/07-01): cost work is an **enabler of firing recall more frequently and organically**, not an end |
| G3 | The right note surfaces when queried (retrieval quality) | 2026-06-28 (`retrieval-probe-results.md`) | Solved — note-floor at embedder ceiling |
| G4 | Memory-backed reasoning routes to cheaper model tiers | 2026-06-28 | Achieved (route skill) |
| G5 | Recall fires **at the right moments**, not only task-init | 2026-06-27 (`recall-trigger-patterns…`) | The live frontier (Track A) |
| G6 | Notes worth surfacing (crystallization quality) | 2026-06-28 (`crystallization-audit.md`) | Every proposed lever measured null on delivery; see §4 |
| G7 | Compounding: the vault accumulates higher-order knowledge | 2026-06-23 (`compounding-eval.md`) | Refuted at tested depths (Δ=0); parked |

## 2. What we built

The ask's five verbs map onto the shipped system (C1 architecture: four flows — recall, learn, please,
update) as follows; the report uses the verbs only through this mapping:

| Ask verb | Shipped mechanism |
|---|---|
| **Capture** | `engram ingest --auto` — mechanical chunk+embed of every session transcript + repo markdown (append-only); learn Step 1 sweeps it each bracket |
| **Distill** | learn Step 2 (correction-driven notes) + recall Step 2.5 (cluster-driven coverage judgment: covered/near/absent → amend/learn); embed-on-write sidecars |
| **Recall** | recall skill (10-phrase deep / 3-phrase glance) → one unified `engram query` (notes + chunks, cosine + recency, AutoK clustering, note-floor, `candidate_l2s` inline); glance/deep depth dial; recall-at-decision-moments guidance (`~/.claude/engram/recall.md`) |
| **Refine** | `engram amend` (relation/provenance merge, field replacement), `resituate`, `activate` (ACT-R recency), recency-weighted conflict resolution (2.5B), flock + atomic writes (Track 0) |
| **Build-upon** | recall Step 2.6 precision-gated cross-cluster linking; Step 3/4 synthesis persistence; please's 7-step gated orchestration consuming recall output |

Nineteen mechanisms shipped with commits (full trail: `git log` + ROADMAP §Done/§Shipped; highlights): chunk layer + auto-ingest
(2026-06-11/20), flat vault (`ec7da40`), please gates (`be2ae8ae`), route (`7f5a0322`), ACT-R recency
(`d2bbb163`), $METER + trap gate (`7c2f2190`/`e9daf8c0`), payload cuts −58% (`5e92fe57`+`50fbfcf3`),
matched-note floor (`33821e64`), tier-routing (`2bf959f4`), glance/deep (`bdb8b0dc`), decision-moments
guidance (`120bb080`/`25093021`), concurrency/write-safety (`f7f6b389`).

## 3. Goals scorecard

Verdicts strictly from the adversarially-verified ledger; every number carries its source. UNMEASURED
rows state what's missing.

| Goal | Verdict | Evidence (metric, n, date, source) |
|---|---|---|
| G0 say-once — **capability half**: memory carries what cold opus cannot derive | **ACHIEVED** | C3 conventions (5 traps × 5 trials): cold applied 0/25 → warm applied 25/25; C4i supersession 0/5 → 5/5; C5 recency-standard 0/5 → 5/5; C6 abduction cold 1/9 → warm 18/18 (2026-06-23, `traps/RESULTS.md`); holds under a 200-note real-vault crowd, Δ=0 (2026-06-26) |
| G0 say-once — **timing half**: the lesson fires at the moment it's needed | **PARTIAL** | 77% of 137 mined real failures sit at decision cues recall never reaches; 56% are application (rule present, unapplied) (2026-06-28, `failure-eval-material.md`). **Vintage: this is the PRE-guidance baseline** — the mined sessions all predate the 2026-07-01 decision-moments guidance, which was then validated to fire at its three cues (headless RED 0/5 → GREEN 4/5; before-declaring-done alone ≈27% of the uncovered set); its effect on the real distribution is unmeasured (re-mine post-guidance sessions once a sample accumulates), and the 56% application share is only partially reachable by firing (glance C5: surfaced 5/5, honored 0/5 — surface ≠ apply). Live example: the labeled-table correction was repeated by Joe mid-session *while building this system* (2026-06-24, transcript `a19e7b75`). Hooks/checkpoints unbuilt |
| G1 persistent substrate + retrieval | **ACHIEVED** | Real-path note recall@5 0.22 → 0.83 = embedder-isolation ceiling (note-floor `33821e64`, 2026-06-28); diagnostic symptom→cause probes 0.99 recall@5 (n=75) |
| G1 original link-enrichment ask (unanticipated-query graph) | **UNMEASURED / parked** | Graph-*expansion at recall* was built + reverted (0 marginal value, 2026-06-23); but the original 2026-06-08 ask — links formed at learn time so unanticipated queries land — was never directly evaluated; `internal/vaultgraph` remains unused (ROADMAP "relational synthesis" long arc). Missing: an eval where cosine fails and only a persisted edge saves it |
| G2 cheaper/faster builds (easy regime) | **REFUTED** | Warm +182 s (d/SE 3.2) and +$3.08 (d/SE 5.1, loses 7/7) vs cold, n=8/arm, opus, 2-round CRUD (2026-06-25, `warm-vs-cold-clean-measurement.md`). **Vintage: pre-payload-cuts (−58%), pre-glance, pre-tier-routing — magnitudes are stale-high; the sign is untested since, though the build-phase payload premium it rides is still unpruned in production** |
| G2 cheaper/faster builds (hard multi-round regime) | **UNMEASURED** | The regime where memory should pay (warm converged 2 rounds 8/8 sd=0 vs cold scatter to 4–5) was never run; Joe's cross-repo harder-builds direction (2026-06-25, transcript) partially executed (failure mining done; builds not). Missing: the harder-builds arm |
| G2′ cheap enough to fire organically | **PARTIAL** | Recall corrected to ~190 s (350 s was a mislabel — round-1 included the first build; 2026-06-25 isolation doc); glance = 42 s / $0.42 vs deep 94 s / $0.78, 2.23× faster, −46% $/fire, n=5/arm real vault (2026-06-29); payload-prune smoke: synthesis-only carry cut build cost −40% ($12.26→$7.39, n=3 apps, capability flat; 2026-06-30) — production form unbuilt (ROADMAP Track B NEXT) |
| G3 retrieval ranking | **ACHIEVED** | Floor at ceiling (above); trap gate C3–C6 GREEN through every subsequent change |
| G4 tier democratization | **ACHIEVED (3/4 axes)** | Sonnet-warm = opus-warm: C3 15/15, C4i 3/3, C6 6/6; sonnet-cold fails C4i/C6; ~25–30% cheaper/trial; C5 inconclusive (opus baseline flaked 0/3) (2026-06-28; note 135; shipped as route tier-routing `2bf959f4`) |
| G5 right-moment firing | **PARTIAL (live frontier)** | Guidance shipped + validated headless (RED 0/5 → GREEN 4/5, 2026-07-01); but the moment-coverage baseline is 9% covered / 77% uncovered, and cue-based levers partially failed (after-failure cue fired 0/2; value-gate rejected — opus over-classifies; #665 open) |
| G6 crystallization quality | **REFUTED levers; gap real but delivery-inert** | Wording gap confirmed (cluster-driven 40% vs correction-driven 79% question-useful, n=98 notes, 2026-06-28) — but the handle prose rule RED-baseline passed (5/6 without it); question-anchoring PARKED (delivery B−A = −10 pp inside 2σ ±22 pp; retrieval lost 10/10, mean cosine 0.52 vs 0.35; 2026-07-01). The concrete idiosyncratic token is load-bearing (note 153) |
| G7 compounding / emergent synthesis | **REFUTED at tested depth** | Synthesis scaffold Δ=0 (warm-only already 18/18); persistence at depth-2 Δ=0 (6/6 = 6/6) (2026-06-23); third consecutive reasoning-scaffold null (note 73) |
| Behavioral value on real (non-idiosyncratic) work | **UNMEASURED — structurally hard** | Behavioral traps don't reproduce in clean toys (cold opus 0/5 correct behavior everywhere, 2026-06-23+); cheap build evals are structurally biased against showing it (note 98). Missing: a rich-context harness |

**Scorecard summary: 4 ACHIEVED, 3 PARTIAL, 3 REFUTED, 3 UNMEASURED.** The refutations are load-bearing
successes of the method: each was a cheap kill of a plausible lever before it shipped.

## 4. Honest gaps

1. **The say-once loop is not closed — and the failure is timing, not memory.** The capability exists
   (C3–C6 clean flips); yet 77% of real failures sit at moments recall never fires, and 56% are
   *application* failures where the rule was even present. The sharpest evidence is reflexive: Joe had
   to repeat the labeled-table correction during the very session that was building the memory system
   (2026-06-24). The lesson existed; nothing fired it at the moment of writing the table.
2. **Net value on ordinary work is negative; the winning regime is unproven.** Memory is a measured tax
   on easy builds (+182 s, +$3.08) and a verified capability only where content is idiosyncratic. The
   "harder builds" regime — where 2-round-vs-5-round convergence should dominate the ~186 s overhead —
   was designed (Joe named the cross-repo corpus 2026-06-25) but never run.
3. **Recall fires once, structurally.** A lever invented mid-synthesis is never re-checked against the
   vault, so known-refuted ideas resurface as fresh (note 82; filed #654/#655). The please/route
   reviewers' recall-first rule patches the review path, not the author path. **Vintage: the 2026-06-24
   diagnosis had three legs, and one — notes drowned by chunks — was FIXED by the note-floor
   (2026-06-28); the fires-once leg is architectural (still true by code) but the miss has not been
   re-reproduced post-floor. #654's C7 harness is exactly that re-test.**
4. **Crystallization quality resists improvement.** Every quality lever nulled on delivery
   (handle-wording, question-anchoring, synthesis persistence, graph expansion). The one durable
   positive lesson is negative-space: keep the concrete token; don't abstract (note 153).
5. **The original unanticipated-query ask is still open.** Precision-gated linking (Step 2.6) ships,
   but nothing measures whether persisted edges ever *save* a recall that cosine alone would miss —
   `vaultgraph` sits unused. Note 68's aggregation-vs-synthesis boundary stands unaddressed.
6. **Refine is thin.** Recency-weighted conflict resolution works at recall time (C4i 5/5), but there
   is no vault-wide consolidation: no dedup pass, no contradiction sweep, no decay-to-prune. At 154
   notes this is invisible; at 1,000 it may not be. (#648 constants never tuned; #659 prune bug open.)
7. **Evidence bounds.** Single user, single ecosystem (Claude Code + opus/sonnet), small n (5–8/arm
   typical), and the capability wins ride idiosyncratic *fictional* traps by design (real conventions
   risk model priors — the C4 confound taught us that). Pass-bars are model-specific (note 146).

## 5. External landscape

Surveyed 2026-07-01 (Task 2a/2b; primary sources; vendor numbers labeled). Per-verb comparison —
"none" is an explicit finding:

| System | Capture | Distill | Recall | Refine | Build-upon |
|---|---|---|---|---|---|
| **engram** | auto-ingest (mechanical, append-only) | agent-judged, two paths (correction + cluster coverage) | agent-initiated skill; unified cosine+recency query; glance/deep | amend/resituate/activate; recall-time recency wins; flock-safe | precision-gated typed linking; gated orchestration (please) |
| **MemGPT/Letta** | agent-decided tool calls | none (direct storage; sleep-time reorganization) | agent-decided 3-tier search (core/recall/archival) | agent-initiated replace; consolidation passes | none native (agent reasoning only) |
| **Mem0** | framework-automatic per message | LLM extract + dedup pipeline | 5-stage app-initiated (vector + entity + temporal rerank) | keep-both-with-timestamps; flagged conflicts | entity graph (paid tier) |
| **Zep/Graphiti** | framework-automatic episodes | LLM triple extraction → bi-temporal knowledge graph + community summaries | hybrid BFS + cosine + BM25 + rerank | temporal edge invalidation (new wins, old kept) | graph traversal; community clustering |
| **LangMem** | app-initiated (+ background reflection) | LLM → typed memories (semantic/episodic/procedural) | auto embedding search | upsert-not-duplicate; opt-in deletes | **procedural: rewrites its own system prompt** |
| **ChatGPT memory (Dreaming V3)** | automatic background synthesis over all history | opaque LLM curation; temporal self-revision | none at query time — pre-injected every session | self-updating + user edit page | none documented |
| **Claude Code stack (CLAUDE.md / auto-memory / claude-mem)** | human-authored / agent-decided / hook-passive | none / none / background compression (~10× claimed) | always-loaded / index+grep / vector+keyword at SessionStart | manual / in-place agent edits / none | none |
| **Cognee** | app-initiated multi-format ingest | 6-stage LLM pipeline → graph+vector | 14 modes; vector entry → graph traversal | `memify` prune/reweight pass | graph as first-class composition substrate |

Technique highlights (full 13-technique survey: `artifacts/2026-07-01-memory-review-techniques-survey.md`;
full systems survey: `artifacts/2026-07-01-memory-review-systems-survey.md`): sleep-time compute (arXiv 2504.13171 —
paper-claimed 5× test-time compute reduction via offline pre-computation); ACE/agentic context
engineering (arXiv 2510.04618 — Generator/Reflector/Curator playbook loop, paper-claimed +10.6%
agentic); HippoRAG (arXiv 2405.14831 — PPR spreading activation over an extracted KG, paper-claimed
+20% multi-hop QA); Reflexion (within-task failure verbalization); Generative-Agents importance
scoring (write-time LLM rating × recency × relevance); MemoryBank Ebbinghaus decay; A-Mem (2502.12110
— LLM-linked zettelkasten, architecturally engram's nearest neighbor).

**Where engram stands out against the field:**
- **Adversarial self-evaluation is our genuine differentiator.** No surveyed system publishes whether
  memory beats what the model derives cold, what fraction of stored memories are ever used, or its
  injection false-positive rate; the 2026 "Are We Ready" critique (arXiv 2606.24775) makes exactly
  this point, and the one independently-measured vendor (Mem0) shows a 25–45-point gap between claimed
  and independent numbers (LongMemEval 94.4% vendor-claimed vs ~49% independent). Our ledger —
  cold-vs-warm flips, refuted levers, corrected mislabels — is the discipline the field lacks.
- **Skills+binary split** (LLM judges, binary computes, local-first, no services) is cleaner than any
  surveyed architecture; A-Mem approximates it with a heavier stack.
- **Precision-gated linking** (default-DROP, hub test) has no counterpart; the field either floods
  (auto entity graphs) or doesn't link at all.

**Where the field does things we don't:**
- **Framework-automatic / passive capture triggers** (Mem0, Zep, Dreaming, claude-mem hooks) — our
  capture is mechanical-on-sweep but our *moments* are agent-judged; the field's determinism is a
  costless trigger, which is exactly the property our over-fire analysis said hooks need.
- **Bi-temporal validity + graph-native supersession** (Zep) — we resolve conflicts at recall time;
  they persist validity intervals at write time.
- **Procedural memory** (LangMem) — an agent that rewrites its own standing instructions; our CLAUDE.md
  equivalent is human-authored.
- **Offline consolidation** (Dreaming, sleep-time, Letta) — we have no between-session pass.

## 6. What to explore next — ranked, evidence-gated

Deflation risk = probability the lever nulls on the deciding delivery metric, given our base rate
(every crystallization-quality lever deflated; timing/capability levers have fared better).

| # | Candidate | Gap addressed | Evidence for headroom | Cheapest validation | Deflation risk | Collision check |
|---|---|---|---|---|---|---|
| 1 | **Payload-prune production build** (subagent-isolated recall) | G2′ cost; §4.2 | Smoke-validated: build cost −40%, capability flat (n=3, 2026-06-30) | Already designed; build + C3–C6 gate + $METER | **Low** (mechanism verified, mechanistic per-round explanation) | ROADMAP Track B "← NEXT" — this IS the roadmap next; not new |
| 2 | **Decision-moment checkpoints as deterministic hooks** — before-declaring-done recall (≈27% of uncovered failures) + after-tool-failure PostToolUse | G0-timing, G5; §4.1 | 77%-uncovered baseline (**pre-guidance vintage** — re-mine post-guidance sessions first to size the residual gap); hooks escape the over-fire ratio bound (free deterministic trigger vs 190 s agent fire — the "before tool calls" 147× analysis); glance at 42 s makes per-fire cost tolerable | Headless RED/GREEN per the recall-moments method (fictional domains, ~$10–20) | **Medium** (guidance GREEN 4/5 is promising; but after-failure cue fired 0/2 — moment-specific) | Track A live; #665 open (value-gate); the hook variant is unfiled — genuinely new |
| 3 | **Recall-before-recommend re-entry** (Reflexion-shaped: re-check mid-synthesis levers against the vault before recommending) | §4.3 structural miss | Note 82 RCA: the miss is timing+phrasing, and the disproving note EXISTS each time; C7 harness designed | Build #654's C7 RED harness first (reproduces the re-proposal), then the re-entry step | **Medium** | Filed: #655 (fix) + #654 (harness) — cite, don't re-propose; this is "schedule the filed work" |
| 4 | **Harder-builds eval** (cross-repo corpus: spaced-repetition, file-sync, spec-review histories) | G2-hard UNMEASURED; §4.2 | Warm round-convergence signal (2 rounds 8/8 sd=0 vs cold 4–5 scatter); Joe specified the corpus 2026-06-25 (transcript; partially executed — mining done, builds not) | 2–3 hard multi-round builds × warm/cold with the $METER (~$40–80) | **Medium-high** (the easy-build result went the wrong way; this could too — but it closes the thesis either way) | Direction documented in failure-mining docs; the builds themselves unfiled |
| 5 | **Between-session consolidation pass** (sleep-time-shaped: batched dedup/contradiction-sweep/decay over the vault) | §4.6 refine-thin | Field-wide pattern (Dreaming, Letta, SCM); paper-claimed 5× amortization (2504.13171) | ONLY with a total-token gate: Joe rejected async-learn because relocation ≠ reduction (2026-07-01, transcript) — the pass must REDUCE total spend (e.g. replace N per-session crystallizations with one batch) or shrink recall payloads measurably; pilot on the 154-note vault, measure $ + trap gate | **High** (relocation trap; vault may be too small to show value) | Async-learn explicitly deferred in ROADMAP §Dead ends — this differs only via the token gate; treat as re-opening a park, requires the NEW fact (total-token reduction) to be demonstrated first |
| 6 | **Edge-payoff probe for the link graph** (does a persisted 2.6 edge ever save a recall cosine misses?) | §4.5 unanticipated-query; note 68 | Un-measured (never evaluated as built); Tier-1-style free retrieval probe possible | Free: replay real queries with links stripped vs present; count rank changes | **High** (graph-expansion already 0-value at recall; this tests the WRITE side, a different claim — but the base rate is bad) | ROADMAP "relational synthesis (note 68)" long arc — this is its cheapest first step |
| 7 | **Write-time importance scoring** (Generative-Agents-style) | ranking follow-ups | None here — ranking is at ceiling; only revisit if a chunk-quality gauge shows drowning returns | n/a until gauge exists | **Very high** | ROADMAP "ranking follow-ups — only if the floor proves too blunt"; parked |

**Explicitly not recommended:** rerankers/stronger embedders (solved axis; three A/B nulls); GraphRAG
community summaries (static-corpus design, heavy ingest; HippoRAG-style PPR is the cheaper probe and
even that is #6 with high risk); MemoryBank decay (recency already shipped; scale problem we don't
have); A-Mem adoption (we are its architecture, leaner); bi-temporal graph rebuild (C4i already 5/5 at
recall time — no measured failure to fix).

## 7. Bottom line

We set out to make corrections stick (say-once), and built something the field genuinely lacks: a
memory system whose value claims are adversarially verified rather than vendor-claimed. The substrate
works — capture is mechanical and comprehensive for Claude Code sources (OpenCode pending, #644),
retrieval is at its embedder ceiling, capability on idiosyncratic content is
a clean, crowd-robust, tier-democratizing win, and the cost curve finally bends (glance, tier-routing,
payload-prune smoke). What we have *not* done is close the loop the mission names: the lesson too often
exists but does not fire at the moment of decision (77% uncovered; the labeled-table correction
repeated mid-build), and memory's net value on ordinary hard work remains unmeasured — refuted on easy
work, never tested where it should win. The next quarter of effort belongs to timing (deterministic
decision-moment hooks, re-entry) and to closing the harder-builds measurement — not to more
crystallization polish, which five consecutive evals say is already good enough.
