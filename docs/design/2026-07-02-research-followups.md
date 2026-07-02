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


# Part 2 — Peer memory systems (research date: 2026-07-02)

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


# Part 3 — PKM/zettelkasten practice

*Human-side evidence on linking disciplines that proved durable. Evidence quality labeled:
[study] = peer-reviewed; [archival] = archival quantitative analysis; [anecdote] = practitioner report.*

### 3.1 Luhmann's archival structure

Luhmann's Zettelkasten II (1963–1998): ~66,000 cards in the main slip box. Archival analysis
(Martijn Aslander, "Mapping Luhmann's Brain," 2024–2025, martijnaslander.github.io) [archival]:
73,715 indexed cards; 59,773 extracted references; 33,650 Fernverweise (cross-section/non-local
references, 56%); 14,389 neighbourhood links (24%); 11,182 unclassified. 23% of cards have no
cross-references beyond their own branch.

Schmidt (2016), "Niklas Luhmann's Card Index: Thinking Tool, Communication Partner, Publication
Machine," in *Forgetting Machines*, Brill [archival summary via zettelkasten.de, 2017]: Luhmann
maintained "hub notes" — Zettels containing extensive lists of links to core Zettels on a topic.
These functioned as "highways between topics." Schmidt's emphasis: references BETWEEN notes (the
Fernverweise) are more important for navigability than references FROM the keyword index.

Implications not captured in the link-value exploration: (a) Hub notes as a MAINTENANCE pattern
(not an automated feature) — engram has no analog; manual hub curation is expensive but the
highest-leverage linking discipline Luhmann employed. (b) The 23% isolation figure is a benchmark:
any agent-memory vault with >30% isolation is structurally under-linked relative to Luhmann's
floor. (c) Luhmann's Folgezettel (sequential branching by ID) were primarily a PHYSICAL constraint
workaround, not a conceptual primitive — their digital equivalent is episode/provenance edges (L7),
which the human evidence suggests are the weakest tier.

Source: martijnaslander.github.io/luhmann-zettelkasten/ (accessed 2026-07-02);
zettelkasten.de/posts/zettelkasten-hubs/ (accessed 2026-07-02).

### 3.2 Capture discipline — the Collector's Fallacy

Practitioners consistently report that collecting notes is easy; adding links and synthesis is hard.
The Collector's Fallacy (zettelkasten.de/posts/collectors-fallacy/) [anecdote]: "having information
≠ understanding it." The PKM community's failure mode: large vaults with dense tag networks and few
contextual links. The observation recurs: "if you just add links without any explanation you will
not create knowledge; your future self has no idea why he should follow the link."
(zettelkasten.de/introduction/) [anecdote].

Implication for engram's learn skill: any link engram creates should include an inline rationale
("why this link exists") — this is the practitioner standard for durable links. Links without
rationale are the collector's-fallacy analog in a linked graph. The JUSTIFY step in L2's LLM gate
already implements this; it should be treated as non-optional, not as a prunable step.

Source: zettelkasten.de (accessed 2026-07-02); zettelkasten forum discussion on link types,
forum.zettelkasten.de/discussion/2023/link-types (accessed 2026-07-02).

### 3.3 Tags as a weak association structure

Andy Matuschak, "Tags are an ineffective association structure" (notes.andymatuschak.org) [anecdote,
continuous since ~2019]: tags are vague, apply to whole items when only a fragment is relevant,
lack context explaining WHY items are associated, and present "jumbled unordered lists." He
recommends instead: explicit links, fine-grained (passage-level rather than note-level), labeled
(inline rationale for the association).

Nick Milo (Medium, 2021): tags are "relatively weak associations" that don't scale; users forget
tag names as vaults grow. [anecdote] Zettelkasten.de strength hierarchy: direct hyperlinks >
Folgezettel sequences > tags > juxtaposition. [anecdote]

Implication: engram should resist a tag-hub strategy as the PRIMARY discovery mechanism. Tags may
be viable as a FILTER at the candidate stage (T4-style nomination), but the practitioner consensus
is that tags decay in value as the vault grows while direct links appreciate.

