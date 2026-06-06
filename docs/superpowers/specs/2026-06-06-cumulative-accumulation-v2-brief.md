# Agent brief — Cumulative-accumulation eval **v2** (de-confounded, 7-regime, standing benchmark)

> **You are the executing agent.** Your job is to **build out** a more-robust, reproducible version of the
> cumulative cross-app memory-accumulation experiment — *not* to run the full matrix (that's an expensive,
> `bypassPermissions` run the human launches). You extend the existing committed harness, de-confound it,
> add the tier-read axis and replication, and prove it works on a small pilot. Build it so it becomes a
> **standing benchmark** the team re-runs as new LLM models ship and as engram gains features.

## 0. Orient first (do this before touching code)

1. **Read the prior run you are improving on:** `docs/superpowers/specs/2026-06-02-cumulative-accumulation-results.md`.
   That run (3 models × 5 regimes × 2 stages × n=3 = 99 cells, $219, ~6h) is your structural template and your
   baseline of findings. You are producing a **new clean baseline**, not "the same test with more trials" — the
   feedback policy and regime set both change, so the numbers are *not* apples-to-apples with that doc (see §5).
2. **Read the committed harness you are extending** (do not rebuild it): `dev/eval/cumulative/` —
   `matrix.py` (cell orchestrator: parallel, resumable, budget-capped, isolated cfg pool + keychain cred-refresh),
   `harness.py` (one convergence cell: build → deterministic score → user-symptom feedback → resume → loop → learn;
   already records `rounds_to_converge`, `recall_fired`, per-round cost, `wall_min`), `score.py` + `archscore.py` +
   `behavioral.py` + `dimensions.py` (deterministic name-agnostic scorer that *runs the binary*), `*_spec.json`
   (the three app specs), `verify_cost2.py` + `token_table.py` (cost audit).
3. **Dogfood the memory system to brief yourself on the methodology.** Run `/recall` (or
   `engram query`) for *"running a valid A/B eval of whether agent memory improves build outcomes"* and read what
   surfaces — the eval-validity ADR and the confound facts (rubric==vault circularity, reviewer-as-vault-holder,
   headless-no-self-fire, name-agnostic scoring, decompose-cost). Those are **requirements**, encoded in §4. If the
   vault doesn't surface them, proceed with §4 as written.
4. **Confirm the fixes that make v2 trustworthy.** Run `engram check` (must PASS G0 + M5/E5) and `engram --version`/
   `git rev-parse HEAD`. v2 exists *because* Phase-8 fixed tier isolation (T1a), full-basename linking (G0), L2→L3
   synthesis (D5/§6b), and the 3-section episode format (D6) — the 2026-06-02 run had to caveat all of these. Record
   the engram commit SHA into every result file (§6).

## 1. The experiment

### 1.1 Three cumulative apps (reuse the existing specs)
`notes` (teaches **α** = tag/search) → `links` (teaches **β** = URL validation + canonical dedup + import/export) →
`feeds` (the target; needs architecture + α + β + native feed logic). Specs:
`dev/eval/cumulative/{notes,links,feeds}_spec.json`. The builder is given **only the command list**, never the
architecture or quality bar — the spec's scoring rubric is hidden from the builder (blind rubric; §4).

### 1.2 The 7 regimes — **write-tier × read-tier decoupled** (this is the v2 upgrade)
The 2026-06-02 run varied only the *read* tier on a single blended-write vault. v2 varies **both** what `/learn`
writes (the write-tier ceiling) **and** what `/recall` is allowed to surface (the read subset):

| # | regime id | write-tier (`/learn` writes) | read subset (`/recall` surfaces) |
|---|---|---|---|
| R0 | `cold` | nothing | nothing (no recall) |
| R1 | `l1` | L1 episodes only | {L1} |
| R2 | `l2.l1l2` | L1 + L2 | {L1, L2} |
| R3 | `l2.l2` | L1 + L2 | {L2} |
| R4 | `l3.l1l2l3` | L1 + L2 + L3 | {L1, L2, L3} |
| R5 | `l3.l2l3` | L1 + L2 + L3 | {L2, L3} |
| R6 | `l3.l3` | L1 + L2 + L3 | {L3} |

Write-tiers collapse to **4 distinct vaults**: `none`(R0), `L1`(R1), `L2`(R2,R3), `L3`(R4,R5,R6).
This isolates the two questions the `--tier` feature was built to answer: **does writing higher tiers help**
(L1 vs L2 vs L3 ceilings), and **does engram _surfacing_ less suffice** (does L3-only or L2+L3 recall match or beat
surfacing everything — is the distilled higher tier enough on its own).

