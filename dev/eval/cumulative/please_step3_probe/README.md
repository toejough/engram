# please_step3_probe

Headless micro-test for `skills/please/SKILL.md`'s Step 3 (Plan) enumeration-grep step and
Gate A's docs/diagrams-alignment charge (engram issue #685, Change #1). Built per
`docs/superpowers/plans/2026-07-15-685-doc-enumeration-grep.md`.

- `--role author` (default): does Step 3's text make a plan author run a real grep and paste
  a complete, verified per-file disposition list, or hand-wave/memory-source the doc scope?
- `--role reviewer`: does the docs/diagrams-alignment charge make a Gate A reviewer catch a
  gap in an author-pasted list via an independent pass, or rubber-stamp what's pasted?
- `--role loaded_author` (or `--loaded`): the clean `--role author` probe above proved #685's
  failure is NOT a capability gap — a capable model enumerates the full doc surface
  thoroughly when doc-scrub IS its salient task, even under explicit pressure (6/6). But
  #685's real production failures happened when doc-scrub was a BURIED SUBTASK under
  feature-work load. This role reproduces that: a substantive 3-component feature ask
  (env-var config, metrics counter, graceful shutdown) that never mentions docs,
  enumeration, or grep — doc-surface enumeration is only an incidental, low-salience
  consequence of correctly scoping the change.

Fixture: `testdata/fixture/` is a tiny "sweeper" service whose cadence invariant ("6 hours")
is echoed as a digit form, a synonym ("hexahourly"), a hyphenated form ("six-hour"), and a
diagram/comment echo across 5 real files, plus one distractor file with 0 mentions.
`testdata/fixture_reviewer/incomplete_plan.md` is a deliberately incomplete author-pasted
list (missing the hardest file) used only by `--role reviewer`.

`testdata/fixture_loaded/` is a small-but-substantive sweeper service (~11 files: 5 feature
Go files under `cmd/`/`internal/`, plus docs) used only by `--role loaded_author`. The cadence
invariant is echoed across `README.md`, `docs/architecture.md`, `docs/glossary.md`, and
`skills/operator/SKILL.md` in digit form ("6h"/"6 hours"); `docs/runbook.md` is the
variant-only discriminator — it references the cadence ONLY via synonym/hyphenated forms
("six-hourly", "hexahourly") and never contains the literal "6h"/"6 hours", so a literal grep
misses it while a concept-variant search finds it. `docs/unrelated.md` is the distractor (0
cadence references).

Usage:
```bash
python3 run_probe.py --skill-text ../../../../skills/please/SKILL.md --n 3 \
    --out results/red_baseline.jsonl --model sonnet

python3 run_probe.py --skill-text ../../../../skills/please/SKILL.md --loaded --n 1 \
    --out results/loaded_smoke.jsonl --model sonnet
```

Scoring is mechanical (file-coverage + grep-evidence regex), not an LLM judge — read
`raw_result` in the output JSONL by hand before trusting the aggregate pass rate.
