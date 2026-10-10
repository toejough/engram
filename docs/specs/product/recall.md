# Recall - Product Spec

`recall` is the skill an agent runs before work that takes more than a single tool call or quick thought. It states the agent's plan, retrieves raw conversation chunks and crystallized vault notes in one ranking, judges what the retrieved items say against the plan, and reports out loud whether memory changed the plan. At its deeper rung it also grows the vault: it refreshes or rewrites the notes that cover a cluster of evidence and writes a note where none does. The retrieval it runs is memory-query's ([memory-query](./memory-query.md)), and the writes go through learn's worker ([write-memory](./write-memory.md)) and note revision ([note-revision](./note-revision.md)); this capability is what the agent does with them. The shipped procedure is `agent-instructions/skills/recall/SKILL.md`. Its steps are numbered in the skill (Step 0 to Step 4) and cited here by those numbers.

## 792a14a The plan is stated before anything is retrieved

Requirement: [792a14a The plan is stated before anything is retrieved](#792a14a-the-plan-is-stated-before-anything-is-retrieved) Recall MUST print, before any `engram` call, three blocks in plain prose: the Ask (what is being asked, in the agent's own words, in one or two sentences), the Situation (the concrete context, such as the repository, the branch, the operation underway and what is loaded) and the Plan (the numbered actions the agent would take next absent any guidance from memory).

Rationale: an unstated plan cannot be tested against memory. The synthesis at the end walks this plan item by item, so skipping the step leaves the rest of the procedure with nothing to compare.

### 792a14a1 UAT Scenarios

```gherkin
Scenario: 792a14a1a The three blocks come first
  Given a user request that needs more than one tool call
  When the agent runs recall
  Then it prints the Ask, the Situation and the Plan before it runs any engram command

Scenario: 792a14a1b The plan is the unguided plan
  Given the agent has not yet queried memory
  When it prints the Plan
  Then the plan lists the actions it would take with no help from memory, as a numbered list
```

## 792a14b The index is swept once per session

Requirement: [792a14b The index is swept once per session](#792a14b-the-index-is-swept-once-per-session) Recall MUST run `engram ingest --auto` before it queries, so that the index holds the latest sessions and document edits, MUST NOT run it again when a sweep already ran earlier in the same session, and MUST continue to query when the sweep fails, saying so.

Rationale: a sweep takes seconds when nothing changed. A re-sweep inside a session indexes only turns already in the agent's context, and a change made elsewhere is picked up by the next sweep, so suspicion of one is not a reason to sweep again. A slightly stale index still beats no retrieval.

### 792a14b1 UAT Scenarios

```gherkin
Scenario: 792a14b1a The first recall of a session sweeps
  Given no sweep has run in this session
  When the agent runs recall
  Then it runs engram ingest --auto before engram query

Scenario: 792a14b1b A later recall does not sweep again
  Given a sweep already ran earlier in this session
  When the agent runs recall again, even suspecting a change from another terminal
  Then it does not run engram ingest --auto

Scenario: 792a14b1c A failed sweep does not stop recall
  Given engram ingest --auto exits with an error
  When the agent continues
  Then it says the sweep failed and still runs the query
```

## 792a14c Recall has two depths

Requirement: [792a14c Recall has two depths](#792a14c-recall-has-two-depths) Recall MUST run in one of two modes chosen by the caller: `deep`, which MUST be the mode when the caller names none, and `glance`, which MUST be used when the caller names it. Deep MUST run the whole procedure, applying memory to the decision and growing the vault. Glance MUST apply memory to the decision without writing vault knowledge: it MUST use about three query phrases instead of ten, MUST keep the reading of candidates (Step 2.5A), the recency weighting (Step 2.5B), the activation (Step 2.7), the closing synthesis (Step 3) and the re-entry query (Step 3.5), and MUST skip the coverage writes (Step 2.5C) and the persisting of a conclusion (Step 4).

Rationale: glance exists so that recall can fire often at everyday decision points, and deep for weighty or
irreversible ones, or when recall should also learn. Glance's activation still bumps the used notes' recency metadata
and pulls a used parent note down as a pending copy, which are kept; neither is a knowledge write.

### 792a14c1 UAT Scenarios

```gherkin
Scenario: 792a14c1a No mode means deep
  Given recall is invoked with no mode word
  When it runs
  Then it follows the deep procedure, with ten phrases and the write side

Scenario: 792a14c1b Glance asks for about three phrases
  Given recall is invoked with the mode glance
  When it phrases its query
  Then it generates about three phrases, not ten

Scenario: 792a14c1c Glance writes no vault knowledge
  Given recall in glance mode surfaces a cluster and reads its candidates
  When it finishes
  Then no note was created or amended by coverage judgment or by persisting a conclusion

Scenario: 792a14c1d Glance still reads, weights, activates and synthesizes
  Given recall in glance mode
  When it runs
  Then it reads the candidates, applies the recency weight, activates the notes it used and closes with the plan synthesis
```

## 792a14d Glance escalates to deep for a recent standard

Requirement: [792a14d Glance escalates to deep for a recent standard](#792a14d-glance-escalates-to-deep-for-a-recent-standard) Recall in glance mode MUST switch to deep when its decision turns on honoring a standard that surfaced in the recent-activity channel (an item such as "use X going forward" or "the new convention is Y"), and MUST NOT escalate for applying conventions, for recency supersession within the matched set, or for abduction and synthesis.

Rationale: glance reliably surfaces a recent-channel item but does not elevate it to a requirement, so deep is the
rung that honors such a standard.

### 792a14d1 UAT Scenarios

```gherkin
Scenario: 792a14d1a A recent convention that the decision turns on
  Given a glance recall surfaces a recent-activity item stating a new convention, and the decision requires applying it
  When the agent reaches the closing synthesis
  Then it has switched to deep first

Scenario: 792a14d1b Ordinary conventions do not escalate
  Given a glance recall surfaces conventions in the matched set, a newer note superseding an older one within it, or evidence that supports an abduction
  When the agent applies them
  Then it stays in glance and completes
```

## 792a14e Ten phrases from ten angles

Requirement: [792a14e Ten phrases from ten angles](#792a14e-ten-phrases-from-ten-angles) Recall in deep mode MUST generate exactly ten short, specific query phrases, one from each of these angles: the situation or setting, the user's intent or goal, the current concrete action, the problem or blocker, a candidate solution or approach, the tooling or technology in play, prior related work, an adjacent technique, a failure mode to avoid and the domain or concept. It MUST NOT pre-filter them beyond dropping obvious dross such as a bare noun, and MUST phrase them by the task and not by fear.

Rationale: the agent cannot know what memory holds before it queries, so breadth is for finding crystallization
candidates. "Implementing Claude Code hooks" retrieves the lessons for the task, while "common mistakes when writing
hooks" retrieves only warnings. The binary caps each phrase to its top 30 matches. Glance's three phrases are the
retrieval floor.

### 792a14e1 UAT Scenarios

```gherkin
Scenario: 792a14e1a Ten phrases, one per angle
  Given a deep recall about adding a flag to a CLI
  When the agent phrases its query
  Then it has exactly ten phrases, one for each of the ten angles

Scenario: 792a14e1b Phrases name the task, not the fear
  Given the angle "failure mode to avoid" for a task about writing hooks
  When the agent phrases it
  Then the phrase names a task or failure as it would occur in work, not a generic warning about hooks
```

## 792a14f One unified query

Requirement: [792a14f One unified query](#792a14f-one-unified-query) Recall MUST run ONE `engram query --lazy-chunks` that carries every phrase as a repeatable `--phrase="<phrase>"`, MUST NOT run one query per phrase, and MUST NOT pass `--vault` or `--chunks-dir`, because the binary resolves the vault and the chunk index itself. It MUST NOT dispatch subagents to synthesize clusters.

Rationale: the query clusters the matched notes and chunks together in one pass, which phrase-by-phrase calls would
split. Cluster synthesis runs inline in the recall skill.

### 792a14f1 UAT Scenarios

```gherkin
Scenario: 792a14f1a All phrases in one call
  Given ten phrases
  When the agent queries memory
  Then it runs one engram query with ten --phrase flags and no other query call for them

Scenario: 792a14f1b No vault flags
  Given the agent composes the query
  When it writes the command
  Then the command has neither --vault nor --chunks-dir

Scenario: 792a14f1c Values are attached to their flags
  Given a phrase that begins with a dash
  When the agent writes the flag
  Then it is --phrase="-..." with the quote touching the equals sign
```

## 792a14g The payload is read by kind and by channel

Requirement: [792a14g The payload is read by kind and by channel](#792a14g-the-payload-is-read-by-kind-and-by-channel) Recall MUST read the items of the matched channel by kind and MUST treat the recent-activity channel as situational context only. A chunk item is evidence from which the agent extracts the convention, decision or correction it shows, and MUST NOT be quoted wholesale. A fact, feedback or runbook note is a crystallized lesson that the agent applies directly from its inline content. An item with `provenance: explore` is judged with the notes the clusters nominate. Items with `provenance: recent` MUST NOT be used for coverage or synthesis judgment, are the agent's own recent activity and MUST NOT be re-derived or met with surprise. When the matched channel is empty, recall MUST say so in one sentence, skip the coverage step and proceed with its plan, and a non-empty recent block alone MUST NOT count as something surfacing.

### 792a14g1 UAT Scenarios

```gherkin
Scenario: 792a14g1a Chunks are evidence, not quotations
  Given a matched chunk that records a reviewer's correction
  When the agent uses it
  Then it states the principle the chunk evidences and does not paste the chunk

Scenario: 792a14g1b Notes are applied as they stand
  Given a matched fact note with its content inline
  When the agent judges it
  Then it applies the note without running engram show

Scenario: 792a14g1c Recent items are read first and not judged
  Given a payload with a recent-activity block
  When the agent reads it
  Then it uses the block to re-immerse in recent work and does not count it for coverage

Scenario: 792a14g1d An empty matched set is reported
  Given the matched channel is empty and only recent items are present
  When the agent reads the payload
  Then it says in one sentence that nothing matched, skips the coverage step and proceeds with its plan
```

## 792a14h Chunk content is read before a cluster is judged

Requirement: [792a14h Chunk content is read before a cluster is judged](#792a14h-chunk-content-is-read-before-a-cluster-is-judged) Recall MUST read the content of every chunk member of a cluster with `engram show-chunk <source#anchor>` before stating any relevance or coverage judgment about that cluster, including a cluster whose candidate list is empty. It MUST read a note candidate's content directly from the payload, MUST NOT run `engram show` for it, and MUST NOT judge a chunk from its path, title or anchor alone.

Rationale: under `--lazy-chunks` a chunk item carries no content, so a judgment from its path is a judgment without
evidence. The rule is in the skill's own text and not in a vault note, because a note about correctly running that
step has no trigger surface during the step's own execution.

### 792a14h1 UAT Scenarios

```gherkin
Scenario: 792a14h1a A zero-note cluster still gets its chunks read
  Given a cluster whose candidate list is empty
  When the agent processes it
  Then it runs engram show-chunk on every chunk member before stating any judgment about the cluster

Scenario: 792a14h1b A title is not evidence
  Given a chunk member whose path suggests it is unrelated
  When the agent has not fetched its content
  Then it does not call the chunk unrelated or not applicable

Scenario: 792a14h1c A load-bearing chunk beside a distractor
  Given a cluster of two chunks and no note, one carrying a hard-requirement convention topically distant from the task
  When the agent processes the cluster
  Then it reads both chunks' content before judging either relevant or irrelevant

Scenario: 792a14h1d Candidate notes are read from the payload
  Given a cluster with candidate notes carrying inline content
  When the agent reads them
  Then it runs no engram show for them and reads their content before judging coverage
```

## 792a14i A newer claim wins only when it conflicts

Requirement: [792a14i A newer claim wins only when it conflicts](#792a14i-a-newer-claim-wins-only-when-it-conflicts) Recall MUST treat evidence as conflicting only when a newer member explicitly negates or reverses an older claim (for example "no longer", "replaced by", "use X not Y", or the same subject and predicate with a different object in a newer item), and when it conflicts MUST let the newer win. When it does not conflict, recall MUST treat older and newer evidence as independently valid and MUST NOT demote a stable convention because it lacks a recent instance.

### 792a14i1 UAT Scenarios

```gherkin
Scenario: 792a14i1a A reversal favors the newer note
  Given two notes on one convention, the newer saying "use X, not Y"
  When the agent judges the cluster
  Then it takes X as the standard

Scenario: 792a14i1b A stable convention is not demoted
  Given an old convention that no newer item contradicts
  When the agent judges the cluster
  Then it keeps the convention as valid

Scenario: 792a14i1c A candidate that only matches the superseded claim is near
  Given a candidate note that states only the superseded stance
  When the agent judges coverage against the recency-weighted view
  Then it judges the candidate near and not covered
```

## 792a14j Coverage is judged by the agent and written once per cluster

Requirement: [792a14j Coverage is judged by the agent and written once per cluster](#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster) In deep mode, recall MUST judge, for each cluster and then for the explore items, whether a candidate note covers the cluster's principle, from the content read and not from cosine alone, and MUST act on the outcome. When a candidate states the principle with no material omission against the recency-weighted view, the outcome is covered and recall MUST run `engram amend --target <candidate> --activate --chunk-source <ids>` and MUST NOT rewrite its content. When a candidate addresses the same situation but omits at least one substantive claim the members evidence, the outcome is near and recall MUST run `engram amend` with the chunk sources and re-synthesized content flags from all members, recency-weighted. When no candidate addresses the situation, the outcome is absent and recall MUST hand off to the write-memory skill one note of kind fact or feedback with the cluster's chunk-source IDs and a descriptive source. When the write corrects, narrows or refutes a surfaced note, recall MUST also pass `--supersedes="<basename>|<type>|<claim>"` with type `updates`, `narrows` or `refutes`. Recall MUST write at most one note per cluster, the representative being a note and never a chunk, MUST wait for each write before the next cluster, and MUST NOT hand-author vocabulary tags.

Rationale: coverage is judged by the agent, not by a cosine threshold, because cosine only nominates the
within-cluster top-five candidates. An empty or absent candidate list is a reason to create, not to skip the cluster.
A write is blocking because the note created or updated for one cluster may be a candidate for another. Updates use
`engram amend` and `engram learn` creates only, so the amend path is the one that keeps both copies of a note in step.

### 792a14j1 UAT Scenarios

```gherkin
Scenario: 792a14j1a Covered refreshes provenance and recency only
  Given a candidate that states the cluster's principle with no omission
  When the agent judges it covered
  Then it runs engram amend --target <candidate> --activate --chunk-source <new chunk IDs> with no content flags

Scenario: 792a14j1b Near re-synthesizes
  Given a candidate that omits one claim the members evidence
  When the agent judges it near
  Then it runs engram amend with the chunk sources and re-synthesized content flags

Scenario: 792a14j1c Absent creates through the worker
  Given a cluster with no candidate that addresses its situation
  When the agent judges it absent
  Then it hands one fact or feedback note, with the chunk sources and a source string, to the write-memory skill

Scenario: 792a14j1d A correction carries a supersession
  Given a write that corrects a surfaced note
  When the agent writes it
  Then the command or handoff carries a supersedes value with the note, the type and the claim

Scenario: 792a14j1e One note per cluster
  Given a cluster judged absent
  When the agent writes
  Then it writes one note of one kind and not both a fact and a feedback

Scenario: 792a14j1f High cosine does not decide
  Given a candidate at cosine 0.95 or above
  When the agent has not read its content
  Then it does not judge it covered

Scenario: 792a14j1g Writes are sequential
  Given two clusters to write
  When the agent processes them
  Then it waits for the first write to finish before judging the second

Scenario: 792a14j1h Updates never use learn
  Given an existing note that needs new content
  When the agent updates it
  Then it uses engram amend and not engram learn --target
```

## 792a14k A note from the parent vault is activated, never amended

Requirement: [792a14k A note from the parent vault is activated, never amended](#792a14k-a-note-from-the-parent-vault-is-activated-never-amended) Recall MUST NOT run `engram amend` on an item with `from_parent: true` or on its pending pulled copy, MUST NOT write a new note that restates it, and MUST activate it when it covers a cluster. This holds even when the user asks to enrich or record against that note: the copy becomes an amend target only after curation accepts it.

### 792a14k1 UAT Scenarios

```gherkin
Scenario: 792a14k1a A parent note covers a cluster
  Given a from_parent item that states a cluster's principle
  When the agent judges coverage
  Then it activates the note and does not amend it or write a new note restating it

Scenario: 792a14k1b A pending copy is not amended
  Given a pulled copy of a parent note that curation has not yet accepted
  When the user asks the agent to enrich the note that states the rule
  Then the agent curates first and amends the copy only afterward
```

## 792a14l Only the notes that were used are activated

Requirement: [792a14l Only the notes that were used are activated](#792a14l-only-the-notes-that-were-used-are-activated) After processing the clusters, recall MUST run one `engram activate` with a `--note` for each note it actually drew on (a candidate judged covered or near, and any note cited in the closing synthesis), and MUST skip the call when it drew on no notes. It MUST NOT activate every returned note, MUST NOT activate a chunk or a recent-channel item, and MUST activate a used `from_parent` note the same way, by its path. Activation applies in both modes. Recall MUST NOT curate the pulled copies as part of the recall in either mode: it MUST leave them pending, finish the user's request, run the curate skill as a separate step afterward, and say in its reply that it did or that the copies await curation.

Rationale: activation is agent-driven: the binary does not pre-judge use. A returned but unused note's last-used time
goes stale and it fades by recency rank, which is the supersede-by-competition mechanism, so bumping every returned
note would defeat it. Regularly useful memory stays fresh and never-retrieved memory decays and loses rank. The "vault
holds pending offer(s) awaiting curation" warning that activation may print is expected, not an error.

### 792a14l1 UAT Scenarios

```gherkin
Scenario: 792a14l1a Used notes are activated
  Given the agent judged one candidate covered and cited another in its synthesis
  When it activates
  Then one engram activate call carries a --note for each and for no other note

Scenario: 792a14l1b Unused returns are not activated
  Given a payload with ten notes of which the agent used two
  When it activates
  Then the other eight are not activated

Scenario: 792a14l1c Chunks and recent items are never activated
  Given a payload with chunks and a recent block
  When the agent activates
  Then no chunk or recent-channel item is among the notes

Scenario: 792a14l1d A used parent note is pulled down and left pending
  Given the agent used a from_parent note
  When it activates the note by its path
  Then a pending local copy appears, the agent does not amend it, and the reply says the copy awaits curation or that curate ran as a separate step

Scenario: 792a14l1e No notes drawn, no activation
  Given the payload was empty or the coverage step was skipped
  When the agent finishes
  Then it runs no engram activate
```

## 792a14m The closing synthesis says whether memory changed the plan

Requirement: [792a14m The closing synthesis says whether memory changed the plan](#792a14m-the-closing-synthesis-says-whether-memory-changed-the-plan) Recall MUST close with a synthesis shown to the user that opens with one sentence counting what the query surfaced and what was crystallized ("Query surfaced N items (K chunks, M notes); crystallized J lessons."), walks the Step 0 plan in order marking each numbered action confirmed, adjusted (and how), contradicted or silent, leads the load-bearing conventions with "Apply these as requirements:" drawn from lessons and chunk evidence alike, and surfaces contradictions inline where they affect the action. It MUST NOT dump the payload, MUST NOT paste whole chunks or list raw scores or paths, and MUST be one sentence when memory was silent on everything.

### 792a14m1 UAT Scenarios

```gherkin
Scenario: 792a14m1a The count opens the synthesis
  Given a query that surfaced 12 items, 7 chunks and 5 notes, and one lesson crystallized
  When the agent writes its synthesis
  Then it begins "Query surfaced 12 items (7 chunks, 5 notes); crystallized 1 lessons."

Scenario: 792a14m1b The plan is walked in order
  Given a three-action plan
  When the agent synthesizes
  Then each action appears in order as confirmed, adjusted, contradicted or silent

Scenario: 792a14m1c Conventions are framed as requirements
  Given a load-bearing convention from a note and one from a chunk
  When the agent synthesizes
  Then both are listed under "Apply these as requirements:"

Scenario: 792a14m1d Silence is brief
  Given memory was silent on every plan item
  When the agent synthesizes
  Then the synthesis is one sentence

Scenario: 792a14m1e No payload dump
  Given a long payload
  When the agent synthesizes
  Then the reply contains no re-emitted payload, pasted chunk or list of raw scores or paths
```

## 792a14n A recommendation is reconciled against the asked task

Requirement: [792a14n A recommendation is reconciled against the asked task](#792a14n-a-recommendation-is-reconciled-against-the-asked-task) Before recall states any option, lever or recommendation that the work generated, it MUST reconcile it against the asked task and the decisions already in context, MUST default to the asked task, and MAY deviate only by naming a genuinely new fact and stating that it is reversing direction. It MUST NOT substitute a prerequisite, a better test or a more rigorous foundation as the thing to do first, and MUST NOT treat reasoning already in context as new evidence.

Rationale: substituting a prerequisite feels like diligence but relitigates the settled task. Reconcile against prior outcomes, not mere mentions.

### 792a14n1 UAT Scenarios

```gherkin
Scenario: 792a14n1a The asked task stays first
  Given a user asked for task T and the agent now sees a more rigorous prerequisite P
  When the agent writes its recommendation
  Then it does T first, and proposes P instead only on a new fact that it names as a reversal

Scenario: 792a14n1b Old reasoning is not new evidence
  Given the agent's own earlier reasoning in context favors P
  When it considers deviating
  Then it does not count that reasoning as the new fact
```

## 792a14o An emergent recommendation is checked against the vault

Requirement: [792a14o An emergent recommendation is checked against the vault](#792a14o-an-emergent-recommendation-is-checked-against-the-vault) Before the synthesis is shown, when it is about to ship a recommendation, lever or approach, named as the thing to do, that does not appear in the Step 0 plan, recall MUST run one more `engram query --lazy-chunks` keyed to the recommendation itself with three phrases (the recommendation in its own words, the recommendation with "rolled back rejected not worth it superseded", and the recommendation with "tried measured outcome"), applying the recency weight to what returns. The synthesis MUST carry one `Re-entry:` line per emergent recommendation directly above the final recommendation statement, in one of three forms: clean (with the best-match basename or "no items"), CLOSED with the prior outcome and "drop", or CLOSED with the prior outcome and "revisit because" named new evidence. A recommendation MUST NOT ship without its line. A CLOSED recommendation marked drop MUST be withdrawn, with the synthesis saying what replaces it. A revisit MUST be valid only with new evidence that did not exist when the note was written, and re-weighing the note's own tradeoffs MUST NOT count. A note this query surfaced and the agent cites MUST be included in the activation, which MUST run again if it already ran. The step MUST run in both modes.

Rationale: this checks the vault for evidence outside the agent's context. The reconcile of [792a14n A recommendation is reconciled against the asked task](#792a14n-a-recommendation-is-reconciled-against-the-asked-task) checks reasoning already in context. The line is the proof that the check ran.

### 792a14o1 UAT Scenarios

```gherkin
Scenario: 792a14o1a A new recommendation triggers a re-entry query
  Given the synthesis is about to recommend an approach that is not in the Step 0 plan
  When the agent prepares to show it
  Then it first runs an engram query keyed to that approach with three phrases

Scenario: 792a14o1b The line sits above the recommendation
  Given a clean re-entry result
  When the agent writes its reply
  Then "Re-entry: <recommendation> — clean (top hit: <basename or no items>)" is directly above the recommendation statement

Scenario: 792a14o1c A closed recommendation is dropped
  Given the re-entry query returns a note recording that the approach was rolled back
  When the agent writes the Re-entry line
  Then it is CLOSED with the prior outcome and "drop", and the synthesis says what replaces it

Scenario: 792a14o1d A revisit needs new evidence
  Given a closed approach and no evidence newer than the note
  When the agent considers keeping it
  Then it does not use "revisit", because re-weighing the note's tradeoffs is not new evidence

Scenario: 792a14o1e A cited re-entry note is activated
  Given the re-entry query surfaced a note that the agent cites
  When the agent activates
  Then that note is included, in a second activation call if the first already ran

Scenario: 792a14o1f A recommendation without its line violates the step
  Given a reply with a "RECOMMENDATION:" line and no Re-entry line directly above it
  When it is checked
  Then it violates the step
```

## 792a14p A sound conclusion is persisted as a derived note

Requirement: [792a14p A sound conclusion is persisted as a derived note](#792a14p-a-sound-conclusion-is-persisted-as-a-derived-note) In deep mode, when the closing synthesis reaches a sound, non-trivial conclusion that no existing note states, recall MUST hand one synthesis note per conclusion to the write-memory skill as a fact or feedback, in which the note is the conclusion itself and not a restatement of an input. Recall MUST state a deduction as following necessarily and MUST mark an abduction or induction as probable, a best explanation or defeasible. It MUST mark the source as derived (for example `--source="synthesis (abduction) from recalled memory"`) and, when the conclusion corrects, narrows or refutes a surfaced note, include that note's basename, type and claim in the handoff. It MUST NOT persist a hunch, a conclusion it would hedge below probable, or a conclusion that merely re-aggregates one note, and MUST NOT hand-author a vocabulary tag or a supersession backlink. It MAY type an inline `[[full-basename]]` citation of an input note into the body.

Rationale: reasoning that is never written down evaporates; the note records the outcome and grows the web. A non-truth-preserving inference is a hypothesis, not a fact (vault note 69), and persisting only sound conclusions keeps the vault from rotting (vault note 68).

### 792a14p1 UAT Scenarios

```gherkin
Scenario: 792a14p1a A sound new conclusion is handed off
  Given a synthesis that reaches a conclusion no note states
  When the agent judges it sound and non-trivial
  Then it hands one note to the write-memory skill with a derived source string

Scenario: 792a14p1b An abduction is marked probable
  Given a conclusion reached by abduction
  When the agent writes the note
  Then its text marks the conclusion probable, a best explanation or defeasible

Scenario: 792a14p1c A hunch is not persisted
  Given a conclusion the agent would hedge below probable
  When it considers persisting
  Then it writes no note

Scenario: 792a14p1d A glance recall persists nothing
  Given a glance recall that reaches a conclusion worth keeping
  When it finishes
  Then it writes no note

Scenario: 792a14p1e Citations are typed inline
  Given a synthesis note that cites an input note's claim
  When the agent writes the body
  Then it types [[full-basename]] itself and writes no vocab tag or Supersedes line
```

## 792a14q A cited synthesis is also captured as a question and answer

Requirement: [792a14q A cited synthesis is also captured as a question and answer](#792a14q-a-cited-synthesis-is-also-captured-as-a-question-and-answer) When a persisted synthesis body contains at least one `[[full-basename]]` wikilink, recall MUST also hand off to the write-memory skill a note of kind qa carrying the verbatim question, the synthesis conclusion as the answer, a certainty matching the synthesis note's label, as contributors the wikilink basenames found in the body, and the source "recall Step 4, session <date>". It MUST NOT ask the agent to list its inputs freely, and MUST skip the capture when the body has no wikilink.

Rationale: contributors are extracted from wikilinks the agent already wrote. A body with no citation has no provenance to record, so at least one citation is the bar.

### 792a14q1 UAT Scenarios

```gherkin
Scenario: 792a14q1a A cited synthesis gets a qa note
  Given a persisted synthesis body that cites [[a]] and [[b]]
  When recall finishes persisting
  Then a qa handoff follows with contributors a and b, the question verbatim, the conclusion as the answer and a matching certainty

Scenario: 792a14q1b An uncited synthesis gets none
  Given a persisted synthesis body with no wikilink
  When recall finishes persisting
  Then no qa note is handed off
```
