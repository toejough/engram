"""#661 Phase 1 — free retrieval spend-gate: glance-vs-deep phrase-count sweep.

For each cosine axis (C3/C4i/C6), seed the load-bearing notes into a temp vault, then run the real
`engram query` with the first-n of /recall's 10 phrases (n in {1,2,3,5,10}) and report whether every
load-bearing target still surfaces and at what worst rank. K=top-5. C5 is recency-not-cosine (deferred
to Phase 2). No LLM — bundled embedder only.
"""
import sys, os, subprocess, tempfile, shutil, json

TRAPS = "/Users/joe/repos/personal/engram/dev/eval/traps"
sys.path.insert(0, TRAPS)
import retrieval_probe as rp
import seed_c3
import c4_idio
import reasoning_recall_eval as rr

LEVELS = [1, 2, 3, 5, 10]
K = 5  # recall@5


def seed_base(axis, temp):
    if axis == "C3":
        seed_c3.seed(temp)
    elif axis == "C4i":
        c4_idio.seed_into(temp)
    elif axis == "C6":
        for case in ("abduction-diag", "abduction-badge"):
            for note in rr.CASES[case]["notes"]:
                rr._learn(temp, *note)


def probe_n(vault, axis, n):
    phrases = rp.AXIS_PHRASES[axis][:n]
    targets = rp.AXIS_TARGETS[axis]
    cmd = ["engram", "query"]
    for p in phrases:
        cmd += ["--phrase", p]
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    r = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if r.returncode != 0:
        raise RuntimeError(f"query failed axis={axis} n={n}: {r.stderr.strip()}")
    payload = rp._parse_payload(r.stdout)
    per = {t: rp.rank_in_payload(payload, t) for t in targets}
    all_surf = bool(per) and all(v["surfaced"] for v in per.values())
    worst = max(v["rank"] for v in per.values()) if all_surf else None
    in_topk = all_surf and worst is not None and worst <= K
    return {"per": {t: v["rank"] for t, v in per.items()}, "all_surfaced": all_surf,
            "worst_rank": worst, "in_top%d" % K: in_topk}


results = {}
for axis in ["C3", "C4i", "C6"]:
    temp = tempfile.mkdtemp(prefix=f"p1-{axis}-")
    try:
        seed_base(axis, temp)
        results[axis] = {n: probe_n(temp, axis, n) for n in LEVELS}
    finally:
        shutil.rmtree(temp, ignore_errors=True)

# Table
print("\n=== #661 Phase 1: glance-vs-deep retrieval (free) — K=top-5 ===")
for axis in ["C3", "C4i", "C6"]:
    print(f"\n{axis}  targets={rp.AXIS_TARGETS[axis]}")
    print(f"  {'n':>3} | {'all_surfaced':>12} | {'worst_rank':>10} | in_top5 | per-target ranks")
    for n in LEVELS:
        r = results[axis][n]
        print(f"  {n:>3} | {str(r['all_surfaced']):>12} | {str(r['worst_rank']):>10} | "
              f"{str(r['in_top5']):>7} | {r['per']}")

out = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad/phase1_results.json"
with open(out, "w") as f:
    json.dump(results, f, indent=2)
print(f"\nwrote {out}")
