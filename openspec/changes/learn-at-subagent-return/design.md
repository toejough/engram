## Context

`learn-rate-skill-only` v1 (shipped 2026-09-07) added a mandatory `LESSONS:` line to every subagent completion report (route), a running list of those lines in please step 4, and a closing-learn scan that judges the list (learn SKILL.md Step 2, "Collected LESSONS lines are an explicit scan input"). Its design D-A/D-B put all judgment at the closing learn: the orchestrator "collects lines without judging them in-flight".

Two post-ship measurements say the closing learn is the wrong place to put the whole load:

| Measure | Unit | Result | Source |
|---|---|---|---|
| W2, learn fired of worth-learning moments | moments | 3/66 = 4.5% [0.0, 10.6] vs baseline 4/71 = 5.6% [1.4, 11.3]; Δ −1.1 pp [−8.5, +6.3] | `REPORT-re-measure-2026-09-28.md` §1 |
| W2 instrument sees parent capture? | — | No: learn counted only inside audited subagent windows (caveat 3a) | same, §3a |
| C0, closing `learn` after the last LESSONS receipt | parent sessions | 0/2 | `REPORT-parent-capture-2026-09-29.md` §2 |
| C1, captured or already covered, of worth-capturing items | items | 93/114 = 81.6% [74.6, 87.7], judged; an upper bound | same, §2–3 |
| C1 without route-record matches (post hoc) | items | 80/114 = 70.2% [62.3, 78.1] | same, §2 |
| Judge (b) matches that held on reading | items | captured 4/10, covered 6/8 (unblinded, one reader) | same, §3 |
| Who wrote the 53 captured items' notes | notes | parent transcript 38, subagent 3, unknown 12 | same, §2 |

The capture that happened was mid-session: parents wrote notes directly with `engram learn` while work was going on (`23a08637` 45 notes, 13 of them principle notes; `01e43979` a batch of 9 principle notes). Joe (2026-09-29): "update the guidance to lean into the agents' tendency to [learn] midstream … after sub agents return, in particular."

Current texts this change touches:

- `agent-instructions/guidance/learn.md` — ambient guidance, imported by the harness config (`production-guidance-activation`). Three cues (review/user rejection, failed check with a non-obvious cause, self-caught reversal) and the fast-path paragraph (lines 15–19). No cue mentions a subagent return. No `openspec/specs` capability governs this file today.
- `agent-instructions/skills/learn/SKILL.md` — the mid-cycle fast-path block (lines 21–27) is scoped to "a CORRECTION moment"; the LESSONS paragraph (lines 89–95) is framed as closing-learn input handed over by please.
- `agent-instructions/skills/please/SKILL.md` — step 4 (lines 95–99): "This list is separate from the lessons audit (step 7) and is not judged in-flight; it is handed to the closing `/learn` as-is." Step 7 (lines 119–122) hands the full list to the closing learn.
- `agent-instructions/skills/route/SKILL.md` — completion-report bullet (lines 70–75): the lessons are "offers for the closing `/learn` to judge".

## Goals / Non-Goals

**Goals:**

- Make a subagent return a learn-firing moment: judge the returned LESSONS lessons then, and capture the ones that clear learn's bar with `/learn` on the fast path before the next dispatch.
- Keep the closing learn as a backstop that sweeps and captures only what was not captured at return.
- Remove the v1 wording in please and route that tells the orchestrator not to judge at return.
- Test the guidance edit with confined headless RED/GREEN arms, with pre-registered pass and false-fire bars; test each SKILL.md edit with writing-skills TDD.
- Record the `learn-rate-skill-only` task 3.4 decision.

**Non-Goals:**

- Changing learn's Step-2 bar, the four kinds, or the kind-4 exemplars.
- Changing the LESSONS contract (mandatory line, format, one re-ask) or the `lessons-contract` capability.
- Building any of the parked D-F layers (watcher, mechanical validator, hooks).
- Subagents running `/learn` themselves (rejected in `learn-rate-skill-only` design, Alternatives).
- A new W2 re-measure. The W2 instrument cannot see this path (caveat 3a); measuring the parent path at scale is a follow-up, not part of this change.
- Editing `delegate.md`, `recall.md`, or `shim.md`.

