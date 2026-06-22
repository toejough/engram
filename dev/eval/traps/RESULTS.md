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
