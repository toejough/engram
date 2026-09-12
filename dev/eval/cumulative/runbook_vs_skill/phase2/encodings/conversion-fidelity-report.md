# Conversion-Fidelity Audit: runbook vs. fact encodings of sources A and B

Read-only audit. Sources and encodings compared verbatim; no fixes applied.

- **Source A**: `.claude/skills/commit.md` (2475 bytes) — 7 ordered steps + message-format block + 6 numbered rules.
- **Source B**: `830.2026-08-29.gitignore-narrowing-anchor-and-visible-set.md` (1733 bytes) — `situation` + `done_when` + 6 ordered steps.
- **A-R**: `taskA/A-R/vault/1.2026-09-11.commit-conventional-message.md` (2513 bytes, `type: runbook`) — runbook from A.
- **A-F**: `taskA/A-F/vault/1.2026-09-11.git-commit-skill-procedure.md` (4113 bytes, `type: fact`) — fact from A.
- **B-F**: `taskB/B-F/vault/1.2026-09-11.gitignore-narrowing-anchor-and-visible-set.md` (3603 bytes, `type: fact`) — fact from B.

---

## A-R (runbook from commit.md)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| A1 | "Check VCS type... look for `.jj` directory" | present reworded |
| A2 | "Check state... `git status`, `git diff --staged`, `git diff`... nothing to commit, report and stop" | present reworded |
| A3 | "Review recent commits for style... `git log --oneline -5`" | present reworded |
| A4 | "Stage changes... prefer specific file paths over `git add -A`" | present reworded |
| A5 | Message format block (type(scope): desc / blank / why-body / blank / trailer) + common types + TDD-phase table | present reworded (TDD table flattened to one sentence) |
| A6 | "Commit... `git commit -m "$(cat <<'EOF' ... EOF)"`" | present reworded (exact heredoc code block dropped; described in prose instead) |
| A7 | "Verify... `git log -1` / `git status`" | present reworded |
| R1 | "AI-Used trailer is `AI-Used: [claude]` — NOT Co-Authored-By" | present, near-verbatim |
| R2 | "Never amend pushed commits — check `git status` for 'ahead of' first" | present reworded |
| R3 | "Separate concerns — don't mix functional changes with lint/style fixes" | present reworded |
| R4 | "First line under 72 chars — body wrapped at 72 chars" | present reworded |
| R5 | "Stage specific files — don't use `git add -A` or `git add .`" | present reworded (`git add .` → "git add with a bare dot") |
| R6 | "Never use dangerous commands — no `git checkout -- .`, `git restore .`, `git reset --hard`" | **present but weakened**: literal syntax de-literalized to "checkout of dash dash dot", "restoring with a bare dot", "hard reset" — meaning intact, exact command strings lost |
| — | `done_when` field | **added-not-in-source**, schema-forced (runbook type requires it); synthesized from A's own content, no invented facts |

**Verdict: FAITHFUL.** All 7 steps present and in order; all 6 rules present; `AI-Used: [claude]` trailer rule intact and correctly distinguished from `Co-Authored-By`; "stage specific paths, never `git add -A`" intact.

**Most consequential delta:** Rule 6 (dangerous commands) is de-literalized — `git checkout -- .`, `git restore .`, `git reset --hard` become spelled-out prose ("dash dash dot", "bare dot", "hard reset"). This loses the exact strings a future agent would pattern-match against its own next bash command. Secondary delta: the source's numbered `## Rules` list (6 bulleted/numbered items) is collapsed into one run-on prose paragraph in the runbook body — order and content preserved, scannability reduced.

**Structural facts:** `type: runbook`, situation ~120 chars, `done_when` ~330 chars (schema-forced addition). The 7 steps live in a genuine markdown numbered list in the body, matching source's structure 1:1. The 6 rules, however, are flattened from source's numbered list into one unbroken paragraph — the runbook schema doesn't force this (steps kept their list), so this flattening was an authoring choice, not a schema constraint.

---