## Decisions

### D1: The cue lives in `learn.md`, the ambient learn guidance

Vault note 855: a rule that must hold at every delegation, whichever skill (if any) mediated it, belongs in always-loaded guidance, not in a route or please template. Parent-capture shows the dispatches that matter are often ad hoc; `01e43979` never invoked please's closing learn at all. `learn.md` is always loaded and already owns "when to fire `/learn` mid-task", so the return cue joins its cue list.

- **Alternative: `delegate.md`.** Note 855's own example names delegate.md, which already says "Reviewing what returns … is your job." Rejected as the home: the rule is about *when to learn*, and learn.md already carries the fast-path instruction and the confirmed-not-guess bar the cue depends on. Splitting them would put the trigger in one file and its bar in another. Putting it only in learn.md also keeps one variable for the eval (D5). **Decided by Joe, 2026-09-29: the cue goes in `learn.md` only.**
- **Alternative: route/please templates.** Rejected per note 855; route and please only get the consistency edits in D4.

### D2: The guidance text

Two edits to `learn.md`; the other text stays byte-identical.

1. The fast-path paragraph (lines 15–19) says "crystallize the one confirmed correction now". It becomes "crystallize the one confirmed lesson now", so it covers a returned lesson as well as a correction.
2. A fourth bullet under "Fire at these cues":

   > - **A subagent just returned, and its report's `LESSONS:` line isn't `none`** — the lesson is in hand now, and at the closing learn it is one line in a long list (if the closing learn runs at all). Judge each lesson as you read the report: if it is a confirmed correction, reversal, or confirmed approach that states a reusable rule, `/learn` it on the fast path **before your next dispatch**. Skip `none`, "done, tests pass", and anything the report doesn't confirm ("might be a race") — those are not lessons, and the closing learn will discard them anyway.

The implementing task checks this pre-written wording against the final spec before applying it (vault note 1072). The GREEN arms test the text as applied, not this draft.

**Amendment (U2+U3 review F5): the applied text, as tested, differs from the draft above.** The 3.1 wording check against the final `guidance-learn-moments` spec (vault note 1072) found three coverage gaps in the draft and closed them before applying, so GREEN tests this text, not item 2's draft:

> - **A subagent just returned, and its report's `LESSONS:` line isn't `none`** — the lesson is in hand
>   now, and at the closing learn it is one line in a long list (if the closing learn runs at all).
>   Judge each lesson as you read the report: if it is a confirmed correction, reversal, explicit
>   save-request, or confirmed approach that states a reusable rule, `/learn` it on the fast path — one
>   note per lesson — **before your next dispatch** (or before you end your turn, if no dispatch is
>   left). Skip `none`, "done, tests pass", and anything the report doesn't confirm ("might be a
>   race") — those are not lessons, and the closing learn will discard them anyway.

The three additions: the spec's four kinds include the explicit save-request, which the draft's three-kind list omitted; the spec requires firing "before it dispatches the next subagent **or ends its turn**", so "one note per lesson" and "(or before you end your turn, if no dispatch is left)" were added so the last return (unit 3, no further dispatch) still triggers the cue; and "one note per lesson" makes explicit the spec's counting unit, which the draft left implicit. Shipped as commit `85e58376`; `git diff` shows exactly two hunks (U1-report.md:354–361).

### D3: The learn skill applies the bar per return; the closing learn is the backstop

- **Fast-path block (SKILL.md lines 21–27).** The trigger widens from "a CORRECTION moment mid-task" to also "a subagent return whose `LESSONS:` line carries a lesson that clears the bar". The body (skip Step 1 and 1.5, go to Step 2, no `engram ingest --auto`) is unchanged.
- **LESSONS paragraph (lines 89–95).** Rewritten to say: the bar applies to each returned result, at the return, on the fast path, one note per lesson that clears it. At the closing learn, the collected list is scanned for lines not yet captured; a line is captured when a note written earlier this session covers it (please marks such lines). Captured lines are skipped, and the rest are judged as before. The silent-discard rule and the bar itself do not change.
- The closing learn keeps its always-sweep rule (Step 1). The backstop matters because the return cue is prose and will miss some returns (vault note 198).

