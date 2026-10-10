# Subprocess Runner - Technical Spec

## 792b4a Orientation

- Ownership: Repository-owned dependency-boundary interaction (operating-system processes).
- Responsibility: runs an external program, such as `git` or `go`, in a chosen directory and hands
  back what it printed, or reports why it failed, telling "program not installed" apart from other
  failures.
- Current realization (informational): `primCommander` in `internal/cli/commander.go` (the
  `update.Commander` interface), over the raw `ExecPrims` that `cmd/engram/main.go` supplies.
  `primSpawner` in `internal/cli/spawner.go` runs a child with inherited stdio ([792b4c4 Run with the caller's streams](#792b4c4-run-with-the-callers-streams)), over the
  `spawnPrimitives` that `cmd/engram/main.go` supplies.
- Qualification: responsibility, running a program and collecting its output is stated without
  naming any caller; ownership, callers choose the program and interpret its output, while this
  component owns collection, error context and not-found translation; independent verification, it
  is tested directly over a fake run primitive and over real processes; stability, the contract
  survives any change to the injection seam; necessity, the uncommitted vault-ID notice and note
  provenance depend on running `git`.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-writing](../product/note-writing.md), [install-and-update](../product/install-and-update.md).

## 792b4c Behavior Requirements

### 792b4c1 Run and collect output

Requirement: [792b4c1 Run and collect output](#792b4c1-run-and-collect-output) The subprocess-runner MUST run the named program with the given arguments in
the given directory, or in the process's working directory when none is given, and MUST return
everything the program wrote to stdout and stderr with no error when it exits successfully.

Supports: [792a2i1a Uncommitted ID is flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a2i1b An ignored ID is flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a2i1c A committed ID is not flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a3o1a The origin remote names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1b Without an origin the top-level directory names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1c Outside a git repository there is no repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3p1a The git email names the user](../product/note-writing.md#792a3p1-uat-scenarios), [792a3p1b The OS username is the fallback](../product/note-writing.md#792a3p1-uat-scenarios).

#### 792b4c1a Scenarios

```gherkin
Scenario: 792b4c1a1 Output is collected on success
  Given a program that writes to stdout and stderr and exits 0
  When it is run
  Then its stdout and stderr are returned and there is no error

Scenario: 792b4c1a2 The program runs in the given directory
  Given a directory
  When pwd -P is run in it
  Then the output is that directory's resolved path, and the program, arguments and directory reach the run primitive unchanged
```

### 792b4c2 Failure reporting

Requirement: [792b4c2 Failure reporting](#792b4c2-failure-reporting) When the program cannot run or exits unsuccessfully, the subprocess-runner MUST
return an error that names the program and its arguments and wraps the underlying failure, along
with whatever output was captured.

Supports: [792a2i1d A vault outside git is not flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a3o1c Outside a git repository there is no repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3p1b The OS username is the fallback](../product/note-writing.md#792a3p1-uat-scenarios), [792a3p1c No detectable user](../product/note-writing.md#792a3p1-uat-scenarios).

#### 792b4c2a Scenarios

```gherkin
Scenario: 792b4c2a1 A failing program is reported with its output
  Given a program that writes output and then fails
  When it is run
  Then an error naming the program and arguments is returned, and the captured output is kept
```

### 792b4c3 Not-found translation

Requirement: [792b4c3 Not-found translation](#792b4c3-not-found-translation) When the failure is the platform's "program not found" error, the
subprocess-runner MUST also make the error satisfy `errors.Is(err, update.ErrCommandNotFound)`, and
it MUST NOT do so for any other failure, including when no platform not-found error is configured.

Supports: [792a2i1d A vault outside git is not flagged](../product/vault-identity.md#792a2i1-uat-scenarios).

Rationale: callers such as the [self-installer](./self-installer.md) report a missing `git` or `go` differently from a
failed one, without knowing the platform's error value.

#### 792b4c3a Scenarios

```gherkin
Scenario: 792b4c3a1 A missing program is reported as not found
  Given no program by the given name is installed
  When it is run
  Then the error satisfies errors.Is for the command-not-found error and still wraps the platform error

Scenario: 792b4c3a2 Without a configured not-found error nothing is translated
  Given no platform not-found error is configured and the program fails to start
  When it is run
  Then the error wraps the failure and does not satisfy errors.Is for the command-not-found error
```

### 792b4c4 Run with the caller's streams

Requirement: [792b4c4 Run with the caller's streams](#792b4c4-run-with-the-callers-streams) The subprocess-runner MUST run the named program with the given arguments, with the
caller's standard input, output and error as the program's, and with the given extra environment entries
added to the process environment. It MUST return the program's exit status and no error whenever the
program started, including a non-zero status, and MUST return an error that wraps the failure and names
the program, and the status -1, when the program could not be started (missing or not executable).

Supports: [792a11e1a The deploy runs in the new binary](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11e1c A failed hand-off falls back](../product/install-and-update.md#792a11e1-uat-scenarios), [792a11o1b Options reach the handed-off run](../product/install-and-update.md#792a11o1-uat-scenarios).

Rationale: the caller needs to tell a child that ran and failed from a child that never started, because
only the second justifies completing the work itself. A handed-off `engram update` writes its report
straight to the terminal, so the child shares the caller's streams rather than having its output
collected ([792b4c1 Run and collect output](#792b4c1-run-and-collect-output)).

#### 792b4c4a Scenarios

```gherkin
Scenario: 792b4c4a1 The call reaches the run primitive unchanged
  Given a fake run primitive that returns status 0
  When engram is spawned with the arguments update and the extra environment entry ENGRAM_UPDATE_REEXEC=1
  Then the primitive receives that program, those arguments and that entry, and the result is status 0 with no error

Scenario: 792b4c4a2 A non-zero status is a status, not an error
  Given a program that exits with status 7, or a fake primitive that returns 3 with no error
  When it is spawned
  Then the result is that status and no error

Scenario: 792b4c4a3 A clean exit is status zero
  Given a program that exits 0
  When it is spawned
  Then the result is status 0 and no error

Scenario: 792b4c4a4 Arguments and extra environment reach the program
  Given a program that succeeds only when its first argument is arg-one and ENGRAM_TEST_SENTINEL is 1
  When it is spawned with that argument and the extra entry ENGRAM_TEST_SENTINEL=1
  Then it exits 0

Scenario: 792b4c4a5 A program that cannot start is an error
  Given a program that does not exist, or a fake primitive that returns -1 and an error
  When it is spawned
  Then the result is an error, never a status, and the error wraps the primitive's failure and the status is -1

Scenario: 792b4c4a6 The program shares the caller's streams
  Given a program that reads its standard input and writes to its standard output and standard error
  When it is spawned
  Then its output appears on the caller's standard output and error, and it reads the caller's standard input
```
