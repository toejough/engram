## ADDED Requirements

### Requirement: Moments SHALL be detected semantically, and successes and dispatches are moments

Any tool auditing the memory loop against real transcripts SHALL detect moments — failures, corrections, rework, successes (a case where a surfaced or injected memory visibly shaped what the agent did), and dispatches (a subagent launch) — by reading the transcript in context, not by matching keywords or fixed phrases (about 68% of corrections are subtle enough that word-matching alone misses roughly two-thirds of them — vault note 112). Detection SHALL flag generously and then discard non-matches, and its quality bar SHALL be a miss rate — how many real moments it fails to catch — measured against hand-labeled examples and agreed in advance, before any full-corpus run; how often it over-flags is never the deciding factor.

#### Scenario: Subtle correction detected
- **WHEN** a transcript window contains a correction with no blunt signal word (a redirect, a quiet reversal, a reviewer rejection in its own idiom)
- **THEN** the auditor reports it as a detected moment

#### Scenario: Success moment detected
- **WHEN** a transcript window shows the agent applying a recalled or prompt-injected memory and the action visibly succeeding
- **THEN** the auditor reports a success moment naming the memory involved (the reinforcement half of the loop, not only the failure half)

#### Scenario: Miss rate agreed before the real run
- **WHEN** the auditor is to be run on the full sample
- **THEN** a hand-labeled calibration sample (≥10–20 windows, stratified by role and repo) exists, the measured miss rate is recorded, and the tolerance was committed before the full run started

### Requirement: Every audited moment SHALL carry a complete record

Each detected moment's record SHALL answer every question needed to see where, if anywhere, the memory loop broke down for it. On the finding side: did a relevant memory exist (with its kind — chunk/fact/feedback/runbook/qa —, whether it was generic or specific to the situation, and whether that existence check was DERIVED or ESTIMATE), did a search actually run (a quick glance, a full recall, an injected prompt, or none), did the search target the right thing (judged separately from whether a search ran at all), was it surfaced (with its rank, and whether an outdated note outranked its replacement), and was it followed (recording how it deviated, if at all: ignored, partial, a step skipped, steps out of order, or contradicted). On the writing side: was this a moment worth learning from, did a learn step fire, did a note get written (and was it well-targeted, did it correctly supersede an old note, was it not a duplicate), and was the used note's strength updated (with mismatches flagged). On the handoff between agents: which memories were actually included in a subagent's prompt, and for dispatch moments, which ones the orchestrator had but left out. Each moment SHALL also carry a failure category from vault runbook 824 (fixable / nothing-could-have-caught-it / found-but-not-followed). No rate SHALL be computed before its counting unit is fixed in advance, and every reported number SHALL be labeled DERIVED (computed from real records) or ESTIMATE (approximated because the exact data wasn't available).

#### Scenario: Complete moment record
- **WHEN** the auditor emits a moment record
- **THEN** every question above has a value or an explicit not-applicable, and the record carries transcript, location, timestamp, repo, role, and a pointer to the evidence

#### Scenario: A rate without a fixed counting unit is a violation
- **WHEN** a rate is reported (e.g., "X% of dispatches lacked a relevant note")
- **THEN** the report names what's being counted (per-moment, per-dispatch, or per-transcript) and labels the number DERIVED or ESTIMATE

### Requirement: Existence SHALL be judged against the vault as of the moment's time