### D4: Consistency edits in please and route (scoped)

**Decided by Joe, 2026-09-29: make the route wording edit, with its own RED/GREEN at n = 5 (D6 R1).**

Each edit is limited to the wording that tells the orchestrator not to judge at return. Nothing else in either skill changes.

| File:lines | Current | New |
|---|---|---|
| please SKILL.md:95–99 (step 4) | record the line verbatim; "not judged in-flight; it is handed to the closing `/learn` as-is" | record the line verbatim (unchanged); judge each lesson then, per the learn guidance's return cue, with a fast-path `/learn` before the next dispatch for each that clears the bar; the full list still goes to the closing learn (Joe, 2026-10-01: captured marking dropped; see the D6 design-change note) |
| please SKILL.md:119–122 (step 7) | hand over the full list; "`/learn`'s Step 2 curates that list … step 7 does not pre-filter it" | hand over the full list; the closing learn skips any lesson a vault note written earlier this session already covers and curates the rest; step 7 still does not drop lines (Joe, 2026-10-01: captured marking dropped; see the D6 design-change note) |
| route SKILL.md:72–73 | "offers for the closing `/learn` to judge" | "offers the orchestrator judges when the report returns (fast-path `/learn` for those that clear the bar), with the closing `/learn` as backstop" |

- **Unchanged:** route's mandatory line, format, and one-re-ask rule; please's red-flag row about reaching step 7 without the list; the `lessons-contract` spec; please's lessons audit.
- **Spec impact:** please's two LESSONS requirements are MODIFIED (the collection requirement drops "collection is not judging"; the hand-off requirement adds the closing learn's dedupe by vault coverage; Joe, 2026-10-01, dropped the captured marks it first added). Route's requirements do not mention who judges, so route has no delta.

### D5: Guidance eval, confined headless RED/GREEN

Subagents inherit session context and would carry the treatment into a RED control (MEMORY: headless, not subagents). Every arm is therefore a fresh `claude -p` process, and the only variable between RED and GREEN is the learn guidance text.

**Confinement.** Per the archived `local-first-parent-sync` design D11, including the tmp-dir narrowing (vault note 1075):

- **Environment.** `env -i` with only `HOME=$ARM/home`, `USER`, `PATH=$ARM/bin:/usr/bin:/bin`, `TERM=dumb`, `TMPDIR=$ARM/tmp`, `XDG_DATA_HOME=$ARM/xdg`, `ENGRAM_VAULT_PATH=$ARM/vault`, and `CLAUDE_CODE_OAUTH_TOKEN`. The token is read from the keychain per batch and never echoed or logged. No `ENGRAM_PARENT`, `ENGRAM_SERVER`, or `ENGRAM_VAULT_NAME`.
- **Paths.** `ARM=$(mktemp -d /private/tmp/engram-arm.XXXXXX)`. The working directory is `$ARM/work`, which is not a repo, so no project CLAUDE.md loads. `HOME=$ARM/home` also means the real `~/.claude/CLAUDE.md` does not load (MEMORY: headless-eval gotchas).
- **Layer 1 (permissions; no `bypassPermissions`).** `--allowedTools "Bash(engram:*)" "Read" "Glob" "Grep" "Skill" "Agent"` (`Skill` so `/learn` can fire; `Agent` so the orchestrator can dispatch the fixture subagent; the harness confirms both tool names against the installed `claude` version in the smoke run). `--disallowedTools "WebFetch" "WebSearch" "Bash(git:*)" "Bash(curl:*)" "Bash(security:*)"`. `permissions.deny: ["Read(//Users/joe/**)"]`.
- **Layer 2 (Seatbelt).** `$ARM/home/.claude/settings.json` holds the sandbox exactly as in D11: `enabled` and `failIfUnavailable` true; `allowUnsandboxedCommands` and `autoAllowBashIfSandboxed` false; `filesystem.allowWrite: ["$ARM"]`; `denyRead: ["/Users/joe"]`; `denyWrite` listing each existing entry of `/private/tmp/claude-<uid>`, generated by `ls` at batch setup; network `allowedDomains: []`, `strictAllowlist: true`.
- **Per-batch write probe.** Before each batch, one probe arm with the batch's settings, plus `Bash(touch:*)`, tries to `touch` a file inside one existing entry of `/private/tmp/claude-<uid>`. The batch runs only if the touch fails and the file is absent afterwards.
- **Per-batch isolation check.** Before and after each batch, the harness saves to `results/<batch>/isolation.json`: the list of files under the real vault (outside `.git/`) newer than the batch start, and the top-level `~/.claude` entries newer than the batch start. Any real-vault entry newer than the batch start fails the batch. Per vault note 956, the orchestrating session writes nothing to the real vault while a batch runs; route-evidence and learn writes wait until the batch ends.

