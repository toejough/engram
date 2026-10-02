## MODIFIED Requirements

### Requirement: A surfaced runbook SHALL render its `red_flags` and be retrievable in full
When a `runbook` note appears in a query payload (`items[]` or `candidate_l2s`), its rendered content SHALL include the `red_flags` field when present. `engram show <basename>` SHALL return the full note (frontmatter and body), with every `red_flags` entry and no omission marker, so an agent can restate every step and check every red flag when the payload's inline content is truncated. `engram show` is the full-fidelity source the guidance shim names, so it SHALL NOT apply any `red_flags` preview budget. When a runbook's `red_flags` list renders to more than the query preview budget (1200 bytes, measured in rendered YAML bytes), `engram query` SHALL keep the most-recently-added entries that fit and SHALL replace the dropped earlier entries with one in-band marker. The marker SHALL state how many entries were omitted and the list's total, and SHALL name the note's actual basename in an `engram show <basename>` command that returns the full list. A `red_flags` list within the budget SHALL render byte-identically to the note file.

#### Scenario: Red flags visible in the query payload
- **WHEN** a runbook note with `red_flags` is returned by `engram query`
- **THEN** the item's content includes the `red_flags` entries

#### Scenario: Full runbook via show
- **WHEN** an agent runs `engram show <runbook basename>`
- **THEN** the output contains the complete frontmatter (situation, done_when, red_flags if any) and the full step body

#### Scenario: Show never truncates red flags
- **WHEN** an agent runs `engram show <basename>` on a runbook whose `red_flags` render to more than 1200 bytes
- **THEN** the output contains every `red_flags` entry from the note file, in file order, and no omission marker

#### Scenario: Oversized red_flags list keeps its newest entry
- **WHEN** `engram query` returns a runbook whose `red_flags` render to more than the 1200-byte preview budget
- **THEN** the item's content keeps the most-recently-added entries that fit, preceded by a marker naming the number of omitted entries, the total entry count, and `engram show <that note's basename>` as the command that returns all of them — `engram show` itself is exempt from this truncation (see "Show never truncates red flags")

#### Scenario: The omission marker's command does not itself truncate
- **WHEN** an agent runs the exact command named in a query omission marker
- **THEN** the output contains every `red_flags` entry, including the ones the marker reported as omitted
