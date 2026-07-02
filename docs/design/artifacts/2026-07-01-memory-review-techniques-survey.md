<!-- Research artifact of docs/design/2026-07-01-memory-system-review.md (Task 2b).
     Web survey performed 2026-07-01; vendor/paper claims labeled as such in-text. -->

## External Techniques Survey for engram (2023–2026)

The five memory verbs used below: **capture** (ingest new information), **distill** (crystallize raw content into lessons), **recall** (surface relevant memory at the right moment), **refine** (update or correct existing memory), **build-upon** (compose across memories into new insight).

---

### 1. RAG + Reranking

**Mechanism.** Retrieve-Augment-Generate: embed a query, pull top-K chunks by cosine similarity, optionally rerank with a cross-encoder (token-level attention over query+chunk pair), then inject ranked chunks into LLM context. Reranking corrects order after approximate nearest-neighbor search. Serves **recall**.

**Evidence quality.** Extensively benchmarked across BEIR, MTRAG, RAGBench (2024–2025). Cross-encoder reranking consistently improves NDCG@10 over bi-encoder retrieval alone; numbers vary by domain (+5–15% typical). Paper-claimed on specific tasks; generalization to agentic memory settings is weaker. The "Fishing for Answers" paper (arxiv 2509.04820, 2025) shows iterative vs one-shot retrieval tradeoffs.

**Applicability.**
- (i) Local-first fit: HIGH. MiniLM-class cross-encoders run locally; no external service needed.
- (ii) Skills+binary fit: HIGH. Binary handles embedding+rerank; LLM does nothing extra.
- (iii) Real gap for engram: NO. Engram's retrieval ranking is explicitly noted as "largely solved" (note floor shipped, recall@5 0.22→0.83). Note 73 records three consecutive A/B negatives on retrieval scaffolds. Adding a reranker is a solved-axis investment.

---

### 2. GraphRAG (Microsoft, Edge et al., April 2024 — arxiv 2404.16130)

**Mechanism.** Two-phase: (1) LLM extracts entities + relations from source documents into a knowledge graph, then runs Leiden clustering to form "communities"; LLM pre-generates a summary for each community. (2) At query time, all community summaries generate partial answers, which are then aggregated. Serves **distill** (indexing phase) and **recall** (global sensemaking queries). Local variant uses nearby graph nodes instead of all communities.

**Evidence quality.** Benchmarked on ~1M-token corpora; paper-claimed "substantial improvements in comprehensiveness and diversity" over flat RAG on global sensemaking queries (Edge et al. 2024, arxiv 2404.16130). Weak on factoid retrieval — this is explicitly a query-focused summarization (QFS) approach, not a precision-retrieval one. No controlled ablations on injection of new memories vs. static corpora.

