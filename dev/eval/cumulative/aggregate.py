#!/usr/bin/env python3
"""Aggregate the cumulative-accumulation matrix into the final tables.
Reads /tmp/cummatrix/results/*.json, prints: per-model bucketed round-1 ladders,
the model×memory delta, the β-accumulation deltas, L2-iso vs blended, and cost.
"""
import json, glob, os, collections, statistics

R = "/tmp/cummatrix/results"
MODELS = ["haiku", "sonnet", "opus"]
KEYS = ["cold", "L1-notes", "L1-noteslinks", "L2-notes", "L2-noteslinks",
        "L3-notes", "L3-noteslinks", "blended-notes", "blended-noteslinks"]

def num(x):
    try: return int(str(x).split("/")[0])
    except Exception: return None

def load():
    data = collections.defaultdict(list)   # (model,key) -> list of dicts
    cost = collections.defaultdict(float)   # model -> $
    for f in glob.glob(f"{R}/*.json"):
        d = json.load(open(f))
        name = os.path.basename(f).replace(".json", "")
        parts = name.split("-")
        model = parts[0]
        cost[model] += d.get("total_cost", 0) or 0
        if "feeds" not in parts:
            continue
        if d.get("round1_score") is None:
            continue
        idx = parts.index("feeds"); key = "-".join(parts[idx + 1:])
        b = d.get("round1_buckets") or {}
        fb = d.get("final_buckets") or {}
        data[(model, key)].append(dict(
            tot=num(d["round1_score"]), a=num(b.get("alpha")), be=num(b.get("beta")),
            nat=num(b.get("native")), ar=num(d.get("round1_arch")),
            conv=bool(d.get("converged")), rounds=d.get("rounds_to_converge"),
            ftot=num(d.get("final_score")), fbe=num(fb.get("beta")),
            tcost=d.get("total_cost", 0) or 0, wall=d.get("wall_min", 0) or 0))
    return data, cost

def mean(xs):
    xs = [x for x in xs if x is not None]
    return sum(xs) / len(xs) if xs else float("nan")

def sd(xs):
    xs = [x for x in xs if x is not None]
    return statistics.pstdev(xs) if len(xs) > 1 else 0.0

def main():
    data, cost = load()
    def cell(m, k, field="tot"): return mean([x[field] for x in data.get((m, k), [])])
    def celln(m, k): return len(data.get((m, k), []))

    print("=" * 78)
    print("ROUND-1 CONFORMANCE /18 — mean ± sd (n) per model × regime-stage")
    print("=" * 78)
    print("%-22s %-14s %-14s %-14s" % ("regime-stage", "haiku", "sonnet", "opus"))
    for k in KEYS:
        row = "  %-20s" % k
        for m in MODELS:
            v = data.get((m, k), [])
            if v:
                row += " %4.1f±%-3.1f(%d) " % (mean([x["tot"] for x in v]), sd([x["tot"] for x in v]), len(v))
            else:
                row += " %-13s" % "—"
        print(row)

    print("\n" + "=" * 78)
    print("MODEL × MEMORY: cold vs best-memory round-1, and the gain (Δ)")
    print("=" * 78)
    for m in MODELS:
        c = cell(m, "cold")
        best_k = max([k for k in KEYS if k != "cold"], key=lambda k: cell(m, k) if celln(m, k) else -1)
        bestv = cell(m, best_k)
        print("  %-7s cold %.1f → best(%s) %.1f   Δ=+%.1f" % (m, c, best_k, bestv, bestv - c))

    print("\n" + "=" * 78)
    print("β-ACCUMULATION: β/4 at +notes → +notes+links (links teaches β)")
    print("=" * 78)
    for reg in ["L1", "L2", "L3", "blended"]:
        print("  %-9s" % reg, end="")
        for m in MODELS:
            n = cell(m, f"{reg}-notes", "be"); nl = cell(m, f"{reg}-noteslinks", "be")
            print("  %s: %.1f→%.1f" % (m, n, nl), end="")
        print()

    print("\n" + "=" * 78)
    print("RECALL REGIME @ +notes+links — round-1 /18 (which recall strategy wins)")
    print("=" * 78)
    print("%-10s %-12s %-12s %-12s" % ("regime", "haiku", "sonnet", "opus"))
    for reg in ["L1", "L2", "L3", "blended"]:
        k = f"{reg}-noteslinks"
        print("  %-8s" % reg, end="")
        for m in MODELS:
            print(" %5.1f(n=%d)  " % (cell(m, k), celln(m, k)), end="")
        print()

    print("\n" + "=" * 78)
    print("CONVERGENCE + COST")
    print("=" * 78)
    for m in MODELS:
        convs = [x["conv"] for k in KEYS for x in data.get((m, k), [])]
        cr = sum(1 for c in convs if c)
        print("  %-7s converged %d/%d feeds cells | model spend $%.2f" % (m, cr, len(convs), cost[m]))
    print("  GRAND TOTAL spend: $%.2f" % sum(cost.values()))

if __name__ == "__main__":
    main()
