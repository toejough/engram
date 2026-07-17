#!/usr/bin/env python3
"""Combined bucketed scorer: ARCH (structural) + alpha/beta/native (behavioral).
Reports per-bucket pass counts and the list of failed items with their gap-level
user-symptom phrasings (the convergence-feedback payload).

Usage: python3 score.py <workdir> <spec.json>
"""
import sys, os, json
import archscore, behavioral


def load_spec(path):
    """Load a spec, merging the shared house-gotcha checks when the spec names one.

    Single source of truth for spec loading: every raw `json.load(open(spec))` site routes
    through here so a standalone behavioral run, the leakage check, and the build loop all
    carry byte-identical house checks. Resolves `house_checks_file` relative to the spec dir."""
    with open(path) as f:
        spec = json.load(f)
    spec.setdefault("checks", [])
    hcf = spec.get("house_checks_file")
    if hcf:
        with open(os.path.join(os.path.dirname(path), hcf)) as f:
            spec["checks"] = spec["checks"] + json.load(f).get("house_checks", [])
    return spec

ARCH_SYMPTOMS = {
    "di": "Your core logic and the file handling are fused together — I want the storage swappable so the core can be exercised without touching a real file.",
    "sentinel": "When something isn't found, a caller can only read a message string — I want a not-found condition detectable in code, not by matching text.",
    "atomic": "Your save rewrites the file in place — if it dies mid-write I could lose everything. I care about the data surviving a crash during a write.",
    "stdlib": "Keep it dependency-free — standard library only.",
    "tests_fake_parallel": "Your tests write real files and run serially — exercise the core against an in-memory stand-in instead, and make the tests safe to run in parallel.",
    "json": "I also want a machine-readable JSON output mode I can pipe into scripts.",
    "nocolor": "When I set NO_COLOR or redirect the output, don't emit any color codes.",
    "wrapped_errors": "Errors don't say what failed — wrap them with context.",
    "named_perms": "There are bare permission numbers in there — give them names so it's clear what they mean.",
    "no_global_data": "Avoid keeping the data in global mutable state.",
}

def score(workdir, specpath):
    spec = load_spec(specpath)
    arch = archscore.score(workdir)
    beh = behavioral.score(workdir, spec)
    out = {"arch": arch["arch"], "arch_pass": arch["passed"]}
    if beh.get("build") != "ok":
        out["build"] = "FAIL"; out["error"] = beh.get("error", "")[:300]
        out["failed"] = [("BUILD", "It doesn't compile / tests don't pass yet.")]
        return out
    out["build"] = "ok"
    bucket_names = {}
    for c in spec["checks"]:
        bucket_names.setdefault(c["bucket"], []).append(c["name"])
    out["feat_buckets"] = {b: f"{sum(1 for n in names if beh['checks'][n]['pass'])}/{len(names)}"
                            for b, names in bucket_names.items()}
    out["arch_fail"] = [k for k, v in arch["detectors"].items() if not v["pass"]]
    out["feat_fail"] = [n for n in beh["checks"] if not beh["checks"][n]["pass"]]
    failed = []
    for d, v in arch["detectors"].items():
        if not v["pass"]:
            failed.append(["ARCH:" + d, ARCH_SYMPTOMS.get(d, d)])
    for c in spec["checks"]:
        if not beh["checks"][c["name"]]["pass"]:
            failed.append([c["bucket"] + ":" + c["name"], c["symptom"]])
    out["failed"] = failed
    # total: 10 arch + N behavioral
    total = arch["n"] + beh["n"]
    passed = arch["passed"] + beh["passed"]
    out["total"] = f"{passed}/{total}"
    return out

if __name__ == "__main__":
    print(json.dumps(score(sys.argv[1], sys.argv[2]), indent=2))
