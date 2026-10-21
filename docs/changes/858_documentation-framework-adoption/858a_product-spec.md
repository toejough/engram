# Documentation Framework Adoption - Product Spec

Public, quotable: `PR1#<id>` = owner review comment on github.com/toejough/toejough.github.io/pull/1 (2026-03-28). `blog <file>:<line>` = published post: `engram.html` = https://toejough.github.io/engram, `engram-update.html` = https://toejough.github.io/engram-update, `engram-2026-05.html` = https://toejough.github.io/engram-2026-05. `May draft L<n>` = line n of `Engram Update 3.md` in toejough.github.io at commit `00fdae1` (unpublished; the May post is its copy-edit). `#<n>` = issue or PR n of github.com/toejough/engram. `<sha>:<path>:<line>` = git object; `5efe13c7` = this branch three commits before the migration base `6a88ca1b` (the cited files are unchanged between them). `ROADMAP:n`, `AGENTS:n` = line n of `docs/ROADMAP.md`, root `AGENTS.md` at `5efe13c7`. `LEDGER#<slug>` = anchor in `dev/eval/LEDGER.md`; `LEDGER:n` = its line n. `.review:<n>` = line n of `.review/events.jsonl`. `793b<n>`, `793c<n>`, `855b<n>`, `855c<n>` = entries of `docs/changes/793_automatic-memory/` and `docs/changes/855_judged-automatic-memory/` at `34e663b2^` (deleted at promotion); the file and line are given at first use. `old 793b<n>` = the abandoned first attempt, `dcf53d07:docs/changes/793_parallel-memory-agent/793b_decision-log.md` (branch docs/parallel-memory-agent-requirements).

Private, paraphrased only, never an excerpt longer than a clause: `s:<session8>@<MM-DDTHH:MMZ>` = Claude Code session transcript of this repo, 2026, UTC. `c:<source>#<anchor> (<date>)` = row of the engram chunk index; dates are estimates (+-1-3 days; `~` approximate, `>=`/`<=` a bound). `note <N>` = vault note N. `arch:<slug>` = March 2026 archive memory of the vault.

Option letters follow the synthesis (`goals-synthesis.md`, given to the owner separately) for 858d9 to 858d28, whose last option is "none of these, or your own answer"; the outcome, non-goal and constraint rows offer ratify, change or reject; rows from 858d48 on carry the author's recommendation. "Rec" is a recommendation, never a decision. "Author note" marks text added here, not taken from the synthesis.

## 858a1 Problem statement

The owner and every agent that works on this repository read documentation rules from six local guides, while the owner's current guidance is the three guides published at github.com/toejough/software-developer-guides (branch `codex/draft-guidance`). The 80 canonical specifications, the glossary and the root instructions follow the local scheme: most files open with a requirement, not with a heading for the capability or component they specify; no product file carries the capability recipe's allocation block (participant responsibilities, interactions, boundary assumptions); 38 technical files qualify their component under the old five criteria and 18 state no qualification; one workflow carries the older `-flow` name; the glossary mixes business and technical terms; and the owner's intended outcomes are recorded nowhere in the specifications. The owner needs the repository to follow the adopted guides with the shipped behavior, its requirements and its IDs unchanged, and every difference between documentation and behavior found, recorded and ruled on.

## 858a2 Added: Proposed outcomes

Each node is proposed and pending the owner's ratification ([858d56 M9 Outcomes are proposals](./858d_questions-and-decisions.md#questions-and-decisions)). Its status is the first line of the node, not part of its heading, so that its anchor does not change at ratification. OUT-1 to OUT-7 and TH-1 name [858a2a say-once](#858a2a-added-say-once) to [858a2h first-pass-success](#858a2h-added-first-pass-success) in order.

### 858a2a Added: say-once

