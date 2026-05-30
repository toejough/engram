# Engram memory validation program — store / retrieve / worth-it

Date: 2026-05-30. Designed via a fan-out/critique/synthesize workflow (6
design variants × adversarial validity critique → integrated program),
seeded with the cold/warm + capture + matrix work and the validity gaps
those left open (no memory-off control, circularity, n=1, accuracy =
planted spec, cost-to-fixed-target never measured, transfer untested).

Answers three questions Joe posed: **can we STORE the right information?
can we RETRIEVE it helpfully? is it EFFICIENT enough to be worth the
effort + complexity?**

## The keystone

Every fatal/near-fatal flaw across the six designs is **the same defect
in three costumes**: the "lesson" being tested is either (a) documented
rulebook already in always-loaded `CLAUDE.md` / `.claude/rules/go.md` /
scenario `ExpectedVault` (so STORE measures transcription, not
foresight), (b) a verbatim fix the note *relays* rather than a principle
that *transfers* (so RETRIEVE/WORTH-IT "wins" are regurgitation), or (c)
a gotcha that doesn't actually bite (note 229/yaml.v3 doesn't reproduce;
trailing-blank-line is zero-headroom, so memory-OFF succeeds incidentally
and there's no room to win).

**One artifact fixes all three: a single validated gotcha-class lesson
`L`.** Validating it *is* the STORE target, *is* the RETRIEVE lesson,
*is* the WORTH-IT lesson. No `L` → retrieve and worth-it are DOA. So `L`
is built and validated **first**, before any scored run.

`L` must clear four bars:
1. **Emergent** — absent from `CLAUDE.md`, `.claude/rules/go.md`, every
   `ExpectedVault`. (Kills rulebook-transcription.)
2. **Adaptation-requiring** — applying `L` to a test task needs
   non-trivial adaptation; the note states a *principle*, not the
   verbatim fix. (Kills relay-as-transfer.)
3. **Reproducibly biting** — memory-OFF agents repeatedly hit the
   dead-end in the pinned toolchain. (Kills incidental-OFF-success.)
4. **Vault-only** — not reachable from any statically-loaded
   instruction. (Kills static-leakage.)

**Top risk, stated honestly:** we do not yet have a concrete `L` meeting
all four bars (yaml.v3/229 disqualified — non-reproducing;
trailing-blank-line disqualified — zero-headroom + relay). Stage 0 is the
*search* for `L`. If it fails, the program halts with "no validated
transferable lesson exists in this family" — itself a real finding about
engram's value ceiling.

## The minimal set — 3 experiments, not 6

| Question | Spine | Folded-in fix | Cut |
| --- | --- | --- | --- |
| STORE | inspection rubric via `--from-transcript-range` replay | real 2-rater IRR; transfer-coverage **net of rulebook-overlap** | rigorous behavioral-injection axis (censored, unbuilt) |
| RETRIEVE | single-lesson probe | **arm C mandatory** (isolates note from recall ritual) | decoy arm (uncontrolled misdirection sign) |
| WORTH-IT | hard `SuccessCmd` gate, no LLM-judge | trial-level pairing; value as **curve over reuse-count** | convention-gated design (censors OFF at ∞) |

## Stage 0 — Build & validate (GATE: does `L` exist?)

**Must be BUILT (verified absent in current code):**
- **`SuccessCmd` execution.** `dev/eval/run.go:97` sets `TaskOK =
  !rs.IsError` ("didn't crash"); the `SuccessCmd` field
  (`model.go`) is `nil` everywhere and never run. Wire it into `runOne`:
  execute in the workspace, set `TaskOK` from exit code. Spine of every
  behavioral gate.
- **Surfacing parser.** The recall payload arrives as a `tool_result` on
  **user-type** transcript lines; `ParseBashCommands`
  (`internal/transcript/transcript.go`) reads only command strings off
  **assistant-type** `tool_use` blocks. This is a **new parse path**.
- **#643 fix** (interactive empty-vault marker breaks headless) — blocks
  the OFF/cold arm running unattended.
- **#642 incremental per-round capture** — bulk end-of-session `learn`
  fails "Prompt is too long" on a long session.
- **Two new arms** in `arms.go` (today only
  `nothing`/`skills-only`/`current-state`): `note-removed` (recall on,
  ritual intact, target note deleted = arm C for RETRIEVE) and `seeded`
  (vault pre-loaded with `L`).

**`L` validation (the gate):** author 1 donor task A and ≥2 distinct test
tasks B in the pinned Go toolchain where a non-obvious dead-end recurs
(candidate classes to screen: a stdlib ordering/iteration-determinism
trap; a context-cancellation-vs-defer interaction; an
interface-satisfaction-at-compile-vs-nil-at-runtime gotcha). Bite test: 5
memory-OFF agents per B; `L` qualifies only if OFF hits the dead-end in
**≥4/5**. Grep + arm-blind auditor confirm the note's
identifiers/fix-text are absent from each B (emergent + non-relay).
Static-leakage scan: `L` appears in none of `~/.claude/CLAUDE.md`,
project `CLAUDE.md`, `.claude/rules/`.

