"""confirm_s2.py — real-binary confirmation of the swept vocab config.

Parses the ACTUAL `vocab:` frontmatter lists written by `engram vocab
bootstrap` (the shipped centroid two-pass at the binary's defaults) from a
bootstrapped vault copy, then runs the same 48-case nomination probe the sweep
used. Confirms the python twopass model against the real implementation and
reports PASS/FAIL against the re-anchored gate (recovery ≥ 54.2% @ median
pool ≤ 40).

Usage:
    python dev/eval/links/confirm_s2.py --vault /path/to/bootstrapped/copy
"""
from __future__ import annotations

import argparse
import os
import re

from sweep_s2 import (
    GATE_MEDIAN_POOL,
    GATE_RECOVERY_PCT,
    MISSES_P1_PATH,
    BRIDGES_P2_PATH,
    P3_BASELINES_PATH,
    REPLAYS_PATH,
    build_cases,
    load_json,
    run_probe,
)

VOCAB_LINE = re.compile(r"^vocab:\s*\[(.*)\]\s*$", re.MULTILINE)


def parse_real_assignments(vault: str) -> dict:
    """Build the l6-equivalent assignments dict from the notes' actual
    `vocab:` frontmatter lists (the binary's flow-style output)."""
    assignments = []
    for fname in sorted(os.listdir(vault)):
        if not fname.endswith(".md") or fname.startswith("vocab."):
            continue
        with open(os.path.join(vault, fname)) as fh:
            text = fh.read()
        match = VOCAB_LINE.search(text)
        if not match:
            continue
        tags = [t.strip() for t in match.group(1).split(",") if t.strip()]
        if tags:
            assignments.append({"note": fname[:-3], "tags": tags})
    return {"assignments": assignments}


def main() -> None:
    parser = argparse.ArgumentParser(description="real-binary vocab config confirmation.")
    parser.add_argument("--vault", required=True, help="Path to the REAL-binary bootstrapped copy.")
    args = parser.parse_args()

    if not os.path.exists(P3_BASELINES_PATH):
        raise SystemExit(f"FATAL: {P3_BASELINES_PATH} missing — no fallback population.")

    cases = build_cases(
        load_json(REPLAYS_PATH),
        load_json(MISSES_P1_PATH),
        load_json(BRIDGES_P2_PATH),
        load_json(P3_BASELINES_PATH),
    )

    real = parse_real_assignments(args.vault)
    n_tagged = len(real["assignments"])
    metrics = run_probe(cases, real)

    gate_pass = (
        metrics["recovery"] >= GATE_RECOVERY_PCT
        and metrics["median_pool"] <= GATE_MEDIAN_POOL
    )

    print(f"REAL-BINARY CONFIRMATION ({len(cases)} cases, {n_tagged} tagged notes)")
    print(
        f"  recovery={metrics['recovery']}% ({metrics['recovered']}/{metrics['n']})  "
        f"median_pool={metrics['median_pool']}  by_kind={metrics['by_kind']}"
    )
    print(
        f"  GATE (≥{GATE_RECOVERY_PCT}% @ ≤{GATE_MEDIAN_POOL}): "
        f"{'PASS' if gate_pass else 'FAIL'}"
    )


if __name__ == "__main__":
    main()
