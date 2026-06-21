#!/usr/bin/env python3
"""Behavioral feature scorer: builds the app and RUNS it, asserting behaviors.

Inherently name-agnostic — it checks what the program *does*, not what its
identifiers are called. Each check runs a command sequence in an isolated HOME
(so the app's real data path lands in a throwaway dir) and asserts on output.

A "spec" is a list of checks: {name, steps:[(argv, assertion)]}. Assertions:
  contains:S / absent:S / json / exit0 / exit_nonzero   (applied to the step's stdout/rc)

Usage: python3 behavioral.py <workdir> <spec.json>
"""
import sys, os, re, json, subprocess, tempfile, shutil

def find_main_pkg(workdir):
    """Locate the buildable `package main` regardless of layout (flat root or cmd/).

    Returns (target, "") where `target` is a `go build` target (import path) for the
    main package, or (None, err) if none can be found / `go list` fails. Handles:
      (a) exactly one main package → that one
      (b) flat root is `package main` → the root main (same as the legacy behavior)
      (c) multiple mains → prefer one under cmd/ matching the workdir's app name,
          else the first
      (d) no main package → a clear error (so the caller fails loudly, not silent 0)
    """
    r = subprocess.run(
        ["go", "list", "-f", '{{if eq .Name "main"}}{{.ImportPath}}{{end}}', "./..."],
        cwd=workdir, capture_output=True, text=True)
    if r.returncode != 0:
        return None, "go list failed:\n" + r.stderr
    mains = [line.strip() for line in r.stdout.splitlines() if line.strip()]
    if not mains:
        return None, "no `package main` found in workdir (no runnable binary to build)"
    if len(mains) == 1:
        return mains[0], ""
    # Multiple mains: prefer one under cmd/<app> matching the workdir's app name.
    app = os.path.basename(os.path.abspath(workdir))
    cmd_match = [m for m in mains if "/cmd/" + app in m or m.endswith("/cmd/" + app)]
    if cmd_match:
        return cmd_match[0], ""
    cmd_any = [m for m in mains if "/cmd/" in m]
    if cmd_any:
        return cmd_any[0], ""
    return mains[0], ""


def build(workdir):
    target, err = find_main_pkg(workdir)
    if not target:
        return None, err
    binp = os.path.join(tempfile.mkdtemp(), "app")
    r = subprocess.run(["go", "build", "-o", binp, target], cwd=workdir,
                       capture_output=True, text=True)
    if r.returncode != 0:
        return None, r.stderr
    return binp, ""

def run_step(binp, argv, home):
    env = dict(os.environ); env["HOME"] = home; env["NO_COLOR"] = ""
    # also point common XDG / app dir envs at the throwaway home
    env["XDG_DATA_HOME"] = os.path.join(home, ".local", "share")
    r = subprocess.run([binp] + argv, env=env, capture_output=True, text=True, timeout=30)
    return r.returncode, r.stdout + r.stderr

def assert_on(rc, out, kind):
    if kind == "json":
        try: json.loads(out); return True
        except Exception: return False
    if kind.startswith("jsonlen:"):
        try: return len(json.loads(out)) == int(kind.split(":", 1)[1])
        except Exception: return False
    if kind == "any": return True
    if kind == "exit0": return rc == 0
    if kind == "exit_nonzero": return rc != 0
    if kind.startswith("contains:"): return kind[9:] in out
    if kind.startswith("absent:"): return kind[7:] not in out
    return False

def run_variant(binp, steps):
    """Run one full step-sequence in a fresh isolated home. Returns (ok, detail)."""
    home = tempfile.mkdtemp()
    tmpfile = os.path.join(home, "exchange.json")  # {TMP} placeholder for export/import
    try:
        for step in steps:
            argv = [a.replace("{TMP}", tmpfile) for a in step["argv"]]
            kind = step["assert"]
            rc, out = run_step(binp, argv, home)
            if not assert_on(rc, out, kind):
                return False, f"step {argv} failed [{kind}]: rc={rc} out={out.strip()[:120]}"
        return True, "ok"
    except Exception as e:
        return False, f"error: {e}"
    finally:
        shutil.rmtree(home, ignore_errors=True)

def check(binp, c):
    """A check passes if its 'steps' pass, OR (if 'variants' given) if ANY variant
    passes. Variants exist to make a check immune to incidental quirks (e.g. flag
    ordering) so it measures the intended capability, not the quirk."""
    variants = c.get("variants")
    if variants:
        last = "no variant passed"
        for v in variants:
            ok, detail = run_variant(binp, v)
            if ok:
                return True, "ok (variant)"
            last = detail
        return False, last
    return run_variant(binp, c["steps"])

def score(workdir, spec):
    binp, err = build(workdir)
    if not binp:
        return {"build": "FAIL", "error": err[:400], "checks": {}}
    res = {}
    for c in spec["checks"]:
        ok, ev = check(binp, c)
        res[c["name"]] = {"pass": ok, "evidence": ev}
    passed = sum(1 for r in res.values() if r["pass"])
    return {"build": "ok", "feat": f"{passed}/{len(res)}", "passed": passed, "n": len(res), "checks": res}

if __name__ == "__main__":
    spec = json.load(open(sys.argv[2]))
    print(json.dumps(score(sys.argv[1], spec), indent=2))
