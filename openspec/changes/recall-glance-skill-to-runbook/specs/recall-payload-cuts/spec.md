## MODIFIED Requirements

### Requirement: Agent reads chunk content before judging cluster relevance, even in zero-note clusters

When Step 2.5 of the recall procedure (carried by the `recall-glance`/`recall-deep` runbook set's shared core sub-runbook; `agent-instructions/skills/recall/SKILL.md` retired) processes
a cluster, the agent SHALL fetch every chunk member's content via `engram show-chunk` before
stating any relevance or coverage judgment about that cluster — including clusters whose
membership is entirely chunks with no matched note.

#### Scenario: Chunk content read before judgment

- **WHEN** Step 2.5 processes a cluster during recall
- **THEN** the agent fetches every chunk member's content via `engram show-chunk` before stating a relevance or coverage judgment about that cluster
