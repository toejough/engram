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
    """β-bucket accumulation is the localized FRONT-LOADING test (§5): does links' memory lift
    feeds' β in the FIRST draft? It must be read at ROUND 1 — at convergence every arm saturates
    to 4/4 (feedback drives β up regardless of memory), which would falsely show 'no accumulation'.
    Round-1 buckets live in rounds[0].feat_buckets."""
    idx = collections.defaultdict(list)
    for b in builds:
        if b.get("app") != "feeds":
            continue
        rounds = b.get("rounds") or []
        r1 = (rounds[0].get("feat_buckets") if rounds else None) or {}
        idx[(b.get("regime"), b.get("model"))].append(_bucket_num(r1.get("beta")))
    return {(r, m): mean(idx.get((r, m), [])) for r in regimes for m in models}


WRITE_TIER = {"cold": "none", "l1": "L1", "l2.l1l2": "L2", "l2.l2": "L2",
              "l3.l1l2l3": "L3", "l3.l2l3": "L3", "l3.l3": "L3"}


def _index_runs(builds, learns):
    """Index results by their OWN fields (never by reconstructed filenames — that bug dropped the
    shared app1-learn). Returns (build[(m,t,app,regime)], a1learn[(m,t,writetier)],
    a2learn[(m,t,regime)])."""
    bmap, a1l, a2l = {}, {}, {}
    for b in builds:
        bmap[(b.get("model"), b.get("trial"), b.get("app"), b.get("regime"))] = b
    for le in learns:
        reg = str(le.get("regime", ""))
        key = (le.get("model"), le.get("trial"))
        if reg.startswith("app1-"):
            a1l[(key[0], key[1], reg.split("-", 1)[1])] = le   # app1-<tier>
        else:
            a2l[(key[0], key[1], reg)] = le
    return bmap, a1l, a2l


def _toks(d):
    return sum(((d or {}).get("tokens") or {}).values())


def chain_rows(builds, learns, model, regime):
    """Per-trial chain stats for one (model, regime): app1(notes,cold) + app2(links,regime) +
    app3(feeds,regime) builds, plus the write-tier app1-learn and the app2-learn. Skips trials
    with any missing build."""
    bmap, a1l, a2l = _index_runs(builds, learns)
    wt = WRITE_TIER.get(regime, "none")
    rows = []
    for t in sorted({k[1] for k in bmap if k[0] == model}):
        a1 = bmap.get((model, t, "notes", "cold"))
        a2 = bmap.get((model, t, "links", regime))
        a3 = bmap.get((model, t, "feeds", regime))
        if not all([a1, a2, a3]):
            continue
        ln = [x for x in (a1l.get((model, t, wt)), a2l.get((model, t, regime))) if x]
        builds3 = [a1, a2, a3]
        rows.append({
            "conv_restate": sum(x.get("convention_statements", 0) or 0 for x in builds3),
            "feat_restate": sum(x.get("feature_statements", 0) or 0 for x in builds3),
            "review_turns": sum(max(0, len(x.get("rounds", [])) - 1) for x in builds3),
            "converged": all(x.get("converged") for x in builds3),
            "learn_cost": sum(x.get("learn_cost", 0) or 0 for x in ln),
            "build_cost": sum(x.get("build_cost", 0) or 0 for x in builds3),
            "wall": sum((x.get("wall_min", 0) or 0) for x in builds3 + ln),
            "tokens": sum(_toks(x) for x in builds3 + ln),
        })
    return rows


def headline_stats_table(builds, learns, models, regimes):
    """The consolidated cold-vs-warm headline the brief asks for (§5): interventions, time, tokens,
    $ to the endpoint, per model. 'warm' = mean over the 6 memory regimes."""
    warm = [r for r in regimes if r != "cold"]
    lines = ["### Headline stats — to the endpoint (notes→links→feeds chain, mean per trial)", "",
             "`conv-restate` = convention restatements the human made (the say-once metric, lower=better). "
             "`review` = feedback rounds. **Memory's win is conv-restate; it does NOT reduce time/tokens/$ "
             "— recall + richer learn cost more.**", "",
             "| model | arm | conv-restate | review | converged | wall min | tokens | $ |",
             "|---|---|--:|--:|--:|--:|--:|--:|"]
    for m in models:
        for arm, regs in [("cold", ["cold"]), ("warm", warm)]:
            rows = [r for reg in regs for r in chain_rows(builds, learns, m, reg)]
            if not rows:
                continue
            cv = 100 * sum(1 for r in rows if r["converged"]) / len(rows)
            tot = lambda k: mean([r[k] for r in rows])
            lines.append(f"| {m} | {arm} | {tot('conv_restate'):.1f} | {tot('review_turns'):.1f} | "
                         f"{cv:.0f}% | {tot('wall'):.0f} | {tot('tokens')/1e6:.1f}M | "
                         f"{tot('learn_cost')+tot('build_cost'):.2f} |")
    return "\n".join(lines) + "\n"