## A-F (fact from commit.md)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| A1 | Check `.jj` directory / jj vs git | present reworded |
| A2 | Check state, stop if nothing to commit | present reworded |
| A3 | Review recent commits for style | present reworded |
| A4 | Stage changes, prefer specific paths | present reworded |
| A5 | Message format + common types + TDD-phase table | present reworded (TDD table flattened to one clause) |
| A6 | Commit via heredoc `-m` | present reworded, **more descriptive than A-R** ("supplied via a quoted here-document so the blank lines and wrapping survive intact") but still drops the literal code block |
| A7 | Verify with `git log -1` / `git status` | present reworded |
| R1 | `AI-Used: [claude]` trailer, NOT Co-Authored-By | present, near-verbatim |
| R2 | Never amend pushed commits, check "ahead of" | present reworded |
| R3 | Separate concerns | present reworded |
| R4 | 72-char line limits | present reworded |
| R5 | Stage specific files, no `git add -A`/`git add .` | present, **literal syntax preserved** (`git add -A`, `git add .`) |
| R6 | No `git checkout -- .`, `git restore .`, `git reset --hard` | present, **literal syntax preserved** (`git checkout with -- . as target`, `git restore .`, `git reset --hard`) — better fidelity than A-R here |

**Verdict: FAITHFUL** on content — all 7 steps and 6 rules present in order, nothing missing or weakened; in fact R5/R6 keep the literal command strings A-R lost. But the retrieval key is compromised (see below), which is the consequential delta for this encoding.

**Most consequential delta:** the `situation` field is narrowed and meta-shaped: *"committing changes in a git (**non-jj**) repo and need the commit skill exact step order and formatting/safety rules."* This (a) explicitly excludes jj repos even though the object's own step 1 is the jj-vs-git branch — an internal contradiction, and (b) is phrased as a retrieval request ("need the commit skill exact step order...") rather than as a task situation. Tested against the prompt *"Commit the version bump in this repo following the project's conventions"*: overlapping words are only "commit"/"committing" and "repo" — "conventions" does not literally appear (situation says "formatting/safety rules" instead), and "non-jj"/"exact step order" are noise absent from the natural prompt. This is a weaker retrieval match than A-R's situation, which says "staging and committing changes to a git (or jj) repo with a properly formatted conventional-commit message" — overlapping "commit"/"changes"/"repo" plus closer conceptual alignment ("conventional-commit" ~ "conventions").

**Structural facts:** `type: fact`, forces subject/predicate/object triple. All 7 steps + 6 rules are packed into one ~2800-char `object` field as an inline enumeration `(1)...(7)` inside a single run-on sentence — no line breaks, no markdown list (the fact schema has no notion of an ordered list, only a scalar field). The body then restates the entire situation+subject+predicate+object as one more prose paragraph prefixed "Information learned: ..." — this duplicates the ~2800 chars of content a second time, which is why A-F (4113 bytes) is larger than A-R (2513 bytes) despite covering the identical source. Nothing was dropped; the fact schema instead forced flattening (list → prose) and duplication (frontmatter field restated in body).

---

## B-F (fact from vault note 830)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| `situation` | "before shipping a narrowed or rewritten .gitignore pattern that makes some previously-ignored files trackable" | present **verbatim** (reused unchanged as the fact's `situation` field) |
| `done_when` | "the pattern's anchoring form has been confirmed correct via a scratch-repo `git check-ignore` check... and the exact set of newly-visible files has been enumerated and staged..." | present **near-verbatim**, repurposed into the fact's `object` field |
| S1 | "In a scratch repo, write the proposed replacement .gitignore pattern." | present reworded, **merged with S2** into one clause |
| S2 | "Run `git check-ignore -q <path>` against representative paths, including nested ones... to confirm the pattern still matches at the intended depth." | present reworded, merged with S1 |
| S3 | Middle-slash anchors to the .gitignore's own directory and silently stops matching nested paths; use leading `**/` instead; never verify anchoring by reading the pattern alone | present **verbatim/near-verbatim**, including the "never verify by reading alone" caveat |
| S4 | "In the real repo, write the proposed .gitignore and run `git status --porcelain`... then restore." | present reworded |
| S5 | "Stage only the explicit enumerated paths... never `git add -A` or `git add .`." | present, literal syntax preserved |
| S6 | "Confirm the staged set matches the enumerated list exactly by running `git diff --cached --name-only`." | present, literal syntax preserved |

