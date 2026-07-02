<!-- Research artifact of docs/design/2026-07-01-memory-system-review.md (Task 2a).
     Web survey performed 2026-07-01; vendor/paper claims labeled as such in-text. -->

# External Agent Memory Systems Survey

**Systems covered:** MemGPT/Letta, Mem0, Zep (Graphiti), LangMem, ChatGPT native memory (including Dreaming V3), Claude Code memory ecosystem (CLAUDE.md / Auto Memory / claude-mem), Cognee (added as a prominent 2025-2026 framework).

---

## 1. MemGPT / Letta

**Primary sources:** Letta docs (docs.letta.com), sureprompts.com walkthrough (2026), letta.com v1 blog post (2026)

### Capture
Agent-decided, mid-conversation. The agent receives messages and calls tool functions (`core_memory_append`, `archival_memory_insert`) to store what it judges worth keeping. The system prompt teaches decision rules (e.g., "append only stable facts about the user"). Forgetting is the default; remembering requires an explicit tool call.

### Distill
Not a pipeline — storage is direct. Semantic blocks (labeled `[human]`, `[persona]`, custom blocks like `current_project`) hold bounded structured state. No LLM extraction pass separate from the agent's own reasoning. During "sleep-time compute" (autonomous reflective passes between conversations), the agent can reorganize: consolidate archival items, rewrite messy blocks, deduplicate near-duplicates — but this is agent-reasoned, not a separate processing stage.

### Recall
Three tiers with distinct retrieval patterns:
- **Core memory (context window / "RAM"):** Always visible on every turn — no retrieval cost or latency.
- **Recall memory ("SSD"):** Recent conversation history; the agent calls a search tool when something earlier in the session is needed.
- **Archival memory ("disk"):** Long-term, vector-indexed. Agent calls `archival_memory_search(query)` when relevant. Trigger is fully agent-decided based on system-prompt protocols.

### Refine
`core_memory_replace(label, old_substring, new_substring)` for in-place rewrites. Contradictions should be replaced rather than appended (failure mode: "runaway human block" appending every observation). Archival consolidation passes merge near-duplicates. All refine operations are agent-initiated tool calls.

### Build-upon
Limited. Agents can store both literal and paraphrased versions of facts to aid cross-phrasing discovery. Custom blocks track structured multi-field state. Integration with external RAG tools is possible (agent can call a RAG tool over a separate document corpus). No native cross-note linking or graph traversal — composition is entirely via agent reasoning over retrieved items.

**Storage substrate:** Agent state persisted in a database (implementation unspecified in public docs). Archival memory backed by a vector index (embedding method unspecified).

**Retrieval tech:** Vector similarity search for archival; full in-context for core; tool-based search for recall memory.

**Trigger model:** Agent-decided for both capture and recall. Framework provides the tools; the agent's reasoning decides when to invoke them.

**Evidence:** No published independent benchmark numbers for Letta specifically. Vendor claims DMR 93.4% (MemGPT baseline, from Zep paper arXiv:2501.13956v1, January 2025 — this is the number Zep beats against). Letta V1 blog (2026) claims improved performance on GPT-5/Claude 4.5 Sonnet with the new loop but provides no memory-specific metrics.

---

## 2. Mem0

**Primary sources:** mem0.ai architecture blog, aikickstart.com.au technical deep-dive, vectorize.io comparison (2026)

### Capture
Framework-automatic on every user message. Raw messages enter short-term memory (Redis) and episodic storage simultaneously. An LLM importance-scoring pass evaluates retention worthiness using direct cues ("remember that..."), implicit signals, contradictions with stored data, and new entities. Content exceeding a relevance threshold advances to long-term storage. The developer calls `add()` to feed content; the extraction pipeline decides what facts warrant storage without further agent involvement.

### Distill
Two-phase pipeline:
1. **Extraction:** LLM identifies facts, preferences, entities worth storing.
2. **Refinement:** Deduplication check prevents near-duplicates (merges rather than duplicates). Entity relationship extraction links "Alpha project" to "John" etc. Qualified memories are embedded and persisted.

### Recall
Five-stage retrieval pipeline:
1. Short-term (Redis, sub-millisecond)
2. Semantic search via vector embedding (~10-50ms)
3. Entity expansion (pull all memories tied to mentioned entities)
4. Temporal relevance re-ranking (recently-used memories ranked higher)
5. Deduplication and final ranking, trimmed to context window