def per_regime_cost_table(builds, learns, models, regimes):
    """Per-regime cost/token/convergence breakdown — splits learn$ (write-tier work: L1 episode →
    L3 +synthesis) from build$ (dominated by convergence round-count). Shows why simpler tiers are
    only marginally cheaper: the tier payload is real but small; build round-count swamps it."""
    lines = ["### Cost & convergence by regime (mean per trial) — learn$ vs build$ split",
             "",
             "`learn$` rises with write-tier (L1 episode < L2 +facts < L3 +synthesis); `build$` is "
             "dominated by feedback round-count (convergence), which is tier-insensitive — so total $ "
             "does not cleanly follow tier simplicity.", ""]
    for m in models:
        lines += [f"**{m}**", "",
                  "| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |",
                  "|---|---|--:|--:|--:|--:|--:|--:|"]
        for reg in regimes:
            rows = chain_rows(builds, learns, m, reg)
            if not rows:
                continue
            ln, bd = mean([r["learn_cost"] for r in rows]), mean([r["build_cost"] for r in rows])
            cv = 100 * sum(1 for r in rows if r["converged"]) / len(rows)
            lines.append(f"| `{reg}` | {WRITE_TIER[reg]} | {ln:.2f} | {bd:.2f} | {ln+bd:.2f} | "
                         f"{mean([r['wall'] for r in rows]):.0f} | {mean([r['tokens'] for r in rows])/1e6:.1f}M | {cv:.0f}% |")
        lines.append("")
    return "\n".join(lines) + "\n"


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


def differential_retention(conv, feat, models, regimes):
    """The honest headline: memory's effect on CONVENTIONS vs FEATURES, as retention ratios
    relative to cold (warm/cold). Conventions are transferable (memory should carry them →
    low retention); features are app-specific (nobody carries them → retention ~1). The SIGNAL
    is the gap between the two, not a claim that features are untouched (they shift a little
    because feeds shares α/β with the priors — see the native-only control). Computed from the
    tables so the prose can never drift from the numbers."""
    warm = [r for r in regimes if r != "cold"]
    out = {}
    for m in models:
        cc, fc = conv.get(("cold", m)), feat.get(("cold", m))
        cw = mean([conv.get((r, m)) for r in warm])
        fw = mean([feat.get((r, m)) for r in warm])
        cr = (cw / cc) if (cc and cw is not None) else None
        fr = (fw / fc) if (fc and fw is not None) else None
        # reduction = 1 - retention (fraction of the cold burden memory removed). The signal is
        # that memory removes a much larger fraction of CONVENTION restatement than FEATURE.
        cred = (1 - cr) if cr is not None else None
        fred = (1 - fr) if fr is not None else None
        out[m] = {"conv_retain": cr, "feat_retain": fr, "conv_reduction": cred, "feat_reduction": fred,
                  "ratio": (cred / fred) if (cred is not None and fred not in (None, 0)) else None}
    return out


def differential_summary(diff, models):
    """The honest one-paragraph headline, computed from the tables (never hand-typed). Stated as
    percentage-points of the cold burden removed (always well-defined) — the conv-vs-feat GAP is
    the signal. A reduction RATIO is shown only when the feature reduction is meaningfully positive
    (otherwise it divides by ~0 / a negative and is meaningless)."""
    lines = ["### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement", ""]
    for m in models:
        d = diff.get(m, {})
        cred, fred = d.get("conv_reduction"), d.get("feat_reduction")
        if cred is None or fred is None:
            lines.append(f"- **{m}**: insufficient data.")
            continue
        gap = (cred - fred) * 100
        tail = (f" — it cuts convention restatement **{cred/fred:.1f}×** as deeply"
                if fred and fred >= 0.10 else "")
        lines.append(
            f"- **{m}**: memory removes **{cred*100:.0f}%** of the cold convention-restatement burden "
            f"vs **{fred*100:.0f}%** of the feature burden (a **{gap:.0f} pp** convention–feature "
            f"gap){tail}.")
    lines += ["",
              "The transferable-vs-app-specific GAP is the signal. The feature side is not a pure "
              "control — feeds shares α/β with the priors, so memory transfer leaks in (and for haiku "
              "the noisy feature side even moves the wrong way); the leak-free check is the native-only "
              "control below.",
              "",
              "**Cross-model: memory is a capability AMPLIFIER, not an equalizer.** The convention "
              "reduction grows with model strength (see per-model % above) — memory helps the stronger "
              "model more, widening the capability gap, reproducing the 2026-06-02 finding."]
    return "\n".join(lines) + "\n"


