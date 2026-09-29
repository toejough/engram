# Parent-capture check plan: do subagent LESSONS reach the vault?

Pre-registered 2026-09-29, before any measurement, per ADR-0028 D-G. Approved by Joe 2026-09-29.
This file is committed on its own before `parent_capture.py` runs; any later change to a
definition below is a deviation and is reported as one.

## Why

`learn-rate-skill-only` v1 (shipped 2026-09-07) captures lessons through a parent-side path:
each dispatched subagent ends its completion report with a `LESSONS:` line (route's
completion-report contract, `agent-instructions/skills/route/SKILL.md` "The handoff is the
unlock"), the orchestrator collects the lines (please step 4), and the closing `/learn` judges
each against its Step-2 bar and crystallizes those that clear it
(`agent-instructions/skills/learn/SKILL.md`, "Collected LESSONS lines are an explicit scan
input"). The W2 re-measure (`REPORT-re-measure-2026-09-28.md`) sampled only subagent windows
and so cannot see this path (its caveat 3a). This check measures that path directly.

**Question:** when a subagent reports a LESSONS item worth capturing, does the parent session
get it into the vault?

## Population

Parent (main) sessions:

1. **Enumerate** with `run_audit.enumerate_corpus()` (its named project-dir rules: `-private-tmp*`
   and `-dev-eval-` dirs skipped), keeping only main transcripts
   (`<project>/<session-id>.jsonl`; nothing under `/subagents/`).
2. **Since** 2026-09-07 with `run_audit.filter_since()` (first timestamp's UTC date).
3. **Exclusions** with `run_audit.filter_exclusions()`: the named rules `excluded_session` (this
   orchestrating session f6dfe139 and its subagents), `headless_eval_arm`,
   `automated_security_review`, and the subagent-only `eval_subject_*` rules. Reused by import,
   never duplicated.
4. **Keep** sessions that received at least one subagent result carrying a non-`none` LESSONS
   block (definitions below).

## Unit and extraction (mechanical)

**Receipt**: one subagent result the parent received, timestamped by the record that delivered
it. Three delivery shapes count:

- a `<task-notification>` user message (string or text block) whose `<tool-use-id>` maps to an
  `Agent`/`Task`/`SendMessage` tool_use in the same transcript, or whose `<summary>` starts
  `Agent "`; the payload is the text inside `<result>…</result>` (no `<result>` → not a
  receipt);
- a `TaskOutput` tool_result; the payload is the text inside `<output>…</output>`;
- a synchronous `Agent`/`Task` tool_result that does not start with `Async agent launched`; the
  payload is the result text.

**LESSONS block**: the LAST line in the payload matching, at line start, optional list marker,
optional `**`/`__`, `LESSONS`, optional `**`/`__`, `:` (then optional `**`/`__`). The block is
the rest of that line plus every following line to the end of the payload. The contract says
the line ends the report, so earlier line-start matches are quotations and are ignored;
mid-line mentions (for example a backticked `` `LESSONS:` ``) never match.

**none**: a block whose text, with markdown emphasis and leading punctuation stripped, starts with
`none` (case-insensitive) yields no items.

**Items** (one lesson each):

- If the block has one or more list-marker lines (`-`, `*`, `•`, or `N.`/`N)` at line start),
  each marker line plus its following non-marker lines up to the next marker is one item. A
  non-marker preamble before the first marker is dropped when it ends with `:`, and is its own
  item otherwise.
- Otherwise the whole block is one item. Inline comma-separated lessons (design D-A's format)
  are **not** split, because lesson sentences contain commas. Where a single-line block bundles
  several lessons, the item count is an undercount; this is a known limitation.
- **Dedupe** per parent session on normalized text (lowercase, markdown emphasis removed,
  whitespace collapsed). Keep the earliest receipt.

Each item is attributed to the parent session that received it, with the receipt timestamp
`t_ret`, the dispatch description where recoverable, and the parent session's last timestamp
`t_end`.

## Vault evidence (mechanical, read-only)

Vault: `~/.local/share/engram/vault`. It is read only through `git log`, `git show`, and file
reads. No worktrees, no writes.

**Note set**: every note ever added.

- Notes in HEAD, with their HEAD content.
- Notes added then deleted or renamed away (`git log --all --no-renames --diff-filter=D`), with
  content from the parent of the deleting commit. They are deduped against HEAD by slug (the
  filename after `<luhmann-id>.<date>.`), and the HEAD version is kept.
- **Uncommitted-then-lost notes.** Vault commits are sparse; 2026-08-29 to 2026-09-26 has no
  commits. So a note written and then removed before the 2026-09-26 snapshot never enters git.
  These are recovered from transcripts: any `engram learn|amend|resituate` Bash call, in any
  transcript under `~/.claude/projects`, whose tool_result prints a real-vault note path that
  is absent from git. The note's content is the call's command text. These notes are flagged
  `source=transcript_only`.

**Write time `t_w`**:

- **Exact**, from the learn-call index. This is the earliest tool_result timestamp of an
  `engram learn|amend|resituate` Bash call, in any transcript, that printed the note's
  real-vault path.
- **Otherwise day-resolution**, from the note's `created:` date, or its filename date when
  `created` is missing. Dates are local (America/New_York). Git commit time is only an upper
  bound here, given the sparse commits, and is never used as the write time.

**Candidate notes for an item**:

- **Post-return (capture) candidates**:
  - with an exact `t_w`: `t_ret < t_w ≤ t_end + 24h`;
  - with a day-resolution `t_w`: `localdate(t_ret) ≤ created ≤ localdate(t_end) + 1 day`.
  
  A day-resolution note created on `localdate(t_ret)` might predate the return. It is kept as a
  capture candidate but flagged `timing=same_day_ambiguous`, and captures that rest only on such
  notes are counted and reported separately.
- **Pre-return (coverage) candidates**: notes with `t_w ≤ t_ret` (exact) or
  `created < localdate(t_ret)` (day). The whole vault is too large to judge per item, so a
  mechanical TF-IDF cosine over each note's slug, `situation`, and
  `action`/`object`/`predicate`/`subject` fields plus the first 600 body characters ranks these
  notes against the item text. The top 10 are candidates. This prefilter bounds how much
  coverage can be found, and it is reported as such.

**Write attribution** (secondary, for a matched post-return note): `parent_tree` if the learn
call that wrote it is in the parent transcript or one of its `subagents/` transcripts;
`elsewhere` if it is in another transcript; `unknown` if the write time is day-resolution only.

## Judge (LLM, sonnet), for two questions only

Calls go through `audit_moments._run_claude_p` (isolated `claude -p` config, the instrument's
retry and exhausted-retries signal), wrapped to log each call's model, prompt size, and
`total_cost_usd` to `cost-log.jsonl`, following `run_window_sample.install_recorders`. Every
raw response is persisted. Exhausted retries halt the run; an unparseable response is retried
once and then recorded as `judge_error` and excluded from the rates, with a count reported.

**(a) Worth capturing?** Items are batched, at most 10 per call, from one parent session. The
prompt, verbatim apart from the `{…}` slots:

> You are applying the engram `learn` skill's capture bar to lessons that subagents reported at
> the end of their completion reports. For each ITEM decide whether it is worth capturing as a
> vault note. An item is worth capturing only if ALL hold:
> 1. It maps to one of four kinds: (1) a correction of an approach or behavior; (2) an explicit
>    save-request; (3) a reversal — a presented conclusion, design, or verdict later overturned,
>    by anyone or by an instrument; (4) a confirmed approach — a specific approach validated by
>    an explicit specific confirmation or by an observable outcome that resolved a real
>    uncertainty (never bare success, never "it worked").
> 2. It is confirmed, not hypothesized: the item reports something that happened or was
>    verified, not a guess, suggestion, or open question.
> 3. It states a general, reusable principle that a future agent in a similar situation could
>    act on — not a session-specific narrative, status report, or task summary.
>
> An item that only reports that a lesson was already captured or recalled, without stating the
> lesson, is not worth capturing. Judge only the item text; do not reward length.
>
> ITEMS:
> {id}: {item text}
> …
>
> Reply with ONLY a JSON array, one object per item, in order:
> [{"id": "<id>", "worth_capturing": true|false, "kind": 1|2|3|4|null, "reason": "<one sentence>"}]

**(b) Does note N capture this item's principle?** Only items judged worth capturing go to (b).
They are batched, at most 5 per call, from one parent session and one `localdate(t_ret)`, so
they share a post-return candidate list. The call's candidate list is the post-return
candidates plus each item's top-10 coverage candidates. Each note is rendered as its ID, slug,
`situation`, `action` (or `subject`/`predicate`/`object`), and the first 300 body characters.
The prompt, verbatim apart from the `{…}` slots:

> Each ITEM below states a lesson a subagent reported. Each CANDIDATE is an engram vault note.
> For every item, list every candidate note that captures the item's principle: the same
> situation-to-action rule, so that a future agent recalling the note would act on the item's
> lesson. Shared topic or vocabulary alone is NOT a match. A note that captures the item's
> central principle matches even if it is broader or worded differently; a note that captures
> only a side detail of the item does not.
>
> ITEMS:
> {id}: {item text}
> …
>
> CANDIDATES:
> [{note_id}] {slug}
> situation: … | action: … | body: …
> …
>
> Reply with ONLY a JSON object mapping each item id to a list (possibly empty) of matches:
> {"<id>": [{"note_id": "<note_id>", "reason": "<one sentence>"}], …}

Everything else is mechanical.

## Metrics

All rates are per item, with a percentile bootstrap 95% interval from
`build_scorecard.bootstrap_ci` (n=2000, seed=739) over per-item flags.

Each worth-capturing item gets exactly one outcome, in this order of precedence:

1. `covered`: a matched pre-return candidate exists. The lesson was already in the vault when
   the parent received it.
2. `captured`: a matched post-return candidate exists.
3. `lost`: neither.

- **C1 (primary): capture rate.** `(captured + covered) / worth-capturing items`. Covered items
  count as captured-by-coverage, per the approval. The two components are always reported
  separately.
- **C1-new** (secondary): `captured / (worth-capturing − covered)`, the rate at which the
  parent wrote a new note for a lesson not already held.
- **C1-strict** (sensitivity): C1 with captures that rest only on `same_day_ambiguous` notes
  counted as lost.
- **C0: closing-learn rate.** The share of population sessions in which a `learn` invocation
  happens after the session's last non-`none` LESSONS receipt. An invocation is a `Skill`
  tool_use with `skill` equal to `learn`, or a user `/learn` command. The same share is also
  reported with invocations whose args say `mid-cycle` excluded.
- **Quality breakdown:** items worth capturing vs not, and worth-capturing items by kind.
- **Duplicates:** the `covered` count, reported separately, as above.
- **Attribution** of `captured` items: `parent_tree` / `elsewhere` / `unknown`.
- Everything is also reported per parent session, since the population is small and clustered.

## No pass bar

This plan sets no pass or fail threshold. What counts as good enough is Joe's decision at the
joint review. The result feeds `learn-rate-skill-only` task 3.4 (the escalation decision),
alongside the W2 re-measure.

## Cost

The estimate comes from `parent_capture.py --dry-run`, which makes no LLM calls. It counts
population sessions, items, and the batches (a) and (b) would send, and measures each batch's
prompt size in characters. Each call is priced with the per-call model fitted to the W2
re-measure's own `cost-log.jsonl` (272 sonnet calls): $0.0464 + $0.00205 per 1000 prompt
characters. The projection assumes every item is judged worth capturing (the upper bound for
(b)) and adds a 25% margin.

- **Budget gate:** if the projected total is over **$25**, stop after the dry run and return the
  counts and projection to Joe. Don't run the judge.
- **Actual cost** is the sum of `total_cost_usd` in `cost-log.jsonl`.

## Disclosure: scouting done before this plan

While designing the extraction, the population and receipt shapes were inspected mechanically.
No judge was run and no vault matching was attempted. That scouting found 9 eligible main
sessions, of which 2 appear to receive non-`none` LESSONS blocks (runbook-vs-skill phase-2
orchestration `23a08637`, 2026-09-09..15; phone-llm `01e43979`, 2026-09-24..26), with
roughly 110 line-start LESSONS lines between them. Two parent learn invocations, both labelled
mid-cycle, were seen in `23a08637`. The population is therefore small and dominated by one
session; interval widths will reflect that, and the report must say so.

## Outputs

- `dev/eval/audit/parent_capture.py` and `dev/eval/audit/test_parent_capture.py`.
- `dev/eval/audit/results-parent-capture-2026-09-29/`: `dryrun.json`, `items.jsonl`,
  `judge-a.jsonl`, `judge-b.jsonl` (raw responses), `cost-log.jsonl`, `scorecard.json`,
  `run-manifest.json`.
- `dev/eval/audit/REPORT-parent-capture-2026-09-29.md` and a `dev/eval/LEDGER.md` row.
