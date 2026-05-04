# C4 Workflow Sequence Diagrams

Sequence diagrams for the workflows defined in `skills/c4/SKILL.md`. Lanes:

- **User** — invokes `/c4 <sub-action>`, approves drafts and propagation proposals.
- **LLM** — the agent following the c4 skill (this is "you" when the skill loads).
- **Subagent** — Haiku-class sub-agent dispatched for Tier 1 wide-scan extraction
  (Rule 7: every `scan-source-for-findings` task at every level uses Two-Tier
  Extraction; Tier 1 = recall, Tier 2 = precision verified by the orchestrator).
- **targ** — the project's build-tool binary (`targ c4-l*-build`, `c4-render`,
  `c4-audit`, `c4-l1-externals`, `c4-history`).
- **FS** — the filesystem under `architecture/c4/` plus repo source (`internal/`,
  `cmd/`, `docs/`, `CLAUDE.md`, etc.).

The audit (`targ c4-audit`) takes the `.json` spec only — the rendered `.md`
is a mechanical artifact and is checked solely for staleness via byte-compare
against a fresh emit.

---

## 1. `create 1 <name>` — author L1 system context

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant Subagent
    participant targ
    participant FS

    User->>LLM: "/c4 create 1 <name>"
    LLM->>FS: read architecture/c4/, CLAUDE.md, docs/
    FS-->>LLM: existing diagrams + intent

    LLM->>targ: "targ c4-l1-externals --root . --packages ./..."
    targ->>FS: walk repo, AST-scan
    FS-->>targ: HTTP / FS / subprocess / env candidates
    targ-->>LLM: external-system candidates JSON

    LLM->>targ: "targ c4-history --since 90d --limit 50"
    targ->>FS: read git log
    FS-->>targ: commits + bodies
    targ-->>LLM: structured commit JSON

    Note over LLM,Subagent: Tier 1 — wide scan (Rule 7)
    LLM->>Subagent: dispatch Haiku at boundary surface
    Subagent->>FS: read code touching externals
    FS-->>Subagent: source
    Subagent-->>LLM: candidates — externals + speculated `source` values<br/>(URLs, vendor IDs, repo paths, descriptive text)

    Note over LLM: Tier 2 — verify (Sonnet+)
    LLM->>FS: re-read code for each candidate
    FS-->>LLM: confirm crossings + source values

    alt code/intent conflict
        LLM->>User: present conflict, ask which is correct
        User-->>LLM: resolution
    end

    LLM->>FS: author "architecture/c4/c1-<name>.json"
    LLM->>targ: "targ c4-l1-build --input ... --noconfirm"
    targ->>FS: read .json, validate spec
    targ->>FS: write c1-<name>.md + svg/c1-<name>.mmd
    targ-->>LLM: build OK

    LLM->>targ: "targ c4-render"
    targ->>FS: write svg/c1-<name>.svg

    LLM->>targ: "targ c4-audit --file c1-<name>.json"
    targ->>FS: load + validate JSON spec
    targ->>FS: validate path-like `source` values resolve
    targ->>FS: byte-compare existing .md vs fresh emit (staleness)
    targ-->>LLM: findings (target zero)

    LLM->>User: show rendered markdown
    User-->>LLM: approve
    LLM->>FS: commit .json + .md + .mmd + .svg
```

---

## 2. `create 2 <name>` — author L2 container diagram

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant Subagent
    participant targ
    participant FS

    User->>LLM: "/c4 create 2 <name>"
    LLM->>FS: read parent c1-*.json (and rendered .md for orientation)
    FS-->>LLM: parent S-IDs + element names

    Note over LLM,Subagent: Tier 1 — wide scan
    LLM->>Subagent: dispatch Haiku at the in-scope L1 element's source surface
    Subagent->>FS: read cmd/, internal/, hooks/, skills/, on-disk stores
    FS-->>Subagent: source
    Subagent-->>LLM: candidates — containers + speculated `source` values<br/>(usually repo paths)

    Note over LLM: Tier 2 — verify
    LLM->>FS: confirm first-class containers, in-scope flag,<br/>from_parent carry-overs match the L1 parent

    alt code/intent conflict
        LLM->>User: present conflict, ask
        User-->>LLM: resolution
    end

    LLM->>FS: author "architecture/c4/c2-<name>.json"
    Note right of LLM: focus refines an L1 element<br/>Nn ids scoped to this diagram<br/>from_parent neighbors carry S-IDs<br/>each element has a `source`
    LLM->>targ: "targ c4-l2-build --input ... --noconfirm"
    targ->>FS: read .json
    targ->>FS: write c2-<name>.md + svg/c2-<name>.mmd
    targ-->>LLM: build OK

    LLM->>targ: "targ c4-render"
    targ->>FS: write svg/c2-<name>.svg

    LLM->>targ: "targ c4-audit --file c2-<name>.json"
    targ-->>LLM: findings (spec validation + source paths + .md staleness)

    Note over LLM,targ: Propagation sweep
    LLM->>FS: open parent c1-*.json
    LLM->>User: propose adding child to "cross_links.refined_by"
    User-->>LLM: apply / skip / defer
    LLM->>targ: rebuild parent c1 (auto sections)
    LLM->>targ: rebuild every c2 sibling (refresh "Siblings:")
    LLM->>targ: "targ c4-audit --file <each-modified>.json"
    targ-->>LLM: zero findings or recorded Drift Notes

    LLM->>User: show rendered .md
    User-->>LLM: approve
    LLM->>FS: commit .json + .md + .mmd + .svg
```

