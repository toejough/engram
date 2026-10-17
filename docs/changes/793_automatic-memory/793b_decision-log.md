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
  owner's own first-person words, with nothing added to it. It opens [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec), the problem
  statement quotes it as the needed outcome, and it is the one line every later artifact serves.
- **Departs from:** the [product spec guide](../../PRODUCT-SPEC-GUIDE.md#purpose), under which a product
  specification contains only a problem statement, product requirements and scenarios unless the user
  explicitly requests more.
- **Rationale:** the owner: "I want to start with one single, focused statement to summarize the product spec
  and be the North Star: I want to add automatic learning and memory retrieval. That's the headline feature of
  this change."
- **Alternatives:** state it only in the problem statement.
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

- **Choice:** the orchestrating [session](../../GLOSSARY.md#session) does no object-level work itself; search and
  lookups go to haiku, all writing and building go to sonnet, and every review goes to opus. A phase, the unit of
  one review round ([793b33 One agent review round per phase](#793b33-one-agent-review-round-per-phase)), is one
  step applied to one unit: step 1 for the whole set, step 2 for one requirement, or step 3, 4 or 5 for one node ([793b38 Depth first: two readings, and the one this change takes](./793b_decision-log.md#793b38-depth-first-two-readings-and-the-one-this-change-takes)).
- **Rationale:** the plain reading of "orchestrate", "search", "work" and "review".
- **Alternatives:** none stated by the owner.
- **Decided by:** Author (reading of the owner's rule; owner may override).

## 793b6 The six requirements are the minimal set

- **Choice:** the six product requirements of [793a2 Automatic memory](./793a_product-spec.md#793a2-added-automatic-memory). Each is in because:
  - [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update): without it nothing is on for the user, and it must need no other step or the feature does
    not reach the user's [sessions](../../GLOSSARY.md#session) automatically.
  - [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt): the memory-retrieval half of the North Star.
  - [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent): the learning half of the North Star.
  - [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory): the other half of what the main [agent](./793a_product-spec.md#793a1a-terms)'s learn did, its sweep ([793b34 The goal: what the main agent was already doing, just async](#793b34-the-goal-what-the-main-agent-was-already-doing-just-async)); without it only the lessons are automated and the conversation the agent's own lookup searches goes stale.
  - [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way): both halves run on every [prompt](./793a_product-spec.md#793a1a-terms) and [turn](./793a_product-spec.md#793a1a-terms), so a delay or a failure would hurt every
    session; bounding them is what lets the user leave them on.
  - [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did): the owner asked for capabilities that are "user visible and user validatable", and
    automatic work the user cannot see cannot be validated.
- **Rationale:** the smallest set in which each half of the North Star can be switched on, relied on, bounded
  and checked by the user.
- **Alternatives:** fold the display into [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), or the install into [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt). Rejected: each has
  its own promise the user can verify (what is shown, and a fresh session after an update), so each keeps its own
  scenarios, though [793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt) works the install and the first delivery together.
- **Decided by:** Author.

## 793b7 The interactions are product requirements too

- **Choice:** the three interactions of [793a3 Interactions](./793a_product-spec.md#793a3-added-interactions) are requirements with their own scenario sections, not a
  prose appendix. Each is a way two capabilities work together, and each can be tested: [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later) joins writing and
  delivery, [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again) joins writing and the [notes](../../GLOSSARY.md#permanent-note) already in the
  [vault](../../GLOSSARY.md#vault), and [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) joins automatic and explicit memory
  use. Where a requirement and an interaction touch one rule, the rule is stated once and the other cites it:
  [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) cites [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again).
- **Rationale:** the owner's step 1 asks for the capabilities "and how they must interact to achieve the goal.
  These are the product requirements."
- **Alternatives:** a non-normative paragraph about interactions. Rejected: the owner's step 2 asks for
  scenarios for every requirement, and a paragraph has none. Review round 1 cut the other three of the original
  six, because each restated a rule of one requirement and was not a way two capabilities work together:
  `793a3b` is folded into [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt), and the words of `793a3d` and `793a3f` moved into [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) and [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update). The
  three IDs are retired and never reused.
- **Decided by:** Author (reading of the owner's step 1; owner may validate).

## 793b8 Exclusions

- **Choice:** ten capabilities, which the earlier attempt had or the main [agent](./793a_product-spec.md#793a1a-terms) used, are left out of the
  minimal set. Each can be a separate change later:
  - **Subagent start and stop:** memory at a [subagent](../../GLOSSARY.md#subagent)'s start and capture at its stop. The North Star is
    met in the user's own [sessions](../../GLOSSARY.md#session), and a subagent works for the agent, not for
    the user.
  - **Route results:** memory when the orchestrator routes a unit. A [prompt](./793a_product-spec.md#793a1a-terms) already brings the
    [notes](../../GLOSSARY.md#permanent-note) that bear on it, and a route result is not a prompt.
  - **The chunk-index deep lookup:** conversation fragments (chunk stubs) delivered with a prompt's notes. Notes
    alone carry what the [vault](../../GLOSSARY.md#vault) has learned, and a lookup that included the chunk index took about 12 seconds on
    the owner's vault against 1.0 to 1.4 seconds for notes alone (`old 793b408`), beyond the 8 seconds of
    [793b17 A prompt is held for at most 8 seconds](./793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds). The agent finds fragments when it asks, because [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) makes the conversation searchable.
  - **Backlog capture of old [transcripts](../../GLOSSARY.md#transcript):** the lessons of new
    [turns](./793a_product-spec.md#793a1a-terms) are written by [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), and old ones can be mined later.
  - **Curation of [pending offers](../../GLOSSARY.md#offer):** lessons are written straight into the vault
    ([793c5 Are lessons written straight into the vault or held for review?](./793c_open-questions.md#793c5-are-lessons-written-straight-into-the-vault-or-held-for-review)).
  - **A model-made relevance judgement before delivery:** the agent judges what it is given, as it did when it
    ran the query itself.
  - **Reinforcing an already-covered note:** a lesson the vault already states is simply not written again
    ([793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again)).
  - **A status line,** a display the [harness](../../GLOSSARY.md#harness) draws apart from the conversation: the owner did not see the
    earlier attempt's status line on his machine and moved its content into the conversation (`old 793c90`,
    2026-10-15), and [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) asks only that the user can see the result in the session.
  - **The Pi harness:** sessions in other harnesses stay as before, and the update installs nothing into Pi
    ([793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)).
  - **Memory at mid-task moments** (tool results, before declaring done, after a failure): the main agent's
    mid-task re-queries were judgement-based moments it chose, and the minimal automatic set delivers at the
    prompt only ([793b34 The goal: what the main agent was already doing, just async](./793b_decision-log.md#793b34-the-goal-what-the-main-agent-was-already-doing-just-async)). Issue #793 asks for tool-result moments, so the owner is asked in [793c12 Is memory delivered at mid-task moments?](./793c_open-questions.md#793c12-is-memory-delivered-at-mid-task-moments).
- **Rationale:** none is needed to deliver the North Star, and the owner asked for the "minimal set". The list
  is in plain words here and not in the product spec, which holds only requirements and scenarios.
- **Alternatives:** leave them unlisted. Rejected: a later reader could not tell a left-out capability from a
  forgotten one. List them in the product spec, as the first version did. Rejected in review round 1: the
  product guide allows a problem statement, requirements and scenarios, and the reasons belong in the log.
- **Decided by:** Author. All but the status line and the Pi harness reverse answers the owner gave in the
  earlier attempt ([793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)); [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse) asks him to confirm.

## 793b9 Test execution

- **Choice:** each scenario in [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec)
  is executed by one Go test, named with the scenario ID, that drives a real Claude Code
  [session](../../GLOSSARY.md#session) in an isolated HOME
  ([793b32 Test isolation](#793b32-test-isolation)) and whose steps mirror the scenario's Given, When and Then
  lines one to one; its run output is the record. Each scenario in
  [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) is executed
  by one Go test whose name carries the scenario ID. A conformance check asserts that every scenario ID in either
  spec has a test.
- **Rationale:** product scenarios are "user exercisable" (the owner's step 2), so a real session is the only
  test that proves what the user sees; the plan guide already requires a test name to begin with its scenario
  ID; mirroring the Given, When and Then lines one to one keeps the Gherkin the source of the test, and the
  conformance check keeps a scenario from losing its test.
- **Alternatives:** Go tests that stand in for the session. Rejected: they would not prove the promise.
- **Decided by:** Author.

## 793b10 Departure: depth-first fill

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec), [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) and [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan) stay partial until the last node has been through step 5. The partial state shows as "Pending"
  placeholders: "Pending step 2" under every requirement of the product spec, "Pending." in every section of the
  technical spec, and empty catalog tables in the implementation plan. Code for a node is written before the later nodes have technical requirements or plan
  rows, and before the later requirements have scenarios.
- **Departs from:** the Required Implementation Workflow in [docs/AGENTS.md](../../AGENTS.md#required-implementation-workflow)
  (identify the requirements, scenarios and plan nodes before editing implementation files), the technical and
  plan guides' rule to start from approved specifications, their rule not to add empty placeholders, and the
  shared checklist's "No placeholders".
- **Rationale:** the owner: "Work depth first through the above algorithm." How deep one unit of work is: [793b38 Depth first: two readings, and the one this change takes](./793b_decision-log.md#793b38-depth-first-two-readings-and-the-one-this-change-takes).
- **Alternatives:** breadth first, as the guides expect.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b11 Departure: no interview or approval gates

- **Choice:** no interview and no approval before each document; the author works autonomously and the owner
  reviews at the end.
- **Departs from:** the Authoring Workflow of the [product](../../PRODUCT-SPEC-GUIDE.md#authoring-workflow),
  [technical](../../TECHNICAL-SPEC-GUIDE.md#authoring-workflow) and
  [plan](../../IMPLEMENTATION-PLAN-GUIDE.md#authoring-workflow) guides, and "resolve material ambiguity with the
  user before writing" in [docs/AGENTS.md](../../AGENTS.md#general-documentation-rules).
- **Rationale:** the author reads the owner's "Work autonomously till you think you're done." as dropping the
  interviews and approvals. He was unavailable, so each question has a default instead
  ([793c Automatic Memory - Open Questions](./793c_open-questions.md#automatic-memory---open-questions)).
- **Alternatives:** stop and wait for the owner at each document.
- **Decided by:** Author (reading of the owner's "Work autonomously"; owner may override).

## 793b12 Departure: no option analysis

- **Choice:** [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) has no solution-space analysis, no relative option impact and no alternative
  options; step 3 takes the minimal components a scenario needs.
- **Departs from:** the Default Structure and Solution-Space and Option Analysis of the
  [technical spec guide](../../TECHNICAL-SPEC-GUIDE.md#default-structure). The guide allows a section to be
  omitted when it does not apply, and asks for the reason where a reviewer first needs it; this entry is the
  reason, and [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) repeats it.
- **Rationale:** the author reads the owner's step 3, "identify the minimal set of components and how they must
  behave and interact to implement the functionality the test requires.", as asking for the minimal components a
  scenario needs and no comparison of options.
- **Alternatives:** compare holistic options before choosing components.
- **Decided by:** Author (reading of the owner's step 3; owner may override).

## 793b13 Departure: one refactor pass, and extra catalog columns

- **Choice:** one refactor commit per test, aimed at minimizing the code the branch adds without obscuring its
  purpose. The statuses are `Not started`, `Red`, `Green-all` and `Done`, with `Done` set by the refactor
  commit. The catalog tables in [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan)
  add three columns, Red commit, Green commit and Refactor commit, that hold the hash of each phase's commit.
  Each phase's own commit sets the row's Status, and each hash is filled in by the next commit, because a commit
  cannot contain its own hash and amending it would change the hash.
- **Departs from:** Test Status in the [plan guide](../../IMPLEMENTATION-PLAN-GUIDE.md#test-status), which has
  `Green-single` and `Green-all`, then four refactor checkpoints (complexity, consistency, boundaries,
  performance), and the guide's Test Catalog columns (Gherkin, Classification, Why and Status, with Properties for
  `rapid` tests), which have no commit columns.
- **Rationale:** the owner's step 5: "write it, make it pass, then refactor to minimize the code introduced by
  the branch without obfuscating its function and purpose." The single refactor pass is the author's reading of
  that step. The owner asked for no record of the commits; the three columns are the author's choice, so that
  each phase of a test can be found.
- **Alternatives:** keep the four checkpoints; keep no commit hashes, which loses the pointer from a row to its
  phases.
- **Decided by:** Author (reading of the owner's step 5; owner may override). The extra columns are the author's
  own choice, which the owner did not ask for.

## 793b14 Departure: the North Star line

Folded into [793b2 North Star](#793b2-north-star) in review round 1. The ID is kept so that links do not dangle.

## 793b15 Artifact names

Withdrawn in review round 1: not a decision. "Issue-Rooted Change IDs" in
[docs/AGENTS.md](../../AGENTS.md#issue-rooted-change-ids) dictates the lettered file names. The ID is kept so
that links do not dangle.

## 793b16 One subject term

Withdrawn in review round 1: not a decision. The product spec names its subject with the term its Terms list
defines. The ID is kept so that links do not dangle.

## 793b17 A prompt is held for at most 8 seconds

- **Choice:** [automatic memory](./793a_product-spec.md#793a1a-terms) may hold a
  [prompt](./793a_product-spec.md#793a1a-terms) for at most 8 seconds, written in
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way).
- **Rationale:** the earlier attempt's author set 8 seconds, 5 for the lookup and 3 for a relevance judgement
  (`old 793b317`). The owner did not set it. When a lookup that included the chunk index took 12 seconds against
  1.0 to 1.4 seconds for [notes](../../GLOSSARY.md#permanent-note) alone (`old 793b408`, 2026-10-14), he was
  offered a longer hold, did not raise it, and had the lookup split instead (`old 793c87`, 2026-10-14). His own
  rule was relative: no longer than the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup on the same
  [vault](../../GLOSSARY.md#vault) and input (`old 793c7` item 4 and `old 793c11` (b), 2026-10-11), which
  measured 10.6 to 11.0 seconds on his vault (`old 793c2`).
- **Alternatives:** a shorter bound such as 3 seconds; a longer one; the owner's relative rule
  ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)).
- **Decided by:** Author (default; owner may override).

## 793b18 Writing finishes within 2 minutes of a turn

- **Choice:** a [note](../../GLOSSARY.md#permanent-note) for a lesson is in the [vault](../../GLOSSARY.md#vault)
  ([793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)), and the [session](../../GLOSSARY.md#session)'s conversation is findable ([793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)), within 2 minutes
  after the [turn](./793a_product-spec.md#793a1a-terms) that held them ends.
- **Rationale:** both promises need a number to be testable. The earlier attempt set no bound for the normal
  path and this attempt has not measured a capture yet, so 2 minutes is a guess with headroom, to be replaced by
  the measurement when [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) is built.
- **Alternatives:** 1 minute or 5 minutes ([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)).
- **Decided by:** Author (default; owner may override).

## 793b19 Volume of delivered notes

- **Choice:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) states no number of [notes](../../GLOSSARY.md#permanent-note); the technical spec limits the number of notes delivered
  to the number the [agent](./793a_product-spec.md#793a1a-terms)'s own explicit lookup returns.
- **Rationale:** the owner's answer on 2026-10-11 in the earlier attempt (`old 793c7` items 1 and 2): volume
  equals what the agent's own lookup returns, and per-[turn](./793a_product-spec.md#793a1a-terms) and per-[session](../../GLOSSARY.md#session) bounds are dropped.
- **Alternatives:** a fixed maximum in the product spec ([793c3 Is the volume of delivered notes limited?](./793c_open-questions.md#793c3-is-the-volume-of-delivered-notes-limited)).
- **Decided by:** Author (default; owner may override).

## 793b20 No modes and no off switch

- **Choice:** [automatic memory](./793a_product-spec.md#793a1a-terms) is on whenever it is installed ([793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)). There is no observe mode and no
  off switch; stopping it means removing what [`engram update`](../../GLOSSARY.md#engram-update) added, and no
  requirement covers how.
- **Rationale:** the owner's answer in the earlier attempt (`old 793c12` (a), 2026-10-11): remove modes, the
  System is active whenever installed, and disabling or uninstalling stops it. The choice follows that
  answer, so it is a decision here and not a question for the owner. He also answered that uninstall is extended
  to remove it (`old 793c18` (a) and `old 793c22` (b), 2026-10-11); no requirement here covers that, which is
  row 10 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses).
- **Alternatives:** observe and active modes with a kill switch, as issue #793 expected.
- **Decided by:** Author (follows the owner's earlier answer; owner may override).

## 793b21 Lessons are written straight into the vault

- **Choice:** a lesson becomes a [note](../../GLOSSARY.md#permanent-note) directly ([793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)), not a [pending offer](../../GLOSSARY.md#offer) awaiting review.
- **Rationale:** the owner's answer on 2026-10-11 in the earlier attempt (`old 793c4`): "written directly to the
  [vault](../../GLOSSARY.md#vault)."
  Curation is excluded ([793b8 Exclusions](#793b8-exclusions)).
- **Alternatives:** pending offers reviewed with the curate [skill](../../GLOSSARY.md#skill), issue #793's open decision 4
  ([793c5 Are lessons written straight into the vault or held for review?](./793c_open-questions.md#793c5-are-lessons-written-straight-into-the-vault-or-held-for-review)).
- **Decided by:** Author (default; owner may override).

## 793b22 The scope is top-level sessions and notes only

- **Choice:** [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) apply to the [prompts](./793a_product-spec.md#793a1a-terms) and [turns](./793a_product-spec.md#793a1a-terms) of the user's own [sessions](../../GLOSSARY.md#session), not to
  [subagents](../../GLOSSARY.md#subagent), and delivery looks up [notes](../../GLOSSARY.md#permanent-note) only, not the chunk index.
- **Rationale:** both follow from "minimal" and from the hold of 8 seconds ([793b17 A prompt is held for at most 8 seconds](#793b17-a-prompt-is-held-for-at-most-8-seconds)): a lookup that included the chunk index
  took about 12 seconds on the owner's [vault](../../GLOSSARY.md#vault) in the earlier attempt.
- **Alternatives:** the owner's answers in the earlier attempt: capture inside subagent sessions (`old 793c6`,
  2026-10-11) and delivery at a subagent's start (`old 793c63`, 2026-10-11), memory for every prompt regardless of
  source (`old 793c5`, 2026-10-11), and chunk stubs (`old 793c15`, 2026-10-11; `old 793c87`, 2026-10-14). This
  entry reverses them for the minimal set. They are rows 1, 2 and 5 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses), and the owner is asked in [793c6 Is the scope top-level sessions and notes only?](./793c_open-questions.md#793c6-is-the-scope-top-level-sessions-and-notes-only) and
  [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse).
- **Decided by:** Author (default; owner may override).

## 793b23 No capture rate or precision targets

- **Choice:** [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) states no target; the scenarios of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) are the proof.
- **Rationale:** the owner's step 2 proves a requirement with scenarios. In the earlier attempt the owner set a
  capture rate above the 5.6% baseline and recorded precision (`old 793c7` items 7 and 8, 2026-10-11), then moved
  the measurement requirements, those targets included, out of the product spec into issue #850
  (`parallel memory agent: measurement and evaluation`; `old 793b182`, 2026-10-11). This change follows that
  move, so it is a decision here and not a question for the owner.
- **Alternatives:** the targets of issue #793's open decision 1, a precision bar of 70% and a capture rate of
  50%.
- **Decided by:** Author (follows the owner's earlier decision; owner may override).

## 793b24 A note holds no secret

- **Choice:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) says a [note](../../GLOSSARY.md#permanent-note) MUST NOT contain a credential, token, key or
  password that appeared in the [turn](./793a_product-spec.md#793a1a-terms). How it is met, and what risk remains, belong to the technical spec
  at step 3.
- **Rationale:** the owner's earlier decision, applied here: secrets are a MUST NOT (`old 793c7` item 10,
  2026-10-11). The earlier attempt's product spec dropped the clause when it was re-derived from his intent
  (`old 793b241`, 2026-10-12), but its technical spec kept pattern-based redaction, and he ratified the residual
  risk on 2026-10-13: secrets of shapes the patterns do not cover can still reach the model provider and the
  [vault](../../GLOSSARY.md#vault) (`old 793b324`, over `old 793d19a`). Automatic notes quote what the user typed,
  including anything typed in a correction, and the vault may be under version control. Review round 1 found that
  the first version of this entry, which left the rule out as unconfirmed scope, missed that decision.
- **Alternatives:** no rule in the product spec. Rejected: the owner decided otherwise.
- **Decided by:** Owner (Joe, 2026-10-11 and 2026-10-13, in the earlier attempt), applied here by the author.

## 793b25 What is shown after a turn, and when

- **Choice:** [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) applies when [automatic memory](./793a_product-spec.md#793a1a-terms) delivered a
  [note](../../GLOSSARY.md#permanent-note), wrote a note, or could not do a delivery or a write; a
  [turn](./793a_product-spec.md#793a1a-terms) with nothing to report shows nothing. By the time the user sends
  the next [prompt](./793a_product-spec.md#793a1a-terms), the user can see what was delivered for the previous
  prompt, what was written from the previous turn, and each delivery or write it could not do, with the reason.
  A write that finishes after the next prompt is sent is shown when it finishes. The requirement fixes no number
  of lines.
- **Rationale:** the earlier attempt omitted zero counts and showed nothing for a turn without events;
  something on every turn would be noise. Its intent was a one-line summary (`old 793b240`), but a write can
  finish up to 2 minutes after the turn ends ([793b18 Writing finishes within 2 minutes of a turn](./793b_decision-log.md#793b18-writing-finishes-within-2-minutes-of-a-turn)), so one line cannot always hold it. Tying the promise to
  the next prompt gives it a time the user can check; the first version, which showed the line "once automatic
  memory has finished its work for the turn", gave none.
- **Alternatives:** something shown on every turn, so the user can tell automatic memory is on ([793c9 Does a turn with nothing to report show anything?](./793c_open-questions.md#793c9-does-a-turn-with-nothing-to-report-show-anything)); the first
  version's single line, shown once the turn's work is finished.
- **Decided by:** Author (default; owner may override).

## 793b26 The line waits for the turn's work

Folded into [793b25 What is shown after a turn, and when](./793b_decision-log.md#793b25-what-is-shown-after-a-turn-and-when) in review round 1. The ID is kept so that links do not dangle.

## 793b27 "Already has" is per session

- **Choice:** in [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) the [agent](./793a_product-spec.md#793a1a-terms) "already has" a [note](../../GLOSSARY.md#permanent-note) when [automatic memory](./793a_product-spec.md#793a1a-terms) delivered it earlier in the
  same [session](../../GLOSSARY.md#session) or the agent looked it up in it, even after the session's context was compacted.
- **Rationale:** it is the literal reading of "this session" and needs no knowledge of what the agent still
  holds. It reverses the owner's answer in the earlier attempt (`old 793c7` item 5, 2026-10-11): never re-inject
  means never a note already in the receiving agent's context, and a truncated note is not present, so a note
  removed by a compaction could be delivered again. It is row 11 of [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses), and the owner is asked in [793c10 Does "already has" survive a compaction?](./793c_open-questions.md#793c10-does-already-has-survive-a-compaction) and
  [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse).
- **Alternatives:** deliver a note again after the context was compacted ([793c10 Does "already has" survive a compaction?](./793c_open-questions.md#793c10-does-already-has-survive-a-compaction)).
- **Decided by:** Author (default; owner may override).

## 793b28 Depth-first order

- **Choice:** the groups in order: [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) and [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) together, with the 8-second hold of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way); then [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent); then
  [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory); then the rest of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way); then [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did). The first node is the first scenario of [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update), which needs the first
  delivery of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) to be visible. An interaction is worked with the first group whose requirements include
  every capability it names: [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later), [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again) and [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working) with the second, because each names the writing of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent).
  The other interactions of the first version were folded or removed in review round 1 ([793b7 The interactions are product requirements too](./793b_decision-log.md#793b7-the-interactions-are-product-requirements-too)).
- **Rationale:** the only user-visible proof that the first is on is the second working in a new
  [session](../../GLOSSARY.md#session). The hold belongs with the first delivery, because a delivery must not
  hold a [prompt](./793a_product-spec.md#793a1a-terms) longer than [793b17 A prompt is held for at most 8 seconds](./793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds) allows from the start. The lessons come next, as the learning half of
  the North Star, and the sweep follows them as the rest of what the main [agent](./793a_product-spec.md#793a1a-terms)'s learn did ([793b34 The goal: what the main agent was already doing, just async](./793b_decision-log.md#793b34-the-goal-what-the-main-agent-was-already-doing-just-async)). The
  rest of the bounds and the display come last because they bound and report what the others do. An interaction
  cannot be exercised before everything it joins exists, and delaying it further would leave its rule unchecked.
- **Alternatives:** the order of the requirements as listed, with the install alone first. Rejected: its
  scenarios could not be run before delivery exists. All interactions in a final node (rules unchecked for
  long), or each with its own earliest capability (it cannot be exercised there).
- **Decided by:** Author.

## 793b29 Interactions join the first node that has every capability they name

Merged into [793b28 Depth-first order](./793b_decision-log.md#793b28-depth-first-order) in review round 1. The ID is kept so that links do not dangle.

## 793b30 Commit hashes are recorded by the next commit

Folded into [793b13 Departure: one refactor pass, and extra catalog columns](#793b13-departure-one-refactor-pass-and-extra-catalog-columns)
in review round 1, which records the extra catalog columns as a departure. Not a separate decision; the ID is
kept so that links do not dangle.

## 793b31 Glossary terms stay in the Terms list until promotion

Withdrawn in review round 1: not a decision. A term the glossary lacks is defined in the Terms list of the
product spec, where the guides put it, and nothing was decided. The ID is kept so that links do not dangle.

## 793b32 Test isolation

- **Choice:** every literal run of a product scenario, and every test or check that runs `engram` or `claude`,
  uses its own HOME, CLAUDE_CONFIG_DIR, GOBIN, ENGRAM_VAULT_PATH, ENGRAM_CHUNKS_DIR and ENGRAM_TRANSCRIPT_DIR,
  with a working directory outside any `.claude` or `.git` ancestor. `dev/eval/isolation.py` builds the engram
  and Claude variables and checks the working directory this way; HOME and GOBIN are what this rule adds.
  "The user runs [`engram update`](../../GLOSSARY.md#engram-update)" in a scenario means this branch's binary,
  built into the run's own GOBIN and run with that run's HOME. The owner's real `~/.claude`,
  `~/.local/share/engram`, `~/.pi` and GOBIN are never touched by the orchestrating
  [agent](./793a_product-spec.md#793a1a-terms), a [subagent](../../GLOSSARY.md#subagent) or a test. The rule is written into
  [793f7 Test isolation](./793f_algorithm.md#793f7-test-isolation),
  [793e2 Test methods](./793e_implementation-plan.md#793e2-test-methods) and
  [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates).
- **Rationale:** the owner's isolation rule from the earlier attempt, set between 2026-10-11 and 2026-10-16 and
  relayed to the author: "temp HOMEs only; every engram/claude command from agents needs explicit env HOME".
  `old 793b245` records its first half: "the owner's isolation rule (temporary HOMEs only)". Review round 1 of
  step 1 found the rule missing from this change's plan and algorithm. Without it, a literal run of "the user
  runs `engram update`" installs into the owner's real `~/.claude`, and an
  [`engram ingest --auto`](../../GLOSSARY.md#engram-ingest-subcommand) run from a directory under a `.claude` or
  `.git` ancestor sweeps the owner's own sources into the run's index, so the rule is restated here.
- **Alternatives:** none: it is the owner's rule.
- **Decided by:** Owner (Joe, in the earlier attempt, 2026-10-11 to 2026-10-16), restated here after review
  round 1.

## 793b33 One agent review round per phase

- **Choice:** each phase gets one round of review, unless the owner asks for more. The author accepts or
  declines each finding and records the outcome; findings are not argued to resolution. For this change that
  supersedes the default of the root [AGENTS.md](../../../AGENTS.md#issue-workflow-default), whose implementation
  and plan reviews are argued to resolution.
- **Rationale:** the owner, in the earlier attempt on 2026-10-11, as relayed to the author: "one agent review
  round per phase unless asked." `old 793b174` records the same cap for a change spec, and
  [docs/AGENTS.md](../../AGENTS.md#decisions-and-open-questions) carries it ("A change spec MUST get one agent
  review round, unless the owner asks for more").
- **Alternatives:** argue each review to resolution, as the root default does. Superseded by the owner's rule.
- **Decided by:** Owner (Joe, 2026-10-11, in the earlier attempt).

## 793b34 The goal: what the main agent was already doing, just async

- **Choice:** the minimal set is read against the owner's statement of the goal. The main
  [agent](./793a_product-spec.md#793a1a-terms)'s recall ran one `engram query` before acting; its learn ran
  [`engram ingest --auto`](../../GLOSSARY.md#engram-ingest-subcommand) as a sweep and then wrote
  [notes](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault) for explicit lessons.
  [Automatic memory](./793a_product-spec.md#793a1a-terms) does those same things without the agent:
  [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) delivers what one query before acting would have returned, [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) is the sweep, and [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) writes the
  notes for explicit lessons. What the main agent did only by its own judgement mid-task, re-querying at moments
  it chose, is not in the set ([793b8 Exclusions](#793b8-exclusions)).
- **Rationale:** the owner, 2026-10-16, when asked why behavior was being adjusted in the earlier attempt, as
  relayed to the author: "The goal was to do what the main agent was already doing, just async."
- **Alternatives:** none.
- **Decided by:** Owner (Joe, 2026-10-16).

## 793b35 The transcript sweep is in the minimal set

- **Choice:** [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) is a requirement: within 2 minutes after a [turn](./793a_product-spec.md#793a1a-terms) ends,
  the [session](../../GLOSSARY.md#session)'s conversation so far can be found by the
  [agent](./793a_product-spec.md#793a1a-terms)'s own memory lookup (`engram query`), without anyone running a
  sweep. The first version of this change left the sweep out, together with the chunk-index deep lookup
  ([793b8 Exclusions](#793b8-exclusions)); only the deep lookup, which delivers conversation fragments with a
  [prompt](./793a_product-spec.md#793a1a-terms)'s [notes](../../GLOSSARY.md#permanent-note), stays out. The 2 minutes
  are the default of [793b18 Writing finishes within 2 minutes of a turn](#793b18-writing-finishes-within-2-minutes-of-a-turn).
- **Rationale:** the review's recommendation, accepted. The main agent's learn ran the sweep as its first job
  ([793b34 The goal: what the main agent was already doing, just async](#793b34-the-goal-what-the-main-agent-was-already-doing-just-async)). Leaving it out automates only half of learning and lets the chunk layer go stale, so the agent's own
  lookup would stop finding recent conversation. The owner's intent in the earlier attempt included it: indexing
  after the turn, never making anyone wait (`old 793b240`, 2026-10-12), swept asynchronously at turn-end capture
  (`old 793c27`, 2026-10-11).
- **Alternatives:** leave the sweep to the agent's own learn [skill](../../GLOSSARY.md#skill), as in the first
  version.
- **Decided by:** Author, on the reviewer's recommendation (opus review round 1, 2026-10-17); owner may override.

## 793b36 Earlier owner answers this minimal set reverses

- **Choice:** the minimal set reverses these answers the owner gave in the earlier attempt. Each reversal's
  default stands as the author's, and [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse) asks the owner about all of them together. Each row names the earlier
  entries and their dates, then what this change does:
  1. [Subagents](../../GLOSSARY.md#subagent): capture inside subagent [sessions](../../GLOSSARY.md#session) (`old 793c6`, 2026-10-11), and delivery at a [subagent](../../GLOSSARY.md#subagent)'s start
     with capture at its stop (`old 793c63` (a), 2026-10-11). Here: the user's own sessions only ([793b22 The scope is top-level sessions and notes only](./793b_decision-log.md#793b22-the-scope-is-top-level-sessions-and-notes-only)).
  2. [Prompt](./793a_product-spec.md#793a1a-terms) sources: memory for every prompt regardless of source (`old 793c5`, 2026-10-11). Here: prompts
     the user sends.
  3. Route results: tier evidence before the orchestrator routes a unit (`old 793c39` (b) and `old 793c64` (a),
     2026-10-11). Here: none.
  4. Mid-task moments: delivery at the moments where recall should fire (`old 793b187`, 2026-10-11, and
     `old 793b240`, 2026-10-12; `old 793c49` (b) and `old 793c55` (a), 2026-10-11). Here: the prompt only
     ([793c12 Is memory delivered at mid-task moments?](./793c_open-questions.md#793c12-is-memory-delivered-at-mid-task-moments)).
  5. Chunk stubs: conversation fragments delivered beside the [notes](../../GLOSSARY.md#permanent-note) (`old 793c15` (b), 2026-10-11), kept by a
     background pass when the full lookup proved too slow (`old 793c87` (a), 2026-10-14). Here: none delivered.
  6. Relevance filter and moment classification: two model judgements before delivery (`old 793c69` (a),
     2026-10-12), kept on haiku at the lowest effort (`old 793c79` (a), 2026-10-14). Here: none.
  7. Curation after the [turn](./793a_product-spec.md#793a1a-terms): indexing and curation after each turn (`old 793b240`, 2026-10-12), curation run by
     the System (`old 793c43` (a), 2026-10-11). Here: indexing is back as [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory); curation of
     [pending offers](../../GLOSSARY.md#offer) is out.
  8. Backlog capture: leftover sessions of any [harness](../../GLOSSARY.md#harness) picked up by the next session (`old 793c67`,
     2026-10-12; `old 793c65` (a), 2026-10-11; cut-off `old 793c75`, 2026-10-13). Here: none ([793b37 A turn followed by the end of the session is not covered](./793b_decision-log.md#793b37-a-turn-followed-by-the-end-of-the-session-is-not-covered)).
  9. Reinforcing a note: a MUST (`old 793c14` (b), 2026-10-11), no longer named in the intent of `old 793b240`.
     Here: a lesson the [vault](../../GLOSSARY.md#vault) already states is not written again, and nothing more.
  10. Stopping it: turned off by uninstalling, with uninstall extended to remove it (`old 793c18` (a) and
      `old 793c22` (b), 2026-10-11). Here: no requirement covers how ([793b20 No modes and no off switch](./793b_decision-log.md#793b20-no-modes-and-no-off-switch)).
  11. "Already has": a note already in the receiving [agent](./793a_product-spec.md#793a1a-terms)'s context is never re-injected, and a truncated
      note is not present (`old 793c7` item 5, 2026-10-11). Here: per session, even after a compaction ([793b27 "Already has" is per session](./793b_decision-log.md#793b27-already-has-is-per-session)).
  12. The prompt hold: no longer than the agent's own lookup on the same vault and input (`old 793c7` item 4 and
      `old 793c11` (b), 2026-10-11). Here: 8 seconds ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)).
- **Rationale:** the owner asked for the minimal set (step 1 of his algorithm) and a restart from before the mod
  existed ([793b1 Start over](#793b1-start-over)). Each reversed capability needs a measurement, a second model
  judgement or machinery that the set leaves out, and a separate change can bring any of them back.
- **Alternatives:** keep any of the twelve in the minimal set ([793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse)).
- **Decided by:** Author (default; owner may override).

## 793b37 A turn followed by the end of the session is not covered

- **Choice:** [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory) promise their work within 2 minutes after a [turn](./793a_product-spec.md#793a1a-terms) ends, and only while the
  [session](../../GLOSSARY.md#session) stays open that long: a turn followed by the end of the session within the 2 minutes is not covered.
  [793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered) asks whether it should be.
- **Rationale:** covering it needs work that outlives the session, a pick-up by a later session, or a hold on the
  exit, and the minimal set leaves out backlog capture ([793b8 Exclusions](./793b_decision-log.md#793b8-exclusions)) and may not hold a session ([793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way)). The earlier
  attempt recorded such a final turn as lost (`old 793b203`) before it kept unfinished work for the next session
  (`old 793c65`, 2026-10-11) and then picked up leftover sessions (`old 793c67`, 2026-10-12). The cost is that a
  lesson in the last turn before an exit can be lost.
- **Alternatives:** covered, by letting the work finish after the session ends or by finishing it in the next
  session ([793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered)).
- **Decided by:** Author (default; owner may override).

## 793b38 Depth first: two readings, and the one this change takes

- **Choice:** the owner's "Work depth first through the above algorithm." has two readings. (1) Per requirement:
  each requirement goes through steps 2 to 5 before the next one starts, as the first version of
  [793f3 Depth first](./793f_algorithm.md#793f3-depth-first) read it. (2) Per scenario: step 2 runs for one
  requirement, and then each product scenario, or a tightly coupled pair, is a node that goes through steps 3 to
  5 before the next node starts. This change takes (2). The first node is the first scenario of [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update), which
  needs the first delivery of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) to be visible. [793f3 Depth first](./793f_algorithm.md#793f3-depth-first),
  [793f4 Delegation](./793f_algorithm.md#793f4-delegation) and
  [793e4 Depth-first order and dependency tree](./793e_implementation-plan.md#793e4-depth-first-order-and-dependency-tree) say so.
- **Rationale:** the owner's step 3 reads "for each acceptance test", and an acceptance test is a product
  scenario, so steps 3 to 5 go one scenario at a time; his step 2 reads "for each product requirement", so
  step 2 stays per requirement.
- **Alternatives:** reading (1), the first version. Rejected for the reason above.
- **Decided by:** Author (reading of the owner's step 3; owner may override), on review round 1's finding.
