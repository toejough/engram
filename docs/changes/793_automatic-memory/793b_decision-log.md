# Automatic Memory - Decision Log

Decisions made while producing #793, for the owner to validate. Decided by: O = owner (Joe, dated); A = author, a derivation or default the owner may override ("A, on review" answers a finding of a review round, numbered by that round: `793b61` is the first group's, `793e6g` the second's; "A, on the docs check" answers the docs check; "A (`793cN`)" is the default of that question); R = reviewer, a review round's own outcome, accepted by the author. Joe was unavailable, so each question the author would have asked has a default here, listed in `793c`. `old 793bN` and `old 793cN` name entries of the earlier attempt (`793_parallel-memory-agent`, as of commit `dcf53d07`).

## 793b1 Start over

Abandon the earlier attempt and produce this change from main in `793_automatic-memory`: "I want to branch off and try something else. I want to start over. Feel free to use what you've learned, but I want to go back to before we had a mod, before this change was implemented." O 2026-10-16.

## 793b2 North Star

The product statement is the owner's words, unchanged: "I want to add automatic learning and memory retrieval." He asked for "one single, focused statement to summarize the product spec and be the North Star", "the headline feature of this change". It opens `793a` and is the one line every artifact serves, though the product guide allows only three sections. O 2026-10-16.

## 793b3 The algorithm and the working rules

Follow the owner's five steps (`793f2`), depth first, committing early and often, autonomously until the author judges the work done: "Write this algorithm down. Follow it. Write the requirements, scenarios, tests, and code. Commit early and often. Work autonomously till you think you're done. Work depth first through the above algorithm." O 2026-10-16.

## 793b4 Delegation

The delegation rule of `793f4`, given a minute after the algorithm and ending: "Write that down, too, and follow it for delegation." O 2026-10-16.

## 793b5 Reading of the delegation rule

The reading in `793f4`: the plain sense of "orchestrate", "search", "work" and "review". A.

## 793b6 The six requirements are the minimal set

