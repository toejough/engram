# Proposals for the "nothing could have caught this" moments (20 original + 29 rescope = 49)

Task 5.4 of the `memory-loop-audit` change collects the moments the auditor labeled
`failure_category: nothing_could_have_caught_it` — originally 20 of the 150 audited moments; the
2026-09-03/04 failure-category rescope (which re-judged every previously-unasked moment type for
real — see the report's §8e) grew the population to **49**. Groups A–G below cover the original
20 unchanged; the clearly-marked **Rescope Extension** section covers the 29 new moments. Each
group gets a fix proposal following vault note 495's two-rung remedy ladder (vault note by
default; a repo/engram-code change only where vault note 803's self-referential-procedure test
says a note genuinely can't reach the bug) and vault note 803's test itself. **Nothing here has
been written to the vault and no GitHub issue has been filed** — this is a draft for Joe's review
at the audit's gate (task 6.2); task 6.1 folds it into the final report.

Vault note 803's test asks: would a note stating the fix ever be retrieved *at the moment the bug
occurs*, given the procedure's own query phrasing at that step? That test only bites when the bug is
IN a self-referential procedure's (recall/learn/please/route/skill) own execution. None of the 49
moments below are that kind of bug — they are all ordinary task-execution moments, not the memory
system misfiring on itself — so rung 2 (a repo/engram-code change) is never proposed here; every group
lands on rung 1 (a vault note) or on no-fix-needed.

**At a glance (updated 2026-09-04 for the 49-moment population):** **18 moments tied to 5
proposed vault notes** — the original 11 (`safe-read-chunk-sizing`,
`git-add-cant-stage-untracked-deletes`, `git-pull-rebase-needs-clean-tree`,
`macos-timeout-is-bsd-not-gnu`, `transient-auth-failure-retry-not-reconfigure`) plus 7 rescope
moments that are recovery-side twins of the same incidents (**the extension proposes 0 new
notes**); **29 moments are no-fix-needed** (7 original + 22 rescope); **2 moments are out of
scope** for this audit (a phone-llm repo tooling bug, unchanged). Full detail and justification
for each group below; see the Summary at the end for the same tally restated with group letters.

## Group A — User-initiated interruptions (3 moments)

- `agent-ac7ec7f863667110d.jsonl#7` (repo: targ) — User interrupts the request before the agent can
  complete the plan review task, leaving work incomplete.
- `agent-a8fe06233a9aa49a1.jsonl#4` (repo: targ) — User interrupted the agent's request before it
  could complete the investigation task.
- `agent-a89cbbb396928d46a.jsonl#14` (repo: dotfiles) — The agent's investigation into the repo's
  config-linking mechanism is interrupted by the user mid-tool-call.

**Verdict: correctly categorized as unfixable. No fix proposed.** A human choosing to interrupt a task
is not a memory-loop failure of any kind — no vault note or process change would or should prevent
this.

## Group B — External data genuinely doesn't exist (2 moments)

- `agent-ac2925d5bb0aa269d.jsonl#44` (repo: llmcpp) — Agent cannot find M1 Pro specific MLX benchmark
  data; llmcheck.net lacks the requested 35-60 tok/s M1 Pro breakdown.
- `agent-ac2925d5bb0aa269d.jsonl#55` (repo: llmcpp) — Agent discovers InsiderLLM provides only generic
  Apple Silicon ranges (25-40 tok/s), not M1 Pro specific tok/s figures as requested.

**Verdict: correctly categorized as unfixable. No fix proposed.** The agent was searching external web
sources for a specific benchmark number that those sources simply don't publish. No vault note or
engram change fixes the internet lacking a specific data point.

## Group C — Task prompt's own assumption was wrong / self-recovered minor tool errors (2 moments)

- `agent-a5f13c6128d39ec08.jsonl#38` (repo: engram) — Grep for 'DRIFT' in docs/architecture/adr.md
  returned no output, contradicting the task prompt's expectation that ADR-0020 would contain DRIFT
  references.
- `agent-a96aadd41edf00c3d.jsonl#10` (repo: engram) — Agent attempts to use Read tool on a directory
  path, receives EISDIR error; recovers by switching to bash ls command.

**Verdict: correctly categorized as unfixable / not worth a fix. No fix proposed.** The first is a
wrong assumption baked into the dispatching prompt itself, not a memory-recall gap. The second is a
trivial, immediately self-corrected tool-usage slip with no repeated cost.

## Group D — Repeated blind retries after a Read/output size or token-limit error, without recalling a known fix (6 moments)

- `agent-a9b6cc5ad1398eb75.jsonl#10` (repo: engram) — Bash tool output exceeded size limits (33KB) and
  was truncated/persisted to a file, requiring the agent to Read the persisted file separately to get
  the full content.
- `agent-a96aadd41edf00c3d.jsonl#41` (repo: engram) — Agent hits token limit (28686 tokens exceeds
  25000 max) when trying to read LEDGER.md with offset/limit parameters.
- `agent-a8ce598d27fa736b9.jsonl#7` (repo: phone-llm) — Agent attempts to read 299KB file but exceeds
  256KB size limit; tool returns error.
- `agent-a8ce598d27fa736b9.jsonl#12` (repo: phone-llm) — Agent retries read with offset/limit
  parameters but still exceeds 25000 token limit (51430 tokens returned). *(Same transcript as the
  previous moment, second failed attempt.)*
- `agent-a81b80335e1cdd38e.jsonl#6` (repo: phone-llm) — Fork's Read tool fails when attempting to read
  full 303.7KB file; exceeds 256KB size limit.
- `agent-a81b80335e1cdd38e.jsonl#13` (repo: phone-llm) — Fork's second read attempt still fails; token
  count (34,637) exceeds allowed maximum (25,000). *(Same transcript as the previous moment, second
  failed attempt.)*

**Verdict: a real, recurring, fixable pattern. PROPOSE a vault note (rung 1).** These 6 moments are
4 distinct underlying incidents (2 standalone: `a9b6cc5a...#10`, `a96aadd4...#41`; 2 explicit
first-attempt/second-attempt pairs: `a8ce598d...#7`+`#12`, `a81b80335...#6`+`#13`). In 2 of those 4
incidents (the two paired ones) the agent's *second* attempt — after already seeing the error once —
still failed the same way, meaning it retried with a guessed offset/limit rather than computing one
actually guaranteed to fit. This session's own guidance (`recall.md`) already documents a firing cue
for exactly this situation ("After a failure you can't immediately explain — recall glance once before
you start guessing — a past lesson may name the cause"), so per vault note 803's test this is
reachable by rung 1: the recall step already exists and already fires on failures, it just doesn't
yet have a note with the concrete fix to surface.

**Proposed vault note slug:** `safe-read-chunk-sizing`

**Proposed vault note content:**

> **Situation:** a Read or Bash tool call fails with a file-size or token-count limit error.
>
> **Action:** don't retry with an arbitrarily smaller or similarly-sized offset/limit guess — compute
> a chunk size that's guaranteed under budget using the numbers the error message itself reports (e.g.
> if the tool reports content is 51,430 tokens against a 25,000 max, that's ~2.06x over budget, so the
> next attempt's line range should be roughly half the previous attempt's, not another guess); or
> better, use grep/head/wc -l to size up a large file BEFORE attempting a full read, rather than
> discovering the limit by hitting it.

## Group E — Reusable git/shell environment conventions (3 moments, one convention each)

- `agent-a427a4df64476d6c0.jsonl#40` (repo: engram) — git add fails because the old note files (never
  git-tracked, only untracked `??`) cannot be staged for deletion since they don't match any git
  pathspec.

  **Proposed vault note slug:** `git-add-cant-stage-untracked-deletes`

  **PROPOSE:**

  > **Situation:** removing a file that git has never tracked (shows as `??` in `git status`, not a
  > tracked-and-modified file).
  >
  > **Action:** `git add <path>` cannot stage the deletion of a file git never tracked — an untracked
  > file doesn't match any pathspec for a delete-staging purpose. Use `rm` directly (no `git add`
  > needed) when removing a never-tracked file.

