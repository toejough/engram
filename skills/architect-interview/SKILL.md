---
name: architect-interview
description: Technology and architecture interview producing architecture spec with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Architect Interview Skill

Interview the user about technology and architecture choices, then produce a structured architecture specification with traceability IDs.

## Purpose

Read the requirements and design documents, interview the user for technology preferences, then produce a structured architecture specification that:
- Reveals the shape and purpose of the application at a glance
- Uses progressive disclosure (high-level to detail)
- Keeps business logic pure and testable via dependency injection
- Separates concerns cleanly (domain, storage, UI, infrastructure)

Every architecture decision gets an `ARCH-NNN` traceability ID linked to upstream REQ and DES IDs.

**This skill always conducts an interactive interview.** Do not skip the interview and just analyze -- the user's preferences and constraints matter.

## Domain Ownership

This skill owns the **implementation solution space**: how the system is built technically.

**Owns:**
- Technology choices (languages, frameworks, databases)
- System structure (modules, layers, boundaries)
- Data models and schemas
- API design and contracts
- Non-functional requirements (performance, security, scalability)
- Build, deployment, and operational concerns

**Does NOT own (defer to other phases):**
- What problem we're solving → PM
- Whether a feature should exist → PM
- How users interact with the system → Design
- Screen layouts, flows, or terminology → Design

Architecture should be invisible to users. Any architectural decision that affects user experience should be coordinated with Design.

When users raise out-of-scope concerns, acknowledge them, note for the appropriate phase, and redirect: "That sounds like a PM/Design concern. I'll note it for that phase. For now, let's focus on technical implementation."

## Input Documents

The skill reads these files (paths from orchestrator context):
- **Requirements:** requirements.md - what to build and why (contains REQ- IDs)
- **Design:** design.md - UI components, tokens, visual patterns (contains DES- IDs, if design phase was not skipped)

## Phases

### 1. UNDERSTAND - Analyze Requirements

**Goal:** Extract technical implications from product requirements.

**Steps:**
1. Read all spec documents
2. Identify key technical characteristics:
   - Real-time vs. batch operations
   - Offline requirements
   - Data sensitivity / encryption needs
   - Scale expectations (users, data volume)
   - Platform targets (web, mobile, desktop, CLI)
   - Integration points (APIs, services)
3. Note constraints explicitly stated in requirements
4. Identify implicit technical needs

**Output:**
- Technical requirements summary
- Constraints list
- Open questions about technical needs

### 2. RESEARCH - Explore Technology Options

**Goal:** Research and present modern technology options tailored to requirements.

**For each relevant category, research and present 2-4 options with:**
- What it is (1 sentence)
- Why it fits this project
- Trade-offs (pros/cons)
- Recommendation with rationale

**Categories to cover (as applicable):**

- **Language/Runtime** (Go, Rust, TypeScript, Python, etc.)
- **Frontend Framework** (if applicable)
- **Frontend Styling** (if applicable)
- **Backend Framework** (if applicable)
- **Database** (if applicable)
- **Authentication** (if applicable)
- **Deployment** (if applicable)
- **Build System** (if applicable)
- **Testing** (property-based testing, assertion libraries)

**Do NOT assume answers. Present options and ask for user preference.**

### 3. INTERVIEW - Gather Preferences

**Goal:** Get user decisions on technology stack through structured questions.

**For each category:**
1. Present the researched options with trade-offs
2. State your recommendation and why
3. Ask user to choose or suggest alternative
4. Confirm choice before moving on

**Example format:**
```
## Frontend Framework

Based on your requirements (PWA, offline-first, minimal bundle), here are the options:

| Option | Fits Because | Trade-off |
|--------|--------------|-----------|
| **Vanilla TS + Web Components** | Zero runtime, native PWA support | More boilerplate |
| **Lit** | Tiny runtime (5kb), web standards | Smaller ecosystem |
| **Svelte** | Compiles away, great DX | Different mental model |

**My recommendation:** Vanilla TS + Web Components
- Matches your "libraries over frameworks" constraint
- Best PWA/offline support (no hydration issues)

Which would you prefer?
```

**Capture decisions with rationale for documentation.**

### 4. STRUCTURE - Design the Architecture

**Goal:** Design a file structure that reveals intent through progressive disclosure. Assign ARCH IDs to decisions.

**Principles:**
1. **Top-level tells the story** - Looking at root directory reveals what the app is
2. **Directories are boundaries** - Each directory has a single responsibility
3. **Public API at entry points** - Entry files export the public interface
4. **Implementation is private** - Details hidden inside directories
5. **Dependencies flow inward** - Domain has no external dependencies

### 5. MODEL - Define Data Structures

**Goal:** Define domain entities and their relationships. Assign ARCH IDs.

**For each entity:**
- Interface/type definition
- Validation rules
- Relationships to other entities
- Storage considerations

### 6. CONTRACT - Define Service Interfaces

**Goal:** Define the contracts between layers. Assign ARCH IDs.