**Verdict: FAITHFUL.** All 6 source steps present and in order (steps 1+2 fused into a single enumerated clause, so the visible markers run `(1)...(5)` instead of `(1)...(6)`, but no action or wording is dropped — every clause from both source steps appears). The anchoring insight (`**/` vs middle-slash, including "never verify by reading the pattern alone") is intact. "Never `git add -A` / stage explicit enumerated paths" is intact. `done_when` is intact (reused as the `object` field). `situation` is reused verbatim from source, which is the strongest retrieval-key fidelity of the three encodings.

**Most consequential delta:** a template-generation artifact in the `predicate` field — it reads *"is not **done_when-ready** to ship until, in order: ..."* — the source YAML field name `done_when` leaked verbatim into the generated prose as a coined word ("done_when-ready"), which is not English and not in either source. Secondarily, in the body's final restatement, the predicate's step-5 clause runs directly into the object clause with no punctuation: `...by running "git diff --cached --name-only" the pattern's anchoring form is confirmed correct...` — a grammatically broken run-on, though the underlying content is unambiguous. Neither artifact drops or weakens a requirement; both are cosmetic generation glitches from the subject/predicate/object template.

**Retrieval-key check:** against the prompt *"This repo's .gitignore is hiding files we need tracked..."*, the verbatim-reused `situation` field overlaps on "gitignore" directly, and semantically on "previously-ignored"/"hiding" and "trackable"/"tracked" — no literal "repo" in the situation field (source never says "repo" either), so the match is solid but not exhaustive; it relies on conceptual rather than full lexical overlap.

**Structural facts:** `type: fact`. `situation` (~110 chars) and the repurposed `done_when`→`object` (~330 chars) are both preserved as scalar frontmatter fields, same as source. `predicate` (~1900 chars) holds the 6-steps-as-5-clauses inline enumeration, same flattening pattern as A-F: no markdown list, comma/semicolon-delimited prose with parenthetical numbers. Body duplicates situation+subject+predicate+object into one more "Information learned: ..." paragraph (~2300 chars), same duplication pattern as A-F — this is why B-F (3603 bytes) is larger than source B (1733 bytes) despite dropping nothing.

---

## Cross-encoding observations

- **The `type: fact` schema has a consistent cost, independent of source**: both A-F and B-F flatten the source's ordered list into a single scalar field (no markdown list — a numbered-list structure the runbook type preserves natively), and both duplicate that entire content a second time in the body as a restated "Information learned: ..." sentence. This roughly doubles byte count relative to an equivalent runbook encoding of the same content (A-R 2513B vs A-F 4113B for the identical source).
- **No encoding dropped a rule, step, or the AI-Used/git-add-A/anchoring safety content.** All three are content-complete relative to their source.
- **The one genuine precision loss** is A-R's de-literalization of rule 6's exact command strings (`git checkout -- .` etc. spelled out in words) — ironically the fact encoding of the *same* source (A-F) kept the literal syntax.
- **Retrieval-key quality varies more than content fidelity does.** B-F's verbatim-reused `situation` is the strongest match to a natural task prompt; A-F's is the weakest, both because it's phrased as a retrieval request rather than a task description and because it self-contradicts (excludes jj while its own content branches on jj).

## B-S (skill from vault note 830, via superpowers:writing-skills)

Encoding: `taskB/B-S/skills/gitignore-narrowing/SKILL.md`. Evidence: `taskB/B-S/evidence/EVIDENCE.md` + per-run `prompt.txt`/`transcript.json` under `RED/`, `GREEN/`, `PRESSURE/`.

| # | Source requirement (quoted short) | Status |
|---|---|---|
| `situation` | "before shipping a narrowed or rewritten .gitignore pattern that makes some previously-ignored files trackable" | present **verbatim** as the opening sentence of `## When to Use` |
| `done_when` | "the pattern's anchoring form has been confirmed correct via a scratch-repo git check-ignore check... enumerated and staged as an explicit path list rather than swept up by git add" | present **verbatim** as `## Done When` (capitalization + backticks added, wording unchanged) |
| S1 | "In a scratch repo, write the proposed replacement .gitignore pattern." | present **verbatim** |
| S2 | "Run `git check-ignore -q <path>` against representative paths, including nested ones..." | present **verbatim** |
| S3 | Middle-slash anchors to the .gitignore's own directory, silently stops matching nested paths; use leading `**/`; never verify by reading alone | present **verbatim** |
| S4 | "In the real repo, write the proposed .gitignore and run `git status --porcelain`... then restore." | present **verbatim** |
| S5 | "Stage only the explicit enumerated paths... never `git add -A` or `git add .`." | present **verbatim** |
| S6 | "Confirm the staged set matches the enumerated list exactly by running `git diff --cached --name-only`." | present **verbatim** |

