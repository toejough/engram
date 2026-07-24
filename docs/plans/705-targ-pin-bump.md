# Plan: #705 — bump targ pin to 822f27b to green check-full's coverage leg

## Ask (verbatim)

"handle 705" — issue #705: *Bump targ pin past 822f27b to green check-full (coverage-leg 30s timeout fix).*

## Verified pre-flight facts

- `go.mod:9` pins `github.com/toejough/targ v0.0.0-20260723030055-c1200afc8001`.
- `git merge-base --is-ancestor 822f27b c1200afc8001` in the targ repo (post-fetch) → NOT-ANCESTOR: the fix
  commit `822f27b` ("fix(dev): raise the fail-fast coverage leg's go-test timeout to 10m") postdates the
  current pin. The bump is needed.
- Note 365 (vault): targ gate behavior rides the consuming repo's go.mod pin — no binary reinstall needed.
- Doc-surface grep for `c1200af`/`822f27b`/`timeout=30s`/`30s timeout` over docs/, .superpowers/, CLAUDE.md,
  README.md: zero hits. No doc scrub needed.

## Decision within the issue's latitude

The issue allows `@822f27b` or `@latest`. Pin **exactly 822f27b** — minimal change scoped to the ask;
targ main's newer #26 feature work (bounded parallel fan-out) rides a future bump on its own merits.

## Tasks

1. In `/Users/joe/repos/personal/engram`: `go get github.com/toejough/targ@822f27b && go mod tidy`.
   Confirm `go.mod` diff touches only the targ pin line (+ go.sum).
2. RED/GREEN framing: the failing check IS the repeatable test. RED is pre-established by #705's evidence
   (toejough/targ#25: coverage leg killed at 30s, 3/3 reproducible under full 8-way load; not re-reproduced
   here since the flake is load-dependent and the fix evidence chain is probe-validated upstream).
   GREEN: `targ check-full` from the repo root — expected all 8 gates green, including
   `check-coverage-for-fail` ("Coverage OK").
3. Commit go.mod/go.sum (conventional message, `AI-Used: [claude]` trailer) BEFORE check-full
   (check-uncommitted gate requires a clean tree).
4. Close #705 with the evidence: pin diff + full-gate green output summary (note 334: every check green or
   explicitly OK'd).
5. Delete this plan file in step 6 (planning artifact).

## Risks

- Newer gate behavior in 822f27b's tree could surface new findings; if check-full fails on anything, STOP,
  collect the full failure list, and report (no whack-a-mole, no silent punts).
- Run all targ commands from the engram checkout only (note 359: wrong-cwd targ runs sync other checkouts' pins).
