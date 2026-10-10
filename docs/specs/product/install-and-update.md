# Install and Update - Product Spec

`engram update` is how an operator brings a machine to the current engram. It installs the engram
binary from a source checkout, then deploys engram's skills (and, on request, its guidance documents)
into each coding harness it finds, Claude Code and Pi. Engram keeps its own copy of what it deploys in
a root of its own for each harness and links the harness's discovery paths to that copy, so that a skill
removed from the source disappears from every harness on the next update, and a file engram does not
own is never deleted. The notices that `engram update` prints about the vault (the vocabulary, the
Luhmann branching, the missing note origin, the uncommitted vault ID, pending offers and the parent
outbox) belong to the capabilities whose state they report, and registering skills as runbook notes
after the deploy belongs to [skill-runbook-registration](./skill-runbook-registration.md). This capability is the install, the deploy and
the report around them.

## 792a11a Installing from a local clone or a fresh clone

Requirement: [792a11a Installing from a local clone or a fresh clone](#792a11a-installing-from-a-local-clone-or-a-fresh-clone) `engram update` MUST install the engram binary from the nearest enclosing
directory of the working directory whose `go.mod` names the module `github.com/toejough/engram`, and
MUST install it from a fresh shallow clone of the engram repository when no enclosing directory has a
`go.mod`. It MUST install into the directory `go install` uses, as the Go toolchain resolves it (including values set with `go env -w`): `GOBIN`, else the first `GOPATH` entry plus `/bin`, else `~/go/bin`, and MUST fail naming the
missing program when `go` (or, for a clone, `git`) is not installed. It MUST stop at the first `go.mod`
it meets, so a working directory inside another Go module installs from a fresh clone.

Rationale: a clone is built rather than fetched with `go install …@latest`, because the Go module proxy
serves repository blobs raw and the embedded model is a git-lfs object, so a proxy build embeds a
133-byte pointer and every embedding call then fails (#645). Running `engram update` from inside a clone
installs what that clone holds, which lets a contributor test a branch.

### 792a11a1 UAT Scenarios

```gherkin
Scenario: 792a11a1a A local clone is installed in place
  Given the working directory is inside a checkout whose go.mod names github.com/toejough/engram
  When the user runs engram update
  Then the binary is built from that checkout, and no clone is made

Scenario: 792a11a1b Outside a checkout the repository is cloned
  Given no directory from the working directory up holds a go.mod
  When the user runs engram update
  Then the repository is shallow-cloned, the binary is built from the clone, and the report names a remote source

Scenario: 792a11a1c The install location follows the Go toolchain
  Given GOBIN is unset, GOPATH is /gp and the home directory is /home/u
  When the user runs engram update
  Then the report names /gp/bin/engram as the install location, and with GOBIN also set to /gb it names /gb/engram

Scenario: 792a11a1d The first go.mod ends the walk
  Given the working directory is inside a checkout of a different Go module
  When the user runs engram update
  Then the binary is built from a fresh clone, not from that checkout

Scenario: 792a11a1e Go is missing
  Given go is not installed
  When the user runs engram update from a local clone
  Then it fails with an error that the go binary is not on PATH

Scenario: 792a11a1f A go env setting decides the install location
  Given GOBIN is unset in the environment but go env -w GOBIN=/gb was run
  When the user runs engram update
  Then the report names /gb/engram as the install location, the directory go install uses

Scenario: 792a11a1g The first GOPATH entry is used
  Given GOBIN is unset and GOPATH is /gp1:/gp2
  When the user runs engram update
  Then the report names /gp1/bin/engram as the install location

Scenario: 792a11a1f Git is missing
  Given git is not installed and the working directory is outside a checkout
  When the user runs engram update
  Then it fails with an error that the git binary is not on PATH
```

## 792a11b A clone without the model is refused

Requirement: [792a11b A clone without the model is refused](#792a11b-a-clone-without-the-model-is-refused) `engram update` MUST fail without installing when the cloned embedding model file
is a git-lfs pointer or holds less than one mebibyte, and its error MUST name git-lfs and tell the user
to install it and rerun `engram update`.

Rationale: git-lfs absent on the machine means the clone's smudge filter never ran, and building from
the pointer would produce a binary whose embedder fails at its first use.

### 792a11b1 UAT Scenarios

```gherkin
Scenario: 792a11b1a A pointer file stops the install
  Given the clone's model.onnx is a git-lfs pointer file
  When the user runs engram update outside a checkout
  Then it fails with an error naming git-lfs and the install command, and no binary is installed

Scenario: 792a11b1b A real model installs
  Given the clone's model.onnx holds the real model
  When the user runs engram update outside a checkout
  Then the binary is built from the clone

Scenario: 792a11b1c An unreadable model is an error
  Given the clone has no model.onnx
  When the user runs engram update outside a checkout
  Then it fails with an error that the cloned model could not be read
```

## 792a11c A provable downgrade is refused

Requirement: [792a11c A provable downgrade is refused](#792a11c-a-provable-downgrade-is-refused) When it installs from a local clone and a binary is already installed at the install
location, `engram update` MUST read that binary's embedded source revision and MUST refuse to install,
exiting non-zero and saying the install would downgrade the installed binary, when git proves that the
clone's current revision is not a descendant of (and not equal to) the installed revision. It MUST
install anyway when given `--allow-downgrade`. It MUST install without refusing when no binary is
installed, when the installed binary carries no readable revision, or when git cannot tell either way
(the installed revision is unknown to the clone, as in a shallow or unrelated repository).

Rationale: running `engram update` from a stale clone or an old worktree silently replaced a newer
installed binary with older code (#715). Only a provable regression blocks, because an unprovable case
must never stop an update.
Rejected alternatives: a warning alone, because visibility already existed and was the weaker option; and
comparing commit timestamps, because clock skew and rebases make wall-clock time a noisy proxy for worse code,
while ancestry answers whether the new build contains everything the old one did. A feature-branch worktree
that has not merged the main tip is also blocked, an accepted trade-off with `--allow-downgrade` as the
intended escape.

### 792a11c1 UAT Scenarios

```gherkin
Scenario: 792a11c1a A provable downgrade is refused
  Given the installed binary was built at a revision that the clone's current revision does not descend from
  When the user runs engram update from the clone
  Then the install is refused, the error says it would downgrade the installed binary and names --allow-downgrade, and the exit status is non-zero

Scenario: 792a11c1b The flag bypasses the check
  Given the same provable downgrade
  When the user runs engram update --allow-downgrade
  Then the binary is installed

Scenario: 792a11c1c A first install is not checked
  Given no binary exists at the install location
  When the user runs engram update from a clone
  Then the binary is installed without a downgrade check

Scenario: 792a11c1d An unknown installed revision does not block
  Given the installed binary's revision is not a commit known to the clone
  When the user runs engram update from the clone
  Then the binary is installed and the report does not claim a downgrade

Scenario: 792a11c1e An unreadable revision does not block
  Given the installed binary carries no revision, or one that does not parse
  When the user runs engram update from the clone
  Then the binary is installed
```

## 792a11d The report names the source and the install

Requirement: [792a11d The report names the source and the install](#792a11d-the-report-names-the-source-and-the-install) `engram update` MUST begin its report with the line `engram update`, then a source
line naming `local clone at <path> (rev <short SHA>)` or `remote module github.com/toejough/engram <short
SHA>`, then a binary line naming the install command, and after a real install `... ok (engram <short SHA>
→ <install location>)`. It MUST show a path under the home directory as `~/…`, and MUST show the revision
as `unknown` when git cannot resolve one. The offer lines of skill registration ([792a12t Update runs registration after deploying the skills](./skill-runbook-registration.md#792a12t-update-runs-registration-after-deploying-the-skills))
are written as they arise, not as part of the report, so on a dry run, or when this process deploys without a terminal, they precede the
`engram update` line, and in a handed-off run ([792a11e The rest of the update runs in the new binary](#792a11e-the-rest-of-the-update-runs-in-the-new-binary)) they follow the three header lines.

### 792a11d1 UAT Scenarios

```gherkin
Scenario: 792a11d1a A local install reports its revision
  Given a local clone at ~/src/engram at commit abc1234
  When the user runs engram update
  Then the report's source line is "local clone at ~/src/engram (rev abc1234)"

Scenario: 792a11d1b A real install reports where the binary went
  Given the install succeeded to /home/u/go/bin/engram
  When the user runs engram update
  Then the report's binary line ends with "ok (engram <short SHA> → ~/go/bin/engram)"

Scenario: 792a11d1c A remote source reports its revision
  Given the binary was built from a fresh clone at commit abc1234
  When the user runs engram update
  Then the report's source line is "remote module github.com/toejough/engram abc1234"
```

## 792a11e The rest of the update runs in the new binary

Requirement: [792a11e The rest of the update runs in the new binary](#792a11e-the-rest-of-the-update-runs-in-the-new-binary) After a successful install, `engram update` MUST print the install result once,
then hand the rest of the update (deploying to the harnesses and checking the vault and chunk index) to
the freshly installed binary at the install location, and MUST exit with that binary's exit status
without repeating those steps itself. The handed-off run MUST NOT install again and MUST NOT print the
install result a second time. When the installed binary cannot be started, `engram update` MUST complete
the rest of the update itself and MUST say in its report that the re-exec failed, with the reason, and
that pre-update logic completed the run. It MUST NOT hand off when `--dry-run` is given, because nothing
was installed.

Rationale: an update that installed a new release and then deployed with the old release's logic made
users one update behind on update behavior itself, so a new deployment rule took effect a run late. The bootstrap run, the first update after this behavior shipped, is the last one to run
stale. Local mode installs from the clone it runs in, not from a fresh pull, so a stale checkout still
downgrades on hand-off; the downgrade check covers it ([792a11c A provable downgrade is refused](#792a11c-a-provable-downgrade-is-refused)).
The old process hands off to a child rather than replacing itself with `syscall.Exec`, because the captured-output
commander cannot inherit stdio or pass an environment and the parent must relay the child's exit code.
The loop guard is an environment variable, not a flag, so it stays off the public command line. A successful
install always hands off with no version comparison: an unchanged binary costs one local spawn.

### 792a11e1 UAT Scenarios

```gherkin
Scenario: 792a11e1a The deploy runs in the new binary
  Given an update that installs successfully
  When the user runs engram update
  Then the install result is printed once, the deploy report is printed once by the installed binary, and the exit status is that binary's

Scenario: 792a11e1b The install result comes first
  Given an update that installs successfully
  When the user runs engram update
  Then the output begins with the "engram update", source and binary lines, followed by the deploy report with no second header

Scenario: 792a11e1c A failed hand-off falls back
  Given an install that succeeds but an installed binary that cannot be started
  When the user runs engram update
  Then the update still deploys to the harnesses, and its report says re-exec failed, completed with pre-update logic, and why

Scenario: 792a11e1d A handed-off run does not claim an install
  Given an update running as the handed-off binary
  When it prints its report
  Then the report does not say that go install ran

Scenario: 792a11e1e A dry run never hands off
  Given any machine
  When the user runs engram update --dry-run
  Then nothing is installed and no second process is started
```

## 792a11f Previewing an update

Requirement: [792a11f Previewing an update](#792a11f-previewing-an-update) `engram update --dry-run` MUST install nothing, MUST create, change and delete no
file, directory or link in a harness, an engram-owned root or the vault (including a root's marker, and
including a sync that would create or delete), and MUST print the deploy plan, prefixing with `[dry-run] ` every report line that
describes something that was or would be done (the title, the `installed:` line, the root deletions, the
adoptions, the dangling-link cleanups and the guidance deploy lines) and leaving notice lines (a refused
deletion, an unattributable file, an unmanaged entry, a skill-row mapping) unprefixed.

Rationale: the same prefix discipline applies to every action line, so a reader can tell what happened
from what would happen without knowing which operation produced the line (#709). A dry run from outside a
checkout still clones the repository into its scratch directory, because the plan is read from the clone;
it builds nothing.

### 792a11f1 UAT Scenarios

```gherkin
Scenario: 792a11f1a A dry run on a never-synced harness leaves no trace
  Given a machine whose ~/.claude holds no engram root
  When the user runs engram update --dry-run
  Then the report lists each skill mapping (unprefixed) under the "[dry-run] engram update" title and above the "[dry-run] installed:" line, and afterwards no engram root, marker, link or file exists

Scenario: 792a11f1b Action lines carry the prefix, notices do not
  Given a dry run that would delete a root file, adopt a copied skill, remove a dangling link and refresh guidance, and that finds an unmanaged entry
  When the user runs engram update --dry-run
  Then each action line begins with "[dry-run] " and the unmanaged line does not

Scenario: 792a11f1c A dry run installs nothing
  Given a local clone newer than the installed binary
  When the user runs engram update --dry-run
  Then the installed binary is unchanged and the binary line shows the planned install command without "ok"
```

## 792a11g The harnesses that are updated

Requirement: [792a11g The harnesses that are updated](#792a11g-the-harnesses-that-are-updated) `engram update` MUST update each of Claude Code (detected by `~/.claude`) and Pi
(detected by `~/.pi`) whose directory exists, MUST NOT create the directory of one that is absent, and
MUST fail with an error saying no supported harness was found at `~/.claude/` or `~/.pi/` when neither
exists, before installing anything. It MUST NOT deploy to any other harness. A failure in one harness MUST
NOT stop the other from being updated, MUST be reported under that harness as `error: <reason>`, MUST be
left out of the `installed:` line, and MUST make the command exit non-zero.

Rationale: OpenCode and Shelley support was removed (#721), so the harness list is Claude Code and Pi.
The dated addendum to the harness decision, rather than a rewrite, keeps the record of why per-file and per-directory
symlink granularity were chosen when three harnesses were in scope. The unused commands-deploy
mechanism was deleted rather than kept dormant, because git history is the fallback and a future harness would
reintroduce it from scratch.

### 792a11g1 UAT Scenarios

```gherkin
Scenario: 792a11g1a Only the harnesses present are updated
  Given ~/.claude exists and ~/.pi does not
  When the user runs engram update
  Then the report has a Claude Code section and no Pi section, and ~/.pi is not created

Scenario: 792a11g1b Both harnesses are updated in a stable order
  Given ~/.claude and ~/.pi both exist
  When the user runs engram update
  Then the report lists Claude Code then Pi, and the installed line names both

Scenario: 792a11g1c No harness is an error before any install
  Given neither ~/.claude nor ~/.pi exists
  When the user runs engram update
  Then it fails saying no supported harness was found at ~/.claude/ or ~/.pi/, and nothing was installed

Scenario: 792a11g1d One failing harness does not stop the other
  Given writing under ~/.claude fails and ~/.pi is healthy
  When the user runs engram update
  Then Pi is updated, the Claude Code section shows "error: <reason>", the installed line names only Pi, and the exit status is non-zero

Scenario: 792a11g1e A missing skills source is an error
  Given the source checkout has no agent-instructions/skills directory
  When the user runs engram update
  Then it fails with an error that the skills source directory is missing
```

## 792a11h Skills are deployed to an engram-owned root and linked

Requirement: [792a11h Skills are deployed to an engram-owned root and linked](#792a11h-skills-are-deployed-to-an-engram-owned-root-and-linked) `engram update` MUST copy every file under `agent-instructions/skills/` of the
source into the `skills/` subtree of the harness's engram-owned root (`~/.claude/engram/` for Claude Code,
`~/.pi/agent/engram/` for Pi), creating a missing file and overwriting a different one, and MUST make
`~/.claude/skills/<skill>` and `~/.pi/agent/skills/<skill>` each a symbolic link to that skill's directory
in the root, one link per skill directory, never a copy. It MUST repoint a link that points anywhere else,
and MUST leave a correct link alone. Its report MUST list each skill as `agent-instructions/skills/<skill>/
→ <skills dir>/<skill>/  (<N> file[s])`, then `installed: <harnesses>`.

### 792a11h1 UAT Scenarios

```gherkin
Scenario: 792a11h1a A skill deploys as a directory link
  Given the source holds skill recall and a symlink-mode harness whose skills dir is ~/.claude/skills
  When the user runs engram update
  Then ~/.claude/skills/recall is a symbolic link to the skills/recall directory of ~/.claude/engram, and its SKILL.md is readable through the link

Scenario: 792a11h1b Added and changed sources deploy
  Given a new skill and an edited file in an existing skill since the last update
  When the user runs engram update
  Then the new skill appears in the engram root with a link from the harness, and the edited file in the root matches the source

Scenario: 792a11h1c A link that points elsewhere is repointed
  Given ~/.claude/skills/recall is a symbolic link to a different directory
  When the user runs engram update
  Then it is a link to the engram root's skills/recall

Scenario: 792a11h1d The report lists each skill and its file count
  Given a source skill learn with 3 files and recall with 1
  When the user runs engram update
  Then the report lists "agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)" and "... recall/ → ...  (1 file)"

Scenario: 792a11h1e Re-running changes nothing
  Given an update has completed against the same source
  When the user runs engram update again
  Then every root file and link is the same as after the first run
```

## 792a11i Removal propagates

Requirement: [792a11i Removal propagates](#792a11i-removal-propagates) `engram update` MUST delete from a harness's engram-owned root every file in a
managed subtree (`skills/`, `commands/` and, while guidance is managed, `guidance/`) that is not in the
deploy set it computes from the source (engram deploys no commands, so it deletes any file in `commands/`), MUST delete a directory in those subtrees that would be left holding nothing
the deploy set keeps, and MUST list each deletion in the report as `engram root: deleted <path>`. The
harness link for a skill whose directory was deleted MUST be gone by the end of the same run.

Rationale: an earlier update copied additively, so an artifact deleted from the source was never removed
from a harness, and a stray file in the source persisted until deleted by hand. Removing a
skill is now a source change followed by an update.

### 792a11i1 UAT Scenarios

```gherkin
Scenario: 792a11i1a A removed skill disappears
  Given a skill directory that was deployed earlier is deleted from agent-instructions/skills/
  When the user runs engram update
  Then the skill's subtree is deleted from the engram root, its link in the harness skills directory is gone, and the report lists the deletion

Scenario: 792a11i1b A removed file inside a kept skill disappears
  Given a file deleted from a skill that is still in the source
  When the user runs engram update
  Then the file and any directory it leaves empty are deleted from the engram root
```

## 792a11j Only what engram owns is deleted

Requirement: [792a11j Only what engram owns is deleted](#792a11j-only-what-engram-owns-is-deleted) `engram update` MUST NOT delete a file in an engram-owned root that lacks the
ownership marker `.engram-owned` at the start of the run, and MUST NOT delete a real (non-link) file or
directory in a harness's skills or guidance directory, other than by replacing a deploy-set copy with a link ([792a11k The first update adopts earlier copies](#792a11k-the-first-update-adopts-earlier-copies)). A root that exists without the marker MUST get the
marker and its deploy set written, MUST NOT have anything deleted in it that run, and MUST be reported
with `engram root <path> exists without the .engram-owned marker — sync-deletion refused there` and one
`unattributable: <name>` line for each file at its top level that matches no deploy-set name. On every
run it MUST delete, from the skills directory and the guidance directory (or the engram root, for a
harness whose guidance directory is its root), exactly the symbolic links that point lexically into the
harness's engram root and whose target no longer exists, reporting each as `cleanup: removed dangling link
<path>`, and MUST NOT touch a link that points anywhere else.

Rationale: engram is the only writer inside its root, and the only deletions outside it are links that
provably point into it. The marker is what proves a directory named `engram/` is engram's.

### 792a11j1 UAT Scenarios

```gherkin
Scenario: 792a11j1a An unmarked root is not purged
  Given the engram root path exists without the ownership marker and holds a file not in the deploy set
  When the user runs engram update
  Then the file is left in place and listed as unattributable, the marker is written, and no deletion happens in that directory

Scenario: 792a11j1b The marker enables deletion on the next run
  Given the previous update wrote the marker into a root that held an extra file
  When the user runs engram update again
  Then a file in the managed subtrees that is not in the deploy set is deleted

Scenario: 792a11j1c A dangling engram link is removed and a foreign link kept
  Given a skills directory holding a link into the engram root whose target was removed and a second link to a path outside the root
  When the user runs engram update
  Then the first link is deleted and the second is untouched

Scenario: 792a11j1d User-owned real files are never deleted
  Given a harness skills directory holds a real skill directory the user installed by other means
  When the user runs engram update
  Then it is untouched by the sync, the cleanup and the migration

Scenario: 792a11j1e A root that is a file is refused
  Given the engram root path is a regular file
  When the user runs engram update
  Then that harness reports an error that the engram root exists and is not a directory, and the file is untouched
```

## 792a11k The first update adopts earlier copies

Requirement: [792a11k The first update adopts earlier copies](#792a11k-the-first-update-adopts-earlier-copies) `engram update` MUST replace each deploy-set skill that is a real directory in the
harness skills directory (a copy made by an earlier version) with a link to the root's copy, and MUST
report each as `adopted: <path>`. On the first update of a harness (its engram root absent or lacking the
marker) it MUST also create the root and its marker, and MUST list every other real file or directory in
the harness's skills or guidance directory as `unmanaged (left alone): <path>` without deleting or
changing it. Later runs MUST NOT list unmanaged entries again.

Rationale: the repository is the source of truth, so an earlier copy is replaced without comparing its
content. Engram cannot prove it wrote an entry it never recorded, so it does not call one engram's
debris (#716).

### 792a11k1 UAT Scenarios

```gherkin
Scenario: 792a11k1a A copied skill becomes a link
  Given the first update of a harness whose skills directory holds recall as a real directory from an earlier copy-mode deploy
  When the user runs engram update
  Then recall's content lives in the engram root, the harness path is a link to it, the skill is still discoverable, and the report lists it as adopted

Scenario: 792a11k1b An unmanaged entry is reported, not deleted
  Given the first update finds a real directory in the skills directory that is not in the deploy set
  When the user runs engram update
  Then it is left in place and the report lists it as unmanaged, left alone

Scenario: 792a11k1c The unmanaged list is not repeated
  Given a later update of the same harness with the same unmanaged entry
  When the user runs engram update
  Then the report does not list it
```

## 792a11l Guidance is managed only after opt-in

Requirement: [792a11l Guidance is managed only after opt-in](#792a11l-guidance-is-managed-only-after-opt-in) `engram update` MUST include the guidance documents of `agent-instructions/guidance/`
(the `.md` files directly in it) in the deploy set of every detected harness only when `--with-guidance`
is given or the config file of at least one detected harness already imports a guidance file, and MUST
otherwise leave the root's `guidance/` subtree and any guidance link untouched, neither refreshing nor
deleting it. A config file imports a guidance file when a line outside a code fence is exactly
`@~/<guidance dir>/<name>.md` or the same path with the home directory spelled out (`~/.claude/CLAUDE.md`
with the `~/.claude/engram/` guidance dir for Claude Code, `~/.pi/agent/AGENTS.md` with
`~/.pi/agent/guidance/` for Pi). It MUST deploy a guidance document as `<engram root>/guidance/<name>.md`
and link it from the harness's guidance directory, and MUST delete from `guidance/`, when guidance is
managed, a document that has left the source.

Rationale: management is an opt-in once, so that after a user wires an import, a plain `engram update`
keeps their guidance current as it does skills. Not being in the deploy set means unmanaged, not removed.
The import lines, the hints that name them and the guidance documents themselves belong to
[ambient-guidance](./ambient-guidance.md).

### 792a11l1 UAT Scenarios

```gherkin
Scenario: 792a11l1a The flag deploys guidance
  Given the source holds guidance recall.md and the user passes --with-guidance
  When the user runs engram update
  Then ~/.claude/engram/guidance/recall.md is written

Scenario: 792a11l1b An existing import keeps guidance current
  Given ~/.claude/CLAUDE.md imports @~/.claude/engram/recall.md and the user passes no flag
  When the user runs engram update
  Then guidance is refreshed

Scenario: 792a11l1c Without opt-in guidance is untouched
  Given guidance files were deployed earlier, no flag is passed, and the config file no longer imports them
  When the user runs engram update
  Then the guidance files are neither refreshed nor deleted

Scenario: 792a11l1d An import inside a code fence does not count
  Given the only line importing guidance is inside a fenced code block of the config file
  When the user runs engram update
  Then guidance is not managed

Scenario: 792a11l1e A guidance document removed from the source is deleted once managed
  Given guidance is managed and a deployed guidance file has left the source
  When the user runs engram update
  Then it is deleted from the engram root's guidance subtree and its link is gone

Scenario: 792a11l1f Pi guidance is linked into Pi's guidance directory
  Given the user passes --with-guidance and ~/.pi exists
  When the user runs engram update
  Then ~/.pi/agent/guidance/recall.md is a link to ~/.pi/agent/engram/guidance/recall.md

Scenario: 792a11l1g One harness's import manages guidance for every harness
  Given only ~/.claude/CLAUDE.md imports a guidance file, no flag is passed, and ~/.pi exists
  When the user runs engram update
  Then guidance is also deployed for Pi
```

## 792a11m Claude import paths keep working

Requirement: [792a11m Claude import paths keep working](#792a11m-claude-import-paths-keep-working) For Claude Code, whose guidance directory is its engram root, `engram update` MUST
leave for each managed guidance document a symbolic link at `~/.claude/engram/<name>.md` pointing at
`~/.claude/engram/guidance/<name>.md`, replacing a real file there from an earlier version, so that an
`@~/.claude/engram/<name>.md` import in `CLAUDE.md` still resolves to the deployed content. It MUST delete
such a link when its document leaves the deploy set, and its report MUST give the canonical `guidance/`
path of each deployed Claude Code document.

Rationale: user-authored import lines name the flat path verbatim, so the canonical move into
`guidance/` could not break them. A later change can retire the links once imports are migrated.

### 792a11m1 UAT Scenarios

```gherkin
Scenario: 792a11m1a An existing import still resolves
  Given the first sync moves recall.md into the root's guidance subtree and CLAUDE.md imports ~/.claude/engram/recall.md
  When the user runs engram update with guidance managed
  Then ~/.claude/engram/recall.md is a link to ~/.claude/engram/guidance/recall.md and the import resolves to the deployed content

Scenario: 792a11m1b A real flat file from an earlier version is adopted
  Given ~/.claude/engram/recall.md is a real file from an earlier version
  When the user runs engram update with guidance managed
  Then it becomes a link to the canonical copy and the report lists it as adopted

Scenario: 792a11m1c A link whose document left the source is removed
  Given a compat link for a guidance document that is no longer in the source
  When the user runs engram update
  Then the link is deleted

Scenario: 792a11m1d The report states the canonical path
  Given Claude Code guidance is deployed
  When the user runs engram update
  Then each guidance line for Claude Code ends with "(canonical: ~/.claude/engram/guidance/<name>)", and Pi's lines carry no such suffix
```

## 792a11n Backlog notices for the chunk index

Requirement: [792a11n Backlog notices for the chunk index](#792a11n-backlog-notices-for-the-chunk-index) `engram update` MUST print the line ``duplicate chunk-index files found — run `engram
prune --duplicates` to clear them (preview with `engram prune --duplicates --dry-run`)`` when running
`engram prune --duplicates` would remove at least one index file, and ``empty chunk-index files found —
run `engram prune --empty` to clear them (preview with `engram prune --empty --dry-run`)`` when the chunk
index holds a zero-byte index file. It MUST print neither when the index is clean, when its manifest or
directory is missing or unreadable, or, for the duplicates notice, when every duplicate group would be
refused by prune's coverage check. It MUST NOT remove anything itself.

Rationale: removing index files is a destructive one-shot migration over a user's memory store. Unlike an
empty file, which holds no records, a duplicate's removal is safe only while the record-subset check holds
at the moment it runs, so it runs when the user asks for it and not as a side effect of a routine
refresh. Detection keeps no sentinel: it reports the live backlog every run and
goes quiet once it is cleared. It runs prune's own dry-run check so the notice fires only when a removal
would happen, and a backlog whose every group would be refused stays silent because no command resolves
it (#713).

### 792a11n1 UAT Scenarios

```gherkin
Scenario: 792a11n1a Update reports the prune command when duplicates exist
  Given a chunk index with several canonical entries per content-hash group whose removal prune would allow
  When the user runs engram update
  Then it prints the duplicates notice naming engram prune --duplicates and its --dry-run preview, and removes nothing

Scenario: 792a11n1b A refusal-only backlog is silent
  Given duplicate groups none of which prune --duplicates would remove
  When the user runs engram update
  Then it prints no duplicates notice

Scenario: 792a11n1c Empty index files are reported
  Given the chunk index holds a zero-byte .jsonl file
  When the user runs engram update
  Then it prints the empty-files notice naming engram prune --empty

Scenario: 792a11n1d A clean or missing index is silent
  Given a chunk index with no duplicates and no empty files, or no chunk index
  When the user runs engram update
  Then it prints neither notice

Scenario: 792a11n1e The notices coexist with the other upgrade notices
  Given both backlogs and a vault with old-format vocabulary files
  When the user runs engram update
  Then the report contains the empty-files notice, the duplicates notice and the vocabulary notice
```

## 792a11o The update options

Requirement: [792a11o The update options](#792a11o-the-update-options) `engram update --help` MUST describe the update as refreshing the engram binary and
harness skills, and MUST list `--dry-run`, `--with-guidance`, `--allow-downgrade`, `--regen-vocab`,
`--reparent-luhmann`, `--answers <string>` and `--backfill-identity`. `engram update` MUST pass
`--with-guidance`, `--regen-vocab` and `--backfill-identity` on to the handed-off run ([792a11e The rest of the update runs in the new binary](#792a11e-the-rest-of-the-update-runs-in-the-new-binary)), and MUST
NOT pass `--dry-run`, which never hands off. `--reparent-luhmann` MUST run only the re-parenting of
[vault-migration](./vault-migration.md), without installing or deploying anything ([792a10b Re-parenting only on request](./vault-migration.md#792a10b-re-parenting-only-on-request)).

### 792a11o1 UAT Scenarios

```gherkin
Scenario: 792a11o1a The help lists every option
  Given any machine
  When the user runs engram update --help
  Then it describes refreshing the binary and harness skills and lists --dry-run, --with-guidance, --allow-downgrade, --regen-vocab, --reparent-luhmann, --answers and --backfill-identity

Scenario: 792a11o1b Options reach the handed-off run
  Given an update that installs and hands off
  When the user runs engram update --with-guidance --regen-vocab --backfill-identity
  Then the installed binary is started as engram update with those three options and no others

Scenario: 792a11o1c Re-parenting skips the install and the deploy
  Given a machine with no harness and no Go toolchain
  When the user runs engram update --reparent-luhmann
  Then it derives candidates and installs and deploys nothing
```