**Applicability.**
- (i) Local-first fit: MEDIUM. Graph construction requires many LLM calls; feasible locally but expensive at ingest time. The open-source `graphrag` Python package exists.
- (ii) Skills+binary fit: MEDIUM. Community summarization is LLM work; binary could maintain the graph index. Architecture mismatch: GraphRAG is designed for static corpora, not dynamic zettelkasten append.
- (iii) Real gap for engram: PARTIAL. Note 68 (engram's own research) explicitly named GraphRAG local search as a potential substrate for compositional/relational recall (the "emergent synthesis" gap engram cannot close with cosine clustering alone). But this gap (C6 cross-domain abduction) is already being served at 18/18 warm — the question is whether it holds in a crowded vault. GraphRAG addresses the gap but at high indexing cost.

---

### 3. HippoRAG (Gutiérrez et al., 2024 — arxiv 2405.14831; HippoRAG 2, 2025)

**Mechanism.** Neurobiologically-inspired: LLM extracts KG triples from passages offline; Personalized PageRank (PPR) over the triple graph spreads activation from query-seeded nodes to surface associatively linked content. Combines dense embedding (for triple/passage scoring) with graph spreading activation. Serves **recall** (especially multi-hop associative retrieval).

**Evidence quality.** Benchmarked on multi-hop QA (MuSiQue, 2WikiMultiHopQA, HotPotQA). Paper-claimed: up to +20% over RAG baselines on multi-hop QA; matches or exceeds iterative CoT retrieval (IRCoT) at 10–30x lower cost and 6–13x faster (arxiv 2405.14831, 2024). HippoRAG 2 (2025) extends with improved entity linking. Evidence is strong for multi-hop factoid QA; less clear for agent procedural memory.

**Applicability.**
- (i) Local-first fit: MEDIUM-HIGH. PPR is a local graph algorithm; the KG extraction LLM calls can be batched offline. No external service dependency once indexed.
- (ii) Skills+binary fit: HIGH fit conceptually. Binary handles graph index + PPR; LLM does extraction at write time (learn skill). Strong alignment with engram's DI-everywhere architecture.
- (iii) Real gap for engram: YES — the relational gap. Engram's vault is already wikilinked (vaultgraph exists); PPR spreading activation over those links is exactly what note 68 described as needed for compositional retrieval that cosine alone cannot do. This is a real, named gap. The question is whether the C6 failures are multi-hop or single-hop (if single-hop, the note floor already solved it).

---

### 4. Reflexion (Shinn et al., NeurIPS 2023 — arxiv 2303.11366)

**Mechanism.** After a task attempt fails, the LLM verbalizes a reflection ("what went wrong and why") and stores it in an episodic memory buffer. On the next attempt, the reflection is injected as context. Serves **capture** (of failure signals) and **refine** (correcting future behavior via stored reflections). Not a retrieval or crystallization system — it's a within-task iterative loop.

**Evidence quality.** Benchmarked on HumanEval (91% pass@1 claimed, vs GPT-4's 80% baseline at the time, NeurIPS 2023), ALFWorld sequential decision-making, HotPotQA reasoning. Numbers are paper-claimed (arxiv 2303.11366). Strong on tasks with a clear success signal; weaker on open-ended tasks without a ground-truth checker. Does not address cross-session memory.

**Applicability.**
- (i) Local-first fit: HIGH. Pure prompting — no external service; episodic buffer is just text.
- (ii) Skills+binary fit: MEDIUM. The reflection loop is all LLM; binary not involved. The `learn` skill is analogous but operates at session end, not within-attempt.
- (iii) Real gap for engram: PARTIAL. The structural recall miss (note 82) is that a known-failed lever gets re-proposed because the disproving note is outranked by raw chunks. Reflexion's within-attempt failure verbalization + injection addresses a different timing unit (within-session attempt, not cross-session). The "recall-before-recommend re-entry" fix described in note 82 is closer to Reflexion's mechanism than what engram currently ships.

---

### 5. Generative Agents — Importance Scoring + Reflection (Park et al., 2023 — Stanford/Google DeepMind)

**Mechanism.** Each memory observation is scored at write time by LLM on a 1–10 importance scale ("mundane" to "poignant"). Retrieval score = recency × relevance × importance (composite). Periodically (every ~100 memories), the agent runs a "reflection" pass: proposes 3 high-level questions its memories can answer, generates 5 insights per question, and stores them as higher-order memories. Serves **capture**, **distill** (reflection), and **recall** (composite scoring).

**Evidence quality.** Evaluated qualitatively in a Smallville simulation; no controlled ablation with metrics published in the original paper. Subsequent papers have benchmarked components separately. Importance scoring is anecdotal at system-level; the composite retrieval formula is widely replicated but rarely ablated cleanly.

**Applicability.**
- (i) Local-first fit: HIGH. Pure prompting + cosine similarity; no external service.
- (ii) Skills+binary fit: HIGH. Importance scoring at write time fits the `learn` skill; composite scoring formula fits the binary's ranking logic.
- (iii) Real gap for engram: PARTIAL. Importance scoring at write time could address the ranking problem (notes outranked by chunks) that note 82 describes. The reflection/distillation loop is analogous to engram's `learn` crystallization but adds periodic unsupervised batching — relevant if the vault grows to a scale where user-initiated crystallization misses patterns. Not a timing/coverage fix.

---

### 6. Voyager — Skill Library (Wang et al., May 2023 — arxiv 2305.16291)

**Mechanism.** Lifelong learning agent in Minecraft: (1) automatic curriculum proposes incrementally harder tasks, (2) an ever-growing skill library stores verified executable code (JavaScript) indexed by natural language description, retrieved by embedding similarity, (3) iterative self-verification loop refines skills before storage. Serves **capture** (skill storage), **recall** (skill retrieval), **build-upon** (skills compose into complex behavior). Avoids catastrophic forgetting by keeping parametric model frozen.

**Evidence quality.** Benchmarked in Minecraft: 3.3x more unique items, 2.3x longer distances, 15.3x faster tech tree milestones vs prior SOTA (paper-claimed, arxiv 2305.16291, 2023). Strong domain-specific results; Minecraft provides a clean binary success signal that makes skill verification tractable. Generalization to open-ended text tasks is unproven.

**Applicability.**
- (i) Local-first fit: HIGH. Skills are code strings; embedding retrieval is local. GPT-4 was used for generation but a local model can substitute.
- (ii) Skills+binary fit: HIGH in spirit. Engram's `skills/` directory is already a skill library. The key Voyager contribution — verified-before-storage with iterative refinement — could apply to how engram crystallizes notes (verify the lesson's truth before committing, not just formatting).
- (iii) Real gap for engram: WEAK. Engram's skill storage is solved; the gap is timing (when to fire recall) and procedure cost. Voyager's curriculum + skill-library architecture doesn't address either.

---

### 7. A-MEM: Agentic Memory (Xu et al., February 2025 — arxiv 2502.12110, NeurIPS 2025)

**Mechanism.** Zettelkasten-inspired: when a memory is stored, an LLM generates structured attributes (contextual description, keywords, tags), retrieves candidate linked notes, and decides whether to create wikilinks. "Memory evolution": new memories trigger LLM-driven updates to existing notes' attributes. ChromaDB for storage. Serves **capture**, **distill** (attribute generation), **refine** (evolution updates), and **recall** (top-k retrieval with structured index).

**Evidence quality.** Evaluated on LoCoMo and DialSim (multi-party long-dialogue QA); "superior improvement against existing SOTA baselines" across 6 foundation models (paper-claimed, NeurIPS 2025 accepted). Specific numbers not available from the abstract; token-length reductions via top-k retrieval vs full-context baseline are reported. Benchmarks are dialogue-focused, not agentic-coding or procedural.

**Applicability.**
- (i) Local-first fit: MEDIUM. ChromaDB is local; attribute generation requires LLM calls at write time. No external service dependency.
- (ii) Skills+binary fit: HIGH conceptual overlap. A-MEM's Zettelkasten-link generation at write time is exactly what engram's `learn` skill does (with engram doing it via LLM judgment + `engram learn/amend`). The "memory evolution" update of historical notes is analogous to `engram amend`. Engram is architecturally ahead of A-MEM in that the binary does computation (embeddings, clustering, ranking) while the LLM does judgment.
- (iii) Real gap for engram: MINIMAL. A-MEM describes engram's current architecture with a different stack (ChromaDB vs Go binary). The main novel claim — memory evolution (retroactive attribute updates) — is something engram can do via `engram amend` but doesn't do automatically. That retroactive propagation is a real gap if the vault grows large enough that stale attributes mislead retrieval.

---

### 8. MemoryBank — Ebbinghaus Decay (Zhong et al., 2023 — arxiv 2305.10250)

**Mechanism.** Stores memories in a hierarchical bank with strength scores. Each memory has a forgetting-curve strength: R = e^(-t/S). When a memory is recalled, its strength increases and decay resets. Unused memories decay toward zero and are eventually pruned. Serves **capture**, **recall** (strength-weighted), and a form of **refine** (strength update on recall). Applied in SiliconFriend companion chatbot.

**Evidence quality.** Demonstrated in a chatbot application (SiliconFriend, tuned on 38k psychological dialogues). Benchmark results are qualitative/demo-level; no controlled ablation on whether decay improves downstream task performance vs a flat store. Evidence quality is anecdotal.

**Applicability.**
- (i) Local-first fit: HIGH. Pure math (exponential decay); no external service.
- (ii) Skills+binary fit: HIGH. The binary is the right place for decay math; LLM doesn't need to touch it.
- (iii) Real gap for engram: WEAK. Engram already uses access-time recency weighting (the `activate` command updates recency metadata; the recent-activity channel surfaces freshly touched items). Decay-to-prune is an orthogonal concern; engram's measured problem is firing recall at the right moments, not pruning stale memories.

---

### 9. Episodic Memory for Agents + "Position: Episodic Memory is the Missing Piece" (2025 — arxiv 2502.06975)

**Mechanism.** Episodic memory stores temporally ordered sequences of agent experiences (state-action-outcome tuples or natural-language episode summaries), enabling agents to reason about "what happened when" rather than just "what is known." Contrasted with semantic memory (facts) and procedural memory (skills). Serves **capture** (episode logging), **recall** (episode retrieval by temporal or semantic similarity), **build-upon** (episode-level reflection).

**Evidence quality.** The 2025 position paper (arxiv 2502.06975) is an argumentative survey, not an empirical benchmark. The claim "episodic memory is the missing piece" is thesis-level, not validated. Individual episodic memory systems (e.g., TiMem, LightMem 2025) show gains in conversational coherence, but controlled evidence for agentic coding tasks is sparse.

**Applicability.**
- (i) Local-first fit: HIGH. Episode storage is text/embedding; no external service.
- (ii) Skills+binary fit: HIGH. Binary handles episode indexing; LLM handles episode synthesis.
- (iii) Real gap for engram: MODERATE. Engram's sessions are already effectively episodes (transcript chunks). The gap is that cross-session temporal ordering is not explicitly surfaced — the recent-activity channel provides recency but not episode-boundary structure. Episode boundary detection + structured replay could address the timing coverage problem (knowing to fire recall "at the start of a new episode on the same task").

---

### 10. Sleep-Time Compute / Offline Consolidation (Letta/various, April 2025 — arxiv 2504.13171; SCM 2025)

**Mechanism.** During idle time between queries, the agent runs LLM reasoning over accumulated context to pre-compute answers, reorganize memory, and prune redundancy — before new queries arrive. Paper (2504.13171): model "thinks offline" about a context, anticipates likely queries, pre-computes useful quantities. SCM (Sleep-Consolidated Memory, 2025): bounded working memory + multi-dimensional importance tagging + offline consolidation pass + algorithmic forgetting. Serves **distill** (offline crystallization), **refine** (index reorganization), and **recall** efficiency (pre-computed answers reduce query-time load).

**Evidence quality.** Sleep-time compute paper (arxiv 2504.13171, April 2025): paper-claimed 5x reduction in test-time compute for equivalent accuracy on Stateful GSM-Symbolic and Stateful AIME; up to 18% accuracy improvement; 2.5x per-query cost reduction via amortization across related queries. Strong empirical results but on math reasoning tasks with a clear success signal. Agentic SWE case-study included. SCM is more architectural/positional.

**Applicability.**
- (i) Local-first fit: HIGH. Offline compute runs between sessions; no external service.
- (ii) Skills+binary fit: HIGH. The offline consolidation phase is exactly what the `learn` skill does today — but reactively (after a session). Sleep-time compute is the proactive version: run `engram learn` as a background job between sessions to pre-distill. Binary handles indexing; LLM does synthesis.
- (iii) Real gap for engram: YES — directly targets procedure cost. The recall+learn procedure tax is ~411s/~67% of the warm op. Moving `learn` (crystallization + linking) to an async offline phase removes it from the critical path entirely. The sleep-time compute paper's per-query cost amortization mirrors engram's multi-session seed investment vs per-session payback.

---

### 11. Context Compaction / Editing (Anthropic, 2025)

**Mechanism.** Server-side: when context approaches the limit, automatically summarize older content and reinitialize with the compressed summary. Tool-result clearing: replace oldest tool results with placeholders, preserving the key architectural decisions and outcomes. Client-side: agents maintain persistent NOTES.md files outside context that reload on demand. Serves **distill** (compaction summarization) and **recall** (structured note reload). The guiding principle: "find the smallest set of high-signal tokens that maximize the likelihood of the desired outcome."

**Evidence quality.** Anthropic engineering blog (June 2025) — qualitative description of production techniques; no controlled benchmarks published. Compaction API available under beta header `context-management-2025-06-27`. Evidence is practitioner-observed, not peer-reviewed.

**Applicability.**
- (i) Local-first fit: MEDIUM. Server-side compaction requires Anthropic API; client-side NOTES.md pattern is fully local.
- (ii) Skills+binary fit: HIGH. The NOTES.md pattern IS engram's architecture (vault notes loaded into context). The binary handles the retrieval; the skill handles the compaction/reload logic. Engram is a more principled implementation of what Anthropic describes informally.
- (iii) Real gap for engram: PARTIAL. Engram already implements the structured note pattern more rigorously than the blog describes. The server-side tool-result clearing is relevant to engram's payload size problem (the ~200KB query payload paging ~50% of recall time per note 91); clearing tool results mid-session could reduce the context that recall needs to summarize.

---

### 12. ACE / Agentic Context Engineering (arxiv 2510.04618, October 2025)

**Mechanism.** Three-agent loop: Generator produces new strategy bullets from task outcomes, Reflector prunes redundant/low-value entries, Curator maintains the context as a structured itemized playbook. Avoids "brevity bias" (LLMs omitting important details) and "context collapse" (information loss under compaction). Serves all five verbs: **capture** (Generator), **distill** (Reflector), **recall** (structured playbook retrieval), **refine** (Curator dedup), **build-upon** (playbook composition).

**Evidence quality.** Paper-claimed: +10.6% on agentic tasks, +8.6% on finance tasks vs reflective prompt evolution baselines; matches top GPT-4.1-based production agent on AppWorld with smaller open-source model; 87% latency reduction vs reflective baseline (arxiv 2510.04618, October 2025, submitted ICLR 2026). Strong empirical results. Benchmarks are task-completion-based; the agentic gains are the most relevant signal.

**Applicability.**
- (i) Local-first fit: HIGH. Three-agent loop is pure prompting; no external service. Works with open-source models.
- (ii) Skills+binary fit: HIGH. The Generator/Reflector/Curator loop maps directly to the `learn`/`please` skill architecture. The binary isn't leveraged by ACE's design, but could handle the structured index while LLM handles generation/reflection/curation.
- (iii) Real gap for engram: YES — procedure cost and timing. ACE's Generator+Reflector+Curator loop is what engram's `learn` skill does manually. ACE automates the distillation cycle and shows it can run continuously without a human triggering it. This directly targets the "crystallization quality levers repeatedly null on delivery" problem: ACE's structured itemized format + automatic dedup prevents the vault-rot that engram gates against manually.

---

### 13. Dynamic Cheatsheet (test-time learning precursor to ACE)

**Mechanism.** A per-task scratchpad of structured notes updated after each task step; the LLM references the cheatsheet on the next step and can amend it. Serves **capture**, **recall** (within-session), **refine**. Precursor to ACE; ACE adds the Generator/Reflector/Curator separation and cross-session persistence.

**Evidence quality.** Demonstrated on coding, statistical analysis, ML workflows (EmergentMind summary); gains reported but not published in a standalone peer-reviewed paper before ACE subsumed it. Evidence is preliminary.

**Applicability.**
- (i) Local-first fit: HIGH.
- (ii) Skills+binary fit: HIGH (same as ACE).
- (iii) Real gap for engram: PARTIAL. The within-session scratchpad is already what engram's recall/learn cycle does at session granularity. Cross-session persistence is engram's contribution that Dynamic Cheatsheet lacked. ACE is the more relevant successor.

---

## Ranked Summary: Most to Least Promising for Engram

**Most promising:**

- **Sleep-Time Compute / Offline Consolidation** — directly targets the #1 live frontier (procedure cost ~411s/67% of warm op); move `learn` async; immediate fit with binary+skills architecture and confirmed gap.

- **ACE / Agentic Context Engineering** — addresses crystallization quality automatically (Generator+Reflector+Curator loop); +10.6% paper-claimed on agentic tasks; structured itemized format prevents vault-rot; maps directly to engram's `learn`/`please` skill architecture.

- **HippoRAG (PPR spreading activation)** — addresses the one real retrieval gap engram named (relational/compositional retrieval across wikilinks); engram's `vaultgraph` is already the substrate; HippoRAG 2's entity-linking is the missing link between chunks and graph nodes.

- **Reflexion-style recall-before-recommend re-entry** — addresses the structural recall miss (note 82): a within-session re-check of proposed levers against memory before recommending; engram has the gap, Reflexion has the mechanism; no infrastructure cost.

- **Generative Agents importance scoring** — LLM-rates memories at write time; could address the chunk-outranks-note ranking problem; fits `learn` skill; low implementation cost.

**Least promising:**

- **RAG + Reranking** — retrieval ranking explicitly solved (note floor shipped, 0.22→0.83); three A/B negatives already recorded in vault; do not invest further.

- **MemoryBank / Ebbinghaus Decay** — recency weighting already implemented via `activate`; decay-to-prune addresses a vault-scale problem engram doesn't yet have; weak evidence base.

- **Voyager skill library** — engram's `skills/` is already a skill library; the iterative self-verification contribution is worth noting at note-write time but doesn't address timing/cost gaps.

- **GraphRAG community summaries** — high ingest cost; designed for static corpora not dynamic append; the relational gap it addresses is better served by HippoRAG's lightweight PPR approach on the existing wikilink graph.

- **A-MEM** — architecturally very close to what engram already does; the main novel claim (memory evolution / retroactive attribute updates) is a real but low-priority gap; engram's binary is already more rigorous than A-MEM's ChromaDB stack.

Sources:
- [RAG Comprehensive Survey 2025](https://arxiv.org/html/2506.00054v1)
- [GraphRAG: From Local to Global (Edge et al., April 2024)](https://arxiv.org/abs/2404.16130)
- [HippoRAG arxiv 2405.14831](https://arxiv.org/abs/2405.14831)
- [Reflexion: NeurIPS 2023](https://arxiv.org/abs/2303.11366)
- [Generative Agents (Park et al., 2023) - MemX summary](https://memx.app/glossary/generative-agents/)
- [Voyager (Wang et al., May 2023)](https://arxiv.org/abs/2305.16291)
- [A-MEM (Xu et al., February 2025)](https://arxiv.org/abs/2502.12110)
- [MemoryBank (Zhong et al., 2023)](https://arxiv.org/abs/2305.10250)
- [Position: Episodic Memory is the Missing Piece (2025)](https://arxiv.org/abs/2502.06975)
- [Sleep-Time Compute (April 2025)](https://arxiv.org/abs/2504.13171)
- [Anthropic Context Compaction docs](https://platform.claude.com/docs/en/build-with-claude/compaction)
- [Anthropic Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [ACE: Agentic Context Engineering (October 2025)](https://arxiv.org/abs/2510.04618)
