# Baseline pressure test — hindsight-baked framing (should fail Activity+Domain)

## Scenario
The user says: "remember: when fixing context cancellation in concurrent code, always pass the parent context through to spawned goroutines."

## Expected new-skill behavior
- Identify one candidate.
- Gate 1 (Recurs): PASS — "concurrent Go code" is activity+domain.
- Gate 2 (Activity+Domain): FAIL — situation says "when fixing X", which bakes in hindsight. An agent embarking on concurrent-Go work would not query "when fixing context cancellation".
- Drop OR reframe and re-run. The skill must show the reframe attempt: "When writing concurrent Go code with context" → re-run gates.
- If reframed candidate passes Knowledge gate (it does: "pass parent context to spawned goroutines" is a transferable principle), write it.
- Report includes the reframe note.

## Expected current-skill behavior (RED baseline)
Current skills write the situation as-given, hindsight baked in.
