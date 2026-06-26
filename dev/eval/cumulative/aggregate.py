#!/usr/bin/env python3
"""Aggregate cumulative-accumulation results into the §5 headline tables.

PRIMARY metric — repeated-convention interventions (the say-once signal): per build we
record how many transferable conventions had to be STATED (round-1 ARCH fails, since §4
feeds back all gaps). Summed across the app1->app2->app3 chain, memory should approach
|conventions| stated once; no-memory approaches |conventions| x 3. App-specific FEATURE
interventions are the control — memory should not move them.

Reads the per-op result JSONs written by harness.py (build/learn) under <root>/results/.
Output goes to --out (default: <root>/results-agg.md in the run workspace) and stdout.
No files are written into the source tree.

Usage: python3 aggregate.py [--root /tmp/cummatrix] [--out /tmp/cummatrix/results-agg.md]
"""
import argparse, collections, glob, json, os, random as _random, statistics


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


def bootstrap_ci(xs, alpha=0.05, n_boot=2000):
    """Bootstrap (mean, lo, hi) for a 1-alpha CI. Returns (mean, lo, hi)."""
    xs = [x for x in xs if x is not None]
    if not xs:
        return (None, None, None)
    rng = _random.Random(0)
    n = len(xs)
    boot_means = []
    for _ in range(n_boot):
        sample = [xs[rng.randrange(n)] for _ in range(n)]
        boot_means.append(statistics.mean(sample))
    boot_means.sort()
    lo_idx = int((alpha / 2) * n_boot)
    hi_idx = int((1 - alpha / 2) * n_boot) - 1
    return (statistics.mean(xs), boot_means[lo_idx], boot_means[hi_idx])


def noise_floor(xs):
    """95% CI half-width of within-group variance (warm-vs-warm contrast)."""
    _, lo, hi = bootstrap_ci(xs, alpha=0.05, n_boot=2000)
    if lo is None or hi is None:
        return 0.0
    return (hi - lo) / 2.0


def gap_label(gap, floor):
    """Label a cold-vs-warm gap relative to the noise floor."""
    if abs(gap) < floor:
        return "underpowered"
    return "significant"


def axis_ci_table(builds, learns, models, regimes):
    """Per (regime, model): bootstrap mean ± 95% CI for each C-axis metric."""
    metrics = [
        ("recall_s", "recall_s"),
        ("build_s", "build_s"),
        ("learn_s", "learn_s"),
        ("build_cost", "axis_c2_cost_usd"),
        ("recall_cost", "axis_c2_recall_cost"),
        ("convention_statements", "axis_c3_interventions"),
        ("feature_statements", "feature_statements"),
    ]
    lines = ["### Axis CI table (bootstrap 95% CI per regime × model)", ""]
    header = "| regime | model | " + " | ".join(label for _, label in metrics) + " |"
    sep = "|---|---|" + "|".join(["---:"] * len(metrics)) + "|"
    lines += [header, sep]

    for r in regimes:
        for m in models:
            row_vals = []
            xs_warm = None
            for field, label in metrics:
                xs = [b.get(field) for b in builds
                      if b.get("regime") == r and b.get("model") == m and b.get(field) is not None]
                if not xs:
                    row_vals.append("—")
                    continue
                mn, lo, hi = bootstrap_ci(xs)
                if mn is None:
                    row_vals.append("—")
                    continue
                hw = (hi - lo) / 2.0 if (lo is not None and hi is not None) else 0
                # Label cold-vs-warm gap relative to floor (compare to cold same model)
                cold_xs = [b.get(field) for b in builds
                           if b.get("regime") == "cold" and b.get("model") == m and b.get(field) is not None]
                tag = ""
                if r != "cold" and cold_xs:
                    cold_mean = mean(cold_xs)
                    warm_mean = mn
                    if cold_mean is not None and warm_mean is not None:
                        gap = abs(cold_mean - warm_mean)
                        floor = noise_floor(xs)
                        if floor > 0 and gap < floor:
                            tag = " ⚠underpowered"
                row_vals.append(f"{mn:.2f}±{hw:.2f}{tag}")
            lines.append(f"| `{r}` | {m} | " + " | ".join(row_vals) + " |")
    lines.append("")
    return "\n".join(lines) + "\n"


def fmt(x, nd=1):
    return "—" if x is None else f"{x:.{nd}f}"


