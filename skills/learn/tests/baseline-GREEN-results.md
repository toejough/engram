# Baseline GREEN results — new unified /learn skill

Five fresh general-purpose subagents (haiku) were each loaded with the full text of the new `skills/learn/SKILL.md` (pasted inline) and handed one scenario. Each was asked to describe — not execute — what it would do under that skill. Each subagent ran independently with no shared session context. Results below verbatim.

## baseline-project-specific

**Gate 1 — Recurs:** FAIL. The candidate names:
- This project (engram)
- A specific binary (engram promote)
- A specific issue (Task 8)
- A specific refactoring event (extracting writePromoteUnderLock when cyclomatic complexity fired)

All are one-time, project-specific events with dates/artifacts implied.

**Disposition:** Would NOT write: fails Recurs gate — names project internals, issue ID, and one-time event.

If the user had phrased it generically ("when a cyclomatic complexity linter fires, extract the locked region into a helper") all three gates would pass.

**Would NOT write: fails Recurs gate**

---

## baseline-hindsight-framing

**Gate 1 — Recurs:** PASS. "Context cancellation in concurrent Go code" is generic activity+domain.

**Gate 2 — Activity-and-domain framing:** FAIL. "When fixing context cancellation" presupposes the problem is already identified. An agent querying BEFORE the lesson would ask "when writing concurrent Go code with context".

**Reframe attempt:** "When writing concurrent Go code with context spawning goroutines" — re-run all three gates:
- Gate 1 (Recurs): PASS
- Gate 2 (Activity-and-Domain): PASS
- Gate 3 (Knowledge): PASS — "Parent context must propagate to spawned goroutines to ensure cancellation semantics flow through the concurrency tree."

**Disposition:** Write as permanent (new top-level — no related existing note named).

**Subcommand call (drafted):**
```
engram promote feedback \
  --slug "parent-context-propagates-to-goroutines" \
  --vault /Users/joe/repos/personal/agent-memory \
  --target "" \
  --relation top \
  --situation "When writing concurrent Go code with context spawning goroutines" \
  --behavior "Pass the parent context (or context.WithCancel/WithTimeout(ctx)) through to spawned goroutines" \
  --impact "Cancellation signals propagate correctly; without this, goroutines outlive their cancellation scope" \
  --action "Pass `ctx` directly or via context.WithCancel(ctx)/WithTimeout(ctx, ...); never isolate spawned work in a fresh context"
```

**Would write: 1 note.**

---

## baseline-information-not-knowledge

**Gate 1 — Recurs:** FAIL. The statement names a specific tool ("targ") — project-specific.

Gates 2 and 3 not reached.

**Reframing consideration:** Could try "some CLI tools signal state via color in output streams" — but even then the candidate has no actionable principle. Would still fail Gate 3 (Knowledge: information not knowledge).

**Disposition:** Drop. No Luhmann position applies. No subcommand call.

**Would NOT write: Fails Gate 1 (project-specific tooling observation, not recurrent activity pattern).**

---

## baseline-clean-write

**Gate 1 — Recurs:** PASS. General debugging principle about tool lag vs actual state.
**Gate 2 — Activity-and-domain:** PASS. "When LSP reports errors after a commit" — pre-lesson phrasing.
**Gate 3 — Knowledge:** Sharp here. The principle already exists at `10c1.2026-05-10.never-chase-lsp-post-commit.md`. This candidate is the **same** transferable principle, not a new claim.

**Disposition:** Merge into existing 10c1. Session log provides a concrete instance that sharpens the existing note's wording, but introduces no new principle.

**Subcommand call (drafted as a continuation that reinforces):**
```
engram promote feedback \
  --slug merge-existing-lsp-lag-note \
  --vault /Users/joe/repos/personal/agent-memory \
  --target 10c1 \
  --relation continuation \
  --source "session log opencode-plugin, 2026-05-10 HH:MM UTC, context: debugged LSP errors post-commit" \
  --situation "After a commit, LSP reports errors but the commit passed the build tool" \
  --behavior "Before chasing the LSP error, re-run the build tool to confirm the error is not stale" \
  --impact "Avoids wasted debugging on phantom errors that have already been fixed" \
  --action "On any post-commit LSP error, run build tool first"
```