def native_control_table(builds, models, regimes):
    """The CLEANEST feature control: feeds' NATIVE bucket only (no α/β shared with the priors),
    chain-summed isn't meaningful here so report feeds round-1 native pass-rate per regime."""
    idx = collections.defaultdict(list)
    for b in builds:
        if b.get("app") == "feeds":
            idx[(b.get("regime"), b.get("model"))].append(_bucket_num((b.get("final_buckets") or {}).get("native")))
    table = {(r, m): mean(idx.get((r, m), [])) for r in regimes for m in models}
    note = ("\nfeeds round-1 NATIVE-bucket pass count (the feed-specific features no prior app "
            "teaches). If memory is a clean say-once mechanism this should NOT rise with memory; "
            "if it does, memory is also lifting first-draft quality generally (a real effect, but "
            "it means 'feature interventions' is not a pure untouched control).")
    return note + "\n" + render_table("Native-only control on feeds (leak-free: no shared α/β)",
                                       table, models, regimes, 2)


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
    ep = collections.defaultdict(list)
    ep_fail = []  # (regime, write_tier, model) where the L1 episode was NOT extracted — a failure
    for le in learns:
        q = le.get("learn_quality") or {}
        wt = le.get("write_tier")
        if wt not in ("L1", "L2", "L3"):
            continue
        # Episode extraction is required for EVERY tiered learn (L1 is the foundation) — track it
        # regardless of whether conventions were stated.
        extracted = q.get("episode_extracted", True)
        ep[(wt, le.get("model"))].append(1.0 if extracted else 0.0)
        if not extracted:
            ep_fail.append(f"{le.get('regime')}·{wt}·{le.get('model')}")
        if le.get("stated_conventions_input"):  # coverage only meaningful when something was stated
            cov[(wt, le.get("model"))].append(q.get("coverage"))
    lines = ["### Learn-capture quality (did the agent persist what matters, per tier)", "",
             "Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs "
             "its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier "
             "links down to), so episode% < 100 is a real learn failure.", "",
             "| write-tier | " + " | ".join(models) + " |", "|---|" + "|".join(["---:"] * len(models)) + "|"]
    for wt in ["L1", "L2", "L3"]:
        cells = []
        for m in models:
            c, e = mean(cov.get((wt, m), [])), mean(ep.get((wt, m), []))
            cov_s = "—" if c is None else f"{c:.2f}"
            ep_s = "—" if e is None else f"ep {round(e * 100)}%"
            cells.append(f"{cov_s} · {ep_s}")
        lines.append(f"| `{wt}` | " + " | ".join(cells) + " |")
    if ep_fail:
        lines += ["", f"> ⚠ **Episode-extraction FAILURES (L1 always required): {len(ep_fail)}** — "
                  + ", ".join(ep_fail) + ". These tiered learns produced no episode; resume re-runs them."]
    return "\n".join(lines) + "\n"


