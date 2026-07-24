# Plan: #705 — bump targ pin to 822f27b to green check-full's coverage leg

## Ask (verbatim)

"handle 705" — issue #705: *Bump targ pin past 822f27b to green check-full (coverage-leg 30s timeout fix).*

## Verified pre-flight facts

- `go.mod:9` pins `github.com/toejough/targ v0.0.0-20260723030055-c1200afc8001`.
- `git merge-base --is-ancestor 822f27b c1200afc8001` in the targ repo (post-fetch) → NOT-ANCESTOR: the fix
  commit `822f27b` ("fix(dev): raise the fail-fast coverage leg's go-test timeout to 10m") postdates the
  current pin. The bump is needed.
- Note 365 (vault): targ gate behavior rides the consuming repo's go.mod pin — no binary reinstall needed.
- Doc-surface grep for `c1200af`/`822f27b`/`timeout=30s`/`30s timeout` over `docs/` (excluding this plan
  file), the `.superpowers/` ledger directory, `CLAUDE.md`, and `README.md`: zero hits. No doc scrub needed.

## Decisions

- **Pin exactly 822f27b** (the issue allows `@822f27b` or `@latest`): minimal change scoped to the ask;
  targ main's newer #26 feature work (bounded parallel fan-out) rides a future bump on its own merits.
- **No local RED re-reproduction.** The failure is pre-established upstream (toejough/targ#25: coverage leg
  killed at the 30s budget, 3/3 reproducible under full 8-way load) and the fix is probe-validated there
  3/3. The flake is load-dependent, so a local re-repro adds cost without evidence value; `targ check-full`
  green is the GREEN measurement.

## Tasks

1. In `/Users/joe/repos/personal/engram`: `go get github.com/toejough/targ@822f27b && go mod tidy`.
   Confirm the diff touches only the targ pin line in `go.mod` (+ `go.sum`).
2. Commit `go.mod`/`go.sum` (conventional message, `AI-Used: [claude]` trailer). Commit comes before the
   gate run because `check-full`'s `check-uncommitted` leg requires a clean tree.
3. Run `targ check-full` from the repo root. Expected: all 8 gates green, including
   `check-coverage-for-fail` ("Coverage OK"). Capture the output summary as closure evidence.
4. Close #705 citing the pin diff and the full-gate green summary. All 8 gates must pass; any failure stops
   the close — collect the complete failure list and report it to Joe for disposition (no silent punts,
   no whack-a-mole).
5. Delete this plan file (planning artifact) in the workflow's completion step, as part of the final commit.

## Risks

- Newer gate behavior in 822f27b's tree could surface new findings; per Task 4, any red gate stops the close.
- Run all targ commands from the engram checkout only (note 359: wrong-cwd targ runs sync other checkouts' pins).