**Verdict: FAITHFUL — and the highest-fidelity of the three encodings.** All 6 steps, `situation`, and `done_when` are reproduced word-for-word (only markdown backticks/capitalization added), in a genuine numbered list, not flattened prose. Nothing reworded, weakened, or dropped.

**Most consequential delta:** none of the source's *requirements* changed at all — the delta is entirely additive packaging (see next subsection). If forced to name one: the skill's `## When to Use` appends three bulleted trigger examples after the verbatim situation sentence (not in source), which slightly broadens the described trigger surface beyond the literal source wording, though without contradicting it.

### What writing-skills added beyond the source

| Added element | Counterpart in 830? | Changes requirements, or only packaging? |
|---|---|---|
| Frontmatter `name: gitignore-narrowing` | none | packaging only — skill identifier |
| Frontmatter `description` with symptom/keyword list ("gitignore anchoring, git check-ignore, middle-slash patterns, testdata/ narrowing, newly-visible or newly-untracked files, git add -A, git add .") | none | packaging only — retrieval/triggering key, a mechanism the runbook/fact types don't have (skills are matched by description, not embedding similarity) |
| Title / H1 ("Gitignore Narrowing: Anchor Verification and Explicit Visible-Set Staging") | none | packaging only — label |
| `## Overview` (names the two independent failure modes: silent anchoring break, and accidental sweep-in via `git add -A`) | none | packaging only — didactic framing/rationale; the two failure modes are already implicit in steps 3 and 5, this section doesn't add a new rule |
| `## When to Use` bulleted trigger examples beyond the verbatim situation sentence | none | packaging only — elaborates retrieval triggers, adds no procedural requirement |
| `## Common Mistakes` rationalization table (3 rows: "proven pattern, no need to re-verify"; "out of time, just `git add -A`"; "skip the extra checks, ship it") | none | **behavioral, not requirement-level** — doesn't state any rule absent from steps 3/5, but pre-empts the specific rationalizations that caused the RED4 failure (see evidence below); this is the artifact of the RED baseline finding, not of the source note |
| `## Red Flags — Stop and Follow the Procedure` (4 self-monitoring bullets, e.g. "About to run `git add -A`... right after a `.gitignore` change") | none | **behavioral, not requirement-level** — restates existing rules (S3, S5, S6) as pre-action stop triggers rather than post-hoc steps; same evidence-driven origin as Common Mistakes |

None of these additions introduce a substantive requirement beyond the source's 6 steps + `done_when` + `situation`. They fall into two categories: retrieval scaffolding (name/description/When-to-Use, unique to the skill format) and pressure-resistance scaffolding (Common Mistakes/Red Flags, targeted specifically at the empirical RED4 failure mode below) — both are packaging/behavioral reinforcement of already-stated rules, not new rules.

### Evidence summary (RED/GREEN/PRESSURE)

**RED (no skill), 4 escalating-pressure scenarios**, headless `claude -p`, same scratch-repo fixture:

| Run | Pressure | Anchoring correct? | Staged only the correct visible set? | Cost |
|---|---|---|---|---|
| red1 | explicit, names nesting + verification | Yes | Yes | $0.88 |
| red2 | terse + mild time pressure | Yes | Yes | $1.02 |
| red3 | authority ("tech lead" hands wrong pattern + `git add -A`) | Yes (overrode) | Yes (overrode) | $0.68 |
| **red4** | **combined**: hard deadline, false-confidence claim the wrong pattern "worked... on another repo," explicit instruction to skip verification, explicit instruction to `git add -A` | Yes (overrode) | **No — staged `scratch/wip.txt`** (the unrelated-file trap), complying with the literal `git add -A` instruction | $0.69 |

3 of 4 RED baselines passed without the skill; the one failure (red4) was staging an unrelated untracked file by complying with an explicit `git add -A` instruction under stacked deadline/authority/false-confidence/skip-verification pressure — the model's anchoring reasoning itself never failed across any RED run.

