#!/usr/bin/env python3
"""Aggregate v2 cumulative-accumulation results into the §5 headline tables and a
results-vN.md in the 2026-06-02 doc's shape.

PRIMARY metric — repeated-convention interventions (the say-once signal): per build we
record how many transferable conventions had to be STATED (round-1 ARCH fails, since §4
feeds back all gaps). Summed across the app1->app2->app3 chain, memory should approach
|conventions| stated once; no-memory approaches |conventions| x 3. App-specific FEATURE
interventions are the control — memory should not move them.

Reads the per-op result JSONs written by harness.py (build/learn) under <root>/results/.

Usage: python3 aggregate.py [--root /tmp/cummatrix] [--out results-v2.md]
"""
import argparse, collections, glob, json, os, statistics


def load(root):
    builds, learns, manifest = [], [], {}
    mp = os.path.join(root, "results", "run-manifest.json")
    if os.path.exists(mp):
        manifest = json.load(open(mp))
    for path in glob.glob(os.path.join(root, "results", "*.json")):
        if path.endswith("run-manifest.json"):
            continue
        try:
            d = json.load(open(path))
        except Exception:
            continue
        if d.get("kind") == "build":
            builds.append(d)
        elif d.get("kind") == "learn":
            learns.append(d)
    return builds, learns, manifest


def mean(xs):
    xs = [x for x in xs if x is not None]
    return statistics.mean(xs) if xs else None


def fmt(x, nd=1):
    return "—" if x is None else f"{x:.{nd}f}"


def chain_intervention_table(builds, models, regimes, key):
    """Per (regime, model): mean over trials of the chain-summed `key` count
    (convention_statements or feature_statements) across app1+app2+app3. app1 (notes)
    is the shared cold build per (model, trial)."""
    idx = collections.defaultdict(dict)
    app1_by_mt = collections.defaultdict(list)
    for b in builds:
        m, r, t, a = b.get("model"), b.get("regime"), b.get("trial"), b.get("app")
        v = b.get(key, 0) or 0
        if a == "notes":
            app1_by_mt[(m, t)].append(v)
        else:
            idx[(m, r, t)][a] = v

    table = {}
    for r in regimes:
        for m in models:
            trials = sorted({t for (mm, rr, t) in idx if mm == m and rr == r}) or \
                     sorted({t for (mm, t) in app1_by_mt if mm == m})
            totals = []
            for t in trials:
                app1 = mean(app1_by_mt.get((m, t), [])) or 0
                app2 = idx.get((m, r, t), {}).get("links", 0)
                app3 = idx.get((m, r, t), {}).get("feeds", 0)
                totals.append(app1 + app2 + app3)
            table[(r, m)] = mean(totals)
    return table


def per_app_numeric(builds, models, regimes, app, key):
    idx = collections.defaultdict(list)
    for b in builds:
        if b.get("app") == app:
            idx[(b.get("regime"), b.get("model"))].append(b.get(key))
    return {(r, m): mean(idx.get((r, m), [])) for r in regimes for m in models}


def beta_table(builds, models, regimes):
    idx = collections.defaultdict(list)
    for b in builds:
        if b.get("app") == "feeds":
            idx[(b.get("regime"), b.get("model"))].append(_bucket_num((b.get("final_buckets") or {}).get("beta")))
    return {(r, m): mean(idx.get((r, m), [])) for r in regimes for m in models}


def cost_time_table(builds, learns, models, regimes):
    """Total $ and wall-min to the endpoint per (regime, model), mean over trials.
    app1's single cold build + its tier learn are amortized into each chain."""
    out = {}
    for r in regimes:
        for m in models:
            trials = sorted({b.get("trial") for b in builds if b.get("model") == m})
            costs, mins = [], []
            for t in trials:
                cost = wall = 0.0
                for b in builds:
                    if b.get("model") == m and b.get("trial") == t and \
                       (b.get("app") == "notes" or b.get("regime") == r):
                        cost += b.get("build_cost", 0) or 0
                        wall += b.get("wall_min", 0) or 0
                for le in learns:
                    if le.get("model") == m and le.get("trial") == t and \
                       (str(le.get("regime", "")).startswith("app1-") or le.get("regime") == r):
                        cost += le.get("learn_cost", 0) or 0
                costs.append(cost)
                mins.append(wall)
            out[(r, m)] = (mean(costs), mean(mins))
    return out


def render_table(title, table, models, regimes, nd=1):
    lines = [f"### {title}", "", "| regime | " + " | ".join(models) + " |",
             "|---|" + "|".join(["---:"] * len(models)) + "|"]
    for r in regimes:
        lines.append(f"| `{r}` | " + " | ".join(fmt(table.get((r, m)), nd) for m in models) + " |")
    return "\n".join(lines) + "\n"


