## MODIFIED Requirements

### Requirement: Write-memory worker SHALL handle three note kinds with distinct field sets

The worker SHALL support fact (subject/predicate/object), feedback (situation/behavior/impact/action), and qa (question/answer/contributors/certainty) kinds, each with their own field structure and engram command form.

#### Scenario: Fact note composition

- **WHEN** kind=fact is provided with situation, subject, predicate, object, and source
- **THEN** the worker SHALL compose: `engram learn fact --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --subject="<subject>" --predicate="<predicate>" --object="<object>" [--tag ...]`

#### Scenario: Feedback note composition

- **WHEN** kind=feedback is provided with situation, behavior, impact, action, and source
- **THEN** the worker SHALL compose: `engram learn feedback --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --behavior="<behavior>" --impact="<impact>" --action="<action>" [--tag ...]`

#### Scenario: QA note composition

- **WHEN** kind=qa is provided with slug, question, answer, contributors, certainty, and source
- **THEN** the worker SHALL compose: `engram learn qa --slug "<kebab-slug>" --question="<question>" --answer="<answer>" --source="<source>" --certainty <high|medium|low> [--contributors <basename> ...]`

### Requirement: Write-memory worker SHALL handle chunk-sources, tags, and supersedes

The worker SHALL append chunk-source flags for provenance tracking, tag flags for categorical tagging (fact/feedback only), and supersedes flags when the write corrects an existing note.

#### Scenario: Chunk-source provenance

- **WHEN** chunk-sources are provided (source#anchor format)
- **THEN** one `--chunk-source <source#anchor>` flag SHALL be appended per provided chunk ID

#### Scenario: Tag handling for fact and feedback

- **WHEN** tags are provided with fact or feedback kind
- **THEN** one `--tag <family>/<value>` flag SHALL be appended per tag
- **AND** tags SHALL be passed through exactly as provided; the worker SHALL NOT invent tags or write the `vocab/` namespace (vocab terms are auto-assigned by the binary)

#### Scenario: Tag rejection for QA

- **WHEN** tags are provided with kind=qa
- **THEN** the worker SHALL drop the tags and report: `tags dropped: qa takes no tag flags`

#### Scenario: Supersedes handling

- **WHEN** supersedes is provided (format: `<basename>|<type>|<claim>` with type one of updates/narrows/refutes)
- **THEN** `--supersedes="<basename>|<type>|<claim>"` flag SHALL be appended (repeatable)

### Requirement: Write-memory worker SHALL pass runbook triggers through as `--trigger` flags
When a kind=runbook handoff carries an optional `triggers` list, the worker SHALL append one `--trigger="<cue>"` per entry, in order, to the `engram learn runbook` command; when the handoff carries none, no `--trigger` flag is emitted. The template SHALL state that a trigger is a cue word or phrase the user (or an engram notice) would literally write, that a single word must be distinctive enough that whole-word matching will not fire on ordinary prose (a very common word is questioned unless the handoff states over-firing is deliberate), and that matching is case-insensitive, whitespace-collapsed, and whole-word at letter/digit edges.

#### Scenario: Runbook handoff with triggers
- **WHEN** the parent hands off kind=runbook with `triggers: ["/please", "take this end-to-end"]`
- **THEN** the composed command ends with `--trigger="/please" --trigger="take this end-to-end"` (after any `--red-flag` flags)

#### Scenario: Runbook handoff without triggers
- **WHEN** the parent hands off kind=runbook with no `triggers`
- **THEN** the composed command contains no `--trigger` flag
