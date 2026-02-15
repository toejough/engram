# Extraction Pipeline Validation

## Goal

Validate the proposed pipeline (strip noise -> Haiku ID events -> Sonnet extract principles) against real sessions where we've already agreed on what the "right" lessons are.

## Pipeline Under Test

1. **Strip noise** — mechanically remove tool results, file contents, system-reminders
2. **Haiku identifies events** — per-chunk, find learning-relevant moments
3. **Sonnet extracts principles** — single call, all events -> reusable principles
4. **Deduplicate** — embedding similarity against existing memories

## Validation Sessions

### Session 1: Keychain Auth Debugging

- **File**: `~/.claude/projects/-Users-joe-repos-personal-projctl/87b657d2-5dda-43bd-8c6b-7cfa306cb79b.jsonl`
- **Project**: projctl
- **Size**: ~368KB
- **Summary**: Debugging keychain auth failure in `projctl memory optimize --review`

**Approved Lessons:**

1. Always handle JSON fields that may be either string or number — macOS keychain stores `expiresAt` as a JSON number, but Go struct declared it as `string`. `json.Unmarshal` silently rejects type mismatches. Use `json.Number` or `any` for fields from external sources.
2. Never chase shell variable tangents when debugging Go code — reproduce the bug in the same language/runtime as the code under test.
3. Always test with real-world data formats, not just test-friendly formats — all existing tests passed `expiresAt` as a string, which happened to work. The real keychain returned a number.

> **Removed:** "Use `user.Current()` as fallback when `$USER` is empty" — this fix was committed _before_ this session started (commit `8d084dd`). The session investigated the `$USER` issue but the actual `user.Current()` edit is not in the transcript.

**Validation artifacts:**
- Stripped file: `docs/plans/validation-s1-stripped.txt` (8KB, 130 lines — from 368KB raw)
- Haiku events: `docs/plans/validation-s1-haiku-events.json`
- Sonnet principles: `docs/plans/validation-s1-sonnet-principles.json` (4 principles)
- Match assessment: **3/3 = 100% recall**, 4/4 precision

---

### Session 2: Team Operations (Dep Group Chaining + Bug Batch)

- **File**: `~/.claude/projects/-Users-joe-repos-personal-targ/beb57e20-4438-435e-a349-c6f93fa51b79.jsonl`
- **Project**: targ
- **Size**: ~2104KB
- **Summary**: User instructed Claude to run teams with haiku watchdog, QA agent, and sonnet workers. Two projects executed: dep group chaining (6 tasks) and 5-bug batch fix.

**Approved Lessons:**

1. When running teams, use a haiku watchdog to coordinate task ordering — not the team lead. The team lead stays out of task dispatch.
2. Split workers by file conflict zones, not by task count — assign workers to avoid touching the same files simultaneously.
3. Always use a QA agent to review each task before moving to the next — QA caught issues on Task 1 that needed fixing before dependent tasks could proceed.
4. When planning team task execution, express dependencies explicitly rather than serializing everything — independent tasks parallelized, dependent tasks sequenced.
5. When two workers race to claim the same task, check actual git state rather than trusting status updates.
6. For binary mode CLI tools, hide framework internals from help output — labels, examples, and commands should use the binary name, not the framework name.

**Validation artifacts:**
- Stripped file: `docs/plans/validation-s2-stripped.txt` (103KB, 1757 lines — from 2104KB raw)
- Haiku events: `docs/plans/validation-s2-haiku-events.json` (19 events, 5 chunks @ 25KB)
- Sonnet principles: `docs/plans/validation-s2-sonnet-principles.json` (9 principles)
- Match assessment: **4.5/6 = 75% recall**, 9/9 precision. L5 (worker race) missed by Haiku. L3 (QA gate) partially merged into P4.

---

### Session 3: Amend-After-Push Corrections

- **File**: `~/.claude/projects/-Users-joe-repos-personal-targ/0f68847f-5b8d-4b51-940c-066e047731f0.jsonl`
- **Project**: targ
- **Size**: ~161MB (very long session, 19K lines)
- **Summary**: Long session covering imptest upgrade, build targets, linting, and repeated amend-after-push violations. Claude violated the "never amend pushed commits" rule multiple times and user escalated corrections.

**Approved Lessons:**

