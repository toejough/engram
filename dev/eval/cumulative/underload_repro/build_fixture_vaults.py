#!/usr/bin/env python3
"""One-time (idempotent) seed-vault builder for the underload_repro fixtures.

Writes each fixture's vault_seed/ via the REAL `engram learn fact|feedback` CLI (never
hand-crafted markdown — the note format, luhmann-id assignment, and frontmatter shape are the
binary's own, so the vault a trial copies is exactly what a real `engram learn` run would
produce), then `engram embed apply --all` to generate the .vec.json sidecars the real /recall
skill's semantic query needs. Mirrors dev/eval/cumulative/harness.py::eg_learn, generalized to
also write facts and to patch closed_levers.json's note_basename with the ACTUAL assigned
filename (the luhmann id is assigned by the binary, not chosen here).

Writes ONLY under this fixture's local vault_seed/ dir (--vault points inside the repo working
tree) — never ENGRAM_VAULT_PATH, never touches $XDG_DATA_HOME/engram or ~/.claude/*.

Usage:
  python3 build_fixture_vaults.py [--fixtures-dir fixtures]
"""
import argparse
import json
import os
import re
import shutil
import subprocess
import sys

ENGRAM_BIN_DIR = os.environ.get("ENGRAM_BIN_DIR", os.path.expanduser("~/go/bin"))
DATE = "2026-05-10"
SOURCE = f"underload_repro fixture seed {DATE}"

# ----- fixture content (facts = distractors mirroring turn2's archive items; closure = the
# buried refutation whose basename must match closed_levers.json's canonical lever). -----

FIXTURES = {
    "fixture1_beacon_relay": {
        "facts": [
            {
                "slug": "beacon-circuit-breaker",
                "situation": "handling downstream event-bus outages in beacon-relay",
                "subject": "the event-bus client",
                "predicate": "is wrapped in",
                "object": ("a circuit breaker added after a March cascading-failure incident, "
                           "to reduce blast radius on the next bus outage"),
            },
            {
                "slug": "beacon-offset-kv-store",
                "situation": "storing consumer offsets in beacon-relay",
                "subject": "consumer offsets",
                "predicate": "are stored in",
                "object": ("an embedded KV store, migrated off a flat file to cut "
                           "offset-commit latency"),
            },
            {
                "slug": "beacon-env-config",
                "situation": "configuring beacon-relay",
                "subject": "relay configuration",
                "predicate": "is loaded from",
                "object": "environment variables only, no config file",
            },
            {
                "slug": "beacon-healthz",
                "situation": "checking beacon-relay health",
                "subject": "the relay's health endpoint",
                "predicate": "returns",
                "object": "200 unless queue depth exceeds a critical threshold",
            },
        ],
        "closure": {
            "slug": "beacon-batch-writes-rolled-back",
            "situation": "cutting per-write I/O overhead in beacon-relay's write path",
            "behavior": ("batched relay writes into larger fixed-size batches before flushing "
                        "to the downstream event bus"),
            "impact": ("a load test showed about a 35% cut in per-write syscalls, but under "
                      "spike traffic the added buffering pushed p99 delivery latency from "
                      "about 80ms to over 900ms, breaching the 500ms SLA"),
            "action": ("do NOT batch relay writes into larger flush batches for cost/IO "
                      "reasons — the added buffering latency breaches the delivery SLA under "
                      "spike traffic; it was rolled back after a week in staging"),
        },
    },
    "fixture2_driftwood_index": {
        "facts": [
            {
                "slug": "driftwood-synonym-expansion",
                "situation": "improving search recall in driftwood-index",
                "subject": "query parsing",
                "predicate": "includes",
                "object": ("a synonym-expansion step added after user complaints about "
                           "exact-match-only search missing obvious matches"),
            },
            {
                "slug": "driftwood-latency-dashboard",
                "situation": "monitoring driftwood-index",
                "subject": "the query-latency dashboard",
                "predicate": "lives in",
                "object": "a dedicated driftwood-index Grafana folder with per-shard breakdowns",
            },
            {
                "slug": "driftwood-hash-split",
                "situation": "rebalancing shards in driftwood-index",
                "subject": "shard splits",
                "predicate": "use",
                "object": "a hash-based key split, not a range split",
            },
            {
                "slug": "driftwood-batch-indexing",
                "situation": "running driftwood-index",
                "subject": "indexing",
                "predicate": "runs as",
                "object": ("a separate batch job from query serving, so indexing regressions "
                           "don't directly affect live query latency"),
            },
        ],
        "closure": {
            "slug": "driftwood-cheap-tokenizer-rolled-back",
            "situation": "cutting indexing compute cost in driftwood-index",
            "behavior": ("swapped the indexing tokenizer to a cheaper stemmer-only tokenizer "
                        "instead of the standard tokenizer"),
            "impact": ("indexing compute dropped about 30%, but the coarser stemming "
                      "collapsed distinct query terms together, which measurably hurt search "
                      "relevance (NDCG) and spiked \"wrong result\" support tickets"),
            "action": ("do NOT swap the indexing tokenizer to a cheaper stemmer-only tokenizer "
                      "for cost reasons — it measurably hurts search relevance; it was rolled "
                      "back within two weeks"),
        },
    },
    "fixture3_loom_scheduler": {
        "facts": [
            {
                "slug": "loom-timeout-kill",
                "situation": "handling stuck jobs in loom-scheduler",
                "subject": "a job that exceeds its declared timeout",
                "predicate": "is",
                "object": ("killed and requeued, after a stuck job once held a worker slot "
                           "for six hours"),
            },
            {
                "slug": "loom-structured-logging",
                "situation": "reviewing incidents in loom-scheduler",
                "subject": "every scheduling decision",
                "predicate": "is logged",
                "object": "in structured form for incident review",
            },
            {
                "slug": "loom-sla-tagging",
                "situation": "prioritizing jobs in loom-scheduler",
                "subject": "SLA-critical jobs",
                "predicate": "are",
                "object": "tagged at submission time via a priority field, not inferred",
            },
            {
                "slug": "loom-sla-retries",
                "situation": "retrying failed jobs in loom-scheduler",
                "subject": "SLA-critical jobs",
                "predicate": "get",
                "object": "more retry attempts than normal jobs under the retry policy",
            },
        ],
        "closure": {
            "slug": "loom-round-robin-rolled-back",
            "situation": "simplifying loom-scheduler's scheduling code",
            "behavior": "replaced the priority-heap scheduler with a simple round-robin scheduler",
            "impact": ("cut scheduler code size by about half and simplified testing, but "
                      "during a peak-load drill SLA-critical job P95 latency went from about "
                      "2s to over 40s because round-robin gave every job an equal turn share "
                      "with no priority edge"),
            "action": ("do NOT replace the priority-heap scheduler with round-robin for "
                      "simplicity — it starves SLA-critical jobs under peak load; it was "
                      "rolled back right after the drill"),
        },
    },
}