Agent-initiated: the developer/application calls the retrieval API when context is needed; no user trigger required.

### Refine
When new information conflicts with stored data: both versions retained with timestamps, confidence scores tracked, source attribution recorded, contradictions flagged for agent resolution. The system does not silently overwrite — it preserves conflicting versions. Automatic deduplication merges near-duplicates at write time.

### Build-upon
Pro tier adds a graph memory layer (Mem0g) capturing relationships between entities, enabling multi-hop retrieval beyond similarity matching. Free/OSS tier is vector-only. Session-level working memory maintains active context across turns.

**Storage substrate:** Layered — Redis (short-term, 24h TTL), pluggable vector DBs (pgvector, Pinecone, Weaviate, Qdrant, Chroma) for long-term, PostgreSQL with JSONB for episodic, in-memory for working context.

**Retrieval tech:** Hybrid — vector embeddings for semantic search + entity graph traversal (Pro tier) + temporal re-ranking.

**Trigger model:** Capture is framework-automatic (fires on every message). Recall is application/agent-initiated.

**Evidence:**
- Mem0's own April 2026 benchmark report (vendor-claimed, unverified): LoCoMo 92.5%, LongMemEval 94.4%, BEAM@1M 64.1%, BEAM@10M 48.6% at 6,787–6,956 tokens/query.
- Independent atlan.com evaluation (2026): LoCoMo 67.13%, P95 latency 59.82s — significantly worse than vendor claims. Source discrepancy unresolved.
- "49.0% on LongMemEval" cited in one independent comparison (vectorize.io, 2026) — much lower than vendor's 94.4%.

---

## 3. Zep / Graphiti

**Primary sources:** arXiv:2501.13956v1 (Zep paper, January 2025), neo4j.com/developer Graphiti post (2025), github.com/getzep/graphiti

### Capture
Framework-automatic. Conversations enter as "Episodes" — raw data units with messages, text, or JSON, actor attribution, and reference timestamps. The system processes the current message plus the prior 4 messages for entity extraction context. A bi-temporal model is applied: Timeline T (when events occurred) and Timeline T' (when Zep ingested them), with explicit valid/invalid timestamps on every graph edge.

### Distill
Three-tier hierarchy built via LLM extraction:
1. **Episode Subgraph (Gₑ):** Non-lossy raw data store; bidirectional indices link episodes to extracted entities. Nothing is discarded.
2. **Semantic Entity Subgraph (Gₛ):** Entities and facts extracted via LLM with a reflexion technique to minimize hallucinations. 1024-dimensional vector embeddings. Hybrid candidate retrieval (embedding + full-text) for deduplication; LLM-driven deduplication generates updated names and summaries.
3. **Community Subgraph (G_c):** High-level clusters of strongly-connected entities with LLM-generated summaries (via iterative map-reduce).

### Recall
Three-stage pipeline (agent-initiated via query):
1. **Search (φ):** Cosine similarity on 1024-dim embeddings, BM25 full-text, and breadth-first graph traversal (n-hop neighborhood from recent episodes).
2. **Reranker (ρ):** Reciprocal Rank Fusion, Maximal Marginal Relevance, episode-mentions frequency, node-distance, and cross-encoder LLM reranking.
3. **Constructor (χ):** Formats retrieved nodes/edges into context strings with fact descriptions, validity dates, entity summaries, community summaries.

### Refine
When new facts arrive, an LLM identifies semantic contradictions with existing edges. Temporally-overlapping conflicts are resolved by setting `t_invalid` on the older edge to the `t_valid` of the newer one — new information takes precedence. The old edge is not deleted (non-lossy). Community summaries are updated via label propagation and periodic full refreshes.

### Build-upon
Graph traversal is the native composition mechanism — BFS from entity nodes surfaces multi-hop relationships that flat similarity search would miss. The bi-temporal model enables queries like "what did the agent know on date X?" enabling temporal reasoning chains. Community subgraph provides emergent entity clustering for higher-level synthesis.

**Storage substrate:** Neo4j (graph DB with Lucene for full-text indexing). 1024-dimensional vector embeddings stored per entity and fact.

