# runbook_vs_skill — Three-Arm Procedural-Memory Eval

Headless measurement of whether a runbook note or fact note in the engram vault can replace a
skill in guiding an agent through a multi-step task (fictional sensor-type registration).

## Design

Three arms, identical task (register `pressure_v2` sensor), procedure written three ways:
- **Arm S (skill):** Procedure as a `SKILL.md` in the trial project's `.claude/skills/sensors-add-skill/`
  (discovered the same way the recall skill is discovered)
- **Arm R (runbook):** Procedure as a runbook note in the fixture vault (situation, steps, `done_when`)
- **Arm F (fact):** Procedure as a semantic fact note in the fixture vault

All arms receive an identical project `CLAUDE.md` — the repo's real `agent-instructions/guidance/recall.md`
inlined verbatim (never `@import`ed), plus a fifth recall cue and a per-run validity marker — and an
identical task prompt (`fixtures/sensor-registration.txt`'s contents, unmodified). Each trial is a fresh
`claude -p`, with an isolated vault/chunks/config that never touches the operator's real memory.

Scoring:
- **FOUND** — Arm S: a `Skill` tool_use naming `sensors-add-skill` before the first procedure-step
  tool call. Arm R/F: a `Bash` tool_use running `engram query` before the first procedure-step tool
  call, whose result contains the fixture note's basename.
- **FOLLOWED** — mean k/6 idiosyncratic steps (registry entry, codegen, migration file, changelog
  entry, git staging, `make validate`), each verified from transcript tool_use or repo state.
- **END-STATE** — `fixtures/done_when_checks.sh <repo>` exits 0 (7 mechanical checks).
- **COST** — `total_cost_usd`/`duration_ms` from the `claude -p` JSON result, plus wall-clock.

Validity gate: a per-run marker (`RUNBOOK-VS-SKILL-PROBE-<uuid>`) is appended to the CLAUDE.md text.
Claude Code attaches a trial's project CLAUDE.md into the session transcript verbatim (a
`"type": "attachment"` record, `attachment.type == "instructions"`) before the first turn, so the
marker's presence in the raw transcript is direct proof the fixture CLAUDE.md reached the trial's
context — no prompt changes or echo instruction needed. `marker_seen=false` trials are recorded with
`valid: false` and excluded from aggregates, never scored 0.

## Usage

```bash
cd dev/eval/cumulative/runbook_vs_skill
```

### Plumbing check (no scoring, proves the pipes are connected)

Runs exactly one Arm-R trial whose prompt asks the agent to run `/recall glance` and report what
the vault returned — it does not touch the sensor task at all. Prints whether the recall skill
fired, whether `engram query` ran, whether the fixture note surfaced in its result, the marker
delivery status, cost, and the transcript path so the print-outs can be hand-verified against the
raw file.

```bash
python3 probe.py --plumbing --model sonnet
```

### Smoke test (validation)

Runs 1 trial per arm with sonnet to validate plumbing (marker delivery, skill loading, vault setup,
transcript parsing, end-state checks).

```bash
python3 probe.py --model sonnet --n 1 --out results/smoke_results.jsonl
```

Expected: 3 results (one per arm), each `valid: true`.

