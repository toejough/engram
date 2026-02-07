---
name: commit-producer
description: Create phase-scoped git commit (commit-red/green/refactor phases)
context: inherit
model: sonnet
skills: ownership-rules
user-invocable: false
role: producer
phase: commit-red | commit-green | commit-refactor
---

# Commit Producer

<!-- Traces: ARCH-039 -->

Create git commits scoped to current TDD phase (red/green/refactor).

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

This skill follows the producer pattern from [PRODUCER-TEMPLATE](../shared/PRODUCER-TEMPLATE.md).

### GATHER Phase

1. Read project context (from spawn prompt in team mode):
   - Current phase (commit-red, commit-green, commit-refactor)
   - Files modified in current phase
   - TASK-N being implemented
2. Run `git status` to see staged and unstaged changes
3. Run `git diff` to see what changed
4. Run `git log -1` to see recent commit message style
5. Proceed to SYNTHESIZE when context is clear

### SYNTHESIZE Phase

1. Determine which files to stage based on phase
2. Validate no secrets in staged files
3. Draft commit message following conventional commits format
4. If blocked (e.g., no changes to commit), yield `blocked`
5. Prepare git commands

### PRODUCE Phase

1. Stage appropriate files with `git add <files>`
2. Create commit with message and AI trailer
3. Verify commit created successfully with `git log -1`
4. Yield `complete` with commit hash and files staged

## Staging Rules by Phase

| Phase | Files to Stage | Rationale |
|-------|----------------|-----------|
| commit-red | Test files only (new tests from tdd-red) | Red phase produces failing tests, no implementation |
| commit-green | Test files + implementation files (from tdd-green) | Green phase writes minimal implementation to pass tests |
| commit-refactor | Implementation files only (refactored code from tdd-refactor) | Refactor phase improves code without changing behavior |

**Staging principle:** Only stage files that were modified in the current TDD phase. Do NOT stage unrelated changes.

## Secret Detection

Before staging, check for:
- `.env`, `.env.*` files
- `credentials.json`, `secrets.yaml`
- Files containing `API_KEY=`, `SECRET=`, `PASSWORD=`
- Private key patterns (`-----BEGIN PRIVATE KEY-----`)

If secrets detected, yield `blocked` with details.

## Commit Message Format

Follow conventional commits format:

```
<type>(<scope>): <description>

<optional body>

AI-Used: [claude]
```

**Type selection:**

| Type | When to Use |
|------|-------------|
| `test` | commit-red (adding tests) |
| `feat` | commit-green (adding feature implementation) |
| `refactor` | commit-refactor (improving code structure) |
| `fix` | Any phase fixing a bug |

**Scope:** Use issue ID without "ISSUE-" prefix (e.g., `issue-92`)

**Description:** Present tense, lowercase, no period (e.g., `add failing tests for step registry`)

**Body:** Optional - add context if needed

**Trailer:** ALWAYS include `AI-Used: [claude]` (NOT `Co-Authored-By`)

## Git Commands

### Staging Files

```bash
# Stage specific files
git add path/to/file1.go path/to/file2.go
```

**Never use `git add .` or `git add -A`** - always stage files explicitly by name.

### Creating Commit

Use heredoc for proper formatting:

```bash
git commit -m "$(cat <<'EOF'
test(issue-92): add failing tests for step registry

Add tests for per-phase QA registry entries.

AI-Used: [claude]
EOF
)"
```

### Verifying Commit

```bash
git log -1 --oneline
git show --stat HEAD
```

## Rules

| Rule | Rationale |
|------|-----------|
| Stage files explicitly by name | Prevents accidentally staging secrets or unrelated changes |
| NO `git add .` or `git add -A` | Too broad, dangerous |
| Follow phase scope strictly | Prevents cross-phase contamination |
| Always include AI trailer | Traceability |
| NO `--no-verify` flag | Respect pre-commit hooks |
| NO force commands | Never `--amend` or `--force` |

## Yield Protocol

### Complete Yield

When commit succeeds:

```toml
[yield]
type = "complete"
timestamp = 2026-02-06T10:30:00Z

[payload]
commit_hash = "abc1234"
files_staged = ["internal/step/registry_test.go", "internal/step/next_test.go"]
commit_message = "test(issue-92): add failing tests for step registry"

[[payload.decisions]]
context = "File selection"
choice = "Staged only test files for commit-red phase"
reason = "Phase scope enforcement"

[context]
phase = "commit-red"
task = "TASK-18"
subphase = "complete"
```

### Blocked Yield

When cannot proceed:

```toml
[yield]
type = "blocked"
timestamp = 2026-02-06T10:35:00Z

[payload]
blocker = "Secrets detected in staged files"
details = ".env file contains API keys"
suggested_resolution = "Remove .env from staging, add to .gitignore"

[context]
phase = "commit-red"
task = "TASK-18"
awaiting = "blocker-resolution"
```

## Failure Recovery

| Symptom | Action |
|---------|--------|
| No changes to commit | Yield `blocked` - verify phase completed |
| Secrets detected | Yield `blocked` with file list |
| Commit fails | Check git status, yield `blocked` with error details |
| Pre-commit hook fails | Read hook output, fix issues, retry commit |

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Check git status | `Bash` |
| Stage files | `Bash` |
| Create commit | `Bash` |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

On completion, send a message to the team lead with:
- Commit hash
- Files staged
- Commit message
- Phase completed

---

## Contract

<!-- Traces: ARCH-040 -->

```yaml
contract:
  outputs:
    - path: ".git/objects/<hash>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Files staged match phase scope"
      severity: error

    - id: "CHECK-002"
      description: "No secrets in staged files"
      severity: error

    - id: "CHECK-003"
      description: "Commit message follows conventional format"
      severity: error

    - id: "CHECK-004"
      description: "AI trailer included in commit message"
      severity: error

    - id: "CHECK-005"
      description: "Commit created successfully"
      severity: error

    - id: "CHECK-006"
      description: "No unrelated changes staged"
      severity: error

    - id: "CHECK-007"
      description: "No blanket lint suppressions added"
      severity: warning
```
