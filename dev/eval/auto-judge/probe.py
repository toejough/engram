#!/usr/bin/env python3
"""Probe of the turn judgement of automatic memory (793e6g14, 793e6g17).

The judgement of a turn is a call of a model, so a change to its instructions is measured by asking the model many
times about a turn whose answer is known. This probe builds the request the way the product does, with the real
`engram auto judge` over a fixture record and a fixture vault, asks the model the way the mod does (haiku, low
effort, the `system` and the `prompt` of the request) a number of times for each variant of the instructions, and
counts what the replies hold. The fixtures are the turns of the live scenarios of 793a2c and 793a3c.

usage:
  python3 dev/eval/auto-judge/probe.py --engram PATH [--runs N] [--fixture NAME]...
      [--variant NAME=FILE]... [--without NAME=TEXT]... [--list]

  --engram    the engram binary that builds the requests: build it with `go build -o PATH ./cmd/engram`, and not the
              binary of your own install
  --runs      how many times each variant is asked about each fixture (default 10)
  --fixture   only these fixtures of fixtures.json (default all)
  --variant   a variant of the instructions: the text of FILE replaces the instructions of the request
  --without   a variant of the instructions: TEXT, whose spaces match any white space, is removed from them; it must
              occur once
  --list      print the fixtures and what each expects, and end
The variant `current`, the instructions of the binary, always runs first.

A fixture has a "stage": "judge" (the default, the lessons of a turn), "moment" (the depth the cheap tier chooses for a
prompt: the reply is {"depth": ...}) or "deep" (the candidate judgement of a deep recall over the notes of the fixture
and its "passages": the reply is the object of the candidates and the lessons). A note of a fixture is a fact, or a
feedback note when it has a "behavior"; its "source" may hold {{SESSION}}, the session of the trial, so that the note
reads as written automatically in it (855b117). "fillers" are facts that crowd the lookup as the many notes of a real
vault do, so that a note of the session is not among the nearest.

Each reply is counted as: none (the reply holds no lesson), lesson (a lesson with the words the fixture expects),
covered (a lesson whose covered names a note), or one of the misses: a reply that is no list, no lesson where one is
expected, a lesson without the expected words, a lesson where none is expected, or a note written where the vault
already states the lesson.

The login is read from CLAUDE_CODE_OAUTH_TOKEN or ~/.config/claude-key.env, and goes to the environment of the model
call only. Every call is isolated by dev/eval/isolation.py: its home, vault, chunk index and records are in a directory
of its own, outside any repository.

Example, the sentence of 793b65 about an approach the agent was unsure of:
  python3 dev/eval/auto-judge/probe.py --engram /path/to/engram --fixture unsure-approach --runs 40 \\
      --without 'without-the-sentence=An approach the agent was unsure of, tried and saw succeed is a lesson even when
      the user asked for the try: "I do not know whether the flag makes the check pass", then a run, then "the check
      passed", leaves the lesson that the flag makes the check pass.'
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
import uuid

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import isolation  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
CLAUDE = os.path.expanduser("~/.local/bin/claude")
CREDENTIALS = os.path.expanduser("~/.config/claude-key.env")
MODEL_TIMEOUT_S = 120
WORKERS = 8
# The secrets of the fixtures are built here and never written whole in a file: a scanner of the repository would
# take a literal for a credential.
SECRETS = {
    "{{DEPLOY_TOKEN}}": "gh" + "p_" + "aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3",
    "{{DEPLOY_PASSWORD}}": "hunter" + "2hunter2",
}


def filler_notes(fillers):
    """Facts that crowd the lookup, as the many notes of a real vault do: each filler is [situation, subject, object]."""
    return [{"slug": "filler-%d" % number, "situation": situation, "subject": subject, "predicate": "is",
             "object": value, "source": "probe"} for number, (situation, subject, value) in enumerate(fillers)]


def learn_notes(engram, env, cwd, session, notes):
    """Write the notes of a fixture to the vault of the trial: a fact, or a feedback note when it has a behavior."""
    for note in notes:
        source = note.get("source", "probe").replace("{{SESSION}}", session)
        common = [engram, "learn"]
        if "behavior" in note:
            body = ["feedback", "--behavior", note["behavior"], "--impact", note["impact"], "--action", note["action"]]
        else:
            body = ["fact", "--subject", note["subject"], "--predicate", note["predicate"], "--object", note["object"]]
        arguments = common + body[:1] + ["--slug", note["slug"], "--position", "top", "--source", source,
                                         "--situation", note["situation"]] + body[1:]
        done = subprocess.run(arguments, env=env, cwd=cwd, capture_output=True, text=True, timeout=MODEL_TIMEOUT_S)
        if done.returncode != 0:
            sys.exit("engram learn failed for %s: %s" % (note["slug"], done.stderr.strip()[:300]))


def token():
    """The login of the model call: from the environment, else from the credential file of the owner."""
    value = os.environ.get("CLAUDE_CODE_OAUTH_TOKEN", "")
    if value:
        return value
    try:
        with open(CREDENTIALS, encoding="utf-8") as handle:
            for line in handle:
                found = re.match(r"^\s*(?:export\s+)?CLAUDE_CODE_OAUTH_TOKEN=(.*?)\s*$", line)
                if found:
                    return found.group(1).strip("\"'")
    except OSError:
        pass
    sys.exit("no login: set CLAUDE_CODE_OAUTH_TOKEN or put it in ~/.config/claude-key.env")


def load_fixtures(path=None):
    """The fixtures of fixtures.json, with the placeholders of the secrets filled in."""
    with open(path or os.path.join(HERE, "fixtures.json"), encoding="utf-8") as handle:
        text = handle.read()
    for placeholder, secret in SECRETS.items():
        text = text.replace(placeholder, secret)
    return json.loads(text)


def trial_env(trial, login=None):
    """The environment of every command of a trial: nothing of the operator's, every state path in the trial."""
    home = os.path.join(trial, "home")
    cwd = os.path.join(trial, "work")
    for directory in (os.path.join(home, ".claude"), cwd):
        os.makedirs(directory, exist_ok=True)
    base = {
        "PATH": "/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin",
        "HOME": home,
        "TMPDIR": trial,
        "LANG": "en_US.UTF-8",
        "DISABLE_AUTOUPDATER": "1",
    }
    if login:
        base["CLAUDE_CODE_OAUTH_TOKEN"] = login
    env = isolation.isolated_env(os.path.join(home, ".claude"), trial, cwd=cwd, base=base)
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(trial, "transcripts")
    return env, cwd