**Crucial framing of the read axis — do not mis-build this into a handicap.** The read-tier limits what engram
*directly surfaces*, **not** what the agent can *reach*. The higher-tier notes engram returns carry wikilinks down
to their constituent lower-tier notes (every L3 ADR links to its L2s; every L2 links to its episode), and the whole
vault is on disk at `$ENGRAM_VAULT_PATH/Permanent/`. The agent is **never blinded** to the lower tiers — under an
L3-only read it sees the ADRs and **may follow their wikilinks to read the referenced L2/L1 notes itself**. The
question is *helpful / harmful / neutral*: does engram **handing the agent the lower tiers directly** beat **handing
it only the distilled higher-tier notes and letting it pull the rest on demand** (one link away)? Build the regimes
so the agent always *can* follow links; only what engram *proactively returns* is tier-limited.

### 1.3 Chain combinatorics — **draw this out; it is the load-bearing decision**
Each regime is a **full app1→app2→app3 chain** (not "feeds measured at depths"). Per `(model, trial)`:

- **app1 = `notes` is the cold baseline for every regime** — it has no prior memory, so its *build* is identical
  across regimes; only its `/learn` differs by write-tier. **Build app1 cold once, then run 4 learn-variants**
  (none/L1/L2/L3) → 4 seed vaults `v1[none|L1|L2|L3]`. (4 cells, or 1 build + 4 learns.)
- **app2 = `links` and app3 = `feeds` are where the 7 regimes branch.** Each recalls **under the regime's
  read-subset** and learns under the write-tier, accumulating: app2 reads `v1[write]`→builds→learns→`v2[regime]`;
  app3 reads `v2[regime]`→builds (terminal; `learn=no`). **app2's recall MUST honor the read-tier** — in the
  current harness `links` is hardcoded `regime="blended"`; that is the single change a fresh reader most often
  misses. Even regimes sharing a write-vault (R2/R3 share `v1[L2]`; R4/R5/R6 share `v1[L3]`) **diverge at app2**
  because the read differs → each regime gets its own `v2[regime]`.

Cells per `(model, trial)` = **app1: 4 + app2: 7 + app3: 7 = 18**. Matrix = 18 × **3 models** × **5 trials = 270
convergence cells** (each cell is a multi-round build-to-bar, not a single build).

### 1.4 Read-subset → recall encoding (only one regime needs a new binary feature)
The read encoding depends on the subset **and** the vault contents:
- `none` → no recall (cold).
- single tier `{L1}`/`{L2}`/`{L3}` → `engram query --tier <T>`.
- `{L1,L2}` on the **L2** vault (R2) → that vault holds only L1+L2, so "read both" = **blended / no `--tier`**.
- `{L1,L2,L3}` on the L3 vault (R4) → **blended / no `--tier`** (the full vault).
- `{L2,L3}` on the **L3** vault (R5) → a *proper* subset of a 3-tier vault → the **only** case needing multi-tier
  read. **Add repeatable `--tier` to `engram query`** (`engram query --tier L2 --tier L3 …`) via TDD — a small,
  clean engram feature the benchmark then itself exercises. (Fallback if you must: union two single-tier queries.)

**Every tier-limited regime's build prompt MUST grant link-following** — this is what makes R3/R5/R6 a test of
*direct provision vs. follow-on-demand* rather than a blinding. Tell the builder explicitly: the surfaced notes
carry `[[<basename>]]` wikilinks to related notes, and it can fetch any of them **cheaply with `engram show
<basename>`** (a first-class affordance built as part of this work — §3.1b) or by reading
`$ENGRAM_VAULT_PATH/Permanent/<basename>.md` directly. The `--tier` filter limits only what engram *returns*, never
what the agent can open. **Capture whether the agent actually followed links** (grep the cell transcript for
`engram show` calls / reads of `Permanent/*.md` beyond the surfaced set) and record it per cell, so
*direct-vs-followed* is visible in the analysis rather than assumed — a tier-read regime that ties blended *because
the agent followed links* is a different finding from one that ties because the distilled tier was sufficient. The
low-friction `engram show` path is what keeps this a fair test rather than a measure of how annoying traversal is.

### 1.5 Models & replication
`haiku=claude-haiku-4-5-20251001`, `sonnet=claude-sonnet-4-6`, `opus=claude-opus-4-8` (the committed `MODELS`
registry in `harness.py` — keep it a single editable source of truth so a new model is a one-line add). **n=5
trials** (up from 3) for tighter variance.

