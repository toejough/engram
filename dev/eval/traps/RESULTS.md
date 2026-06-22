# Opus-trap cold-confirmation results

Cold opus (`claude-opus-4-8`, no memory / no CLAUDE.md / clean cfg), N=5 trials each, deterministic
check per trap. A trap is **CONFIRMED** when cold opus produces the natural-but-locally-wrong form
in ≥4/5 trials. Each prompt carries NO hint of the convention.

## Confirmed reproducible (5/5 unless noted) — the warm exercise set

| trap | cold opus does (verified in generated code) | the local convention it violates | check |
|---|---|---|---|
| `req-with-context` | `http.Get(url)` | use `http.NewRequestWithContext` | grep |
| `nocolor` | emits `\033[32m` raw | gate ANSI on `NO_COLOR` + TTY | grep |
| `t-parallel` | zero `t.Parallel()` in tests | `t.Parallel()` on every test/subtest | grep |
| `nil-guard-split` | `bytes.SplitN(b,…)[0]` unguarded | nil/len guard before indexing a split (nilaway) | grep |
| `wrapped-error` | bare `return nil, err` | wrap with `fmt.Errorf("…: %w", err)` | grep |

## Saturated — DROPPED (opus already applies the convention cold; weak exercises)

`slices-backward` (5/5 applied), `crypto-rand` (4/5 applied), `named-const` (4/5), `sentinel-err`
(3/5), `make-cap` (5/5 applied), `table-test-loop` (5/5 applied).

This is the saturation gate working as designed: ~half the candidates are conventions opus applies
unprompted, so memory has nothing to add — correctly invalidated by the cheap cold trial before any
warm spend.

## Cost

Cold confirmation: 55 opus trials total, **$6.94** (run1 40 trials $5.07 + run2 15 trials $1.87).
~$0.13/trial. The next step (not done here) is the WARM side: same prompts with the matching note
in the vault, recall surfacing it — the deterministic check should flip trap→applied. The cold→warm
flip on these 5 is the opus-strength memory payoff the CRUD oracle could not measure.

---

# Behavioral / conceptual traps — NONE reproduce (the key finding)

Built 4 behavioral traps that recreate a *condition* and inspect what opus DID (transcript / git /
LLM-judge), then ran cold opus n=5 at the **5/5 (100%) bar**. All four came back **0/5 — opus did
the right thing every trial** (verdicts verified genuine, not lenient checks):

| trap | condition | opus's cold behavior | verdict |
|---|---|---|---|
| `verify-before-done` | buggy `Add`, "fix it, tell me when done" (no verify cue) | ran `go run calc.go` unprompted, every trial | 0/5 — verifies |
| `commit-core-artifact` | "implement Reverse + note in README, commit" | committed strutil.go (not just docs), every trial | 0/5 — commits the code |
| `over-engineer` | "write IsPrime + main" | 25-line trial-division loop, no sieve/goroutines | 0/5 — stays simple |
| `scope-discipline` | "add Slugify, that's the ask" | added only the function, no extras | 0/5 — stays in scope |

## Why behavioral traps don't reproduce cheaply — and what it means

The behavioral corrections in the catalog happened in **rich session context**: long multi-step
runs, time/economic pressure, competing goals, complex features where green unit tests masked a real
failure, or a sprawling session where the core artifact got lost. A clean 5-turn toy task **strips
exactly the context that produces the failure** — so cold opus, unpressured and unconfused, does the
right thing.

The tactical traps reproduce 5/5 for the mirror-image reason: they are **context-free**. Opus's
idiomatic-Go default (`http.Get`, raw ANSI, no `t.Parallel`, bare `return err`) fires regardless of
context, because it's a token-level habit, not a judgment made under pressure.

**Conclusion:** cheap, isolated, reproducible traps are *necessarily* tactical. Behavioral/conceptual
lessons cannot be validated by cheap exercises — reproducing them requires an expensive, complex,
multi-step scenario (the very thing the "fast and cheap" goal was avoiding). This is a real limit on
what the saturation-breaking eval can measure at opus strength, not a harness defect.

Behavioral cold-confirm spend: 20 opus trials + judge calls, $2.34 (all would early-stop at trial 1
under the 5/5 policy).
