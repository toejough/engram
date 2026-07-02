"""Pure scoring helpers for the S3 delivery eval (no I/O, unit-tested in test_s3_score.py).

split_tally : rows [{arm, hit, population, ...}] -> per-population per-arm tally dicts.
              Wraps qanchor_score.tally per sub-group.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "traps"))
from qanchor_score import tally, verdict  # re-export for callers; also used here  # noqa: F401


def split_tally(rows: list, population_key: str = "population") -> dict:
    """Group rows by population_key, then tally per-arm within each group.

    Args:
        rows: list of dicts with at least {arm, hit, population_key}.
        population_key: field name for the population label.

    Returns:
        {population_label: {arm: {n, hits, rate, sigma}}}
    """
    groups: dict[str, list] = {}
    for r in rows:
        pop = r.get(population_key, "unknown")
        groups.setdefault(pop, []).append(r)
    return {pop: tally(group_rows) for pop, group_rows in groups.items()}
