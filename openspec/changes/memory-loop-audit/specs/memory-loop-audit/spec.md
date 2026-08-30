## ADDED Requirements

### Requirement: Moments SHALL be detected semantically, and successes and dispatches are moments

Any instrument auditing the memory loop on real transcripts SHALL detect moments — failures, corrections, rework, successes (a surfaced or injected memory visibly shaping an action), and dispatches (a subagent launch) — by semantic judgment of the transcript in context, not by keyword or word-match heuristics (~68% of corrections are subtle; word-match misses ~2/3 — vault note 112). Detection SHALL over-match then prune, and its quality bar SHALL be a pre-registered false-negative (miss) rate measured against hand-labeled windows before any full-corpus run; false-positive rate is never the gate.

#### Scenario: Subtle correction detected
- **WHEN** a transcript window contains a correction with no blunt signal word (a redirect, a quiet reversal, a reviewer rejection in its own idiom)
- **THEN** the auditor reports it as a detected moment

#### Scenario: Success moment detected
- **WHEN** a transcript window shows the agent applying a recalled or prompt-injected memory and the action visibly succeeding
- **THEN** the auditor reports a success moment naming the memory involved (the reinforcement half of the loop, not only the failure half)

#### Scenario: Miss-rate pre-registration
- **WHEN** the auditor is to be run on the full sample
- **THEN** a hand-labeled calibration sample (≥10–20 windows, stratified by role and repo) exists, the measured miss rate is recorded, and the tolerance was committed before the full run started

### Requirement: Every audited moment SHALL carry the full funnel record

Each detected moment's record SHALL answer every loss stage of the memory loop: read side — exists (with memory kind chunk/fact/feedback/runbook/qa, generic vs idiosyncratic, and the oracle method used), fired (recall run: glance/deep/injected/none), keyed (phrases matched the situation — scored separately from fired), surfaced (with rank, and whether a superseded note outranked its supersessor), applied (with deviation shape: none/ignored/partial/step-skip/mis-order/contradicted); write side — learn-moment warranted, learn fired, written (with quality: situation keyed as a future task, supersession used when correcting, non-duplicate), strengthened (used notes activated; mismatches flagged); boundary — injected memory refs, and for dispatch moments the orchestrator-surfaced-but-not-injected refs. Each moment SHALL carry a klass per vault runbook 824 (addressable / capture-ceiling / application-or-ranking-miss). No ratio SHALL be computed before its fire-unit is pinned, and every reported number SHALL be labeled DERIVED or ESTIMATE.

#### Scenario: Funnel-complete record
- **WHEN** the auditor emits a moment record
- **THEN** every stage column above is present with a value or an explicit not-applicable, and the record carries transcript, anchor, timestamp, repo, role, and evidence pointer

#### Scenario: Ratio without a pinned fire-unit is a violation
- **WHEN** a rate is reported (e.g., "X% of dispatches lacked a relevant note")
- **THEN** the report names the fire-unit the denominator counts (per-moment, per-dispatch, per-transcript) and labels the number DERIVED or ESTIMATE

### Requirement: Existence SHALL be judged against the vault as of the moment's time