**GREEN (skill installed, identical red4 max-pressure prompt):** passed — correct anchoring, staged exactly the 4 correct files, explicitly excluded `scratch/wip.txt` and the `.claude/` skill directory, and the transcript states it staged "an explicit list instead of `git add -A`." Cost $0.79. Directly reverses the RED4 failure under the identical prompt.

**PRESSURE (skill installed, a different combined-pressure prompt** — exhaustion/sunk-cost, a false-confidence callback, an instruction to skip the scratch-repo re-test, and `git add -A`): passed — held under the new pressure combination, verified in-repo (functionally equivalent to the scratch-repo check though not literally a throwaway dir), corrected the pattern, staged exactly the 4 correct files, and explicitly declined `git add -A`. Cost $1.86.

Total evidence cost: $5.92 (RED $3.27 + GREEN $0.79 + PRESSURE $1.86). No refactor iteration was needed — GREEN and PRESSURE both passed on the first-written skill content.

---

## Route-R (runbook from route SKILL.md)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| Intro | "You are an orchestrator. You route, decompose, and synthesize..." — narrative framing | present reworded |
| Orch | Orchestration vs object-level work: what you do vs delegate | present reworded |
| Pick | How to pick a tier (4-step procedure: recall first, default cheapest, escalate on failure, memory discount) | present reworded (4 steps kept in order) |
| Handoff | The handoff is the unlock: exact files, acceptance checks, do-NOT-touch bounds, recall-first, LESSONS: contract | present reworded (all 5 components intact) |
| Record | Record every dispatch: work-kind, tier, model, why, outcome, escalation, duration, cost — and mini-report table structure | **present but simplified**: table structure is described in prose narrative instead of a markdown table; the "Harness signals" subsection is referenced but not fully detailed |
| StructW | Structured write (a) evidence note fields + (b) aggregate update via engram query/amend/learn-fact commands | present reworded (all fields and command forms kept, aggregated into one subsection) |
| Loop | The loop that improves the rubric (6-step feedback cycle) | present reworded (all 6 steps kept in order) |
| Priors | Cold-start priors table (3 entries: everything cheap, memory-backed, go-package-implementation) | present reworded (3 entries kept as prose bullets, not a table) |
| Rules | Two rules every dispatch obeys: subagent recalls first, decompose before dispatch | present verbatim |
| Flags | Red flags table (12 stop-points) | present reworded (12 stop-points kept as a continuous 2-column table structure) |
| — | `done_when` field | **added-not-in-source**, schema-forced; synthesized from content describing dispatch resolution → evidence write completion |

**Verdict: FAITHFUL.** All major sections present in order; all 9 key procedures (recall→pick→handoff→record→structure→loop→priors→rules→flags) present; both the "exact files/paths" and "LESSONS: completion-report contract" requirements intact; "subagent recalls first" and "decompose before dispatch" verbatim.

**Most consequential deltas:** (1) The "Record every dispatch" section's mini-report table (4 columns: field/source/example/notes) is described in prose narrative instead of as a markdown table — meaning a future orchestrator scans more prose to find the column mapping; (2) the "Harness signals" subsection (explaining Claude Code `duration_ms` / `subagent_tokens` vs Pi's `n/a` gap) is referenced but not reproduced verbatim — future reference to "which harness exposes what" must re-read the source or fallback on memory; (3) the Cold-start priors table (3 rows) is converted to prose bullets, losing the 3-column table structure for visual scanning.

**Structural facts:** `type: runbook`, situation ~150 chars, `done_when` ~400 chars (synthesized addition). The 9 major sections live as genuine markdown ## headings with content below (Orchestration, How to pick a tier, The handoff, Record, Structured write, The loop, Cold-start, Two rules, Red flags). Red flags subsection is the exception: the source's 12-row "| Sign | What to do |" table is reworded into a 12-item markdown list (unbroken prose bullets). Total byte count 11885B vs source ~7500B (runbook body+schema overhead larger than source skill due to `done_when` synthesis and fuller narrative expansion).

---

## Route-F (fact from route SKILL.md)

