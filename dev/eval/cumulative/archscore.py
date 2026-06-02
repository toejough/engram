#!/usr/bin/env python3
"""Deterministic, name-agnostic architecture scorer for a Go CLI app.

Detects PATTERNS, never vocabulary tokens, so a build that chooses `Repository`
over `Store` (the bias that burned the prior scorer twice) is judged on structure.
Operates on the .go sources in a workdir. Returns per-detector pass/fail + evidence.

Usage: python3 archscore.py <workdir>
"""
import sys, os, re, glob, json

def load(workdir):
    src, tst, mod = [], [], ""
    for p in glob.glob(os.path.join(workdir, "**", "*.go"), recursive=True):
        txt = open(p).read()
        (tst if p.endswith("_test.go") else src).append(txt)
    modp = glob.glob(os.path.join(workdir, "**", "go.mod"), recursive=True)
    if modp:
        mod = open(modp[0]).read()
    return "\n".join(src), "\n".join(tst), mod

PERSIST_VERBS = r"(?:Save|Load|Get|Put|List|Add|Delete|All|Read|Write|Fetch|Store|Insert|Remove|Find|Persist|Append)"

def d_di(src, tst, mod):
    # an interface whose method set includes a persistence verb, used as a field/param (injected)
    for m in re.finditer(r"type\s+(\w+)\s+interface\s*\{([^}]*)\}", src, re.S):
        name, body = m.group(1), m.group(2)
        if re.search(PERSIST_VERBS + r"\s*\(", body):
            # injected: referenced as a struct field type or function param type elsewhere
            uses = len(re.findall(r"[\s,(]" + re.escape(name) + r"\b", src))
            if uses >= 1:
                return True, f"interface {name} with persistence methods, injected"
    return False, "no injected persistence interface (concrete storage / no DI)"

def d_sentinel(src, tst, mod):
    has = re.search(r"var\s+Err\w+\s*=\s*errors\.New", src)
    wrap = "%w" in src
    is_ = ("errors.Is" in src) or ("errors.Is" in tst)
    if has and wrap and is_:
        return True, "sentinel var Err*, %w wrap, errors.Is"
    return False, f"sentinel={bool(has)} wrap={wrap} errors.Is={is_}"

def d_atomic(src, tst, mod):
    if re.search(r"CreateTemp", src) and re.search(r"\.Rename\(|os\.Rename", src):
        return True, "CreateTemp + Rename (atomic)"
    return False, "no temp+rename (in-place write)"

def d_stdlib(src, tst, mod):
    # a require line with a domain-path module = external dep
    for line in mod.splitlines():
        if re.search(r"^\s*[\w.-]+\.[\w.-]+/[\w./-]+\s+v", line):
            return False, "external module in go.mod"
    return True, "stdlib only"

def d_tests_fake_parallel(src, tst, mod):
    # convention: parallel tests + a non-file fake implementation of the store.
    # name-agnostic: count distinct receiver types implementing a persistence verb.
    # >=2 impls (real file adapter + an in-memory fake) is the structural signature.
    par = "t.Parallel()" in tst
    storage_verbs = r"(?:Save|Load|Get|Put|Read|Write|All|Fetch|Persist)"
    impls = set(re.findall(r"func\s*\(\s*\w+\s+\*?(\w+)\s*\)\s*" + storage_verbs + r"\s*\(", src + tst))
    has_fake = len(impls) >= 2
    if par and has_fake:
        return True, f"parallel tests + {len(impls)} store impls (file+fake)"
    return False, f"parallel={par} store_impls={sorted(impls)}"

def d_json(src, tst, mod):
    if re.search(r"json\.NewEncoder|json\.Marshal", src) and re.search(r'"json"|--json|jsonOut|JSON', src):
        return True, "json encode + json flag"
    return False, "no machine-readable json output mode"

def d_nocolor(src, tst, mod):
    if "NO_COLOR" in src:
        return True, "respects NO_COLOR"
    return False, "no NO_COLOR handling"

def d_wrapped(src, tst, mod):
    n = src.count("%w")
    if n >= 2:
        return True, f"{n} wrapped errors"
    return False, f"only {n} wrapped errors"

def d_named_perms(src, tst, mod):
    # bare file-mode octal literal used directly = magic number
    bare = re.findall(r"(?:WriteFile|MkdirAll|OpenFile|Mkdir|Chmod)\([^)]*?,\s*0o?[0-7]{3,4}\s*[,)]", src)
    namedconst = bool(re.search(r"(?:Perm|Mode|dirPerms|filePerms)\w*\s+(?:os\.FileMode\s*)?=\s*0o?[0-7]{3}", src))
    if not bare or namedconst:
        return True, "perms named or none"
    return False, f"{len(bare)} bare file-mode literal(s)"

def d_no_global_data(src, tst, mod):
    if re.search(r"^var\s+\w+\s+(\[\]\w+|map\[)", src, re.M):
        return False, "package-level mutable data var"
    return True, "no global mutable data"

DETECTORS = [
    ("di", d_di), ("sentinel", d_sentinel), ("atomic", d_atomic), ("stdlib", d_stdlib),
    ("tests_fake_parallel", d_tests_fake_parallel), ("json", d_json), ("nocolor", d_nocolor),
    ("wrapped_errors", d_wrapped), ("named_perms", d_named_perms), ("no_global_data", d_no_global_data),
]

def score(workdir):
    src, tst, mod = load(workdir)
    res = {}
    for name, fn in DETECTORS:
        ok, ev = fn(src, tst, mod)
        res[name] = {"pass": ok, "evidence": ev}
    passed = sum(1 for r in res.values() if r["pass"])
    return {"workdir": workdir, "arch": f"{passed}/{len(DETECTORS)}", "passed": passed, "n": len(DETECTORS), "detectors": res}

if __name__ == "__main__":
    out = score(sys.argv[1])
    print(json.dumps(out, indent=2))