def chain_intervention_table(builds, models, regimes, key):
    """Per (regime, model): mean over trials of the chain-summed `key` count
    (convention_statements or feature_statements) across app1+app2+app3. app1 (notes)
    is the shared cold build per (model, trial)."""
    idx = collections.defaultdict(dict)
    # app1 (notes) is the shared COLD build per (model, trial). Anchor BOTH the cold and warm chains
    # to the cold-app1 value — the SAME anchoring headline_stats/chain_rows uses (bmap[..,"cold"]).
    # The old code appended every `notes` build (any regime) into app1_by_mt and averaged, which
    # blended a warm-app1 restatement into the cold baseline and made the two tables disagree
    # (cold 20.5/warm 13.5 here vs cold 21/warm 14 in headline) — retro Bug 2.
    app1_cold_by_mt = collections.defaultdict(list)
    for b in builds:
        m, r, t, a = b.get("model"), b.get("regime"), b.get("trial"), b.get("app")
        v = b.get(key, 0) or 0
        if a == "notes":
            if r == "cold":
                app1_cold_by_mt[(m, t)].append(v)
        else:
            idx[(m, r, t)][a] = v

    table = {}
    for r in regimes:
        for m in models:
            trials = sorted({t for (mm, rr, t) in idx if mm == m and rr == r}) or \
                     sorted({t for (mm, t) in app1_cold_by_mt if mm == m})
            totals = []
            for t in trials:
                app1 = mean(app1_cold_by_mt.get((m, t), [])) or 0
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


WRITE_TIER = {
    "cold": "none",
    "real.full": "skill",
}


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
            # per-build convergence (retro: the all-3-product `converged` above collapses to ~0% under
            # a ~50% per-build stall rate and reads as "model can't converge" when individual builds
            # converge fine — report build-level fraction instead).
            "conv_builds": sum(1 for x in builds3 if x.get("converged")),
            "n_builds": len(builds3),
            # learn_cost = separate-op learns (cold-tier learns) PLUS each build's IN-SESSION /learn
            # cost. real.full runs /learn inside the build session, so `ln` is empty and the
            # in-session cost lives in build["learn"]["cost"] — omitting it understated warm cost
            # ~6% (retro Bug 3). build_cost is the feedback rounds only and does NOT include it.
            "learn_cost": (sum(x.get("learn_cost", 0) or 0 for x in ln)
                           + sum((x.get("learn") or {}).get("cost", 0) or 0 for x in builds3)),
            "build_cost": sum(x.get("build_cost", 0) or 0 for x in builds3),
            # $METER: recall_cost is billed separately from build_cost; keep it as its own row field so
            # every chain total (headline, per-regime) can fold it back in and stay whole (note 66).
            "recall_cost": sum(x.get("recall_cost", 0) or 0 for x in builds3),
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
             "| model | arm | conv-restate | review | build-conv% | wall min | tokens | $ |",
             "|---|---|--:|--:|--:|--:|--:|--:|"]
    for m in models:
        for arm, regs in [("cold", ["cold"]), ("warm", warm)]:
            rows = [r for reg in regs for r in chain_rows(builds, learns, m, reg)]
            if not rows:
                continue
            cv = 100 * sum(r["conv_builds"] for r in rows) / sum(r["n_builds"] for r in rows)
            tot = lambda k: mean([r[k] for r in rows])
            lines.append(f"| {m} | {arm} | {tot('conv_restate'):.1f} | {tot('review_turns'):.1f} | "
                         f"{cv:.0f}% | {tot('wall'):.0f} | {tot('tokens')/1e6:.1f}M | "
                         f"{tot('learn_cost')+tot('build_cost')+tot('recall_cost'):.2f} |")
    return "\n".join(lines) + "\n"


SEED_APPS = ["notes"]            # app1 — seeds memory at full cost, no prior memory to recall
PAYBACK_APPS = ["links", "feeds"]  # apps 2..N — where memory pays back


