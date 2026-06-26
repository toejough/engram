"""Pure verdict logic for the trap regression gate — no I/O, unit-tested."""


def axis_verdict(trials, bar, contam_threshold=0.2):
    n = len(trials)
    contaminated = sum(1 for t in trials if t["contaminated"])
    valid = [t for t in trials if not t["contaminated"]]
    passed = sum(1 for t in valid if t["pass"])
    if n and contaminated / n > contam_threshold:
        status = "INCONCLUSIVE"
    elif passed == len(valid) and len(valid) > 0:
        status = "GREEN"          # exact bar: every VALID trial passes
    else:
        status = "RED"
    return {"valid": len(valid), "contaminated": contaminated, "passed": passed,
            "bar": bar, "status": status}


def gate_verdict(axes):
    statuses = [a["status"] for a in axes.values()]
    if "RED" in statuses:
        verdict = "RED"
    elif "INCONCLUSIVE" in statuses:
        verdict = "INCONCLUSIVE"
    else:
        verdict = "GREEN"
    return {"verdict": verdict, "axes": axes}


def _norm_c3(rows):
    # wrun.py warm-results.json: verdict is "applied"|"trap"|"nobuild" (nobuild = degenerate trial).
    return [{"pass": r["verdict"] == "applied", "contaminated": r["verdict"] == "nobuild"}
            for r in rows]


def _norm_c4i(rows):
    # c4-idio-results.json: only the warm-XXp arm exercises the gate; score is None when unbuilt.
    out = []
    for r in rows:
        if r.get("arm") != "warm-XXp":
            continue
        out.append({"pass": bool((r.get("score") or {}).get("supersession_correct")),
                    "contaminated": not r["built"]})
    return out


def _norm_c5(rows):
    # c5-results.json warm arm: an unbuilt trial is degenerate; honored is the pass signal.
    return [{"pass": bool(r.get("honored")), "contaminated": not r["built"]} for r in rows]


def _norm_c6(rows):
    # c6-warm.json: no "built" field — an empty answer marks the degraded (contaminated) build.
    return [{"pass": bool(r.get("hit")), "contaminated": not (r.get("answer") or "").strip()}
            for r in rows]


_ADAPTERS = {"C3": _norm_c3, "C4i": _norm_c4i, "C5": _norm_c5, "C6": _norm_c6}


def normalize(axis, rows):
    """Normalize a harness's result rows to the common [{"pass","contaminated"}] trial shape."""
    if axis not in _ADAPTERS:
        raise ValueError(f"unknown axis {axis!r}")
    return _ADAPTERS[axis](rows)
