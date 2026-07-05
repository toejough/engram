"""Reasoning-mode tests — idiosyncratic instantiations of the literature forms in
docs/research/2026-06-23-reasoning-modes.md, deleted 2026-07; git log (deduction / induction / abduction).

Each case gives the premises in-prompt (pure reasoning capability; no memory yet) and asks for the
conclusion + how certain it can be. An independent sonnet judge checks BOTH:
  - the agent reaches the target conclusion for that logical form, AND
  - the certainty is right: deduction = necessary; induction/abduction = only probable/defeasible.
(Novel tokens => the answer can't be recalled from the famous example; it must be reasoned.)

Usage: python3 reasoning_eval.py [--n 3] [--modes deduction,induction,abduction] [--workers 6]
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

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/reasoning-eval")

# Each case: premises, question, target conclusion, required certainty, and a one-line form label.
CASES = {
    "deduction": [
        (["All zorbs are flicks.", "Trell is a zorb."], "What can you conclude about Trell?",
         "Trell is a flick", "necessary", "syllogism (Socrates/mortal)"),
        (["Every marble in jar-K is opaque.", "This marble was taken from jar-K."],
         "Is this marble opaque?", "this marble is opaque", "necessary", "beans (deductive)"),
        (["If the vorl is active, the gantry locks.", "The vorl is active."],
         "What is the state of the gantry?", "the gantry locks/is locked", "necessary", "modus ponens"),
        (["Every quibit has exactly three prongs.", "X is a quibit."],
         "How many prongs does X have?", "three prongs", "necessary", "geometry"),
        (["Pim outranks Dax.", "Dax outranks Wol."], "What is the rank relation between Pim and Wol?",
         "Pim outranks Wol", "necessary", "transitivity"),
    ],
    "induction": [
        (["Five marbles were drawn from jar-K and all five were opaque."],
         "What can you infer about the marbles in jar-K?",
         "(probably) all marbles in jar-K are opaque", "probable", "beans (inductive)"),
        (["The zellithe flower has bloomed every spring for the last 40 recorded years."],
         "What about next spring?", "it will (probably) bloom next spring", "probable", "sunrise"),
        (["Every quibit examined so far has had three prongs."],
         "What can you infer about quibits in general?", "(probably) all quibits have three prongs",
         "probable", "black crows"),
        (["Every grix sampled in region-A has been blue."],
         "Are all grix blue?", "(probably) all grix are blue, but this could be falsified",
         "probable", "swans/defeasible"),
        (["Heating sample-1, sample-2, and sample-3 of alloy-Z each made it glow."],
         "What happens if you heat alloy-Z?", "(probably) heating alloy-Z makes it glow", "probable",
         "thermal expansion"),
    ],
    "abduction": [
        (["Every marble in jar-K is opaque.", "Here is a loose opaque marble of unknown origin."],
         "Where did this marble most likely come from?",
         "you CANNOT warrant 'from jar-K': that affirms the consequent (opaque isn't exclusive to "
         "jar-K); jar-K is at most a weak guess absent a uniqueness premise",
         "weak/unwarranted", "beans (abductive — fallacy guard)"),
        (["The courtyard tiles are wet this morning.", "If the sprinklers ran overnight, the tiles would be wet.",
          "It did not rain."], "What most likely happened?", "the sprinklers (probably) ran overnight",
         "probable", "wet grass + distractor"),
        (["A stranger has deeply ink-stained fingertips, carries a jeweller's loupe, and squints when reading."],
         "What is this person's most likely occupation?",
         "(probably) a watchmaker / engraver / jeweller (fine close-up work)", "probable", "Holmes"),
        (["The reactor shows rising pressure, an audible hiss, and a steadily dropping coolant level.",
          "A coolant-line leak would produce all three."], "What is the most likely cause?",
         "(probably) a coolant-line leak", "probable", "diagnosis"),
        (["There are fresh claw-marks on the bark and tufts of red fur caught in it."],
         "What most likely happened here?", "a red-furred animal (e.g. fox/cat) recently climbed the tree",
         "probable", "footprints"),
    ],
}

PROMPT_TMPL = (
    "Facts:\n{facts}\n\nQuestion: {q}\n\nReason it through, then give your conclusion AND state how "
    "certain you can be — is the conclusion NECESSARILY true given the facts, or only PROBABLE?")

JUDGE_TMPL = (
    "Grade a reasoning answer.\n\nFACTS+QUESTION:\n{facts}\nQ: {q}\n\n"
    "TARGET CONCLUSION: {target}\n"
    "REQUIRED CERTAINTY: {certainty}  (deduction => 'necessary'; induction/abduction => 'probable', "
    "must be hedged/defeasible, NOT stated as certain)\n\n"
    "ANSWER:\n{answer}\n\n"
    "HIT only if the answer reaches the target conclusion AND frames its certainty correctly "
    "(necessary vs probable). Over-claiming a probable inference as certain, or hedging a necessary one, "
    "is a MISS. Reply 'HIT' or 'MISS' on line 1, then one sentence.")


def _run(prompt, cfg, model):
    env = dict(os.environ); env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "6000"
    wd = tempfile.mkdtemp(dir=os.path.join(ROOT, "ws"))
    args = ["claude", "-p", prompt, "--output-format", "json", "--model", MODELS[model],
            "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45):
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


def run_one(mode, case_i, cfg, idx):
    facts, q, target, certainty, label = CASES[mode][case_i]
    factstr = "\n".join("- " + f for f in facts)
    out = _run(PROMPT_TMPL.format(facts=factstr, q=q), cfg, "opus")
    answer = out.get("result") or ""
    j = _run(JUDGE_TMPL.format(facts=factstr, q=q, target=target, certainty=certainty,
                               answer=answer or "(none)"), cfg, "sonnet")
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return {"mode": mode, "case": case_i, "label": label, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + (j.get("total_cost_usd", 0) or 0),
            "answer": answer}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--modes", default="deduction,induction,abduction")
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()
    modes = a.modes.split(",")

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "cfg"); build_cold_cfg(cfg)   # clean: pure reasoning, no memory/skills

    jobs = [(m, ci, i) for m in modes for ci in range(len(CASES[m])) for i in range(a.n)]
    print(f"reasoning tests: modes={modes} cases={ {m:len(CASES[m]) for m in modes} } n={a.n} = {len(jobs)} trials")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, m, ci, cfg, i): (m, ci, i) for m, ci, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['mode']:10} c{r['case']} {r['label'][:22]:22} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")

    print(f"\n=== REASONING-MODE hit rate (conclusion + correct certainty) ===")
    for m in modes:
        v = [r for r in results if r["mode"] == m]
        h = sum(r["hit"] for r in v)
        print(f"  {m:10} {h}/{len(v)} ({100*h/len(v):.0f}%)")
        for ci in range(len(CASES[m])):
            cv = [r for r in v if r["case"] == ci]
            ch = sum(r["hit"] for r in cv)
            print(f"     c{ci} {CASES[m][ci][4][:26]:26} {ch}/{len(cv)}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "reasoning-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
