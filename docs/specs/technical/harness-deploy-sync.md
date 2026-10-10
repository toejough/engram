# Harness Deploy Sync - Technical Spec

## 792b29a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the harness configuration directories
  `~/.claude` and `~/.pi`).
- Responsibility: for each detected harness, keeps an engram-owned root equal to the deploy set
  computed from the source (skills, and guidance while it is managed), makes the harness's discovery
  paths links into that root, adopts earlier copies of deploy-set artifacts, removes what left the
  source and the links that dangle, deletes nothing it cannot prove it owns, and reports what it did
  and what it left alone.
- Current realization (informational): `internal/update/update.go` (`planAndApply`, `applyOps`,
  `applyForHarnessSymlinkMode`, `applyForHarnessManifestMode`, `applyEngramRootSync`,
  `planEngramRootSync`, `materializeSymlink`, `materializeOrAdopt`, `cleanupDanglingLinks`,
  `surfaceStrays`, `detectGuidanceImports`, `supportedHarnesses`), and the harness sections of
  `internal/cli/update.go` (`writeHarnessSections`, `writeEngramRootNotice`, `writeSkillRows`,
  `writeHarnessGuidanceHints`), over the injected `update.Filesystem`.
- Qualification: responsibility, converging a harness onto a deploy set without deleting foreign files
  is stated without naming any function; ownership, it owns the roots, links and findings, while the
  source choice, install and hand-off belong to [self-installer](./self-installer.md) and the guidance documents and the hints
  that activate them to [ambient-guidance](../product/ambient-guidance.md); independent verification, every step runs over an in-memory
  filesystem with injected faults; stability, the contract survives any change to the harness list or the
  deploy mode; necessity, a skill reaches a harness only through it.
- Supported capabilities: [install-and-update](../product/install-and-update.md). Skill-runbook-registration ([skill-runbook-registration](../product/skill-runbook-registration.md)) and [ambient-guidance](../product/ambient-guidance.md)
  extend it (the registration sources read the deployed skills, and the activation hints name the
  guidance it deploys).

Rationale: real files live only in one root per harness that engram alone writes, and every discovery path is a link
into it or a file the user owns. Because the only deletions are inside a root engram proved is its own and of links
that provably point into it, the sync can propagate removals without risking user files. A harness whose discovery
through links fails runs in manifest mode, which keeps the same removal semantics by recording the paths it wrote.
Claude Code's guidance directory is its engram root, so its guidance is linked at the root's top level rather than
into a separate directory.

## 792b29c Behavior Requirements

### 792b29c1 Harness detection

