#!/usr/bin/env python3
"""Confined headless-arm launcher for the learn-at-subagent-return eval (design D5; tasks 1.1, 1.4).

Every arm is a fresh `claude -p` process launched with `env -i` and D5's exact variable allowlist,
HOME=$ARM/home under /private/tmp, the D5 permissions (no bypassPermissions) and Seatbelt sandbox,
and a guidance set identical across arms except learn.md. Per batch:

  1. results/<batch>/ is created; the batch start time is taken; isolation "before" snapshot.
  2. The OAuth token is read from the keychain (never echoed, logged, written to a file, or put in argv:
     it is handed to the arm through an inherited pipe fd).
  3. denyWrite is generated from `ls /private/tmp/claude-<uid>`.
  4. A probe arm (batch settings + Bash(touch:*)) tries to touch a file inside an existing entry of
     /private/tmp/claude-<uid> and under the real home; both must fail while a control touch in
     $ARM/work succeeds. Otherwise the batch aborts before any arm runs.
  5. Arms run; each is gated and scored (score.py), appended to arms.jsonl and cost-log.jsonl, and
     its $ARM is deleted.
  6. Isolation "after" snapshot -> results/<batch>/isolation.json; a real-vault entry newer than the
     batch start fails the batch.
  7. run-manifest.json.

    python3 lar.py batch --name smoke --arm P1:GREEN:provisional --arm N1:RED:pin

Arm spec: CELL:ARM:LEARN_SOURCE[:DOMAIN], LEARN_SOURCE in {pin, worktree, provisional}.
  pin         learn.md at config.PIN (RED)
  worktree    agent-instructions/guidance/learn.md as it is in this checkout (GREEN after task 3.1)
  provisional learn.md at PIN with design D2's draft edits applied (smoke only, task 1.5)
"""
import argparse
import datetime as dt
import hashlib
import json
import os
import pwd
import secrets
import shutil
import subprocess
import sys
import tempfile
import threading
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

import score  # noqa: E402
from config import (ALLOWED_TOOLS, ARM_TOKENS, DISALLOWED_TOOLS, FILE_MARKERS, FIXTURE_AGENT,  # noqa: E402
                    FIXTURE_MODEL, GUIDANCE, IMPORT_LINES, MARKER_PREFIX, PIN, PROBE_EXTRA_TOOL, SKILLS)

REPO = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
RESULTS = os.path.join(HERE, "results")
KEYCHAIN_ITEM = "Claude Code-credentials"
ARM_PREFIX = "/private/tmp/engram-arm."
DEFAULT_TIMEOUT_S = 1500
LEARN_REL = "agent-instructions/guidance/learn.md"


class HarnessError(RuntimeError):
    """The harness refuses to proceed (bad input, confinement not proven, isolation failed)."""


# ---------------------------------------------------------------------------
# text sources
# ---------------------------------------------------------------------------


class GitSource:
    """Installed texts: pinned ones from `git show PIN:path`, the GREEN learn.md from the worktree."""

    def __init__(self, repo: str = REPO, pin: str = PIN):
        self.repo, self.pin = repo, pin

    def _git(self, *args: str) -> str:
        return subprocess.run(["git", "-C", self.repo, *args], check=True, capture_output=True,
                              text=True).stdout

    def pinned(self, path: str) -> str:
        return self._git("show", f"{self.pin}:{path}")

    def pinned_tree(self, prefix: str) -> Dict[str, str]:
        names = self._git("ls-tree", "-r", "--name-only", self.pin, prefix + "/").split("\n")
        return {n[len(prefix) + 1:]: self.pinned(n) for n in names if n}

    def worktree(self, path: str) -> str:
        with open(os.path.join(self.repo, path)) as f:
            return f.read()

    def head(self) -> str:
        return self._git("rev-parse", "HEAD").strip()

    def worktree_state(self, path: str) -> Dict[str, Any]:
        return {"dirty": bool(self._git("status", "--porcelain", "--", path).strip()),
                "git_blob": self._git("hash-object", path).strip()}


