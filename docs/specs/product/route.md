# Route - Product Spec

`route` is the skill an orchestrating agent follows when it is about to dispatch a subagent and must choose the agent type, the model and the effort. It makes the choice from evidence and not from a fixed table: every unit of work starts at the cheapest tier, and only recalled evidence, or a failed review, raises it. Every dispatch is recorded as a note, and the records become memory that the next routing decision recalls, so the effective rubric improves without anyone editing the skill. Route is skill-level guidance for the orchestrator. It adds no system boundary and no interaction of its own: it is a sibling of please, applied over the existing relationship between the harness and engram. The shipped procedure is `agent-instructions/skills/route/SKILL.md`, with the rates in `price-table.md`. The completion-report contract it hands to every subagent is lessons-contract's ([lessons-contract](./lessons-contract.md)).

Rationale: route encodes the delegate-everything doctrine. Two measured results shaped it. Sonnet with memory fully matched opus with memory across applying conventions (15 of 15), recency supersession (3 of 3) and abduction (6 of 6), while sonnet cold failed the same axes, so memory lets a cheaper model do work that needs a stronger one without it (vault note 135). That was measured at the deep-to-mid boundary only; other boundaries are inferred, and the retry rule is the safety net for a wrong discount (`dev/eval/LEDGER.md#tier-routing-parity`). The discount is the largest whole-task cost lever found, larger than any payload-byte-level cut. Second, the earlier fixed table (mechanical to cheap, moderate to mid, complex to deep) was asserted and never measured. A baseline showed an agent reading it over-provisioned 5 of 6 generic units to mid or deep by surface look, and a control with no guidance did the same, because the table encoded the model's untested "this looks hard, so use a strong model" instinct. A first draft that only said "don't upgrade on a looks-hard hunch" still routed perceived difficulty to the deep tier; adding an explicit "genuinely hard is not evidence" test closed the loophole and flipped 5 of 6 over-provisioned units to 0 of 6. Cold-start cost rises for genuinely hard work, which can fail twice before it reaches the tier it needs; that is an accepted bootstrap, recovered by the evidence loop.

## 792a17a The orchestrator routes and delegates