Source: notes.andymatuschak.org/Tags_are_an_ineffective_association_structure (accessed 2026-07-02);
Nick Milo, "In what ways can we form useful relationships between notes," Medium 2021
(accessed 2026-07-02).

### 3.4 Review cadence and link discovery

Matuschak ("Evergreen notes should be densely linked"): "finding the right links requires reading
old notes, so it's an organic mechanism for intermittently reviewing the notes we've written." [anecdote]
Dense linking creates a spaced-repetition analog — the act of linking forces review. Progressive
summarization (Tiago Forte): revisit and compress notes after a day or two. [anecdote]

The fractal review pattern (daily → weekly → monthly → yearly note compilation) [anecdote, PKM
community]: creates emergent structure through iterative review cycles rather than upfront
classification. Links discovered at review time are reported as more meaningful than links
created mechanically at write time.

Implication: engram's learn skill fires at session end (write time). Practitioners report review-time
links feel more earned and get traversed more. A deferred-linking pass (review a batch of recent
notes and link them to older content) may yield higher-quality edges than write-time-only linking.
This is a potential Track-A trigger moment (after a gap between sessions, offer a linking review
over recent notes). Not in scope for the current exploration but a concrete expansion direction.

Source: notes.andymatuschak.org/Evergreen_notes_should_be_densely_linked (accessed 2026-07-02);
incremental formalization pattern from no.silverbullet.plus/incremental-formalization
(accessed 2026-07-02).

### 3.5 Note granularity and atomicity

Matuschak ("Evergreen notes should be concept-oriented"): factor notes by concept, not by author,
book, event, or project. One concept per note. "By discovering connections across books and domains
as you update and link to the note over time." [anecdote] Nick Milo: same principle; MOCs
explicitly link to atomic notes, not to chapter-level summaries.

Luhmann: cards were typically short (one to a few paragraphs). His 90,000 cards represent highly
granular capture.

Implication for engram: memory notes that bundle multiple lessons from the same session limit
retrieval precision (a multi-lesson note can only rank once in a cosine search). This is the
practitioner case for atomic memory notes — one lesson per note — and an argument for splitting
composite notes during the linking sweep. Out of scope for the current exploration; relevant to the
shape of step 3 (retroactive linking sweep) if it includes a note-splitting pass.

Source: notes.andymatuschak.org/Evergreen_notes_should_be_concept-oriented (accessed 2026-07-02);
Ernest Chiang, "Niklas Luhmann's Original Zettelkasten Method," 2025,
ernestchiang.com/en/posts/2025/niklas-luhmann-original-zettelkasten-method/ (accessed 2026-07-02).

### 3.6 MOC curation as a maintenance pattern

Nick Milo's MOC (Map of Content) pattern [anecdote, Medium 2021]: a note containing curated links
to other notes, functioning as an index for a topic, question, or perspective. Key property: many
MOCs can link to the same note (unlike folders). MOCs enable "deliberate positioning" — they are
curated, not auto-generated. dsebastien.net analysis of 8,000 notes [anecdote, 2024]: MOCs average
90.1 links/note vs. 2.9 links/note for regular notes; MOCs are "used more and more" as the vault
grows.

Schmidt's Luhmann hub notes are the historical precedent: curated link lists to core notes,
functioning as "article outlines or book tables of contents."

Implication for engram: a periodic MOC-generation step (LLM pass over the vault to generate
curated link-list notes for recurring topics) would be the practitioner-validated alternative to
hub-by-automation (L6). Unlike L6's taxonomy approach, MOC curation is additive (doesn't modify
existing notes) and doesn't require classification of every note. This could be a post-step-3
deliverable if the link-value exploration finds that curated hubs outperform automated tag hubs.
Likely a Track-A direction.

Source: Nick Milo, Medium 2021 (accessed 2026-07-02);
dsebastien.net, "PKM at scale: analyzing 8,000 notes and 64,000 links" (accessed 2026-07-02);
zettelkasten.de/posts/zettelkasten-hubs/ (accessed 2026-07-02).

