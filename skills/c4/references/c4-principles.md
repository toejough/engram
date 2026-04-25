# C4 Principles

Distilled from c4model.com. The skill consults this when it needs to verify a level boundary,
abstraction choice, or relationship style.

## The 4 Abstractions

| Abstraction      | What it represents                                                                 | Examples                                                                                  |
|------------------|------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------|
| Person           | A human actor, role, persona, or named individual that uses the system             | End user, admin, support engineer, "Customer", "Operator"                                 |
| Software System  | The highest-level unit of software that delivers value; made of one or more containers | "Engram", "Internet Banking System", a third-party SaaS, an external auth provider     |
| Container        | A separately runnable/deployable unit (application or data store) inside a system  | Web app, API service, mobile app, SPA, database, message broker, file store, cache        |
| Component        | A logical grouping of related functionality inside a single container, with a clear responsibility | A package/module, a controller, a "RecallPipeline", a "TokenResolver"             |

Code elements (classes, functions, tables) sit below components but are not a top-level abstraction —
they are the substrate components are built from.

## The 4 Levels

### L1 — System Context
Shows: the system in scope as a single box, the people who use it, and the external systems it
talks to. High-level shape only.
Hides: containers, components, technologies, protocols, internal structure.
Audience: everyone — technical and non-technical, inside and outside the team.
Count: exactly one per software system.

### L2 — Container
Shows: the containers (apps, data stores) that make up the system, the major technology choices,
and how they communicate (protocol, sync/async).
Hides: components inside containers, deployment topology (clustering, load balancers, replication,
failover — those belong in deployment diagrams), code-level detail.
Audience: technical stakeholders — architects, developers, ops/support.
Count: exactly one per software system.

### L3 — Component
Shows: the components inside one container, their responsibilities, and the technology/implementation
notes for each. One diagram zooms into one container.
Hides: code-level detail (classes, functions), other containers' internals.
Audience: architects and developers working on that container.
Count: at most one per container, and only when it adds value. Prefer to automate generation for
long-lived docs; otherwise omit.

### L4 — Code
The official model uses class/UML-style diagrams here. **Engram-specific deviation:** this skill
replaces L4 with a property/invariant ledger (see `property-ledger-format.md`). Reason: UML
goes stale fast, IDEs show class structure, and the durable thing is what the code GUARANTEES.

## Common Pitfalls

1. **Conflating containers with components.** A container is independently runnable/deployable;
   a component is an in-process module. If two things run in separate processes, they are
   separate containers, not components.
2. **Mixing levels in one diagram.** Containers and components on the same canvas, or context-level
   externals drawn alongside L3 components, breaks the hierarchy and the audience promise.
3. **Anonymous or vague arrows.** Every relationship needs a labeled verb phrase ("reads from",
   "publishes events to") and ideally a protocol/technology. "Uses" and unlabeled lines are
   pitfalls — they hide the actual interaction.
4. **Putting deployment detail in the container diagram.** Load balancers, replicas, regions, and
   failover belong in a deployment diagram, not L2.
5. **Drawing every component for every container.** L3 is opt-in. Only draw it when the container
   is complex enough that the picture earns its keep.
6. **Skipping the legend / inconsistent notation.** Shape, color, and arrow style must be defined
   on the diagram or in a shared key; viewers should not have to guess.

## When to Split a New Diagram

- **L1 to L2:** when readers ask "what's inside the system?" — add a container diagram for the
  one system in scope. Don't expand the L1 diagram itself.
- **L2 to L3:** when one container is complex enough that its responsibilities aren't obvious
  from name + technology — add a component diagram for that single container only.
- **L3 to L4:** when a component encodes non-obvious invariants or contracts that must not
  regress — record them in the property ledger, not as a UML class diagram.
