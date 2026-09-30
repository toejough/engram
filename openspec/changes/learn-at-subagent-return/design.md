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

- **Alternative: `delegate.md`.** Note 855's own example names delegate.md, which already says "Reviewing what returns … is your job." Rejected as the home: the rule is about *when to learn*, and learn.md already carries the fast-path instruction and the confirmed-not-guess bar the cue depends on. Splitting them would put the trigger in one file and its bar in another. Putting it only in learn.md also keeps one variable for the eval (D5). (Open question 1.)
- **Alternative: route/please templates.** Rejected per note 855; route and please only get the consistency edits in D4.

### D2: The guidance text

Two edits to `learn.md`; the other text stays byte-identical.

1. The fast-path paragraph (lines 15–19) says "crystallize the one confirmed correction now". It becomes "crystallize the one confirmed lesson now", so it covers a returned lesson as well as a correction.
2. A fourth bullet under "Fire at these cues":

   > - **A subagent just returned, and its report's `LESSONS:` line isn't `none`** — the lesson is in hand now, and at the closing learn it is one line in a long list (if the closing learn runs at all). Judge each lesson as you read the report: if it is a confirmed correction, reversal, or confirmed approach that states a reusable rule, `/learn` it on the fast path **before your next dispatch**. Skip `none`, "done, tests pass", and anything the report doesn't confirm ("might be a race") — those are not lessons, and the closing learn will discard them anyway.

The implementing task checks this pre-written wording against the final spec before applying it (vault note 1072). The GREEN arms test the text as applied, not this draft.

### D3: The learn skill applies the bar per return; the closing learn is the backstop

- **Fast-path block (SKILL.md lines 21–27).** The trigger widens from "a CORRECTION moment mid-task" to also "a subagent return whose `LESSONS:` line carries a lesson that clears the bar". The body (skip Step 1 and 1.5, go to Step 2, no `engram ingest --auto`) is unchanged.
- **LESSONS paragraph (lines 89–95).** Rewritten to say: the bar applies to each returned result, at the return, on the fast path, one note per lesson that clears it. At the closing learn, the collected list is scanned for lines not yet captured; a line is captured when a note written earlier this session covers it (please marks such lines). Captured lines are skipped, and the rest are judged as before. The silent-discard rule and the bar itself do not change.
- The closing learn keeps its always-sweep rule (Step 1). The backstop matters because the return cue is prose and will miss some returns (vault note 198).

### D4: Consistency edits in please and route (scoped)

Each edit is limited to the wording that tells the orchestrator not to judge at return. Nothing else in either skill changes.

| File:lines | Current | New |
|---|---|---|
| please SKILL.md:95–99 (step 4) | record the line verbatim; "not judged in-flight; it is handed to the closing `/learn` as-is" | record the line verbatim (unchanged); judge each lesson then, per the learn guidance's return cue, and mark the entry captured when a fast-path `/learn` writes it; the full list, with marks, still goes to the closing learn |
| please SKILL.md:119–122 (step 7) | hand over the full list; "`/learn`'s Step 2 curates that list … step 7 does not pre-filter it" | hand over the full list with captured marks; the closing learn skips captured lines and curates the rest; step 7 still does not drop lines |
| route SKILL.md:72–73 | "offers for the closing `/learn` to judge" | "offers the orchestrator judges when the report returns (fast-path `/learn` for those that clear the bar), with the closing `/learn` as backstop" |

- **Unchanged:** route's mandatory line, format, and one-re-ask rule; please's red-flag row about reaching step 7 without the list; the `lessons-contract` spec; please's lessons audit.
- **Spec impact:** please's two LESSONS requirements are MODIFIED (the collection requirement drops "collection is not judging"; the hand-off requirement adds the captured marks). Route's requirements do not mention who judges, so route has no delta.

### D5: Guidance eval, confined headless RED/GREEN

Subagents inherit session context and would carry the treatment into a RED control (MEMORY: headless, not subagents). Every arm is therefore a fresh `claude -p` process, and the only variable between RED and GREEN is the learn guidance text.

**Confinement.** Per the archived `local-first-parent-sync` design D11, including the tmp-dir narrowing (vault note 1075):

