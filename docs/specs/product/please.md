# Please - Product Spec

`please` is the skill an agent follows when the user hands over a multi-step piece of work to be carried end-to-end. It runs a fixed seven-step workflow against the user's ask: an opening learn, orientation, a committed plan, execution under test-driven development, documentation, completion, and a closing learn. Four adversarial review gates, each staffed by fresh reviewers that try to refute the artifact, punctuate the workflow. It is meta-orchestration: it tracks the steps on the task list, uses whatever relevant skills the environment provides (it names only `recall` and `learn`, the two skills it relies on engram to ship, and every other step leans on capabilities and not names), and delegates object-level work and gate reviews to subagents routed by [route](./route.md). It has no subcommand of its own. The harness loads it as it loads recall and learn, matched by its own triggers (`/please`, `take this end-to-end`, `please`), and a session that received only its registered runbook note follows it through the runbook shim. The shipped procedure is `agent-instructions/skills/please/SKILL.md`. It is a single skill and a single runbook note: its former sub-runbooks were merged back into it. The completion-report contract and the collection of subagent lessons are lessons-contract's ([lessons-contract](./lessons-contract.md)), and the closing learn is learn's ([learn](./learn.md)).

Rationale: LLM-generated artifacts do not self-certify. Catching one's own plan's flaws is necessary and not sufficient, because the author's context carries the author's blind spots; the gates put a fresh reviewer per angle outside that context. The workflow is non-waivable because the cost of a step scales with the work, so a small ask makes a cheap workflow and never an exemption.

## 792a18a The skill fires on a handed-over multi-step ask