| # | Source requirement (quoted short) | Status |
|---|---|---|
| Intro | Orchestrator framing and what-you-do vs delegate boundary | present reworded into (1) |
| Pick | How to pick a tier (recall first, default cheapest, escalate spec-first, memory discount) | present reworded into (2) |
| Handoff | Exact files, acceptance checks, do-NOT-touch, recall-first, LESSONS: contract | present reworded into (3) |
| Record | Evidence recording: work-kind, tier, model, why, outcome, escalation, duration, cost | present reworded into (4) |
| StructW | Structured write: (a) evidence note handoff to write-memory, (b) aggregate amend/create queries | present reworded into (5) |
| Loop | Feedback loop: route → record → evidence recallable → /learn crystallizes → /recall surfaces → tier improves | present reworded into (6) |
| Priors | Cold-start priors: everything cheap, memory-backed one-tier-down, go-package-implementation mid | present reworded into (7) |
| Rules | Two rules: subagent recalls first, decompose before dispatch | present reworded into (8) |
| Flags | Red flags (12 stop-points: don't pick on "looks hard", never escalate on first fail, etc.) | present reworded into (9) |

**Verdict: FAITHFUL.** All 9 major topics present in a 9-clause enumeration, nothing dropped or weakened.

**Most consequential delta:** the entire route skill — a ~7500B document with nested sections, markdown tables (the mini-report table, cold-start priors table, red flags table), and procedural narrative — is collapsed into a single 2800-char `object` field, then duplicated verbatim in the "Information learned:" body paragraph. Unlike A-F, which flattened a skill's 7 ordered steps + 6 rules into one run-on sentence, Route-F must enumerate 9 complex topics (each itself a multi-part procedure — e.g. "record every dispatch" contains 8 fields + 2 forms of aggregate update branching on a lookup result). The fact schema's constraint (scalar fields, no markdown structure within object text) forces every table, every bullet, every sub-section into inline prose within the parenthetical enumeration, with no line breaks or punctuation clarity. This reduces scannability compared to the source's structure or Route-R's preserved markdown sections.

**Retrieval-key check:** the `situation` field reads "routing a subagent dispatch and deciding which agent type, model tier, and effort level to use, based on work characteristics and prior evidence" — a close semantic match to a natural task prompt like "I have a large refactor for a subagent; should I route it to haiku or sonnet?" (overlapping: subagent, dispatch, model tier, effort). Stronger retrieval than A-F, weaker than B-F (which reused situation verbatim from source).

**Structural facts:** `type: fact`. `situation` (~135 chars) is author-composed descriptively (not a retrieval request). `subject` ("the route skill's dispatch and tier-selection procedure"), `predicate` ("requires, in order"), `object` (~2800 chars as 9-clause inline enumeration). Body duplicates situation+subject+predicate+object into "Information learned:" paragraph (~2900 chars), same duplication pattern as A-F and B-F. Total byte count 7291B vs source ~7500B (smaller than source because the enumerated prose is more compact than the source's prose + tables, but larger than Route-R 11885B would naively suggest because the fact schema's duplication is internal while Route-R's byte count reflects genuinely richer markdown structure).

---

## Cross-encoding observations (Tasks A + B + Route)

- **Schema-driven byte overhead:** All fact encodings duplicate content (frontmatter object + body paragraph), roughly doubling byte count relative to a runbook of the same source. Route-R 11885B vs Route-F 7291B shows the same pattern, though inverted ratio (runbook is larger here because of richer markdown structure).
- **Table collapse to enumeration:** Cold-start priors table, mini-report table, and red flags table all collapse to prose when encoding as fact (Route-F) or runbook (Route-R reduces table → bullets). Runbooks preserve more table structure when source is a skill; facts flatten everything.
- **No encoding dropped a core requirement or safety rule.** All 9 route topics present in Route-F; all red flags preserved in Route-R; the "subagent recalls first" rule is verbatim in both.
- **Retrieval-key quality:** Route-R's situation field ("routing a subagent dispatch...") is better than A-F (phrased as retrieval request) but weaker than B-F (situation reused verbatim). Route-F's situation is similarly descriptive and strong.
- **Procedural complexity scales the delta impact.** Route is more complex than commit or gitignore (3 major procedures vs 7-8 steps + rules). Collapsing procedural depth into fact enumeration is costlier here than in A-F.