# Design D2, draft wording (the provisional GREEN text for the smoke only; task 3.1 applies the
# checked wording to the repo file, and scored GREEN arms use that via learn_source=worktree).
D2_OLD = "the one confirmed correction now"
D2_NEW = "the one confirmed lesson now"
D2_ANCHOR = "  `/learn` the root cause at the moment you catch it.\n"
D2_BULLET = (
    "- **A subagent just returned, and its report's `LESSONS:` line isn't `none`** — the lesson is in hand\n"
    "  now, and at the closing learn it is one line in a long list (if the closing learn runs at all).\n"
    "  Judge each lesson as you read the report: if it is a confirmed correction, reversal, or confirmed\n"
    "  approach that states a reusable rule, `/learn` it on the fast path **before your next dispatch**.\n"
    "  Skip `none`, \"done, tests pass\", and anything the report doesn't confirm (\"might be a race\") —\n"
    "  those are not lessons, and the closing learn will discard them anyway.\n"
)


def provisional_green(text: str) -> str:
    if text.count(D2_OLD) != 1 or text.count(D2_ANCHOR) != 1:
        raise HarnessError("learn.md does not have the D2 edit sites exactly once")
    return text.replace(D2_OLD, D2_NEW).replace(D2_ANCHOR, D2_ANCHOR + D2_BULLET)


def learn_text(source: str, src) -> str:
    if source == "pin":
        return src.pinned(LEARN_REL)
    if source == "worktree":
        return src.worktree(LEARN_REL)
    if source == "provisional":
        return provisional_green(src.pinned(LEARN_REL))
    raise HarnessError(f"unknown learn source {source!r}")


# ---------------------------------------------------------------------------
# fixtures
# ---------------------------------------------------------------------------


def load_fixtures() -> Dict[str, Any]:
    with open(os.path.join(HERE, "fixtures", "reports.json")) as f:
        return json.load(f)


def domain_for(cell: str, index: int) -> str:
    """P1 is Quillfeather-only and P2 Tarnbrook-only; N cells alternate by arm index."""
    domains = load_fixtures()["cells"][cell]["domains"]
    return domains[index % len(domains)]


def unit1_report(fx: Dict[str, Any], cell: str, domain: str) -> str:
    c = fx["cells"][cell]
    if domain not in c["domains"]:
        raise HarnessError(f"cell {cell} has no {domain} report")
    body = c.get("unit1_body", {}).get(domain) or fx["domains"][domain]["unit1_neutral_body"]
    return f"{body}\n\nLESSONS: {c['lessons']}"


def fixture_dir(arm: str, domain: str) -> str:
    return os.path.join(arm, "fixture", domain)


def fixture_tools(fix: str) -> List[str]:
    """The only extra permissions the arm gets (D5 amendment): edits inside the fixture checkout and
    its one check script. Read is already allowed."""
    return [f"Edit(/{fix}/**)", f"Write(/{fix}/**)", f"Bash({fix}/run-tests)"]


def build_fixture_checkout(fix: str, fx: Dict[str, Any], domain: str) -> None:
    d = fx["domains"][domain]
    for n, (rel, title) in enumerate(zip(d["unit_files"], d["unit_titles"]), 1):
        pkg = os.path.basename(os.path.dirname(rel))
        _write(os.path.join(fix, rel), f"package {pkg}\n\n// {title}\n// TODO(unit {n}): implement\n")
    lines = "".join(f"echo 'ok  \t{d['go_package']}/{os.path.dirname(rel)}\t0.0{n}2s'\n"
                    for n, rel in enumerate(d["unit_files"], 1))
    _write(os.path.join(fix, "run-tests"), "#!/bin/sh\n# fixture check: prints go-test-style results\n" + lines)
    os.chmod(os.path.join(fix, "run-tests"), 0o755)