def amortized_economics_table(builds, models, regimes):
    """Seed (app1) vs payback (apps 2..N) economics, per axis.

    app1 SEEDS memory at full recall+learn cost with NO prior memory to recall — a one-time
    investment. Apps 2..N are where memory PAYS BACK. Chain totals/per-trial averages blend the
    seed's one-time cost into every app and misread it as a per-app penalty (this is the cost/time
    analogue of the convention metric's app1 cold-anchoring). This table separates the two so the
    amortized economics are legible. Δ% = warm vs cold; negative = memory cheaper/faster/fewer."""
    import statistics
    warm = [r for r in regimes if r != "cold"]

    axes = [("convention restatements", "count", "convention_statements", False),
            ("feedback rounds", "rounds", "total_rounds", False),
            ("build time", "s", "build_s", False),
            ("recall time", "s", "recall_s", False),
            ("learn time", "s", "learn_s", False),
            ("cost", "USD", "axis_c2_cost_usd", False),
            ("recall cost", "USD", "axis_c2_recall_cost", False),
            ("tokens", "Mtok", "tokens", True)]
    lines = ["### Amortized economics — seed (app1·notes) vs payback (apps 2–3·links+feeds)", "",
             "app1 seeds memory at full recall+learn cost with no prior memory to recall — a one-time "
             "investment. Apps 2–3 are where memory pays back. Chain totals blend the seed cost into "
             "every app and misread it as a per-app penalty; this table separates them so the "
             "amortized economics are legible. Δ% = warm vs cold; negative = memory better. The seed "
             "cost is paid once per chain — the longer the chain, the more the payback dominates.", ""]
    for m in models:
        mb = [b for b in builds if b.get("model") == m]
        if not mb:
            continue
        lines += [f"**{m}**", "",
                  "| segment | axis | unit | cold | warm | Δ | Δ% |",
                  "|---|---|---|--:|--:|--:|--:|"]
        for segname, appset in [("seed (app1)", SEED_APPS), ("payback (2–3)", PAYBACK_APPS)]:
            def g(regset, key, tok=False):
                return sum(
                    (statistics.mean(v) if (v := [(_toks(b) if tok else (b.get(key, 0) or 0))
                     for b in mb if b.get("app") == a and b.get("regime") in regset]) else 0.0)
                    for a in appset)
            for lab, unit, key, tok in axes:
                c = g(["cold"], key, tok); w = g(warm, key, tok)
                if tok:
                    c /= 1e6; w /= 1e6
                d = w - c; pct = 100 * d / c if c else 0
                lines.append(f"| {segname} | {lab} | {unit} | {c:.2f} | {w:.2f} | {d:+.2f} | {pct:+.0f}% |")
            ct = g(["cold"], "recall_s") + g(["cold"], "build_s") + g(["cold"], "learn_s")
            wt = g(warm, "recall_s") + g(warm, "build_s") + g(warm, "learn_s")
            d = wt - ct; pct = 100 * d / ct if ct else 0
            lines.append(f"| {segname} | **total active time** | s | {ct:.0f} | {wt:.0f} | {d:+.0f} | {pct:+.0f}% |")
        lines.append("")
    return "\n".join(lines) + "\n"


