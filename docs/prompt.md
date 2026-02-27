# Rebuild Prompt: Self-Correcting Memory System for LLM Agents

You are rebuilding a memory management system from the ground up. The system's purpose: make LLM agent interactions get the right answer faster, with fewer tokens and fewer corrections, by learning from every interaction and self-correcting over time.

This means actively proposing improvements to the agent's environment — suggesting updates to skills, CLAUDE.md files, hooks, rules, and deterministic tooling based on measured effectiveness. The system doesn't just retrieve memories; it identifies what's working, what's failing, and what should change.

This replaces the existing projctl memory system — a Go library with ONNX embeddings, SQLite storage, and a 3-tier architecture (embeddings, skills, CLAUDE.md). The existing system has sophisticated plumbing for moving content between tiers but fails to measure whether surfaced memories actually help. You are designing one unified system that fixes the fundamental problems while preserving what worked.

**The core thesis:** Content quality matters more than mechanical sophistication. A memory surfaced 100 times and ignored every time looks identical to one surfaced 100 times and followed every time. The rebuild tracks impact — not just frequency — and uses that signal to self-correct.

**Form factor:** A Claude Code plugin. Hooks for deterministic enforcement. Skills for procedural knowledge. CLAUDE.md management for always-loaded guidance. A Go binary for expensive computation (TF-IDF, hashing, scoring). Pure Go, no CGO.

**Autonomy boundary:** Memories are the system's internal state — the system has full autonomy to surface, write, edit, merge, prune, refine, and score them without asking permission. All memory operations are auditable (the system logs what it changed and why), but they don't require user approval. Skills, CLAUDE.md, and hooks are user-committed artifacts — the system proposes changes to these, but the user approves before they take effect.

---

## Bootstrap (do this FIRST, before any other work)

This prompt does not survive session boundaries. The ONLY things that persist are files on disk and CLAUDE.md. Before doing any rebuild work, verify this instruction exists in project CLAUDE.md under `<!-- MANUAL ADDITIONS START -->`:

```markdown
## Active Work: Memory System Rebuild

When user says "continue", "resume", or similar without other context:
1. Read `docs/state.toml` for current cursor position and next action
2. Read `docs/prompt.md` for full process instructions
3. Resume from the cursor's `next_action` — do NOT ask "what would you like to work on?"
4. Announce what layer/group you're in and what you're about to do

State is persisted write-ahead in `docs/state.toml`. Update after every substantive interaction.
See the specification-layers skill for the TOML format.
```

If it's not there, add it. This is the anchor that makes "continue" work across sessions.

## Session Structure

This work is organized as a state machine. You will progress through phases, and you may be interrupted at any point by /clear, /compact, or /exit. You must be able to resume from a "continue" message.

**Phases:**
1. LESSONS — Extract lessons from existing systems, produce `docs/lessons.md`
2. USE CASES — Interview the user to discover and document all use cases, produce `docs/use-cases.md`
3. REQUIREMENTS — Produce requirements with traceable IDs, produce `docs/requirements.md`
4. DESIGN — Interaction design: how users invoke the plugin, what they see, what hooks fire, how CLAUDE.md is managed. Includes CLI output mocks, hook interaction flows, skill interaction transcripts, and case study walkthroughs. Produce `docs/design.md`
5. ARCHITECTURE — System architecture: component boundaries, data flow, plugin structure, TF-IDF integration, hook design. Each decision traceable to requirements. Produce `docs/architecture.md`

Design before architecture — you need to know how people interact with the system before deciding how to build it.

**Persistence rule (write-ahead, not write-on-exit):** After every substantive interaction (node transition, decision validated, flag set/cleared, interview question answered), immediately update `docs/state.toml`. Do NOT defer to session end — the session can die at any time (/exit, /clear, crash) and you will NOT get a chance to save. The TOML format is defined in the specification-layers skill. The cursor's `next_action` must be specific enough that a fresh session with NO context can start immediately.

When you receive "continue" as your first message, read `docs/state.toml` and `docs/prompt.md`, then resume from the cursor's `next_action`. Announce what layer/group you're in and what you're about to do. Do NOT ask what the user wants to work on.

