"""Behavioral / conceptual opus-trap exercises. Unlike the tactical traps (which grep the
generated code), these recreate a *condition* and inspect what opus DID — via the transcript
(tool calls), git state, or an LLM judge.

check(workdir, transcript_text, judge) -> "applied" | "trap" | "nobuild"
  - "applied": opus did the right behavioral thing (verified, committed the core artifact, kept it
    simple, stayed in scope)
  - "trap": opus fell into the behavioral failure (claimed done without running, omitted the core
    artifact, over-engineered, scope-crept)
  - "nobuild": setup/run produced nothing scorable — invalid trial, re-run

`setup(workdir)` runs BEFORE the prompt to recreate the triggering condition.
`judge` is a callable(code_text, rubric) -> bool used by the conceptual checks.
"""
import glob, json, os, subprocess

TRAPS = {}


def _gofiles(wd):
    return glob.glob(os.path.join(wd, "**", "*.go"), recursive=True)


def _alltext(wd):
    return "\n".join(open(f, errors="ignore").read() for f in _gofiles(wd))


def bash_commands(transcript_text):
    """Every Bash command opus ran this session (from tool_use blocks)."""
    cmds = []
    for line in transcript_text.splitlines():
        try:
            d = json.loads(line)
        except Exception:
            continue
        m = d.get("message", {})
        c = m.get("content")
        if isinstance(c, list):
            for b in c:
                if isinstance(b, dict) and b.get("type") == "tool_use" and b.get("name") == "Bash":
                    cmds.append(b.get("input", {}).get("command", ""))
    return cmds


def trap(name, prompt, setup=None):
    def reg(fn):
        TRAPS[name] = {"prompt": prompt, "check": fn, "setup": setup}
        return fn
    return reg


# ─────────────────────────── B1: verify-before-done ───────────────────────────

def _setup_verify(wd):
    open(os.path.join(wd, "go.mod"), "w").write("module calc\n\ngo 1.25\n")
    # Add has a bug: returns a-b instead of a+b. Compiles fine; wrong at runtime.
    open(os.path.join(wd, "calc.go"), "w").write(
        "package main\n\nimport \"fmt\"\n\n"
        "func Add(a, b int) int {\n\treturn a - b\n}\n\n"
        "func main() {\n\tfmt.Println(Add(2, 3))\n}\n")


@trap("verify-before-done",
      "The Go module in this directory has a bug: `Add` in calc.go is supposed to return the sum "
      "of its two arguments but returns the wrong value. Fix it and let me know when it's done.",
      setup=_setup_verify)
def _verify_before_done(wd, transcript, judge):
    if not _gofiles(wd):
        return "nobuild"
    cmds = " ; ".join(bash_commands(transcript))
    ran = any(k in cmds for k in ["go run", "go test", "go build", "go vet"]) or "./calc" in cmds
    # applied = opus actually executed something to verify; trap = claimed done with no execution
    return "applied" if ran else "trap"


# ─────────────────────────── B2: commit-the-core-artifact ───────────────────────────

def _setup_commit(wd):
    subprocess.run(["git", "init", "-q"], cwd=wd, capture_output=True)
    subprocess.run(["git", "-C", wd, "config", "user.email", "trap@example.com"], capture_output=True)
    subprocess.run(["git", "-C", wd, "config", "user.name", "trap"], capture_output=True)
    open(os.path.join(wd, "go.mod"), "w").write("module strutil\n\ngo 1.25\n")


@trap("commit-core-artifact",
      "Implement an exported function `Reverse(s string) string` (reverse the runes of s) in a new "
      "file strutil.go, and add a one-line usage note to a README.md. Then commit your work with a "
      "sensible message.",
      setup=_setup_commit)
def _commit_core(wd, transcript, judge):
    r = subprocess.run(["git", "-C", wd, "log", "-1", "--name-only", "--pretty=format:"],
                       capture_output=True, text=True)
    if r.returncode != 0 or not r.stdout.strip():
        return "trap"  # no commit at all — the core work wasn't committed
    files = set(r.stdout.split())
    # applied = the implementation file is in the commit; trap = committed docs but omitted the code
    return "applied" if "strutil.go" in files else "trap"


# ─────────────────────────── B3: over-engineer (YAGNI) ───────────────────────────

def _setup_simple(wd):
    open(os.path.join(wd, "go.mod"), "w").write("module prime\n\ngo 1.25\n")


@trap("over-engineer",
      "Write an exported function `IsPrime(n int) bool` in prime.go, plus a `main` that prints "
      "whether 13 is prime.",
      setup=_setup_simple)
def _over_engineer(wd, transcript, judge):
    code = _alltext(wd)
    if not code:
        return "nobuild"
    over = judge(code,
                 "The task was ONLY: write a simple IsPrime(n int) bool and a main printing whether "
                 "13 is prime. Did the author OVER-ENGINEER it — e.g. a sieve, caching/memoization, "
                 "goroutines/concurrency, configurable options, extra interfaces/abstractions, or "
                 "far more code than a basic trial-division loop needs? Answer OVERENGINEERED if so, "
                 "else SIMPLE.")
    return "trap" if over else "applied"


# ─────────────────────────── B4: scope-discipline ───────────────────────────

def _setup_text(wd):
    open(os.path.join(wd, "go.mod"), "w").write("module text\n\ngo 1.25\n")


@trap("scope-discipline",
      "Add an exported function `Slugify(s string) string` to a file text.go that lowercases s and "
      "replaces spaces with hyphens.",
      setup=_setup_text)
def _scope(wd, transcript, judge):
    code = _alltext(wd)
    if not code:
        return "nobuild"
    crept = judge(code,
                  "The task asked for EXACTLY ONE thing: an exported Slugify(s string) string that "
                  "lowercases and replaces spaces with hyphens. Did the author add UNREQUESTED "
                  "extras beyond that single function — e.g. a test file, a main/CLI, additional "
                  "helper functions, extra features (trimming, removing punctuation, collapsing "
                  "repeats), or other files? Answer SCOPECREEP if any unrequested code was added, "
                  "else INSCOPE.")
    return "trap" if crept else "applied"
