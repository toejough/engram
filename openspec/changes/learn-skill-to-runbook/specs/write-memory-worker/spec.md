## MODIFIED Requirements

**Ordering note:** this delta is written against the main spec's CURRENT text (both `recall` and `learn` still described as skills), because at proposal time `recall-glance-skill-to-runbook`'s own delta to this same requirement has not yet archived. If `recall-glance-skill-to-runbook` archives first, this delta will conflict at archive time — resolve by fixing THIS delta (re-base it on the newer main-spec text, editing only the `learn` half of the sentence; the `recall` half must be updated to whatever `recall-glance-skill-to-runbook` actually shipped, e.g. "the `recall-deep` runbook" rather than silently continuing to assert "`recall`, still a skill"), never by hand-editing the main spec, per the precedent this session already set for an identical conflict between `please-skill-to-runbook` and `runbook-lexical-triggers`'s deltas to `guidance-runbook-follow-frame`.

### Requirement: Write-memory worker SHALL accept handoff and execute vault writes

The worker is carried by the write-memory runbook (`agent-instructions/skills/write-memory/SKILL.md` retired) and is invoked by a parent (`recall`, still a skill; `learn`, a runbook) that has already made the judgment (what to write and why), by fetching the runbook by basename/wikilink as its next action — write-memory has no trigger of its own and is never reached by a user's own words. The worker SHALL receive a structured handoff containing kind (fact/feedback/qa/runbook), content fields, source, and optional chunk-sources, tags, and supersedes. The worker SHALL NOT re-judge the parent's decision and SHALL NOT decide whether to write.

#### Scenario: Worker receives handoff from parent

- **WHEN** `recall` (a skill) or the agent following the `learn` runbook fetches and follows the write-memory runbook with a complete handoff (kind, required content fields, source)
- **THEN** the runbook's steps SHALL compose the corresponding `engram learn` command from the provided fields

#### Scenario: Worker rejects incomplete handoff

- **WHEN** required handoff fields are missing
- **THEN** the worker SHALL ask the parent (via in-session context) to provide the missing fields
- **AND** the worker SHALL NOT invent content on behalf of the parent

#### Scenario: Parent locates the worker by name, not by query

- **WHEN** `recall` or the agent following the `learn` runbook reaches a write site
- **THEN** it names the write-memory runbook's basename (a `[[wikilink]]` or equivalent explicit reference) as the next action, and the agent fetches it with `engram show <basename>` rather than relying on `engram query`'s first-action trigger or similarity match to surface it
