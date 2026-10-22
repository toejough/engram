# dev/eval

Maintainer tooling that measures engram: repository tooling, not product, not part of the shipped command line.

- `targ eval <arm> [trials]` runs the Go harness here (arms `nothing`, `skills-only`, `current-state`).
- The Python drivers (`isolation.py`, `traps/`, `cumulative/`) run headless trials in an isolated environment and score them.
- The memory-loop audit (`audit/`) reads real transcripts and records where the recall, action and learning loop broke.

The pytest suites (`python3 -m pytest dev/eval`, in no gate) and the Go tests here (`*_test.go`) describe and check the tools. The former specs (eval-trial-harness, memory-loop-auditor, memory-effectiveness-evaluation) are in git history: promoted in 4d75d0d5, last version at fd15771a.
