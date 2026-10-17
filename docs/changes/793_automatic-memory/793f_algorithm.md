# Automatic Memory - Algorithm

How the change for issue #793 is produced. This is change context, deleted with the change directory at
promotion ([lifecycle guide](../../SPECIFICATION-LIFECYCLE-GUIDE.md#atomic-promotion)). The owner's words are
quoted exactly; Joe gave them on 2026-10-16 ([793b1 Start over](./793b_decision-log.md#793b1-start-over)).

## 793f1 North Star

> I want to add automatic learning and memory retrieval.

The one-line product statement and the headline feature of this change
([793b2 North Star](./793b_decision-log.md#793b2-north-star)). Every requirement, scenario, component and
test in this change exists to deliver it.

## 793f2 The algorithm

> 1: break it down into the minimal set of user visible and user validatable capabilities the system must have, and how they must interact to achieve the goal. These are the product requirements.
> 2: for each product requirement, identify the minimum user visible and user exercisable gherkin scenarios that will prove the requirement has been met.
> 3: for each acceptance test, identify the minimal set of components and how they must behave and interact to implement the functionality the test requires. Each component behavior and interaction sequence becomes a technical requirement.
> 4: for each technical requirement, what are the minimal gherkin tests that will prove the req has been met?
> 5: for each test, write it, make it pass, then refactor to minimize the code introduced by the branch without obfuscating its function and purpose.

Recorded as [793b3 The algorithm and the working rules](./793b_decision-log.md#793b3-the-algorithm-and-the-working-rules).

## 793f3 Depth first

> Work depth first through the above algorithm.

Step 1 is done for the whole set first. Then each requirement goes through steps 2 to 5, one requirement at a time, in the
order of [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan),
before the next requirement starts. The owner also said "Commit early and often." Every document change and
every red, green and refactor phase is its own commit, and the work continues autonomously until the author
judges it done.

## 793f4 Delegation

> You should only orchestrate. Use haiku to search, sonnet to work, and opus to review.

Reading ([793b5 Reading of the delegation rule](./793b_decision-log.md#793b5-reading-of-the-delegation-rule)):
the orchestrating [session](../../GLOSSARY.md#session) does no object-level work itself. Search and lookups go to haiku, all writing and
building go to sonnet, and every review goes to opus, one review round per phase. A phase is one step applied
to one unit: step 1 for the whole set, or step 2, 3, 4 or 5 for one requirement.

## 793f5 How this change applies it

| Steps | File | What it holds |
| --- | --- | --- |
| 1, 2 | [793a Automatic Memory - Product Spec](./793a_product-spec.md#automatic-memory---product-spec) | Step 1: the product requirements and how they interact. Step 2: the Gherkin scenarios under each requirement, and the canonical specs the requirement adds to or modifies. |
| 3, 4 | [793d Automatic Memory - Technical Spec](./793d_technical-spec.md#automatic-memory---technical-spec) | Step 3: the technical requirements, each a component behaviour or an interaction sequence, each tracing to the product scenarios it serves. Step 4: the Gherkin scenarios under each technical requirement. |
| 5 | [793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan) | The red, green and refactor record, one row per test: red commit, green commit, refactor commit. |

Test execution ([793b9 Test execution](./793b_decision-log.md#793b9-test-execution)): each scenario in the
product spec is executed literally against a real Claude Code [session](../../GLOSSARY.md#session), so the Gherkin in the spec is the
test source. Each scenario in the technical spec is executed by one Go test whose name carries the scenario
ID.

## 793f6 Departures from the guides

Each follows from the owner's algorithm and is his decision.

- **Depth first:** the documents are partial until the last requirement is done, where the guides expect each
  complete first ([793b10 Departure: depth-first fill](./793b_decision-log.md#793b10-departure-depth-first-fill)).
- **No interview or approval gates:** the work is autonomous and the owner reviews at the end
  ([793b11 Departure: no interview or approval gates](./793b_decision-log.md#793b11-departure-no-interview-or-approval-gates)).
- **No option analysis:** step 3 takes the minimal components a scenario needs
  ([793b12 Departure: no option analysis](./793b_decision-log.md#793b12-departure-no-option-analysis)).
- **One refactor pass:** one pass per test replaces the plan guide's four refactor checkpoints
  ([793b13 Departure: one refactor pass](./793b_decision-log.md#793b13-departure-one-refactor-pass)).
- **A North Star line:** the product spec opens with it, beyond the product guide's three sections
  ([793b14 Departure: the North Star line](./793b_decision-log.md#793b14-departure-the-north-star-line)).