Requirement: [792a17a The orchestrator routes and delegates](#792a17a-the-orchestrator-routes-and-delegates) The orchestrator following route MUST do orchestration itself (routing and decomposition decisions, dispatching subagents, sequencing steps, updating the task list, running the meta-skills that are the workflow such as recall, learn and planning, and synthesizing returned results into the next decision or the report) and MUST delegate every piece of object-level work (writing code or prose, running tests or builds, judging the artifact, reviewing it) to a subagent. It MUST NOT do easy work inline: easy work goes to a cheap model and is not skipped.

### 792a17a1 UAT Scenarios

```gherkin
Scenario: 792a17a1a Object-level work is dispatched
  Given a unit that writes code and runs its tests
  When the orchestrator follows route
  Then it dispatches the unit to a subagent and does not do it inline

Scenario: 792a17a1b Orchestration stays with the orchestrator
  Given results returned from three subagents
  When the next step must be decided
  Then the orchestrator synthesizes them itself and does not delegate the act of delegating

Scenario: 792a17a1c Easy work is delegated cheaply
  Given a one-line rename
  When the orchestrator routes it
  Then it dispatches it to the cheapest tier instead of doing it inline
```

## 792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier

Requirement: [792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier](#792a17b-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier) Before dispatching, the orchestrator MUST check recalled memory for tier-performance evidence on this kind of work, using the plain recall of the agent (the aggregates surface as ordinary memories) and no special query or counting on the read path. Absent recalled evidence, it MUST start every unit at the cheapest, fastest available tier, with no exception for work that merely looks hard (hard debugging, a cross-cutting refactor, a correctness-critical review, a greenfield design). The starting tier MUST NOT be raised for any reason other than recalled evidence of failure at a cheaper tier, or a cold-start prior that records such evidence ([792a17f Cold-start priors carry only evidence-backed escalations](#792a17f-cold-start-priors-carry-only-evidence-backed-escalations)), and then only to the tier the evidence supports. Predicted difficulty ("genuinely complex", "needs careful reasoning", "broad impact") MUST NOT count as evidence.

Rationale: a sense that a unit needs a strong model is a prediction, and the loop replaces the prediction with observation: a unit is found to need a higher tier by watching the cheap tier fail, never by forecasting. On a cold start a race condition, an eight-file refactor and a new API design all start cheap, like a variable rename.

### 792a17b1 UAT Scenarios

```gherkin
Scenario: 792a17b1a No evidence, cheapest tier
  Given a unit of work with no recalled evidence on similar work-kinds
  When the orchestrator routes it
  Then it selects the cheapest tier and records the choice as a cold-start default

Scenario: 792a17b1b Hard-looking work starts cheap
  Given a cross-cutting refactor, a new API design and a race-condition investigation
  When the orchestrator routes each with no recalled evidence
  Then each starts at the cheapest tier

Scenario: 792a17b1c Recalled failure raises the start
  Given recall surfaces evidence that this work-kind failed at the cheap tier before
  When the orchestrator routes it
  Then it starts at the tier the evidence supports and no higher

Scenario: 792a17b1d A hunch is not evidence
  Given the orchestrator reasons that a unit is correctness-critical and so needs the deep tier
  When it chooses the tier
  Then it starts cheap, because that reasoning forecasts difficulty

Scenario: 792a17b1e Evidence is read by plain recall
  Given a route-evidence aggregate for the work-kind exists
  When the orchestrator recalls for the dispatch
  Then it reads the aggregate's tallies from the recalled item and runs no engram count to route
```

## 792a17c A failed unit retries the spec before the tier

Requirement: [792a17c A failed unit retries the spec before the tier](#792a17c-a-failed-unit-retries-the-spec-before-the-tier) When a dispatched unit fails review, the orchestrator MUST first rewrite the handoff (sharper files, acceptance checks and do-not-touch bounds) and retry at the same tier, and MUST escalate one tier only when the same unit fails a second time, repeating until it passes or reaches the deep tier. It MUST NOT escalate on the first failure.

Rationale: a first failure is usually a failure of the spec and not of the model, and a cheap-tier model matches an expensive one when the spec is exact. The builder never gets to guess twice off the same spec.

### 792a17c1 UAT Scenarios

```gherkin
Scenario: 792a17c1a The first failure rewrites the spec
  Given a unit dispatched at the cheap tier that fails review
  When the orchestrator retries
  Then it rewrites the handoff and retries at the cheap tier

Scenario: 792a17c1b The second failure escalates
  Given the same unit fails again after the rewrite
  When the orchestrator retries
  Then it retries one tier up

Scenario: 792a17c1c Escalation stops at deep
  Given a unit that fails at the deep tier after its spec was rewritten
  When the orchestrator retries
  Then it does not escalate beyond the deep tier
```

## 792a17d A memory-backed unit drops one tier

Requirement: [792a17d A memory-backed unit drops one tier](#792a17d-a-memory-backed-unit-drops-one-tier) A unit whose needed knowledge is recallable (a known convention, a prior decision, a crystallized diagnostic) MUST be routed one tier cheaper than it otherwise would be, floored at the cheap tier.

Rationale: the model applies recalled knowledge instead of deriving it.

### 792a17d1 UAT Scenarios

```gherkin
Scenario: 792a17d1a The discount applies
  Given a unit whose needed convention is already a recallable note, and evidence that selects the mid tier
  When the orchestrator routes it
  Then it starts at the cheap tier and records "memory-discount applied" as the reason

Scenario: 792a17d1b The discount stops at cheap
  Given a memory-backed unit that would start at the cheap tier
  When the orchestrator routes it
  Then it starts at the cheap tier
```

## 792a17e The cheap tier is the cheapest agent in the current environment

Requirement: [792a17e The cheap tier is the cheapest agent in the current environment](#792a17e-the-cheap-tier-is-the-cheapest-agent-in-the-current-environment) The orchestrator MUST resolve the cheap tier against the current environment: a free local model that is available (for example an LMStudio provider) MUST be tried first for cheap-tier work, and a paid model MUST enter the roster only above the free local options. When no free local model is available, the cheap tier MUST resolve to the cheapest paid model. The example roster (cheap haiku, mid sonnet, deep opus) is an example, not a prescription, and a model name MUST NOT be hard-coded as a rule.

Rationale: a roster table that named models read as a prescription, and a real cost incident followed: five or more paid dispatches while a free local model sat idle (#702).

### 792a17e1 UAT Scenarios

```gherkin
Scenario: 792a17e1a A free local model is preferred
  Given a free local model is registered and available, and the orchestrator is routing cheap-tier work
  When it dispatches
  Then it dispatches to the free local model and not to a paid model

Scenario: 792a17e1b No free local model, the cheapest paid one
  Given no free local model is available
  When the orchestrator resolves the cheap tier
  Then it resolves to the cheapest paid model on the roster

Scenario: 792a17e1c A paid dispatch with a free option available is a red flag
  Given the orchestrator is about to dispatch cheap work to a paid model while a free local model was available and suitable
  When it checks its red flags
  Then it resolves the roster against the environment first
```

## 792a17f Cold-start priors carry only evidence-backed escalations

Requirement: [792a17f Cold-start priors carry only evidence-backed escalations](#792a17f-cold-start-priors-carry-only-evidence-backed-escalations) The cold-start priors table MUST list the cheapest tier as the posture for everything by default, MUST list the one-tier discount of a memory-backed unit, and MUST carry a row for a work-kind that starts above cheap only when recorded dispatch evidence for it clears the escalation thresholds (a cheap-tier pass rate below 70% and a mid-tier pass rate above 85%, both past the five-dispatch evidence floor), citing the tally that justified it. Recalled evidence MUST override a prior, and the table MUST NOT be treated as a prescription.

Rationale: priors are hypotheses that the dispatch record confirms or overturns. When strong repeated evidence accrues for a work-kind, it is folded into the table at periodic consolidation, so a cold session starts warm; until then recall does the improving. Leaving `go-package-implementation` at the default cost every new session a doomed cheap-tier attempt and a spec-retry before recall taught it what the evidence already showed (#722).

### 792a17f1 UAT Scenarios

```gherkin
Scenario: 792a17f1a A work-kind clears the thresholds
  Given recorded evidence for work-kind/go-package-implementation of 8 of 12 passes at the cheap tier and 10 of 10 at the mid tier
  When the cold-start priors table is read
  Then it has a row for the work-kind starting at mid, citing that tally

Scenario: 792a17f1b A high cheap pass rate stays at the default
  Given a work-kind whose cheap-tier pass rate is at or above 90%
  When the cold-start priors table is read
  Then it has no escalated row for it

Scenario: 792a17f1c A recalled outcome overrides a prior
  Given a prior that starts a work-kind at mid and recalled evidence that cheap sufficed
  When the orchestrator routes the work-kind
  Then recalled evidence wins
```

## 792a17g Every dispatch hands over an exact spec

Requirement: [792a17g Every dispatch hands over an exact spec](#792a17g-every-dispatch-hands-over-an-exact-spec) The orchestrator MUST hand every dispatched subagent the exact files and paths to create or modify (not "the auth code" but a file and line range), the acceptance checks that mean done (a command with its expected output, a test that must pass, or a property that must hold), the explicit bounds that the unit must leave alone (files, interfaces and behaviors), the instruction to recall first ([792a17h Every subagent recalls first, and large units are decomposed](#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed)), and the completion-report contract of [792a16b Every dispatch is told to report lessons](./lessons-contract.md#792a16b-every-dispatch-is-told-to-report-lessons). A vague handoff MUST be fixed before the model is blamed.

Rationale: a cheap-tier model matches an expensive one when the spec is exact; when it fails, suspect the spec before the model.

### 792a17g1 UAT Scenarios

```gherkin
Scenario: 792a17g1a A complete handoff
  Given a unit to implement a function
  When the orchestrator writes the handoff
  Then it names the exact files and lines, the acceptance checks, the do-not-touch bounds, the recall-first instruction and the LESSONS line mandate

Scenario: 792a17g1b A vague handoff is fixed
  Given a handoff that says only "fix the auth code"
  When the orchestrator checks it
  Then it sharpens the paths, checks and bounds before dispatching
```

## 792a17h Every subagent recalls first, and large units are decomposed

Requirement: [792a17h Every subagent recalls first, and large units are decomposed](#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed) The orchestrator MUST instruct every dispatched subagent that its first action is recall, with phrases drawn from its unit, before it does the work, and this instruction MUST NOT be waived. A unit that spans multiple files or concerns or needs more than one clear deliverable MUST NOT be dispatched as it stands: the orchestrator MUST break it into smaller units and route each, and the decomposition MUST be done by the orchestrator.

### 792a17h1 UAT Scenarios

```gherkin
Scenario: 792a17h1a Recall first is in every handoff
  Given a subagent that already has a detailed prompt
  When the orchestrator dispatches it
  Then the handoff still tells it to recall first

Scenario: 792a17h1b A large unit is decomposed
  Given a unit that touches six packages and has two deliverables
  When the orchestrator routes it
  Then it splits it into smaller units and routes each, and does not dispatch it as one agent
```

## 792a17i Each dispatch is recorded in a mini-report row

Requirement: [792a17i Each dispatch is recorded in a mini-report row](#792a17i-each-dispatch-is-recorded-in-a-mini-report-row) After each dispatch resolves, the orchestrator MUST record one evidence row and collect the rows into a compact table in its report. The row MUST carry the work-kind (the orchestrator's classification of the unit, kebab-case, reusing a prior kind before minting a new one), the tier used, the concrete model the tier resolved to at dispatch time, the reason for the tier choice ("recalled evidence (kind K passed at tier T)", "memory-discount applied" or "cold-start default"), the outcome, the tier the unit finally passed at if it escalated, its duration and its cost. The outcome MUST be the review or gate verdict (pass or fail) and MUST NOT be the subagent's own report of success. The record MUST be built from what the orchestrator already knows plus the review's verdict, with no privileged telemetry.

Rationale: the record is the rubric: without it there is no evidence, and the rubric never improves. A subagent's self-report confabulates (vault notes 148 and 162). Deriving the record from what the orchestrator knows keeps the loop harness-agnostic.

### 792a17i1 UAT Scenarios

```gherkin
Scenario: 792a17i1a A passing dispatch is recorded
  Given a dispatch whose review passes
  When the orchestrator records it
  Then the row has the work-kind, tier, model at dispatch, reason, outcome pass and the cost and duration

Scenario: 792a17i1b The outcome is the verdict
  Given a subagent that reports "done" and a reviewer that fails its work
  When the orchestrator records the outcome
  Then the outcome is fail

Scenario: 792a17i1c An escalated dispatch records where it passed
  Given a unit that failed at cheap and passed at mid
  When the orchestrator records it
  Then the escalation field names mid
```

## 792a17j Cost and duration are real or stated as unavailable

Requirement: [792a17j Cost and duration are real or stated as unavailable](#792a17j-cost-and-duration-are-real-or-stated-as-unavailable) The orchestrator MUST take a dispatch's duration and cost from the harness's usage signal when it exposes one (on Claude Code, the `duration_ms` and `subagent_tokens` of the subagent's Task-completion `<usage>` block), MUST record the unit explicitly, and MUST convert tokens to a dollar estimate only when the maintained price table (`price-table.md`, keyed by the concrete model) has an exact-match row for the dispatch's model, showing for example `45,231 tok (~$0.68 @ opus)`; priced by split input, output and cache columns when the usage block splits them, and otherwise as a blended estimate at the input rate labelled as such. When the model has no row, or its platform (Bedrock, Vertex or Foundry) prices it differently, it MUST show the raw unit-labeled token count with its reason (for example `45,231 tok (no rate on hand)`). On a harness with no per-subagent usage signal and no documented alternate source (Pi, as of the skill's writing), it MUST show `n/a` with the stated reason, and MUST NOT fabricate a cost, a duration or a dollar figure.

Rationale: tokens to dollars is a blended estimate from a maintained table, not a bill. A harness that exposes the signals under other names changes nothing in the record.
The rates live in a maintained table rather than a live pricing lookup, because a lookup adds an I/O
dependency and a failure mode; a stale row degrades to the raw token count, which is safer than a flaky lookup
silently defaulting to a wrong number, and updating a rate is a one-line data diff. Each non-Claude-Code
harness gets its own documented cost and duration source, or an explicit `n/a` with a reason, because
harnesses expose different signals and a forced common interface would lose information or invite fabrication.

### 792a17j1 UAT Scenarios

```gherkin
Scenario: 792a17j1a A rate is known
  Given a dispatch on a model with a row in the price table
  When the orchestrator renders the cost
  Then it shows a dollar estimate computed from subagent_tokens and that row

Scenario: 792a17j1b No rate is known
  Given a dispatch on a model that has no row in the price table
  When the orchestrator renders the cost
  Then it shows the unit-labeled token count and no dollar figure

Scenario: 792a17j1c A harness with no usage signal
  Given a dispatch on Pi, which exposes no per-subagent usage signal
  When the orchestrator renders cost and duration
  Then both show n/a with the reason, such as n/a — Pi exposes no per-subagent usage signal

Scenario: 792a17j1d A non-Anthropic platform
  Given a dispatch on a model priced by Bedrock
  When the orchestrator renders the cost
  Then it is treated as having no rate and the platform is noted

Scenario: 792a17j1e A bare number is never recorded
  Given a cost in tokens
  When the orchestrator records it
  Then the unit is written with the number
```

## 792a17k Each dispatch is also written to the vault as a tagged evidence note

Requirement: [792a17k Each dispatch is also written to the vault as a tagged evidence note](#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) After each dispatch resolves, with the review verdict in hand, the orchestrator MUST hand a fact to the write-memory skill as a field handoff and not as a command it composes, with the slug `route-dispatch-<work-kind>`, the tags `work-kind/<k>`, `tier/<cheap|mid|deep>` and `outcome/<pass|fail>`, the situation "routing <work-kind> work", the subject "<work-kind> dispatch at <tier> (<model @ dispatch>)", the predicate "resolved as", an object that states the verdict, the reason, the escalation, the duration and the cost with their units, and the source "route dispatch record, <project>, <date>". It MUST state the fields as written and MUST NOT invent a flag for them. The tags MUST be the three closed families only, work-kind an open kebab-case set, tier and outcome closed sets; duration and cost MUST stay in the object prose and never become tags. Producing only the table row MUST NOT count as recording.

Rationale: the table row is the mini-report; the durable evidence is the vault write. Categorical tags are the low-cardinality mechanism that recall, `engram count` and Obsidian can all read ([792a6c Categorical tags share one list](./controlled-vocabulary.md#792a6c-categorical-tags-share-one-list)).

### 792a17k1 UAT Scenarios

```gherkin
Scenario: 792a17k1a A passing dispatch gets a tagged note
  Given a dispatch with a passing verdict
  When the orchestrator records it
  Then write-memory receives a fact tagged work-kind/<k>, tier/<t> and outcome/pass with the dispatch details in the object

Scenario: 792a17k1b A failing dispatch is tagged fail with its escalation
  Given a dispatch that fails and escalates
  When the orchestrator records it
  Then the note is tagged outcome/fail and its object names the escalation tier

Scenario: 792a17k1c The tag format is validated
  Given a dispatch record
  When its tags are written
  Then each is work-kind/<k>, tier/<t> or outcome/<o> with only lowercase letters, digits and hyphens, and an invalid tag is rejected before the write

Scenario: 792a17k1d Fields, not flags
  Given the orchestrator hands off the evidence note
  When it states the handoff
  Then it states the fields as written and invents no flag such as --kind
```

## 792a17l Each work-kind keeps one aggregate note

Requirement: [792a17l Each work-kind keeps one aggregate note](#792a17l-each-work-kind-keeps-one-aggregate-note) After the evidence note, the orchestrator MUST amend or create the aggregate fact `route-evidence-<work-kind>`, composing and running that write itself and not handing it to write-memory, which has no amend form; this is a stated, structurally forced exception to the write-site doctrine and the skill MUST say so. It MUST look the aggregate up with `engram query --lazy-chunks --phrase="route evidence <work-kind> tier tally"` and take a returned path as a match only when its basename, without `.md` and split on `.`, ends in a slug equal to `route-evidence-<work-kind>` exactly. On a match it MUST recompute the tally from the aggregate's current text plus this dispatch and run `engram amend --target <basename> --object="..."`, keeping the existing evidence wikilinks and appending the new one. On no match it MUST first run a uniqueness check, a listing of vault basenames matching `route-evidence-<work-kind>*` under the same exact rule, and create the aggregate (`engram learn fact --slug route-evidence-<work-kind> --position top`, with a tier tally of 1/1 and a wikilink to the evidence note) only when the check finds nothing; when it finds more than one, it MUST merge them by uniting their evidence wikilinks, recomputing the tally with the count commands of [792a17m Counting audits the tallies and never routes](#792a17m-counting-audits-the-tallies-and-never-routes), amending one into the merged aggregate and recording the incident as drowning-remedy evidence. An aggregate MUST NOT carry the work-kind, tier or outcome tags. Its wikilink list MUST be the append-only evidence trail. When recording a dispatch before the lookup has run, the orchestrator MUST state both candidate commands and not guess.

Rationale: the aggregate is the unit the read path surfaces by plain recall. A missed lookup (an embedding miss, or the drowning case of [792a6c Categorical tags share one list](./controlled-vocabulary.md#792a6c-categorical-tags-share-one-list)) must not mint a duplicate, so a deterministic uniqueness check rules it out; aggregates are untagged and cannot be enumerated with `engram count`, which is why the check is a basename listing.

### 792a17l1 UAT Scenarios

```gherkin
Scenario: 792a17l1a The first dispatch creates the aggregate
  Given no aggregate exists for the work-kind and the uniqueness check finds nothing
  When the first dispatch is recorded
  Then the orchestrator creates route-evidence-<work-kind> with the tally from that dispatch, a link to its evidence note and no tags

Scenario: 792a17l1b A later dispatch amends it
  Given an aggregate for the work-kind
  When a subsequent dispatch is recorded
  Then the orchestrator amends the aggregate, recomputes the tier tally and appends the new evidence wikilink while keeping the old ones

Scenario: 792a17l1c A fuzzy match is not a match
  Given a lookup that returns a note whose slug only begins with route-evidence-<work-kind>
  When the orchestrator applies the match rule
  Then it does not count as the aggregate

Scenario: 792a17l1d A no-match result is checked before creating
  Given the lookup returns no match
  When the orchestrator would create an aggregate
  Then it first lists vault basenames matching route-evidence-<work-kind>* to rule out a missed lookup

Scenario: 792a17l1e Duplicates are merged
  Given the uniqueness check finds two aggregates for one work-kind
  When the orchestrator resolves it
  Then it unites their evidence wikilinks, recomputes the tally by count, amends one into the merged aggregate and records the incident

Scenario: 792a17l1f The write-path split is stated
  Given an orchestrator reading the aggregate procedure
  When it reaches the aggregate write
  Then the skill says the evidence note goes through write-memory while the aggregate is composed directly, because write-memory has no amend form
```

## 792a17m Counting audits the tallies and never routes

Requirement: [792a17m Counting audits the tallies and never routes](#792a17m-counting-audits-the-tallies-and-never-routes) The orchestrator MUST recompute aggregate tallies from the evidence notes' tags with `engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass` for the numerators and `engram count --group-by tags --filter tags=tier/<t>` for the denominators (reading the `work-kind/<k>` rows), or with `--filter tags=work-kind/<k> --filter tags=tier/<t>` and the `total:` line for one kind, when a tally is doubted and at periodic consolidation, and MUST amend an aggregate that disagrees to the recomputed numbers. It MUST NOT use count on the read path, MUST NOT group by `work-kind` (a tag value, not an attribute), and MUST NOT use `--group-by tier`, which runs without error but groups the note-tier frontmatter field (L1, L2, L3) and not the `tier/` tag family.

Rationale: tallies an LLM maintains will drift, so ground truth comes from tags and never from the aggregate's prose.

### 792a17m1 UAT Scenarios

```gherkin
Scenario: 792a17m1a A work-kind is audited
  Given an aggregate whose tally is doubted
  When the orchestrator audits tier routing parity for the work-kind
  Then it runs engram count --group-by tags with the work-kind and tier filters to recompute the numerators from the evidence tags

Scenario: 792a17m1b A drifted aggregate is repaired
  Given the count output disagrees with an aggregate's prose tally
  When the orchestrator repairs it
  Then it amends the aggregate's object to the recomputed numbers

Scenario: 792a17m1c The footgun is avoided
  Given the orchestrator needs the count by evidence tier
  When it composes the command
  Then it uses --group-by tags --filter tags=tier/<t> and not --group-by tier
```

## 792a17n Drowning is reported, not patched

Requirement: [792a17n Drowning is reported, not patched](#792a17n-drowning-is-reported-not-patched) At the same moments as the count audit, the orchestrator MUST run the aggregate lookup for a work-kind with many evidence notes and confirm that the aggregate's path still surfaces in the items or in any cluster's candidates; when it does not, it MUST report the pre-registered drowning case and MUST NOT patch the skill ad hoc.

Rationale: many near-identical evidence notes could outrank their aggregate on the read query. A scratch-vault gauge (20 evidence notes and one aggregate) passed, so no remedy was built, and the two remedies on record are in [792a6c Categorical tags share one list](./controlled-vocabulary.md#792a6c-categorical-tags-share-one-list).

### 792a17n1 UAT Scenarios

```gherkin
Scenario: 792a17n1a The aggregate still surfaces
  Given a work-kind with 20 evidence notes
  When the orchestrator runs the lookup
  Then the aggregate's path is among the results

Scenario: 792a17n1b A missing aggregate is reported
  Given a lookup in which a known aggregate does not surface
  When the orchestrator audits
  Then it reports the drowning case and does not patch the skill
```

## 792a17o The loop improves the rubric

Requirement: [792a17o The loop improves the rubric](#792a17o-the-loop-improves-the-rubric) When a finding about a work-kind is strong and general (for example, "cheap failed N times on work-kind K, so it needs mid"), the orchestrator MUST crystallize it with learn, as a confirmed approach when a tier's outcome confirms the routing (for example, "cheap sufficed for K") or as a reversal when it overturns a prior tier assumption. The next recall MUST surface that evidence so that the starting tier for similar work reflects it.

### 792a17o1 UAT Scenarios

```gherkin
Scenario: 792a17o1a A confirming tier outcome is a confirmed approach
  Given cheap sufficed repeatedly for a work-kind
  When the orchestrator crystallizes the finding
  Then it records a confirmed approach

Scenario: 792a17o1b An overturned tier assumption is a reversal
  Given evidence that overturns a prior tier assumption
  When the orchestrator crystallizes the finding
  Then it records a reversal
```
