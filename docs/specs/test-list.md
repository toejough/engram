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

### T-281: Rules classified as always-loaded

**Traces to:** ARCH-70

Given: Registry entry with source_type "rule", 5 evaluations (3 followed, 2 ignored)
When: Classify is called
Then: Returns Working (binary classification — no Hidden Gem/Noise possible)

---

### T-282: Skills classified as always-loaded

**Traces to:** ARCH-70

Given: Registry entry with source_type "skill", 5 evaluations (1 followed, 4 contradicted)
When: Classify is called
Then: Returns Leech (binary classification — low effectiveness, always-loaded)

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
| ARCH-70 | T-281, T-282 |
| ARCH-71 | T-283, T-284 |

All ARCH items have test coverage.