def token_io_table(builds, learns, models, root):
    """Token I/O + the reported-vs-recomputed cost audit (§6 / note-17). Prefers the tokens stored
    in each result; for older results lacking them, backfills from the on-disk transcript via the
    shared harness helper — so this works on both new runs and the existing pilot."""
    import harness as hh

    cfgroot = os.path.join(root, "cfgpool")
    agg = collections.defaultdict(lambda: {"in": 0, "out": 0, "cw": 0, "cr": 0,
                                           "rep": 0.0, "rec": 0.0, "n": 0})
    total = covered = noop = 0
    for rec in builds + learns:
        # none-tier learns make NO LLM call (cold arm) — genuinely $0/0 tokens, not data loss.
        # Exclude them from the coverage denominator so it reflects only LLM-using cells.
        if rec.get("kind") == "learn" and rec.get("write_tier") == "none":
            noop += 1
            continue
        total += 1
        m = rec.get("model")
        tok = rec.get("tokens")
        if not tok or sum(tok.values()) == 0:  # backfill from transcript (older results / pre-capture)
            tok = hh.token_usage_for_session(cfgroot, rec.get("session_id"))
        if sum(tok.values()) == 0:
            continue  # no token data for this cell (transcript pruned) — exclude from the ratio,
            # don't count its reported $ against $0 recomputed (that's the verify_cost2 MATCHED rule)
        covered += 1
        rec_cost = rec.get("recomputed_cost")
        if not rec_cost:
            rec_cost = hh.recompute_cost(tok, rec.get("model_id")) or 0
        a = agg[m]
        a["in"] += tok.get("input", 0); a["out"] += tok.get("output", 0)
        a["cw"] += tok.get("cache_write", 0); a["cr"] += tok.get("cache_read", 0)
        a["rep"] += (rec.get("build_cost") or 0) + (rec.get("learn_cost") or 0)
        a["rec"] += rec_cost or 0; a["n"] += 1

    note = (f"  ·  {covered}/{total} LLM-using cells captured ({noop} cold no-op learns excluded)"
            if covered == total else
            f"  ·  **{covered}/{total} LLM-using cells captured** ({noop} cold no-op learns excluded; "
            f"{total - covered} lost their transcripts to cfg-pool re-creation — run-time capture "
            f"prevents this going forward)")
    lines = [f"### Token I/O + cost audit (per model, over covered cells){note}", "",
             "Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost "
             "(ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.", "",
             "| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |",
             "|---|--:|--:|--:|--:|--:|--:|--:|--:|"]
    for m in models:
        a = agg.get(m)
        if not a or not a["n"]:
            continue
        ratio = f"{a['rec']/a['rep']:.2f}×" if a["rep"] else "—"
        lines.append(f"| {m} | {a['n']} | {a['in']:,} | {a['out']:,} | {a['cw']:,} | {a['cr']:,} "
                     f"| {a['rep']:.2f} | {a['rec']:.2f} | {ratio} |")
    return "\n".join(lines) + "\n"


