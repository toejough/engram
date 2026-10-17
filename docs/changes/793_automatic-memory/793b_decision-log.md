# Automatic Memory - Decision Log

Decisions made while producing [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) (#793) and the process around it, recorded for Joe to validate.
"Decided by" follows [Decisions and Open Questions](../../AGENTS.md#decisions-and-open-questions): Owner for
Joe's own words, Author for the author's derivations and defaults, Reviewer for what a review found and
the author accepted. Joe was unavailable, so every question the author would have asked has a default, recorded
here as "Decided by: Author (default; owner may override)" and listed in
[793c Automatic Memory - Open Questions](./793c_open-questions.md#automatic-memory---open-questions).

An entry written in code as `old 793bNNN` or `old 793cNNN` is an entry of the earlier attempt (change directory
`793_parallel-memory-agent` on branch `docs/parallel-memory-agent-requirements`, as of commit `dcf53d07`), not
an entry of this change, though the numbers overlap.

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
  step 1 ([793b39 Review round 1 of step 1](#793b39-review-round-1-of-step-1)) found the rule missing from this change's plan and algorithm. Without it, a literal run of "the user
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
     the user sends; until [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) the mod delivers for every origin ([793b58 The first group delivers for every origin of a prompt](#793b58-the-first-group-delivers-for-every-origin-of-a-prompt)).
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
  13. Sessions covered: the promise covers the sessions in which the earlier attempt's mod runs, and
      [`engram update`](../../GLOSSARY.md#engram-update) reports when mods cannot load (`old 793c66` (a), 2026-10-11). Here: every Claude Code
      session the user starts after the update ([793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)); where it cannot run, [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) has it step aside, and no
      check is required.
  14. Activity for the agent: the activity lines may reach the agent (`old 793c90` (b), 2026-10-15), and the
      previous turn's sentences ride the next prompt (`old 793c92` (b), 2026-10-16). Here: the user sees the
      activity in the session ([793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did)); nothing carries it to the agent.
- **Rationale:** the owner asked for the minimal set (step 1 of his algorithm) and a restart from before the mod
  existed ([793b1 Start over](#793b1-start-over)). Each reversed capability needs a measurement, a second model
  judgement or machinery that the set leaves out, and a separate change can bring any of them back.
- **Alternatives:** keep any of the fourteen in the minimal set ([793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse)).
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

## 793b39 Review round 1 of step 1

- **Choice:** the step 1 documents, as first written in seven commits (`1bcf81b3` to `57372f32`), had one review
  round, run by opus, which returned approve with fixes. The author accepted every finding and declined none.
  The findings, by severity, and where each is applied:
  - **Critical:** nothing kept the literal runs of scenarios, or any test or command that runs `engram` or
    `claude`, off the owner's real home, [vault](../../GLOSSARY.md#vault), `~/.pi` and GOBIN. Applied in [793b32 Test isolation](./793b_decision-log.md#793b32-test-isolation),
    [793f7 Test isolation](./793f_algorithm.md#793f7-test-isolation), [793e2 Test methods](./793e_implementation-plan.md#793e2-test-methods) and [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates).
  - **Major:** three departures were credited to the owner but are the author's readings of his words: [793b11 Departure: no interview or approval gates](./793b_decision-log.md#793b11-departure-no-interview-or-approval-gates),
    [793b12 Departure: no option analysis](./793b_decision-log.md#793b12-departure-no-option-analysis) and [793b13 Departure: one refactor pass, and extra catalog columns](./793b_decision-log.md#793b13-departure-one-refactor-pass-and-extra-catalog-columns). [793f6 Departures from the guides](./793f_algorithm.md#793f6-departures-from-the-guides) now says which two are his.
  - **Major:** the owner's statement of the goal was missing, and mid-task moments were not listed as left out:
    [793b34 The goal: what the main agent was already doing, just async](./793b_decision-log.md#793b34-the-goal-what-the-main-agent-was-already-doing-just-async), [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions) and [793c12 Is memory delivered at mid-task moments?](./793c_open-questions.md#793c12-is-memory-delivered-at-mid-task-moments).
  - **Major:** the earlier owner answers that the set reverses were not listed, and [793b27 "Already has" is per session](./793b_decision-log.md#793b27-already-has-is-per-session) stated a reversal that
    [793c10 Does "already has" survive a compaction?](./793c_open-questions.md#793c10-does-already-has-survive-a-compaction) did not: [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses) and [793c13 Which earlier owner answers may the minimal set reverse?](./793c_open-questions.md#793c13-which-earlier-owner-answers-may-the-minimal-set-reverse).
  - **Major:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) forbade changing "what the [agent](./793a_product-spec.md#793a1a-terms) answered"; it now forbids editing the user's
    [prompt](./793a_product-spec.md#793a1a-terms) and the text of the reply.
  - **Major:** three of the six interactions restated one requirement's rule: [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update), [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt), [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) and
    [793b7 The interactions are product requirements too](./793b_decision-log.md#793b7-the-interactions-are-product-requirements-too).
  - **Major:** what the user sees, and when, was vague: [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) and [793b25 What is shown after a turn, and when](./793b_decision-log.md#793b25-what-is-shown-after-a-turn-and-when).
  - **Major:** the MUST NOT on secrets that the owner set was missing: [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), [793b24 A note holds no secret](./793b_decision-log.md#793b24-a-note-holds-no-secret) and [793c8 What about secrets in written notes?](./793c_open-questions.md#793c8-what-about-secrets-in-written-notes).
  - **Minor:** the 8-second bound was called the owner's, and the 2-minute delay lacked a statement that a measurement
    replaces it: [793b17 A prompt is held for at most 8 seconds](./793b_decision-log.md#793b17-a-prompt-is-held-for-at-most-8-seconds), [793b18 Writing finishes within 2 minutes of a turn](./793b_decision-log.md#793b18-writing-finishes-within-2-minutes-of-a-turn), [793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held) and [793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing).
  - **Minor:** [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later) said "later prompts"; [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) did not say which [notes](../../GLOSSARY.md#permanent-note) bear
    on a prompt; [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)'s rerun, other [harness](../../GLOSSARY.md#harness) and dry-run clauses were untestable or
    missing; [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) named three kinds of lesson, not the learn [skill](../../GLOSSARY.md#skill)'s four.
  - **Minor:** defined words stood where the number or the plain phrase belonged (`wait bound`, `capture delay`,
    `provenance`, skipped steps): [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent), [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) and [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did).
  - **Minor:** the problem statement paraphrased the North Star, ended on "cannot tell when it was not" and held
    a paragraph about the process: [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement) and [793f5 How this change applies it](./793f_algorithm.md#793f5-how-this-change-applies-it).
  - **Minor:** [automatic memory](./793a_product-spec.md#793a1a-terms) could act on its own [sessions](../../GLOSSARY.md#session), and a
    [turn](./793a_product-spec.md#793a1a-terms) followed by the end of the session was not addressed: [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), [793b37 A turn followed by the end of the session is not covered](./793b_decision-log.md#793b37-a-turn-followed-by-the-end-of-the-session-is-not-covered) and [793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered).
  - **Minor:** the product spec held a section beyond the guide's three, now in [793b8 Exclusions](./793b_decision-log.md#793b8-exclusions); `793c4`, `793c7` and
    `793c11` were not blocking questions, and the first two are now only in [793b20 No modes and no off switch](./793b_decision-log.md#793b20-no-modes-and-no-off-switch) and [793b23 No capture rate or precision targets](./793b_decision-log.md#793b23-no-capture-rate-or-precision-targets).
  - **Minor:** the decision log held entries that were not decisions or repeated another, now stubs: [793b14 Departure: the North Star line](./793b_decision-log.md#793b14-departure-the-north-star-line),
    [793b15 Artifact names](./793b_decision-log.md#793b15-artifact-names), [793b16 One subject term](./793b_decision-log.md#793b16-one-subject-term), [793b26 The line waits for the turn's work](./793b_decision-log.md#793b26-the-line-waits-for-the-turns-work), [793b29 Interactions join the first node that has every capability they name](./793b_decision-log.md#793b29-interactions-join-the-first-node-that-has-every-capability-they-name), [793b30 Commit hashes are recorded by the next commit](./793b_decision-log.md#793b30-commit-hashes-are-recorded-by-the-next-commit) and [793b31 Glossary terms stay in the Terms list until promotion](./793b_decision-log.md#793b31-glossary-terms-stay-in-the-terms-list-until-promotion).
  - **Minor:** the commit columns were described as a record the owner asked for: [793b13 Departure: one refactor pass, and extra catalog columns](./793b_decision-log.md#793b13-departure-one-refactor-pass-and-extra-catalog-columns).
  - **Minor:** the depth-first reading was not examined ([793b38 Depth first: two readings, and the one this change takes](./793b_decision-log.md#793b38-depth-first-two-readings-and-the-one-this-change-takes), [793f3 Depth first](./793f_algorithm.md#793f3-depth-first), [793f4 Delegation](./793f_algorithm.md#793f4-delegation), [793e4 Depth-first order and dependency tree](./793e_implementation-plan.md#793e4-depth-first-order-and-dependency-tree) and [793b10 Departure: depth-first fill](./793b_decision-log.md#793b10-departure-depth-first-fill)); [793b5 Reading of the delegation rule](./793b_decision-log.md#793b5-reading-of-the-delegation-rule) rested on the
    owner's silence about reviews ([793b33 One agent review round per phase](./793b_decision-log.md#793b33-one-agent-review-round-per-phase)); product-scenario tests were described four ways ([793b9 Test execution](./793b_decision-log.md#793b9-test-execution), [793e2 Test methods](./793e_implementation-plan.md#793e2-test-methods), [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates) and
    [793f5 How this change applies it](./793f_algorithm.md#793f5-how-this-change-applies-it)); the 8-second hold belongs in the first node ([793e4 Depth-first order and dependency tree](./793e_implementation-plan.md#793e4-depth-first-order-and-dependency-tree) and [793b28 Depth-first order](./793b_decision-log.md#793b28-depth-first-order)).
  - **Nit:** [793b10 Departure: depth-first fill](./793b_decision-log.md#793b10-departure-depth-first-fill) did not name the placeholders as the departure; [793e2 Test methods](./793e_implementation-plan.md#793e2-test-methods) named categories the plan guide does not
    define; [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates) lacked the guide's gates for `rapid` properties, `imptest` evidence and valid- and invalid-path
    behavior; [793f4 Delegation](./793f_algorithm.md#793f4-delegation) defined a phase over requirements, not nodes.
  - **Recommendation:** bring the [transcript](../../GLOSSARY.md#transcript) sweep into the minimal set ([793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory),
    [793b35 The transcript sweep is in the minimal set](./793b_decision-log.md#793b35-the-transcript-sweep-is-in-the-minimal-set) and [793e4 Depth-first order and dependency tree](./793e_implementation-plan.md#793e4-depth-first-order-and-dependency-tree)); keep [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did), with its mobile view parked ([793c15 Should the activity be shown in the agent's reply, so that it reaches the claude.ai mobile and web view?](./793c_open-questions.md#793c15-should-the-activity-be-shown-in-the-agents-reply-so-that-it-reaches-the-claudeai-mobile-and-web-view)).

  The IDs `793a3b`, `793a3d`, `793a3f`, `793a4`, `793c4`, `793c7` and `793c11` are retired and never reused.
- **Rationale:** [Decisions and Open Questions](../../AGENTS.md#decisions-and-open-questions) has each review round
  record its outcome in the decision log, and its questions in the open-questions file, before anything goes to
  the owner.
- **Alternatives:** decline a finding. None was declined.
- **Decided by:** Reviewer (opus review round 1, 2026-10-17), accepted by the author.

## 793b40 The scenarios of the first group

- **Choice:** step 2 for the first group ([793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt))
  writes ten scenarios, one per mandatory branch and no more: five under [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update)
  (the next [session](../../GLOSSARY.md#session) has memory in any project; a second update still delivers once; a dry run [turns](./793a_product-spec.md#793a1a-terms) nothing on; Pi gets
  nothing; a [vault](../../GLOSSARY.md#vault) that does not exist is not created), four under [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)
  (the [notes](../../GLOSSARY.md#permanent-note) that bear on a [prompt](./793a_product-spec.md#793a1a-terms) arrive with it; a note delivered earlier is not delivered again; a note the [agent](./793a_product-spec.md#793a1a-terms)
  looked up itself is not delivered; a prompt that matches no note, even when the conversation matches, arrives with
  nothing) and one under [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) (a slow lookup does not hold a prompt for more
  than 8 seconds). The clause of [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) about lessons is proved by a scenario written with [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent).
  The scenario for the hold is in this group because [793e4 Depth-first order and dependency tree](./793e_implementation-plan.md#793e4-depth-first-order-and-dependency-tree) gives the first group the
  8-second hold, and the other scenarios of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) wait for the group that finishes it.
- **Rationale:** the owner's step 2 asks for the minimum scenarios that prove the requirement, and the product guide
  asks for end-user-verifiable coverage of every mandatory branch. Two shapes are the author's: "the agent looked the
  note up itself" is staged before the update and the session is resumed after it, because a prompt of that session
  would otherwise deliver the note too and the two routes could not be told apart; and the missing vault is a vault
  that was removed after the update, because [`engram update`](../../GLOSSARY.md#engram-update) itself creates a missing vault, so a machine that never
  ran it is not a machine on which the mod is installed.
- **Alternatives:** fewer scenarios, folding the dry run and Pi into the first (rejected: their outcomes differ and
  each needs its own report); a scenario for each of the delivered-note routes inside one (rejected: the plan guide
  has one test per scenario ID, and the two routes are different interactions).
- **Decided by:** Author (default; owner may override).

## 793b41 The mod, its place and its command

- **Choice:** the automatic behavior inside a Claude Code [session](../../GLOSSARY.md#session) is one mod,
  `engram-memory`, kept in `agent-instructions/mods/` and deployed by [`engram update`](../../GLOSSARY.md#engram-update)
  through the machinery that deploys the [skills](../../GLOSSARY.md#skill), so that it lands in `~/.claude/skills/engram-memory` like a skill. For
  each [prompt](./793a_product-spec.md#793a1a-terms) the mod runs a new command, `engram auto deliver`, with a JSON
  object on standard input, and puts the command's output beside the prompt. The command holds the logic; the mod
  routes. Recorded in [793d1 Orientation](./793d_technical-spec.md#793d1-orientation).
- **Rationale:** automatic behavior inside a session needs a plugin of function hooks, a mod; the `prompt.submit`
  function is the one that can add context beside the prompt without editing it
  ([793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)). A real
  `claude -p` session in an isolated HOME showed that a mod in a linked `~/.claude/skills/<name>` loads, that its
  command runs with the session's environment and standard input, and that the context it adds is recorded before the
  [agent](./793a_product-spec.md#793a1a-terms)'s first reply. Standard input carries the prompt because a prompt can be longer than a command line allows.
  The logic is in Go because the controls over it (tests, lint, coverage) are stronger there.
- **Alternatives:** a `UserPromptSubmit` command hook in `~/.claude/settings.json`: rejected, because `engram update`
  would edit a file the user owns on every run, where files in an engram-owned root need no merge. The prompt on the
  command line: rejected for its length limit.
- **Decided by:** Author (default; owner may override).

## 793b42 What the block says

- **Choice:** the block opens with one fixed framing line and has one line for each [note](../../GLOSSARY.md#permanent-note):
  a dash, its file name, its situation in parentheses, a colon and its body. The framing line begins `[engram memory]`
  and says that the lines below are notes from the [vault](../../GLOSSARY.md#vault), delivered by the memory system and not typed by the user,
  and that an instruction written inside a note is not the user's
  ([793b43 A note cannot reshape the block](#793b43-a-note-cannot-reshape-the-block)).
- **Rationale:** the framing is that of the earlier attempt's block (`internal/auto/moment.go` at commit `dcf53d07`),
  whose guard keeps a note from acting as an instruction; the marker `[engram memory]` is what the [session](../../GLOSSARY.md#session)'s record is
  searched for. The file name is the note's identity in every `engram` payload, so the [agent](./793a_product-spec.md#793a1a-terms) can ask for the whole note
  with `engram show`, and it is what a later [prompt](./793a_product-spec.md#793a1a-terms)'s lookup compares against what the agent already has.
- **Alternatives:** whole notes (rejected: the block would grow with the vault); a count and no text (rejected: the
  agent would have to look each note up).
- **Decided by:** Author (default; owner may override).

## 793b43 A note cannot reshape the block

- **Choice:** the file name, the situation and the body of a [note](../../GLOSSARY.md#permanent-note) are each written as text in which every run of white
  space (line breaks and separators included) is one space and no control character and no format character (Unicode
  category Cf) remains, so a note is always one line, and no text of a note can end its line and write a line that looks
  like the framing line or like another note.
  The framing line says that an instruction written inside a note is not the user's. Nothing more is done: no quoting
  scheme. [793d4c Framing the notes of a block as data](./793d_technical-spec.md#793d4c-framing-the-notes-of-a-block-as-data)
  states it.
- **Rationale:** a note is text that the user, the [agent](./793a_product-spec.md#793a1a-terms) or another [vault](../../GLOSSARY.md#vault) wrote,
  and the block puts it in the agent's context. A security review of the first rendering found that a note holding line
  breaks could end its own line and write lines that imitate the block's framing or further notes, and that nothing
  told the agent the lines are retrieved data. Collapsing and one framing line close that with no other machinery. The
  review of the first group added the format characters: a zero-width space or joiner, a bidirectional override and a
  byte-order mark are not control characters, yet they are invisible or reorder the text, so a note could carry an
  instruction that a person reading the [session](../../GLOSSARY.md#session) does not see; they are dropped, as control characters are, and the words
  around them are joined.
- **Alternatives:** a quoting or escaping scheme (rejected: more code for the same guarantee, since the model reads the
  lines either way); leaving out a note that holds control characters (rejected: the note would be lost).
- **Decided by:** Reviewer (security review of the delivery node's rendering, relayed by the coordinator,
  2026-10-17), accepted by the author. The format characters are the review of the first group's finding 20, accepted by
  the author.

## 793b44 Only the notes that match are delivered, whole

- **Choice:** a [note](../../GLOSSARY.md#permanent-note) is delivered when the lookup returns it as a matched item (not a
  chunk, and not picked by the explore half alone), and its body is delivered whole, on one line. There is no cut of
  the body: the lookup's own limit bounds the number of notes
  ([793b19 Volume of delivered notes](#793b19-volume-of-delivered-notes)), as it bounds what the [agent](./793a_product-spec.md#793a1a-terms)'s own lookup
  returns, and the agent's own lookup prints whole notes.
- **Rationale:** an explore pick is a note sampled from the vocabulary to widen what the agent reads, not one that
  bears on the [prompt](./793a_product-spec.md#793a1a-terms); delivering it with every prompt would put an unrelated note in the agent's context each time,
  which [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does not
  ask for. A cut of the body would hide the later steps of a runbook, and the scenarios do not need one.
- **Alternatives:** deliver every non-chunk item of the payload (rejected: it delivers explore picks); cut each body
  to 400 characters, as the earlier attempt did (rejected: no scenario needs it, and a cut loses what the agent would
  have read from its own lookup).
- **Decided by:** Author (default; owner may override).

## 793b45 What a session holds

- **Choice:** the [agent](./793a_product-spec.md#793a1a-terms) holds a [note](../../GLOSSARY.md#permanent-note) when the note's basename, its file name without
  the suffix `.md`, appears in what the [session](../../GLOSSARY.md#session)'s record shows the agent was given about notes. Two things count: the
  file names on the note lines of the blocks delivered beside [prompts](./793a_product-spec.md#793a1a-terms), and the command and the result of each lookup the
  agent ran itself, a Bash command whose text contains `engram query` or `engram show`, its result paired with it by the
  tool's ID. Nothing else of the record counts: not the other context that a hook put beside a prompt, not the situations
  and bodies of a block, not the result of any other tool, such as an `ls` of the [vault](../../GLOSSARY.md#vault) or a `cat` of a note file. The
  whole record counts, whatever a compaction removed ([793b27 "Already has" is per session](#793b27-already-has-is-per-session)).
  The record is found by the session's ID, in the directory of records of sessions
  ([793b51 The delivery finds the records where Claude Code writes them](#793b51-the-delivery-finds-the-records-where-claude-code-writes-them)),
  directly in it or in a subdirectory for a project. A session with no record holds nothing, which is the first prompt
  of every session. An ID that is not a plain file name holds nothing, so an ID cannot name a file outside the directory.
- **Rationale:** the record is the only durable account of what reached the agent: the blocks a hook adds are recorded
  there as `hook_additional_context` records before the agent's first reply, and a resumed session reads the same
  record, so a module's own memory of what it delivered would be lost with the process. A file name is how every
  `engram` payload and every block names a note, so comparing it is enough, and needs no parsing of the note. The first
  version counted a note as held when its file name appeared anywhere in any tool result. The review of the first group
  reproduced both errors of that: a session whose only contact with the vault was an `ls` of it was never delivered the
  note it had not read, and `engram show <basename>` prints the note without its file name, so a note fetched that way
  was delivered again. The basename is what `engram show` takes and what a wikilink holds, and every file name that names
  the note contains it. A note that the agent read by another command (`cat`, `grep`, a read of the file) is not held
  and is delivered again: that is the cost of counting only the lookups the agent is told to use.
- **Alternatives:** a ledger of delivered notes kept by the mod or the command (rejected: it is lost when the process ends
  and needs a place to live); the session's messages from the module's `$.session.messages()` (rejected: they do not
  include the context a hook added, which this checked); every tool result, the first version (rejected: reproduced).
- **Decided by:** Author (default; owner may override); the pairing of lookups with their results and the basename are
  the review's findings, accepted by the author.

## 793b46 The command is ended at 7 seconds

- **Choice:** the mod waits for `engram auto deliver` for 7 seconds at most, by the time limit of the process call, which
  ends the command at that time, and a [prompt](./793a_product-spec.md#793a1a-terms) whose command did not finish goes on with nothing added, with no sign of
  a failure in the [session](../../GLOSSARY.md#session). The 8 seconds of [793b17 A prompt is held for at most 8 seconds](#793b17-a-prompt-is-held-for-at-most-8-seconds)
  are the hold of the prompt: the 7 seconds are the command, and the second left is for starting the process, the call
  and the prompt's hooks to run: a prompt whose command takes 20 seconds waited 7.07 seconds before the model was asked
  (the scenario [793d2a1d A slow command is ended and the prompt continues](./793d_technical-spec.md#793d2a1-scenarios), `targ acceptance --run Test793d2a1d`).
- **Rationale:** the platform's own limit on a hook counts only the hook's own code, not a call it makes, so nothing
  in the platform bounds a slow command; the bound is the mod's. A failure that the platform logs as a skipped hook is
  shown to the user in the session, so the slow command is caught in the mod and not left to fail it.
- **Alternatives:** the hook's own limit of 10 seconds (rejected: it does not count the call, and it is more than the
  8 seconds); a limit inside the command (rejected: a command that hangs cannot end itself).
- **Decided by:** Author (default; owner may override).

## 793b47 A dry run reports the mod as a mod

- **Choice:** the report of [`engram update`](../../GLOSSARY.md#engram-update), with or without `--dry-run`, lists a mod
  in the section of Claude Code with `agent-instructions/mods/<name>/` as its source, where a [skill](../../GLOSSARY.md#skill) is listed with
  `agent-instructions/skills/<name>/`. The dry run plans and reports the mod as it does the skills, and writes none of
  it. The plan marks the files of a mod, and the report data marks each directory it counts, as a skill or a mod.
- **Rationale:** a preview exists to tell the user what an update would [turn](./793a_product-spec.md#793a1a-terms) on
  ([793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios)). A mod listed under the source of
  a skill names a directory it does not come from: the first run of that scenario showed the row
  `agent-instructions/skills/engram-memory/`. The mark costs one field on the plan and one on the report data.
- **Alternatives:** leave the row as it was (rejected: it names the wrong source); a section of the report for mods
  (rejected: it changes the shape of the report for every reader, for no scenario).
- **Decided by:** Author (default; owner may override).

## 793b48 The conformance check reads this change's specs and fails when they are gone

- **Choice:** the check that every scenario ID of the product spec and the technical spec has a test, a function
  declared with the name `Test<ID>_...` in any `_test.go` file whatever its build tags, is a test of the dev integration
  suite, so `targ check-full` runs it. It reads the declarations of the test files with the Go parser, not their text: a
  comment, a string literal, a method, a function literal and a file that does not parse declare no test. It reads the
  specs from this change's directory and fails, naming the cause, when they are
  not there; at promotion, when the directory is deleted, it moves to read the promoted specs or is deleted with the
  change. It checks one direction: a test whose name carries no scenario ID, or one that names an ID no spec holds, is
  not an error.
- **Rationale:** the owner's rule ([793b9 Test execution](#793b9-test-execution)) asks for the check, and running it
  with the other checks keeps it from being forgotten. Skipping when the specs are missing would let promotion drop the
  guard unnoticed, and a missing input is a failure, not a pass. One direction is what the rule says: helper tests and
  the tests of other changes carry no scenario ID of this one. The first version searched the text of the files for
  `func Test<ID>_`. The review of the first group reproduced that it counted the declarations that the check's own
  fixtures hold in string literals, and stayed green when the real `Test793a2a1a_` and `Test793d4b1h_` were renamed;
  reading the declarations closes that. The allow list of the dev lint configuration (`dev/golangci-lint.toml`) gains
  `go/ast`, `go/parser` and `go/token` for non-test files, the three packages it already allowed for tests, because the
  check's code is not a test file; no rule is switched off.
- **Alternatives:** a target of its own outside `targ check-full` (rejected: it could be forgotten); skipping when the
  directory is missing (rejected: it passes with nothing checked); both directions (rejected: other changes' tests carry
  other IDs); the text search of the first version (rejected: it counts text that declares nothing).
- **Decided by:** Author (default; owner may override). The reading of declarations is the review's finding, accepted by
  the author.

## 793b49 The acceptance run is capped at four tests at once

- **Choice:** `targ acceptance` runs at most 4 of its tests at the same time (`go test -parallel=4`). The tests stay
  parallel, each on a machine of its own; the default is the number of cores, 10 on the computer this was built on.
- **Rationale:** the tests start real processes (Claude Code [sessions](../../GLOSSARY.md#session), a build of the binary, copies of it), and two kinds
  of them are bounded by time: the mod ends its command at 7 seconds, and a [prompt](./793a_product-spec.md#793a1a-terms) may be held for 8
  ([793b46 The command is ended at 7 seconds](#793b46-the-command-is-ended-at-7-seconds)). Measured on the full set of 14
  tests, 10 at once: two tests failed in each of two runs (the stand-in `engram` that the mod started did nothing for the
  7 seconds, so the command it was given was never recorded); 8 at once: one test failed in each of two runs; 6 at once:
  none in two runs; 4 at once: none in three runs; 3 at once: none in one. The failures were different tests in
  different runs and every test passes alone and in small groups, so the cause is the load of the others; the exact
  step that stalls is not established. A cap of 4 leaves a margin below 6 and costs about 40 seconds for the full set.
  The review of the first group read the failing runs again and found more than test load: `$.process.run` logged the
  spawn at once, and then `prompt.submit settled in 7060.0ms` with no output from the child, so the stall is between the
  spawn and the child's first instruction, and a user on a loaded machine would have the prompt held about 7 seconds with
  nothing delivered or shown. That is an open risk, not an artifact of the tests: [793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?](./793c_open-questions.md#793c32-is-the-stall-between-the-spawn-of-the-command-and-its-first-instruction-a-risk-to-the-product).
- **Alternatives:** one test at a time (rejected: about four minutes for no gain over 4); no `t.Parallel()` on the timed
  tests (rejected: the tests share no state, so the cause is not shared state); a longer bound in the mod (rejected: the
  7 seconds are the product's, [793b46 The command is ended at 7 seconds](#793b46-the-command-is-ended-at-7-seconds)).
- **Decided by:** Author (default; owner may override).

## 793b50 The mod uses what the command printed only when it succeeded and printed a block

- **Choice:** the mod adds the output of `engram auto deliver` beside the [prompt](./793a_product-spec.md#793a1a-terms) only when the command exited with 0 and
  the output begins with `[engram memory]`, the line that opens every block
  ([793d4c Framing the notes of a block as data](./793d_technical-spec.md#793d4c-framing-the-notes-of-a-block-as-data)).
  In every other case the prompt goes on as it was, with nothing added
  ([793d2a Prompt delivery flow](./793d_technical-spec.md#793d2a-prompt-delivery-flow)).
- **Rationale:** the review of the first group reproduced that the first version added the standard output whatever the
  exit code. The `engram` of main, which has no `auto` command, prints `Unknown command: auto` and 2,117 bytes of usage to
  standard output and exits 1, and this branch's binary prints `Error: flag provided but not defined: --bogus` to standard
  output; both reached the [agent](./793a_product-spec.md#793a1a-terms)'s context as if they were [notes](../../GLOSSARY.md#permanent-note). `$.process.run` returns the exit code and the output
  without throwing on a non-zero exit, so the mod reads the exit code itself. The marker keeps out text that a command
  prints on exit 0 and that is not a block.
- **Alternatives:** the exit code alone (rejected: an `engram` that exits 0 with other text would still reach the
  context); the marker alone (rejected: a failing command that happens to print a block would still be used); any
  non-empty output, the first version (rejected: reproduced).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b51 The delivery finds the records where Claude Code writes them

- **Choice:** `engram auto deliver` reads the records of [sessions](../../GLOSSARY.md#session) from the directory its `--sessions` flag names, else
  from `ENGRAM_TRANSCRIPT_DIR`, else from `$CLAUDE_CONFIG_DIR/projects`, else from `<home>/.claude/projects`
  ([793d4f Finding the records of sessions](./793d_technical-spec.md#793d4f-finding-the-records-of-sessions)).
  [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) is not changed: it keeps `ENGRAM_TRANSCRIPT_DIR`, else `<home>/.claude/projects`
  ([792b17c6 Session-log directory](../../specs/technical/ingestion-source-discovery.md#792b17c6-session-log-directory)).
- **Rationale:** a session started with `CLAUDE_CONFIG_DIR` set writes its record under `$CLAUDE_CONFIG_DIR/projects`, and
  not under `<home>/.claude/projects`: a headless `claude -p` session of Claude Code 2.1.302 in an isolated home, with
  `CLAUDE_CONFIG_DIR` set to a directory other than `<home>/.claude` and no `ENGRAM_TRANSCRIPT_DIR`, left its record at
  `$CLAUDE_CONFIG_DIR/projects/<project>/<session ID>.jsonl` and nowhere under the home. The first version read only
  `<home>/.claude/projects`, so for such a user, whose [skills](../../GLOSSARY.md#skill) directory links to the one [`engram update`](../../GLOSSARY.md#engram-update) fills, no record
  was ever found and every held [note](../../GLOSSARY.md#permanent-note) was delivered again with each [prompt](./793a_product-spec.md#793a1a-terms) (review finding 13). `ENGRAM_TRANSCRIPT_DIR`
  stays first because it is engram's own override and what every isolated run sets. The sweep is left as it is because
  its order is a canonical requirement of `engram ingest`, and the standing rule is to default to today's behaviour.
- **Alternatives:** change the one function that both commands use (rejected: it changes what `engram ingest` sweeps for
  a user with `CLAUDE_CONFIG_DIR` set, which the owner did not ask for; asked in
  [793c26 Does the delivery read the records where Claude Code writes them, and does the sweep follow?](./793c_open-questions.md#793c26-does-the-delivery-read-the-records-where-claude-code-writes-them-and-does-the-sweep-follow)).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b52 The delivery runs its lookup through an injected function

- **Choice:** `AutoDeliverDeps.Query` is a function that runs the lookup of `engram query` for the arguments it is
  given and prints its payload, wired at the composition root over the query dependencies, where it was the query
  dependencies themselves. The delivery builds the arguments, the [prompt](./793a_product-spec.md#793a1a-terms)'s text as the message and as the only phrase and
  the [vault](../../GLOSSARY.md#vault) it was given ([793d4a Delivering the notes that bear on a prompt](./793d_technical-spec.md#793d4a-delivering-the-notes-that-bear-on-a-prompt)),
  and the tests that need the real lookup hand it a function over the in-memory vault.
- **Rationale:** the completion gate of [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates)
  asks `imptest` evidence of the call and arguments of every injected collaboration. The lookup's arguments could be seen
  only through the effects of the real engine, and the message, which the engine matches literally against runbook
  triggers, has no test that plants a runbook, so a delivery that left the message out passed every test. A function
  is the smallest seam that shows the arguments. The review of the first group asked for the evidence or a waiver
  (finding 16).
- **Alternatives:** a waiver of the gate for the lookup (rejected: a change of the arguments survives every test); a
  test that plants a runbook and reads the order of the payload (rejected: it tests the query engine, which has its
  own tests, and not the arguments the delivery builds).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b53 A missing login or Claude Code fails an acceptance test

- **Choice:** the tests of `dev/acceptance` fail, naming the cause, when `claude` is not on the path or no login is found
  ([793e2 Test methods](./793e_implementation-plan.md#793e2-test-methods)); they no longer skip. They run only through
  `targ acceptance`, or an explicit `go test` with the `network` tag, so a machine that cannot run them is one on which
  somebody asked for them.
- **Rationale:** the review of the first group found that `targ acceptance` exits 0 having run nothing: with no login both
  live tests of a run reported SKIP and the package reported `ok` (reproduced here with `-run 'Test793a2b1a|Test793a2b1b'`
  in an isolated home with no credential: two SKIP, exit 0). A run that proves nothing must not look like one that
  passed, for the reason [793b48 The conformance check reads this change's specs and fails when they are gone](#793b48-the-conformance-check-reads-this-changes-specs-and-fails-when-they-are-gone)
  makes a missing input a failure.
- **Alternatives:** keep the skip and make the target fail on any SKIP (rejected: it reads the test output for one case,
  where a failing test says the same in the form every other failure takes).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b54 Two acceptance scenarios prove more than that nothing arrived

- **Choice:** [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios)
  has the [vault](../../GLOSSARY.md#vault) gain a second [note](../../GLOSSARY.md#permanent-note) about the staging database after the [agent](./793a_product-spec.md#793a1a-terms)'s lookup, so the lookup could not have
  printed it, and its outcome is that the [prompt](./793a_product-spec.md#793a1a-terms) arrives with that note and not the staging note. The test of
  [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios) asserts that the mod
  ran its command for the prompt, with the marker that the test of [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](./793a_product-spec.md#793a2d1-uat-scenarios) already uses; the scenario's text does not
  change.
- **Rationale:** the review of the first group found that both tests asserted only that nothing arrived, so both also pass
  when the mod never loads. With one note, "nothing arrives" is the outcome of the right delivery and also of a mod that
  never ran; with a second note that bears on the prompt, the right outcome is the second note alone, which only a mod
  that runs, and leaves out what the [session](../../GLOSSARY.md#session) holds, produces. The second note comes after the agent's lookup because a
  note that existed then is named in the lookup's payload even when the limit cuts it from the items: the clusters of the
  payload list every note of the matched set, so a vault of two related notes cannot hold one the lookup left out. The requirement
  ([793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)) already says
  the notes delivered are those the lookup returns, less those the agent has, so the product promise does not change.
- **Alternatives:** leave the scenarios and add a separate scenario for a second note (rejected: the plan guide has one test
  per scenario ID and the held rule is what [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios) is for); a marker in the product scenario's text for [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios)
  (rejected: that the mod ran is not something the user checks).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b55 The mod's JavaScript is parsed by a test of the dev integration suite

- **Choice:** `targ check-full` parses the JavaScript of every file under `agent-instructions/mods/` with `node`, which
  reads the file on standard input as an ES module and checks its syntax, through a test of the dev integration suite, as
  the conformance check runs ([793b48 The conformance check reads this change's specs and fails when they are gone](#793b48-the-conformance-check-reads-this-changes-specs-and-fails-when-they-are-gone)).
  The test fails, naming node, when node is not found; it does not skip.
- **Rationale:** the review of the first group found that no default gate parsed `register.js`, so a syntax error in the mod
  would pass `targ check-full` and be found by the first live [session](../../GLOSSARY.md#session), where Claude Code skips a hook that does not load. A
  test of the suite that already runs adds no target to the aggregates that `Test794b4a1c` keeps equal to Targ's. A check
  that cannot run must not pass ([793b53 A missing login or Claude Code fails an acceptance test](#793b53-a-missing-login-or-claude-code-fails-an-acceptance-test)),
  so a machine without node fails the gate: node is the runtime the mod is written for, and a machine that builds and
  checks the mod is one that has it. The file is read on standard input as a module so that the result does not depend
  on how a node release detects the syntax of a `.js` file.
- **Alternatives:** a new target in `check-full` (rejected: it changes the aggregates that 794b4a1c pins to Targ's); a
  skip when node is not found (rejected: the gate would pass unchecked); a JavaScript parser in Go (rejected: a new
  dependency for what node does).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b56 The 7 seconds bound the whole hook, not the command alone

- **Choice:** the mod races everything it does for a [prompt](./793a_product-spec.md#793a1a-terms), the call for the [session](../../GLOSSARY.md#session)'s ID and the command, against one
  timer of 7 seconds, and the prompt goes on with nothing added when the timer wins
  ([793d2a Prompt delivery flow](./793d_technical-spec.md#793d2a-prompt-delivery-flow)). The command is still ended at 7
  seconds by the time limit of the process call ([793b46 The command is ended at 7 seconds](#793b46-the-command-is-ended-at-7-seconds)),
  counted from its start: a call for the ID that took part of the bound leaves a command that outlives the prompt's wait by
  that part, and the prompt does not wait for it.
- **Rationale:** the review of the first group found that `await $.session.id()` ran outside the bound, so a call to the
  host that did not answer would hold the prompt for as long as the host took, and the 8 seconds of
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) promise no longer than that for any reason.
- **Alternatives:** keep the bound on the command (rejected: reproduced by a host that never gives the ID); a second timer
  for the ID (rejected: the two would add up to more than the 7 seconds).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b57 The delivery looks up the local vault only, until a later node

- **Choice:** the delivery of the first group looks up the local [vault](../../GLOSSARY.md#vault) only, with
  `RunQuery`, where `engram query` also merges the parent vault (`ENGRAM_PARENT`). The merge is not built here. It is a
  named node of the plan, [793e6f The delivery merges the parent vault](./793e_implementation-plan.md#793e6f-the-delivery-merges-the-parent-vault),
  after the nodes that exist, and the owner is asked in
  [793c34 Does the delivery merge the parent vault, as `engram query` does?](./793c_open-questions.md#793c34-does-the-delivery-merge-the-parent-vault-as-engram-query-does).
- **Rationale:** the review of the first group (finding 4) found that `lookUpNotes` runs the lookup on the local vault
  while `engram query` merges the parent's [notes](../../GLOSSARY.md#permanent-note), so [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt),
  which delivers the notes `engram query` returns, is met in part for a vault that has a parent. The review asked that the
  deferral be recorded and the merge not be built.
- **Alternatives:** build the merge now (rejected: the review asked not to); leave it unrecorded (rejected: a requirement
  that the delivery under-meets would have no node that meets it).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b58 The first group delivers for every origin of a prompt

- **Choice:** until [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent),
  the mod delivers for every [prompt](./793a_product-spec.md#793a1a-terms) that a [session](../../GLOSSARY.md#session) raises, whatever its origin. [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) takes the clause of
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) that [automatic memory](./793a_product-spec.md#793a1a-terms) MUST NOT deliver to or
  write from its own sessions, with its scenario and the filter by the origin of a prompt, because it is the first group
  that starts sessions of its own; [793e6c Never in the way](./793e_implementation-plan.md#793e6c-never-in-the-way) no
  longer takes it.
- **Rationale:** the review of the first group (findings 5 and 6) found that row 2 of
  [793b36 Earlier owner answers this minimal set reverses](./793b_decision-log.md#793b36-earlier-owner-answers-this-minimal-set-reverses)
  says "prompts the user sends" while the mod delivers on every `prompt.submit`, and that the own-session clause was
  planned in [793e6c Never in the way](./793e_implementation-plan.md#793e6c-never-in-the-way), after [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) starts automatic memory's own sessions, which would deliver to themselves for a node.
  Before [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) there are no sessions of its own, so a mod that delivers for every origin delivers to the user's.
- **Alternatives:** filter by origin now (rejected: there is no prompt of another origin to tell apart yet); leave the
  clause in [793e6c Never in the way](./793e_implementation-plan.md#793e6c-never-in-the-way) (rejected: [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) would start the sessions first).
- **Decided by:** Author (default; owner may override), on the review's finding.

## 793b59 The builder ran the tests once with the real home

- **Choice:** the gate of [793e7 Completion gates](./793e_implementation-plan.md#793e7-completion-gates) that no test,
  check or run touched the owner's real `~/.claude`, `~/.local/share/engram`, `~/.pi` or GOBIN is qualified by this
  record, and not dropped. The builder ran `targ test-for-fail` once with the real home, in the build window 2026-10-17
  00:10:49 to 03:41:40 EDT. The review compared modification times and found that nothing under `~/.claude`, `~/.pi`, the
  GOBIN or the top level of the [vault](../../GLOSSARY.md#vault) changed in the window; the only vault files in the window are [notes](../../GLOSSARY.md#permanent-note) 1205 to 1208
  and their sidecars, which the parent [session](../../GLOSSARY.md#session) wrote; and the chunks in the window all map to the owner's own [transcripts](../../GLOSSARY.md#transcript).
- **Rationale:** a gate stated without qualification must not stand over a run that is known to have broken it, whatever
  the run left behind ([793f7 Test isolation](./793f_algorithm.md#793f7-test-isolation)).
- **Alternatives:** leave the gate as it was (rejected: it was not true of the build).
- **Decided by:** Reviewer (the review's check of modification times), recorded by the author.

## 793b60 The restated sentences of 792a2e are kept as they are

- **Choice:** [793a5a 792a2e First-use vault creation](./793a_product-spec.md#793a5a-modified-792a2e-first-use-vault-creation)
  restates the canonical requirement with one clause added. Its sentence about the files of a new [vault](../../GLOSSARY.md#vault), which names
  `.obsidian/app.json`, a `.gitignore`, a starter `README.md` and the `.engram/` state directory, is the canonical
  requirement's own text, kept verbatim; only the clause about the commands that [automatic memory](./793a_product-spec.md#793a1a-terms) runs on its own is new.
- **Rationale:** the product spec guide keeps file names out of a product specification, and the docs check flags them
  by pattern, but these words are not new: rewording a restatement would change the canonical requirement under a Modified
  operation, which the lifecycle guide keeps for the change a requirement is meant to get.
- **Alternatives:** reword the sentence without file names (rejected: it changes what a new vault contains, which is not
  this change's to change).
- **Decided by:** Author (default; owner may override), on the docs check.

## 793b61 Review of the first group

- **Choice:** the first group ([793e6a On after update, and memory arrives with the prompt](./793e_implementation-plan.md#793e6a-on-after-update-and-memory-arrives-with-the-prompt))
  had one review round, run by opus on commit `e62be9a4`, which found 3 major, 16 minor and 7 nit items, numbered 1 to 26.
  The author accepted every finding but 25 and applied them in a fix pass of new commits on the branch; no existing commit
  was amended, rebased or rewritten. Where each is applied:
  - **Major.** 1, the mod used the output of a command whatever its exit code:
    [793b50 The mod uses what the command printed only when it succeeded and printed a block](#793b50-the-mod-uses-what-the-command-printed-only-when-it-succeeded-and-printed-a-block).
    2 and 22, a [note](../../GLOSSARY.md#permanent-note) counted as held when its file name was in any tool result, and the second [prompt](./793a_product-spec.md#793a1a-terms) of the product
    scenario ran in a resumed process: [793b45 What a session holds](#793b45-what-a-session-holds) and the product scenario
    of the second prompt, now run through one live process. 3, the conformance check counted text:
    [793b48 The conformance check reads this change's specs and fails when they are gone](#793b48-the-conformance-check-reads-this-changes-specs-and-fails-when-they-are-gone).
  - **Minor.** 4, the parent [vault](../../GLOSSARY.md#vault): [793b57 The delivery looks up the local vault only, until a later node](#793b57-the-delivery-looks-up-the-local-vault-only-until-a-later-node).
    5 and 6, delivery for every origin and the own-[session](../../GLOSSARY.md#session) clause:
    [793b58 The first group delivers for every origin of a prompt](#793b58-the-first-group-delivers-for-every-origin-of-a-prompt).
    7, a skip that ran nothing: [793b53 A missing login or Claude Code fails an acceptance test](#793b53-a-missing-login-or-claude-code-fails-an-acceptance-test).
    8, two scenarios that proved only silence: [793b54 Two acceptance scenarios prove more than that nothing arrived](#793b54-two-acceptance-scenarios-prove-more-than-that-nothing-arrived).
    9, the mutation checks of two tests that passed when written, recorded in their catalog rows and not by amending their
    commits: a delivery that prints the block when the lookup fails fails the first, and a delivery of every note of the
    vault fails the second, which a delivery of every item of the payload does not. 10, tests that resolved a directory under
    the real home: they name a directory of their own and stub the environment, and no guard test is added, because other
    tests rely on the real `Getenv` by design and a guard would need an opt-out in each. 11, the one run with the real home:
    [793b59 The builder ran the tests once with the real home](#793b59-the-builder-ran-the-tests-once-with-the-real-home).
    12, the stall: [793b49 The acceptance run is capped at four tests at once](#793b49-the-acceptance-run-is-capped-at-four-tests-at-once)
    and [793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?](./793c_open-questions.md#793c32-is-the-stall-between-the-spawn-of-the-command-and-its-first-instruction-a-risk-to-the-product).
    13, `CLAUDE_CONFIG_DIR`: [793b51 The delivery finds the records where Claude Code writes them](#793b51-the-delivery-finds-the-records-where-claude-code-writes-them).
    14, no gate parsed the mod: [793b55 The mod's JavaScript is parsed by a test of the dev integration suite](#793b55-the-mods-javascript-is-parsed-by-a-test-of-the-dev-integration-suite).
    15, the wording of [793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt),
    which names the notes that bear on a prompt by a term the user can check. 16, the `imptest` evidence:
    [793b52 The delivery runs its lookup through an injected function](#793b52-the-delivery-runs-its-lookup-through-an-injected-function).
    17, three stale sentences of the technical spec, corrected. 18, the dashes of the Green column, filled. 19, the entries
    of the open questions for the decisions of the first group, added, with the two non-blocking owner questions of the review
    ([793c19 Is the body of a note delivered whole, and on one line?](./793c_open-questions.md#793c19-is-the-body-of-a-note-delivered-whole-and-on-one-line)
    and [793c33 Does the promise of automatic memory name a minimum version of Claude Code?](./793c_open-questions.md#793c33-does-the-promise-of-automatic-memory-name-a-minimum-version-of-claude-code)).
  - **Nit.** 20, format characters in a note: [793b43 A note cannot reshape the block](#793b43-a-note-cannot-reshape-the-block).
    21, the call for the session's ID outside the bound: [793b56 The 7 seconds bound the whole hook, not the command alone](#793b56-the-7-seconds-bound-the-whole-hook-not-the-command-alone).
    23, the shared checkout hard-linked the worktree's files, now copies. 24, the fields and the parameter nothing read, removed.
    26, a test that took its lock in the source directory: found by bisecting the 1,660 tests of the package, and given a
    temporary vault. 25 is declined: two red commits carry unrelated [harness](../../GLOSSARY.md#harness) edits, and the history stays as it is, because
    commits are not to be rewritten.
  - **Docs checks.** The heading of the conformance section has its ID, the bare IDs and the Modified labels are fixed, the
    first use of every defined term in a node is linked, and the restated sentences of 792a2e stay verbatim:
    [793b60 The restated sentences of 792a2e are kept as they are](#793b60-the-restated-sentences-of-792a2e-are-kept-as-they-are).
  - **Found on the way.** `Test792b17c6a1` read the machine's environment and failed under the isolation of
    [793f7 Test isolation](./793f_algorithm.md#793f7-test-isolation), which sets `ENGRAM_TRANSCRIPT_DIR`; it stubs the
    environment now.
- **Rationale:** [Decisions and Open Questions](../../AGENTS.md#decisions-and-open-questions) has each review round record
  its outcome in the decision log, and its questions in the open-questions file, before anything goes to the owner; the
  author's choices of the fix pass are the entries that come before this one, from the mod's use of the command's output
  to the restated sentences, each with its entry in the open questions.
- **Alternatives:** decline a finding. One was declined (25).
- **Decided by:** Reviewer (opus review of the first group), accepted by the author, except 25, declined by the author.

## 793b62 The scenarios of the second group

- **Choice:** step 2 for the second group ([793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent))
  writes fourteen scenarios, one for each clause that has none and no more: seven under
  [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)
  (the five kinds of confirmed [lesson](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) each become a [note](../../GLOSSARY.md#permanent-note):
  a correction, a request to remember, an approach the user confirmed, the [agent](./793a_product-spec.md#793a1a-terms)'s own correction and an
  approach that succeeded with evidence; a [turn](./793a_product-spec.md#793a1a-terms) of only guesses, plans and chatter produces no note; a credential
  of the turn is not in the note), two under [793a2a On after `engram update`](./793a_product-spec.md#793a2a-on-after-engram-update) (a second update still
  writes each lesson as one note; no [vault](../../GLOSSARY.md#vault) is created by a write, the clause that [793a5a 792a2e First-use vault creation](./793a_product-spec.md#793a5a-modified-792a2e-first-use-vault-creation)
  added), one under [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) (the clause about its own [sessions](../../GLOSSARY.md#session)), one under
  [793a3a What is written is delivered later](./793a_product-spec.md#793a3a-what-is-written-is-delivered-later) (the note arrives with later [prompts](./793a_product-spec.md#793a1a-terms) of the same
  [session](../../GLOSSARY.md#session) and of a new one), two under [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again) (a note saved earlier by
  someone else, and a note the agent saved itself in the turn) and one under
  [793a3e Explicit memory use keeps working](./793a_product-spec.md#793a3e-explicit-memory-use-keeps-working).
- **Rationale:** the owner's step 2 asks for the minimum scenarios that prove the requirement, and each MUST and
  MUST NOT clause needs one that fails when the clause is broken. The five kinds are the clause's own list, so each is
  a scenario; the two clauses that forbid (guesses, a credential) are each a scenario because a forbidding clause is
  proved by a turn that tempts the breach and a note that is still written beside it. The two scenarios of
  [793a3c Nothing the vault already says is written again](./793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again) are the two writers a note can come from ("whoever wrote that note"): a note that was
  there before the [turn](./793a_product-spec.md#793a1a-terms), and one the agent writes during it. The steps that stage a lesson are the test's: a session in which
  the agent may not run engram commands is how a test proves that a note was written "by [automatic memory](./793a_product-spec.md#793a1a-terms) and not by
  the agent", since the agent's own learn [skill](../../GLOSSARY.md#skill) would otherwise be free to write the note first.
- **Alternatives:** one scenario with an outline of the five kinds (rejected: the plan guide has one test per scenario ID,
  and each kind needs its own staging of a turn); a scenario for each origin of a prompt (rejected: no clause of a
  requirement names origins, see [793b64 The prompts the user sends](#793b64-the-prompts-the-user-sends)).
- **Decided by:** Author (default; owner may override).

## 793b63 A turn is judged by one model call, and no session is started

- **Choice:** after a [turn](./793a_product-spec.md#793a1a-terms) of a [prompt](./793a_product-spec.md#793a1a-terms) the user sent, the memory mod runs `engram auto judge`, which prints the
  request of one model call, makes that call through the mod's model API (`$.model.complete`: one prompt string, the
  model `haiku`, effort `low`, so the user's own login pays and no key or account is configured), and runs
  `engram auto write` with the reply, which parses the verdict, redacts credentials, leaves out a lesson the
  [vault](../../GLOSSARY.md#vault) already states and writes the rest through the learn path, marked as written automatically. The three steps run
  after the turn, one turn after another, each bounded in time, and none holds the next [prompt](./793a_product-spec.md#793a1a-terms) or the reply.
  [Automatic memory](./793a_product-spec.md#793a1a-terms) starts no [session](../../GLOSSARY.md#session) of its own: the call raises neither a prompt nor a turn, so there is
  nothing to deliver to and nothing to write from, and the clause of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) about its own sessions holds by
  construction. [793a2d1b Automatic memory's work starts no session of its own](./793a_product-spec.md#793a2d1-uat-scenarios) observes it, and no marker or guard for a session of
  its own is built, because nothing would start such a session.
- **Rationale:** verified with Claude Code 2.1.302 in a headless session in an isolated home: the `turn.complete` event
  carries the final text of the turn (`answer`), the turn's ID and the reason it ended, and the session's record already
  holds the turn's last row when the event fires; `$.model.complete` answers `{isAnswered, text}` in about half a second to
  two seconds for `haiku`, from the hook that called it or from work left running after the hook returned; no
  prompt or turn is raised by it. [793b58 The first group delivers for every origin of a prompt](#793b58-the-first-group-delivers-for-every-origin-of-a-prompt)
  and the plan expected this group to start sessions of its own, on the assumption that the judgement runs in one; the
  model API shows it need not. The prompt is that of the earlier attempt, which the owner's earlier work tuned, with the
  turn marked off from the context before it and the nearest [notes](../../GLOSSARY.md#permanent-note) named by file name.
- **Alternatives:** a headless `claude -p` session of its own started with `$.process.run`, with an environment marker
  that makes the mod step aside in it (rejected: a second process that loads the user's whole configuration for each
  turn, and the clause becomes breakable); `$.model.fork` over the user's own [transcript](../../GLOSSARY.md#transcript) (rejected: it needs the main
  thread's cache and sends the whole conversation, where the judgement needs the turn).
- **Decided by:** Author (default; owner may override). It departs from the plan's expectation recorded in
  [793b58 The first group delivers for every origin of a prompt](#793b58-the-first-group-delivers-for-every-origin-of-a-prompt); the owner is asked in
  [793c38 Does automatic memory start sessions of its own?](./793c_open-questions.md#793c38-does-automatic-memory-start-sessions-of-its-own).

## 793b64 The prompts the user sends

- **Choice:** a [prompt](./793a_product-spec.md#793a1a-terms) is the user's when its origin, as Claude Code stamps it on the `prompt.submit` event, is `composer` (typed
  at the terminal), `bridge` (the phone or web client), `slack-ping` (the owner pinging from Slack) or `sdk` (the
  headless host, where `claude -p` is how the user sends a prompt). Every other origin is not: a background task's
  notification, a scheduled trigger, a peer or coordinating [session](../../GLOSSARY.md#session), another plugin's prompt, a prompt of no origin the engine
  can name. The mod gives the origin to `engram auto deliver` and `engram auto judge`, and the commands decide, so the
  definition is in Go: `deliver` prints nothing for a prompt that is not the user's, and `judge` prints nothing for its
  [turn](./793a_product-spec.md#793a1a-terms). The mod leaves out the turns of a [subagent](../../GLOSSARY.md#subagent) itself (the event names the [agent](./793a_product-spec.md#793a1a-terms)), as [793b8 Exclusions](#793b8-exclusions) decided.
- **Rationale:** [793b22 The scope is top-level sessions and notes only](#793b22-the-scope-is-top-level-sessions-and-notes-only) and row 2 of
  [793b36 Earlier owner answers this minimal set reverses](#793b36-earlier-owner-answers-this-minimal-set-reverses) say prompts the user sends, and
  [793b58 The first group delivers for every origin of a prompt](#793b58-the-first-group-delivers-for-every-origin-of-a-prompt) moved the filter here. The event's origin is the one source:
  the session's record spells the origin of a task notification with an underscore where the event uses a hyphen, and
  the record of a prompt does not exist yet when the prompt is delivered. `sdk` counts because a headless session's
  prompts are sent by the user's own script, and every acceptance test drives one. No clause of a requirement names
  origins, so no product scenario does; the technical spec proves the filter.
- **Alternatives:** the origin read from the record at capture time (rejected: another vocabulary, and no record
  at delivery); `composer` alone (rejected: it would leave out the user's own headless and phone prompts).
- **Decided by:** Author (default; owner may override).

## 793b65 What the judgement reads, and what it is asked

- **Choice:** the request for the model holds the instructions, then a window of the conversation: up to 6 lines
  before the [turn](./793a_product-spec.md#793a1a-terms), the line `--- the turn ---`, and the lines of the turn. A line is one message, labelled by the
  system with `User:`, `Agent:` or `Agent used` and the tool's name; the inputs and results of tools are not shown, so a
  tool the [agent](./793a_product-spec.md#793a1a-terms) ran is one line; a break inside a message is written ` ⏎ ` so that no message makes a line of its own.
  A line is cut to 1,200 characters and the window to 8,000, the oldest context first and then the middle of the turn,
  and the final text is added when the [session](../../GLOSSARY.md#session)'s record lacks it. Every text is redacted before it is cut ([793b69 Credentials](#793b69-credentials-are-redacted-by-shape-and-the-rest-is-a-known-risk)).
  The instructions are those of the earlier attempt, with a sentence that an approach the agent was unsure of, tried
  and saw succeed is a lesson. The reply is a JSON list of lessons, each with a kind, a slug, a situation, the file name
  of a [note](../../GLOSSARY.md#permanent-note) that already states it, and the fields of its kind ([793b66 What a lesson becomes](#793b66-what-a-lesson-becomes)).
- **Rationale:** a lesson comes from the user's words and from the agent's claims and what overturned them, and the
  inputs and results of tools are where a credential or a file's bulk is, so the window leaves them out. The sentence
  was added because a probe of the judgement alone, over a turn in which the agent tried a flag and the check passed,
  wrote the lesson in 11 of 20 runs without it and in 20 of 20 with it, and the live scenario
  [793a2c1e An approach that succeeded with evidence becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios) then passed 10 of 10 runs.
- **Alternatives:** the whole [transcript](../../GLOSSARY.md#transcript) of the session (rejected: its size and its credentials, and the call is paid by the
  user's login); the inputs and results of tools (rejected for the same reason); the instructions of the earlier attempt
  unchanged (rejected by the probe above).
- **Decided by:** Author (default; owner may override); the owner is asked in
  [793c40 Is the window of the judgement the right one?](./793c_open-questions.md#793c40-is-the-window-of-the-judgement-the-right-one).

## 793b66 What a lesson becomes

- **Choice:** a lesson becomes a feedback or a fact [note](../../GLOSSARY.md#permanent-note) through the learn path of
  [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), at the top of the order of the [vault](../../GLOSSARY.md#vault), with the source
  `written automatically (session <the session's ID>)`, so that the note embeds on write as any note does and a reader
  of the vault can tell it from one the [agent](./793a_product-spec.md#793a1a-terms) saved. A reply that is not a list, and a lesson that lacks a field or has a kind it does not
  know, write nothing and are recorded with the reason. The other kinds of note, a link or a replacement of a note, are
  not written.
- **Rationale:** the learn path is the one writer of notes, so a note written automatically cannot differ in form from
  one the agent writes, and the mark is the only difference the scenarios rely on.
- **Alternatives:** a store of its own for the lessons (rejected: the delivery and the agent's lookup read the vault);
  runbooks and replacements of notes (rejected: [793b8 Exclusions](#793b8-exclusions) leaves curation out).
- **Decided by:** Author (default; owner may override).

## 793b67 What automatic memory records

- **Choice:** the writing makes one record for each thing that became of a reply, a [turn](./793a_product-spec.md#793a1a-terms) or a failure: `written` with the file
  name of the [note](../../GLOSSARY.md#permanent-note), `skipped` with the reason, `nothing` for a reply with no lesson, and `failed` with the failure. A record is
  a JSON file of its own in the directory `auto` of the engram data directory, below the directory named for the
  [session](../../GLOSSARY.md#session), and named for the time it was made, its place among the records of the reply and its kind; the flag `--records` names the directory.
  A record that cannot be written does not fail the command.
- **Rationale:** a test of an absence has to know that the judgement finished, and the display of
  [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) will read the same files. One file for each effect needs no lock and no append, and the records are
  made last, so that their existence means the write has ended.
- **Alternatives:** one log file (rejected: concurrent appends and a line cut by a crash); the records inside the
  [vault](../../GLOSSARY.md#vault) (rejected: the vault is the user's notes, and the vault may not exist, see [793b71 First-use step](#793b71-the-commands-of-the-group-auto-take-no-first-use-step)).
- **Decided by:** Author (default; owner may override).

## 793b68 A lesson an earlier note states

- **Choice:** the request shows the model the [notes](../../GLOSSARY.md#permanent-note) nearest to the [turn](./793a_product-spec.md#793a1a-terms): the lookup of
  `engram query` for the first 500 characters of the text of the user's [prompts](./793a_product-spec.md#793a1a-terms) of the turn, each note as one line of
  the form of the delivery, redacted and cut to 400 characters. The model may name, as `covered`, the file name of a note
  that already states a lesson, and the writing leaves such a lesson out when the name is a plain file name of a file
  that is in the [vault](../../GLOSSARY.md#vault), recording it as skipped; a name of another file, or a path, covers nothing.
- **Rationale:** whether a note states the same lesson is a judgement of meaning, which the model makes and a
  similarity threshold cannot; the lookup only offers the candidates. Checking that the named file exists keeps a
  name the model made up from hiding a lesson. The note the [agent](./793a_product-spec.md#793a1a-terms) saves during the turn is in the vault when the turn ends, so it is among the
  candidates, and that is how a lesson the agent saved itself is not written again.
- **Alternatives:** a threshold on the distance between the lesson and the notes (rejected: no value separates "the
  same lesson" from "a related one"); the whole vault in the request (rejected: its size).
- **Decided by:** Author (default; owner may override).

## 793b69 Credentials are redacted by shape, and the rest is a known risk

- **Choice:** the judgement redacts every text of the [turn](./793a_product-spec.md#793a1a-terms) before it cuts or joins it and before the model sees it, and the writing
  redacts every field of a lesson before it writes it. A text is a credential when it is a private key block; a JSON web
  token; a key or token that opens with the name of its issuer; a `Bearer` or `Basic` credential; the user and the
  password of a URL; the value of an `Authorization` header; the value given to a name that holds `key`, `token`,
  `secret`, `password` and the like; or a run of 32 characters or more of letters and digits that is not the 40
  hexadecimal digits of a commit. It is replaced by `[redacted]`, and the name of an assignment is kept. A secret of
  another shape, such as a password in a sentence or a token split over two lines, is not redacted, and the wrapped and
  percent-encoded patterns of the earlier attempt are not ported.
- **Rationale:** the instruction not to write a credential is not a proof: a live model never wrote one in 10 runs of
  the scenario [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios) even with that sentence removed, so only properties over the redaction prove it,
  and they hold for the shapes above.
- **Alternatives:** a library of secret patterns (rejected: a dependency, and this change adds none); redacting the
  [note](../../GLOSSARY.md#permanent-note) after the model wrote it only (rejected: the credential would have reached the model and the call is logged).
- **Decided by:** Author (default; owner may override); the residual risk is asked in
  [793c41 Is the risk of a secret of another shape acceptable?](./793c_open-questions.md#793c41-is-the-risk-of-a-secret-of-another-shape-acceptable).

## 793b70 The capture reports its own failures, and nothing times it as a whole

- **Choice:** after a [turn](./793a_product-spec.md#793a1a-terms) the mod gives the event's result back at once and runs the three steps one after the other apart
  from the event: the judgement, the model's call and the writing. A step that fails, because a command exits with other
  than 0 or prints what is not a request, or a call is ended or rejected, or the model gives no text, is reported to
  the writing as a `failure` of up to 300 characters that names the step, and the writing records it as failed. The model's
  call is bounded by the 60 seconds of the request, the commands by the host's own limit, and no timer bounds the
  chain as a whole.
- **Rationale:** [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) is kept by stepping aside, and a record of the failure is what lets the user see
  that a write did not happen. In 147 [notes](../../GLOSSARY.md#permanent-note) written by the live runs of this change, the note was in the [vault](../../GLOSSARY.md#vault) between
  0.9 and 3.9 seconds after the turn ended (median 1.9, 90th percentile 2.8), so the 2 minutes of
  [793b18 Writing finishes within 2 minutes of a turn](#793b18-writing-finishes-within-2-minutes-of-a-turn) is a bound under load and not the usual time.
- **Alternatives:** a timer of 2 minutes over the chain (rejected: it would end a write that is about to finish, and
  nothing in the [session](../../GLOSSARY.md#session) waits for it); a retry of a failed step (rejected: a second model call, and the next turn
  captures its own lessons).
- **Decided by:** Author (default; owner may override); the bound is asked in
  [793c42 Should the bound of 2 minutes be tightened?](./793c_open-questions.md#793c42-should-the-bound-of-2-minutes-be-tightened).

## 793b71 The commands of the group auto take no first-use step

- **Choice:** `engram auto judge` and `engram auto write`, like `engram auto deliver`, run against the path of the
  [vault](../../GLOSSARY.md#vault) as it is and create nothing when it holds no vault ([793d4e Modified: First-use gating](./793d_technical-spec.md#793d4e-modified-first-use-gating)). The learn path fails
  at the lock of a vault that is not there, so a [lesson](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent) is not written, the writing records the failure, and the [turn](./793a_product-spec.md#793a1a-terms) has
  cost a call of the model.
- **Rationale:** the plan expected a change here: the learn path creating the vault. A first live run of the
  scenario [793a2a1g A lesson is not written to a vault that does not exist](./793a_product-spec.md#793a2a1-uat-scenarios) showed that it does not, so the node adds a test and no code.
- **Alternatives:** a clean skip before the call of the model when the vault is missing (rejected: more code for a state the
  user made on purpose, and no scenario asks for it); the first-use step for the two commands (rejected: it creates
  the vault the user removed).
- **Decided by:** Author (default; owner may override); the cost is asked in
  [793c43 Should a missing vault end the capture before the model is asked?](./793c_open-questions.md#793c43-should-a-missing-vault-end-the-capture-before-the-model-is-asked).

## 793b72 The origin of a turn

- **Choice:** the mod keeps the origin of the last [prompt](./793a_product-spec.md#793a1a-terms) its [session](../../GLOSSARY.md#session) raised, binds it to the next [turn](./793a_product-spec.md#793a1a-terms) of the main loop when
  that turn starts, by the turn's ID, and gives it to the judgement when the turn completes; a turn that a
  [subagent](../../GLOSSARY.md#subagent) ran is not captured. Where the mod learned no origin, it gives none, and the commands do nothing.
- **Rationale:** [793b64 The prompts the user sends](#793b64-the-prompts-the-user-sends) puts the definition in Go and leaves the mod to hand over what it received; the end of a turn
  names only the turn, and the prompt names the origin, so the turn's start is where the two meet. Turns of the main
  loop do not overlap, but one that is typed over can end after the next has started, so the origin is kept by ID.
- **Alternatives:** the origin read from the session's record when the turn ends (rejected in
  [793b64 The prompts the user sends](#793b64-the-prompts-the-user-sends)); one variable for the origin of the running turn (rejected: a turn typed over would read the next
  turn's origin).
- **Decided by:** Author (default; owner may override).

## 793b73 The tests of an absence wait for the end of the write, and for a settling time

- **Choice:** a test that a [note](../../GLOSSARY.md#permanent-note) was not written waits until the records of the [turn](./793a_product-spec.md#793a1a-terms) exist, which are made last, and for 5 seconds
  after them, before it counts the [notes](../../GLOSSARY.md#permanent-note); a test that a note is delivered waits for the records too, because a lookup in a
  [vault](../../GLOSSARY.md#vault) whose only note is being written finds no embedding and the delivery steps aside.
- **Rationale:** the first run of [793a3a1a A written note is delivered with later prompts](./793a_product-spec.md#793a3a1-uat-scenarios) sent its next [prompt](./793a_product-spec.md#793a1a-terms) as soon as the note's file existed and
  got no note with it; the failure of the lookup, `vault has notes but no current-model embeddings`, lasted the 0.3
  seconds the write took to embed the note. An absence proved at the instant of the first record would pass a write
  that is about to put a second note beside the first, and the writes of one turn end within a second of each other.
- **Alternatives:** wait the whole 2 minutes (rejected: every absence would cost 2 minutes); count the notes at the
  first record (rejected above).
- **Decided by:** Author (default; owner may override); the window is asked in
  [793c44 Is the window between a note's file and its embedding a risk?](./793c_open-questions.md#793c44-is-the-window-between-a-notes-file-and-its-embedding-a-risk).
