# Self Installer - Technical Spec

## 792b28a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the Go toolchain and git), and the
  repository component that runs `engram update` end to end.
- Responsibility: refreshes the engram binary from a source checkout, or from a fresh clone when there
  is none, refuses an install that would provably downgrade the installed binary, hands the rest of the
  update to the binary it just installed, and, in whichever process carries out the deploy, collects the
  vault and chunk-index checks and writes the update report with its exit status.
- Current realization (informational): `internal/update/update.go` (`Updater.Run` and its source,
  install, downgrade and hand-off steps, `resolveBinaryPath`, `walkUpForModule`) and
  `internal/cli/update.go` (`runUpdate`, `runPostUpdateChecks`, `reexecArgsFrom`, `writeUpdateReport`,
  `finishUpdate`), over the injected `update.Commander`, `update.Spawner`, `update.Env` and
  `update.Filesystem`.
- Qualification: responsibility, installing a newer binary and carrying the update out in it is stated
  without naming any function; ownership, it owns acquisition, the downgrade gate, the hand-off and the
  report, while the deploy of skills and guidance belongs to [harness-deploy-sync](./harness-deploy-sync.md) and each notice's
  detection to the component that owns the state it reports; independent verification, the install
  steps run over a fake commander, filesystem and spawner, and the update run over fakes of those and a
  recording exit function; stability, the contract survives any change to how the deploy works;
  necessity, `engram update` is the only way a machine receives a new release.
- Supported capabilities: [install-and-update](../product/install-and-update.md), [vault-migration](../product/vault-migration.md) (the notices and the flag
  routing), [controlled-vocabulary](../product/controlled-vocabulary.md) (the notices and the `--regen-vocab` summary).
  Also [skill-runbook-registration](../product/skill-runbook-registration.md), [parent-sync](../product/parent-sync.md) and [curate](../product/curate.md) (the registration, outbox and pending-offer
  steps and notices of the update run).

Rationale: the installed binary, not the running one, must carry out the deploy, because a release that
changes how an update works would otherwise take effect one update late. Because the child
inherits the parent's streams and starts after the parent has printed, the output is one report in the
order install result, then deploy and checks. A downgrade gate that fails open is the only safe shape: an
update whose safety cannot be proved must still be possible, and only a provable regression blocks. The checks run in the process that deployed so that they and their notices come from the same
release as the deploy.

## 792b28c Behavior Requirements

### 792b28c1 Install location