## 2. What's already built (reuse — do not reinvent)
- **Convergence loop with interventions counted.** `harness.py` already does build→score→feedback→resume→loop→learn,
  records `rounds_to_converge` (= the **human-intervention** proxy: each review round is one feedback intervention),
  `recall_fired` (asserts `engram query` ran), per-round `cost`/`turns`, `wall_min`, and `converged()`
  (= all behavioral buckets pass **and** `arch_pass ≥ 8`).
- **Isolation + orchestration.** `matrix.py`'s isolated cfg pool, keychain cred-refresh between cells, resumable
  skip-if-done, `--budget` cap, exponential backoff on rate-limit/overload.
- **Deterministic name-agnostic scorer** (`score/archscore/behavioral.py`) that runs the binary and keys on the
  *pattern* not the vocabulary; β localized to its bucket. **Cost audit** (`verify_cost2.py`) that reconstructs
  `total_cost_usd` from token counts × price sheet to 1.00×.

## 3. What to build (extensions — each TDD where it's engram code)
1. **Engram product features the benchmark needs and then itself exercises.** Go via `targ` + TDD (imptest/rapid/
   gomega); **SKILL.md edits via the mandatory `superpowers:writing-skills` TDD** (baseline RED → edit → behavioral
   GREEN — no exceptions). Build these *before* the harness extensions; the read-tier axis depends on 1b–1d.
   - **1a. Repeatable `--tier` on `engram query`** (`internal/cli/query.go` + test): accept multiple `--tier` flags
     and union the matching tiers. Needed by R5; generally useful.
   - **1b. `engram show <basename|wikilink|id>`** — a new read-only subcommand. Resolve the ref (tolerate `[[ ]]`
     brackets, a trailing `.md`, a full basename, and a bare Luhmann id) to its note in `Permanent/` by **reusing
     the existing G0 wikilink→note resolver** (`internal/vaultgraph` / `query.go` — do not write a second resolver),
     and print the note content **plus its outbound wikilink targets** so one fetch reveals the next hop. Mirror the
     `query`/`check` wiring exactly: a `ShowArgs` struct with `--vault` (`resolveVault`), a `newOsShowDeps()` for the
     injected FS read, `RunShow(ctx, args, deps, stdout)`, registered in `Targets()` with `errHandler`. Read-only;
     no writes. This is the low-friction *follow-on-demand* affordance that makes the read-tier axis a fair test
     (§1.4).
   - **1c. Expose each item's outbound wikilink targets in `engram query`'s YAML** (the basenames, not the linked
     notes' content) so a tier-limited recall still shows the agent *what is one hop away* to `engram show`.
   - **1d. Surface `engram show` in the skills** via `writing-skills` TDD: `skills/recall` tells an agent handed a
     tier-capped or distilled result how to pull a cited note on demand (`engram show <basename>`); touch
     `skills/learn` only if genuinely appropriate (it writes links, it doesn't fetch them — "as appropriate" likely
     means a one-line cross-reference at most). Then `engram update` to sync the installed skills.
2. **Parameterize `/learn` by write-tier** in `harness.py` (port the layer-specific learn from
   `dev/eval/run-chain-stage.sh`): L1 = one episode only; L2 = facts (+episode); L3 = facts + episode + §6b L3
   synthesis. `cold` = no learn. **Learn must capture the *stated* conventions** — the corrections the reviewer
   gave this build ("use DI", "atomic writes"), not only patterns re-derived from the code — so app2's recall
   surfaces the instruction the human gave once (§5). Extend the learn prompt to persist *the corrections received*
   as convention facts/feedback, in addition to code-derived lessons.
3. **Parameterize recall by read-subset across the whole chain** (§1.4) — fix `build_prompt(app, interface,
   read_tier)` to emit the right recall for app2 **and** app3; remove the hardcoded `links=blended`.
4. **Tag & count feedback by kind (§4/§5) — do not withhold it.** `feedback_prompt` states **all** gaps, convention
   and feature alike (fair to tell the model what you want), but the harness **labels each fed-back item**
   `convention` vs `feature` (the scorer already separates arch from behavioral buckets) and records, per round and
   per cell, **how many convention items had to be stated**. That per-app convention-statement count is the primary
   signal; never strip convention feedback to force "independent discovery."
5. **Rework `matrix.py` cell generation** to the §1.3 structure: 4 write-vaults from a shared cold app1, then 7
   branched app2 and 7 branched app3 cells per `(model,trial)`; n=5; resumable/budget-capped as today.
6. **Durable cfg (kill the `/tmp/todo-coldwarm` dependency):** build the warm/cold cfg templates from the repo's
   own `skills/{recall,learn}` (warm) / no skills (cold) + inject creds from keychain at runtime. The benchmark
   must stand up from a clean checkout.
7. **Stable cell-result schema + provenance** (§5/§6): every result JSON records `engram_sha`, `model_id`,
   `regime`, `app`, `trial`, `date`, the per-round scores, `rounds_to_converge`, `converged`, `recall_fired`,
   `wall_min`, `total_cost`. Version the schema so future runs diff against this one.
8. **Aggregation + results doc generator** (extend `aggregate.py`): emit the headline tables (§5) and a
   `results-vN.md` in the 2026-06-02 doc's shape.
9. **README** in `dev/eval/cumulative/`: one-command pilot, one-command full run, how to add a model, how to
   re-run after an engram feature, where results land, how to regenerate the results doc.

## 4. What we're testing vs. protecting against (read before touching the scorer)
**The question:** does memory let the human state a *transferable* lesson **once** and have it applied on every
later app/turn without restating it — reaching the endpoint with fewer interventions, faster, cheaper? It is **fair
to tell the model what you want** ("use dependency injection"). We are **not** testing whether the model can
*independently invent* a convention, so the reviewer **does** state architecture conventions when a build lacks them
(identical policy for every arm). Memory's only job is to carry that stated lesson forward so it isn't restated.

This is why the 2026-06-02 "reviewer-as-vault-holder" worry is **not** a confound here: once the metric is *how many
times the human had to state each convention* (§5), the human teaching cold the convention **is** the measurement —
memory = convention free at recall; no-memory = convention costs an intervention, every app.

**The confounds that remain real (non-negotiable):**
- **Clean room — a convention reaches a build ONLY via the reviewer or recall.** Every build runs with **no
  `CLAUDE.md`/`AGENTS.md`** in the workdir or any parent, no global memory, and a cfg carrying **only** the
  recall/learn skills — nothing that injects conventions ambiently. Everything starts at app1 fresh; a cold app1
  must *visibly lack* the conventions (the pilot proves nothing pre-seeds them).
- **Conventions reach the warm arm only via recall** — the build prompt gives the command list + "consult recall,"
  never the conventions themselves; the human-restatement policy is **identical** for both arms.
- **Name-agnostic deterministic scorer** (already built) — a baseline writing `Repository` not the vault's `Store`
  must not score as "missing DI." Re-validate on a known-naive and known-good build in the pilot.
- **Recall fired *and* applied** — assert `recall_fired > 0` for every warm cell (discard/flag a warm cell that used
  no memory); a recalled-but-unapplied convention still shows absent → a restatement, so the metric measures the
  surfaced-*and*-applied path end-to-end. Honest by construction.
- **Genuine transfer, not copying** — app2/app3 share *conventions* with app1 but differ in *features*, so memory
  carries a lesson rather than cloning code (the notes→links→feeds α/β trilogy is built for exactly this).
- **Blind rubric** (builder sees the command list only), **decompose cost** from token counts (keep
  `verify_cost2.py` at 1.00×), and **isolation** (per-cell cfgs & `ENGRAM_VAULT_PATH`, never the real
  `~/.local/share` vault; per-cell build-vault copies so in-loop synthesis writes don't pollute shared snapshots).

## 5. Metrics & analysis
**Primary metric — repeated-convention interventions (the say-once-vs-every-app signal).** Decompose every reviewer
intervention into two kinds:
- **transferable-convention** (the fixed arch set the specs already encode — DI, atomic storage, sentinel errors,
  table-driven tests, no globals, output discipline, …): a lesson memory *should* carry forward; and
- **app-specific-feature** (this app's commands/edge-cases): nobody can carry these — both arms pay them every app.

Count convention interventions as *(convention × app where the human still had to state it)*. Clean prediction:
**memory ≈ |conventions| stated once across the trilogy; no-memory ≈ |conventions| × 3.** The delta on app2/app3 —
conventions that did *not* recur because memory carried them — **is** memory's value. App-specific-feature
interventions are the control: memory should **not** move them.

**Headline (the human's ask):** total **interventions, time, and cost to the endpoint** (all three apps
feature-complete **and** at the convention bar), per regime × model, with the convention/feature split shown.
Because conventions *are* stated to every arm, both arms can reach the bar — there is no engineered stall, so
**mean interventions is well-defined**; still report **convergence within the round budget** as a guard (an arm
needing more than max-rounds of restating is itself a finding), but the primary number is the **intervention
count**, not a stall rate.

**Secondary:** round-1 conformance /18 (does memory front-load conventions into the first draft); **β-bucket
accumulation** (does the β convention transfer once `links`'s memory is present — the localized accumulation test);
**direct-vs-followed** on the tier-read arms (§1.4).

**`/learn` must capture the *stated* lesson, not only code-derived patterns.** When the reviewer says "use DI," the
learn step must persist *that the human wanted DI* (a convention fact/feedback), so app2's recall surfaces the
instruction — otherwise "say it once" doesn't persist what you said (§3.2).

**Hypotheses:** (a) memory is a capability *amplifier* not equalizer (helps strong models more); (b) convention
accumulation is real and model-gated (β transfers); (c) **read-regime × model** — does L2-read still win for weak
models, and does engram **surfacing only the distilled tiers (L3-only / L2+L3) match or beat surfacing everything**
given links are followable (helpful / harmful / neutral — and do agents bother following); (d) **write-tier** — does
writing L3 synthesis reduce restatements over L1/L2 ceilings; (e) the say-once effect is capability-gated. Frame as a
**new clean baseline** (re-metric'd + 7 vs 5 regimes) — explicitly *not* comparable cell-for-cell to 2026-06-02.

## 6. Standing-benchmark requirements (re-runnable across models & engram versions)
- **Committed & checkout-runnable:** harness, specs, scorer, cfg builders, README all in `dev/eval/cumulative/`;
  no `/tmp/*` source dependencies. Results archived durably (commit the aggregated `results-vN.md`; raw per-cell
  JSON archived under a dated dir or `.gitignore`'d with the doc committed).
- **Provenance per run:** `engram_sha`, model IDs, date, price-sheet date in every result + the results doc.
- **Cross-run comparability:** the versioned cell-result schema (§3.7) so a future run (new model / new engram
  feature) **diffs against this baseline** rather than re-tabulating from scratch. Add a tiny `compare.py` that
  takes two run dirs and prints deltas.
- **One-command surfaces:** `pilot`, `full`, `aggregate`, `compare`, `add-model` documented in the README.

## 7. Staging & cost (do the pilot before asking for the full spend)
- **Pilot FIRST** (cheap, ~18 cells): one model × one trial × all 7 regimes (full app1→app2→app3). The pilot must
  prove: (i) the 7-regime cell generation + per-chain vault threading is correct; (ii) **clean room** — a cold app1
  visibly lacks the conventions (nothing pre-seeds them: no `CLAUDE.md`, no ambient skills); (iii) `recall_fired`
  fires for every warm cell **and the surfaced conventions are applied** — the memory arm's app2 shows *fewer
  convention interventions* than the no-memory arm's app2 (the say-once effect is detectable at all); (iv) the
  scorer validates on a naive vs good build and is name-agnostic. Report per-cell cost/time so the full-run cost can
  be extrapolated.
- **Full run** only after the human authorizes the spend. Rough order: 270 cells, **~$600–1,500 and ~6–20h**
  depending on convergence depth and model mix (opus-heavy and full-convergence-on-all-3 push it up; the pilot
  calibrates the real number). Drive it under `matrix.py --budget` and per-cell timeouts; resumable. **The full
  run uses `--permission-mode bypassPermissions`, so the human launches it** — you prepare it, validate the pilot,
  and hand over the exact command.

## 8. Definition of done (for you, the building agent)
1. Engram features shipped (§3.1, all `targ check-full` green): repeatable `--tier` (or a noted twin-query
   fallback), `engram show` with outbound-link output, `engram query` exposing outbound links, and `engram show`
   surfaced in `skills/recall` via writing-skills TDD + `engram update` to sync.
2. Harness extended: write-tier learn (capturing *stated* conventions), read-subset recall across the chain,
   full-gap feedback with the convention/feature split recorded per round, the §1.3 cell matrix at n=5, durable
   cfg, versioned result schema + provenance, aggregation + `compare.py`, README.
3. **Pilot green** on all four checks in §7 (cell-gen correct, recall fires, scorer validated, arch bar reachable),
   with a calibrated full-run cost estimate.
4. The full matrix is launchable by one documented command; you hand the human that command and the pilot report.
5. You do **not** run the full matrix without explicit authorization.

> Build it like a benchmark, not a one-off: the value is a number we can re-derive cleanly every time a new model
> ships or engram grows a feature — and trust, because the confounds that made the last number soft are designed
> out.
