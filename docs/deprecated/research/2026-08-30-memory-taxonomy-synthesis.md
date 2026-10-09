> Landscape research (docs/README.md `design/`-charter sibling rule: conclusions graduate to ADR/specs/ROADMAP; file is deletable once extracted). Produced 2026-08-30 by the memory-taxonomy research workflow (4 parallel researchers + critic), explore session on #735 → memory north star. Synthesis + decisions D1-D6 context (authoritative record: ADR-0027).

# Engram × Human Memory: North-Star Synthesis

*2026-08-30 — synthesis of 4 research briefings (human memory science, LLM agent memory systems, procedure composition, engram inventory) + adversarial critic pass. Raw briefings in session scratchpad.*

---

## Verdict up front

**No redesign needed.** The human episodic/semantic/procedural taxonomy is now the *mainstream organizing frame* of the LLM-agent-memory field (2025–26 surveys use exactly it), and engram already implements more of it than any published system implements of itself. The taxonomy's correct role — per the field's own evidence — is a **coverage checklist, not an architecture**: no published ablation shows that splitting the store by memory type causes wins. What the checklist reveals for engram is five genuine gaps, and every one of them maps onto an issue you already have open or a mechanism you already half-built. The north star doesn't demand new architecture; it demands finishing the pipeline you have, in a specific order.

