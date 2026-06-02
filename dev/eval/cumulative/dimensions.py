#!/usr/bin/env python3
"""Per-model measured dimensions across regimes/stages: conformance, human
review-turns, LLM turns, cost, wall-time, convergence rate."""
import json, glob, os, collections, statistics

R = "/tmp/cummatrix/results"
MODELS = ["haiku", "sonnet", "opus"]
KEYS = ["cold", "L1-notes", "L1-noteslinks", "L2-notes", "L2-noteslinks",
        "L3-notes", "L3-noteslinks", "blended-notes", "blended-noteslinks"]

def num(x):
    try: return int(str(x).split("/")[0])
    except Exception: return None

def load():
    d = collections.defaultdict(list)
    for f in glob.glob(f"{R}/*feeds*.json"):
        o = json.load(open(f))
        if o.get("round1_score") is None: continue
        p = os.path.basename(f).replace(".json", "").split("-")
        m = p[0]; key = "-".join(p[p.index("feeds") + 1:])
        rounds_used = len(o.get("rounds", []))
        conv = bool(o.get("converged"))
        # human review-turns = feedback messages = rounds after the first build.
        # If converged, that's rounds_to_converge-1; if not, it used (rounds_used-1) and still failed.
        rtc = o.get("rounds_to_converge")
        human_turns = (rtc - 1) if (conv and rtc) else (rounds_used - 1)
        d[(m, key)].append(dict(
            r1=num(o["round1_score"]), final=num(o.get("final_score")),
            conv=conv, human_turns=human_turns, rounds_used=rounds_used,
            llm_turns=o.get("build_turns", 0) or 0,
            cost=o.get("total_cost", 0) or 0, wall=o.get("wall_min", 0) or 0))
    return d

def mean(xs):
    xs = [x for x in xs if x is not None]
    return sum(xs) / len(xs) if xs else float("nan")

def main():
    d = load()
    for m in MODELS:
        print("\n" + "=" * 92)
        print(f"  {m.upper()} — measured dimensions per regime × stage (mean over n=3)")
        print("=" * 92)
        print("  %-20s %6s %6s %8s %8s %8s %8s %7s" % (
            "regime-stage", "r1/18", "fin/18", "humanTr", "llmTrns", "cost$", "wall_m", "conv"))
        for k in KEYS:
            v = d.get((m, k), [])
            if not v:
                print("  %-20s  (none)" % k); continue
            n = len(v)
            convn = sum(1 for x in v if x["conv"])
            print("  %-20s %6.1f %6.1f %8.1f %8.1f %8.2f %8.1f %5d/%d" % (
                k, mean([x["r1"] for x in v]), mean([x["final"] for x in v]),
                mean([x["human_turns"] for x in v]), mean([x["llm_turns"] for x in v]),
                mean([x["cost"] for x in v]), mean([x["wall"] for x in v]), convn, n))

    # cross-model rollup: averaged over all 9 feeds configs
    print("\n" + "=" * 92)
    print("  CROSS-MODEL ROLLUP (mean over all 9 regime-stage feeds configs, n=27/model)")
    print("=" * 92)
    print("  %-8s %7s %7s %9s %9s %9s %9s %9s" % (
        "model", "r1/18", "fin/18", "humanTr", "llmTrns", "cost$", "wall_m", "conv%"))
    for m in MODELS:
        allc = [x for k in KEYS for x in d.get((m, k), [])]
        convn = sum(1 for x in allc if x["conv"])
        print("  %-8s %7.1f %7.1f %9.1f %9.1f %9.2f %9.1f %8.0f%%" % (
            m, mean([x["r1"] for x in allc]), mean([x["final"] for x in allc]),
            mean([x["human_turns"] for x in allc]), mean([x["llm_turns"] for x in allc]),
            mean([x["cost"] for x in allc]), mean([x["wall"] for x in allc]),
            100 * convn / len(allc)))

    # cold baseline isolated (the no-memory reference)
    print("\n  COLD baseline only (no memory), per model:")
    for m in MODELS:
        v = d.get((m, "cold"), [])
        print("    %-8s r1=%.1f final=%.1f humanTurns=%.1f llmTurns=%.1f cost=$%.2f wall=%.1fm conv=%d/3" % (
            m, mean([x["r1"] for x in v]), mean([x["final"] for x in v]),
            mean([x["human_turns"] for x in v]), mean([x["llm_turns"] for x in v]),
            mean([x["cost"] for x in v]), mean([x["wall"] for x in v]),
            sum(1 for x in v if x["conv"])))

if __name__ == "__main__":
    main()
