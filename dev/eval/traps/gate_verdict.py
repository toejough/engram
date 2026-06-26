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
