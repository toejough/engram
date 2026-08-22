## MODIFIED Requirements

### Requirement: Served writes are attributed to the client-declared caller identity
For a served `learn` or `amend` request, the server SHALL stamp the note's `user:` field from
the identity value the calling `engram` instance declared in the request body — the same
locally-detected value that instance would have used for a local write. The server SHALL NOT
require or consult any edge-authentication header (Cloudflare Access or otherwise) to resolve
this field. The server SHALL reject a served `learn` or `amend` request whose declared
identity is empty, and SHALL perform no vault write in that case.

#### Scenario: A caller's declared identity is stamped as-is
- **WHEN** a served `learn` or `amend` request arrives with a self-detected `user:` value naming identity A
- **THEN** the resulting pending-offer note's `user:` field is A

#### Scenario: An empty declared identity is rejected
- **WHEN** a served `learn` or `amend` request arrives with an empty `user:` value
- **THEN** the server returns an error and performs no vault write

#### Scenario: No edge-authentication header is required
- **WHEN** a served `learn` or `amend` request arrives with a non-empty declared `user:` value and no Cloudflare Access (or other edge-authentication) header at all
- **THEN** the request succeeds and the resulting note's `user:` field is the declared value
