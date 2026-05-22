# Research brief: tiered memory architecture for engram

## Goal

Design (not yet build) a redesigned memory system for engram that addresses
a felt limitation in the current vault: too much raw signal is discarded
because the only durable artifacts are heavily-distilled "permanent" notes
under a Luhmann-ID Zettelkasten. Cross-session patterns that need multiple
exposures to recognize are being lost.

Produce a design document, not code. The terminal deliverable is a written
spec the user can review and iterate on. If implementation seems obvious,
resist — surface the design decision first.

## Vision (the user's framing — preserve these constraints)

Memory as a lopsided hourglass: very wide at the bottom (raw data), narrowing
upward through distillation, then re-widening slightly into synthesis. Tiers:

- **L0 — references.** URI/file-handle pointers to raw source material:
  Claude Code session JSONL files, code commits, external docs. No content
  stored; just addressable handles. ~1.3 GB / 4,712 sessions exist today
  under ~/.claude/projects/.

- **L1 — stripped source.** Likely-relevant material extracted from L0,
  stripped of formatting cruft and extraneous metadata. Still raw-shaped
  (conversational turns, code, prose) — not yet a "fact." Purely additive.

- **L2 — facts and feedback.** Atomic extracted claims, roughly what the
  current vault calls Permanent notes (without Luhmann IDs). Purely additive.

- **L3+ — synthesis.** MOCs, themes, MOCs-of-MOCs, up to a single top-level
  synthesis of memory. **Mutable** — regularly reassessed as L2 grows.
  Conflict resolution lives here. Conflict at L2 surfaces both; L3 reconciles.

L0–L2 are append-only. L3+ is regenerated. Luhmann IDs are dropped — they
don't survive a mutable layer.

### MOC dimensions

To prevent arbitrary connection explosion, MOCs aggregate along a constrained
set of dimensions. Initial candidates:

- time
- reference metadata (file types, authors, locations)
- subject / predicate / object
- situation / behavior / impact / action

A single L2 note may appear in MOCs along multiple dimensions.

### Recall as deep-dive

A user query "what do we know about X" with no direct L3/L2 hit should not
return empty. The system descends — through related L3 nodes, into related
L2 facts, into L1/L0 if warranted — searching for an un-noticed pattern.
On finding one, it writes new L2 facts and _may_ cascade upward into L3+
regeneration. Cascade depth depends on impact; small patterns don't redraw
the top of the hourglass.

### Retrieval substrate

The current recall pipeline reaches notes by anchors, recent activity, and
basename-driven cascade — effectively exact-match graph traversal seeded
by a few entry points. That's the bottleneck this redesign needs to break.

The target is **fast semantic lookup that does not require exact matches
and does not call an LLM from inside the binary.** Hard constraints
inherited from the engram codebase:

- Pure Go, no CGO (rules out ONNX runtime, FAISS bindings, etc.).
- No LLM calls from the binary — keep `engram recall` snappy and offline.
- External embedding API at _write_ time is acceptable; the binary should
  query precomputed vectors at read time.

Design dimensions to explore:

- **Vector source.** External embedding API (Voyage, OpenAI, Cohere) called
  by `/learn` at write time and cached alongside the note. Re-embedding
  cost on model changes.
- **Vector store.** Flat files + cosine in Go? sqlite-vec / sqlite-vss?
  An HNSW Go library (e.g., `hnsw-go`)? Tradeoffs: build complexity,
  recall@k quality, query latency at vault size of 10²–10⁴ notes.
- **Lexical complement.** TF-IDF / BM25 in pure Go as a cheap baseline or
  hybrid signal. Reciprocal rank fusion between lexical and dense.
- **Per-tier indexing.** L1 (raw-ish, long) and L2 (short atomic claims)
  embed differently — chunking strategy, summary embeddings, or both.
  L3 MOCs may benefit from being embedded as their constituent L2 set
  rather than their prose.
- **Query expansion.** The recall skill already runs explicit + situational
  queries; embeddings let the situational baseline expand cheaply. What
  does the cascade look like when the frontier is "k nearest" instead of
  "wikilinks from this note"?
- **Freshness vs. recall@k.** Tradeoff between always-precomputed vectors
  (stale on edits) and lazy re-embed (slower writes). Single-stage `/learn`
  already pays an external-API cost; piggyback?

### TDD for memories

Every L2 and L3+ entry carries 2–3 tests, modeled on the TDD pattern in the
existing writing-skills skill. Each test answers:

1. When would we expect this memory to be relevant?
2. What would the LLM plausibly search for in that situation?
3. Does that search actually surface this memory?
4. Does the surfaced memory lead to better behavior on the first pass?

A memory that fails its own tests is a candidate for revision or deletion.

## Current state (read these before designing)

- Repo: ~/repos/personal/engram (and worktrees)
- Vault: ~/repos/personal/agent-memory (Permanent/, MOCs/, Fleeting/)
- Active skills: skills/recall/SKILL.md, skills/learn/SKILL.md, skills/dev,
  skills/audit
- Recent collapse: the Fleeting tier was just removed; capture/promote became
  single-stage /learn writes. Filenames standardized to
  `<luhmann-id>.<YYYY-MM-DD>.<slug>.md`. Read recent commits and the notes
  under tag "tier-collapse" to understand what was just simplified — the
  redesign should not silently re-introduce complexity that was deliberately
  removed.
- engram CLI: `engram recall`, `engram transcript --from --to`, vault path
  via `--vault` or `ENGRAM_VAULT_PATH`. Read `cmd/engram/*`,
  `internal/recall/*`, `internal/cli/*`.
- Session data: `~/.claude/projects/`, ~1.3 GB, JSONL per session.

## Motivating example: "you did it but you don't remember"

Concrete case from 2026-05-13 that the redesign should make tractable.

Context: during the `engram update` implementation, two uncommitted edits
(`docs/issue-612-plan.md` and `skills/recall/SKILL.md`) got stashed and
then accidentally dropped during an impgen module-rename detour. The user
asked me to recover them. My first instinct — "I never saw the diff
content in context, so it's unrecoverable" — was wrong. The user
corrected: try reading session JSONL files, since *I* was the only editor
and the edits had to be in *some* prior session's transcript.

What worked: grep `~/.claude/projects/<encoded-path>/*.jsonl` for the
file path, find tool-use records of `Edit`/`Write` with their
`old_string`/`new_string` payloads, correlate with `tool_result` records
to confirm which ones succeeded, replay the successful Edits onto the
current file. Three Edits in session `a22ad7f7` and three more in
session `677d4acf` — all six recovered verbatim.

Why this matters for the design:

- The information was there the whole time; it was just outside the
  current conversation's context window. The current vault would never
  surface it because the vault only contains heavily-distilled Permanent
  notes — raw session content lives in L0 and is invisible to recall.
- The recovery method was a manual L0 traversal: a human-prompted grep
  over JSONL. The system should be able to do this kind of
  deep-into-L0 dive on its own when a query (or a situation) implies
  "the answer might exist in a prior session even though no L2 fact
  captures it yet."
- This is exactly the "recall as deep-dive" cascade pattern from the
  vision: query → L3/L2 misses → descend through L1 → reach into L0,
  searching for an un-noticed pattern. On finding one, *write a new L2
  fact* so the next time the system doesn't need to re-dive.

What the design should specify, prompted by this example:

- **L0 indexing for file-path / symbol queries.** Not full semantic
  embedding — just a fast inverted index over file paths, tool-use
  names, and `old_string`/`new_string` snippets in session JSONL, so a
  query like "edits to `skills/recall/SKILL.md`" returns a ranked list
  of sessions with timestamps. Without this, every dive is a linear
  scan of ~1.3 GB.
- **Tool-result correlation.** Edits whose `tool_result` records an
  error didn't actually apply. The recovery loop has to distinguish
  succeeded-and-applied from attempted-but-rejected. The L0→L1 selector
  function needs to know about tool-result semantics, not just
  tool-use payloads.
- **Provenance survives into L1/L2.** A note synthesized from L0 dives
  must carry the source session ID and tool-use ID, so the system can
  re-derive (or contradict) later. Otherwise the next dive doesn't know
  which sessions it has already mined.
- **Triggers for write-back.** Should every successful L0 dive write an
  L2 fact? Probably not (creates churn). But "I had to dive twice for
  the same answer" is a strong signal that an L2 note is overdue.

This example also illustrates a subtler failure mode the design needs
to name: the system *appearing* to lack information when in fact it
has it but in the wrong tier. A user asking "remember when we…" should
not be told "no" just because L2/L3 didn't hit — that's a recall bug,
not a memory gap.

## Context as a harvestable substrate

The prior section framed the recovery as an L0 indexing problem. There's a
parallel question the design should treat as a first-class option:

**The agent's live context window is itself a tier — finer than L0, coarser
than L1, and *attention-filtered* in a way L0 isn't.** Raw session JSONL
contains everything; the live context contains what the agent currently
judges relevant. Capturing context at the right moments could produce
higher-signal L1 segments than mechanical L0 selection ever will.

But context is also messy: stale tool output, abandoned hypotheses, system
reminders. A bulk dump would be worse than JSONL. The valuable artifact is
the agent's *segmentation* of its own context — "these 4 tool calls
together established X," "this user correction reframed Y" — which can't
be extracted mechanically. It requires the agent to emit it.

### Three production paths to evaluate

In increasing order of agent involvement:

1. **Mechanical context snapshots.** Harness hook (PreCompact, Stop) dumps
   the live context window. Cheap, mostly redundant with JSONL because
   compaction is rare and Stop already triggers `/learn` today. Probably
   not worth its own tier.

2. **Agent-emitted L1 segments at task boundaries.** On task close, the
   agent writes N small L1 notes: "Working on X, I found Y via Z; here
   are the entities and links," with structured provenance (session ID,
   file paths, tool-use IDs). Distinct from L2 in gating: `/learn` writes
   L2 only when content passes Recurs/Activity/Knowledge gates. L1 is
   everything below that gate, with a softer test: "would a future query
   about this *situation* benefit from finding this verbatim?"

3. **Live inline indexing.** Agent emits "this matters" markers as it
   works (a small `index(entities, situation, link-back)` tool call),
   not just at task end. Each one is cheap; the index is built
   incrementally. Avoids the end-of-task amnesia problem where what was
   load-bearing 80 turns ago is no longer salient enough to be captured.

These aren't mutually exclusive — (1) is a fallback when the agent
forgets, (2) is the workhorse, (3) catches what (2) would lose to
attention decay. The design should pick a default and name when the
others kick in.

### The actual blocker is cross-session dedup and linking

Capture is the easy half. The hard half:

- Session A writes `engram-update-walkup-bug` describing a finding. Three
  days later, session B is investigating the same bug area and re-derives
  the same finding because the system never surfaced A's note.
- Or worse: B writes a contradicting note without seeing A's, and now L2
  has two facts that need L3 reconciliation that no one asked for.

Today's vault links by filename basename — a wikilink to `[[foo]]` resolves
iff `foo.md` exists with the same slug. Renames break it, fuzzy matches
don't exist, semantic equivalence is invisible. With L1 capturing
hundreds of segments per session, naive linking will produce either
duplicate sprawl or arbitrary divergence depending on whether the agent
happened to land on the same slug.

Design must specify:

- **Stable handles that survive content rewrites.** Content hash? UUID
  plus title? Vector centroid?
- **A dedup pass on write.** When an L1 segment is created, the system
  must check whether a sufficiently-similar segment already exists, and
  link/merge rather than duplicate. This is the same operation the
  agent's `/learn` skill performs today via the Recurs gate, but at L1
  it has to run automatically — there's no human in the loop for every
  small segment.
- **Backward links.** If A merges into B, anything that linked to A's
  handle has to follow. The Luhmann-ID system avoided this by being
  immutable; once L3 is mutable, this becomes a graph-maintenance
  problem.

This is the same hard problem as the existing "identity without Luhmann"
question — just pushed down a tier. Resolving it at L1 likely resolves
it at L3 too.

## Open questions to grapple with

These are what the design must resolve. Do not pre-answer them in this brief
— answer them in the design doc, with rationale or with the experiment
needed to resolve each.