**Retrieval tech:** Hybrid — graph traversal (BFS) + cosine similarity + BM25 + cross-encoder reranking.

**Trigger model:** Capture is framework-automatic (fires on every conversation message). Recall is agent-initiated (explicit query).

**Evidence:**
- Zep paper (arXiv:2501.13956, January 2025, vendor-authored): DMR benchmark 94.8% vs MemGPT 93.4% (vendor-claimed, unverified by independent party).
- LongMemEval: up to 18.5% accuracy improvement with 90% latency reduction vs baselines (vendor-claimed, unverified; paper does not name the specific baselines).
- The "are we ready" paper (arXiv:2606.24775, 2026) notes graph-based systems like Zep can cost 155+ seconds per query in some workloads.

---

## 4. LangMem (LangGraph)

**Primary sources:** langchain-ai.github.io/langmem API docs, atlan.com LangMem guide (2026), DigitalOcean LangMem tutorial (2026)

### Capture
Two paths:
- **Inline (`create_memory_manager`):** Analyzes conversation messages synchronously; identifies implicit preferences and key information; generates structured memory entries.
- **Background (`ReflectionExecutor`):** Asynchronous enrichment scheduled after user interactions with configurable debounce delay. Processes conversation batch without blocking the agent.

Developer-or-framework-initiated: the application calls `ainvoke()` with a conversation batch; LangMem's LLM pass decides what to extract.

### Distill
LLM analyzes conversation to produce either:
- Simple string-based memory entries, or
- Structured Pydantic-validated schemas (developer-defined fields and validation rules).

Three explicit memory types: **Semantic** (user preferences, facts, background), **Episodic** (past successful interactions as examples), **Procedural** (system prompt as a behavioral rule that gets iteratively refined — unique to LangMem).

### Recall
Dual pathways:
1. **Direct embedding search:** New conversation messages embedded; semantically similar memories retrieved from BaseStore.
2. **Query-optimized search:** Optional separate `query_model` generates optimized search queries, then retrieves up to `query_limit` (default: 5) memories.

Fires automatically during processing against namespaces like `("memories", "{langgraph_user_id}")`.

### Refine
`enable_updates: True` (default) — manager references existing memories and revises rather than duplicating. `enable_deletes: False` by default (explicit opt-in required for deletion). `enable_inserts: True`. Upsert logic prevents duplicate entries. Procedural memory updates rewrite the system prompt itself based on accumulated experience.

### Build-upon
Procedural memory is the distinctive mechanism: the agent's own system prompt evolves based on feedback and past interactions — "the agent rewrites its own instructions." No native graph linking. Composition relies on the LLM reasoning over retrieved semantic/episodic/procedural items.

**Storage substrate:** LangGraph's `BaseStore` with semantic indexing. Requires embedding configuration (`text-embedding-3-small`, 1536 dims). Namespace-structured for multi-user isolation.

**Retrieval tech:** Vector embeddings (OpenAI text-embedding-3-small by default) in LangGraph's BaseStore.

**Trigger model:** Developer/framework-initiated (application calls the memory manager). Background execution is framework-automatic after user interactions.

**Evidence:**
- Independent atlan.com evaluation (2026, third-party): LoCoMo accuracy 58.10%, P95 latency 59.82 seconds per query. The guide explicitly warns LangMem is "unsuitable for interactive agents" due to latency; recommends background/batch use only.
- No vendor benchmark numbers found for LangMem.

---

## 5. ChatGPT Native Memory (OpenAI)

**Primary sources:** OpenAI blog (openai.com/index/memory-and-new-controls-for-chatgpt), OpenAI Dreaming announcement (June 2026), nerdleveltech.com Dreaming V3 analysis (2026)

### Capture
Two mechanisms operating in parallel:
1. **Explicit save ("Saved Memories"):** User instructs ChatGPT to remember something specific; agent stores it. User-initiated.
2. **Dreaming (background synthesis):** Since April 2025; Dreaming V3 launched June 4, 2026. A background process reads across years of past conversations and automatically curates what to remember without user prompting. Framework-automatic.

