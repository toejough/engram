# Memory Agent Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace engram's hook-based memory surfacing/evaluation/correction with a file-comms-based memory agent skill, while retaining the binary for recall and show.

**Architecture:** Two skills (file-comms update, memory-agent) replace hooks + Go surfacing pipeline. The Go binary is stripped to recall + show only. Memory TOML files get three new fields. Hooks are removed from the plugin.

**Tech Stack:** Claude Code skills (Markdown), Go (binary cleanup), TOML (memory files), shell (locking/fswatch)

---

### Task 1: Add new fields to MemoryRecord

**Files:**
- Modify: `internal/memory/record.go`
- Modify: `internal/memory/record_test.go` (if exists, otherwise the tests that exercise MemoryRecord)

- [ ] **Step 1: Write failing test for new fields**

Add a test that creates a MemoryRecord with the three new fields and verifies they serialize/deserialize correctly:

```go
func TestMemoryRecord_NewFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := memory.MemoryRecord{
		Situation:         "test situation",
		SchemaVersion:     1,
		MissedCount:       3,
		InitialConfidence: 0.7,
	}

	g.Expect(record.SchemaVersion).To(Equal(1))
	g.Expect(record.MissedCount).To(Equal(3))
	g.Expect(record.InitialConfidence).To(Equal(0.7))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: compilation error — SchemaVersion, MissedCount, InitialConfidence not defined on MemoryRecord

- [ ] **Step 3: Add new fields to MemoryRecord struct**

In `internal/memory/record.go`, add to the MemoryRecord struct:

```go
SchemaVersion     int     `toml:"schema_version,omitempty"`
MissedCount       int     `toml:"missed_count"`
InitialConfidence float64 `toml:"initial_confidence,omitempty"`
```

Place `SchemaVersion` before the SBIA fields. Place `MissedCount` after `IrrelevantCount`. Place `InitialConfidence` after `MissedCount`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: all tests pass including the new one

- [ ] **Step 5: Write test for opportunistic pending_evaluations cleanup**

Add a test that verifies writing a MemoryRecord without PendingEvaluations produces clean TOML (no pending_evaluations key). This is already the default behavior since the field has `omitempty`, but verify it explicitly. Also test that reading a file WITH pending_evaluations populates the struct, and re-writing it WITHOUT setting the field drops it.

- [ ] **Step 6: Run test to verify behavior**

Run: `targ test`
Expected: pass — omitempty on PendingEvaluations already handles this

- [ ] **Step 7: Commit**

```
feat(memory): add schema_version, missed_count, initial_confidence fields

New fields for the memory agent skill:
- schema_version: memory file schema version (default 1)
- missed_count: times memory was relevant but not surfaced
- initial_confidence: confidence at creation (1.0/0.7/0.2)
```

---

### Task 2: Strip Go binary to recall + show

**Files:**
- Modify: `internal/cli/cli.go` — remove case branches for removed commands
- Modify: `internal/cli/targets.go` — remove arg structs, flag builders, and targ targets for removed commands
- Delete: `internal/correct/` — entire package
- Delete: `internal/surface/` — entire package
- Delete: `internal/evaluate/` — entire package
- Delete: `internal/maintain/` — entire package
- Delete: `internal/hooks/` — entire package (unused)
- Delete: `internal/anthropic/` — only used by removed commands (verify recall doesn't need it)
- Delete: `internal/bm25/` — only used by surface/correct
- Delete: `internal/tokenresolver/` — only used by surface
- Delete: `internal/track/` — only used by surface
- Modify: `internal/cli/cli.go` — remove unused imports

**Important:** Before deleting any package, verify it is NOT imported by recall or show. Use `grep -r "package-name" internal/recall/ internal/cli/show.go` to check.

- [ ] **Step 1: Identify which packages recall and show depend on**

Run grep to find all imports used by recall and show:
```bash
grep -r '"engram/internal/' internal/recall/ internal/cli/cli.go | grep -E 'recall|show' | sort -u
```

This tells you what to keep. Everything else in internal/ that's only imported by removed commands can go.

- [ ] **Step 2: Remove case branches from cli.go**

In `internal/cli/cli.go`, remove the `case` branches for: `correct`, `surface`, `evaluate`, `maintain`, `apply-proposal`, `reject-proposal`, `refine`, `migrate-scores`, `migrate-slugs`, `migrate-sbia`. Keep only `recall` and `show`.

- [ ] **Step 3: Remove unused imports from cli.go**

Remove imports that were only used by removed commands. Keep: `memory`, `recall`, `policy` (if used by show), `context` (the session context package, if used by recall). Remove: `correct`, `surface`, `maintain`, `anthropic`, `tokenresolver`, `track`, `tomlwriter` (if not used by show).

- [ ] **Step 4: Remove arg structs and targets from targets.go**

In `internal/cli/targets.go`:
- Remove all Args structs except `RecallArgs` and `ShowArgs`
- Remove all Flags functions except `RecallFlags` and `ShowFlags`
- In `BuildTargets`, keep only the `recall` and `show` targ entries
- Keep `BuildFlags`, `AddBoolFlag`, `DataDirFromHome`, `ProjectSlugFromPath`, `RunSafe`, `Targets` — these are shared utilities

- [ ] **Step 5: Delete removed internal packages**

Delete entire directories (verify each is not imported by recall/show first):
```bash
rm -rf internal/correct/
rm -rf internal/surface/
rm -rf internal/evaluate/
rm -rf internal/maintain/
rm -rf internal/hooks/
rm -rf internal/bm25/
rm -rf internal/tokenresolver/
rm -rf internal/track/
```

Check if `internal/anthropic/` is used by recall (it likely is, for the Haiku summarizer). If so, keep it. If not, delete it too.

- [ ] **Step 6: Run targ check-full**

Run: `targ check-full`
Expected: builds cleanly, all remaining tests pass, no lint errors. Fix any compilation errors from missing imports or references to deleted code.

- [ ] **Step 7: Commit**

```
refactor(cli): strip binary to recall + show only

