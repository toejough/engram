# Memory-System Review — plan

> **For agentic workers:** research + synthesis plan (no code). Tasks are delegated to subagents per
> the route skill; the orchestrator synthesizes the report. (Amended post-Gate-A: added Task 1c
> conversational history — the ask's first-listed source; reordered Honest Gaps before External
> landscape; specified extraction rules the reviewers flagged as judgment calls.)

**Ask (Joe, 2026-07-01, verbatim):** "do a review of our conversational history, our docs, what we've
built, and of competing ways to capture, distill, recall, refine, and build upon memories in an llm
ecosystem. Present a report of what you find, particularly how well or poorly we've achieved our
goals, and what else we might want to explore to improve upon what we've built."

**Deliverable:** `docs/design/2026-07-01-memory-system-review.md` — the report — PLUS an
in-conversation presentation of its substance (the ask says *present*, not just commit). Location
convention: plans live in `docs/superpowers/plans/`; dated analyses/reports live in `docs/design/`.

**Vocabulary rule (Gate A):** the ask's five verbs (capture, distill, recall, refine, build-upon) are
the *report's comparison axes*; the repo's architecture speaks in four flows (recall, learn, please,
update — C1 diagram). §2 opens with an explicit mapping table (capture=learn's ingest; distill=learn's
crystallization + recall Step 2.5; recall=recall skill + `engram query`; refine=amend/resituate/
activate/prune + please's gates; build-upon=Step-3 synthesis + linking + please) and the report uses
the five verbs only through that mapping, so it stays anchored to the C1 taxonomy.

**Report structure (fixed; Honest Gaps deliberately precedes the external landscape — the ask's
emphasis is the self-assessment):**
1. **Goals** — what engram set out to do, as stated over time. Sources: `README.md`, ROADMAP preamble
   + tracks, early design docs; CLAUDE.md only for stated aspirations (it is process rules, not
   design goals — scope it as such). Each goal dated and cited.
2. **What we built** — the shipped system per flow (with the five-verb mapping table up front):
   binary mechanisms, skills, guidance, deployment. Concise; cites code/docs.
3. **Goals scorecard** — labeled table with units: goal × verdict (ACHIEVED / PARTIAL / REFUTED /
   UNMEASURED) × evidence (eval, date, numbers). Evidence base ("the ledger") = vault note 99 + the
   Task-1b note set (95, 98, 91, 103, 135, 73, 76, 68, 82, 119, 120, 149–154) + their cited design
   docs; numbers ALWAYS from the design docs (notes are the index, docs are the source). Superseded
   results shown post-correction (e.g. the 350s→190s recall mislabel,
   `docs/design/2026-06-25-recall-cost-isolation.md`).
4. **Honest gaps** — where we fell short or the evidence is unflattering. Draws on the same ledger
   set: cost/speed refuted on easy builds (95/91); behavioral value unproven outside idiosyncratic
   content (98/99); the structural recall-miss (82: recall fires once, never re-checks mid-synthesis
   levers); 68% of real failure cues uncovered (failure-mining); crystallization-quality levers
   repeatedly delivery-inert (119/120/153); procedure tax; plus anything Task 1c surfaces.
5. **External landscape** — competing approaches, qualitative/architectural (NOT benchmarked), with
   the five verbs as REQUIRED columns per system/technique — "no meaningful mechanism" is an explicit
   cell value, not an omission (refine/build-upon are where engram is most differentiated; they must
   not collapse away). Vendor performance claims are quoted as-stated and labeled "(vendor-claimed,
   unverified)"; the no-bare-% rule applies to OUR results only.
6. **Exploration candidates** — ranked, evidence-gated: each = {gap addressed, evidence for headroom,
   cheapest validation, deflation risk}. **Deflation risk** = the probability the lever nulls on the
   deciding delivery metric given our base rate (every crystallization-quality lever to date
   deflated — notes 119/99). Collision rule: for each candidate, check ROADMAP Track 0/A/B + the
   open-issue list (via `gh issue list --state open` — the set is sparse, includes #637; never a
   numeric range) — if it matches shipped work cite the ship, if parked cite the park rationale
   instead of re-proposing (note 143).

## Tasks