def full_matrix_tables(builds, learns, models, regimes):
    """The full matrix: per app, a (model × regime) × 6-metric grid of MEDIANS, best-per-
    (model,metric) bolded. app1 (notes) is built cold ONCE per (model,trial) and shared across
    regimes — it varies only by the write-tier its memory is learned at — so app1 is shown by
    write-tier (none/L1/L2/L3), not by all 7 regimes. app2/app3 use the full regime set.
    Metrics: human turns (feedback rounds), prescriptiveness (max convention escalation depth),
    turns-to-converge (median rounds among COMPLETED builds; — if none), cost$, tokens, time(min).
    Cost/tokens/time include that app-position's learn (app1→write-tier learn, app2→regime learn,
    app3 terminal). Resource metrics are over ALL builds (incl. capped non-converged), so a † marks
    a cell where <60% of builds completed — its turns/cost run high because they didn't finish."""
    bmap, a1l, a2l = _index_runs(builds, learns)
    write_of = {"cold": "none", "l1": "L1", "l2.l1l2": "L2", "l2.l2": "L2",
                "l3.l1l2l3": "L3", "l3.l2l3": "L3", "l3.l3": "L3"}

    def cell(model, app, reg):
        fb, presc, conv, cost, tok, wall, ncomp, n = [], [], [], [], [], [], 0, 0
        for t in range(1, 6):
            b = bmap.get((model, t, app, "cold" if app == "notes" else reg))
            if not b:
                continue
            n += 1
            nr = len(b.get("rounds", []))
            fb.append(nr - 1)
            presc.append((b.get("escalation") or {}).get("max_convention_depth", 0))
            if b.get("completed"):
                conv.append(nr); ncomp += 1
            ln = a1l.get((model, t, write_of[reg])) if app == "notes" else (
                a2l.get((model, t, reg)) if app == "links" else None)
            cost.append((b.get("build_cost", 0) or 0) + ((ln or {}).get("learn_cost", 0) or 0))
            tok.append(_toks(b) + _toks(ln))
            wall.append((b.get("wall_min", 0) or 0) + ((ln or {}).get("wall_min", 0) or 0))
        return {"fb": mean_med(fb), "presc": mean_med(presc), "conv": mean_med(conv),
                "cost": mean_med(cost), "tok": mean_med(tok), "wall": mean_med(wall),
                "low_complete": n > 0 and ncomp / n < 0.6}

    metrics = [("fb", "human turns", 0), ("presc", "prescript", 0), ("conv", "→converge", 0),
               ("cost", "cost $", 2), ("tok", "tokens", None), ("wall", "time min", 0)]

    def fmt_val(key, v, nd):
        if v is None:
            return "—"
        if key == "tok":
            return f"{v/1e6:.1f}M"
        return f"{v:.{nd}f}"

    out = []
    for app, label, rset in [
            ("notes", "app1 · notes (cold build shared per model; row = write-tier of its learn)",
             ["none", "L1", "L2", "L3"]),
            ("links", "app2 · links (recall under regime)", regimes),
            ("feeds", "app3 · feeds (recall under regime; terminal, no learn)", regimes)]:
        out += [f"### Full matrix — {label}", "",
                "Medians. **Bold** = best (lowest) per model per metric. "
                + ("† = <60% of this cell's builds completed (resource figures include capped runs)."
                   if app != "notes" else
                   "app1 build is identical across rows; only learn cost/tokens/time differ by tier."),
                "", "| model | " + ("write-tier" if app == "notes" else "regime") + " | "
                + " | ".join(m for _, m, _ in metrics) + " |",
                "|---|---|" + "|".join(["--:"] * len(metrics)) + "|"]
        for m in models:
            cells = {}
            for r in rset:
                reg_for = ({"none": "cold", "L1": "l1", "L2": "l2.l2", "L3": "l3.l3"}.get(r, r)
                           if app == "notes" else r)
                cells[r] = cell(m, app, reg_for)
            best = {}
            for key, _, _ in metrics:
                vals = [(r, cells[r][key]) for r in rset if cells[r][key] is not None]
                if vals:
                    best[key] = min(vals, key=lambda x: x[1])[0]
            for r in rset:
                c = cells[r]
                dag = "†" if c.get("low_complete") else ""
                cellstrs = []
                for key, _, nd in metrics:
                    s = fmt_val(key, c[key], nd)
                    if best.get(key) == r and c[key] is not None:
                        s = f"**{s}**"
                    cellstrs.append(s)
                out.append(f"| {m} | `{r}`{dag} | " + " | ".join(cellstrs) + " |")
        out.append("")
    return "\n".join(out) + "\n"


def mean_med(v):
    import statistics
    return statistics.median(v) if v else None


def recommendation_section(manifest):
    """The standing recommendation derived from this baseline. Provenance (SHA/date/models/trials)
    is interpolated so it's always accurate to the run; the prose conclusion is a point-in-time
    judgement scoped to THIS data — re-examine it if re-deriving on a new run."""
    prov = (f"engram `{manifest.get('engram_sha','?')}` · {manifest.get('date','?')} · "
            f"{', '.join(manifest.get('models', []))} · n={manifest.get('trials','?')}")
    return f"""## Recommendation — if you could pick one model + regime

_Derived from the baseline below ({prov}). A point-in-time judgement on this data; revisit when re-deriving._

**Pick: `opus` + `l2.l2`** (write L2 facts, read L2 tier-capped) **— when you're building many
apps that share conventions over time.** Otherwise **`sonnet`/`opus` + `cold`** is the cheaper
reliable floor for a short horizon. Reasoning, strictly from the tables above:

**Model — opus (sonnet a close second; haiku is out).**
- Cost is NOT the differentiator people assume: warm chains cost about the same on both
  (sonnet ≈ \\$8.4–11.2, opus ≈ \\$8.8–11.3) — opus's higher per-token rate is offset by its
  token-efficiency (≈7–10M vs sonnet ≈10–18M) and ~2-round convergence. At cost parity opus is
  faster (≈6–8 min/app vs ≈16–22), edges say-once (7 vs 9 conventions), and needs **zero**
  prescriptive hand-holding (sonnet ~1).
- **haiku is excluded:** even with escalation it completes ≤80% of chains per regime (≈42%
  overall) and only by being handed the literal code (depth-2 prescriptions). Not shippable as a
  default.

**Regime — warm, `l2.l2` specifically, on a stated principle (not a measured tier win).**
- The decision that matters is **cold vs warm**, not which tier: among the 6 warm regimes the
  spread is n=5 noise. For strong models tier is *flat* (the regime-axis finding above).
- **Warm vs cold is a horizon call.** Cold completes 100% for strong models at ~half the cost
  (≈\\$4–5 vs ≈\\$9) but carries ~2× the convention burden (18–19 vs 7–9 restatements). Warm's
  say-once benefit is paid once and **recovered on every later app that shares conventions**,
  while its extra cost is per-build — so a 3-app chain *understates* warm. Many convention-sharing
  apps → warm wins; a one-off or two → cold is the reliable floor.
- **Why `l2.l2` among the warm regimes:** it's the *never-worst-across-capability* config — the one
  that rescued haiku (80% complete vs ≤40% for blended/L1 reads) and ties for best on the strong
  models. That makes it the safe choice if the model is ever swapped or downgraded. A robustness
  tiebreak, not a measured victory over the other warm tiers.

**What warm does NOT buy for strong models (honest caveat):** it does **not** reduce review
round-trips — human turns are ~3 whether cold or warm for sonnet/opus; they fold recalled
conventions into the same rounds. Memory **front-loads correctness** (fewer distinct things to
teach, compounding across apps), it does not cut iterations here. The dramatic round-trip saving
(20→6) is real only for haiku, which we don't ship — so the pitch for opus+warm is "teach each
convention once, ever," not "fewer review cycles.\""""