"Was a relevant memory in the vault" SHALL be judged by searching the vault as it existed at the moment's own timestamp T, using search phrases the auditor writes based on the moment itself (the agent's own phrases at the time are judged separately, under "did the search target the right thing"). For notes: a read-only checkout of the vault at the last commit at or before T gives a DERIVED (exact) answer; where git history doesn't exist or doesn't reach back to T, filtering notes by their creation date gives an ESTIMATE (approximate) answer, labeled as such. For chunks: filtering by ingestion date always gives a DERIVED answer. This check SHALL never write to the live vault, and every engram command it runs SHALL run under the existing eval-harness sandboxing (#708).

#### Scenario: Exact answer via git checkout
- **WHEN** the vault's git history covers T
- **THEN** the check queries a detached, read-only checkout at the last commit at or before T, and the moment's existence judgment is labeled DERIVED

#### Scenario: Approximate answer without history
- **WHEN** the vault has no git history covering T
- **THEN** the check filters notes by `created ≤ T`, labels the judgment ESTIMATE, and the report's limits section notes that any edits made to a note after T won't show up in this fallback

#### Scenario: Live vault untouched
- **WHEN** an audit run completes
- **THEN** a before/after fingerprint of the live vault shows no change attributable to the audit

### Requirement: Prompt-injected memories count as searched and surfaced

For subagent moments, any memory present in the dispatch prompt SHALL count as found (Joe's rule, 2026-08-30) — it's recorded as "a search ran (injected)" and "surfaced: yes." Forked subagents inherit the whole parent conversation, so everything in it counts as found for them automatically. The gap on the finding side — memories the orchestrator had surfaced but didn't hand over — SHALL be measured from the orchestrator's side, at the moment of dispatch, using a simple structural check across the full corpus, validated against a smaller LLM-judged sample that tells apart genuinely missing content from content that was just paraphrased into the prompt instead of quoted directly.

#### Scenario: Injected memory counts as found for the subagent
- **WHEN** a subagent transcript's dispatch prompt contains a vault note's content and the subagent later hits a moment that note addresses
- **THEN** the moment records "search ran: injected" and "surfaced: yes," and the audit measures whether the subagent followed the memory, not whether it found one

#### Scenario: Dispatch moment measures the handoff on the orchestrator side
- **WHEN** an orchestrator dispatches a fresh-context subagent while holding memories relevant to the dispatched unit
- **THEN** the dispatch moment records which of those memories the prompt included and which it left out, and the leftover ones count toward the finding-side gap at the per-dispatch counting unit

#### Scenario: Checking for paraphrased content
- **WHEN** the structural check flags a note as missing from a dispatch prompt
- **THEN** a sampled LLM check judges whether the content was actually paraphrased in, genuinely missing, or ambiguous, and the structural check's rate is adjusted before the gap is reported

### Requirement: The audit SHALL emit credit records and SHALL NOT consume them

The audit SHALL emit draft outcome records — `{moment_id, note_ref, situation_text, outcome ∈ {applied-helped, applied-hurt, surfaced-ignored, injected-unused, absent-needed}, ts, evidence{transcript, anchor}}` — as plain data files. The audit SHALL NOT write, amend, activate, supersede, or remove any vault note: acting on these records requires a human judgment call (ADR-0028 D3) and belongs to a separate, later capability.

#### Scenario: Output only, no side effects
- **WHEN** an audit run finishes over a corpus containing applied-hurt evidence for a note
- **THEN** an outcome record for that note exists in the output files, and the note itself is byte-identical to before the run

### Requirement: Go/no-go thresholds SHALL be agreed in advance, and #739 Phase 2 waits on this audit's result

Any threshold that decides something based on the audit's numbers — including the #739 Phase 2 go/no-go decision (on the writing side: what fraction of lost lessons a completion-report step could realistically catch; on the finding side: what fraction of dispatch-time losses a prompt-injection step could realistically catch) — SHALL be committed in writing, along with what result counts as green, yellow, or red, before the full run starts. The final decision SHALL be read straight off that pre-agreed mapping, never re-decided after seeing the actual numbers. Phase 2 (injecting memories into dispatch prompts, plus a completion-report lessons step) SHALL NOT be proposed unless the result is green.

#### Scenario: The pre-agreed threshold decides
- **WHEN** the full run's #739-specific numbers are in
- **THEN** the decision is whatever the pre-agreed mapping says for those numbers, and it's recorded on #739 with the ledger row linked

### Requirement: Results SHALL land as LEDGER rows plus the coverage-map index, additively

Audit findings SHALL be recorded as `dev/eval/LEDGER.md` rows (one claim per row, in the existing format, each pointing at its raw data) and reflected in a coverage-map index section that maps memory function × kind to the rows backing it and the measured real-world rate. Recording audit results SHALL never rename, rewrite (beyond the format's own built-in way of superseding an old row), or remove any existing ledger row or anchor.

#### Scenario: Adding without disturbing what's there
- **WHEN** an audit run's findings are recorded
- **THEN** every pre-existing ledger anchor still resolves, and the new claims appear as new rows referenced from the coverage-map index
