# Recall Skill: Empty-Vault = CREATE Band Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:writing-skills (this is a SKILL.md behavior change — RED/GREEN/pressure-test, not code TDD). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strengthen `skills/recall/SKILL.md` Step 2.5 so agents treat clusters with no/low `nearest_l2` — especially on an empty or L2-less vault — as the CREATE band, instead of skipping crystallization.

**Architecture:** Pure skill-text change. The band table already lists `< 0.80, or no nearest_l2 → CREATE`, but agents repeatedly rationalize "the vault has no L2s, so there's nothing to band against → skip." Fix targets the rationalization: explicit bootstrap framing, an unmissable empty-vault rule, and a red-flag row. Deployed copy at `~/.claude/skills/recall/SKILL.md` is a file copy of the repo source and must be synced.

**Tech Stack:** Markdown skill text; writing-skills TDD (subagent pressure tests via Agent tool).

**Considered and raised:** the rule technically already exists in the band table; user's field reports (multiple machines) show it gets ignored, so the change is anti-rationalization hardening, not a new rule. User's framing affirmed.

---

### Task 1: RED — baseline pressure test against current skill text

**Files:**
- Create: `skills/recall/tests/baseline-empty-vault-skip-l2.md` (scenario + results)

- [ ] **Step 1: Write the pressure scenario.** Subagent prompt: paste current Step 2.5 verbatim; present a query payload summary where the vault contains zero L2 notes, clusters came back with `nearest_l2` absent (or omitted entirely by the binary), and a colleague argues "there are no L2s to compare against and zero items under the 0.80 threshold surfaced, so Step 2.5 doesn't apply — proceed to Step 3." Ask the agent what it does.
- [ ] **Step 2: Run it (Agent tool, fresh subagent, no other context).** Expected RED: agent agrees to skip, or hedges. If the agent already creates notes, tighten the scenario (e.g., binary output shows no `nearest_l2` key at all) until the failure reproduces or three honest variants all pass.
- [ ] **Step 3: Record verbatim verdicts in the test file.**

### Task 2: GREEN — strengthen Step 2.5 and red flags

**Files:**
- Modify: `skills/recall/SKILL.md` (Step 2.5 + red-flag table)

- [ ] **Step 1: Apply the edit.** After the band table, add:

```markdown
**Empty/L2-less vault = CREATE band, always.** Every vault starts with zero L2 notes. When the
vault has no L2s — `nearest_l2` missing, null, or wildly low on every cluster — that is not
"Step 2.5 doesn't apply"; it is the strongest possible CREATE signal. The first L2s a vault
ever gets are created exactly here, by this step. "No items under 0.80 came through" cannot
mean skip: a cluster with nothing above 0.80 *is* the `< 0.80` band. If you process N chunk
clusters and write 0 notes on an L2-less vault, you have executed the step wrong — the only
exemption is the vocabulary-coincidence gate, stated per cluster, out loud.
```

And add red-flag rows:

```markdown
| You skipped Step 2.5 because the vault has no L2 notes yet (or no `nearest_l2` came back) | That IS the CREATE band — bootstrap L2s are created here; missing `nearest_l2` never means "not applicable" |
| You wrote 0 notes from N clusters on an L2-less vault without stating a per-cluster vocabulary-coincidence call | Band every cluster out loud; absent `nearest_l2` defaults to CREATE |
```

- [ ] **Step 2: Re-run the Task 1 scenario verbatim against the new text.** Expected GREEN: agent refuses the skip and creates notes.

### Task 3: Pressure tests (REFACTOR loopholes)

- [ ] **Step 1: Run 2–3 fresh-subagent pressure variants:** (a) "the clusters are low-silhouette, probably noise"; (b) "we're in a hurry, crystallize next session"; (c) "all clusters are about the same topic, one note feels redundant — skip all." Expected: agent holds the line (writes, or states per-cluster vocabulary-coincidence rationale).
- [ ] **Step 2: Close any loophole found with a wording tweak; re-run that variant.**
- [ ] **Step 3: Append results to the test file.**

### Task 4: Sync deploy + document + commit

- [ ] **Step 1:** `cp skills/recall/SKILL.md ~/.claude/skills/recall/SKILL.md`
- [ ] **Step 2:** Check `docs/` (GLOSSARY, architecture) for Step 2.5 descriptions needing the same nuance; update if present.
- [ ] **Step 3:** Commit via /commit: `fix(skills): recall — empty/L2-less vault is the CREATE band, not a skip`
