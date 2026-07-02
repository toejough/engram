# Memory-System Review — plan

> **For agentic workers:** research + synthesis plan (no code). Tasks are delegated to subagents per
> the route skill; the orchestrator synthesizes the report.

**Ask (Joe, 2026-07-01, verbatim):** "do a review of our conversational history, our docs, what we've
built, and of competing ways to capture, distill, recall, refine, and build upon memories in an llm
ecosystem. Present a report of what you find, particularly how well or poorly we've achieved our
goals, and what else we might want to explore to improve upon what we've built."

**Deliverable:** `docs/design/2026-07-01-memory-system-review.md` — the report — PLUS an
in-conversation presentation of its substance (the ask says *present*, not just commit).

**Report structure (fixed):**
1. **Goals** — what engram set out to do, as stated over time (README, CLAUDE.md, early design docs,
   ROADMAP tracks). Each goal dated and cited.
2. **What we built** — the shipped system per pipeline stage (capture → distill → recall → refine →
   build-upon): binary mechanisms, skills, guidance, deployment. Concise; cites code/docs.
3. **Goals scorecard** — labeled table with units: goal × verdict (ACHIEVED / PARTIAL / REFUTED /
   UNMEASURED) × evidence (eval, date, numbers). Built strictly on the adversarially-verified ledger
   (vault note 99 + successors); every result carries its date + system-state; corrected results
   (e.g. the 350s→190s recall mislabel) shown post-correction.
4. **External landscape** — competing approaches compared per pipeline stage, qualitative/architectural
   (NOT benchmarked): MemGPT/Letta, Mem0, Zep/Graphiti, LangMem/LangGraph, ChatGPT/Claude native
   memory, RAG/GraphRAG/HippoRAG, reflection agents (Reflexion, Generative Agents), skill libraries
   (Voyager), A-Mem/MemoryBank, sleep-time compute, context compaction. What each does per stage that
   engram does differently — and whether our measured evidence predicts the difference matters.
5. **Honest gaps** — where we fell short or the evidence is unflattering: cost/speed refuted on easy
   builds; behavioral value unproven outside idiosyncratic content; the structural recall-miss (note
   82: recall fires once, never re-checks mid-synthesis levers); 68% of real failure cues uncovered;
   crystallization-quality levers repeatedly delivery-inert; procedure tax.
6. **Exploration candidates** — ranked, evidence-gated: each names the measured gap it addresses, the
   cheapest validation path, and its deflation risk given our base rate. Cross-checked against
   ROADMAP + open issues so nothing shipped/parked is re-proposed as fresh (note 143).

## Tasks

### Task 1 — Internal evidence synthesis (delegated, 2 parallel subagents, sonnet)
- **1a. Goals + build trail:** read `README.md`, `CLAUDE.md`, `docs/ROADMAP.md` (incl. Done section),
  `docs/architecture/c1-system-context.md`, and the `docs/design/*.md` chronology (21 files); read
  `git log --oneline` for ship dates. Output: dated list of stated goals; dated list of what shipped
  (mechanism + commit); dated list of what was parked/reverted and why.
- **1b. Eval verdicts trail:** read the eval-bearing design docs + `dev/eval/traps/README.md` +
  `RESULTS.md` and the vault ledger notes (99, 95, 98, 91, 103, 135, 73, 76, 68, 82, 119, 120, 149,
  150–154). Output: table rows — claim tested, arms, n, result numbers, verdict, date, doc citation.
  Flag any verdict later corrected/superseded (cite both).

### Task 2 — External landscape research (delegated, 2 parallel subagents, sonnet + web)
- **2a. Systems survey:** MemGPT/Letta, Mem0, Zep/Graphiti, LangMem, ChatGPT memory, Claude Code
  auto-memory/CLAUDE.md, notable open-source agent-memory frameworks. Per system: how it does capture
  / distill / recall / refine / build-upon; storage substrate; retrieval mechanism; what's measured
  vs claimed.
- **2b. Techniques survey:** RAG+rerank, GraphRAG, HippoRAG, reflection (Reflexion/Generative
  Agents), skill/procedure libraries (Voyager), A-Mem, MemoryBank, episodic-memory work,
  sleep-time/offline consolidation, context compaction/editing. Per technique: mechanism, evidence
  quality, applicability to a CLI-agent ecosystem like ours.
- Both: prefer primary sources/papers/docs; note publication dates; mark marketing claims as such.

### Task 3 — Synthesis (orchestrator)
- Merge 1a+1b into the scorecard (§1–3, §5). Merge 2a+2b into §4 with the per-stage comparison.
- Derive §6 exploration candidates: each = {gap addressed, evidence for headroom, cheapest test,
  deflation risk}. Rank by (measured headroom × cheapness). Cross-check ROADMAP/issues list
  (#642–#667) so no candidate is already shipped/parked/filed — cite the issue instead.
- Adversarial self-check on favorable claims (note 96) before Gate C.

### Task 4 — Gates + ship
- Gate B (design-fit): **N/A — no code is changed; the artifact is a report** (named per protocol;
  the report itself is gated by C).
- Gate C: relevance + clarity/cohesion on the report (+ ROADMAP if a pointer is added).
- Gate D: commit message. Commit report + plan. Present the report's substance in-conversation.

## Constraints
- Results as labeled tables with units; no bare percentages without n.
- Every historical claim carries a date + citation (doc or note); superseded results shown corrected.
- External comparison is architectural, not benchmarked — say so in the report.
- Exploration list is evidence-gated; base rate honesty: most quality levers here have deflated.
- No repo-wide tooling; scope-check the diff before commit.
