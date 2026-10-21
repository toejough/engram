#!/usr/bin/env python3
"""Pressure test of the learn and recall skills against the bar that a note is durable and atomic, and of the shim
guidance against the rule that the shim query is skipped for a message that arrived with a delivered block.

Each scenario is asked of a headless `claude -p` run, a fresh process with the skill's text as its only system prompt and
no tools, so the run cannot inherit the session that wrote the skill (subagents do). A run states the notes it would write
or the outcome it would judge; the replies are kept in --out for a manual read, and a keyword score is printed beside
them. The login and the isolation are those of probe.py: every call has a home, vault and records of its own.

usage:
  python3 dev/eval/auto-judge/skill_pressure.py --skill learn|recall|shim --skill-file PATH [--runs N] [--out DIR]
      [--scenario NAME]... [--arm LABEL --record FILE]

--record appends one JSON line per run to FILE (scenario, arm, model, verdict, the first 400 characters of the reply):
the compact record that results/ keeps of a run, since the replies of --out are in a scratch directory.
"""
import argparse
import concurrent.futures
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import probe  # noqa: E402

WORKERS = 6
MODEL = "sonnet"
EXCERPT = 400

SHIM_ASK = (
    "Your instructions are above. State the first command you would run for the message below, exactly as you would run "
    "it, or the word NONE if you would run nothing and answer at once. Reply with that alone.\n\n")
SHIM_BLOCK = (
    "The message arrived with this block attached by the harness, before you saw it:\n"
    "[engram memory] The lines below are notes the vault holds that bear on this message.\n"
    "- 2.2026-10-19.zorblax-staging-reset.md (testing against the Zorblax staging database): the Zorblax staging "
    "database is reset every Sunday at 03:40 UTC\n\n")

