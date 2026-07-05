"""Question-anchored distillation — delivery eval (design:
docs/design/2026-07-01-question-anchored-distillation.md, deleted 2026-07; git log).

The claim under test: a note anchored to the QUESTION the agent was investigating (arm B) is
retrieved+applied better on a FUTURE related question than a note anchored to the evidence's content
TOPIC (arm A, the current recall Step-2.5 mechanism). Same evidence, same model; only the distillation
anchor differs.

Stages (fail-fast):
  crystallize : distill each pair BOTH ways (A=topic, B=question), freeze to notes.json, print for review.
  headroom    : run the `none` condition (no note) cold — CHECKPOINT. If cold opus already applies the
                principle (none-rate high) there is NO headroom; a null result would be ceiling-limited,
                NOT a tie (see feedback_gap_below_noise_is_underpowered). Fix the corpus before spending.
  deliver     : run all 3 conditions (none / +A / +B), blind opus judge on principle-application, tally.
  all         : crystallize -> headroom -> deliver.

Application-injected (the note is placed in the prompt, no recall/crowd) is the cheap UPPER BOUND on
full-path delivery: retrieval can only lose a note, never add application. If B does not beat A here,
the full crowded-recall path cannot save it -> park. Only if B wins here is the retrieval path worth
testing (Stage 4, deferred behind this verdict per the design).

Usage:
  python3 qanchor_eval.py --stage crystallize
  python3 qanchor_eval.py --stage headroom [--reps 3]
  python3 qanchor_eval.py --stage deliver  [--reps 3]
"""
import argparse
import concurrent.futures as cf
import json
import os
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from run import MODELS, build_cold_cfg
from qanchor_corpus import PAIRS
from qanchor_score import tally, verdict  # pure scoring (unit-tested in test_qanchor.py)

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/qanchor-eval")
NOTES_PATH = os.path.join(ROOT, "notes.json")

# --- crystallization prompts: same evidence, differ ONLY in the anchor -------------------------------

_NOTE_FORMAT = ("Output EXACTLY two lines and nothing else:\n"
                "SITUATION: <one short retrieval-shaped phrase naming when this note applies>\n"
                "LESSON: <the claim / what to do>")

CRYSTALLIZE_A = (  # topic-anchored — the current recall Step-2.5 mechanism (one note per content cluster)
    "You are crystallizing ONE memory note from a cluster of recalled evidence. The fragments below "
    "were retrieved together because they share a topic.\n\nEVIDENCE:\n{evidence}\n\n"
    "Write ONE note that captures the principle these members share, pitched at the cluster's topic. "
    "Phrase the SITUATION as the setting these members are about.\n\n" + _NOTE_FORMAT)

CRYSTALLIZE_B = (  # question-anchored — the prototype (pitch the note at the investigating question)
    "You are crystallizing ONE memory note from recalled evidence. The agent retrieved this evidence "
    "while investigating a specific QUESTION.\n\nINVESTIGATING QUESTION:\n{question}\n\n"
    "EVIDENCE (retrieved in service of that question):\n{evidence}\n\n"
    "Write ONE note pitched at answering THAT question — the transferable principle the question was "
    "seeking, phrased so a future agent asking a similar question (even about a different part of the "
    "system) would retrieve and apply it. Phrase the SITUATION as the question/failure the note "
    "answers, not the evidence's topic.\n\n" + _NOTE_FORMAT)

# --- delivery + judge prompts ------------------------------------------------------------------------

DELIVER_WITH_NOTE = ("A note from your engram memory surfaced for this task:\n\n{note}\n\n"
                     "Now answer:\n{future_q}")

