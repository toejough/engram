# Premortem: Cycle 4 Feature Requests — User Interaction, Performance, Sophistication

**Date:** 2026-03-18
**Scenario:** "It is 2026-04-01. Cycle 4 shipped ten features centered on user interaction,
performance, and core value proposition sophistication. Several are broken or actively harmful.
What went wrong?"

## Hypothetical Features

Ten features imagined for Cycle 4:

1. TF-IDF secondary duplicate detection signal (UC-34, #305)
2. Concept-aware spreading activation via graph traversal
3. Memory decay / auto-expiration based on low effectiveness
4. Interactive memory triage CLI (approve/reject surfaced candidates)
5. Real-time session effectiveness dashboard
6. Multi-project memory scoping (memories tagged per project)
7. Smart token budget allocation (dynamic per hook phase)
8. Adversarial prompt injection detection in memory content
9. Session summary injection at session start
10. Memory export / portability across machines

---

### Failure 1: TF-IDF False Positives Silently Merge Unrelated Memories

**What went wrong:**
TF-IDF scores computed over 5–15 keyword fields produce high similarity scores for short
memories that share common words ("test", "error", "function") but express unrelated
constraints. The similarity threshold tuned on synthetic test data doesn't hold at real
keyword density. The cluster merge code treats TF-IDF as a confirmation signal and promotes
two unrelated memories into a merge cluster. The absorbed memory is permanently deleted.
The user notices weeks later that a constraint they remember setting — "never use `var`
in Go" — is no longer being respected. There is no audit trail of why it was merged.

**Principle violated:** "Content quality > mechanical sophistication." Adding a signal
that sounds rigorous but degrades correctness is worse than no signal.

**What would have caught it:**
1. A property-based test: for any two memories with zero semantic relationship (disjoint
   keyword sets of length ≥ 3), TF-IDF similarity score must be < merge threshold.
2. A dry-run mode (already planned, see prior premortem) that outputs the proposed
   merge cluster before executing, so the user can audit the decision.
3. A test that seeds the real memory corpus with 50 entries and asserts dedup rate
   stays under a known ceiling (e.g., < 5% merge rate).

**Remediation:**
Add a REQ that TF-IDF is a tie-breaker signal only — it can raise confidence of an
existing keyword-overlap cluster but cannot form a cluster on its own. Add a test that
seeds two memories with identical keywords but semantically opposite content and asserts
they are NOT merged. Ship `--dry-run` before TF-IDF lands.

**Likelihood × Impact:** HIGH × HIGH = **#1 risk**

---

### Failure 2: Graph Spreading Activation Cycles Exhaust Token Budget

**What went wrong:**
Concept-aware spreading activation traverses the memory link graph starting from a seed
memory and follows links to surface related memories. The graph has cycles (A → B → C → A)
because link recomputation after merge doesn't guarantee acyclicity. The traversal has a
depth limit but no visited-node guard. At depth limit = 3 with 3-node cycles, the traversal
visits the same nodes repeatedly. The returned candidate set balloons to 40+ memories,
blowing the token budget and either (a) truncating the prompt catastrophically or (b)
blocking the hook entirely. The user experiences a 15-second freeze on every session start.

**Principle violated:** "DI everywhere." The traversal logic has no injectable
depth/budget limit interface — limits are hardcoded in the traversal function, untestable
without a real graph.

**What would have caught it:**
1. A unit test with a 3-node cycle graph that asserts the traversal terminates in < N
   calls and returns ≤ K results.
2. An ARCH constraint: spreading activation must track visited nodes and terminate in
   O(|nodes|) calls, not O(budget^depth).
3. A performance integration test: call surface with a 100-node cyclic graph, assert
   wall-clock < 1s.

**Remediation:**
Add ARCH constraint: graph traversal must use a visited-set and terminate in linear time.
Add T-item: given a cyclic graph, spreading activation returns without revisiting any node.
Wire `WithMaxActivationDepth` and `WithVisitedSet` as DI parameters so they're testable.

**Likelihood × Impact:** HIGH × HIGH = **#2 risk**

---

### Failure 3: Effectiveness-Based Decay Wipes New-Session Memory Corpus

**What went wrong:**
Memory decay assigns a "health score" derived from evaluation outcomes. Memories that
haven't been evaluated (new session, no historical data) have health = 0. The decay
threshold check fires before any evaluation data exists. On the first run of a fresh
install — or after deleting the `evaluations/` directory — the decay pass expires all
memories because none have positive health scores. The user loses their entire corpus
on first maintenance run with no warning and no undo.

**Principle violated:** "Passing tests ≠ usable system." All decay tests used pre-seeded
evaluation data. The zero-evaluation case was never exercised.

**What would have caught it:**
A binary smoke test (already required by #322) that runs maintain with an empty
`evaluations/` directory and asserts zero memories are deleted.

**Remediation:**
Add a REQ: decay must not mark any memory for expiration unless it has ≥ N evaluation
records (minimum evidence threshold). Add a T-item for the zero-evaluation cold-start
case. Wire the minimum-evidence threshold as a DI parameter.

**Likelihood × Impact:** MEDIUM × HIGH = **#3 risk**

---

### Failure 4: Session Summary Race With stop.sh Injects Stale Context

**What went wrong:**
Session summary injection reads from `evaluations/` to synthesize a previous-session
summary at session start. stop.sh writes evaluation data asynchronously after the session
ends. If the user starts a new session within seconds of ending the previous one (common
in rapid-iteration workflows), the summary reads incomplete evaluation data — the last 3
turns are missing. The injected summary says the previous session accomplished nothing
meaningful. The agent re-does work that was already completed, wasting a full turn and
confusing the user.

**Principle violated:** The per-turn log isolation fix (T-345, ARCH-81) only isolated
within a session. Cross-session read/write ordering was never specified.

**What would have caught it:**
A T-item: given that stop.sh has not yet completed writing evaluations, session-start
summary must either (a) skip injection or (b) explicitly note data may be incomplete.

**Remediation:**
Add a REQ: session summary must read only evaluation files whose modification time is
more than T seconds old (configurable fence). Add an ARCH constraint: session-start
hook fires after stop.sh completion, or session summary explicitly marks recency gaps.

**Likelihood × Impact:** MEDIUM × HIGH = **#4 risk**

---

### Failure 5: Memory Export Breaks on Absolute Paths at Import Target

**What went wrong:**
Memory export writes TOML files with `links[].target` values as absolute paths
(e.g., `/Users/joe/memories/concept.toml`). Importing on a second machine fails silently:
the registry loads the file, but all links resolve to non-existent paths. The graph
is empty on the target machine even though 80 memories imported successfully. The user
assumes their memory graph didn't transfer, files a bug, and is told "works on my machine."

**Principle violated:** The `toRelID` fix in Cycle 3 converted absolute paths to
relative IDs before registry storage — but export bypassed the registry and read raw
TOML files directly, missing the relativization step.

**What would have caught it:**
A round-trip test: export to a tempdir with a different base path, re-import, assert
all link targets resolve against the new base path.

**Remediation:**
Add REQ: exported memory files must use registry-relative paths for all `links[].target`
fields. Add T-item: export → import to different base path → all links valid.

**Likelihood × Impact:** MEDIUM × MEDIUM = **#5 risk**

---

## Priority Matrix

| Failure | Likelihood | Impact | Priority |
|---------|-----------|--------|----------|
| 1: TF-IDF false positives → silent data loss | High | High | 🔴 P0 |
| 2: Graph cycles exhaust token budget | High | High | 🔴 P1 |
| 3: Decay wipes cold-start corpus | Medium | High | 🟡 P2 |
| 4: Session summary reads incomplete data | Medium | High | 🟡 P3 |
| 5: Export breaks on path relocation | Medium | Medium | 🟡 P4 |

---

## Mitigations (Top 2)

### Mitigation A: TF-IDF as Tie-Breaker Only, Not Cluster Initiator (Failure 1)

Before UC-34 / TF-IDF ships:
1. Add REQ: TF-IDF may only increase confidence in an existing keyword-overlap cluster.
   It may not initiate a cluster on its own.
2. Add T-item: two memories with disjoint keyword sets must never form a merge cluster
   regardless of TF-IDF score.
3. Require `--dry-run` on the maintain command before UC-34 lands (no new memory-mutating
   feature ships without dry-run visibility).

**Action:** File issue for dry-run gate on TF-IDF merge (blocks UC-34 start).
File issue for TF-IDF-as-tie-breaker REQ before spec derivation begins.

### Mitigation B: Graph Traversal Termination Guarantee (Failure 2)

Before spreading activation ships:
1. Add ARCH constraint: all graph traversal must track a visited-node set and terminate
   in O(|V|) node visits.
2. Add T-item: given a cyclic graph with N nodes, activation returns ≤ N results and
   completes without infinite recursion.
3. Wire `maxDepth` and visited-set as DI parameters so they're unit-testable without
   a real graph.

**Action:** File issue for spreading activation termination ARCH constraint and
accompanying T-item before any graph traversal implementation starts.
