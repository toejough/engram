"""Pure scoring + verdict for the question-anchored delivery eval (no I/O, unit-tested in test_qanchor).

tally   : rows [{arm, hit, ...}] -> per-arm {n, hits, rate, sigma(binomial)}.
verdict : apply the decision gate — headroom first (none-floor), then the 2σ separation on B−A.

The 2σ bar and the headroom gate encode two standing lessons:
  - a gap below the noise floor is UNDERPOWERED ("can't distinguish"), never a confirmed tie;
  - if the none-floor is high, cold already applies the principle => the run is ceiling-limited and a
    null A-vs-B result says nothing about the lever (feedback_gap_below_noise_is_underpowered).
"""


def tally(rows):
    by_arm = {}
    for r in rows:
        by_arm.setdefault(r["arm"], []).append(bool(r["hit"]))
    out = {}
    for arm, hits in by_arm.items():
        n = len(hits)
        k = sum(hits)
        p = k / n if n else 0.0
        sigma = (p * (1 - p) / n) ** 0.5 if n else 0.0
        out[arm] = {"n": n, "hits": k, "rate": p, "sigma": sigma}
    return out


def verdict(t, none_ceiling=0.5):
    none_rate = t.get("none", {}).get("rate", 0.0)
    headroom = none_rate < none_ceiling
    a = t.get("A")
    b = t.get("B")
    if not (a and b):
        return {"headroom": headroom, "none_rate": none_rate, "status": "INCOMPLETE (need A and B arms)"}

    diff = b["rate"] - a["rate"]
    sigma_diff = (a["sigma"] ** 2 + b["sigma"] ** 2) ** 0.5
    threshold = 2 * sigma_diff

    if not headroom:
        status = (f"UNDERPOWERED — none-floor {none_rate*100:.0f}% ≥ ceiling {none_ceiling*100:.0f}%; "
                  "cold already applies the principle, so this is ceiling-limited, NOT a tie")
    elif sigma_diff > 0 and diff >= threshold:
        status = f"B_WINS — B−A = {diff*100:+.0f}pp ≥ 2σ ({threshold*100:.0f}pp); delivery moved above noise"
    elif sigma_diff > 0 and diff <= -threshold:
        status = f"A_WINS — question-anchoring HURT: B−A = {diff*100:+.0f}pp ≤ −2σ ({threshold*100:.0f}pp)"
    else:
        status = (f"PARK — B−A = {diff*100:+.0f}pp within 2σ ({threshold*100:.0f}pp); "
                  "can't distinguish, park with evidence")

    return {"headroom": headroom, "none_rate": none_rate, "diff": diff,
            "sigma_diff": sigma_diff, "threshold_2sigma": threshold, "status": status}
