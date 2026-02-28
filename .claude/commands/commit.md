---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*), Bash(git diff:*), Bash(git log:*)
description: Create a git commit with proper formatting and AI-Used trailer
model: haiku
---

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`
- Recent commits: !`git log --oneline -10`

## Your task

Based on the above changes, create a single git commit. Follow these rules exactly:

### Staging
- Stage files relevant to the current change using specific file paths
- Do NOT use `git add -A` or `git add .`
- Do not stage unrelated files

### Message format

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

### Rules
1. **Trailer is `AI-Used: [claude]`** — NOT Co-Authored-By
2. **Never amend pushed commits** — check git status for "ahead of" first
3. **First line under 72 chars** — body wrapped at 72 chars
4. **Separate concerns** — don't mix functional changes with lint/style fixes
5. Use HEREDOC for the commit message:
   ```bash
   git commit -m "$(cat <<'EOF'
   <message here>

   AI-Used: [claude]
   EOF
   )"
   ```

Stage and create the commit. Do not use any other tools. Do not send any other text besides the tool calls.
