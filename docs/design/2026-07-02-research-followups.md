# Research followups for engram (2026-07-02)

> THE consolidated report of vision-relevant findings from the link-value exploration's S0 research
> sweep that are out-of-scope for the linking work itself but engram-relevant. Parked here per note
> 154 (parked ≠ unrecorded). One report for the whole sweep — compound engineering, peer memory
> systems' linking practice, PKM/zettelkasten evidence, IR/KG literature, and Claude-Code-ecosystem
> memory tools. Sources surveyed 2026-07-02; vendor/blog claims labeled (claimed, unverified).

# Part 1 — Compound Engineering (every.to / EveryInc plugin)

## What compound engineering actually does

Kieran Klaassen's compound engineering system (published at every.to/guides/compound-engineering;
plugin open-sourced at github.com/EveryInc/compound-engineering-plugin, 22.5k stars as of
2026-06-03) implements a four-to-eight-step engineering loop:

  Ideate → Brainstorm → Plan → Work → Review → Polish → Compound → Repeat

The Compound step is the knowledge-capture step that makes each cycle faster than the last.
Source: every.to/guides/compound-engineering-gets-an-upgrade (2026).

### The ce-compound skill (publicly verified)

The actual `skills/ce-compound/SKILL.md` is publicly available in the EveryInc repo. Summary of
what the guide describes as "six parallel subagents" (claimed) vs what the skill actually deploys:

**Phase 1 — three parallel research subagents:**
- **Context Analyzer** — reads conversation history, classifies as bug-track vs knowledge-track,
  proposes filename and target directory
- **Solution Extractor** — structures the fix with before/after code; populates `root_cause`
  (controlled enum) and `resolution_type` (controlled enum)
- **Related Docs Finder** — greps `docs/solutions/` across 5 overlap dimensions: problem statement,
  root cause, solution approach, referenced files, prevention rules; recommends update-vs-create

**Phase 1 optional (off by default):**
- **Session Historian** — searches prior sessions across Claude Code / Codex / Cursor for related
  context

**Phase 3 specialized reviewers (post-assembly):**
- Performance Oracle, Security Sentinel, Data Integrity Guardian

The guide's "six" appears to count the three Phase-1 agents + three Phase-3 reviewers. The
"prevention strategist" and "category classifier" from the guide's marketing description are
functional roles embedded in the Phase-1 agents (Context Analyzer handles classification; Solution
Extractor handles prevention structure), not separate agents.
Source: github.com/EveryInc/compound-engineering-plugin/blob/main/docs/skills/ce-compound.md (2026).

### YAML frontmatter schema (publicly verified from references/schema.yaml)

Two tracks with distinct schemas:

**Bug Track** (build errors, test failures, runtime errors, performance, database, security, UI,
integration, logic):

```yaml
module: <category>
date: YYYY-MM-DD
problem_type: <enum>
component: <affected system>
severity: <level>
symptoms:
  - <1-5 observable error descriptions>
root_cause: <fundamental technical cause — from controlled enum>
resolution_type: <type of fix applied — from controlled enum>
related_components: [optional array]
tags: [optional; ≤8 lowercase hyphen-separated keywords]
```

**Knowledge Track** (best practices, architecture patterns, design patterns, tooling decisions,
conventions, workflow, developer experience, documentation gaps):

```yaml
module: <category>
date: YYYY-MM-DD
problem_type: <enum>
component: <affected system>
severity: <level>
related_components: [optional array]
tags: [optional; ≤8 lowercase hyphen-separated keywords]
```

On update: `last_updated: YYYY-MM-DD` is added to the existing document.

Source: github.com/EveryInc/compound-engineering-plugin/blob/main/skills/ce-compound/references/schema.yaml (2026).

### Directory structure

```
docs/solutions/
  build-errors/
  test-failures/
  runtime-errors/
  performance-issues/
  database-issues/
  security-issues/
  ui-bugs/
  integration-issues/
  logic-errors/
  architecture-patterns/
  design-patterns/
  tooling-decisions/
  conventions/
  workflow-issues/
  developer-experience/
  documentation-gaps/
  best-practices/
```

Source: docs/skills/ce-compound.md in the EveryInc repo (2026).

### How retrieval works (critically different from engram's model)

Compound engineering does NOT use graph traversal or embedding-ranked retrieval for solution docs.
Their retrieval is LLM-agent judgment over the full `docs/solutions/` directory:

1. **`ce-learnings-researcher`** agent fires as part of `ce-plan` — reads `docs/solutions/` and
   judges relevance to the current task using LLM judgment, not embedding similarity or tag queries
2. **`repo-research-analyst`** agent also consults `docs/solutions/` during planning
3. **Local-first methodology** — check `docs/solutions/` BEFORE performing web research
4. **Discoverability check** — after each `/ce-compound` run, the skill checks whether
   `AGENTS.md`/`CLAUDE.md` mentions `docs/solutions/`; if not, it proposes a minimal one-line addition

There is no embedding index of `docs/solutions/`, no tag-graph, and no explicit frontmatter-field
queries. The LLM agent reads the files and judges relevance. This scales with LLM cost (not vault
size directly).
Sources: deepwiki.com/EveryInc/compound-engineering-plugin/7-knowledge-systems;
skills/ce-plan/SKILL.md (2026).

### Trigger model (how human-dependent is their fire)

**Capture (the Compound step):**
- Primary: human-triggered post-fix via `/ce-compound` command
- Semi-automatic: auto-trigger on success phrases ("that worked", "it's fixed", "working now",
  "problem solved") — described in SKILL.md (claimed; delivery rate unverified)
- Full-automatic: `/lfg` meta-command chains the complete pipeline including compound — so compound
  runs automatically in end-to-end workflow mode, but `/lfg` itself is human-invoked

**Recall (during planning):**
- Automatic within `ce-plan` — `ce-learnings-researcher` fires for every planning invocation
- But `ce-plan` itself is human-invoked

**Net assessment:** Human-invoked at the workflow level, automatic within the workflow. The entry
point is still a human calling `/ce-plan` or `/lfg`. Within that invocation, recall fires without
a separate human step. This is more autonomous than engram's current model (recall and learn are
both human-invoked independently).

## Capture-quality practices compound engineering has that engram lacks

### 1. Two-track document structure

Compound engineering enforces two distinct document schemas: bug track and knowledge track, with
different required sections. Engram uses a single note format for all content types. The missing
structure is:

- Bug track: a dedicated "What Didn't Work" section (failed approaches) and an enforced "Prevention"
  section separate from the lesson body
- Knowledge track: a "When to Apply" section (applicability conditions) and an "Examples" section

The "What Didn't Work" section is particularly relevant to engram's recall problem: the vault
accumulates lessons about what worked, but failed levers are often only mentioned in passing. An
enforced "What Didn't Work" section would give failed approaches their own retrieval signal.

### 2. Write-time overlap detection (5-dimension deduplication)

At write time, the Related Docs Finder checks new content against existing docs across 5 dimensions:
problem statement, root cause, solution approach, referenced files, prevention rules. High overlap
(4-5 dims) → update existing doc. Moderate overlap (2-3 dims) → create new, flag for refresh.

Engram has no write-time deduplication. Near-duplicate notes accumulate and split the cosine
retrieval signal, reducing recall precision. This is a capture-quality gap.

### 3. Controlled vocabulary fields (root_cause and resolution_type enums)

Bug-track notes have `root_cause` and `resolution_type` fields drawn from controlled enumerations.
These controlled-vocab tokens make two notes with the same root cause co-retrievable by an LLM
agent even when the natural-language descriptions differ.

Engram notes have emergent tags (free-form keywords) but no controlled vocabulary for the type of
problem or the type of fix. This limits cross-note linkability via shared token signals.

### 4. Scratch artifact pattern (issue #956 summary-collapse prevention)

Phase-1 subagents in ce-compound write full structured output to per-run scratch artifacts under
`/tmp/compound-engineering/ce-compound/{run_id}/` and return only the artifact path. The
orchestrator reads artifacts in Phase 2. This prevents "summary-collapse" — the failure mode where
inline LLM returns become executive summaries that lose specific detail.

Engram's learn skill collects subagent outputs inline. If the learn skill's agents are returning
summaries rather than full structured content, this is the same failure mode. The scratch-artifact
pattern is worth adopting.

### 5. Post-write discoverability check

After writing a solution, ce-compound verifies that `AGENTS.md`/`CLAUDE.md` surfaces `docs/solutions/`
to future agents. If not, it proposes a minimal one-line addition. Engram has no equivalent
post-write discoverability verification — a new note is added to the vault but the guidance for
when to recall it is not updated.

### 6. Structured staleness sweep (ce-compound-refresh)