1. **L1 selection function.** What promotes a session segment from L0 to L1?
   Pre-extraction at session end? Post-hoc on demand? A scheduled triage
   pass? Agent-emitted from live context (per the "Context as a harvestable
   substrate" section)? What's the cost ceiling, and which of the three
   production paths is default?

2. **Identity without Luhmann.** If L3 is mutable, what's the stable
   reference handle for an L2 note cited by an L3 MOC? Slug? Content hash?
   UUID? How do wikilinks resolve after L3 regeneration moves things?

3. **Regeneration triggers and cadence.** When does L3+ rebuild? On every
   L2 write (expensive)? Batched? Drift-detected? On query?

4. **MOC dimension orthogonality.** Are the four dimensions independent
   indices (a note appears in N MOCs)? Or does each MOC commit to one
   dimension? How do MOCs-of-MOCs aggregate across dimensions?

5. **Cascade write-path.** When deep-dive discovers a pattern, what
   exactly gets written? New L2 only? Or also a draft L3 update? What's
   the trust model — is the LLM's mid-query extraction durable, or does it
   need a confirmation step?

6. **Conflict semantics.** Two L2 facts disagree. L3 resolves "to what" —
   a third synthesizing note that cites both? A scored winner? Both
   preserved with a contradiction flag, surfaced together at recall?

7. **Test storage and execution.** Where do the 2–3 tests per memory live?
   Same file? Sidecar? Index? What runs them — a separate `engram test`
   command? How often? What's the failure consequence — auto-archive,
   flag-for-review, or noop until human triage?

8. **Top-level synthesis.** What does the single root MOC look like?
   Generated artifact, regenerated each session? Living document edited
   by hand and validated by the system? Something else?

9. **Recall cascade redesign.** Current recall expands a frontier across
   one tier (Permanent/MOCs/Fleeting). With four+ tiers, where does the
   cascade start, and what makes it descend? Cost model — descending to
   L0 reads raw JSONL, which is expensive; what gates that?

10. **Migration.** The existing vault has Luhmann-IDed Permanent notes
    and several MOCs. How do they ma tp onto L2 vs L3? Is migration
    automatic, scripted, or done by re-running /learn against L0?

11. **Failure modes.** What does this system look like when broken —
    L1 grows unboundedly? L3 thrashes on every write? Tests pass but
    behavior doesn't improve? Spec out the diagnostics.

12. **Retrieval substrate choice.** Pick a concrete combination of
    embedding provider + vector store + lexical fallback. Justify against
    the constraints (pure Go, no in-binary LLM, vault size 10²–10⁴ today,
    growing). Name the recall@k target and the latency budget for
    `engram recall`. Specify what happens when the embedding API is
    unreachable (graceful degrade to lexical-only? hard fail?).

## Prior art to survey

- Zettelkasten — Niklas Luhmann's original, plus Sönke Ahrens (_How to Take
  Smart Notes_). Confirm what's being kept vs. dropped.
- LLM long-term memory papers: MemGPT (Letta), Generative Agents (Park et
  al. 2023, the Smallville paper) — both stratify memory by abstraction
  level with reflection/synthesis cycles. Compare their tiering.
- OS memory hierarchies (L1/L2/L3 cache, virtual memory) — borrowed
  terminology suggests borrowed mechanics; check what carries over and
  what doesn't (cache eviction has no L3-style synthesis analogue).
- RAG and re-ranking literature for the L0→L1 selection problem.
- The existing engram codebase and the writing-skills TDD discipline for
  the testing-of-memories pattern.
- Embedding + vector-store options compatible with the pure-Go / no-CGO
  constraint: sqlite-vec, sqlite-vss, `github.com/coder/hnsw`,
  `philippgille/chromem-go`, flat-file cosine. External embedding
  providers: Voyage (voyage-3, voyage-code-2), OpenAI text-embedding-3,
  Cohere embed-v3.
- Lexical retrieval baselines: BM25, TF-IDF (already referenced in the
  engram CLAUDE.md as the fallback), reciprocal rank fusion for
  hybrid retrieval.
- Late-interaction and re-ranking: ColBERT, cross-encoder re-rankers —
  probably out of scope for the in-binary path but worth naming.

## Deliverable

A design document at
`docs/superpowers/specs/YYYY-MM-DD-tiered-memory-design.md` that:

- Restates the vision in your own words (verify understanding).
- For each of the 11 open questions, proposes an answer with rationale,
  OR explicitly defers it with the experiment needed to resolve it.
- Sketches the data layout: directories, file formats, naming, indexes.
- Sketches the lifecycle: write paths, regeneration paths, recall paths,
  test execution.
- Names what's deliberately out of scope (don't re-litigate decisions the
  recent tier-collapse settled).
- Identifies the smallest first slice that could be built and measured.

Do not write code yet. Do not start the implementation plan. The next
session, after the user reviews this design, will run writing-plans.

## Working style

- Read source code and existing notes before guessing at formats or
  contracts.
- Ask one clarifying question at a time when the user is available.
- When the user isn't available, make the most defensible choice, name it
  as a choice (not a finding), and continue.
- Treat the vision section as constraints, not suggestions. If a
  constraint seems wrong, surface it explicitly before designing around it.