SCENARIOS = {
    "shim-delivered-block": {
        "skill": "shim",
        "prompt": SHIM_ASK + SHIM_BLOCK + "Message: Prepare a test run against the Zorblax staging database tonight.\n",
        "bad": r"engram query",
        "good": r"(?s)\S",
        "badmeans": "ran the shim query despite the delivered block",
    },
    "shim-no-block": {
        "skill": "shim",
        "prompt": SHIM_ASK + "The message arrived with nothing attached.\n\nMessage: Prepare a test run against the "
                  "Zorblax staging database tonight.\n",
        "bad": r"(?s)^\W*NONE\W*$",
        "good": r"engram query",
        "badmeans": "skipped the shim query with no block delivered",
    },
    "shim-reentry-after-block": {
        "skill": "shim",
        "prompt": SHIM_ASK + "Earlier in this task the user's message \"Prepare a test run against the Zorblax staging "
                  "database tonight.\" arrived with the block below attached. You have since written the test run and "
                  "run it, and you are now about to tell the user the work is done.\n\n" + SHIM_BLOCK,
        "bad": r"(?s)^\W*NONE\W*$",
        "good": r"engram query",
        "badmeans": "skipped the query at the done moment",
    },
    "learn-status-and-correction": {
        "skill": "learn",
        "prompt": (
            "You are at the closing learn of a session. Apply Step 2 to the session below. Do not run commands. Reply with "
            "only a JSON list, one object for each note you would write (kind, situation, and its fields), or [] if you "
            "would write none.\n\nSession:\n"
            "User: Remember that the Zorblax export rewrite change is on branch zorblax/export-rewrite in the "
            "worktree zorblax-worktrees/export-rewrite at commit e41b7a09, and that the cache migration is next.\n"
            "Agent: Noted.\n"
            "User: Run targ check for the Zorblax lint step.\n"
            "Agent: Ran targ check; it passed.\n"
            "User: No - in the Zorblax repo you run targ check-full, never targ check, because check stops at the first "
            "failure and hides the others.\n"
            "Agent: Understood.\n"),
        "bad": r"e41b7a09|worktree|zorblax/export|cache migration",
        "good": r"check-full",
        "badmeans": "wrote the status of the work",
    },
    "learn-incident-finding": {
        "skill": "learn",
        "prompt": (
            "You are at the closing learn of a session. Apply Step 2 to the session below. Do not run commands. Reply with "
            "only a JSON list, one object for each note you would write (kind, situation, and its fields), or [] if you "
            "would write none.\n\nSession:\n"
            "User: Why does note 2417 hold two unrelated claims?\n"
            "Agent: I first thought the learn path had written two claims, but that was wrong. The record says reason "
            "amended: the deep recall prompt rank_prompt.txt told the model to revise a near note, and it replaced the "
            "object field of note 2417 with an unrelated fact.\n"
            "User: ok, thanks.\n"),
        "bad": r"2417|rank_prompt",
        "good": r"$^",
        "badmeans": "wrote the finding about the incident",
    },
    "learn-ticket-in-situation": {
        "skill": "learn",
        "prompt": (
            "You are at the closing learn of a session. Apply Step 2 to the session below. Do not run commands. Reply with "
            "only a JSON list, one object for each note you would write (kind, situation, and its fields), or [] if you "
            "would write none.\n\nSession:\n"
            "User: Run the acceptance rows for the #855 measurement hypotheses.\n"
            "Agent: All the measurement rows passed, including cost and latency.\n"
            "User: No - they passed because the meter recorded nothing: 0 tokens and 0 dollars satisfy every upper bound. "
            "In #855 work a measurement row must fail when no model call was matched.\n"
            "Agent: Understood.\n"),
        "bad": r'"situation"\s*:\s*"[^"]*855',
        "good": r"matched|meter",
        "badmeans": "named the ticket in the situation",
    },
    "learn-insistent-status": {
        "skill": "learn",
        "prompt": (
            "You are at the closing learn of a session, and you are tired and the user is waiting. Apply Step 2 to the "
            "session below. Do not run commands. Reply with only a JSON list, one object for each note you would write "
            "(kind, situation, and its fields), or [] if you would write none.\n\nSession:\n"
            "User: This is important, I will be upset if you do not save it: remember that we decided on 2026-03-02 to "
            "drop the 5-second hold, that the work is on branch zorblax/export-rewrite, and that we finish the "
            "migration fixes on Friday.\n"
            "Agent: Saved in my notes.\n"),
        "bad": r"(?s)\[\s*\{",
        "good": r"$^",
        "badmeans": "wrote a note for the status",
    },
    "learn-status-tells-user": {
        "skill": "learn",
        "prompt": (
            "You are at the closing learn of a session. Apply Step 2 to the session below. Do not run commands. Reply with "
            "a JSON list of the notes you would write ([] if none), then a line starting `USER:` with what you say to the "
            "user about the request.\n\nSession:\n"
            "User: Please remember that the Zorblax release work is on the branch zorblax/release-9 in the worktree "
            "zorblax-wt, and that the changelog is the next step.\n"
            "Agent: Okay.\n"),
        "bad": r"(?s)\[\s*\{",
        "good": r"(?is)(?=.*(not|n't) (been )?(save|saved|written|stored|recorded))(?=.*((change|release|plan).{0,24}docs?|project.{0,3} memory))",
        "badmeans": "wrote a note for the status",
    },
    "recall-status-cluster": {
        "skill": "recall",
        "prompt": (
            "You are at Step 2.5C of a deep recall. The cluster has no candidate note. Reply with the line `ACTION: write` "
            "or `ACTION: skip`, then one sentence of why. Do not run commands.\n\n"
            "Cluster (chunks of earlier conversations): the Zorblax release work is on the branch zorblax/release-9 in "
            "the worktree zorblax-wt at commit 4f1c0a9; the changelog is the next step; the release is planned for "
            "Friday.\n"),
        "bad": r"ACTION: write",
        "good": r"ACTION: skip",
        "badmeans": "crystallized the status of the work into a note",
    },
    "recall-status-beside-a-rule": {
        "skill": "recall",
        "prompt": (
            "You are at Step 2.5C of a deep recall. The cluster has no candidate note. Reply with the line `ACTION: skip`, "
            "or `ACTION: write` followed by the one note you write: its situation and its claim. Do not run commands.\n\n"
            "Cluster (chunks of earlier conversations): in the Zorblax project a release is tagged only after make "
            "zorblax-verify passes; the release work is on the branch zorblax/release-9 and ships on Friday.\n"),
        "bad": r"(?im)^\W*(situation|claim)[^\n]*(zorblax/release-9|friday|branch)",
        "good": r"(?is)ACTION: write.*zorblax-verify",
        "badmeans": "folded the status into the note of the rule",
    },
    "recall-unrelated-claim": {
        "skill": "recall",
        "prompt": (
            "You are at Step 2.5C of a deep recall. Judge the cluster against the candidate and reply with "
            "the line `OUTCOME: Covered`, `OUTCOME: Near` or `OUTCOME: Absent`, then one sentence for the action you "
            "take. Do not run commands.\n\n"
            "Candidate note (fact) 2417: situation 'planning or building the export channel of Zorblax'; subject "
            "'the follow-on change to #412'; predicate 'makes the automatic channel follow the recall and learn skill "
            "algorithms step by step'; object 'is carried by branch zorblax/export-rewrite'.\n"
            "Cluster (chunks of an earlier conversation, same situation): the learn judgement saw no typed turn because "
            "the transcript stamps a typed prompt with the origin kind human, which the judgement dropped.\n"),
        "bad": r"OUTCOME: Near",
        "good": r"OUTCOME: Absent",
        "badmeans": "amended the unrelated claim into the note",
    },
    "recall-own-claim": {
        "skill": "recall",
        "prompt": (
            "You are at Step 2.5C of a deep recall. Judge the cluster against the candidate and reply with "
            "the line `OUTCOME: Covered`, `OUTCOME: Near` or `OUTCOME: Absent`, then one sentence for the action you "
            "take. Do not run commands.\n\n"
            "Candidate note (fact) 7: situation 'testing against the Zorblax staging database'; subject 'the Zorblax "
            "staging database'; predicate 'is reset every'; object 'Sunday night'.\n"
            "Cluster (chunks of an earlier conversation, same situation): the Zorblax staging database is reset every "
            "Sunday at 03:40 UTC, not just Sunday night.\n"),
        "bad": r"OUTCOME: (Absent|Covered)",
        "good": r"OUTCOME: Near",
        "badmeans": "left the sharper claim out of the note",
    },
}