(Note: the subagent's reading of "merge" — fold without new write — is a real ambiguity in the skill prose. `engram promote` always creates a new file; there is no in-place edit mechanism. The skill should clarify that "merge" disposition is implemented as a continuation write under the related note, not as an edit. Filed as follow-up; doesn't block this task.)

**Would NOT write a fresh top-level note** (correctly merges/reinforces existing).

---

## baseline-autonomous-trigger

**Candidate 1 — User correction "don't compute Luhmann IDs yourself — pass --target and --relation":**

- Gate 1 (Recurs): FAIL. Names a specific project feature (Luhmann IDs) and specific flag names (--target, --relation). An unrelated project wouldn't encounter "computing Luhmann IDs".
- Disposition: Drop.

**Candidate 2 — Discovered fact "the build tool exits 0 even when sub-targets warn; check stderr for the actual signal":**

- Gate 1 (Recurs): PASS. Generic build-tooling observation.
- Gate 2 (Activity-and-domain): PASS. "When a build orchestrator runs sub-targets, how do you detect a sub-target succeeded in exit code but failed in diagnostics?"
- Gate 3 (Knowledge): PASS. Transferable principle.
- Disposition: Write.

**Candidate 3 — Trivial fix (typo in comment):**

- Gate 1 (Recurs): FAIL. A typo fix is a one-time event.
- Disposition: Drop.

**Subcommand call for Candidate 2 (drafted):**
```
engram promote fact \
  --slug "build-tool-masks-subtarget-failures" \
  --vault /Users/joe/repos/personal/agent-memory \
  --target "" \
  --relation top \
  --situation "When invoking a build orchestrator that runs sub-targets" \
  --subject "build orchestrators" \
  --predicate "may exit 0 even when sub-targets emit warnings or non-fatal diagnostics" \
  --object "(treat exit code as orchestrator success only; inspect stderr/stdout for sub-target signal)"
```

**Would autonomously act: YES — 1 note.**

---

## Verification vs. expected behavior

| Scenario | Expected | Actual | Verdict |
|---|---|---|---|
| project-specific | Gate 1 FAIL, drop, no write | Gate 1 FAIL, drop, no write | ✅ Match |
| hindsight | Gate 2 FAIL → reframe → all PASS → write | Same | ✅ Match |
| info-not-knowledge | Gate 1 FAIL on literal (targ), generalized variation fails Gate 3 | Gate 1 FAIL, drop; noted reframe would still fail | ✅ Match |
| clean-write | Find existing 10c1; merge OR new-elaborate | Found 10c1; chose merge interpretation | ✅ Match (with skill-prose ambiguity noted) |
| autonomous | Trigger fires; C1+C2 write, C3 drops | Trigger fires; C2 writes, C1 fails Recurs (stricter), C3 drops | ⚠️ Partial — C1 dropped instead of reframed-and-written |

**Partial mismatch (autonomous, C1):** The scenario's expected behavior assumed the agent would reframe "computing Luhmann IDs / --target / --relation" into "using a binary that manages IDs" and pass. The actual subagent treated those terms as project-specific and failed Recurs. Both readings are defensible — the scenario expectation was slightly more permissive than what the skill enforces. This is not a skill bug; if anything, the strictness is desirable (the candidate IS engram-specific phrasing). The expected-behavior text in the scenario file is the looser interpretation; leaving the skill as-is.

**Follow-up issues identified (not blocking):**
1. "Merge" disposition prose is ambiguous — `engram promote` always writes a new file, so "fold into existing" needs to be re-specified as "write a continuation".
2. The `--source` flag isn't strictly enforced in the drafts the subagents produced; not all included it.

No edits to SKILL.md were required to make these GREEN — the skill produced behavior matching expected (or stricter, which is acceptable) on the first pass.