Requirement: [792a18a The skill fires on a handed-over multi-step ask](#792a18a-the-skill-fires-on-a-handed-over-multi-step-ask) Please MUST fire on the explicit form `/please <ask>` however trivial the ask looks, because the user chose the workflow deliberately, and on natural-language phrasings of the same intent ("please do X", "please work through X", "please take care of X", "work through X end-to-end", "take this end-to-end", "drive X to completion", "see this through"). It MUST NOT fire on a casual "please" attached to a single trivial action ("please read this file", "please rename this var"). For a natural-language ask, it MUST fire when the ask plausibly entails any of planning, more than one file, testing, documentation, a commit or closing an issue, MUST NOT fire when the ask plausibly resolves to a single file edit or tool call with no follow-up, and MUST fire when the boundary is genuinely unclear.

Rationale: the workflow tolerates small asks, and the reverse mistake is more expensive. The bare word "please" is a deliberately over-firing trigger of the runbook note; a trigger hit only makes the runbook a candidate, and the agent judges it against the skill's own conditions ([792a13m Trigger cues are specific, author-chosen strings](./write-memory.md#792a13m-trigger-cues-are-specific-author-chosen-strings)).
A runbook's `situation` is the only discriminator retrieval sees, so it is positive-only and process-shaped.
Embeddings do not encode negation, so a trailing never-for-a-casual-please clause attracted the excluded
phrases, and ceremony wording was never retrieved because agents phrase the query as the concrete work. A
situation carrying the eval task's own example would be test-fitting.

### 792a18a1 UAT Scenarios

```gherkin
Scenario: 792a18a1a The slash form always fires
  Given the user types /please rename the tally flag --out to --output
  When the agent reads the request
  Then it runs the seven-step workflow even though the ask looks trivial

Scenario: 792a18a1b A handed-over piece of work fires
  Given the user says "take this end-to-end" about a feature that needs a plan, tests and documentation
  When the agent reads the request
  Then it runs the workflow

Scenario: 792a18a1c A casual please does not
  Given the user says "please read this file"
  When the agent reads the request
  Then it handles the request directly and does not run the workflow

Scenario: 792a18a1d An unclear boundary fires
  Given a natural-language ask whose size the agent cannot judge
  When the agent decides
  Then it runs the workflow
```

## 792a18b An ask is required

Requirement: [792a18b An ask is required](#792a18b-an-ask-is-required) When `/please` is invoked bare, or the natural-language request has no concrete ask, please MUST ask exactly one targeted question ("What would you like me to work on?") through `AskUserQuestion` and wait, and MUST take no other action in the meantime: it MUST NOT create tasks, run recall or learn, read files, or open the transcript.

### 792a18b1 UAT Scenarios

```gherkin
Scenario: 792a18b1a A bare invocation gets one question
  Given the user types /please with nothing after it
  When the agent reads the request
  Then its whole turn is the single question "What would you like me to work on?"

Scenario: 792a18b1b Nothing happens meanwhile
  Given the single question is pending
  When the turn ends
  Then no task was created, no recall or learn ran, and no file was read
```

## 792a18c The seven steps are tracked, ordered and non-waivable

Requirement: [792a18c The seven steps are tracked, ordered and non-waivable](#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable) At the start of execution, please MUST push all seven steps to the task list in a single `TaskCreate` call, mark each step in progress when it begins and completed when it finishes, and start each step only after the previous step is completed. A step MUST be marked not applicable only when its mechanism does not exist in the environment (for example step 6's commit when the repository is not under version control, or step 7's closing learn when the engram binary is absent), with a one-line rationale naming the missing mechanism written into the task description. "Too small to bother", "nothing to document" and "no memories will be relevant" MUST NOT count. The user MUST NOT be able to waive steps by saying "no ceremony", "skip the plan" or "hurry". When the user interrupts and redirects, please MUST update the task list to the new shape and MUST NOT abandon open steps without an explicit close.

Rationale: a step that produces nothing costs little, and the closing learn records that it did. The workflow is fast when there is little to capture.

### 792a18c1 UAT Scenarios

```gherkin
Scenario: 792a18c1a Seven tasks in one call
  Given a valid ask
  When please begins execution
  Then the task list gets seven tasks from one TaskCreate call and each moves in progress and then completed in order

Scenario: 792a18c1b No step starts early
  Given step 1's learn is still in progress
  When the agent considers reading repository docs
  Then it waits until step 1 is completed

Scenario: 792a18c1c A missing mechanism is the only N/A
  Given a repository not under version control
  When the agent reaches step 6
  Then it marks the commit not applicable with the rationale "no VCS" written into the task description

Scenario: 792a18c1d Small is not N/A
  Given a change that touched no documentation
  When the agent reaches step 5
  Then it runs the step, which may produce nothing, and does not mark it not applicable

Scenario: 792a18c1e Hurry does not waive
  Given the user says "no ceremony, skip the plan"
  When the agent proceeds
  Then it runs the workflow, including the plan

Scenario: 792a18c1f A redirect updates the list
  Given the user redirects mid-workflow
  When the agent continues
  Then the task list reflects the new shape and no open step is dropped without an explicit close
```

## 792a18d The workflow opens with learn and orients with recall

Requirement: [792a18d The workflow opens with learn and orients with recall](#792a18d-the-workflow-opens-with-learn-and-orients-with-recall) Please MUST begin with step 1, the `learn` skill, to preserve anything pending from the session so far, before starting new work. Step 2 MUST begin with a literal invocation of recall, with queries derived from the ask, and MUST NOT substitute reading the working tree for it ("I already know this repo", "small ask", "I'll just grep" are the moments recall matters most): file reads surface working-tree content and recall surfaces vault content. Step 2 MUST loop until the ask is understood: it MUST read the repository's relevant markdown (the project instructions, the files under `docs/`, any glossary, architecture or design notes), MUST ask the user clarifying questions about intent or new concepts through `AskUserQuestion`, MUST NOT move on while material doubt remains about the ask's meaning or whether it should be done as stated, and MUST state the agent's own assessment of the ask (sound, flawed or underspecified).

### 792a18d1 UAT Scenarios

```gherkin
Scenario: 792a18d1a Learn first
  Given a valid ask
  When please starts
  Then the first thing it runs is the learn skill

Scenario: 792a18d1b Recall is literal
  Given a small ask whose diff is right there
  When the agent orients
  Then it invokes recall and also reads files, and does not read files instead of recalling

Scenario: 792a18d1c Orientation loops on doubt
  Given an ask whose meaning is unclear after recall and reading
  When the agent orients
  Then it asks the user and repeats the loop until understanding is solid
```

## 792a18e The ask is challenged once, directly, then committed to

Requirement: [792a18e The ask is challenged once, directly, then committed to](#792a18e-the-ask-is-challenged-once-directly-then-committed-to) Please MUST evaluate the ask on its merits during orientation, treating flaws, cheaper alternatives, contradictions with repository norms or recalled memory, and unnamed risks as findings to raise. It MUST challenge plainly and directly before planning around the problem, stating the issue and its concrete stakes in declarative sentences, and MUST NOT soften it into a leading question, bury it mid-plan or preface it with reflexive praise. A reply MUST NOT open with praise or agreement before analysis. It MUST challenge once, clearly; when the user weighs the challenge and reaffirms, it MUST proceed wholeheartedly and record the dissent in the plan ("considered X; user chose Y because Z") and MUST NOT relitigate it later or hedge passively in the work.

Rationale: a collaborator is not a yes-machine. Silent execution of a flawed ask is the core sycophancy failure.

### 792a18e1 UAT Scenarios

```gherkin
Scenario: 792a18e1a A flaw is stated with stakes
  Given an ask that stores credentials in plaintext
  When the agent orients
  Then it says plainly, before planning, that the design stores credentials in plaintext and that anyone with file access owns every account

Scenario: 792a18e1b The challenge is not softened
  Given a flaw in the ask
  When the agent raises it
  Then it is a declarative statement and not a leading question

Scenario: 792a18e1c The user's reaffirmation is recorded and honored
  Given the user reaffirms the ask after the challenge
  When the agent plans
  Then the plan records "considered X; user chose Y because Z" and the agent does not raise the challenge again
```

## 792a18f A challenge does not displace the asked task

Requirement: [792a18f A challenge does not displace the asked task](#792a18f-a-challenge-does-not-displace-the-asked-task) Once the user gives the ask, or a gate approves the plan, please MUST treat the asked task as settled and begin it. It MUST NOT substitute adjacent, deferred or "more rigorous" prerequisite work (a better test, "the real blocker") for what was asked, and MUST NOT treat a dependency-order or "we can't fully verify yet" argument as new evidence. It MUST NOT deviate except by naming a genuinely new fact and stating that it is reversing direction.

Rationale: challenging the ask means stress-testing it in place. Recommending a prerequisite first feels like diligence but relitigates the settled task. The same rule binds recall's recommendations ([792a14n A recommendation is reconciled against the asked task](./recall.md#792a14n-a-recommendation-is-reconciled-against-the-asked-task)).

### 792a18f1 UAT Scenarios

```gherkin
Scenario: 792a18f1a The asked task begins
  Given an approved plan for task T and a more rigorous prerequisite P that the agent could build first
  When the agent starts execution
  Then it begins T

Scenario: 792a18f1b A reversal needs a new fact
  Given the agent wants to do P before T
  When it explains
  Then it names a genuinely new fact and says it is reversing direction
```

## 792a18g The plan is written, committed and gated

Requirement: [792a18g The plan is written, committed and gated](#792a18g-the-plan-is-written-committed-and-gated) Step 3 MUST write a plan that accomplishes the ask, using a plan-writing skill when one is installed and writing it directly otherwise. When the user already supplied a plan, please MUST NOT skip the step: it MUST capture the plan as the planning artifact, review it, fill gaps and write it down. When the repository is under version control, it MUST commit the plan, and the plan MUST NOT count as approved until gate A closes. It MUST NOT write code before the plan is committed.

### 792a18g1 UAT Scenarios

```gherkin
Scenario: 792a18g1a A user-supplied plan is still captured
  Given the user supplied a plan with the ask
  When the agent reaches step 3
  Then it reviews it, fills gaps and writes it down as the planning artifact

Scenario: 792a18g1b The plan is committed before code
  Given a repository under version control
  When step 3 completes
  Then the plan is committed and gate A has closed before any code is written
```

## 792a18h A change to a repeated invariant gets an enumeration grep

Requirement: [792a18h A change to a repeated invariant gets an enumeration grep](#792a18h-a-change-to-a-repeated-invariant-gets-an-enumeration-grep) When a planned change alters a repeated invariant (a payload shape, a sweep cadence, a command set, a count or a naming convention echoed across docs, diagrams or skills), please MUST run, before writing the documentation-scrub task, a search of the repository for the term itself, its synonyms, its hyphenated or compound forms and the old text's echoes in diagram labels and comments, and MUST paste the resulting per-file disposition list into the plan. Each entry MUST name the file, a disposition (keep, update, rewrite or N/A) and a one-line reason. The grep MUST run however large the surface looks or however small the change appears.

Rationale: the failure it targets is salience under load, not a capability gap (issue #685). In a clean probe where documentation scrub was the salient task, the model enumerated the whole surface without the rule, 6 of 6 times; the failure showed up only when the scrub was an incidental consequence of a larger feature ask (`dev/eval/LEDGER.md#685-doc-enumeration-grep`, harness `dev/eval/cumulative/please_step3_probe/`). The rule's cost scales with the surface, so a small surface is a cheap grep and never an exemption.

### 792a18h1 UAT Scenarios

```gherkin
Scenario: 792a18h1a A repeated invariant changes
  Given a plan that renames a field echoed across multiple files
  When the author finishes the plan
  Then it contains a per-file disposition list from a grep for the term, its synonyms, its hyphenated forms and the old text's echoes

Scenario: 792a18h1b A small surface still greps
  Given a surface that looks small enough to skip
  When the author plans
  Then the grep still runs

Scenario: 792a18h1c The list names file, disposition and reason
  Given the grep results
  When the author compiles the list
  Then each line names a path, a disposition and a reason, such as "docs/ROADMAP.md:3 → update: cites old FEATURES, retarget to openspec/specs/"

Scenario: 792a18h1d A minimal change to a widely referenced concept
  Given a plan that touches a concept present in many files
  When the author plans
  Then the full grep runs and the cost does not exempt the change
```

## 792a18i Execution follows test-driven development

Requirement: [792a18i Execution follows test-driven development](#792a18i-execution-follows-test-driven-development) Step 4 MUST take each unit through red, green and refactor, using a test-driven-development skill when one is installed and applying the discipline directly otherwise. Red MUST validate the challenge with a repeatable test (or the closest analogue for non-code work), green MUST make it pass with the minimal change, and refactor MUST keep the result DRY, SRP-respecting and YAGNI-compliant so that updates fit as if written as part of the whole. Each refactor MUST pass gate B before the unit is declared done. Please MUST drive the plan from step 3 with a plan-execution skill when one is installed and by working it task by task otherwise, and MUST verify before declaring any unit done, by a verification skill when installed and otherwise by running the actual commands and reading their output.

### 792a18i1 UAT Scenarios

```gherkin
Scenario: 792a18i1a RED is not skipped as too simple
  Given a unit that looks too simple to test
  When the agent executes it
  Then it still writes a repeatable test first

Scenario: 792a18i1b A unit is verified by running the commands
  Given a unit the agent believes is done
  When it declares it done
  Then it has run the real commands and read their output

Scenario: 792a18i1c Each refactor is gated
  Given a refactor phase has finished
  When the unit is to be declared done
  Then gate B has closed over the refactored diff
```

## 792a18j The documentation matches the new reality

Requirement: [792a18j The documentation matches the new reality](#792a18j-the-documentation-matches-the-new-reality) Step 5 MUST update every piece of documentation the change touches (`README.md`, `CLAUDE.md`, `docs/`, glossaries, skill references) so that the documentation matches the new reality, and the step MUST complete only when gate C closes over every touched document.

### 792a18j1 UAT Scenarios

```gherkin
Scenario: 792a18j1a Touched docs are updated and gated
  Given a change that alters a documented command
  When step 5 runs
  Then every doc that names the command is updated and gate C has closed over each one
```

## 792a18k Completion closes the issue, cleans up and commits

Requirement: [792a18k Completion closes the issue, cleans up and commits](#792a18k-completion-closes-the-issue-cleans-up-and-commits) Step 6 MUST close the originating issue when the work came from one, MUST delete any planning or temporary build and test artifacts created along the way, and when the repository is under version control MUST stage and commit the changes, through a commit-focused skill when one is installed and directly otherwise. Commit messages and any outward prose MUST pass gate D before the commit or close.

### 792a18k1 UAT Scenarios

```gherkin
Scenario: 792a18k1a Completion cleans up and commits
  Given work that originated from an issue
  When step 6 runs
  Then the issue is closed, temporary artifacts are deleted, the changes are committed, and the commit messages passed gate D first
```

## 792a18l Four gates, each with a fresh reviewer per angle

Requirement: [792a18l Four gates, each with a fresh reviewer per angle](#792a18l-four-gates-each-with-a-fresh-reviewer-per-angle) Please MUST run four adversarial review gates. Gate A fires at the end of step 3, before any execution, over the committed plan or spec. Gate B fires in step 4 after every refactor phase, over the refactored unit's diff. Gate C fires at the end of step 5, over every documentation file touched. Gate D fires in step 6 before the commit or close, over commit messages, issue text and any outward prose. At each gate please MUST fan out one fresh-context reviewer subagent per angle, sharing no author context, and the gated step MUST NOT be completed until every finding is resolved. The angles MUST be: for gate A, ask-alignment, code-alignment, docs/diagrams-alignment and clarity/standards; for gate B, design-fit; for gate C, relevance and clarity/cohesion; for gate D, clarity/standards. Running a gate MUST be non-waivable, MUST NOT merge angles into one reviewer, and MUST NOT be skipped because the artifact is "obviously fine" or small; the model for each reviewer MUST be routed per angle by route, with the defaults sonnet for ask-alignment, code-alignment and design-fit and haiku for the others.

Rationale: a small artifact is a cheap review, and "obviously fine" is the author's blind spot. Merging angles merges blind spots.

### 792a18l1 UAT Scenarios

```gherkin
Scenario: 792a18l1a Gate A fans out four reviewers
  Given a committed plan
  When gate A fires
  Then four fresh reviewers run, one each for ask-alignment, code-alignment, docs/diagrams-alignment and clarity/standards, and none shares the author's context

Scenario: 792a18l1b Gate B follows every refactor
  Given two refactor phases in step 4
  When each finishes
  Then a design-fit reviewer reviews each refactored diff

Scenario: 792a18l1c Gate C reviews every touched doc
  Given three touched documents
  When gate C fires
  Then relevance and clarity/cohesion reviewers cover all three

Scenario: 792a18l1d Gate D reviews outward prose
  Given a commit message and an issue comment
  When gate D fires
  Then a clarity/standards reviewer reviews both before the commit or close

Scenario: 792a18l1e A small artifact is still gated
  Given a one-line plan
  When the agent considers skipping gate A
  Then it runs the gate

Scenario: 792a18l1f Angles are not merged
  Given gate A's four angles
  When the agent staffs the gate to save cost
  Then it still uses one reviewer per angle

Scenario: 792a18l1g Staffing is routed
  Given a gate angle with a default model
  When the agent staffs the reviewer
  Then it applies the route rubric to confirm or adjust the default
```

## 792a18m Each reviewer tries to refute the artifact

Requirement: [792a18m Each reviewer tries to refute the artifact](#792a18m-each-reviewer-tries-to-refute-the-artifact) Each reviewer MUST be charged to refute the artifact and not to bless it, by its angle. Ask-alignment MUST trace every element of the user's verbatim ask to a plan item and every plan item back to the ask, with gaps and scope creep as findings, and ask whether the point of the ask is front-and-center. Code-alignment MUST check the plan against the actual working tree (paths, interfaces, patterns, conventions) and verify against the code and not the plan's claims. Docs/diagrams-alignment MUST check the plan against the architecture diagrams, design docs and glossaries, MUST verify every disposition of an author-pasted enumeration-grep list against the files, and MUST still run its own independent discovery pass, because the reviewer is never the list's source and the list never narrows its scan. Clarity/standards MUST check that the prose is clear, concise and per the repository's writing standards. Design-fit MUST check that the refactored result is DRY, SRP-respecting and YAGNI-compliant and reads as if written as part of the whole. Relevance MUST check that each updated document needs the change and whether any other document is now stale. Clarity/cohesion MUST check that each change is clear, concise and cohesive with the surrounding document.

### 792a18m1 UAT Scenarios

```gherkin
Scenario: 792a18m1a Ask-alignment finds a gap and scope creep
  Given a plan that omits one element of the ask and adds an unrequested feature
  When the ask-alignment reviewer runs
  Then it reports the uncovered ask item and the plan item beyond the ask

Scenario: 792a18m1b The enumeration list does not narrow the scan
  Given a plan that carries an author-pasted disposition list
  When the docs/diagrams-alignment reviewer runs
  Then it verifies every listed disposition and also searches independently for surfaces the list missed

Scenario: 792a18m1c A clean pass says what was checked
  Given an artifact with no flaw
  When a reviewer finds nothing
  Then its report states what it checked and does not just bless the artifact
```

## 792a18n Findings are argued to resolution

Requirement: [792a18n Findings are argued to resolution](#792a18n-findings-are-argued-to-resolution) Each reviewer MUST recall first, with phrases drawn from the artifact and its angle, before it reviews. Its findings MUST be refutations with concrete stakes (a quote or `file:line`, why it fails the angle, and what better looks like), and a clean pass MUST state what was checked. The author MUST address every finding by fixing it or rebutting it with reasons, the reviewer MUST ACK or counter, and the reviewer MUST be kept alive until its gate closes. A finding MUST NOT be dropped silently. After about two unresolved rounds, please MUST stop, summarize both positions and escalate through `AskUserQuestion`. The gate itself (a fresh reviewer per angle, recall-first, argued to resolution) MUST NOT be waived by the availability of a review-focused skill, which a reviewer may apply as its discipline source. An angle MUST be marked not applicable only when its subject is absent from the environment (no diagrams in the repository, no documents touched) and the missing subject MUST be named out loud; "the artifact is small" MUST NOT be a skip.

### 792a18n1 UAT Scenarios

```gherkin
Scenario: 792a18n1a A reviewer recalls first
  Given a dispatched reviewer
  When it starts
  Then its first action is to recall with phrases from the artifact and its angle

Scenario: 792a18n1b Every finding is fixed or rebutted
  Given a reviewer raises three findings
  When the author responds
  Then each is fixed or rebutted to the reviewer's ACK and none is dropped

Scenario: 792a18n1c A deadlock is escalated
  Given a finding that is still unresolved after two rounds
  When the author reaches the limit
  Then please summarizes both positions and asks the user through AskUserQuestion

Scenario: 792a18n1d An absent subject is named
  Given a repository with no diagrams
  When gate A reaches the diagrams angle
  Then it marks the angle not applicable and says out loud that no diagrams exist
```

## 792a18o Measured claims in an escalation carry provenance

Requirement: [792a18o Measured claims in an escalation carry provenance](#792a18o-measured-claims-in-an-escalation-carry-provenance) Any measured claim (a count, rate, cost or duration) in a mid-cycle escalation, an `AskUserQuestion` or a stop report, MUST carry its evidence pointer (the file or command that produced it) and a one-line validity statement that answers "verified how?". A claim whose honest validity line is "not verified" MUST NOT ship as a finding: it MUST be verified first or presented explicitly as an unverified hypothesis.

Rationale: the user decides on escalations, and they get provenance.

### 792a18o1 UAT Scenarios

```gherkin
Scenario: 792a18o1a A measured claim has a pointer
  Given an escalation that says a test suite takes 40 seconds
  When the agent writes it
  Then it names the command that produced the figure and how it was verified

Scenario: 792a18o1b An unverified figure is labeled
  Given a figure the agent cannot verify
  When it must still mention it
  Then it presents it as an unverified hypothesis and not a finding
```

## 792a18p The closing learn is preceded by a lessons audit

Requirement: [792a18p The closing learn is preceded by a lessons audit](#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit) Before the closing learn at step 7, please MUST audit the cycle's mechanical corpus: every pre-registered stop that fired, every gate FAIL verdict, every commit whose message contains CORRECTION, supersede or superseded, instrument-invalid, or redraw or redrawn, and every mid-cycle escalation to the user. It MUST map each item to the vault note that captures its lesson, or write the one-line "no lesson: <why>"; MUST treat an unmapped item as a reversal handoff for the closing learn ([792a15e The four kinds of explicit lesson](./learn.md#792a15e-the-four-kinds-of-explicit-lesson)); and MUST show the audit list (item → note or no-lesson) in the cycle's closing report to the user. It MUST NOT use the audit for positive reinforcement, which has no mechanical marker and belongs to the closing learn's confirmed-approach scan.

Rationale: the audit is failure-shaped by design.

### 792a18p1 UAT Scenarios

```gherkin
Scenario: 792a18p1a The corpus is enumerated
  Given a cycle with one fired stop, one gate FAIL, one commit named CORRECTION and one escalation
  When please runs the audit
  Then all four items are on the list

Scenario: 792a18p1b Each item maps or says why not
  Given an enumerated item
  When the audit maps it
  Then it names the vault note that captures the lesson or writes "no lesson: <why>"

Scenario: 792a18p1c An unmapped item becomes a reversal handoff
  Given an item with no vault note
  When the closing learn runs
  Then the item is handed to learn as a reversal

Scenario: 792a18p1d The list is in the closing report
  Given a finished audit
  When please reports to the user
  Then the report includes the item → note-or-no-lesson list
```

## 792a18q A lesson about to be captured asks what should have surfaced it

Requirement: [792a18q A lesson about to be captured asks what should have surfaced it](#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it) For each lesson the cycle is about to capture, please MUST ask which artifact that already existed should have surfaced it first, MUST search the vault for a note that already covers the situation before writing a new one, and MUST, when a note existed but did not surface at the moment it was needed, check whether its `situation:` line matches how the moment actually presented and reword the note when it does not, in preference to writing a second note on the same lesson. It MUST propose a change that ships from the repository (code, skills, guidance) only as a last resort, and MUST NOT file it automatically.

Rationale: a vault note is the user's own and takes effect on their next recall, while anything that ships from the repository makes the user wait on a release. This decision-support addition was committed in 662e50ba and is unvalidated: the larger "surprise harvest" it was proposed with, a seven-marker second pass, rested on a premise that a probe refuted (`dev/eval/LEDGER.md#687-surprise-harvest`), and was not built.

### 792a18q1 UAT Scenarios

```gherkin
Scenario: 792a18q1a An existing note is searched for first
  Given a lesson about to be captured
  When please prepares the capture
  Then it asks which existing artifact should have surfaced it and searches the vault for a note covering the situation

Scenario: 792a18q1b A stale situation line is reworded
  Given a note existed but did not surface because its situation line does not match how the moment presented
  When please resolves it
  Then it rewords the note and does not write a second note on the same lesson

Scenario: 792a18q1c A repository change is a last resort
  Given a lesson that a repository change could prevent
  When please considers it
  Then it proposes the change only as a last resort and files nothing automatically
```

## 792a18r The closing learn gets the lessons and the audit

Requirement: [792a18r The closing learn gets the lessons and the audit](#792a18r-the-closing-learn-gets-the-lessons-and-the-audit) Please MUST run the `learn` skill again at step 7 and hand it the full list of `LESSONS:` lines collected across step 4's dispatches ([792a16f The closing learn gets the whole list](./lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list)) together with the audit's map, and MUST NOT duplicate learn's ad-hoc question-and-answer capture ([792a15k An answered question is captured once](./learn.md#792a15k-an-answered-question-is-captured-once)), which learn performs for substantive answered questions.

### 792a18r1 UAT Scenarios

```gherkin
Scenario: 792a18r1a The closing learn receives both inputs
  Given a finished cycle with a LESSONS list and an audit map
  When please runs the closing learn
  Then learn is given the full list, none entries included, and the audit's map

Scenario: 792a18r1b The question-and-answer capture is learn's
  Given a substantive answered question in the session
  When please closes
  Then it does not write the pair itself
```

## 792a18s Object-level work is delegated

Requirement: [792a18s Object-level work is delegated](#792a18s-object-level-work-is-delegated) Please MUST delegate object-level work and gate reviews to subagents per the route skill, and the orchestrator MUST route, decompose and synthesize without doing the object-level work itself.

### 792a18s1 UAT Scenarios

```gherkin
Scenario: 792a18s1a Reviews and building are dispatched
  Given a plan with a code change and a gate A review
  When please executes
  Then the code change and each reviewer are subagents routed per route, and the orchestrator synthesizes their results
```