### Task 1 — Internal evidence synthesis (delegated, 3 parallel subagents, sonnet)
- **1a. Goals + build trail:** read `README.md`, `CLAUDE.md` (aspirations only), `docs/ROADMAP.md`
  (preamble, tracks, `## Done`, `## Shipped`), `docs/architecture/c1-system-context.md`, and ALL
  `docs/design/*.md` sorted by filename date-prefix (currently 21). ROADMAP Done entries are
  narrative — follow their design-doc backlinks and `git log --oneline` for mechanisms + commit
  hashes. Output: dated goals list; dated shipped list (mechanism + commit); dated parked/reverted
  list with rationale.
- **1b. Eval verdicts trail:** read the eval-bearing design docs + `dev/eval/traps/README.md` +
  `RESULTS.md`, using vault notes 99, 95, 98, 91, 103, 135, 73, 76, 68, 82, 119, 120, 149–154 as the
  index (read each via `engram show`; take NUMBERS from the design docs). Output: table rows — claim
  tested, arms, n, result numbers, verdict, date, doc citation. A "verdict" = the doc's stated
  conclusion + its numbers; flag any verdict later corrected/superseded (cite both docs).
- **1c. Conversational history (the ask's first source):** query the chunk index directly
  (`engram query` with phrases like "what engram should do", "the goal of memory", "why we built",
  "decided not to", "next for engram") and read surfaced chunks via `engram show-chunk`; additionally
  sample 3–5 session transcripts across project phases (list
  `~/.claude/projects/-Users-joe-repos-personal-engram/*.jsonl`; mtime approximates but does not
  guarantee session order — the chunk-index query is the primary source; pick earliest/middle/recent
  by best available signal) and skim for: in-session goal statements, informal decisions that never
  became docs, usage patterns, abandoned directions. Output: dated list of conversational findings
  NOT already in docs (explicitly marked "transcript-only").

### Task 2 — External landscape research (delegated, 2 parallel subagents, sonnet + web)
- **2a. Systems survey:** MemGPT/Letta, Mem0, Zep/Graphiti, LangMem, ChatGPT memory, Claude Code
  auto-memory/CLAUDE.md, notable open-source agent-memory frameworks. Per system: mechanism for EACH
  of the five verbs (explicit "none" allowed); storage substrate; retrieval mechanism; what's
  measured vs claimed (label vendor claims).
- **2b. Techniques survey:** RAG+rerank, GraphRAG, HippoRAG, reflection (Reflexion/Generative
  Agents), skill/procedure libraries (Voyager), A-Mem, MemoryBank, episodic-memory work,
  sleep-time/offline consolidation, context compaction/editing. Per technique: mechanism (noting
  which of the five verbs it serves — explicit "none" allowed, mirroring 2a), evidence quality, and
  applicability on three named axes — (i) local-first CLI agent fit (no external service
  dependency), (ii) skills+binary architecture fit, (iii) addresses a gap our evidence says is real
  (not an axis we've refuted/solved).
- Both: prefer primary sources/papers/docs; note publication dates.

### Task 3 — Synthesis (orchestrator)
- Merge 1a+1b+1c into §1–4. Merge 2a+2b into §5 with the required five-verb columns.
- Derive §6 per the collision rule above.
- Adversarial self-check on favorable claims (note 96) before Gate C.

### Task 4 — Gates + ship
- Gate B (design-fit): **N/A — no code is changed; the artifact is a report** (named per protocol;
  the report itself is gated by C).
- Gate C concrete pass checks: (a) every historical claim carries a date + doc/note citation;
  (b) each §3 verdict is traceable to a Task-1b row; (c) §5 comparisons are architectural — no
  unlabeled vendor numbers, no superiority claims; (d) no §6 candidate duplicates shipped/parked/
  filed work uncited. Plus the standard relevance + clarity/cohesion angles.
- Gate D: commit message. Commit report + plan. Present the report's substance in-conversation.

## Constraints
- Results as labeled tables with units; no bare percentages without n (ours; vendor claims labeled).
- Every historical claim carries a date + citation; superseded results shown corrected.
- External comparison is architectural, not benchmarked — the report says so.
- Exploration list is evidence-gated; base rate honesty: most quality levers here have deflated.
- No repo-wide tooling; scope-check the diff before commit.
