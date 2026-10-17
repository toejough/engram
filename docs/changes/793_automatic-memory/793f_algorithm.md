# Automatic Memory - Algorithm

How change #793 is produced; change context, deleted at promotion. The owner's words are quoted exactly (Joe, 2026-10-16, `793b1`).

## 793f1 North Star

> I want to add automatic learning and memory retrieval.

Everything in this change serves it (`793b2`).

## 793f2 The algorithm

> 1: break it down into the minimal set of user visible and user validatable capabilities the system must have, and how they must interact to achieve the goal. These are the product requirements.
> 2: for each product requirement, identify the minimum user visible and user exercisable gherkin scenarios that will prove the requirement has been met.
> 3: for each acceptance test, identify the minimal set of components and how they must behave and interact to implement the functionality the test requires. Each component behavior and interaction sequence becomes a technical requirement.
> 4: for each technical requirement, what are the minimal gherkin tests that will prove the req has been met?
> 5: for each test, write it, make it pass, then refactor to minimize the code introduced by the branch without obfuscating its function and purpose.

Documentation is held to the same minimum-sufficient bar as code (owner, 2026-10-17).

## 793f3 Depth first

> Work depth first through the above algorithm.

Step 1 runs for the whole set, step 2 for one requirement at a time in the order of `793e`, steps 3 to 5 for one node (a scenario, or a tightly coupled pair) at a time, the next node starting when the previous is done (`793b38`). "Commit early and often": each document change and each red, green and refactor phase is its own commit.

## 793f4 Delegation

> You should only orchestrate. Use haiku to search, sonnet to work, and opus to review.

The orchestrator does no object-level work: haiku searches, sonnet writes and builds, opus reviews, one review round per phase, a phase being one step applied to one unit (`793b5`, `793b33`).

## 793f5 How this change applies it

| Steps | File | Holds |
| --- | --- | --- |
| 1, 2 | [793a](./793a_product-spec.md) | product requirements and Gherkin scenarios |
| 3, 4 | [793d](./793d_technical-spec.md) | technical requirements (a component behavior or an interaction sequence, each tracing to the product scenarios it serves) and Gherkin scenarios |
| 5 | [793e](./793e_implementation-plan.md) | the red, green and refactor commits, one row per test |

Each scenario is run by one Go test named with its ID. A product scenario's test drives a real Claude Code [session](../../GLOSSARY.md#session) in an isolated HOME (`793f7`), its steps mirroring the Given, When and Then lines one to one (`793b9`). A conformance check asserts that every scenario ID has a test.

## 793f6 Departures from the guides

Depth first and the North Star line are the owner's; the others are the author's readings of his words, which he may override.

- Depth first: the documents are partial until the last node is done (`793b10`).
- A North Star line opens the product spec, beyond the product guide's three sections (`793b2`).
- No interview or approval gates, from "work autonomously": the owner reviews at the end (`793b11`).
- No option analysis, from step 3: it takes the minimal components a scenario needs (`793b12`).
- One refactor pass per test replaces the plan guide's four checkpoints, and the catalog gains three commit columns (`793b13`).

## 793f7 Test isolation

Every literal run of a product scenario, and every test or check that runs `engram` or `claude`, uses its own HOME, CLAUDE_CONFIG_DIR, GOBIN, ENGRAM_VAULT_PATH, ENGRAM_CHUNKS_DIR and ENGRAM_TRANSCRIPT_DIR and a working directory outside any `.claude` or `.git` ancestor (`dev/eval/isolation.py` builds the engram and Claude variables and checks the directory; HOME and GOBIN are added here). Anyone running `engram` or `claude` for this change sets them on the command line.

"The user runs [`engram update`](../../GLOSSARY.md#engram-update)" in a scenario means this branch's binary, built into the run's GOBIN and run with its HOME. The real `~/.claude`, `~/.local/share/engram`, `~/.pi` and GOBIN are never touched. The owner's rule from the earlier attempt (`793b32`).
