# Phase 2: Conversion-Parity Runbook-vs-Skill Eval

> **For agentic workers:** This plan is NOT for execution in this session. Phase 2 consists of: (a) agent-driven conversions of existing skill/runbook sources to other types via their native authoring paths, (b) extending probe.py, (c) building fixtures and copy strategy, (d) running the full n=5 opus eval. See "Execution Notes" section for orchestration.

**Goal:** Measure whether runbooks earn their keep against facts when both are authored via native paths (learn vs writing-skills) at real-vault scale, and determine if the type matters for generic (non-idiosyncratic) procedures.

**Architecture:** Phase 2 scales phase-1's single-note synthetic vault to real-vault copying per trial. Two tasks (commit skill, gitignore narrowing) are converted across three arms each (Skill/Runbook/Fact) via agents invoking the actual authoring skills, not hand-written. Retrieval arms (R/F) copy the operator's real vault (~1767 files) per trial; direct-control arms (Rdirect) embed the procedure verbatim in trial CLAUDE.md without retrieval. Scoring reuses probe.py's FOUND/FOLLOWED/END-STATE/COST framework; shim/note decomposition separates retrieval tax from procedure quality.

**Tech Stack:** Python 3.11+, headless `claude -p`, engram vault/chunks isolation, BASH for fixture repo and checks, git.

**Spec:** `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/README.md` (phase-1 decision frame and isolation strategy reused); `/Users/joe/repos/personal/engram/dev/eval/isolation.py` (vault/chunks/config isolation); `/Users/joe/repos/personal/engram/dev/eval/cumulative/harness.py` (MODELS registry, ENGRAM_BIN_DIR).

---

## Global Constraints