The `ce-compound-refresh` skill systematically reviews solution docs against the current codebase
state and classifies each: Keep / Update / Consolidate / Replace / Delete. Engram's `engram amend`
is human-initiated and ad hoc; there is no systematic sweep for drift.

## Autonomy gap analysis

Compound engineering's more autonomous recall comes from bundling `ce-learnings-researcher` into
every `ce-plan` invocation — recall fires automatically at planning time without a separate human
step. Engram's equivalent would be: recall fires automatically inside the brainstorming or planning
skill, not as a separate `/recall` invocation.

The success-phrase auto-trigger for capture ("that worked" → fires `/ce-compound`) is the capture
equivalent. Engram's current learn trigger is entirely human-initiated; adopting in-workflow triggers
(e.g., learn fires automatically inside `/please` after the work step) would close this gap.

These both route to **Track A (decision-moment hooks)** as defined in the link-value plan's S4
autonomy routing question. The link exploration's T6 variant (glance-breadth under 3-phrase
conditions) is the closest current analog: automatic recall under a specific constrained condition.

## Parity assessment (qualitative, honest)

Engram's recall (embedding-ranked, note-floor-guaranteed, clustering-synthesized) is architecturally
more rigorous than compound engineering's flat-LLM-agent-judgment model. For a vault of 135 notes,
compound engineering's agent-judgment approach would be feasible but expensive (~$0.10–0.50/query
at agent-level rates). Engram's approach scales better.

Engram lags in:
- **Capture structure** — no two-track schema, no controlled vocabulary, no write-time deduplication
- **Capture trigger** — no in-workflow automatic capture; entirely human-initiated
- **Recall trigger** — no automatic recall at planning time; entirely human-initiated
- **Staleness management** — no systematic staleness sweep

These are capture-quality and trigger-model gaps, not retrieval-quality gaps. Closing them does not
require winning the link-value exploration — they are independent improvements.

## Recommended next steps (out of scope for link exploration)

These are not conditioned on the link exploration results:

1. **Adopt two-track note structure** — add a template or learn-skill prompt for "bug-type" notes
   (with What-Didn't-Work and Prevention sections) vs "knowledge-type" notes (with When-To-Apply
   and Examples). Low implementation cost; high capture-quality gain.

2. **Add write-time overlap detection** — when `engram learn` writes a new note, check for high
   overlap (≥3 of: situation, outcome, lesson body token overlap) with existing notes; prompt user to
   update the existing note rather than create a new one.

3. **Structured staleness sweep** — a periodic `engram refresh` command that surfaces notes older
   than N months in active use areas and prompts for Keep/Update/Replace/Delete.

4. **Scratch artifact pattern for learn** — learn skill's subagents should write full output to tmp
   artifacts, not return inline; orchestrator reads artifacts. Prevents summary-collapse in the
   crystallization output.

5. **In-workflow recall trigger** — recall fires automatically inside `please` / brainstorming skill
   at the planning phase, not as a separate human-invoked step. This matches compound engineering's
   most important autonomy improvement.

## Source URLs

- https://every.to/guides/compound-engineering (2026; partial paywall) (claimed, unverified for paywalled portions)
- https://every.to/chain-of-thought/compound-engineering-how-every-codes-with-agents (2025)
- https://every.to/guides/compound-engineering-gets-an-upgrade (2026)
- https://github.com/EveryInc/compound-engineering-plugin (2026; publicly verified)
- https://github.com/EveryInc/compound-engineering-plugin/blob/main/docs/skills/ce-compound.md (2026; publicly verified)
- https://raw.githubusercontent.com/EveryInc/compound-engineering-plugin/main/skills/ce-compound/references/schema.yaml (2026; publicly verified)
- https://raw.githubusercontent.com/EveryInc/compound-engineering-plugin/main/skills/ce-plan/SKILL.md (2026; publicly verified)
- https://deepwiki.com/EveryInc/compound-engineering-plugin/7-knowledge-systems (2026)
- https://bitsby.me/2026/03/compound-engineering/ (2026-03)
- https://lethain.com/everyinc-compound-engineering/ (2026)
- https://davidguttman.github.io/every-vibe-code-camp-distilled/13_kevin_kieran.html (2026)


## Part 2 — Peer memory systems (research date: 2026-07-02)

### B2-1 · A-Mem write-time link enrichment as a note-quality lever (not a retrieval lever)

