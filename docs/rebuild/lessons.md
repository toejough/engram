# Lessons Learned: Self-Correcting Memory System

Distilled from projctl (15+ features), traced (spec-driven development), research synthesis, and continuous evaluation design work. Each lesson is evaluated against the core question: does it help the agent get the right answer **faster**, **cheaper**, or with **fewer corrections**?

---

## What Worked

### Retrieval & Storage

**1. Hybrid search worked because its signals were genuinely complementary — not because there were two of them.**
BM25 (exact term matching) + dense vector embeddings (semantic similarity) merged via RRF. These caught different failure modes: BM25 missed paraphrases, vectors missed exact keywords. The principle is "complementary signals from different paradigms," not "more signals." BM25 + TF-IDF would NOT be meaningful hybrid search — both are sparse term-based methods. Real hybrid means sparse + dense (or another genuinely different paradigm). **Open question for architecture:** The existing stack used CGO for both BM25 (go-sqlite3 FTS5) and embeddings (ONNX). The rebuild requires pure Go, no CGO — but that's a constraint on the implementation, not a reason to abandon these capabilities. Pure Go SQLite drivers, pure Go embedding inference, or rewrites of key libraries may preserve BM25 and/or dense vectors without CGO. This needs research before committing to a retrieval architecture. *Evidence: projctl's hybrid search outperformed single-signal retrieval; the value came from paradigm diversity (sparse + dense).*

**2. Semantic enrichment at write-time.**
LLM extraction creates structured metadata (observation_type, concepts, principle, anti_pattern, rationale, enriched_content). Dramatically improves retrieval quality compared to raw text storage. This is the single most impactful content quality operation in the existing system. *Evidence: enriched memories surface more accurately than raw text.*

