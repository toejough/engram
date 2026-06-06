#!/usr/bin/env python3
"""Compare two cumulative-accumulation run dirs and print the deltas on the primary
metric (convention interventions to endpoint, per regime x model). This is what makes
the benchmark a STANDING one: a future run (new model / new engram feature) diffs against
this baseline instead of re-tabulating from scratch (§6).

Usage: python3 compare.py <baseline-run-dir> <new-run-dir>
       (a run dir is a --root, e.g. /tmp/cummatrix or an archived results dir's parent)
"""
import argparse, os, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import aggregate


def conv_table(root):
    builds, learns, manifest = aggregate.load(root)
    if not builds:
        return None, [], [], manifest
    models = manifest.get("models") or sorted({b.get("model") for b in builds})
    regimes = manifest.get("regimes") or sorted({b.get("regime") for b in builds if b.get("app") != "notes"})
    return aggregate.chain_intervention_table(builds, models, regimes, "convention_statements"), models, regimes, manifest


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("baseline")
    ap.add_argument("new")
    args = ap.parse_args()

    base, bmodels, bregimes, bman = conv_table(args.baseline)
    new, nmodels, nregimes, nman = conv_table(args.new)
    if base is None or new is None:
        print("one or both run dirs have no build results")
        return

    print("# Compare — convention interventions to endpoint (primary metric)\n")
    print(f"baseline: `{bman.get('engram_sha','?')}` ({bman.get('date','?')})   "
          f"new: `{nman.get('engram_sha','?')}` ({nman.get('date','?')})\n")

    models = [m for m in nmodels if m in bmodels] or nmodels
    regimes = [r for r in nregimes if r in bregimes] or nregimes

    header = "| regime | " + " | ".join(f"{m} (base→new Δ)" for m in models) + " |"
    print(header)
    print("|---|" + "|".join(["---:"] * len(models)) + "|")
    for r in regimes:
        cells = []
        for m in models:
            b = base.get((r, m))
            n = new.get((r, m))
            if b is None or n is None:
                cells.append("—")
            else:
                cells.append(f"{b:.1f}→{n:.1f} ({n-b:+.1f})")
        print(f"| `{r}` | " + " | ".join(cells) + " |")


if __name__ == "__main__":
    main()