Proposed; pending owner ratification ([858d1 OUT-1 say-once: a lesson given once is applied from then on](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions); working default the owner, working with LLM agents across his own projects, machines and harnesses (low confidence).
- Intended change: a correction, convention, decision or confirmed approach that the user gives once is applied in later work. The user does not repeat it; the agent neither re-derives it nor re-explores a path already known to be bad.
- Scope: lessons the user gives or confirms and the vault keeps. Whether memory makes whole tasks faster or cheaper is TH-1 ([858a2h first-pass-success](#858a2h-added-first-pass-success)), not this outcome.
- External assumptions: the agent reads and follows what memory surfaces (baseline: 26 of 50 surfaced moments followed, 26 of 27 where a verdict could be rendered). The proven wins are on content a model cannot derive: local conventions, a newer claim superseding an older one, a recent standard, a conclusion drawn from two notes.

Success criteria (all proposed; no numeric target set by the owner; the three rest on measurements of the software, so confidence as business criteria is low):

1. 858a2a1 Content a model cannot derive is applied. Target: none agreed; option: the existing trap-gate bars (every valid trial passes for C3 conventions, C4i supersession, C5 standard, C6 combined notes), which held from 0 to 400 notes.
2. 858a2a2 In real sessions surfaced memory is followed. Target: none agreed; baseline 52% (26/50), 96% (26/27) where a verdict could be rendered.
3. 858a2a3 Procedures (runbooks) are applied as well as facts. Target: none agreed; unmeasured today (#737).

Evidence: ROADMAP:339; c:ee8329d2#turn-2 (~05-27), c:d498a354#turn-40 (~06-05), c:d149c84e#turn-92 (~07-14); PR1#3004743876; #23 (agent-filed, relays 2026-03-05); arch:memory-must-drive-behavior-change 2026-03-20; LEDGER#tactical-convention-warm-flip, LEDGER#crowded-vault-capability-robustness, LEDGER#733-c5-threshold-decision, LEDGER#739-audit-f5-followed; note 99; c:a19e7b75#turn-87 (one-line form, assistant relay only).

Confidence: high (founding mission, stated in March, May and June).

Recommendation (synthesis): ratify as written ([858d1 OUT-1 say-once: a lesson given once is applied from then on](./858d_questions-and-decisions.md#questions-and-decisions)); its relation to OUT-2 is [858d10 Q2 How do say-once (OUT-1) and automatic memory (OUT-2) relate?](./858d_questions-and-decisions.md#questions-and-decisions).

Contributing capabilities: [Recall](../../specs/product/recall.md), [Memory Query](../../specs/product/memory-query.md), [Controlled Vocabulary](../../specs/product/controlled-vocabulary.md), [Learn](../../specs/product/learn.md), [Note Writing](../../specs/product/note-writing.md), [Write Memory](../../specs/product/write-memory.md), [Lessons Contract](../../specs/product/lessons-contract.md), [Ambient Guidance](../../specs/product/ambient-guidance.md), [Curate](../../specs/product/curate.md), [Note Revision](../../specs/product/note-revision.md), [Vault Migration](../../specs/product/vault-migration.md), [Skill Runbook Registration](../../specs/product/skill-runbook-registration.md), [Route](../../specs/product/route.md), [Please](../../specs/product/please.md).

### 858a2b Added: memory-without-remembering

Proposed; pending owner ratification ([858d2 OUT-2 memory-without-remembering: memory works without anyone remembering to use it](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions); working default the owner across his projects, machines and harnesses (low).
- Intended change: relevant memory reaches the agent at the moments it matters and lessons worth keeping are saved, without the user or the working agent having to remember to do it, and without filling the agent's context with what does not bear on the moment.
- Scope: Claude Code (automatic memory runs as a mod); intentional recall and learn stay beside it (dual system). Earlier answers on subagent capture, prompt sources, mid-task moments and the prompt hold are reversed or overtaken by the restart, pending [858d25 Q16 The 2026-10-16 restart reverses 14 earlier answers by the author's default (793c13, still open): accept all, or name rows to keep?](./858d_questions-and-decisions.md#questions-and-decisions).
- External assumptions: the host exposes its mod and model-call facilities; the judging model is reached through the host (CON-4, [858d43 CON-4: pure Go, no CGO, local embeddings; the binary makes no model or agent calls of its own; network only for parent sync and serve](./858d_questions-and-decisions.md#questions-and-decisions)); the vault holds the notes.

Success criteria (proposed; none ratified; no ratified metric exists, targets moved to #850):

1. 858a2b1 The first attempt's success test: "a user who never thinks about memory still gets the right notes at the right moments, and nothing worth keeping is lost." Assistant-written, approved by the owner on 2026-10-12, not carried into the restart; asked as [858d14 Q6 What proves OUT-2 succeeded?](./858d_questions-and-decisions.md#questions-and-decisions) option (d).
2. 858a2b2 Capture beats the baseline and precision is recorded. Target: none agreed; baseline 4/71 = 5.6% of worth-learning moments (3/66 after the prose fix); the owner asked to exceed it and record precision (s:7bc2f2bd@10-11T15:55Z) and compared 29% with the 5% the main agent managed (s:7bc2f2bd@10-15T02:46Z).
3. 858a2b3 Recall reaches the moments where a relevant memory exists. Target: none agreed; baseline 52/150 = 34.7%.
4. 858a2b4 Nothing irrelevant, repeated or ephemeral arrives or is written: what arrives is what an intentional recall would surface, what is written is what an intentional learn would write ([855a2a Delivery writes what its depth writes](../../specs/product/automatic-memory.md#855a2a-delivery-writes-what-its-depth-writes) through [855a2d Lessons that correct a note say so](../../specs/product/automatic-memory.md#855a2d-lessons-that-correct-a-note-say-so) in [Automatic Memory](../../specs/product/automatic-memory.md); s:7bc2f2bd@10-18T17:16Z).
5. 858a2b5 The owner sees it working in his own sessions before accepting it (s:7bc2f2bd@10-15T12:40Z; s:7bc2f2bd@10-20T22:10Z).

Options for 1-3: see [858d14 Q6 What proves OUT-2 succeeded?](./858d_questions-and-decisions.md#questions-and-decisions) and #850 (criteria removed from 793a: capture rate on at least 60 labeled moments, independently graded precision and activation accuracy, controlled comparison at N = 20 per arm, no added delay).

Evidence: s:7bc2f2bd@10-17T03:44Z, s:f03ac62f@10-03T13:27Z, c:9e790e0a#turn-8 (~07-02), s:7bc2f2bd@10-11T23:38Z, s:7bc2f2bd@10-11T19:08Z, s:7bc2f2bd@10-18T17:09Z, s:7bc2f2bd@10-11T15:55Z, s:7bc2f2bd@10-15T02:46Z, s:7bc2f2bd@10-15T12:40Z, s:7bc2f2bd@10-20T22:10Z; PR1#3004743876; May draft L35; blog engram-2026-05.html:416, :428; old 793b240 (dcf53d07:docs/changes/793_parallel-memory-agent/793b_decision-log.md:3263-3266); LEDGER#739-audit-w2-learn-fired, LEDGER#learn-rate-remeasure-w2, LEDGER#739-audit-f2-search-ran; 793b36 (34e663b2^:docs/changes/793_automatic-memory/793b_decision-log.md:145-165).

Confidence: high for the aim (owner-typed in March, May, July, September, October); medium as engram's overall goal (the "headline" wording is scoped to change 793; [858d26 Q17 How do the three "north stars" layer?](./858d_questions-and-decisions.md#questions-and-decisions)).

Recommendation (synthesis): ratify as written, criteria from [858d14 Q6 What proves OUT-2 succeeded?](./858d_questions-and-decisions.md#questions-and-decisions).

Contributing capabilities: [Automatic Memory](../../specs/product/automatic-memory.md), [Recall](../../specs/product/recall.md), [Memory Query](../../specs/product/memory-query.md), [Learn](../../specs/product/learn.md), [Lessons Contract](../../specs/product/lessons-contract.md), [Ambient Guidance](../../specs/product/ambient-guidance.md), [Chunk Indexing](../../specs/product/chunk-indexing.md), [Install and Update](../../specs/product/install-and-update.md), [Please](../../specs/product/please.md).

### 858a2c Added: pick-up-where-we-left-off

Proposed; pending owner ratification ([858d3 OUT-3 pick-up-where-we-left-off: work resumes without re-briefing](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions); working default the owner (low).
- Intended change: after a /clear, a compaction, a new session or a month away, the agent knows what was being done and what comes next, from memory; status notes do not clutter the vault.
- Scope: episodic memory of sessions and files; excludes status notes in the vault for now (CON-5, [858d44 CON-5: notes are durable and atomic, in three kinds; a request to save status writes no note, for now](./858d_questions-and-decisions.md#questions-and-decisions); follow-up #857).
- External assumptions: the agent takes recalled chunks seriously and reads them (#857 records that chunk-only answers were judged without being read).

Success criteria:

1. 858a2c1 None agreed. Option: the bar proposed in #857, "The agent answers 'what's next for X?' from memory, correctly and from the current state, without status notes polluting the vault", after measuring how such questions are answered today. Measured so far: recency value comes from re-ranking (C4i 5/5); the recent-fill channel adds nothing for related items (note 297; LEDGER#646-recency-recent-fill-selfcapture is unmeasured).

Evidence: arch:preserve-context-across-session-reset 2026-03-07; May draft L33, blog engram-2026-05.html:426; c:c1195a11#preamble (~06-16); d425a153^:questions.md:2-3; s:7bc2f2bd@10-18T18:26Z; s:9af40acb@10-19T19:12Z; 855c11 (34e663b2^:docs/changes/855_judged-automatic-memory/855c_open-questions.md:88); #857.

Confidence: medium (stated in March, May, June, October; not called a goal of engram as a whole; may be only a criterion of OUT-2, [858d21 Q12 Is pick-up-where-we-left-off (OUT-3) its own outcome?](./858d_questions-and-decisions.md#questions-and-decisions)).

Recommendation (synthesis): ratify as its own outcome ([858d21 Q12 Is pick-up-where-we-left-off (OUT-3) its own outcome?](./858d_questions-and-decisions.md#questions-and-decisions) option (a)).

Contributing capabilities: [Recall](../../specs/product/recall.md), [Chunk Indexing](../../specs/product/chunk-indexing.md), [Automatic Memory](../../specs/product/automatic-memory.md).

### 858a2d Added: memory-follows-the-user

Proposed; pending owner ratification ([858d4 OUT-4 memory-follows-the-user: the same memory wherever the user works](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions) (the owner's own environments, a team or org, or any user).
- Intended change: the same memory is available, current and complete in each harness, machine and environment after one setup step, and keeps working when a parent vault is unreachable.
- Scope: Claude Code and Pi (CON-10, [858d23 Q14 Harness scope (also decides CON-10)](./858d_questions-and-decisions.md#questions-and-decisions)); Pi gets no automatic memory. The synthesis proposes splitting into OUT-4a "one setup step" (core) and OUT-4b "other harnesses keep working" (the owner called those items preferences, not essential); IDs for a split are minted after [858d4 OUT-4 memory-follows-the-user: the same memory wherever the user works](./858d_questions-and-decisions.md#questions-and-decisions) is answered.
- External assumptions: a parent vault, when configured, may be unreachable; the user gives the initial setup instruction once.

Success criteria (all software promises, not business criteria; low confidence; no measured baseline):

1. 858a2d1 One `engram update` turns everything on, with no other step ([793a2a On after `engram update`](../../specs/product/automatic-memory.md#793a2a-on-after-engram-update); its "off by uninstalling" part is pending [858d25 Q16 The 2026-10-16 restart reverses 14 earlier answers by the author's default (793c13, still open): accept all, or name rows to keep?](./858d_questions-and-decisions.md#questions-and-decisions)).
2. 858a2d2 A fresh install has the full skill set (note 1054).
3. 858a2d3 Removals propagate ([792a11i Removal propagates](../../specs/product/install-and-update.md#792a11i-removal-propagates)).
4. 858a2d4 With the parent down, the local answer is still given ([792a21k An unreachable parent leaves the local answer](../../specs/product/parent-sync.md#792a21k-an-unreachable-parent-leaves-the-local-answer)).

Target for each: none agreed.

Evidence: c:b1ffad9e#turn-18 (<=08-10), c:3613955e#turn-90 (~09-21), c:f6dfe139#turn-21 (~09-27), s:7bc2f2bd@10-11T20:18Z, s:7bc2f2bd@10-12T15:14Z, s:7bc2f2bd@10-12T19:30Z, s:7bc2f2bd@10-11T18:06Z; cd230c21:openspec/changes/update-deploy-as-sync/proposal.md:5 (#706); #726 (agent-filed, relays 2026-08-22); 793b36 rows 8 and 10.

Confidence: medium for "one setup step"; low for "other harnesses keep working".

Recommendation (synthesis): split into the two parts if the owner agrees; ratify the core part.

Contributing capabilities: [Parent Sync](../../specs/product/parent-sync.md), [Vault Serving](../../specs/product/vault-serving.md), [Vault Identity](../../specs/product/vault-identity.md), [Install and Update](../../specs/product/install-and-update.md), [Skill Runbook Registration](../../specs/product/skill-runbook-registration.md), [Chunk Indexing](../../specs/product/chunk-indexing.md), [Curate](../../specs/product/curate.md).

### 858a2e Added: see-and-trust-memory

Proposed; pending owner ratification ([858d5 OUT-5 see-and-trust-memory: the user can see what memory did and tell whether it helps](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions) (the user of memory).
- Intended change: in the session the user can see what memory delivered, wrote, left out or failed to do, and why; from real use and honest measurement the user can tell whether memory is helping.
- Scope: user-visible activity and the audit of real transcripts; whether benchmark measurement is product or maintainer tooling is [858d15 Q7 Is measuring memory's value part of the product?](./858d_questions-and-decisions.md#questions-and-decisions). A memory system whose purpose is to grade itself is excluded (NG-19, [858d40 NG-19: a memory system whose purpose is to grade itself or answer questions about itself](./858d_questions-and-decisions.md#questions-and-decisions)).
- External assumptions: the user reads the session; the audit reads real transcripts.

Success criteria (software promises; low confidence; deliberately no thresholds, dev/eval/audit/re-measure-plan.md:40-41):

1. 858a2e1 What was delivered, written, left out and not done is shown in the session in at most four lines ([793a2e The user can see what it did](../../specs/product/automatic-memory.md#793a2e-the-user-can-see-what-it-did)). Target: none agreed.
2. 858a2e2 Every warning names the operation and the reason (note 1166). Target: none agreed.
3. 858a2e3 Value claims carry one of four ledger verdicts: proven, refuted, unmeasured, superseded (LEDGER:9-16). Target: none agreed. The audit's funnel rates are the only real-use baseline.

Evidence: s:7bc2f2bd@10-11T20:07Z, s:7bc2f2bd@10-16T03:01Z, c:a19e7b75#turn-9 (~06-20); .review:32 (thread 5974d361, 2026-08-30); May draft L36, blog engram-2026-05.html:429; PR1#3004763779 (conflicting, 2026-03-28); 793b6 (34e663b2^:docs/changes/793_automatic-memory/793b_decision-log.md:25-27).

Confidence: medium.

Recommendation (synthesis): ratify as written; ruling on measurement with [858d15 Q7 Is measuring memory's value part of the product?](./858d_questions-and-decisions.md#questions-and-decisions).

Contributing capabilities: [Automatic Memory](../../specs/product/automatic-memory.md), [Controlled Vocabulary](../../specs/product/controlled-vocabulary.md), [Vault Health](../../specs/product/vault-health.md), [Vault Counting](../../specs/product/vault-counting.md), [Memory Effectiveness Evaluation](../../specs/product/memory-effectiveness-evaluation.md) (classification pending [858d15 Q7 Is measuring memory's value part of the product?](./858d_questions-and-decisions.md#questions-and-decisions)).

### 858a2f Added: memory-worth-its-cost

Proposed; pending owner ratification ([858d6 OUT-6 memory-worth-its-cost: engram's own overhead is low enough to fire freely (contested, deferred)](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions).
- Intended change: the overhead of firing memory, in time, tokens and money, is low enough that memory can fire often without slowing the work, and a cheaper model does the memory judgements.
- Scope: engram's own overhead and tier substitution; not whether memory makes whole tasks faster or cheaper (TH-1, [858a2h first-pass-success](#858a2h-added-first-pass-success)). Contested and deferred: the owner's latest words defer time ("land all the right behavior before optimizing", [858d47 CON-12: land the right behavior before optimizing latency (the 8-second hold)](./858d_questions-and-decisions.md#questions-and-decisions)).
- External assumptions: a cheaper model is adequate for the judgements (the owner called this a hypothesis to test).

Success criteria: none agreed; options:

1. 858a2f1 A cheaper model with memory matches a stronger one with memory (sonnet = opus on C3 15/15, C4i 3/3, C6 6/6; sonnet cold fails; LEDGER#tier-routing-parity).
2. 858a2f2 The cheap depth costs less per fire (glance 2.23x faster and 46% cheaper than deep; LEDGER#glance-cost-realvault).
3. 858a2f3 Over-firing tolerance: 10% more calls than necessary is acceptable, 10x is not (c:95570838#turn-20 (~06-27)); later the owner asked whether over-firing is a problem at all (c:95570838#turn-63 (~06-29)).
4. 858a2f4 Only real token, money and time cuts count; moving work off the perceived path does not (NG-2, [858d29 NG-2 and NG-15: cost-claim hygiene and over-firing](./858d_questions-and-decisions.md#questions-and-decisions)). No latency target exists (recall median 51.9 s, n=3, directional; LEDGER#recall-time-remeasure).

Evidence: c:95570838#turn-13 (~06-26), c:d149c84e#turn-40 (~07-12), s:7bc2f2bd@10-12T12:15Z, s:7bc2f2bd@10-16T16:20Z, s:7bc2f2bd@10-18T18:01Z, s:7bc2f2bd@10-12T19:30Z, s:7bc2f2bd@10-14T12:19Z, c:95570838#turn-19 (~06-27); #684, #690, #693 (ROADMAP:385-387); note 135.

Confidence: low; must be ratified.

Recommendation (synthesis): constraint only for now, no outcome file, with TH-1 kept as the thesis ([858d11 Q3 Is cost or speed an outcome, and is "faster, cheaper first-pass success" still the thesis?](./858d_questions-and-decisions.md#questions-and-decisions) option (d)).

Contributing capabilities: [Route](../../specs/product/route.md), [Vault Counting](../../specs/product/vault-counting.md); by the synthesis's capability list (not its file column) [Recall](../../specs/product/recall.md) and [Automatic Memory](../../specs/product/automatic-memory.md) (cheap judge).

### 858a2g Added: find-the-right-memory

Proposed; pending owner ratification ([858d7 OUT-7 find-the-right-memory: the most relevant memories are found quickly](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions).
- Intended change: for a given task or moment, the memories that bear on it are found, and found fast.
- Scope: retrieval of notes and chunks; the secondary "strengthen connections" half is [858d19 Q10 Are links and a compounding knowledge graph still wanted?](./858d_questions-and-decisions.md#questions-and-decisions).
- External assumptions: the embedder's ceiling bounds note recall (rare exact tokens are weak).

Success criteria: none agreed; evidence is software, not business criteria:

1. 858a2g1 Option: the right note surfaces for a query at the practical ceiling of the embedder, holding under the standing regression gate (ROADMAP:345-348); real-path note recall@5 rose from 0.22 to 0.83 with the matched-note floor (LEDGER#matched-note-floor). Target: none agreed.

Evidence: c:ee8329d2#turn-1 (~05-27); d425a153^:docs/design/2026-06-28-retrieval-probe-results.md:3; blog engram-update.html:363, :384, :427; ROADMAP:345-348.

Confidence: high that it is a goal; separate from OUT-1 or not is [858d28 Q19 Is finding the right memory (OUT-7) its own outcome?](./858d_questions-and-decisions.md#questions-and-decisions).

Recommendation (synthesis): ratify as its own outcome ([858d28 Q19 Is finding the right memory (OUT-7) its own outcome?](./858d_questions-and-decisions.md#questions-and-decisions) option (a)).

Contributing capabilities: [Recall](../../specs/product/recall.md), [Memory Query](../../specs/product/memory-query.md); by the synthesis's capability list (not its file column) [Chunk Indexing](../../specs/product/chunk-indexing.md).

### 858a2h Added: first-pass-success

Proposed; pending owner ratification ([858d8 TH-1 first-pass-success: memory makes first-pass success faster and cheaper (your stated thesis; refuted on easy builds; not withdrawn)](./858d_questions-and-decisions.md#questions-and-decisions)).

Desired result:

- Beneficiary: pending [858d9 Q1 Who is the beneficiary of these outcomes?](./858d_questions-and-decisions.md#questions-and-decisions).
- Intended change: memory makes first-pass success faster and cheaper on a variety of tasks, with fewer mistakes, reviews, tests and rebuilds: the owner's stated thesis.
- Scope: whole-task outcomes with versus without memory; not engram's own overhead ([858a2f memory-worth-its-cost](#858a2f-added-memory-worth-its-cost)).
- External assumptions: self-seeding build evals cannot measure the harder regime (note 286); real long-session work, read with the audit, is the proposed instrument (#850).

Success criteria: none agreed; options:

1. 858a2h1 The owner's May criterion: an honest benchmark eval showing faster, cheaper first-pass success on a variety of tasks (May draft L36; blog engram-2026-05.html:429). No target; the owner had no such system.

Status of the measurements: refuted on easy builds (warm +182 s, +$3.08, n=8/arm; LEDGER#c1-c2-warm-op-negatives); unmeasured on the harder regime (LEDGER#harder-regime-op-cost-unmeasurable, LEDGER:40).

Evidence: c:ee8329d2#turn-20 (~05-30), c:a19e7b75#turn-9 (~06-20), c:ec819aed#turn-8 (~06-25); arch:memory-must-drive-behavior-change 2026-03-20; May draft L36.

Confidence: low; the owner has not withdrawn the thesis.

Recommendation (synthesis): keep as the thesis, measure on real long-session work with the audit ([858d11 Q3 Is cost or speed an outcome, and is "faster, cheaper first-pass success" still the thesis?](./858d_questions-and-decisions.md#questions-and-decisions) option (d)).

Contributing capabilities: none in the synthesis's file column; by its capability list, [Route](../../specs/product/route.md).

## 858a3 Capabilities

The 24 files of `docs/specs/product/`. Class key: CAP capability, WF business workflow, MECH technical mechanism or feature-shaped, CTX contextual entry. The proposed homes rest on titles and heading ranges, not requirement bodies (synthesis section 5): every row is proposed. This change restructures each file in place and reclassifies none ([858d53 M6 Capability recipe in place](./858d_questions-and-decisions.md#questions-and-decisions), [858d57 M10 Business workflows](./858d_questions-and-decisions.md#questions-and-decisions)).

| File | Defining node | Class now | Ability in one line | Proposed home | Outcomes |
| --- | --- | --- | --- | --- | --- |
| [Ambient Guidance](../../specs/product/ambient-guidance.md) | 792a19 | MECH (four firing policies in one file) | Standing instruction text that a user imports into the config file of an agent harness, so that every session carries it without anyone invoking a skill. | split by owner: [792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments) through [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep) to `recall`; [792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment) through [792a19f A returned line that fails the bar is not captured](../../specs/product/ambient-guidance.md#792a19f-a-returned-line-that-fails-the-bar-is-not-captured) to `learn` / `delegated-work-workflow`; [792a19g Every unit of work is planned and routed first](../../specs/product/ambient-guidance.md#792a19g-every-unit-of-work-is-planned-and-routed-first) through [792a19i Returned work is reviewed fresh and reported with its evidence](../../specs/product/ambient-guidance.md#792a19i-returned-work-is-reviewed-fresh-and-reported-with-its-evidence) to `route`; [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query) through [792a19t The query runs again at four moments mid-task](../../specs/product/ambient-guidance.md#792a19t-the-query-runs-again-at-four-moments-mid-task) (query first, runbook follow frame, re-entry) to `recall`; [792a19u The shim joins the other guidance additively](../../specs/product/ambient-guidance.md#792a19u-the-shim-joins-the-other-guidance-additively) through [792a19v The update report says how to activate guidance](../../specs/product/ambient-guidance.md#792a19v-the-update-report-says-how-to-activate-guidance) to `set-up-engram` | OUT-1, 2 |
| [Automatic Memory](../../specs/product/automatic-memory.md) | [793a2 Automatic memory](../../specs/product/automatic-memory.md#793a2-automatic-memory) | feature-shaped | Puts the notes and passages that apply to a recall moment in the agent's context, writes the lessons of a turn as notes and makes the conversation searchable, without the user or the agent asking. | keep as `automatic-memory-workflow` (composition: moment judged, delivered, agent acts, lesson judged, written, shown; handoffs host/judge/worker/agent/user; CON-1; visibility). Content rules go to owners: delivery rules ([793a2b Memory arrives with the prompt](../../specs/product/automatic-memory.md#793a2b-memory-arrives-with-the-prompt), [855a2a Delivery writes what its depth writes](../../specs/product/automatic-memory.md#855a2a-delivery-writes-what-its-depth-writes), [855a2e Recall moments within a turn](../../specs/product/automatic-memory.md#855a2e-recall-moments-within-a-turn), modified scenarios [855a3b1 UAT Scenarios](../../specs/product/automatic-memory.md#855a3b1-uat-scenarios), [855a3c1 UAT Scenarios](../../specs/product/automatic-memory.md#855a3c1-uat-scenarios)) to `recall`; writing rules ([793a2c Lessons are written without the agent](../../specs/product/automatic-memory.md#793a2c-lessons-are-written-without-the-agent), [855a2b Lessons are written as the learn skill writes them](../../specs/product/automatic-memory.md#855a2b-lessons-are-written-as-the-learn-skill-writes-them) through [855a2d Lessons that correct a note say so](../../specs/product/automatic-memory.md#855a2d-lessons-that-correct-a-note-say-so), [855a2g Lessons are durable and atomic](../../specs/product/automatic-memory.md#855a2g-lessons-are-durable-and-atomic), [793a3c Nothing the vault already says is written again](../../specs/product/automatic-memory.md#793a3c-nothing-the-vault-already-says-is-written-again)) to `learn`; [793a2f The conversation becomes searchable memory](../../specs/product/automatic-memory.md#793a2f-the-conversation-becomes-searchable-memory) to `remember-what-happened`; [793a2a On after `engram update`](../../specs/product/automatic-memory.md#793a2a-on-after-engram-update) to `set-up-an-environment-workflow`; [793a2e The user can see what it did](../../specs/product/automatic-memory.md#793a2e-the-user-can-see-what-it-did) to the workflow | OUT-2, 5, 3 |
| [Chunk Indexing](../../specs/product/chunk-indexing.md) | 792a5 | MECH, reread as CAP | Turns session transcripts and Markdown documents into a chunk index outside the vault, so that `engram query` can search them as evidence. | keep, renamed `remember-what-happened` | OUT-3, 2, 4 (section 3.1 adds 7) |
| [Controlled Vocabulary](../../specs/product/controlled-vocabulary.md) | 792a6 | MECH (tags) | Keeps a controlled vocabulary of defined terms and tags every note engram writes with the terms closest to it. | merge into `recall` (explore nomination); the Obsidian visibility part to `keep-memory-accurate` | OUT-1, 5 |
| [Curate](../../specs/product/curate.md) | 792a22 | CAP | Judges each pending offer against what the vault already holds and settles it. | merge into `keep-memory-accurate`; its offer-judging rules referenced by `share-memory` | OUT-1, 4 |
| [Development Checks](../../specs/product/development-checks.md) | 794a2 | MECH (repo nil analysis) | Reports nil check and fix findings only in Engram's module packages, under the existing command names. | contextual entry or a technical spec; serves no product outcome (NG-5, [858d32 NG-5: general development tooling outside memory](./858d_questions-and-decisions.md#questions-and-decisions)) | none |
| [Install and Update](../../specs/product/install-and-update.md) | 792a11 | WF (leans) | Brings a machine to the current engram: installs the binary and deploys skills and mods into each coding harness it finds. | keep as `set-up-an-environment-workflow` (install, deploy, register, init) | OUT-4, 2 |
| [Learn](../../specs/product/learn.md) | 792a15 | CAP | Keeps memory true at the end of work and at the moments a lesson is confirmed. | keep as `learn` | OUT-1, 2 |
| [Lessons Contract](../../specs/product/lessons-contract.md) | 792a16 | WF (subagent handoff) | The agreement between the orchestrating agent and every subagent it dispatches: each completion report ends with a `LESSONS:` line. | keep as `delegated-work-workflow`, with route and please | OUT-1, 2 |
| [Memory Effectiveness Evaluation](../../specs/product/memory-effectiveness-evaluation.md) | 792a23 | unsure | Measures whether engram's memory helps an agent, with dev tooling in this repository that is not part of the shipped command line. | contextual entry (maintainer tooling) under [858d15 Q7 Is measuring memory's value part of the product?](./858d_questions-and-decisions.md#questions-and-decisions) option (b), or capability `measure-memory-value` under (a) | OUT-5 |
| [Memory Query](../../specs/product/memory-query.md) | 792a7 | MECH (`engram query` engine) | Searches the vault's notes and the chunk index for what is relevant to a set of phrases and prints one YAML payload. | merge into `recall` as software promises (a user-facing command stays a product promise, s:7bc2f2bd@10-15T01:13Z) | OUT-1, 2, 7 |
| [Note Revision](../../specs/product/note-revision.md) | 792a4 | MECH (amend, resituate, activate) | Changes notes that already exist: amend replaces fields, supersessions and chunk sources, resituate rewrites the situation, activate records that a note was used. | merge into `keep-memory-accurate` | OUT-1 |
| [Note Writing](../../specs/product/note-writing.md) | 792a3 | MECH (`engram learn` fields, placement) | Writes new fact, feedback, runbook and question-and-answer notes into the vault's Luhmann hierarchy. | merge into `learn` | OUT-1 |
| [Parent Sync](../../specs/product/parent-sync.md) | 792a21 | CAP (leans) | Lets a vault exchange notes with one parent vault: offers what it writes upward and reads the parent's notes back, while the local vault always answers. | keep as `share-memory` | OUT-4 |
| [Please](../../specs/product/please.md) | 792a18 | WF | Carries a handed-over multi-step piece of work end-to-end through a fixed seven-step workflow. | keep as `handed-over-task-workflow` | OUT-1, 2 |
| [Recall](../../specs/product/recall.md) | 792a14 | CAP | Runs before work that takes more than a single tool call or quick thought: states the plan, retrieves chunks and notes in one ranking and judges them against the plan. | keep as `recall` | OUT-1, 2, 3, 7 (section 3.1 adds 6) |
| [Route](../../specs/product/route.md) | 792a17 | CAP | Chooses the agent type, model and effort of a subagent dispatch from recorded evidence, starting each unit at the cheapest tier. | keep as `route` | OUT-6, 1 (section 3.1 adds TH-1) |
| [Skill Runbook Registration](../../specs/product/skill-runbook-registration.md) | 792a12 | MECH (scanners, keys, declines) | Offers a runbook note in the vault for each skill, command and prompt template a harness loads, so that memory can surface the procedure by situation and trigger. | merge into `set-up-an-environment-workflow` ([792a12t Update runs registration after deploying the skills](../../specs/product/skill-runbook-registration.md#792a12t-update-runs-registration-after-deploying-the-skills): update runs it); [792a12v Skill notes behave like captured runbooks](../../specs/product/skill-runbook-registration.md#792a12v-skill-notes-behave-like-captured-runbooks) (skill notes behave as runbooks) into `recall` | OUT-4, 1 |
| [Vault Counting](../../specs/product/vault-counting.md) | 792a8 | MECH (`engram count`) | Answers how many notes carry each value of a frontmatter attribute and how many notes link to a given note. | merge into `keep-memory-accurate` (inspect); route audits reference it | OUT-5, 6 |
| [Vault Health](../../specs/product/vault-health.md) | 792a9 | MECH (check, embed census) | Reports where a vault breaks its invariants and counts and repairs the sidecars that hold note embeddings. | [792a9a The check report](../../specs/product/vault-health.md#792a9a-the-check-report) through [792a9f A check changes nothing](../../specs/product/vault-health.md#792a9f-a-check-changes-nothing) into `keep-memory-accurate`; [792a9g The embedding census](../../specs/product/vault-health.md#792a9g-the-embedding-census) through [792a9j CRLF notes embed as LF](../../specs/product/vault-health.md#792a9j-crlf-notes-embed-as-lf) (embedding) to a technical spec | OUT-5 |
| [Vault Identity](../../specs/product/vault-identity.md) | 792a2 | MECH (location, ID, first use) | Locates the vault, brings it into existence and gives it a stable identity that keeps a copied, cloned or self-parented vault from exchanging notes under another vault's name. | [792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location) through [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) into `set-up-engram`; [792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id) through [792a2q A vault does not exchange with itself](../../specs/product/vault-identity.md#792a2q-a-vault-does-not-exchange-with-itself) (ID, exchange state) into `share-memory` | OUT-4 |
| [Vault Migration](../../specs/product/vault-migration.md) | 792a10 | MECH (one-off re-parent, backfill) | Offers the one-off changes that no ordinary write makes: re-parenting top-level notes and stamping a missing note origin. | merge into `keep-memory-accurate` | OUT-1 |
| [Vault Serving](../../specs/product/vault-serving.md) | 792a20 | MECH (HTTP routes) | Serves a node's notes to its children over HTTP through four commands only: query, show, activate and learn. | merge into `share-memory` | OUT-4 |
| [Write Memory](../../specs/product/write-memory.md) | 792a13 | MECH (worker) | Turns a decision to remember something into a vault note: composes and runs the matching `engram learn` command and reports the written path. | merge into `learn` (allocation), or move to a technical spec; qa rows [792a13f Question-and-answer notes](../../specs/product/write-memory.md#792a13f-question-and-answer-notes) and [792a13j A qa note takes no tags](../../specs/product/write-memory.md#792a13j-a-qa-note-takes-no-tags) retire with NG-7 | OUT-1 |

The Outcomes column is the capability-to-outcome map; it is applied at promotion, not in this change ([858d53 M6 Capability recipe in place](./858d_questions-and-decisions.md#questions-and-decisions)).

Proposed capabilities of the synthesis (its section 3.1), not applied in this change ([858d80 Regroup the 24 product files per synthesis section 3.2? (author addition; the synthesis asks "Confirm")](./858d_questions-and-decisions.md#questions-and-decisions)). Under the guides, "automatic" is an allocation of recall and learn to the software, not a separate ability.

| Capability (proposed slug) | Ability | Outcomes served |
| --- | --- | --- |
| `recall` | Consult what memory holds before acting, judge what applies, and work it into the plan | OUT-1, 2, 3, 6, 7 |
| `learn` | Recognize a lesson worth keeping, state it as a durable general rule, and place it | OUT-1, 2 |
| `remember-what-happened` | Keep a searchable record of past sessions and files (episodic) | OUT-3, 2, 4, 7 |
| `keep-memory-accurate` | Keep the vault non-duplicated, current and inspectable: judge offers, revise, supersede, check, count | OUT-1, 5, 7 |
| `share-memory` | Exchange notes between a local vault and a parent | OUT-4 |
| `route` | Choose the cheapest adequate worker from recorded evidence | OUT-6, 1, TH-1 |
| `set-up-engram` (or a workflow; the synthesis names it both ways) | Make memory available, current and complete in each harness and environment | OUT-4, 2 |

## 858a4 Business workflow candidates

Per [858d57 M10 Business workflows](./858d_questions-and-decisions.md#questions-and-decisions) none is created as a file in 858; each is a candidate pending [858d81 Business workflow candidates (synthesis section 3.3) and the headline file (author addition)](./858d_questions-and-decisions.md#questions-and-decisions). SPEC 106-107: a separate workflow spec only when composition needs its own maintained software contract. The synthesis's recommendation is its section 3.2 home where it gives one.

| Candidate | Participants | Evidence (current owner files; outcomes) | Recommendation (synthesis) |
| --- | --- | --- | --- |
| Intentional memory loop: a task starts; recall, act, learn at a correction or at the end | user, agent | [Recall](../../specs/product/recall.md) (792a14), [Learn](../../specs/product/learn.md) (792a15), [Ambient Guidance](../../specs/product/ambient-guidance.md) (792a19); OUT-1 | No file proposed in section 3.2; composition of recall and learn; keep as capability context unless it needs its own contract |
| Automatic memory: a prompt or a moment within a turn arrives; the moment is judged none/glance/deep; recall is delivered; the agent acts; the lesson is judged and written; the activity is shown | user, host (mod), judge (cheap tier), worker (mid tier), agent | [Automatic Memory](../../specs/product/automatic-memory.md) ([793a2 Automatic memory](../../specs/product/automatic-memory.md#793a2-automatic-memory), [793a3 Interactions](../../specs/product/automatic-memory.md#793a3-interactions) and [855a2 Judged automatic memory](../../specs/product/automatic-memory.md#855a2-judged-automatic-memory), with their descendants); the technical workflows [Prompt Delivery](../../specs/technical/prompt-delivery-workflow.md), [Turn Capture](../../specs/technical/turn-capture-workflow.md), [Conversation Sweep](../../specs/technical/conversation-sweep-workflow.md) and [Mid-Turn Recall](../../specs/technical/mid-turn-recall-workflow.md); OUT-2, 3, 5 | Keep as `automatic-memory-workflow` (the headline file; owner decision, [858d26 Q17 How do the three "north stars" layer?](./858d_questions-and-decisions.md#questions-and-decisions), [858d81 Business workflow candidates (synthesis section 3.3) and the headline file (author addition)](./858d_questions-and-decisions.md#questions-and-decisions)); content rules go to their capability owners |
| Delegated work: a dispatch goes out; the subagent recalls; it returns with a LESSONS line; the orchestrator judges; the closing learn runs | orchestrator, subagent | [Lessons Contract](../../specs/product/lessons-contract.md) ([792a16a A completion report ends with a LESSONS line](../../specs/product/lessons-contract.md#792a16a-a-completion-report-ends-with-a-lessons-line) through [792a16g A confirmed approach is looked for at the moment of confirmation](../../specs/product/lessons-contract.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation)), [Route](../../specs/product/route.md), [Ambient Guidance](../../specs/product/ambient-guidance.md) [792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment) through [792a19f A returned line that fails the bar is not captured](../../specs/product/ambient-guidance.md#792a19f-a-returned-line-that-fails-the-bar-is-not-captured); OUT-1, 2, 6 | Keep as `delegated-work-workflow` with route and please; open conflict: the owner said agents need not report back once capture is automatic (s:7bc2f2bd@10-11T19:19Z; .review:39) and subagent capture is itself pending [858d25 Q16 The 2026-10-16 restart reverses 14 earlier answers by the author's default (793c13, still open): accept all, or name rows to keep?](./858d_questions-and-decisions.md#questions-and-decisions) row 1 ([858d16 Q8 Are please, route and the delegation guidance product outcomes, and does the lessons contract stay?](./858d_questions-and-decisions.md#questions-and-decisions)) |
| Handed-over task (please): `/please`, then seven gated steps with fresh reviewers | user, orchestrator, reviewers | [Please](../../specs/product/please.md) ([792a18a The skill fires on a handed-over multi-step ask](../../specs/product/please.md#792a18a-the-skill-fires-on-a-handed-over-multi-step-ask) through [792a18s Object-level work is delegated](../../specs/product/please.md#792a18s-object-level-work-is-delegated)); OUT-1 ([858d16 Q8 Are please, route and the delegation guidance product outcomes, and does the lessons contract stay?](./858d_questions-and-decisions.md#questions-and-decisions)) | Keep as `handed-over-task-workflow`, subject to [858d16 Q8 Are please, route and the delegation guidance product outcomes, and does the lessons contract stay?](./858d_questions-and-decisions.md#questions-and-decisions) |
| Offer lifecycle: a child learn; outbox; the parent receives it as pending; curate accepts, folds or discards; pull-down | child vault, parent vault, curator | [Parent Sync](../../specs/product/parent-sync.md) (792a21), [Curate](../../specs/product/curate.md) (792a22); OUT-4 | Section 3.2 gives it no file: the offer-judging rules are referenced by share-memory; candidate only |
| Set up an environment: `engram update` or `init`; install, deploy to each harness, register skills, opt into guidance, create the vault | user, installer | [Install and Update](../../specs/product/install-and-update.md) (792a11, leans workflow), [Skill Runbook Registration](../../specs/product/skill-runbook-registration.md) (792a12), [Vault Identity](../../specs/product/vault-identity.md) ([792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation)), [Automatic Memory](../../specs/product/automatic-memory.md) [793a2a On after `engram update`](../../specs/product/automatic-memory.md#793a2a-on-after-engram-update); OUT-4, 2 | Keep as `set-up-an-environment-workflow` (install-and-update is flagged "leans") |
| Measure memory's value, if product (858d15): a trial or audit run; score; a ledger row | maintainer | [Memory Effectiveness Evaluation](../../specs/product/memory-effectiveness-evaluation.md) (792a23); OUT-5 | Only if [858d15 Q7 Is measuring memory's value part of the product?](./858d_questions-and-decisions.md#questions-and-decisions) option (a); otherwise a contextual entry |

## 858a5 Modified: Scenario of existing behavior

### 858a5a Modified: A request to remember is written when the agent says it did not save it

Target: [793a2c1i A request to remember is written when the agent says it did not save it](../../specs/product/automatic-memory.md#793a2c1-uat-scenarios). Requirement [793a2c Lessons are written without the agent](../../specs/product/automatic-memory.md#793a2c-lessons-are-written-without-the-agent) as modified at promotion by 855a3w says "durable"; the scenario's When said "remember something".

```gherkin
Scenario: 793a2c1i A request to remember is written when the agent says it did not save it
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something durable and not to save it itself, the agent says that it did not save it, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically
```