A-Mem (arXiv 2502.12110, NeurIPS 2025) uses LLM-generated links at write time to trigger
"memory evolution" — retroactive updates to neighboring notes' contextual descriptions, keywords,
and tags. The ablation (Table 2) shows multi-hop F1 improves from 24.55 → 31.24 when link
generation is added, and from 31.24 → 45.85 when memory evolution is added on top.
Links serve write-time enrichment, NOT retrieval-time traversal — retrieval is pure cosine.

**Vision relevance for engram**: The gain from A-Mem's link generation is entirely from better
note quality (linked notes get their attributes updated when new related notes are written). This
is analogous to engram's `learn` skill triggering `engram amend` on related notes — a lever that
is separate from the retrieval traversal experiments in this workstream. Worth a dedicated
exploration once the link-value exploration settles which note update triggers are worth
instrumenting at `learn` time.

Source: arXiv:2502.12110v1, §3.3–3.4, Table 2. Accessed 2026-07-02.

---

### B2-2 · Graphiti's bi-temporal edge model for precise supersession tracking

Graphiti (arXiv 2501.13956, January 2025) models every edge with four timestamp fields:
t_valid (event-world start), t_invalid (event-world end), created_at (system ingestion),
expired_at (system invalidation). New information that contradicts an existing edge triggers
LLM-based semantic comparison; the old edge's t_invalid is set to the new edge's t_valid.
Rule: "Graphiti consistently prioritizes new information."

Entity dedup: 1024-dim embedding cosine search + full-text name/summary search + LLM judge.
An externally documented MinHash + LSH fast path exists but threshold values are not published.

**Vision relevance**: The bi-temporal model is the production-grade implementation of engram's
L5/T5 supersession concept. If the link-value exploration validates T5, the L5 edge schema
should adopt Graphiti's four-field structure. The LLM-based conflict detection (semantic, not
exact-match) prevents missed supersessions where the wording differs.

Source: arXiv:2501.13956v1, §3.2–3.4; getzep/graphiti README; deepwiki.com/getzep/graphiti.
Accessed 2026-07-02.

---

### B2-3 · Mem0g: the most directly applicable graph-vs-flat ablation for agent memory

Mem0 (arXiv 2504.19413, April 2025) published a controlled comparison of Mem0g (graph variant,
entity/relation triplet graph with LLM-based entity resolution) vs. plain Mem0 (flat vector
store) on LOCOMO, a conversational agent memory benchmark. Results (GPT-4o):

- Overall: Mem0g 68.44 vs. Mem0 66.88 (+1.56)
- Temporal: Mem0g 58.13 vs. Mem0 55.51 (+2.62)
- Open-domain: 75.71 vs. 72.93 (+2.78)
- Single-hop: 65.71 vs. 67.13 (−1.42, graph loses)
- Multi-hop: 47.19 vs. 51.15 (−3.96, graph loses)
- Memory cost: 14k tokens (Mem0g) vs. 7k tokens (Mem0) — 2× overhead
- Search speed: ~3× slower

**Vision relevance**: This is the closest published ablation to engram's S2 evaluation design
(agent memory, not factoid QA corpora). The result pattern — graph helps temporal/relational
reasoning, hurts precise single-fact lookup — is a calibration prior for engram's miss-population
design. P3 (supersession) and open-domain bridge queries are the predicted win zone; P1 real-query
single-hop misses may be the risk zone. The 2× token overhead is a concrete viability factor
against deploying T2 in engram's payload-sensitive context.

Source: arXiv:2504.19413, Table 4 (Mem0g vs. Mem0 ablation). Accessed 2026-07-02.

---

### B2-4 · HippoRAG's hub-suppression via node specificity inverse weighting

HippoRAG (arXiv 2405.14831) applies node specificity weighting before running PPR: each phrase
node i is weighted by sᵢ = |Pᵢ|⁻¹ (inverse of the number of passages that contain the phrase).
This suppresses high-degree hub phrases (e.g., "the model," "the system") from dominating PPR
propagation. The effect: low-degree specific phrases (rare concepts) get proportionally more
influence in the spreading activation.

**Vision relevance**: Engram's hub-kill problem (high-connectivity notes dominating traversal) is
directly addressed by this mechanism. If T2 is implemented, node specificity inverse weighting
should be applied to note degree, not just phrase frequency — notes with many incoming links
should have their PPR seed weight divided by their in-degree.

Source: arXiv:2405.14831v1 (HippoRAG 1), §3.3; arXiv:2502.14802v1 (HippoRAG 2), §3.2.
Accessed 2026-07-02.

