## ADDED Requirements

### Requirement: The API SHALL rate-limit the list endpoint

The system SHALL reject list requests from a client exceeding its configured per-minute rate limit.

#### Scenario: Client exceeds rate limit
- **WHEN** a client issues more list requests than its configured per-minute limit
- **THEN** the response is a 429 with a Retry-After header