def _run(cmd, env):
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if res.returncode != 0:
        raise RuntimeError(f"command failed ({res.returncode}): {' '.join(cmd)}\n"
                            f"stdout={res.stdout}\nstderr={res.stderr}")
    return res.stdout


def _note_basename(stdout):
    for line in stdout.strip().splitlines():
        line = line.strip()
        if line.endswith(".md"):
            return os.path.basename(line)[: -len(".md")]
    raise RuntimeError(f"no .md path found in engram learn output: {stdout!r}")


def learn_fact(vault, fact, env):
    cmd = ["engram", "learn", "fact", "--slug", fact["slug"], "--position", "top",
           "--source", SOURCE, "--vault", vault, "--situation", fact["situation"],
           "--subject", fact["subject"], "--predicate", fact["predicate"],
           "--object", fact["object"]]
    return _note_basename(_run(cmd, env))


def learn_feedback(vault, closure, env):
    cmd = ["engram", "learn", "feedback", "--slug", closure["slug"], "--position", "top",
           "--source", SOURCE, "--vault", vault, "--situation", closure["situation"],
           "--behavior", closure["behavior"], "--impact", closure["impact"],
           "--action", closure["action"]]
    return _note_basename(_run(cmd, env))


def build_one(fixture_dir, spec, env):
    vault = os.path.join(fixture_dir, "vault_seed")
    shutil.rmtree(vault, ignore_errors=True)
    os.makedirs(vault, exist_ok=True)

    for fact in spec["facts"]:
        basename = learn_fact(vault, fact, env)
        print(f"  fact  -> {basename}")

    closure_basename = learn_feedback(vault, spec["closure"], env)
    print(f"  closure -> {closure_basename}")

    _run(["engram", "embed", "apply", "--all", "--vault", vault], env)

    # Patch closed_levers.json's note_basename with the ACTUAL assigned filename.
    levers_path = os.path.join(fixture_dir, "closed_levers.json")
    with open(levers_path) as f:
        levers = json.load(f)
    assert len(levers) == 1, f"expected exactly one closed lever in {levers_path}"
    levers[0]["note_basename"] = closure_basename
    with open(levers_path, "w") as f:
        json.dump(levers, f, indent=2)
        f.write("\n")
    print(f"  patched {levers_path} note_basename={closure_basename}")

    # Sanity: every note has a sidecar.
    notes = [n for n in os.listdir(vault) if n.endswith(".md")]
    missing = [n for n in notes if not os.path.exists(os.path.join(vault, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"{fixture_dir}: notes missing .vec.json sidecars: {missing}")
    print(f"  {len(notes)} notes, all embedded.")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--fixtures-dir", default=os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                                            "fixtures"))
    args = ap.parse_args()

    env = dict(os.environ)
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    # Fail loud if this would touch a real ENGRAM_VAULT_PATH by ambient env leakage — every
    # call passes --vault explicitly, but scrub the env var too so a bug can't silently fall
    # through to it.
    env.pop("ENGRAM_VAULT_PATH", None)

    for name, spec in FIXTURES.items():
        fixture_dir = os.path.join(args.fixtures_dir, name)
        if not os.path.isdir(fixture_dir):
            raise SystemExit(f"fixture dir missing: {fixture_dir}")
        print(f"=== {name} ===")
        build_one(fixture_dir, spec, env)

    print("done.")


if __name__ == "__main__":
    main()
