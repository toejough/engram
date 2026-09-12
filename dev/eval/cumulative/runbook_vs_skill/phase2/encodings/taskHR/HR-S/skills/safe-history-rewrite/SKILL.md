---
name: safe-history-rewrite
description: Use when rewriting git history with filter-branch or similar all-refs rewrites, especially before force-pushing with a backup branch present. Keywords — backup branch sweep, refs/original cleanup, force-with-lease, tracking refs refresh, pre-rewrite recovery.
---

# Safe Git History Rewrite and Force-Push

## Overview

History rewrites with `git filter-branch` (or similar tools that rewrite all refs) are powerful for removing sensitive data but dangerous if you lose the ability to recover the pre-rewrite state or if you force-push against stale tracking refs. This skill prevents three critical failures: the backup branch getting swept up and lost, force-push failing with outdated tracking refs, and sensitive data becoming unrecoverable.

## When to Use

Before rewriting git history with `git filter-branch` or any all-refs history rewrite, especially when:

- Keeping a safety backup branch for recovery
- Removing sensitive data (API keys, passwords, tokens)
- Force-pushing the result to a remote
- Multiple branches need to be rewritten at once

## Procedure

1. **Record the pre-rewrite tip outside the ref space.** Before starting any rewrite, capture the current tip of each branch you're about to rewrite as a bare SHA (not a branch name). Alternatively, rely on `refs/original/` (created by filter-branch) plus the reflog for recovery — but do NOT rely on a plain backup branch, since the rewrite can sweep it.

2. **Scope the rewrite to explicit refs, not `-- --all`.** A backup branch is not a safe exception to this rule — `git filter-branch ... -- --all` rewrites every ref it can see, backup branches included. Specify the exact refs you want rewritten (e.g., `-- main dev-feature`) or use `-- --all` only if you intend to rewrite the backup branch too (and understand it will be gone after).

3. **Run the history rewrite.** Execute your filter-branch command with the appropriate filters (e.g., `--index-filter`, `--tree-filter`, or `--subdirectory-filter`) to achieve the rewrite goal.

4. **Before force-pushing, refresh local tracking refs with `git fetch`.** The rewrite also rewrites your local remote-tracking refs (like `origin/main` in `.git/refs/remotes/origin/`). These now point to the old, pre-rewrite commits. A stale tracking ref will cause `--force-with-lease` to fail, thinking the remote has changed. Run `git fetch origin` to refresh `refs/remotes/origin/*` with the real remote state.

5. **Force-push with `--force-with-lease`.** Now that your tracking refs are fresh, use `--force-with-lease` (not bare `--force`) to push. The lease checks that your expected remote state (the freshly-fetched tracking ref) matches reality, protecting against a concurrent force-push from another user.

6. **If sensitive data required scrubbing, file a GitHub support request for a full server-side purge.** GitHub retains unreachable objects by commit SHA for some time. If you scrubbed truly sensitive data, file a support request asking GitHub to run server-side garbage collection and fully remove the unreachable commits. Public repos are cached and cloned; don't assume a rewrite alone removes the data from all copies.

## Done When

The pre-rewrite tip is safely recoverable (bare SHA / refs/original / reflog, not a branch `-- --all` could sweep), the force-with-lease push has succeeded against a freshly-fetched remote state, and if sensitive data needed scrubbing, a GitHub support purge request has been filed for unreachable objects.

## Common Mistakes

| Excuse under pressure | Reality |
|---|---|
| "I'll use `-- --all` but keep a backup branch — it won't get swept" | `-- --all` rewrites every ref it can see, backup branches included. After the rewrite, the backup branch points to rewritten commits and provides no recovery path. Move the backup outside the ref space first (e.g., `git bundle` or a local bare SHA record). |
| "Force-push will just work after the rewrite" | Filter-branch also rewrites your local tracking refs. If you force-push without running `git fetch` first, `--force-with-lease` will fail because your tracking ref is stale. Always `git fetch` before force-push. |
| "I'll use plain `--force` to be safe" | `--force` blindly overwrites without checking if the remote changed. Use `--force-with-lease` — it's safer, validating that the remote hasn't diverged since you last fetched. |
| "I'll clean up `refs/original/` later" | Clean them before running `gc` after the rewrite. If you force-push first, the key stays in reflog entries and `refs/original/` entries — not immediately leaked to the remote, but recoverable locally. Delete them early. |

## Red Flags — Stop and Follow the Procedure

- About to run `git filter-branch -- --all` and a backup branch exists in the same repo.
- About to force-push without running `git fetch` after the rewrite.
- Assuming a backup branch in the remote is a safe recovery point after a rewrite.
- Planning to use bare `--force` (instead of `--force-with-lease`) for the push.
- Sensitive data scrub finished but no GitHub support request filed yet.

## Why This Matters

History rewrites are irreversible at the remote once pushed. A backup branch swept away by `-- --all` is unrecoverable. Force-pushing with stale tracking refs can fail silently or silently overwrite a concurrent change. Sensitive data in unreachable commits can persist in forks, clones, and GitHub's cache. This procedure closes all three holes.
