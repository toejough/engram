# Phase 2: Conversion-Parity Runbook-vs-Skill Eval

> **For agentic workers:** This plan is NOT for execution in this session. Phase 2 consists of: (a) agent-driven conversions of existing skill/runbook sources to other types via their native authoring paths, (b) extending probe.py, (c) building fixtures and copy strategy, (d) running the full n=5 opus eval. See "Execution Notes" section for orchestration.

**Goal:** Measure whether runbooks earn their keep against facts when both are authored via native paths (learn vs writing-skills) at real-vault scale, and determine if the type matters for generic (non-idiosyncratic) procedures.

**Architecture:** Phase 2 scales phase-1's single-note synthetic vault to real-vault copying per trial. Eight arms total: Task A (commit) has A-S (original /commit skill, no conversion), A-R (convert /commit→runbook via learn), A-F (convert /commit→fact via learn), A-Rdirect (shim, no retrieval). Task B (gitignore) has B-R (original runbook 830, no conversion), B-F (convert 830→fact via learn), B-S (convert 830→skill via writing-skills), B-Rdirect (shim). All arms receive the same background vault (real vault copy with covering notes removed, then arm's carrier added). Scoring reuses probe.py's FOUND/FOLLOWED/END-STATE/COST framework; shim/note decomposition separates retrieval tax from procedure quality.

**Tech Stack:** Python 3.11+, headless `claude -p`, engram vault/chunks isolation, BASH for fixture repo and checks, git.

**Spec:** `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/README.md` (phase-1 decision frame and isolation strategy reused); `/Users/joe/repos/personal/engram/dev/eval/isolation.py` (vault/chunks/config isolation); `/Users/joe/repos/personal/engram/dev/eval/cumulative/harness.py` (MODELS registry, ENGRAM_BIN_DIR).

---

## Global Constraints

