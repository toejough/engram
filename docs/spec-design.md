# Spec Design Principles

Distilled from design sessions. Reference these when writing specs.

## Architecture

1. **Server-side, not client-side.** If something needs to happen, put it in one place (the server), not every client. If there are N clients, anything replicated N times belongs in the server.

2. **Shared log with independent consumers, not routed queues.** Let consumers watch a shared log and decide what to do. Don't build a router that classifies and dispatches request types.

3. **Validation at the boundary, not in the transport.** Validate in one place (the server). Clients pass through errors unmodified. Don't replicate validation logic across clients.

4. **Error recovery as an escalation ladder.** Retry same context (Nx) -> reset context -> retry fresh (Nx) -> escalate loudly to the user. Structured recovery with different strategies at each level, always ending with a human-visible escalation.

## Scope

5. **Simplify scope before designing.** Before designing, ask "what can I cut?" and cut aggressively. Defer features to future specs, not future stages of the current spec.

6. **Retire immediately, don't deprecate.** If something is being replaced, cut it. Dead code is a liability, not backwards compatibility.

## Staging

7. **Each stage must be internally self-consistent.** Skills and docs at each stage describe only what exists at that stage. No forward references to capabilities that land in a later stage.

## User Interaction

8. **User chooses explicitly, no implicit config.** Prefer explicit user decisions over hidden configuration (env vars, convention-based defaults).
