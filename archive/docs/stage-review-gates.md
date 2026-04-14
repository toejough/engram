# Stage Review Gates

After completing a stage, run through these gates before declaring the stage done. No deferrals.

## Gate 1: Quality Checks

Run `targ check-full`. All checks must pass (except `check-uncommitted` during active work).

- [ ] `check-coverage-for-fail` — all new functions at or above threshold
- [ ] `reorder-decls-check` — declarations in canonical order
- [ ] `lint-fast` — no lint issues
- [ ] `lint-full` — no lint issues (full report)
- [ ] `deadcode` — no dead code
- [ ] `check-thin-api` — public API is thin wrappers
- [ ] `check-nils-for-fail` — no nilaway issues

## Gate 2: E2E Testing

Build the binary. Run each new command manually.

- [ ] Binary builds with `go build ./cmd/engram/`
- [ ] Each new command prints correct validation errors for missing/invalid args
- [ ] Each new command produces correct error messages when the server is unavailable
- [ ] Happy path works against a real or fake server — output matches expected format

## Gate 3: Form and Function

The stage delivers what the spec says it should. No partial implementations, no stubs, no TODOs.

- [ ] Every command/feature listed in the plan is implemented
- [ ] Every feature works end-to-end (not just unit-tested)
- [ ] No `// TODO` or `// FIXME` in new code
- [ ] No deferred functionality that was supposed to land in this stage

## Gate 4: Report

Write a completion report covering:

- What was delivered (commands, packages, patterns)
- Architecture summary (DI boundaries, test strategy)
- Quality gate results
- E2E test results
- Commit count and branch

## Workflow

After passing all gates:

1. Commit the stage review gate results
2. Move to the next stage: read `docs/exec-planning.md`, write the plan, execute with subagents, run through these gates
3. Repeat until the design spec is fulfilled
