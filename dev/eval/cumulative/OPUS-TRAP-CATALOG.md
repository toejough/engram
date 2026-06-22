# Opus-Trap Catalog & Memory-Validation Exercises

Catalog of conditions where opus needed correction (or engram recall changed course), mined from
the user's session history, with minimal exercise specs to validate that **memory prevents
re-correction** — the north star: *"When I give a correction, I don't want to have to give it
again."*

This operationalizes the EXPERIMENT-LOG "harder test cases / saturation gate" requirement. The
existing eval scores how often the human must restate a convention while building three toy apps
(notes → links → feeds, simple CRUD services); that "convention oracle" *saturates* for opus — it
one-shots those conventions cold 70% of the time, leaving no restatement for memory to remove — so
we source *real* opus failure modes here instead of inventing synthetic conventions.

## Corpus & method

- **Source A — engram vault feedback notes (23):** `~/.local/share/engram/vault/*.md` (`type:
  feedback`). Pre-abstracted corrections (situation/behavior/impact/action).
- **Source B — global CLAUDE.md (§ Critical Warnings) + `.claude/rules/go.md` + auto-memory
  feedback notes (~22+12).**
- **Source C — 38 opus-tagged session transcripts** (verified skew: **20 engram, 14 synthetic
  please-tests, 4 elsewhere** — toejough-github-io ×2, dotfiles ×2). A streaming pre-pass flagged
  127 candidate correction-turns; 4 cheap-model miners filtered them to **26 real corrections**
  (the rest were system-message false positives).

