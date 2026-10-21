# Documentation Framework Adoption - Technical Spec

This change documents unchanged behavior: it selects no design and changes no contract, so it has no recommended design diffs, no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./858d_questions-and-decisions.md#questions-and-decisions).

## 858b1 Existing-system orientation

The technical map: the 51 components and 5 technical workflows of `docs/specs/technical/`, each with the node that defines it and its responsibility in one line. The map is context, not a requirement.

| Spec | Kind | Defining node | Responsibility |
| --- | --- | --- | --- |
| [Agent Instruction Bundle](../../specs/technical/agent-instruction-bundle.md) | component | 792b32 | Constrains the instruction artifacts that engram ships to an agent harness: the metadata a harness reads to load a skill, the form of every `engram` command they show, and how the agent reaches engram and writes the vault. |
| [Automatic Memory Mod](../../specs/technical/automatic-memory-mod.md) | component | 793d3a | The `engram-memory` mod, a hooks module that routes the four automatic-memory workflows, places the delivered block beside a prompt, makes the model calls and bounds every wait; the `engram auto` commands hold the logic. |
| [Chunk Index Store](../../specs/technical/chunk-index-store.md) | component | 792b18 | Keeps the chunk index in its directory, one index file of embedded chunk records per indexed source and a manifest of every known source, and reconciles it with the gathered sources under the manifest lock. |
| [Command Line Surface](../../specs/technical/command-line-surface.md) | component | 792b2 | Turns one process invocation into one command run: parses arguments and environment into a typed request, resolves the vault and the chunk index directory, routes to the command and maps the result to the output streams and the exit status. |
| [Conversation Sweep](../../specs/technical/conversation-sweep.md) | component | 793d3f | The command `engram auto sweep`: turns a session's transcript into chunks of the chunk index, owning the choice of the transcript and of the turns it acts for. |
| [Conversation Sweep Technical Workflow](../../specs/technical/conversation-sweep-workflow.md) | technical workflow | [793d2c Conversation sweep flow](../../specs/technical/conversation-sweep-workflow.md#793d2c-conversation-sweep-flow) | After each turn, apart from the capture, the Automatic Memory Mod has the session's conversation indexed so that `engram query` finds it. |
| [Development Check Orchestration](../../specs/technical/development-check-orchestration.md) | component | 794b4 | Preserves the existing nil-check entry points under their shared target identity. |
| [Durable File Store](../../specs/technical/durable-file-store.md) | component | 792b3 | Performs every engram read and write of vault, sidecar, state and chunk-index files: replaces a file atomically, creates one exclusively, reports failures with the path and operation, and lists a vault directory's Markdown files. |
| [Embedding Sidecar](../../specs/technical/embedding-sidecar.md) | component | 792b10 | Owns each note's `.vec.json` sidecar: decides what text is embedded and hashed, builds, writes, decodes and validates it, classifies it against the current model, and runs the census and re-embed of a vault's sidecars. |
| [Eval Trial Harness](../../specs/technical/eval-trial-harness.md) | component | 792b38 | Gives an eval trial a configuration that carries the skills it claims and an environment off the maintainer's real engram state, guards the real vault against a leak, and decides from a trial's transcript what the agent did. |
| [Exchange Versioning](../../specs/technical/exchange-versioning.md) | component | 792b33 | Gives a note the identity that two vaults compare and the links that tie it to a note in another vault: a versioned content hash, a three-way comparison of two hashes, the optional exchange fields and the edits that record an offer receipt. |
| [Harness Deploy Sync](../../specs/technical/harness-deploy-sync.md) | component | 792b29 | For each detected harness, keeps an engram-owned root equal to the deploy set computed from the source, makes the harness's discovery paths links into that root, and deletes nothing it cannot prove it owns. |
| [Ingestion Source Discovery](../../specs/technical/ingestion-source-discovery.md) | component | 792b17 | Turns one `engram ingest` invocation into the list of source files to index: named files, Pi session directories and sweep roots walked under exclusion rules, each tagged with how it was found. |
| [Lesson Placement](../../specs/technical/lesson-placement.md) | component | 855d3d | The command `engram auto place`: turns the lessons into the request of the mid-tier judgement of their placement and supersession against the seen notes. |
| [Lesson Writing](../../specs/technical/lesson-writing.md) | component | 793d3e | The command `engram auto write`: turns the model's reply into notes of the vault, owning the reading of the reply and the mark of a note, and keeps the record of what became of each reply. |
| [Luhmann Addressing](../../specs/technical/luhmann-addressing.md) | component | 792b7 | The Luhmann ID scheme: parses an ID into its alternating number and letter segments, reads the ID that leads a note's filename, orders letter segments and allocates the next unused ID for a new note. |
| [Memory Loop Auditor](../../specs/technical/memory-loop-auditor.md) | component | 792b39 | Reads a real agent transcript in context and records, moment by moment, where the loop of recall, action and learning broke; it never writes a note. |
| [Mid-Turn Recall Technical Workflow](../../specs/technical/mid-turn-recall-workflow.md) | technical workflow | [855d2c Mid-turn recall flow](../../specs/technical/mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow) | Within a turn, at the tool events of the main loop, the Automatic Memory Mod runs the steps of [793d2a Prompt delivery flow](../../specs/technical/prompt-delivery-workflow.md#793d2a-prompt-delivery-flow) for the cue of the event and puts the block beside the tool's result. |
| [Moment Judgement](../../specs/technical/moment-judgement.md) | component | 855d3a | The command `engram auto moment`: turns a cue into the request of the cheap-tier judgement of whether it is a recall moment and at what depth, or a learn moment. |
| [Nil Check Run Technical Workflow](../../specs/technical/nil-check-run-flow.md) | technical workflow | [794b4b Nil check run](../../specs/technical/nil-check-run-flow.md#794b4b-nil-check-run) | A developer runs a nil check or fix through the existing entry points; orchestration takes the mode's arguments from the invocation policy, hands one request to the NilAway subprocess and returns the analyzer's success or failure. |
| [NilAway Invocation Policy](../../specs/technical/nilaway-invocation-policy.md) | component | 794b3 | Builds NilAway's arguments: the analysis scope for Engram's module and the check or fix mode. |
| [NilAway Subprocess](../../specs/technical/nilaway-subprocess.md) | component | [794b4c Run NilAway](../../specs/technical/nilaway-subprocess.md#794b4c-run-nilaway) | Starts the NilAway analyzer process for a configured request, inheriting the developer's environment. |
| [Note Amendment](../../specs/technical/note-amendment.md) | component | 792b14 | Revises a note that already exists: under the vault lock it applies field, supersession, chunk-source and identity changes, writes the note once and rebuilds its sidecar when what is embedded changed. |
| [Note Authoring](../../specs/technical/note-authoring.md) | component | 792b12 | Carries out `engram learn fact`, `feedback`, `runbook` and `qa`: validates the arguments, turns them into a note's fields and orders the write from the vault lock to the reported path. |
| [Note Document Format](../../specs/technical/note-document-format.md) | component | 792b6 | Defines the on-disk shape of a vault note: its filename, the frontmatter block with each kind's keys in a fixed order, the body formulas and link lines, and the reading and in-place editing of an existing note. |
| [Note Provenance](../../specs/technical/note-provenance.md) | component | 792b11 | Works out who and where a note write comes from, the repository of the working directory and the writing user, and stamps them with the vault name on a first write and re-stamps them on a revision. |
| [Note Reparenting](../../specs/technical/note-reparenting.md) | component | 792b27 | Moves existing notes to new Luhmann IDs and keeps the vault consistent with the move: proposes which top-level notes may belong under another, takes an agent's answers, renames the notes and rewrites every link to the old basename. |
| [Offer Outbox](../../specs/technical/offer-outbox.md) | component | 792b36 | Decides which local writes are offered to the parent vault, queues each offer in a durable per-vault outbox in the same critical section as the write, sends the queued offers in order and folds every outcome back into the outbox and the notes. |
| [Parent Vault Client](../../specs/technical/parent-vault-client.md) | component | 792b37 | Asks the configured parent vault for what a child reads from it and brings the answer into the child's vault: sends the requests, merges the parent's query answer into the local one, shows a parent note and pulls one down as a local pending offer. |
| [Pending Offer Bookkeeping](../../specs/technical/pending-offer-bookkeeping.md) | component | 792b34 | Decides whether a note is a pending offer, finds the pending offers of a vault and leaves them out of a list of notes, and carries the bookkeeping that settles an offer: the check that the judged version is still there and the fold that deletes the offer. |
| [Phrase Judgement](../../specs/technical/phrase-judgement.md) | component | 855d3b | The command `engram auto phrases`: turns the depth of the moment judgement into the request of the mid-tier judgement of the situation phrases. |
| [Prompt Delivery](../../specs/technical/prompt-delivery.md) | component | 793d3b | The command `engram auto deliver`: turns the candidate judgement of a recall moment into the block of notes and passages beside a prompt or within a turn, and into the lines that show the user what automatic memory did. |
| [Prompt Delivery Technical Workflow](../../specs/technical/prompt-delivery-workflow.md) | technical workflow | [793d2a Prompt delivery flow](../../specs/technical/prompt-delivery-workflow.md#793d2a-prompt-delivery-flow) | For each prompt of a session, the Automatic Memory Mod has the prompt judged as a recall moment or not, the lookup phrased, the candidates looked up and judged, and the notes and passages that apply put beside the prompt. |
| [Query Engine](../../specs/technical/query-engine.md) | component | 792b24 | Answers a query: reads the vault's notes, their sidecars and the chunk index, scores notes and chunks against each embedded phrase, merges, bounds and clusters the matches and hands the result to the renderer; it writes nothing. |
| [Query Payload Renderer](../../specs/technical/query-payload-renderer.md) | component | 792b23 | Turns what the query engine found into the payload the reader gets: puts trigger hits first and the newest chunks last, cuts to the limit, applies the chunk content policy, labels each item and encodes the YAML document; it decides nothing about what matched. |
| [Recall Lookup](../../specs/technical/recall-lookup.md) | component | 855d3c | The command `engram auto recall`: turns the phrases of the phrases judgement into the candidate notes and passages and the request of the mid-tier judgement of them. |
| [Recency Ranking](../../specs/technical/recency-ranking.md) | component | 792b21 | Turns time into ranking weight and orders chunks by recency: the factor for an age and a position in a session, a note's age, a chunk's turn number and the newest chunks; it does no input or output and reads no clock. |
| [Runbook Trigger Matching](../../specs/technical/runbook-trigger-matching.md) | component | 792b22 | Decides which runbooks a user's message triggers and puts them first: matches the literal triggers of the runbook notes against the message as whole words and places each hit ahead of every other result. |
| [Self Installer](../../specs/technical/self-installer.md) | component | 792b28 | Refreshes the engram binary from a source checkout or a fresh clone, refuses an install that would provably downgrade it, hands the rest of the update to the binary it just installed and writes the update report with its exit status. |
| [Session Context](../../specs/technical/session-context.md) | component | 793d3c | Tells what a session's agent already holds of the vault, read from the session's transcript: the notes and passages delivered beside prompts or within turns and what its own memory commands returned. |
| [Similarity Clustering](../../specs/technical/similarity-clustering.md) | component | 792b19 | Partitions a set of embedding vectors into clusters by cosine distance, scores a partition by its silhouette and chooses the number of clusters with the best score within a range; the same vectors, range and seed give the same result. |
| [Skill Registration Ledger](../../specs/technical/skill-registration-ledger.md) | component | 792b31 | Turns the keyed candidates of skill-source-discovery and the vault's skill notes into offers, remembers declined hashes, answers the offers and carries an accepted offer out, so that nothing is written without an answer. |
| [Skill Source Discovery](../../specs/technical/skill-source-discovery.md) | component | 792b30 | Finds every skill, command and prompt template that a detected harness loads, reads each one's bytes and resolved path and gives each a source-qualified key, so that registration never takes an unreadable source for an empty one. |
| [Source Chunking](../../specs/technical/source-chunking.md) | component | 792b16 | Cuts one source's text into embedding-sized chunks, each with an anchor that locates it in the source: a Markdown document by heading section, a transcript's conversation text by user turn; it does no input or output. |
| [Subprocess Runner](../../specs/technical/subprocess-runner.md) | component | 792b4 | Runs an external program, such as `git` or `go`, in a chosen directory and hands back what it printed or why it failed, telling a missing program apart from other failures. |
| [Text Embedder](../../specs/technical/text-embedder.md) | component | 792b9 | Turns text into a fixed-length vector with the bundled model, in process and offline, and reports the model ID and dimension of the vectors it makes. |
| [Transcript Reader](../../specs/technical/transcript-reader.md) | component | 792b15 | Turns one session transcript file into conversation text: keeps the user and assistant entries, drops harness noise, compresses each tool call and its result to one line and returns the timestamp of the last entry kept. |
| [Turn Capture Technical Workflow](../../specs/technical/turn-capture-workflow.md) | technical workflow | [793d2b Turn capture flow](../../specs/technical/turn-capture-workflow.md#793d2b-turn-capture-flow) | After each turn, the Automatic Memory Mod has the turn judged as a learn moment or not, its lessons judged and placed, and the lessons written as notes. |
| [Turn Judgement](../../specs/technical/turn-judgement.md) | component | 793d3d | The command `engram auto judge`: turns a learn moment into the request of the mid-tier judgement of the lessons of four kinds, redacting credentials first. |
| [Vault Counter](../../specs/technical/vault-counter.md) | component | 792b25 | Turns the notes at a vault's root into exact counts: how many carry each value of a frontmatter attribute, and how many notes link to a named note and which; it reads notes and never writes. |
| [Vault HTTP Server](../../specs/technical/vault-http-server.md) | component | 792b35 | Answers a child's HTTP requests for `query`, `show`, `activate` and `learn` against one fixed vault under the commands' own locks, and writes a served `learn` as a pending offer. |
| [Vault Identity Registry](../../specs/technical/vault-identity-registry.md) | component | 792b5 | Brings a vault into existence on first use and owns its identity: the tracked vault ID, the untracked `.engram/` state directory and the location record that tells a copied, moved or cloned vault from the original. |
| [Vault Invariant Checker](../../specs/technical/vault-invariant-checker.md) | component | 792b26 | Reads the notes and sidecars at a vault's root and reports which of the vault's invariants they violate, as correctness failures or corpus-health warnings; it writes nothing. |
| [Vault Write Lock](../../specs/technical/vault-write-lock.md) | component | 792b8 | Gives one process at a time exclusive hold of a lock file, `.luhmann.lock` at the vault root, creating it when absent and waiting while another holder has it. |
| [Vocabulary Engine](../../specs/technical/vocabulary-engine.md) | component | 792b20 | Keeps the vault's controlled vocabulary: recognizes definition notes, tags a note with its closest terms, reports statistics and carries out bootstrap, propose, refit, the self-tag backfill and the old-format migration. |
| [Wikilink Graph](../../specs/technical/wikilink-graph.md) | component | 792b13 | Sees the vault as a set of linked notes: scans the notes at the vault root, resolves a user's reference to a note, builds the directed link graph, reads a note's in-degree and lists the links that resolve to no note. |

## 858b2 Added: scenario and reading rule of existing behavior

### 858b2a Added: A lesson the vault already states is not placed

Target: [855d4h1 Scenarios](../../specs/technical/lesson-placement.md#855d4h1-scenarios), after [855d4h1h A lesson supersedes a note only when it is about the note's own claim](../../specs/technical/lesson-placement.md#855d4h1-scenarios). Evidence: the test `Test858b2a1a_Place_ALessonTheVaultAlreadyStatesIsNotPlaced` (renamed from `TestPlace_ALessonTheVaultAlreadyStatesIsNotPlaced` in `internal/cli/auto_place_test.go`).

#### 858b2a1 Scenarios

```gherkin
Scenario: 858b2a1a A lesson the vault already states is not placed
  Given a vault holding a note that states a lesson, a session that was delivered that note, and a reply of two whole lessons of which one, or both, give that note's basename as the note that covers them
  When the placement is given the reply
  Then the lesson the note covers is not asked about, and when every lesson is covered it prints nothing and succeeds
```

### 858b2b Added: The scenarios of 793d4g1 are read with the reply that the turn is a learn moment

Target: directly after the gherkin block of [793d4g1 Scenarios](../../specs/technical/turn-judgement.md#793d4g1-scenarios), before [855d4g1 Scenarios](../../specs/technical/turn-judgement.md#855d4g1-scenarios). The pre-promotion change text said to read the scenarios of the target with the reply `{"learn":true}` beside the fields they name, except [793d4g1a The request holds the instructions, the context and the turn](../../specs/technical/turn-judgement.md#793d4g1-scenarios); promotion dropped the sentence and the scenarios no longer say so. The node below, without the words that were relative to the change, is added; the scenarios are not changed.

```markdown
#### 858b2b1 Reading rule

The scenarios of 793d4g1 other than 793d4g1a are read with the reply `{"learn":true}` beside the fields they name.
```

## 858b3 Modified: scenarios that differed from the shipped behavior

Each operation replaces one canonical target completely. The shipped behavior is the later deliberate decision (the tests fake `auto moment`), so the documentation is corrected.

### 858b3a Modified: The work after a turn is started from a timer, with no call in flight when the event settles

Target: [793d2b1l The work after a turn is started from a timer, with no call in flight when the event settles](../../specs/technical/turn-capture-workflow.md#793d2b1-scenarios). Differs from the former text in `auto moment` for `auto judge` (the test `Test793d2b1l_Mod_...` in `dev/mods_integration_internal_test.go` holds `auto moment` and `auto sweep`).

```gherkin
Scenario: 793d2b1l The work after a turn is started from a timer, with no call in flight when the event settles
  Given the mod, a host that counts the calls it has in flight when the hook of an event settles and runs the mod's timers only when the driver lets them, and an engram whose auto moment and auto sweep do not answer
  When a turn completes
  Then no call of the host was made or in flight when the event's hook settled, and auto moment and auto sweep were run when the timers were let go
```

### 858b3b Modified: A long failure is reported whole

Target: [793d2b1m A long failure is reported whole](../../specs/technical/turn-capture-workflow.md#793d2b1-scenarios). Differs in `auto moment` for `auto judge` (the failure the test reads begins `engram auto moment exited with 1`).

```gherkin
Scenario: 793d2b1m A long failure is reported whole
  Given the mod, and an engram whose auto moment exits with 1 and prints on standard error a line of 1000 characters that begins by naming the step and ends with a cause
  When a turn completes
  Then auto write received a failure that holds the whole line, from the name of the judgement and the status to the cause
```

### 858b3c Modified: The mod sweeps the conversation after a turn, apart from the capture

Target: [793d2c1a The mod sweeps the conversation after a turn, apart from the capture](../../specs/technical/conversation-sweep-workflow.md#793d2c1-scenarios). Differs in `auto moment` for `auto judge` (the test holds the moment judgement).

```gherkin
Scenario: 793d2c1a The mod sweeps the conversation after a turn, apart from the capture
  Given the mod, a session whose ID is known, an engram whose auto moment does not answer, and an engram whose auto sweep exits with 0
  When a prompt is raised, the turn starts, and it completes
  Then the event's result is given back at once, and auto sweep received the session's ID and the origin of the prompt while the judgement was still running
```
