## 1. Recall skill wording fix (writing-skills TDD — required, no exceptions per vault note 26)

- [ ] 1.1 Baseline (RED): cite the preserved failing-trial transcript
      (`gate-C5-6l_mjvl1/warm-cfg/projects/.../warm-3-*/*.jsonl`, or re-run
      `dev/eval/traps/seed_c5.py && dev/eval/traps/c5.py --arms warm --n 5` fresh if the
      original artifacts have been cleaned up) as RED evidence — the agent's own synthesis text
      ("0 notes; crystallized 0 lessons... unrelated seed content") demonstrates the failure
      mode on the CURRENT Step 2.5A wording.
- [ ] 1.2 Invoke `superpowers:writing-skills` to edit `agent-instructions/skills/recall/SKILL.md`
      Step 2.5A: strengthen the read-before-judge instruction specifically for the zero-note,
      chunks-only cluster case (see this change's `design.md` Open Question 2).
- [ ] 1.3 GREEN: re-run `dev/eval/traps/seed_c5.py && dev/eval/traps/c5.py --arms warm --n 5`
      against the new wording; confirm the failure mode from 1.1 no longer reproduces.
- [ ] 1.4 Pressure-test the new wording against fresh rationalization loopholes (writing-skills
      TDD requirement) — e.g. a cluster with 0 notes AND 0 load-bearing chunks, to confirm the
      agent doesn't over-correct into reading chunks it has no real need to fetch.
- [ ] 1.5 Sync the fixed skill text to deployable form (`engram update --with-guidance` or this
      repo's current equivalent); confirm via a diff against
      `agent-instructions/skills/recall/SKILL.md`.

## 2. Re-measure and decide the gate threshold

- [ ] 2.1 Run a larger-n measurement (`dev/eval/traps/c5.py --arms warm --n 15` or more) against
      the FIXED wording for a real post-fix compliance rate with a narrower confidence interval
      than n=5.
- [ ] 2.2 Using the rate from 2.1, decide `design.md` Open Question 1: keep
      `dev/eval/traps/gate_verdict.py`'s exact-bar C5 verdict (if the fix reliably clears it),
      or move to a threshold-based test (if it doesn't) — record the decision and rationale in
      `dev/eval/LEDGER.md`.
- [ ] 2.3 If the gate criterion changes, update `dev/eval/traps/gate_verdict.py`
      (`_norm_c5` / `axis_verdict`) and its unit tests in `test_gate.py` accordingly.
- [ ] 2.4 Using the rate from 2.1, update `docs/GLOSSARY.md`'s lazy-chunks entry (lines 372-379)
      and both architecture diagram comments (`docs/architecture/c2-containers.md:126`,
      `docs/architecture/c1-system-context.md:165-166`) to add the zero-note-cluster
      mandatory-read case explicitly and re-state the fetch-frequency figure against the
      post-fix measurement rather than the pre-fix "0/13".

## 3. Close out

- [ ] 3.1 Sync specs (`/opsx:sync` or `openspec` equivalent) to merge the ADDED requirement into
      `openspec/specs/recall-payload-cuts/spec.md`.
- [ ] 3.2 Update `dev/eval/traps/README.md`'s C5 bar description and `dev/eval/LEDGER.md`'s
      `733-c5b-honoring-rate` row with the resolution (per #733's own acceptance criteria).
- [ ] 3.3 Close GitHub issue #733, referencing the merged spec, the SKILL.md commit, and the
      final measured rate.
- [ ] 3.4 Archive this change (`openspec archive recall-chunk-only-cluster-read-gate`).
