"""Candidate opus-trap exercises. Each: a no-hint prompt that invites the natural-but-
locally-wrong idiom, plus a deterministic `check(workdir) -> ("applied"|"trap"|"nobuild")`.

`check` returns:
  "applied"  — the lesson/convention was applied (warm-correct form present)
  "trap"     — the natural-but-wrong form present (cold fell in)  <-- what we want to reproduce
  "nobuild"  — no go file / unparseable (treat as invalid trial, re-run)

A trap is "confirmed reproducible" if cold opus returns "trap" in >= THRESHOLD of N trials.
Prompts deliberately carry NO mention of the convention so cold opus defaults naturally.
"""
import glob, os, re


def _gofiles(workdir):
    return [f for f in glob.glob(os.path.join(workdir, "**", "*.go"), recursive=True)]


def _alltext(workdir):
    return "\n".join(open(f, errors="ignore").read() for f in _gofiles(workdir))


TRAPS = {}


def trap(name, prompt):
    def reg(fn):
        TRAPS[name] = {"prompt": prompt, "check": fn}
        return fn
    return reg


@trap("slices-backward",
      "Create a Go module `rev` (run `go mod init rev` first). In a file, write an exported "
      "function `Reverse(lines []string) []string` that returns the elements in reverse order, "
      "and a `main` that prints the reversed lines. Build it so `go build` succeeds.")
