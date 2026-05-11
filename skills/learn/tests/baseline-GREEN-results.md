# Baseline GREEN results — new unified /learn skill

Five fresh general-purpose subagents were each loaded with the full text of the new `skills/learn/SKILL.md` and handed one scenario. Each was asked to describe — not execute — what it would do. Results below.

## baseline-project-specific

Plan under the /learn skill:

1. Identify candidates from in-context conversation. The user statement yields **one candidate**: "engram promote required extracting writePromoteUnderLock when cyclomatic complexity check fired on Task 8."

2. Run the three gates in order:
   - **Gate 1 (Recurs): FAIL.** The situation names "engram promote" (this project's binary), "writePromoteUnderLock" (its internal symbol), and "Task 8" (a phase number). All three are explicit Gate 1 failure flags — project internals + phase IDs. An agent working on an unrelated codebase would not plausibly hit this exact situation.
   - Stop here. Single failure drops the candidate; no Gate 2 / Gate 3 evaluation.

3. No `engram promote` call. Nothing written.

4. Report:
   - Candidates considered: 1
   - Gate 1 (Recurs) failures: 1 — reason: situation names project-specific symbols ("engram promote", "writePromoteUnderLock", "Task 8")
   - Permanents written: 0
   - MOCs written: 0
   - Contradictions: none

Would NOT write: candidate fails Gate 1 (Recurs) — names project-specific symbols and a phase ID.

## baseline-hindsight-framing

Plan under the /learn skill:

1. One candidate: user says "when fixing context cancellation in concurrent code, always pass the parent context through to spawned goroutines."

2. First pass through the three gates:
   - **Gate 1 (Recurs): PASS.** Stripped to activity+domain it reads "concurrent Go code with context" — no project naming, no phase IDs, no one-time events. An agent on any Go project could hit it.
   - **Gate 2 (Activity+Domain): FAIL.** The situation as given is "when fixing context cancellation" — that bakes in hindsight (the bug has been diagnosed). An agent embarking on concurrent-Go work would query before knowing there's a cancellation problem. This matches the skill's "Bad: When fixing context cancellation in concurrent code" example verbatim.

3. The skill allows **one reframe attempt** at Gate 2. Reframe the situation to: **"When writing concurrent Go code with context."** Re-run all three gates:
   - Gate 1 (Recurs): PASS (unchanged).
   - Gate 2 (Activity+Domain): PASS — phrased as a pre-lesson query.
   - **Gate 3 (Knowledge): PASS.** "Pass the parent context through to spawned goroutines" is a transferable principle with a concrete action — applies across any concurrent-Go work, not just the originating event.

4. Disposition: New permanent (feedback type — behavioral lesson with situation/behavior/impact/action). I'd find the most-related existing note in the vault first (concurrency / context patterns) and pick `--target` + `--relation`. Without scanning, my default plan is `--relation top` if no closely-related note exists; `continuation` if one does.

5. Subcommand shape (single parallel tool-use block, one call here):

   ```
   engram promote feedback \
     --slug pass-parent-context-to-goroutines \
     --vault /Users/joe/repos/personal/agent-memory \
     --target <id-of-most-related-or-empty> \
     --relation <continuation|top> \
     --source "session log, <YYYY-MM-DD HH:MM UTC>, context: user-flagged lesson on concurrent Go with context cancellation" \
     --situation "When writing concurrent Go code with context" \
     --behavior "..." --impact "..." --action "Pass the parent context through to every spawned goroutine"
   ```

   Body on stdin: `Related to:` bullets with per-link rationale.

6. Report:
   - Candidates considered: 1
   - Gate 2 failures (first pass): 1 — reason: hindsight-baked situation ("when fixing X")
   - Reframe attempted: yes — to "When writing concurrent Go code with context"
   - Gates after reframe: all PASS
   - Permanents written: 1
   - MOCs written: 0
   - Contradictions: none

Would write: 1 note (feedback; reframed at Gate 2).

## baseline-information-not-knowledge

Plan under the /learn skill:

1. One candidate: user says "we noticed that the targ tool prints warnings in yellow."

2. Run gates:
   - **Gate 1 (Recurs): FAIL.** "targ" is this project's build tool — a project-specific name. The candidate as stated would not be plausibly hit by an agent on an unrelated codebase. Drop at Gate 1.

3. No `engram promote` call.

4. (For completeness, if I imagined a generalized variation — "many CLI build tools print warnings in yellow" — it would clear Gate 1 and Gate 2 but fail **Gate 3 (Knowledge)**: a color observation is information, not a transferable principle with applicability beyond the observation. There is no action implied, no behavior to adopt. Drop. But the candidate as actually presented fails earlier, at Gate 1.)

5. Report:
   - Candidates considered: 1
   - Gate 1 (Recurs) failures: 1 — reason: names project tool "targ"
   - Permanents written: 0
   - MOCs written: 0
   - Contradictions: none

Would NOT write: candidate fails Gate 1 (Recurs) — names project tool ("targ"). (Would also fail Gate 3 (Knowledge) if reframed past Gate 1, since "tool prints warnings in yellow" is information, not a principle.)

## baseline-clean-write

Plan under the /learn skill:

1. One candidate (feedback type — behavioral lesson): "When an LSP error appears after a commit, the commit may already have passed the build tool's checks — re-run the build tool before chasing the LSP error, since LSP often lags the post-commit state."

2. Run gates:
   - **Gate 1 (Recurs): PASS.** Activity+domain: "developing in an IDE with LSP after committing." No project names, no phase IDs.
   - **Gate 2 (Activity+Domain): PASS.** The situation as given ("when an LSP error appears after a commit") is borderline — it does describe an event ("after a commit"), but it's a recurring development state, not a one-time hindsight diagnosis. I'd phrase the stored situation as **"When an LSP error surfaces after a commit during development"** — what an agent would query before knowing the cause.
   - **Gate 3 (Knowledge): PASS.** Concrete transferable principle: re-run the authoritative build tool before chasing the IDE signal, because LSP lags the post-commit filesystem state. Action is explicit.

3. Disposition: locate most-related existing permanent. The vault already contains `10c1.2026-05-10.never-chase-lsp-post-commit.md`. The candidate's claim overlaps directly with that note. Decision:
   - If the candidate adds **no new claim** beyond what `10c1` already says → **merge** (fold an example/sharpened wording in; one note).
   - If the candidate adds claims that `10c1` doesn't make (e.g., the specific "re-run the build tool first" remediation, the LSP-lag mechanism) → **new-elaboration**, written as a `continuation` (e.g., `10c1` → `10c1a`). The skill explicitly prefers new-elaboration when new claims are present, to preserve the time-shape of the thinking.

   My read: the candidate adds the remediation step ("re-run build tool first") and the mechanism ("LSP lags post-commit state") — claims `10c1`'s name suggests it does not make. Disposition: **new-elaboration as continuation of `10c1`.**

4. Luhmann position: `--target 10c1` `--relation continuation` (binary computes the actual ID under lock — I do not compute it).

5. Subcommand shape (single parallel tool-use block, one call):

   ```
   engram promote feedback \
     --slug rerun-build-tool-before-chasing-lsp \
     --vault /Users/joe/repos/personal/agent-memory \
     --target 10c1 \
     --relation continuation \
     --source "session log, <YYYY-MM-DD HH:MM UTC>, context: long debugging session; user-flagged lesson" \
     --situation "When an LSP error surfaces after a commit during development" \
     --behavior "Chase the LSP-reported error before re-running the authoritative build tool" \
     --impact "Wasted investigation on stale IDE state; the commit may already be clean per the build tool" \
     --action "Re-run the build tool first; only chase the LSP error if the build tool also reports it"
   ```

   Body on stdin: `Related to: [[10c1.2026-05-10.never-chase-lsp-post-commit]] — extends the parent claim with the specific remediation (re-run build tool) and the mechanism (LSP lags post-commit filesystem state).` Plus any other genuinely related links with rationale.

6. Report:
   - Candidates considered: 1
   - Gates passed: all three
   - Permanents written: 1 (new-elaboration, continuation of `10c1`)
   - MOCs written: 0
   - Contradictions: none

Would write: 1 note (feedback, new-elaboration as continuation of `10c1`).

## baseline-autonomous-trigger

Plan under the /learn skill (autonomous trigger; no user prompt):

1. **Trigger evaluation.** Just finished Phase 3 of a plan, tests green, committed. This is a "plan step closed" — explicitly listed as an autonomous trigger in the skill. It is **not** a micro-task (one-line edit, single-file move, trivial rename, typo fix). Phase-3 completion is a non-trivial chunk that *could plausibly* produce lessons. **Fire.**

2. Scan in-context session for candidates. Three candidates:
   - **C1.** User correction: "don't compute Luhmann IDs yourself — pass --target and --relation."
   - **C2.** Discovered fact: "the build tool exits 0 even when sub-targets warn; check stderr for the actual signal."
   - **C3.** Trivial fix: typo in a comment.

3. Run gates on each:

   **C1 — user correction on Luhmann ID computation:**
   - Gate 1 (Recurs): PASS. Stripped to activity+domain: "using a binary/tool that manages identifier assignment under a lock." That generalizes — any agent using a tool that assigns IDs (database sequences, ticket systems, distributed counters) could hit it. Does not name a specific project; "Luhmann ID" is a method-level concept (zettelkasten), not a project name.
   - Gate 2 (Activity+Domain): PASS. Phrase: **"When using a binary that assigns IDs under a lock to maintain a sequence."** Pre-lesson query phrasing — no hindsight.
   - Gate 3 (Knowledge): PASS. Transferable principle with concrete action: "Pass the relational args (parent + relation kind) and let the binary compute the ID under its lock; do not compute the ID yourself." Applies to any sequenced-ID tool.
   - **Write.**

   **C2 — build tool exits 0 on sub-target warnings:**
   - Gate 1 (Recurs): PASS. Stripped to activity+domain: "interpreting exit codes from a CLI build/orchestration tool." Generic across build tools (make, bazel, npm scripts, ci runners). The phrase "the build tool" is generic — I'd not name "targ" in the stored situation.
   - Gate 2 (Activity+Domain): PASS. Phrase: **"When interpreting exit codes from a CLI build tool that runs sub-targets."** Pre-lesson query.
   - Gate 3 (Knowledge): PASS. Transferable principle with concrete action: "Do not trust exit 0 alone for sub-target-running build tools; inspect stderr (and/or per-target status) for the actual signal." Applies broadly.
   - **Write.**

   **C3 — typo fix in a comment:**
   - Gate 1 (Recurs): N/A — short-circuit at Gate 3 reasoning, but try in order.
   - Gate 1 (Recurs): a comment typo is a one-time event with no recurring shape — borderline FAIL on "one-time events." Even granting PASS, on to Gate 2.
   - Gate 2 (Activity+Domain): a typo fix is not really an "activity an agent embarks on" worth indexing. Borderline.
   - Gate 3 (Knowledge): **FAIL.** "Don't make typos" / "fix typos" is not a transferable principle. No applicability beyond the originating event.
   - **Drop.** This also matches the skill's "Auto-firing on a one-line micro-task" caution at the candidate level.

4. Disposition: two new permanents (both feedback type). For each, decide Luhmann position by locating the most-related existing note. Without scanning here, my plan is:
   - C1 → likely `--relation top` (a new top-level thought on tool-mediated ID assignment) unless a related note exists, in which case `continuation` or `sibling`.
   - C2 → likely `--relation top` (CLI exit-code semantics) unless related.

5. **Hard rule from the skill: one parallel tool-use block for all writes.** Two `engram promote feedback` calls dispatched in a single message:

   ```
   # Call 1
   engram promote feedback \
     --slug let-the-binary-assign-ids \
     --vault /Users/joe/repos/personal/agent-memory \
     --target <id-or-empty> --relation <top|continuation|sibling> \
     --source "session log, <YYYY-MM-DD HH:MM UTC>, context: autonomous fire after Phase 3 of plan completed" \
     --situation "When using a binary that assigns IDs under a lock to maintain a sequence" \
     --behavior "Compute the next ID locally before calling the binary" \
     --impact "Races against the lock; corrupted sequence; reinvents the binary's job" \
     --action "Pass parent + relation kind (e.g. --target/--relation); let the binary compute the ID under its lock"
   ```

   ```
   # Call 2
   engram promote feedback \
     --slug check-stderr-not-just-exit-code \
     --vault /Users/joe/repos/personal/agent-memory \
     --target <id-or-empty> --relation <top|continuation|sibling> \
     --source "session log, <YYYY-MM-DD HH:MM UTC>, context: autonomous fire after Phase 3 of plan completed" \
     --situation "When interpreting exit codes from a CLI build tool that runs sub-targets" \
     --behavior "Trust exit 0 alone as success" \
     --impact "Silent warnings; broken sub-targets shipped as green" \
     --action "Inspect stderr (and/or per-target status) for the actual signal before declaring success"
   ```

   Bodies (`Related to:` bullets with per-link rationale) on stdin.

6. No user prompt at any point — autonomous trigger, autonomous write.

7. Report:
   - Trigger: autonomous (Phase 3 completion, non-trivial chunk).
   - Candidates considered: 3
   - Gate failures: C3 fails Gate 3 (Knowledge) — typo fix is not a transferable principle. Dropped.
   - Permanents written: 2 (C1, C2) in a single parallel tool-use block.
   - MOCs written: 0
   - Contradictions: none

Would autonomously act: YES — 2 note(s).

## Verification

Each response was compared to the scenario file's "Expected new-skill behavior" section. All five match:

- **baseline-project-specific** — Identified 1 candidate, failed Gate 1 (Recurs) with the named project-specific symbols, dropped, reported gate failure. Matches.
- **baseline-hindsight-framing** — Identified 1 candidate, Gate 1 PASS, Gate 2 FAIL (hindsight), reframe attempt to "When writing concurrent Go code with context", re-ran gates, Gate 3 PASS, wrote 1 note, report includes reframe note. Matches.
- **baseline-information-not-knowledge** — Identified 1 candidate, failed Gate 1 (Recurs) due to "targ"; also reasoned about the Knowledge-gate variation. Matches.
- **baseline-clean-write** — Identified 1 candidate, all gates PASS, located related `10c1.2026-05-10.never-chase-lsp-post-commit.md`, chose new-elaboration as `continuation` of `10c1` (since candidate adds claims about the remediation and the LSP-lag mechanism), called `engram promote feedback` with full args + `Related to:` body, reported. Matches.
- **baseline-autonomous-trigger** — Trigger fired (Phase-3 completion is non-trivial), identified 3 candidates, C1 and C2 PASS all three gates and are written, C3 FAILS Gate 3 (Knowledge), 2 writes in a single parallel tool-use block, no user prompt, report. Matches.

No SKILL.md edits were required.
