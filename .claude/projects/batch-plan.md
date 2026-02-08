# Batch Plan: Open Issues

## ISSUE-158: Create plan-producer and evaluation-producer skills
**Scope:** Create 2 new SKILL.md files + Go tests
**Files:**
- Create `~/.claude/skills/plan-producer/SKILL.md`
- Create `~/.claude/skills/evaluation-producer/SKILL.md`
- Add tests in `internal/skills/` for both

**plan-producer:**
- Frontmatter: `name: plan-producer, model: opus, user-invocable: true, role: producer, phase: plan`
- GATHER: Read issue description, query memory for past project patterns
- SYNTHESIZE: Structure problem → approach → task breakdown
- PRODUCE: Write plan.md with clear sections (Problem, Approach, Tasks, Risks)
- Contract: outputs plan.md, traces to issue

**evaluation-producer:**
- Frontmatter: `name: evaluation-producer, model: sonnet, user-invocable: true, role: producer, phase: evaluation`
- workflows.toml already defines: `artifact = "evaluation.md"`, `fallback_model = "sonnet"`
- GATHER: Read all project artifacts (requirements, design, arch, tasks, retro outputs)
- SYNTHESIZE: Assess outcomes vs. goals, extract learnings, identify process improvements
- PRODUCE: Write evaluation.md with retrospective findings + project summary (combined per ISSUE-148)
- Includes interview gate (evaluation_interview state follows evaluation_produce)
- Contract: outputs evaluation.md, traces to all upstream artifacts

**Tests (TDD red first):**
- Verify both SKILL.md files exist and have required frontmatter
- Verify contract sections exist
- Verify GATHER/SYNTHESIZE/PRODUCE sections
- Verify traces_to fields reference correct upstream artifacts

---

## ISSUE-159: Add plan mode to plan-producer
**Scope:** SKILL.md update + possible Go code change
**Depends on:** ISSUE-158

**Plan:**
1. Add `mode: plan` guidance to plan-producer SKILL.md — instruct the skill to use EnterPlanMode
2. Check if TaskParams needs a `mode` field (currently has subagent_type, name, model, team_name, prompt)
3. If mode field needed: add to TaskParams struct, update buildSpawnResult, update tests
4. If not needed: the plan-producer skill itself calls EnterPlanMode — no Go changes required
5. Plan-producer writes plan.md AND enters plan mode for user review before completion

**Likely outcome:** No Go changes needed — the skill calls EnterPlanMode itself. Just SKILL.md wording.

---

## ISSUE-162: Make interview producers plan-aware
**Scope:** Update 3 SKILL.md files
**Depends on:** ISSUE-158

**Plan:**
Add a "Plan Check" step as GATHER step 0 in each producer:
```
0. Check for approved plan:
   - Look for .claude/projects/<issue>/plan.md
   - If found: read plan, extract relevant section (requirements/design/architecture)
   - Produce artifact from plan content, only interview for gaps
   - If not found: proceed with full interview (current behavior)
```

Files to modify:
- `~/.claude/skills/pm-interview-producer/SKILL.md` — extract requirements from plan
- `~/.claude/skills/design-interview-producer/SKILL.md` — extract design decisions from plan
- `~/.claude/skills/arch-interview-producer/SKILL.md` — extract architecture decisions from plan

No Go code changes — this is pure SKILL.md documentation.

---

## ISSUE-161: Clarify model precedence
**Scope:** Rename field in TOML + Go code + tests

**Plan:**
1. Rename `fallback_model` → `default_model` in `internal/workflow/workflows.toml` (21 occurrences)
2. Update TOML parsing in `internal/workflow/config.go` — change struct field name
3. Update `internal/step/registry.go` — update references to the field
4. Update tests in `internal/step/registry_test.go` and `internal/workflow/`
5. Add comment to resolveModel: "Precedence: SKILL.md frontmatter > TOML default_model"
6. Run full test suite

---

## ISSUE-163: Full skill audit
**Scope:** Update ~18 SKILL.md files with workflow context
**Depends on:** ISSUE-158, 159, 162, 161

**Plan:**
1. For each skill referenced in workflows.toml, add a "## Workflow Context" section:
   - Which phases invoke this skill
   - What artifacts exist before it runs (upstream)
   - What comes after it (downstream: QA, crosscut, join)
2. Verify model declarations match workflow intent
3. Remove or document dormant skills (12+ unused QA skills)
4. Verify every skill in workflows.toml has a SKILL.md

This is the largest task — systematic but repetitive.

---

## ISSUE-167: Task selection before TDD spawn
**Scope:** Go code change in step package

**Plan:**
1. item_select state currently falls through to default (transition to item_fork)
2. Add handler for StateTypeSelect (or register item_select) that:
   - Reads tasks.md via task.Parallel()
   - Picks first unblocked task
   - Sets current_task via state.Set()
   - Returns transition to item_fork
3. Add tests: verify current_task is set after item_select processing
4. Verify current_task propagates to TDD producer prompt (already works per existing tests)

---

## ISSUE-166: Parallel item spawning in step loop
**Scope:** SKILL.md update (step loop) + possible Go changes for fork handling
**Depends on:** ISSUE-167

**Plan:**
1. Update step loop in SKILL.md to check `result.tasks` array length
2. When tasks.length > 1: spawn N teammates (one per task) with worktrees
3. When tasks.length == 1: sequential execution (current behavior)
4. Add fork handler in step.Next(): when StateTypeFork with multiple targets, return parallel spawn action
5. Add join handler: verify all branches complete before advancing
6. Wire worktree create/merge/cleanup into the loop

---

## ISSUE-156: Task owner/status on spawn/complete
**Scope:** SKILL.md update only

**Plan:**
Add to orchestrator control loop:
- On spawn: `TaskUpdate(taskId, status: "in_progress", owner: "<teammate-name>")`
- On step complete (done): `TaskUpdate(taskId, status: "completed")`
- On step complete (failed): clear owner, keep in_progress
- Mapping: use phase name to find matching task entry

---

## ISSUE-157: Top-level orchestration task
**Scope:** SKILL.md update only

**Plan:**
Add to Startup section:
- After TeamCreate, create top-level task: `TaskCreate(subject: "ISSUE-NNN: <title>", activeForm: "Orchestrating ISSUE-NNN")`
- Always prefix phase tasks with issue ID
- Mark top-level task completed at end-of-command

---

## ISSUE-169: Fix dangling trace references
**Scope:** docs/ file edits

**Plan:**
1. Grep for REQ-018 through REQ-023 references
2. Either define them in requirements.md or remove dangling references
3. Run projctl trace validate to verify clean