---

## 3. `create 3 <name>` — author L3 component diagram

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant Subagent
    participant targ
    participant FS

    User->>LLM: "/c4 create 3 <name>"
    LLM->>FS: read parent c2-*.json
    FS-->>LLM: parent N-IDs + element names

    Note over LLM,Subagent: Tier 1 — wide scan (Rule 7)
    LLM->>Subagent: dispatch Haiku at packages in scope
    Subagent->>FS: read every Go file / cluster of functions in focus container
    FS-->>Subagent: source
    Subagent-->>LLM: candidates — components + speculated `source` paths<br/>(file:line)

    Note over LLM: Tier 2 — verify
    LLM->>FS: read source per candidate, merge / split / prune<br/>locate exact `source` path, confirm `kind: "component"`<br/>requires a path that resolves on disk

    LLM->>FS: author "architecture/c4/c3-<name>.json"
    Note right of LLM: focus.id matches parent's<br/>element, Mn ids scoped<br/>within this diagram<br/>each component has `source`
    LLM->>targ: "targ c4-l3-build --input ... --noconfirm"
    targ->>FS: read .json + verify component `source` paths
    targ->>FS: write c3-<name>.md + svg/c3-<name>.mmd
    targ-->>LLM: build OK

    LLM->>targ: "targ c4-render"
    targ->>FS: write svg/c3-<name>.svg

    LLM->>targ: "targ c4-audit --file c3-<name>.json"
    targ-->>LLM: findings (spec + source paths + .md staleness)

    Note over LLM,targ: Propagation sweep
    LLM->>FS: open parent c2-*.json
    LLM->>User: propose adding child to "cross_links.refined_by"
    User-->>LLM: apply / skip / defer
    LLM->>targ: rebuild parent c2 (auto sections)
    LLM->>targ: rebuild every c3 sibling (refresh "Siblings:")

    LLM->>User: show rendered .md
    User-->>LLM: approve
    LLM->>FS: commit .json + .md + .mmd + .svg
```

---

## 4. `create 4 <name>` — author L4 property ledger (two-tier + two-diagram)

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant Subagent
    participant targ
    participant FS

    User->>LLM: "/c4 create 4 <name>"
    LLM->>FS: read parent c3-*.json
    FS-->>LLM: M-IDs + neighbor names

    Note over LLM,Subagent: Tier 1 — wide scan (Rule 7)
    LLM->>Subagent: dispatch Haiku at component source
    Subagent->>FS: read package files / hook scripts / skill md
    FS-->>Subagent: source
    Subagent-->>LLM: candidates across four types —<br/>properties, call-diagram nodes,<br/>manifest rows (field/type/wirer/wrapped_entity),<br/>R-edge property tags + speculated enforced_at/tested_at

    Note over LLM,Subagent: Tier 2 — verify
    LLM->>FS: re-read source for each candidate
    FS-->>LLM: verified facts
    Note right of LLM: prune false positives,<br/>confirm wrapped_entity_id<br/>matches a call-diagram node,<br/>locate enforced_at / tested_at,<br/>mark UNTESTED if no test

    LLM->>FS: author "architecture/c4/c4-<name>.json"
    LLM->>targ: "targ c4-l4-build --input ... --noconfirm"
    targ->>FS: read .json
    Note right of targ: validate wrapped_entity_id<br/>strict-alignment<br/>reject D-edges
    targ->>FS: write c4-<name>.md
    targ->>FS: write svg/c4-<name>.mmd (call diagram)
    targ->>FS: write svg/c4-<name>-wiring.mmd (derived)
    targ-->>LLM: build OK

    LLM->>targ: "targ c4-render"
    targ->>FS: render both .mmd → .svg
    targ-->>LLM: 2 SVGs written

    LLM->>targ: "targ c4-audit --file c4-<name>.json"
    targ->>FS: load + validate L4 spec, check L3-parent carryover<br/>resolve every property `enforced_at`/`tested_at` path<br/>byte-compare existing .md vs fresh emit
    targ-->>LLM: property_link_unresolved + carryover findings + staleness

    Note over LLM,targ: Propagation sweep
    LLM->>targ: rebuild peer L4 siblings (refresh "Siblings:")
    LLM->>FS: verify parent c3 catalog row current

    LLM->>User: show rendered .md + 2 SVGs
    User-->>LLM: approve
    LLM->>FS: commit .json + .md + 2 .mmd + 2 .svg
```

