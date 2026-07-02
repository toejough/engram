# Compound Engineering: Findings for Engram (2026-07-02)

> Vision-relevant findings from the S0 link-value research that are out-of-scope for the link-value
> exploration itself but engram-relevant. Parked here per note 154 (parked ≠ unrecorded).
> Sources surveyed 2026-07-02; vendor/blog claims labeled (claimed, unverified).

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
