# Use Cases

## UC-1: Session Learning

**Description:** Extract learnings from conversation transcripts after a session ends.

**Starting state:** A session has ended (Stop hook fires). The transcript contains corrections, decisions, patterns, and mistakes.

**End state:** New memories exist in the memory store with structured metadata (observation_type, concepts, principle, anti_pattern, rationale, enriched_content) and LLM-generated keywords optimized for retrieval. Extracted content has passed a quality gate before storage. Existing memories that overlap with new learnings have been enriched rather than duplicated.

**Actor:** System (background Go binary triggered by Stop hook).

**Key interactions:**
- Stop hook fires, invokes Go binary with session transcript
- LLM enrichment generates keywords, concepts, and anti-patterns for retrieval
- Confidence tiers: A (user explicitly stated), B (agent inferred and visible to user — user had opportunity to correct but didn't), C (agent inferred post-session — user never saw it, zero validation)
- Reconciliation against existing memories before insert (retrieve candidates, haiku overlap gate, enrich existing or create new)
- Quality gate: extracted memories must be specific and actionable — mechanical patterns and vague generalizations are rejected before storage
- Mid-session corrections (UC-3) are reconciled — session-end extraction doesn't duplicate what was already captured

---

## UC-2: Hook-Time Surfacing

**Description:** Retrieve and surface relevant memories when hooks fire during a session.

**Starting state:** A hook fires (SessionStart, UserPromptSubmit, or PreToolUse). The memory store contains previously extracted learnings.

**End state:** Relevant memories are surfaced as system reminders in the agent's context.

**Actor:** System (hook scripts invoke Go binary for retrieval).

**Key interactions:**
- **SessionStart:** Broad context retrieval — project-level memories, recent high-importance items
- **UserPromptSubmit:** Task-relevant retrieval — memories matching the user's current request
- **PreToolUse:** Smallest budget, latency-critical — only the single most relevant memory surfaced
- All three hooks use local retrieval only (no LLM calls at retrieval time). Specific retrieval algorithm (TF-IDF, BM25, dense vectors, or hybrid) is an architecture decision constrained by pure Go. Retrieval quality depends on keyword enrichment at extraction time (UC-1), not on retrieval-time judgment.
- Ranking: frecency (recency × impact) as primary signal, confidence tier (UC-1) as tiebreaker. During cold start, recency dominates; as evaluation data accumulates, impact becomes the dominant signal.

---

## UC-3: Self-Correction on Failures

**Description:** When the user corrects the agent, turn the correction into a memory that prevents the same mistake, reconciling against existing memories.

**Starting state:** The agent made a mistake. The user corrected it explicitly.

**End state:** Either a new enriched memory exists, or an existing memory has been enhanced with the correction context. Artifacts that led the agent down the wrong path have been reclassified immediately.

**Actor:** System (two-speed: inline detection at UserPromptSubmit + session-end catch-up at Stop hook).

**Key interactions:**
- **Inline detection (UserPromptSubmit):** Deterministic pattern matcher checks the user's prompt for correction signals (~15 patterns covering ~85% of explicit corrections: `^no,`, `^wait`, `^hold on`, `\bwrong\b`, `\bdon't\s+[verb]`, `\bstop\s+[verb]ing`, `\btry again`, `\bgo back`, `\bthat's not`, `^actually,`, `\bremember\s+(that|to)`, `\bstart over`, `\bpre-?existing`, `\byou're still`, `\bincorrect`). On match → reconciliation against existing memories.
- **If overlap found:** Enhance the full memory — trigger terms, refined observation, new anti-patterns, updated rationale, concrete examples. Memories mature through corrections.
- **If no overlap:** Create new memory with enriched keywords.
- **Immediate reclassification:** When an inline correction is detected, the system reclassifies artifacts that were surfaced (or should have been surfaced) this session. If a skill or memory led the agent wrong, it moves toward the leech quadrant now, not at session-end.
- **Feedback:** A system reminder is injected showing: the correction detected, the memory created or enriched, and the keywords added for future retrieval.
- **Session-end catch-up (Stop hook):** LLM evaluates the transcript for corrections the pattern matcher missed (~15% — typically observational corrections like "you didn't shut them down" that lack imperative correction language). Missed correction phrases are added to the deterministic pattern corpus — the matcher self-improves over time.
- Session-end extraction (UC-1) deduplicates against mid-session corrections.

---

## UC-4: Skill Generation

**Description:** Promote a memory to a skill when the repeated surfacing cost exceeds the fixed cost of a skill slot.

**Starting state:** A memory has been surfaced repeatedly with high "followed" rate across sessions.

**End state:** A skill file exists with procedural knowledge, RED/GREEN test scenarios, and a CLAUDE.md pointer.

**Actor:** System (proposes), human (approves).

**Key interactions:**
- Promotion is a token economics decision — not cluster-based (UC-3 reconciliation prevents clusters by merging similar memories)
- Requires: procedural complexity outgrowing memory format + sustained retrieval frequency + high "followed" rate
- "Followed" measured by haiku-level post-session evaluation: "did agent behavior align with this memory's guidance?"
- RED (should-trigger) and GREEN (should-not-trigger) test scenarios generated and validated before deployment
- CLAUDE.md pointer created at skill creation time, coupled to skill lifecycle

---

## UC-5: CLAUDE.md Management

**Description:** Propose additions, removals, and updates to always-loaded guidance based on measured effectiveness.

**Starting state:** CLAUDE.md exists with a dynamic budget (~50 lines for memory/skill entries out of ~100 total).

**End state:** CLAUDE.md reflects the highest-value guidance, with stale or ineffective entries removed and high-value new entries added.

**Actor:** System (proposes), human (approves).

**Key interactions:**
- Same evaluation pipeline as memories/skills at every Stop and correction
- Skill pointers coupled to skill lifecycle — created/updated/removed together
- Principle promotion requires cross-project spread (3+ projects) + cross-task-type + terse 1-2 lines
- Competitive ranking, not threshold — new candidate must beat worst existing entry
- User-level `~/.claude/CLAUDE.md` for cross-project universals, project-level `CLAUDE.md` for project-specific
- Selection by project spread: 1 project = project file, 3+ = user file

---

## UC-6: Skill Evaluation and Maintenance

**Description:** Track whether skills are followed, contradicted, or irrelevant, and revise accordingly.

**Starting state:** Skills exist and are being loaded in sessions.

**End state:** Skills are revised, narrowed, or removed based on measured effectiveness.

**Actor:** System (evaluates and proposes), human (approves changes).

**Key interactions:**
- Same followed/contradicted/irrelevant signals at Stop and correction
- **Contradicted:** User corrected after following skill guidance → skill is wrong/outdated → propose revision
- **Irrelevant:** Loaded but didn't relate → trigger mismatch → update trigger description
- RED/GREEN tests re-run after revision
- All changes require user approval

---

## UC-7: Working Artifact Maintenance

**Description:** Maintain artifacts in the high-importance + high-impact quadrant.

**Starting state:** An artifact (memory, skill, or CLAUDE.md entry) is frequently surfaced and frequently followed.

**End state:** Artifact is retained. Promotion candidates are flagged.

**Actor:** System (automated).

**Key interactions:**
- Memories: keep, flag promotion candidates (to skill or CLAUDE.md)
- Skills: keep, verify CLAUDE.md pointer is current
- CLAUDE.md: keep, validates budget spend

---

## UC-8: Leech Diagnosis and Repair

**Description:** Diagnose and fix artifacts that are surfaced often but don't improve outcomes.

**Starting state:** An artifact is in the high-importance + low-impact quadrant — frequently surfaced but rarely followed.

**End state:** The artifact has been fixed (rewritten, moved, converted, or narrowed) so it either improves outcomes or stops being surfaced.

**Actor:** System (diagnoses and proposes), human (approves).

**Key interactions:**
- Four diagnosis categories:
  - **Content quality:** Rewrite if unclear — unclear guidance doesn't reduce corrections
  - **Wrong tier:** Promote if surfaced too late to prevent the mistake (e.g., memory → CLAUDE.md)
  - **Enforcement gap:** Convert to hook if agent understands but doesn't comply — needs deterministic enforcement
  - **Retrieval mismatch:** Narrow keywords if surfaced in wrong contexts, wasting tokens
- Memories: rewrite, promote, convert to hook, or narrow keywords
- Skills: revise guidance, narrow trigger, update/remove if outdated, convert core rule to hook
- CLAUDE.md: make specific, update/remove if contradicts practice, remove if redundant with pointer

---

## UC-9: Hidden Gem Discovery

**Description:** Find and broaden artifacts that are effective when surfaced but aren't surfaced often enough.

**Starting state:** An artifact is in the low-importance + high-impact quadrant — rarely surfaced but followed when it is.

**End state:** The artifact's triggers are broadened so it surfaces in more relevant contexts.

**Actor:** System (automated for memories, proposed for skills — per design decision #7).

**Key interactions:**
- Memories: enhance keywords + broaden triggers (automatic, auditable)
- Skills: propose broadened trigger description + added semantic terms to user for approval
- CLAUDE.md: shouldn't exist here (always loaded = always high importance). If somehow here, demote to skill/memory.

---

## UC-10: Noise Pruning

**Description:** Remove artifacts that are neither surfaced often nor effective when surfaced.

**Starting state:** An artifact is in the low-importance + low-impact quadrant.

**End state:** The artifact has been pruned or is on a decay path toward pruning.

**Actor:** System (memories decay automatically), human (approves skill/CLAUDE.md removal).

**Key interactions:**
- Memories: confidence decays naturally, prune below threshold
- Skills: propose removal to user, remove CLAUDE.md pointer
- CLAUDE.md: propose removal, free budget

---

## UC-11: Plugin Installation

**Description:** A new user installs the plugin and starts using the memory system.

**Starting state:** User has Claude Code but no memory plugin.

**End state:** Plugin is installed with sensible defaults, database initialized, indexes created, starter CLAUDE.md section added, system begins learning from the first session.

**Actor:** Human (installs), system (bootstraps).

**Key interactions:**
- Installation via `claude plugin install` (git URL or local path)
- Bootstrap with sensible defaults — create memory directory, init index cache, starter CLAUDE.md section
- If CLAUDE.md already exists, append a memory system section rather than overwriting
- If a memory database already exists, preserve it (idempotent bootstrap)
- The user sees a one-time system reminder at first SessionStart explaining what the plugin does
- No wizard, no interview
- First SessionStart begins learning immediately

---

## UC-12: Hook Enforcement Lifecycle

**Description:** Create, evaluate, revise, and remove deterministic hooks that enforce guidance the agent understands but doesn't consistently follow.

**Starting state:** UC-8 leech diagnosis identified an enforcement gap — the agent knows the rule but doesn't comply.

**End state:** A deterministic hook enforces the rule, or an existing hook has been revised/removed based on effectiveness data.

**Actor:** System (proposes), human (approves).

**Key interactions:**
- Creation triggered by UC-8 enforcement gap diagnosis
- Deterministic hook if checkable by pattern, LLM-based if requires judgment
- Evaluation with same importance x impact matrix
- Hook that fires often but gets overridden = leech
- Hook that never fires = ambiguous — need "was topic relevant this session" to disambiguate deterrent vs noise
- Revision when contradicted, removal when underlying guidance outdated
- All changes require user approval

---

## UC-13: Sharing and Portability

**Description:** Export a curated selection of memory system artifacts as a portable zip, or import from one.

**Starting state:** Export: user has a working memory system with artifacts worth sharing. Import: user has received a zip from someone else (or their own export).

**End state:** Export: a self-contained zip file on disk. Import: selected artifacts installed into the user's system.

**Actor:** Human (chooses what to export/import), system (clusters artifacts for review, handles packaging/installation).

**Key interactions:**
- **Export:** System inventories available artifacts (hooks, skills, CLAUDE.md entries, memories — both project and personal levels). Clusters them into logical groups (e.g., "Go testing conventions" = 3 memories + 1 skill + 1 hook). Presents groups for selection. Packages selected items into a zip.
- **Import:** System unpacks zip, presents contents in the same grouped structure. User selects groups or individual items. Interview mode for selective import ("This skill enforces TDD — want it?"). Full import for "just give me everything."
- Clustering keeps review manageable — choosing 5-10 groups, not 100 individual items
- Distribution is the user's problem — the system produces/consumes a zip file

---

## UC-14: Structured Session Continuity

**Description:** Maintain a curated set of working context files that progressively record meaningful work, survive session boundaries, and replace compaction with archive-and-resume.

**Starting state:** User is working on a project with active context accumulating across turns.

**End state:** Context files (prompt, state, session log, lessons) are up-to-date. Older context is archived. New/resumed sessions pick up from artifacts instead of relying on compacted transcript.

**Actor:** System (progressive recording, compact interception), human (reviews archived context if needed).

**Key interactions:**
- **Four files:** Prompt (what are we trying to get done), State (how far along are we), Session Log (what meaningful discussion happened, what artifacts were created), Lessons (what did we decide/learn)
- **Progressive recording:** After any of these triggers, the system updates context files immediately — not at session end, not on compact: phase transitions, artifact creation/modification, decisions made (user chose between options), corrections detected (UC-3), and interview answers received.
- **Compact interception:** Requires a hookable compact/clear event. If this hook does not exist in Claude Code, this is a blocking platform dependency — discuss alternatives before proceeding, no silent fallback. When the event fires, system ensures artifacts are current, then clears transcript and continues from artifacts. No lossy summarization.
- **Archival:** When files get too long or the user switches projects/prompts, archive current files (timestamped or versioned) and start fresh. Latest artifacts are always the resumption point.
- **Resume:** "Continue" reads state + prompt, announces position, picks up immediately.

---

## Key Design Decisions (Cross-Cutting)

These decisions emerged during use case discussion and apply across multiple UCs:

1. **Effectiveness measurement:** "Was it followed?" evaluated at session-end by haiku-level LLM. Three signals: followed, contradicted, irrelevant. Applied uniformly to memories, skills, and CLAUDE.md entries.

2. **Importance x impact matrix:** Four quadrants applied consistently across all artifact types. Working (UC-7), Leech (UC-8), Hidden Gem (UC-9), Noise (UC-10). Quadrant classification is computed at two points: (a) session-end scoring (Stop hook) after followed/contradicted/irrelevant signals are collected — the system recalculates importance and impact for every artifact surfaced during the session and updates quadrant assignments; (b) immediately when an explicit correction is detected (UC-3) — artifacts that led the agent down the wrong path are reclassified in real time so the system learns now, not later.

3. **Memory enrichment at reconciliation (UC-3):** When corrections match existing memories, enrich the whole memory — trigger terms, refined observation, anti-patterns, rationale, examples. Memories mature through corrections.

4. **Skill promotion is token economics (UC-4):** Not cluster-based. Repeated surfacing cost exceeds fixed skill slot cost + procedural complexity outgrows memory format + high followed rate.

5. **CLAUDE.md competitive ranking (UC-5):** Not threshold-based. New candidates must beat worst existing entry. Dynamic budget ~50 lines. User vs project by project spread.

6. **No skill-to-CLAUDE.md promotion:** Skills stay skills. CLAUDE.md gets a one-liner pointer coupled to skill lifecycle. Principles promoted to CLAUDE.md come from memories, not skills.

7. **Autonomy boundary:** Memories are internal state — full system autonomy. Skills, CLAUDE.md, and hooks are user-committed artifacts — system proposes, user approves.