def fixture_agent(fx: Dict[str, Any], cell: str, domain: str, fix: str) -> str:
    d = fx["domains"][domain]
    t = d["unit_titles"]
    reports = [unit1_report(fx, cell, domain), d["unit2_report"], d["unit3_report"]]
    blocks = "\n\n".join(f"=== REPORT {i} ===\n{r}\n=== END REPORT {i} ===" for i, r in enumerate(reports, 1))
    steps = "\n".join(
        f"- Unit {n} ({title}): Read `{fix}/{rel}`, then Edit it to replace the line `// TODO(unit {n}): implement` "
        f"with `// unit {n}: implemented`, then run `{fix}/run-tests` with the Bash tool."
        for n, (rel, title) in enumerate(zip(d["unit_files"], t), 1))
    return (
        "---\n"
        f"name: {FIXTURE_AGENT}\n"
        f"description: Worker for the {d['project']} units. It has the {d['project']} checkout; dispatch each "
        "numbered unit to it and it returns that unit's completion report.\n"
        f"model: {FIXTURE_MODEL}\n"
        "tools: Read, Write, Edit, Bash\n"
        "disallowedTools: Skill, Agent, Task, WebFetch, WebSearch, NotebookEdit, TodoWrite, Bash(engram:*)\n"
        "---\n\n"
        "You are a scripted test fixture. Work only inside the checkout below and run no other commands.\n\n"
        "Choose the unit this request asks you to do now. If the request mentions more than one unit, the "
        "one to do now is the one being assigned, not one described as already done.\n\n"
        f"Step 1: do that unit's three tool calls, exactly as listed:\n{steps}\n\n"
        "Step 2: reply with exactly one of the three reports below, copied verbatim and character for "
        "character, with nothing before or after it and without the === marker lines. "
        f"Unit 1 ({t[0]}) gets REPORT 1; Unit 2 ({t[1]}) gets REPORT 2; Unit 3 ({t[2]}) gets REPORT 3.\n\n"
        f"{blocks}\n"
    )


def task_prompt(fx: Dict[str, Any], domain: str) -> str:
    with open(os.path.join(HERE, "fixtures", fx["domains"][domain]["prompt"])) as f:
        return f.read()


# ---------------------------------------------------------------------------
# arm construction
# ---------------------------------------------------------------------------


@dataclass
class ArmSpec:
    cell: str
    arm: str
    learn_source: str
    domain: str


def parse_arm_spec(text: str, index: int) -> ArmSpec:
    parts = text.split(":")
    if len(parts) not in (3, 4):
        raise HarnessError(f"arm spec {text!r}: want CELL:ARM:LEARN_SOURCE[:DOMAIN]")
    cell, arm, src = parts[:3]
    if cell not in load_fixtures()["cells"]:
        raise HarnessError(f"unknown cell {cell!r}")
    if arm not in ARM_TOKENS:
        raise HarnessError(f"unknown arm {arm!r}")
    if src not in ("pin", "worktree", "provisional"):
        raise HarnessError(f"unknown learn source {src!r}")
    return ArmSpec(cell, arm, src, parts[3] if len(parts) == 4 else domain_for(cell, index))