def learn_quality_table(learns, models):
    """Per (write-tier, model): did the agent's /learn actually capture the conventions we expect
    for that level? mean coverage = captured/stated; engaged = fraction that wrote any vault note.
    A measured output (the agent runs /learn itself; learn-quality is part of the evaluation)."""
    cov = collections.defaultdict(list)
    eng = collections.defaultdict(list)
    for le in learns:
        q = le.get("learn_quality") or {}
        if not le.get("stated_conventions_input") and le.get("write_tier") != "L1":
            continue  # nothing was stated → coverage is trivially 1.0; skip to avoid inflating
        wt = le.get("write_tier")
        if wt in ("L1", "L2", "L3"):
            cov[(wt, le.get("model"))].append(q.get("coverage"))
            eng[(wt, le.get("model"))].append(1.0 if q.get("engaged") else 0.0)
    lines = ["### Learn-capture quality (did the agent persist the stated conventions per tier)", "",
             "Cell = mean coverage (captured/stated) · engaged% (wrote any note). The agent runs its "
             "own /learn skill — this measures whether the memory system captured what matters.", "",
             "| write-tier | " + " | ".join(models) + " |", "|---|" + "|".join(["---:"] * len(models)) + "|"]
    for wt in ["L1", "L2", "L3"]:
        cells = []
        for m in models:
            c, e = mean(cov.get((wt, m), [])), mean(eng.get((wt, m), []))
            cells.append("—" if c is None else f"{c:.2f} · {0 if e is None else round(e*100)}%")
        lines.append(f"| `{wt}` | " + " | ".join(cells) + " |")
    return "\n".join(lines) + "\n"


def cost_calibration(builds, learns):
    """Per-operation cost (build vs learn, by app, by model) for grounding the full-run
    spend estimate — builds are multi-round, learns single (advisor §4)."""
    bcost, brounds, lcost = collections.defaultdict(list), collections.defaultdict(list), collections.defaultdict(list)
    for b in builds:
        bcost[(b.get("model"), b.get("app"))].append(b.get("build_cost", 0) or 0)
        brounds[(b.get("model"), b.get("app"))].append(len(b.get("rounds", [])))
    for le in learns:
        if le.get("learned"):
            lcost[(le.get("model"), le.get("app"))].append(le.get("learn_cost", 0) or 0)

    lines = ["### Cost calibration (per-operation; grounds the full-run estimate)", "",
             "| op | model | app | n | mean $ | mean rounds |", "|---|---|---|--:|--:|--:|"]
    for (m, a), xs in sorted(bcost.items()):
        lines.append(f"| build | {m} | {a} | {len(xs)} | {fmt(mean(xs),2)} | {fmt(mean(brounds[(m,a)]),1)} |")
    for (m, a), xs in sorted(lcost.items()):
        lines.append(f"| learn | {m} | {a} | {len(xs)} | {fmt(mean(xs),2)} | — |")
    return "\n".join(lines) + "\n"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=os.environ.get("CUMMATRIX_ROOT", "/tmp/cummatrix"))
    ap.add_argument("--out", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "results-v2.md"))
    args = ap.parse_args()

    builds, learns, manifest = load(args.root)
    if not builds:
        print(f"no build results under {args.root}/results/ — run matrix.py first")
        return

    models = manifest.get("models") or sorted({b.get("model") for b in builds})
    regimes = manifest.get("regimes") or sorted({b.get("regime") for b in builds if b.get("app") != "notes"})

    conv = chain_intervention_table(builds, models, regimes, "convention_statements")
    feat = chain_intervention_table(builds, models, regimes, "feature_statements")
    beta = beta_table(builds, models, regimes)
    followed = per_app_numeric(builds, models, regimes, "feeds", "link_followed")
    costtime = cost_time_table(builds, learns, models, regimes)

    stub_note = "  ·  **STUB RUN (no LLM — mechanics only, numbers are not real)**" if manifest.get("stub") else ""
    doc = ["# Cumulative cross-app memory accumulation — results (v2)", "",
           f"Engram SHA: `{manifest.get('engram_sha','?')}` · date: {manifest.get('date','?')} · "
           f"models: {', '.join(models)} · trials: {manifest.get('trials','?')} · "
           f"price sheet: {manifest.get('price_sheet_date','?')}" + stub_note, "",
           "> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable "
           "cell-for-cell to the 2026-06-02 run.", "",
           "## Primary — repeated-convention interventions (say-once vs every-app)", "",
           "Chain-summed conventions the human had to STATE (app1+app2+app3). "
           "Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. "
           "The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.", "",
           render_table("Convention interventions to endpoint (mean/trial)", conv, models, regimes),
           render_table("Feature interventions — CONTROL (memory should not move these)", feat, models, regimes),
           "## Secondary", "",
           render_table("β-bucket on feeds (does β transfer once links' memory is present)", beta, models, regimes, 2),
           render_table("Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)",
                        followed, models, regimes, 2),
           "### Cost + time to endpoint (mean $/min per trial)", "",
           "| regime | " + " | ".join(models) + " |", "|---|" + "|".join(["---:"] * len(models)) + "|"]
    for r in regimes:
        cells = []
        for m in models:
            c, mn = costtime.get((r, m), (None, None))
            cells.append(f"{fmt(c,2)} / {fmt(mn,0)}")
        doc.append(f"| `{r}` | " + " | ".join(cells) + " |")
    doc += ["", learn_quality_table(learns, models), "", cost_calibration(builds, learns)]

    out = "\n".join(doc)
    open(args.out, "w").write(out)
    print(out)
    print(f"\n[written to {args.out}]")


def _bucket_num(s):
    try:
        return float(str(s).split("/")[0])
    except Exception:
        return None


if __name__ == "__main__":
    main()
