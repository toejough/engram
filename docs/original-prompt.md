You are reviewing recent development sessions on the engram project to extract permanent lessons. Your job is to identify patterns — corrections the user made, failures encountered, and behaviors that should be prevented or enforced going forward — then propose concrete, permanent fixes (CLAUDE.md updates, hook rules, skill updates, linter configs).

Do NOT propose vague improvements. Every proposal must be a specific file edit, config change, or rule.

---

## Evidence: User Corrections (last ~3 days)

These are things the user explicitly corrected or reinforced. Each one is a signal that the LLM's default behavior is wrong:

1. **"there's no such thing as pre-existing issues"** — The LLM suggested uncommitted formatting changes "predated our session" and offered to leave them. The user's stance: if it's broken, fix it. Don't categorize issues as someone else's problem.

2. **"you never asked me for the implementation validation tools. Doesn't the spec layer skill say to interview me for tools to run?"** — The LLM skipped the specification-layers skill's explicit instruction to ask the user what tools to run at GREEN and REFACTOR steps for each layer. It just started implementing without asking.

3. **"we should be using the targ tooling for this. green: `targ test`, refactor: `targ check`"** — The LLM didn't use the project's build system. It tried to run `go test` directly or replicate targ's coverage logic instead of using `targ test` and `targ check`.

4. **"targ uses very specific commands... don't try to replicate it, just use it"** — The LLM attempted to reverse-engineer targ's coverage computation rather than running the tool. Use project tooling as-is.

5. **"we're playing whack-a-mole with targ targets that are stopping at the first failure instead of showing you all the failures at a time"** — The LLM kept fixing one failure, re-running, finding the next failure, fixing it, etc. It should have recognized the tool was showing only the first failure and asked for all errors at once, or at least anticipated multiple failures.

6. **"unit tests are for business logic coverage. if there is real business logic in the store package, then we need to use DI to allow us to mock the IO (with imptest) in tests, and just wire up the actual IO in thin wrapper functions"** — The LLM had the wrong mental model for what goes in unit tests vs integration tests. Business logic uses DI + mocks. Thin wrappers with real IO get integration-tagged tests.

7. **"/commit skill broken — it output to the screen rather than actually performing any commands"** — A skill was producing text output instead of executing bash commands. When the LLM was told to fix it: "no, if you execute it now, there's nothing for the command to do and no way for me to test any fix being right. Fix the command, then I'll use it." — Don't test a fix by running it when there's nothing for it to operate on.

8. **"it's not good practice to send the other repo to our own"** — The LLM committed a design doc that referenced another repo and linked to it. Cross-repo coupling is wrong. Put the content where it belongs (in the target repo's issue tracker) and don't check artifacts into the wrong repo.

9. **The vertical slice completion gap** (biggest correction) — The specification-layers skill completed L1A through L5 with all 58 tests green, but the resulting code was a library that couldn't be run. No CLI binary, no LLM adapters, no deployed hooks. The skill verified internal logic but never checked end-to-end usability.

## Evidence: Test & Linter Failures

These are failure modes encountered during implementation:

1. **Nilaway static analysis fights** — Extensive back-and-forth fixing nilaway violations in store tests. gomega assertions (`g.Expect(err).NotTo(HaveOccurred())`) don't satisfy nilaway's nil-checking. Required adding explicit `if s == nil` guards, replacing `err.Error()` calls with `g.Expect(err).To(MatchError(...))`, and adding nil guards before field access on `*Memory` pointers. This pattern recurred across 20+ test functions.

2. **Race condition in corpus tests** — TestT21 parallel subtests write to a shared slice. The race detector flags it but it's a test-only issue (corpus is read-only after init).

3. **Coverage threshold failures** — Multiple packages fell below 80% threshold. `targ check` only reports the first failure, so the LLM played whack-a-mole instead of addressing all failures at once.

4. **Lint-for-fail cascades** — Lint issues triggered chains of fix commits. Known model defaults (from lessons.md): magic numbers, short variable names, long lines, missing error wrapping, inline error creation, missing t.Parallel(), HTTP without context.

5. **Context exhaustion** — Session 565ea ran out of context 6+ times, requiring continuation summaries. Critical context was lost each time, leading to repeated mistakes and re-discovery of already-known constraints.

## Evidence: Existing Lessons Already Captured

The project has `docs/lessons.md` with 24 lessons from prior work, and `CLAUDE.md` files with critical warnings. Review these to avoid proposing something already documented. The question is: **are these lessons actually being followed, or are they just documented and ignored?**

---

## Your Task

Work through this in three phases:

### Phase 1: Pattern Analysis

For each piece of evidence above, answer:
- Is this already captured in CLAUDE.md or lessons.md? If so, why wasn't it followed?
- Is this a one-off mistake or a recurring pattern?
- What is the root cause? (Wrong default behavior? Missing enforcement? Unclear instruction?)

### Phase 2: Permanent Fixes

For each recurring pattern or root cause, propose ONE of these fix types (prefer enforcement over documentation):

| Fix Type | When to Use | Example |
|----------|-------------|---------|
| **Hook rule** | Deterministic enforcement needed | Block `go test` when `targ test` exists |
| **CLAUDE.md update** | Behavioral guidance for LLM | "Use `targ test` not `go test`" |
| **Linter/tool config** | Catch code-level patterns | nilaway-compatible gomega patterns |
| **Skill update** | Process gap in a skill | Add tool interview step enforcement |
| **Code pattern** | Structural prevention | Template for nilaway-safe test assertions |

For each proposal, specify: the exact file to edit, the exact content to add/change, and which evidence item(s) it addresses.

### Phase 3: Verify Coverage

Walk through all 9 user corrections and 5 failure modes above. For each one, confirm which Phase 2 proposal addresses it. Flag any that have no permanent fix proposed.

Present findings at each phase. Wait for user review before proceeding.
