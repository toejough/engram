# Please-skill generalization + stale-learn fix

> **For agentic workers:** executed inline via the please workflow; skill edit follows
> writing-skills RED→GREEN.

**Goal:** `skills/please/SKILL.md` stops referencing the retired transcript-marker behavior of
`/learn` and stops naming concrete optional skills (`superpowers:*`, `/commit`) that users may
not have installed.

**Scope:** `skills/please/SKILL.md` only. `commands/please.md` is a thin pointer — unaffected.
`/recall` and `/learn` remain named: they ship with engram itself and the workflow depends on
them specifically.

### Edits

1. **Overview (line ~16):** replace the named-skill enumeration ("brainstorming, writing-plans,
   executing-plans, test-driven-development, verification-before-completion, `/commit`") with
   capability language: the workflow uses `/recall` and `/learn` (engram's own) plus, for each
   non-trivial step, *a relevant skill if one is installed* (planning, TDD, verification,
   committing) — falling back to doing the step directly when none is.
2. **Step 1 (line ~28):** drop "advances the transcript marker and clears the slate" — describe
   the current learn: sweeps raw conversation/doc memory into the chunk index and crystallizes
   any explicit lessons.
3. **Step 3 (line ~34):** "Use `superpowers:writing-plans`" → "use a skill geared toward writing
   plans, if one is available"; "prefer the `/commit` skill" → "use a commit-focused skill if
   available, otherwise commit directly".
4. **Step 4 (lines ~35–39):** generalize the three superpowers references; keep RED/GREEN/REFACTOR
   discipline mandatory regardless of skill availability ("with or without a TDD skill installed").
5. **Step 6 (line ~41):** same commit generalization as edit 3.
6. **Red flags (lines ~60–61):** generalize the two superpowers citations to the discipline names.

### Test (writing-skills)

- **RED:** subagent follows the current skill text for a toy ask in an environment WITHOUT
  superpowers/commit skills; document that it tries to invoke the named skills / repeats the
  transcript-marker claim.
- **GREEN:** same scenario against the edited skill; agent applies the disciplines directly,
  never cites unavailable plugin names, describes learn correctly.

### Verification

GREEN subagent run passes the checks above; deployed copy refreshed via `engram update`; diff
shows no remaining `superpowers:`/`transcript marker` strings in the skill.
