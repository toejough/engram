## ADDED Requirements

### Requirement: Agent reads chunk content before judging cluster relevance, even in zero-note clusters

When Step 2.5 of the recall procedure (`agent-instructions/skills/recall/SKILL.md`) processes
a cluster, the agent SHALL fetch every chunk member's content via `engram show-chunk` before
stating any relevance or coverage judgment about that cluster — including clusters whose
`candidate_l2s` list is empty (no note candidates, chunks-only membership). The absence of a
note candidate to write does not exempt a cluster's chunk members from the read-before-judge
requirement.

#### Scenario: Zero-note cluster still gets its chunks read
- **WHEN** Step 2.5 processes a cluster whose `candidate_l2s` list is empty
- **THEN** the agent invokes `engram show-chunk` on every chunk member of that cluster before
  stating any relevance or coverage judgment about it

#### Scenario: Metadata-only judgment is a violation
- **WHEN** a chunk member's `path` or anchor text (title) alone — without its fetched content
  — is used as the basis for judging it "unrelated," "not applicable," or otherwise irrelevant
- **THEN** the judgment fails this requirement, regardless of how many other members of the
  same cluster were genuinely read and correctly judged

#### Scenario: A load-bearing chunk surfaces in a zero-note cluster
- **WHEN** a chunk carrying a hard-requirement convention (e.g. a recency-channel standard
  planted topically distant from the task) matches into a cluster with no note candidates,
  alongside an unrelated distractor chunk
- **THEN** reading both chunks' content (not just their titles) is required before either is
  judged relevant or irrelevant — this is the exact configuration in which #733's C5b honoring
  miss occurred (`dev/eval/traps/c5.py`, trial idx=3, `gate-C5-6l_mjvl1`)