**Arm contents (held constant except `learn.md`):**

- **Full production guidance set (decided by Joe, 2026-09-29).** Both arms load the same four guidance files that production imports: `recall.md`, `delegate.md`, `learn.md` and `shim.md`. They are copied to `$ARM/home/.claude/engram/` and imported by `$ARM/home/.claude/CLAUDE.md` with the same `@` import lines and order as the real `~/.claude/CLAUDE.md`. `recall.md`, `delegate.md` and `shim.md` are byte-identical to the repo at 67911117 in both arms. **The only difference between the arms is `learn.md`:** RED gets `learn.md` at 67911117, and GREEN gets the edited `learn.md`. Each copy also carries the marker lines described under the delivery gate below.
- `$ARM/home/.claude/skills/` holds all six engram skills (`recall`, `learn`, `please`, `route`, `curate`, `write-memory`), copied from 67911117, in **both** arms. The full guidance set refers to recall, route and please, so these skills have to be present, and pinning them to the pre-edit commit keeps the SKILL.md edits from becoming a second variable.
- `$ARM/bin/engram` is the installed engram binary. `$ARM/vault` starts empty.
- `$ARM/home/.claude/agents/unit-worker.md` is the fixture subagent: `model: haiku`, no tools, and a system prompt that returns one fixed report verbatim, whatever the input. The report is chosen per cell.

**Amendment (task 1 ruling T3): the worker is keyed by unit, not a single fixed report.** D5's "one fixed report, whatever the input" cannot produce unit 2's and unit 3's `LESSONS: none` reports from a single template; the harness instead keys the worker's scripted report by unit number (1, 2 or 3), identical across arms and cells. Accepted per ruling T3.

**Amendment (ruling T10, Joe 2026-09-30): the fixture worker does real work.** The tool-less worker above was visibly fake — RED v1 showed 18/19 arms flagging "0 tool uses" and 3/19 abandoning the task, which produced N1's one false fire from worker-doubt rather than the LESSONS cue (not a RED/GREEN confound, since the fixture is identical across arms, but a realism threat to both RED and GREEN). After two further STOPs (T8: Read/Write/Edit with a scripted implementation and a pre-generated test-output log — the orchestrator still fired `/learn` about the worker's unverifiable claim, outside the window; T9: narrow Bash limited to `cat`-ing the log — the orchestrator sent the worker back to run real tests and abandoned when it couldn't), the shipped fixture (T10) is a real, stdlib-only Go module per domain: the worker Reads a stub, Writes a pre-authored implementation and `_test.go`, and runs real `go -C <module> test ./...` under the D5/D11 sandbox (`$ARM/bin/go`, `GOCACHE`/`GOPATH`/`GOMODCACHE` inside `$ARM`, `GOPROXY=off`). The report lines, including every `LESSONS:` line, are unchanged and pinned by a test. The v1 tool-less results are superseded (`dev/eval/learn-at-return/results/superseded-v1/`).

**Amendment (ruling T5, T6): route records and `qa` writes are excluded from the window scoring.** A route-evidence or route-dispatch note (`engram learn --slug route-dispatch-*`/`route-evidence-*`, or a written note basename `<id>.<date>.route-(dispatch|evidence)-*.md`) written inside the window counts as neither a pass nor a false fire — it is listed in `window_route_records` for audit. Likewise, `engram learn qa` is not a lesson capture for this gate; it is listed in `window_qa_writes` for audit. Only `Skill`(learn), or an `engram learn` write of kind `feedback`/`fact`/`runbook` that is not a route record, count as lesson captures.

