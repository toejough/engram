"""Does RECALL need a reasoning step? — clean A/B.

Premises live in engram MEMORY (idiosyncratic notes). The task is OPEN-ENDED (does not tell the agent
to reason/compose). Two arms differ ONLY in the recall SKILL:
  asis   : the current recall skill (surfaces + curates memory; never says 'reason over it').
  reason : recall + an appended 'Step 2.8 — reason over what you recalled' instruction.

If `reason` reaches the correct reasoned conclusion more than `asis`, recall should prompt reasoning.
Built on the vetted reasoning forms (docs/research/2026-06-23-reasoning-modes.md). Independent sonnet
judge. Full answers saved for manual transcript reading at small n.

Usage: python3 reasoning_recall_eval.py [--case abduction-diag] [--n 4] [--workers 4]
"""
import argparse
import concurrent.futures as cf
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from run import MODELS
from wrun import build_warm_cfg, _slug

REPO = "/Users/joe/repos/personal/engram"
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/reasoning-recall")

NEUTRAL_PREFIX = ("Invoke your /recall skill, then answer the question from what you recall. "
                  "Question:\n\n")

# The reasoning step appended to the recall skill for the `reason` arm.
REASON_STEP = """

### Step 2.8 — Reason over what you recalled (draw the conclusion)

After surfacing memory, do NOT just report what you recalled. REASON over it: if the recalled facts,
combined, support a conclusion that bears on the user's question — a deduction (necessary), an
abduction / best-explanation, or an inductive generalization (both probable, defeasible) — STATE that
conclusion explicitly, even if the question did not ask you to derive it. Connect the relevant facts
into the inference and hedge its certainty correctly. The user needs the implication, not just the notes.
"""

CASES = {
    # abduction: a distinctive rule + a matching observation -> abduce the cause. Idiosyncratic tokens.
    "abduction-diag": {
        "notes": [
            ("zephyr-leak-signature", "in the zephyr-3 reactor a coolant-line leak",
             "is the ONLY fault that simultaneously causes",
             "rising chamber pressure, an audible hiss, and a falling coolant level"),
            ("zephyr-current-state", "right now the zephyr-3 reactor",
             "is showing", "rising chamber pressure, an audible hiss, and a falling coolant level"),
        ],
        "task": ("The zephyr-3 reactor is behaving oddly and the team needs to know what to investigate "
                 "first. What does your memory suggest?"),
        "E": ("a coolant-line leak is the most likely cause — the three signs now present (rising "
              "pressure, hiss, falling coolant) are exactly the signature that ONLY a coolant-line leak "
              "produces; hedged as the best explanation, not certain."),
    },
    # HARDER: non-obvious cross-domain composition (facilities hardware x badge policy), genuinely open
    # prompt naming only the SYMPTOM. The two notes don't obviously belong together; the agent must
    # notice they combine. This is where unprompted reasoning might fail (recall both, never connect).
    "abduction-badge": {
        "notes": [
            ("badge-reader-swap", "the lobby badge readers",
             "were all replaced last month with", "the new RX-9 model"),
            ("rx9-rejects-old", "RX-9 badge readers",
             "silently reject", "any access badge that was issued before 2021"),
        ],
        "task": ("A handful of our longer-tenured employees have started complaining they can't get "
                 "into the building, while everyone else is fine. Any idea what's going on?"),
        "E": ("the new RX-9 lobby readers silently reject pre-2021 badges, so the longer-tenured "
              "employees (who still carry pre-2021 badges) are the ones being locked out — composed "
              "from the reader swap + the RX-9 rejection rule."),
    },
}


CASES["control-unrelated"] = {  # no sound emergent conclusion -> the gate must persist NOTHING
    "notes": [
        ("offsite-lisbon", "the marketing team offsite", "will be held in", "Lisbon in May"),
        ("staging-pg14", "the staging database", "runs on", "PostgreSQL 14"),
    ],
    "task": "I'm doing some general planning. Anything in memory I should keep in mind?",
    "E": "(there is no sound emergent conclusion combining these two unrelated facts)",
}


def _build_reason_cfg(dst):
    build_warm_cfg(dst)
    skill = os.path.join(dst, "skills", "recall", "SKILL.md")
    with open(skill, "a") as f:
        f.write(REASON_STEP)


def _learn(vault, slug, subj, pred, obj):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(["engram", "learn", "fact", "--slug", slug, "--position", "top",
                    "--source", f"reasoning fixture: {slug}", "--situation", f"{subj} {pred} {obj}",
                    "--subject", subj, "--predicate", pred, "--object", obj],
                   env=env, check=True, capture_output=True, text=True)


def _run(prompt, cfg, model, vault=None, wd=None):
    env = dict(os.environ); env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "10000"
    if vault:
        env["ENGRAM_VAULT_PATH"] = vault
        env["ENGRAM_CHUNKS_DIR"] = os.path.join(wd, "chunks"); os.makedirs(env["ENGRAM_CHUNKS_DIR"], exist_ok=True)
        env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    args = ["claude", "-p", prompt, "--output-format", "json", "--model", MODELS[model],
            "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=(wd or cfg), env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.02:
            continue
        break
    return out


JUDGE = ("Grade whether an answer reaches a specific reasoned conclusion from recalled facts.\n\n"
         "QUESTION:\n{task}\n\nREQUIRED CONCLUSION:\n{E}\n\nANSWER:\n{answer}\n\n"
         "HIT only if the answer explicitly states the required conclusion (names the specific cause and "
         "connects it to the observed signs) — not merely listing the recalled facts without concluding. "
         "Reply 'HIT' or 'MISS' on line 1, then one sentence.")


def run_one(case, arm, cfg, judge_cfg, idx):
    spec = CASES[case]
    wd = tempfile.mkdtemp(prefix=f"{case}-{arm}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault"); os.makedirs(vault)
    for n in spec["notes"]:
        _learn(vault, *n)
    out = _run(NEUTRAL_PREFIX + spec["task"], cfg, "opus", vault=vault, wd=wd)
    answer = out.get("result") or ""
    j = _run(JUDGE.format(task=spec["task"], E=spec["E"], answer=answer or "(none)"), judge_cfg, "sonnet")
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return {"case": case, "arm": arm, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + (j.get("total_cost_usd", 0) or 0), "answer": answer}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--case", default="abduction-diag")
    ap.add_argument("--n", type=int, default=4)
    ap.add_argument("--workers", type=int, default=4)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    asis_cfg = os.path.join(ROOT, "asis-cfg"); build_warm_cfg(asis_cfg)
    reason_cfg = os.path.join(ROOT, "reason-cfg"); _build_reason_cfg(reason_cfg)
    judge_cfg = os.path.join(ROOT, "judge-cfg"); build_warm_cfg(judge_cfg)

    jobs = [(a.case, "asis", asis_cfg, i) for i in range(a.n)] + \
           [(a.case, "reason", reason_cfg, i) for i in range(a.n)]
    print(f"recall-reasoning A/B: case={a.case} n={a.n} = {len(jobs)} trials")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, c, arm, cfg, judge_cfg, i): (c, arm, i) for c, arm, cfg, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['arm']:7} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")

    print(f"\n=== RECALL reasoning A/B — conclusion reached ===")
    for arm in ("asis", "reason"):
        v = [r for r in results if r["arm"] == arm]; h = sum(r["hit"] for r in v)
        print(f"  {arm:7} {h}/{len(v)} ({100*h/len(v):.0f}%)")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, f"recall-reason-{a.case}.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