**3. Session extraction with confidence tiers.**
A/B/C confidence (A = user explicitly stated, B = agent inferred and visible during session — user had opportunity to correct but didn't, C = agent inferred post-session from transcript patterns — user never saw it). Confidence governs surfacing aggressiveness — higher confidence gets surfaced more readily. *Evidence: prevents low-confidence noise from overwhelming high-signal corrections.*

**4. Deterministic hashing for change detection.**
SHA-256 truncated to 8 hex chars. If nothing changed, skip the expensive check. Free, instant, and catches stale content without LLM calls. *Evidence: traced's hash-gated validation eliminates redundant LLM calls entirely.*

### Evaluation & Scoring

**5. Importance x Impact matrix replaces twelve thresholds.**
Two measured dimensions: importance (how often it comes up, via ACT-R activation) and impact (does it help when surfaced, via faithfulness scoring). Four quadrants: Working (keep), Leech (diagnose), Hidden Gem (surface more), Noise (decay). *Evidence: the existing 12-threshold system had undocumented interdependencies and no measurement of effectiveness.*

**6. Leech diagnosis over leech deletion.**
High importance + low impact is the most valuable signal. Four diagnoses: content quality (rewrite), wrong tier (move earlier), enforcement gap (convert to hook), retrieval mismatch (narrow scope). The problem is the card, not the learner. *Evidence: spaced repetition research + projctl's observation that frequently-surfaced-but-ignored memories are fixable, not disposable.*

**7. End-of-session scoring as default, with immediate learning for urgent signals.**
End-of-session scoring (PreCompact/PreClear/Stop hooks) is the right default — natural boundaries, no per-interaction latency, 100% evaluation. But some signals demand immediate learning: explicit user corrections ("stop doing X"), repeated violations of the same rule, and direct feedback (`/memory wrong`). These can't wait for session end — the agent needs to learn *now*, not after the conversation where the user was frustrated. Two-speed system: batch scoring at session end for aggregate signal, immediate write for high-confidence corrections. *Evidence: specs/015 chose end-of-session for aggregate scoring; user experience demands immediate response to explicit corrections.*

**8. Distribution tracking for threshold auto-tuning.**
Monitor what percentage of memories fall into each quadrant, adjust thresholds to maintain target distributions (e.g., leeches should be ~5-15%). Avoids hardcoded magic numbers while keeping the system self-calibrating. *Evidence: the existing 12 thresholds had no guidance on when to adjust.*

### Architecture & Process

**9. Hook infrastructure for deterministic enforcement.**
Shell-based hooks (PreToolUse, PostToolUse, Stop, SessionStart) that exit 0 (pass) or exit 2 (block with feedback). Deterministic — no model variability. The strongest enforcement tier. *Evidence: hooks are the only enforcement mechanism that works 100% of the time, independent of model behavior.*

**10. Maximize local free signal before model calls.**
The model hierarchy is: deterministic (hash, config) → local IR (TF-IDF, BM25, embeddings) → haiku → sonnet → opus. Each step costs more. TF-IDF is one proven pure Go option (~295 lines, zero deps, proven in traced). BM25 and dense embeddings are also valuable if achievable in pure Go. The principle: exhaust cheap local analysis before spending on API calls. *Evidence: traced's TF-IDF pruning eliminates most candidates before LLM evaluation; projctl's BM25+vectors further improved retrieval quality.*

**11. Specification-first workflow.**
Spec → clarify → plan → tasks → implement pipeline. Features are well-defined before code is written. Prevents "code first, understand later" failures. *Evidence: projctl features 014-015 followed this pattern successfully; earlier features that skipped it required more rework.*

**12. Skills with verifiable contracts and standardized phases.**
Skills that define expected outputs and validation criteria are measurably higher quality than freeform ones. A QA step that validates output against the contract catches issues before they propagate. Standard phases (GATHER → SYNTHESIZE → CLASSIFY → PRODUCE) give structure without rigidity. The specific contract format is a design decision for this project — the principle is: skill quality should be verifiable, not subjective. *Evidence: 23 skills in projctl follow this pattern; paired QA catches issues that self-review misses.*

**13. Adaptive interview with gap assessment.**
Before asking questions, gather context from files, memory, and project structure. Calculate coverage percentage. Ask 1-2 questions for small gaps, 3-5 for medium, 6+ for large. Respects user time. *Evidence: three producer variants (pm, design, arch) use this pattern successfully.*

**14. Paired QA (every executor gets a validator).**
Producer creates artifact, QA validates against contract. Catches quality issues before they propagate. Max iterations prevent infinite loops. *Evidence: projctl's team orchestration uses this for all producer outputs.*

**15. Cost-optimized model selection across roles.**
Haiku for mechanical/exploration work, Sonnet for implementation, Opus for strategic coordination. Model hierarchy: deterministic → TF-IDF → haiku → sonnet → opus. Each step costs more and should only be used when the previous can't answer. *Evidence: team orchestration reduced costs by using Haiku for orchestration and QA.*

**16. Fire-and-forget schema migration.**
ALTER TABLE with IF NOT EXISTS for additive-only changes. No migration version tracking needed when all changes are additive and idempotent. Simpler than migration frameworks. *Evidence: specs/015 plan uses this pattern for new columns.*

---

## What Failed

### Dependency & Wiring

**1. Nil dependencies at runtime (wiring failure).**
SemanticMatcher had an interface, a constructor, callers, and tests — but was never wired in production. The nil check at the call site silently skipped the entire feature. Nobody noticed because the system "worked" without it. The ONNX/E5 embedding system wasn't a distribution problem — it was a DI and wiring problem. The code that used ONNX didn't follow DI, so it was hard to test, which meant wiring was never verified, which meant the feature silently did nothing.
*Root cause: `if dep != nil { ... }` silent degradation. No smoke tests confirming real components are connected. Unit tests passed because mocks worked — but mocks don't test wiring.*

**2. Hardcoded I/O in library functions.**
`os.Stat`, `os.MkdirAll`, `sql.Open` called directly inside `internal/` code. Made functions untestable without real filesystems, real databases, real networks. Every integration test written around hardcoded I/O was a signal the function needed DI refactoring.
*Root cause: Expedience. Writing `os.ReadFile()` is faster than defining a FileSystem interface. The debt compounds when every caller becomes untestable.*

### Content Quality

**3. Mechanical synthesis produces noise.**
`generatePattern()` extracted top-3 keywords via word frequency → "important pattern for review" appeared 56+ times. The LLM synthesis path existed but wasn't the default. Frequency-based extraction without quality gates produces noise, not knowledge.
*Root cause: Building the pipeline before validating that what flows through it is good.*

**4. Count-based promotion (frequency != quality).**
5 retrievals + 3 projects = promote. No measurement of whether the memory actually helps when surfaced. A memory surfaced 100 times and ignored looks identical to one surfaced 100 times and followed.
*Root cause: Optimizing for the measurable (frequency) while ignoring the meaningful (impact).*

**5. No impact tracking.**
Retrieved-often was tracked. Improved-outcomes-when-retrieved was not. The entire tier movement system optimized for a single dimension while the dimension that matters was unmeasured.
*Root cause: Impact is harder to measure than frequency, so it was deferred indefinitely.*

**6. CLAUDE.md as append-only log.**
Everything promoted landed in "## Promoted Learnings" as flat bullets. No section routing, no quality gate, no size discipline. Grew past 100-line budget repeatedly.
*Root cause: No quality gate between "meets promotion threshold" and "written to file." The system treated the threshold as sufficient evidence of quality.*

### Mechanical Complexity

**7. Over-engineered tier movement (12+ thresholds, 3 merge/split implementations).**
Three separate merge implementations (active skills, compile-time, periodic reorg). Twelve configurable thresholds with undocumented interdependencies. Sophisticated plumbing that reliably moved noise between tiers.
*Root cause: Building movement machinery before validating content quality at each tier.*

**8. Split operations were stubs masquerading as features.**
Embeddings split: `return fmt.Errorf("not yet implemented")`. Skills split: re-clusters at 0.6 but no topic detection. CLAUDE.md split: sentence boundaries, not semantic coherence. mergeEntries() picks the longer string and discards the shorter entirely.
*Root cause: Feature breadth over depth. The operations existed in the interface but not in substance.*

**9. Periodic reorganization (O(n²) with no quality measurement).**
Every 30+ days: fetch ALL memories, compute similarity matrix, re-cluster, regenerate all skills. Expensive, disruptive, and no before/after quality measurement to justify it.
*Root cause: Assumption that reorganization improves quality, without measuring whether it does.*

**10. State tracking proliferation.**
14+ columns in embeddings, 15+ in skills. Many overlap in purpose (confidence vs utility vs alpha/beta) or go unused. Each column was added for a specific feature but the aggregate complexity wasn't managed.
*Root cause: Additive design without periodic simplification. Each feature added columns; none removed them.*

### Process

**11. Batch-only optimization.**
Learning only happened during manual `projctl memory optimize` runs. Between runs, the system accumulated noise without self-correcting.
*Root cause: Designing for batch processing when the use case demands continuous feedback.*

**12. Integration tests around I/O instead of DI refactoring.**
When a function was hard to test, the response was integration tests depending on real files, databases, or ONNX models. The correct response was DI refactoring. If achieving coverage requires real I/O, the function needs DI — not a more elaborate test setup.
*Root cause: Path of least resistance. An integration test is quicker to write than a DI refactor.*

**13. Allowlists masking design problems.**
When entities didn't fit the model, when functions couldn't be tested, when lint rules failed — the response was to allowlist rather than fix. Allowlists are debt that compounds silently.
*Root cause: Treating symptoms rather than causes.*

**14. 40% of commits were lint/housekeeping rework — and we know the specific model defaults that cause it.**
Linters enabled incrementally created waves of fix commits. But "establish standards early" is too vague. The lint config reveals the specific anti-patterns Opus/Sonnet default to:
- **Magic numbers** (249 violations) — models hardcode numeric literals instead of named constants
- **Short variable names** (330) — models use `s`, `m`, `r` instead of descriptive names
- **Long lines** (313) — models don't line-break
- **Missing error wrapping** (172) — models return `err` without `fmt.Errorf("context: %w", err)`
- **Inline error creation** (257) — models use `fmt.Errorf()` inline instead of sentinel errors
- **Missing t.Parallel()** (415) — models never add parallel test markers
- **HTTP without context** (207) — models use `http.Get()` instead of `req.WithContext()`
- **Security issues** (131) — models use `math/rand` instead of `crypto/rand`, etc.

The lesson isn't "establish standards" — it's: **know your models' defaults, configure linters to catch them from commit one, and ideally enforce via pre-commit hooks so violations never land.** The rebuild should start with a full lint config (all linters enabled, zero pre-existing violations) and a hook that blocks commits with lint failures.
*Root cause: Models produce code with predictable anti-patterns. Without upfront tooling, those patterns accumulate into thousands of violations that are painful to fix retroactively.*

---

## Design Constraints

These are non-negotiable for the rebuild. Each traces to specific failures above.

**1. Deterministic first, local free analysis second, models last.** (Traces to: failures #7, #8)
Model hierarchy: deterministic (hash, config) → local IR (TF-IDF, BM25, embeddings — all pure Go) → haiku → sonnet → opus. Each step costs more. Never use a model where local analysis suffices.

**2. DI everywhere.** (Traces to: failures #1, #2, #12)
Every function in `internal/` that does I/O takes dependencies as parameters. `FileSystem`, `Database`, `LLMClient`, `Clock` — all injected. Library code is pure. Wire at the edges.

**3. Wiring is where systems fail.** (Traces to: failure #1)
No `if dep != nil { ... }` silent degradation. If a function needs a dependency, require it. Verify wiring with integration-level smoke tests that confirm real components are connected.

**4. Pure Go, no CGO.**
All compiled code must be pure Go — no CGO dependency. This is a constraint on implementation, not on capabilities. BM25, embeddings, and other retrieval signals are valuable if achievable in pure Go (pure Go SQLite drivers with FTS5, pure Go ONNX inference or model rewrites, etc.). Research needed during architecture phase to determine what's feasible. The goal: maximize local, free analysis before spending money on model calls.

**5. Plugin form factor.** (Traces to: successes #9, #11, #15)
Claude Code plugin: hooks (deterministic enforcement), skills (procedural knowledge), CLAUDE.md management (always-loaded guidance), Go binary (TF-IDF, hashing, scoring, database).

**6. Content quality over mechanical sophistication.** (Traces to: failures #3, #4, #5, #6, #7, #8, #9)
Don't build tier movement machinery before validating content quality. Fix synthesis quality, add quality gates, measure impact — then optimize movement.

**7. Measure impact, not just frequency.** (Traces to: failures #4, #5)
Every surfaced memory gets tracked for importance AND impact. The importance x impact matrix drives all tier decisions.

**8. Autonomous memories, proposed artifacts.** (Traces to: failure #6, #3)
Full autonomy over internal memory state (write, edit, merge, prune, score) — all auditable. Skills, CLAUDE.md, and hooks are user-committed — the system proposes, the user approves.

**9. Guard against known model defaults from commit one.** (Traces to: failure #13)
Enable all linters before the first feature commit. Enforce via pre-commit hook so violations never land. The specific anti-patterns to guard against are documented in failure #14 — magic numbers, short names, missing error wrapping, inline errors, no t.Parallel(), HTTP without context, math/rand over crypto/rand. Zero pre-existing violations policy.

---

## Patterns to Carry Forward

| Pattern | Source | Why |
|---------|--------|-----|
| Complementary retrieval signals (sparse + dense) | projctl memory | Different paradigms catch different failure modes; two sparse methods don't count |
| Maximize local free analysis before model calls | traced + projctl | TF-IDF, BM25, embeddings — exhaust cheap signals first |
| Semantic enrichment at write-time | projctl memory | Dramatically improves retrieval quality |
| ACT-R activation (frequency × recency × spread) | projctl memory | Proven importance scoring |
| Session extraction with confidence tiers | projctl memory | Distinguishes signal strength |
| Hook infrastructure (exit 0/2 protocol) | projctl hooks | Deterministic enforcement |
| Skills with verifiable contracts + QA validation | projctl skills | Measurable quality; specific format is a design choice |
| GATHER → SYNTHESIZE → CLASSIFY → PRODUCE | projctl skills | Proven producer workflow |
| Adaptive interview with gap assessment | projctl skills | Respects user time |
| Deterministic hashing for change detection | projctl + traced | Free staleness detection |
| Fire-and-forget additive migrations | specs/015 | Simple schema evolution |
| End-of-session scoring at natural boundaries | specs/015 | Works for ephemeral CLI sessions |
| Importance × impact quadrant classification | eval design | Two dimensions replace twelve thresholds |

## Patterns to Abandon

| Pattern | Source | Why |
|---------|--------|-----|
| CGO dependencies (go-sqlite3, ONNX CGO bindings) | projctl memory | Pure Go constraint; CGO itself wasn't the problem — poor DI around CGO-dependent code was |
| BM25 + TF-IDF as "hybrid search" | lessons review | Both are sparse term-based methods; not complementary paradigms |
| Count-based promotion (5 retrievals + 3 projects) | projctl memory | Frequency != quality |
| Keyword-based synthesis (generatePattern) | projctl memory | Produces "important pattern for review" × 56 |
| 12+ configurable thresholds | projctl memory | Undocumented interdependencies, no measured impact |
| 3 separate merge implementations | projctl memory | Complexity without value |
| Periodic full reorganization (30-day re-cluster) | projctl memory | O(n²) with no quality measurement |
| `if dep != nil { ... }` silent degradation | projctl memory | Hides wiring failures |
| CLAUDE.md auto-writes | projctl memory | No quality gate, append-only bloat |
| mergeEntries() = pick longer string | projctl memory | Loses nuance |
| Sampling-based evaluation (10%) | eval design (early) | Misses patterns, unreliable for auto-tuning |
| Background worker for scoring | eval design (rejected) | CLI sessions are ephemeral |

---

## Process Lessons (Surfaced in Phase 3: Requirements)

**15. Definitions need observable conditions, not labels.**
The A/B/C confidence tiers were labeled "uncorrected" vs "unvalidated" — near-synonyms. The fix: describe the observable mechanism (did the user have the opportunity to see it?). Labels categorize; observable conditions are testable. *Surfaced when: tier definitions were ambiguous during requirements extraction.*

**16. Requirements must demand wiring, not just capabilities.**
REQ-8 said "when a hook fires, surface memories" — satisfiable by an unconnected function. Design constraint #3 ("wiring is where systems fail") covers this at the code level, but requirements are the first line of defense. If the requirement doesn't demand end-to-end connection, a disconnected implementation can reasonably claim to satisfy it. *Surfaced when: reviewing REQ-8 for verifiability.*

**17. Fix ambiguity at the source, don't paper over it downstream.**
UC-2 was ambiguous about retrieval mechanism for SessionStart and UserPromptSubmit. Rather than deriving a requirement from architectural preferences, the fix was to update UC-2 to explicitly state TF-IDF for all hooks, then derive the requirement. Requirements trace down from use cases — if the UC doesn't support the requirement, fix the UC first. *Surfaced when: REQ-12 over-constrained beyond what UC-2 stated.*

**18. Don't import constraints the use case didn't state.**
Performance numbers (200ms, 50ms) were pulled from global design rules and attributed to UC-1/UC-2. They weren't wrong, but they weren't these use cases' requirements. Constraints belong to the layer that states them. Importing them creates false traceability and premature optimization. *Surfaced when: REQ-13/14 introduced performance constraints not grounded in UC-1/UC-2.*

**19. Don't re-derive what validated artifacts explicitly state.**
Both UCs explicitly said "Go binary." The requirement REQ-15 was almost dropped as "implementation detail" by re-analyzing whether a Go binary was truly necessary. When a validated artifact explicitly states something, the requirement should reflect it — not second-guess it from first principles. *Surfaced when: REQ-15 was nearly cut during review.*

**20. Requirements must be more specific than their source use cases, not less.**
When a validated UC contains enumerable items (patterns, fields, signals), include them verbatim in the requirement. Requirements are downstream artifacts — they refine, they don't summarize. Writing "~15 patterns" when the UC lists all 15 is a loss of specificity. *Surfaced when: REQ-13 summarized UC-3's correction patterns instead of listing them.*

**21. Unknown thresholds need a decision mechanism and a data plan, not a placeholder.**
Writing "above a similarity threshold" is a non-requirement — it defers the "what" to architecture. When the right threshold isn't known: (a) specify the fallback decision mechanism (e.g., LLM gate), (b) specify what data to log so the threshold can be derived later, (c) record an issue for future evaluation. *Surfaced when: REQ-14 used a placeholder threshold with no strategy to determine it.*

**22. Evaluation criteria must trace to system purpose.**
When a requirement involves promotion, scoring, or ranking decisions, connect each criterion to the system's stated purpose (faster, cheaper, fewer corrections). Requirements that optimize for internal metrics (frequency, cost) without connecting to user outcomes are architecturally ungrounded. *Surfaced when: REQ-19's initial criteria didn't connect to the plugin's purpose.*

**23. Prefer transparent file storage over databases when the access patterns allow it.**
When storage requirements are single-user, structured documents, with reasonable scale (hundreds not millions), prefer files over databases. Files are inspectable (`cat`, `grep`), user-editable, git-friendly (diff, commit, branch), and have zero dependencies. Databases add complexity (drivers, schemas, migrations, query languages) that's only justified by concurrent access, transactional integrity across multiple writes, or query sophistication beyond file scanning. For a personal memory system, numbered TOML files with a cached TF-IDF index eliminate the entire database dependency while making the data transparent and portable. *Surfaced when: evaluating whether SQLite was necessary for memory storage given pure Go constraint and single-user access pattern.*
