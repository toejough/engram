---
name: commit
description: Create a well-formatted git commit following project conventions
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
---

# Commit Skill

Create a well-formatted git commit following project conventions.

## Purpose

Stage and commit changes with a properly formatted message. Receives context about what phase is being committed (red/green/refactor or other) to format the appropriate message.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Phase being committed (tdd-red, tdd-green, tdd-refactor, or other)
- Task ID
- Summary of changes from the preceding phase
- Scope (component/package affected)

## Process

1. **Check VCS type**
   - Look for `.jj` directory or check session env info
   - If jj repo, use `jj` commands, not `git`
   - This check is non-negotiable

2. **Check state**
   ```bash
   git status
   git diff --staged
   git diff
   ```
   - If nothing to commit, report and stop
   - Note what's staged vs unstaged

3. **Review recent commits for style**
   ```bash
   git log --oneline -5
   ```

4. **Stage changes**
   - Stage files relevant to the current phase
   - Prefer specific file paths over `git add -A`
   - Do not stage unrelated files

5. **Compose message based on phase**

   **TDD Red:**
   ```
   test(scope): add tests for <feature>

   Tests for <what> - currently failing.

   AI-Used: [claude]
   ```

   **TDD Green:**
   ```
   feat(scope): implement <feature>

   <Brief description of implementation approach>

   AI-Used: [claude]
   ```

   **TDD Refactor:**
   ```
   refactor(scope): <what was cleaned up>

   AI-Used: [claude]
   ```

   **Other phases:**
   ```
   <type>(scope): <description>

   <Body if needed>

   AI-Used: [claude]
   ```

6. **Commit**
   ```bash
   git commit -m "$(cat <<'EOF'
   <message here>

   AI-Used: [claude]
   EOF
   )"
   ```

7. **Verify**
   ```bash
   git log -1
   git status
   ```

## Rules

1. **AI-Used trailer is `AI-Used: [claude]`** - NOT Co-Authored-By. This is non-negotiable.
2. **Never amend pushed commits** - Check `git status` for "ahead of" first. If not ahead, create new commit.
3. **Separate concerns** - Don't mix functional changes with lint/style fixes. Each logical change gets its own commit.
4. **First line under 72 chars** - Body wrapped at 72 chars.
5. **Stage specific files** - Don't use `git add -A` or `git add .`.
6. **Never use dangerous commands** - No `git checkout -- .`, `git restore .`, `git reset --hard`.

## Pre-amend Check

Before ANY amend:
```bash
git status
```
Look for "Your branch is ahead of". If NOT ahead, the commit is pushed - CREATE A NEW COMMIT instead.

## Structured Result

When the commit is complete, produce:

```
Status: success | failure | blocked
Summary: Committed [phase] changes for [task].
Commit hash: [hash]
Commit message: [first line]
Files committed: [list]
Traceability: [TASK ID]
Context for next phase: [commit hash, any notes]
```

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = []

[[decisions]]
context = "Process choice"
choice = "Follow established convention"
reason = "Consistency with existing patterns"
alternatives = []

[[learnings]]
content = "Captured from execution"
```