Remove surface, evaluate, correct, maintain, refine, and all
migrate-* commands. Delete supporting packages that are no longer
imported. The memory agent skill replaces this functionality.
```

---

### Task 3: Remove hooks

**Files:**
- Delete: `hooks/stop.sh`
- Delete: `hooks/stop-surface.sh`
- Delete: `hooks/session-start.sh`
- Delete: `hooks/user-prompt-submit.sh`
- Modify: `.claude-plugin/plugin.json` — remove hooks configuration if present (currently it's minimal, but check if hooks are auto-discovered)

- [ ] **Step 1: Check if hooks are registered in plugin.json or auto-discovered**

Read `.claude-plugin/plugin.json`. If it references hooks, remove those references. If hooks are auto-discovered by directory convention, deleting the files is sufficient.

- [ ] **Step 2: Delete hook files**

```bash
rm hooks/stop.sh hooks/stop-surface.sh hooks/session-start.sh hooks/user-prompt-submit.sh
```

If the `hooks/` directory is now empty, remove it too:
```bash
rmdir hooks/
```

- [ ] **Step 3: Verify the plugin still loads**

Install the plugin locally and verify Claude Code doesn't error on startup. The plugin should still provide the recall and memory-triage skills.

- [ ] **Step 4: Commit**

```
refactor(hooks): remove all engram hooks

Memory surfacing is now handled by the memory agent skill
via file-comms, not by hooks calling the engram binary.
```

---

### Task 4: Remove memory-triage skill

**Files:**
- Delete: `skills/memory-triage/SKILL.md`
- Delete: `skills/memory-triage/` directory

- [ ] **Step 1: Delete the skill**

```bash
rm -rf skills/memory-triage/
```

- [ ] **Step 2: Verify recall skill is unaffected**

Read `skills/recall/SKILL.md` and confirm it only references `engram recall`, which is retained.

- [ ] **Step 3: Commit**

```
refactor(skills): remove memory-triage skill