### Distill
Dreaming V3 replaces the manually-curated saved-memories list with a background synthesis process that:
- Reviews all past conversation history.
- Extracts durable preferences, facts, and behavioral patterns.
- Resolves temporal evolution (e.g., updates "going to Singapore in July" to "went to Singapore in July 2026" after the trip ends).
- Maintains a synthesized memory state separate from conversation logs, injected into the system prompt at inference time.

The distillation is LLM-driven and opaque — internals not publicly documented.

### Recall
The synthesized memory state is injected into every new conversation's system prompt at inference time. No query-time retrieval step visible to the user or agent; memory is pre-loaded as context. Users can view/edit their memory on a "memory summary page."

### Refine
Temporal revision is built into Dreaming — memory self-updates as time passes. Users can manually correct or dismiss specific memories via the summary page. The older saved-memories list allowed user adds/deletes; Dreaming V3 largely automates this. No published details on deduplication or conflict resolution mechanics.

### Build-upon
Not documented. Memory items appear to be relatively flat; no public evidence of cross-memory linking or graph-based composition.

**Storage substrate:** Opaque (closed system). Memory state maintained in a "separate data layer" and injected at inference time.

**Retrieval tech:** Opaque. System-prompt injection rather than query-time retrieval.

**Trigger model:** Capture — hybrid (user-initiated for explicit saves; framework-automatic for Dreaming). Recall — framework-automatic (injected every session without agent or user query).

**Evidence:**
- OpenAI internal evaluation (vendor-claimed, unverified, June 2026): Factual recall 82.8% (vs 67.9% in 2025, vs 41.5% in 2024). Preference adherence 71.3%. Time-sensitive accuracy 75.1%. Compute reduced ~5x to serve Dreaming to Free tier.
- No released eval set or independent replication of these numbers confirmed.

---

## 6. Claude Code Memory Ecosystem (CLAUDE.md / Auto Memory)

**Primary sources:** sfeir.com CLAUDE.md guide, milvus.io four-layer analysis (2026), augmentcode.com claude-mem analysis (2026)

This is a layered, file-based ecosystem rather than a single unified system.

### Layer 1: CLAUDE.md (Explicit Rules)
**Capture:** Human-authored. User writes durable workflow rules, conventions, project structure into Markdown files. User-initiated.
**Distill:** None — content is pre-written by the human, not extracted from experience.
**Recall:** Loaded at every session start; injected into system prompt. No search needed; always in context.
**Refine:** Manual user edits only.
**Build-upon:** None (static).
**Storage:** Files on disk; project-level committed to git, personal at `~/.claude/CLAUDE.md`.

### Layer 2: Auto Memory (MEMORY.md)
**Capture:** Agent-decided during conversations and on explicit user requests ("remember that..."). Claude writes observations into MEMORY.md.
**Distill:** No dedicated extraction pass — agent writes what it judges worth keeping during the conversation.
**Recall:** 200-line index loads each session; grep-based keyword matching pulls specific entries when relevant.
**Refine:** Agent can update in-place; contradictions resolved when new entries written. No automatic conflict detection.
**Build-upon:** None. Index is flat; no semantic or graph search.
**Storage:** Markdown files at `.claude/projects/*/memory/MEMORY.md`.
**Limits:** 200-line cap, keyword-only retrieval (no semantic search), can only bridge days not months.

### Layer 3: Auto Dream (Background Consolidation)
**Capture:** Timer-triggered (24h+ elapsed, 5+ sessions). Background consolidation pass.
**Distill:** Resolves contradictions, updates dated references, integrates stale memories.
**Recall:** Results feed back into MEMORY.md.
**Trigger:** Automatic timer or manual `dream` command.
**Limits:** Helps with days-old clutter; cannot bridge months-old context.

### Layer 4: claude-mem (Third-Party Plugin)
**Capture:** Automatic via Claude Code lifecycle hooks (SessionStart, PostToolUse, Stop, UserPromptSubmit, SessionEnd) — records every tool use, file read, edit passively.
**Distill:** Background worker compresses observations using Claude API; stores compressed results in SQLite with FTS5. ~10x token reduction vs. raw logs.
**Recall:** Hybrid vector search (Chroma) + keyword lookup. Automatic context injection at SessionStart based on current task relevance.
**Refine:** Not documented (no explicit pruning or conflict resolution described).
**Build-upon:** None documented.
**Storage:** SQLite (compressed observations) + Chroma (vector index).