The smallest set in which each half of the North Star can be switched on (`793a2a`), relied on (`793a2b` retrieval, `793a2c` learning, `793a2f` the sweep that the main [agent](./793a_product-spec.md#793a1a-terms)'s learn also did), bounded (`793a2d`) and checked (`793a2e`: the owner asked for "user visible and user validatable" capabilities). The display and the install are not folded into others: each has its own promise the user can verify. A.

## 793b7 The interactions are product requirements too

`793a3a`, `793a3c` and `793a3e` are requirements with scenarios, each a way two capabilities work together (step 1: "and how they must interact"); a rule shared with a requirement is stated once and cited. Review round 1 cut three more that restated a rule: `793a3b` into `793a2b`, `793a3d` and `793a3f` into `793a2d`, `793a2e` and `793a2a`; the three IDs are retired, never reused. A.

## 793b8 Exclusions

Left out of the minimal set, each a possible later change: [subagent](../../GLOSSARY.md#subagent) start and stop; route results; the chunk-index deep lookup (about 12 seconds on the owner's [vault](../../GLOSSARY.md#vault) against 1.0 to 1.4 for [notes](../../GLOSSARY.md#permanent-note) alone, `old 793b408`, beyond the 8 seconds of `793b17`; the [agent](./793a_product-spec.md#793a1a-terms) finds fragments when it asks, `793a2f`); backlog capture of old [transcripts](../../GLOSSARY.md#transcript); curation of [pending offers](../../GLOSSARY.md#offer) (lessons go straight into the vault, `793c5`); a model-made relevance judgement before delivery; reinforcing a covered note (`793a3c`); a status line (the owner moved the earlier attempt's content into the conversation, `old 793c90`); the Pi [harness](../../GLOSSARY.md#harness) (`793a2a`); memory at mid-task moments, which the main agent chose by judgement and issue #793 asks for (`793c12`). None is needed for the North Star; the list lives here, not in the product spec. All but the status line and Pi reverse earlier owner answers (`793b36`, `793c13`). A.

## 793b9 Test execution

Each product scenario is run by one Go test named with its ID, driving a real Claude Code [session](../../GLOSSARY.md#session) in an isolated HOME (`793b32`), its steps mirroring the Given, When and Then lines and its run output the record; each technical scenario by one test named with its ID; a conformance check asserts that every scenario ID has a test. Only a real session proves what the user sees. A.

## 793b10 Departure: depth-first fill

`793a`, `793d` and `793e` stay partial ("Pending" placeholders, empty catalog tables) until the last node has been through step 5, against the guides' rule to start from approved specifications: "Work depth first through the above algorithm." (`793b38`). O 2026-10-16.

## 793b11 Departure: no interview or approval gates

No interview or approval gates: the author works autonomously and the owner reviews at the end, reading "Work autonomously till you think you're done." as dropping them; each question has a default (`793c`). A.

## 793b12 Departure: no option analysis

`793d` has no solution-space analysis, option impact or alternative options: step 3 takes the minimal components a scenario needs. A (reading of step 3).

## 793b13 Departure: one refactor pass, and extra catalog columns

One refactor commit per test, with the statuses `Not started`, `Red`, `Green-all` and `Done` (the refactor commit sets `Done`); the `793e` catalog gains Red, Green and Refactor commit columns, each hash filled in by the next commit (a commit cannot contain its own hash). The single pass is the author's reading of step 5, the columns his own choice. A.

## 793b14 Departure: the North Star line

Folded into `793b2` in review round 1.

## 793b15 Artifact names

Withdrawn in review round 1: not a decision (`docs/AGENTS.md` dictates the file names).

## 793b16 One subject term

Withdrawn in review round 1: not a decision.

## 793b17 A prompt is held for at most 8 seconds

A [prompt](./793a_product-spec.md#793a1a-terms) is held at most 8 seconds (`793a2d`): the earlier attempt's author set it (`old 793b317`), the owner did not. It overrides his rule, no longer than the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup on the same [vault](../../GLOSSARY.md#vault) and input (`old 793c7` item 4: 10.6 to 11.0 seconds on his), because when a lookup with the chunk index took 12 seconds he was offered a longer hold, did not raise it and had the lookup split (`old 793c87`, 2026-10-14); the alternatives are in `793c1`. A.

## 793b18 Writing finishes within 2 minutes of a turn

A lesson's [note](../../GLOSSARY.md#permanent-note) is in the [vault](../../GLOSSARY.md#vault) (`793a2c`), and the conversation findable (`793a2f`), within 2 minutes after the [turn](./793a_product-spec.md#793a1a-terms) ends: both promises need a testable number, and 2 minutes is a guess to be replaced by a measurement (`793c2`). A.

## 793b19 Volume of delivered notes

`793a2b` states no number of [notes](../../GLOSSARY.md#permanent-note); the technical spec limits delivery to what the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup returns, the owner's earlier answer (`old 793c7` items 1 and 2); a fixed maximum is open in `793c3`. A.

## 793b20 No modes and no off switch

[Automatic memory](./793a_product-spec.md#793a1a-terms) is on whenever installed (`793a2a`), with no mode and no off switch; stopping it means removing what [`engram update`](../../GLOSSARY.md#engram-update) added, and no requirement covers how (row 10 of `793b36`). It follows the owner's earlier answer (`old 793c12`). A.

## 793b21 Lessons are written straight into the vault

A lesson becomes a [note](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault) directly (`793a2c`), not a [pending offer](../../GLOSSARY.md#offer) awaiting review: the owner's earlier answer, "written directly to the vault" (`old 793c4`); curation is excluded (`793b8`) and open in `793c5`. A.

## 793b22 The scope is top-level sessions and notes only

`793a2b` and `793a2c` apply to the [prompts](./793a_product-spec.md#793a1a-terms) and [turns](./793a_product-spec.md#793a1a-terms) of the user's own [sessions](../../GLOSSARY.md#session), not [subagents](../../GLOSSARY.md#subagent), and delivery looks up [notes](../../GLOSSARY.md#permanent-note) only, not the chunk index (a lookup with it took about 12 seconds), reversing earlier owner answers (rows 1, 2 and 5 of `793b36`; `793c6`, `793c13`). A.

## 793b23 No capture rate or precision targets

`793a` states no capture-rate or precision target; the scenarios of `793a2c` are the proof (step 2). The owner moved such targets to issue #850 in the earlier attempt (`old 793b182`) and this change follows. A.

## 793b24 A note holds no secret

`793a2c` says a [note](../../GLOSSARY.md#permanent-note) MUST NOT contain a credential, token, key or password from the [turn](./793a_product-spec.md#793a1a-terms); how it is met and the risk that remains are the technical spec's. It applies the owner's earlier decision (`old 793c7` item 10) and the residual risk he ratified: secrets of shapes the patterns miss can reach the model provider and the [vault](../../GLOSSARY.md#vault) (`old 793b324`, `old 793d19a`); the earlier attempt's product spec had dropped the clause (`old 793b241`). O 2026-10-11 and 2026-10-13 (earlier attempt).

## 793b25 What is shown after a turn, and when

`793a2e` shows something only when [automatic memory](./793a_product-spec.md#793a1a-terms) delivered or wrote a [note](../../GLOSSARY.md#permanent-note) or could not (open: `793c9`; a [turn](./793a_product-spec.md#793a1a-terms) with nothing to report shows nothing, as the earlier attempt omitted zero counts and a line on every turn would be noise); by the next [prompt](./793a_product-spec.md#793a1a-terms) the user can see what was delivered for the previous prompt, what was written from the previous turn and each failure with its reason, and a later write is shown when it finishes. A write can finish up to 2 minutes after the turn (`793b18`), so the next prompt gives the promise a time to check. A.

## 793b26 The line waits for the turn's work

Folded into `793b25` in review round 1.

## 793b27 "Already has" is per session

In `793a2b` the [agent](./793a_product-spec.md#793a1a-terms) already has a [note](../../GLOSSARY.md#permanent-note) when it was delivered earlier in the same [session](../../GLOSSARY.md#session) or the agent looked it up there, even after a compaction: the literal reading of "this session", which needs no knowledge of what the agent still holds. It reverses the owner's earlier answer (`old 793c7` item 5; row 11 of `793b36`; `793c10`, `793c13`). A.

## 793b28 Depth-first order

The groups, in order: `793a2a` with `793a2b` and the 8-second hold of `793a2d`; `793a2c`; `793a2f`; the rest of `793a2d`; `793a2e`. The first node is the first scenario of `793a2a`, whose only user-visible proof is the first delivery working in a new [session](../../GLOSSARY.md#session); the hold goes with the first delivery so that none exceeds `793b17`; the lessons are the learning half of the North Star, the sweep the rest of what the main [agent](./793a_product-spec.md#793a1a-terms)'s learn did (`793b34`), and the bounds and the display come last because they bound and report the others. An interaction is worked with the first group that has every capability it names (`793a3a`, `793a3c`, `793a3e` with the second). Rejected: the requirements' listed order, with the install alone first (its scenarios cannot run before a delivery exists); all interactions in a final node (their rules stay unchecked); each interaction at its own earliest capability (it cannot be exercised there). A.

## 793b29 Interactions join the first node that has every capability they name

Merged into `793b28` in review round 1.

## 793b30 Commit hashes are recorded by the next commit

Folded into `793b13` in review round 1.

## 793b31 Glossary terms stay in the Terms list until promotion

Withdrawn in review round 1: not a decision.

## 793b32 Test isolation

The isolation rule of `793f7`, restated in `793e2` and `793e7`, binds every [agent](./793a_product-spec.md#793a1a-terms)'s commands: the owner's rule from the earlier attempt, set between 2026-10-11 and 2026-10-16 and relayed to the author, "temp HOMEs only; every engram/claude command from agents needs explicit env HOME" (`old 793b245` records the first half). Review round 1 found it missing: a literal run of "the user runs [`engram update`](../../GLOSSARY.md#engram-update)" would install into the real `~/.claude`. O 2026-10-11 to 2026-10-16 (earlier attempt), restated after review round 1.

## 793b33 One agent review round per phase

One [agent](./793a_product-spec.md#793a1a-terms) review round per phase unless the owner asks for more; the author accepts or declines each finding and findings are not argued to resolution, superseding the root `AGENTS.md` default. The owner: "one agent review round per phase unless asked." (`old 793b174` records the same cap). O 2026-10-11 (earlier attempt).

## 793b34 The goal: what the main agent was already doing, just async

What the main [agent](./793a_product-spec.md#793a1a-terms) was already doing, made async, is the minimal set: `793a2b` is the query before acting, `793a2f` the sweep, `793a2c` the [notes](../../GLOSSARY.md#permanent-note) for explicit lessons; mid-task re-querying is not in the set (`793b8`). The owner: "The goal was to do what the main agent was already doing, just async." O 2026-10-16.

## 793b35 The transcript sweep is in the minimal set

`793a2f` is a requirement: within 2 minutes after a [turn](./793a_product-spec.md#793a1a-terms) the [session](../../GLOSSARY.md#session)'s conversation is findable by the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup, without anyone running a sweep; only the chunk-index deep lookup stays out (`793b8`). Leaving the sweep out automates half of learning and lets the chunk layer go stale. A, on the reviewer's recommendation (opus round 1, 2026-10-17).

## 793b36 Earlier owner answers this minimal set reverses

The minimal set reverses these earlier owner answers, each needing a measurement, a second model judgement or machinery the set leaves out; `793c13` asks the owner about all of them. A.

| # | What | Earlier owner answer | Here |
| --- | --- | --- | --- |
| 1 | [Subagents](../../GLOSSARY.md#subagent) | capture inside subagent [sessions](../../GLOSSARY.md#session) (`old 793c6`, 2026-10-11); delivery at a subagent's start, capture at its stop (`old 793c63` (a), 2026-10-11) | the user's own sessions only (`793b22`) |
| 2 | [Prompt](./793a_product-spec.md#793a1a-terms) sources | memory for every prompt whatever its source (`old 793c5`, 2026-10-11) | prompts the user sends; until `793e6b` the mod delivers for every origin (`793b58`) |
| 3 | Route results | tier evidence before the orchestrator routes a unit (`old 793c39` (b), `old 793c64` (a), 2026-10-11) | none |
| 4 | Mid-task moments | delivery at the moments where recall should fire (`old 793b187`, 2026-10-11; `old 793b240`, 2026-10-12; `old 793c49` (b), `old 793c55` (a), 2026-10-11) | the prompt only (`793c12`) |
| 5 | Chunk stubs | conversation fragments beside the [notes](../../GLOSSARY.md#permanent-note) (`old 793c15` (b), 2026-10-11), kept by a background pass when the full lookup proved too slow (`old 793c87` (a), 2026-10-14) | none delivered |
| 6 | Relevance filter, moment classification | relevance filter and moment classification, two model judgements before delivery (`old 793c69` (a), 2026-10-12), on haiku at the lowest effort (`old 793c79` (a), 2026-10-14) | none |
| 7 | Curation | indexing and curation after each [turn](./793a_product-spec.md#793a1a-terms) (`old 793b240`, 2026-10-12), curation run by the System (`old 793c43` (a), 2026-10-11) | indexing is back as `793a2f`; curation of [pending offers](../../GLOSSARY.md#offer) is out |
| 8 | Backlog capture | backlog capture: leftover sessions of any [harness](../../GLOSSARY.md#harness) picked up by the next session (`old 793c67`, 2026-10-12; `old 793c65` (a), 2026-10-11; cut-off `old 793c75`, 2026-10-13) | none (`793b37`) |
| 9 | Reinforcing a note | a MUST (`old 793c14` (b), 2026-10-11), no longer named in the intent of `old 793b240` | a lesson the [vault](../../GLOSSARY.md#vault) already states is not written again, nothing more |
| 10 | Stopping it | turned off by uninstalling, with uninstall extended to remove it (`old 793c18` (a), `old 793c22` (b), 2026-10-11) | no requirement covers how (`793b20`) |
| 11 | "Already has" | a note in the receiving [agent](./793a_product-spec.md#793a1a-terms)'s context is never re-injected, and a truncated note is not present (`old 793c7` item 5, 2026-10-11) | per session, even after a compaction (`793b27`) |
| 12 | The prompt hold | no longer than the agent's own lookup on the same vault and input (`old 793c7` item 4, `old 793c11` (b), 2026-10-11) | 8 seconds (`793c1`) |
| 13 | Sessions covered | the promise covers the sessions where the earlier attempt's mod runs, and [`engram update`](../../GLOSSARY.md#engram-update) reports when mods cannot load (`old 793c66` (a), 2026-10-11) | every Claude Code session started after the update (`793a2a`); where it cannot run, `793a2d` has it step aside, and no check is required (`793c33`) |
| 14 | Activity for the agent | the activity lines may reach the agent (`old 793c90` (b), 2026-10-15), and the previous turn's sentences ride the next prompt (`old 793c92` (b), 2026-10-16) | the user sees the activity in the session (`793a2e`); nothing carries it to the agent |

## 793b37 A turn followed by the end of the session is not covered

A [turn](./793a_product-spec.md#793a1a-terms) followed by the end of the [session](../../GLOSSARY.md#session) within the 2 minutes is not covered by `793a2c` and `793a2f` (`793c14`): covering it needs work that outlives the session, a pick-up by a later session or a hold on the exit, which the minimal set excludes (the earlier attempt recorded such a turn as lost, `old 793b203`). The cost: a lesson in the last turn before an exit can be lost. A.

## 793b38 Depth first: two readings, and the one this change takes

Depth first is read per scenario: step 2 per requirement, then each product scenario (or a tightly coupled pair) is a node through steps 3 to 5 before the next (`793f3`, `793e4`), because step 3 reads "for each acceptance test". The per-requirement reading, which the first version took, was rejected. A (reading of step 3), on review round 1's finding.

## 793b39 Review round 1 of step 1

One opus review round on the step 1 documents (seven commits, `1bcf81b3` to `57372f32`): approve with fixes, all accepted. Critical: nothing kept literal runs and `engram` or `claude` commands off the owner's real home, [vault](../../GLOSSARY.md#vault), `~/.pi` and GOBIN (`793b32`, `793f7`, `793e2`, `793e7`). Major: three departures credited to the owner were the author's readings (`793b11` to `793b13`); the goal, the mid-task exclusion, the reversed owner answers, the secrets MUST NOT and the display were missing or vague (`793b34`, `793b8`, `793b36`, `793b24`, `793b25`); three of the six interactions restated a rule (`793b7`); `793a2d` forbade changing "what the [agent](./793a_product-spec.md#793a1a-terms) answered", now editing the user's [prompt](./793a_product-spec.md#793a1a-terms) and the text of the reply. Minor: the 8-second bound was called the owner's and the 2-minute delay lacked "a measurement replaces it" (`793b17`, `793b18`); `793a3a` said "later prompts", `793a2b` did not say which [notes](../../GLOSSARY.md#permanent-note) bear on a prompt, `793a2a` had untestable or missing rerun, other-[harness](../../GLOSSARY.md#harness) and dry-run clauses, `793a2c` named three kinds of lesson for the [skill](../../GLOSSARY.md#skill)'s four; defined words stood where a number belonged; the problem statement paraphrased the North Star and held a paragraph about the process; own [sessions](../../GLOSSARY.md#session) and a [turn](./793a_product-spec.md#793a1a-terms) followed by an exit were unaddressed (`793b37`); a section beyond the guide's three is now `793b8`, and `793c4`, `793c7` and `793c11` did not block; seven entries were no decision (the stubs); the commit columns were called the owner's record (`793b13`); the depth-first reading was unexamined (`793b38`), `793b5` rested on silence about reviews (`793b33`), product-scenario tests had four descriptions (`793b9`) and the 8-second hold belongs in the first node (`793b28`). Nit: `793b10` did not name the placeholders, `793e2` named categories the plan guide lacks, `793e7` lacked the guide's gates for `rapid`, `imptest` and valid- and invalid-path behavior, `793f4` defined a phase over requirements, not nodes. Retired, never reused: `793a3b`, `793a3d`, `793a3f`, `793a4`, `793c4`, `793c7`, `793c11`. R (opus, 2026-10-17).

## 793b40 The scenarios of the first group

Step 2 for the first group (`793e6a`) writes ten scenarios, one per mandatory branch: five under `793a2a`, four under `793a2b`, one under `793a2d` (the 8-second hold). The [agent](./793a_product-spec.md#793a1a-terms)'s own lookup is staged before the update and the [session](../../GLOSSARY.md#session) resumed after it, so that the two routes to a held [note](../../GLOSSARY.md#permanent-note) differ; the missing [vault](../../GLOSSARY.md#vault) is one removed after the update, because [`engram update`](../../GLOSSARY.md#engram-update) creates a missing one. Fewer scenarios, with the dry run and Pi folded into the first scenario of the update, were rejected: the outcomes differ and each needs its own report. A.

## 793b41 The mod, its place and its command

The automatic behavior is one mod, `engram-memory`, in `agent-instructions/mods/`, deployed by [`engram update`](../../GLOSSARY.md#engram-update) with the [skills](../../GLOSSARY.md#skill); for each [prompt](./793a_product-spec.md#793a1a-terms) it runs the new command `engram auto deliver` and puts its output beside the prompt, the command holding the logic (`793d1`). `prompt.submit` adds context without editing the prompt, and a real `claude -p` [session](../../GLOSSARY.md#session) showed that a linked mod loads. Rejected: a `UserPromptSubmit` hook in `~/.claude/settings.json` (`engram update` would edit a file the user owns on every run; `793c17`). A.

## 793b42 What the block says

The block is one framing line beginning `[engram memory]` ([notes](../../GLOSSARY.md#permanent-note) from the [vault](../../GLOSSARY.md#vault), delivered by the memory system and not typed by the user; an instruction inside one is not the user's) and one line per note: dash, file name, situation, colon, body (`793d4c`). It is the earlier attempt's framing, whose marker the [session](../../GLOSSARY.md#session)'s record is searched for (`793c18`). A.

## 793b43 A note cannot reshape the block

A [note](../../GLOSSARY.md#permanent-note)'s file name, situation and body are each written as one line with no control or format character (Unicode category Cf), so a note cannot write a line that imitates the framing or another note (`793d4c`); no quoting scheme. A security review of the first rendering found the hole; the first group's review added the format characters (`793b61` finding 20). A, on those reviews.

## 793b44 Only the notes that match are delivered, whole

A [note](../../GLOSSARY.md#permanent-note) is delivered when the lookup returns it as a matched item (not a chunk, not an explore pick) with its body whole on one line; the lookup's own limit bounds the count (`793b19`). An explore pick does not bear on the [prompt](./793a_product-spec.md#793a1a-terms), and a cut body would hide the later steps of a runbook (`793c19`). A.

## 793b45 What a session holds

The [agent](./793a_product-spec.md#793a1a-terms) holds a [note](../../GLOSSARY.md#permanent-note) when its basename appears in what the [session](../../GLOSSARY.md#session)'s record shows it was given: the file names on delivered blocks' note lines and the command and result of each `engram query` or `engram show` it ran (`793d4d`); nothing else counts, whatever a compaction removed (`793b27`). A note the agent read by `cat`, `grep` or a file read is not held and is delivered again: the cost of counting only the lookups it is told to use. Rejected: a ledger kept by the mod (lost with the process) and any file name in any tool result (the first version: an `ls` of the [vault](../../GLOSSARY.md#vault) suppressed a note never read; `793c20`). A, with the pairing and the basename from the first group's review.

## 793b46 The command is ended at 7 seconds

The mod waits for `engram auto deliver` at most 7 seconds, by the process call's time limit; a [prompt](./793a_product-spec.md#793a1a-terms) whose command did not finish goes on with nothing added and no sign of failure. The 8 seconds of `793b17` leave the second for process start and hooks (a 20-second command held the prompt 7.07 seconds); the platform's hook limit does not count the call (`793c21`). A.

## 793b47 A dry run reports the mod as a mod

The report of [`engram update`](../../GLOSSARY.md#engram-update), dry run or not, lists a mod in the Claude Code section with `agent-instructions/mods/<name>/` as its source (a [skill](../../GLOSSARY.md#skill) has `agent-instructions/skills/<name>/`); a dry run plans and reports the mod and writes none of it (`793a2a1c`). The first run showed the mod under `agent-instructions/skills/` (`793c22`). A.

## 793b48 The conformance check reads this change's specs and fails when they are gone

The conformance check (every scenario ID has a `Test<ID>_...` function in a `_test.go` file whatever its build tags, read with the Go parser, not as text) is a test of the dev integration suite, so `targ check-full` runs it; it reads this change's specs and fails, naming the cause, when they are missing; at promotion, when the directory is deleted, it moves to read the promoted specs or is deleted with the change. One direction only. The first version searched text, counted string-literal fixtures and stayed green when the real `Test793a2a1a_` and `Test793d4b1h_` were renamed. The allow list of `dev/golangci-lint.toml` gains `go/ast`, `go/parser` and `go/token` for non-test files, the three it already allows for tests; no rule is switched off. Rejected: a target of its own (it could be forgotten), a skip when the specs are missing (it passes with nothing checked), and both directions (other changes' tests carry other IDs). A, on review.

## 793b49 The acceptance run is capped at four tests at once

`targ acceptance` runs at most 4 tests at once (`go test -parallel=4`): at 10 at once two tests failed per run, at 8 one, at 6 and below none; 4 leaves a margin below 6 and costs about 40 seconds for the full set, while one at a time costs about four minutes for no gain, and the number of cores (10 here) is where tests failed. The review found the stall is between the command's spawn and its first instruction, so a loaded user machine could hold a [prompt](./793a_product-spec.md#793a1a-terms) about 7 seconds with nothing delivered: an open risk (`793c32`). A.

## 793b50 The mod uses what the command printed only when it succeeded and printed a block

The mod adds the command's output only when it exited 0 and the output begins `[engram memory]` (`793d2a`); the first version added any output, and the main binary's usage text reached the [agent](./793a_product-spec.md#793a1a-terms). A, on the first group's review.

## 793b51 The delivery finds the records where Claude Code writes them

`engram auto deliver` reads records from `--sessions`, else `ENGRAM_TRANSCRIPT_DIR`, else `$CLAUDE_CONFIG_DIR/projects`, else `<home>/.claude/projects` (`793d4f`); [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) keeps its order ([792b17c6 Session-log directory](../../specs/technical/ingestion-source-discovery.md#792b17c6-session-log-directory); the same order for the sweep would change a canonical requirement, a separate change). A [session](../../GLOSSARY.md#session) started with `CLAUDE_CONFIG_DIR` writes there, so reading only the home never found the record. A, on review.

## 793b52 The delivery runs its lookup through an injected function

`AutoDeliverDeps.Query` is an injected function that runs the lookup for the arguments it is given (the [prompt](./793a_product-spec.md#793a1a-terms)'s text as message and only phrase, `793d4a`), so a test can see them; `793e7` asks `imptest` evidence of every injected collaboration, and a delivery that left the message out passed every test; a waiver of the gate, with the arguments tested through the real engine, was rejected for that reason. A, on review.

## 793b53 A missing login or Claude Code fails an acceptance test

Acceptance tests fail, naming the cause, when `claude` is not on the path or no login is found (`793e2`), and do not skip: `targ acceptance` exited 0 having run nothing; a skip with a target that fails on any skip was rejected, as the tests run only when somebody asks for them. A, on review.

## 793b54 Two acceptance scenarios prove more than that nothing arrived

In `793a2b1c` the [vault](../../GLOSSARY.md#vault) gains a second [note](../../GLOSSARY.md#permanent-note) after the [agent](./793a_product-spec.md#793a1a-terms)'s lookup, and the test of `793a2a1e` asserts that the mod ran its command; both used to pass when the mod never loaded. The Given and Then of `793a2b1c` change; the text of `793a2a1e` does not. Rejected: a separate scenario for the second note (the plan guide has one test per scenario ID, and the held rule is what `793a2b1c` is for) and a marker in the text of `793a2a1e` (that the mod ran is not something the user checks). A, on review.

## 793b55 The mod's JavaScript is parsed by a test of the dev integration suite

`targ check-full` parses every JavaScript file under `agent-instructions/mods/` with `node`, which reads each file on standard input as an ES module so that the result does not depend on how a node release detects the syntax of a `.js` file, through a test of the dev integration suite (as the conformance check runs), and fails, naming node, when node is missing (a check that cannot run must not pass, and a skip would pass unchecked): no gate parsed `register.js`. A new target would change the aggregates `Test794b4a1c` pins, and a parser in Go would be a new dependency for what node does. A, on review.

## 793b56 The 7 seconds bound the whole hook, not the command alone

The mod races the call for the [session](../../GLOSSARY.md#session)'s ID and the command against one 7-second timer (`793d2a`), and the command is also ended at 7 seconds from its start (`793b46`); `await $.session.id()` ran outside the bound, and the 8 seconds of the promise hold for any reason, so a timer over the command alone was rejected. A, on review.

## 793b57 The delivery looks up the local vault only, until a later node

The first group's delivery looks up the local [vault](../../GLOSSARY.md#vault) only (`RunQuery`), where `engram query` also merges the parent vault (`ENGRAM_PARENT`); the merge is the plan node `793e6f` (`793c34`). A, on review (`793b61` finding 4).

## 793b58 The first group delivers for every origin of a prompt

Until `793e6b` the mod delivers for every origin of a [prompt](./793a_product-spec.md#793a1a-terms); `793e6b` takes the clause of `793a2d` that [automatic memory](./793a_product-spec.md#793a1a-terms) MUST NOT deliver to or write from its own [sessions](../../GLOSSARY.md#session), with the origin filter, being the first group that could start such sessions; a filter before it had no prompt of another origin to tell apart. A, on review (`793b61` findings 5 and 6).

## 793b59 The builder ran the tests once with the real home

The gate of `793e7` that nothing touched the owner's real `~/.claude`, `~/.local/share/engram`, `~/.pi` or GOBIN is qualified, not dropped: the builder ran `targ test-for-fail` once with the real home (2026-10-17 00:10:49 to 03:41:40 EDT), and the review found nothing under `~/.claude`, `~/.pi`, GOBIN or the [vault](../../GLOSSARY.md#vault)'s top level changed in that window; the only vault files in it are [notes](../../GLOSSARY.md#permanent-note) 1205 to 1208 with their sidecars, written by the parent [session](../../GLOSSARY.md#session), and its chunks all map to the owner's own [transcripts](../../GLOSSARY.md#transcript). R, recorded by the author.

## 793b60 The restated sentences of 792a2e are kept as they are

`793a5a` keeps the sentence of the canonical [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) about a new [vault](../../GLOSSARY.md#vault)'s files (`.obsidian/app.json`, `.gitignore`, `README.md`, `.engram/`) verbatim, though the product guide keeps file names out and the docs check flags them: rewording would change the canonical requirement under a Modified operation. A, on the docs check.

## 793b61 Review of the first group

One opus review round on commit `e62be9a4` found 3 major, 16 minor and 7 nit items (numbered 1 to 26); all were accepted but 25 and applied in a fix pass of new commits, none rewritten: `793b43`, `793b45` and `793b48` to `793b60`, with the entries `793c19`, `793c32` and `793c33`. Also applied: 10, tests that resolved a directory under the real home name their own and stub the environment, with no guard test (other tests rely on the real `Getenv`, and a guard would need an opt-out in each); 15, `793a2b` names the [notes](../../GLOSSARY.md#permanent-note) that bear on a [prompt](./793a_product-spec.md#793a1a-terms) by a term the user can check; 26, a test that took its lock in the source directory (found by bisecting the 1,660 tests of the package) has a temporary [vault](../../GLOSSARY.md#vault); and `Test792b17c6a1`, which read the machine's environment and failed under the `ENGRAM_TRANSCRIPT_DIR` of `793f7`, stubs it. 25 is declined: two red commits carry unrelated [harness](../../GLOSSARY.md#harness) edits, and commits are not rewritten. R (opus), accepted by the author except 25.

## 793b62 The scenarios of the second group

[Automatic memory](./793a_product-spec.md#793a1a-terms)'s second group (`793e6b`): step 2 writes fourteen scenarios, one per clause without one (seven under `793a2c`, two under `793a2a`, one each under `793a2d` and `793a3a`, two under `793a3c`, one under `793a3e`). A [session](../../GLOSSARY.md#session) whose [agent](./793a_product-spec.md#793a1a-terms) may not run engram commands proves a [note](../../GLOSSARY.md#permanent-note) was written "by automatic memory and not by the agent". Fewer scenarios, folding the two of `793a3c` or the three kinds of lesson from the agent's own words, were rejected: the clause lists the kinds, and "whoever wrote that note" names two writers a test can tell apart. A.

## 793b63 A turn is judged by one model call, and no session is started

After a [turn](./793a_product-spec.md#793a1a-terms) of a [prompt](./793a_product-spec.md#793a1a-terms) the user sent, the mod runs `engram auto judge`, makes the model call through the mod's model API (`$.model.complete`, `haiku`, effort `low`: the user's own login pays, no key or account) and runs `engram auto write` with the reply, apart from the turn and each bounded. The call raises no prompt or turn, so [automatic memory](./793a_product-spec.md#793a1a-terms) starts no [session](../../GLOSSARY.md#session) of its own and the own-sessions clause of `793a2d` holds by construction (`793a2d1b` observes it), departing from the plan's expectation of a session of its own (`793b58`). Rejected: a headless `claude -p` session with an environment marker (a second process loading the user's configuration per turn) and `$.model.fork` (it sends the whole conversation). A (`793c38`).

## 793b64 The prompts the user sends

A [prompt](./793a_product-spec.md#793a1a-terms) is the user's when Claude Code stamps its origin `composer`, `bridge`, `slack-ping` or `sdk`; every other origin (a background task's notification, a scheduled trigger, a peer or coordinating [session](../../GLOSSARY.md#session), another plugin's prompt, none) is not. The mod passes the origin to `engram auto deliver` and `engram auto judge`, which decide (`793d4m`); the mod itself leaves out [subagent](../../GLOSSARY.md#subagent) [turns](./793a_product-spec.md#793a1a-terms). The event's origin is the one source: the record spells it differently and does not exist yet at delivery (`793c39`). A.

## 793b65 What the judgement reads, and what it is asked

The model request holds the instructions and a window of the [turn](./793a_product-spec.md#793a1a-terms) with up to 6 lines before it (`793d4g`), no tool inputs or results (they hold credentials and bulk; the whole turn with tool results and a smaller window, the turn alone, were rejected), every text redacted before the cut (`793b69`). The instructions are the earlier attempt's, plus a sentence that an approach the [agent](./793a_product-spec.md#793a1a-terms) was unsure of, tried and saw succeed is a lesson (a probe: 11 of 20 runs without it, 20 of 20 with it), plus the learn [skill](../../GLOSSARY.md#skill)'s bar (`793b87`); a reversal of the agent's own claim is a `fact`, since the window of one turn holds the conclusion and what overturned it, not the cost or the guard that a feedback needs (`793c57`); a request to remember is a lesson even when it holds a secret or the agent says it did not save it (`793b90`). A.

## 793b66 What a lesson becomes

A lesson becomes a feedback or fact [note](../../GLOSSARY.md#permanent-note) through the learn path of [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), at the top of the order of the [vault](../../GLOSSARY.md#vault), with the source `written automatically (session <the session's ID>)`; a reply that is not a list, or a lesson that lacks a field or has an unknown kind, writes nothing and is recorded (`793d4h`). The learn path is the one writer of notes, so links and replacements are not written. A.

## 793b67 What automatic memory records

One JSON record per outcome of a reply (`written`, `skipped`, `nothing`, `failed`) in the engram data directory, below the [session](../../GLOSSARY.md#session)'s ID (`793d4k`); a record that cannot be written does not fail the command. One file per effect needs no lock, and the records are made last, so a test of an absence can wait for them. A.

## 793b68 A lesson an earlier note states

The request lists the [notes](../../GLOSSARY.md#permanent-note) nearest to the [turn](./793a_product-spec.md#793a1a-terms) (the lookup for the first 500 characters of the user's [prompts](./793a_product-spec.md#793a1a-terms), one line each, redacted, cut to 400 characters); the model may name as `covered` a note that already states a lesson, and the writing skips and records that lesson only when the name is a plain file name of a file in the [vault](../../GLOSSARY.md#vault). Whether a note states the same lesson is the model's judgement, not a distance threshold's. A.

## 793b69 Credentials are redacted by shape, and the rest is a known risk

Redaction by shape, before the cut, in the judgement and the writing (the shapes: `793d4j`). Secrets of other shapes are a known risk (`793c41`), and the earlier attempt's wrapped and percent-encoded patterns are not ported; the instruction not to write a credential is not a proof. Amended by `793b78`. A.

## 793b70 The capture reports its own failures, and nothing times it as a whole

After a [turn](./793a_product-spec.md#793a1a-terms) the mod gives the event's result back at once and runs the judgement, the model call and the writing apart from it; a failed step is reported to the writing as a `failure` of up to 300 characters naming the step, and recorded. Each stage has its own bound (`793b85`), 115 seconds together; no timer bounds the chain (it would end a write about to finish, and nothing waits for it), and a failed step is not retried (a second model call, and the next turn captures its own lessons). In 246 live [notes](../../GLOSSARY.md#permanent-note) the note was in the [vault](../../GLOSSARY.md#vault) 0.9 to 6.0 seconds after the turn (median 1.8), so 2 minutes is a bound under load and not the usual time (`793c2`). A.

## 793b71 The commands of the group auto take no first-use step

`engram auto judge` and `engram auto write`, like `deliver`, create no [vault](../../GLOSSARY.md#vault) (`793d4e`). The learn path fails at the missing vault's lock, so the lesson is not written, the failure is recorded and the [turn](./793a_product-spec.md#793a1a-terms) has cost a model call (`793c43`). Superseded in part by `793b82`. A.

## 793b72 The origin of a turn

The mod keeps the origin of the last [prompt](./793a_product-spec.md#793a1a-terms), binds it to the next [turn](./793a_product-spec.md#793a1a-terms) by the turn's ID and gives it to the judgement; a [subagent](../../GLOSSARY.md#subagent)'s turn is not captured, and with no origin none is given. Superseded in part by `793b76`. A.

## 793b73 The tests of an absence wait for the end of the write, and for a settling time

A test that a [note](../../GLOSSARY.md#permanent-note) was not written waits for the [turn](./793a_product-spec.md#793a1a-terms)'s records (made last) and 5 seconds more before counting notes (the writes of one turn end within a second of each other, and an absence proved at the first record would pass a write about to put a second note beside it; waiting the whole 2 minutes would cost that for every absence); a test that a note is delivered waits for the records too, because a lookup in a [vault](../../GLOSSARY.md#vault) whose only note is being written finds no embedding for about 0.3 seconds. A (`793c44`).

## 793b74 A row the harness wrote is no prompt

A record row is a user's [prompt](./793a_product-spec.md#793a1a-terms) only if it is not marked meta, has no origin the user does not send (`793b64`), and keeps text after the cleaning of [792b15c3 Harness noise](../../specs/technical/transcript-reader.md#792b15c3-harness-noise), which the judgement reuses; otherwise a [skill](../../GLOSSARY.md#skill)'s body, a task notification and a local command's caveat, name or output were each a `User:` line and could begin the [turn](./793a_product-spec.md#793a1a-terms). Rejected: the origin and the flag alone (it leaves the command rows) and a list of marks of its own (it copies a rule that has an owner). A, on review (`793e6g` finding 1).

## 793b75 The record is read once, and the lines of the window are the only ones redacted

The judgement reads the record once and redacts, writes as one line and cuts only the context and [turn](./793a_product-spec.md#793a1a-terms) lines; older messages are read for kind and text. Redacting every message cost 2.2 seconds for 2 MB and 8.4 for 11 MB per reading, and there were two readings; the 2-second bound for 1 MB is a regression guard with room for a loaded machine, as the command now spends most of that time parsing. A, on review (`793e6g` finding 19).

## 793b76 A turn is found by the text it began with

The mod keeps each [prompt](./793a_product-spec.md#793a1a-terms)'s text and origin, and the judgement begins the [turn](./793a_product-spec.md#793a1a-terms) at the last record prompt with the text the turn started with (else the last prompt) and ends it after the final text (`793d2b`, `793d4g`): a prompt typed during a turn starts a turn of its own, so the last prompt is not always the turn's. Two prompts with the same text in one window are not told apart: rare, accepted. Rejected: the last prompt of the record, as before (it loses the correction of a turn when the user types ahead, the common way to work), and the record's prompt ID, if the event gives it later. A, on review (`793e6g` findings 2 and 8).

## 793b77 The instructions are the system of the request, and a < is written &lt;

The request has a `system` (the instructions, nothing of the record) and a `prompt` (the window and [notes](../../GLOSSARY.md#permanent-note) between their tags); a `<` in the window or notes is written `&lt;`, which the instructions explain, so no message or note can close a section or forge a tag (review probe B); a random delimiter in each request was rejected, as the escape is deterministic and testable. A, on review (`793e6g` finding 5).

## 793b78 The promise names the forms of credential that are redacted

The promise of `793a2c` is that no [note](../../GLOSSARY.md#permanent-note) holds a credential in a recognised form, and it lists the forms; the redaction matches names holding a word of a secret with anything around it, redacts a value given by `=`, `:` or `is` with up to six words between, and a `Bearer` credential from 12 characters like `Basic` (a `Bearer` of one character redacted the word after it in prose). The live scenario passed with the redaction absent, so an absolute promise could not be failed. A, on review (`793e6g` findings 3, 4 and 25; `793c49`).

## 793b79 A field of a lesson is one line, and a reply that is cut keeps its whole lessons

Every lesson field is redacted and written as one line before writing (a `Supersedes:` line in a field is read by the [vault](../../GLOSSARY.md#vault) as a link), and lessons are read from the reply one by one, so a reply cut by the 1,500-token cap keeps its whole lessons, the cut recorded as skipped. Rejected: writing nothing from a cut reply, as before (a lesson is whole or it is not, whatever follows it), and a higher cap (it does not remove the cut). A, on review (`793e6g` findings 11 and 26).

## 793b80 The delivered line shows the mark of a note written automatically

The line of a [note](../../GLOSSARY.md#permanent-note) whose source opens with `written automatically` ends with the fixed words `(written automatically)`, appended so that the line's form (`793d4c`) and the reading of the file name (`793d4d`) do not change; the [agent](./793a_product-spec.md#793a1a-terms) should weigh such a note differently. Rejected: no mark, as before, and the mark in the framing line as a count; the mark costs 22 characters for a note and tells the agent what the [vault](../../GLOSSARY.md#vault) already tells a reader. A, on review (`793e6g` finding 15).

## 793b81 A text is written as one line before it is redacted, and cut last

Every text is written as one line first (control and format characters dropped, white space collapsed), then redacted, then cut, then escaped in the window; `⏎` counts as white space and an `Authorization` header ends at it. Redacting first missed a credential split by hidden characters that the one-line writing then joined; redacting before and after was rejected, as the first pass is the one that cannot see a split credential. A, on the security review relayed by the coordinator.

## 793b82 A judgement without a vault or a record fails, and the model is not asked

`engram auto judge` fails, printing nothing and naming the cause, when no [vault](../../GLOSSARY.md#vault) exists at its path or no record of the [session](../../GLOSSARY.md#session) is found; the mod reports it, the writing records it once and the model is not asked. A record with no [prompt](./793a_product-spec.md#793a1a-terms) is judged nothing. Before, a missing vault cost a model call and two records per [turn](./793a_product-spec.md#793a1a-terms), and a wrong records directory lost captures silently. Rejected: a recorded skip with the reason while the judgement still runs, a `skipped` record (it needs a second report from the mod) and success with no trace, as before. A, on review (`793e6g` findings 9 and 13); it amends `793b71`.

## 793b83 A failure that is recorded does not fail the command

When a lesson cannot be written, `engram auto write` records `failed` and succeeds; it fails only when it could not record the failure, so one failure is neither recorded twice nor lost; the record is the display of what happened and the mod runs the command, so failing with the mod not reporting the writing's failure, and failing with the writing not recording it, were rejected. A, on review (`793e6g` finding 10).

## 793b84 A headless prompt is judged only when the user asks

The delivery acts for the four origins of `793b64`; the judgement acts for `composer`, `bridge` and `slack-ping`, and for `sdk` only when `--capture-sdk` or `ENGRAM_AUTO_CAPTURE_SDK` (a true value) is set, so a script run with `claude -p` under the user's real home writes no fiction into the [vault](../../GLOSSARY.md#vault) (the delivery only reads). The acceptance machine sets the variable. A, on review (`793e6g` finding 6); `793c39`.

## 793b85 Each stage of the capture has a bound of its own

Stage bounds: the [session](../../GLOSSARY.md#session)-ID call 5 seconds (raced by a timer; no ID, no capture), `engram auto judge` 15, the model call 60, `engram auto write` 20, the failure report 15: 115 seconds, under 2 minutes; a stage that overruns has failed, and a non-JSON request is a failure of the judgement. The host's default of 30 seconds per call and an unbounded ID call left no margin; a timer over the chain is rejected as in `793b70`, and shorter bounds because a loaded machine took 7 seconds to start a command (`793c32`). A, on review (`793e6g` findings 12 and 24; `793c2`).

## 793b86 The acceptance run takes a count

`targ acceptance` takes `--count` (default 1) and a 30-minute limit per repetition of the whole set; a model-dependent scenario's tally is its passes in 10 runs, each red attributed to its cause, and the bar is 9 in 10. Six of the second group's 14 rows had no tally. Rejected: 10 in 10 (one unlucky run of a model's judgement would be a red) and a bar for each scenario by the cost of a miss. A, on review (`793e6g` finding 7).

## 793b87 The instructions carry the bar of the learn skill

The instructions carry the learn [skill](../../GLOSSARY.md#skill)'s bar: a lesson in the form in which it applies again, with exact names, commands, flags and values; a thanks or "it worked" that names no approach, or a success no uncertainty preceded, is no lesson; the `Brief:` clause is dropped; a lesson a nearest [note](../../GLOSSARY.md#permanent-note) already states is still listed with that note's file name. Without the last, `793a3c1a` passed 3 of 5 runs; with it, 10 of 10. The skill's "general principle, not the instance" is rejected: a note that names the project's build target and not `make zorblax-build` cannot be told from a guess, and the rows of `793a2c1a` and `793a2c1d` check the exact words. A, on review (`793e6g` findings 14 and 22; `793c56`).

## 793b88 The judgement is measured by a probe of its own

Instructions are changed with `dev/eval/auto-judge/probe.py` at hand: it builds the request with the real `engram auto judge` over fixtures of the live scenarios' [turns](./793a_product-spec.md#793a1a-terms), asks the model as the mod does 60 times or more per variant (20 did not separate 80% from 98%), and keeps a sentence only when its variant beats the one without on its fixture and no other falls by more than noise. The probe is part of no check. Rerun: unsure-approach 19 of 60 without the sentence, 60 with; thanks, no lesson, 43 then 60; covered 51 then 59; credential 54 throughout. The probe asks through `claude -p` where the mod uses the host's model call, so its numbers and the live rows' are close but not equal, and the live rows decide; the live rows alone and a probe through the host's model call in a mod of its own were rejected (the probe is cheap enough to run 60 times). A, on review (`793e6g` finding 7).

## 793b89 Each automatic note is offered to the parent vault, and the duplicate check stays local

`engram auto write` goes through the learn path, which under `ENGRAM_PARENT` offers each [note](../../GLOSSARY.md#permanent-note) to the parent [vault](../../GLOSSARY.md#vault); the judgement's nearest-notes lookup is local only, so a lesson the parent already states is written and offered again. Nothing is built: the learn path is the one writer, and the merge is the node `793e6f` (`793b57`). A, on review (`793e6g` finding 20); `793c59`.

## 793b90 A request to remember is a lesson whatever the agent says about saving it

The instructions say a request by the user to remember something is a lesson even when it holds a secret (the lesson names it as redacted) and even when the [agent](./793a_product-spec.md#793a1a-terms) says it did not save it. `793a2c1g` tallied 8 of 10 (both misses wrote a rule against keeping credentials from the agent's remark) and `793a2c1i` passed 0 of 10; with both phrases 10 of 10 each, the other fixtures unmoved. The phrases placed elsewhere or worded differently lowered the covered lesson or the credential row. A (`793c60`).

## 793b91 The documents follow the guides and are held to the minimum sufficient

Two owner instructions of 2026-10-17. The reduction: "even the documents are wordy. I'd like you to try to reduce the documentation to the absolute minimum sufficient, similar to the rule about the code" (the rule of step 5, `793f2`), for the documents of this change and the lines it adds to `README.md` and `AGENTS.md`. Then: "Continue to follow the guide rules." So the documents lose prose and keep every element the guides require: the "ID title" links, the restated product scenarios, and the context, lettered options, recommendation, blocks and status of a question. O 2026-10-17.

## 793b92 Review of the documentation reduction

The reduction of these documents (13 commits, `408e5d16` to `8f07087b`, about 73,000 words to 36,000) had one opus review round. It found the cut below sufficient where it removed what a reader needs to act or verify (decision reasons and rejected alternatives, accepted review findings, a measured baseline, clauses of requirements), and elements the guides require. Applied in new commits: the reasons restored; the guides' elements restored (`793b91`); the two older non-conformances asked in `793c66` and `793c67`; the questions that block nothing withdrawn from `793c` to their entries here. R (opus, 2026-10-17), accepted by the author.

## 793b93 References are links

Withdrawn: the owner said to follow the guides (`793b91`), so a reference to a node is its "ID title" link.

## 793b94 The mapping restates the product scenario

Withdrawn: the owner said to follow the guides (`793b91`), so `793d5` restates each product scenario at its first mapping, and all 27 scenarios of `793a` are mapped.

## 793b95 An open question has the guide's elements

Withdrawn: the owner said to follow the guides (`793b91`), so a question has its context, lettered options, recommendation, blocks and status; a question that blocks nothing is an author choice and is not asked, but is withdrawn to its entry here (`docs/AGENTS.md`).

## 793b96 The order of the groups is a table

`793e4` gives the order the groups are worked in as a table, with no second graph (the first version's 90-line chain graph listed the same order); `793e9`, with the status of each node in its label, is the one dependency graph and the source of what depends on what (`793c64`). A, on the review of the reduction (`793b92`).

## 793b97 The commit columns stay

The catalog keeps its Red, Green and Refactor commit columns (`793b13`) although the reduction rule would cut them as hashes: they are the record of the owner's step 5 (`793f2`), and the gate of `793e7` asks every `Done` row for its three commits (`793c65`). A, on the review of the reduction (`793b92`).

## 793b98 Two requirements name two owners

`793d4j` (redaction) is owned by the `turn-judgement` and the `lesson-writing`, and `793d4m` (origins) by the `prompt-delivery` and the `turn-judgement`; `docs/TECHNICAL-SPEC-GUIDE.md` has one requirement for each owning component. Each stays whole in this change: the shapes of a credential and the origins of a [prompt](./793a_product-spec.md#793a1a-terms) are stated once, and two copies would drift (`793c66`). A, on the review of the reduction (`793b92`).

## 793b99 Default-suite tests that cross the filesystem

Seven rows of `793e6a` and `793e6g` (`793d4d1b`, `793d4a1d`, `793d4a1g`, `793d4f1a`, `793d4e1a`, `793d4g1q`, `793d4g1w`) are in the default suite and run over a real temporary directory, which `docs/TECHNICAL-SPEC-GUIDE.md` reserves for the `integration` tag (the tests of `internal/cli` for `engram auto` carry no tag and use `t.TempDir`). They pass in `targ test` and are not changed in this pass (`793c67`). A, on the review of the reduction (`793b92`).
