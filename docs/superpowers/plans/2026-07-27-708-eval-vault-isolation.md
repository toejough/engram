# Design: #708 — eval-harness vault isolation

Status: approved 2026-07-27 (Joe). Scope ratified as tier A + tier B + enforcement.

## Problem

Every harness under `dev/eval` spawns `claude -p --permission-mode bypassPermissions` with `engram`
reachable on the inherited `PATH`. Twenty spawn sites do not point engram's state at anything
disposable, so a trial's `engram learn` writes to the operator's real vault and its `engram query`
reads the operator's real chunk index.

This is measured, not theoretical. During #687's measurement a `please_step7_probe` trial wrote seven
real notes into the live vault (IDs 533–537, 539, 540). They looked like genuine memory — two
duplicated existing legitimate notes, two were near-duplicates of each other from separate trials —
and would have surfaced at future `engram recall` calls as experience that never happened. The `.md`
files were deleted by hand afterward.

The repo has already fixed this failure class once. #642 (commit `d943ea93`, 2026-07-17) found the
cumulative harness's warm arm reading the operator's global chunk index — the eval answer key — and
scoring round-1 8/8 from an empty vault. The fix wired `ENGRAM_CHUNKS_DIR` through
`harness.claude()` and added `test_chunks_isolation.py` to guard it. That guard covers exactly one
call site. The other nineteen were never in its reach.

### Two claims in the issue body that no longer hold

- The two orphaned `.vec.json` sidecars #708 reports as "still sitting in the vault right now" are
  gone; they were cleaned between filing and this design. The mechanism is untouched, but that
  evidence line is stale.
- Note 486's finding that `engram update` ignores an externally-set `ENGRAM_CHUNKS_DIR` no longer
  describes the code: `internal/cli/update.go:295` calls
  `ResolveChunksDir("", report.Home, deps.Env.Getenv)`, which reads the env var through the standard
  resolver (`internal/cli/ingest.go:71`). The design does not depend on that limitation, and
  implementation confirms it empirically rather than trusting either the note or this paragraph.

## Systems and artifacts (current state)

### Tier A — no engram isolation at all

Each builds `env = dict(os.environ)`, sets only `CLAUDE_CONFIG_DIR` (and sometimes
`CLAUDE_CODE_MAX_OUTPUT_TOKENS`), and spawns with `bypassPermissions`.

| File | Spawn site |
|---|---|
| `dev/eval/cumulative/please_step7_probe/run_probe.py` | 341–353 — the #708 incident |
| `dev/eval/cumulative/please_step3_probe/run_probe.py` | 328–340 |
| `dev/eval/cumulative/endorse_cue/probe.py` | 150–154 |
| `dev/eval/traps/brun.py` | 38–42 and 61–63 |
| `dev/eval/traps/run.py` | 41–44 |
| `dev/eval/traps/qanchor_eval.py` | 85–88 |
| `dev/eval/traps/reasoning_eval.py` | 93–96 |

All seven build their trial cwd with `tempfile.mkdtemp` under `/tmp/...` or the system temp dir.

### Tier B — vault isolated, chunk index not

The #642 gap, still open outside `harness.claude()`. Reads resolve to the operator's global index.

- `dev/eval/adapters.go` `agentEnv` (line 167) — the Go harness. `AgentInvocation` (`deps.go:8`)
  carries `VaultPath` and has no chunks field at all.
- `dev/eval/run-chain-stage.sh:51`, `run-layer-arm.sh:47`, `run-layer-resume.sh:33`
- `dev/eval/traps/retrieval_probe.py:97`, `seed_c3.py:47`, `crowd.py:194`, `synth_fixtures.py:73`,
  `compound_fixtures.py:169`, `cake_fixtures.py:29` and `:49`, `qanchor_retrieval_probe.py:36`
- `dev/eval/cumulative/harness.py:319` and `:828`

### Tier C — enforcement

None exists. There is no shared choke point, no preflight assertion, and no post-run check. The
Python tests that do guard isolation are not executed by any gate: `targ check-full`'s eight legs are
Go-only, so `test_chunks_isolation.py` and the other twenty `test_*.py` files run only when a human
types `pytest`.

### Reference points already in the repo

- `dev/eval/cumulative/harness.py:78` — `claude()`, the proven isolation helper. Sets
  `CLAUDE_CONFIG_DIR`, `ENGRAM_TRANSCRIPT_DIR`, `ENGRAM_VAULT_PATH`, and `ENGRAM_CHUNKS_DIR`.
- `dev/eval/cumulative/underload_repro/build_fixture_vaults.py:247–250` — an existing fail-loud
  guard that pops `ENGRAM_VAULT_PATH` so ambient env leakage cannot reach a real vault.
- `dev/eval/qa/p1_retrieval_pollution.sh:110` — prose already stating the rule this design mechanizes.

## Solution

One dependency-free module, `dev/eval/isolation.py`, plus its adoption at every site and a matching
field on the Go harness. Stdlib only, and deliberately no import of `harness.py`, so
`dev/eval/traps/run.py` can use it without pulling in the cumulative harness's model tables.