### 3.7 Typed-link maintenance overhead — the abandonment signal

Zettelkasten forum link-types discussion [anecdote, ~2023]: MartinBB: "I don't use link types at
all. A link is a link for me, and that is that. When you have several hundred or several thousand
notes there is a lot of labour involved in keeping up such categories." Practitioner bradfordfournier
uses inline commentary (why the link exists, in the note body) rather than formal link types.
Sascha (zettelkasten.de): "link types naturally emerge when you connect knowledge; they are a
posteriori descriptions, not a priori requirements."

The Obsidian Breadcrumbs plugin (typed hierarchy links) has sustained active use (transferred
maintainers, still alive as of 2026-05 per obsidianstats.com) but no large-scale community
evidence of deep typed-taxonomy adoption. The plugin's primary value appears to be up/down/next/prev
(structural hierarchy) rather than semantic typed edges (supports/refutes/updates).

Implication: the human evidence argues for a minimal typed-link strategy (one high-value type:
supersession/refutation) over a rich semantic taxonomy. Any future expansion of L5 beyond
supersession edges should be gated on demonstrated value per type, not adopted as a category system
upfront.

Source: forum.zettelkasten.de/discussion/2023/link-types (accessed 2026-07-02);
obsidianstats.com/plugins/breadcrumbs (accessed 2026-07-02).

### 3.8 Graph view — awareness tool, not retrieval interface

Arthur Perret (2022), "What is the point of a graph view?" (arthurperret.fr) [opinion]: the graph
view reveals abstract structure and prevents siloed thinking, but "links aren't users' top priority
— the spot is occupied by mobile and sync." The graph is a memory aid that surfaces connections
users weren't actively seeking; it is not a primary retrieval interface. Perret notes that
"contextualized links accelerate information retrieval" — implying that uncontextualized link
graphs have limited navigability.

The practitioner consensus: the Obsidian graph view is aesthetically appealing but practically
underused for actual navigation. Search + backlinks panel is the primary daily retrieval path.
[anecdote, multiple community reports]

Implication for engram: `vaultgraph` as a visualization substrate is an awareness tool. Its value
for engram is as a substrate for traversal algorithms (T2–T6), not as a UI feature. The practitioner
evidence that graph navigation is underused does NOT argue against algorithmic traversal — it argues
against exposing the graph as a user-facing navigation surface.

Source: arthurperret.fr/blog/2022-02-13-what-is-the-point-of-a-graph-view.html
(accessed 2026-07-02).

### 3.9 Empirical study: how industry researchers use Obsidian

"How People Manage Knowledge in their 'Second Brains': A Case Study with Industry Researchers
Using Obsidian," INTERACT 2025 (arXiv:2509.20187, Springer LNCS 16111) [study — qualitative]:
key finding: "participants' knowledge retrieval strategy significantly influences how they build and
maintain their content." The abstract does not disclose which specific strategies dominated, but
the framing (retrieval strategy → organization structure, not the reverse) aligns with the broader
practitioner observation that link discipline needs to be calibrated to how you actually recall, not
how you aspire to organize.

This is the only peer-reviewed empirical study found in this beat on PKM tool usage patterns.
Full paper not accessed (PDF binary-encoded); the finding is from the abstract only.

Source: arxiv.org/abs/2509.20187 (accessed 2026-07-02).

# Part 4 — IR/KG literature + Claude-Code-ecosystem memory tools

*Research performed 2026-07-02. All sources cited inline with date. Claim quality labels per MEMORY.md conventions.*

#### 4.1 Learned associations vs explicit wikilinks

