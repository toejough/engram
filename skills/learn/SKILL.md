---
name: learn
description: >
  Use after completing any action we started with a recall call, or after any work that involved more than one tool
  call or more than quick shallow thinking. Also use immediately on explicit save-requests like "remember this",
  "remember that X", "save that for later", "note for next time", "don't forget X", "write this down". This preserves
  relevant memories that are VITAL to recall for a good user experience and a greater chance at first-pass success
  for similar work in the future.
---

# Learn — write to the agent-memory vault

Preserve lessons from completed work as **permanent notes** (and **MOCs** when a real framing paragraph emerges across notes). One stage — no fleeting tier, no escape hatch.

This vault is your (the LLM's) persistent memory. You write everything; the human curates by directing what gets worked on. **Don't draft and ask for review** — you decide what becomes permanent and write it.

## The core principle: write what recall would have wanted to find

Recall and learn are paired. Recall reads the vault by phrasing queries from your stated plan and situational features. Learn writes to the vault in the same shape — so the next agent in a similar situation, querying the same way, will surface what you learned.

Concretely, three paths — chosen per candidate, after classifying the candidate's **injection locus** (the work that *caused* the lesson, not the work that *surfaced* it).

**Path A — current-locus candidate; a recall bracketed its segment.** The mistake or discovery originated in *this session*, and a recall ran during that segment. Its Step 0 (Ask / Situation / Plan) and Step 1 (5–15 short queryable phrases) are in your context. **Every note you write must be framed so a future recall using one of those phrases (or a close variant) would surface it.** Lift `--situation` strings directly from Step 1 phrasings where possible; for net-new lessons, write the `--situation` so it would appear in a parallel phrase added to Step 1.

**Path B — current-locus candidate; no recall bracketed its segment.** The lesson originated in *this session*, but no recall ran during that segment. Mentally reconstruct what Step 0 and Step 1 would have been, before you started the work. What would I have searched for? What ambient features would I have queried under? Write the list down internally — even 3–5 phrases is enough — then frame writes against it.

**Path C — retro-locus candidate.** The lesson's *injection locus* is a prior session, even though the candidate surfaced during current-session work (or came from `engram transcript --mark` history of an earlier session). Reconstruct the scratch list from what the **injecting agent** was doing — the activity and domain in flight when the mistake was made — not from any recall that bracketed this session's discovery. Sources for the reconstruction: `git blame` / `git log` on the offending line (commit message + surrounding work), prior-session transcript content (if `engram transcript --mark` produced it), or behavioral inference for purely conceptual mistakes. **Path C overrides Path A** — when the candidate is retro-locus, ignore the in-session recall framing even if a recall bracketed the discovery; the discovery context is not the retrieval target.

The recall-mirror test, applied to every candidate note:

> Phrase the `--situation` out loud. Would a future agent, querying for the same kind of work **this candidate's scratch list targets** (current-locus → this session's work; retro-locus → the injecting agent's work), see this note in their cascade? If no, rephrase. If still no, the lesson is real but the framing is wrong — fix the framing, not the lesson.

## Vault paths

The binary resolves the vault automatically — `--vault` and
`ENGRAM_VAULT_PATH` are overrides, not requirements. Default:
`$XDG_DATA_HOME/engram/vault` (typically `~/.local/share/engram/vault`).
On first `engram learn` against a non-existent vault, the directory is
bootstrapped with `Permanent/`, `MOCs/`, `MEMORY.md`, a minimal
`.obsidian/` config, a `.gitignore`, and a `README.md`. **Do not pass
`--vault` in `engram learn` / `engram recall` invocations unless the
user explicitly tells you the vault is elsewhere.**

Layout:

- Permanents: `<vault>/Permanent/`
- MOCs: `<vault>/MOCs/`

No `Fleeting/` directory. No `Main Index.md`. No log file. Chronology lives in filenames; navigation lives in MOCs and link context.

## Trigger

Default to firing after any action that meets the description threshold: post-recall work, or any work involving more than one tool call or more than quick shallow thinking. Also fires on explicit cues — `/learn`, "remember this", "save that for later", "write up what we just did".

**Do not fire on micro-tasks** (one-line edits, single-file moves, trivial renames, typo fixes, pure lookups) where no lesson could plausibly exist. When unsure between firing and not firing, fire.

## What to write

Two kinds of notes, distinguished by why a future agent would want them:

- **Feedback** — anything you'd do differently next time. Mistakes, user corrections, reasonable actions that didn't pan out, dead-ends, surprising costs. The note exists so future-you avoids the same loss.
- **Fact** — anything else that would help reach the right outcome more efficiently (time- or cost-wise) in similar situations. Tool behaviors, idioms, conventions, integration shapes, gotchas, the way a thing actually works. The note exists so future-you spends less to get to the same right answer.

If a single observation has both a "should have done X" component and a "here's how Y works" component, write two notes — one Feedback, one Fact.

**MOCs** emerge when a real framing paragraph forms across notes — a synthesis you can write in your own words, not a list of constituents. Judgement-based; no count threshold.

## Workflow

### 1. Identify candidates

Always run `engram transcript --mark` to fetch transcripts since the last `/learn` for this project. The command scans forward chronologically from the marker, stops when it would exceed the byte cap (~200KB by default), and advances the marker to the effective scan end (`now` if everything fit, otherwise the Mtime of the last fully-included session). Its trailing status line — `[engram transcript: scanned [<from>, <to>]; marker advanced to <to>]` — tells you the new marker position; **capture it and include it verbatim in your final report (§9).**

**First-run handling.** If `engram transcript --mark` exits non-zero with `transcript: no progress marker (...) ... earliest session: <date>`, this is the project's first scan (no marker yet). **Stop and prompt the user via `AskUserQuestion`** — do not pick a date yourself. The error message names each source's earliest session date; offer at least two options: "Scan from the beginning (`--from all`)" and "Scan from <earliest>". After the user answers, re-run `engram transcript --mark --from <chosen>`. Capture that re-run's status line for the final report.

**Byte-cap continuation.** If the output includes a `[engram transcript: byte cap hit; ... onward not yet scanned; run again to continue]` line, more sessions remain unread. Include the continuation line verbatim in your final report, and mention in the report that `/learn` should be re-run (after `/clear` if context is tight) to catch up. Do not loop in this pass — one transcript scan per `/learn` invocation.

If the in-context conversation also covers relevant turns from this session, scan it too — but the transcript fetch is non-optional and runs every `/learn` pass so the marker keeps moving forward.

Look for, in either source:

- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about tools, idioms, conventions, gotchas
- **Recurring patterns** — behaviors that should be codified

For each candidate, also note its **injection locus** — the work that *caused* the lesson, not the work that *surfaced* it. A current-session discovery can have a prior-session origin: a bug found today during a docs cleanup may have been wired wrong six commits ago by a different agent. Cheap signals — apply in order:

- **Concrete file/decision locus** — `git blame` / `git log` on the offending line or config decision. Authorship before this session's first commit → retro-locus.
- **Session-log locus** — prior-session transcript content present in `engram transcript --mark` output that names the moment the mistake was made → retro-locus.
- **Behavioral inference** — for purely conceptual mistakes (a misconception carried into this session, a correction the agent had absorbed wrong), ask: *would I have done the wrong thing in this session independently, or did I inherit it?* If inherited → retro-locus. Otherwise → current-locus.

Carry the locus tag forward into §2 — it determines which path applies.

### 2. Anchor on the recall framing — per candidate

For each candidate, lay out the framing **that candidate's** write will be measured against. The choice of framing is per-candidate (or per-segment), not session-global — different candidates can come from different segments of work and need different scratch lists.

**Selection is two-step: classify locus first (from §1), then pick the path.**

For retro-locus candidates → **Path C** (regardless of whether a recall bracketed the discovery this session). For current-locus candidates → **Path A or Path B** depending on whether a recall bracketed the candidate's segment.

- **Path A — current-locus, recall bracketed the candidate's segment.** Scroll back to the recall whose Step 0 (Ask / Situation / Plan) bracketed the work that produced this candidate, and copy its Step 1 phrases verbatim into the candidate's scratch list. If multiple recalls ran in-session, pick the one that bracketed *this* candidate's segment — not necessarily the most recent. **Do not apply Path A to a retro-locus candidate even if a current-session recall bracketed the discovery** — the discovery context is not the retrieval target; the injection context is.
- **Path B — current-locus, no recall bracketed the candidate's segment.** The lesson originated in this session but no recall ran during that segment. Write down 3–5 phrases capturing what an agent doing this kind of work would have queried *at the time*: plan-grounded phrases (the actions then in flight) and situational phrases (the ambient features). Same shape recall uses.
- **Path C — retro-locus, injecting agent's situation reconstructed.** The lesson's cause is in a prior session, even if the discovery happened this session. Reconstruct the scratch list from the **injecting agent's** situation, not the surfacing agent's. Sources (use what's available):
  - `git blame` / `git log` on the offending line — read the commit message and surrounding work to infer the activity in flight: *what was that agent trying to accomplish?*
  - Prior-session transcript content from `engram transcript --mark` output — the ambient plan and situational features from the actual session.
  - Behavioral inference — for conceptual mistakes with no file locus, ask: *under what kind of work would I have first formed this misconception?* That activity + domain is the retrieval target.
  - Write 3–5 phrases capturing what an agent doing **the injecting kind of work** would have queried — plan-grounded and situational, same shape recall uses.

Each candidate's scratch list is the retrieval target for its write. Every `--situation` will be tested against its own candidate's scratch list — not against a shared session-global list, and not against a list built from the surfacing session's framing when the candidate is retro-locus.

### 3. Apply the recall-mirror test per candidate

For each candidate, phrase the `--situation` and ask:

> Would a future agent querying any of the phrases in **this candidate's** scratch list (or close paraphrases) surface this note?

Three outcomes:

- **Yes** → write it. Use the closest matching phrase from this candidate's scratch list as the `--situation`, lightly normalized to "When …" form. If multiple phrases match, pick the one most specific to the lesson.
- **Not yet, but the lesson is real** → rephrase the `--situation`. Lessons are durable; framings are revisable. You may rephrase as many times as needed; each rephrase re-tests against this candidate's scratch list.
- **No, even after rephrasing** → drop. Either the lesson is too project-specific, too event-bound, or not a transferable principle. Report the drop with a one-line reason in §7.

Common ways a candidate fails the test (and what to do):

| Failure mode                                                                 | Fix                                                                                              |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Situation names this project, this file, this issue, or today's date         | Rephrase to the activity + domain. If the project name is load-bearing for the lesson, the lesson is project knowledge, not vault knowledge — drop. |
| Situation bakes in hindsight ("When fixing X", "When debugging Y")           | Rephrase as the activity you'd be embarking on **before** the lesson is known.                   |
| Situation describes one event, not a recurring kind of work                  | Generalize to the kind of work; if you can't, drop.                                              |
| Situation phrasing wouldn't appear in any plausible recall                   | Look at the candidate's scratch list; pick the closest phrase and rebuild the situation around it. |
| Measured against the wrong segment's scratch list (e.g., topic-B framing applied to a topic-A candidate, or session-global anchor applied to a `transcript --mark` candidate) | Re-select the scratch list per §2 for *this* candidate — Path A from the recall that bracketed its segment, or Path B reconstructed for its segment if no such recall ran. |
| Scratch list anchored on the **discovery** situation rather than the **injection** situation (e.g., docs-cleanup phrases used for a CLI-wiring lesson surfaced during the cleanup) | The candidate is retro-locus. Re-select via Path C per §2 — reconstruct the injecting agent's situation from git blame / prior-session transcript / behavioral inference, not from the current-session recall. |

Note: this replaces the older Recurs / Activity-and-Domain / Knowledge gate machinery. The same disciplines (no project names, no hindsight, must be a principle) are still enforced — but as outcomes of the recall-mirror test rather than as standalone gates.

### 4. Categorize: Feedback or Fact

For each surviving candidate:

- "I would do this differently next time" / "we got it wrong and corrected" / "the user told us to change course" → **Feedback**.
- "Here's how X actually works / behaves / is shaped" / "this saves time when …" → **Fact**.

Both kinds use the same retrieval framing. The split tells future-you what kind of help to expect.

**Locus check (sanity gate before writing).** For Feedback especially, ask: *who made the mistake — me this session, or someone earlier?* If earlier, the candidate should already be tagged retro-locus from §1 and routed through Path C in §2. If the categorization felt like Feedback but the locus was retro and you didn't pick Path C, back up to §2 and re-select — the framing will retrieve under the wrong activity otherwise. Facts that describe how a thing actually behaves are usually current-locus (the discovery is the lesson), but a Fact whose *content* names a prior-session decision or wiring may still be retro-locus — the framing should still target the kind of work the injecting agent was doing.

### 5. Decide disposition and Luhmann position

- **New permanent** — one candidate → one new top-level permanent (`--position top`).
- **Continuation** — write a new permanent as a continuation under an existing one (`--target <id> --position continuation`, e.g. existing `1` → new `1a`). Covers both sharpening the parent's wording with another instance and adding claims that elaborate it.
- **Sibling** — parallel branch at the same level (`--position sibling`, e.g. `1a` → `1b`).
- **Split** — one candidate bundles multiple principles → multiple permanents.

The binary computes the actual ID under a vault lock. **You do not compute the ID yourself.**

`--position` controls Luhmann placement. **`--relation` is a separate, repeatable flag** that supplies the `Related to:` bullets — see step 6.

### 6. Draft body in LLM voice

**Every `engram learn` invocation MUST include `--source`.** It is a required flag; the binary errors out when it is missing. Forms:

- For feedback/fact derived from session activity: `session log <project>, <YYYY-MM-DD HH:MM UTC>, context: <short description>`
- For MOCs synthesized from cluster analysis: `constructed from cluster analysis, <YYYY-MM-DD>`
- For end-to-end smoke or test runs: a short label naming the run

**All body content is supplied via flags. Stdin is not read.**

- `--relation <wikilink-target>|<rationale>` — repeatable; each instance adds one `Related to:` bullet. The pipe `|` separates the wikilink target from its per-link rationale. Example: `--relation "1a.foo|same shape, different domain"`.
- `--framing "..."` — MOC only; the framing paragraph(s) that form the MOC body. Do NOT auto-list constituents; backlinks already do that.

The `Lesson learned: ...` / `Information learned: ...` opener line is auto-generated from `--situation` and `--action`/`--subject`/`--predicate`/`--object`. **Do not duplicate it in any flag.**

**Feedback:**

```
engram learn feedback \
  --slug <kebab-case-tag> \
  --target <luhmann-id-of-related-note-or-empty> \
  --position <top|continuation|sibling> \
  --source "session log <project>, <YYYY-MM-DD HH:MM UTC>, context: ..." \
  --situation "..." --behavior "..." --impact "..." --action "..." \
  --relation "<wikilink>|<rationale>" \
  --relation "<wikilink>|<rationale>"
```

**Fact:**

```
engram learn fact \
  --slug <kebab-case-tag> \
  --target <id-or-empty> \
  --position <top|continuation|sibling> \
  --source "..." \
  --situation "..." --subject "..." --predicate "..." --object "..." \
  --relation "<wikilink>|<rationale>"
```

**MOC** (judgement-based, no count threshold):

```
engram learn moc \
  --slug <kebab-case-tag> \
  --target <id-or-empty> \
  --position <top|continuation|sibling> \
  --source "constructed from cluster analysis, <YYYY-MM-DD>" \
  --topic "<theme name>" \
  --framing "<framing paragraph(s)>" \
  --relation "<wikilink>|<rationale>"
```

### 7. Contradictions

If a new permanent contradicts an existing one, write the new permanent with a `Related to:` bullet whose rationale names the discrepancy. Surface in the final report. Don't smooth.

### 8. Write — one parallel tool-use block

**Hard rule: all `engram learn` invocations for a single /learn pass go in a single parallel tool-use block.** Serial writes cost a tool roundtrip each (~15–20s); batching collapses that.

### 9. Report

The final user-facing report is **only** these things:

- The `engram transcript --mark` status line(s) verbatim.
- Any `[engram transcript: byte cap hit; ...]` continuation lines verbatim, plus a one-sentence note that `/learn` should be re-run to catch up.
- The permanents and MOCs written, each as one line: `Permanent/<id>` or `MOCs/<id>` + slug.

Nothing else. Do not include the Path A/B disclosure, the scratch list, the candidates-considered table, the dropped-with-reasons list, a recap of `--situation` strings, or a separate "Contradictions surfaced" section. Those are scaffolding for the writer, not output for the reader.

If a permanent you just wrote contradicts an existing note, mention it **inline** with that permanent's one-line entry (e.g. `Permanent/87 — one-canonical-handle-per-node (contradicts Permanent/4e on …)`). No separate section.

**Red flag:** if your report contains any of the words "Path", "scratch list", "candidates", "dropped", or a `| # | … |` table, you are leaking workflow scaffolding into the report. Rewrite to the two-line form above.

## Quality bars

- **Atomicity** — one idea per permanent.
- **Autonomy** — permanents are understandable without context. Strip "this case", "the incident", "we did X" framing.
- **Retrieval-shaped** — every `--situation` is phrased so a future recall using a Step 1 phrase (or the equivalent reconstructed phrase) would surface it.
- **LLM voice** — translate raw material into your own synthesis. Verbatim user quotes get rephrased on writing.
- **Per-link rationale** — every `Related to:` bullet explains why the connection exists. No bare wikilinks.
- **Heterarchy** — a permanent can belong to multiple MOCs; one `Related to:` bullet per MOC with its own rationale.
- **Surface contradictions** — link them with rationale naming the discrepancy.

## Red flags — STOP and rephrase

| Sign you're off the principle                                                       | What you should be doing                                                                                          |
| ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| You started writing without locating each candidate's recall (Path A) or reconstructing it (Path B) | Stop. Write a per-candidate scratch list first. Without it, you're guessing at retrieval framings — and the guess is almost always the most-recent-recall framing, which mis-judges earlier-segment and prior-session candidates. |
| You produced a single session-global scratch list and applied it to every candidate     | Per §2, scratch lists are per-candidate. Re-select each one for the segment that produced its candidate.           |
| You picked Path A for a candidate whose mistake originated in a prior session, because a current-session recall happened to bracket the discovery | Classify locus first per §1. A retro-locus candidate takes Path C regardless of what bracketed the discovery — frame against the injecting agent's situation, not the surfacing agent's. |
| Your `--situation` names this project, this commit, today's date                    | Project-specific knowledge doesn't belong in the vault. Either generalize or drop.                                 |
| Your `--situation` reads like a diagnosis ("When fixing the X bug")                 | Pre-lesson framing only. Rewrite to the activity an agent would be starting, before the lesson exists.            |
| You're categorizing a "here's how X works" note as Feedback                         | That's a Fact. Feedback is for "do differently next time" only.                                                   |
| You're categorizing a user correction or dead-end as Fact                           | That's Feedback. Facts describe how things are; corrections describe how to act differently.                      |
| You can't say which Step 1 phrase (or scratch-list phrase) the note retrieves under | The framing is wrong. Lift the closest phrase and rebuild the situation around it.                                |
| You're invoking "Recurs / Activity-and-Domain / Knowledge" by name                  | Those gates have been replaced by the recall-mirror test. Apply that test instead.                                |

## Common mistakes

| Mistake                                                                                                | Fix                                                                                                                                                       |
| ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Writing a note whose situation names "engram", "Task 8", "promote.go"                                  | Project-specific knowledge — drop or generalize.                                                                                                          |
| Hindsight-baked situation ("When fixing the bug in X")                                                 | Rewrite to pre-lesson query phrasing.                                                                                                                     |
| Writing "we observed X" without stating it as a principle                                              | Restate as principle or drop.                                                                                                                             |
| Drafting and asking for human voice rewrite                                                            | You're the writer. Just write.                                                                                                                            |
| Writing files directly with the filesystem                                                             | Use `engram learn {feedback|fact|moc}` — handles ID assignment under lock.                                                                                |
| Computing the Luhmann ID yourself                                                                      | Pass `--target` and `--position`; binary computes the ID.                                                                                                 |
| Putting a `Lesson learned:`/`Information learned:` opener inside `--framing` or any flag               | The opener is auto-generated; never repeat it. Body bullets go in `--relation`, framing in `--framing`.                                                   |
| Piping body content via stdin                                                                          | Stdin is ignored. All body content goes through `--relation` and `--framing` flags.                                                                       |
| Auto-listing MOC constituents in body                                                                  | Backlinks already do this — MOC body is framing prose only.                                                                                               |
| Bare wikilinks without rationale                                                                       | Every `Related to:` bullet must include per-link rationale.                                                                                               |
| Serial `engram learn` calls across tool turns                                                          | One message, N parallel tool calls.                                                                                                                       |
| Auto-firing on a one-line micro-task                                                                   | Only autonomous-trigger on chunks that plausibly produce lessons; when unsure, don't fire.                                                                |
| Creating a MOC because the cluster crossed a count threshold                                           | Judgement, not count — a real framing paragraph must emerge.                                                                                              |
| Putting an H1 title or `Luhmann-ID · date` line in the body                                            | Filename is the display name; `luhmann` and `created` live in frontmatter.                                                                                |
| Smoothing over contradictions                                                                          | Write `Related to:` bullets that name the discrepancy.                                                                                                    |
| Categorizing every survivor as Feedback because the old gates didn't distinguish                       | Feedback = do-differently; Fact = how-it-works. Methodological principles with no mistake or correction are usually Facts.                                |