1. When corrected, write it down immediately — don't acknowledge, don't promise, just edit CLAUDE.md. Claude needed two prompts before actually writing down the amend rule.
2. Before any `git commit --amend`, run `git status` and check for "Your branch is ahead of" — if not ahead, the commit is already pushed and amending will cause conflicts.
3. Keep commits focused on single concerns — a lint fix commit shouldn't include a behavior change.
4. Don't overcomplicate isolated build systems — copy go.mod and let `go mod tidy` resolve imports rather than complex replace directives.
5. Structural linting issues (missing returns, unused vars) take priority over style issues (naming, line length).

**Validation artifacts:**
- Stripped file: `docs/plans/validation-s3-stripped.txt` (1555KB, 36470 lines — from 161MB raw)
  - **Note:** Still too large for single LLM call. Validates need for chunking.
- Haiku events: `docs/plans/validation-s3-haiku-events.json` (286 events, 64 chunks @ 25KB, 5 failures = 92% success)
- Sonnet principles: `docs/plans/validation-s3-sonnet-principles.json` (13 principles)
- Match assessment: **4/5 = 80% recall**, 13/13 precision. L3 (focused commits) missed.

---

## Observations from Stripping

| Session | Raw | Stripped | Reduction | Single-call feasible? |
|---------|-----|---------|-----------|----------------------|
| S1 | 368KB | 8KB | 98% | Yes (~4K tokens) |
| S2 | 2104KB | 103KB | 95% | Borderline (~50K tokens) |
| S3 | 161MB | 1555KB | 99% | No (~750K tokens) — needs chunking |

**Stripping logic applied:**
- Remove `<system-reminder>` tags and content
- Keep `<teammate-message>` content (extracted with sender ID)
- Keep raw user messages (only system-reminders stripped)
- Collapse skill content to `(skill loaded)` (detect skill headers)
- Omit successful tool results entirely (assistant narrates outcomes)
- Keep error tool results (up to 300 chars)
- Edit operations: show diff (what changed between old and new) with ~60 chars context
- Truncate inline scripts (heredocs) to first line + char count
- Keep assistant reasoning text, tool invocations (name + key args)

**Issues found and fixed:**
- ~~Skill content still leaks if loaded outside `<system-reminder>` tags~~ — fixed with header detection
- ~~Teammate messages stripped, losing worker race condition signal (S2 L5)~~ — fixed by keeping teammate messages
- ~~Edit truncation at 100 chars loses actual changes~~ — fixed with diff-based Edit content
- ~~L2 "user.Current() fallback" attributed to S1 but not in session~~ — removed from S1 approved lessons

---

## Scoring

For each session, compare Sonnet's extracted principles against the approved lessons:

| Metric | Definition |
|--------|-----------|
| **Recall** | What fraction of approved lessons did Sonnet find? |
| **Precision** | What fraction of Sonnet's output was actually useful? (no junk) |
| **Quality** | Are the principles actionable and specific, not generic platitudes? |

**Pass criteria**: Recall >= 80%, Precision >= 70%, Quality subjectively good.

## Results

| Session | Approved | Covered | Recall | Principles | Precision | Quality |
|---------|----------|---------|--------|------------|-----------|---------|
| S1 | 3 | 3 | 100% | 4 | 4/4 | Good |
| S2 | 6 | 4.5 | 75% | 9 | 9/9 | Good |
| S3 | 5 | 4 | 80% | 13 | 13/13 | Good |
| **Total** | **14** | **11.5** | **82%** | **26** | **26/26** | **Good** |

**Verdict: Approved.** Overall recall 82% meets threshold. Precision is excellent — every principle Sonnet produces is useful and actionable. Known weakness: brief process/coordination events (2-3 line observations) are occasionally missed by Haiku or merged by Sonnet.

### Validated Pipeline Parameters

- **Chunk size**: 25KB (balances Haiku attention vs call count)
- **Parallelism**: 4 workers (ThreadPoolExecutor)
- **Haiku success rate**: 92-100% per chunk (prefill technique required)
- **Sonnet prompt**: Must explicitly weight process/coordination lessons equally with technical lessons
- **Haiku event types**: error-and-fix, user-correction, strategy-change, root-cause-discovery, environmental-issue, pattern-observed, coordination-issue