- `7c95676b-a345-466d-a782-7f9c8df6647b.jsonl#25` (repo: phone-llm) — Git pull fails with error
  "cannot pull with rebase: You have unstaged changes" blocking the workflow.

  **Proposed vault note slug:** `git-pull-rebase-needs-clean-tree`

  **PROPOSE:**

  > **Situation:** about to run `git pull --rebase` (or any rebase-based pull).
  >
  > **Action:** `git pull --rebase` requires a clean working tree — check for and stash (or commit)
  > unstaged changes before attempting a rebase-pull, rather than discovering the requirement via the
  > failure.

- `agent-ae96e991d50f060c0.jsonl#16` (repo: phone-llm) — After timeout command fails on macOS (line
  15), agent re-runs engram ingest without timeout wrapper.

  **Proposed vault note slug:** `macos-timeout-is-bsd-not-gnu`

  **PROPOSE:**

  > **Situation:** wrapping a command with `timeout` on macOS.
  >
  > **Action:** macOS's default `/usr/bin/timeout` (BSD) doesn't exist / behaves differently than GNU
  > `timeout`. On macOS, either install `coreutils` and use `gtimeout`, or don't rely on a `timeout`
  > wrapper being present at all.

**Verdict for all three: PROPOSE a vault note each (rung 1).** These are ordinary, reusable
dev-practice conventions — not bugs in engram's own recall/learn/please/route procedures — so vault
note 803's self-referential-procedure test doesn't apply and doesn't block rung 1. **Honest limitation:**
whether a note actually gets recalled at the right moment (before running an unfamiliar git/shell
command) is genuinely less certain than Group D's case — there's no equivalent "after a failure, recall
before guessing" applicability here, since these often fail on the *first* attempt, before any
failure-triggered recall cue would fire.