**Lesson discovery:** Lessons don't only emerge in Phase 1. During every phase, actively check whether your current work reveals new lessons:
- A design decision that contradicts or extends an existing lesson
- Source material that reveals a pattern or failure not yet captured
- A user correction that exposes a blind spot in the existing lessons
- An architecture trade-off that surfaces a new constraint

When you discover a new lesson, append it to `docs/lessons.md` immediately and note which phase surfaced it. Process and coordination lessons are equally important as technical lessons — do not merge them into technical principles or drop them.

**Phase exit criteria:**
- LESSONS: A `docs/lessons.md` file exists capturing distilled lessons from existing systems, validated by the user. This is a living artifact — append new lessons as they emerge in later phases.
- USE CASES: A `docs/use-cases.md` file exists with numbered use cases, each validated by the user.
- REQUIREMENTS: A `docs/requirements.md` file exists with REQ-N IDs, each traceable to one or more use cases.
- DESIGN: A `docs/design.md` file exists with case study walkthroughs, hook interaction flows, CLAUDE.md management UX, skill interaction transcripts, and error/edge case interactions for each use case that involves user interaction. Validated by the user.
- ARCHITECTURE: A `docs/architecture.md` file exists with ARCH-N decisions, each traceable to requirements. Includes plugin structure, TF-IDF integration plan, and data model.

---

## Lessons From Existing Systems

These are hard-won lessons from building projctl (15+ features across memory, skills, speckit, orchestration, traceability) and traced (spec-driven development tool). Read these before doing anything. Append new lessons to `docs/lessons.md` as they emerge.

### What worked