**Pre-registered smoke bars:**
- Marker delivery: 3/3 `marker_seen=true` (validity gate works)
- Plumbing: ≥1/3 `end_state=true` (at least one arm registers the sensor end-to-end)
- Recall delivery: Arm R transcript contains a `Skill` tool_use with `skill=recall` (proves the
  recall+learn skills were copied from the repo into that trial's worker cfg, never from the
  operator's real `~/.claude/`) — checkable directly via `recall_fired=true` in the result record
- Cost: ≤$0.80 total across the 3 sonnet trials

### Full run (paid opus trials)

Runs 5 trials per arm with opus.

```bash
python3 probe.py --model opus --n 5 --out results/opus_results.jsonl
```

Other flags: `--arms S,R,F` (restrict to a subset), `--workers N` (default 4, parallel trials),
`--timeout 900` (per-trial wall-clock seconds; `timed_out` is recorded and repo state is still
scored on a timeout), `--keep` (keep the per-run scratch directory instead of deleting it on exit —
useful for hand-inspecting a transcript after the run).

### Summarize a results file

```bash
python3 probe.py --summarize results/opus_results.jsonl
```

Prints a labeled table (arms as columns; rows FOUND k/n, FOLLOWED mean k/6, END-STATE k/n,
recall_fired k/n, cost mean USD, duration mean s, valid n) followed by the decision frame below,
computed from that file's `valid: true` records.

## Joe's Decision Frame

Two questions, honest n (no rounding; report exact k/n per arm):

### (a) PARITY: Is runbook indistinguishable from skill?

Per metric (FOUND, FOLLOWED, END-STATE):

| Outcome | Definition |
|---------|-----------|
| Indistinguishable | Runbook within 1 trial of skill (e.g., skill 4/5, runbook 3–5/5) |
| Worse | Runbook 2+ below skill (e.g., skill 4/5, runbook ≤2/5) |
| Better | Runbook 2+ above skill (e.g., skill 2/5, runbook ≥4/5) |
| Can't distinguish | Gap ≤1 trial — NOT called "tie" |

FOLLOWED is a continuous per-trial mean (k/6), not a pass/fail count, so `--summarize` converts it
to the same 0..n "trial-equivalent" scale as FOUND/END-STATE — the sum of each valid trial's k/6
(e.g. 5 trials averaging 4/6 → a trial-equivalent of 3.33) — before applying the same ±1/±2 gap
rule. This is the harness's documented generalization of the plan's count-based frame to a
continuous metric; it is exact for FOUND/END-STATE (already integer trial counts) and a linear
proxy for FOLLOWED.

**If Arm S END-STATE < 3/5 (proportionally, <60% of valid trials):** Baseline too hard; parity
uninterpretable. Triggers a fixture/spec fix before re-run — reflected as
`baseline_uninterpretable: true` in `--summarize`'s printed decision frame.

### (b) FUNCTIONALITY: Does runbook exceed fact functionally?

Runbook exceeds fact ONLY if R is 2+ trial-equivalents ahead of F on FOLLOWED or END-STATE;
otherwise "can't distinguish" (`--summarize` prints this as `fact.verdict`).

## Cost Estimate

These trials run tools (Bash, Skill invocations) and edit files, unlike endorse_cue's
single-response probe.

**Source:** `dev/eval/LEDGER.md` row 39–40 (C1-C2 warm-vs-cold): clean n=8/arm measurement shows
"$3.08 costlier" total warm op with tool/file operations. These are tool-rich trials;
endorse_cue (single response) is $0.40/opus trial. Scaling from the LEDGER reference (tool-based,
file-edit operations), budget **$0.60–1.00 per opus trial**.

**Estimate:**
- Smoke (sonnet, 3 trials): 3 × $0.20 = $0.60
- Full run (opus, 15 trials): 15 × $0.80 = $12.00 (mid-range estimate)
- **Range: $12.60–$22.00** (smoke + full at $0.60–1.00/opus trial)
- **Smoke separately:** $0.60 (commit-point after validation)
- **Full run separately:** $12.00–15.00 (commit after results aggregated)

(The one-off `--plumbing` sonnet trial that validated this harness cost $0.16 — see the task
report for the transcript and hand-verified evidence.)

## Fixture & Vault Isolation

Each trial gets:
- Its own fixture repo (`init_fixture_repo.sh` clone of `fixture-repo-template/`), `CLAUDE.md`
  written in, and (Arm S only) `.claude/skills/sensors-add-skill/` copied in, then committed
  (`git add -A && git commit -m "add project config"`) so the trial starts on a clean tree.
- `ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR` under a per-trial scratch directory (never the
  operator's real vault/chunks) — Arm R/F get a copy of `fixture-vaults/{runbook,fact}/vault`;
  Arm S gets an empty vault dir.
- One `CLAUDE_CONFIG_DIR` PER WORKER, not one shared across the whole run — `probe.py::build_cfg_pool`
  builds a single warm cfg template (copying the repo's real recall+learn skills from
  `agent-instructions/skills/` — never the operator's `~/.claude/skills/`) once, then copies it
  into `--workers` independent `cfg-<i>` directories so concurrent trials never contend on the
  same `cfg/.claude.json` / `cfg/projects/` tree. Each trial's `ENGRAM_TRANSCRIPT_DIR` points at
  its own project slug under its checked-out worker's cfg dir.

All trials for one run live under one scratch root
(`$RUNBOOK_VS_SKILL_ROOT`, defaulting to a path under the harness's scratchpad directory) and are
deleted on exit unless `--keep` is passed. The operator's real vault is fingerprinted (file count +
newest mtime) before and after every run; if it changed, the run prints an `ABORT-REPORT` to
stderr — a trial reached real memory and the results should not be trusted.

## Reuse Notes

- Probe design is easily adapted for other skills/runbooks: only the fixture repo, task scenario,
  `done_when` checks, and procedure encodings are task-specific.
- Fixture repo template (sensor system with idiosyncratic registry/migration/ledger format) can
  seed other procedural-memory evals.
- FOUND/FOLLOWED/END-STATE/COST scoring framework reusable; the decision frame (parity +
  functionality questions) generalizes to other note-vs-skill comparisons.