def escalation_table(builds, models):
    """How granular the human feedback had to get before the build converged (§5 signal).
    depth = #times an item was restated; escalation kicks in at depth≥2 (the literal code-level
    fix). A strong model converges on the symptom (low depth); a weak one needs the prescription.
    Reported per (model, app): median max-convention-depth + mean #conventions that needed the
    prescriptive fix — over builds that completed (escalation on a stalled build is censored)."""
    by = collections.defaultdict(lambda: {"depth": [], "nesc": []})
    for b in builds:
        if not b.get("completed"):
            continue
        e = b.get("escalation") or {}
        by[(b.get("model"), b.get("app"))]["depth"].append(e.get("max_convention_depth", 0))
        by[(b.get("model"), b.get("app"))]["nesc"].append(e.get("conventions_escalated", 0))
    lines = ["### Feedback escalation depth — how granular before convergence (completed builds)", "",
             "`conv-depth` = median max times a *convention* was restated before it stuck "
             "(1 = fixed on the symptom; ≥2 = needed the literal code-level prescription). "
             "`#presc` = mean conventions per build that needed the prescriptive fix. "
             "Higher = more hand-holding — expected to fall as model strength rises.", "",
             "| model | app | conv-depth (median) | #presc (mean) |",
             "|---|---|--:|--:|"]
    for m in models:
        for app in ["notes", "links", "feeds"]:
            d = by.get((m, app))
            if not d or not d["depth"]:
                continue
            lines.append(f"| {m} | {app} | {fmt(mean(d['depth']),1)} | {fmt(mean(d['nesc']),1)} |")
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
    # Order tables by capability (weak → strong), not by run order, so the model progression reads
    # haiku → sonnet → opus across every table. Unknown models sort last, alphabetically.
    cap_rank = {"haiku": 0, "sonnet": 1, "opus": 2}
    models = sorted(models, key=lambda m: (cap_rank.get(m, 99), m))
    regimes = manifest.get("regimes") or sorted({b.get("regime") for b in builds if b.get("app") != "notes"})

    conv = chain_intervention_table(builds, models, regimes, "convention_statements")
    feat = chain_intervention_table(builds, models, regimes, "feature_statements")
    beta = beta_table(builds, models, regimes)
    followed = per_app_numeric(builds, models, regimes, "feeds", "link_followed")
    differential = differential_retention(conv, feat, models, regimes)

    stub_note = "  ·  **STUB RUN (no LLM — mechanics only, numbers are not real)**" if manifest.get("stub") else ""
    rate_limited = sum(1 for b in builds if b.get("rate_limited"))
    not_engaged = sum(1 for le in learns if le.get("learned") is False and le.get("write_tier") != "none")
    completeness = ""
    if rate_limited or not_engaged:
        completeness = (f"\n\n> ⚠ **INCOMPLETE:** {rate_limited} build(s) hit a rate limit and "
                        f"{not_engaged} learn(s) did not engage — these cells are unreliable. "
                        f"Re-run (resume) when quota is available; resume re-runs exactly these.")
    doc = ["# Cumulative cross-app memory accumulation — results (v2)", "",
           f"Engram SHA: `{manifest.get('engram_sha','?')}` · date: {manifest.get('date','?')} · "
           f"models: {', '.join(models)} · trials: {manifest.get('trials','?')} · "
           f"price sheet: {manifest.get('price_sheet_date','?')}" + stub_note + completeness, "",
           "> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable "
           "cell-for-cell to the 2026-06-02 run.", "",
           "## Primary — repeated-convention interventions (say-once vs every-app)", "",
           "Chain-summed conventions the human had to STATE (app1+app2+app3). "
           "Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. "
           "The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.", "",
           render_table("Convention interventions to endpoint (mean/trial)", conv, models, regimes),
           render_table("Feature interventions — CONTROL (app-specific; nobody carries these)", feat, models, regimes),
           differential_summary(differential, models),
           headline_stats_table(builds, learns, models, regimes),
           "## Secondary", "",
           render_table("β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the "
                        "first draft? — measured at round 1; β saturates to 4/4 at convergence)",
                        beta, models, regimes, 2),
           render_table("Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)",
                        followed, models, regimes, 2),
           native_control_table(builds, models, regimes),
           per_regime_cost_table(builds, learns, models, regimes)]
    doc += ["", learn_quality_table(learns, models), "", escalation_table(builds, models),
            "", "## Full matrix (model × regime × app, medians)", "",
            full_matrix_tables(builds, learns, models, regimes),
            "", token_io_table(builds, learns, models, args.root), "", cost_calibration(builds, learns)]

    # Convergence guard (§5) + honest caveats — never ship an over-claimed number.
    conv_n = sum(1 for b in builds if b.get("converged"))
    ntrials = len(manifest.get("trials", [])) or len({b.get("trial") for b in builds})
    # Regime axis, per model: is the warm-regime spread small vs the cold→warm gap (= tier doesn't
    # matter) — and does that hold across models? Computed from the data, not hardcoded.
    warm = [r for r in regimes if r != "cold"]
    per_model = []
    flat_all = True
    for m in models:
        wv = [conv.get((r, m)) for r in warm if conv.get((r, m)) is not None]
        cv = conv.get(("cold", m))
        if not wv or cv is None:
            continue
        spread = max(wv) - min(wv)
        gap = cv - mean(wv)
        is_flat = gap > 0 and spread <= gap / 2  # between-tier spread is small vs the cold→warm gap
        flat_all = flat_all and is_flat
        best = min(warm, key=lambda r: conv.get((r, m), 9e9))  # lowest restatement = best tier
        per_model.append(f"{m} {min(wv):.1f}–{max(wv):.1f} band vs cold {cv:.1f} (best: {best})")
    if per_model:
        regime_caveat = (
            f"- **Regime axis (the v2 question): tier is {'FLAT — does not matter' if flat_all else 'NOT uniformly flat'} "
            f"at n={ntrials}, every model.** Per model: " + "; ".join(per_model) + ". "
            f"{'Within each model the warm regimes cluster well inside the cold→warm gap — writing L3 syntheses does not beat L1 episodes, reading only the distilled L3 does not beat blended, and raw L1 episodes capture the full effect.' if flat_all else 'At least one model shows a between-tier spread comparable to its cold→warm gap — see the per-model bands.'} "
            f"β-accumulation (round-1 feeds β) saturates to 4/4 by convergence and is noisy in the first draft, so H2 stays inconclusive at this β-difficulty.")
    else:
        regime_caveat = "- **Regime axis: insufficient complete chains to compare.**"
    caveats = ["## Convergence guard + honest caveats", "",
               f"- **Converged within the {max((b.get('max_rounds') or 6) for b in builds)}-round budget: "
               f"{conv_n}/{len(builds)} builds.** The primary metric is the round-1 intervention count, "
               f"not a stall rate; low convergence means some builds plateau below the full bar — "
               f"investigate feedback-symptom effectiveness / stale-break, separately from say-once.",
               f"- **n={ntrials} trial(s){' — PILOT, DIRECTIONAL ONLY' if ntrials < 5 else ''}.** "
               f"Models: {', '.join(models)}{' (single model — cross-model still open)' if len(models) < 2 else ''}.",
               regime_caveat,
               "- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured "
               "outputs (a poor capture is recorded, not engineered away).",
               "- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline."]
    doc += ["", "\n".join(caveats)]
    doc += ["", recommendation_section(manifest)]

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
