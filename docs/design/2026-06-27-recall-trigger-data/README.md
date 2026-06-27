# Recall-trigger analysis — data trail (2026-06-27)

Provenance + reproducibility for `../2026-06-27-recall-trigger-patterns-and-proposals.md`.

- **`extract_moments.py`** — deterministic extractor over `~/.claude/projects/*engram*/*.jsonl`.
  Reproduces the denominators (8674 tool calls, 635/586 user turns, the 551→78 candidate filter,
  the tool/git/skill counts) and writes `candidates.json`. Re-run: `python3 extract_moments.py`
  (paths are absolute to Joe's machine; adjust `ENGRAM_DIR`/`OUT`). The eval-vocab (85) and
  done-claim (~1141) counts come from a sibling ad-hoc pass described in the doc's §2.
- **`moments_all.json`** (83) / **`moments_trigger.json`** (59) — the classified moments, each with
  `summary`, `source`, `signal_category`, `klass` (TRIGGER/CAPTURE/APPLICATION), `preceding_cue`,
  `lesson`. These are the "long list in substance" (the summaries) and the audit trail for the
  59 TRIGGER / 23 CAPTURE split and the per-cluster source IDs.

Classification was done by LLM subagents (Tier-A over 52 vault feedback notes exhaustively, Tier-B
over the 78 raw candidates, Tier-C spot-check of non-engram logs). The denominators are deterministic;
the classifications are judgment and were adversarially critiqued (see git history of the doc).
