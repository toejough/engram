---
name: gitignore-narrowing
description: Use when narrowing or rewriting a .gitignore pattern so that some previously-ignored files become trackable, before staging or committing that change. Symptoms/keywords — gitignore anchoring, git check-ignore, middle-slash patterns, testdata/ narrowing, newly-visible or newly-untracked files, git add -A, git add .
---

# Gitignore Narrowing: Anchor Verification and Explicit Visible-Set Staging

## Overview

Narrowing a `.gitignore` pattern (e.g. `testdata/` -> `testdata/rapid/`) has two
independent failure modes: the new pattern can silently stop matching nested
paths (a gitignore anchoring gotcha), and the files it newly exposes can get
swept into a commit alongside unrelated untracked work if staged with a bare
`git add -A`/`git add .`. This skill is the verification-and-staging procedure
that catches both before anything is committed.

## When to Use

Before shipping a narrowed or rewritten `.gitignore` pattern that makes some
previously-ignored files trackable. Triggers include:

- Tightening a broad ignore rule (e.g. `testdata/` -> `testdata/rapid/`) so
  only a subset stays ignored.
- Any `.gitignore` edit expected to newly expose files in nested
  subdirectories, not just at the repo root.
- About to stage the result of a `.gitignore` narrowing with `git add`.

## Procedure

1. In a scratch repo, write the proposed replacement `.gitignore` pattern.
2. Run `git check-ignore -q <path>` against representative paths, including
   nested ones under subdirectories (e.g. `internal/*/testdata/rapid/`,
   `test/testdata/rapid/`), to confirm the pattern still matches at the
   intended depth.
3. If a pattern contains a middle slash (e.g. narrowing `testdata/` to
   `testdata/rapid/`), know it anchors to the `.gitignore`'s own directory
   and silently stops matching nested paths; use a leading `**/` form
   instead (e.g. `**/testdata/rapid/`) to keep matching at any depth. Never
   verify anchoring by reading the pattern alone.
4. In the real repo, write the proposed `.gitignore` and run
   `git status --porcelain` to enumerate the full set of
   newly-untracked/visible files it exposes, then restore.
5. Stage only the explicit enumerated paths from that list — never
   `git add -A` or `git add .`.
6. Confirm the staged set matches the enumerated list exactly by running
   `git diff --cached --name-only`.

## Done When

The pattern's anchoring form has been confirmed correct via a scratch-repo
`git check-ignore` check against representative (including nested) paths,
and the exact set of newly-visible files has been enumerated and staged as
an explicit path list rather than swept up by `git add`.

## Common Mistakes

| Excuse under pressure | Reality |
|---|---|
| "This exact pattern worked on another repo, it's proven — no need to re-derive or re-verify it" | Anchoring depends on where the `.gitignore` lives and the directory structure of *this* repo. A pattern that matched elsewhere can still fail to match nested paths here. Re-verify with `git check-ignore -q` every time, even a "proven" pattern. |
| "We're out of time, just `git add -A` so nothing gets missed" | `git add -A` sweeps in whatever else is untracked in the repo, not just the files the `.gitignore` change exposed. Enumerate the visible set from `git status --porcelain` and stage that explicit list instead. |
| "Skip the extra checks, just ship it" | Skipping `git check-ignore` and the final `git diff --cached --name-only` check is exactly how a silently-broken anchor or an accidentally-staged unrelated file reaches a commit. |

## Red Flags — Stop and Follow the Procedure

- About to run `git add -A` or `git add .` right after a `.gitignore` change.
- Trusting a narrowed pattern's anchoring "by reading it" instead of running
  `git check-ignore -q` against a nested path.
- Reusing a pattern from another repo or an earlier change without
  re-verifying it against *this* repo's directory structure.
- Committing before running `git diff --cached --name-only` to confirm the
  staged set matches the enumerated visible-set list exactly.
