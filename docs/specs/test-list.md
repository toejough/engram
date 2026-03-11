# Test List

Behavioral tests derived from architecture items. Each T-* item specifies a test case in Given/When/Then format.

---

## UC-23: Unified Instruction Registry Tests

### T-182: Registry JSONL bounded growth — one line per instruction

**Traces to:** ARCH-52

Given: A registry with 100 registered instructions, each with different source types
When: I read instruction-registry.jsonl
Then: File contains exactly 100 lines, one per instruction ID

---

### T-183: Registry ID format determinism

**Traces to:** ARCH-53

Given: A memory instruction "always-use-targ-reminder.toml" and a CLAUDE.md instruction "use-targ"
When: I register both instructions
Then: Memory ID is "memory:always-use-targ-reminder.toml", CLAUDE.md ID is "claude-md:CLAUDE.md:use-targ"

---

### T-184: Effectiveness ratio computation

**Traces to:** ARCH-54

Given: An instruction with followed=20, contradicted=5, ignored=3
When: I read the instruction's effectiveness signal
Then: Effectiveness is 20/28 = 0.714, not stored in registry

---

### T-185: Frecency blend with decay

**Traces to:** ARCH-54

Given: Instruction A surfaced 100 times, last surfaced 1 day ago; Instruction B surfaced 50 times, last surfaced 7 days ago
When: I compute frecency for both with 7-day half-life
Then: Instruction A has higher frecency despite fewer total surfacings

---

### T-186: Content hash change detection

**Traces to:** ARCH-55

Given: A memory instruction with content_hash="abc123"
When: The memory TOML is edited and re-registered
Then: New content_hash differs from "abc123"

---

### T-187: Absorbed history preserves counters

**Traces to:** ARCH-56

Given: Instruction X (surfaced_count=500, followed=30, contradicted=20) and Instruction Y
When: I merge X → Y
Then: Y's absorbed array contains one entry from X with all counters intact

---

### T-188: Idempotent merge

**Traces to:** ARCH-56

Given: I have merged X → Y once, creating one absorbed entry
When: I merge X → Y again with same instructions
Then: No duplicate absorbed entry is created, operation is idempotent

---

### T-189: Concurrent writes safety — read-all write-full

**Traces to:** ARCH-57