**Amendment (ruling T7): every scored arm is hand-audited, not just gated on the mechanical flags.** Before any cell's verdict counts, the scoring step hand-audits every arm carrying `window_end_unit: null`, an unparsed in-window learn mention, a route record, or a `qa` write — and, in practice, every arm regardless of flags — to confirm the mechanical label against a read of the transcript (see the hand audits in `.superpowers/sdd/tasks/U1-report.md`, tasks 2.1, 3.2 and 4.1/4.3).

**Scenario (fictional domains; vault notes 283, 1037).** The `-p` prompt gives the orchestrator a three-unit task in a fictional project and says to dispatch each unit, in order, to the `unit-worker` agent and not to do the units itself. The task is unambiguous, so a question-stop is not the expected response. Unit 1's report carries the cell's `LESSONS:` line; units 2 and 3 return `LESSONS: none`. The lesson is incidental to a substantive task, not the point of the prompt (buried-subtask shape, note 283). Two domains alternate across arms:

- **Quillfeather**, a static-site generator (front-matter dates, slug rules, a template cache);
- **Tarnbrook**, an inventory CLI (unit conversions, a CSV importer, a lock file).

**Cells and fixture LESSONS lines.** Each worth-keeping line is checked against learn's Step-2 bar and frozen in the harness before any arm runs.

| Cell | Unit-1 `LESSONS:` content | Expected |
|---|---|---|
| P1 (kind 1) | a reviewer rejected an approach and named the project convention (e.g. "reviewer rejected local-time dates; Quillfeather's convention is UTC ISO-8601 in front matter") | fire |
| P2 (kind 4) | an uncertain approach, acted on and confirmed by an observed outcome (e.g. "wasn't sure the CSV importer mis-parsed quoted commas; a targeted repro test confirmed it before the fix") | fire |
| N1 | `none` | no fire |
| N2 | a bare success ("completed the unit, all tests pass") | no fire |
| N3 | an unconfirmed hunch ("the lock-file flake might be a race in the cache warmer") | no fire |

**Delivery gate (MEMORY: verify treatment delivery; vault notes 284, 939).** The arm must prove it loaded the whole guidance set as well as its own arm's `learn.md`.

- **Markers.** The harness appends one plain `Session tracking token: …` line to each arm copy:
  - `recall.md` gets `LAR-RECALL-2H6W`;
  - `delegate.md` gets `LAR-DELEGATE-9C3N`;
  - `shim.md` gets `LAR-SHIM-5V1R`;
  - `learn.md` gets `LAR-LEARN-8J2D`, plus the arm token: `LAR-GREEN-7Q4K` in GREEN, `LAR-RED-3M8T` in RED.

  The shipped files never carry a token.
- **Check.** After each arm, the harness greps every record type in the arm's session JSONL under `$ARM/home/.claude/projects/`. Note 939: `claude -p` attaches CLAUDE.md, and its imports, as non-message records. A scored arm must contain all four file markers and its own arm token, and must not contain the other arm's token.
- Arms that fail the gate are discarded and replaced, never scored.
- The prompt does not ask for any token to be echoed.
- **If imports are not recorded.** If the smoke (task 1.5) shows that imported file content does not appear in the transcript, so the gate cannot be checked, the implementer stops and asks Joe. It does not substitute another delivery mechanism.

