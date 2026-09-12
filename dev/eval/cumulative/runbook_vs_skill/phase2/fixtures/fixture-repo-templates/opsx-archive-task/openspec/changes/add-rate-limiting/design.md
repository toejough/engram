## Context

The widget API list endpoint currently has no limit on request rate per client.

## Goals / Non-Goals

- Goals: bound per-client request rate on the list endpoint.
- Non-Goals: rate limiting other endpoints.

## Decisions

- Use a fixed per-client token bucket, checked before the list handler runs.

## Risks / Trade-offs

- A too-low limit could throttle legitimate bursty clients; start permissive and tune later.