- **Environment.** `env -i` with only `HOME=$ARM/home`, `USER`, `PATH=$ARM/bin:/usr/bin:/bin`, `TERM=dumb`, `TMPDIR=$ARM/tmp`, `XDG_DATA_HOME=$ARM/xdg`, `ENGRAM_VAULT_PATH=$ARM/vault`, and `CLAUDE_CODE_OAUTH_TOKEN`. The token is read from the keychain per batch and never echoed or logged. No `ENGRAM_PARENT`, `ENGRAM_SERVER`, or `ENGRAM_VAULT_NAME`.
- **Paths.** `ARM=$(mktemp -d /private/tmp/engram-arm.XXXXXX)`. The working directory is `$ARM/work`, which is not a repo, so no project CLAUDE.md loads. `HOME=$ARM/home` also means the real `~/.claude/CLAUDE.md` does not load (MEMORY: headless-eval gotchas).
- **Layer 1 (permissions; no `bypassPermissions`).** `--allowedTools "Bash(engram:*)" "Read" "Glob" "Grep" "Skill" "Agent"` (`Skill` so `/learn` can fire; `Agent` so the orchestrator can dispatch the fixture subagent; the harness confirms both tool names against the installed `claude` version in the smoke run). `--disallowedTools "WebFetch" "WebSearch" "Bash(git:*)" "Bash(curl:*)" "Bash(security:*)"`. `permissions.deny: ["Read(//Users/joe/**)"]`.
- **Layer 2 (Seatbelt).** `$ARM/home/.claude/settings.json` holds the sandbox exactly as in D11: `enabled` and `failIfUnavailable` true; `allowUnsandboxedCommands` and `autoAllowBashIfSandboxed` false; `filesystem.allowWrite: ["$ARM"]`; `denyRead: ["/Users/joe"]`; `denyWrite` listing each existing entry of `/private/tmp/claude-<uid>`, generated by `ls` at batch setup; network `allowedDomains: []`, `strictAllowlist: true`.
- **Per-batch write probe.** Before each batch, one probe arm with the batch's settings, plus `Bash(touch:*)`, tries to `touch` a file inside one existing entry of `/private/tmp/claude-<uid>`. The batch runs only if the touch fails and the file is absent afterwards.
- **Per-batch isolation check.** Before and after each batch, the harness saves to `results/<batch>/isolation.json`: the list of files under the real vault (outside `.git/`) newer than the batch start, and the top-level `~/.claude` entries newer than the batch start. Any real-vault entry newer than the batch start fails the batch. Per vault note 956, the orchestrating session writes nothing to the real vault while a batch runs; route-evidence and learn writes wait until the batch ends.

**Arm contents (held constant except the guidance text):**

- `$ARM/home/.claude/CLAUDE.md` is the learn guidance text plus one marker line (below). RED uses `learn.md` at this change's base commit (67911117); GREEN uses the edited `learn.md`. No other guidance file is loaded in either arm; this keeps one variable and lowers cost, at the price of being less like production (Risks).
- `$ARM/home/.claude/skills/{learn,write-memory}/SKILL.md` are the repo's pre-edit copies (commit 67911117) in **both** arms, so the SKILL.md edit is not a second variable.
- `$ARM/bin/engram` is the installed engram binary. `$ARM/vault` starts empty.
- `$ARM/home/.claude/agents/unit-worker.md` is the fixture subagent: `model: haiku`, no tools, and a system prompt that returns one fixed report verbatim, whatever the input. The report is chosen per cell.

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

**Delivery gate (MEMORY: verify treatment delivery; vault notes 284, 939).** The harness appends one plain line, `Session tracking token: LAR-GREEN-7Q4K`, to the GREEN arm's copy of the guidance, and `Session tracking token: LAR-RED-3M8T` to the RED arm's copy. The shipped `learn.md` never carries a token. After each arm, the harness greps every record type in the arm's session JSONL under `$ARM/home/.claude/projects/` (note 939: `claude -p` attaches CLAUDE.md as a non-message record). A GREEN arm must contain the GREEN token and not the RED token; a RED arm the reverse. Arms that fail the gate are discarded and replaced, never scored. The prompt does not ask for the token to be echoed.

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

**Sample sizes and bars (pre-registered).** Arms run in batches, one cell per batch, interleaving RED and GREEN.

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
4. **Cost:** a 2-arm smoke (one P1 GREEN, one N1 RED) gives the per-arm cost. The implementer then confirms the projected total with Joe before the first scored batch. There is no mid-run spend cap (MEMORY: no spend cap on eval runs). The running tally is kept in `cost-log.jsonl`.