**Scoring (mechanical, from the arm's stream-json transcript):**

- **Return event:** the tool result of the `Agent` call that dispatched unit 1.
- **Window:** from the return event to the next `Agent` tool_use (the unit-2 dispatch), or to the end of the session if no further dispatch happens.
- **Fire:** a `Skill` tool_use with `skill` = `learn` inside the window.
- **Fast path:** no Bash command containing `engram ingest` between the return event and the first `engram learn` inside the window.
- **Pass (P cells):** a fire, on the fast path, with at least one `engram learn` write in the window. Reported separately and never counted as passes: fired-with-sweep; `engram learn` run directly without the `Skill` call ("captured, not via skill"); a fire only after the unit-2 dispatch ("late").
- **False fire (N cells):** any `Skill`(learn) or `engram learn` inside the window.
- **Not scored, replaced by a fresh arm:**
  - **Question-stop** (the turn ends with a question before the unit-2 dispatch; vault notes 1030/1037). Reported as an instruction-clarity finding, not a failure. At most 2 replacements per cell; a third question-stop in a cell stops that cell, the fixture's clarity gap is fixed, and the cell is rerun in both arms from scratch.
  - **Degenerate run** (API error, empty result, or unit 1 never dispatched to `unit-worker`). Discard counts are reported per cell (MEMORY: detect degraded builds).

**Amendment (U2+U3 review F6): a fourth degenerate class exists beyond the three above.** Because both arms load the GREEN `learn.md` (the full guidance set includes it for the fixture worker too, not only the orchestrator — see "Arm contents" above), the worker itself is exposed to the treatment. In the GREEN N3 cell, 2 of the first 7 arms had the worker return `LESSONS: none` instead of the scripted hunch line — "unit-1 `LESSONS:` line never returned / not verbatim" — and were discarded and replaced (`results/green-N3-r1`). This is legitimate (the treatment was not delivered as scripted) but was not one of D5's three listed kinds; it does not hide a false fire, because in both discarded arms the orchestrator made no learn call of any kind and the vault stayed empty.

**Sample sizes and bars (pre-registered).** Arms run in batches, one cell per batch.

**Amendment (U2+U3 review F10): batches did not interleave RED and GREEN.** D7's order (RED first, then the `learn.md` edit, then GREEN) makes interleaving within a batch impossible; batches instead ran all-RED (19:02–19:53 UTC, 2026-09-30) and then all-GREEN (20:12–21:24 UTC). Drift risk is accepted as low: one ~2.5 h span, the same `claude` 2.1.282 and `claude-opus-5-5`, and the same engram binary sha across both halves.

| Cell | GREEN n | RED n |
|---|---|---|
| P (P1 5 + P2 5) | 10 | 10 |
| N1 / N2 / N3 | 5 / 5 / 5 | 3 / 3 / 3 |

About 44 scored arms, plus probes and replacements.

1. **Ship gate (GREEN), all three must hold:**
   - P: ≥ 8/10 passes;
   - N: ≤ 1/15 false fires in total;
   - N1: 0/5 false fires.
2. **Effect label (reported; not a ship gate):**
   - RED P ≤ 4/10: "effect shown";
   - RED P ≥ 5/10: "high baseline — lift not attributable at this n". Shipping still follows item 1, because the ask is to reinforce a tendency agents already have.
   - This is a difference test that can only see large gaps (about 4/10 or more). It is not an equivalence test (vault note 987).
3. **On a failed ship gate:**
   - Revise the wording once. The revised text gets new tokens (`LAR-GREEN2-…`).
   - Rerun only the failing GREEN cells, fresh, at the same n.
   - A second failure stops the work, and the result goes to Joe with the transcripts. The implementer does not loosen a bar.
4. **Cost:** a 2-arm smoke (one P1 GREEN, one N1 RED) re-measures the per-arm cost with the full guidance set loaded. The prompts are larger than a `learn.md`-only arm, and shim's `engram query` and recall add tool calls, so no earlier per-arm figure is reused. The implementer then confirms the projected total with Joe before the first scored batch. There is no mid-run spend cap (MEMORY: no spend cap on eval runs). The running tally is kept in `cost-log.jsonl`.

### D6: SKILL.md edits under writing-skills TDD

Each of the three SKILL.md edits runs `superpowers:writing-skills` (RED → GREEN → REFACTOR/pressure). The arms use the D5 confinement and delivery-gate pattern. The variable is the SKILL.md under test, with a skill-specific token appended to the arm's copy. Both arms load the full production guidance set with the GREEN `learn.md` and the D5 file markers, because that is the text the skills ship alongside. The other five skills stay pinned to 67911117. Each cell has n = 5 per arm, and the bars are fixed now:

| Skill | Cell | Scenario | RED expectation | GREEN bar |
|---|---|---|---|---|
| learn | L1 fast path at return | Prompt: a subagent just returned with one kind-1 `LESSONS:` lesson; `/learn` it now, then continue to unit 2 | the skill treats it as closing-learn input and sweeps (`engram ingest`) or writes no note, in ≥ 2/5 | 5/5 write exactly one note, with no `engram ingest` and no `vocab stats` |
| learn | L2 closing dedupe | Closing learn with 3 collected lines; one is marked captured, and its note is already in `$ARM/vault` | the captured line is written again in ≥ 2/5 | 5/5 sweep, write no duplicate, and judge the other two per the bar |
| learn | L3 pressure | L2 with 6 lines (2 captured, 1 trivial, 1 hunch, 2 worth keeping) under "we're out of time, just write them all" | — | 5/5 write only the 2 uncaptured worth-keeping lines and still sweep |
| please | Q1 step-4 return | Continuing please at step 4: unit 1 of 3 returns a kind-1 lesson | no fire before unit 2 in ≥ 2/5 ("not judged in-flight") | ≥ 4/5 fire on the fast path before unit 2 (the original "and the list entry is marked captured" half was dropped by Joe, 2026-10-01: a design change, see below) |
| route | R1 return after a routed dispatch | route was invoked for the dispatch; the return carries a kind-1 lesson | no fire before the next dispatch in ≥ 2/5 | ≥ 4/5 fire on the fast path before the next dispatch |

**Design change (Joe, 2026-10-01): please drops the captured mark.** Q1's GREEN bar first required both a fast-path fire and marking the running-list entry captured. The fire held at 5/5 in every please arm, RED included. The mark did not hold: GREEN Q1 4/5, Q1P pressure 3/5, and a structural refactor (a required status field) regressed both cells to 1/5. Joe dropped the marking requirement. please still judges each lesson at its return and fires the fast-path `/learn` before the next dispatch. At close, the closing learn dedupes by vault coverage (a note written earlier this session covers the lesson), which L2 showed working at 5/5 even on the pre-edit learn SKILL.md (`results/red-L2`). This is a recorded design change, not a loosened bar: the Q1/Q1P bar is now the fire alone, at the same ≥ 4/5. The two earlier please GREEN rounds are kept, marked superseded by the design change.

**RED shows no gap.** If a RED cell meets its GREEN bar, the premise is falsified for that cell (MEMORY: a RED baseline can falsify the design premise). Record the numbers. The edit then ships only as wording consistency, with GREEN run as a non-regression check at the same bar, and the finding goes to Joe. The fixture is not re-engineered to force a RED.

### D7: Order of work

1. RED guidance arms, against the unedited `learn.md`.
2. Edit `learn.md`, then GREEN guidance arms.
3. writing-skills cycles for learn, please and route. Their GREEN arms carry the GREEN `learn.md`. (Joe, 2026-10-01: please's captured marking was dropped after two GREEN rounds missed the Q1P pressure bar on the mark alone; the please GREEN cell reruns on the unmarked text. See D6.)
4. Docs from `enumeration.md`.
5. Deploy, verify, close out (D8, D9).

Guidance comes first because the skill edits refer to its cue.

### D8: Recording `learn-rate-skill-only` task 3.4

3.4's done-when is a joint decision with Joe, recorded in `dev/eval/audit/escalation-decision-<date>.md`. Joe's 2026-09-29 instruction is that decision:

- None of D-F's three layers escalates now.
- The next lever is this md-only change.
- The D-F revisit condition stays as written.

At close-out, the implementer:

1. writes `dev/eval/audit/escalation-decision-2026-09-29.md`, quoting Joe, citing both reports, and pointing to this change;
2. ticks 3.4 with a pointer to that file and to this change;
3. updates the ROADMAP NOW-table row 4 to "3.2–3.4 done; next lever `learn-at-subagent-return`".

This proposal carries the pointer now; the tick waits for close-out, so that 3.4 is not marked done before the lever it names has shipped.

**Amendment (ruling T1): 3.4 was ticked early, at Joe's explicit 2026-09-30 instruction.** Joe instructed "sync and archive `learn-rate-skill-only`, then apply this one" before this change's own tasks 1–8 completed. `dev/eval/audit/escalation-decision-2026-09-29.md` was written, 3.4 was ticked with the pointer to that file and this change, and `learn-rate-skill-only` was archived (commits `d66facd4`, `a9d39cb2`) — all ahead of this change's own close-out (D8 steps 1–2, and D9 step 2). Joe's explicit instruction overrides this design's "tick only after ship" ordering; D9's archive-order steps 1–2 ran before this change's tasks 1–7, not after.

### D9: Archive order

`learn` and `please` are delta specs in the unarchived `learn-rate-skill-only`, so `openspec/specs/{learn,please}` do not exist yet. `openspec validate --strict` passes without them (checked 2026-09-29); `openspec archive` applies MODIFIED against the main specs and would fail. The close-out order is therefore:

1. Tick 3.4 (D8).
2. `openspec archive learn-rate-skill-only`.
3. Confirm that `openspec/specs/{learn,please}/spec.md` contain the two headers this change modifies, byte-for-byte.
4. Run the collision sweep.
5. `openspec archive learn-at-subagent-return`.

## Risks / Trade-offs

- **[Clean probe over-fires, so GREEN may look better than production]** (vault note 277). → Buried-subtask fixture: three units, and the lesson is incidental (note 283). The RED arm measures the baseline under the same load. The ship gate is on GREEN, and the effect label is reported honestly.
- **[RED is already high]** Parents already capture midstream (the parent-capture report). → The pre-registered effect label ("high baseline — lift not attributable") covers this, and it does not block shipping. The ask is to reinforce the tendency and move capture to the return; the false-fire gate is what guards against harm.
- **[Over-capture: writing trivial or unconfirmed lessons mid-task rots the vault]** → False-fire gate (≤ 1/15, and 0/5 on `none`). The cue restates the existing bar; the curate skill remains downstream.
- **[Noise from the full guidance set]** Joe chose production realism (2026-09-29). shim's `engram query`, recall firing and delegate's dispatch doctrine add behaviour and variance to every arm, and they can compete with the learn cue for attention. The set is identical in both arms, so it is not a confound, but it widens per-arm variance at these n. → Accepted as a caveat and reported with the results. **No bar is lowered to compensate.** Any recall or query activity inside the window is recorded but does not change a pass or false-fire classification.
- **[A scripted haiku subagent is not a real worker]** → What is under test is the orchestrator's reaction to a return, not the worker. A fixed report makes the LESSONS content controlled and identical across arms.
- **[Dedupe relies on "a note written this session covers it"]** → The L2/L3 cells test it. On every path, the closing learn reads its own session's writes (please's captured marks were dropped by Joe, 2026-10-01; L2 showed vault-coverage dedupe at 5/5 even on the pre-edit text).
- **[Prose caps below ~95%]** (vault note 198). → The closing backstop stays, and the parked mechanical layers remain the escalation path.
- **[Archive dependency on a sibling change]** → D9 fixes the order, and the collision sweep reads the sibling's full tasks.md (vault note 757).
- **[Info note, U2+U3 review F4] `delegate.md`'s "never trust the builder's own done" versus the return cue's "confirmed".** `delegate.md` says reviewing what returns is the orchestrator's job and warns against trusting "done" as the subagent reports it. The return cue (D2) has the orchestrator judge a lesson as "confirmed" directly from the subagent's own report. These are not a contradiction: the cue judges the *lesson's* confirmation (is it a correction, reversal, or validated approach the report states as fact, not a hypothesis), not the *unit's* completion — delegate.md's skepticism about "done" is a separate axis. No rule reconciles the two in text today, though GREEN transcripts show orchestrators doing both (capturing the lesson, then separately challenging the worker with SendMessage). No action is required for ship; worth reconsidering if `please`/`route` wording is revisited again.

## Migration Plan

Markdown only. Deploy with `engram update --with-guidance` (learn.md) and `engram update` (skills). Check that each deployed copy is byte-identical to its source. Rollback is `git revert` of the edit commits followed by the same two commands.

## Open Questions

None open. Joe settled the three earlier questions on 2026-09-29:

- `learn.md` only (D1);
- the route wording edit with its own RED/GREEN at n = 5 (D4);
- the full production guidance set in both arms (D5).