JUDGE = (
    "Grade whether an answer applies a specific required principle to the question's domain.\n\n"
    "QUESTION:\n{future_q}\n\n"
    "REQUIRED PRINCIPLE-APPLICATION (what a correct answer must do):\n{E}\n\n"
    "ANSWER:\n{answer}\n\n"
    "HIT only if the answer applies the required principle to THIS question's domain — showing the "
    "reasoning described above. Track the REASONING, not any specific vocabulary or note name: an "
    "answer that names a mechanism but does not apply it to this question is a MISS; an answer that "
    "applies the right reasoning in different words is a HIT. Reply 'HIT' or 'MISS' on line 1, then "
    "one sentence.")


def _claude(prompt, cfg, model, max_tokens=8000):
    """One isolated `claude -p` call with degraded-build retry (is_error + near-zero cost => transient)."""
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = str(max_tokens)
    wd = tempfile.mkdtemp(dir=os.path.join(ROOT, "ws"))
    args = ["claude", "-p", prompt, "--output-format", "json", "--model", MODELS[model],
            "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.01:
            continue
        break
    return out


def _evidence_block(pair):
    return "\n".join("- " + e for e in pair["evidence"])


def crystallize(cfg, workers):
    """Distill each pair BOTH ways; freeze to NOTES_PATH."""
    def one(key, arm):
        pair = PAIRS[key]
        tmpl = CRYSTALLIZE_A if arm == "A" else CRYSTALLIZE_B
        prompt = tmpl.format(evidence=_evidence_block(pair), question=pair.get("question", ""))
        out = _claude(prompt, cfg, "opus")
        return key, arm, (out.get("result") or "").strip(), (out.get("total_cost_usd", 0) or 0)

    jobs = [(k, arm) for k in PAIRS for arm in ("A", "B")]
    notes, spent = {k: {} for k in PAIRS}, 0.0
    with cf.ThreadPoolExecutor(max_workers=workers) as ex:
        futs = [ex.submit(one, k, arm) for k, arm in jobs]
        for fut in cf.as_completed(futs):
            key, arm, note, cost = fut.result()
            notes[key][arm] = note
            spent += cost
            print(f"  [{key:18} {arm}] {'OK' if note else 'EMPTY'} ${cost:.2f}")
    json.dump(notes, open(NOTES_PATH, "w"), indent=1)
    print(f"\nwrote {NOTES_PATH}  (crystallize spend ${spent:.2f})")
    return notes


def _judge(pair, answer, cfg):
    j = _claude(JUDGE.format(future_q=pair["future_q"], E=pair["E"], answer=answer or "(none)"),
                cfg, "opus", max_tokens=1500)
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return hit, (j.get("total_cost_usd", 0) or 0)


def run_condition(key, arm, note, cfg, judge_cfg, idx):
    """One delivery trial: pose future_q under a condition (none/A/B), judge principle-application blind."""
    pair = PAIRS[key]
    if arm == "none":
        prompt = pair["future_q"]
    else:
        prompt = DELIVER_WITH_NOTE.format(note=note, future_q=pair["future_q"])
    out = _claude(prompt, cfg, "opus")
    answer = out.get("result") or ""
    hit, jcost = _judge(pair, answer, judge_cfg)
    return {"pair": key, "arm": arm, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + jcost, "answer": answer}


def deliver(cfg, judge_cfg, reps, workers, arms, notes):
    jobs = []
    for k in PAIRS:
        for arm in arms:
            note = "" if arm == "none" else notes[k][arm]
            for i in range(reps):
                jobs.append((k, arm, note, i))
    label = "+".join(arms)
    print(f"delivery: pairs={len(PAIRS)} arms={arms} reps={reps} = {len(jobs)} trials ({label})")
    results = []
    with cf.ThreadPoolExecutor(max_workers=workers) as ex:
        futs = {ex.submit(run_condition, k, arm, note, cfg, judge_cfg, i): (k, arm, i)
                for k, arm, note, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [{r['pair']:18} {r['arm']:4} #{r['idx']}] hit={int(r['hit'])} ${r['cost']:.2f}")
    return results


def _print_tally(results, none_ceiling):
    t = tally(results)
    v = verdict(t, none_ceiling=none_ceiling)
    print(f"\n=== DELIVERY RATE (knowledge applied to the future question) ===")
    print(f"  {'arm':5} {'hits':>6} {'n':>4} {'rate':>7} {'±1σ':>7}")
    for arm in ("none", "A", "B"):
        if arm in t:
            a = t[arm]
            print(f"  {arm:5} {a['hits']:>6} {a['n']:>4} {a['rate']*100:>6.0f}% {a['sigma']*100:>6.1f}%")
    # per-pair paired view (B vs A), only when both present
    if "A" in t and "B" in t:
        print(f"\n  per-pair paired (B−A), one point per pair (mean over reps):")
        bwin = awin = tie = 0
        for k in PAIRS:
            av = [r["hit"] for r in results if r["pair"] == k and r["arm"] == "A"]
            bv = [r["hit"] for r in results if r["pair"] == k and r["arm"] == "B"]
            if not av or not bv:
                continue
            am, bm = sum(av)/len(av), sum(bv)/len(bv)
            mark = "B>A" if bm > am else ("A>B" if am > bm else "=")
            bwin += bm > am; awin += am > bm; tie += bm == am
            print(f"    {k:18} A={am*100:>3.0f}%  B={bm*100:>3.0f}%  {mark}")
        print(f"    paired sign test: B>A in {bwin} pairs, A>B in {awin}, tie in {tie}")
    print(f"\n=== VERDICT ===")
    for kk, vv in v.items():
        print(f"  {kk}: {vv}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    return t, v


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--stage", required=True, choices=["crystallize", "headroom", "deliver", "all"])
    ap.add_argument("--reps", type=int, default=3)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--none-ceiling", type=float, default=0.5,
                    help="none-rate at/above this => ceiling-limited, report UNDERPOWERED not tie")
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "cfg"); build_cold_cfg(cfg)
    judge_cfg = os.path.join(ROOT, "judge-cfg"); build_cold_cfg(judge_cfg)

    if a.stage in ("crystallize", "all"):
        print("=== STAGE: crystallize (A=topic, B=question; same evidence) ===")
        notes = crystallize(cfg, a.workers)
        for k in PAIRS:
            print(f"\n--- {k} ---\n  [A/topic]    {notes[k]['A']}\n  [B/question] {notes[k]['B']}")

    if a.stage in ("headroom", "all"):
        print("\n=== STAGE: headroom (none-condition — is there room for a note to help?) ===")
        results = deliver(cfg, judge_cfg, a.reps, a.workers, ["none"], notes={})
        json.dump(results, open(os.path.join(ROOT, "headroom-results.json"), "w"), indent=1)
        t = tally(results)
        none_rate = t.get("none", {}).get("rate", 0.0)
        print(f"\nnone-rate = {none_rate*100:.0f}%  (ceiling gate = {a.none_ceiling*100:.0f}%)")
        if none_rate >= a.none_ceiling:
            print("  ==> CEILING-LIMITED: cold opus already applies the principle. A null A-vs-B result "
                  "here would be UNDERPOWERED, not a tie. Strengthen the corpus (more idiosyncratic) "
                  "before spending on deliver.")
        else:
            print(f"  ==> HEADROOM CONFIRMED ({(1-none_rate)*100:.0f}% of cases fail cold). Proceed to deliver.")

    if a.stage in ("deliver", "all"):
        print("\n=== STAGE: deliver (none / +A / +B, blind opus judge) ===")
        if not os.path.exists(NOTES_PATH):
            sys.exit(f"no {NOTES_PATH} — run --stage crystallize first")
        notes = json.load(open(NOTES_PATH))
        results = deliver(cfg, judge_cfg, a.reps, a.workers, ["none", "A", "B"], notes)
        json.dump(results, open(os.path.join(ROOT, "deliver-results.json"), "w"), indent=1)
        _print_tally(results, a.none_ceiling)


if __name__ == "__main__":
    main()