**Go/no-go:** `L` clears all four bars on ≥2 B-tasks → proceed. Else
**STOP** ("no validated transferable lesson in this family").
Cost: ~$15–40 compute + the harness build (the real cost; all TDD'd).

## Stage 1 — STORE (GATE: can we capture the emergent lesson?)

Inspection-only, no builds — `--from-transcript-range` replay decouples
capture from build variance.
- **Arms:** `baseline-learn` (reconstruct from before commit
  `e07bde3d`), `current-learn`, `freeform-floor` (unconstrained,
  methodology-free control). Memory-OFF N/A (inspection design).
- **Anti-circularity:** gold set = **`L` only** (emergent), plus a
  **rulebook-overlap** control metric (fraction of captured lessons
  already verbatim in CLAUDE.md+rules). Primary metric = `L`-capture
  **net of rulebook-overlap**, so transcription scores zero.
  Format-normalize notes before grading (house style is a tell).
- **IRR:** two *distinct* raters (different model family + human
  spot-check); the "2 passes of one model" number is self-consistency
  only.
- **Sample:** n=2 replays × 3 sources × 3 arms = directional only;
  pre-register "same direction across all 3 sources," no pooled means, no
  "% higher" at this n.
- **Falsification:** A-fidelity high but `L`-capture-net ≈ 0 (captured
  the rulebook, missed the emergent lesson).
- **Go/no-go:** `current-learn` captures `L` net-of-overlap → proceed.
Cost: ~$10–30.

## Stage 2 — RETRIEVE (GATE: does the surfaced note convert to use?)

- **Arms:** `nothing` (OFF floor), `seeded` (recall on, `L` in vault = B),
  **`note-removed` (recall on, ritual intact, `L` deleted = C,
  MANDATORY)**. B-vs-A confounds retrieval with the recall planning
  ritual; **only B-vs-C isolates `L`.**
- **Anti-circularity:** `L` is principle-level + adaptation-requiring,
  buried among real distractors. **Surfacing = attended/read** (in the
  agent's own reasoning, or read via the new parser), not merely present
  in `items[]`.
- **Quality bar:** `SuccessCmd` (wired in Stage 0) — hard behavioral
  assertion on the dead-end inputs, authored independent of the vault,
  never written into a note.
- **Sample:** n=10/arm (n=5 can't see the predicted *partial* effect:
  Fisher 2/5-vs-4/5 ≈ p 0.26). Report `P(applied | surfaced)` as
  descriptive/post-treatment-conditioned, not a causal conversion rate.
- **Falsification skeleton:** retrieval-broken (low surfacing) vs
  application-fails (surfaced, not applied) vs no-headroom (OFF already
  passes). Headline causal claim rests on **B-vs-C**.
- **Go/no-go:** surfacing high AND `seeded` >> `note-removed` on
  application → proceed.
Cost: ~$40–80.

## Stage 3 — WORTH-IT (the verdict; run only if 1 & 2 pass)

- **Arms:** memory-ON (`seeded` + recall + per-round capture) vs
  memory-OFF (`nothing`), **plus ON-cold mandatory** (the n=1 probe
  showed recall *overhead* dominates: warm 57t/$1.30 vs cold 29t/$0.51 —
  without ON-cold you can't separate retrieval-benefit from recall-tax).
- **Anti-circularity:** same `L` (principle); **transfer and relay
  reported as separate numbers** — an "ON wins" that's relay is not a
  transfer win.
- **Quality bar:** `SuccessCmd` including **all** dead-end inputs (so
  "fewer turns" can't mean "gave up on the edge and passed anyway").
- **Cost-model fixes (decisive):**
  - *Censoring:* per-trial turn/$ cap; non-completion = right-censored;
    **always report pass-RATE per arm** alongside cost (cost-on-survivors
    is comparable only when pass-rates match).
  - *Closing-learn OUT of the numerator:* score cost-to-B as recall +
    solve only; report a separate steady-state per-task ledger (recall +
    solve + one incremental capture) so A-capture and B-capture are
    symmetric.
  - *Incremental capture* (Stage-0 #642), or the ON ledger aborts on
    "Prompt is too long."
  - *Recall-not-firing = an ON FAILURE outcome*, not a discarded trial.
- **Sample:** n=10/arm, paired on (task+seed); statistic = Mann-Whitney /
  bootstrap median-diff (not a paired sign test); threshold ≥7/8 and
  label even that weak.
- **Headline:** **value as a curve over reuse-count** — total amortized
  cost (capture + recall + solve) to reach the bar, ON vs OFF, with the
  **break-even reuse-count** marked.
- **Falsification:** ON never crosses OFF, OR crosses only via relay
  (transfer number flat) → "not worth it for this family."
Cost: ~$60–120.

## Shared infrastructure & cross-cutting rules

Reuse: `dev/eval` DI harness, headless `claude -p` isolation,
`--from-transcript-range` replay, the existing `nothing` arm as OFF
floor. Build first (Stage 0): `SuccessCmd` execution; the
`tool_result`/user-line surfacing parser; #643; #642 incremental
capture; `note-removed` + `seeded` arms; the validated `L` + frozen blind
rubric.

Apply uniformly: transfer vs relay = always separate numbers; any cost
metric = pass-rate + survival among reachers (never difference-of-means
over arm-correlated survivors); pre-register every decision rule at its n
before any run; format-normalize / arm-blind every judge call.

## Cost envelope & the decisive number

**Total ~$125–270** (Stage 0 ~$15–40, Stage 1 ~$10–30, Stage 2 ~$40–80,
Stage 3 ~$60–120). The dominant *real* cost is Stage-0 instrument
engineering, not API spend — and it's gated so a failed `L` search halts
before the expensive stages.

**The single most important result: the break-even reuse-count** from
Stage 3 — amortized cost (capture + recall + solve) to reach the
independently-authored behavioral bar, memory-ON vs memory-OFF on `L`,
with honest censoring and a transfer-vs-relay split.
- ON crosses OFF at a small reuse-count via genuine **transfer** →
  **worth it** for this family.
- ON never crosses, or only via relay, or recall-overhead dominates in
  ON-cold → **not worth it.**

If Stage 0 can't produce `L`, the honest verdict is delivered early and
cheaply: engram has no validated transferable lesson to store, retrieve,
or pay for in this family.
