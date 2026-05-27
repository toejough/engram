# Fix #640 — Recall SKILL References to `engram recall`

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (or just execute inline — this is a four-step fix). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Resolve #640. The deployed `~/.claude/skills/recall/SKILL.md` references an `engram recall` subcommand and `--follow` / `--already-read` / `--recent` flags that the binary does not expose. A new agent following the deployed SKILL hits "Unknown command: recall" on the very first cascade step.

**Architecture:** The source `skills/recall/SKILL.md` in the repo has already been rewritten to drive recall via a single `engram query --phrase ...` call (the F6+F9.1 payload). The drift is purely in the deployed copy — `engram update` is the propagation mechanism but it hasn't been run since the rewrite. The fix is to run it, plus tidy one stray reference in `docs/triage.md`.

**Tech Stack:** `engram update` for SKILL propagation; markdown edit for the triage entry; `gh issue close` for closure.

---

## Pre-work findings (out of scope, intentionally not touched)

- `skills/recall/tests/baseline-{multi-query,judgement-and-cascade,RED-results,GREEN-results}.md` — frozen RED/GREEN behavioral test records; references to `engram recall` are intentional historical record.
- `docs/superpowers/research/*.md` and `docs/superpowers/specs/*.md` — pre-v2 design snapshots; revising them would be revisionist.
- The "Add `engram recall` as a thin wrapper" option from the issue — the user picked the SKILL-side fix, not this one.

---

## File Structure

- Modify: `docs/triage.md` — entry "12. `engram recall` 'no-arg' behavior overlap with `--recent`" is obsolete; the no-arg / `--recent` distinction it discusses no longer exists in the SKILL (which now runs one `engram query --phrase` call). Either remove the entry or annotate it as resolved-by-v2-rewrite. Decision: remove — the issue tracker is for live triage items; v2 closed this concern.
- Run: `engram update` to propagate the source SKILL to `~/.claude/skills/recall/SKILL.md` (and any OpenCode harness directory if installed).
- Verify: `diff skills/recall/SKILL.md ~/.claude/skills/recall/SKILL.md` returns empty.
- Close: `gh issue close 640`.

---

## Task 1: Remove obsolete triage entry

**Files:** Modify `docs/triage.md` (entry "### 12. `engram recall` 'no-arg' behavior overlap with `--recent`").

- [ ] **Step 1: Read the section** and confirm scope.

```bash
sed -n '125,150p' docs/triage.md
```

- [ ] **Step 2: Delete the entry**. The whole section (heading + body bullets) goes. If subsequent triage entries are numbered, renumber them — or leave the numbering gap (triage docs commonly have gaps).

- [ ] **Step 3: Verify the file still makes sense** (read the surrounding sections).

- [ ] **Step 4: Commit**.

```bash
git add docs/triage.md
git commit -m "$(cat <<'EOF'
docs(triage): drop entry 12 (engram recall no-arg overlap) — resolved by v2 rewrite

The /recall SKILL no longer has a no-arg recap mode separate from
--recent; v2 collapsed the cascade into a single engram query --phrase
call. The triage concern no longer applies.

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Propagate the source SKILL via `engram update`

**Files:** None modified directly in repo; the side effect is `~/.claude/skills/recall/SKILL.md` (and OpenCode equivalent if present) being overwritten with the source.

- [ ] **Step 1: Confirm source SKILL is current** (no `engram recall` references).

```bash
grep -c "engram recall" skills/recall/SKILL.md
```
Expected: `0`.

- [ ] **Step 2: Show what `engram update --dry-run` would do**.

```bash
engram update --dry-run 2>&1 | head -30
```

- [ ] **Step 3: Run `engram update`**.

```bash
engram update 2>&1 | tail -10
```

- [ ] **Step 4: Verify deployed copy now matches source**.

```bash
diff skills/recall/SKILL.md ~/.claude/skills/recall/SKILL.md
grep -c "engram recall" ~/.claude/skills/recall/SKILL.md
```
Expected: empty diff; `0` matches.

- [ ] **Step 5: Check OpenCode harness if installed**.

```bash
test -f ~/.config/opencode/skills/recall/SKILL.md && \
  diff skills/recall/SKILL.md ~/.config/opencode/skills/recall/SKILL.md ; \
  echo "(no opencode harness detected, OK)"
```

---

## Task 3: Close the issue

- [ ] **Step 1: Run `gh issue close 640`** with a comment naming the root cause (source was already correct, deployed copy hadn't been refreshed) and the fix (re-ran `engram update`; dropped one obsolete triage entry).

- [ ] **Step 2: Confirm closure**.

```bash
gh issue view 640 --json state,closedAt | head -3
```

---

## Task 4: Delete this plan doc

- [ ] **Step 1**.

```bash
git rm docs/superpowers/plans/2026-05-27-fix-640-recall-skill-drift.md
git commit -m "chore: remove completed plan doc for #640

AI-Used: [claude]"
```

---

## Self-Review

**1. Spec coverage** — user said "fix 640. Remove references to the recall command in the recall skill":
- ✓ Active SKILL references removed (Task 2 propagates the already-rewritten source).
- ✓ Stray active-doc reference removed (Task 1, triage entry).
- ✓ Issue closed (Task 3).
- Intentionally NOT touched: frozen test records and pre-v2 research/spec docs (historical snapshots).

**2. Placeholder scan** — none.

**3. Type consistency** — N/A (no code changes).

**4. Surprises to flag to user** — yes: the source SKILL was already correct, so the "fix" is propagation, not edit. Worth surfacing in the closure comment and in the user-facing summary.
