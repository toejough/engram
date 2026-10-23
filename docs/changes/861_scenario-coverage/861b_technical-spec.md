# Scenario Coverage - Technical Spec

This change adds tests for unchanged, documented behavior and one development target: it selects no other design and changes no contract, so it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./861d_questions-and-decisions.md#questions-and-decisions).

## 861b1 Existing-system orientation

The product problem is [861a1 Problem statement](./861a_product-spec.md#861a1-problem-statement). The software in scope is the development tooling of the `dev` package: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration) owns the registration of Engram's [Targ](../../GLOSSARY.md#targ) targets in `dev/targs.go`; its Responsibility names its nil, aggregate and watch targets only, so [861b3a Development Check Orchestration](#861b3a-modified-development-check-orchestration) adds the new target to it. Today `targ test` takes no argument, runs the whole module with `-race` and a coverage profile, and the Project Profile forbids `go test` outside `targ`; no target takes a package, and the only target with a test filter, `acceptance` (`--run`), runs the fixed live-session package. Insufficiency: a builder cannot run the tests of the package it touched, and up to four builders at once would run four concurrent race and coverage builds of the whole module. Recommendation: one target, `test-pkg`, modeled on `acceptance` ([861b2a A scoped test run](#861b2a-added-a-scoped-test-run)).

## 861b2 Added: Scoped test run

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the operations below are applied at promotion ([861d15 861 merges without promotion](./861d_questions-and-decisions.md#questions-and-decisions), [861d28 Promotion order of 861, 862 and 858](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861b2a Added: A scoped test run

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points).

Requirement: 861b2a Development-check orchestration MUST register a target `test-pkg` that takes one package path, an optional `--run` regular expression, `--no-race` and an optional `--tags` value, and runs `go test -count=1 -timeout=2m -race` on that package, omitting `-race` with `--no-race`, adding `-tags=<value>` when a value is given and `-run <expression>` when an expression is given. It MUST add no build tag unless `--tags` is given, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: A developer needs the tests of the package they touched without running the whole module. Tagged `dev` tests run through `test-dev` and `test-dev-integration`. A per-unit run skips the race detector and runs the tagged tests of one package; the batch-end gates keep `-race` and the integration legs ([861d400 Per-unit runs skip the race detector and the integration suites](./861d_questions-and-decisions.md#questions-and-decisions)).

Supports [861a3a1a A developer runs the tests of one package](./861a_product-spec.md#861a3a1-uat-scenarios).

#### 861b2a1 Contract Scenarios

```gherkin
Scenario: 861b2a1a The package path is required
  Given the development targets
  When the developer runs test-pkg with no package path
  Then it reports the missing package path and starts no go test

Scenario: 861b2a1b A filter narrows the run
  Given the development targets
  When the developer runs test-pkg with a package path and --run with an expression
  Then go test receives -run with that expression before the package path

Scenario: 861b2a1c The command line is pinned
  Given the development targets
  When the developer runs test-pkg with a package path and no --run
  Then go test is run with -count=1, -timeout=2m, -race and that package path, and with no other flag

Scenario: 861b2a1d A failing test fails the target
  Given a package whose tests fail
  When the developer runs test-pkg on it
  Then the target fails with the failure of go test and the developer sees the output of go test

Scenario: 861b2a1e --no-race leaves out the race detector
  Given the development targets
  When the developer runs test-pkg with a package path and --no-race
  Then go test is run without -race and with the same flags and the package path otherwise

Scenario: 861b2a1f --tags runs the package under the build tags
  Given the development targets
  When the developer runs test-pkg with a package path and --tags with a value
  Then go test receives -tags=<value> before the filter and the package path
```

### 861b2b Added: The conformance gate covers every root

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [861b2a A scoped test run](#861b2a-added-a-scoped-test-run).

Requirement: 861b2b Development-check orchestration MUST fail `targ check-full` when a scenario of `docs/specs/product` or `docs/specs/technical`, of any root (an ID of three digits, a letter, then letters and digits), has no top-level test function named `Test<ID>_...` and is not on the allowlist. The allowlist MUST be a list of scenario IDs sorted by ID, each with a reason that names a decision row or an issue. The gate MUST fail on an allowlist entry whose scenario no spec holds or whose scenario has a test, and on an entry whose reason names neither a decision row nor an issue or that is out of order or listed twice. It MUST pin no scenario count, and it MUST fail when a retired scenario ID is in a spec again.

Rationale: The gate pinned the scenario counts of two roots, so a new scenario of any other root, or a deleted test of one, passed. The owner chose to widen it to every root once the 792 gaps are closed or ruled ([861d20 Scope of the conformance test](./861d_questions-and-decisions.md#questions-and-decisions)). A scenario held out stays on the allowlist with its reason until it has a test; a stale entry fails so the list cannot outlive its reason. The gate checks the form of a reason (a decision-row ID or an issue number), not that the row or issue exists. The entries for the scenarios of Open rows wait for [861d401 Allowlist entries for the scenarios of Open rows](./861d_questions-and-decisions.md#questions-and-decisions).

Supports [861a1 Problem statement](./861a_product-spec.md#861a1-problem-statement).

#### 861b2b1 Contract Scenarios

```gherkin
Scenario: 861b2b1a A scenario of any root needs a test
  Given scenarios of the roots 792, 793, 855 and 861, some with a test function named for the ID and some without
  When the gate reads them with an empty allowlist
  Then it names each scenario that has no test, whatever its root, in the order of the specs

Scenario: 861b2b1b An allowlisted scenario is not named
  Given scenarios with no test, some of them on the allowlist
  When the gate reads them
  Then it names only the scenarios with no test that are not on the allowlist

Scenario: 861b2b1c An allowlist entry names its reason
  Given an allowlist
  When the gate checks its entries
  Then it names each entry whose reason names neither a decision row nor an issue and each entry that is out of order or listed twice

Scenario: 861b2b1d A stale entry is named
  Given an allowlist and the scenarios of the specs
  When an entry's scenario has a test or is in no spec
  Then the gate names that entry's ID

Scenario: 861b2b1e No count is pinned
  Given specs that gain or lose scenarios, each with a test
  When the gate reads them
  Then it names none, whatever the number of scenarios
```

## 861b3 Modified: Development-check orchestration component

The operation replaces the component block of the canonical file completely. It is written on the block as it stands at the base of this change; [862b3c Development Check Orchestration](../862_test-suite-classification/862b_technical-spec.md#862b3c-modified-development-check-orchestration) is written on the block as this operation leaves it.

### 861b3a Modified: Development Check Orchestration

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration). Differs from the current text in Ownership, Responsibility, Current realization and one added Required behavior bullet.

```markdown
## 794b4 Development Check Orchestration

- Ownership: repository-owned component (the `dev` package); it owns
  [794b4a Preserve entry points](#794b4a-preserve-entry-points),
  [861b2a A scoped test run](#861b2a-a-scoped-test-run) and
  [861b2b The conformance gate covers every root](#861b2b-the-conformance-gate-covers-every-root).
- Responsibility: registers Engram's own nil check and fix targets, its aggregate targets and `watch` in place of
  [Targ](../../GLOSSARY.md#targ)'s shared development targets, under the existing names and with Targ's dependencies,
  order and dependency mode, and registers Targ's other shared targets unchanged. It also registers `test-pkg`, which
  runs the tests of one package ([861b2a A scoped test run](#861b2a-a-scoped-test-run)). Its dev integration tests
  gate the scenarios of every root ([861b2b The conformance gate covers every root](#861b2b-the-conformance-gate-covers-every-root)). As the caller of
  [794b4b Nil check run](./nil-check-run-workflow.md#794b4b-nil-check-run) it obtains the selected mode's arguments from
  the invocation policy, sends one request to the analyzer subprocess and returns its success or its failure without
  retrying.
- Current realization (informational): `registerTargets`, the target variables and `watch` in `dev/targs.go`;
  `testPkg` and its target variable in `dev/targs.go`; `scenarioIDs`, `scenariosNeedingATest` and the allowlist in
  `dev/conformance.go` and `dev/conformance_allowlist.go`; `checkNilsFix`, `checkNilsForFail`, `runNilaway` and
  `runConfiguredNilaway` in `dev/nilaway.go`.
- Qualification: Purpose: the existing check entry points keep their names and reach Engram's nil targets, so that the
  nil check and fix commands report only on Engram's module packages
  ([794a2a Project analysis](../product/development-checks.md#794a2a-project-analysis)). Contract: the names,
  dependencies, order and dependency mode of the registered targets, tested directly against Targ's own targets
  (`dev/nilaway_internal_test.go`, `dev/nilaway_integration_internal_test.go`). Value: Targ's aggregates hold direct
  references to its own nil targets and its `watch` calls its own `check`, so replacing the nil targets alone would
  leave them running Targ's versions; deregistering and re-registering the shared targets together does not.
- Required behavior: Targ (external) is assumed to support customizing its shared development targets by deregistering
  them and registering others under the same names (the Rationale of
  [794b4a Preserve entry points](#794b4a-preserve-entry-points)); authoritative contract: not recorded in this
  repository; compatibility verification: `dev/nilaway_internal_test.go` compares Engram's aggregates with Targ's own
  (`Test794b4a1c`), and `dev/nilaway_integration_internal_test.go` (tags `targ`, `integration`) checks that every Targ
  shared target is kept or replaced (`Test794b4a1d`).
- Required behavior: the Go toolchain (external) is assumed to accept `-count`, `-timeout`, `-race`, `-tags` and `-run` on
  `go test` and to exit non-zero when a test fails; authoritative contract: `go help test`, not recorded in this
  repository; compatibility verification: `Test861b2a1d_` runs `test-pkg` on a fixture package whose test fails.
- Required behavior: [NilAway Invocation Policy](./nilaway-invocation-policy.md) returns the arguments for the selected
  mode ([794b3a Mode and scope](./nilaway-invocation-policy.md#794b3a-mode-and-scope)).
- Required behavior: [NilAway Subprocess](./nilaway-subprocess.md) starts NilAway with the request's arguments and the
  parent environment unchanged and returns NilAway's success or failure
  ([794b4c Run NilAway](./nilaway-subprocess.md#794b4c-run-nilaway)).
- Supported capabilities: [Development Checks](../product/development-checks.md) (the existing entry points reach
  Engram's nil targets, never Targ's; a developer runs the tests of one package, [861a3a1a A developer runs the tests of one package](../product/development-checks.md#861a3a1-uat-scenarios)).
```

## 861b4y Modified: Offer Outbox scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Offer Outbox](../../specs/technical/offer-outbox.md) in the commit that records it.

### 861b4y1 Modified: 792b36c4a3 The vault ID is stamped before the state directory

Target: [792b36c4a Scenarios](../../specs/technical/offer-outbox.md#792b36c4a-scenarios), Scenario 792b36c4a3. Differs from the current text in the Then line: the location record is the file `.engram/home.json`, so it cannot be stamped before `.engram` exists ([861a4y2 792a21q1g The first offer creates the exchange state in the right order](./861a_product-spec.md#861a4y2-modified-792a21q1g-the-first-offer-creates-the-exchange-state-in-the-right-order) is its product form).

```gherkin
Scenario: 792b36c4a3 The vault ID is stamped before the state directory
  Given a vault with no vault ID and no .engram
  When the first entry is queued
  Then the vault ID is stamped before .engram exists, and its location record is then written into .engram
```

## 861b4zzl Modified: Note Provenance scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Note Provenance](../../specs/technical/note-provenance.md) in the commit that records it.

### 861b4zzl1 Modified: 792b11c7a5 Unrelated content is skipped

Target: [792b11c7a Scenarios](../../specs/technical/note-provenance.md#792b11c7a-scenarios), Scenario 792b11c7a5. Differs from the current text in the Given line: a vocab definition note is a fact note ([vocab definition note](../../GLOSSARY.md#vocab-definition-note)) that the backfill stamps, so the note skipped is the [vocab term-note (retired)](../../GLOSSARY.md#vocab-term-note-retired), the shared stale Given of [861a4zzl1 792a10k1g Other note types are left alone](./861a_product-spec.md#861a4zzl1-modified-792a10k1g-other-note-types-are-left-alone) ([861d153 A vocab definition note is a fact, and the backfill stamps it (792a10k1g)](./861d_questions-and-decisions.md#questions-and-decisions)).

```gherkin
Scenario: 792b11c7a5 Unrelated content is skipped
  Given a note without frontmatter and a vocab term-note (retired)
  When the vault is backfilled
  Then neither fails nor changes, and a read error is returned
```

## 861b4zzze Added: Lesson Writing scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Lesson Writing](../../specs/technical/lesson-writing.md) in the commit that records it.

### 861b4zzze1 Added: The reading of the reply of a deep delivery

Target: [855d4j1 Scenarios](../../specs/technical/lesson-writing.md#855d4j1-scenarios), after [855d4j1h A carry that is not of depth deep writes nothing](../../specs/technical/lesson-writing.md#855d4j1-scenarios). The eight tests of `internal/auto/deep_test.go` that named the group without a scenario of it verify what `auto.ReadDeep` reads from a reply, before any note is written; the seven scenarios of the group are those of the write ([861d322 The eight Test855d4j1_ReadDeep functions verify the reading of a deep reply, not a scenario of the write](./861d_questions-and-decisions.md#questions-and-decisions)).

```gherkin
Scenario: 861b4zzze1a A supersession is read as a file name, an object or a list
  Given a carry of depth deep that holds a note, and a reply with a fact lesson whose supersession is, in turn, the note's file name beside the type and the claim of the lesson, an object that holds the note, the type and the claim, and a list of that object
  When the reply is read
  Then the lesson ties the note with the type and the claim, for each of the three

Scenario: 861b4zzze1b A lesson that cannot be read, and has no slug, is left out by its place
  Given a reply with a lesson that has no slug and whose chunks are a string, before a lesson that is whole
  When the reply is read
  Then the whole lesson is read, and the other is left out with a reason that names its place in the list

Scenario: 861b4zzze1c A list of names is the ties of those notes
  Given a carry that holds a note, and a reply with a fact lesson whose supersession is a list of the note's file name, beside a type and a claim
  When the reply is read
  Then the lesson ties the note with the type and the claim

Scenario: 861b4zzze1d A near note is revised by the fields of its kind
  Given a carry that holds a note, in turn a fact, a feedback and a runbook, and a reply that gives it the verdict near with a field its kind has and a field its kind has not
  When the reply is read
  Then the note is amended with the field its kind has, and the field its kind has not is left out and recorded as skipped

Scenario: 861b4zzze1e A reply that is not an object writes nothing and says why
  Given a reply that is a list
  When the reply is read
  Then no lesson is read and no note is amended, and one record, of the kind skipped, says why

Scenario: 861b4zzze1f A reply with no candidates, or cut short, is no deep reply
  Given a reply that is an object with no candidates, and a reply cut in the middle of an entry
  When each reply is read
  Then nothing is read from either, and one record, of the kind skipped, says why for each

Scenario: 861b4zzze1g A runbook lesson is left out and said so
  Given a reply with a lesson of the kind runbook
  When the reply is read
  Then the lesson is not read, and one record, of the kind skipped, names it by its slug and says that a deep delivery writes facts and feedback only

Scenario: 861b4zzze1h A supersession named without its extension, or with spaces, is stored under the note's ID
  Given a carry that holds a note, and a reply with a fact lesson whose supersession names the note without the extension of its file and with spaces around the name
  When the reply is read
  Then the lesson ties the note by its file name in the carry, with the type and the claim
```

## 861b4zzzf Added: Lesson Writing scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Lesson Writing](../../specs/technical/lesson-writing.md) in the commit that records it. The two tests of `internal/cli/auto_write_deep_test.go` that named the group ID without a scenario letter each verify more than one scenario of the group, or a rule none states ([861d327 The first Test855d4j1_ function of internal/cli verifies a deep delivery's failures (node 861c2zzzf)](./861d_questions-and-decisions.md#questions-and-decisions), [861d328 The second Test855d4j1_ function of internal/cli verifies a rule over the reply (node 861c2zzzf)](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861b4zzzf1 Added: Scenarios of the deep write

Target: [855d4j1 Scenarios](../../specs/technical/lesson-writing.md#855d4j1-scenarios), after [855d4j1h A carry that is not of depth deep writes nothing](../../specs/technical/lesson-writing.md#855d4j1-scenarios). Evidence: the tests `Test861b4zzzf1a_WriteDeep_AFailedAmendAndAFailedLearnAreRecordedAndALessonTheVaultStatesIsSkipped` and `Test861b4zzzf1b_WriteDeep_NothingOutsideTheCarryIsWritten` (renamed from `Test855d4j1_WriteDeep_ALessonTheVaultStatesAndAWriteThatFailsAreRecorded` and `Test855d4j1_WriteDeep_NothingOutsideTheCarryIsWritten` in `internal/cli/auto_write_deep_test.go`).

```gherkin
Scenario: 861b4zzzf1a An amend and a learn path that fail are recorded as failed, and a lesson the vault states is skipped
  Given a carry of depth deep that holds a note and a passage, and a reply that gives the note the verdict covered and lists the passage, and holds a lesson the note covers and a whole lesson no note covers, and an amend and a learn path that both fail
  When the writing is given the reply
  Then the command succeeds, and the records are one of the kind skipped for the lesson the note covers and one of the kind failed for each of the amend and the learn path

Scenario: 861b4zzzf1b Nothing outside the carry is written
  Given a carry of depth deep that holds a note and a passage, and any reply: any candidate ID with any verdict, any chunk IDs and no fields, and a lesson with any chunk IDs and any superseded note
  When the writing is given the reply
  Then the one note written has as chunk sources only the passage of the carry, when the lesson lists it, and supersedes at most the note of the carry, and the note of the carry is amended exactly when the reply gives it the verdict covered or near and lists the passage, with that passage as the only chunk source
```

## 861b4g Modified: Command Line Surface scenarios

Applied in place in [Command Line Surface](../../specs/technical/command-line-surface.md) in the commit that records them, as decided in [861d29 stderr of an argument refusal](./861d_questions-and-decisions.md#questions-and-decisions); the stream of the refusal is [targ issue 79](https://github.com/toejough/targ/issues/79).

### 861b4g1 Modified: 792b2c5 Argument refusal

Target: [792b2c5 Argument refusal](../../specs/technical/command-line-surface.md#792b2c5-argument-refusal). Differs from the current text in the stream and the flag naming: the requirement keeps only that the invocation is refused. The Supports line stays.

```markdown
Requirement: 792b2c5 Argument refusal The command-line-surface MUST refuse an invocation that passes a flag the
command does not define, or omits a flag the command marks required, exiting non-zero without running the command.
```

### 861b4g2 Modified: 792b2c5a2 A missing required flag is refused

Target: [792b2c5a Scenarios](../../specs/technical/command-line-surface.md#792b2c5a-scenarios), Scenario 792b2c5a2. Differs from the current text in the Then line: the stream and the flag naming are dropped, the same reduced form as 792b2c5a1.

```gherkin
Scenario: 792b2c5a2 A missing required flag is refused
  Given a command that marks --done-when required
  When it is dispatched without --done-when
  Then the exit status is non-zero and the command does not run
```