def record_of(rows):
    """The lines of a session record for the rows of a fixture: a prompt, text of the agent, or a tool the agent used."""
    lines = []
    for number, row in enumerate(rows):
        if "user" in row:
            lines.append({"type": "user", "message": {"role": "user", "content": row["user"]},
                          "promptSource": "sdk", "turnOrigin": "sdk"})
        elif "agent" in row:
            lines.append({"type": "assistant", "message": {"role": "assistant",
                                                           "content": [{"type": "text", "text": row["agent"]}]}})
        elif "tool" in row:
            ident = "toolu_%04d" % number
            lines.append({"type": "assistant", "message": {"role": "assistant", "content": [
                {"type": "tool_use", "id": ident, "name": row["tool"], "input": {}}]}})
            lines.append({"type": "user", "message": {"role": "user", "content": [
                {"type": "tool_result", "tool_use_id": ident, "content": row.get("result", "")}]}})
        else:
            sys.exit("a row of a fixture is a prompt, text of the agent or a tool: %r" % (row,))
    return "\n".join(json.dumps(line) for line in lines) + "\n"


def drop_once(system, text):
    """The instructions without TEXT, whose spaces match any white space; TEXT must occur exactly once."""
    pattern = r"\s+".join(re.escape(word) for word in text.split())
    found = list(re.finditer(pattern, system))
    if len(found) != 1:
        sys.exit("the text to remove occurs %d times in the instructions: %r" % (len(found), text[:80]))
    start, end = found[0].span()
    return system[:start] + system[end:]


def window_of(fixture):
    """The window of a moment: a line for each row of the conversation, then the prompt (a tool row is a line too)."""
    lines = []
    for row in fixture["rows"]:
        if "user" in row:
            lines.append("User: " + row["user"])
        elif "agent" in row:
            lines.append("Agent: " + row["agent"])
        elif "tool" in row:
            lines.append("Agent used " + row["tool"])
    lines.append("User: " + fixture["turn"]["text"])
    return "\n".join(lines)


def run_engram(engram, arguments, env, cwd, stdin, what):
    """One run of the engram binary that prints a step; its request, the object the mod asks the model with."""
    done = subprocess.run([engram] + arguments, env=env, cwd=cwd, input=stdin, capture_output=True, text=True,
                          timeout=MODEL_TIMEOUT_S)
    if done.returncode != 0 or not done.stdout.strip():
        sys.exit("engram %s: exit %d, %s" % (what, done.returncode, done.stderr.strip()[:300]))
    step = json.loads(done.stdout)
    return step.get("request", step)