def per_regime_cost_table(builds, learns, models, regimes):
    """Per-regime cost/token/convergence breakdown — learn$ from in-session /learn vs build$."""
    lines = ["### Cost & convergence by regime (mean per trial)",
             "",
             "`learn$` = in-session /learn cost (real.full only); `build$` dominated by "
             "feedback round-count (convergence).", ""]
    for m in models:
        lines += [f"**{m}**", "",
                  "| regime | learn$ | build$ | total$ | wall | tokens | build-conv% |",
                  "|---|--:|--:|--:|--:|--:|--:|"]
        for reg in regimes:
            rows = chain_rows(builds, learns, m, reg)
            if not rows:
                continue
            ln, bd = mean([r["learn_cost"] for r in rows]), mean([r["build_cost"] for r in rows])
            rc = mean([r.get("recall_cost", 0) or 0 for r in rows])  # $METER: fold billed recall into total$
            cv = 100 * sum(r["conv_builds"] for r in rows) / sum(r["n_builds"] for r in rows)
            lines.append(f"| `{reg}` | {ln:.2f} | {bd:.2f} | {ln+bd+rc:.2f} | "
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
                        cost += b.get("recall_cost", 0) or 0  # $METER: billed recall is part of op cost
                        cost += (b.get("learn") or {}).get("cost", 0) or 0  # in-session /learn (Bug 3)
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
    (otherwise it divides by ~0 / a negative and is meaningless).

    Guard: only draws conclusions for models/regimes present in the data."""
    lines = ["### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement", ""]
    data_models = [m for m in models if diff.get(m) is not None]
    if not data_models:
        lines.append("_Insufficient data to draw conclusions — run the eval first._")
        return "\n".join(lines) + "\n"
    for m in data_models:
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
              "control — feeds shares α/β with the priors, so memory transfer leaks in; the leak-free "
              "check is the native-only control below."]
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


def notes_written_table(builds, models, regimes):
    """Per (regime, model): mean notes_written (fact+feedback) and crystallizations_at_recall
    after the learn step. These are the modern replacements for the tier-keyed learn_quality_table
    (v3 — no episodes/tiers)."""
    idx_notes = collections.defaultdict(list)
    idx_cryst = collections.defaultdict(list)
    idx_kind = collections.defaultdict(lambda: {"fact": [], "feedback": []})
    for b in builds:
        r, m = b.get("regime"), b.get("model")
        if b.get("notes_written") is not None:
            idx_notes[(r, m)].append(b["notes_written"])
        if b.get("crystallizations_at_recall") is not None:
            idx_cryst[(r, m)].append(b["crystallizations_at_recall"])
        kd = b.get("learn_kind_breakdown") or {}
        if kd:
            idx_kind[(r, m)]["fact"].append(kd.get("fact", 0))
            idx_kind[(r, m)]["feedback"].append(kd.get("feedback", 0))
    lines = ["### Learn output (v3: notes written + lazy crystallizations at recall)", "",
             "notes_written = fact+feedback notes in vault after /learn. "
             "crystallizations = engram learn/amend calls fired by /recall during build. "
             "cold should show 0 for all.", "",
             "| regime | model | notes_written | crystallizations | fact notes | feedback notes |",
             "|---|---|--:|--:|--:|--:|"]
    for r in regimes:
        for m in models:
            nw = mean(idx_notes.get((r, m), []))
            cr = mean(idx_cryst.get((r, m), []))
            kd = idx_kind.get((r, m), {})
            fa = mean(kd.get("fact", []))
            fb = mean(kd.get("feedback", []))
            lines.append(f"| `{r}` | {m} | {fmt(nw, 1)} | {fmt(cr, 1)} | {fmt(fa, 1)} | {fmt(fb, 1)} |")
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
        if rec.get("kind") == "learn" and rec.get("write_mode") == "none":
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
        a["rep"] += (rec.get("build_cost") or 0) + (rec.get("learn_cost") or 0) + (rec.get("recall_cost") or 0)
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
    (model,metric) bolded. Each regime (cold, real.full) runs its own 3-app chain; app1/app2/app3
    are shown per regime.
    Metrics: human turns (feedback rounds), prescriptiveness (max convention escalation depth),
    turns-to-converge (median rounds among COMPLETED builds; — if none), cost$, tokens, time(min).
    Cost/tokens/time include that app-position's in-session learn (real.full only).
    Resource metrics are over ALL builds (incl. capped non-converged), so a † marks
    a cell where <60% of builds completed — its turns/cost run high because they didn't finish."""
    bmap, a1l, a2l = _index_runs(builds, learns)
    write_of = {
        "cold": "none",
        "real.full": "skill",
        "none": "none",
    }

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
            ln = a1l.get((model, t, write_of.get(reg, reg))) if app == "notes" else (
                a2l.get((model, t, reg)) if app == "links" else None)
            cost.append((b.get("build_cost", 0) or 0) + (b.get("recall_cost", 0) or 0)
                        + ((ln or {}).get("learn_cost", 0) or 0))
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
            ("notes", "app1 · notes (cold vs real.full)", regimes),
            ("links", "app2 · links (recall under regime)", regimes),
            ("feeds", "app3 · feeds (recall under regime; terminal, no learn)", regimes)]:
        out += [f"### Full matrix — {label}", "",
                "Medians. **Bold** = best (lowest) per model per metric. "
                "† = <60% of this cell's builds completed (resource figures include capped runs).",
                "", "| model | regime | "
                + " | ".join(m for _, m, _ in metrics) + " |",
                "|---|---|" + "|".join(["--:"] * len(metrics)) + "|"]
        for m in models:
            cells = {}
            for r in rset:
                reg_for = r  # real-skill regimes: app1 uses same regime key (no separate learn op)
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