Association-Augmented Retrieval (AAR, arxiv 2604.20850, 2026) shows a 4.2M-parameter MLP trained on passage co-occurrence annotations recovers +28.5 pp on hard multi-hop cases where dense retrieval fails, with 3.7ms inference overhead. The core principle: "association is not similarity" — passages copresent in reasoning chains are linked even when semantically distant. **Vision-relevant implication:** if engram accumulates sufficient co-retrieval history (which notes were retrieved together for the same task session), it could compute a co-occurrence signal and train or approximate an association scorer — a zero-link-curation alternative to explicit wikilink maintenance. Deferred because it requires labeled task data or proxy co-retrieval logs that engram does not currently emit. [paper-claimed]

Sources: [Association Is Not Similarity (arxiv 2604.20850)](https://arxiv.org/abs/2604.20850)

#### 4.2 GRAFT — post-retrieval graph repair

GAAMA (arxiv 2603.27910, 2025) includes a post-retrieval corrective layer (GRAFT: Graph Repair by Augmenting Facts & Topology) that diagnoses retrieval failures and surgically adds edges to the KG at the identified gap. The diagnostic pass identifies whether failure was due to (a) missing edges to a needed note, (b) hub dilution, or (c) incomplete fact extraction. **Vision-relevant implication:** after S2 PoC probes identify miss cases where a traversal variant fails to recover a needed note, GRAFT-style targeted edge addition to the L2 fabric is a principled repair step. This is directly relevant to the retroactive linking sweep (step 3 of the original ask) and could be the mechanism for making that sweep intelligent rather than exhaustive. [empirical, 10k-node graph, LoCoMo-10]

Sources: [GAAMA: Graph Augmented Associative Memory for Agents (arxiv 2603.27910)](https://arxiv.org/abs/2603.27910)

#### 4.3 Temporal validity windows on supersession edges

Zep/Graphiti (arxiv 2501.13956, 2025) attaches temporal validity windows (valid_from, valid_until) to every graph edge. For supersession/update relationships: a superseded note's edges carry `valid_until = T_supersession`, making time-sensitive recall possible ("surface the version valid at time T"). **Vision-relevant implication:** L5 supersession edges in engram currently model binary supersession (note A is superseded by note B). A temporal field would enable queries anchored to a specific time to surface the historically valid note rather than always the latest superseder. Relevant to P3 (supersession pairs): the old note may be the correct answer for "what was the state of our understanding at time T." Deferred — requires timestamp metadata on notes and a time-anchored query interface. [practitioner, 94.8% DMR accuracy]

Sources: [Zep: A Temporal Knowledge Graph Architecture for Agent Memory (arxiv 2501.13956)](https://arxiv.org/abs/2501.13956)

#### 4.4 Dynamic alpha routing for PPR (query-type-conditional)

MixPR (arxiv 2412.06078, 2024) uses α=0.6 for QA/reasoning queries and α=0 (pure global PageRank) for summarization, routed by a lightweight LLM classifier. **Vision-relevant implication:** engram's T2 could use higher PPR weight (α≈0.5–0.6) during deep-recall and lower weight (α≈0.2) during glance, since glance already has fewer phrases and flooding the context with PPR-activated nodes is more costly. This is a conditional parameterization of T2 rather than a new variant. Deferred to post-stop-point if T2 survives S3. [paper-claimed, TF-IDF-based graph, large corpora]

Sources: [Mixture-of-PageRanks (arxiv 2412.06078)](https://arxiv.org/abs/2412.06078)

#### 4.5 Concept-mediated nodes as a bridge fabric design

GAAMA builds concept/topic nodes (like "pottery_hobby", "camping_trip") that multiple episode and fact nodes link to, creating cross-cutting traversal paths 30x sparser than entity-centric designs. These concept nodes prevent hub dilution while enabling multi-session associative recall. **Vision-relevant implication:** a hybrid fabric design — L2 note-to-note edges + L6 concept/tag nodes that sit in the graph — would allow PPR to route through tag/topic nodes to reach non-obvious neighbors. This is not currently in the L×T matrix (L6 is tag-only, not used as PPR traversal node). Deferred to post-stop-point as a potential hybrid if L2+T2 and L6-filter both show partial benefit. [empirical, 10k-node graph]

Sources: [GAAMA: Graph Augmented Associative Memory for Agents (arxiv 2603.27910)](https://arxiv.org/abs/2603.27910)

#### 4.6 Obsidian-vault wikilink traversal as context-building (Basic Memory / agentcairn)

Basic Memory (github.com/basicmachines-co/basic-memory) and agentcairn (mcpservers.org) both implement wikilink-aware retrieval over Obsidian-style markdown vaults. Basic Memory's `build_context` tool navigates `memory://` URLs (wikilinks) to assemble related note content; agentcairn uses DuckDB with hybrid BM25 + vector + graph recall. Neither uses PPR. Both treat wikilinks as context-building (content assembly) rather than as a ranking signal. **Vision-relevant implication:** this is essentially engram's T4 or T6 in implementation form — wikilinks expand context, not ranking. The ecosystem data point says wikilink traversal is being deployed as "follow the link and include the content" rather than as a scoring input. If T4/T6 probe shows recovery, this confirms the ecosystem is on the right track with the simpler approach. [practitioner, no controlled benchmarks]

Sources: [Basic Memory (GitHub)](https://github.com/basicmachines-co/basic-memory) · [Memory MCP Servers (mcpservers.org)](https://mcpservers.org/category/memory)

#### 4.7 MCP knowledge-graph server: graph not consulted algorithmically at query time

The official MCP `memory` server (modelcontextprotocol/servers, github.com/modelcontextprotocol/servers/tree/main/src/memory) uses entity-relation-observation triplets stored as JSONL. `search_nodes` is keyword-based (entity names, types, observation text — NOT embedding). `open_nodes` retrieves by name and returns co-relations between requested entities. The graph structure (relation edges) is exposed to the agent for manual reasoning but is NOT traversed algorithmically for discovery. **Implication:** the official MCP baseline is a named lookup with keyword index — far below what engram's query + cosine clustering already does. Engram's current retrieval is substantially more sophisticated than the MCP baseline. [practitioner, official Anthropic/MCP implementation]

Sources: [Knowledge Graph Memory Server (GitHub)](https://github.com/modelcontextprotocol/servers/tree/main/src/memory) · [Glama overview](https://glama.ai/mcp/servers/@modelcontextprotocol/knowledge-graph-memory-server)

#### 4.8 GraphRAG frequently underperforms vanilla RAG on average-case queries

GraphRAG-Bench (arxiv 2506.05690, 2025) was motivated by the observation that "GraphRAG frequently underperforms vanilla RAG on many real-world tasks" and was designed to characterize when it does and does not help. This is consistent with our settled null (note 73, Δ=0 on average-case recall) and GAAMA's +1pp finding. **Vision-relevant implication:** graph retrieval's benefit is concentrated in the tail (hard multi-hop cases) — exactly the miss-population approach this plan uses. The GraphRAG-Bench characterization of when graphs help (graph characteristics, query types) would be valuable for future fabric design decisions. Deferred — paper found, full results not extracted. [empirical, paper-claimed; exact conditions pending full paper access]

Sources: [When to use Graphs in RAG (arxiv 2506.05690)](https://arxiv.org/abs/2506.05690)

#### 4.9 Link prediction does not have downstream retrieval validation

No controlled study was found demonstrating that embedding-based link prediction (TransE, RotatE, etc.) — adding model-predicted edges to a KG — improves downstream RAG retrieval metrics. The link prediction literature optimizes for graph-structural metrics (Hits@k, MRR on held-out triples). A PMC paper (2024) notes "significant drop in quality when evaluated on localized metrics," suggesting predicted edges that look good globally may degrade local retrieval. **Vision-relevant implication:** do not invest in an LP-predicted edge fabric variant without a specific downstream retrieval benchmark to validate it against. This closes out the "embedding-based link prediction" sub-question with a negative finding. [structural inference, no direct empirical validation found]

Sources: [Link prediction using low-dimensional node embeddings: The measurement problem (PMC, 2024)](https://pmc.ncbi.nlm.nih.gov/articles/PMC10895345/)

---