Given: Two concurrent hook calls to RecordSurfacing for different instructions
When: Both calls write to the registry
Then: Both updates persist, no data corruption (acceptable: one update may not include the other's frequency delta)

---

### T-190: Backfill aggregation — surfacing log

**Traces to:** ARCH-58

Given: surfacing-log.jsonl with 10 surface events for memory X, last at timestamp T
When: I run engram registry init
Then: Registry entry for X has surfaced_count=10, last_surfaced=T

---

### T-191: Backfill retirement mapping

**Traces to:** ARCH-58

Given: Retired memory "old-targ-reminder" with retired_by="CLAUDE.md:use-targ", surfaced_count=200, followed=15
When: I run engram registry init
Then: Registry entry for "claude-md:CLAUDE.md:use-targ" has absorbed array with one entry from retired memory

---

### T-192: Quadrant classification — Working

**Traces to:** ARCH-59

Given: An instruction with surfaced_count=150 (> threshold), effectiveness=0.85 (> threshold)
When: I classify it
Then: Quadrant is "Working"

---

### T-193: Quadrant classification — Leech

**Traces to:** ARCH-59

Given: A CLAUDE.md instruction with surfaced_count=200 (always-loaded, maximal), effectiveness=0.20 (< threshold)
When: I classify it
Then: Quadrant is "Leech" (binary, not HiddenGem)

---

### T-194: Quadrant classification — HiddenGem

**Traces to:** ARCH-59

Given: A rule instruction with surfaced_count=5 (< threshold), effectiveness=0.95 (> threshold)
When: I classify it
Then: Quadrant is "HiddenGem"

---

### T-195: Registry interface DI boundary

**Traces to:** ARCH-60

Given: A test that injects mock Registry interface
When: The test calls internal/registry functions
Then: No os.*, io.*, or file operations happen in internal/ — all I/O deferred to concrete implementation

---

### T-196: CLI subcommand registry init — dry-run

**Traces to:** ARCH-61

Given: `engram registry init --dry-run`
When: I run the command
Then: Registry file is not written, only summary of what would be created is printed

---

### T-197: CLI subcommand review — quadrant output

**Traces to:** ARCH-61

Given: `engram review --format json`
When: I run the command and the registry has 10 instructions (3 Working, 2 Leech, 3 HiddenGem, 2 Noise)
Then: Output is JSON array grouped by quadrant, showing all entries

---

### T-198: CLI subcommand merge — absorbs and deletes

**Traces to:** ARCH-61

Given: Two instructions X and Y, and `engram registry merge --source memory:X.toml --target claude-md:CLAUDE.md:X`
When: I run the command
Then: Y's absorbed array includes X's counters, X is deleted from registry (and source file deleted if applicable)

---

### T-199: Hook auto-integration — surfacing

**Traces to:** ARCH-61

Given: A surfacing hook that calls Registry.RecordSurfacing("instruction-id")
When: The hook fires
Then: Registry entry is updated: surfaced_count incremented, last_surfaced set to current time

---

### T-200: Hook auto-integration — evaluation

**Traces to:** ARCH-61

Given: An evaluation hook that calls Registry.RecordEvaluation("instruction-id", "followed")
When: The hook fires
Then: Registry entry is updated: followed counter incremented

---

## L4 → ARCH Traceability (UC-23)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-52 | T-182 |
| ARCH-53 | T-183 |
| ARCH-54 | T-184, T-185 |
| ARCH-55 | T-186 |
| ARCH-56 | T-187, T-188 |
| ARCH-57 | T-189 |
| ARCH-58 | T-190, T-191 |
| ARCH-59 | T-192, T-193, T-194 |
| ARCH-60 | T-195 |
| ARCH-61 | T-196, T-197, T-198, T-199, T-200 |

All UC-23 ARCH items have test coverage.

---

## UC-4: Skill Generation Tests

### T-238: Candidate detection — threshold filtering

**Traces to:** ARCH-62

Given: Registry with 5 memories: surfaced_count = [10, 50, 75, 100, 200], threshold = 50
When: I call Candidates(50)
Then: Returns 4 candidates (50, 75, 100, 200) sorted by surfaced_count descending, excluding the one below threshold

---

### T-239: Candidate detection — excludes non-memory sources

**Traces to:** ARCH-62

Given: Registry with entries: memory (surfaced=100), claude-md (surfaced=200), skill (surfaced=150)
When: I call Candidates(50)
Then: Returns only the memory entry; claude-md and skill entries excluded

---

### T-240: Skill file generation — valid format

**Traces to:** ARCH-63

Given: A memory with title="Use targ build", content="Always use targ...", principle="Use targ for builds", anti_pattern="Don't run raw go test", keywords=["targ", "build"]
When: I generate a skill file
Then: Output has YAML frontmatter with description, markdown body with title, principle section, "What to avoid" section, and context section

---

### T-241: Skill file generation — no anti_pattern

**Traces to:** ARCH-63

Given: A memory with title and content but no anti_pattern (tier C factual)
When: I generate a skill file
Then: Output omits the "What to avoid" section; other sections present

---

### T-242: Plugin registration — write to skills directory

**Traces to:** ARCH-62

Given: A generated skill file and skills directory exists
When: I call SkillWriter.Write("use-targ-build", content)
Then: File written to skills/use-targ-build.md with correct content

---

### T-243: Plugin registration — name collision error

**Traces to:** ARCH-62

Given: skills/use-targ-build.md already exists
When: I call SkillWriter.Write("use-targ-build", content)
Then: Error returned, existing file not overwritten

---

### T-244: Source retirement — merge and delete

**Traces to:** ARCH-62

Given: Memory "memory:use-targ-build.toml" in registry, new skill "skill:use-targ-build" registered
When: I retire the source memory
Then: Registry.Merge called (memory→skill), memory TOML deleted, skill's absorbed array contains memory's counters

---

### T-245: Promote flow — full end-to-end with confirmation

**Traces to:** ARCH-62

Given: A memory above threshold, LLM returns valid skill content, user confirms "y"
When: I call Promote(ctx, candidateID)
Then: Skill file written, registry merged, memory deleted, confirmation output shown

---

### T-246: Promote flow — user declines

**Traces to:** ARCH-62

Given: A memory above threshold, user responds "n"
When: I call Promote(ctx, candidateID)
Then: No files written, no registry changes, no deletions

---

### T-247: CLI subcommand promote --to-skill

**Traces to:** ARCH-62

Given: `engram promote --to-skill --data-dir <dir> --threshold 50`
When: Registry has 2 candidates above threshold
Then: Candidate list displayed, user prompted for selection

---

## UC-5: CLAUDE.md Management Tests

### T-248: Promotion candidate detection — Working skills

**Traces to:** ARCH-64

Given: Registry with skills: Working (surfaced=150), Leech (surfaced=100), HiddenGem (surfaced=30)
When: I detect promotion candidates with threshold=100
Then: Returns only the Working skill (surfaced=150)

---

### T-249: Demotion candidate detection — Leech claude-md entries

**Traces to:** ARCH-64

Given: Registry with claude-md entries: Working (eff=0.85), Leech (eff=0.20)
When: I detect demotion candidates
Then: Returns only the Leech entry

---

### T-250: CLAUDE.md entry generation — matches style

**Traces to:** ARCH-64

Given: A skill with title, content, principle; existing CLAUDE.md with bullet-point style
When: LLM generates a CLAUDE.md entry
Then: Generated entry follows bullet-point style with source traceability comment

---

### T-251: CLAUDE.md add entry — section appended

**Traces to:** ARCH-65

Given: Existing CLAUDE.md with 3 sections
When: I call AddEntry(content, newEntry)
Then: New section appended, other sections unchanged

---

### T-252: CLAUDE.md remove entry — section removed

**Traces to:** ARCH-65

Given: CLAUDE.md with entry containing `<!-- promoted from skill:X -->`
When: I call RemoveEntry(content, "skill:X")
Then: That section removed, other sections unchanged

---

### T-253: Demotion execution — CLAUDE.md entry to skill

**Traces to:** ARCH-64, ARCH-65

Given: A Leech CLAUDE.md entry, user confirms
When: I execute demotion
Then: Skill file generated from entry content, entry removed from CLAUDE.md, registry merged (claude-md→skill)

---

### T-254: CLI subcommand promote --to-claude-md

**Traces to:** ARCH-64

Given: `engram promote --to-claude-md --data-dir <dir>`
When: Registry has 1 Working skill candidate
Then: Candidate displayed with evidence, user prompted for confirmation

---

### T-255: CLI subcommand demote --to-skill

**Traces to:** ARCH-64

Given: `engram demote --to-skill --data-dir <dir>`
When: Registry has 1 Leech claude-md entry
Then: Candidate displayed with evidence, user prompted for confirmation

---

## UC-24: Proposal Application Tests

### T-256: Proposal ingestion — valid JSON

**Traces to:** ARCH-66

Given: JSON array with 3 proposals (Working, Leech, Noise)
When: I ingest proposals from stdin
Then: All 3 parsed with correct quadrant, action, target_path, evidence

---

### T-257: Proposal ingestion — invalid schema skipped

**Traces to:** ARCH-66

Given: JSON array with 2 valid proposals and 1 missing required fields
When: I ingest proposals
Then: 2 valid proposals parsed, invalid one skipped with warning

---

### T-258: Working staleness — content rewrite

**Traces to:** ARCH-66, ARCH-67

Given: A Working proposal with action "update_content", memory TOML exists
When: LLM generates rewrite, user confirms
Then: Memory TOML rewritten with new content, other fields preserved, registry content_hash updated

---

### T-259: Leech rewrite — root cause content_quality

**Traces to:** ARCH-66, ARCH-67

Given: A Leech proposal with root cause "content_quality"
When: LLM generates rewrite, user confirms
Then: Memory principle and anti_pattern rewritten, keywords preserved

---

### T-260: HiddenGem broadening — keywords added

**Traces to:** ARCH-66, ARCH-67

Given: A HiddenGem proposal, memory has keywords=["targ", "build"]
When: LLM suggests ["test", "check"] as additions, user confirms
Then: Memory keywords updated to ["targ", "build", "test", "check"]

---

### T-261: Noise removal — file deleted and registry entry removed

**Traces to:** ARCH-66

Given: A Noise proposal, memory TOML exists, registry entry exists
When: User confirms removal
Then: Memory TOML deleted, registry entry removed

---

### T-262: User confirmation — skip and quit

**Traces to:** ARCH-66

Given: 3 proposals, user responds "a" (apply), "s" (skip), "q" (quit)
When: Walking proposals
Then: First applied, second skipped, third not reached. Report: "Applied 1/3 (1 skipped, 1 not reached)"

---

### T-263: No-token behavior — LLM proposals skipped

**Traces to:** ARCH-66

Given: 2 proposals (1 Working rewrite, 1 Noise removal), no API token
When: I apply proposals
Then: Working skipped ("no token"), Noise applied (deterministic). Report: "Applied 1/2 (1 skipped: no token)"

---

### T-264: CLI subcommand maintain --apply

**Traces to:** ARCH-66

Given: `engram maintain --apply --data-dir <dir> --proposals <file>`
When: File has 4 proposals
Then: Summary displayed, then per-proposal confirm/apply loop

---

### T-265: Memory TOML rewriter — atomic write preserves fields

**Traces to:** ARCH-67

Given: Memory TOML with title, content, keywords, concepts, observation_type, principle
When: I rewrite only content and principle
Then: title, keywords, concepts, observation_type unchanged; content and principle updated

---

## UC-25: Evaluate Strip Preprocessing Tests

### T-266: Strip applied before LLM evaluation

**Traces to:** ARCH-68

Given: Transcript with 100 lines (80 tool results, 20 conversation)
When: Evaluator runs with StripFunc=sessionctx.Strip
Then: LLM receives only the 20 conversation lines

---

### T-267: Empty post-strip transcript — evaluation skipped

**Traces to:** ARCH-68

Given: Transcript that is entirely tool results (0 conversation lines after strip)
When: Evaluator runs with StripFunc
Then: No LLM call made, no error returned, stderr message logged

---

### T-268: Default StripFunc is no-op — backward compatible

**Traces to:** ARCH-68

Given: Evaluator created without WithStripFunc option
When: Evaluator runs with a transcript
Then: Full transcript passed to LLM (no stripping)

---

### T-269: CLI wiring injects sessionctx.Strip

**Traces to:** ARCH-68

Given: `engram evaluate --data-dir <dir>` with transcript on stdin
When: CLI wires evaluator
Then: WithStripFunc(sessionctx.Strip) is passed to evaluator

---

## L4 → ARCH Traceability (UC-4, UC-5, UC-24, UC-25)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-62 | T-238, T-239, T-242, T-243, T-244, T-245, T-246, T-247 |
| ARCH-63 | T-240, T-241 |
| ARCH-64 | T-248, T-249, T-250, T-253, T-254, T-255 |
| ARCH-65 | T-251, T-252, T-253 |
| ARCH-66 | T-256, T-257, T-258, T-259, T-260, T-261, T-262, T-263, T-264 |
| ARCH-67 | T-258, T-259, T-260, T-265 |
| ARCH-68 | T-266, T-267, T-268, T-269 |

All ARCH items have test coverage.

---

## UC-26: First-Class Non-Memory Instruction Sources Tests

### T-270: Discover CLAUDE.md sources

**Traces to:** ARCH-69

Given: Two CLAUDE.md files exist (project and global), each with bullet items
When: Registrar discovers sources with configured paths
Then: Entries extracted from both files using ClaudeMDExtractor, source type "claude-md"

---

### T-271: Discover rule files

**Traces to:** ARCH-69

Given: Rules directory contains 2 rule files
When: Registrar discovers sources
Then: Entries extracted for each rule file using RuleExtractor, source type "rule"

---

### T-272: Discover skill files

**Traces to:** ARCH-69

Given: Skills directory contains 2 skill subdirectories with SKILL.md files
When: Registrar discovers sources
Then: Entries extracted for each skill using SkillExtractor, source type "skill"

---

### T-273: Register new entries

**Traces to:** ARCH-69

Given: Registry is empty, 3 sources discovered (1 claude-md, 1 rule, 1 skill)
When: Registrar runs registration phase
Then: Registry.Register called for each entry with correct ID, source type, content hash

---

### T-274: Update changed entries — content hash differs

**Traces to:** ARCH-69

Given: Registry has entry with ID "rule:go.md" and content hash "abc123"
When: Registrar discovers same rule with different content (new hash "def456")
Then: Entry updated with new content_hash and updated_at; surfaced_count and evaluations preserved

---

### T-275: Stale entry pruning — removes absent non-memory entries

**Traces to:** ARCH-69

Given: Registry has entries ["rule:go.md", "rule:deleted.md", "memory:foo.toml"]
When: Registrar discovers only ["rule:go.md"] (deleted.md source file gone)
Then: "rule:deleted.md" removed via Registry.Remove; "memory:foo.toml" untouched

---

### T-276: Memory entries never pruned

**Traces to:** ARCH-69

Given: Registry has memory entry "memory:old-memory.toml" not in discovered set
When: Registrar runs pruning phase
Then: Memory entry is NOT removed (memory pruning is UC-16 Noise removal)

---

### T-277: Implicit surfacing — records for always-loaded entries

**Traces to:** ARCH-69

Given: Registry has 3 always-loaded entries (1 claude-md, 1 rule, 1 skill)
When: Registrar runs implicit surfacing phase
Then: RecordSurfacing called on registry for each; LogSurfacing called on surfacing log for each with mode "session-start"

---

### T-278: Missing source paths silently skipped

**Traces to:** ARCH-69

Given: Configured CLAUDE.md path does not exist, rules dir does not exist
When: Registrar discovers sources
Then: No error returned; discovered set contains only entries from existing paths

---

### T-279: Idempotent registration — second run is no-op

**Traces to:** ARCH-69

Given: Registrar already ran once (3 entries registered)
When: Registrar runs again with same source files unchanged
Then: No new registrations, no removals, no content hash updates; surfacing incremented again (expected — once per session)

---

### T-280: Fire-and-forget — registration errors don't fail hook

**Traces to:** ARCH-69

Given: Registry.Register returns an error for one entry
When: Registrar runs
Then: Error logged to stderr; remaining entries still registered; no error returned from Run

---

### T-323: Merge rejects non-memory source type

**Traces to:** ARCH-23 (registry Merge operation)

Given: Registry has one entry with source_type "rule" and one with source_type "memory"
When: Merge is called with either order
Then: Returns ErrMergeSourceType; neither entry is modified

---

### T-P0a-1: New InstructionEntry defaults enforcement_level to advisory

**Traces to:** REQ-P0a-1 (EnforcementLevel field default)

Given: An InstructionEntry is registered with no EnforcementLevel set
When: The entry is retrieved from the store
Then: EnforcementLevel equals EnforcementAdvisory

---

### T-P0a-2: Load backfills missing enforcement_level to advisory

**Traces to:** REQ-P0a-1 (EnforcementLevel backfill on load)

Given: A JSONL file containing an entry with no enforcement_level field
When: The store loads and the entry is retrieved
Then: EnforcementLevel equals EnforcementAdvisory

---

### T-283: Session-start mode triggers auto-registration

**Traces to:** ARCH-71

Given: `engram surface --mode session-start --data-dir <dir>`
When: CLI runs surface command
Then: Registrar.Run is called before memory surfacing

---

### T-284: Non-session-start modes skip auto-registration

**Traces to:** ARCH-71

Given: `engram surface --mode prompt --data-dir <dir>`
When: CLI runs surface command
Then: Registrar.Run is NOT called; only memory surfacing executes

---

## L4 → ARCH Traceability (UC-26)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-69 | T-270, T-271, T-272, T-273, T-274, T-275, T-276, T-277, T-278, T-279, T-280 |
| ARCH-71 | T-283, T-284 |

All ARCH items have test coverage.

---

## UC-27: Global Binary Installation

### T-285: Symlink created when none exists

**Traces to:** ARCH-72 (REQ-119)

Given: `~/.local/bin/engram` does not exist, `~/.claude/engram/bin/engram` is a valid binary
When: SessionStart hook runs
Then: `~/.local/bin/engram` is a symlink pointing to `~/.claude/engram/bin/engram`

- Verification: shell (readlink check)

---

### T-286: Target directory created if missing

**Traces to:** ARCH-72 (REQ-119)

Given: `~/.local/bin/` directory does not exist
When: SessionStart hook runs
Then: `~/.local/bin/` is created and symlink is placed inside it

- Verification: shell (directory existence + symlink check)

---

### T-287: Idempotent — correct symlink unchanged

**Traces to:** ARCH-72 (REQ-120)

Given: `~/.local/bin/engram` already symlinks to `~/.claude/engram/bin/engram`
When: SessionStart hook runs
Then: Symlink unchanged, no errors logged

- Verification: shell (readlink before/after, stderr empty)

---

### T-288: Stale symlink not replaced (no-clobber)

**Traces to:** ARCH-72 (REQ-120, REQ-121)

Given: `~/.local/bin/engram` symlinks to `/old/path/engram` (wrong target)
When: SessionStart hook runs
Then: Warning logged to stderr, symlink NOT replaced

- Verification: shell (readlink unchanged, stderr contains warning)

---

### T-289: No-clobber — regular file preserved

**Traces to:** ARCH-72 (REQ-121)

Given: `~/.local/bin/engram` is a regular file (not a symlink)
When: SessionStart hook runs
Then: File preserved intact, warning logged to stderr

- Verification: shell (file unchanged, stderr contains warning)

---

### T-290: Fire-and-forget — permission error doesn't block session

**Traces to:** ARCH-72 (REQ-122)

Given: `~/.local/bin/` is read-only (permission denied for symlink creation)
When: SessionStart hook runs
Then: Session start continues, surface output still produced, error logged to stderr

- Verification: shell (hook exits 0, stdout has surface JSON)

---

## L4 → ARCH Traceability (UC-27)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-72 | T-285, T-286, T-287, T-288, T-289, T-290 |

All ARCH items have test coverage.

---

## UC-28: Automatic Maintenance and Promotion Triggers Tests

### T-291: Noise memory detected → noise_removal signal

**Traces to:** ARCH-73 (REQ-123)

Given: A memory with surfaced_count=10, effectiveness=0% (all ignored), ≥5 evaluations
When: Detector.Detect runs
Then: Signal with type="maintain", signal="noise_removal", quadrant="Noise" is returned

---

### T-292: Leech memory detected → leech_rewrite signal

**Traces to:** ARCH-73 (REQ-123)

Given: A memory with surfaced_count=20, effectiveness=15%, ≥5 evaluations (high surfacing, low follow-through)
When: Detector.Detect runs
Then: Signal with type="maintain", signal="leech_rewrite", quadrant="Leech" is returned

---

### T-293: Stale Working memory → staleness_review signal

**Traces to:** ARCH-73 (REQ-123)

Given: A memory in Working quadrant with last_surfaced >90 days ago
When: Detector.Detect runs
Then: Signal with type="maintain", signal="staleness_review", quadrant="Working" is returned

---

### T-294: Hidden Gem detected → hidden_gem_broadening signal

**Traces to:** ARCH-73 (REQ-123)

Given: A memory with surfaced_count=2, effectiveness=90%, ≥5 evaluations (low surfacing, high follow-through)
When: Detector.Detect runs
Then: Signal with type="maintain", signal="hidden_gem_broadening", quadrant="HiddenGem" is returned

---

### T-295: Memory with <5 evaluations → no signal

**Traces to:** ARCH-73 (REQ-123)

Given: A memory with surfaced_count=3, 2 evaluations (InsufficientData quadrant)
When: Detector.Detect runs
Then: No signal generated for this memory

---

### T-296: Memory-to-skill candidate detected

**Traces to:** ARCH-73 (REQ-124)

Given: A memory with surfaced_count >= promotion threshold, not InsufficientData quadrant
When: Detector.Detect runs
Then: Signal with type="promote", signal="memory_to_skill" is returned

---

### T-297: Skill promotion candidate emits graduation signal

**Traces to:** ARCH-73 (REQ-124)

Given: A skill in Working quadrant with surfaced_count >= promotion threshold
When: Detector.Detect runs
Then: Signal with type="promote", signal="graduation" is returned, and Summary contains recommendation text about promoting to CLAUDE.md

---

### T-298: CLAUDE.md demotion candidate emits graduation signal

**Traces to:** ARCH-73 (REQ-124)

Given: A CLAUDE.md entry in Leech quadrant
When: Detector.Detect runs
Then: Signal with type="promote", signal="graduation" is returned, and Summary contains recommendation text about demoting to skill

---

### T-299: Queue write creates file if absent

**Traces to:** ARCH-74 (REQ-125)

Given: No proposal-queue.jsonl exists
When: QueueStore.Append is called with signals
Then: File created with one JSON line per signal

---

### T-300: Queue write is atomic (temp+rename)

**Traces to:** ARCH-74 (REQ-125)

Given: An existing proposal-queue.jsonl
When: QueueStore.Append is called
Then: createTemp is called, content written to temp, rename to final path

---

### T-301: Queue read skips malformed lines

**Traces to:** ARCH-74 (REQ-125)

Given: A proposal-queue.jsonl with 3 valid lines and 1 malformed line
When: QueueStore.Read is called
Then: Returns 3 signals, malformed line skipped without error

---

### T-302: Stale entries (>30d) pruned on detect

**Traces to:** ARCH-74 (REQ-126)

Given: Queue with entries: one 10 days old, one 45 days old
When: QueueStore.Prune is called
Then: 45-day entry removed, 10-day entry preserved

---

### T-303: Entries for deleted memories pruned

**Traces to:** ARCH-74 (REQ-126)

Given: Queue with entry for memory path that no longer exists
When: QueueStore.Prune is called with existsCheck returning false
Then: Entry removed

---

### T-304: Duplicate signals deduplicated

**Traces to:** ARCH-74 (REQ-126)

Given: Queue with two entries having same source_id + type
When: QueueStore.Prune is called
Then: Only newest entry preserved

---

### T-305: SessionStart surfaces enriched signal detail with memory content

**Traces to:** ARCH-75 (REQ-127)

Given: Queue with 2 signals, corresponding memory TOMLs exist
When: signal-surface runs
Then: Output includes memory title, content excerpt, effectiveness stats, and CLI action instructions for each signal

---

### T-306: Empty queue produces no output

**Traces to:** ARCH-75 (REQ-127)

Given: Empty proposal-queue.jsonl (or file doesn't exist)
When: signal-surface runs
Then: No output produced

---

### T-307: signal-surface output is valid hook JSON

**Traces to:** ARCH-75, ARCH-76 (DES-46)

Given: Queue with signals
When: signal-surface --format json runs
Then: Output is valid JSON with `additionalContext` field compatible with SessionStart hook schema

---

### T-309: Detection with missing registry → skip silently

**Traces to:** ARCH-73 (REQ-123)

Given: No instruction-registry.jsonl exists
When: Detector.Detect runs
Then: Returns empty signal list, no error

---

### T-310: Stop hook calls signal-detect after audit

**Traces to:** ARCH-76 (DES-45)

Given: A complete Stop hook script
When: Inspecting hook flow
Then: signal-detect invocation appears after audit, with fire-and-forget error handling

---

### T-311: SessionStart merges signal-surface into existing output

**Traces to:** ARCH-76 (DES-44)

Given: SessionStart hook produces surface output + signal-surface produces signal output
When: SessionStart hook runs
Then: Both outputs merged into single additionalContext

---

### T-312: Remove action deletes TOML + removes registry entry

**Traces to:** ARCH-77 (REQ-128)

Given: A memory TOML file exists and is registered
When: Applier.Apply with action="remove"
Then: TOML file deleted, registry entry removed, queue entry cleared

---

### T-313: Rewrite action updates TOML fields + registry hash

**Traces to:** ARCH-77 (REQ-128)

Given: A memory TOML file exists with title="old" and content="old content"
When: Applier.Apply with action="rewrite", fields={"title":"new","content":"new content"}
Then: TOML updated with new values, registry content_hash updated

---

### T-314: Broaden action appends keywords to TOML

**Traces to:** ARCH-77 (REQ-128)

Given: A memory TOML with keywords=["go","test"]
When: Applier.Apply with action="broaden", keywords=["lint","ci"]
Then: TOML keywords=["go","test","lint","ci"]

---

### T-315: Escalate action updates escalation_level field

**Traces to:** ARCH-77 (REQ-128)

Given: A memory TOML with escalation_level=1
When: Applier.Apply with action="escalate", level=2
Then: TOML escalation_level=2

---

### T-316: Apply clears matching signal from queue

**Traces to:** ARCH-77 (REQ-128)

Given: Queue has signal for memory "x.toml"
When: Applier.Apply runs successfully for "x.toml"
Then: Signal for "x.toml" removed from queue

---

### T-317: Apply with missing memory file returns error

**Traces to:** ARCH-77 (REQ-128)

Given: Memory path points to nonexistent file
When: Applier.Apply with action="rewrite"
Then: ApplyResult with success=false, error describes missing file

---

### T-318: Apply rewrite is atomic (temp+rename)

**Traces to:** ARCH-77 (REQ-128)

Given: A memory TOML file exists
When: Applier.Apply with action="rewrite"
Then: writeFile uses atomic temp+rename pattern

---

### T-319: Promote --content skips LLM generation

**Traces to:** ARCH-78 (REQ-129)

Given: Promote called with PromoteOpts{Content: "skill content"}
When: Promote executes
Then: Generator.Generate is NOT called, supplied content used instead

---

### T-320: Promote --yes skips confirmation prompt

**Traces to:** ARCH-78 (REQ-129)

Given: Promote called with PromoteOpts{SkipConfirm: true}
When: Promote executes
Then: Confirmer.Confirm is NOT called

---

### T-321: Promote --content + --yes still does registry merge

**Traces to:** ARCH-78 (REQ-129)

Given: Promote called with PromoteOpts{Content: "...", SkipConfirm: true}
When: Promote executes
Then: Registry merge still happens (source merged into target)

---

### T-322: Promote without --content uses existing LLM flow

**Traces to:** ARCH-78 (REQ-129)

Given: Promote called with zero-value PromoteOpts
When: Promote executes
Then: Generator.Generate IS called (existing flow unchanged)

---

## L4 → ARCH Traceability (UC-28)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-73 | T-291, T-292, T-293, T-294, T-295, T-296, T-297, T-298, T-309 |
| ARCH-74 | T-299, T-300, T-301, T-302, T-303, T-304 |
| ARCH-75 | T-305, T-306, T-307, T-311 |
| ARCH-76 | T-307, T-310, T-311 |
| ARCH-77 | T-312, T-313, T-314, T-315, T-316, T-317, T-318 |
| ARCH-78 | T-319, T-320, T-321, T-322 |

All ARCH items have test coverage.

---

## P6e: Escalation wiring to enforcement_level

### T-P6e-1: LevelGraduated is 4th escalation level

**Traces to:** ARCH-P6e-1 (REQ-P6e-1)

Given: EscalationEngine with memory at `reminder` level
When: Analyze called
Then: Proposal type is "escalate" and ProposedLevel is "graduated"

---

### T-P6e-2: ApplyEscalationProposal calls SetEnforcementLevel

**Traces to:** ARCH-P6e-1 (REQ-P6e-2)

Given: ApplyEscalationProposal called with a proposal and mock EnforcementApplier
When: Function executes
Then: SetEnforcementLevel called with MemoryPath, ProposedLevel, and Rationale

---

### T-P6e-3: ApplyEscalationProposal with nil applier is no-op

**Traces to:** ARCH-P6e-1 (REQ-P6e-2)

Given: ApplyEscalationProposal called with nil applier
When: Function executes
Then: Returns nil, no panic

---

### T-P6e-4: ApplyEscalationProposal to graduated emits graduation signal

**Traces to:** ARCH-P6e-1 (REQ-P6e-3)

Given: ApplyEscalationProposal called with ProposedLevel="graduated" and mock GraduationEmitter
When: Function executes
Then: EmitGraduation called with MemoryPath, non-empty recommendation, and correct timestamp

---

### T-P6e-5: ApplyEscalationProposal to non-graduated does NOT emit signal

**Traces to:** ARCH-P6e-1 (REQ-P6e-3)

Given: ApplyEscalationProposal called with ProposedLevel="emphasized_advisory"
When: Function executes
Then: EmitGraduation is NOT called

---

### T-P6e-6: ClassifyContent returns "settings.json" for linter/config content

**Traces to:** ARCH-P6e-1 (REQ-P6e-4, DES-P6e-1)

Given: Content contains "golangci linter settings"
When: ClassifyContent called
Then: Returns "settings.json"

---

### T-P6e-7: ClassifyContent returns ".claude/rules/" for file-scoped content

**Traces to:** ARCH-P6e-1 (REQ-P6e-4, DES-P6e-1)

Given: Content contains "glob pattern"
When: ClassifyContent called
Then: Returns ".claude/rules/"

---

### T-P6e-8: ClassifyContent returns "skill" for procedural content

**Traces to:** ARCH-P6e-1 (REQ-P6e-4, DES-P6e-1)

Given: Content contains "step-by-step procedure"
When: ClassifyContent called
Then: Returns "skill"

---

### T-P6e-9: ClassifyContent returns "CLAUDE.md" as behavioral default

**Traces to:** ARCH-P6e-1 (REQ-P6e-4, DES-P6e-1)

Given: Content contains no classification keywords
When: ClassifyContent called
Then: Returns "CLAUDE.md"

---

### T-P6e-10: ApplyEscalationProposal propagates applier error

**Traces to:** ARCH-P6e-1 (REQ-P6e-2)

Given: EnforcementApplier.SetEnforcementLevel returns error
When: ApplyEscalationProposal called
Then: Returns wrapped error containing "setting enforcement level"

---

### T-P6e-11: emphasized_advisory renders with IMPORTANT: prefix in tool mode

**Traces to:** ARCH-P6e-2 (REQ-P6e-5, DES-P6e-2)

Given: Memory at emphasized_advisory level, tool mode surfacing
When: Surface Run executes
Then: Output contains "IMPORTANT:" and memory slug

---

### T-P6e-12: reminder renders with REMINDER: prefix in tool mode

**Traces to:** ARCH-P6e-2 (REQ-P6e-6, DES-P6e-2)

Given: Memory at reminder level, tool mode surfacing
When: Surface Run executes
Then: Output contains "REMINDER:" and memory slug

---

### T-P6e-13: advisory level renders with normal format

**Traces to:** ARCH-P6e-2 (REQ-P6e-5, DES-P6e-2)

Given: No EnforcementReader set (all memories default to advisory)
When: Surface Run executes in tool mode
Then: Output does NOT contain "IMPORTANT:" or "REMINDER:"

---

### T-P6e-14: emphasized_advisory memories sorted before advisory in tool mode

**Traces to:** ARCH-P6e-2 (REQ-P6e-7)

Given: Two matching memories — one advisory, one emphasized_advisory
When: Surface Run executes in tool mode
Then: emphasized_advisory memory appears first in output

---

## L4 → ARCH Traceability (P6e)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-P6e-1 | T-P6e-1, T-P6e-2, T-P6e-3, T-P6e-4, T-P6e-5, T-P6e-6, T-P6e-7, T-P6e-8, T-P6e-9, T-P6e-10 |
| ARCH-P6e-2 | T-P6e-11, T-P6e-12, T-P6e-13, T-P6e-14 |

---

## P1: Contradiction Detection Tests

### T-P1-1: Detector returns empty for single memory
A slice with one memory produces no pairs. (unit, contradict pkg)

### T-P1-2: Heuristic fires on opposing verb pair (use/avoid)
Memory A principle: "Always use targ". Memory B principle: "Avoid using targ". Expect pair returned. (unit)

### T-P1-3: Heuristic fires on always/never pair
Memory A: "Always add t.Parallel()". Memory B: "Never use t.Parallel() in benchmarks". Expect pair returned. (unit)

### T-P1-4: Heuristic does not fire on unrelated memories
Memory A about git commits. Memory B about test patterns. No pair. (unit)

### T-P1-5: BM25 high similarity without heuristic → borderline, sent to LLM
Two nearly-identical memories that don't trigger verb heuristic but score high BM25. Classifier called. (unit with mock)

### T-P1-6: LLM budget enforced — max 3 calls
6 borderline pairs: only first 3 trigger Classify calls. (unit with mock classifier counting calls)

### T-P1-7: Classifier error treated as non-contradiction
Classifier returns error. Pair not included in result. (unit)

### T-P1-8: High-confidence pair skips LLM
Heuristic fires AND BM25 > 0.3 → pair returned without Classify call. (unit)

### T-P1-9: KindContradiction constant value is "contradiction"
Compile-time check. (unit, signal pkg)

### T-P1-10: Surface runSessionStart suppresses contradicted memory
Mock detector returns pair (A, B). B removed from output. Signal emitted for B. (unit, surface pkg)

### T-P1-11: Surface proceeds without suppression when detector is nil
No detector set. No panic. Normal output. (unit, surface pkg)

### T-P1-12: Detector with no classifier skips LLM phase
Borderline pair: no Classify called, pair not returned. (unit)

---

### T-P4e-1: SessionStart limits to top 7
15 memories, no effectiveness data → 7 surfaced. (unit, surface pkg)

### T-P4e-2: SessionStart gates out low-effectiveness memories
Memory with SurfacedCount=10, score=20% excluded; memory with score=75% included. (unit)

### T-P4e-3: SessionStart includes insufficient-data memories regardless of score
Memory with SurfacedCount=3, score=0% included. No-data memory included. (unit)

### T-P4e-4: SessionStart ranks by effectiveness descending
Low-scorer listed first in retriever; high-scorer appears first in output. (unit)

### T-P4e-5: Default budgets are 600/250/150
Compile-time assertion on DefaultSessionStartBudget, DefaultUserPromptSubmitBudget, DefaultPreToolUseBudget. (unit)

### T-P4e-6: PreToolUse limits to top 2
5 matching memories → 2 surfaced. (unit)

### T-P4e-7: PreToolUse gates out low-effectiveness memories
Memory with SurfacedCount=10, score=15% excluded; score=75% included. (unit)

### T-P4e-8: InvocationTokenLogger called with positive token count
SessionStart with one memory → LogInvocationTokens called once with mode="session-start" and tokenCount>0. (unit)

### T-P4e-9: LogInvocationTokens appends token-count event to surfacing log
Event JSON contains token_count and mode; no memory_path field. (unit, surfacinglog pkg)

### T-P4e-10: LogInvocationTokens propagates append errors
Real logger with /dev/null path → error wrapped as "appending invocation token log". (unit)

---

## Tests for UC-33: Merge-on-Write (P5c)

### T-P5c-1: High overlap returns merge pair not surviving candidate
Candidate with 3/4 matching keywords against existing → `ClassifyResult.MergePairs` has 1 entry; `Surviving` is empty. (unit, dedup pkg)

### T-P5c-2: Low overlap passes through as surviving candidate
Candidate with 2/4 matching keywords (exactly 50%) → `ClassifyResult.Surviving` has 1 entry; `MergePairs` is empty. (unit, dedup pkg)

### T-P5c-3: LLM merger called with existing and candidate principles
`MergePrinciples` receives existing.Principle and candidate.Principle; mock returns merged string; merged principle passed to `MergeWriter`. (unit, learn pkg)

### T-P5c-4: Fallback merge takes candidate principle when it is longer
No merger configured; candidate principle longer than existing → merged TOML uses candidate principle; keywords = union; concepts = union. (unit, learn pkg)

### T-P5c-5: Fallback merge keeps existing principle when it is longer
No merger configured; existing principle longer → merged TOML keeps existing principle; keywords still unioned. (unit, learn pkg)

### T-P5c-6: Absorbed record appended after merge
After merge: `RegistryAbsorber.RecordAbsorbed` called once with existing file path, candidate title, non-empty hash, and timestamp. (unit, learn pkg with mock absorber)

### T-P5c-7: LLM merger error falls back to deterministic merge
`MergePrinciples` returns error → fallback applied (longer principle, union keywords, union concepts); no error propagated. (unit, learn pkg)

### T-P5c-8: Empty candidate keywords are not merged
Candidate with no keywords → appears in `Surviving` (create-new path), not in `MergePairs`. (unit, dedup pkg)

---

## P3: Memory Graph with Spreading Activation Tests

### T-P3-1: Jaccard zero for disjoint token sets
`tokenSet("foo bar") ∩ tokenSet("baz qux") = ∅` → Jaccard = 0. No link produced. (unit, graph pkg)

### T-P3-2: Jaccard correct for overlapping sets
`tokenSet("use targ build") ∩ tokenSet("use targ test") = {use, targ}`, union size 4 → Jaccard = 0.5. Link produced with Weight=0.5. (unit, graph pkg)

### T-P3-3: BuildConceptOverlap threshold 0.15
3 existing entries: one with Jaccard=0.2, one with 0.1, one with 0.0 against new entry → only the 0.2 entry produces a concept_overlap link. (unit, graph pkg)

### T-P3-4: BuildConceptOverlap self-link excluded
Entry in existing list has same ID as new entry → no self-link in output. (unit, graph pkg)

### T-P3-5: BuildContentSimilarity threshold 0.05
Entry scoring BM25=0.06 raw → content_similarity link with Weight=min(1.0,0.06/5.0). Entry scoring 0.03 → no link. (unit, graph pkg)

### T-P3-6: BuildContentSimilarity weight capped at 1.0
Entry scoring BM25=10.0 raw → Weight=1.0 (capped). (unit, graph pkg)

### T-P3-7: UpdateCoSurfacing increments existing co_surfacing link
Existing link: Weight=0.5, CoSurfacingCount=3 → after update: Weight=0.6, CoSurfacingCount=4. (unit, graph pkg)

### T-P3-8: UpdateCoSurfacing creates new link if none exists
No existing co_surfacing link → creates one with Weight=0.1, CoSurfacingCount=1. (unit, graph pkg)

### T-P3-9: UpdateCoSurfacing caps weight at 1.0
Existing link: Weight=0.95, CoSurfacingCount=5 → after update: Weight=1.0, CoSurfacingCount=6. (unit, graph pkg)

### T-P3-10: Prune removes links below threshold with sufficient count
Link: Weight=0.05, CoSurfacingCount=10 → removed. (unit, graph pkg)

### T-P3-11: Prune preserves links below threshold with insufficient count
Link: Weight=0.05, CoSurfacingCount=9 → preserved. (unit, graph pkg)

### T-P3-12: Prune preserves links at or above weight threshold regardless of count
Link: Weight=0.1, CoSurfacingCount=20 → preserved. (unit, graph pkg)

### T-P3-13: Registry.UpdateLinks stores and retrieves links
Call UpdateLinks with 2 links, then Get → entry.Links contains the 2 links. (unit, registry pkg)

### T-P3-14: Registry.UpdateLinks returns ErrNotFound for unknown id
UpdateLinks("nonexistent", links) → ErrNotFound. (unit, registry pkg)

### T-P3-15: Registry.UpdateLinks replaces existing links entirely
Entry has 3 links, UpdateLinks called with 1 link → entry has 1 link. (unit, registry pkg)

### T-P3-16: applySpreadingActivation base + linked contribution
Memory A (base=1.0) links to B (base=2.0, weight=0.5). Activated A = 1.0 + 0.3×(2.0×0.5) = 1.3. (unit, surface pkg)

### T-P3-17: applySpreadingActivation linked target not in candidate set → zero contribution
Memory A links to C which is not in baseScores → C contributes 0. Activated A = base. (unit, surface pkg)

### T-P3-18: applySpreadingActivation with no LinkReader → unchanged scores
No LinkReader set → scores unchanged from base. (unit, surface pkg)

### T-P3-19: Surface co_surfacing update called for all pairs in top-N
3 memories surfaced → 3 pairs (A-B, A-C, B-C) each get UpdateCoSurfacing call. (unit, surface pkg with mock LinkUpdater)

### T-P3-20: Surface co_surfacing update error does not abort surfacing
LinkUpdater.SetEntryLinks returns error → surface proceeds, output unchanged. (unit, surface pkg)

### T-P3-21: formatClusterNotes returns top-2 links by weight
Memory has 3 links with weights 0.9, 0.7, 0.3 → only top-2 noted (0.9, 0.7). (unit, surface pkg)

### T-P3-22: formatClusterNotes skips links with no known title
TitleFetcher returns (_, false) for a link target → that link excluded from notes. (unit, surface pkg)

### T-P3-23: formatClusterNotes format is "  • see also: <title>"
Link target title "use targ for builds" → note line is `  • see also: use targ for builds`. (unit, surface pkg)

### T-P3-24: Surface cluster notes absent when no TitleFetcher set
No TitleFetcher option → no cluster note lines in output. (unit, surface pkg)

### T-P3-25: updateEvaluationCorrelationLinks updates links for followed pairs
2 memories both followed → each gets evaluation_correlation link to the other (+0.05). (unit, evaluate pkg with mock EvalLinkUpdater)

### T-P3-26: updateEvaluationCorrelationLinks ignores non-followed outcomes
Memory followed, another contradicted → no evaluation_correlation link between them. (unit, evaluate pkg)

### T-P3-27: EvalLinkUpdater error does not abort evaluate
SetEntryLinks returns error → evaluate proceeds normally. (unit, evaluate pkg)

---

## L4 → ARCH Traceability (P3)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-P3-1 | T-P3-1..T-P3-12 |
| ARCH-P3-2 | T-P3-13, T-P3-14, T-P3-15 |
| ARCH-P3-3 | T-P3-19, T-P3-20 |
| ARCH-P3-4 | T-P3-16, T-P3-17, T-P3-18 |
| ARCH-P3-5 | T-P3-21, T-P3-22, T-P3-23, T-P3-24 |
| ARCH-P3-6 | T-P3-25, T-P3-26, T-P3-27 |
| ARCH-P3-7 | (CLI wiring — integration verified by targ check) |

---

## P6f: Graduation Signal Lifecycle

### T-P6f-1: GraduationStore Append writes entry to JSONL

**Traces to:** ARCH-P6f-1 (REQ-P6f-2)

Given: A GraduationStore with in-memory file DI and a GraduationEntry with status="pending"
When: Append called with the entry and a path
Then: The JSONL file contains exactly one line with valid JSON matching the entry

---

### T-P6f-2: GraduationStore List reads all entries, skips malformed lines

**Traces to:** ARCH-P6f-1 (REQ-P6f-2)

Given: A JSONL file with two valid entries and one malformed line
When: List called
Then: Returns exactly two entries; no error

---

### T-P6f-3: GraduationStore List returns empty slice for missing file

**Traces to:** ARCH-P6f-1 (REQ-P6f-2)

Given: No graduation-queue.jsonl file exists
When: List called
Then: Returns empty slice with no error

---

### T-P6f-4: GraduationStore SetStatus updates matching entry

**Traces to:** ARCH-P6f-1 (REQ-P6f-2)

Given: A JSONL file with one pending entry (id="abc123def456")
When: SetStatus called with id="abc123def456", status="accepted", issueURL="https://..."
Then: Entry status is "accepted" and issue_url is set; resolved_at is non-empty

---

### T-P6f-5: GraduationStore SetStatus returns ErrGraduationNotFound for unknown ID

**Traces to:** ARCH-P6f-1 (REQ-P6f-2)

Given: A JSONL file with one entry (id="aaa")
When: SetStatus called with id="zzz"
Then: Returns ErrGraduationNotFound

---

### T-P6f-6: GraduationQueueEmitter EmitGraduation appends pending entry

**Traces to:** ARCH-P6f-2 (REQ-P6f-3)

Given: A GraduationQueueEmitter with in-memory store
When: EmitGraduation("mem/foo.toml", "CLAUDE.md", now) called
Then: Store contains one entry with status="pending", memory_path="mem/foo.toml", recommendation="CLAUDE.md"

---

### T-P6f-7: GraduationQueueEmitter ID is deterministic from memory_path

**Traces to:** ARCH-P6f-2 (REQ-P6f-3)

Given: Two calls to EmitGraduation with the same memory_path
When: Both entries written
Then: Both entries have the same ID

---

### T-P6f-8: graduate accept calls IssueCreator and marks entry accepted

**Traces to:** ARCH-P6f-3 (REQ-P6f-4)

Given: A GraduationStore with one pending entry; a mock IssueCreator returning "https://github.com/x/y/issues/1"
When: runGraduateAccept called with that entry's ID
Then: IssueCreator.Create called once; entry status becomes "accepted"; issue_url set; URL printed to stdout

---

### T-P6f-9: graduate dismiss marks entry dismissed

**Traces to:** ARCH-P6f-3 (REQ-P6f-5)

Given: A GraduationStore with one pending entry
When: runGraduateDismiss called with that entry's ID
Then: Entry status becomes "dismissed"; resolved_at non-empty; confirmation printed to stdout

---

### T-P6f-10: graduate list shows pending entries and quality metric

**Traces to:** ARCH-P6f-3 (REQ-P6f-6)

Given: Store with 2 pending, 3 accepted, 1 dismissed entries
When: runGraduateList called
Then: Output lists 2 pending entries; quality metric shows "75.0% accepted (3 accepted, 1 dismissed)"

---

### T-P6f-11: graduate list shows "n/a" quality metric when no resolved entries

**Traces to:** ARCH-P6f-3 (REQ-P6f-6, DES-P6f-1)

Given: Store with 2 pending, 0 accepted, 0 dismissed
When: runGraduateList called
Then: Quality line shows "n/a"

---

### T-P6f-12: graduate-surface outputs JSON with pending entries and instructions

**Traces to:** ARCH-P6f-3 (REQ-P6f-7, DES-P6f-2)

Given: Store with 1 pending entry
When: runGraduateSurface called with --format json
Then: JSON output has "summary" and "context"; context contains the entry ID and graduation accept/dismiss command instructions

---

### T-P6f-13: graduate-surface produces no output when queue is empty

**Traces to:** ARCH-P6f-3 (REQ-P6f-7)

Given: No graduation-queue.jsonl or all entries resolved
When: runGraduateSurface called
Then: No output written; exit 0

---

### T-P6f-14: graduate accept with unknown ID returns error

**Traces to:** ARCH-P6f-3 (REQ-P6f-4)

Given: Store with one entry id="abc"
When: runGraduateAccept called with id="zzz"
Then: Returns error; IssueCreator not called

---

### T-P6f-15: ApplyEscalationProposal passes ClassifyContent result as recommendation

**Traces to:** ARCH-P6e-1 (REQ-P6f-8, REQ-P6e-3, REQ-P6e-4)

Given: Memory with content containing procedural keywords ("run", "execute")
When: ApplyEscalationProposal called with ProposedLevel="graduated"
Then: EmitGraduation called with recommendation="skill" (ClassifyContent result, not hardcoded)

---

## L4 → ARCH Traceability (P6f)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-P6f-1 | T-P6f-1, T-P6f-2, T-P6f-3, T-P6f-4, T-P6f-5 |
| ARCH-P6f-2 | T-P6f-6, T-P6f-7 |
| ARCH-P6f-3 | T-P6f-8, T-P6f-9, T-P6f-10, T-P6f-11, T-P6f-12, T-P6f-13, T-P6f-14 |
| ARCH-P6e-1 | T-P6f-15 (REQ-P6f-8 traceability) |