**Trigger model overall:** CLAUDE.md — user-initiated capture, automatic recall (always loaded). Auto Memory — agent-decided capture, automatic recall (index loaded each session). Auto Dream — framework-automatic capture. claude-mem — framework-automatic capture, framework-automatic recall at session start.

**Evidence:** No published benchmarks for any Claude Code memory layer. Milvus blog notes Auto Memory's grep-only retrieval as a structural limitation. Vendor Claude Code documentation describes the layers but provides no performance data.

---

## 7. Cognee

**Primary sources:** cognee.ai architecture blog, codepointer.substack.com comparison (2026), tryxlr8.ai 2026 survey

### Capture
The `add` operation ingests data from files, directories, raw text, URLs, or S3 URIs across 38+ formats (PDF, CSV, JSON, audio, images, code). Content is normalized to plain text, deduplicated via hashing, and organized into datasets with ownership controls. User/application-initiated.

### Distill
The `cognify` operation runs a six-stage LLM pipeline:
1. Document classification and permission verification
2. Chunk extraction
3. LLM-based entity and relationship extraction
4. Summary generation
5. Vector embedding
6. Graph commitment

Only new or modified files are reprocessed (incremental updates). Every node in the graph has a corresponding embedding for coherent semantic-relational movement.

### Recall
The `search` operation queries across vector and graph layers via 14 retrieval modes. Default `GRAPH_COMPLETION` mode: vector search identifies entry points, then graph traversal builds structured context. User/agent-initiated; session context preserved for conversational coherence.

### Refine
The `memify` operation prunes stale nodes, strengthens frequent connections, reweights edges based on usage signals, and adds derived facts. This is explicitly a post-hoc graph refinement step, making memory "an evolving structure that adapts based on feedback." User or scheduled invocation.

### Build-upon
The graph structure is the primary composition substrate. Session memory (short-term working context) and permanent memory (long-term artifacts) are both stored as graph nodes and remain "continuously cross-connected inside the graph while remaining linked to their vector representations." Multi-hop entity relationship traversal is a first-class capability. The `improve` API call allows agents to strengthen or add connections based on interaction feedback.

**Storage substrate:** Three-layer hybrid — Graph store (default Kuzu; alternatives: Neo4j, FalkorDB, Neptune, Memgraph), Vector store (default LanceDB; alternatives: Qdrant, pgvector, Redis, DuckDB, Pinecone, ChromaDB), Relational store (default SQLite; alternative: PostgreSQL).

**Retrieval tech:** 14 retrieval modes combining vector similarity + graph traversal. Default GRAPH_COMPLETION uses vectors as entry-point hints then traverses graph for structured context.

**Trigger model:** Capture — user/application-initiated (`add` call). Distillation — application-initiated (`cognify` call) or scheduled. Recall — user/agent-initiated (`search` call). Refinement — application-initiated (`memify` call) or scheduled.

**Evidence:** Vendor growth claims (cognee.ai, 2026): pipeline volume grew from ~2,000 to 1M+ runs in 2025 (500x); 70+ companies in production. ~17.6k GitHub stars (Apache 2.0). No published benchmark accuracy numbers on standard memory benchmarks (LoCoMo, LongMemEval, BEAM).

---

## Cross-System Observations

**Patterns everyone shares:**

1. **Vector embeddings as the baseline retrieval substrate.** Every system uses dense vector search as at minimum a component of retrieval — even graph-heavy systems (Zep, Cognee) use vectors as entry-point finders before graph traversal. Pure keyword/grep approaches (Claude Code Auto Memory) are explicitly identified as limitations.

2. **LLM-as-judge for distillation decisions.** All systems that perform distillation (Mem0, Zep, Cognee, LangMem, ChatGPT Dreaming) use an LLM to decide what is worth storing and how to structure it. None use statistical or rule-based extraction alone.

3. **Recall is almost universally query-initiated.** With the exception of ChatGPT (which pre-injects memory at session start) and claude-mem (which auto-injects at SessionStart), every system requires an explicit query/retrieval call. Memory does not proactively interrupt agents.

4. **Conflict resolution is primitive across the field.** The dominant approach is either temporal-wins (Zep), retain-both-with-timestamps (Mem0), or agent-must-resolve (Letta, LangMem). No system publishes a measured accuracy figure for conflict resolution specifically.