## Group F — Auth/login failure mid-session (2 moments, same incident)

- `1df64b43-aa52-402f-a852-0534776b788e.jsonl#15` (repo: joe home dir session) — Agent encounters
  authentication_failed error and cannot respond to user's debugging request.
- `1df64b43-aa52-402f-a852-0534776b788e.jsonl#20` (repo: joe home dir session) — User corrects
  authentication failure by running /login command, enabling the agent to proceed. *(Same incident's
  resolution — moment_type: correction, not a separate failure.)*

**Verdict: PROPOSE a vault note (rung 1), with an honest scope caveat.** This is a single incident in
this corpus (n=1) — Joe has separately described this pattern (a fresh, valid token flipping between
401 and 200 due to transient server-side instability) in other contexts, but that isn't tied to any
citable source within this audit's own artifacts, so treat "recurring" as Joe's prior experience, not
something this audit's data independently establishes. The proposed note itself is still a reasonable
rung-1 candidate — it's cheap to write and cheap to be wrong about (worst case, an unnecessary retry
before diagnosing something real) — Joe should judge from his own broader experience, not just this
one incident, whether the pattern is common enough to write down.

**Proposed vault note slug:** `transient-auth-failure-retry-not-reconfigure`

**Proposed vault note content:**

> **Situation:** an agent hits an `authentication_failed` error mid-session with credentials that were
> working moments before.
>
> **Action:** try `/login` (or simply retry after a short wait) before assuming a genuine
> credential/config problem — a fresh, valid token *can* flip between failing and working due to
> transient server-side instability. (This audit observed one such incident; treat "how often this
> happens" as your own broader experience, not something this specific note's evidence establishes.)

## Group G — Repo-specific tooling bug, out of scope (2 moments)

- `agent-a64467fe6c8268fdd.jsonl#6` (repo: phone-llm) — First fleet offer command fails with exit code
  1 (baseline stamp error) instead of expected outcomes (changed/current/locally modified).
- `agent-a64467fe6c8268fdd.jsonl#8` (repo: phone-llm) — Second fleet offer command fails with same
  baseline stamp error as first command.

**Verdict: out of scope for this audit's remedy ladder entirely. No vault note or engram fix
proposed.** This is a bug in the phone-llm repo's own "fleet offer" CLI tooling (a baseline-stamp
error) — not a memory-recall gap, and not something a vault note or an engram-repo change could fix.
It needs an actual code fix in the phone-llm repo's own tooling. If Joe wants this pursued, it should
be filed directly against the phone-llm repo, not folded into this audit's proposals.

---

# Rescope Extension (2026-09-04): the 29 moments added by the failure-category re-judgment