**Headline finding about where exercise value lives:** the *buildable, deterministically-checkable,
high-cold-falls-in* traps are overwhelmingly **idiosyncratic local code conventions** — and those
are concentrated in Sources A/B (the vault + rules files), because code-convention corrections are
exactly what got *crystallized*. Source C is mostly **behavioral/process** corrections (claimed-done-
without-verifying, scope-creep, don't-add-parallel-mechanism) — valuable but harder to score
deterministically. A large slice of C is engram L2/L3 *design-discussion* corrections that are now
**stale** (those tiers were removed); those are excluded from exercises (we don't test removed
mechanisms).

`cold_falls_in` is a **pre-empirical estimate** (high = arbitrary local rule contradicting opus's
default → opus re-commits unprompted ≥50% of trials; med 20–50%; low <20%). Confirm with one cold
trial before building the warm side.

---

## Category 1 — Idiosyncratic code conventions (EXERCISE GOLD: buildable + deterministic + high cold-falls-in)

These contradict opus's idiomatic-Go / standard-tooling priors, so cold opus writes the natural-
but-locally-wrong form. Each has a one-line deterministic check.

| id | lesson | cold wrong move | deterministic check | source | cold_falls_in |
|---|---|---|---|---|---|
| `slices-backward` | backward iteration uses `slices.Backward`, not a C-style index loop | `for i := len(x)-1; i >= 0; i--` | grep: C-style backward loop present → FAIL; `slices.Backward(` → PASS | A `engram-modernize-linter-slices-backward` | high |
| `nilaway-split-guard` | guard `nil` after `bytes.Split` before indexing (nilaway) | indexes `parts[0]` with no nil check | `go vet`/nilaway clean; grep `if .* == nil` guard after split | A `nilaway-bytes-split-nil-slice-indexing` | high |
| `funlen-wsl-inline` | inline the value at the call site, don't add a temp var (funlen+wsl) | introduces `x := defaultParams().floor` outer var | lint (funlen+wsl_v5) passes; no new var | A `funlen-wsl-scope-lift-inline-instead-of-var` | high |
| `unused-field-consumer` | a struct field used only by a later task needs a real consumer this commit, not `//nolint:unused` | adds `//nolint:unused` | grep: no `nolint:unused`; field is read somewhere | A `unused-struct-fields-need-early-consumer-not-nolint` | high |
| `crypto-rand` | use `crypto/rand`, never `math/rand` | imports `math/rand` for a token/id | grep import `math/rand` → FAIL | B `go.md` | high |
| `req-with-context` | `http.NewRequestWithContext`, not `http.Get`/`http.NewRequest` | `http.Get(url)` | grep `http.Get(`/`http.NewRequest(` (no Context) → FAIL | B `go.md` | high |
| `sentinel-errors` | package-level `var ErrX = errors.New(...)`, not inline `fmt.Errorf` for the sentinel | inline `fmt.Errorf("not found")` | grep sentinel `var Err`; errors.Is-able | B `go.md` | med |
| `named-constants` | named const, not bare magic number | bare `3`, `0.25` | lint (gomnd/mnd) clean | B `go.md` | med |
| `split-join-trailing-newline` | after `bytes.Join`, re-append trailing newline if absent | drops the trailing `\n` | unit: round-trip preserves trailing newline | A `bytes-split-join-trailing-newline-no-separator` | med |

## Category 2 — Git / build / commit / test-and-tooling conventions (buildable, deterministic, varies cold-falls-in)

(Boundary vs Category 1: Cat 1 = the form of the *code* the model writes; Cat 2 = conventions about
the surrounding *workflow* — which build tool, commit trailer, issue tracker, test structure, and
output presentation. Both are deterministically checkable.)

| id | lesson | cold wrong move | deterministic check | source | cold_falls_in |
|---|---|---|---|---|---|
| `targ-not-gotest` | use `targ test`/`targ check-full`, never `go test`/`go build`/`go vet` directly | runs `go test ./...` | did the agent invoke `targ`? (transcript grep `go test` → FAIL) | C#1198, B `CLAUDE.md`/`engram CLAUDE.md` | high |
| `ai-used-trailer` | commit trailer is `AI-Used: [claude]`, NOT `Co-Authored-By` | `Co-Authored-By: Claude` | grep commit msg: `AI-Used:` present, `Co-Authored-By` absent | B `CLAUDE.md` | high |
| `gh-not-issuesmd` | file deferred work via `gh` issues, not a local `issues.md` | creates/edits `issues.md` | no `issues.md` created; `gh issue` used | C#769, auto-mem | low (it's a reference/preference note, not a strong correction — opus often already defaults to `gh`) |
| `make-cap` | `make([]T, 0, capacity)` when size known | `var s []T` / `make([]T, 0)` | grep `make([]T, 0, ` at known-size sites | B `CLAUDE.md` | low |
| `no-lint-suppression` | fix the underlying lint issue; never blanket-suppress / `//nolint` | adds `//nolint` or disables a linter | grep no new `nolint`/config disable | B `CLAUDE.md` | med |
| `parallel-test-state` | parallel subtests get own data; fix shared state, never remove `t.Parallel()` | removes `t.Parallel()` to fix a flake | `t.Parallel()` retained; no shared mutable state | B `CLAUDE.md` | med |
| `model-order-table` | present model comparisons weak→strong: haiku → sonnet → opus | sonnet → haiku → opus | check table column order | C#2070 | high |

## Category 3 — Behavioral / process traps (reasoning_only or hybrid-checkable)

Higher-value for the north star (these are what most cost the user), but need an LLM judge or a
structured artifact check rather than a grep. Lower exercise priority (costlier to validate).

| id | lesson | cold wrong move | check type | source |
|---|---|---|---|---|
| `verify-before-done` | run the real binary/entry point with real args before claiming done; passing unit tests ≠ usable | declared done on green tests + no-op deploy check | hybrid (binary runs + output non-nil) | A `verify-cli-with-real-binary-real-args`, B, C#5117 |
| `reuse-not-parallel-mechanism` | reuse existing logic; don't add a second mechanism for the same job | added incremental-ingest path beside existing chunking | hybrid (no new parallel path) | C#5011, C#4853 |
| `whack-a-mole` | one tool failure means more — collect the full list, fix in one pass; read the schema first | fixed first error, re-ran, hit the next | reasoning | B `CLAUDE.md` |
| `ask-before-restore` | unexplained deletions aren't auto-damage — surface and ask before `git restore` | git-restored uncommitted deletions unasked | reasoning | A `ask-before-restoring-deleted-user-data` |
| `lean-aggressive-cleanup` | on THIS user's cleanup asks, lean aggressive: consolidate to one doc, delete the rest, prune branches | defaulted to "keep all" | reasoning | A `cleanup-asks-lean-aggressive-git-as-safety-net` |
| `deliver-full-set` | when asked for N options, deliver all N with honest ratings; don't prune to a subset | collapsed to a recommended subset | reasoning | A `deliver-full-diverse-set-not-pruned` |
| `synthesis-as-artifact` | write synthesis to a reviewable artifact (user-decided vs agent-proposed split) before asking ratification | asked approval of chat-table summaries | hybrid (artifact exists) | A `present-synthesis-as-artifact-not-chat-summary` |
| `stop-at-done` | when work is complete, stop; don't propose extra work / scope creep | asked permission to do extra verification | reasoning | C#4223 |
| `no-spend-cap` | don't impose a budget ceiling that interrupts a run; estimate+confirm up front, let it finish | added a spend cap mid-run | reasoning | C#3857, auto-mem |
| `trust-the-llm` | tell the LLM the goal/situation directly; don't engineer constraints to prevent bad choices | added constraints/info-withholding to steer a weak model | reasoning | C#1869, auto-mem |
| `read-the-doc-comment` | "I left a comment in the doc" → read the doc before responding | responded without reading the referenced doc | hybrid (did it read the file?) | C#6583 |
| `dont-bypass-system-under-test` | tests must run the identical shipping code path; no harness prompt-overrides that alter behavior | validated via a harness override of the real skill | reasoning | A (eval), C#3453 |
| `commit-the-core-artifact` | committing skill/feature work includes the skill/feature code itself, not just docs/cleanup | pushed docs/plans without the skill code | hybrid (core artifact in the commit) | C#218 |
| `stale-doc-scrub` | a removal isn't done until docs/memory/glossary/diagrams naming the old path are updated/deleted | left stale references after a mechanism removal | hybrid (grep removed-term across docs) | A `doc-memory-scrub-is-part-of-architectural-removal`, C#5958 |

## Excluded — stale (removed-mechanism) corrections

Engram L2/L3/MOC/episode design-discussion corrections (C#2817, #2220, #2226, #311, #341, #350,
#5633, harness-tier overrides) are **not** exercise candidates: those tiers/folders were removed in
recall-v2 + the flat-vault migration. Testing them would test a dead architecture. (Per vault
lesson `feedback-stale-eval-results-dont-bind-new-architecture`.)

---

## Exercise specs (top buildable, deterministic, high cold-falls-in)

The minimum-viable form for every Category-1/2 trap is the same cheap shape: **one tiny Go task
whose prompt invites the natural-but-wrong idiom, scored by a one-line deterministic check.** This
is far cheaper than an app build and exactly recreates the trigger.

```
EX-slices-backward:
  complexity_tier: simple-check  (why: a single function recreates the trigger; no app needed)
  setup:    a Go file with a slice to process in reverse; repo has .claude/rules/go.md present
  prompt:   "Add a function that iterates `lines []string` in reverse and returns them joined."
  cold_predict: writes `for i := len(lines)-1; i >= 0; i--`
  pass_check_type: deterministic
  pass_check: `grep -qE 'for .*:= len\(.*\)-1; .*>= 0; .*--' file && exit 1; grep -q 'slices.Backward(' file`
  warm_input: vault note `engram-modernize-linter-slices-backward`
  lesson_delivery: engram recall surfaces the note into the prompt context
  cost: ~$0.05-0.15 (one cold + one warm, single-file task)

EX-crypto-rand:
  complexity_tier: simple-check
  setup:    a Go file needing a random token/id generator
  prompt:   "Add a function that generates a random 16-byte session token as hex."
  cold_predict: imports `math/rand`
  pass_check_type: deterministic
  pass_check: `grep -q '\"math/rand\"' file && exit 1; grep -q '\"crypto/rand\"' file`
  warm_input: `.claude/rules/go.md` crypto-rand rule (as a vault note)
  lesson_delivery: engram recall
  cost: ~$0.05-0.15

EX-ai-used-trailer:
  complexity_tier: simple-check
  setup:    a repo with one staged change; ask for a commit
  prompt:   "Commit this change with an appropriate message and trailer."
  cold_predict: `Co-Authored-By: Claude ...`
  pass_check_type: deterministic
  pass_check: last commit msg contains `AI-Used: [claude]` and NOT `Co-Authored-By`
  warm_input: CLAUDE.md trailer rule (vault note)
  lesson_delivery: engram recall
  cost: ~$0.05-0.15

EX-targ-not-gotest:
  complexity_tier: customization  (why: needs the targ build system present to be a real choice)
  setup:    the engram repo (or a stub with a `targ` target and a tempting `go test`)
  prompt:   "Run the tests and report coverage."
  cold_predict: runs `go test ./...`
  pass_check_type: deterministic
  pass_check: `grep -qE '\bgo (test|build|vet)\b' transcript && exit 1; grep -qE '\btarg (test|check)' transcript`
              (FAIL if any bare go test/build/vet appears; PASS requires a targ invocation)
  warm_input: CLAUDE.md/engram-CLAUDE.md targ rule (vault note)
  lesson_delivery: engram recall
  cost: ~$0.10-0.30

EX-nilaway-split-guard / EX-funlen-wsl-inline / EX-unused-field-consumer:
  complexity_tier: simple-check
  setup:    a Go file + the repo's linter config (nilaway / funlen / wsl_v5 / unused active)
  prompt:   the natural task (split+index / lift a value to an outer scope / add a field for later)
  cold_predict: the idiomatic-but-lint-failing form
  pass_check_type: deterministic
  pass_check: `targ check-full` (or the specific linter) exits 0
  warm_input: the matching vault note
  lesson_delivery: engram recall
  cost: ~$0.05-0.20 each
```

Ranking (high cold-falls-in, cheapest first): `slices-backward`, `crypto-rand`, `ai-used-trailer`,
`nilaway-split-guard`, `funlen-wsl-inline`, `unused-field-consumer`, `req-with-context`,
`model-order-table`, then `targ-not-gotest` (needs the build system present).

## Validation protocol (designed, not run)

1. **Cheap cold confirmation first (the fast invalidate path).** For each ranked exercise, run ONE
   cold opus trial (no memory). The `pass_check` must FAIL — i.e., cold opus commits the trap. If
   it PASSES unaided, the exercise is **saturated for opus → drop it** (the trap isn't a real opus
   blind spot). This single cheap trial validates or invalidates each exercise before any warm-side
   investment.
2. **Warm trial.** Run opus with the matching note in the vault; `engram recall` must surface it.
   `pass_check` should now PASS. The cold→warm flip on a deterministic check is the memory payoff,
   measured per-correction.
3. **Plug into the saturation gate.** Per EXPERIMENT-LOG, the harness should refuse to report a
   memory number for any model whose cold trial passes the check (saturated). This catalog's
   per-exercise cold-confirm IS that gate at the case level.
4. **Aggregate.** Memory's value at opus strength = fraction of confirmed-cold-failing exercises
   that flip to pass warm. This is the opus-strength analogue of the haiku/sonnet convention-cut,
   on traps that do NOT saturate.

## Honest caveats

- Corpus is engram-heavy (20/38 transcripts); cross-project opus corrections are thin (4 files).
  The Category-1 code conventions are partly engram-specific (nilaway/funlen/wsl/slices.Backward
  reflect this repo's linter stack) — they generalize as "arbitrary local lint/convention the model
  won't apply cold," but the specific rules are this repo's.
- `cold_falls_in` ratings are estimates; step 1 of the protocol confirms them empirically.
- Category-3 behavioral traps are the most valuable to the user but the least deterministically
  checkable — they need an LLM judge (costlier, noisier). Start with Category 1/2 to prove the
  cold-fails/warm-passes loop cheaply, then extend.