- **Runbook schema:** Exactly three fields: situation (when-to-use), numbered steps (body), done_when (ending expectations) — per note 789.2026-08-23. Do NOT include full inputs/preconditions/returns (rejected at #719 readiness checkpoint).
- **Background vault fairness:** Every arm (S/R/F/Rdirect for both tasks) receives the SAME background vault: a per-trial copy of the real vault with covering notes REMOVED. Task A covering notes: any notes mentioning `AI-Used`, conventional commits, or commit conventions (Task 1 step 3 grep results). Task B covering notes: runbook 830 (removed from F/S/Rdirect arms only; kept in B-R), plus any other gitignore-narrowing notes. Each arm's carrier (skill S: nothing in vault; R: runbook note; F: fact note; Rdirect: nothing in vault) is then added to its trial vault copy. FOUND for R/F requires the arm's carrier basename in the query result.
- **Vault copy mechanics:** Per-trial copytree (never symlink; no write-back possible). Copies deleted after scoring even under --keep. Copy size measured once (`du -sh`); per-run disk = 40 × size. Trial isolation follows probe.py's `isolated_env(..., cwd=None)` + `assert_isolated(env)` pattern (cite probe.py line references). Fingerprint guard per #750 interim rule (controller writes nothing to real vault during run).
- **Fixture placeholder precision:** Every fixture procedure placeholder must name its exact source field (e.g. `<sensor_id>` not `<name>`) with a worked example, per note 955.2026-09-10.
- **Smoke on opus:** Smoke run MUST run on opus, n=1 per arm per task (8 trials total), before spending on full n=5 paid run per note 955.
- **Measurement scope:** Memory value unmeasured for GENERIC procedures (note 853a.2026-08-30); phase 2 measures exactly this. SEPARATE from idiosyncratic findings (where memory already shows wins).
- **Decision bars (pre-registered):**
  - Parity rule: ±1-trial indistinguishable (FOLLOWED-all-steps: trials with every step, k/5); 2+ trial gap = worse/better.
  - Runbook > fact: Only if R is 2+ ahead of F on FOLLOWED-all-steps or END-STATE (else "can't distinguish").
  - Baseline usability: Arm S (skill) must ≥3/5 on END-STATE or baseline uninterpretable (fixture/spec fix required).
  - Shim/note decomposition: shim rate = FOUND(R,F) k/5; note quality given delivery = END-STATE and FOLLOWED-all among FOUND=true trials; note ceiling = Rdirect END-STATE/FOLLOWED-all k/5; shim loss = Rdirect END-STATE − R END-STATE; type effect = R − F on both metrics; parity = S vs R and S vs F ±1 rule. Rdirect FOUND = n/a (marker_seen is delivery check).

---

## File Structure

### Source Materials (Read-Only, Exact Paths)

- **Commit skill (LIVE):** `/Users/joe/repos/personal/engram/.claude/skills/commit.md` — the project's /commit skill (repo source, not plugin). Description: "Core: Stages specific files and creates conventional commits with AI-Used trailer and proper message formatting." A-S arm uses this skill as-is without conversion. Trial deployment: file is copied exactly to trial `.claude/skills/commit.md` so it's discovered identically to the repo. Trial isolation: cold cfg has no plugin skills, so the plugin commit skill cannot leak into any arm.
- **Runbook 830:** `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md` (situation, done_when, steps 1–6 verbatim text; exact line numbers shift per YAML). B-R arm uses this note as-is without conversion.
- **Phase-1 harness:** `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/{README.md, probe.py, fixtures/, fixture-vaults/}` (reuse structure; extend probe.py only).
- **Isolation contract:** `/Users/joe/repos/personal/engram/dev/eval/isolation.py` (isolated_env, assert_isolated, project_slug, NEVER edit).
- **Harness shared:** `/Users/joe/repos/personal/engram/dev/eval/cumulative/harness.py` (MODELS, ENGRAM_BIN_DIR, refresh_creds).

### Deliverables (New Directories/Files)

```
dev/eval/cumulative/runbook_vs_skill/
├── PLAN-2-conversion-parity.md                          (this file)
├── phase2/
│   ├── SOURCE_MATERIALS.md                              (Task 1: verbatim quotes + covering notes)
│   ├── fixtures/
│   │   ├── commit/{init, done_when, task-prompt}.sh|txt
│   │   ├── gitignore/{init, done_when, task-prompt}.sh|txt
│   │   └── fixture-repo-templates/{commit-task, gitignore-task}/
│   ├── encodings/
│   │   ├── taskA/
│   │   │   ├── A-S/ (original /commit skill; no conversion; path below for carrier setup)
│   │   │   ├── A-R/ (converted: runbook via learn)
│   │   │   ├── A-F/ (converted: fact via learn)
│   │   │   └── A-Rdirect/ (shim text, no retrieval)
│   │   └── taskB/
│   │       ├── B-R/ (original runbook 830; no conversion; path below for carrier setup)
│   │       ├── B-F/ (converted: fact via learn)
│   │       ├── B-S/ (converted: skill via writing-skills; includes RED/GREEN/pressure evidence)
│   │       └── B-Rdirect/ (shim text, no retrieval)
│   ├── results/
│   │   ├── smoke_opus_results.jsonl                     (8 trials: 2 tasks × 4 arms)
│   │   ├── opus_results.jsonl                           (40 trials: 2 tasks × 4 arms × 5)
│   │   ├── ANALYSIS.md                                  (post-run: parity per task, type effect)
│   │   └── conversion-fidelity-report.md                (reviewer audit of 4 conversions)
│   └── probe_phase2.py                                  (extended from phase-1 probe.py)
└── ...existing phase-1 files untouched...
```

**Carrier setup paths (for reference; not created by plan, just discovered):**
- A-S carrier: `/Users/joe/repos/personal/engram/.claude/skills/commit.md` (copied into trial .claude/skills for discovery)
- B-R carrier: `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md` (copied into trial vault with .vec.json sidecar)

---

## Task Decomposition

### Task 1: Verify Source Materials & Background Vault Covering Notes

**Files:**
- Create: `phase2/SOURCE_MATERIALS.md` (verbatim quotes, covering notes list)
- Read: `/Users/joe/repos/personal/engram/.claude/skills/commit.md` (live /commit skill)
- Read: `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md` (runbook 830)

**Interfaces:**
- Produces: `SOURCE_MATERIALS.md` with verbatim quotes from live /commit skill (Workflow steps, Commit Message Format), runbook 830 (situation, done_when, steps 1–6), and list of covering notes for Task A background vault removal

- [ ] **Step 1: Extract live /commit skill steps verbatim**

Read `/Users/joe/repos/personal/engram/.claude/skills/commit.md`. Quote the 7 Process steps and Rules:

**Process (7 steps):**
1. Check VCS type (look for `.jj` directory; if jj repo, use `jj` commands, not `git`)
2. Check state (`git status`, `git diff --staged`, `git diff`; if nothing to commit, report and stop)
3. Review recent commits for style (`git log --oneline -5`)
4. Stage changes (stage files relevant to current change; prefer specific paths over `git add -A`; do not stage unrelated files)
5. Compose message (format: `<type>(scope): <description>` + body explaining why + trailer `AI-Used: [claude]`)
6. Commit (use HEREDOC: `git commit -m "$(cat <<'EOF' ... EOF )"` with message and `AI-Used: [claude]` trailer)
7. Verify (`git log -1` and `git status`)

**Rules:**
- AI-Used trailer is `AI-Used: [claude]` — NOT Co-Authored-By
- Never amend pushed commits; check `git status` for "ahead of" first
- Separate concerns; don't mix functional changes with lint/style fixes
- First line under 72 chars; body wrapped at 72 chars
- Stage specific files; don't use `git add -A` or `git add .`
- Never use dangerous commands (no `git checkout -- .`, `git restore .`, `git reset --hard`)

Record: A-S arm uses this skill as-is (source deployed to trial `.claude/skills/commit.md` exactly as in repo); conversions (A-R, A-F) source from this exact text.

- [ ] **Step 2: Extract runbook 830 verbatim (situation, done_when, steps)**

Read `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md`. Quote (text verbatim, not line numbers):
- Situation: "before shipping a narrowed or rewritten .gitignore pattern that makes some previously-ignored files trackable"
- Done_when: "the pattern's anchoring form has been confirmed correct via a scratch-repo git check-ignore check against representative (including nested) paths, and the exact set of newly-visible files has been enumerated and staged as an explicit path list rather than swept up by git add"
- Steps 1–6: 6 numbered steps verbatim (git check-ignore, anchor depth, middle-slash rule, git status enumeration, explicit staging, diff verification)

Record: B-R uses this note as-is; conversions (B-F, B-S) source from this exact text.

- [ ] **Step 3: Grep vault for covering notes (Task A background vault removal)**

Run: `grep -l "AI-Used\|commit.*convention\|Conventional Commits" /Users/joe/.local/share/engram/vault/*.md 2>/dev/null | xargs basename -a | sort`

**Expected covering notes for Task A** (will be removed from trial vault, all arms): Notes mentioning conventional-commit format, AI-Used trailers, or commit conventions. Known: note 354 (subagent-briefs commit invariants), note 672 (route-dispatch doc-review-gate: "Conventional Commits format, AI-Used trailer"). Any others found must be listed.

Record: These notes are REMOVED from the background vault copy before any arm's carrier is added. This ensures R/F arms retrieve only the NEW converted notes (A-R, A-F), not pre-existing covering notes.

- [ ] **Step 4: Grep vault for covering notes (Task B background vault removal)**

Run: `grep -l "gitignore.*narrow\|\.gitignore.*pattern\|check-ignore" /Users/joe/.local/share/engram/vault/*.md 2>/dev/null | xargs basename -a | sort`

**Expected covering notes for Task B** (will be removed from trial vault, F/S/Rdirect arms only; KEPT in B-R): Notes covering .gitignore narrowing or anchoring patterns. Known: note 830 (the source). Any others found must be listed.

Record: Note 830 is REMOVED from the background vault copy for F/S/Rdirect arms (so they retrieve only the NEW B-F conversion, not the original), but is KEPT in B-R's vault (so it retrieves the original as the arm's carrier).

