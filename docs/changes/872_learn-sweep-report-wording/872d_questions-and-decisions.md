# Learn Sweep Report Wording - Questions and Decisions

Pointers: `#<n>` = issue n of github.com/toejough/engram.

## Questions and decisions

| ID and topic | Status | Context, options, recommendation | Decision and rationale | Decided by / date | Blocks |
| --- | --- | --- | --- | --- | --- |
| 872d1 The sweep report is worded in one scenario only | Decided | #871 row 861d382 reworded Step 1 of the [learn skill](../../GLOSSARY.md#learn-skill) to "Report the per-source lines it prints, or "memory index up to date" when it prints nothing", and [792a15b1a The sweep runs and is reported](../../specs/product/learn.md#792a15b1-uat-scenarios) owns that sentence and its test. [792a15a1a A session with no lesson writes nothing](../../specs/product/learn.md#792a15a1-uat-scenarios) still says the agent "reports its tally". Options: (a) copy the per-source wording into the Then; (b) say the agent reports what the [sweep](../../GLOSSARY.md#sweep) prints; (c) drop the report from the Then. | (b). The sentence keeps the one home it has, so (a) is refused: a second wording would drift from it. (c) is refused: [792a15a Learn writes only explicit lessons](../../specs/product/learn.md#792a15a-learn-writes-only-explicit-lessons) requires a [session](../../GLOSSARY.md#session) with no lesson to end with the sweep and its report. The test of the scenario asserts no wording of the report, so it keeps its assertions. | author / 2026-10-22 | - |
