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

| # | Source requirement | Status |
|---|---|---|
| Body | skill body (lines 9-286: heading, 6 sections, tables, rules, red flags) | **verbatim byte-for-byte** |
| Frontmatter | skill frontmatter (name, description, context, model, user-invocable) | dropped; replaced with runbook frontmatter (type, tier, situation, done_when, created, source, repo, user, vault) |
| — | `done_when` field | **added-not-in-source**, schema-forced; synthesized from content describing dispatch resolution → evidence write completion |

**Verdict: FAITHFUL — zero-delta body, frontmatter only differs.** Body is identical to source skill body (lines 9-286 of SKILL.md). Diff verification: `diff <(tail -n +9 SKILL.md) <(tail -n +14 Route-R note)` yields empty output (0 lines).

**Delta (frontmatter only):** Skill frontmatter (name, description, context, model, user-invocable) is dropped and replaced with runbook schema frontmatter. No change to the procedures, tables, red flags, or content.

**Structural facts:** `type: runbook`, situation ~150 chars, `done_when` ~400 chars (synthesized addition). Byte count 20,581B vs source skill 19,955B (overhead = runbook frontmatter ~600B + done_when field + newlines; body is identical).

---

## Route-F (fact from route SKILL.md)

| # | Source requirement | Status |
|---|---|---|
| Body | skill body (lines 9-286: heading, 6 sections, tables, rules, red flags) | **verbatim byte-for-byte** in body section after "Information learned:" preamble |
| Object field | one-to-two-sentence summary of the procedure | author-composed summary (not source verbatim) |
| Frontmatter | skill frontmatter (name, description, context, model, user-invocable) | dropped; replaced with fact schema frontmatter (type, tier, situation, subject, predicate, object) |

**Verdict: FAITHFUL — zero-delta body, frontmatter and object field only differ.** Body (in "Information learned:" section) is identical to source skill body (lines 9-286 of SKILL.md), carried verbatim after a 2-line preamble. Diff verification: `diff <(tail -n +9 SKILL.md) <(tail -n +3 <(tail -n +16 Route-F note))` yields empty output (0 lines). The `object` field contains a one-sentence summary pointing to the full procedure in the body, not an enumeration of the procedure itself.

**Deltas (frontmatter and summary field only):** Skill frontmatter is dropped and replaced with fact schema frontmatter. The `object` field contains a brief summary ("a procedure for orchestrators to route subagent work...") rather than enumerating the full procedure. No change to the procedures, tables, red flags, or body content.

**Structural facts:** `type: fact`, situation ~135 chars, subject/predicate/object are fact-schema-shaped (summary in object). Byte count 20,714B vs source skill 19,955B (overhead = fact frontmatter + "Information learned:" preamble ~2 lines; body is identical).

---

## Cross-encoding observations (Tasks A + B + Route)

- **Body fidelity ladder:** A-R (simplified rewrite), A-F (flattened summary), B-F (verbatim body), now Route-R and Route-F (both verbatim bodies). Route achieves the highest fidelity by carrying source skill body byte-for-byte into both runbook and fact note, with only schema-required frontmatter fields differing.
- **Schema-driven frontmatter overhead:** Route-R adds ~600B (runbook frontmatter + synthesized done_when); Route-F adds ~760B (fact frontmatter + "Information learned:" preamble + summary object field). Both preserve the ~19,955B source body exactly.
- **No encoding dropped a core requirement or safety rule.** All procedures intact in both Route-R (as markdown sections) and Route-F (as verbatim body); all red flags present; the "subagent recalls first" and "decompose before dispatch" rules are identical to source.
- **Retrieval-key quality:** Route-R's situation field ("routing a subagent dispatch...") matches A-F pattern (descriptive, not retrieval-phrased). Route-F's situation similarly strong. Object field in Route-F is a summary, not full procedure.
- **High-fidelity minimum for procedural content:** Achieving zero-delta body required rejecting intermediate simplifications (tables-as-prose, flattened enumerations). The verbatim-body standard applies the "as little change as possible" rule correctly.
