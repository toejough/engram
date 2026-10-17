# Automatic Memory - Decision Log

Decisions made while producing [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) (#793) and the process around it, recorded for Joe to validate.
"Decided by" follows [Decisions and Open Questions](../../AGENTS.md#decisions-and-open-questions): Owner for
Joe's own words, Author for the author's derivations and defaults. Joe was unavailable, so every question the
author would have asked has a default, recorded here as "Decided by: Author (default; owner may override)" and
listed in [793c Automatic Memory - Open Questions](./793c_open-questions.md#automatic-memory---open-questions).

## 793b1 Start over

- **Choice:** abandon the earlier attempt (change directory `793_parallel-memory-agent` on branch
  `docs/parallel-memory-agent-requirements`) and produce this change from main in `793_automatic-memory`,
  keeping what was learned but not that design.
- **Rationale:** the owner, 2026-10-16: "I want to branch off and try something else. I want to start over.
  Feel free to use what you've learned, but I want to go back to before we had a mod, before this change was
  implemented."
- **Alternatives:** none stated.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b2 North Star

- **Choice:** the product statement is "I want to add automatic learning and memory retrieval.", kept in the
  owner's own first-person words. It opens [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) and is the one line every later artifact serves.
- **Rationale:** the owner: "I want to start with one single, focused statement to summarize the product spec
  and be the North Star: I want to add automatic learning and memory retrieval. That's the headline feature of
  this change."
- **Alternatives:** none stated.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b3 The algorithm and the working rules

- **Choice:** follow the owner's five steps, quoted in [793f2 The algorithm](./793f_algorithm.md#793f2-the-algorithm), depth first, committing early and often
  and working autonomously until the author judges the work done.
- **Rationale:** the owner: "Write this algorithm down. Follow it. Write the requirements, scenarios, tests,
  and code. Commit early and often. Work autonomously till you think you're done. Work depth first through the
  above algorithm."
- **Alternatives:** none stated.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b4 Delegation

- **Choice:** the delegation rule, quoted in [793f4 Delegation](./793f_algorithm.md#793f4-delegation).
- **Rationale:** the owner, a minute after the algorithm: "You should only orchestrate. Use haiku to search,
  sonnet to work, and opus to review. Write that down, too, and follow it for delegation."
- **Alternatives:** none stated.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b5 Reading of the delegation rule

- **Choice:** the orchestrating [session](../../GLOSSARY.md#session) does no object-level work itself; search and lookups go to
  haiku, all writing and building go to sonnet, and every review goes to opus, one review round per phase. A
  phase is one step applied to one unit: step 1 for the whole set, or step 2, 3, 4 or 5 for one requirement.
- **Rationale:** the plain reading of "orchestrate", "search", "work" and "review". One review round follows
  [docs/AGENTS.md](../../AGENTS.md#decisions-and-open-questions) ("A change spec MUST get one agent review
  round, unless the owner asks for more"). For code it departs from the default of reviews argued to
  resolution in the root [AGENTS.md](../../../AGENTS.md#issue-workflow-default).
- **Alternatives:** argue each review to resolution, as the root default does. Not adopted: the owner named one
  model for review and said nothing of argument.
- **Decided by:** Author (reading of the owner's rule; owner may override).

## 793b6 The five requirements are the minimal set

- **Choice:** the five product requirements of [793a2 Automatic memory](./793a_product-spec.md#793a2-added-automatic-memory). Each is in because:
  - [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update): without it nothing is on for the user, and it must need no other step or the feature does
    not reach the user's [sessions](../../GLOSSARY.md#session) automatically.
  - [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt): the memory-retrieval half of the North Star.
  - [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent): the learning half of the North Star.
  - [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way): both halves run on every [prompt](./793a_product-spec.md#793a1a-terms) and [turn](./793a_product-spec.md#793a1a-terms), so a delay or a failure would hurt every
    session; bounding them is what lets the user leave them on.
  - [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did): the owner asked for capabilities that are "user visible and user validatable", and
    automatic work the user cannot see cannot be validated.
- **Rationale:** the smallest set in which each half of the North Star can be switched on, relied on, bounded
  and checked by the user.
- **Alternatives:** fold the display into [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), or the install into [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt). Rejected: each has
  its own promise the user can verify (the line, and a fresh session after an update), so each keeps its own
  scenarios, though [793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt) works the last pair together.
- **Decided by:** Author.

## 793b7 The interactions are product requirements too

- **Choice:** the six interactions of [793a3 Interactions](./793a_product-spec.md#793a3-added-interactions) are requirements with their own scenario sections, not a
  prose appendix. Where a requirement and an interaction touch one rule, the rule is stated once and the other
  cites it: [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) cites [793a3b Nothing the agent already has is delivered again](./793a_product-spec.md#793a3b-nothing-the-agent-already-has-is-delivered-again), [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) cites [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again), and [793a3d Failures never break a prompt and always show](./793a_product-spec.md#793a3d-failures-never-break-a-prompt-and-always-show) joins
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) to [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did).
- **Rationale:** the owner's step 1 asks for the capabilities "and how they must interact to achieve the goal.
  These are the product requirements."
- **Alternatives:** a non-normative paragraph about interactions. Rejected: the owner's step 2 asks for
  scenarios for every requirement, and a paragraph has none.
- **Decided by:** Author (reading of the owner's step 1; owner may validate).

## 793b8 Exclusions

- **Choice:** nine capabilities the earlier attempt had are left out and listed in [793a4 Not in the minimal set](./793a_product-spec.md#793a4-not-in-the-minimal-set): memory at
  [subagent](../../GLOSSARY.md#subagent) start and capture at subagent stop; memory at route results; the chunk-index deep lookup and the
  [transcript](../../GLOSSARY.md#transcript) sweep ([`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand)); backlog capture of old transcripts; curation of
  [pending offers](../../GLOSSARY.md#offer); a model-made relevance judgement before delivery; reinforcing an already-covered [note](../../GLOSSARY.md#permanent-note); a
  status line; the Pi [harness](../../GLOSSARY.md#harness).
- **Rationale:** none is needed to deliver the North Star, and the owner asked for the "minimal set". The
  earlier attempt measured a lookup that included the chunk index at about 12 seconds on the owner's
  [vault](../../GLOSSARY.md#vault), beyond the [wait bound](./793a_product-spec.md#793a1a-terms), against about 1.4 seconds for notes alone. The section
  goes beyond the product guide's three sections so that the exclusions stay visible and each can be a separate
  change.
- **Alternatives:** leave them unlisted. Rejected: a later reader could not tell a left-out capability from a
  forgotten one.
- **Decided by:** Author. The subagent exclusion reverses an answer the owner gave in the earlier attempt on
  2026-10-11 (capture inside subagent [sessions](../../GLOSSARY.md#session) too); [793c6 Is the scope top-level sessions and notes only?](./793c_open-questions.md#793c6-is-the-scope-top-level-sessions-and-notes-only) asks him to confirm.

## 793b9 Test execution

- **Choice:** each scenario in the product spec is executed literally against a real Claude Code
  [session](../../GLOSSARY.md#session), so the Gherkin in the spec is the test source; each scenario in the technical spec is
  executed by one Go test whose name carries the scenario ID.
- **Rationale:** product scenarios are "user exercisable" (the owner's step 2), so a real session is the only
  test that proves what the user sees; the plan guide already requires a test name to begin with its scenario
  ID.
- **Alternatives:** Go tests that stand in for the session. Rejected: they would not prove the promise.
- **Decided by:** Author.

## 793b10 Departure: depth-first fill

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec), [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) and [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan) stay partial until the last requirement has been
  through step 5, and code for a requirement is written before the later requirements have scenarios,
  technical requirements or plan nodes.
- **Departs from:** the Required Implementation Workflow in [docs/AGENTS.md](../../AGENTS.md#required-implementation-workflow)
  (identify the requirements, scenarios and plan nodes before editing implementation files) and the technical
  and plan guides' rule to start from approved specifications.
- **Rationale:** the owner: "Work depth first through the above algorithm."
- **Alternatives:** breadth first, as the guides expect.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b11 Departure: no interview or approval gates

- **Choice:** no interview and no approval before each document; the author works autonomously and the owner
  reviews at the end.
- **Departs from:** the Authoring Workflow of the [product](../../PRODUCT-SPEC-GUIDE.md#authoring-workflow),
  [technical](../../TECHNICAL-SPEC-GUIDE.md#authoring-workflow) and
  [plan](../../IMPLEMENTATION-PLAN-GUIDE.md#authoring-workflow) guides, and "resolve material ambiguity with the
  user before writing" in [docs/AGENTS.md](../../AGENTS.md#general-documentation-rules).
- **Rationale:** the owner: "Work autonomously till you think you're done." He was unavailable, so each
  question has a default instead ([793c Automatic Memory - Open Questions](./793c_open-questions.md#automatic-memory---open-questions)).
- **Alternatives:** stop and wait for the owner at each document.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b12 Departure: no option analysis

- **Choice:** [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) has no solution-space analysis, no relative option impact and no alternative
  options; step 3 takes the minimal components a scenario needs.
- **Departs from:** the Default Structure and Solution-Space and Option Analysis of the
  [technical spec guide](../../TECHNICAL-SPEC-GUIDE.md#default-structure). The guide allows a section to be
  omitted when it does not apply, and asks for the reason where a reviewer first needs it; this entry is the
  reason, and [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) repeats it.
- **Rationale:** the owner's step 3: "identify the minimal set of components and how they must behave and
  interact to implement the functionality the test requires."
- **Alternatives:** compare holistic options before choosing components.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b13 Departure: one refactor pass

- **Choice:** one refactor commit per test, aimed at minimizing the code the branch adds without obscuring its
  purpose. The statuses are `Not started`, `Red`, `Green-all` and `Done`, with `Done` set by the refactor
  commit.
- **Departs from:** Test Status in the [plan guide](../../IMPLEMENTATION-PLAN-GUIDE.md#test-status), which has
  `Green-single` and `Green-all`, then four refactor checkpoints (complexity, consistency, boundaries,
  performance).
- **Rationale:** the owner's step 5: "write it, make it pass, then refactor to minimize the code introduced by
  the branch without obfuscating its function and purpose."
- **Alternatives:** keep the four checkpoints.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b14 Departure: the North Star line

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) opens with the North Star line under its title.
- **Departs from:** the [product spec guide](../../PRODUCT-SPEC-GUIDE.md#purpose), under which a product
  specification contains only a problem statement, product requirements and scenarios unless the user
  explicitly requests more. The "Not in the minimal set" section is a further addition, the author's
  ([793b8 Exclusions](#793b8-exclusions)).
- **Rationale:** the owner: "I want to start with one single, focused statement to summarize the product
  spec".
- **Alternatives:** state it only in the problem statement.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b15 Artifact names

- **Choice:** the change directory is `793_automatic-memory`. The letters are a product spec, b decision log,
  c open questions, d technical spec, e implementation plan and f algorithm; the process document is
  `793f_algorithm.md`, not an unlettered `793_algorithm.md`.
- **Rationale:** "Issue-Rooted Change IDs" in [docs/AGENTS.md](../../AGENTS.md#issue-rooted-change-ids)
  requires every change artifact to extend the issue key with alternating lowercase letters and numbers, and no
  guide names a process document, so it takes the next free letter.
- **Alternatives:** `793_algorithm.md`. Rejected: it breaks the ID rule.
- **Decided by:** Author.

## 793b16 One subject term

- **Choice:** the spec's subject is "[automatic memory](./793a_product-spec.md#793a1a-terms)", defined in [793a1a Terms](./793a_product-spec.md#793a1a-terms), throughout; not "the memory
  system" or "the System".
- **Rationale:** one term for one thing, shortened from the North Star's own words. A "system" has no
  definition in the glossary and would be a second name for the same behavior.
- **Alternatives:** "the memory system". Rejected for that reason.
- **Decided by:** Author.

## 793b17 Wait bound is 8 seconds

- **Choice:** the [wait bound](./793a_product-spec.md#793a1a-terms) is 8 seconds, stated once in [793a1a Terms](./793a_product-spec.md#793a1a-terms) so one edit changes it.
- **Rationale:** 8 seconds is the ceiling the owner approved for holding a [prompt](./793a_product-spec.md#793a1a-terms) in the earlier attempt,
  where a lookup of [notes](../../GLOSSARY.md#permanent-note) alone took about 1.4 seconds with six at once.
- **Alternatives:** a shorter bound such as 3 seconds, or a longer one ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)).
- **Decided by:** Author (default; owner may override).

## 793b18 Capture delay is 2 minutes

- **Choice:** the [capture delay](./793a_product-spec.md#793a1a-terms) is 2 minutes, stated once in [793a1a Terms](./793a_product-spec.md#793a1a-terms).
- **Rationale:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) needs a number to be testable. The earlier attempt set no bound for the normal
  path and this attempt has not measured a capture yet, so 2 minutes is a guess with headroom, to be checked
  when the requirement is built.
- **Alternatives:** 1 minute or 5 minutes ([793c2 How soon after a turn must a lesson be in the vault?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-a-lesson-be-in-the-vault)).
- **Decided by:** Author (default; owner may override).

## 793b19 Volume of delivered notes

- **Choice:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) states no number of [notes](../../GLOSSARY.md#permanent-note); the technical spec limits the number of notes delivered
  to the number the [agent](./793a_product-spec.md#793a1a-terms)'s own explicit lookup returns.
- **Rationale:** the owner's answer on 2026-10-11 in the earlier attempt: volume equals what the agent's own
  lookup returns, and per-[turn](./793a_product-spec.md#793a1a-terms) and per-[session](../../GLOSSARY.md#session) bounds are dropped.
- **Alternatives:** a fixed maximum in the product spec ([793c3 Is the volume of delivered notes limited?](./793c_open-questions.md#793c3-is-the-volume-of-delivered-notes-limited)).
- **Decided by:** Author (default; owner may override).

## 793b20 No modes and no off switch

- **Choice:** [automatic memory](./793a_product-spec.md#793a1a-terms) is on whenever it is installed ([793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)). There is no observe mode and no
  off switch; stopping it means removing what [`engram update`](../../GLOSSARY.md#engram-update) added, and no requirement
  covers how.
- **Rationale:** the owner's answer on 2026-10-11 in the earlier attempt: "remove modes; the System is active
  whenever installed", and disabling or uninstalling it stops it.
- **Alternatives:** observe and active modes with a kill switch, as issue #793 expected ([793c4 Are there modes or an off switch?](./793c_open-questions.md#793c4-are-there-modes-or-an-off-switch)).
- **Decided by:** Author (default; owner may override).

## 793b21 Lessons are written straight into the vault

- **Choice:** a lesson becomes a [note](../../GLOSSARY.md#permanent-note) directly ([793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)), not a [pending offer](../../GLOSSARY.md#offer) awaiting review.
- **Rationale:** the owner's answer on 2026-10-11 in the earlier attempt: "written directly to the [vault](../../GLOSSARY.md#vault)."
  Curation is excluded ([793a4 Not in the minimal set](./793a_product-spec.md#793a4-not-in-the-minimal-set)).
- **Alternatives:** pending offers reviewed with the curate skill, issue #793's open decision 4
  ([793c5 Are lessons written straight into the vault or held for review?](./793c_open-questions.md#793c5-are-lessons-written-straight-into-the-vault-or-held-for-review)).
- **Decided by:** Author (default; owner may override).

## 793b22 The scope is top-level sessions and notes only

- **Choice:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) apply to the [prompts](./793a_product-spec.md#793a1a-terms) and [turns](./793a_product-spec.md#793a1a-terms) of the user's own [sessions](../../GLOSSARY.md#session), not to
  [subagents](../../GLOSSARY.md#subagent), and delivery looks up [notes](../../GLOSSARY.md#permanent-note) only, not the chunk index.
- **Rationale:** both follow from "minimal" and from the [wait bound](./793a_product-spec.md#793a1a-terms): a lookup that included the chunk index
  took about 12 seconds on the owner's [vault](../../GLOSSARY.md#vault) in the earlier attempt.
- **Alternatives:** the owner's answers on 2026-10-11 in the earlier attempt: capture inside subagent sessions,
  and memory for every prompt regardless of source. This entry reverses them for the minimal set
  ([793c6 Is the scope top-level sessions and notes only?](./793c_open-questions.md#793c6-is-the-scope-top-level-sessions-and-notes-only)).
- **Decided by:** Author (default; owner may override).

## 793b23 No capture rate or precision targets

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) states no target; the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) are the proof.
- **Rationale:** the owner's step 2 proves a requirement with scenarios, and the earlier attempt's targets (a
  capture rate above the 5.6% baseline and recorded precision, the owner on 2026-10-11) need measurement the
  minimal set leaves out.
- **Alternatives:** the targets of issue #793's open decision 1 ([793c7 Are there capture rate or precision targets?](./793c_open-questions.md#793c7-are-there-capture-rate-or-precision-targets)).
- **Decided by:** Author (default; owner may override).

## 793b24 Secrets in written notes are not addressed

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) states no rule about secrets in a written [note](../../GLOSSARY.md#permanent-note).
- **Rationale:** [docs/AGENTS.md](../../AGENTS.md#general-documentation-rules) says to document only
  confirmed scope, and the restart confirmed five requirements. The earlier attempt held a MUST NOT on it (the
  owner, 2026-10-11) that its final set no longer carries, keeping only a rule that hides secrets in warning
  texts.
- **Alternatives:** add "a note MUST NOT contain a secret" to [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent). The author recommends it
  ([793c8 What about secrets in written notes?](./793c_open-questions.md#793c8-what-about-secrets-in-written-notes)).
- **Decided by:** Author (default; owner may override).

## 793b25 A turn with nothing to report shows no line

- **Choice:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) applies to [turns](./793a_product-spec.md#793a1a-terms) on which something was delivered, written or skipped.
- **Rationale:** the earlier attempt omitted zero counts and showed nothing for a turn without events; a line
  on every turn would be noise.
- **Alternatives:** a line on every turn, so the user can tell [automatic memory](./793a_product-spec.md#793a1a-terms) is on ([793c9 Does a turn with nothing to report get a line?](./793c_open-questions.md#793c9-does-a-turn-with-nothing-to-report-get-a-line)).
- **Decided by:** Author (default; owner may override).

## 793b26 The line waits for the turn's work

- **Choice:** the line of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) appears once [automatic memory](./793a_product-spec.md#793a1a-terms) has finished its work for the [turn](./793a_product-spec.md#793a1a-terms), with
  no number of its own.
- **Rationale:** a write that completes after the turn ends can only be reported when it is done; the latest
  that can be is the [capture delay](./793a_product-spec.md#793a1a-terms) plus the time to show it.
- **Alternatives:** show the line at the end of the turn and report the write in a second line. Rejected:
  [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) promises one line.
- **Decided by:** Author.

## 793b27 "Already has" is per session

- **Choice:** in [793a3b Nothing the agent already has is delivered again](./793a_product-spec.md#793a3b-nothing-the-agent-already-has-is-delivered-again) the [agent](./793a_product-spec.md#793a1a-terms) "already has" a [note](../../GLOSSARY.md#permanent-note) when [automatic memory](./793a_product-spec.md#793a1a-terms) delivered it earlier in the
  same [session](../../GLOSSARY.md#session) or the agent looked it up in it, even after the session's context was compacted.
- **Rationale:** it is the literal reading of "this session" and needs no knowledge of what the agent still
  holds; the earlier attempt used the receiving agent's context instead (the owner, 2026-10-11).
- **Alternatives:** deliver a note again after the context was compacted ([793c10 Does "already has" survive a compaction?](./793c_open-questions.md#793c10-does-already-has-survive-a-compaction)).
- **Decided by:** Author (default; owner may override).

## 793b28 Depth-first order

- **Choice:** [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) go first and together, then [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), then
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), then [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did).
- **Rationale:** the only user-visible proof that the first is on is the second working in a new [session](../../GLOSSARY.md#session). The
  learning half comes before the bounds and the line because those bound and report what the two halves do.
- **Alternatives:** the order of the requirements as listed, with the install alone first. Rejected: its
  scenarios could not be run before delivery exists.
- **Decided by:** Author.

## 793b29 Interactions join the first node that has every capability they name

- **Choice:** in [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan), an interaction is worked with the first node whose requirements include every
  capability it names: [793a3b Nothing the agent already has is delivered again](./793a_product-spec.md#793a3b-nothing-the-agent-already-has-is-delivered-again) with the first node; [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later), [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again), [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) and
  [793a3f Everything runs on the existing login and vault](./793a_product-spec.md#793a3f-everything-runs-on-the-existing-login-and-vault) with the second; [793a3d Failures never break a prompt and always show](./793a_product-spec.md#793a3d-failures-never-break-a-prompt-and-always-show) with the fourth.
- **Rationale:** an interaction cannot be exercised before everything it joins exists, and delaying it further
  would leave its rule unchecked.
- **Alternatives:** all interactions in a final node (rules unchecked for long), or each with its own earliest
  capability (it cannot be exercised there).
- **Decided by:** Author.

## 793b30 Commit hashes are recorded by the next commit

- **Choice:** in [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan), the Red, Green and Refactor columns of a row hold the hashes of those phases'
  commits, each filled in by the next commit; each phase's own commit sets the row's Status.
- **Rationale:** a commit cannot contain its own hash, and amending it to add one changes the hash; the plan
  guide requires the status to change in the commit that establishes the checkpoint.
- **Alternatives:** amend each commit with its own hash (impossible), or keep no hashes (this loses the red,
  green and refactor record the owner asked for).
- **Decided by:** Author.

## 793b31 Glossary terms stay in the Terms list until promotion

- **Choice:** the terms [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) uses that the glossary lacks stay in [793a1a Terms](./793a_product-spec.md#793a1a-terms) until the change is
  promoted ([793c11 Glossary gaps](./793c_open-questions.md#793c11-glossary-gaps)).
- **Rationale:** the glossary describes shipped behavior, and these terms are not shipped yet.
- **Alternatives:** add them to the glossary now.
- **Decided by:** Author (default; owner may override).