**One sharp upgrade to the mental model:** a runbook is *not* procedural memory in the strict sense. Human procedural memory is compiled and automatic — you don't retrieve a description of riding a bike. A retrieved runbook is **declarative memory about a procedure**: the agent is permanently in the "novice reading the recipe" stage (Fitts & Posner's cognitive stage). Only two things in an agent stack are procedural in the strict sense: **model weights** (compiled by pretraining) and **executable code** (skills' scripts, hooks). This is not pedantry — it *predicts* the field's best-documented failure mode ("retrieved but not applied," now a named benchmark category), which is exactly #738's link (c). Keep the name "runbook"; keep the distinction in view.

---

## 1. The expanded taxonomy (what "matching human memory" actually means)

Your three categories are right but incomplete. The full set of human memory *functions* an agent system can mirror:

| Function | What it is in humans | What it is for an agent |
|---|---|---|
| **Episodic** | This happened (event + context, "I was there") | Raw session transcripts, timestamped, verbatim |
| **Semantic** | This is what it means (decontextualized facts/rules) | Crystallized notes: facts, lessons, decisions |
| **Procedural** | How to do it (compiled, automatic, non-declarative) | Strictly: weights + executable code. Runbooks = declarative *descriptions* of procedures |
| **Working** | What's active right now (~4 chunks, attention-limited) | The context window; payload budgeting is WM hygiene |
| **Prospective** | Remembering to remember (act when cue X occurs) | The firing problem: when does recall get invoked at all |
| **Consolidation** (process) | Episodes → gist/rules, via offline replay; originals kept | Chunks → notes crystallization |
| **Reconsolidation** (process) | Retrieval destabilizes a memory; it's re-stored, possibly updated — gated by *prediction error* (surprise) | Update-on-use: amend when reality mismatched the note |
| **Forgetting** (process) | Not decay — retrieval *competition* (cue overload); adaptive need-probability tracking | Ranking demotion + supersession suppression, not deletion |

Three load-bearing findings from the human literature (not decoration — each makes a testable engineering prediction):

1. **Encoding specificity (Tulving & Thomson):** retrieval succeeds when the cue at retrieval matches what was encoded. This is *why* the `situation:` field works, why idiosyncratic tokens proved load-bearing in the qanchor eval (abstracting them away destroys encoding-specific cues — the literature predicted your result), and why runbooks must be keyed by the *situation the agent will be in*, not the topic of the solution.
2. **Prospective memory multiprocess theory (McDaniel & Einstein):** intentions fire either by *strategic monitoring* (check constantly — taxes every step; your 147× over-fire measurement is exactly this signature) or by *focal cues* (the cue is part of what you're already processing — near-zero cost). The strongest applied result in all of psychology's PM literature: **implementation intentions** (Gollwitzer) — "when X, then do Y" wording with a concrete cue dramatically beats "remember to Y." Your vault note 137 (name the action, not the purpose) independently rediscovered this.
3. **Two-variable strength (Bjork & Bjork):** storage strength (how well learned — never decreases) vs retrieval strength (current accessibility — recency/frequency). A single decay scalar conflates them. A verified lesson from 2025 is still *true*; it should lose *rank*, not confidence. ACT-R's activation (log of power-decayed use history) is the reference design — `last_used` as a single date (10 uses = 1 use) is a crude version of it.

**Where the analogy breaks (don't import these):** no weight updates at inference → no automatization path exists in-session, ever; the pretrained model already *is* a colossal semantic+procedural memory (so recording what it already knows adds interference — your C1–C6 ledger measured exactly this); human forgetting is substrate-enforced, yours is a policy choice; retrieval for humans is free/always-on, for agents it's a costly explicit action — which is why prospective memory is THE load-bearing problem for agents in a way it never is for humans.

---

## 2. Engram mapped (mechanism → analog → fit)

| Engram mechanism | Analog | Fit |
|---|---|---|
| Chunk index (append-only turn-anchored transcripts) | Episodic | **Strong** |
| fact/feedback notes + wikilinks + Luhmann tree | Semantic | **Strong** |
| Supersedes edges (updates/narrows/refutes) + recency-wins rule | Belief revision | **Strong** |
| recall 2.5C crystallization + learn moments (chunk-sources back-links) | Episodic→semantic consolidation | Partial→Strong (pull-triggered only; no background pass) |
| amend-on-recall (covered/near) + resituate | Reconsolidation | Partial (deep-mode only; glance — the mode meant to fire often — is read-only) |
| runbook notes (situation/steps/done_when) | Procedural (declarative form) | Partial — by design; see verdict |
| route evidence loop (dispatch records → recalled evidence sets tier) | Procedural *learning* | **Strong** — the truest closed skill-learning loop in the system |
| activate/last_used + 60d half-life rank decay | Retrieval strength | Partial (date not count; no frequency) |
| Guidance cue lists (recall.md etc.) | Prospective memory | **Weak — the biggest gap** |
| Payload budgets, lazy-chunks, recent-fill | Working-memory hygiene | Partial |
| Explore channel (vocab-centroid sampling) | Spreading activation / priming | Partial |
| curate + pending offers | Social memory gating (testimony evaluation) | Strong |
| Vocab lifecycle (geometry-derived terms) | Schema/category formation | Partial→Strong |

**Functions with no analog at all:** composition of multiple procedures (see §4); forgetting of note content (nothing is ever suppressed — superseded notes still surface as ride-alongs); background consolidation over never-queried episodes; skill promotion/demotion pipeline (hot runbook → always-loaded; cold skill → vault note); per-note frequency/outcome strength (#718).

---

## 3. What the field proves (and doesn't)

**The proven corner — and you're standing in it.** Mining procedural/workflow memory from experience is the *best-replicated positive result in agent memory*: AWM (+51% relative on WebArena; **mined workflows beat human-authored ones by 7.9%**), Memp (procedural memory as build/retrieve/update lifecycle; **strong-model-built procedures transfer to weaker models** — your tier-routing result, independently confirmed), ReasoningBank (harvest failures too — your learn skill's confirmed-negative rule), Voyager (executable skills, composition in code space), Dynamic Cheatsheet (one retrievable artifact: 10%→99%). These are task-success metrics, not QA-recall proxies.

**Also proven:** curation beats accumulation (add-all store fell to 13% accuracy vs 39% curated — mechanism: agents *replicate the quality of whatever they retrieve*, so bad entries propagate); working-memory management (context editing: 84% token cut); progressive disclosure (Claude Skills' index-always/body-on-demand); temporal invalidation for facts (Zep's bi-temporal edges); always-on injection mechanically solves firing for small content.

**Fashionable but unvalidated:** taxonomy-mirroring as an architecture (LangMem/MIRIX ship your taxonomy as SDKs; zero ablations showing the split itself wins); graph memory as a category win; LLM-scored importance at write time; automatic self-evolution of memory without regression gates (documented rot: "context collapse," rationale erosion — engram's *judged* curation is the safer variant); conversational memory leaderboards (vendor-war numbers don't survive correct competitor configuration).

**Where engram is ahead of the field:** evaluation honesty (headless RED/GREEN, treatment-delivery gates, noise floors — stronger than published norms); the only quantified firing-cost analysis anywhere (147×/380×/7× at pinned fire-units); judged curation; the offer/curate multi-vault trust model.

**A scope limit the critic caught in your own doctrine:** "memory only pays on idiosyncratic content" was verified on *facts and conventions* (C1–C6). The field's procedural wins are mostly on *generic* procedures — content the model could derive but expensively (AWM's web workflows, a Game-of-24 solver). The idiosyncratic-only rule may not bind for runbooks. That's testable inside #737, and it matters: it decides whether runbooks should capture only house procedures or also "the usual dance" everyone knows.

---

## 4. The composition question (your TDD + Go + remote-env goal)

The single most useful result: **classify the procedures, because the three classes compose differently.**

- **Constraint-like** (Go standards, lint rules, "wrap errors with %w") → compose by **conjunction**. Like non-colliding CSS properties, they blend safely. No machinery needed.
- **Plan-like** (remote-env steps, release steps) → compose by **sequencing/interleaving**. Need ordering decisions at the seams.
- **Same-slot** (two different "commit" procedures) → need **arbitration or supersession**. Blending is the failure mode — and LLMs fail it *silently*: benchmarks (ConInstruct) show models detect instruction conflicts but rarely say so unprompted; they just pick one. You don't get an incoherent blend; you get an unflagged coin-flip.

TDD + Go + remote-env is mostly classes 1+2 — which is why in-context blending (what happens today when several notes surface) mostly works, and why no published system does better: **explicit read-time composition of prose procedures is a genuine open problem — nobody has shipped it.** Voyager composes only in code space (functions calling functions); AWM/Memp/LEGOMem all retrieve-and-dump.

The proven, cheap mechanisms all live at **write time and retrieval time** — exactly the points a zettelkasten controls:

1. **Supersession at curation time** (engram has 80% of this; the missing 20% is retrieval *suppressing* superseded notes rather than ride-along-surfacing them).
2. **Scope-specificity ordering** ("this repo" beats "Go projects" beats "everywhere"; ties → newer) — the one primitive frontier models were literally *trained* to respect (instruction-hierarchy training).
3. **A conflict-surfacing rule in prose** (SOAR's impasse): "if two retrieved procedures prescribe incompatible actions for the same step, say so and adjudicate — never silently pick." Near-zero cost; directly targets the measured silent-pick failure.
4. **Chunk the resolution**: after any adjudicated conflict, write the merged procedure as a new note superseding the parts (this is just /learn + supersedes — SOAR's chunking, ACT-R's compilation, and human skill-chunking all converge on it).
5. **Compose-then-critique** (NOAH's critics): for multi-runbook plans, draft the combined plan, then one targeted pass for clobbered preconditions / duplicate steps / ordering conflicts — a review gate, and `please`'s gates are already the chassis.

What's *not* warranted: numeric utilities, formal preference calculi, read-time auto-merge. No evidence LLMs respect fine-grained priorities, and the common case needs nothing.

One tension the critic surfaced, resolved: the human briefing says two cognitive-stage procedures in one context is the worst dual-task case (delegate each to a subagent — delegation is the only automaticity substitute an agent has); the composition briefing says blending usually suffices. The reconciliation: **blend different-concern procedures in one context; delegate or arbitrate same-slot ones.** The class, not the count, decides.

---

## 5. What this resolves for #735 (the framing fork)

The prospective-memory literature answers the recurring-vs-situational question directly:

- Event-based PM works by **cue-noticing**, and the cue that works is **focal** — part of what the agent is already processing. At task start, the agent is by definition processing *the task's situation*. "Do I recognize this situation?" is focal. "Have I done this before?" is not — it demands a memory search to decide whether to search memory (strategic monitoring — the architecture that measured 147× over-fire).
- Encoding specificity says the same thing from the write side: runbooks are keyed by `situation:`, so the firing cue must be situation-shaped for cue and key to meet. The 27 real runbooks confirm it — the corpus is dominated by situational binds ("OrbStack DNS fails," "deleting a dead Go symbol"), not named routines.
- Cue wording should be an **implementation intention**: "When you're about to start a multi-step task — run `/recall glance` first; the vault may hold a runbook for exactly this situation." Concrete cue, named action (note 137), no value gate (notes 144/145: under-firing is the risk; worded gates don't hold; glance is cheap by design).
- "Recurring routine" (the widget-release shape) is a *special case* of situation-recognition and can be a positive example inside the cue. Fixtures should still span both shapes so the eval measures the general cue, not the special case.
- Over-fire budget: fire-unit pins at task-init (~7× on your own data — tolerable per runbook 824), and glance is the cheap rung built precisely so over-fire doesn't matter (note 144).

Also relevant to #736 (the apply link): the field has named "retrieved-but-not-applied" as its dominant failure mode and its best mitigation is **decision-point injection** — the matched runbook goes into the working prompt at task start as instructions, not as reference material. Worth keeping in #736's option set.

---

## 6. The coherence picture: gaps → work, in dependency order

```
Human-memory function      Engram gap                        Existing home
─────────────────────      ──────────────────────────        ─────────────
Prospective (firing)    →  no task-start cue                 #735 (live)
Apply (the 166-watch)   →  follow/verify unmeasured          #736
Procedural value        →  runbook-vs-original A/B           #737 (+ idio-vs-generic scope probe)
Strength/forgetting     →  date-not-count; no suppression    #718 (+ new: superseded-note suppression)
Composition             →  absent (and unsolved anywhere)    new — after #737 proves the kind
Consolidation (replay)  →  pull-triggered only               new — background/batch pass (evidence thin; low priority)
Promotion/demotion      →  no runbook↔skill tier migration   #728 probes one direction
WM management           →  budgets only                      mostly fine; lowest priority
```

Plus three cross-cutting items the critic flagged that nothing currently owns: **memory poisoning** (ingest --auto persists raw transcripts unjudged; serve accepts external offers — trust tiers/quarantine are unaddressed), **multi-agent memory routing** (recall fires in the orchestrator but subagents do the work; subagent-discovered lessons die with their discarded contexts — a real gap given the delegate-everything doctrine), and **hybrid lexical+dense retrieval** (MiniLM is weak at exact rare-token matching; verbatim idiosyncratic tokens are the proven value carrier; a BM25-style lexical channel is the standard fix and a cheap probe).

---

## 7. Tensions worth an explicit decision (not resolving them here)

1. **Ride-along supersession vs suppression.** Engram deliberately re-surfaces superseded notes (belief-revision transparency). Three independent lines — cue-overload theory, the composition shortlist's #1 mechanism, and the experience-following result (deleting outdated entries drove the 39%-vs-13% win) — all vote for *suppressing* superseded content at retrieval instead. Keep the history, stop serving it.
2. **Idiosyncratic-only vs generic procedures.** §3's scope limit. Decides runbook capture policy.
3. **Update-on-use breadth.** Reconsolidation says amend-on-mismatch *in the turn the mismatch happens* — but glance (the frequent mode) is read-only by design, and auto-evolution is a documented rot vector. Where between "deep-only" and "every glance can amend" does the line sit? The human literature's gate is useful: update on *prediction error* only, strengthen on confirmation, never churn on mere reactivation.
4. **Background consolidation.** CLS theory says a replay job is mandatory for a healthy memory; the engineering evidence (sleep-time compute) is thin and first-party. Cheap to defer; dishonest to pretend it's settled.

---

*Raw briefings: research-human.md, research-llm.md, research-compose.md, research-engramMap.md, research-critic.md (session scratchpad; ask to commit under docs/research/ if you want them durable).*