**Provenance:** the original run's judgment prompt only asked the `failure_category` question on
`moment_type=="failure"` moments. The Joe-directed rescope (2026-09-03) widened the prompt and
re-judged every previously-unasked moment for real; 29 of those re-judgments came back
`nothing_could_have_caught_it`, growing this document's population from 20 to 49. Of the 29: **26
were judged from their raw transcripts** (`results/rejudge-failure-category.jsonl`,
`failure_category_provenance: 'rejudge-transcript'`) and **3 from engram's chunk index** after
retention had deleted their transcripts (`results/rejudge-failure-category-from-chunks.jsonl`,
provenance `'rejudge-chunks'`: `3d697973-…#27`, `agent-a6faba7b…#13`, `agent-ac2925d5…#40`).
Every re-judgment carries a one-sentence rationale (the original run's judgments had none); the
groupings below derive from each moment's description plus that rationale.

**Caveat carried from the rescope's independent verification:** 15 of these 29 are rework
moments, and every rework moment re-judged from a raw transcript in the whole rescope (14/14)
came back uniformly `nothing_could_have_caught_it` — the prevention-question framing may
under-assign `fixable` to rework. If Joe wants a second look anywhere in this extension, the
rework-heavy groups (D/E extensions, Group J, Group C extension) are where flips would land; the
one individually-flagged candidate is in Group J.

**Bottom line first: the extension proposes 0 new vault notes.** 7 moments are the recovery-side
twins of incidents whose failure half already has a proposed note (Groups D/E extensions below);
22 are genuine no-fixes.

## Group D extension — recovery twins of the size/token-limit incidents (5 moments, covered by the already-proposed `safe-read-chunk-sizing` note)

- `agent-a96aadd41edf00c3d.jsonl#43` (repo: engram, rework) — after the LEDGER.md file-size
  failure (Group D's `#41`), agent pivots to a grep-based query for the specific entry. Rationale:
  routine adaptation to a generic limit on an unfamiliar large file.
- `agent-a8ce598d27fa736b9.jsonl#9` (repo: phone-llm, rework) — after the 299KB size-limit failure
  (Group D's `#7`), agent pivots to `wc` to count lines. Rationale: routine adaptive tool-usage
  recovery.
- `agent-a8ce598d27fa736b9.jsonl#23` (repo: phone-llm, rework) — after the second token-limit
  failure (Group D's `#12`), agent reduces the read window (limit 119 vs prior 832). Rationale:
  routine adaptive retry against a hard limit.
- `agent-a81b80335e1cdd38e.jsonl#9` (repo: phone-llm, rework) — after the fork's read failure
  (Group D's `#6`), pivots to grep-based section-boundary detection. Rationale: routine, low-cost
  operational adjustment.
- `agent-a81b80335e1cdd38e.jsonl#14` (repo: phone-llm, rework) — after the fork's second failure
  (Group D's `#13`), reduces read limit 1004 → 500 lines. Rationale: mechanical retry adjustment.

**Verdict: no NEW fix — already covered by Group D's proposed `safe-read-chunk-sizing` note.**
These 5 are the recovery halves of the exact incidents Group D already proposes a note for. There
is no contradiction between the rejudge's "nothing could have caught it" and Group D's "fixable
pattern": the rejudge asked whether anything could have prevented *this* moment (the recovery
step — correctly executed, nothing to prevent), while Group D's proposal targets the *preceding*
moment (the blind guess-retry that made the recovery necessary). The recovery behavior itself
needs no fix; the incident does, and its note is already on the table.

## Group E extension — recovery twins of the git-convention incidents (2 moments, covered by 2 of the already-proposed Group E notes)

- `agent-a427a4df64476d6c0.jsonl#46` (repo: engram, rework) — agent realizes the old files were
  never git-tracked and adjusts to stage only the two new files (the recovery of Group E's `#40`
  incident). Rationale: minor self-corrected discrepancy, caught immediately via git's own error.
  Covered by the proposed `git-add-cant-stage-untracked-deletes` note.
- `7c95676b-a345-466d-a782-7f9c8df6647b.jsonl#31` (repo: phone-llm, rework) — agent works around
  the pull failure with a stash-pull-pop sequence (the recovery of Group E's `#25` incident).
  Rationale: routine one-off local conflict resolved in two quick tool calls. Covered by the
  proposed `git-pull-rebase-needs-clean-tree` note.

**Verdict: no NEW fix — same recovery-twin logic as the Group D extension.** The incidents' notes
are already proposed; the recoveries themselves were correct.

## Group B extension — external facts memory can't reliably supply (2 moments)

- `agent-ac2925d5bb0aa269d.jsonl#40` (repo: llmcpp, correction, judged from chunks) — Phi-5 is
  pre-release only and Phi-4.5 doesn't exist, contrary to the user's request framing. Rationale:
  a fast-moving external fact requiring live research, not something a prior-session lesson could
  reliably encode.
- `agent-aa31ba62ae2e98519.jsonl#36` (repo: llmcpp, rework) — analysis of llama.cpp v0.31.3
  `server.py` was incomplete; v0.31.2 had added system-prompt caching for non-trimmable caches.
  Rationale: research-discoverable domain knowledge, not prior-session experience memory would
  plausibly store.

**Verdict: correctly categorized, no fix proposed.** Slightly broader than the original Group B
("the data doesn't exist" → "the fact isn't memory material"), but the same class: the vault
cannot encode fast-moving or never-encountered external-world facts.

## Group C extension — self-caught one-off slip (1 moment)

- `agent-ad3f5d4aa55408741.jsonl#40` (repo: phone-llm, rework) — fork re-executes the amendment
  against the correct note (743) after confusing a grep line-number with a Luhmann note ID.
  Rationale: a one-off execution-detail slip, self-caught within the same task via verification,
  not a recurring behavioral pattern.

**Verdict: correctly categorized, no fix proposed.** Same class as the original Group C's trivial
self-corrected tool slip.

## Group H — routine clean successes with no capturable lesson (8 moments, new group)

- `e5ccd18d-c5d3-4593-9c41-61394832e987.jsonl#31` (repo: engram, success) — prioritized open
  issues by the user's criteria; user immediately acted on the top recommendation. Rationale: a
  straightforward success with no mistake, correction, or non-obvious technique to distill.
  **⚠ Flag:** the rescope's independent verification noted this judgment effectively disputed
  `worth_learning_from` (currently `true` on the record) rather than answering reachability.
- `agent-a5e3cc9e58c463d77.jsonl#5` (repo: engram, success) — fork executed its deliberate no-op
  placeholder directive. Rationale: a lessons step would only echo the parent's own progress
  summary.
- `agent-a6faba7b1e62b8aad.jsonl#13` (repo: targ, success, judged from chunks) — reviewer
  independently verified each brief constraint against the diff and issued a well-calibrated
  Approved verdict. Rationale: already-correct, thorough behavior; nothing to fix.
- `agent-a839afe56971d95b7.jsonl#14` (repo: engram, success) — synthesized gathered issue data
  into well-organized output. Rationale: per-instance judgment-call quality, not a generalizable
  reusable lesson.
- `b1ffad9e-5318-418a-8719-442cdffe06c1.jsonl#53` (repo: engram, success) — wrote a high-quality
  proposal.md for #701 with accurate ADR references. Rationale: a demonstration of successful
  execution, not a discrete transferable lesson.
- `agent-a98aa3776b49be8ce.jsonl#116` (repo: phone-llm, success) — comprehensive verification of
  artifact claims; clean pass. Rationale: no discovered issues, pitfalls, or novel technique
  worth distilling.
- `agent-ac2971510a7adecdf.jsonl#46` (repo: engram, success) — multi-window splitting on a real
  623KB transcript produced correct boundaries. Rationale: a validated engineering fact from
  ad-hoc manual testing with no natural learn-call hook.
- `agent-ac2971510a7adecdf.jsonl#50` (repo: engram, success) — window-location joining logic
  filtered events correctly. Rationale: generic verification methodology the agent wouldn't
  reliably recognize and articulate as a capturable lesson.

**Verdict: no fix needed.** These are the loop's non-events: work that went well and holds no
lesson a mandatory completion-report step could usefully capture. (Whether some should have been
`worth_learning_from == false` in the first place — see the flag on the first bullet — is a
judgment-consistency question, not a remedy-ladder one.)

## Group I — already-codified rules correctly applied in the moment (5 moments, new group)

- `agent-a98aa3776b49be8ce.jsonl#9` (repo: phone-llm, success) — absorbed and committed to an
  already-recalled engram lesson (Gate A catches fabricated claims). Rationale: in-session
  application of an existing lesson, nothing novel to capture.
- `agent-a5a8a3513b296c3ab.jsonl#8` (repo: phone-llm, success) — executed recall Step 0 correctly
  (Ask/Situation/Plan upfront). Rationale: following an already-mandatory procedural step the
  skill framework enforces.
- `f4dbacdc-2279-458d-bfd4-22d35c0b01e9.jsonl#35` (repo: engram, success) — cited and complied
  with the CLAUDE.md rule requiring `superpowers:writing-skills` for SKILL.md edits. Rationale:
  following an already-documented rule, not discovering a new insight.
- `agent-a898d11986e0f5882.jsonl#24` (repo: engram, success) — created the RED baseline per the
  writing-skills transient-evidence convention. Rationale: correctly-executed procedural step the
  skill file already prescribes.
- `agent-a6493eb381d2089cc.jsonl#2` (repo: engram, dispatch) — fork executed an explicit,
  well-scoped directive exactly as asked. Rationale: any value lies in the parent's dispatch
  design, with nothing observable from within the fork's own execution.

**Verdict: no fix needed — these are the memory loop *working*, not failing.** The
lesson/rule/procedure already exists in the vault, a skill, or CLAUDE.md and was correctly
applied; there is nothing new for a capture step to write.

## Group J — correct adaptive recovery from one-off or external conditions (6 moments, new group)

- `agent-ac2971510a7adecdf.jsonl#60#rework-1` (repo: engram, rework) — systematically
  re-investigated an auth failure by comparing against 8+ established patterns, confirming the
  root cause. Rationale: this IS the desired debugging behavior, not a mistake to prevent.
- `agent-a53c98dac1552d0d9.jsonl#21` (repo: phone-llm, rework) — re-reviewed the corrected plan
  and retracted a rebutted finding. Rationale: a routine review-and-retraction cycle.
- `agent-aa0a29568578223b9.jsonl#20` (repo: phone-llm, rework) — detected in-flight working-tree
  mutations from a parallel process and pivoted to a clean HEAD copy via `git archive`.
  Rationale: an external environmental condition no prior-session memory could predict; handled
  well.
- `7fa0ee08-4191-46a0-985e-e21af514c450.jsonl#39` (repo: phone-llm, rework) — recognized a
  truncated task-notification result and read the full output file directly. Rationale: inherent
  platform behavior, immediately and correctly handled.
- `6eecc514-5a9e-469d-8fa9-7fed249fb9b8.jsonl#46` (repo: dotfiles, rework) — pivoted from a
  failed brew one-liner to a simpler `find`. Rationale: normal iterative shell debugging.
- `3d697973-ee50-42a8-9657-f57181ca991a.jsonl#27` (repo: targ, rework, judged from chunks) —
  after a failed targ command, searched the filesystem, then succeeded via `gh issue view`.
  Rationale: routine trial-and-error discovery with a quick successful pivot. **⚠ Flag: the one
  individually-disputed judgment in this extension** — this record's own `memory_existed==true`
  conflicts with the "nothing" verdict (neither rejudge prompt passed `finding_side.memory_existed`
  to the judge); plausibly miscategorized, and a flip would move it out of this document into the
  fixable bucket.

**Verdict: no fix needed** (with the explicit flag on the last bullet left to Joe's read). The
recoveries were correct and cheap; the triggering conditions were one-off or external.

## Summary

Of the 49 "nothing could have caught this" moments (20 original + 29 rescope, out of 150 total
audited moments):

- **18 moments are tied to the 5 proposed vault notes** — original: Group D (6 moments, 1 note) +
  Group E (3 moments, 3 notes) + Group F (2 moments, 1 note) = 11 moments; rescope: Group D
  extension (5 recovery twins of Group D's incidents) + Group E extension (2 recovery twins of
  Group E's incidents). **Still 5 distinct proposed vault notes — the rescope extension proposes
  0 new ones** (no new recurring reachable pattern emerged from the 29; the only recurring
  patterns found were the recovery halves of incidents already covered).
- **29 moments are correctly-categorized non-fixes** — original: Group A (3) + Group B (2) +
  Group C (2) = 7; rescope: Group B extension (2) + Group C extension (1) + Group H routine clean
  successes (8) + Group I already-codified rules correctly applied (5) + Group J correct adaptive
  recoveries (6) = 22. No fix of any kind is warranted. Two individually-flagged judgments inside
  this bucket (Group H's `e5ccd18d…#31`, Group J's `3d697973…#27`) are left to Joe's read.
- **2 moments are out of scope for this audit entirely** — Group G, a phone-llm repo tooling bug that
  belongs in that repo's own issue tracker, not this audit's remedy ladder.

No vault writes and no GitHub issues have been made as part of this draft (the rescope extension
included). All five proposed vault notes above are candidates for Joe to accept, edit, or reject
at the audit's review gate (task 6.2).