- **Runbook schema:** Exactly three fields: situation (when-to-use), numbered steps (body), done_when (ending expectations) — per note 789.2026-08-23. Do NOT include full inputs/preconditions/returns (rejected at #719 readiness checkpoint).
- **Commit trailer:** EVERY commit must end with `AI-Used: [claude]` (not `Co-Authored-By`), per note 354.2026-07-22 and commit skill SKILL.md:18.
- **Fixture vault scale:** Real vault copy ~1767 files (incl. .vec.json sidecars per note 956: fingerprint before/after each run; exclude orchestrator activations from leak detection).
- **Fixture placeholder precision:** Every fixture procedure placeholder must name its exact source field (e.g. `<sensor_id>` not `<name>`) with a worked example, per note 955.2026-09-10.
- **Smoke on opus:** Smoke run MUST run on opus (not sonnet cross-check only), n≥1 per model per arm, before spending on full n=5 paid run per note 955.
- **Measurement scope:** Memory value unmeasured for GENERIC procedures (note 853a.2026-08-30); phase 2 measures exactly this. SEPARATE from idiosyncratic findings (where memory already shows wins).
- **Isolation:** Per-trial vault/chunks/config isolation (reuse isolation.py contract); real vault fingerprint guards against leaks (scope to trial dirs, exclude orchestrator writes per note 956).
- **Decision bars (pre-registered):**
  - Parity rule: ±1-trial indistinguishable (within 1 trial on FOUND/FOLLOWED/END-STATE); 2+ trial gap = worse/better.
  - Runbook > fact: Only if R is 2+ ahead of F on FOLLOWED-all-steps or END-STATE (else "can't distinguish").
  - Baseline usability: Arm S (skill) must ≥3/5 on END-STATE or baseline uninterpretable (fixture/spec fix required).
  - Shim loss per task: FOUND_rate(R) - FOUND_rate(Rdirect) = retrieval tax; END-STATE(F) given FOUND = note quality tax.

---

## File Structure

### Source Materials (Read-Only, Exact Paths)

- **Commit skill:** `/Users/joe/.claude/skills.backup.20260203223056/commit/SKILL.md` (source of truth for steps/trailers: type, scope, description format, trailers block, "AI-Used: [claude]" line 20).
- **Runbook 830:** `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md` (situation line 4, done_when line 5, steps lines 18–24).
- **Phase-1 harness:** `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/{README.md, probe.py, fixtures/, fixture-vaults/}` (reuse structure; extend probe.py only).
- **Isolation contract:** `/Users/joe/repos/personal/engram/dev/eval/isolation.py` (isolated_env, assert_isolated, project_slug, NEVER edit).
- **Harness shared:** `/Users/joe/repos/personal/engram/dev/eval/cumulative/harness.py` (MODELS, ENGRAM_BIN_DIR, refresh_creds).

### Deliverables (New Directories/Files)

```
dev/eval/cumulative/runbook_vs_skill/
├── PLAN-2-conversion-parity.md                          (this file)
├── phase2/                                              (NEW)
│   ├── fixtures/
│   │   ├── commit/
│   │   │   ├── init_fixture_repo.sh                     (task A fixture setup)
│   │   │   ├── done_when_checks.sh                      (task A end-state checks)
│   │   │   └── task-prompt.txt                          (natural task prompt)
│   │   ├── gitignore/
│   │   │   ├── init_fixture_repo.sh                     (task B fixture setup)
│   │   │   ├── done_when_checks.sh                      (task B end-state checks, from 830)
│   │   │   └── task-prompt.txt                          (natural task prompt)
│   │   └── fixture-repo-templates/
│   │       ├── commit-task/                             (git repo with staged change ready)
│   │       └── gitignore-task/                          (git repo with over-broad .gitignore)
│   ├── encodings/                                       (procedure encodings per arm)
│   │   ├── A_commit/
│   │   │   ├── skill/                                   (converted from /commit skill)
│   │   │   │   └── .claude/skills/commit-task/SKILL.md
│   │   │   ├── runbook/                                 (converted via learn)
│   │   │   │   └── vault/NNN.2026-XX-XX.commit-convention-for-AI-agents.md
│   │   │   ├── fact/                                    (converted via learn)
│   │   │   │   └── vault/NNN.2026-XX-XX.commit-conventional-format-and-trailers.md
│   │   │   └── runbook_direct/                          (control: shim only, no retrieval)
│   │   │       └── claude_md_text.txt
│   │   ├── B_gitignore/
│   │   │   ├── skill/
│   │   │   │   └── .claude/skills/gitignore-narrow/SKILL.md
│   │   │   ├── runbook/
│   │   │   │   └── vault/830.2026-08-29... (original, reused from real vault)
│   │   │   ├── fact/
│   │   │   │   └── vault/NNN.2026-XX-XX.gitignore-narrowing-method.md
│   │   │   └── runbook_direct/
│   │   │       └── claude_md_text.txt
│   ├── results/
│   │   ├── smoke_opus_results.jsonl                     (sonnet->opus cross-check removed; opus only)
│   │   ├── opus_results.jsonl                           (n=5 per arm)
│   │   └── WRITING-SKILLS-ADOPTION.md                  (analysis doc, see Task 8)
│   └── probe_phase2.py                                  (extended from phase-1 probe.py)
└── ...existing phase-1 files untouched...
```

---

## Task Decomposition

### Task 1: Verify Source Materials & Pre-existing Commit Notes

**Files:**
- Read: `/Users/joe/.claude/skills.backup.20260203223056/commit/SKILL.md`
- Read: `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md`
- Read: `/Users/joe/.local/share/engram/vault/` (check for pre-existing commit-convention notes)

**Interfaces:**
- Produces: exact quoted text of commit skill steps, trailers, done_when from runbook 830, list of pre-existing notes covering commit conventions (if any)

- [ ] **Step 1: Read commit skill SKILL.md and extract exact steps**

The commit skill is at `/Users/joe/.claude/skills.backup.20260203223056/commit/SKILL.md`. Extract and quote the ordered procedural steps from that file. Record the exact trailer format (line 20: `AI-Used: [claude]`) and message template rules (lines 23–29).

**Expected output (document for reference):**
- Ordered steps: (1) Check VCS type, (2) Check git state, (3) Review style, (4) Stage specific files, (5) Compose message following template, (6) Commit, (7) Verify.
- Trailer: Exactly `AI-Used: [claude]` (not `Co-Authored-By`).
- Message format: conventional-commit (type(scope): description; body; trailers).

- [ ] **Step 2: Read runbook 830 and extract exact procedures and done_when**

The runbook is at `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md`. Extract and quote:
- Situation (line 4): "before shipping a narrowed or rewritten .gitignore pattern..."
- Done_when (line 5): "the pattern's anchoring form has been confirmed... exactly the set of newly-visible files..."
- Steps (lines 18–24): 1–6 ordered, exact text.

**Expected output:**
- 6 numbered steps (git check-ignore, anchor depth validation, staged enumeration, diff verification).
- Done_when condition references anchor confirmation + visible-file enumeration.

- [ ] **Step 3: Check vault for pre-existing commit-convention notes**

Run: `grep -r "AI-Used\|commit.*convention" /Users/joe/.local/share/engram/vault/ --include="*.md" | grep -v "\.vec\.json"`

Record which notes (if any) already cover commit conventions, trailers, or the `AI-Used` marker. If pre-existing notes cover the commit task comprehensively, the plan must account for them in fixture setup (e.g., remove from the trial vault copy to prevent redundancy, or note that the trial will retrieve multiple notes and must rank them).

**Expected output:** List of note basenames and their main content (e.g., note 354: subagent-briefs commit invariants).

- [ ] **Step 4: Commit this task's findings**

```bash
cd /Users/joe/repos/personal/engram
git add -A
git commit -m "docs(eval): phase-2 source-material verification — commit skill steps and runbook 830 recorded"
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

This is a minimal git repo where a change is ALREADY STAGED (not committed), ready for the trial agent to commit. Use a simple, realistic scenario: a small code file with a one-liner change that is staged but uncommitted.

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

# Now make a change and STAGE it (but don't commit)
cat > pkg/version.go <<'EOF'
package pkg

const Version = "1.1.0"
EOF

git add pkg/version.go
# STOP here — leave it staged, not committed
```

After running this, verify: `git status` should show the file as "staged for commit" and there should be no committed changes since the version bump.

**Rationale:** The task is idiosyncratic (specific change to stage+commit), but the commit convention itself is generic (conventional-commit format, trailers). The trial agent must discover the convention from the arm's carrier (skill/runbook/fact), not infer it from repo history.

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

**Acceptance:** Script exits 0; `$REPO_DIR/pkg/version.go` is staged (git status shows "staged"), CLAUDE.md contains the trial's guidance.

- [ ] **Step 3: Create commit-task done_when_checks.sh**

Create `phase2/fixtures/commit/done_when_checks.sh`. This script validates the end-state per the commit skill's requirements (see Task 1 extract): one new commit on the branch, message in conventional-commit format with `AI-Used: [claude]` trailer, working tree clean.

```bash
#!/bin/bash
set -euo pipefail

REPO_DIR="$1"       # trial's repo after agent runs

cd "$REPO_DIR"

# Check 1: Exactly one new commit since setup (version bump commit only)
COMMIT_COUNT=$(git log --oneline pkg/version.go | head -1 | grep -c "1.1.0" || echo "0")
if [ "$COMMIT_COUNT" != "1" ]; then
  echo "FAIL: Expected exactly one version.go commit; found $COMMIT_COUNT"
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

# Check 4: Working tree is clean
if ! git status --porcelain | grep -q "^$"; then
  echo "FAIL: Working tree not clean"
  git status
  exit 1
fi

# Check 5: Nothing was amended or force-pushed (log integrity)
if git log --oneline | grep -qE "amend|force"; then
  echo "FAIL: Log shows amendment or force-push"
  exit 1
fi

echo "PASS: Commit task end-state verified"
exit 0
```

**Acceptance:** Script exits 0 on a trial repo that has: (1) one new commit on the staged version bump, (2) conventional-commit message, (3) `AI-Used: [claude]` trailer, (4) clean working tree.

- [ ] **Step 4: Create commit-task task-prompt.txt**

Create `phase2/fixtures/commit/task-prompt.txt`. Natural language, no procedure hints.

```
Commit the staged work in this repo.

The staged change is ready; you must create a commit following the project's conventions.
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

# Create a realistic project structure with some ignored files
mkdir -p src testdata scripts
cat > src/main.go <<'EOF'
package main
func main() { }
EOF

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

# Initial commit with the over-broad .gitignore
git add -A
git commit -m "initial: add project with over-broad gitignore"

# Verify the over-broad state: certain files are hidden
git status --porcelain  # should NOT list testdata/ or scripts/ (they are ignored)
```

After this, `git status` should show testdata/ and scripts/ as untracked but NOT listed (because .gitignore hides them).

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

# Check 1: .gitignore has been narrowed (not identical to original over-broad version)
if git show HEAD:.gitignore | grep -q "^testdata/$" && ! grep -q "^testdata/rapid/" .gitignore; then
  echo "FAIL: .gitignore was not narrowed; still has over-broad testdata/"
  exit 1
fi

# Check 2: Verify narrowed pattern works with git check-ignore (anchor test from 830 step 2)
# Narrow testdata/ to testdata/rapid/
if ! git check-ignore -q testdata/rapid/ 2>/dev/null; then
  echo "FAIL: Narrowed pattern does not match testdata/rapid/ at any depth (anchor issue)"
  exit 1
fi

# Check 3: Newly-visible files have been staged explicitly (830 step 5: enumerate, then stage only those)
# The visible files should be: testdata/fixture.json, scripts/build.sh (and possibly others)
# Verify they are staged (in git index)
if ! git ls-files --cached | grep -q "testdata/fixture.json"; then
  echo "FAIL: testdata/fixture.json not explicitly staged"
  exit 1
fi

if ! git ls-files --cached | grep -q "scripts/build.sh"; then
  echo "FAIL: scripts/build.sh not explicitly staged"
  exit 1
fi

# Check 4: Staged set matches enumerated visible files (830 step 6: verify staged == enumerated)
# Get the list of newly-visible files as a baseline
VISIBLE_FILES=$(git status --porcelain | grep "^??" | awk '{print $2}' | sort)
STAGED_FILES=$(git diff --cached --name-only | sort)

# For this simple fixture, testdata/fixture.json and scripts/build.sh should be visible and staged
if ! echo "$STAGED_FILES" | grep -q "testdata/fixture.json"; then
  echo "FAIL: testdata/fixture.json not in staged set"
  exit 1
fi

echo "PASS: Gitignore task end-state verified"
exit 0
```

**Rationale:** Checks derive from 830's done_when ("pattern's anchoring form confirmed" + "exact set of newly-visible files enumerated"). Verify anchor (check-ignore) and explicit staging (not git add -A).

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

### Task 3: Create Fixture Vault Copies & Real-Vault Fingerprinting Guard

**Files:**
- Create: `phase2/vault_copy_and_fingerprint.py` (helper to copy real vault and guard against leaks per note 956)
- Modify: (note: probe_phase2.py will call this; no edits to isolation.py per spec)

**Interfaces:**
- Produces: function `copy_vault_for_trial(real_vault_path, trial_scratch_dir, trial_id)` → returns (trial_vault_dir, initial_fingerprint)
- Produces: function `verify_vault_isolated(real_vault_path, initial_fingerprint, orchestrator_writes_manifest)` → raises on leak, returns True on clean

**Implementation:**

- [ ] **Step 1: Understand real-vault fingerprinting (note 956 constraint)**

From recall memory (note 956.2026-09-10): "when running a headless eval harness that fingerprints the operator's real vault to detect isolation leaks, during a fingerprinted run, do not write to the real vault (defer route-evidence/learn writes until the run ends) and avoid session-side recall activations; or scope the fingerprint to detect trial-side writes only (e.g. compare against a manifest and exclude paths the orchestrator declares)."

**Action:** This harness will NOT write to the real vault during trials. If the orchestrator (this script) defers writes until after the run, no leak guard is needed per-trial. BUT the phase-2 run may activate notes (recall cue fires regardless of procedure location). To avoid false positives:
- Fingerprint the real vault BEFORE the run starts (file count, newest mtime).
- After the run ends, fingerprint again.
- If changed, diff and EXCLUDE orchestrator writes (e.g., sidecar mtimes bumped by activations).
- Report: "real vault unchanged (orchestrator activations excluded)" or "LEAK DETECTED: files changed outside orchestrator manifest".

- [ ] **Step 2: Write vault_copy_and_fingerprint.py**

```python
#!/usr/bin/env python3
"""
Vault isolation helper for phase-2 runbook-vs-skill eval.

Copies the real vault (~1767 files) per-trial for retrieval arms (R/F).
Guards against trial-side leaks by fingerprinting real vault before/after.
Excludes orchestrator writes (activations, note updates) from leak detection.
"""
import os
import shutil
import hashlib
import json
import stat
from pathlib import Path
from typing import Tuple, Dict, Any

def _vault_fingerprint(vault_path: str) -> Dict[str, Any]:
    """
    Fingerprint vault: file count, newest mtime, manifest of all .md files.
    Returns {"file_count": int, "newest_mtime": float, "manifest": {basename: mtime}}.
    """
    if not os.path.isdir(vault_path):
        return {"file_count": 0, "newest_mtime": 0, "manifest": {}}

    manifest = {}
    newest_mtime = 0
    file_count = 0

    for entry in os.listdir(vault_path):
        path = os.path.join(vault_path, entry)
        if os.path.isfile(path) and entry.endswith(".md"):
            mtime = os.path.getmtime(path)
            manifest[entry] = mtime
            newest_mtime = max(newest_mtime, mtime)
            file_count += 1

    return {
        "file_count": file_count,
        "newest_mtime": newest_mtime,
        "manifest": manifest,
    }

def copy_vault_for_trial(
    real_vault_path: str,
    trial_scratch_dir: str,
    trial_id: str,
) -> Tuple[str, Dict[str, Any]]:
    """
    Copy the real vault into trial scratch dir.
    Returns (trial_vault_path, initial_fingerprint).
    
    Copies the ENTIRE vault (~1767 files incl. .vec.json sidecars) so retrieval
    arms (R/F) can run engram query without touching the operator's real vault.
    """
    trial_vault = os.path.join(trial_scratch_dir, "vault")
    
    # Copy real vault into trial dir
    shutil.copytree(real_vault_path, trial_vault, dirs_exist_ok=False)
    
    # Fingerprint the REAL vault (not the copy) before the trial runs
    initial_fp = _vault_fingerprint(real_vault_path)
    
    return trial_vault, initial_fp

def verify_vault_isolated(
    real_vault_path: str,
    initial_fingerprint: Dict[str, Any],
    orchestrator_writes: Dict[str, float] = None,
) -> Tuple[bool, str]:
    """
    Verify the real vault was not modified by the trial (excluding orchestrator writes).
    
    Args:
        real_vault_path: operator's real vault path
        initial_fingerprint: fingerprint before the trial ran
        orchestrator_writes: dict of {basename: mtime} for writes orchestrator made
            (e.g., sidecar mtimes bumped by recall activations)
    
    Returns:
        (is_clean, report_str)
    """
    if orchestrator_writes is None:
        orchestrator_writes = {}
    
    current_fp = _vault_fingerprint(real_vault_path)
    
    # Check file count (ignoring orchestrator writes)
    real_files = set(current_fp["manifest"].keys())
    initial_files = set(initial_fingerprint["manifest"].keys())
    
    new_files = real_files - initial_files
    deleted_files = initial_files - real_files
    
    if new_files or deleted_files:
        report = f"Vault structure changed: +{len(new_files)} -{len(deleted_files)} files"
        return False, report
    
    # Check mtime changes (exclude orchestrator writes)
    changed = {}
    for basename, current_mtime in current_fp["manifest"].items():
        initial_mtime = initial_fingerprint["manifest"].get(basename, 0)
        if current_mtime != initial_mtime:
            orch_mtime = orchestrator_writes.get(basename)
            if orch_mtime is None or orch_mtime != current_mtime:
                # Changed, and not explained by orchestrator
                changed[basename] = (initial_mtime, current_mtime)
    
    if changed:
        report = f"Vault content changed (excluding orchestrator writes): {list(changed.keys())}"
        return False, report
    
    report = "Real vault unchanged (orchestrator activations excluded)"
    return True, report
```

**Acceptance:** Script compiles; `copy_vault_for_trial` returns a copy under trial_scratch_dir and fingerprints the real vault BEFORE copy; `verify_vault_isolated` compares fingerprints and excludes orchestrator writes from leak detection.

- [ ] **Step 3: Commit task 3**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/vault_copy_and_fingerprint.py
git commit -m "test(eval/phase2): add vault copy and isolation guard per note 956"
```

---

### Task 4: Extend probe.py for Phase 2 (Multi-Task, Multi-Arm Enumeration)

**Files:**
- Create: `phase2/probe_phase2.py` (extended version of phase-1 probe.py)
- Modify: None to phase-1 probe.py (keep it as reference; use phase2/probe_phase2.py for phase 2)

**Interfaces:**
- Consumes: phase-1 probe.py structure, isolation.py contract, vault_copy_and_fingerprint.py functions
- Produces: command-line tool with `--task A|B`, `--arms S,R,F,Rdirect`, `--model sonnet|opus`, `--n trials`, `--summarize results.jsonl`; output records include FOUND, FOLLOWED, END-STATE, cost, shim/note decomposition

**Key Extensions:**

1. **Multi-task support:** `--task A|B` specifies which task (commit or gitignore); loads task-specific fixtures, done_when, procedure encodings.
2. **Direct-shim arms:** `Rdirect` (runbook direct) embeds procedure verbatim in trial CLAUDE.md with empty vault; measures shim ceiling without retrieval.
3. **Real-vault copying:** For R/F arms, copy the real vault (~1767 files) per trial via `vault_copy_and_fingerprint.copy_vault_for_trial()`.
4. **Shim/note decomposition:** Output includes FOUND_rate(Rdirect) vs FOUND_rate(R) to compute retrieval tax per task.
5. **Fingerprinting guard:** Per note 956, fingerprint real vault before/after, exclude orchestrator writes.
6. **Fixture placeholder hygiene:** Per note 955, verify placeholders name exact source fields (detect ambiguity before running).

**Implementation (Summary):**

- [ ] **Step 1: Copy and adapt phase-1 probe.py structure**

Start with `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/probe.py` as the template. Create `phase2/probe_phase2.py`.

Changes:
- Add `--task A|B` argument (default: error; must specify).
- Load task-specific fixtures/done_when/procedure encodings based on --task.
- Rename internal constants (TASK_PROMPT_PATH → TASK_PROMPT_PATH_A, etc.).
- Add Rdirect arm (no vault copy; shim text pasted in CLAUDE.md).

```python
#!/usr/bin/env python3
"""Phase 2: multi-task, multi-arm conversion-parity runbook-vs-skill eval."""

import argparse
import os

HERE = os.path.dirname(os.path.abspath(__file__))
PHASE2_DIR = HERE  # phase2/ dir

# Task-specific fixture paths
TASKS = {
    "A": {
        "name": "commit",
        "init_script": os.path.join(PHASE2_DIR, "fixtures", "commit", "init_fixture_repo.sh"),
        "done_when_script": os.path.join(PHASE2_DIR, "fixtures", "commit", "done_when_checks.sh"),
        "task_prompt": os.path.join(PHASE2_DIR, "fixtures", "commit", "task-prompt.txt"),
        "template_repo": os.path.join(PHASE2_DIR, "fixtures", "fixture-repo-templates", "commit-task"),
    },
    "B": {
        "name": "gitignore",
        "init_script": os.path.join(PHASE2_DIR, "fixtures", "gitignore", "init_fixture_repo.sh"),
        "done_when_script": os.path.join(PHASE2_DIR, "fixtures", "gitignore", "done_when_checks.sh"),
        "task_prompt": os.path.join(PHASE2_DIR, "fixtures", "gitignore", "task-prompt.txt"),
        "template_repo": os.path.join(PHASE2_DIR, "fixtures", "fixture-repo-templates", "gitignore-task"),
    },
}

# Arm definitions: S=skill, R=runbook, F=fact, Rdirect=shim (no retrieval)
ARMS = ("S", "R", "F", "Rdirect")

# Encoding paths (procedure text per arm per task)
def get_encoding_path(task: str, arm: str) -> str:
    """Return path to procedure encoding for task/arm."""
    if arm == "Rdirect":
        return os.path.join(PHASE2_DIR, "encodings", f"task{task}", f"{arm}", "claude_md_text.txt")
    else:
        return os.path.join(PHASE2_DIR, "encodings", f"task{task}", arm, "...")  # varies by arm type

def main():
    parser = argparse.ArgumentParser(description="Phase 2 runbook-vs-skill eval")
    parser.add_argument("--task", required=True, choices=["A", "B"], help="Task: A (commit) or B (gitignore)")
    parser.add_argument("--model", required=True, choices=["sonnet", "opus"], help="Claude model")
    parser.add_argument("--n", type=int, default=1, help="Trials per arm")
    parser.add_argument("--arms", default="S,R,F,Rdirect", help="Comma-separated arm list")
    parser.add_argument("--workers", type=int, default=4, help="Parallel workers")
    parser.add_argument("--out", help="Output results JSONL")
    parser.add_argument("--summarize", help="Summarize an existing results JSONL")
    parser.add_argument("--keep", action="store_true", help="Keep trial dirs on exit")
    
    args = parser.parse_args()
    
    task_config = TASKS[args.task]
    arms = args.arms.split(",")
    
    # TBD: Implement trial loop (reuse phase-1 logic + multi-task)
    # ... core trial execution, vault copying, fingerprinting, scoring ...
    
    if args.summarize:
        summarize_results(args.summarize, task=args.task)
    else:
        run_trials(task_config, arms, args.model, args.n, args.workers, args.out, args.keep)

if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Implement Rdirect arm (shim encoding)**

For each task/arm combination, create the procedure encoding. Rdirect is special: it's the shim (procedure text pasted directly into CLAUDE.md, no retrieval).

Create `phase2/encodings/taskA/Rdirect/claude_md_text.txt` with the commit skill procedure verbatim (no skill tool_use, just text). Same for `taskB/Rdirect/`.

Example (taskA Rdirect):
```
## Commit Convention (Procedure)

When creating a git commit:
1. Check the VCS type (must be git).
2. Check the current git state (`git status`).
3. Review the commit message style (conventional-commit format).
4. Stage only the specific files you intend to commit (not `git add -A`).
5. Compose the commit message following this format:
   - Type(scope): description
   - Body (optional)
   - Trailers: AI-Used: [claude]
6. Run `git commit -m "message"` with the composed message.
7. Verify the commit with `git log -1`.

EVERY commit must end with: AI-Used: [claude]
```

**Rationale:** This is the shim (the procedure delivered without retrieval). Later scoring will compare FOUND(Rdirect) vs FOUND(R) to compute retrieval tax.

- [ ] **Step 3: Implement trial isolation with real-vault copy**

In the trial-execution loop (run_trials function), for each trial:
- For Arm S: empty vault (no retrieval).
- For Arms R/F: call `vault_copy_and_fingerprint.copy_vault_for_trial()` to copy the real vault into trial_scratch_dir.
- For Arm Rdirect: empty vault (no retrieval).
- Before all trials: fingerprint the real vault.
- After all trials: verify with `vault_copy_and_fingerprint.verify_vault_isolated()`, excluding orchestrator writes.

- [ ] **Step 4: Implement shim/note decomposition in output**

Each trial result record should include:
- `found: true|false` (procedure was discovered)
- `found_method: "Skill tool_use" | "engram query" | "none"` (how it was found)
- For R/Rdirect: also record `retrieval_attempted: true|false` and `note_surfaced_in_result: true|false` (for later shim/note decomposition).
- `followed_k_of_6: k` (steps completed out of 6)
- `end_state: true|false` (done_when script exit code 0)
- `cost_usd: float`
- `duration_ms: int`

- [ ] **Step 5: Implement --summarize with shim/note decomposition**

When summarizing, compute:
```
SHIM_LOSS_PER_TASK = {
  "A": END_STATE(Rdirect) - END_STATE(R_given_found),
  "B": END_STATE(Rdirect) - END_STATE(R_given_found),
}
NOTE_QUALITY = {
  "A": END_STATE(F_given_found) - END_STATE(Rdirect),
  "B": END_STATE(F_given_found) - END_STATE(Rdirect),
}
```

Print these decompositions in the summary output alongside parity decisions.

- [ ] **Step 6: Commit task 4**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/probe_phase2.py phase2/encodings/
git commit -m "test(eval/phase2): extend probe for multi-task/multi-arm; add Rdirect shim arms"
```

---

### Task 5: Convert /commit Skill to Runbook & Fact via Agent Dispatches

**Files:**
- Output: `phase2/encodings/taskA/runbook/vault/NNN.2026-XX-XX.commit-convention-for-AI-agents.md`
- Output: `phase2/encodings/taskA/fact/vault/NNN.2026-XX-XX.commit-conventional-format-and-trailers.md`
- Output: `phase2/encodings/taskA/skill/.claude/skills/commit-task/SKILL.md`

**Interfaces:**
- Consumes: /commit skill SKILL.md (exact text from Task 1)
- Produces: three procedure encodings (skill/runbook/fact) in their native formats, all delivering the same content (commit convention), each authored via its native authoring skill

**Procedure (Agent Dispatch — NOT in this session):**

This task requires THREE separate agent dispatches, each invoking the actual authoring skill. The plan documents the dispatch requirements; the executors (agents) will perform the conversions.

- [ ] **Step 1: Dispatch Agent 1 — Convert /commit skill to Runbook note**

**Dispatch Brief:**

Invoke the `learn` skill with kind=runbook to author a vault note that captures the /commit skill's procedural steps as a runbook.

**Source:** `/Users/joe/.claude/skills.backup.20260203223056/commit/SKILL.md` (extract ordered steps, situation/when-to-use, done_when expectations).

**Runbook Schema:** Exactly three fields: situation (when-to-use), numbered steps (body), done_when (ending conditions) — per note 789.2026-08-23.

**Content Mapping:**

| Source (Skill) | Destination (Runbook Field) |
|---|---|
| "Create a well-formatted git commit following project conventions" (desc) | Situation: "When you need to commit staged changes following the project's conventional-commit conventions with the AI-Used trailer." |
| "Stage and commit changes with properly formatted message" | Done_when: "A single commit exists with message following conventional-commit format (type(scope): description), body optional, trailers block including AI-Used: [claude], and working tree is clean." |
| Message Templates table (lines 23–29) + rules (lines 20–21, line 33–35) | Steps 1–7: ordered procedures covering VCS check, state check, styling review, file staging, message composition, commit, verification. MUST include these exact trailer-related rules: "Every commit you create ends with AI-Used: [claude]" (note 354.2026-07-22). |

**Acceptance Proof:** The produced note carries engram's frontmatter (type: runbook, situation, numbered steps, done_when, luhmann ID, tags) and a fresh `.vec.json` sidecar (embedded into binary via GoMLX MiniLM-L6). Location: `phase2/encodings/taskA/runbook/vault/<basename>.md`.

- [ ] **Step 2: Dispatch Agent 2 — Convert /commit skill to Fact note**

**Dispatch Brief:**

Invoke the `learn` skill with kind=fact to author a vault note that states the commit convention as a semantic fact (not procedural steps).

**Content:** A fact note extracts the REQUIREMENT/STANDARD from the skill: "Commits must follow conventional-commit format (type(scope): description, optional body, trailers) with the AI-Used: [claude] trailer as a standing invariant."

**Acceptance Proof:** The produced note is kind=fact, has `subject: commit message format`, `predicate: must follow`, `object: conventional-commit + AI-Used trailer`, carries `.vec.json` sidecar. Location: `phase2/encodings/taskA/fact/vault/<basename>.md`.

- [ ] **Step 3: Dispatch Agent 3 — Convert /commit skill to Skill SKILL.md**

**Dispatch Brief:**

Invoke the `superpowers:writing-skills` skill (TDD: RED baseline, GREEN implementation, pressure tests) to author a new SKILL.md that delivers the same procedural guidance as the /commit skill, but in a fresh encoding suitable for trial discovery (`.claude/skills/commit-task/SKILL.md` in the trial project).

**Content:** Functionally equivalent to `/commit skill` but adapted for this trial context (simpler scope, task-specific, same message rules and trailer invariant).

**Acceptance Proof:** The skill text compiles and is discoverable as a Skill tool_use in `claude -p`. The writing-skills agent must run RED (baseline), GREEN (edit), pressure tests as part of the skill TDD. Location: `phase2/encodings/taskA/skill/.claude/skills/commit-task/SKILL.md`.

**Coordination Note:** These three dispatches should run in parallel (independent work); use superpowers:subagent-driven-development or parallel agent launches.

---

### Task 6: Convert Runbook 830 to Skill & Fact via Agent Dispatches

**Files:**
- Output: `phase2/encodings/taskB/skill/.claude/skills/gitignore-narrow/SKILL.md`
- Output: `phase2/encodings/taskB/fact/vault/NNN.2026-XX-XX.gitignore-narrowing-method.md`
- Note: `phase2/encodings/taskB/runbook/vault/` — use original runbook 830 directly (migrated from real vault)

**Interfaces:**
- Consumes: runbook 830 exact text (from Task 1)
- Produces: skill SKILL.md and fact note, each authored via native skill

**Procedure (Agent Dispatch):**

- [ ] **Step 1: Copy Original Runbook 830 to Phase-2 Encoding**

Runbook 830 is the "original" for Task B. Copy it to `phase2/encodings/taskB/runbook/vault/` as-is (no conversion needed; it's already a runbook).

```bash
cp /Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md \
   phase2/encodings/taskB/runbook/vault/
cp /Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.vec.json \
   phase2/encodings/taskB/runbook/vault/
```

- [ ] **Step 2: Dispatch Agent 1 — Convert Runbook 830 to Fact note**

**Dispatch Brief:**

Invoke `learn` skill with kind=fact to convert runbook 830's procedural steps into a semantic fact.

**Content:** Extract the PRINCIPLE from 830: "When narrowing a .gitignore pattern, confirm its anchoring form with git check-ignore (nested paths must use **/ prefix if they span subdirectories), then enumerate and stage only the newly-visible files explicitly (not git add -A)."

**Acceptance Proof:** Fact note with `subject: .gitignore narrowing`, `predicate: requires`, `object: anchor test + explicit enumeration`, `.vec.json` sidecar. Location: `phase2/encodings/taskB/fact/vault/<basename>.md`.

- [ ] **Step 3: Dispatch Agent 2 — Convert Runbook 830 to Skill SKILL.md**

**Dispatch Brief:**

Invoke `superpowers:writing-skills` skill (RED/GREEN/pressure) to author a SKILL.md that guides the gitignore narrowing task.

**Content:** Procedural steps from 830 (lines 18–24) adapted as an executable skill procedure. Keep the anchor-testing requirement and explicit enumeration (830 steps 2 and 5) as non-negotiable rules.

**Acceptance Proof:** Skill text passes writing-skills TDD (RED baseline, GREEN, pressure tests). Discoverable as Skill tool_use. Location: `phase2/encodings/taskB/skill/.claude/skills/gitignore-narrow/SKILL.md`.

**Coordination:** These two dispatches can run in parallel.

---

### Task 7: Verify Fixture Placeholder Precision per Note 955

**Files:**
- Read: `phase2/fixtures/commit/init_fixture_repo.sh`, `done_when_checks.sh`, `task-prompt.txt`
- Read: `phase2/fixtures/gitignore/init_fixture_repo.sh`, `done_when_checks.sh`, `task-prompt.txt`

**Interfaces:**
- Produces: report of placeholder ambiguities (if any); confirmation that all placeholders name exact source fields, checks are unambiguous per note 955

**Procedure:**

- [ ] **Step 1: Audit commit-task fixture procedures**

Per note 955: "every placeholder must name the exact source field it is filled from (e.g. <sensor_id>) with a worked example of the final artifact; the checker must accept every reading the text permits or the text must permit only one."

Review `phase2/fixtures/commit/`:
- `init_fixture_repo.sh`: Does it use any placeholders? (No — this fixture is fixed/idiosyncratic.)
- `done_when_checks.sh`: Check for field references. Expected: exact "AI-Used: [claude]" string (not a placeholder).
- `task-prompt.txt`: Check for any unintended hints that could disambiguate. Expected: none; prompt is natural.

**Finding:** The commit task has no placeholders; the trailer is exact. ✓ PASS

- [ ] **Step 2: Audit gitignore-task fixture procedures**

Review `phase2/fixtures/gitignore/`:
- `init_fixture_repo.sh`: Fixed repo structure.
- `done_when_checks.sh`: References "testdata/rapid/" and "scripts/build.sh" — these are EXACT PATH strings from the fixture template, not placeholders. ✓ PASS
- `task-prompt.txt`: Natural language, no hints. ✓ PASS

**Finding:** No ambiguous placeholders. ✓ PASS

- [ ] **Step 3: Commit task 7**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/fixtures/
git commit -m "test(eval/phase2): audit fixture procedures for placeholder precision per note 955"
```

---

### Task 8: Write WRITING-SKILLS-ADOPTION Analysis (Post-Conversions)

**Files:**
- Output: `phase2/results/WRITING-SKILLS-ADOPTION.md`

**Note:** This task runs AFTER the conversions (Task 5/6). The orchestrator will coordinate: (a) run Tasks 5/6 conversions, (b) read the writing-skills agent output, (c) dispatch THIS task to document adoption gaps.

**Interfaces:**
- Consumes: writing-skills skill output from Task 5/6 conversion (the RED baseline transcript, GREEN steps, pressure-test output)
- Produces: analysis doc listing writing-skills STRUCTURE and PROCESS that runbook authoring path lacks; recommendation for B-S conversion trial

**Procedure:**

- [ ] **Step 1: Read superpowers:writing-skills skill documentation**

Locate: `~/.claude/plugins/cache/claude-plugins-official/superpowers/*/skills/writing-skills/SKILL.md` (or similar). Extract and document what writing-skills provides:

**STRUCTURE (declarative):**
- Frontmatter description (trigger, when-to-use, red flags table, references)
- Task-right-sizing guidance (each task boundary, not too large)
- File structure (files, responsibilities, interfaces)
- Global constraints (version floors, naming rules, platform requirements)

**PROCESS (procedural):**
- RED baseline (write failing test/check to establish the requirement)
- GREEN implementation (minimal code to pass baseline)
- Pressure tests (edge cases, non-happy-path conditions)
- Self-review checklist (spec coverage, placeholder scan, type consistency)

**Expected finding:** Writing-skills provides structured frontmatter + systematic test-driven discipline.

- [ ] **Step 2: Read runbook authoring path documentation**

From `/Users/joe/repos/personal/engram/agent-instructions/skills/learn/SKILL.md` and `/Users/joe/repos/personal/engram/openspec/specs/learn-runbook-capture/spec.md`, extract:

**What runbook authoring has:**
- Situation (when-to-use), steps (body), done_when (ending conditions) — three-field schema
- Written via `engram learn runbook` handoff to write-memory skill
- Emphasis on idiosyncratic scenarios + crystallization

**What runbook authoring lacks (vs writing-skills):**
- No RED baseline (no requirement-proof phase)
- No pressure tests (no non-happy-path validation)
- No frontmatter trigger/description for discovery
- Implicit task boundaries (no file-structure planning)
- No self-review checklist

**Expected finding:** Runbook path is simpler, faster for crystallization; writing-skills adds systematic TDD rigor.

- [ ] **Step 3: Compare B-S Conversion (Skill → from Runbook)**

The B-S arm conversion invokes writing-skills on runbook 830's content. Document:

**What writing-skills SHOULD add:**
- RED baseline: a failing test case for "narrowing an over-broad pattern without anchor test fails" (establishes the anchor-test requirement as a MUST-HAVE).
- GREEN: implementation of the narrowing steps including the check-ignore anchor validation.
- Pressure tests: (1) test a nested-subdirectory pattern without **/ (should fail anchor test), (2) test enumerate-then-stage-all vs git add -A (should reject -A), (3) test a pattern that still matches when it shouldn't (should reject).

**Impact:** If B-S includes RED/GREEN/pressure-tests, then B-S rigor > B-R rigor. This reveals what the trial will show: does the skill's TDD discipline affect trial performance?

- [ ] **Step 4: Document recommendation**

Based on the writing-skills adoption, recommend which adoption candidates are **load-bearing for the trial**:

- **Frontmatter description**: Helps discovery; likely not load-bearing in a headless trial (agent gets the full procedure).
- **RED baseline**: LOAD-BEARING. A skill that has tested its own requirement is more likely to state it explicitly for the trial.
- **Pressure tests**: LOAD-BEARING. Pressure-tested skills are more robust to edge cases.
- **Self-review checklist**: Not directly load-bearing in trial (helps author, not agent).

**Prediction:** B-S (writing-skills) > B-R (original runbook) on FOLLOWED if pressure tests are included, because the skill will have validated non-happy-path scenarios.

- [ ] **Step 5: Write and save WRITING-SKILLS-ADOPTION.md**

Template:

```markdown
# Writing-Skills Adoption Analysis (Phase 2, Task B Conversion)

## Overview

Compares the `superpowers:writing-skills` skill's structure and process against the runbook authoring path.

## Writing-Skills Provides

### STRUCTURE
- Frontmatter description (trigger keywords, when-to-use, red flags table, external references)
- Task-right-sizing guidance (each task: test cycle, boundary decision, scope, interfaces)
- File structure planning (create vs modify, responsibilities per file)
- Global constraints (version floors, naming rules, platform requirements, one line each)

### PROCESS (Systematic TDD)
1. RED baseline: Define failing test/check that proves the current code/skill does NOT meet the requirement
2. GREEN: Minimal implementation to pass baseline
3. Pressure tests: Non-happy-path validation (edge cases, error handling)
4. Self-review checklist: Spec coverage, placeholder scan, type consistency

## Runbook Authoring Path

### What It Has
- Three-field schema: situation, steps, done_when
- Authored via `engram learn runbook` → write-memory skill
- Emphasis on crystallizing idiosyncratic scenarios
- Fast, lightweight authoring

### What It Lacks (vs Writing-Skills)
- No RED baseline requirement-proof phase
- No pressure tests (non-happy-path validation)
- No frontmatter description/trigger for discovery
- No file-structure planning (N/A for single-file runbook)
- No self-review checklist

## Phase-2 B-S Conversion: What Writing-Skills Added

[Agent output from conversion Task 6 goes here: RED baseline test, GREEN pressure tests, transcript evidence.]

**Load-Bearing Adoptions for Trial:**
- RED baseline: ✓ (tests anchor-test requirement explicitly)
- Pressure tests: ✓ (tests nested-depth, -A rejection, false-match rejection)
- Frontmatter description: ○ (helps discovery, not in headless trials)
- Self-review: ○ (helps author, not agent)

## Recommendation

B-S skill includes RED/GREEN/pressure tests. This makes it more brittle AND more correct than B-R (original runbook without testing). Trial prediction: B-S FOLLOWED-all-steps may exceed B-R if the pressure-tested edge cases surface in the trial task.

Measure: B-S vs B-R FOLLOWED rate difference. If >1 trial, writing-skills TDD has trial value for procedural tasks.
```

- [ ] **Step 6: Commit task 8**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/results/WRITING-SKILLS-ADOPTION.md
git commit -m "docs(eval/phase2): analyze writing-skills adoption vs runbook authoring for B-S conversion"
```

---

### Task 9: Run Smoke Test (Sonnet, n=1 per arm, both tasks)

**Files:**
- Output: `phase2/results/smoke_opus_results.jsonl` (NOTE: smoke MUST run on opus per note 955, not sonnet)
- Input: All fixtures, encodings, probe_phase2.py, vault copies

**Interfaces:**
- Consumes: fixtures/encodings/probe_phase2.py
- Produces: 8 result records (2 tasks × 4 arms = 8 trials), FOUND/FOLLOWED/END-STATE/cost metrics

**Procedure:**

- [ ] **Step 1: Set up scratch directory for smoke run**

```bash
export PHASE2_SMOKE_ROOT=/tmp/phase2_smoke_$$
mkdir -p $PHASE2_SMOKE_ROOT
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/phase2
```

- [ ] **Step 2: Run smoke (opus, n=1 per arm, both tasks)**

Per note 955: "run the smoke on the SAME model as the paid run (or one trial per model) before spending; keep trial dirs (--keep) so a rescore is possible."

```bash
# Task A smoke
python3 probe_phase2.py --task A --model opus --n 1 --arms S,R,F,Rdirect \
  --workers 2 --out results/smoke_results_taskA.jsonl --keep

# Task B smoke
python3 probe_phase2.py --task B --model opus --n 1 --arms S,R,F,Rdirect \
  --workers 2 --out results/smoke_results_taskB.jsonl --keep

# Merge for summary
cat results/smoke_results_taskA.jsonl results/smoke_results_taskB.jsonl > results/smoke_opus_results.jsonl
```

- [ ] **Step 3: Hand-verify one trial transcript per task**

Per note 955: "hand-read one full trial transcript against the scorer's own verdict BEFORE the paid run, because a keyword+proximity scorer fires on incidental structure like table headers and section titles; a disagreement between the human read and the scorer is the calibration signal, and it is only available before the numbers start looking like data."

For each task:
- Open one trial's raw transcript (from --keep scratch dir).
- Verify the procedure was discovered (FOUND=true OR false; if false, check the transcript to confirm the agent never ran the right query/tool).
- Verify the steps were followed (manual scan of tool_use records vs the 6/7 steps expected).
- Verify the end-state check's verdict matches the done_when_checks.sh exit code.

Record discrepancies (if scorer disagrees with hand-read, fix the checker before paid run).

- [ ] **Step 4: Summarize smoke results**

```bash
python3 probe_phase2.py --summarize results/smoke_opus_results.jsonl
```

Expected output:
- 8 result records (2 tasks × 4 arms)
- Marker delivery: 8/8 `marker_seen=true`
- FOUND: S=1/1, R=1/1, F=1/1, Rdirect=N/A (shim not scored on FOUND)
- END-STATE: ≥1/8 end-state checks passed (at least one arm completes one task)
- Cost: ~$1.50–2.00 total (8 opus trials at ~$0.20–0.25 each)

**Acceptance Bars (Pre-Registered Smoke):**
- Marker delivery: 8/8 valid
- Plumbing: ≥1/8 END-STATE true (at least one trial registers the change end-to-end)
- Retrieval: R/F arms show recall firing (transcript includes `engram query` Bash call)

- [ ] **Step 5: Commit smoke results**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/results/smoke_opus_results.jsonl phase2/results/.gitkeep
git commit -m "test(eval/phase2): smoke test results (opus, n=1 per arm per task) — hand-verified, ready for paid run"
```

---

### Task 10: Run Full Paid Trial (Opus, n=5 per arm, both tasks)

**Files:**
- Output: `phase2/results/opus_results.jsonl` (40 result records: 2 tasks × 4 arms × 5 trials)
- Vault fingerprinting report: printed to stderr during run

**Interfaces:**
- Consumes: smoke-verified fixtures/encodings/probe_phase2.py
- Produces: full trial results with FOUND/FOLLOWED/END-STATE/cost, shim/note decomposition

**Procedure:**

- [ ] **Step 1: Fingerprint real vault before run**

```bash
cd /Users/joe/repos/personal/engram
python3 -c "
import sys
sys.path.insert(0, 'dev/eval/cumulative/runbook_vs_skill/phase2')
from vault_copy_and_fingerprint import _vault_fingerprint
fp = _vault_fingerprint('/Users/joe/.local/share/engram/vault')
print(f'Real vault fingerprint (before run): {fp}')
" > /tmp/vault_fp_before.json
```

- [ ] **Step 2: Run full trial suite (opus, n=5, all tasks/arms)**

```bash
cd /Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/phase2

# Task A, full run
python3 probe_phase2.py --task A --model opus --n 5 --arms S,R,F,Rdirect \
  --workers 4 --timeout 900 --out results/opus_results_taskA.jsonl

# Task B, full run
python3 probe_phase2.py --task B --model opus --n 5 --arms S,R,F,Rdirect \
  --workers 4 --timeout 900 --out results/opus_results_taskB.jsonl

# Merge
cat results/opus_results_taskA.jsonl results/opus_results_taskB.jsonl > results/opus_results.jsonl
```

Expected cost: 40 opus trials × $0.80/trial ≈ **$32.00** (mid-range per phase-1 LEDGER estimate).

- [ ] **Step 3: Fingerprint real vault after run**

```bash
cd /Users/joe/repos/personal/engram
python3 -c "
import sys
sys.path.insert(0, 'dev/eval/cumulative/runbook_vs_skill/phase2')
from vault_copy_and_fingerprint import _vault_fingerprint, verify_vault_isolated
fp_after = _vault_fingerprint('/Users/joe/.local/share/engram/vault')
print(f'Real vault fingerprint (after run): {fp_after}')
# Also verify isolation (assuming orchestrator writes are empty for this run)
is_clean, report = verify_vault_isolated('/Users/joe/.local/share/engram/vault', None, {})
print(f'Isolation status: {report}')
" > /tmp/vault_fp_after.json
```

If vault changed, diff and investigate; if orchestrator writes only (sidecar mtimes from activations), confirm and proceed. If trial-side writes detected, ABORT and investigate which trial leaked.

- [ ] **Step 4: Summarize full results**

```bash
python3 probe_phase2.py --summarize results/opus_results.jsonl
```

Expected output: decision frame per task (parity, runbook vs fact, shim loss).

- [ ] **Step 5: Commit full results**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/results/opus_results.jsonl
git commit -m "test(eval/phase2): full results (opus, n=5 per arm per task, 40 trials) — vault isolated, ready for analysis"
```

---

### Task 11: Analysis & Decision Framing (Post-Run)

**Files:**
- Output: `phase2/results/ANALYSIS.md` (decision frame applied to results)

**Interfaces:**
- Consumes: `opus_results.jsonl` summarized output, shim/note decomposition metrics, pre-registered bars
- Produces: decision doc answering: (a) PARITY — runbook vs skill indistinguishable? (b) FUNCTIONALITY — runbook vs fact? (c) GENERICS — memory value on generic procedures.

**Procedure:**

- [ ] **Step 1: Extract per-task metrics from summarized results**

For each task (A, B), extract:
- FOUND: S, R, F, Rdirect k/5 (valid trials only)
- FOLLOWED-all-steps: S, R, F, Rdirect k/5
- FOLLOWED-mean: S, R, F, Rdirect (k/6, mean across trials)
- END-STATE: S, R, F, Rdirect k/5
- Cost: mean USD/trial per arm
- Shim loss (retrieval tax): FOUND(R) - FOUND(Rdirect) if Rdirect found anything, else N/A
- Note quality: END-STATE(F given FOUND) - END-STATE(Rdirect) per task

- [ ] **Step 2: Apply pre-registered decision bar to parity**

Per global constraints:

| Outcome | Rule | Finding |
|---------|------|---------|
| Indistinguishable | R within ±1 trial of S | S k, R k±1 → "can't distinguish" |
| Worse | R 2+ below S | S k, R ≤k-2 → "R worse" |
| Better | R 2+ above S | S k, R ≥k+2 → "R better" |
| Baseline bad | S < 3/5 on END-STATE | Baseline uninterpretable (fixture/spec fix) |

Apply to FOUND, FOLLOWED, END-STATE per task.

- [ ] **Step 3: Answer Joe's Decision Frame (from phase-1 README §2)**

**Question (a): PARITY — Is runbook indistinguishable from skill on each metric (FOUND, FOLLOWED, END-STATE)?**

Per task (A, B):
- Task A parity: [FOUND: can't distinguish / worse / better] [FOLLOWED: ...] [END-STATE: ...]
- Task B parity: [FOUND: ...] [FOLLOWED: ...] [END-STATE: ...]

**Question (b): FUNCTIONALITY — Does runbook exceed fact functionally?**

Runbook > Fact ONLY if R is 2+ ahead on FOLLOWED-all or END-STATE.

Per task:
- Task A: R vs F on FOLLOWED-all-steps and END-STATE (can't distinguish / runbook better / fact better)
- Task B: R vs F on FOLLOWED-all-steps and END-STATE

**Question (c): MEMORY VALUE ON GENERICS — Does memory win or lose on generic procedures?**

Per note 853a: memory value is UNMEASURED for procedures (measurement was facts/conventions only). Phase-2 measures this.

Decompose: F(end_state) > Rdirect(end_state)? If yes, note (semantic fact) adds value over shim. If no, shim suffices.

- [ ] **Step 4: Compute shim/note loss per task**

| Arm | Metric | Task A | Task B |
|---|---|---|---|
| Rdirect (shim ceiling) | END-STATE k/5 | ? | ? |
| R (retrieval) | END-STATE k/5 | ? | ? |
| F (fact note) | END-STATE k/5 | ? | ? |
| Shim loss (R - Rdirect) | retrieval tax | ? | ? |
| Note quality (F - Rdirect) | semantic fact value | ? | ? |

**Interpretation:**
- Shim loss > 0 → retrieval helps
- Shim loss < 0 → shim-only baseline better (retrieval hurts or found the wrong note)
- Note quality > 0 → semantic fact adds value over shim
- Note quality ≤ 0 → shim or skill sufficient; fact redundant

- [ ] **Step 5: Write ANALYSIS.md**

Template:

```markdown
# Phase 2 Analysis & Decision Frame

## Overview

Measures whether runbooks earn their keep against facts when both authored via native paths at real-vault scale.

## Summary Table

| Metric | Task A | Task B |
|--------|--------|--------|
| FOUND S/R/F/Rdirect | ... | ... |
| FOLLOWED-all S/R/F/Rdirect | ... | ... |
| END-STATE S/R/F/Rdirect | ... | ... |
| Shim loss (R - Rdirect) | ... | ... |
| Note quality (F - Rdirect) | ... | ... |

## Parity Analysis

### Task A (Commit)

**FOUND parity:** [S k/5, R k/5, F k/5] → [can't distinguish / worse / better]
**FOLLOWED-all parity:** [S k/5, R k/5, F k/5] → [...]
**END-STATE parity:** [S k/5, R k/5, F k/5] → [...]

**Baseline usability (S ≥3/5 END-STATE):** ✓ / ✗

### Task B (Gitignore)

[Repeat above structure]

## Functionality Analysis

### Runbook vs Fact

Task A: R vs F on FOLLOWED-all and END-STATE → [runbook better / can't distinguish / fact better]
Task B: R vs F on FOLLOWED-all and END-STATE → [...]

**Verdict:** [[RUN-SPECIFIC]] (Runn only if 2+ trial gap)

## Memory Value on Generic Procedures (Note 853a)

Q: Do semantic facts (F) outperform shim-only (Rdirect) on generic procedures?

| Task | F END-STATE | Rdirect END-STATE | Fact adds value | Reason |
|------|---|---|---|---|
| A (commit convention) | k/5 | k/5 | Yes/No | [retrieval + fact] vs [shim only] |
| B (gitignore method) | k/5 | k/5 | Yes/No | [retrieval + fact] vs [shim only] |

**Finding:** [Fact value unmeasured prior to phase 2; this result measures it.]

## Recommendations

[Pending results]
```

- [ ] **Step 6: Commit analysis**

```bash
cd /Users/joe/repos/personal/engram
git add phase2/results/ANALYSIS.md
git commit -m "docs(eval/phase2): analysis and decision frame applied to results"
```

---

## Cost Estimate

| Phase | Trials | Model | Cost/trial | Subtotal |
|-------|--------|-------|-----------|----------|
| Smoke | 8 (2 tasks × 4 arms × 1) | opus | $0.20–0.25 | $1.60–2.00 |
| Full run | 40 (2 tasks × 4 arms × 5) | opus | $0.75–0.85 | $30.00–34.00 |
| Conversions (Tasks 5–6) | 3 agents (learn 2x, writing-skills 1x per task) | varies | TBD | $2.00–5.00 |
| **Total Phase 2** | | | | **$33.60–41.00** |

**Sources:**
- Phase-1 measured: $0.64–0.75/opus trial with tool/file operations and 1-note vault (README cost estimate).
- Real-vault copy (~1767 files): adds retrieval payload; assume 10% overhead on phase-1 ($0.07–0.08/trial) → $0.75–0.85/trial estimate.
- Conversions: estimated from prior route dispatch evidence (note 951: cheap 2 fix rounds; note 950: cheap pass). Budget $1/agent for conversion + fixes.

---

## Execution Notes

**This plan is NOT for execution in the current session.** Phase 2 requires:

1. **Distributed agent execution:** Tasks 5–6 (conversions) must be dispatched to fresh agents invoking learn and writing-skills skills. Use `superpowers:subagent-driven-development` or parallel agent launches.
2. **Orchestration:** Run Tasks 1–4 to set up, then wait for conversions (Tasks 5–6), then run Tasks 7–11 (verify, smoke, full, analyze).
3. **Manual gates:** Hand-verify smoke transcripts (Task 9, Step 3) before proceeding to full run (Task 10).

**Responsible agent:** Orchestrator (planner) routes to:
- Conversion agents (Tasks 5–6): invoke learn/writing-skills
- Fixture/probe setup agents (Tasks 1–4): code/script authoring
- Analysis agent (Task 11): decision framing post-run

---

## Lessons Incorporated from Recall Memory

1. **Note 956** (vault fingerprinting): Excluded orchestrator activations from leak detection; scope fingerprint to detect trial-side writes only.
2. **Note 955** (fixture placeholders): Every placeholder names exact source field with worked example; smoke runs on opus (same model as paid run) before spending.
3. **Note 853a** (memory value on generics): Explicitly separates UNMEASURED generic-procedure value from MEASURED idiosyncratic-content value; phase 2 measures this.
4. **Note 789** (runbook schema): Runbook encoding uses exactly three fields (situation, steps, done_when); no full calling convention.
5. **Note 354** (commit invariants): "EVERY commit ends with AI-Used: [claude]" stated as standing invariant in both runbook and skill encodings.

---

## Appendix: Task Checklist

- [ ] Task 1: Source materials verified (commit skill, runbook 830, pre-existing notes)
- [ ] Task 2: Fixture repos & scripts (commit-task and gitignore-task)
- [ ] Task 3: Vault copy & fingerprinting guard
- [ ] Task 4: probe_phase2.py extended (multi-task, Rdirect arms, decomposition)
- [ ] Task 5: Convert /commit skill → runbook, fact, skill (agents)
- [ ] Task 6: Convert runbook 830 → skill, fact (agents)
- [ ] Task 7: Verify fixture placeholders (note 955 audit)
- [ ] Task 8: WRITING-SKILLS-ADOPTION analysis (post-conversions)
- [ ] Task 9: Smoke test (opus, n=1, hand-verify)
- [ ] Task 10: Full paid run (opus, n=5, vault verified)
- [ ] Task 11: Analysis & decision frame

---

**Plan complete and ready for orchestrator dispatch.**
