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
are implemented. Policy/routing and command-boundary entries are Done after committed
checkpoints and passing applicable suites. Final validation remains in progress.
No canonical promotion, merge, or push is claimed.

Local checkpoint commits are explicitly authorized. An earlier sandbox excluded the original
sibling worktree, so continuation checkpoints were created in a shared clone at
`/tmp/engram-794-resume`. After permissions changed, those reviewed commits were transferred
back to the original worktree with an ff-only update. Canonical promotion and deletion of the
active change directory belong to a later authorized integration.

## 794d4 Regression Replay

With the original unrestricted arguments, no subprocess environment overrides, and original
shared target wiring restored, `targ test-dev` fails the three scenario-prefixed policy/routing
tests for missing include arguments, missing resource defaults, and missing configured target
routing. This is a regression replay observed during remediation, not a claim that the original
implementation was test-first. Other dev-package tests pass.

Commands in the temporary clone use `XDG_CACHE_HOME=/tmp/engram-794-cache` and
`GOCACHE=/tmp/engram-794-go-build` to keep build/cache writes inside the sandbox.

## 794d5 Policy Green Checkpoint

Restoring the Engram include prefix, resource defaults, and in-place target replacement makes
all three policy/routing scenario tests pass. `targ test-dev` passes the entire dev suite with
the race detector. The check retains both shared target objects and the existing aggregate
entry points; explicit values, including empty values, are preserved by the policy.

## 794d6 Policy Review Checkpoints

The complete applicable dev suite passes. A clean-checkout `targ check-full --dep-mode serial`
also passes lint, ordering, deadcode, thin API, NilAway, integration, and clean-tree gates.
Its coverage/test gate fails because the pre-existing untagged HTTP-server test cannot bind
`[::1]:0` inside this sandbox. Repository-wide validation and delivery remain incomplete.
No unrelated test is changed to conceal this environment restriction.

Complexity review: the policy has two pure operations and no extra configuration layer.
Consistency review: check and fix share the same policy and existing target identities.
Boundary review: process environment access and command execution remain at the dev edge;
policy inputs are injected for pure tests. Command orchestration tests are the next work package.
Performance review: no production changes follow the successful scoped real-analyzer run;
no new performance guarantee is inferred. These review checkpoints do not require code changes.

## 794d7 Command-boundary RED checkpoint

`targ test-dev` and `targ test-dev-integration` fail as expected when the injected
executor receives no environment, its error is discarded, and the adapter ignores
settings. Successful invocation, failure propagation, isolation, default and
partial-override scenarios are Red. The real analyzer scope characterization passes:
explicitly selecting both fixture packages exposes the dependency diagnostic without
the include flag, and removes only that diagnostic with the include flag. This is
Green-single characterization of existing policy, not claimed historical TDD.

Policy work is Done. Boundary implementation and final validation remain incomplete.

## 794d8 Command-boundary GREEN checkpoint

`targ test-dev` and `targ test-dev-integration` both pass with race detection.
The configured invocation forwards resolved settings and the executor's exact error.
The adapter applies settings only to its child command. The real-analyzer fixture
verifies first-party diagnostics remain while selected dependency diagnostics vanish.
The integration target explicitly includes the targ and integration build tags.

## 794d9 Boundary Refactor Review

Complexity: the added seam injects only lookup and one executor; no configuration framework.
Consistency: declarations follow repository ordering; check/fix retain one shared policy.
Boundaries: fixtures now require all child resource variables to be present, even when empty,
and verify both absent and present parent variables remain unchanged. Interaction tests use
imptest; real subprocess and analyzer contracts stay integration-tagged.
Performance: no additional analyzer invocation in production, no retry, and no process-wide
environment mutation. The fixture analyzes only two tiny packages. No hard RSS guarantee.

After these refinements, `targ reorder-decls-check`, `targ test-dev`, and
`targ test-dev-integration` pass. The prior full check passed seven gates, failed declaration
ordering (now fixed) and the existing sandbox-denied loopback test. Final validation stays open.

## 794d10 Final Local Validation and Handoff

Independent reviewer ACKs `c7f7d1c5` for implementation and plan reconciliation, not delivery.
The final clean-checkpoint `targ check-full --dep-mode serial` reports eight passing gates:
clean tree, declaration ordering, fast lint, full lint, deadcode, thin API, configured NilAway
(9.658 seconds), and integration. Only `check-coverage-for-fail` fails because existing
`Test792a20e1c_ServeAndFetch_LearnRoundTrip` cannot bind `[::1]:0` in this sandbox.
The two explicit dev suites also pass with race detection. `git diff --check` passes.

The original sibling worktree remains outside writable roots. A verified Git bundle at
`/tmp/engram-794-checkpoints.bundle` preserves the continuation commits, with `e38a2f17`
as its prerequisite. Transfer must reconcile the original worktree's uncommitted intermediate
files; do not blindly overwrite or discard them. The `gh issue comment` update could not
reach `api.github.com`; no successful tracker update is claimed. Final validation, original
worktree transfer, authorized integration, and canonical promotion remain open.

## 794d11 Resumed Original-Worktree Validation

The original worktree was fast-forwarded from `e38a2f17` to `5c47c4e0`. Its three
intermediate regression-replay files were preserved in the named stash
`794 pre-transfer regression replay preserved`, not discarded. No unrelated edits were found.

With auto-reviewed escalation allowing normal caches and localhost listeners,
`targ test-dev`, `targ test-dev-integration`, and `targ check-full --dep-mode serial`
all pass in the original worktree. The full check includes coverage, clean tree,
declaration ordering, both lint gates, deadcode, thin API, configured NilAway, and integration.
The formerly blocked HTTP-server test passes without changing its code.

The existing independent implementation ACK remains applicable: transferred production and
tests are unchanged. Every catalog scenario has an ID-prefixed test; direct policy, wiring,
interaction, subprocess, and real-analyzer contracts cover their owning requirements.
The tracker progress update succeeded. Earlier filesystem, localhost, and network blockers
are resolved. Integration into main, canonical promotion, and cleanup remain unperformed.