def request_for(engram, name, fixture):
    """The request the judgement prints for a fixture, built over a trial of its own."""
    trial = tempfile.mkdtemp(prefix="auto-judge-probe-")
    try:
        env, cwd = trial_env(trial)
        session = str(uuid.uuid4())
        stage = fixture.get("stage", "judge")
        learn_notes(engram, env, cwd, session, filler_notes(fixture.get("fillers", [])) + fixture.get("notes", []))
        project = os.path.join(trial, "transcripts", "probe")
        os.makedirs(project, exist_ok=True)
        with open(os.path.join(project, session + ".jsonl"), "w", encoding="utf-8") as handle:
            handle.write(record_of(fixture["rows"]))
        if stage == "moment":
            moment = json.dumps({"session": session, "cue": "prompt", "origin": "composer",
                                 "text": fixture["turn"]["text"]})
            return run_engram(engram, ["auto", "moment"], env, cwd, moment, "auto moment for " + name)
        if stage == "deep":
            recall = json.dumps({"session": session, "reply": json.dumps({"phrases": fixture["phrases"]}),
                                 "carry": {"cue": "prompt", "depth": "deep", "window": window_of(fixture)}})
            request = run_engram(engram, ["auto", "recall"], env, cwd, recall, "auto recall for " + name)
            lines = "\n".join("- %s (%s; passage): %s" % (one["id"], one["date"], one["text"])
                              for one in fixture.get("passages", []))
            request["prompt"] = request["prompt"].replace("\n</candidates>", "\n" + lines + "\n</candidates>")
            return request
        turn = json.dumps({"session": session, "turn": "probe", "text": fixture["turn"]["text"],
                           "answer": fixture["turn"]["answer"], "origin": "composer", "reply": '{"learn":true}'})
        return run_engram(engram, ["auto", "judge"], env, cwd, turn, "auto judge for " + name)
    finally:
        shutil.rmtree(trial, ignore_errors=True)


def ask(request, system, login):
    """One reply of the model to the request, with the given instructions."""
    trial = tempfile.mkdtemp(prefix="auto-judge-probe-ask-")
    try:
        env, cwd = trial_env(trial, login)
        argv = [CLAUDE, "-p", "--model", request["model"], "--system-prompt", system, "--tools", "",
                "--disable-slash-commands", "--no-session-persistence"]
        if request.get("effort"):
            argv += ["--effort", request["effort"]]
        done = subprocess.run(argv, env=env, cwd=cwd, input=request["prompt"], capture_output=True, text=True,
                              timeout=MODEL_TIMEOUT_S)
        return done.stdout.strip().replace(login, "[redacted]")
    finally:
        shutil.rmtree(trial, ignore_errors=True)


def verdict(reply, expect):
    """What a reply holds, against what the fixture expects."""
    text = re.sub(r"^```(?:json)?\s*|\s*```$", "", reply.strip())
    try:
        lessons = json.loads(text)
    except ValueError:
        return "invalid"
    if not isinstance(lessons, list):
        return "invalid"
    if expect.get("not_written"):
        # an empty list proves nothing (the reply may drop the lesson for another reason): suppression is a lesson
        # whose covered names the note of the session (855b117, the note 1377a)
        if any(not lesson.get("covered") for lesson in lessons):
            return "written (a miss)"
        if not lessons:
            return "none (no lesson names the covering note, a miss)"
        named = expect.get("covered_by", "")
        if not any(named in str(lesson.get("covered")) for lesson in lessons):
            return "covered by another note (a miss)"
        return "suppressed"
    if not lessons:
        return "none (a miss)" if expect.get("lesson") or expect.get("covered") else "none"
    if expect.get("covered"):
        return "covered" if any(lesson.get("covered") for lesson in lessons) else "written (a miss)"
    if not expect.get("lesson"):
        return "lesson (a miss)"
    words = [word.lower() for word in expect["lesson"]]
    held = any(all(word in json.dumps(lesson).lower() for word in words) for lesson in lessons)
    if not held:
        return "lesson without the words (a miss)"
    if any(word.lower() in json.dumps(lesson).lower() for lesson in lessons for word in expect.get("absent", [])):
        return "lesson with a word it must lack (a miss)"
    if any(word.lower() in str(lesson.get("situation", "")).lower() for lesson in lessons
           for word in expect.get("situation_lacks", [])):
        return "lesson with a situation that names the work (a miss)"
    return "lesson"


def moment_verdict(reply, expect):
    """The depth a reply of the moment judgement chooses, against the depths the fixture allows."""
    text = re.sub(r"^```(?:json)?\s*|\s*```$", "", reply.strip())
    try:
        depth = json.loads(text)["depth"]
    except (ValueError, KeyError, TypeError):
        return "invalid"
    return depth if depth in expect["depth"] else depth + " (a miss)"


