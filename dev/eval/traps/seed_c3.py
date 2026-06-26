"""C3 warm-vault fixture — the 5 Go convention notes opus must apply when warm.

Each note states one project convention whose `object` names the exact code form the build must
emit: http.NewRequestWithContext, a NO_COLOR gate, t.Parallel(), a len( guard before indexing a
split, and fmt.Errorf("...: %w", err). seed(vault) writes them via `engram learn fact` (the verified
CLI contract used by c4_idio.py) and RAISES on any non-zero exit — fail loud, never silent-pass.
"""
import os
import subprocess

SOURCE = "engram Go codebase convention"

C3_NOTES = [
    {"slug": "req-with-context",
     "situation": "making an outbound HTTP request in Go",
     "subject": "every outbound HTTP request in Go code",
     "predicate": "must be constructed so it carries a context —",
     "object": "use http.NewRequestWithContext, never http.Get, so the request can be cancelled"},
    {"slug": "nocolor",
     "situation": "printing colored output from a Go CLI",
     "subject": "ANSI color output from a Go CLI",
     "predicate": "must be made conditional —",
     "object": "gate color output behind a NO_COLOR env check before emitting any escape codes"},
    {"slug": "t-parallel",
     "situation": "writing a Go test or subtest",
     "subject": "every Go test function and subtest",
     "predicate": "must opt into parallelism —",
     "object": "call t.Parallel() in every test and subtest (with no shared mutable state)"},
    {"slug": "nil-guard-split",
     "situation": "indexing the result of strings.Split in Go",
     "subject": "the result of strings.Split before it is indexed",
     "predicate": "must be length-checked —",
     "object": "guard with a nil/len( check before indexing a split result, never assume parts[1]"},
    {"slug": "wrapped-error",
     "situation": "returning an error from a Go function",
     "subject": "every error returned from a Go function",
     "predicate": "must be wrapped with context —",
     "object": 'wrap errors with fmt.Errorf("...: %w", err) to preserve the error chain'},
]


def seed(vault_path):
    """Run `engram learn fact` for each C3 note into vault_path. Raise RuntimeError on any
    non-zero exit so a broken seed fails loud instead of silently leaving an empty vault."""
    os.makedirs(vault_path, exist_ok=True)
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault_path
    for note in C3_NOTES:
        result = subprocess.run(
            ["engram", "learn", "fact",
             "--slug", note["slug"], "--position", "top",
             "--source", SOURCE, "--situation", note["situation"],
             "--subject", note["subject"], "--predicate", note["predicate"],
             "--object", note["object"]],
            env=env, capture_output=True, text=True)
        if result.returncode != 0:
            raise RuntimeError(
                f"engram learn fact failed for {note['slug']!r} (exit {result.returncode}): "
                f"{result.stderr.strip()}")