Requirement: [792b28c1 Install location](#792b28c1-install-location) The self-installer MUST name the install location `go install` uses, reading `GOBIN` and `GOPATH` as the Go toolchain
does (`go env GOBIN GOPATH`, which includes values set with `go env -w`): `$GOBIN/engram` when `GOBIN` is
non-empty, else `<first GOPATH entry>/bin/engram` when `GOPATH` is non-empty, else `<home>/go/bin/engram`.

Supports: [792a11a1c The install location follows the Go toolchain](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1f A go env setting decides the install location](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1g The first GOPATH entry is used](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11d1b A real install reports where the binary went](../product/install-and-update.md#792a11d1-uat-scenarios).

Rationale: this is where `go install` puts the binary, in the toolchain's own order and from the toolchain's own settings, so the hand-off
starts the binary that was just installed.

#### 792b28c1a Scenarios

```gherkin
Scenario: 792b28c1a1 GOBIN wins
  Given GOBIN is /custom/gobin and GOPATH is /custom/gopath
  When the install location is resolved
  Then it is /custom/gobin/engram

Scenario: 792b28c1a2 GOPATH is next
  Given GOBIN is unset and GOPATH is /custom/gopath
  When the install location is resolved
  Then it is /custom/gopath/bin/engram

Scenario: 792b28c1a3 The home directory is the fallback
  Given neither is set and the home directory is /home/joe
  When the install location is resolved
  Then it is /home/joe/go/bin/engram

Scenario: 792b28c1a4 A go env setting is honored
  Given GOBIN is unset in the environment and go env -w GOBIN=/custom/gobin was run
  When the install location is resolved
  Then it is /custom/gobin/engram

Scenario: 792b28c1a5 The first GOPATH entry is used
  Given GOBIN is unset and GOPATH is /gp1:/gp2
  When the install location is resolved
  Then it is /gp1/bin/engram
```

### 792b28c2 Source choice

Requirement: [792b28c2 Source choice](#792b28c2-source-choice) The self-installer MUST look for a `go.mod` in the working directory and then in each
parent up to the filesystem root, MUST decide on the first one it finds, and MUST take its directory as a
local source only when the first line of the file that is neither blank nor a `//` comment is a `module`
directive naming `github.com/toejough/engram` (a trailing `//` comment after the name is ignored). It MUST
use a fresh clone when it finds no `go.mod`, or finds one for another module, or one whose first directive
is not `module`. It MUST fail naming the path when a `go.mod` exists but cannot be read, and the walk MUST
end for every path.

Supports: [792a11a1a A local clone is installed in place](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1b Outside a checkout the repository is cloned](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1d The first go.mod ends the walk](../product/install-and-update.md#792a11a1-uat-scenarios).

Rationale: stopping at the nearest `go.mod` means a checkout of another Go module, such as an engram
vendored inside a larger project, is not mistaken for the engram source.

#### 792b28c2a Scenarios

```gherkin
Scenario: 792b28c2a1 The module is at the working directory
  Given /repo/go.mod names github.com/toejough/engram and the working directory is /repo
  When the source is chosen
  Then the local source is /repo

Scenario: 792b28c2a2 The module is in an ancestor
  Given /repo/go.mod names the module and the working directory is /repo/internal/update
  When the source is chosen
  Then the local source is /repo

Scenario: 792b28c2a3 No go.mod means a clone
  Given no directory from the working directory up holds a go.mod
  When the source is chosen
  Then no local source is found and a clone is used

Scenario: 792b28c2a4 Another module ends the walk
  Given /repo/internal/go.mod names a different module and /repo/go.mod names engram, and the working directory is /repo/internal
  When the source is chosen
  Then no local source is found

Scenario: 792b28c2a5 Blank lines and comments come before the module line
  Given a go.mod that begins with a blank line and a // comment before module github.com/toejough/engram
  When the source is chosen
  Then the local source is its directory

Scenario: 792b28c2a6 A trailing comment is ignored
  Given a go.mod whose module line ends in // comment
  When the source is chosen
  Then the local source is its directory

Scenario: 792b28c2a7 A first directive that is not module does not match
  Given a go.mod whose first directive is go 1.22
  When the source is chosen
  Then no local source is found

Scenario: 792b28c2a8 An unreadable go.mod is an error
  Given reading /repo/go.mod fails for a reason other than not existing
  When the source is chosen
  Then the choice fails with an error naming /repo/go.mod

Scenario: 792b28c2a9 The walk ends for every path
  Given any working directory path
  When the source is chosen
  Then the walk terminates
```

### 792b28c3 Preconditions

Requirement: [792b28c3 Preconditions](#792b28c3-preconditions) The self-installer MUST fail with an error saying it could not resolve the home
directory when the home directory cannot be resolved, MUST then detect the harnesses ([792b29c1 Harness detection](./harness-deploy-sync.md#792b29c1-harness-detection)) and fail
with an error that satisfies `update.ErrNoHarness` and names `~/.claude/` and `~/.pi/` when none is found,
and MUST then fail with an error naming `getwd` when the working directory cannot be resolved. It MUST
run no command and write nothing before these checks pass.

Supports: [792a11g1c No harness is an error before any install](../product/install-and-update.md#792a11g1-uat-scenarios).

#### 792b28c3a Scenarios

```gherkin
Scenario: 792b28c3a1 An unresolvable home directory is an error
  Given the home directory cannot be resolved
  When the update runs
  Then it fails with an error containing "resolving home", and no command runs

Scenario: 792b28c3a2 No harness is an error before any command
  Given neither ~/.claude nor ~/.pi exists
  When the update runs
  Then it fails with an error satisfying ErrNoHarness, no command runs, and the sentinel survives the report tail's wrapping

Scenario: 792b28c3a3 An unresolvable working directory is an error
  Given a harness exists and the working directory cannot be resolved
  When the update runs
  Then it fails with an error containing "getwd"
```

### 792b28c4 Installing from a local source

Requirement: [792b28c4 Installing from a local source](#792b28c4-installing-from-a-local-source) When the source is local ([792b28c2 Source choice](#792b28c2-source-choice)), the self-installer MUST resolve the module root's
revision with `git rev-parse --short HEAD` run in the module root, and MUST use `unknown` when git fails or
prints nothing. After the downgrade gate ([792b28c7 Downgrade gate](#792b28c7-downgrade-gate)) it MUST run `go install ./cmd/engram/` in the module
root, not the working directory, unless the run is a dry run or the handed-off run ([792b28c8 Installs that are skipped](#792b28c8-installs-that-are-skipped)). A missing
`go` MUST be an error satisfying `update.ErrGoNotFound`, and any other install failure an error that
wraps the failure and names `go install (local)`.

Supports: [792a11a1a A local clone is installed in place](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1e Go is missing](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11d1a A local install reports its revision](../product/install-and-update.md#792a11d1-uat-scenarios).

#### 792b28c4a Scenarios

```gherkin
Scenario: 792b28c4a1 The install runs in the module root
  Given a local source /repo and the working directory /repo/internal/update
  When the update installs
  Then git rev-parse and go install ./cmd/engram/ both run with the directory /repo

Scenario: 792b28c4a2 The revision is resolved before the install and reported
  Given a local source whose HEAD is abc1234
  When the update installs
  Then the calls are git rev-parse --short HEAD and then go install ./cmd/engram/, and the report's source revision is abc1234

Scenario: 792b28c4a3 An unresolvable revision is unknown
  Given git fails or prints nothing for rev-parse
  When the source is resolved
  Then the revision is unknown

Scenario: 792b28c4a4 A missing go is a typed error
  Given go is not installed, so the commander reports command-not-found
  When the update installs
  Then the run fails with an error satisfying ErrGoNotFound, and the sentinel survives the report tail's wrapping

Scenario: 792b28c4a5 Another install failure surfaces
  Given go install fails with an error
  When the update installs
  Then the run fails with an error wrapping that failure
```

### 792b28c5 Installing from a fresh clone

Requirement: [792b28c5 Installing from a fresh clone](#792b28c5-installing-from-a-fresh-clone) When there is no local source, the self-installer MUST remove any earlier scratch
clone at `$TMPDIR/engram-update-clone` (`/tmp/engram-update-clone` when `TMPDIR` is empty), run `git clone
--depth 1 https://github.com/toejough/engram.git` into it, check the model ([792b28c6 Model check](#792b28c6-model-check)), run `go install
./cmd/engram/` in the clone unless the run is a dry run or the handed-off run, and resolve the clone's
revision as in [792b28c4 Installing from a local source](#792b28c4-installing-from-a-local-source). It MUST NOT use `go install …@latest`. A missing `git` MUST be an error
satisfying `update.ErrGitNotFound`, a failed removal an error naming the clone directory, and any other
clone failure an error that wraps it and names `git clone`.

Supports: [792a11a1b Outside a checkout the repository is cloned](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11a1f Git is missing](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11d1c A remote source reports its revision](../product/install-and-update.md#792a11d1-uat-scenarios).

Rationale: the module proxy serves repository blobs raw, so the git-lfs model arrives as a 133-byte
pointer and a proxy build embeds it; a clone runs git's smudge filter.

#### 792b28c5a Scenarios

```gherkin
Scenario: 792b28c5a1 The binary is built from a clone
  Given no local source and a clone that holds a real model
  When the update installs
  Then git clone --depth 1 of the repository URL runs, go install ./cmd/engram/ runs with the clone as its directory, no command contains @latest, and the report's source is remote with that revision

Scenario: 792b28c5a2 An earlier clone is removed first
  Given a scratch clone left by an earlier run
  When the update clones
  Then it is removed before the new clone starts

Scenario: 792b28c5a3 Missing git is a typed error
  Given git is not installed, so the commander reports command-not-found
  When the update clones
  Then the run fails with an error satisfying ErrGitNotFound

Scenario: 792b28c5a4 Another clone failure surfaces
  Given git clone fails with a network error
  When the update clones
  Then the run fails with an error wrapping it that names git clone

Scenario: 792b28c5a5 Both harnesses are deployed from the clone
  Given no local source, a clone holding a skill, and both harnesses present
  When the update runs
  Then the report's source is the clone and its revision, and each harness's engram root holds the skill from the clone
```

### 792b28c6 Model check

Requirement: [792b28c6 Model check](#792b28c6-model-check) After cloning, the self-installer MUST read `internal/embed/assets/model/model.onnx`
in the clone, and MUST fail with an error satisfying `update.ErrModelLFSStub` that names the file, its
size, git-lfs and the install command (``brew install git-lfs && git lfs install``) and says to rerun
`engram update`, without running `go install`, when the file begins with `version https://git-lfs` or
holds less than one mebibyte. It MUST fail with an error saying `reading cloned model` when the file cannot
be read, and MUST continue otherwise.

Supports: [792a11b1a A pointer file stops the install](../product/install-and-update.md#792a11b1-uat-scenarios), [792a11b1b A real model installs](../product/install-and-update.md#792a11b1-uat-scenarios), [792a11b1c An unreadable model is an error](../product/install-and-update.md#792a11b1-uat-scenarios).

#### 792b28c6a Scenarios

```gherkin
Scenario: 792b28c6a1 A pointer file is refused with guidance
  Given the cloned model.onnx begins with "version https://git-lfs"
  When the update checks the model
  Then it fails with ErrModelLFSStub, the message names git-lfs, and go install never runs

Scenario: 792b28c6a2 A file under one mebibyte is refused
  Given the cloned model.onnx holds a few bytes that are not a pointer
  When the update checks the model
  Then it fails with ErrModelLFSStub

Scenario: 792b28c6a3 An unreadable model is an error
  Given the clone has no model.onnx
  When the update checks the model
  Then it fails with an error containing "reading cloned model"

Scenario: 792b28c6a4 A real model passes
  Given the cloned model.onnx holds more than one mebibyte that is not a pointer
  When the update checks the model
  Then the build goes ahead
```

### 792b28c7 Downgrade gate

Requirement: [792b28c7 Downgrade gate](#792b28c7-downgrade-gate) For a local install that is neither a dry run nor the handed-off run, when a file
exists at the install location, the self-installer MUST read the installed binary's revision as the value
of the `vcs.revision=` line of `go version -m <install location>`, and MUST install without any ancestry
check when no line is found or the value is empty, or when `--allow-downgrade` is given. Otherwise it MUST
run `git merge-base --is-ancestor <installed revision> <resolved revision>` in the module root, MUST install
when it succeeds or when git's error output contains `fatal:` (ancestry is unknown), and MUST otherwise
refuse with an error that satisfies `update.ErrLocalDowngrade` and names the install location, both
revisions, the module root and `--allow-downgrade`, without running `go install`. It MUST NOT run `go
version -m` when no file exists at the install location.

Supports: [792a11c1a A provable downgrade is refused](../product/install-and-update.md#792a11c1-uat-scenarios), [792a11c1b The flag bypasses the check](../product/install-and-update.md#792a11c1-uat-scenarios), [792a11c1c A first install is not checked](../product/install-and-update.md#792a11c1-uat-scenarios), [792a11c1d An unknown installed revision does not block](../product/install-and-update.md#792a11c1-uat-scenarios), [792a11c1e An unreadable revision does not block](../product/install-and-update.md#792a11c1-uat-scenarios).

Rationale: `git merge-base --is-ancestor` exits non-zero both for "not an ancestor" and, with `fatal:`, for
"cannot tell"; only the first is proof. The remote route is exempt, because a fresh clone always builds
the repository's current tip.

#### 792b28c7a Scenarios

```gherkin
Scenario: 792b28c7a1 A provable downgrade is refused
  Given a binary at the install location built at revision deadbeef, and the ancestry check exits non-zero without a fatal error
  When the update installs locally
  Then it fails with ErrLocalDowngrade naming deadbeef, the resolved revision and --allow-downgrade, and go install never runs

Scenario: 792b28c7a2 The flag bypasses the check
  Given the same conditions and --allow-downgrade
  When the update installs locally
  Then go install runs and the ancestry check never runs

Scenario: 792b28c7a3 No prior binary means no check
  Given no file at the install location
  When the update installs locally
  Then go install runs and go version -m never runs

Scenario: 792b28c7a4 An unknown installed revision fails open
  Given the ancestry check exits non-zero with "fatal: Not a valid commit name" in its error output
  When the update installs locally
  Then go install runs and the report keeps the resolved revision

Scenario: 792b28c7a5 An unparseable revision fails open
  Given go version -m prints no vcs.revision line
  When the update installs locally
  Then go install runs and the ancestry check never runs

Scenario: 792b28c7a6 A descendant installs
  Given the ancestry check succeeds
  When the update installs locally
  Then go install runs
```

### 792b28c8 Installs that are skipped

Requirement: [792b28c8 Installs that are skipped](#792b28c8-installs-that-are-skipped) The self-installer MUST run no `go install`, and MUST NOT hand off ([792b28c9 Hand-off](#792b28c9-hand-off)), when
the run is a dry run or when the environment variable `ENGRAM_UPDATE_REEXEC` is non-empty (the
handed-off run), and MUST still resolve the local revision, or clone and check the model for a remote
source, so that the report names the source. It MUST then plan and deploy in this process. It MUST mark
the report of a handed-off run as such, and MUST NOT mark the report of an installing run.

Supports: [792a11e1d A handed-off run does not claim an install](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1e A dry run never hands off](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11f1c A dry run installs nothing](../product/install-and-update.md#792a11f1-uat-scenarios).

Rationale: the sentinel is what ends the loop: a handed-off run that installed again would hand off
again.

#### 792b28c8a Scenarios

```gherkin
Scenario: 792b28c8a1 A dry run installs and spawns nothing
  Given a local source
  When the update runs with --dry-run
  Then the only command run is git rev-parse --short HEAD, no process is spawned, and no hand-off exit code is recorded

Scenario: 792b28c8a2 The handed-off run installs nothing and spawns nothing
  Given ENGRAM_UPDATE_REEXEC is set
  When the update runs
  Then only git rev-parse runs, no process is spawned, and the deploy completes in this process

Scenario: 792b28c8a3 The handed-off run's report says so
  Given ENGRAM_UPDATE_REEXEC is set
  When the update runs
  Then its report is marked as a handed-off run's

Scenario: 792b28c8a4 A remote dry run still clones
  Given no local source
  When the update runs with --dry-run
  Then the repository is cloned and its model checked, and go install does not run
```

### 792b28c9 Hand-off

Requirement: [792b28c9 Hand-off](#792b28c9-hand-off) After an install that succeeded, unless [792b28c8 Installs that are skipped](#792b28c8-installs-that-are-skipped) applies, the self-installer MUST
first have the install result written through the injected hand-off reporter, MUST then start the
installed binary at the install location (never the running process's own path) with the arguments
`update` followed by the forwarded flags and with `ENGRAM_UPDATE_REEXEC=1` added to the environment, MUST
record the started binary's exit code and return without planning or deploying, and MUST NOT mark its
own report as a handed-off run's. When the binary cannot be started it MUST record `re-exec failed,
completed with pre-update logic: <error>` on the report and complete the deploy in this process. When the
install result cannot be written it MUST fail with an error saying `writing re-exec handoff report` and
MUST NOT start the binary or fall back.

Supports: [792a11e1a The deploy runs in the new binary](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1b The install result comes first](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1c A failed hand-off falls back](../product/install-and-update.md#792a11e1-uat-scenarios).

Rationale: a failed write to the output means the same broken output would fail the fallback's report, so
smoothing it into the fallback only hides it.

#### 792b28c9a Scenarios

```gherkin
Scenario: 792b28c9a1 The installed binary is started with the flags and the sentinel
  Given an install that succeeded and the flag --with-guidance
  When the update hands off
  Then the install location is started with the arguments update --with-guidance and ENGRAM_UPDATE_REEXEC=1 in its environment, the child's exit code 7 is recorded, and no harness is deployed by this process

Scenario: 792b28c9a2 The install result is written before the spawn
  Given an install that succeeded and a hand-off reporter
  When the update hands off
  Then the reporter is called before the spawner

Scenario: 792b28c9a3 The parent's report is not a handed-off run's
  Given an install that succeeded
  When the hand-off reporter receives the report
  Then the report is not marked as a handed-off run's

Scenario: 792b28c9a4 A binary that cannot start falls back
  Given an install that succeeded and a spawner that fails to start the binary
  When the update hands off
  Then no exit code is recorded, the deploy completes here, and the report says "re-exec failed, completed with pre-update logic:" with the error

Scenario: 792b28c9a5 A report that cannot be written aborts
  Given a hand-off reporter that fails with "write: broken pipe"
  When the update hands off
  Then the run fails with an error containing "writing re-exec handoff report:", no process is spawned, there is no fallback note, and no harness is deployed
```

### 792b28c10 The update run and its flags

Requirement: [792b28c10 The update run and its flags](#792b28c10-the-update-run-and-its-flags) The self-installer MUST, when `--reparent-luhmann` is given, run only the
re-parenting of note-reparenting ([792b27c3 Derive output](./note-reparenting.md#792b27c3-derive-output), [792b27c9 Apply flow](./note-reparenting.md#792b27c9-apply-flow), [792b27c10 Preview](./note-reparenting.md#792b27c10-preview)) against the vault and chunk index resolved from the home
directory and environment, writing its output to the command's output, with no harness detection, install
or deploy, and MUST fail with an error naming `update --reparent-luhmann: resolving home` when the home
cannot be resolved. Otherwise it MUST run the update with `--with-guidance` and `--allow-downgrade`
carried into its options, MUST forward `--with-guidance`, `--regen-vocab` and `--backfill-identity`, and
no other flag, to the handed-off run, and MUST NOT call the re-parenting. After a hand-off it MUST call the
injected exit function once with the child's exit code, run none of the checks of [792b28c11 Checks after the deploy](#792b28c11-checks-after-the-deploy) and write no
second report; after a fallback it MUST NOT call the exit function. Its help MUST describe the command and
list its seven flags.

Supports: [792a11o1a The help lists every option](../product/install-and-update.md#792a11o1-uat-scenarios), [792a11o1b Options reach the handed-off run](../product/install-and-update.md#792a11o1-uat-scenarios), [792a11o1c Re-parenting skips the install and the deploy](../product/install-and-update.md#792a11o1-uat-scenarios), [792a11e1a The deploy runs in the new binary](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1b The install result comes first](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1c A failed hand-off falls back](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1d A handed-off run does not claim an install](../product/install-and-update.md#792a11e1-uat-scenarios), [792a10b1a Plain update never re-parents](../product/vault-migration.md#792a10b1-uat-scenarios), [792a10b1b The flag runs nothing else](../product/vault-migration.md#792a10b1-uat-scenarios).

Rationale: the re-parenting is a one-shot batch whose answers come from an agent, so it stays apart from
the routine refresh instead of running as one more step of it.

#### 792b28c10a Scenarios

```gherkin
Scenario: 792b28c10a1 The re-parenting flag runs nothing else
  Given deps whose filesystem and commander are nil and a vault of one top-level note
  When the update runs with --reparent-luhmann
  Then it prints "no candidates found" and neither the install nor the deploy runs

Scenario: 792b28c10a2 The flags forwarded to the hand-off
  Given --with-guidance, --regen-vocab, --backfill-identity, --allow-downgrade and --dry-run
  When the arguments of the handed-off run are built
  Then they are --with-guidance, --regen-vocab and --backfill-identity

Scenario: 792b28c10a3 The guidance flag reaches the options
  Given --with-guidance and --dry-run
  When the update runs
  Then the updater's WithGuidance option is set

Scenario: 792b28c10a4 After a hand-off the parent exits with the child's code
  Given a hand-off whose child exits 9
  When the update runs
  Then the exit function is called once with 9, no harness section or checks are printed, and "engram update" appears once in the output

Scenario: 792b28c10a5 The header precedes the spawn
  Given a hand-off
  When the child is started
  Then the parent's header is already on the output

Scenario: 792b28c10a6 A fallback does not exit early
  Given a spawner that fails to start the binary
  When the update runs
  Then the exit function is not called and the output holds the fallback note and an installed line

Scenario: 792b28c10a7 The handed-off run prints no header
  Given ENGRAM_UPDATE_REEXEC is set
  When the update runs
  Then the output holds no "engram update" header and no "go install" claim, and holds the installed line

Scenario: 792b28c10a8 A plain update never re-parents
  Given a vault of similar top-level notes
  When the update runs without --reparent-luhmann
  Then the re-parenting is not called

Scenario: 792b28c10a9 The help lists every flag
  Given any machine
  When the user runs engram update --help
  Then it describes refreshing the binary and harness skills and lists --dry-run, --with-guidance, --allow-downgrade, --regen-vocab, --reparent-luhmann, --answers and --backfill-identity
```

### 792b28c11 Checks after the deploy

Requirement: [792b28c11 Checks after the deploy](#792b28c11-checks-after-the-deploy) When the deploy ran in this process and `Updater.Run` returned no error, the
self-installer MUST resolve the vault path and the chunk-index directory from the home directory and the
environment, record on the report whether the vault holds old-format vocabulary files, untagged
definition self-tags, only top-level notes, notes missing identity, pending offers and an uncommitted
vault ID, and whether the chunk index holds empty files or a duplicate backlog `prune --duplicates` would
remove, and MUST read nothing outside the resolved home, vault and chunk-index directory. It MUST then
record the outbox notice, apply `--regen-vocab` and `--backfill-identity` (honoring `--dry-run`) and run
the registration step, and MUST fail with `update: <error>` when a requested regeneration or backfill
fails. It MUST run none of this when the run failed, and a failing detection MUST leave its flag clear
rather than fail the update.

Supports: [792a11n1a Update reports the prune command when duplicates exist](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1b A refusal-only backlog is silent](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1c Empty index files are reported](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1d A clean or missing index is silent](../product/install-and-update.md#792a11n1-uat-scenarios), [792a6w1a An old-format vocabulary is migrated](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1b A dry run changes nothing](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1c Nothing to migrate](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a10k1a The current environment is stamped](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1f A dry run writes nothing](../product/vault-migration.md#792a10k1-uat-scenarios).

Rationale: every check reads the machine and the vault as they stand after the deploy, in the release that
deployed, and an unreadable path means no signal, since a detection failure must never fail `engram update`'s
primary job.

#### 792b28c11a Scenarios

```gherkin
Scenario: 792b28c11a1 The regeneration flag is applied and summarized
  Given a vault with one old-format vocabulary file and --regen-vocab with --dry-run
  When the update runs
  Then the output contains "[dry-run] update --regen-vocab: would remove 1 old-format file(s)" and not the old-format notice

Scenario: 792b28c11a2 The backfill flag is applied and summarized
  Given a vault with one note missing identity and --backfill-identity with --dry-run
  When the update runs
  Then the output contains "[dry-run] update --backfill-identity: would stamp 1 note(s)" and not the identity notice

Scenario: 792b28c11a3 A dry run reads only the resolved home
  Given a dry run with the home directory pinned to a scratch directory
  When the update runs
  Then no path the filesystem sees lies under the real home's data or Claude directories

Scenario: 792b28c11a4 A failed run runs no checks
  Given Updater.Run returned an error
  When the update finishes
  Then no check ran and the error is returned wrapped
```

### 792b28c12 Report

Requirement: [792b28c12 Report](#792b28c12-report) The self-installer MUST write the report as the lines `engram update`,
`  source: <source>`, `  binary: <binary>`, then (when set) the re-exec failure note, then each harness
section, which opens with `  <harness> (~/<probe dir>/):`, then `installed: <names>` listing the harnesses without an error, then the guidance hints and the
notices ([792b28c13 Notices](#792b28c13-notices)). The source MUST be `local clone at <root>` with ` (rev <revision>)` when a revision
is known, `remote module github.com/toejough/engram <revision>`, or `unknown`. The binary MUST be the
install command for a dry run, else the install command followed by ` ... ok (engram <revision> →
<install location>)` or `(engram → …)` when the revision is empty. It MUST show a path under the home
directory as `~/…`, MUST omit the three-line header from the report of a handed-off run, MUST write only
the header (no harness sections, notices or installed line) as the parent's hand-off report, MUST print
`error: <reason>` under a harness that failed and omit it from the installed line, and MUST omit the
installed line when no harness succeeded. In a dry run it MUST prefix `[dry-run] ` to the title and the
installed line and to each other line that describes something done or that would be done, and MUST NOT
prefix a notice line.

Supports: [792a11d1a A local install reports its revision](../product/install-and-update.md#792a11d1-uat-scenarios), [792a11d1b A real install reports where the binary went](../product/install-and-update.md#792a11d1-uat-scenarios), [792a11d1c A remote source reports its revision](../product/install-and-update.md#792a11d1-uat-scenarios), [792a11e1b The install result comes first](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1d A handed-off run does not claim an install](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11f1b Action lines carry the prefix, notices do not](../product/install-and-update.md#792a11f1-uat-scenarios), [792a11g1d One failing harness does not stop the other](../product/install-and-update.md#792a11g1-uat-scenarios).

#### 792b28c12a Scenarios

```gherkin
Scenario: 792b28c12a1 A dry run with both harnesses
  Given a dry-run report for a local source and both harnesses
  When the report is written
  Then it holds "[dry-run] engram update", "source: local clone at ~/src/engram", "binary: go install ./cmd/engram/", a section for each harness and "[dry-run] installed: Claude Code, Pi"

Scenario: 792b28c12a2 A local source shows its revision
  Given a report whose local source has revision abc1234
  When the report is written
  Then the source line is "local clone at ~/src/engram (rev abc1234)"

Scenario: 792b28c12a3 A real run without a revision
  Given a real-run report with an empty revision
  When the report is written
  Then the binary line is "go install ./cmd/engram/ ... ok (engram → ~/go/bin/engram)"

Scenario: 792b28c12a4 A remote source with a revision
  Given a real-run report for a remote source at v0.2.0 installed to /home/joe/go/bin/engram
  When the report is written
  Then the source line is "remote module github.com/toejough/engram v0.2.0" and the binary line ends "ok (engram v0.2.0 → ~/go/bin/engram)"

Scenario: 792b28c12a5 An unknown source mode
  Given a report whose source mode is not set
  When the report is written
  Then the source line says unknown

Scenario: 792b28c12a6 Paths under the home directory are shortened
  Given paths /home/joe/x, /other/x, and /home/joe/x with an empty home
  When they are shown with the home /home/joe
  Then they read ~/x, /other/x and /home/joe/x

Scenario: 792b28c12a7 A failing harness is shown and left out of the installed line
  Given a report in which the only harness failed with "disk full"
  When the report is written
  Then it holds "error: disk full" and no installed line

Scenario: 792b28c12a8 A dry run from a checkout names a local source
  Given a dry run from a working directory inside the repository
  When the update runs
  Then the output holds "[dry-run] engram update" and "source: local clone at "

Scenario: 792b28c12a9 Action lines carry the dry-run prefix and notices do not
  Given a dry-run report with a deletion, an adoption, a dangling-link cleanup, a refreshed and a deployed guidance file, a refused deletion, an unattributable file and an unmanaged entry
  When the report is written
  Then every action line begins with "[dry-run] " and none of the notice lines does
```

### 792b28c13 Notices

Requirement: [792b28c13 Notices](#792b28c13-notices) The self-installer MUST print each notice below, in this order and after the
`installed:` line and the guidance hints, exactly when the report's flag for it is set, and MUST print
nothing for a clear flag: the old-format vocabulary notice, the missing self-tag notice, the Luhmann
branching notice, the empty chunk-index notice, the duplicates notice and the identity-backfill notice
(each with the text its capability specifies: [792a6v Update notices](../product/controlled-vocabulary.md#792a6v-update-notices), [792a10a The flat-vault notice](../product/vault-migration.md#792a10a-the-flat-vault-notice), [792a10j The identity-backfill notice](../product/vault-migration.md#792a10j-the-identity-backfill-notice) and [792a11n Backlog notices for the chunk index](../product/install-and-update.md#792a11n-backlog-notices-for-the-chunk-index)), then the pending-offer
notice, the vault-ID notice ([792b5c13 Uncommitted ID detection](./vault-identity-registry.md#792b5c13-uncommitted-id-detection)), the outbox notice and the skill-registration error line, which
their capabilities specify. The vocabulary and identity notices MUST be replaced by their summaries
([792b28c14 Summaries of the opt-in flags](#792b28c14-summaries-of-the-opt-in-flags)) on a run that applied the matching flag. Notices MUST NOT mention a README section.

Supports: [792a6v1a An old-format vault is flagged](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a6v1b A missing self-tag is flagged](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a6v1c A current vault gets no vocab notice](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a10a1a Only top-level notes](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10a1b A branched note silences the notice](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10a1c An empty vault has no notice](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10a1d The notice names the answering procedure](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10j1a Notes missing identity are noticed](../product/vault-migration.md#792a10j1-uat-scenarios), [792a10j1b No notice when nothing is missing](../product/vault-migration.md#792a10j1-uat-scenarios), [792a11n1a Update reports the prune command when duplicates exist](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1b A refusal-only backlog is silent](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1c Empty index files are reported](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1d A clean or missing index is silent](../product/install-and-update.md#792a11n1-uat-scenarios), [792a11n1e The notices coexist with the other upgrade notices](../product/install-and-update.md#792a11n1-uat-scenarios), [792a21za1a Update reports a stuck outbox](../product/parent-sync.md#792a21za1-uat-scenarios), [792a21za1b Update reports an entry that needs attention](../product/parent-sync.md#792a21za1-uat-scenarios), [792a22o1g Update surfaces a notice](../product/curate.md#792a22o1-uat-scenarios).

#### 792b28c13a Scenarios

```gherkin
Scenario: 792b28c13a1 The old-format vocabulary notice
  Given a report flagged for old-format vocabulary files, then one that is not
  When each is written
  Then the first names "run `engram update --regen-vocab`" and neither mentions Upgrading or README.md, and the second has no "old-format vocab"

Scenario: 792b28c13a2 The missing self-tag notice
  Given a report flagged for untagged vocabulary definitions, then one that is not
  When each is written
  Then the first names `engram vocab tag-definitions` and the second prints no such notice

Scenario: 792b28c13a3 The Luhmann branching notice
  Given a report flagged for a vault holding only top-level notes, then one that is not
  When each is written
  Then the first names `engram update --reparent-luhmann` and the learn skill's batch mode and the second prints no such notice

Scenario: 792b28c13a4 The empty chunk-index notice
  Given a report flagged for empty chunk-index files, then one that is not
  When each is written
  Then the first names `engram prune --empty` without mentioning Upgrading or README.md, and the second prints no such notice

Scenario: 792b28c13a5 The duplicates notice
  Given a report flagged for a prunable duplicate backlog, then one that is not
  When each is written
  Then the first names `engram prune --duplicates` without mentioning Upgrading or README.md, and the second prints no such notice

Scenario: 792b28c13a6 The identity-backfill notice
  Given a report flagged for notes missing identity, then one that is not
  When each is written
  Then the first names `engram update --backfill-identity` and the second prints no such notice

Scenario: 792b28c13a7 Notices coexist
  Given a report flagged for both chunk-index backlogs and for old-format vocabulary files
  When it is written
  Then it holds the empty-files, duplicates and old-format notices

Scenario: 792b28c13a8 The notices follow a fixed order
  Given a report with every notice flagged
  When it is written
  Then the notices appear in the order vocabulary, self-tag, Luhmann, empty files, duplicates, identity, pending offers, vault ID, outbox, registration error

Scenario: 792b28c13a9 The outbox notice is written
  Given a report that carries an outbox notice
  When it is written
  Then the output contains the notice text

Scenario: 792b28c13a10 The pending-offer notice is written
  Given a report flagged for pending offers, then one that is not
  When each is written
  Then the first names the pending_offers flag and carries the shared curate instruction and its engram query command, without naming a skill, and the second prints no pending-offer line
```

### 792b28c14 Summaries of the opt-in flags

Requirement: [792b28c14 Summaries of the opt-in flags](#792b28c14-summaries-of-the-opt-in-flags) When `--regen-vocab` ran, the self-installer MUST print its summary in place of the
old-format notice: `update --regen-vocab: nothing to regenerate` when it removed no file and cleaned no
member note, else `update --regen-vocab: removed <F> old-format file(s), cleaned <C> member note(s), seeded
<S> term(s), reassigned <A> note(s)`, saying `would remove` for a dry run and prefixing `[dry-run] `. When
`--backfill-identity` ran it MUST print `update --backfill-identity: nothing to backfill` when it stamped
no note, else `update --backfill-identity: stamped <N> note(s) with repo:/user:/vault: provenance`, with
`would stamp` and the prefix for a dry run, in place of the identity notice.

Supports: [792a6w1a An old-format vocabulary is migrated](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1b A dry run changes nothing](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1c Nothing to migrate](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a10k1a The current environment is stamped](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1f A dry run writes nothing](../product/vault-migration.md#792a10k1-uat-scenarios).

#### 792b28c14a Scenarios

```gherkin
Scenario: 792b28c14a1 A regeneration is summarized
  Given a report of a --regen-vocab run that removed 2 files, cleaned 3 member notes, seeded 1 term and reassigned 4 notes
  When the report is written
  Then it holds "update --regen-vocab: removed 2 old-format file(s), cleaned 3 member note(s), seeded 1 term(s), reassigned 4 note(s)" and not the old-format notice

Scenario: 792b28c14a2 A dry-run regeneration is phrased as a preview
  Given the same counts from a dry run
  When the report is written
  Then it holds "[dry-run] update --regen-vocab: would remove 2 old-format file(s)"

Scenario: 792b28c14a3 A regeneration with nothing to do
  Given a --regen-vocab run that removed and cleaned nothing
  When the report is written
  Then it holds "update --regen-vocab: nothing to regenerate"

Scenario: 792b28c14a4 A backfill is summarized
  Given a --backfill-identity run that stamped 3 notes
  When the report is written
  Then it holds "update --backfill-identity: stamped 3 note(s) with repo:/user:/vault: provenance" and not the identity notice

Scenario: 792b28c14a5 A dry-run backfill is phrased as a preview
  Given a --backfill-identity dry run that would stamp 3 notes
  When the report is written
  Then it holds "[dry-run] update --backfill-identity: would stamp 3 note(s)"

Scenario: 792b28c14a6 A backfill with nothing to do
  Given a --backfill-identity run that stamped nothing
  When the report is written
  Then it holds "update --backfill-identity: nothing to backfill"
```

### 792b28c15 Exit status and errors

Requirement: [792b28c15 Exit status and errors](#792b28c15-exit-status-and-errors) The self-installer MUST return an error `update: <error>` that preserves the error
chain, without writing the report, when the run failed, MUST return `update: writing report: <error>` when
the report cannot be written, MUST return the error "update: one or more detected harnesses failed" after
writing the report when any harness failed, and MUST otherwise return no error.

Supports: [792a11g1d One failing harness does not stop the other](../product/install-and-update.md#792a11g1-uat-scenarios), [792a11a1e Go is missing](../product/install-and-update.md#792a11a1-uat-scenarios), [792a11g1c No harness is an error before any install](../product/install-and-update.md#792a11g1-uat-scenarios).

#### 792b28c15a Scenarios

```gherkin
Scenario: 792b28c15a1 A harness failure is an error
  Given a report in which one harness failed, then one in which none did
  When the update finishes
  Then the first returns the harnesses-failed error and the second returns none

Scenario: 792b28c15a2 A partial failure is an error
  Given one harness that failed and one that succeeded
  When the update finishes
  Then the harnesses-failed error is returned

Scenario: 792b28c15a3 A run error is wrapped
  Given a run that failed with an error
  When the update finishes
  Then the returned error is "update: " plus that error and the original error is still matched by errors.Is

Scenario: 792b28c15a4 A clean run succeeds
  Given a report in which every harness succeeded
  When the update finishes
  Then no error is returned

Scenario: 792b28c15a5 The exit status follows the harnesses
  Given any combination of harness outcomes
  When the update finishes
  Then an error is returned exactly when at least one harness failed

Scenario: 792b28c15a6 A missing skills source keeps its typed error
  Given a harness and no skills source directory
  When the update runs and finishes
  Then the error satisfies ErrSkillsSrcMissing after the report tail's wrapping

Scenario: 792b28c15a7 A report that cannot be written is an error
  Given an output that fails on write
  When the update finishes
  Then the returned error begins "update: writing report:"
```

### 792b28c16 The registration step

Requirement: [792b28c16 The registration step](#792b28c16-the-registration-step) After the checks and the opt-in summaries, the self-installer MUST run skill
registration over the resolved vault and vault name, with the home directory, the update's `--dry-run`
flag and no explicit answers, and with its output going straight to the command's output. It MUST do
nothing when no skill-source filesystem is configured. It MUST record a registration failure on the report
as `skill registration error: <message>` ([792b28c13 Notices](#792b28c13-notices)) and MUST NOT let it fail the checks or the update.

Supports: [792a12t1a An interactive update](../product/skill-runbook-registration.md#792a12t1-uat-scenarios), [792a12t1b A non-interactive update](../product/skill-runbook-registration.md#792a12t1-uat-scenarios), [792a12t1c A dry run previews the offers too](../product/skill-runbook-registration.md#792a12t1-uat-scenarios), [792a12t1d Update and register-skills see the same offers](../product/skill-runbook-registration.md#792a12t1-uat-scenarios), [792a12t1e A registration failure does not roll back the deploy](../product/skill-runbook-registration.md#792a12t1-uat-scenarios).

Rationale: the step reads the freshly deployed copies and the user's own folders, and a failure there must
never undo a deploy that already succeeded.

#### 792b28c16a Scenarios

```gherkin
Scenario: 792b28c16a1 The step forwards the dry-run flag and reads the default sources
  Given a Claude user skill and a source checkout whose agent-instructions/skills holds other skills
  When the step runs with --dry-run
  Then the user skill is offered as a preview, the checkout's skills are never read, and nothing is written

Scenario: 792b28c16a2 Without a terminal the step only reports
  Given stdin is not a terminal and one skill has no note
  When the step runs
  Then nothing is written and the output names the skill's scope

Scenario: 792b28c16a3 An unconfigured step does nothing
  Given a zero-value registration configuration
  When the step runs
  Then it returns without error and writes nothing

Scenario: 792b28c16a4 The step returns a registration failure
  Given registration fails
  When the step runs
  Then the failure is returned to its caller

Scenario: 792b28c16a5 A failure is recorded and does not fail the checks
  Given registration fails during the checks
  When the checks run
  Then the report holds the failure message and the checks return no error

Scenario: 792b28c16a6 The report line appears only on failure
  Given a report with a registration failure "boom", then one without
  When each is written
  Then the first holds "skill registration error: boom" and the second holds no such line
```
