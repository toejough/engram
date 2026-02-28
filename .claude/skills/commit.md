---
name: commit
description: |
  Core: Stages specific files and creates conventional commits with AI-Used trailer and proper message formatting.
  Triggers: commit my changes, create a commit, git commit, save my work, commit these files.
  Domains: git, version-control, commits, conventional-commits, VCS.
  Anti-patterns: NOT for pushing to remote, NOT for amending commits, NOT for destructive git operations like reset --hard.
context: inherit
model: haiku
user-invocable: true
---

# Commit Skill

Stage and commit changes with a properly formatted message.

## Process

1. **Check VCS type**
   - Look for `.jj` directory — if jj repo, use `jj` commands, not `git`

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
   - Stage files relevant to the current change
   - Prefer specific file paths over `git add -A`
   - Do not stage unrelated files

5. **Compose message**

   Format:

   ```
   <type>(scope): <description>

   <why we made this commit. what is the motivation, what problem does it solve, how does it fit into the bigger
   picture? What decisions were made that led to this commit? were any lessons learned that future developers should be
   aware of? This section should provide context and rationale for the change, not just a summary of what was done.>

   AI-Used: [claude]
   ```

   Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

   TDD phases:
   | Phase | Format |
   |-------|--------|
   | TDD Red | `test(scope): add tests for <feature>` |
   | TDD Green | `feat(scope): implement <feature>` |
   | TDD Refactor | `refactor(scope): <cleanup>` |

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

1. **AI-Used trailer is `AI-Used: [claude]`** — NOT Co-Authored-By
2. **Never amend pushed commits** — check `git status` for "ahead of" first
3. **Separate concerns** — don't mix functional changes with lint/style fixes
4. **First line under 72 chars** — body wrapped at 72 chars
5. **Stage specific files** — don't use `git add -A` or `git add .`
6. **Never use dangerous commands** — no `git checkout -- .`, `git restore .`, `git reset --hard`