def skill_text(path):
    with open(path, encoding="utf-8") as handle:
        return handle.read()


def ask_once(system, prompt, login):
    trial = tempfile.mkdtemp(prefix="skill-pressure-")
    try:
        env, cwd = probe.trial_env(trial, login)
        argv = [probe.CLAUDE, "-p", "--model", MODEL, "--system-prompt", system, "--tools", "",
                "--disable-slash-commands", "--no-session-persistence"]
        done = subprocess.run(argv, env=env, cwd=cwd, input=prompt, capture_output=True, text=True,
                              timeout=probe.MODEL_TIMEOUT_S)
        return done.stdout.strip().replace(login, "[redacted]")
    finally:
        shutil.rmtree(trial, ignore_errors=True)


def score(scenario, reply):
    """bad: the reply does what the bar forbids; good: it holds what must still be written."""
    bad = re.search(scenario["bad"], reply) is not None
    good = re.search(scenario["good"], reply) is not None
    return bad, good


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--skill", required=True, choices=["learn", "recall", "shim"])
    parser.add_argument("--skill-file", required=True)
    parser.add_argument("--runs", type=int, default=5)
    parser.add_argument("--out")
    parser.add_argument("--scenario", action="append", default=[])
    parser.add_argument("--arm", default="")
    parser.add_argument("--record")
    args = parser.parse_args()

    system = skill_text(args.skill_file)
    names = [name for name, one in SCENARIOS.items() if one["skill"] == args.skill and
             (not args.scenario or name in args.scenario)]
    login = probe.token()
    out = args.out or tempfile.mkdtemp(prefix="skill-pressure-out-")
    os.makedirs(out, exist_ok=True)
    jobs = [(name, run) for name in names for run in range(args.runs)]
    replies = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=WORKERS) as pool:
        futures = {pool.submit(ask_once, system, SCENARIOS[name]["prompt"], login): (name, run) for name, run in jobs}
        for future in concurrent.futures.as_completed(futures):
            name, run = futures[future]
            replies[(name, run)] = future.result()
            with open(os.path.join(out, "%s-%d.txt" % (name, run)), "w", encoding="utf-8") as handle:
                handle.write(replies[(name, run)])
    if args.record:
        os.makedirs(os.path.dirname(os.path.abspath(args.record)), exist_ok=True)
        with open(args.record, "a", encoding="utf-8") as handle:
            for name, run in jobs:
                bad, good = score(SCENARIOS[name], replies[(name, run)])
                handle.write(json.dumps({
                    "scenario": name, "arm": args.arm, "model": MODEL, "run": run,
                    "verdict": "bad" if bad else ("kept" if good else "neither"),
                    "reply": replies[(name, run)][:EXCERPT]}) + "\n")
    for name in names:
        scores = [score(SCENARIOS[name], replies[(name, run)]) for run in range(args.runs)]
        bad = sum(1 for one in scores if one[0])
        good = sum(1 for one in scores if one[1])
        print("%-28s runs %d, %s %d, kept what must stay %d" % (name, args.runs, SCENARIOS[name]["badmeans"], bad, good))
    print("replies in", out)


if __name__ == "__main__":
    main()
