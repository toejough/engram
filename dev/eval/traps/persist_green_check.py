"""GREEN check for recall Step 4 (persist reasoned conclusion, linked to inputs).

Builds a warm cfg from the CURRENT repo recall skill (now with Step 4), runs the badge abduction case,
and inspects each resulting vault: did a NEW synthesis note appear, is it marked derived, and does it
link BACK to both input notes with relationship rationales? Reads the actual notes (manual verify).

Usage: python3 persist_green_check.py [--n 3]
"""
import argparse, glob, json, os, re, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import reasoning_recall_eval as rr
from run import MODELS
from wrun import build_warm_cfg, _slug

ROOT = "/tmp/persist-green"


def run_one(cfg, idx):
    spec = rr.CASES["abduction-badge"]
    wd = tempfile.mkdtemp(prefix=f"green-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault"); os.makedirs(vault)
    for n in spec["notes"]:
        rr._learn(vault, *n)
    before = set(os.path.basename(f) for f in glob.glob(vault + "/*.md"))
    out = rr._run(rr.NEUTRAL_PREFIX + spec["task"], cfg, "opus", vault=vault, wd=wd)
    after = glob.glob(vault + "/*.md")
    new = [f for f in after if os.path.basename(f) not in before]
    # inspect the new synthesis note(s)
    info = {"idx": idx, "note_delta": len(after) - len(before), "new": [], "cost": out.get("total_cost_usd", 0) or 0}
    for f in new:
        body = open(f, errors="ignore").read()
        rels = re.findall(r"\[\[([^\]]+)\]\]", body)
        derived = "synthesis" in body.lower() or "derived" in body.lower() or "abduction" in body.lower()
        links_both = sum(1 for inp in ("badge-reader-swap", "rx9-rejects-old") if any(inp in r for r in rels))
        info["new"].append({"name": os.path.basename(f), "derived_marker": derived,
                            "links_to_inputs": links_both, "n_rels": len(rels)})
    return info


def main():
    ap = argparse.ArgumentParser(); ap.add_argument("--n", type=int, default=3); a = ap.parse_args()
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "cfg"); build_warm_cfg(cfg)
    assert "persist the reasoned conclusion" in open(os.path.join(cfg, "skills/recall/SKILL.md")).read().lower(), \
        "Step 4 not in the warm cfg skill — did the repo skill edit land?"
    print(f"GREEN check: badge case, n={a.n}, skill has Step 4")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.n) as ex:
        futs = {ex.submit(run_one, cfg, i): i for i in range(a.n)}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [#{r['idx']}] note_delta={r['note_delta']} new={r['new']} ${r['cost']:.2f}")
    persisted = sum(1 for r in results if r["note_delta"] >= 1)
    linked = sum(1 for r in results if any(n["links_to_inputs"] == 2 for n in r["new"]))
    print(f"\npersisted a synthesis note: {persisted}/{a.n} | linked to BOTH inputs: {linked}/{a.n}")
    json.dump(results, open(os.path.join(ROOT, "green.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