```python
isolated_env(cfg, trial_dir, cwd=None, base=None) -> dict
```
Returns a subprocess environment with `CLAUDE_CONFIG_DIR=cfg`,
`ENGRAM_VAULT_PATH=<trial_dir>/vault`, `ENGRAM_CHUNKS_DIR=<trial_dir>/chunks`, and — when `cwd` is
given — `ENGRAM_TRANSCRIPT_DIR=<cfg>/projects/<slug(cwd)>`, matching `harness.claude()`'s existing
shape. Creates the directories, calls `assert_isolated`, and returns the env. `base` defaults to
`os.environ`.

```python
assert_isolated(env, cwd=None) -> None
```
Raises `IsolationError` when any of the three engram variables is unset or empty; when any of them
`realpath`s to a location inside the operator's engram data directory; or when `cwd` has a `.claude`,
`.git`, `.hg`, or `.jj` ancestor. The data directory resolves the way the binary resolves it —
`$XDG_DATA_HOME/engram`, else `~/.local/share/engram` — so the comparison targets the real
destination rather than a hardcoded path.

The cwd-ancestor check exists because env isolation alone is insufficient: `ingest --auto` walks the
cwd's ancestors collecting `.claude` directories, which is correct in production and, in an eval,
swept roughly 48 operator-global files into a supposedly-isolated per-trial index, displacing the
planted chunk and confounding the measurement (vault note 296).

```python
vault_fingerprint() -> tuple[int, str]
assert_vault_unchanged(before) -> None
```
The fingerprint is the real vault's `*.md` count plus a sha256 over the sorted basename list.
`assert_vault_unchanged` raises when it has moved. Each harness `main()` snapshots before its run and
asserts at exit.

### Adoption

- **Tier A** — each site replaces `env = dict(os.environ)` with `isolated_env(...)`, reached via a
  two-line `sys.path` shim (the idiom `smoke_prune.py` already uses for `import harness`).
- **Tier B** — the same helper supplies the missing `ENGRAM_CHUNKS_DIR`. The shell harnesses gain the
  variable directly, since they cannot import Python.
- **`harness.py:78`** — `claude()` keeps its signature and delegates its env construction to
  `isolated_env`, so one implementation exists rather than two.
- **Go** — `AgentInvocation` gains `ChunksDir`; `agentEnv` emits `ENGRAM_CHUNKS_DIR` when it is
  non-empty; `runEval` (`dev/targs.go:44`) populates it with a per-run directory under `cfg.OutDir`
  instead of letting resolution fall through to the global index.

### Failure behavior

Every check raises; nothing warns and nothing falls back. This follows the repo's standing rule that
a missing eval input must fail loud rather than silently defaulting to a strawman condition. The
`IsolationError` message names the offending variable and the path it resolved to — "isolation
failed" without the resolved path costs a diagnosis round-trip.

## After

Every harness trial reads and writes engram state inside a per-trial directory. A run can be killed,
rerun, or abandoned mid-flight without touching `~/.local/share/engram`. A site that skips the module
fails at spawn time with a named variable and a resolved path. A path neither mechanism anticipated —
a subcommand resolving its own way, the class note 486 describes — still fails at exit, when the
fingerprint moves.

Unchanged: what each harness measures and how it records it. The JSONL result schemas, the
`dev/eval/LEDGER.md` rows citing them, the trap fixtures and their pre-registered pass bars, the
`bypassPermissions` mode each trial runs under, and the per-trial `CLAUDE_CONFIG_DIR` and
`mkdtemp` cwd isolation already in place all keep their current behavior. Also unchanged: the
shipped binary, which none of this touches.

## How it solves it

The three mechanisms fail differently, which is why all three are here rather than whichever one
looks sufficient.

The module makes isolation the default: a site that calls it cannot forget a variable, because the
helper owns all four. That addresses the cause of tier A — twenty hand-rolled environments exist
because building one by hand is a one-liner and nothing objects.

The preflight catches the site that skips the module. Without it, the module is a convenience that a
future harness author can route around exactly as the seven tier-A sites routed around
`harness.claude()`.

The fingerprint catches what neither anticipates. It makes no assumption about which subcommands
honor which variable, so a resolution path we did not enumerate surfaces as a failed assertion at
exit rather than as unexplained real notes days later.

## Testing

Example-based `pytest` beside the module, in the shape `test_chunks_isolation.py` established:
env-wiring assertions, one test per rejection reason, and a fingerprint round-trip. No property tests
and no new `check-full` leg — property rigor is scoped to production code (vault note 556), and the
Python harness earns real gate coverage under #710, where a Python-only leg added now would be
deleted.

Manual end-to-end verification is the actual bar. Fingerprint the real vault, run one live
`please_step7_probe` trial against the installed binary, fingerprint again, and confirm both that the
real vault is unchanged and that the trial's scratch vault is where its notes landed. Passing tests
do not establish that the feature works.

## Related

#708 is this work. #710 ports `dev/eval` to Go; #711 then holds it to production rigor — this
module ports forward as a struct constructor rather than being thrown away. #642 (`d943ea93`) is the
prior instance of the class. Vault notes 160 (eval arms escaping a cwd-only sandbox), 287 (isolate
the chunk index, not just the vault), 296 (the cwd-ancestor sweep), and 556 (property rigor is for
production code) are the recalled lessons this design is built on.