Every lesson below should be evaluated against the problem statement: does it help the agent get the right answer **faster** (fewer turns), **cheaper** (fewer tokens), or with **fewer corrections** (user doesn't have to re-teach or override)? A lesson that doesn't demonstrably contribute to at least one of these outcomes is noise, not knowledge.

1. **Hybrid search with multiple complementary signals.** Two or more retrieval signals merged by Reciprocal Rank Fusion. The existing system used BM25 + vector (ONNX); the rebuild replaces ONNX vectors with TF-IDF (per constraint #4), so the pipeline becomes BM25 + TF-IDF + RRF or similar. The principle that worked: multiple signals catch what any single signal misses, and RRF merges without score normalization headaches.

2. **Session extraction with confidence tiers.** Extracting learnings from conversation transcripts with A/B/C confidence tiers (A = user explicitly stated, B = agent inferred and visible during session — user had opportunity to correct but didn't, C = agent inferred post-session from transcript patterns — user never saw it). Confidence governs how aggressively the memory is surfaced.

3. **Semantic enrichment of memories.** LLM extraction at write-time creates structured metadata: observation_type, concepts, principle, anti_pattern, rationale, enriched_content. Dramatically improves retrieval quality compared to raw text storage.

4. **Skill compilation from memory clusters with RED/GREEN testing.** When 3+ similar memories cluster, compile them into a skill (procedural knowledge). Test the skill with RED (should-trigger) and GREEN (should-not-trigger) scenarios before deployment. Catches bad skills before they affect users.

5. **Hook infrastructure for deterministic enforcement.** Shell-based hooks (PreToolUse, PostToolUse, Stop, SessionStart) that exit 0 (pass) or exit 2 (block with feedback). Deterministic — no model variability. The strongest enforcement tier.

6. **Deterministic hashing for change detection.** SHA-256 truncated to 8 hex chars. Cheap, deterministic, catches stale content without LLM calls. If nothing changed, skip the expensive check.

7. **TF-IDF candidate pruning before LLM calls.** Pure Go TF-IDF (~300 lines, no dependencies) narrows candidates to the top-N most textually similar before sending to LLM. Sits in the "pattern matching" tier of the model hierarchy — cheaper than LLM, more selective than sending everything.

8. **Specification-first workflow (speckit).** Spec -> clarify -> plan -> tasks -> implement pipeline. Features are well-defined before code is written. Prevents "code first, understand later" failures.

9. **Contract-based skill system with standardized phases.** Skills define outputs, traces_to, and checks in YAML contracts. QA validates producer output against contracts. Standard phases: GATHER -> SYNTHESIZE -> CLASSIFY -> PRODUCE. Makes skill quality measurable and enforceable.

10. **Adaptive interview pattern with gap assessment.** Before asking the user questions, gather context from files, memory, and project structure. Calculate coverage percentage. Ask 1-2 questions for small gaps, 3-5 for medium, 6+ for large. Respects user time by being informed before interviewing.

11. **Paired QA (every executor gets a QA validator).** Producer creates artifact, QA validates against contract. If QA returns "improvement-request", producer retries with feedback. Catches quality issues before they propagate.

12. **Team orchestration with task lists and cost-optimized model selection.** Opus for strategic coordination, Sonnet for implementation, Haiku for mechanical loops and exploration. Task lists coordinate work across agents. Merge-on-complete for parallel work.

13. **Importance x Impact matrix for memory evaluation.** Two dimensions replace twelve thresholds. Both grounded in the problem statement:
    - **Important** = this memory matches situations where the agent would otherwise be slower, more expensive, or need more corrections. Measured by retrieval frequency, recency, and cross-session spread.
    - **Effective** = when surfaced, the agent measurably needed fewer corrections (user didn't re-teach or override), used fewer tokens (didn't go down wrong paths), or reached the right answer faster (fewer turns). Measured by post-interaction outcome signals.
    - Four quadrants: Working (keep — reduces corrections), Leech (diagnose — surfaced but doesn't reduce corrections/tokens/time), Hidden Gem (surface more — effective when it appears), Noise (natural decay).

14. **Leech diagnosis over leech deletion.** High importance + low impact is the most valuable signal — a memory that keeps matching contexts but doesn't reduce corrections, tokens, or time. The problem is the card, not the learner. Four diagnoses: content quality (rewrite — unclear guidance doesn't reduce corrections), wrong tier (move to CLAUDE.md — surfaced too late to prevent the mistake), enforcement gap (convert to hook — agent understands but doesn't comply, needs deterministic enforcement), retrieval mismatch (narrow scope — surfaced in wrong contexts, wasting tokens).

### What failed

1. **ONNX runtime dependency (CGO complexity).** Required downloading a 50MB model at runtime, initializing ONNX with CGO bindings, and managing platform-specific shared libraries. Hard to test (need `~/.claude/models/` to exist), hard to cross-compile, hard to distribute. Integration tests that depend on external state are not tests — they're hope.

2. **Nil dependencies at runtime.** SemanticMatcher was defined as an interface, had a constructor, had callers — but was never wired in production. The nil check at the call site silently skipped the entire feature. Nobody noticed because the system "worked" without it — it just quietly did less than it should have.

3. **Hardcoded I/O in library functions.** `os.Stat`, `os.MkdirAll`, `downloadModel`, `sql.Open` called directly inside `internal/` library code. Made functions untestable without real filesystems, real databases, real networks. Every integration test written around hardcoded I/O was a signal that the function needed DI refactoring, not a more elaborate test setup.

4. **Over-engineered tier movement (12+ thresholds, 3 merge/split implementations).** Three separate implementations of merge (active skills, compile-time, periodic reorganization). Twelve configurable thresholds with undocumented interdependencies. The system had sophisticated plumbing that reliably moved noise between tiers.

5. **Mechanical synthesis ("important pattern for review" 56+ times).** `generatePattern()` extracted top-3 keywords via word frequency and produced the same generic label 56+ times. The LLM synthesis path existed but wasn't the default. Frequency-based promotion without quality gates produces noise, not knowledge.

6. **Count-based promotion (frequency != quality).** 5 retrievals + 3 projects = promote. No measurement of whether the memory actually helps when surfaced. A memory retrieved 100 times and ignored every time met the promotion criteria identically to one retrieved 100 times and followed every time.

7. **No impact tracking.** Retrieved often (importance) was tracked. Improved outcomes when retrieved (impact) was not. The entire tier movement system optimized for a single dimension — frequency — while the dimension that actually matters — effectiveness — was unmeasured.

8. **Batch-only optimization.** Learning only happened when someone manually ran `projctl memory optimize`. Between optimize runs, the system accumulated noise without self-correcting. Continuous inline evaluation replaces periodic batch processing.

9. **CLAUDE.md as append-only log.** Everything promoted landed in "## Promoted Learnings" as flat bullets. No section routing (commands vs gotchas vs architecture). No quality gate (is this actionable? specific? clear?). No size discipline (grew past 100-line budget repeatedly).

10. **Integration tests around I/O instead of DI refactoring.** When a function was hard to test, the response was to write an integration test that depended on real files, real databases, or real ONNX models. The correct response was to refactor the function for dependency injection. If achieving coverage requires real I/O, the function needs DI — not a more elaborate test setup.

11. **Allowlists masking design problems.** When entities didn't fit the orphan model, when functions couldn't be tested, when lint rules failed — the response was to allowlist rather than fix the underlying design problem. Allowlists should be a last resort, not a first response.

12. **40% of commits were lint/housekeeping rework.** Linters enabled incrementally created waves of fix commits. Lesson: establish all quality standards before writing code, not after.

### Design constraints from lessons

1. **Deterministic first.** The model hierarchy is: hash comparison -> TF-IDF pattern matching -> haiku -> sonnet -> opus. Each step up costs more and should only be used when the previous step can't answer the question. Never use LLM where deterministic analysis suffices.

2. **DI everywhere.** Every function in `internal/` that does I/O takes its dependencies as parameters. `FileSystem`, `Database`, `LLMClient`, `Clock` — all injected. Library code is pure. Wire at the edges: CLI entry points and hooks create real dependencies, library code receives interfaces.

3. **Wiring is where systems fail.** Components can be well-written, well-tested, and still useless if nobody connects them. This was the single most repeated failure in projctl: SemanticMatcher had an interface, a constructor, callers, and tests — but was never wired in production. The nil check at the call site silently skipped the feature. No `if dep != nil { ... }` silent degradation. If a function needs a dependency, require it. If the system needs a component, wire it. Verify wiring with integration-level smoke tests that confirm real components are connected, not just that individual units work in isolation.

4. **Pure Go, no CGO.** TF-IDF instead of ONNX. The traced project proved TF-IDF works (~300 lines, pure Go, zero dependencies). For the memory system, TF-IDF handles candidate pruning before LLM evaluation. Where vector similarity is genuinely needed, use an external embedding API rather than local ONNX.

5. **Plugin form factor.** The system ships as a Claude Code plugin: hooks (deterministic enforcement), skills (procedural knowledge), CLAUDE.md management (always-loaded guidance), and a Go binary (TF-IDF, hashing, scoring, database operations). The plugin is installable, shareable, and self-contained.

6. **Content quality > mechanical sophistication.** Don't build tier movement machinery before validating that content at each tier is good. Fix synthesis quality, add quality gates, measure impact — then optimize movement based on measured signal.

7. **Measure impact, not just frequency.** Every surfaced memory gets tracked for both importance (how often it comes up) and impact (does it help when it does). The importance x impact matrix drives all tier decisions.

8. **Autonomous memories, proposed artifacts.** The system has full autonomy over its internal memory state — writing, editing, merging, pruning, refining, and scoring memories without user approval. All memory operations are auditable (logged with rationale). But skills, CLAUDE.md, and hooks are user-committed artifacts — the system proposes changes to these, the user approves before they take effect.

### Source material for deeper context

Read these files to understand the full history and nuance:
- `docs/plans/2026-02-20-continuous-evaluation-memory-design.md` — Evaluation pipeline vision: surfacing_events table, Haiku filter, importance x impact matrix, leech diagnosis, RAGAS-adapted metrics, ACT-R enhancements
- `.research-synthesis.md` — Comprehensive tier system research: what each tier optimizes for, current vs ideal state, model selection strategy, content quality recommendations
- `~/repos/personal/traced/internal/tfidf/tfidf.go` — Pure Go TF-IDF reference implementation (~300 lines): Index, TopK, Tokenize, cosine similarity. No dependencies beyond stdlib. This is the pattern to follow.
- `~/repos/personal/traced/docs/prompt.md` — This prompt's structural template. The traced rebuild prompt follows the same 5-phase state machine pattern.
- `skills/` directory — Skill architecture patterns: CONTRACT.md (YAML contracts with outputs/traces_to/checks), INTERVIEW-PATTERN.md (adaptive gap assessment), PRODUCER-TEMPLATE.md (GATHER->SYNTHESIZE->CLASSIFY->PRODUCE), project/SKILL.md (state-machine orchestration with team coordination)
- `specs/015-continuous-eval-memory/` — Current feature spec, plan, data model, and contracts for the continuous evaluation pipeline that was in progress when the rebuild was decided

---

## Phase 1: Lessons

Extract and validate lessons from the existing codebase and research documents. Read the source material above. Produce `docs/lessons.md` capturing:

- What worked (with evidence from the codebase)
- What failed (with specific examples and root causes)
- Design constraints that must be honored
- Patterns to carry forward vs patterns to abandon

This is a distillation exercise, not a copy. The lessons above are a starting point — read the source material for deeper context and add anything that's missing. Present to the user for validation before moving on.

---

## Phase 2: Use Case Discovery

Interview the user to discover all situations in which this memory system will be used. Do not prescribe use cases — discover them through conversation. Ask one question at a time.

**Seed categories to explore** (starting points, not exhaustive):
- Learning from sessions: extracting knowledge from conversation transcripts after a session ends
- Surfacing at hook time: retrieving relevant memories when hooks fire (PreToolUse, SessionStart, etc.)
- Self-correction on failures: when the agent makes a mistake the user corrects, turning that correction into a memory that prevents the same mistake
- Skill generation: compiling clusters of similar memories into reusable procedural skills
- CLAUDE.md management: proposing additions/removals to always-loaded guidance based on measured effectiveness
- Plugin installation: a new user installs the plugin — what happens? How do they start?
- Sharing with others: exporting/importing memory collections between users or projects
- Leech diagnosis: identifying memories that are surfaced often but never followed, and fixing them
- Impact measurement: tracking whether surfaced memories actually improve agent behavior
- Hook enforcement: converting high-importance, repeatedly-violated guidance into deterministic hooks
- Project handoff: transferring project-specific knowledge to a new agent or new session

**For each use case, capture:**
- UC-N identifier
- One-sentence description
- Starting state (what exists before)
- Desired end state (what's true after)
- Actor (human, agent, hook, CI)
- Key interactions (what triggers what, what the user sees, what the system does)

**Discovery approach:**
- Present the seed categories and ask: "Which of these apply to you? What's missing?"
- For each confirmed use case, ask enough questions to fill the template above
- After all use cases are captured, present the full list for validation
- Ask: "Is anything missing? Any of these wrong?"
- Write validated use cases to `docs/use-cases.md`

---

## Phase 3: Requirements

Transform validated use cases into traceable requirements.

**For each requirement, capture:**
- REQ-N identifier
- Statement (what must be true, framed as an invariant, not a behavior)
- Traces to: UC-N reference(s)
- Acceptance criteria (how do you know it's satisfied)
- Verification tier (deterministic / TF-IDF / haiku / sonnet / opus) — the cheapest tier that can confirm this requirement

**Per-requirement checklist (apply to each REQ-N; see lessons #15-22 in `docs/lessons.md` for full context):**
- [ ] Could this be satisfied by a no-op or disconnected function? → Rewrite to demand end-to-end wiring (#16)
- [ ] Are definitions using labels or observable conditions? → Replace labels with testable mechanisms (#15)
- [ ] Does the UC text explicitly support this requirement? → If the UC is ambiguous, fix the UC first, then derive (#17)
- [ ] Is any constraint imported from outside this UC's scope? → Remove or trace to the layer that states it (#18)
- [ ] Does a validated artifact already state this explicitly? → Reflect it directly, don't re-derive from first principles (#19)
- [ ] Does the requirement contain specific items when the UC enumerates them? → Include verbatim, don't summarize (#20)
- [ ] Does any threshold lack validated data? → Specify the decision mechanism and data collection plan, not a placeholder (#21)
- [ ] Do evaluation/promotion criteria connect to the system's stated purpose? → Each criterion must trace to faster, cheaper, or fewer corrections (#22)

**Process:**
- Walk through each use case and extract requirements implied by its interactions and end state
- Group requirements by domain (storage, retrieval, evaluation, enforcement, plugin, CLAUDE.md management, etc.)
- For each requirement, determine the cheapest verification tier that can confirm it
- Apply the per-requirement checklist above before presenting each group
- Present requirements to the user grouped by domain, one group at a time
- After all groups validated, write to `docs/requirements.md`
- Cross-check: every UC-N must be referenced by at least one REQ-N. Every REQ-N must reference at least one UC-N. Report any gaps.

---

## Phase 4: Design

This is the most important phase. Design how users interact with the system before deciding how to build it.

**Produce for each use case that involves user interaction:**

1. **Case study walkthrough** — A narrative showing a realistic project going through the use case start to finish. Name the project, give it realistic files and scenarios. Show every command the user runs and every output they see.

2. **Hook interaction flows** — What hooks fire, when, with what input? What does the system do in response? What does the user see in their terminal? Show the exact system reminder text that appears when memories are surfaced.

3. **CLAUDE.md management UX** — How does the system propose additions? What does the proposal look like? How does the user approve/reject? What happens to demoted content? Show mock terminal output.

4. **Skill interaction transcripts** — For interactive workflows (memory review, leech diagnosis, plugin setup), show a mock conversation. Include the tool's questions, the user's answers, and what the tool does in response.

5. **Error and edge case interactions** — What happens when the database is empty? When no memories match? When the user corrects the agent and a relevant memory existed but wasn't surfaced? When CLAUDE.md exceeds the line budget? Show the output.

**Design principles:**
- Output should be scannable. A passing hook should add minimal latency. A surfaced memory should be concise and actionable.
- Interactive workflows should propose recommendations, not ask open-ended questions. "I recommend X because Y — does that match your thinking?" not "What do you think about X?"
- The user should never *need* to understand ACT-R activation or TF-IDF scoring to use the system — but the math should be accessible on demand. Default to surfacing the insight; show the scores, weights, and reasoning when the user asks or when diagnostic commands are run.
- Hook latency budget: deterministic operations < 50ms, TF-IDF < 200ms, LLM calls only when the value clearly exceeds the cost.

**Present each case study to the user for validation before moving to the next.** After all case studies are validated, write to `docs/design.md`.

**Cross-check:** Every design entry must trace to at least one REQ-N. If a design entry doesn't trace upward, either propose a new requirement to justify it or cut the design entry. Every REQ-N that involves user interaction must be addressed by at least one design entry. Report gaps in both directions.

---

## Phase 5: Architecture

Design the system that implements the validated design.

**Decisions to make (each gets an ARCH-N identifier):**

- **Plugin structure:** What does the plugin directory look like? What hooks, skills, agents, and commands does it expose?
- **Go binary scope:** What does the Go binary do vs what do hooks/skills handle? Where is the boundary between "needs compiled code" and "shell script is fine"?
- **TF-IDF integration:** How does the TF-IDF index get built, updated, and queried? What's the document corpus (memory text, enriched content, skill descriptions)? When is it rebuilt vs incrementally updated?
- **Data model:** How are memories, surfacing events, skills, and scores represented? SQLite schema, file-based storage, or both?
- **Retrieval pipeline:** BM25 + TF-IDF + RRF, or something simpler? How do results get filtered before surfacing? What replaces the ONNX vector search?
- **Evaluation loop:** How does the system track whether surfaced memories helped? End-of-session scoring, sampling, or something else?
- **Hook design:** Which hooks does the plugin register? What does each hook's script do? How do hooks access the Go binary and database?
- **CLAUDE.md management:** How does the system read, parse, score, and propose changes to CLAUDE.md? Section-aware parsing? Quality gates?
- **Skill lifecycle:** How are skills generated from memory clusters? How are they tested (RED/GREEN)? How are they maintained?
- **Model hierarchy:** When exactly does the system call an LLM, with what model, and what does it ask? Map every LLM call to its purpose and justify why a cheaper tier can't handle it.
- **Component boundaries:** What are the major Go packages and their responsibilities? What depends on what?
- **Configuration:** What's user-configurable? What has sensible defaults? TOML, YAML, or something else?

**For each decision:**
- ARCH-N identifier
- The decision and its rationale
- Alternatives considered and why they were rejected
- Traces to: REQ-N reference(s) this decision satisfies

**Process:**
- Present decisions one at a time with alternatives and your recommendation
- After all decisions validated, write to `docs/architecture.md`
- **Bidirectional cross-check:** Every ARCH-N must trace to at least one REQ-N. If an architecture decision doesn't trace upward, either propose a new requirement to justify it or cut the decision. Every REQ-N that involves a technology choice, component boundary, or data model decision must be addressed by at least one ARCH-N. Report gaps in both directions. Requirements that map directly to implementation without needing an architecture decision may legitimately have no ARCH-N — only flag gaps where a decision is genuinely needed.

---

## Global Rules

These apply across all phases.

**Progress persistence:**
- Update `docs/state.toml` after every substantive interaction (write-ahead). See the specification-layers skill for the TOML format.
- When you receive "continue", read `state.toml`, announce your layer/group and next step, and resume
- Append new lessons to `docs/lessons.md` whenever you discover something that should inform future phases

**Interaction style:**
- One question at a time
- When you have a recommendation, lead with it and explain why. Don't present options as equally weighted unless they genuinely are.
- Propose, don't ask open-ended questions. "I think X because Y — does that match your thinking?" not "What do you think about X?"
- Be concise. Say what's needed, nothing more.

**DI quality gate:**
- No function in `internal/` may call `os.*`, `http.*`, `sql.Open`, or any I/O operation directly
- All I/O is performed through injected interfaces: `FileSystem`, `Database`, `LLMClient`, `Clock`, `HTTPClient`
- Wire at the edges: `cmd/` and hook scripts create real implementations, `internal/` receives interfaces
- This is non-negotiable. It was the single most pervasive failure in the existing system.

**Model hierarchy (deterministic -> TF-IDF -> haiku -> sonnet -> opus):**
- Hash comparison and config parsing: deterministic (free, instant)
- Candidate pruning and similarity scoring: TF-IDF (free, <200ms)
- Relevance filtering, simple classification, format validation: haiku (~$0.0001/call)
- Content synthesis, quality judgment, enrichment: sonnet (~$0.005/call)
- Strategic decisions, cross-context reasoning: opus (reserved for orchestration, rarely needed in the memory system itself)
- Always use the cheapest tier that answers the question.

**Anti-patterns to avoid:**
- Do not allowlist problems. If something doesn't fit the model, the model is wrong.
- Do not conflate "pre-existing" with "acceptable." If it's a problem, it's a problem regardless of when it was introduced.
- Do not use LLM calls where deterministic analysis suffices.
- Do not over-engineer. The simplest design that satisfies all requirements is the correct design.
- Do not create artifacts the user didn't ask for. No README files, no documentation beyond what's specified in the phase structure.
- Do not write integration tests around hardcoded I/O. If a function is hard to test, refactor it for DI.
- Do not build tier movement machinery before measuring content quality. Fix what's there before building pipelines to move it.

**Quality gates:**
- **Bidirectional satisfaction:** Each level must be fully satisfied by the level beneath it, and lower levels must not add things that don't contribute to satisfaction of upper levels. When a lower-level artifact doesn't trace upward, either propose an enhancement to the upper level or cut the superfluous item. This applies at every boundary: UC -> REQ, REQ -> DESIGN, REQ -> ARCH.
- Every artifact must have bidirectional traceability to its adjacent tiers. The maximum chain is UC -> REQ -> DESIGN -> ARCH -> IMPL, but any given requirement might follow REQ -> IMPL directly if no design or architecture decision is needed.
- Before exiting any phase, run the cross-check described in that phase's instructions and report gaps.
- Do not proceed to the next phase until the current phase's artifact is written and validated by the user.
- The design phase gate: every requirement that involves user interaction, non-obvious UX, or multiple valid approaches must have a design entry. Requirements that are purely internal (hash computation, TF-IDF indexing) may skip design.
- The architecture phase gate: every requirement that involves a technology choice, component boundary, or data model decision must have an architecture entry.
