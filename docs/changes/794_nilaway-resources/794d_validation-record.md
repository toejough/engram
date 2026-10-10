# 794d NilAway Resource Configuration - Validation Record

## 794d1 Provenance and Limits

This record transcribes command results observed during the original implementation in this
session, before the documentation remediation. It is historical evidence, not proof of a
test-first cycle or committed implementation-plan checkpoints. The commands ran in the
`fix/794-nilaway-resources` worktree on 2026-10-10. No unrestricted before-change measurement
was taken. Remediation validation must be recorded separately when it is performed.

## 794d2 Original Validation

- `targ test-dev`: passed all dev-package tests with the race detector.
- CLI smoke checks with a local NilAway stand-in: the check target received the include prefix
  and defaults; the fix target received the include prefix, `-fix`, and explicit
  `GOMEMLIMIT=4GiB GOGC=25 GOMAXPROCS=3`; an exit status of 7 propagated as a failed target.
- `/usr/bin/time -l targ check-nils-for-fail`: exit status 0; elapsed 11.90 seconds;
  reported maximum resident set size 2,650,865,664 bytes (approximately 2.47 GiB).
  This is the command's reported maximum RSS, not a hard limit or a per-phase profile.
- `targ check-full --dep-mode serial`: exit status 1; eight substantive gates passed
  (coverage, declaration order, fast lint, full lint, deadcode, thin API, NilAway, integration).
  The uncommitted-changes gate failed. This was not a fully passing check.

## 794d3 Remediation Status

The documentation and root instruction edits are committed. The original tests now have
scenario-prefixed names, and the additional command-boundary/acceptance tests in
[794c8 Implementation-Ordered Test Catalog](./794c_implementation-plan.md#794c8-implementation-ordered-test-catalog)
have not been implemented. Policy/routing entries have reached Red; other entries remain
Not started. No canonical promotion, merge, or push is claimed.

Local checkpoint commits are explicitly authorized. The updated sandbox excludes the original
sibling worktree, so validation and local commits continue in a local shared clone at
`/tmp/engram-794-resume` on the same branch name. The original worktree is not modified by this
clone. Canonical promotion and deletion of the active change directory belong to a later
authorized integration.

## 794d4 Regression Replay

With the original unrestricted arguments, no subprocess environment overrides, and original
shared target wiring restored, `targ test-dev` fails the three scenario-prefixed policy/routing
tests for missing include arguments, missing resource defaults, and missing configured target
routing. This is a regression replay observed during remediation, not a claim that the original
implementation was test-first. Other dev-package tests pass.

Commands in the temporary clone use `XDG_CACHE_HOME=/tmp/engram-794-cache` and
`GOCACHE=/tmp/engram-794-go-build` to keep build/cache writes inside the sandbox.
