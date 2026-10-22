# Scenario Coverage - Product Spec

## 861a1 Problem statement

The owner and every agent that relies on the specifications cannot tell which documented behavior is verified. Of the 3,562 scenarios in `docs/specs`, 1,655 have no test traced to them, and 19 test functions name an ID that no specification holds. A regression in an untraced scenario goes unseen, and a verified promise cannot be told from an unverified one. The owner needs every documented scenario traced to a test that fails when the promise breaks, or to a recorded reason why none exists, without changing what the software promises.

## 861a2 References

This change adds no outcome. The scenarios in scope belong to the capabilities and components of `docs/specs`; the change serves no outcome of its own, and its one new promise is [861a3 Scoped test runs](#861a3-added-scoped-test-runs). They are listed by owner file in [861c2 Scenario tests by owner file and package](./861c_implementation-plan.md#861c2-scenario-tests-by-owner-file-and-package), with the blocked kinds in [861c3 Blocked nodes](./861c_implementation-plan.md#861c3-blocked-nodes). The gap record this change works from is [858c2a Scenarios without a test, by owner file](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2a-scenarios-without-a-test-by-owner-file) and [858c2b Test functions whose ID is in no spec](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2b-test-functions-whose-id-is-in-no-spec); the issue is [#861](https://github.com/toejough/engram/issues/861).

## 861a3 Added: Scoped test runs

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the operation below is applied at promotion ([861d15 861 merges without promotion](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861a3a Added: A package's tests run alone

Target: [794a2 Development Checks](../../specs/product/development-checks.md#794a2-development-checks), after [794a2c Resource settings pass-through](../../specs/product/development-checks.md#794a2c-resource-settings-pass-through).

Requirement: 861a3a A developer MUST be able to run the tests of one package, with an optional filter on test names, through a development command, without running the tests of the rest of the module.

Rationale: A developer needs the result for the package they touched without waiting for the whole module.

#### 861a3a1 UAT Scenarios

```gherkin
Scenario: 861a3a1a A developer runs the tests of one package
  Given a module with several packages
  When the developer runs the scoped test command on one of them
  Then only the tests of that package run and the command reports their result
```

## 861a4y Modified: Parent Sync scenarios, part 2

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the two below are applied in [Parent Sync](../../specs/product/parent-sync.md) in the commit of their test.

### 861a4y1 Modified: 792a21p1a Placement is not sent

Target: [792a21p1 UAT Scenarios](../../specs/product/parent-sync.md#792a21p1-uat-scenarios), Scenario 792a21p1a. Differs from the current text in the Given line: `--position child` is not a position the CLI accepts.

```gherkin
Scenario: 792a21p1a Placement is not sent
  Given a note learned with --target 12 --position continuation
  When it is offered
  Then the offer request carries no target and no position
```

### 861a4y2 Modified: 792a21q1g The first offer creates the exchange state in the right order

Target: [792a21q1 UAT Scenarios](../../specs/product/parent-sync.md#792a21q1-uat-scenarios), Scenario 792a21q1g. Differs from the current text in the Then line: the location record is the file `.engram/home.json`, so it cannot be stamped before `.engram` exists.

```gherkin
Scenario: 792a21q1g The first offer creates the exchange state in the right order
  Given a vault with no vault ID and no .engram directory
  When the first offer is queued
  Then the vault ID is stamped before .engram exists, and its location record is then written into .engram
```

## 861a4zzk Modified: Vault Migration scenarios, part 2

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Vault Migration](../../specs/product/vault-migration.md) in the commit of its decision.

### 861a4zzk1 Modified: 792a10f1b A failed re-index leaves the rename

Target: [792a10f1 UAT Scenarios](../../specs/product/vault-migration.md#792a10f1-uat-scenarios), Scenario 792a10f1b. Differs from the current text in the Then line: the Given covers any failing chunk-index step, and the failed prune names `engram prune` alone, so the Then names the commands that finish the reconciliation, as the technical 792b27c9 does.

```gherkin
Scenario: 792a10f1b A failed re-index leaves the rename
  Given a rename that succeeded and a chunk-index step that fails
  When the apply finishes
  Then the renamed files stay renamed, and the output names the failed stage and the manual commands that finish the reconciliation
```

## 861a4zw Modified: Skill Runbook Registration scenarios, part 6

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the two below are applied in [Skill Runbook Registration](../../specs/product/skill-runbook-registration.md) in the commit of their test.

### 861a4zw1 Modified: 792a12t1b A non-interactive update

Target: [792a12t1 UAT Scenarios](../../specs/product/skill-runbook-registration.md#792a12t1-uat-scenarios), Scenario 792a12t1b. Differs from the current text in the Then line: the report names the offers by scope selector and count ([792a12g Registration never prompts or writes without a terminal](../../specs/product/skill-runbook-registration.md#792a12g-registration-never-prompts-or-writes-without-a-terminal)), not the skills.

```gherkin
Scenario: 792a12t1b A non-interactive update
  Given `engram update` runs with stdin not a terminal and offers outstanding
  When the user runs engram update
  Then the skills are deployed, no vault note or decline record is written, and the report names the scope selector and count of the offers awaiting an answer
```

### 861a4zw2 Modified: 792a12u1e The identity backfill preserves the identity fields

Target: [792a12u1 UAT Scenarios](../../specs/product/skill-runbook-registration.md#792a12u1-uat-scenarios), Scenario 792a12u1e. Differs from the current text in the Then line: the backfill stamps fact and feedback notes only and leaves every other note unwritten ([792a10k Stamping the missing origin](../../specs/product/vault-migration.md#792a10k-stamping-the-missing-origin)), a skill note included.

```gherkin
Scenario: 792a12u1e The identity backfill preserves the identity fields
  Given a skill note missing repo, user and vault
  When the user runs engram update --backfill-identity
  Then the note is not rewritten, and its three skill fields are unchanged
```

## 861a4zzh Modified: Vault Serving scenarios, part 3

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Vault Serving](../../specs/product/vault-serving.md) in the commit of its test.

### 861a4zzh1 Modified: 792a20j1a Remote-set link and registration fields are ignored

Target: [792a20j1 UAT Scenarios](../../specs/product/vault-serving.md#792a20j1-uat-scenarios), Scenario 792a20j1a. Differs from the current text in the Then line: every pending note carries an xid, which the server mints, so the note cannot carry "none of those fields".

```gherkin
Scenario: 792a20j1a Remote-set link and registration fields are ignored
  Given a served learn body that includes parent, aliases, xid, skillHash or SkillKey keys in camelCase or PascalCase
  When the pending note is written
  Then it carries no parent, aliases or skill field and none of the values sent, and its xid is one the server minted
```

## 861a4zzl Modified: Vault Migration scenarios, part 3

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Vault Migration](../../specs/product/vault-migration.md) in the commit of its decision.

### 861a4zzl1 Modified: 792a10k1g Other note types are left alone

Target: [792a10k1 UAT Scenarios](../../specs/product/vault-migration.md#792a10k1-uat-scenarios), Scenario 792a10k1g. Differs from the current text in the Given line: a vocab definition note is a fact note ([vocab definition note](../../GLOSSARY.md#vocab-definition-note)), which [792a10k Stamping the missing origin](../../specs/product/vault-migration.md#792a10k-stamping-the-missing-origin) stamps like any other, so the note left alone is the [vocab term-note (retired)](../../GLOSSARY.md#vocab-term-note-retired).

```gherkin
Scenario: 792a10k1g Other note types are left alone
  Given a vocab term-note (retired) and a runbook note without user:
  When the user runs engram update --backfill-identity
  Then neither is modified
```

## 861a4zzn Modified: Chunk Indexing scenarios, part 2

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Chunk Indexing](../../specs/product/chunk-indexing.md) in the commit of its test.

### 861a4zzn1 Modified: 792a5i1b A changed source embeds only its new text

Target: [792a5i1 UAT Scenarios](../../specs/product/chunk-indexing.md#792a5i1-uat-scenarios), Scenario 792a5i1b. Differs from the current text in the Then line: the index writes the chunks it already held in no promised order and appends the new text after them, so the third chunk is the new text, not B's earlier text.

```gherkin
Scenario: 792a5i1b A changed source embeds only its new text
  Given a Markdown document with sections A and B that was ingested, after which section B was edited
  When the user runs engram ingest again with the document
  Then stdout reads "ingest <path>: 3 chunks (1 reused, 1 embedded)", and the index holds A's chunk, B's earlier text (792a5j) and B's new text
```

## 861a4zzv Modified: Note Revision scenarios, part 1

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Note Revision](../../specs/product/note-revision.md) in the commit of its test.

### 861a4zzv1 Modified: 792a4f1a An invalid amend argument is refused before any write

Target: [792a4f1 UAT Scenarios](../../specs/product/note-revision.md#792a4f1-uat-scenarios), Scenario Outline 792a4f1a. Differs from the current text in two Examples cells: the refusal of an unknown supersession type reads `type must be updates|narrows|refutes: got "bogus"` and the refusal of an unknown chunk source reads `unresolved chunk-source id: "nope#x"`, each with a colon where the text had a comma or none.

```gherkin
Scenario Outline: 792a4f1a An invalid amend argument is refused before any write
  Given a vault holding fact note 1
  When the user runs engram amend --target 1 with <argument>
  Then it fails with an error saying <problem>, and the note is unchanged

  Examples:
    | argument                     | problem                                        |
    | --supersedes "a\|bogus\|c"   | type must be updates\|narrows\|refutes: got "bogus" |
    | --supersedes "a\|updates"    | format must be "<basename>\|<type>\|<claim>"    |
    | --supersedes "a\|updates\|"  | claim must be non-empty                        |
    | --trigger ab                 | trigger must be at least 3 characters after trimming |
    | --chunk-source nope#x        | unresolved chunk-source id: "nope#x"           |
```