**For each service:**
- Interface definition (what it does)
- Method signatures
- Error handling approach
- Dependencies it requires

**Dependency injection approach:**
- Services depend on interfaces, not implementations
- Composition root wires concrete implementations
- Enables testing without mocks

### 7. DOCUMENT - Write the Spec

**Goal:** Produce structured architecture specification document with traceability IDs.

**Output file:** `architecture.md` in the project directory.

**Document structure:**

```markdown
# Technical Architecture: <Project Name>

## 1. Overview

<2-3 paragraphs: what this is, key technical decisions, architecture philosophy>

## 2. Requirements Traceability

| Requirement | Technical Implication | Addressed By |
|-------------|----------------------|--------------|
| REQ-001 | Offline storage needed | ARCH-003 (IndexedDB) |

## 3. Technology Stack

| Layer | Choice | Rationale | ARCH ID |
|-------|--------|-----------|---------|
| Language | Go | Performance, stdlib | ARCH-001 |

## 4. Architecture

### ARCH-NNN: <Decision Name>
Traces to: REQ-NNN, DES-NNN
<Description and rationale>

### 4.1 System Context
<High-level diagram: user, system, external services>

### 4.2 Component Overview
<Layer diagram with responsibilities>

### 4.3 Data Flows
<Key flows>

### 4.4 Dependency Injection
<How dependencies are wired>

## 5. Data Models

### ARCH-NNN: <Entity>
Traces to: REQ-NNN
<Type definition, validation, relationships>

## 6. Service Interfaces

### ARCH-NNN: <Service>
Traces to: REQ-NNN, DES-NNN
<Interface definition, methods, errors>

## 7. File Structure

<Full directory tree with annotations>

## 8. Technology Decisions

### Decisions Made
| Decision | Choice | Alternatives Considered | Rationale | ARCH ID |
|----------|--------|------------------------|-----------|---------|

### Patterns Used
| Pattern | Where | Why | ARCH ID |
|---------|-------|-----|---------|

## 9. Error Handling

<Strategy for each layer>

## 10. Testing Strategy

### Test Tooling Requirements
- **Human-readable matchers**: Assertion library that reads like sentences
- **Randomized property exploration**: Property-based testing library

### Testing by Layer
<How DI enables testing, what to test where>

## 11. Open Questions

<Unresolved technical decisions>
```

## Traceability

### ARCH ID Assignment

- Assign sequential `ARCH-NNN` IDs starting from `ARCH-001`
- Every technology choice, architectural decision, data model, service interface, and pattern gets an ARCH ID
- Each ARCH ID must reference at least one upstream REQ or DES ID

### Upstream References

When reading requirements.md and design.md:
- Note all REQ- and DES- IDs
- Map each technical decision to the REQ/DES items it addresses
- Flag any REQ/DES items that have no corresponding ARCH decision

## Interview Rules

1. **Research before recommending** - Don't just list options; explain fit
2. **State opinions clearly** - "I recommend X because..." not "You could use X or Y"
3. **Respect stated constraints** - If requirements say "no frameworks", don't push React
4. **One decision at a time** - Don't overwhelm with all choices at once
5. **Confirm understanding** - Summarize choice and rationale before moving on
6. **Document the "why"** - Capture rationale for future reference

## Transitions

After each technology decision, confirm:

```
"So we're going with [choice] because [rationale].

This means:
- [implication 1]
- [implication 2]

Ready to move on to [next category]?"
```

## Error Handling

**Requirements are vague on technical needs:**
- Make reasonable assumptions
- State assumptions explicitly
- Ask user to confirm

**User wants something that conflicts with requirements:**
- Point out the conflict
- Explain the trade-off
- Ask how to resolve

**User doesn't know/care about a choice:**
- State your recommendation
- Explain it will be documented
- Move on (don't block)

**Technology choice has risks:**
- Name the risk explicitly
- Suggest mitigation
- Get acknowledgment

## Structured Result

When the architecture spec is complete, produce a result summary for the orchestrator:

```
Status: success
Summary: Conducted architecture interview. Produced architecture.md with N architecture decisions.
Files created: architecture.md
Traceability: ARCH-001 through ARCH-NNN assigned, all linked to REQ/DES IDs
Findings: (any spec gaps, requirement conflicts, or cross-skill issues)
Context for next phase: (technology stack summary, key patterns, file structure, testing strategy)
```

## Output Expectations

The architecture spec should be:
- **Revealing** - Structure shows what the app is at a glance
- **Layered** - Clear separation of concerns
- **Testable** - DI enables unit testing without mocks
- **Traceable** - Every decision has an ARCH-NNN ID linked to REQ/DES IDs
- **Opinionated** - Clear recommendations, not just options

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = ["docs/architecture.md"]

[[decisions]]
context = "Module structure"
choice = "Use internal/ for implementation"
reason = "Go convention for non-public code"
alternatives = ["Flat package structure"]

[[learnings]]
content = "Codebase uses dependency injection pattern"
```
