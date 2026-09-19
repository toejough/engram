## Why

The shim's stop-and-ask rule (`agent-instructions/guidance/shim.md`, follow-frame rule 5, backed by
`guidance-runbook-follow-frame`'s "agent SHALL stop and ask when a step is unclear or would be
deviated from" requirement) is unscoped: it reads as blessing *any* question-stop, not only ones
caused by genuine ambiguity. Concrete evidence from this session's `route-skill-to-runbook` D8
rerun (trial `route-R-0`, issue #757): the agent verified the runbook's primary/most-visible
deliverable (a CLI flag implementation) and then asked permission before the runbook's remaining
mandatory evidence-write steps, despite its own words confirming it already knew those steps were
owed — "Per the runbook I still owe the evidence record... Should I go ahead and write those now,
or would you rather I stop here since the code task itself is verified and complete?" This is not
the clarity-signal case the rule was written for; it is an unnecessary halt on unambiguous,
already-mandatory work. Filed and decided in issue #762
(https://github.com/toejough/engram/issues/762) and vault note
`1037.2026-09-18.question-stop-default-scoped-to-genuine-ambiguity` (narrows `1030`).

## What Changes

- Narrow the shim's stop-and-ask default (rule 5 of the follow frame) so it applies only under
  genuine next-step ambiguity, and explicitly names the failure mode it does NOT license: treating
  a verified "primary" step as an implicit checkpoint to ask permission before continuing with
  other steps the runbook already marks mandatory — including when the agent's own stated
  reasoning already confirms the obligation.
- Update `guidance-runbook-follow-frame`'s "stop and ask" requirement text and add one new scenario
  covering this case (an unambiguous remaining mandatory step must not stop, even when a prior step
  looks like task completion).
- **Not BREAKING**: this narrows an existing SHALL, it does not add a new capability or remove an
  existing one; every runbook already following the frame keeps working, and the change only
  removes one over-broad license to stop.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `guidance-runbook-follow-frame`: the "agent SHALL stop and ask when a step is unclear or would be
  deviated from" requirement is narrowed to genuine ambiguity, with new normative text explicitly
  excluding "a prior step is verified" as a reason to stop before an unambiguous remaining
  mandatory step.

## Impact

- `agent-instructions/guidance/shim.md` — rule 5 of "the follow frame" section, reworded.
- `openspec/specs/guidance-runbook-follow-frame/spec.md` — the stop-and-ask requirement's
  normative text and scenario set, via this change's delta spec.
- Deployed copies at `~/.claude/engram/guidance/shim.md` and `~/.pi/agent/guidance/shim.md` (synced
  by `engram update --with-guidance`) once implemented.
- No code changes — this is a guidance-text-only change. No existing eval harness in
  `dev/eval/cumulative/runbook_vs_skill/` exercises this exact shape (a verified primary step with
  mandatory steps still pending) generically across runbooks; see design.md's Risks section.
