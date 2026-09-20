<!-- engram-owned: the follow-frame shim. NEW file (openspec runbook-shim-follow-frame, D0) — not a
     replacement for recall.md/delegate.md/learn.md, which stay as-is for sessions that still
     install the recall/learn/write-memory skills. In a shim-only session this file is the ONLY
     custom guidance text. Synced by 'engram update --with-guidance' to canonical paths (mirrors
     recall.md's sync convention). Edit via writing-skills-equivalent TDD (RED baseline, GREEN,
     pressure test) — never hand-edit past a RED/GREEN cycle. -->

## Before your first tool call, on every request: run `engram query`

Run this before anything else — before reading a file, before answering, before asking a
clarifying question, and before you have even decided *how* to handle the request. "How" includes
whether to delegate it: if dispatching a subagent is about to be your first tool call, the
abstraction below still has to happen first, in your own reasoning, not after the dispatch —
deciding to delegate is itself the situation to query on, never an exemption from querying. Do not
decide first whether the request is "multi-step" or "something you've done before": that judgment
is what the query answers, not a gate in front of it. A one-line request still gets a query; if
nothing matches, proceed without one.

```
engram query --lazy-chunks \
  --phrase "<the request, in your own words>" \
  --phrase "<the kind of situation this is — the same shape of wording a runbook's own 'situation'
             field would use: '<verb>-ing <object> in <context>', not a casual paraphrase>"
```

Two phrases minimum, both shapes above. The second phrase names your own process, not the ticket:
strip out the concrete deliverable and name the kind of work-handling decision in front of you.
For example, given "add a `--version` flag to the CLI; delegate it to a subagent" — "adding a
`--version` flag to the CLI" is task-surface wording and belongs in the FIRST phrase only; reused
as the second phrase, it is the wrong shape and reliably fails to surface a process runbook (e.g.
one about dispatch/tier selection). The situation-shaped second phrase drops the flag, the CLI,
and every other ticket-specific detail: "routing a subagent dispatch and deciding its tier for a
scoped unit of implementation work." Casual task-phrasing alone does not reliably surface a
matching runbook; situation-shaped wording does. Add more phrases if the request has distinct
facets worth querying separately.

## What each returned item is for

The query is not filtered by kind — read every item's `kind` field and treat it accordingly:

- **`fact`** — knowledge and context. Take it as true for this task unless the repository in
  front of you contradicts it.
- **`feedback`** — a correction the user already gave you. Treat it as a standing instruction: do
  not repeat the mistake it names, and do not re-derive whether to honor it.
- **`runbook`** — a procedure to execute. See the frame below.
- **`chunk`** — raw evidence, not an instruction. Fetch it (`engram show-chunk <source#anchor>`)
  only when the notes above leave a gap; never treat a chunk's presence alone as something to act
  on.

## When a `runbook` matches your task: the follow frame

A runbook whose `situation` matches the task at hand — not merely a related one — gets this
treatment, every time, no exceptions:

1. **Announce it by name**, in a text block, before your first mutating action. State which
   runbook you are executing.
2. **Restate its steps as your plan** before that same first mutating action. Where your harness
   offers a todo list, make one todo per step. Do the steps in the runbook's order.
3. **Read `red_flags`** (if the runbook carries any) before you start. If a listed condition
   occurs while you work, stop — in your next text block, name the red flag and the step you are
   rereading — before your next mutating command.
4. **`done_when` is the completion bar.** Do not report the task done until you have verified each
   `done_when` condition holds. Verify it; do not assume it.
5. **Stop and ask when a step's outcome is genuinely uncertain — not for every rough edge in its
   wording, and not because an earlier step is already done.** If a step's specific instruction
   and its own incidental example seem to clash, follow the specific instruction; resolving that
   by rereading is not, by itself, a reason to stop. Completing one step — even the one that
   produces the task's most visible deliverable — is not, by itself, evidence that a later step
   is ambiguous; when your own restated plan already names that later step as required, proceed
   to it without asking whether to. Do stop when a step names a target that plainly isn't present
   in front of you (no GitHub remote, no ticket queue, no CI) — inventing "the local equivalent"
   yourself is still a deviation, not a resolution of the step — or when two things the runbook
   requires are genuinely in tension and you cannot tell which one the person needs preserved.
   Name the step and the specific gap, then end your turn. This is a clarity signal for the
   person who gave you the
   task — they own supplying what's missing, now and for next time — never a failure, and never a
   reason to substitute your own approach and continue. Weigh this most heavily before an
   irreversible action: if the honest answer is "I'm guessing," and what you're about to do can't
   be undone, stop and ask instead of proceeding on your own judgment.
6. **A wikilinked runbook (`[[basename]]`) is fetched and followed the same way.** Run
   `engram show <basename>`, then apply this whole frame to it — announce, restate, red_flags,
   done_when, stop-and-ask — before the step that linked it.
7. **If the query payload truncates a matched runbook**, run `engram show <basename>` for the full
   body before you restate its steps. Never restate from a truncated fragment.

## The floor beneath every runbook

These rules apply uniformly, to every runbook, and are not repeated in any runbook's own body:

- **The urge to shortcut a step is the cue to reread it — never a reason to skip it.**
- **Substituting a related action for the one the step names is a skip**, not a completion of that
  step — including "the named thing doesn't apply here, so I'll do what accomplishes the same
  goal." That is still a substitution. Ask; don't invent one.
- **The letter of a step is its spirit.** If the step says to run a command, run that command —
  not one that seems to accomplish the same thing.

## Re-entry: four moments to query again, mid-task

Firing the first query does not cover the rest of the task. Run `engram query` again — same two
phrase shapes as above, re-keyed to what's actually in front of you now — at each of these
moments:

- **Before you endorse or rank a proposed approach** — even just discussing it. A past decision
  may have already tried and killed it.
- **Before you declare work done.** Query first, then verify — the query names gotchas a
  self-check alone would ship past.
- **After a failure you can't immediately explain** — once, before you start guessing. A past
  lesson may already name the cause.
- **Before you start building a new approach** — while the path is still cheap to change.

Escalate any of these to the runbook's `deep` procedure (if the matched runbook offers one) when
the decision is weighty or irreversible, or when this pass surfaces a gap it can't resolve.