Requirement: [792b29c1 Harness detection](#792b29c1-harness-detection) The harness-deploy-sync MUST detect Claude Code when `~/.claude` is a directory and Pi
when `~/.pi` is a directory, MUST return the detected harnesses in the order Claude Code then Pi, and MUST
return none when neither exists. Claude Code's skills directory is `~/.claude/skills`, its guidance
directory and engram root `~/.claude/engram`, its imports file `~/.claude/CLAUDE.md`; Pi's are
`~/.pi/agent/skills`, `~/.pi/agent/guidance`, `~/.pi/agent/engram` and `~/.pi/agent/AGENTS.md`; both run in
link mode. It MUST need only a stat capability to detect, and MUST fail with an error naming the probe
when a probe fails for a reason other than not existing.

Supports: [792a11g1a Only the harnesses present are updated](../product/install-and-update.md#792a11g1-uat-scenarios), [792a11g1b Both harnesses are updated in a stable order](../product/install-and-update.md#792a11g1-uat-scenarios), [792a11g1c No harness is an error before any install](../product/install-and-update.md#792a11g1-uat-scenarios).

Rationale: the same detection is used by [skill-source-discovery](./skill-source-discovery.md) so the two never
disagree on which harnesses exist.

#### 792b29c1a Scenarios

```gherkin
Scenario: 792b29c1a1 Both harnesses are found in a stable order
  Given ~/.claude and ~/.pi both exist
  When the harnesses are detected
  Then Claude Code is first and Pi second

Scenario: 792b29c1a2 Only Claude Code
  Given only ~/.claude exists
  When the harnesses are detected
  Then only Claude Code is found

Scenario: 792b29c1a3 No harness
  Given neither exists
  When the harnesses are detected
  Then none is found

Scenario: 792b29c1a4 Detection needs only a stat capability
  Given a prober that can only stat
  When the harnesses are detected through the exported entry point
  Then they are found

Scenario: 792b29c1a5 The well-known paths
  Given the two harness specifications
  When their paths are read
  Then each holds the skills, guidance, imports-file and engram-root paths above and the link deploy mode

Scenario: 792b29c1a6 Any combination of probes gives a subset in canonical order
  Given any combination of the two probe directories
  When the harnesses are detected
  Then the result is a subset in which Claude Code never follows Pi

Scenario: 792b29c1a7 An unreadable probe is an error
  Given statting ~/.claude fails for a reason other than not existing
  When the harnesses are detected
  Then detection fails with an error naming the probe
```

### 792b29c2 The deploy set

Requirement: [792b29c2 The deploy set](#792b29c2-the-deploy-set) The harness-deploy-sync MUST plan one copy for every file under
`agent-instructions/skills/` of the source, for every harness, keeping its path relative to the skills
directory and naming its top-level skill directory, and MUST fail with an error satisfying
`update.ErrSkillsSrcMissing` and naming the directory when that source directory does not exist. It MUST
plan one copy for every `.md` file directly in `agent-instructions/guidance/` for every harness that has a
guidance directory, and none, without an error, when that source directory does not exist.

Supports: [792a11g1e A missing skills source is an error](../product/install-and-update.md#792a11g1-uat-scenarios), [792a11h1a A skill deploys as a directory link](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11l1a The flag deploys guidance](../product/install-and-update.md#792a11l1-uat-scenarios).

#### 792b29c2a Scenarios

```gherkin
Scenario: 792b29c2a1 Every skill file is planned for every harness
  Given any set of skill files
  When the skill copies are planned
  Then each file appears once per harness under that harness's skills directory

Scenario: 792b29c2a2 A missing skills source is an error
  Given no agent-instructions/skills directory
  When the skill copies are planned
  Then planning fails with ErrSkillsSrcMissing naming the directory

Scenario: 792b29c2a3 Guidance files are planned per harness
  Given guidance files recall.md and delegate.md, and a harness with a guidance directory
  When the guidance copies are planned
  Then each file is planned once for that harness

Scenario: 792b29c2a4 A missing guidance source plans nothing
  Given no agent-instructions/guidance directory
  When the guidance copies are planned
  Then no copy is planned and there is no error
```

### 792b29c3 Guidance management

Requirement: [792b29c3 Guidance management](#792b29c3-guidance-management) The harness-deploy-sync MUST manage guidance, so that the guidance copies are in the
deploy set and the root's `guidance/` subtree is synced including deletions, only when `--with-guidance`
was given or at least one harness imports at least one guidance file ([792b29c4 Import detection](#792b29c4-import-detection)), and MUST otherwise plan no
guidance and leave the `guidance/` subtree and its links untouched and unreported. When it manages
guidance, it MUST deploy every guidance file for every harness, including those not imported, and MUST add a
guidance file that joins the source after the first sync on the next sync. A failure to deploy a guidance
file MUST be an error of that harness.

Supports: [792a11l1a The flag deploys guidance](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11l1b An existing import keeps guidance current](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11l1c Without opt-in guidance is untouched](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11l1e A guidance document removed from the source is deleted once managed](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11l1g One harness's import manages guidance for every harness](../product/install-and-update.md#792a11l1-uat-scenarios).

Rationale: opting in once is enough because the import in the user's config file is the standing record of
it, so a plain update keeps the guidance current as it does skills. A harness that is not opted in keeps
whatever it holds: not managed is not removed.

#### 792b29c3a Scenarios

```gherkin
Scenario: 792b29c3a1 The flag deploys guidance
  Given the source holds guidance recall.md and --with-guidance is given
  When the update runs
  Then ~/.claude/engram/guidance/recall.md holds the source content

Scenario: 792b29c3a2 An import keeps guidance current
  Given CLAUDE.md imports one guidance file and no flag is given
  When the update runs
  Then every guidance file is refreshed

Scenario: 792b29c3a3 A single imported file refreshes all
  Given CLAUDE.md imports only delegate.md and the source also holds recall.md
  When the update runs
  Then both files are deployed

Scenario: 792b29c3a4 Without opt-in guidance is skipped and its subtree untouched
  Given no flag, no import, and a guidance/stale.md left by an earlier opted-in sync
  When the update runs
  Then no guidance is deployed and stale.md remains, unreported

Scenario: 792b29c3a5 A new guidance file joins on a later sync
  Given guidance was deployed earlier and the source now holds one more file
  When the update runs
  Then the new file is deployed and the earlier ones remain

Scenario: 792b29c3a6 A deploy failure is a harness error
  Given writing a guidance file fails
  When the update runs
  Then that harness reports the error

Scenario: 792b29c3a7 A removed guidance file is deleted once managed
  Given guidance is managed and guidance/stale.md is not in the source
  When the update runs
  Then stale.md is deleted and reported as a deletion

Scenario: 792b29c3a8 One harness's import manages guidance for every harness
  Given only Claude Code's config file imports a guidance file, no flag, and both harnesses present
  When the update runs
  Then guidance is deployed for Pi as well as for Claude Code
```

### 792b29c4 Import detection

Requirement: [792b29c4 Import detection](#792b29c4-import-detection) For each detected harness that has an imports file, the harness-deploy-sync MUST read
that file and MUST record as imported each line, outside a fenced code block (a line beginning with three
backticks toggles the fence, and an unclosed fence hides the rest), that after trimming is exactly the
prefix `@~/<guidance dir>/` or `@<home>/<guidance dir>/` followed by a single name that ends in `.md` and
has no `/`. The guidance directory of Claude Code is `.claude/engram` and of Pi `.pi/agent/guidance`.
It MUST attribute an import only to the harness whose prefix it carries, MUST find none when the file is
missing, and MUST derive both prefixes from the harness specification.

Supports: [792a11l1b An existing import keeps guidance current](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11l1d An import inside a code fence does not count](../product/install-and-update.md#792a11l1-uat-scenarios).

#### 792b29c4a Scenarios

```gherkin
Scenario: 792b29c4a1 An import is detected
  Given CLAUDE.md holds @~/.claude/engram/recall.md, then one that holds none
  When imports are detected
  Then the first reports guidance imported and the second does not

Scenario: 792b29c4a2 A missing config file imports nothing
  Given no CLAUDE.md
  When imports are detected
  Then none is found and there is no error

Scenario: 792b29c4a3 The set is per file
  Given an imports file naming recall.md and delegate.md
  When imports are detected
  Then both basenames are recorded for that harness

Scenario: 792b29c4a4 An import is attributed to its own harness
  Given CLAUDE.md imports ~/.claude/engram/recall.md and AGENTS.md imports ~/.pi/agent/guidance/delegate.md, or either file imports the other harness's path
  When imports are detected
  Then each harness records only the files its own prefix names

Scenario: 792b29c4a5 A fence hides imports
  Given an import line inside a closed fence, or after an unclosed fence
  When imports are detected
  Then it is not recorded

Scenario: 792b29c4a6 Stale and foreign prefixes do not count
  Given the expanded-home form of a Pi import, which counts, and ~/.pi/agent/engram/recall.md, which does not
  When imports are detected
  Then only the first is recorded

Scenario: 792b29c4a7 Attribution holds for any combination
  Given any files in either harness's imports file
  When imports are detected
  Then each harness's recorded set is exactly the set of its own valid import lines

Scenario: 792b29c4a8 Prefixes come from the specification
  Given every supported harness
  When its import prefixes are derived
  Then they are built from its guidance directory in the tilde and expanded forms
```

### 792b29c5 Root ownership

Requirement: [792b29c5 Root ownership](#792b29c5-root-ownership) The harness-deploy-sync MUST give each detected harness its own engram-owned root
(`~/.claude/engram`, `~/.pi/agent/engram`). When the root does not exist it MUST create the root and the
empty marker file `.engram-owned` and sync with deletion allowed. When the root holds the marker it MUST
sync with deletion allowed. When the root exists without the marker it MUST write the marker and the
deploy set, MUST delete nothing in it in that run, and MUST report the root as refused and each
top-level file whose name matches no deploy-set name as unattributable (a Claude Code flat guidance file
whose name is in the deploy set is adoptable, not unattributable). It MUST fail the harness with
`update.ErrEngramRootNotDir` when the root path is not a directory, and with the underlying error when the
root, the marker or the top-level listing cannot be read. On the next run it MUST treat an adopted root as
owned.

Supports: [792a11j1a An unmarked root is not purged](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11j1b The marker enables deletion on the next run](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11j1e A root that is a file is refused](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11k1a A copied skill becomes a link](../product/install-and-update.md#792a11k1-uat-scenarios).

Rationale: the marker, not the path, proves a directory named `engram/` is engram's. A user who deletes the
marker degrades to refused deletion, which is a fail-safe and not a failure.

#### 792b29c5a Scenarios

```gherkin
Scenario: 792b29c5a1 An absent root is created with its marker
  Given no engram root
  When the update runs
  Then the root, the marker and the skill files exist, with no refusal and nothing unattributable

Scenario: 792b29c5a2 Every harness has its own root
  Given both harnesses
  When the update runs
  Then each report names its own root, ~/.claude/engram and ~/.pi/agent/engram

Scenario: 792b29c5a3 A marked root syncs with deletion
  Given a root with the marker holding a stale skill file and directory
  When the update runs
  Then the stale file and its emptied directory are deleted and reported, and the skill is overwritten

Scenario: 792b29c5a4 An unmarked root refuses deletion
  Given a root without the marker holding mystery.md
  When the update runs
  Then mystery.md remains, the report lists it as unattributable and the root as refused, the marker is written, and the skill is created

Scenario: 792b29c5a5 A flat guidance file in the deploy set is adoptable
  Given an unmarked Claude Code root holding recall.md that is in the deploy set
  When the update runs
  Then it is not unattributable, it is replaced by a link to guidance/recall.md and reported as adopted, and the marker is written

Scenario: 792b29c5a6 A root that is a file is an error
  Given the root path is a regular file
  When the update runs
  Then that harness fails with ErrEngramRootNotDir and the file is untouched

Scenario: 792b29c5a7 An unreadable root or marker is an error
  Given statting the root, or the marker, fails with an error other than not existing
  When the update runs
  Then that harness reports the error

Scenario: 792b29c5a8 An unreadable top level is an error
  Given an unmarked root whose listing fails
  When the update runs
  Then that harness reports the error

Scenario: 792b29c5a9 The marker is stamped after the adoption pass and then enables deletion
  Given an unmarked root and a copied skill
  When the update runs, then runs again after a stale file appears in a managed subtree
  Then the first run writes the marker after adopting, and the second treats the root as owned and deletes the stale file
```

### 792b29c6 Root synchronization

Requirement: [792b29c6 Root synchronization](#792b29c6-root-synchronization) The harness-deploy-sync MUST sync a root to the deploy set with deletion allowed by
creating each deploy-set file that is missing, overwriting each that exists, and deleting each file in the
managed subtrees (`skills/`, `commands/`, and `guidance/` while guidance is managed) that is not in the
deploy set, in path order, and MUST then delete each directory in those subtrees that holds nothing the
deploy set keeps (reporting only the outermost). It MUST touch nothing outside the managed subtrees.
Planning MUST run in a dry run so the report shows the deletions and pruned directories, but nothing is
written or removed. It MUST fail the harness with the underlying error when creating the root or marker,
creating a directory, reading a source file, writing a file, removing a path or listing a subtree fails, and
the failure MUST NOT touch any other harness.

Supports: [792a11h1a A skill deploys as a directory link](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11h1b Added and changed sources deploy](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11i1a A removed skill disappears](../product/install-and-update.md#792a11i1-uat-scenarios), [792a11i1b A removed file inside a kept skill disappears](../product/install-and-update.md#792a11i1-uat-scenarios).

Rationale: a directory holding no intended file is as unintended as a file, and leaving it as a hollow shell would let
a stale link still resolve through it, so the link would look healthy while the skill was gone. `commands/` remains a
managed subtree though engram deploys no commands, so anything found there is deleted.

#### 792b29c6a Scenarios

```gherkin
Scenario: 792b29c6a1 The root equals the deploy set
  Given any deploy set and any pre-existing files inside and outside the managed subtrees
  When the root is planned, applied and pruned
  Then the managed subtrees hold exactly the deploy set, no managed directory is empty, and nothing outside the root changed

Scenario: 792b29c6a2 A removed skill disappears with its link in one run
  Given a skill synced earlier, then deleted from the source
  When the update runs
  Then its file, its emptied directory and its link in the skills directory are all gone after that single run

Scenario: 792b29c6a3 A dry run previews a prune without removing
  Given a stale skill directory in the root and a dry run
  When the update runs
  Then the report lists the stale file and directory as deleted, and the files and directory remain

Scenario: 792b29c6a4 A directory that cannot be created fails the harness
  Given creating a directory under the root fails
  When the root is applied
  Then the error names the failure

Scenario: 792b29c6a5 A source file that cannot be read fails the harness
  Given reading a source file fails
  When the root is applied
  Then the error names the failure

Scenario: 792b29c6a6 A file that cannot be written fails the harness
  Given writing a file fails
  When the root is applied
  Then the error names the failure

Scenario: 792b29c6a7 A path that cannot be removed fails the harness
  Given removing a stale path fails
  When the root is applied
  Then the error names the failure

Scenario: 792b29c6a8 A marker that cannot be written fails the harness
  Given writing the marker fails
  When the update runs
  Then the harness reports the error

Scenario: 792b29c6a9 A root that cannot be created fails the harness
  Given creating the root fails
  When the update runs
  Then the harness reports the error

Scenario: 792b29c6a10 A write failure during the sync fails the harness
  Given writing a skill file under the root fails
  When the update runs
  Then the harness reports the error

Scenario: 792b29c6a11 A subtree that cannot be listed fails the harness
  Given listing a managed subtree fails
  When the update runs
  Then the harness reports the error
```

### 792b29c7 Skill links

Requirement: [792b29c7 Skill links](#792b29c7-skill-links) For each skill directory of the deploy set, the harness-deploy-sync MUST make
`<skills dir>/<skill>` a symbolic link to `<engram root>/skills/<skill>`: creating it, with its parent
directory, when absent; leaving it when it already points there; repointing it, by removing and re-creating
it, when it points elsewhere; and leaving a real file or directory there untouched and reporting that it is
blocked ([792b29c8 Adopting earlier copies](#792b29c8-adopting-earlier-copies) decides what follows). In a dry run it MUST write nothing in any of these cases. It MUST
fail the harness with the underlying error when it cannot inspect, create, read, remove or link the path.

Supports: [792a11h1a A skill deploys as a directory link](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11h1c A link that points elsewhere is repointed](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11j1d User-owned real files are never deleted](../product/install-and-update.md#792a11j1-uat-scenarios).

#### 792b29c7a Scenarios

```gherkin
Scenario: 792b29c7a1 An absent link is created
  Given no entry at the link path
  When the link is materialized
  Then a link to the target exists and the link is not blocked

Scenario: 792b29c7a2 An absent link is not created in a dry run
  Given no entry at the link path and a dry run
  When the link is materialized
  Then nothing is written

Scenario: 792b29c7a3 A correct link is left alone
  Given a link already pointing at the target
  When the link is materialized
  Then nothing changes

Scenario: 792b29c7a4 A wrong link is repointed
  Given a link pointing elsewhere
  When the link is materialized
  Then it points at the target

Scenario: 792b29c7a5 A wrong link is not repointed in a dry run
  Given a link pointing elsewhere and a dry run
  When the link is materialized
  Then it still points elsewhere

Scenario: 792b29c7a6 A real directory is untouched and blocked
  Given a real directory at the link path
  When the link is materialized
  Then it is left in place and the result says the path is blocked

Scenario: 792b29c7a7 A real file is untouched and blocked
  Given a real file at the link path
  When the link is materialized
  Then it is left in place and the result says the path is blocked

Scenario: 792b29c7a8 An inspection failure is an error
  Given inspecting the link path fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a9 A parent that cannot be created is an error
  Given creating the parent directory fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a10 A link that cannot be read is an error
  Given reading an existing link fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a11 A repoint that cannot remove the old link is an error
  Given removing the wrong link fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a12 A repoint that cannot re-create the link is an error
  Given creating the link after removing the wrong one fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a13 A link that cannot be created is an error
  Given creating the link fails
  When the link is materialized
  Then the error is returned

Scenario: 792b29c7a14 A wrong link is repointed by an update
  Given a skill whose link in the skills directory points at another engram path
  When the update runs
  Then the link points at the engram root's skill directory

Scenario: 792b29c7a15 A repoint failure fails that harness
  Given removing the wrong link fails during an update
  When the update runs
  Then the harness reports the removal error
```

### 792b29c8 Adopting earlier copies

Requirement: [792b29c8 Adopting earlier copies](#792b29c8-adopting-earlier-copies) Where a deploy-set skill directory, Pi guidance file or Claude Code flat guidance
file is a real file or directory at its link path, the harness-deploy-sync MUST replace it, by removing it
and creating the link, and MUST report the path as adopted, without comparing its content; where the link
is absent or already correct it MUST report nothing adopted. In a dry run it MUST report the path as adopted
and change nothing. It MUST fail the harness when removing the real copy or creating the link fails.

Supports: [792a11k1a A copied skill becomes a link](../product/install-and-update.md#792a11k1-uat-scenarios), [792a11j1d User-owned real files are never deleted](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11m1b A real flat file from an earlier version is adopted](../product/install-and-update.md#792a11m1-uat-scenarios).

Rationale: the repository is the source of truth, so no comparison is needed, and the deploy set is the
only thing that can be replaced: a real entry outside it is never adopted ([792b29c11 Unmanaged entries](#792b29c11-unmanaged-entries)).

#### 792b29c8a Scenarios

```gherkin
Scenario: 792b29c8a1 An absent link is created, not adopted
  Given no entry at the link path
  When it is materialized or adopted
  Then the link exists and nothing is reported adopted

Scenario: 792b29c8a2 A correct link is not adopted
  Given a link already pointing at the target
  When it is materialized or adopted
  Then nothing is reported adopted

Scenario: 792b29c8a3 A real directory is replaced
  Given a real directory with stale content at the link path
  When it is adopted
  Then the directory is gone, a link to the target exists, and the path is reported adopted

Scenario: 792b29c8a4 A real file is replaced
  Given a real file at the link path
  When it is adopted
  Then it is replaced by a link and reported adopted

Scenario: 792b29c8a5 A dry run previews the adoption
  Given a real directory at the link path and a dry run
  When it is adopted
  Then the path is reported adopted and nothing is written

Scenario: 792b29c8a6 A copy that cannot be removed is an error
  Given removing the real copy fails
  When it is adopted
  Then the error is returned

Scenario: 792b29c8a7 A link that cannot be created is an error
  Given creating the link after the removal fails
  When it is adopted
  Then the error is returned

Scenario: 792b29c8a8 A copied skill directory is adopted by an update
  Given a harness whose skills directory holds a deploy-set skill as a real directory with stale content
  When the update runs
  Then the stale content is gone, the link points at the root, the path is in the adopted list and the unmanaged list is empty

Scenario: 792b29c8a9 A copied skill is adopted on the first sync
  Given the first sync of a harness whose skills directory holds recall as a copy
  When the update runs
  Then recall's content lives in the root and the path is a link to it

Scenario: 792b29c8a10 A real Pi guidance file is adopted
  Given a real guidance file in Pi's guidance directory that is in the deploy set
  When the update runs with guidance managed
  Then it is replaced by a link into Pi's engram root

Scenario: 792b29c8a11 A real flat Claude guidance file is adopted
  Given a real recall.md at the top of Claude Code's engram root
  When the update runs with guidance managed
  Then it is replaced by a link to the canonical copy and reported adopted
```

### 792b29c9 Guidance links

Requirement: [792b29c9 Guidance links](#792b29c9-guidance-links) For a harness whose guidance directory is its engram root (Claude Code), the
harness-deploy-sync MUST link `<root>/<name>.md` to `<root>/guidance/<name>.md` for each managed guidance
file, repointing or adopting as in [792b29c7 Skill links](#792b29c7-skill-links) and [792b29c8 Adopting earlier copies](#792b29c8-adopting-earlier-copies), and MUST delete such a link, by the cleanup of
[792b29c10 Dangling-link cleanup](#792b29c10-dangling-link-cleanup), when its file leaves the deploy set. For a harness with a separate guidance directory (Pi) it
MUST link `<guidance dir>/<name>.md` to `<root>/guidance/<name>.md` for each of that harness's own
guidance files only, and MUST fail the harness when linking fails. It MUST record each linked file as a
guidance file of the harness.

Supports: [792a11l1f Pi guidance is linked into Pi's guidance directory](../product/install-and-update.md#792a11l1-uat-scenarios), [792a11m1a An existing import still resolves](../product/install-and-update.md#792a11m1-uat-scenarios), [792a11m1b A real flat file from an earlier version is adopted](../product/install-and-update.md#792a11m1-uat-scenarios), [792a11m1c A link whose document left the source is removed](../product/install-and-update.md#792a11m1-uat-scenarios).

#### 792b29c9a Scenarios

```gherkin
Scenario: 792b29c9a1 Claude Code gets compat links at the top of its root
  Given guidance is managed for Claude Code
  When the update runs
  Then ~/.claude/engram/recall.md is a link to guidance/recall.md in the root, not a copy

Scenario: 792b29c9a2 Pi gets one link per file in its guidance directory
  Given guidance is managed for both harnesses
  When the update runs
  Then ~/.pi/agent/guidance/recall.md links into Pi's root, and Claude Code's files are not linked there

Scenario: 792b29c9a3 A Pi link failure fails Pi
  Given inspecting the link path in Pi's guidance directory fails
  When the update runs
  Then Pi reports the error

Scenario: 792b29c9a4 A compat link is deleted when its file leaves the source
  Given a Claude Code compat link and its guidance file, then the file is removed from the source
  When the update runs
  Then the link is deleted
```

### 792b29c10 Dangling-link cleanup

Requirement: [792b29c10 Dangling-link cleanup](#792b29c10-dangling-link-cleanup) On every run, the harness-deploy-sync MUST scan, one level deep, the skills
directory and, for a harness with guidance, either its guidance directory (Pi) or its engram root (Claude
Code), and MUST delete exactly the symbolic links whose literal target, resolved lexically against the
link's own directory (never by following links), lies within the harness's engram root and does not exist,
recording each. It MUST NOT touch a real file or directory, a healthy link, or a link whose target lies
outside the root, whether or not that target exists. A path is within the root when it equals the root or
lies beneath it by whole path segments (a sibling that merely shares the prefix is outside). In a dry run
it MUST record without removing. It MUST scan nothing when given no directory or a missing one, and MUST
fail the harness with the underlying error when listing, inspecting, reading or removing fails.

Supports: [792a11j1c A dangling engram link is removed and a foreign link kept](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11j1d User-owned real files are never deleted](../product/install-and-update.md#792a11j1-uat-scenarios).

Rationale: lexical resolution is essential: `filepath.EvalSymlinks` on macOS rewrites `/var` to
`/private/var`, which would corrupt the prefix match even for a tree with no links, so every recorded path
must be the exact logical spelling.

#### 792b29c10a Scenarios

```gherkin
Scenario: 792b29c10a1 A dangling engram link is removed
  Given a link into the root whose target is missing
  When the directory is cleaned
  Then it is removed and recorded

Scenario: 792b29c10a2 A healthy engram link is kept
  Given a link into the root whose target exists
  When the directory is cleaned
  Then it is kept

Scenario: 792b29c10a3 A foreign link with a missing target is untouched
  Given a link outside the root whose target is missing
  When the directory is cleaned
  Then it is untouched

Scenario: 792b29c10a4 A foreign link with a present target is untouched
  Given a link outside the root whose target exists
  When the directory is cleaned
  Then it is untouched

Scenario: 792b29c10a5 A real directory is untouched
  Given a real directory
  When the directory is cleaned
  Then it is untouched

Scenario: 792b29c10a6 A real file is untouched
  Given a real file
  When the directory is cleaned
  Then it is untouched

Scenario: 792b29c10a7 A relative dangling target resolves lexically
  Given a relative link that resolves into the root and is missing
  When the directory is cleaned
  Then it is removed

Scenario: 792b29c10a8 A relative healthy target resolves lexically
  Given a relative link that resolves into the root and exists
  When the directory is cleaned
  Then it is kept

Scenario: 792b29c10a9 A dry run records without removing
  Given a dangling engram link and a dry run
  When the directory is cleaned
  Then it is recorded and still exists

Scenario: 792b29c10a10 No directory is a no-op
  Given an empty directory argument
  When the directory is cleaned
  Then nothing is scanned

Scenario: 792b29c10a11 A missing directory is a no-op
  Given a directory that does not exist
  When the directory is cleaned
  Then nothing is removed and there is no error

Scenario: 792b29c10a12 Only root-matched dangling links are removed
  Given any mix of dangling and healthy engram links, foreign links, real files and real directories
  When the directory is cleaned
  Then exactly the dangling engram links are removed

Scenario: 792b29c10a13 A listing failure is an error
  Given listing the directory fails
  When the directory is cleaned
  Then the error is returned

Scenario: 792b29c10a14 An inspection failure is an error
  Given inspecting an entry fails
  When the directory is cleaned
  Then the error is returned

Scenario: 792b29c10a15 A link that cannot be read is an error
  Given reading a link fails
  When the directory is cleaned
  Then the error is returned

Scenario: 792b29c10a16 A target that cannot be statted is an error
  Given statting an engram link's target fails for a reason other than not existing
  When the directory is cleaned
  Then the error is returned

Scenario: 792b29c10a17 A removal failure is an error
  Given removing a dangling link fails
  When the directory is cleaned
  Then the error is returned

Scenario: 792b29c10a18 A harness without guidance scans only its skills directory
  Given a harness with no guidance directory
  When its links are cleaned
  Then only the skills directory is scanned

Scenario: 792b29c10a19 A scan failure ends the harness
  Given scanning a directory fails
  When its links are cleaned
  Then the harness reports the error and no further directory is scanned

Scenario: 792b29c10a20 A link target is resolved lexically
  Given an absolute target, a relative target and a target in the link's own directory
  When each is resolved against the link's directory
  Then the absolute path is kept and the relative ones are joined to the directory and cleaned

Scenario: 792b29c10a21 A path is within the root by whole segments
  Given the root itself, a nested path, an outside path and a sibling sharing the root's prefix
  When each is tested against the root
  Then the first two are within and the last two are not

Scenario: 792b29c10a22 A dangling link is removed by an update
  Given a skill whose source artifact is gone, so its link now dangles
  When the update runs
  Then the link is removed in that run

Scenario: 792b29c10a23 A foreign link is never touched by an update
  Given a link in the skills directory to a path outside the root
  When the update runs
  Then it is untouched

Scenario: 792b29c10a24 A tree with no links records only exact logical paths
  Given a surface holding only real pre-migration content
  When the update runs
  Then every recorded path is the exact joined and cleaned spelling, with no rewritten prefix
```

### 792b29c11 Unmanaged entries

Requirement: [792b29c11 Unmanaged entries](#792b29c11-unmanaged-entries) On the first sync of a harness, the harness-deploy-sync MUST scan, one level deep,
its skills directory and, for a harness whose guidance directory is separate from its root, its guidance
directory, and MUST report each real (non-link) entry whose name is not in the harness's
deploy set as unmanaged, never changing it, sorted by path within each directory (the skills directory first). It MUST skip every link, report nothing for a missing
directory, and fail the harness when listing or inspecting fails. It MUST NOT scan on a later run.

Supports: [792a11k1b An unmanaged entry is reported, not deleted](../product/install-and-update.md#792a11k1-uat-scenarios), [792a11k1c The unmanaged list is not repeated](../product/install-and-update.md#792a11k1-uat-scenarios).

Rationale: a link is the cleanup's concern, and an entry the sync has no record of writing may be the user's own, so
it is listed neutrally ("unmanaged, left alone") and never as debris.

#### 792b29c11a Scenarios

```gherkin
Scenario: 792b29c11a1 An entry in the deploy set is not reported
  Given a real entry whose name is in the deploy set
  When strays are scanned
  Then it is not reported

Scenario: 792b29c11a2 An inspection failure is an error
  Given inspecting an entry fails
  When strays are scanned
  Then the error is returned

Scenario: 792b29c11a3 A missing directory has none
  Given a directory that does not exist
  When strays are scanned
  Then none is reported and there is no error

Scenario: 792b29c11a4 A listing failure is an error
  Given listing the directory fails
  When strays are scanned
  Then the error is returned

Scenario: 792b29c11a5 A real entry outside the deploy set is reported
  Given a real directory not in the deploy set
  When strays are scanned
  Then it is reported

Scenario: 792b29c11a6 A link is skipped
  Given a link whose name is not in the deploy set
  When strays are scanned
  Then it is not reported

Scenario: 792b29c11a7 An unmanaged entry is reported and never deleted
  Given the first sync of a harness with an unrelated real skill directory
  When the update runs
  Then the report lists it as unmanaged, it still exists, and it is not in the adopted list
```

### 792b29c12 First sync without a trace

Requirement: [792b29c12 First sync without a trace](#792b29c12-first-sync-without-a-trace) In a dry run the harness-deploy-sync MUST plan every first-sync step (the root, the
marker, the sync, the adoptions and the links) without creating the root, the marker, a link or a file, both
for a harness never synced and for one whose root exists without the marker.

Supports: [792a11f1a A dry run on a never-synced harness leaves no trace](../product/install-and-update.md#792a11f1-uat-scenarios).

#### 792b29c12a Scenarios

```gherkin
Scenario: 792b29c12a1 A never-synced harness leaves no trace
  Given a harness that has never been synced and a dry run
  When the update runs
  Then the report lists the planned steps, and no root, marker, link or file exists afterward

Scenario: 792b29c12a2 A pre-existing unmarked root leaves no trace
  Given a root without the marker and a dry run
  When the update runs
  Then no marker is written and no file is changed
```

### 792b29c13 Manifest mode

Requirement: [792b29c13 Manifest mode](#792b29c13-manifest-mode) For a harness in manifest mode, the harness-deploy-sync MUST copy each skill file
and each guidance file to its harness path, clearing a skill directory once before writing its files,
MUST record every path it wrote in `.engram-manifest.json` in the harness's root, and on a later run MUST
delete exactly the recorded paths that are no longer in the deploy set, drop them from the manifest, and
leave every unrecorded file alone. A missing manifest MUST be empty and an undecodable one MUST be an error.
A dry run MUST preview the deletions and write nothing. A failure to clear, read, create a directory,
write or delete MUST fail that harness. The root sync ([792b29c6 Root synchronization](#792b29c6-root-synchronization)) still runs last. No supported harness
uses this mode.

Supports: [792a11i1a A removed skill disappears](../product/install-and-update.md#792a11i1-uat-scenarios).

Rationale: a harness whose discovery fails the link check must still get removals that propagate; recording
what was written replaces link identity as the proof of ownership.

#### 792b29c13a Scenarios

```gherkin
Scenario: 792b29c13a1 Files are copied and recorded
  Given a manifest-mode harness and a skill and a guidance file
  When the update runs
  Then the files exist at their harness paths, no link is created, and the manifest records them

Scenario: 792b29c13a2 A manifest-mode update copies as before
  Given a manifest-mode Claude Code harness and a Pi harness not present
  When the update runs
  Then the skill files and the guidance file are written under Claude Code's paths, the guidance list holds the file, and nothing is written for Pi

Scenario: 792b29c13a3 An obsolete recorded file is deleted on the next run
  Given a recorded file whose source was deleted
  When the update runs again
  Then the file is deleted and dropped from the manifest

Scenario: 792b29c13a4 Removal propagates through the manifest
  Given a deployed file recorded in the manifest and its source deleted
  When the manifest deletion runs
  Then the file is deleted and unrecorded files in the same directory survive

Scenario: 792b29c13a5 Unrecorded files survive
  Given a file in a harness directory that the manifest does not record
  When the manifest deletion runs
  Then it is not deleted

Scenario: 792b29c13a6 Only recorded paths are deleted
  Given any mix of recorded and unrecorded files
  When the manifest deletion runs
  Then exactly the recorded files absent from the deploy set are deleted

Scenario: 792b29c13a7 A missing manifest is empty
  Given no manifest
  When the manifest deletion runs
  Then nothing is deleted and there is no error

Scenario: 792b29c13a8 A corrupt manifest is an error
  Given a manifest that is not valid JSON
  When the update runs
  Then that harness reports an error

Scenario: 792b29c13a9 A deletion failure is a harness error
  Given deleting a recorded path fails
  When the update runs
  Then that harness reports the error

Scenario: 792b29c13a10 A dry run previews and writes nothing
  Given recorded obsolete files and a dry run
  When the update runs
  Then the deletions are previewed and no file or manifest is written or deleted

Scenario: 792b29c13a11 A clear, copy, create or write failure is a harness error
  Given clearing a skill directory, reading a source, creating a directory, writing a skill file or writing a guidance file fails
  When the update runs
  Then that harness reports the error
```

### 792b29c14 Dry run

Requirement: [792b29c14 Dry run](#792b29c14-dry-run) In a dry run the harness-deploy-sync MUST create, write, link, repoint, adopt and
delete nothing, in the root, the marker and the harness directories, while still reporting what it would
do (the root deletions, the adoptions and the dangling-link removals).

Supports: [792a11f1a A dry run on a never-synced harness leaves no trace](../product/install-and-update.md#792a11f1-uat-scenarios), [792a11f1b Action lines carry the prefix, notices do not](../product/install-and-update.md#792a11f1-uat-scenarios).

#### 792b29c14a Scenarios

```gherkin
Scenario: 792b29c14a1 The root sync writes nothing
  Given a harness with no root and a dry run
  When the update runs
  Then no file is written and the root does not exist, and the report names the root

Scenario: 792b29c14a2 No link is created
  Given skills to link and a dry run
  When the update runs
  Then no link is created

Scenario: 792b29c14a3 A dry-run deploy plans the skills
  Given a source with one skill and a dry run
  When the update runs
  Then the report counts the skill's files and nothing is written
```

### 792b29c15 Harness isolation

Requirement: [792b29c15 Harness isolation](#792b29c15-harness-isolation) The harness-deploy-sync MUST record a failure of one harness's deploy on that
harness's report, MUST still deploy every other harness, and MUST NOT turn the failure into an error of
the whole run.

Supports: [792a11g1d One failing harness does not stop the other](../product/install-and-update.md#792a11g1-uat-scenarios).

#### 792b29c15a Scenarios

```gherkin
Scenario: 792b29c15a1 A failing harness is reported and the run continues
  Given the only harness cannot be written
  When the update runs
  Then the run returns no error and the harness's report holds the failure

Scenario: 792b29c15a2 The failure does not stop the other harness
  Given Claude Code fails and Pi is healthy
  When the update runs
  Then Pi is deployed
```

### 792b29c16 Idempotence

Requirement: [792b29c16 Idempotence](#792b29c16-idempotence) Running the harness-deploy-sync twice over the same source MUST leave the same files
and links after the second run as after the first, whether or not the harness held earlier content.

Supports: [792a11h1e Re-running changes nothing](../product/install-and-update.md#792a11h1-uat-scenarios).

#### 792b29c16a Scenarios

```gherkin
Scenario: 792b29c16a1 A second run changes nothing
  Given any skill files, with or without a pre-existing copy of one
  When the update runs twice
  Then the destination files are byte-identical after the second run

Scenario: 792b29c16a2 A second run through the report tail succeeds and changes nothing
  Given a local source and a first run that deployed it
  When the update runs again and finishes
  Then both runs succeed and the destination is the same
```

### 792b29c17 Harness sections of the report

Requirement: [792b29c17 Harness sections of the report](#792b29c17-harness-sections-of-the-report) For each harness the harness-deploy-sync MUST give the report: one line per skill,
`agent-instructions/skills/<skill>/ → <skills dir>/<skill>/  (<N> file[s])`, with `file` for 1 and `files`
otherwise, paths under the home shown as `~/…`; then one line `engram root: deleted <path>` for each sync
deletion; the line `engram root <root> exists without the .engram-owned marker — sync-deletion refused
there` when deletion was refused; one `unattributable: <name>` per unattributable file; `adopted: <path>` per
adoption; `unmanaged (left alone): <path>` per unmanaged entry; and `cleanup: removed dangling link <path>`
per removal. It MUST print nothing for a harness with nothing to report, MUST prefix `[dry-run] ` to the
deletion, adoption and cleanup lines of a dry run, and MUST NOT prefix the refusal, unattributable,
unmanaged and skill lines. For Claude Code, whose guidance directory is its root, each guidance line MUST
end with `(canonical: ~/.claude/engram/guidance/<name>)`, and Pi's MUST NOT.

Supports: [792a11f1b Action lines carry the prefix, notices do not](../product/install-and-update.md#792a11f1-uat-scenarios), [792a11g1d One failing harness does not stop the other](../product/install-and-update.md#792a11g1-uat-scenarios), [792a11h1d The report lists each skill and its file count](../product/install-and-update.md#792a11h1-uat-scenarios), [792a11i1a A removed skill disappears](../product/install-and-update.md#792a11i1-uat-scenarios), [792a11j1a An unmarked root is not purged](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11j1c A dangling engram link is removed and a foreign link kept](../product/install-and-update.md#792a11j1-uat-scenarios), [792a11k1a A copied skill becomes a link](../product/install-and-update.md#792a11k1-uat-scenarios), [792a11k1b An unmanaged entry is reported, not deleted](../product/install-and-update.md#792a11k1-uat-scenarios), [792a11m1d The report states the canonical path](../product/install-and-update.md#792a11m1-uat-scenarios).

#### 792b29c17a Scenarios

```gherkin
Scenario: 792b29c17a1 Deletions, the refusal and unattributable files are reported
  Given a harness report with a deleted skill file, a refused root and an unattributable mystery.md
  When the report is written
  Then it holds "engram root: deleted skills/oldskill/SKILL.md", "engram root ~/.claude/engram exists without the .engram-owned marker" and "unattributable: mystery.md"

Scenario: 792b29c17a2 Adoptions and unmanaged entries are reported
  Given a harness report with an adopted skill and an unmanaged entry
  When the report is written
  Then it holds "adopted: ~/.claude/skills/recall" and "unmanaged (left alone): ~/.claude/skills/user-own-tool"

Scenario: 792b29c17a3 A dry run prefixes the adoption
  Given the same adoption in a dry run
  When the report is written
  Then the line is "[dry-run] adopted: ~/.claude/skills/recall"

Scenario: 792b29c17a4 A dry run prefixes the deletion
  Given a deletion in a dry run
  When the report is written
  Then the line is "[dry-run] engram root: deleted guidance/stale.md"

Scenario: 792b29c17a5 A harness with nothing to report is silent
  Given a harness report with no findings
  When the report is written
  Then it holds no "engram root" or "unattributable" line

Scenario: 792b29c17a6 Claude Code guidance states its canonical path
  Given a Claude Code report with guidance recall.md
  When the report is written
  Then the guidance line ends "(canonical: ~/.claude/engram/guidance/recall.md)"

Scenario: 792b29c17a7 Pi guidance states no canonical path
  Given a Pi report with guidance recall.md
  When the report is written
  Then no line holds "canonical:"

Scenario: 792b29c17a8 File counts are singular or plural
  Given skills of 1 file and of 3 files
  When the rows are written
  Then they read "(1 file)" and "(3 files)"

Scenario: 792b29c17a9 The skill rows map source to skills directory
  Given skills learn with 3 files and recall with 1 for each harness
  When the report is written
  Then each row reads "agent-instructions/skills/<skill>/ → ~/<harness skills dir>/<skill>/  (<N> file[s])"
```

### 792b29c18 Guidance activation hints

Requirement: [792b29c18 Guidance activation hints](#792b29c18-guidance-activation-hints) For each harness that has an imports file, and for each guidance file in its deploy set,
the harness-deploy-sync MUST give the report one line: `guidance refreshed: ~/<guidance dir>/<name>` when that
harness's imports include the file, and otherwise `guidance deployed to ~/<guidance dir>/<name> — add
'@~/<guidance dir>/<name>' to ~/<imports file> to activate it (<Harness> will ask you to approve the import once)`,
using forward slashes on every operating system and judging each harness and each file on its own imports. It
MUST give nothing for a harness without an imports file. When no harness deployed any guidance, the run was not
given `--with-guidance` and no config file imports guidance, it MUST give the one line `engram ships recall- and
delegation-firing guidance; run 'engram update --with-guidance' to deploy it`, and otherwise MUST NOT. It MUST
prefix `[dry-run] ` to the refreshed and deployed lines of a dry run, and MUST NOT prefix the nudge.

Supports: [792a19u1a A config with three imports gains the shim](../product/ambient-guidance.md#792a19u1-uat-scenarios), [792a19u1b The shim does not replace the other documents](../product/ambient-guidance.md#792a19u1-uat-scenarios), [792a19v1a A Claude Code document not yet imported is deployed with its line to add](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1b An imported document is refreshed](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1c A Pi document is deployed with Pi's paths](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1d The harnesses are wired independently](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1e Each document is judged on its own](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1f A user who has not opted in is nudged](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1g A user who already imports is not nudged](../product/ambient-guidance.md#792a19v1-uat-scenarios),
[792a19v1h The flag suppresses the nudge](../product/ambient-guidance.md#792a19v1-uat-scenarios), [792a19v1i A dry run prefixes the guidance lines](../product/ambient-guidance.md#792a19v1-uat-scenarios).

Rationale: an import is a line in a file that the user owns, so the report names the exact line to add. The
canonical-path suffix of Claude Code's lines is [792b29c17 Harness sections of the report](#792b29c17-harness-sections-of-the-report).

#### 792b29c18a Scenarios

```gherkin
Scenario: 792b29c18a1 A Claude Code file not imported is deployed with its line to add
  Given a Claude Code report with recall.md deployed and not imported, from a run with --with-guidance
  When the report is written
  Then it holds "guidance deployed to ~/.claude/engram/recall.md — add '@~/.claude/engram/recall.md' to ~/.claude/CLAUDE.md to activate it (Claude Code will ask you to approve the import once)"

Scenario: 792b29c18a2 An imported Claude Code file is refreshed
  Given a Claude Code report with recall.md deployed and imported
  When the report is written
  Then it holds "guidance refreshed: ~/.claude/engram/recall.md" and no "add '@~/.claude/engram/recall.md'"

Scenario: 792b29c18a3 A Pi file not imported is deployed with Pi's paths
  Given a Pi report with recall.md deployed and not imported
  When the report is written
  Then it holds "guidance deployed to ~/.pi/agent/guidance/recall.md — add '@~/.pi/agent/guidance/recall.md' to ~/.pi/agent/AGENTS.md to activate it (Pi will ask you to approve the import once)"

Scenario: 792b29c18a4 An imported Pi file is refreshed
  Given a Pi report with recall.md deployed and imported
  When the report is written
  Then it holds "guidance refreshed: ~/.pi/agent/guidance/recall.md" and no line to add

Scenario: 792b29c18a5 Each harness is judged on its own imports
  Given both harnesses with recall.md deployed and only Claude Code importing it
  When the report is written
  Then it holds "guidance refreshed: ~/.claude/engram/recall.md" and the line adding '@~/.pi/agent/guidance/recall.md' to ~/.pi/agent/AGENTS.md, and no line adding the Claude Code import

Scenario: 792b29c18a6 A plain update with nothing deployed or imported gives the nudge
  Given a Claude Code report with no guidance deployed, no --with-guidance and no import
  When the report is written
  Then it holds "engram ships recall- and delegation-firing guidance"

Scenario: 792b29c18a7 An existing import silences the nudge
  Given a Claude Code report with no guidance deployed, no --with-guidance and a guidance import
  When the report is written
  Then it holds neither "engram ships" nor "activate it"

Scenario: 792b29c18a8 Each file is judged on its own imports
  Given Claude Code with recall.md and delegate.md deployed and only recall.md imported
  When the report is written
  Then recall.md is reported refreshed and delegate.md is reported deployed with "@~/.claude/engram/delegate.md" to add

Scenario: 792b29c18a9 The flag silences the nudge
  Given a report from a run with --with-guidance in which no harness deployed guidance
  When the report is written
  Then it does not hold "engram ships"

Scenario: 792b29c18a10 A dry run prefixes the guidance lines
  Given a dry run with guidance deployed and not imported
  When the report is written
  Then the line begins "[dry-run] guidance deployed to"
```