"Was a relevant memory in the vault" SHALL be judged with an oracle search against the vault state at the moment's timestamp T, using phrases the auditor derives from the moment itself (the agent's own phrases are judged separately, under keyed). Notes: a read-only git-worktree checkout at the last vault commit ≤ T yields a DERIVED judgment; where git history is absent or does not cover T, a `created ≤ T` filter yields an ESTIMATE judgment and the label says so. Chunks: filtered by `IngestedAt ≤ T` (DERIVED). The oracle SHALL never write the live vault, and every engram invocation it makes SHALL run under the eval-harness isolation regime (#708).

#### Scenario: DERIVED oracle via git worktree
- **WHEN** the vault's git history covers T
- **THEN** the oracle queries a detached read-only worktree at the last commit ≤ T, and the moment's exists judgment is labeled DERIVED

#### Scenario: Degraded oracle without history
- **WHEN** the vault has no git history covering T
- **THEN** the oracle filters notes by `created ≤ T`, labels the judgment ESTIMATE, and the report's honest-limits section carries the caveat that post-T amendments are invisible to this filter

#### Scenario: Live vault untouched
- **WHEN** an audit run completes
- **THEN** a before/after fingerprint of the live vault shows no change attributable to the audit

### Requirement: Prompt-injected memories count as searched and surfaced

For subagent moments, memories present in the dispatch prompt SHALL score fired=injected and surfaced=yes (Joe's rule, 2026-08-30); fork dispatches inherit the parent context and are injected structurally. The read-side dispatch-handoff gap SHALL be measured on the orchestrator side at dispatch moments — memories the orchestrator had surfaced but did not inject — via a structural proxy over the full corpus validated by an LLM-judged calibration sample that distinguishes genuinely-missing from paraphrase-transferred content.

#### Scenario: Injected memory scores the subagent's funnel
- **WHEN** a subagent transcript's dispatch prompt contains a vault note's content and the subagent later hits a moment that note addresses
- **THEN** the moment records fired=injected and surfaced=yes, and the audit measures apply (did the subagent act in accordance), not retrieval

#### Scenario: Dispatch moment measures the handoff on the orchestrator side
- **WHEN** an orchestrator dispatches a fresh-context subagent while holding surfaced notes relevant to the dispatched unit
- **THEN** the dispatch moment records which of those notes the prompt carried and which it omitted, and omissions feed the read-side gap at the per-dispatch fire-unit

#### Scenario: Paraphrase calibration
- **WHEN** the structural proxy flags a note as missing from a dispatch prompt
- **THEN** a sampled LLM calibration judges transferred-paraphrased vs genuinely-missing vs ambiguous, and the proxy's rate is adjusted before the gap is reported

### Requirement: The audit SHALL emit credit records and SHALL NOT consume them

The audit SHALL emit draft credit records — `{moment_id, note_ref, situation_text, outcome ∈ {applied-helped, applied-hurt, surfaced-ignored, injected-unused, absent-needed}, ts, evidence{transcript, anchor}}` — as data files. The audit SHALL NOT write, amend, activate, supersede, or prune any vault note: credit application is judged-only (ADR-0028 D3) and belongs to a separate capability.

#### Scenario: Emission only
- **WHEN** an audit run finishes over a corpus containing applied-hurt evidence for a note
- **THEN** a credit record for that note exists in the output files, and the note itself is byte-identical to before the run

### Requirement: Gates SHALL be pre-registered, and #739 Phase 2 is blocked on this audit's gate

Decision thresholds consuming audit numbers — including the #739 Phase 2 gate (write-side: % of lost lessons reachable by completion-report harvest; read-side: % of dispatch-moment losses reachable by dispatch injection) — SHALL be committed, with an acceptable-outcomes mapping to GREEN/YELLOW/RED, before the full run starts. The recorded decision SHALL be read off that mapping, never re-derived after seeing the numbers. Phase 2 (dispatch injection + completion-report LESSONS) SHALL NOT be filed before a GREEN.

#### Scenario: Pre-registered gate decides
- **WHEN** the full run's #739-slice numbers are in
- **THEN** the gate decision is the pre-registered mapping's output for those numbers, and it is recorded on #739 with the LEDGER row linked

### Requirement: Results SHALL land as LEDGER rows plus the coverage-map index, additively

Audit findings SHALL be recorded as `dev/eval/LEDGER.md` rows (one claim per row, existing format, raw-data paths) and reflected in a coverage-map index section mapping memory function × kind to the rows evidencing it and the measured real-work rate. Existing LEDGER rows and anchors SHALL never be renamed, rewritten (beyond the format's own supersession mechanism), or removed by audit recording.

#### Scenario: Additive recording
- **WHEN** an audit run's findings are recorded
- **THEN** every pre-existing LEDGER anchor still resolves, and the new claims appear as new rows referenced from the coverage-map index