def _slices_backward(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    cstyle = re.search(r"for\s+\w+\s*:=\s*len\([^)]*\)\s*-\s*1\s*;.*>=\s*0\s*;.*--", t)
    if "slices.Backward(" in t:
        return "applied"
    if cstyle:
        return "trap"
    # neutral third path (forward range + reverse-index, append-prepend) — convention never bites;
    # NOT a trap. Count as applied (the bad C-style form was avoided).
    return "applied"


@trap("crypto-rand",
      "Create a Go module `code` (run `go mod init code` first). Write an exported function "
      "`NewCode() string` that returns a random 8-character lowercase alphanumeric code, and a "
      "`main` that prints one. Make `go build` succeed.")
def _crypto_rand(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if re.search(r'"math/rand"', t):
        return "trap"
    if re.search(r'"crypto/rand"', t):
        return "applied"
    return "nobuild"


@trap("req-with-context",
      "Create a Go module `fetch` (run `go mod init fetch` first). Write an exported function "
      "`Body(url string) (string, error)` that does an HTTP GET and returns the response body as "
      "a string, and a `main` that fetches a URL passed as os.Args[1]. Make `go build` succeed.")
def _req_context(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "NewRequestWithContext" in t:
        return "applied"
    if re.search(r"http\.Get\(|http\.NewRequest\(", t):
        return "trap"
    return "nobuild"


@trap("nocolor",
      "Create a Go module `greet` (run `go mod init greet` first). Write a `main` that prints "
      "the word HELLO in green using ANSI color codes. Make `go build` succeed.")
def _nocolor(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "NO_COLOR" in t:
        return "applied"
    if re.search(r"\\033\[|\\x1b\[|\\u001[bB]\[", t) or "[32m" in t:
        return "trap"
    return "nobuild"


@trap("t-parallel",
      "Create a Go module `calc` (run `go mod init calc` first). Write `Add(a, b int) int` and a "
      "table-driven test `TestAdd` in calc_test.go covering a few cases. Make `go test` pass.")
def _t_parallel(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "_test.go" not in " ".join(_gofiles(workdir)):
        return "nobuild"
    # applied = t.Parallel() called in the test (both the top test and the subtest ideally)
    if "t.Parallel()" in t or "tt.Parallel()" in t:
        return "applied"
    return "trap"


@trap("nil-guard-split",
      "Create a Go module `firstline` (run `go mod init firstline` first). Write an exported "
      "function `First(body []byte) []byte` that splits `body` on newline bytes and returns the "
      "first line. Add a `main`. Make `go build` succeed.")
def _nil_guard(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "bytes.Split" not in t and "bytes.SplitN" not in t:
        # used Index/Cut instead — not the targeted idiom; treat as applied (avoided the trap)
        return "applied" if ("bytes.Index" in t or "bytes.Cut" in t) else "nobuild"
    # applied = explicit nil/len guard before indexing the split result
    if re.search(r"if\s+len\([^)]*\)\s*==\s*0|if\s+\w+\s*==\s*nil|len\([^)]*\)\s*>\s*0", t):
        return "applied"
    return "trap"


@trap("named-const",
      "Create a Go module `retry` (run `go mod init retry` first). Write an exported function "
      "`Do(f func() error) error` that retries `f` up to 3 times with a short sleep between "
      "attempts, returning the last error. Add a `main`. Make `go build` succeed.")
def _named_const(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    # applied = a named const for the retry count; trap = bare 3 in the loop
    if re.search(r"const\s+\w+\s*=\s*3|const\s*\(\s*\n[^)]*=\s*3", t):
        return "applied"
    if re.search(r"(<\s*3|<=\s*3|i\s*:=\s*0;.*<\s*3|range\s+\[\]|for\s+\w+\s*:=\s*0;.*<\s*3)", t):
        return "trap"
    return "trap"


@trap("sentinel-err",
      "Create a Go module `store` (run `go mod init store` first). Write a `Get(key string) "
      "(string, error)` method on a map-backed `Store` that returns an error when the key is "
      "missing, plus a `main`. Make `go build` succeed.")
def _sentinel(workdir):
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if re.search(r"var\s+Err\w+\s*=\s*errors\.New", t):
        return "applied"
    if re.search(r"errors\.New\(|fmt\.Errorf\(", t):
        return "trap"
    return "nobuild"


@trap("wrapped-error",
      "Create a Go module `cfgload` (run `go mod init cfgload` first). Write an exported function "
      "`Load(path string) ([]byte, error)` that reads a file and returns its bytes, returning any "
      "error to the caller. Add a `main`. Make `go build` succeed.")
def _wrapped_error(workdir):
    # local convention: wrap errors with context using fmt.Errorf("...: %w", err), not bare `return ..., err`
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "os.ReadFile" not in t and "ioutil.ReadFile" not in t and "os.Open" not in t:
        return "nobuild"
    # applied = the returned read error is wrapped with %w; trap = bare `return nil, err`
    if re.search(r'fmt\.Errorf\([^)]*%w', t):
        return "applied"
    if re.search(r"return\s+nil\s*,\s*err\b", t):
        return "trap"
    return "trap"


@trap("make-cap",
      "Create a Go module `dbl` (run `go mod init dbl` first). Write an exported function "
      "`Doubled(in []int) []int` that returns a new slice where each element is doubled, and a "
      "`main`. Make `go build` succeed.")
def _make_cap(workdir):
    # local convention: make([]T, 0, len(in)) when the size is known, not append to a nil slice
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if re.search(r"make\(\[\]int,\s*0,\s*len\(", t) or re.search(r"make\(\[\]int,\s*len\(", t):
        return "applied"
    if re.search(r"var\s+\w+\s*\[\]int|:=\s*\[\]int\{\}|make\(\[\]int,\s*0\)", t):
        return "trap"
    return "trap"


@trap("table-test-loop",
      "Create a Go module `mathx` (run `go mod init mathx` first). Write `Max(a, b int) int` and "
      "an exhaustive test `TestMax` in mathx_test.go. Make `go test` pass.")
def _table_test(workdir):
    # local convention: subtests via t.Run(name, ...) inside the table loop, not flat asserts
    t = _alltext(workdir)
    if not t:
        return "nobuild"
    if "_test.go" not in " ".join(_gofiles(workdir)):
        return "nobuild"
    if "t.Run(" in t:
        return "applied"
    return "trap"