---

### B2-5 · HippoRAG 2's passage nodes: dense-sparse integration via context edges

HippoRAG 2 (arXiv 2502.14802, February 2025) introduces passage nodes alongside phrase nodes,
connected by "contains" context edges (each passage node links to all phrase nodes extracted
from it). At retrieval, passage nodes receive PPR seed probability proportional to
embedding_similarity × w_dp where w_dp=0.05 (optimal per Table 5 ablation). This allows both
specific phrase matching (phrase node seeds) and broader passage embedding to seed the graph.
Result on MuSiQue R@5: 74.7 (HippoRAG 2) vs. 53.2 (HippoRAG 1) vs. 69.7 (NV-Embed-v2 flat).
HippoRAG 1 was worse than flat NV-Embed on MuSiQue; passage nodes are the swing.

**Vision relevance**: For engram, "passage nodes" maps to the source session chunk (a transcript
chunk) and "phrase nodes" map to the crystallized note. The L7 (provenance/episode) fabric creates
this structure. L7 + T2 (PPR) is the combination HippoRAG 2 validates — the combination may
outperform either L2 alone or embedding-only retrieval, but only if chunk vectors are used as
PPR seeds alongside note vectors. The retrieval probe harness already has access to chunk vectors
(.jsonl index); this integration would require seeding PPR from both note and chunk nodes.

Source: arXiv:2502.14802v1, §3.2–3.3, Tables 4–5. Accessed 2026-07-02.

---

### B2-6 · Cognee's multi-mode retrieval (14 modes, auto-routing) as a retrieval strategy

Cognee (topoteretes/cognee, 2025; arXiv 2505.24478 Markovic et al.) ships 14 named retrieval
modes ranging from classic vector RAG to chain-of-thought graph traversal (GRAPH_COMPLETION).
Auto-routing selects the mode based on query characteristics. GRAPH_COMPLETION: vector search
seeds matching entities/triples, then an LLM reasons over the subgraph. Vendor-claimed HotpotQA:
F1 0.63 vs. flat RAG 0.12; BEAM at 100K tokens: 0.79 vs. prior SOTA 0.735. No controlled ablation
separating graph traversal from LLM reasoning.

**Vision relevance**: Cognee's multi-mode design validates engram's cell matrix approach (T2/T3/T5/
T6 as different modes for different query types) rather than a single unified traversal. The auto-
routing pattern — let query characteristics determine traversal depth — maps to engram's phrase-
count heuristic for T6 (under 3 phrases → T6 breadth recovery). Worth tracking Cognee's BEAM and
forthcoming arXiv 2505.24478 results for evidence on which retrieval mode wins which query type.

Source: cognee.ai blog (vendor); deepnote.com Cognee vs RAG comparison (vendor); arXiv:2505.24478
abstract. Accessed 2026-07-02.

---

### B2-7 · No peer system publishes a clean graph-edges-present vs. absent ablation (the evidence gap)

Across all five systems surveyed:

- **A-Mem**: Ablation conflates link generation with memory evolution; retrieval is not graph-traversal.
- **Graphiti/Zep**: No ablation isolating graph traversal vs. flat retrieval; paper shows only full-
  system vs. full-context/session-summary/MemGPT baselines.
- **HippoRAG 1**: Ablates PPR vs. no-traversal within the graph system, but NOT graph-present vs.
  strong flat embedding (NV-Embed-v2). HippoRAG 2 does compare against NV-Embed-v2 flat but the
  HippoRAG 1 graph actually underperforms NV-Embed on MuSiQue (53.2 vs. 69.7 R@5), underscoring
  that graph traversal on a weak graph is worse than strong embeddings on no graph.
- **Mem0g**: The cleanest ablation (graph vs. flat on agent memory LOCOMO), but on a
  conversational-memory corpus, not agent procedural-lesson memory.
- **Cognee**: Vendor-reported comparisons only; graph and LLM-reasoning contributions are inseparable.

**Vision relevance**: Engram's S2 probe is filling a genuine evidence gap in the literature. The
clean ablation design — recover from engram's miss population with/without graph traversal, on the
actual vault and actual recall binary — is not replicated by any peer system on comparable data.
The pre-registered prune rules are conservative and appropriate given the evidence base.

Sources: arXiv:2502.12110, 2501.13956, 2405.14831, 2502.14802, 2504.19413. Accessed 2026-07-02.