def deep_verdict(reply, expect):
    """What a reply of a deep recall revises and writes, against what the fixture expects of the revisions and lessons."""
    text = re.sub(r"^```(?:json)?\s*|\s*```$", "", reply.strip())
    try:
        parsed = json.loads(text)
        candidates = parsed["candidates"]
        lessons = parsed.get("lessons") or []
    except (ValueError, KeyError, TypeError):
        return "invalid"
    fields = [json.dumps(one.get("fields") or {}).lower() for one in candidates if isinstance(one, dict)]
    written = json.dumps(lessons).lower()
    if any(word.lower() in held for held in fields for word in expect.get("fields_lack", [])):
        return "revision with a foreign claim (a miss)"
    if any(word.lower() in written for word in expect.get("lessons_lack", [])):
        return "lesson with a word it must lack (a miss)"
    if "fields_have" in expect and not any(all(word.lower() in held for word in expect["fields_have"])
                                           for held in fields):
        return "no revision with the words (a miss)"
    return "ok"


def verdict_of(fixture, reply):
    """The verdict of a reply for the stage of its fixture."""
    stage = fixture.get("stage", "judge")
    if stage == "moment":
        return moment_verdict(reply, fixture["expect"])
    if stage == "deep":
        return deep_verdict(reply, fixture["expect"])
    return verdict(reply, fixture["expect"])


def variants_of(args):
    """The variants to run, each a name and a function from the instructions of the request to the ones to ask with."""
    variants = [("current", lambda system: system)]
    for spec in args.variant:
        label, _, path = spec.partition("=")
        with open(path, encoding="utf-8") as handle:
            text = handle.read()
        variants.append((label, lambda system, text=text: text))
    for spec in args.without:
        label, _, drop = spec.partition("=")
        variants.append((label, lambda system, drop=drop: drop_once(system, drop)))
    return variants


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--engram")
    parser.add_argument("--runs", type=int, default=10)
    parser.add_argument("--fixture", action="append", default=[])
    parser.add_argument("--variant", action="append", default=[])
    parser.add_argument("--without", action="append", default=[])
    parser.add_argument("--list", action="store_true")
    parser.add_argument("--fixtures", help="a fixtures file other than fixtures.json")
    parser.add_argument("--record", help="append one JSON line per run (fixture, variant, model, verdict) to this file")
    parser.add_argument("--dump", help="a directory that keeps the raw reply of every run, for a manual read")
    args = parser.parse_args()

    fixtures = load_fixtures(args.fixtures)
    if args.list:
        for name, fixture in fixtures.items():
            print("%-22s %-7s expects %-34s %s" % (name, fixture.get("stage", "judge"), json.dumps(fixture["expect"]),
                                                  fixture["about"]))
        return
    if not args.engram:
        parser.error("--engram is required")
    names = args.fixture or list(fixtures)
    login = token()
    requests = {name: request_for(args.engram, name, fixtures[name]) for name in names}
    variants = variants_of(args)

    jobs = [(name, label, change(requests[name]["system"]))
            for name in names for label, change in variants for _ in range(args.runs)]
    results = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=WORKERS) as pool:
        futures = {pool.submit(ask, requests[name], system, login): (name, label) for name, label, system in jobs}
        for future in concurrent.futures.as_completed(futures):
            name, label = futures[future]
            reply = future.result()
            kind = verdict_of(fixtures[name], reply)
            results.setdefault((name, label), []).append(kind)
            if args.dump:
                os.makedirs(args.dump, exist_ok=True)
                number = len(results[(name, label)])
                with open(os.path.join(args.dump, "%s.%s.%d.txt" % (name, label, number)), "w", encoding="utf-8") as out:
                    out.write("VERDICT: %s\n%s\n" % (kind, reply))

    if args.record:
        os.makedirs(os.path.dirname(os.path.abspath(args.record)), exist_ok=True)
        with open(args.record, "a", encoding="utf-8") as handle:
            for (name, label), kinds in sorted(results.items()):
                for number, kind in enumerate(kinds):
                    handle.write(json.dumps({"fixture": name, "variant": label, "model": requests[name]["model"],
                                             "run": number, "verdict": kind}) + "\n")
    print("%-22s %-14s %s" % ("fixture", "variant", "replies of %d runs" % args.runs))
    for name in names:
        for label, _ in variants:
            counts = {}
            for kind in results[(name, label)]:
                counts[kind] = counts.get(kind, 0) + 1
            print("%-22s %-14s %s" % (name, label, ", ".join("%s %d" % (kind, counts[kind]) for kind in sorted(counts))))


if __name__ == "__main__":
    main()