References engram maintain/apply-proposal/reject-proposal which
are removed. Maintenance functionality deferred to #471.
```

---

### Task 5: Update recall skill path

**Files:**
- Modify: `skills/recall/SKILL.md`

- [ ] **Step 1: Read the current recall skill**

Check the path it references (`~/.claude/engram/bin/engram recall`). Verify this is still the correct path after the binary strip. If the binary path has changed, update it.

- [ ] **Step 2: Update if needed and commit**

```
chore(skills): verify recall skill references correct binary path
```

---

### Task 6: Finalize file-comms skill

**Files:**
- Modify: `~/.claude/skills/file-comms/SKILL.md`

The file-comms skill has already been updated during brainstorming with:
- Agent roles (active/reactive)
- User input parroting
- Argument protocol (3 inputs + escalation, early concession)
- Background task fswatch loop pattern
- Heartbeat
- Chat file management

- [ ] **Step 1: Review the skill against the spec**

Read `~/.claude/skills/file-comms/SKILL.md` and `docs/superpowers/specs/2026-04-02-memory-agent-skill-design.md` Part 1. Verify every spec requirement is covered in the skill. Check for any inconsistencies introduced during the iterative editing.

- [ ] **Step 2: Fix any gaps found**

- [ ] **Step 3: Commit (if changes made)**

```
docs(skills): finalize file-comms skill against spec
```

---

### Task 7: Finalize memory-agent skill

**Files:**
- Modify: `~/.claude/skills/memory-agent/SKILL.md`

The memory-agent skill has already been written during brainstorming. It needs a final review pass against the spec to catch any drift from the iterative changes.

- [ ] **Step 1: Review the skill against the spec**

Read `~/.claude/skills/memory-agent/SKILL.md` and `docs/superpowers/specs/2026-04-02-memory-agent-skill-design.md` Part 2. Check:
- Situations-only loading described correctly
- Background task fswatch loop pattern matches spec
- Surfacing flow matches (situation match → subagent reads full file → behavior judgment)
- Learning flow matches (confidence tiers, incomplete SBIA handling, rate limiting)
- Locking matches (per-file, stale recovery, mkdir 300s timeout, no multi-file)
- Subagent management matches (monotonic IDs, thread exclusivity, routing)
- New fields (schema_version, missed_count, initial_confidence) documented

- [ ] **Step 2: Fix any gaps found**

Key items to verify are present:
- Incomplete SBIA → ask active agent to prompt user
- Rate limiting (>5 in 10 min → warning)
- Per-intent dedup on missed_count
- Effectiveness formula in performance tracking
- Monotonically increasing subagent IDs

- [ ] **Step 3: Commit (if changes made)**

```
docs(skills): finalize memory-agent skill against spec
```

---

### Task 8: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md` (repo root)

- [ ] **Step 1: Read current CLAUDE.md**

Check for references to hooks, the old surfacing pipeline, or commands that no longer exist.

- [ ] **Step 2: Update to reflect new architecture**

Remove or update any references to:
- Hook-based memory surfacing
- `engram surface`, `engram evaluate`, `engram correct`, `engram maintain`
- The plugin form factor (if it mentions hooks as the primary mechanism)

Add a note that memory surfacing is now handled by the memory-agent skill via file-comms.

- [ ] **Step 3: Commit**

```
docs: update CLAUDE.md for skill-based memory architecture
```

---

### Task 9: Clean up worktree artifacts

**Files:**
- Delete: `.worktrees/memory-consolidation/` (if stale)
- Delete: `.claude/worktrees/agent-*/` (if stale)
- Delete: `chat.toml` (dogfood artifact)

- [ ] **Step 1: Check which worktrees are active**

```bash
git worktree list
```

Only delete worktrees that are no longer active. If any are active, leave them.

- [ ] **Step 2: Clean up stale worktrees and artifacts**

```bash
git worktree prune
rm -f chat.toml
```

- [ ] **Step 3: Add chat.toml to .gitignore if not already present**

Check `.gitignore` for `chat.toml`. Add it if missing — it's a coordination artifact, not source.

- [ ] **Step 4: Commit**

```
chore: clean up stale worktrees and dogfood artifacts
```

---

### Task 10: Final verification

- [ ] **Step 1: Run targ check-full**

Run: `targ check-full`
Expected: all tests pass, all linting clean, coverage acceptable.

- [ ] **Step 2: Verify engram recall still works**

```bash
targ build
engram recall --project-slug test --query "test query"
engram show --name some-memory-slug
```

Both commands should work. If recall needs an API token, verify the token env var path is correct.

- [ ] **Step 3: Verify plugin loads in Claude Code**

Start a new Claude Code session in the engram directory. Verify:
- No hook errors on startup
- The recall skill is available
- The memory-triage skill is NOT available
- The memory-agent skill is available (from ~/.claude/skills/)

- [ ] **Step 4: Dogfood test — run memory agent**

Open a second terminal, start Claude Code, tell it to be the memory agent. Verify:
- It loads the skill
- It reads memory situations
- It enters the fswatch loop
- It reacts to chat.toml changes

- [ ] **Step 5: Commit any final fixes**
