# SOURCE_MATERIALS.md — Phase-2 Conversion-Parity Eval

## 1. Source A: /commit Skill

**Path:** `/Users/joe/repos/personal/engram/.claude/skills/commit.md`

**Diff result:** Files differ:
```
Files /Users/joe/repos/personal/engram/.claude/skills/commit.md 
  and /Users/joe/repos/personal/engram/.claude/commands/commit.md differ
```

**Skill file frontmatter (verbatim):**
```yaml
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
```

**Process steps (verbatim, ordered 1-7):**

```
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
```

**Message format block (verbatim):**
```
<type>(scope): <description>

<why we made this commit. what is the motivation, what problem does it solve, how does it fit into the bigger
picture? What decisions were made that led to this commit? were any lessons learned that future developers should be
aware of? This section should provide context and rationale for the change, not just a summary of what was done.>

AI-Used: [claude]
```

**Trailer rule (verbatim, from Rules section):**
```
1. **AI-Used trailer is `AI-Used: [claude]`** — NOT Co-Authored-By
```

**Step count:** N_A = 7

---

## 2. Source B: Vault Note 830

**Path:** `/Users/joe/.local/share/engram/vault/830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md`

**Frontmatter situation and done_when (verbatim):**
```yaml
type: runbook
tier: L2
situation: before shipping a narrowed or rewritten .gitignore pattern that makes some previously-ignored files trackable
done_when: the pattern's anchoring form has been confirmed correct via a scratch-repo git check-ignore check against representative (including nested) paths, and the exact set of newly-visible files has been enumerated and staged as an explicit path list rather than swept up by git add
luhmann: "830"
created: "2026-08-29"
source: migrated from 443.2026-07-24.gitignore-narrowing-needs-anchor-test-and-visible-set.md, engram#730
repo: vault
user: toejough@gmail.com
vault: personal
tags:
    - vocab/change-scope-control
    - vocab/go-code-conventions
    - vocab/corpus-mining
```

**Body numbered steps (verbatim):**
```
1. In a scratch repo, write the proposed replacement .gitignore pattern.
2. Run `git check-ignore -q <path>` against representative paths, including nested ones under subdirectories (e.g. internal/*/testdata/rapid/, test/testdata/rapid/), to confirm the pattern still matches at the intended depth.
3. If a pattern contains a middle slash (e.g. narrowing 'testdata/' to 'testdata/rapid/'), know it anchors to the .gitignore's own directory and silently stops matching nested paths; use a leading '**/' form instead (e.g. '**/testdata/rapid/') to keep matching at any depth. Never verify anchoring by reading the pattern alone.
4. In the real repo, write the proposed .gitignore and run `git status --porcelain` to enumerate the full set of newly-untracked/visible files it exposes, then restore.
5. Stage only the explicit enumerated paths from that list — never `git add -A` or `git add .`.
6. Confirm the staged set matches the enumerated list exactly by running `git diff --cached --name-only`.
```

**Step count:** N_B = 6

---

## 3. Covering Notes — Vault Grep Results

### Task A: commit-related grepping
**Grep pattern:** `AI-Used|conventional commit|commit message format|commit trailer`

**Removal list (notes whose content reveals the convention the source teaches):**
- `354.2026-07-22.subagent-briefs-state-commit-invariants-not-per-commit.md` — quotes "EVERY commit you create ends with AI-Used: [claude]" (reveals the trailer convention)
- `392.2026-07-23.route-dispatch-doc-review-gate.md` — states "Conventional Commits format, AI-Used trailer (no Co-Authored-By)" as repo rule (reveals the trailer and format conventions)
- `672.2026-07-29.route-dispatch-doc-review-gate.md` — states "Conventional Commits format, AI-Used trailer (no Co-Authored-By)" as repo rule (reveals the trailer and format conventions)

**Mentions only, keep (notes that mention commit terms in passing):**
- `459.2026-07-25.route-dispatch-plan-gate-review.md` — dispatch outcome fact; mentions commit trailer principles (Closes vs Refs) in context of a gate review finding, not the repo's own trailer convention

### Task B: gitignore-related grepping
**Grep pattern:** `gitignore|check-ignore|git check-ignore`

**Removal list (notes whose content reveals the gitignore procedure or related conventions):**
- `420.2026-07-24.folder-move-surface-gitignore-anchors-and-silent-optional-consumers.md` — teaches ".gitignore anchored glob patterns rooted at the old path silently stop matching after the move" (830's step-2 anchored-glob insight)
- `447.2026-07-24.framework-owned-testdata-not-dead-just-because-app-code-ignores-it.md` — records targ#33 ".gitignore narrowing + tracked fixtures" episode and framework-interaction analysis
- `448.2026-07-24.route-dispatch-design-fit-review.md` — records targ#33 ".gitignore narrowing + tracked fixtures" episode outcomes and learnings

**Note 830 status:** Removed from every arm except B-R (kept only in B-R background vault per spec).

**Mentions only, keep (notes that mention gitignore terms in passing):**
- `360.2026-07-22.scope-review-checks-complete-file-list-not-expected-files.md` — feedback on scope review; mentions .gitignore as backstop for cache dirs but focuses on commit file-list checking, not .gitignore narrowing

---

## 3. Source Route: Route Skill

**Path:** `/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/agent-instructions/skills/route/SKILL.md`

**Skill file frontmatter (verbatim):**
```yaml
---
name: route
description: >
  Use when you are about to dispatch a subagent and must decide its agent type, model, and
  effort level. Triggers on any delegation decision, and when you recognize a unit is too large
  for one focused agent and needs decomposition before dispatch.
---
```

**Key sections (verbatim section headings):**
- `# Route — default to the cheapest tier, escalate on evidence, remember what works`
- `## Orchestration work vs object-level work` (defines you do / you delegate boundary)
- `## How to pick a tier` (4-step procedure: recall first, default cheapest, escalate on spec-first, memory discounts tier)
- `## The handoff is the unlock` (exact files, acceptance checks, do-NOT-touch, recall-first, LESSONS: contract)
- `## Record every dispatch (the evidence)` (work-kind, tier, model, why, outcome, escalation, duration, cost; mini-report table structure)
- `### The structured write (one evidence note + one aggregate update per dispatch)` (evidence note handoff to write-memory, aggregate amend-or-create via engram commands)
- `### Count as audit (never on the read path)` (engram count validation procedures; drowning audit)
- `## The loop that improves the rubric` (6-step feedback cycle)
- `## Cold-start priors (unproven — evidence overwrites these)` (3 priors table)
- `## Two rules every dispatch obeys` (subagent recalls first, decompose before dispatch)
- `## Red flags — STOP and re-read` (12-row stop-points table: "Sign you're off | What to do")

**Content size estimate:** ~7500 bytes (286 lines × ~26 chars/line average)

**Step count:** Not discrete steps like A/B; structured as 4 major procedures (pick tier, handoff, record, loop) + supporting tables + 2 mandatory rules + 12 red-flag stop-points.

---

## 4. Vault Size

**Vault directory size:** 23M

**Note count:** 883 notes total

**Embed status head (first 10 lines):**
```
total:           883
with-embeddings: 883
without:         0
stale:           0
incompatible:    0
broken:          0
```