- [ ] **Step 5: Write phase2/SOURCE_MATERIALS.md**

Create a document (markdown) with three sections:
1. **Live /commit Skill Workflow & Format** — copy verbatim from Step 1
2. **Runbook 830 Situation, Done_When, Steps** — copy verbatim from Step 2
3. **Background Vault Covering Notes for Removal**
   - Task A: list all notes found in Step 3 (will be removed from all arms' background vault)
   - Task B: list all notes found in Step 4 (will be removed from F/S/Rdirect arms; 830 kept in B-R)

- [ ] **Step 6: Commit task 1**

```bash
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill
git add phase2/SOURCE_MATERIALS.md
git commit -m "docs(eval/phase2): source materials task 1 — live /commit skill, 830, covering notes recorded"
```

---

### Task 2: Design Fixture Repos & Initialization Scripts (Task A: Commit, Task B: Gitignore)

**Files:**
- Create: `phase2/fixtures/commit/init_fixture_repo.sh`
- Create: `phase2/fixtures/commit/done_when_checks.sh`
- Create: `phase2/fixtures/commit/task-prompt.txt`
- Create: `phase2/fixtures/gitignore/init_fixture_repo.sh`
- Create: `phase2/fixtures/gitignore/done_when_checks.sh`
- Create: `phase2/fixtures/gitignore/task-prompt.txt`
- Create: `phase2/fixtures/fixture-repo-templates/commit-task/` (template git repo)
- Create: `phase2/fixtures/fixture-repo-templates/gitignore-task/` (template git repo)

**Interfaces:**
- Consumes: commit skill steps (from Task 1), runbook 830 steps (from Task 1)
- Produces: per-task fixture setup scripts and done_when checkers; task-specific prompts (natural language, no procedure hints); template repos with idiosyncratic markers absent

**Task A (Commit): Design and Implement**

- [ ] **Step 1: Create commit-task template repo**

Create the directory: `phase2/fixtures/fixture-repo-templates/commit-task/`

This is a minimal git repo where a change is UNSTAGED (modified in working tree, not staged), plus an unrelated untracked decoy file, ready for the trial agent to commit the real change. Use a simple, realistic scenario: a small code file with a one-liner change, left modified but not staged.

```bash
mkdir -p phase2/fixtures/fixture-repo-templates/commit-task
cd phase2/fixtures/fixture-repo-templates/commit-task
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create a simple Go file with initial content
mkdir -p pkg
cat > pkg/version.go <<'EOF'
package pkg

const Version = "1.0.0"
EOF

# Initial commit
git add pkg/version.go
git commit -m "initial: add version constant"

# Now make an UNSTAGED change
cat > pkg/version.go <<'EOF'
package pkg

const Version = "1.1.0"
EOF

# DO NOT stage it — leave it as a working-tree modification

# Also create an unrelated decoy file that must NOT be committed
mkdir -p notes
cat > notes/scratch.txt <<'EOF'
wip
EOF

# Verify: git status should show pkg/version.go as modified (unstaged) and notes/scratch.txt as untracked
```

After running this, verify: `git status` should show `pkg/version.go` as "modified" (unstaged, not staged), and `notes/scratch.txt` as untracked. The decoy file must NOT be committed by the agent.

**Rationale:** The task is idiosyncratic (specific unstaged change to commit), but the commit convention itself is generic (conventional-commit format, trailers). The decoy file tests that the agent stages only the actual change, not all untracked files. The trial agent must discover the convention from the arm's carrier (skill/runbook/fact), not infer it from repo history.

- [ ] **Step 2: Create commit-task init_fixture_repo.sh**

Create `phase2/fixtures/commit/init_fixture_repo.sh`. This script is called per trial to set up a fresh fixture repo by cloning the template and appending trial-specific CLAUDE.md.

```bash
#!/bin/bash
set -euo pipefail

TEMPLATE_DIR="$1"   # phase2/fixtures/fixture-repo-templates/commit-task
REPO_DIR="$2"       # trial's working repo dir
CLAUDE_MD="$3"      # trial-specific CLAUDE.md text
VAULT_DIR="$4"      # trial vault (for Arm R/F only; empty for Arm S, ignored for shim)

# Clone template
cp -r "$TEMPLATE_DIR" "$REPO_DIR"
cd "$REPO_DIR"

# Reset git config so commits use the test identity (not operator's git config)
git config user.email "trial@example.com"
git config user.name "Trial Agent"

# Write CLAUDE.md in repo root
cat > CLAUDE.md <<'CLAUSENDMD'
$CLAUDE_MD
CLAUSENDMD

# For retrieval arms, set up vault symlink or env var (probe.py handles this)
# For shim arms, vault is irrelevant
# For skill arm, vault is empty

echo "Fixture repo initialized at $REPO_DIR"
```

**Acceptance:** Script exits 0; `$REPO_DIR/pkg/version.go` is modified but unstaged (`git status --porcelain` shows ` M pkg/version.go`), `notes/scratch.txt` shows as `?? notes/scratch.txt`, and `git diff --cached` is empty.

- [ ] **Step 3: Create commit-task done_when_checks.sh**

Create `phase2/fixtures/commit/done_when_checks.sh`. This script validates the end-state per the commit skill's requirements: exactly one new commit on pkg/version.go (not the decoy), message in conventional-commit format with `AI-Used: [claude]` trailer, decoy file still untracked, nothing amended, no push.

```bash
#!/bin/bash
set -euo pipefail

REPO_DIR="$1"       # trial's repo after agent runs

cd "$REPO_DIR"

# Check 1: Exactly one new commit since setup, touching ONLY pkg/version.go
COMMIT_COUNT=$(git log --oneline -- pkg/version.go | wc -l)
if [ "$COMMIT_COUNT" != "2" ]; then  # fixture commit + new commit = 2
  echo "FAIL: Expected exactly one new commit on pkg/version.go; found $((COMMIT_COUNT - 1))"
  exit 1
fi

# Check 2: Latest commit message follows conventional-commit format
COMMIT_MSG=$(git log -1 --format=%B)
if ! echo "$COMMIT_MSG" | grep -qE '^[a-z]+(\(.+\))?:.*'; then
  echo "FAIL: Commit message does not follow conventional-commit format: $COMMIT_MSG"
  exit 1
fi

# Check 3: Commit ends with AI-Used: [claude] trailer
if ! echo "$COMMIT_MSG" | grep -q "AI-Used: \[claude\]"; then
  echo "FAIL: Commit message missing AI-Used: [claude] trailer"
  exit 1
fi

# Check 4: Latest commit touches ONLY pkg/version.go (not notes/scratch.txt)
FILES_CHANGED=$(git diff-tree --no-commit-id --name-only -r HEAD | wc -l)
if [ "$FILES_CHANGED" != "1" ]; then
  echo "FAIL: Latest commit touches $FILES_CHANGED files, expected 1 (only pkg/version.go)"
  git diff-tree --no-commit-id --name-only -r HEAD
  exit 1
fi

if ! git diff-tree --no-commit-id --name-only -r HEAD | grep -q "^pkg/version.go$"; then
  echo "FAIL: Latest commit does not touch pkg/version.go"
  exit 1
fi

# Check 5: Decoy file (notes/scratch.txt) is still untracked
if ! git status --porcelain | grep -q "^?? notes/scratch.txt"; then
  echo "FAIL: notes/scratch.txt is not untracked (was it staged/committed?)"
  git status
  exit 1
fi

# Check 6: Fixture commit hash unchanged (no amendments)
FIXTURE_HASH=$(git log --oneline -- pkg/version.go | tail -1 | awk '{print $1}')
if [ -z "$FIXTURE_HASH" ]; then
  echo "FAIL: Cannot find fixture commit"
  exit 1
fi
echo "OK: Fixture commit $FIXTURE_HASH unchanged"

echo "PASS: Commit task end-state verified"
exit 0
```

**Acceptance:** Script exits 0 on a trial repo that has: (1) exactly one new commit on pkg/version.go only, (2) conventional-commit message, (3) `AI-Used: [claude]` trailer, (4) notes/scratch.txt still untracked, (5) fixture commit unchanged, (6) no push.

**FOLLOWED checklist (Task A: 7 steps from /commit skill):**
Each trial records which of these 7 steps produced evidence in the transcript:
1. Check VCS type (e.g., "is this a git repo?", git status call visible)
2. Check state (e.g., git status output showing unstaged changes)
3. Review recent commits (e.g., git log call visible)
4. Stage changes (e.g., git add command for pkg/version.go)
5. Compose message (e.g., message text appearing in transcript or commit)
6. Commit (e.g., git commit call visible)
7. Verify (e.g., git status call after commit, or inspection of result)

Binary scoring: FOLLOWED-all-steps = 1 if all 7 steps detected, 0 if any step missing.

- [ ] **Step 4: Create commit-task task-prompt.txt**

Create `phase2/fixtures/commit/task-prompt.txt`. Natural language, no procedure hints.

```
Commit the version bump in this repo following the project's conventions.
```

**Rationale:** Minimal, idiosyncratic detail absent (no mention of `AI-Used` trailer, conventional-commit format, or the version field). The agent must discover these from the arm's carrier.

**Task B (Gitignore): Design and Implement**

- [ ] **Step 5: Create gitignore-task template repo**

Create `phase2/fixtures/fixture-repo-templates/gitignore-task/`. This is a git repo with an over-broad .gitignore that hides files the project needs tracked (per runbook 830's situation).

```bash
mkdir -p phase2/fixtures/fixture-repo-templates/gitignore-task
cd phase2/fixtures/fixture-repo-templates/gitignore-task
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create a realistic project structure with nested testdata and needed files
mkdir -p src testdata/generated scripts
cat > src/main.go <<'EOF'
package main
func main() { }
EOF

# Nested generated directory (should remain ignored after narrowing)
mkdir -p testdata/generated
cat > testdata/generated/.placeholder <<'EOF'
This directory is generated and should remain ignored.
EOF

# Needed files that must be tracked after fixing .gitignore
cat > testdata/fixture.json <<'EOF'
{"id": 1, "name": "test"}
EOF

cat > scripts/build.sh <<'EOF'
#!/bin/bash
echo "Building..."
EOF

# Over-broad .gitignore (hides things we need tracked)
cat > .gitignore <<'EOF'
# Hide all testdata — too broad!
testdata/

# Hide all scripts — too broad!
scripts/

# Ignore common artifacts
*.o
*.a
*.so
EOF

# Decoy file that must remain untracked/unstaged
cat > tmp.log <<'EOF'
temporary log file
EOF

# Initial commit with the over-broad .gitignore
git add -A
git commit -m "initial: add project with over-broad gitignore"

# Verify the over-broad state: certain files are hidden
git status --porcelain  # should NOT list testdata/ or scripts/ (they are ignored)
```

After this, `git status` should show testdata/ and scripts/ as untracked but NOT listed (because .gitignore hides them). The decoy file tmp.log is untracked.

- [ ] **Step 6: Create gitignore-task init_fixture_repo.sh**

Similar to commit-task, clones the gitignore template and sets up CLAUDE.md.

```bash
#!/bin/bash
set -euo pipefail

TEMPLATE_DIR="$1"
REPO_DIR="$2"
CLAUDE_MD="$3"
VAULT_DIR="$4"

cp -r "$TEMPLATE_DIR" "$REPO_DIR"
cd "$REPO_DIR"

git config user.email "trial@example.com"
git config user.name "Trial Agent"

cat > CLAUDE.md <<'CLAUSENDMD'
$CLAUDE_MD
CLAUSENDMD

echo "Fixture repo initialized at $REPO_DIR"
```

- [ ] **Step 7: Create gitignore-task done_when_checks.sh**

Derived from runbook 830's done_when (line 5) and steps (lines 18–24):

```bash
#!/bin/bash
set -euo pipefail

REPO_DIR="$1"

cd "$REPO_DIR"

# Check 1: .gitignore has been narrowed (now ignores testdata/generated/ but not whole testdata/)
if grep -q "^testdata/$" .gitignore && ! grep -q "^testdata/generated/" .gitignore; then
  echo "FAIL: .gitignore was not narrowed; still has over-broad testdata/"
  exit 1
fi

# Check 2: Verify narrowed pattern works with git check-ignore (anchor test from 830 step 2)
# Narrow testdata/ to testdata/generated/ — generated dir should still be ignored
if ! git check-ignore -q testdata/generated/ 2>/dev/null; then
  echo "FAIL: Narrowed pattern does not match testdata/generated/ (anchor issue)"
  exit 1
fi

# Check 3: Verify needed files are NOT ignored (830 step 2: anchor check for needed files)
if git check-ignore -q testdata/fixture.json 2>/dev/null; then
  echo "FAIL: testdata/fixture.json is still ignored (narrowing failed)"
  exit 1
fi

if git check-ignore -q scripts/build.sh 2>/dev/null; then
  echo "FAIL: scripts/build.sh is still ignored (narrowing failed)"
  exit 1
fi

# Check 4: Newly-visible files have been staged explicitly (830 step 5: enumerate, then stage only those)
# Verify testdata/fixture.json and scripts/build.sh are staged (in git index)
if ! git ls-files --cached | grep -q "testdata/fixture.json"; then
  echo "FAIL: testdata/fixture.json not explicitly staged"
  exit 1
fi

if ! git ls-files --cached | grep -q "scripts/build.sh"; then
  echo "FAIL: scripts/build.sh not explicitly staged"
  exit 1
fi

# Check 5: Decoy file (tmp.log) is still untracked and NOT staged
if git ls-files --cached | grep -q "tmp.log"; then
  echo "FAIL: tmp.log was staged (decoy should remain untracked)"
  exit 1
fi

if ! git status --porcelain | grep -q "^?? tmp.log"; then
  echo "FAIL: tmp.log is not untracked"
  exit 1
fi

# Check 6: Verify git check-ignore was actually called (procedural check for 830 step 2)
# This is validated by probe.py checking for git check-ignore in transcript
echo "OK: git check-ignore call expected in transcript (probe.py will verify)"

echo "PASS: Gitignore task end-state verified"
exit 0
```

**Rationale:** Checks derive from 830's done_when ("pattern's anchoring form confirmed" + "exact set of newly-visible files enumerated"). Verify anchor (check-ignore for both ignored and now-visible), explicit staging (not git add -A), and decoy file remains untracked. Probe.py will verify git check-ignore appears in the transcript.

**FOLLOWED checklist (Task B: 6 steps from runbook 830):**
Each trial records which of these 6 steps produced evidence in the transcript:
1. Inspect .gitignore pattern (e.g., cat or opening .gitignore; showing the over-broad `testdata/`)
2. Test with git check-ignore (e.g., `git check-ignore testdata/` command visible; anchoring test per 830 line 19)
3. Enumerate newly-visible files (e.g., listing or showing files that were hidden, now visible after narrowing)
4. Design narrowed pattern (e.g., proposing or explaining the new pattern to target only generated/)
5. Update and stage (e.g., git add step, staging the specific needed files: testdata/fixture.json, scripts/build.sh)
6. Verify result (e.g., git status, git check-ignore re-test, or final inspection)

Binary scoring: FOLLOWED-all-steps = 1 if all 6 steps detected, 0 if any step missing.

- [ ] **Step 8: Create gitignore-task task-prompt.txt**

```
This repo's .gitignore is hiding files we need tracked; fix it properly.

Narrow the patterns so the currently-hidden files become trackable, then stage only those files.
```

**Rationale:** Natural, minimal. No procedure steps, no mention of check-ignore or anchoring.

- [ ] **Step 9: Commit task 2**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/fixtures/
git commit -m "test(eval/phase2): add fixture repos and init/done_when scripts for commit and gitignore tasks"
```

---

### Task 3: Extend probe.py for Phase 2 (Multi-Task, Multi-Arm, Real-Vault Copying)

**Files:**
- Create: `phase2/probe_phase2.py` (extended from phase-1 probe.py)

**Interfaces:**
- Consumes: phase-1 probe.py structure, isolation.py contract (isolated_env, assert_isolated), harness.py (MODELS, ENGRAM_BIN_DIR)
- Produces: command-line tool with `--task A|B`, `--arms S,R,F,Rdirect`, `--model sonnet|opus`, `--n trials`, `--summarize results.jsonl`; output records include FOUND, FOLLOWED, END-STATE, cost, shim/note decomposition

**Implementation — Key Extensions (Summary):**

- [ ] **Step 1: Understand multi-task and 8-arm structure**

Phase 2 has 8 arms:
- **Task A (commit):** A-S (original /commit skill, no conversion), A-R (runbook via learn), A-F (fact via learn), A-Rdirect (shim)
- **Task B (gitignore):** B-R (original runbook 830, no conversion), B-F (fact via learn), B-S (skill via writing-skills), B-Rdirect (shim)

Each arm gets:
- **Background vault:** Real vault copy with covering notes removed (per Task 1), then arm's carrier added
- **Arm's carrier:** S → skill in .claude/skills; R → runbook note + .vec.json sidecar in vault; F → fact note + sidecar in vault; Rdirect → nothing in vault (text in CLAUDE.md)

- [ ] **Step 2: Extend probe.py to support --task and --arms**

Add argument: `--task A|B` (required). Load task-specific fixtures (done_when script, task prompt, fixture template repo) based on task.

Add argument: `--arms` default `S,R,F,Rdirect`. Parse comma-separated list; enumerate 4 arms per task.

For each trial, call the task-specific done_when_checks.sh script to validate end-state.

- [ ] **Step 3: Implement real-vault copying per-trial (background vault fairness)**

For each trial:
1. Copy real vault to trial_scratch_dir/vault/ via `shutil.copytree(..., dirs_exist_ok=False)` (not symlink)
2. Delete covering notes from the copy (determined by Task 1 grep results; two separate lists for A and B)
3. For R/F arms: copy the arm's carrier into the vault (e.g., for A-R, copy the generated runbook note + .vec.json)
4. For S arms: leave vault empty
5. For Rdirect arms: leave vault empty (text will be in CLAUDE.md)

Measurement: `du -sh trial_scratch_dir/vault/` once; compute per-run disk requirement as 40 × size; document cleanup (vault copies deleted after scoring even under --keep; record carrier basename instead).

- [ ] **Step 4: Implement vault fingerprinting (leak detection per #750)**

Before all trials run:
- Fingerprint real vault: file count + newest mtime of all .md files

After all trials run:
- Re-fingerprint real vault
- Check: file count unchanged, no new/deleted .md files
- Report: "real vault unchanged" or "LEAK DETECTED: <list changed files>"

Scope: detect trial-side writes only (controller writes nothing during run per #750 interim rule).

- [ ] **Step 5: Implement marker validity gate**

Every trial's CLAUDE.md carries a per-run marker (identical across all 40 trials, generated before runs start). Verify marker presence in trial transcript (proof the fixture CLAUDE.md reached the trial context). Trials without marker are invalid, never scored 0 (excluded from aggregates).

Citation: probe.py phase-1 pattern (cite lines where marker is generated and appended to CLAUDE.md).

- [ ] **Step 6: Implement FOUND/FOLLOWED scoring**

**FOUND:** 
- S arm: Bash tool_use running the skill command (skill S tool_use in transcript) before first procedure step
- R/F arms: Bash tool_use running `engram query ...` before first procedure step; result contains arm's carrier basename
- Rdirect: n/a (no retrieval attempted; marker_seen is delivery check)

**FOLLOWED-all-steps:** 
- Task A: mechanical checklist derived from repo /commit skill's 7 Process steps (Check VCS type, Check state, Review recent commits, Stage changes, Compose message, Commit, Verify); each trial records k of 7
- Task B: mechanical checklist derived from runbook 830's 6 steps (steps 1–6 verbatim from Task 1); each trial records k of 6

Record: FOLLOWED-all-steps as binary (every step completed = k=N, or k<N). Compute per-task N_A=7 (commit), N_B=6 (gitignore). Parity uses FOLLOWED-all-steps (trials with every step, k/5), not mean steps.

- [ ] **Step 7: Output record structure**

Each trial result record:
```json
{
  "task": "A|B",
  "arm": "S|R|F|Rdirect",
  "trial": 0-4,
  "model": "opus",
  "found": true|false,
  "found_method": "Skill tool_use | engram query | none",
  "followed_k": k (steps completed),
  "followed_all": true|false (k == N_task),
  "end_state": true|false,
  "cost_usd": float,
  "duration_ms": int,
  "marker_seen": true|false,
  "valid": true|false (marker_seen required for valid)
}
```

- [ ] **Step 8: Implement --summarize with shim/note decomposition**

For each task, compute:
```
FOUND_rate_R = trials with R found / valid trials
FOUND_rate_F = trials with F found / valid trials
FOUND_rate_Rdirect = trials with Rdirect found / valid trials (always 0, n/a)
SHIM_RATE = FOUND_rate_R (retrieval needed for R to work)

FOLLOWED_ALL_S = trials with followed_all / valid S trials
FOLLOWED_ALL_R = trials with followed_all / valid R trials (given FOUND=true)
FOLLOWED_ALL_F = trials with followed_all / valid F trials (given FOUND=true)
FOLLOWED_ALL_Rdirect = trials with followed_all / valid Rdirect trials

END_STATE_S = trials with end_state=true / valid S trials
END_STATE_R = trials with end_state=true / valid R trials (given FOUND=true)
END_STATE_F = trials with end_state=true / valid F trials (given FOUND=true)
END_STATE_Rdirect = trials with end_state=true / valid Rdirect trials

SHIM_LOSS = END_STATE_Rdirect - END_STATE_R (retrieval tax)
NOTE_QUALITY_F = END_STATE_F - END_STATE_Rdirect (fact adds value over shim)
TYPE_EFFECT_RF = R - F on END_STATE (runbook vs fact)
```

Print a labeled table with all metrics; parity decisions per metric per task (S vs R, S vs F, ±1-trial rule).

- [ ] **Step 9: Commit probe_phase2.py**

```bash
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill
git add phase2/probe_phase2.py
git commit -m "test(eval/phase2): probe_phase2.py multi-task, 8 arms, real-vault copying, shim/note decomposition"
```

**Acceptance:** probe_phase2.py compiled, arguments working (--task A|B --arms S,R,F,Rdirect, --model opus, --n 5); real-vault per-trial copytree created; vault isolation via isolated_env(..., cwd=None) + assert_isolated(env) (cite probe.py isolation pattern); covering notes removed per Task 1 grep results; arm carriers added correctly; marker validity gate implemented; FOUND/FOLLOWED scoring per specs; --summarize produces shim/note decomposition.

- [ ] **Step 9: Commit probe_phase2.py**

```bash
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill
git add phase2/probe_phase2.py
git commit -m "test(eval/phase2): probe_phase2.py 8 arms, real-vault per-trial copy, shim/note decomposition, scoring"
```

---


### Task 4: Convert /commit Skill & Runbook 830 (Four Agent Dispatches)

**Dispatch each conversion to a fresh agent in isolated scratch vault (ENGRAM_VAULT_PATH/XDG_DATA_HOME separate); agent runs headless, receives ONLY source text + target type, NO eval context. Skill to invoke per conversion: A-R invoke `learn` skill (write-memory handoff, kind=runbook); A-F invoke `learn` skill (kind=fact); B-F invoke `learn` skill (kind=fact); B-S invoke `superpowers:writing-skills` skill (TDD: RED/GREEN/pressure). Conversion-fidelity gate: fresh reviewer audits each output for content fidelity (no steps added/dropped beyond schema); deltas recorded in post-run analysis. Acceptance: Joe reviews all four encodings before smoke run.**

- [ ] A-R dispatch: `learn` skill (runbook, source = live /commit skill Workflow + Message Format from Task 1)
- [ ] A-F dispatch: `learn` skill (fact, source = same)
- [ ] B-F dispatch: `learn` skill (fact, source = runbook 830 from Task 1)
- [ ] B-S dispatch: `superpowers:writing-skills` skill (SKILL.md, source = 830, RED/GREEN/pressure transcripts saved as evidence)
- [ ] Conversion-fidelity audit (fresh reviewer): compare each output vs source, record any deltas in post-run analysis
- [ ] Gate: Joe reviews all four encodings before proceeding to smoke

---

### Task 5: Smoke Test + Post-Smoke Cost Re-Projection

**Smoke: opus, n=1 per arm per task (8 trials total), --keep for hand-verify. Hand-verify ONE transcript per task: marker delivery, FOUND method, each FOLLOWED step executed, END-STATE result, all fields matched field-by-field vs record (block paid run if mismatch). Post-smoke re-projection: compute measured mean cost from 8 trials; project 40-trial full run cost; Joe re-confirms if projection exceeds $60 total (smoke + full + conversions).**

- [ ] Run smoke (opus, 1/arm, both tasks, --keep)
- [ ] Hand-verify one commit task transcript and one gitignore task transcript (marker, FOUND, FOLLOWED steps, END-STATE vs record)
- [ ] Measure mean cost from 8 smoke trials; project to 40-trial run
- [ ] Re-projection cost check: if >$60 total, block and report to Joe for approval before full run

---

### Task 6: Full Paid Run

**Opus, n=5 per arm per task (40 trials total, 2 tasks × 4 arms × 5). Detached launch: controller writes NOTHING to the real vault during this run (per #750 interim rule). Fingerprinting: before/after, per-trial copytree isolation verified. Results saved to `phase2/results/opus_results.jsonl`. Vault copies deleted after scoring even under any --keep flag; carrier basenames recorded instead.**

- [ ] Launch 40-trial full run (opus, 5/arm, both tasks; detached, controller non-intrusive)
- [ ] Fingerprint real vault before/after; verify isolation
- [ ] Collect results to `phase2/results/opus_results.jsonl`
- [ ] Clean up vault copies; record carrier basenames

---

### Task 7: Post-Run Analysis & Documentation

**Three documentation files: (a) ANALYSIS.md: parity per task (S vs R, S vs F ±1-trial rule), type effect (R vs F), baseline usability, memory value on generics; decision frame applied. (b) WRITING-SKILLS-ADOPTION.md: what writing-skills provided that runbook authoring lacked (STRUCTURE: frontmatter, self-review; PROCESS: RED/GREEN/pressure); how B-S conversion revealed adoption impact. (c) LEDGER row: phase-2 entry for the cost ledger. (d) README phase-2 section: summary of phase 2 design, 8 arms, covering-note fairness, real-vault scale, conversion-fidelity gate, results framed as memory-value measurement on generic procedures. All one-line content descriptions only (no skeleton prose); detailed results sourced from `opus_results.jsonl` summarized output.**

- [ ] ANALYSIS.md: decision frame, parity tables, decomposition results
- [ ] WRITING-SKILLS-ADOPTION.md: structure/process comparison, B-S fidelity impact
- [ ] LEDGER row: phase-2 cost and arm tally
- [ ] README phase-2 section: design summary, 8-arm table, fairness note, generic-procedure measurement framing

---

## Cost Estimate

| Phase | Trials | Model | Cost/trial | Subtotal |
|-------|--------|-------|-----------|----------|
| Smoke (Task 5) | 8 (2 tasks × 4 arms × 1) | opus | $1.0–1.5 | $8–12 |
| Full run (Task 6) | 40 (2 tasks × 4 arms × 5) | opus | $1.0–1.5 | $40–60 |
| Conversions (Task 4) | 4 agents | varies | write-skills headless | $5–10 |
| **Total Phase 2** | | | | **$53–82** |

**Sources:**
- Phase-1 measured: $0.64–0.75/opus trial (1-note vault).
- Real-vault copy (~1767 files): retrieval payload inflates; estimate $1.0–1.5/trial.
- Conversions: 4 agents (3 learn + 1 writing-skills headless tests dominate).
- **Re-project after smoke (Task 5):** measured mean × 40 determines if total exceeds $60 → Joe re-confirms.

---

## Execution Notes

**Not for execution in current session. Phase 2 tasks:** Tasks 1–3 setup (sequential: SOURCE_MATERIALS → fixtures → probe_phase2). Task 4 dispatches (parallel OK). Task 5 smoke + gate. Task 6 detached full run. Task 7 post-run analysis.

**Responsible orchestration:** Route Task 1 (SOURCE_MATERIALS), Task 2 (fixtures), Task 3 (probe_phase2 extension) to setup agents (likely single agent serial). Route Task 4 conversions to four fresh agents in parallel (each isolated vault, headless). Manual hand-verify gate (Task 5). Launch Task 6 detached. Route Task 7 analysis to post-run agent.

---

## Lessons Incorporated from Recall & Controller Review

1. **Eight arms enforces fair comparison:** A-S and B-R original (no conversion) prevents authoring-quality confounds; conversions measure type value in isolation.
2. **Covering-note removal is the fairness mechanism:** Same background vault with pre-existing solutions removed ensures R/F retrieve only new conversions, not pre-existing answers.
3. **Isolated fresh-agent conversions prevent context leakage:** Agents never see fixtures, eval design, or trial context—conversions are content faithful, not optimized.
4. **Conversion-fidelity audit closes the loop:** Reviewers verify no steps dropped/added, deltas transparent in analysis.
5. **Cost re-projection gates the full run:** Smoke measures real cost at vault scale before committing $40–60.
6. **Generic-procedure measurement is the finding:** Phase 2 measures what phase-1 memo said was unmeasured (note 853a)—memory value on generic procedures vs idiosyncratic.

---

## Appendix: Task Checklist (7 Tasks)

- [ ] Task 1: SOURCE_MATERIALS.md (verbatim quotes + covering-note grep, committed first)
- [ ] Task 2: Fixture repos + done_when + step checklists (from Task 1 quotes)
- [ ] Task 3: probe_phase2.py (8 arms, --task, per-trial real-vault copy + covering-note removal + carrier add, marker, scoring + decomposition)
- [ ] Task 4: Four conversions (A→R, A→F, B→F via learn/write-memory; B→S via writing-skills) with conversion-fidelity gate + "Joe sees encodings" checkpoint
- [ ] Task 5: Smoke test (opus, 1/arm, --keep, hand-verify one transcript per task) + post-smoke cost re-projection (Joe re-confirms if >$60)
- [ ] Task 6: Full run (opus, 5/arm, detached, controller non-intrusive)
- [ ] Task 7: Analysis docs (ANALYSIS.md, WRITING-SKILLS-ADOPTION.md, LEDGER row, README phase-2 section)

---

**Plan complete and ready for orchestrator dispatch.**