### D6: SKILL.md edits under writing-skills TDD

Each of the three SKILL.md edits runs `superpowers:writing-skills` (RED → GREEN → REFACTOR/pressure). The arms use the D5 confinement and delivery-gate pattern. The variable is the SKILL.md under test, with a skill-specific token appended to the arm's copy. The GREEN `learn.md` is present in both arms, because that is the text the skills ship alongside. Each cell has n = 5 per arm, and the bars are fixed now:

| Skill | Cell | Scenario | RED expectation | GREEN bar |
|---|---|---|---|---|
| learn | L1 fast path at return | Prompt: a subagent just returned with one kind-1 `LESSONS:` lesson; `/learn` it now, then continue to unit 2 | the skill treats it as closing-learn input and sweeps (`engram ingest`) or writes no note, in ≥ 2/5 | 5/5 write exactly one note, with no `engram ingest` and no `vocab stats` |
| learn | L2 closing dedupe | Closing learn with 3 collected lines; one is marked captured, and its note is already in `$ARM/vault` | the captured line is written again in ≥ 2/5 | 5/5 sweep, write no duplicate, and judge the other two per the bar |
| learn | L3 pressure | L2 with 6 lines (2 captured, 1 trivial, 1 hunch, 2 worth keeping) under "we're out of time, just write them all" | — | 5/5 write only the 2 uncaptured worth-keeping lines and still sweep |
| please | Q1 step-4 return | Continuing please at step 4: unit 1 of 3 returns a kind-1 lesson | no fire before unit 2 in ≥ 2/5 ("not judged in-flight") | ≥ 4/5 fire on the fast path before unit 2, and the list entry is marked captured |
| route | R1 return after a routed dispatch | route was invoked for the dispatch; the return carries a kind-1 lesson | no fire before the next dispatch in ≥ 2/5 | ≥ 4/5 fire on the fast path before the next dispatch |

**RED shows no gap.** If a RED cell meets its GREEN bar, the premise is falsified for that cell (MEMORY: a RED baseline can falsify the design premise). Record the numbers. The edit then ships only as wording consistency, with GREEN run as a non-regression check at the same bar, and the finding goes to Joe. The fixture is not re-engineered to force a RED.

### D7: Order of work

1. RED guidance arms, against the unedited `learn.md`.
2. Edit `learn.md`, then GREEN guidance arms.
3. writing-skills cycles for learn, please and route. Their GREEN arms carry the GREEN `learn.md`.
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
- **[Arm guidance is less than production]** (only `learn.md`, no recall/delegate/shim). → Accepted, to keep one variable and the cost down. Deployment makes no claim about the interaction with shim or delegate beyond what the skill cycles exercise.
- **[A scripted haiku subagent is not a real worker]** → What is under test is the orchestrator's reaction to a return, not the worker. A fixed report makes the LESSONS content controlled and identical across arms.
- **[Dedupe relies on "a note written this session covers it"]** → The L2/L3 cells test it. please's captured marks make it explicit on the please path. Off the please path, the closing learn reads its own session's writes.
- **[Prose caps below ~95%]** (vault note 198). → The closing backstop stays, and the parked mechanical layers remain the escalation path.
- **[Archive dependency on a sibling change]** → D9 fixes the order, and the collision sweep reads the sibling's full tasks.md (vault note 757).

## Migration Plan

Markdown only. Deploy with `engram update --with-guidance` (learn.md) and `engram update` (skills). Check that each deployed copy is byte-identical to its source. Rollback is `git revert` of the edit commits followed by the same two commands.

## Open Questions

1. **learn.md or delegate.md (D1).** Default: learn.md only. Vault note 855's example names delegate.md, but the trigger and its bar belong together, and one file keeps the eval to one variable.
2. **Route's wording edit (D4, third row; D6 R1).** Default: make it, with its n = 5 RED/GREEN. It is one phrase, but an orchestrator that has just read route sees "for the closing `/learn` to judge" at the moment the cue should fire.
3. **Guidance set in the arms (D5).** Default: learn.md alone in both arms. The alternative is the full production set (recall, delegate, learn, shim): more realistic, but costlier per arm, and shim's `engram query` adds noise.