def recommendation_section(manifest, builds, models, regimes):
    """Point-in-time recommendation, strictly derived from the data present.
    Guard: only references models and regimes actually present in the results."""
    prov = (f"engram `{manifest.get('engram_sha','?')}` · {manifest.get('date','?')} · "
            f"{', '.join(models)} · n={manifest.get('trials','?')}")
    # Determine which regime had lower mean convention statements
    warm = [r for r in regimes if r != "cold"]
    if not warm or not models:
        return "## Recommendation\n\n_Insufficient data — run the eval first._\n"
    lines = [f"## Recommendation", "",
             f"_Derived from: {prov}. A point-in-time judgement on this data; revisit when re-deriving._", "",
             "**The key question is cold vs real.full**, not model choice within a regime. "
             "Read the convention-interventions table: if memory removes a substantial fraction of the "
             "cold burden, real.full is worth the extra recall+learn overhead per build. "
             "If the gap is within the noise floor, cold is the reliable cheaper default.", "",
             "Cross-model conclusions and regime-tier comparisons require data from multiple models "
             "and regimes — do not draw them from a single-model or single-regime run."]
    return "\n".join(lines) + "\n"


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
        bcost[(b.get("model"), b.get("app"))].append((b.get("build_cost", 0) or 0)
                                                      + (b.get("recall_cost", 0) or 0))
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
    ap.add_argument("--out", default=None)  # default is set after --root is parsed (see below)
    args = ap.parse_args()
    if args.out is None:
        args.out = os.path.join(args.root, "results-agg.md")

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
    not_engaged = sum(1 for le in learns if le.get("learned") is False and le.get("write_mode") != "none")
    completeness = ""
    if rate_limited or not_engaged:
        completeness = (f"\n\n> ⚠ **INCOMPLETE:** {rate_limited} build(s) hit a rate limit and "
                        f"{not_engaged} learn(s) did not engage — these cells are unreliable. "
                        f"Re-run (resume) when quota is available; resume re-runs exactly these.")
    doc = ["# Cumulative cross-app memory accumulation — results (v3: cold vs real.full)", "",
           f"Engram SHA: `{manifest.get('engram_sha','?')}` · date: {manifest.get('date','?')} · "
           f"models: {', '.join(models)} · trials: {manifest.get('trials','?')} · "
           f"price sheet: {manifest.get('price_sheet_date','?')}" + stub_note + completeness, "",
           "> v3 design: two regimes (cold, real.full). No tiers/episodes/eager-L2. "
           "Memory = chunks (engram ingest) + notes (engram learn fact|feedback). "
           "Recall = /recall skill → unified clustering → lazy crystallization.", "",
           "## Primary — repeated-convention interventions (say-once vs every-app)", "",
           "Chain-summed conventions the human had to STATE (app1+app2+app3). "
           "Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. "
           "The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.", "",
           render_table("Convention interventions to endpoint (mean/trial)", conv, models, regimes),
           render_table("Feature interventions — CONTROL (app-specific; nobody carries these)", feat, models, regimes),
           differential_summary(differential, models),
           headline_stats_table(builds, learns, models, regimes),
           amortized_economics_table(builds, models, regimes),
           "## Secondary", "",
           render_table("β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the "
                        "first draft? — measured at round 1; β saturates to 4/4 at convergence)",
                        beta, models, regimes, 2),
           render_table("Link-following rate on feeds (mean, real.full only)",
                        followed, models, regimes, 2),
           native_control_table(builds, models, regimes),
           per_regime_cost_table(builds, learns, models, regimes)]
    doc += ["", notes_written_table(builds, models, regimes), "", escalation_table(builds, models),
            "", "## Full matrix (model × regime × app, medians)", "",
            full_matrix_tables(builds, learns, models, regimes),
            "", token_io_table(builds, learns, models, args.root), "",
            axis_ci_table(builds, learns, models, regimes),
            "", cost_calibration(builds, learns)]

    # Convergence guard + honest caveats — never ship an over-claimed number.
    conv_n = sum(1 for b in builds if b.get("converged"))
    ntrials = len(manifest.get("trials", [])) or len({b.get("trial") for b in builds})
    warm = [r for r in regimes if r != "cold"]
    # cold→warm gap: is the gap above the noise floor?
    gap_notes = []
    for m in models:
        cold_v = conv.get(("cold", m))
        for r in warm:
            warm_v = conv.get((r, m))
            if cold_v is not None and warm_v is not None:
                gap = cold_v - warm_v
                xs_warm = [b.get("convention_statements") for b in builds
                           if b.get("regime") == r and b.get("model") == m
                           and b.get("convention_statements") is not None]
                floor = noise_floor(xs_warm)
                label = gap_label(gap, floor)
                gap_notes.append(f"{m}/{r}: gap={gap:.1f} floor={floor:.1f} ({label})")
    caveats = ["## Convergence guard + honest caveats", "",
               f"- **Converged within the {max((b.get('max_rounds') or 8) for b in builds)}-round budget: "
               f"{conv_n}/{len(builds)} builds.**",
               f"- **n={ntrials} trial(s){' — PILOT, DIRECTIONAL ONLY' if ntrials < 5 else ''}.** "
               f"Models present: {', '.join(models)}{' (single model — cross-model still open)' if len(models) < 2 else ''}.",
               "- cold→warm gap vs noise floor: " + ("; ".join(gap_notes) if gap_notes else "insufficient data"),
               "- Learn is agent-driven (fact/feedback notes, no tiers/episodes); notes_written and "
               "crystallizations_at_recall are measured outputs.",
               "- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.",
               "- **Guard:** conclusions in this report reference only the models/regimes present in the data above."]
    doc += ["", "\n".join(caveats)]
    doc += ["", recommendation_section(manifest, builds, models, regimes)]

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