---

## 5. `update <name>` — modify an existing diagram

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant Subagent
    participant targ
    participant FS

    User->>LLM: "/c4 update <name> ..."
    LLM->>FS: read target + parent + children (per front-matter)
    FS-->>LLM: linked diagrams
    LLM->>FS: re-read affected packages
    FS-->>LLM: current code

    opt re-discovery needed (e.g. new components, renamed externals)
        Note over LLM,Subagent: Tier 1 → Tier 2 extraction (Rule 7)
        LLM->>Subagent: dispatch wide-scan
        Subagent-->>LLM: candidates
        LLM->>FS: verify each
    end

    alt new code/intent conflict
        LLM->>User: ask, record resolution
        User-->>LLM: resolution
    end

    Note right of LLM: classify change:<br/>renamed / removed / new element /<br/>changed responsibility /<br/>source change

    LLM->>FS: edit target .json
    LLM->>targ: "targ c4-l*-build" on target
    targ->>FS: rebuild target .md + .mmd
    LLM->>targ: "targ c4-render"
    targ->>FS: rebuild .svg

    Note over LLM,targ: Propagation sweep
    LLM->>FS: open parent + sibling + child .json files
    LLM->>User: present per-file unified diffs<br/>(JSON edits = approval-required<br/>auto-section rebuilds = no approval)

    loop for each propagation proposal
        User-->>LLM: apply / skip / defer
        alt apply
            LLM->>FS: edit affected .json
            LLM->>targ: rebuild affected .md
        else defer
            LLM->>FS: append Drift Note to target file
        end
    end

    LLM->>targ: "targ c4-audit --file <each-modified>.json"
    targ-->>LLM: zero findings (or recorded Drift Notes)
    LLM->>User: present final diff
    User-->>LLM: approve commit
    LLM->>FS: commit
```

---

## 6. `audit [<name>]` — read-only drift report

`audit` collapsed `review` + the old `audit` into one sub-action. With `<name>`,
scoped to one file. Without, sweeps `architecture/c4/`. No edits in either mode.

```mermaid
sequenceDiagram
    actor User
    participant LLM
    participant targ
    participant FS

    User->>LLM: "/c4 audit [<name>]"

    alt scoped (<name> given)
        LLM->>targ: "targ c4-audit --file architecture/c4/c<level>-<name>.json"
        targ->>FS: load + validate spec
        targ->>FS: walk elements/properties, resolve every path-like `source`,<br/>`enforced_at`, `tested_at`
        targ->>FS: byte-compare existing .md vs fresh emit (staleness)
        targ-->>LLM: findings for this file
    else sweep (no name)
        LLM->>FS: list architecture/c4/c*-*.json
        loop per file
            LLM->>targ: "targ c4-audit --file <f>.json"
            targ-->>LLM: findings for <f>
        end
    end

    Note right of LLM: roll-up:<br/>- spec_invalid / source_path_unresolved (L1/L2/L3)<br/>- property_link_unresolved (L4)<br/>- l4_carryover (L4↔L3 element parity)<br/>- rendered_markdown_missing / _stale<br/>- registry conflicts, orphan refined_by entries

    LLM-->>User: findings or roll-up drift report (no edits)
```

---

## Notes on what these diagrams elide

- **Each `targ c4-l*-build` invocation is itself a sub-process** that reads the
  JSON spec, validates the schema, and writes the rendered `.md` + `.mmd`. Treat
  as a black box per the project rule "don't reverse-engineer targ's behavior."
- **The audit reads the `.json` spec, not the `.md`.** Passing a `.md` is rejected
  with a hint. The rendered `.md` is checked only for staleness via byte-compare
  against a fresh in-memory emit. The fresh emit reuses the SHA stamped in the
  existing front-matter so audit-time vs. build-time SHA drift doesn't false-positive.
- **The "user approves" steps are not single yes/no prompts** — propagation proposals
  come in `[a]pply / [s]kip / [d]efer` lists, one per file. Idempotent rebuilds of
  auto-generated sections (mermaid block, catalog, cross-links) do NOT require
  approval; JSON edits to non-target files DO.
- **Tier 1 / Tier 2 (Rule 7) applies at every level**, not just L4. The L1/L2/L3
  diagrams above show explicit Tier 1 dispatch boxes; in practice the ratio of
  candidates to verified findings shrinks as the level deepens (L1 has few
  externals; L4 has many properties). The principle is the same: Tier 1 owns
  recall, Tier 2 owns precision and verifies every *where* against source.
- **Drift Notes** persist across sessions; they're appended to the target file's
  `## Drift Notes` section when the user defers a propagation proposal or chooses
  to record a code/intent gap rather than resolve it.