**Axes where systems genuinely differ:**

5. **Who decides to store (trigger model).** This is the sharpest structural divide: Letta is agent-decided (the model calls tools); Mem0 and Zep are framework-automatic on every message; LangMem and claude-mem are application/developer-initiated with background processing; CLAUDE.md is fully user-initiated; ChatGPT Dreaming is fully framework-automatic. These produce fundamentally different cost profiles and control/surprise trade-offs.

6. **Structural richness of the knowledge representation.** Range from flat key-value/vector stores (LangMem BaseStore, CLAUDE.md) through document-in-vector-DB (Mem0 OSS) to temporal knowledge graphs (Zep, Cognee) to bounded semantic blocks (Letta core memory). Graph-based systems outperform flat systems on cross-session and multi-hop queries (arXiv:2606.24775, 2026) but can cost 155+ seconds per query at scale.

7. **Procedural memory (self-rewriting instructions)** is unique to LangMem — no other surveyed system provides a native mechanism for an agent to rewrite its own system prompt based on accumulated experience. Claude Code's CLAUDE.md is closest but requires human authorship.

**Notably absent across the field:**

8. **Adversarial evaluation of memory value.** Standard benchmarks (LoCoMo, LongMemEval, BEAM) measure recall accuracy under benign conditions. No mainstream system publishes numbers showing: (a) what fraction of stored memories are ever retrieved and used, (b) whether memory improves task outcomes beyond what the model can derive cold, or (c) the false-positive rate (memories retrieved and injected that hurt rather than help). The 2026 "Are We Ready" paper (arXiv:2606.24775) explicitly criticizes this: evaluations treat systems "as monolithic black boxes" and fail to decompose failures into retrieval vs. synthesis vs. capture.

9. **Independent replication of vendor benchmark numbers.** Every system reporting benchmark performance on standard datasets is reporting its own numbers. The one system where independent measurement exists (Mem0) shows a 25-45 point gap between vendor claims (LongMemEval 94.4%, LoCoMo 92.5%) and independent measurements (LongMemEval ~49%, LoCoMo 67.13%). The gap is unresolved and unaddressed in public documentation.

10. **Memory value decomposition across task types.** No system separates "memory wins because the model couldn't derive this cold" (genuine idiosyncratic capability) from "memory wins because the prompt-priming from retrieved text helped" (a prompt-engineering effect achievable without persistent storage). These two mechanisms have different cost/benefit profiles but are conflated in all published evaluations.

---

**Sources consulted (primary):**
- Zep paper: https://arxiv.org/abs/2501.13956 (January 2025)
- Letta docs: https://docs.letta.com/guides/legacy/memgpt_agents_legacy
- Letta v1 blog: https://www.letta.com/blog/letta-v1-agent
- Letta walkthrough: https://sureprompts.com/blog/letta-memgpt-walkthrough
- Mem0 architecture: https://mem0.ai/blog/memory-in-agents-what-why-and-how
- Mem0 technical deep-dive: https://aikickstart.com.au/news/mem0-architecture-how-agent-memory-works
- Mem0 2026 state report: https://mem0.ai/blog/state-of-ai-agent-memory-2026
- Mem0 vs Letta comparison: https://vectorize.io/articles/mem0-vs-letta
- LangMem API docs: https://langchain-ai.github.io/langmem/reference/memory/
- LangMem guide (atlan.com): https://atlan.com/know/long-term-memory-langchain-agents/
- ChatGPT Dreaming V3: https://nerdleveltech.com/chatgpt-dreaming-v3-memory-architecture
- Claude Code four layers: https://milvus.io/blog/claude-code-memory-memsearch.md
- Claude Code CLAUDE.md: https://institute.sfeir.com/en/claude-code/claude-code-memory-system-claude-md/
- claude-mem plugin: https://www.augmentcode.com/learn/claude-mem-persistent-memory-claude-code
- Cognee architecture: https://www.cognee.ai/blog/fundamentals/how-cognee-builds-ai-memory
- "Are We Ready" paper: https://arxiv.org/html/2606.24775 (2026)
- Framework survey: https://atlan.com/know/best-ai-agent-memory-frameworks-2026/