def _sha(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()


def _write(path: str, text: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)


def settings(arm: str, deny: List[str], home: str) -> Dict[str, Any]:
    """Design D5 layer 1 (permissions.deny) and layer 2 (sandbox), exactly."""
    return {
        "sandbox": {
            "enabled": True, "failIfUnavailable": True,
            "allowUnsandboxedCommands": False, "autoAllowBashIfSandboxed": False,
            "filesystem": {"allowWrite": [arm], "denyRead": [home], "denyWrite": list(deny)},
            "network": {"allowedDomains": [], "strictAllowlist": True},
        },
        "permissions": {"deny": [f"Read(/{home}/**)"]},
    }


def make_dirs(arm: str) -> None:
    for d in ("home/.claude", "work", "tmp", "xdg", "vault", "bin"):
        os.makedirs(os.path.join(arm, d), exist_ok=True)


def build_arm(arm: str, spec: ArmSpec, src, engram_bin: str, deny_entries: List[str], home: str) -> Dict[str, Any]:
    fx = load_fixtures()
    make_dirs(arm)
    cfg = os.path.join(arm, "home", ".claude")
    texts = []
    _write(os.path.join(cfg, "settings.json"), json.dumps(settings(arm, deny_entries, home), indent=2) + "\n")
    for g in GUIDANCE:
        if g == "learn":
            body = learn_text(spec.learn_source, src)
            tail = MARKER_PREFIX + FILE_MARKERS[g] + "\n" + MARKER_PREFIX + ARM_TOKENS[spec.arm] + "\n"
            commit = PIN if spec.learn_source in ("pin", "provisional") else src.head()
            meta = {"source": spec.learn_source, "commit": commit, "content_sha256": _sha(body)}
            if spec.learn_source == "worktree":
                meta.update(src.worktree_state(LEARN_REL))
        else:
            body = src.pinned(f"agent-instructions/guidance/{g}.md")
            tail = MARKER_PREFIX + FILE_MARKERS[g] + "\n"
            meta = {"source": "pin", "commit": PIN, "content_sha256": _sha(body)}
        text = body + "\n" + tail
        _write(os.path.join(cfg, "engram", f"{g}.md"), text)
        texts.append({"path": f"engram/{g}.md", "sha256": _sha(text), **meta})
    _write(os.path.join(cfg, "CLAUDE.md"), "\n".join(IMPORT_LINES) + "\n")
    for s in SKILLS:
        for rel, body in src.pinned_tree(f"agent-instructions/skills/{s}").items():
            _write(os.path.join(cfg, "skills", s, rel), body)
            texts.append({"path": f"skills/{s}/{rel}", "commit": PIN, "sha256": _sha(body)})
    fix = fixture_dir(arm, spec.domain)
    build_fixture_checkout(fix, fx, spec.domain)
    agent = fixture_agent(fx, spec.cell, spec.domain, fix)
    _write(os.path.join(cfg, "agents", f"{FIXTURE_AGENT}.md"), agent)
    head = src.head()
    texts.append({"path": f"agents/{FIXTURE_AGENT}.md", "commit": head, "sha256": _sha(agent)})
    prompt = task_prompt(fx, spec.domain)
    texts.append({"path": f"prompt:{spec.domain}", "commit": head, "sha256": _sha(prompt)})
    shutil.copy2(engram_bin, os.path.join(arm, "bin", "engram"))
    return {"texts": texts, "prompt": prompt, "unit1_report": unit1_report(fx, spec.cell, spec.domain),
            "fixture_dir": fix, "fixture_tools": fixture_tools(fix),
            "lessons": fx["cells"][spec.cell]["lessons"]}


# ---------------------------------------------------------------------------
# launch
# ---------------------------------------------------------------------------


# The token never enters argv: the launcher writes it into a pipe, and this handoff (run under
# `env -i`, so its environment is exactly the allowlist) reads it from the inherited fd, closes the
# fd, adds it to its own environment and execs claude. argv carries only the fd number.
HANDOFF_PYTHON = os.path.realpath(sys.executable)
ENV_ALLOWLIST = ("HOME", "USER", "PATH", "TERM", "TMPDIR", "XDG_DATA_HOME", "ENGRAM_VAULT_PATH")
# The handoff execs claude with an explicit env of exactly the allowlist plus the token, dropping
# anything the interpreter itself adds (macOS python sets LC_CTYPE and __CF_USER_TEXT_ENCODING).
HANDOFF_CODE = ("import os,sys\n"
                "fd=int(sys.argv[1]);t=os.read(fd,65536).decode().strip();os.close(fd)\n"
                f"env={{k:os.environ[k] for k in {ENV_ALLOWLIST!r} if k in os.environ}}\n"
                "env['CLAUDE_CODE_OAUTH_TOKEN']=t\n"
                "os.execve(sys.argv[2],sys.argv[2:],env)\n")


def arm_argv(arm: str, model: str, user: str, claude_bin: str, token_fd: int, extra_tools=()) -> List[str]:
    env_pairs = [
        f"HOME={arm}/home", f"USER={user}", f"PATH={arm}/bin:/usr/bin:/bin", "TERM=dumb",
        f"TMPDIR={arm}/tmp", f"XDG_DATA_HOME={arm}/xdg", f"ENGRAM_VAULT_PATH={arm}/vault",
    ]
    return (["/usr/bin/env", "-i", *env_pairs, HANDOFF_PYTHON, "-I", "-c", HANDOFF_CODE, str(token_fd),
             claude_bin, "-p", "--output-format", "stream-json", "--verbose",
             "--model", model, "--allowedTools", *ALLOWED_TOOLS, *extra_tools,
             "--disallowedTools", *DISALLOWED_TOOLS])


def extract_token(keychain_json: str) -> str:
    try:
        return json.loads(keychain_json)["claudeAiOauth"]["accessToken"]
    except (ValueError, KeyError, TypeError):
        raise HarnessError("keychain item has no claudeAiOauth.accessToken") from None


def read_token() -> str:
    p = subprocess.run(["/usr/bin/security", "find-generic-password", "-s", KEYCHAIN_ITEM, "-w"],
                       capture_output=True, text=True)
    if p.returncode != 0:
        raise HarnessError(f"keychain read failed (exit {p.returncode})")
    return extract_token(p.stdout)


def run_claude(arm: str, prompt: str, token: str, model: str, out_dir: str, timeout: int,
               claude_bin: str, extra_tools=()) -> Dict[str, Any]:
    os.makedirs(out_dir, exist_ok=True)
    r, w = os.pipe()
    try:
        os.write(w, token.encode())
    finally:
        os.close(w)
    argv = arm_argv(arm, model, pwd.getpwuid(os.getuid()).pw_name, claude_bin, r, extra_tools)
    t0 = time.time()
    timed_out = False
    try:
        with open(os.path.join(out_dir, "stream.jsonl"), "w") as so, \
                open(os.path.join(out_dir, "stderr.txt"), "w") as se:
            try:
                p = subprocess.run(argv, input=prompt, stdout=so, stderr=se, text=True,
                                   cwd=os.path.join(arm, "work"), timeout=timeout, pass_fds=(r,))
                rc = p.returncode
            except subprocess.TimeoutExpired:
                rc, timed_out = None, True
    finally:
        os.close(r)
    rec = {"argv": argv, "returncode": rc, "timed_out": timed_out, "wall_s": round(time.time() - t0, 1)}
    if token in json.dumps(rec):
        raise HarnessError("token reached the launch record")
    _write(os.path.join(out_dir, "launch.json"), json.dumps(rec, indent=2) + "\n")
    return rec


def collect_sessions(arm: str, dest: str):
    """Copy every session JSONL out of $ARM before it is deleted. Returns (main texts, subagent texts)."""
    root = os.path.join(arm, "home", ".claude", "projects")
    main, others = [], []
    for dirpath, _, files in os.walk(root):
        for f in sorted(files):
            if f.endswith(".jsonl"):
                src = os.path.join(dirpath, f)
                rel = os.path.relpath(src, root)
                dst = os.path.join(dest, rel)
                os.makedirs(os.path.dirname(dst), exist_ok=True)
                shutil.copy2(src, dst)
                with open(src, errors="replace") as fh:
                    (others if "subagents" in rel.split(os.sep) else main).append(fh.read())
    return main, others


def vault_notes(vault: str) -> List[str]:
    notes = []
    for dirpath, dirnames, files in os.walk(vault):
        if ".git" in dirnames:
            dirnames.remove(".git")
        notes += [os.path.relpath(os.path.join(dirpath, f), vault) for f in files if f.endswith(".md")]
    return sorted(notes)


def remove_arm(arm: str) -> None:
    if not arm.startswith(ARM_PREFIX):
        raise HarnessError(f"refusing to delete {arm}: not an arm dir")
    shutil.rmtree(arm, ignore_errors=True)
    if os.path.exists(arm):
        raise HarnessError(f"$ARM {arm} survived deletion")


def new_arm_dir() -> str:
    return tempfile.mkdtemp(prefix="engram-arm.", dir="/private/tmp")


# ---------------------------------------------------------------------------
# guards (task 1.4)
# ---------------------------------------------------------------------------


def claude_tmp_root() -> str:
    return f"/private/tmp/claude-{os.getuid()}"


def deny_entries(root: str) -> List[str]:
    """Each existing entry of /private/tmp/claude-<uid> -- never the directory itself (note 1075)."""
    return [os.path.join(root, e) for e in sorted(os.listdir(root))]


def probe_target(entries: List[str], name: str) -> str:
    dirs = [e for e in entries if os.path.isdir(e) and not os.path.islink(e)]
    if not dirs:
        raise HarnessError("no existing directory entry under the claude tmp root to probe")
    return os.path.join(dirs[0], name)


def probe_verdict(target_exists: bool, home_exists: bool, control_exists: bool,
                  seatbelt_blocked: Optional[bool] = None) -> Dict[str, Any]:
    """seatbelt_blocked is None for the layer1+2 invocation; for the layer-2 invocation it must be True
    (the target touch ran and the OS refused it), or layer 2 was never exercised."""
    if not control_exists:
        return {"ok": False, "reason": "control touch in $ARM/work did not land; confinement not proven"}
    if target_exists:
        return {"ok": False, "reason": "ESCAPE: probe wrote inside /private/tmp/claude-<uid>"}
    if home_exists:
        return {"ok": False, "reason": "ESCAPE: probe wrote under the real home"}
    if seatbelt_blocked is False:
        return {"ok": False, "reason": "layer 2 not exercised: the target touch was not refused by the OS sandbox"}
    return {"ok": True, "reason": "touch blocked in claude tmp entry and real home; control landed"}


# Probe invocations: the task-1.4 letter (batch settings + Bash(touch:*)), which layer 1 alone can
# stop ("requires approval", path outside the working dir), and D11's layer-2 invocation with Bash
# opened, so only the Seatbelt denyWrite stands between the arm and /private/tmp/claude-<uid>.
PROBE_VARIANTS = {"layer1+2": (PROBE_EXTRA_TOOL,), "layer2": ("Bash",)}


def seatbelt_blocked(stream_lines: List[str], target: str) -> bool:
    uses = {}
    for kind, *rest in score._steps(score._events(stream_lines)):
        if kind == "use" and rest[0].get("name") == "Bash":
            uses[rest[0].get("id")] = str((rest[0].get("input") or {}).get("command", ""))
        elif kind == "result":
            cmd = uses.get(rest[0].get("tool_use_id"), "")
            if target in cmd and "Operation not permitted" in score._text_of(rest[0].get("content")):
                return True
    return False


def run_probe(batch_dir: str, token: str, deny: List[str], home: str, claude_bin: str) -> Dict[str, Any]:
    out: Dict[str, Any] = {"variants": {}, "cost_usd": 0.0}
    for variant, extra in PROBE_VARIANTS.items():
        rec = _probe_once(os.path.join(batch_dir, "probe", variant), token, deny, home, claude_bin, extra,
                          require_seatbelt=(variant == "layer2"))
        out["variants"][variant] = rec
        out["cost_usd"] += rec["cost_usd"] or 0.0
    bad = [f"{v}: {r['reason']}" for v, r in out["variants"].items() if not r["ok"]]
    out["ok"] = not bad
    out["reason"] = "; ".join(bad) if bad else "both probe invocations blocked; controls landed"
    _write(os.path.join(batch_dir, "probe", "probe.json"), json.dumps(out, indent=2) + "\n")
    return out


def _probe_once(out_dir: str, token: str, deny: List[str], home: str, claude_bin: str, extra,
                require_seatbelt: bool) -> Dict[str, Any]:
    arm = new_arm_dir()
    name = f"lar-probe-{secrets.token_hex(4)}"
    target = probe_target(deny, name)
    home_target = os.path.join(home, name)
    control = os.path.join(arm, "work", name)
    try:
        make_dirs(arm)
        _write(os.path.join(arm, "home", ".claude", "settings.json"),
               json.dumps(settings(arm, deny, home), indent=2) + "\n")
        prompt = ("This is a sandbox check. Run each of these three commands with the Bash tool, one Bash call "
                  "per command, in this order, even if one fails. Then list each command with its exit status "
                  "and any error text.\n\n"
                  f"1. touch {target}\n2. touch {home_target}\n3. touch {control}\n")
        launch = run_claude(arm, prompt, token, "haiku", out_dir, 300, claude_bin, extra_tools=extra)
        with open(os.path.join(out_dir, "stream.jsonl")) as f:
            lines = f.readlines()
        sb = seatbelt_blocked(lines, target)
        v = probe_verdict(os.path.exists(target), os.path.exists(home_target), os.path.exists(control),
                          sb if require_seatbelt else None)
        res = score._events(lines)
        cost = next((e.get("total_cost_usd") for e in reversed(res) if e.get("type") == "result"), None)
        rec = {**v, "target": target, "home_target": home_target, "control": control,
               "seatbelt_blocked_target": sb, "launch": launch, "cost_usd": cost}
    finally:
        for p in (target, home_target):
            if os.path.exists(p):
                os.remove(p)  # clean up an escaped probe file; the verdict already records it
        remove_arm(arm)
    rec["arm_deleted"] = not os.path.exists(arm)
    return rec


def isolation_snapshot(vault: str, claude_dir: str, start: float) -> Dict[str, List[str]]:
    vault_newer = []
    for dirpath, dirnames, files in os.walk(vault):
        if ".git" in dirnames:
            dirnames.remove(".git")
        for f in files:
            p = os.path.join(dirpath, f)
            try:
                if os.lstat(p).st_mtime > start:
                    vault_newer.append(os.path.relpath(p, vault))
            except FileNotFoundError:
                continue
    top = []
    for e in sorted(os.listdir(claude_dir)):
        try:
            if os.lstat(os.path.join(claude_dir, e)).st_mtime > start:
                top.append(e)
        except FileNotFoundError:
            continue
    return {"vault_newer": sorted(vault_newer), "claude_top_newer": top}


def isolation_record(start: float, vault: str, claude_dir: str, before, after) -> Dict[str, Any]:
    return {"batch_start": dt.datetime.fromtimestamp(start, dt.timezone.utc).isoformat(),
            "batch_start_epoch": start, "real_vault": vault, "claude_dir": claude_dir,
            "before": before, "after": after, "ok": not after["vault_newer"] and not before["vault_newer"],
            "rule": "any real-vault file (outside .git/) newer than batch start fails the batch; "
                    "~/.claude top-level entries are recorded, not failed (the host session writes there)"}


# ---------------------------------------------------------------------------
# batch
# ---------------------------------------------------------------------------


def _append(path: str, rec: Dict[str, Any], lock: threading.Lock) -> None:
    with lock:
        with open(path, "a") as f:
            f.write(json.dumps(rec) + "\n")
            f.flush()


def claude_version(claude_bin: str) -> str:
    return subprocess.run(["/usr/bin/env", "-i", "PATH=/usr/bin:/bin", claude_bin, "--version"],
                          capture_output=True, text=True, timeout=60).stdout.strip()


def real_paths():
    home = pwd.getpwuid(os.getuid()).pw_dir
    return home, os.path.join(home, ".local", "share", "engram", "vault"), os.path.join(home, ".claude")


def run_batch(name: str, specs: List[ArmSpec], model: str, timeout: int, claude_bin: str, engram_bin: str) -> int:
    batch_dir = os.path.join(RESULTS, name)
    if os.path.exists(batch_dir):
        raise HarnessError(f"{batch_dir} exists; pick a new batch name")
    os.makedirs(batch_dir)
    home, vault, claude_dir = real_paths()
    start = time.time()
    before = isolation_snapshot(vault, claude_dir, start)
    lock = threading.Lock()
    cost_log = os.path.join(RESULTS, "cost-log.jsonl")
    manifest: Dict[str, Any] = {"batch": name, "started": dt.datetime.fromtimestamp(start, dt.timezone.utc).isoformat(),
                                "arms": []}
    status = 0
    try:
        src = GitSource()
        with open(engram_bin, "rb") as fh:
            engram_sha = hashlib.sha256(fh.read()).hexdigest()
        manifest.update({
            "claude_bin": claude_bin, "claude_version": claude_version(claude_bin),
            "engram_bin": engram_bin, "engram_bin_sha256": engram_sha,
            "orchestrator_model_requested": model, "fixture_model_requested": FIXTURE_MODEL,
            "repo_head": src.head(), "pin": PIN,
            "tokens": {"file_markers": FILE_MARKERS, "arm_tokens": ARM_TOKENS},
            "allowed_tools": list(ALLOWED_TOOLS), "disallowed_tools": list(DISALLOWED_TOOLS),
        })
        token = read_token()
        deny = deny_entries(claude_tmp_root())
        manifest["deny_write"] = deny
        probe = run_probe(batch_dir, token, deny, home, claude_bin)
        _append(cost_log, {"batch": name, "arm_id": "probe", "kind": "probe", "cost_usd": probe["cost_usd"]}, lock)
        manifest["probe"] = {k: probe[k] for k in ("ok", "reason", "cost_usd")}
        if not probe["ok"]:
            manifest["aborted"] = "probe failed: " + probe["reason"]
            status = 2
        else:
            for i, spec in enumerate(specs):
                _run_one(i, spec, batch_dir, src, engram_bin, deny, home, token, model, timeout, claude_bin,
                         name, cost_log, lock, manifest)
    except BaseException as e:  # record, then re-raise: the after-check and manifest must still land
        manifest["error"] = f"{type(e).__name__}: {e}"
        status = 4
        raise
    finally:
        after = isolation_snapshot(vault, claude_dir, start)
        iso = isolation_record(start, vault, claude_dir, before, after)
        _write(os.path.join(batch_dir, "isolation.json"), json.dumps(iso, indent=2) + "\n")
        manifest["isolation_ok"] = iso["ok"]
        if not iso["ok"]:
            manifest["failed"] = "real-vault entry newer than batch start"
            status = status or 3
        manifest["finished"] = dt.datetime.now(dt.timezone.utc).isoformat()
        _write(os.path.join(batch_dir, "run-manifest.json"), json.dumps(manifest, indent=2) + "\n")
        print(f"probe={manifest.get('probe', {}).get('ok')} isolation={iso['ok']} status={status}", flush=True)
    return status


def _run_one(i, spec, batch_dir, src, engram_bin, deny, home, token, model, timeout, claude_bin, name, cost_log,
             lock, manifest) -> None:
    arm_id = f"{i:02d}-{spec.cell}-{spec.arm}-{spec.domain}"
    out_dir = os.path.join(batch_dir, arm_id)
    arm = new_arm_dir()
    try:
        info = build_arm(arm, spec, src, engram_bin, deny, home)
        manifest["arms"].append({"arm_id": arm_id, "cell": spec.cell, "arm": spec.arm, "domain": spec.domain,
                                 "texts": info["texts"]})
        _write(os.path.join(out_dir, "unit1-report.txt"), info["unit1_report"])
        launch = run_claude(arm, info["prompt"], token, model, out_dir, timeout, claude_bin,
                            extra_tools=tuple(info["fixture_tools"]))
        main_texts, other_texts = collect_sessions(arm, os.path.join(out_dir, "session"))
        notes = vault_notes(os.path.join(arm, "vault"))
        _write(os.path.join(out_dir, "vault-notes.json"), json.dumps(notes, indent=2) + "\n")
    finally:
        remove_arm(arm)
    with open(os.path.join(out_dir, "stream.jsonl")) as f:
        verdict = score.score_arm(f.readlines(), main_texts, spec.cell, spec.arm, info["unit1_report"],
                                  info["lessons"], other_texts, notes)
    rec = {"arm_id": arm_id, "arm_dir": arm, "arm_deleted": not os.path.exists(arm),
           "learn_source": spec.learn_source, "domain": spec.domain, "timed_out": launch["timed_out"],
           "wall_s": launch["wall_s"], **verdict}
    _write(os.path.join(out_dir, "score.json"), json.dumps(rec, indent=2) + "\n")
    _append(os.path.join(batch_dir, "arms.jsonl"), rec, lock)
    _append(cost_log, {"batch": name, "arm_id": arm_id, "kind": "arm", "cell": spec.cell, "arm": spec.arm,
                       "cost_usd": rec["cost_usd"], "label": rec["label"]}, lock)
    manifest["arms"][-1].update({"orchestrator_model": rec["orchestrator_model"],
                                 "fixture_models": rec["fixture_models"]})
    print(f"{arm_id}: {rec['label']} gate={rec['gate']['ok']} cost=${rec['cost_usd']}", flush=True)


def main(argv: Optional[List[str]] = None) -> int:
    ap = argparse.ArgumentParser(description="learn-at-return confined arm launcher")
    sub = ap.add_subparsers(dest="cmd", required=True)
    b = sub.add_parser("batch")
    b.add_argument("--name", required=True)
    b.add_argument("--arm", action="append", required=True, help="CELL:ARM:LEARN_SOURCE[:DOMAIN]")
    b.add_argument("--model", default="opus")
    b.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_S)
    b.add_argument("--claude-bin", default=os.path.realpath(os.path.expanduser("~/.local/bin/claude")))
    b.add_argument("--engram-bin", default=shutil.which("engram") or "")
    a = ap.parse_args(argv)
    if not a.engram_bin or not os.path.exists(a.engram_bin):
        raise HarnessError("installed engram binary not found")
    counts: Dict[str, int] = {}
    specs = []
    for text in a.arm:
        cell = text.split(":")[0]
        specs.append(parse_arm_spec(text, counts.get(cell, 0)))
        counts[cell] = counts.get(cell, 0) + 1
    return run_batch(a.name, specs, a.model, a.timeout, a.claude_bin, a.engram_bin)


if __name__ == "__main__":
    sys.exit(main())
