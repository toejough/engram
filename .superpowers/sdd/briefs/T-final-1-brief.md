# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T-final-1: <SET AT DISPATCH — after T-polish ACK>. Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- ACCUMULATED DISPATCH NOTES (binding):
  - The migration inventory is COMPLETE and ledgered: at dispatch time internal/ has ZERO non-test os/syscall/os-exec/hugot imports (T13 left update.go, T16/T17 killed it — verify with the sweeps below before you start; a hit = tree doesn't match ledger = STOP).
  - Step-4 discipline is absolute: a depguard/forbidigo finding means the migration missed a SITE — fix the site or escalate; NEVER a carve-out/nolint/allow-entry. A check-thin-api finding = a prior task regressed cmd — escalate the exact finding.
  - Step-5 negative probe is mandatory BEFORE commit (a green gate that can't fail is no gate); revert the probe cleanly and verify `git status` clean-modulo-your-toml afterward.
  - R9: try `files = ['internal/**', '!$test']` FIRST; only if the step-5 probe stays silent, fall back to `'**/internal/**'` AND report that the issue-AC wording needs amending (orchestrator owns the issue edit).
  - Step 5.5's coverage-stance decision must land in the commit body — if it resolves to call (b), STOP and surface the choice, don't decide unilaterally.
  - Empirically-confirmed linter semantics (vault note 301, plan-validated): depguard allow-only = default-deny; allow/deny are PREFIX-based; forbidigo custom `forbid` REPLACES print defaults; `analyze-types` needed for pkg-qualified patterns; `issues.fix=true` auto-modifies (do not enable); max-issues truncation hid 14/24 findings at prototype time — step 3's =0 is load-bearing.
  - gates run FOREGROUND; stage EXPLICIT paths only (`git add dev/golangci-lint.toml` exactly, per step 6).
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone) + the 2 protected dev/eval reorder fixtures; lint-full must be 0.
- REPORT: `.superpowers/sdd/briefs/T-final-1-report.md` BEFORE your final message — status, commit SHA, verbatim gate outcomes INCLUDING the step-5 probe finding text and the step-5.5 stance evidence, deviations, concerns. Final message: STATUS line, SHA, summary, concerns.

---

### Task T-final-1: Enforcement flip — depguard + forbidigo land with zero carve-outs

**Files:**
- Modify: `dev/golangci-lint.toml`

**Interfaces:**
- Consumes: the fully-migrated tree (all prior tasks complete; no os/exec/signal/syscall/hugot imports and no time.Now/Since/Tick references remain in internal/ non-test code).
- Produces: the enforced purity boundary; `targ check-full` fails on any internal-side regression, and `targ check-thin-api` (authoritative, unchanged by this task) keeps guarding the cmd side.

- [ ] **Step 1: RED — add the depguard rule and confirm it currently passes only because migration is done.** Add to `dev/golangci-lint.toml` alongside the existing `[linters.settings.depguard.rules.all]`:

```toml
# #700: internal/ purity — default-deny. Anything not prefix-matched below is denied
# in internal non-test code. NO file carve-outs: raw I/O primitives enter from
# cmd/engram's declaration-free main (cli.Primitives; targ check-thin-api enforces that
# side), ALL adapter composition is injected in internal/cli, and the only real-os code
# under internal/ sits in _test files — excluded via '!$test' (sanctioned by the revised
# composition doctrine).
# Glob form per R9: start with the issue-AC literal 'internal/**'; Step 5's negative
# probe validates it fires. Fall back to the prototype-confirmed '**/internal/**'
# ONLY if the probe stays silent, and amend the issue AC wording (see R9).
[linters.settings.depguard.rules.internal-purity]
files = ['internal/**', '!$test']
allow = [
	'strings', 'fmt', 'errors', 'sort', 'slices', 'maps', 'strconv', 'unicode',
	'bufio', 'bytes', 'io', 'path', 'regexp',
	'encoding/json', 'encoding/hex', 'crypto/sha256', 'hash/fnv', 'math', 'time',
	'context', 'sync', 'embed',
	'github.com/toejough/engram',
	'go.yaml.in/yaml/v3',
	'github.com/toejough/targ',
]
```

Notes: prefix matching means `io` admits `io/fs`, `path` admits `path/filepath`, `sync` admits `sync/atomic`, `math` admits `math/rand/v2` (kmeans' seeded PCG — misuse is forbidigo's job below), `time` admits types/parsing (clock calls are forbidigo's job). No `deny` entries: a `math/rand` deny would prefix-catch the legal seeded v2 import.

- [ ] **Step 2: enable forbidigo with the custom list.** Remove `'forbidigo'` from the `disable` list and add:

```toml
[linters.settings.forbidigo]
# Custom list REPLACES the fmt.Print defaults — printing stays legal (repo prints on purpose).
analyze-types = true

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Now$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Since$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Tick$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^targ\.Main$'
msg = 'targ dispatch is edge work — call from cmd/engram only (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '.*'
pkg = '^math/rand$'
msg = 'math/rand v1 is banned — use a seeded math/rand/v2 *rand.Rand (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^rand\.(N|Int32|Int64|Int32N|Int64N|IntN|Int|Uint32|Uint64|Uint32N|Uint64N|UintN|Uint|Float32|Float64|NormFloat64|ExpFloat64|Perm|Shuffle)$'
pkg = '^math/rand/v2$'
msg = 'auto-seeded global PRNG is banned — use a seeded *rand.Rand (#700)'
```

And scope it with exclusion rules (forbidigo applies only to internal non-test code):

```toml
[[linters.exclusions.rules]]
linters = ['forbidigo']
path-except = '^internal/'

[[linters.exclusions.rules]]
linters = ['forbidigo']
path = '_test\.go$'
```

- [ ] **Step 3: set `max-issues-per-linter = 0`** (deliberate: never truncate findings; the default 10 hid 14 of 24 findings during the plan-time prototype). Find the existing `max-issues-per-linter` line and set it to `0`.

- [ ] **Step 4: verify — `targ check-full` AND `targ check-thin-api`.** Expected: check-full GREEN, check-thin-api PASS (`All N public API files are thin wrappers.`). If depguard/forbidigo report findings, the migration missed a site — fix the SITE (relocate/thread it per the relevant task's pattern); NEVER add a carve-out, nolint, or allow-list entry to make it pass (that violates the issue's zero-grandfathering acceptance criteria; escalate to the orchestrator if a finding looks structural). If check-thin-api reports a finding, a prior task regressed the declaration-free cmd shape — escalate the exact finding (doctrine flag SIG-1); never suppress.

- [ ] **Step 5: negative self-test of the gate (temporary, not committed).** Add `_ = os.Getenv("PROBE")` (+ `"os"` import) to any internal/ non-test file; run `targ check-full`; expect a depguard finding naming `internal-purity`. Revert the probe. This proves the rule fires (a green gate that can't fail is no gate).

- [ ] **Step 5.2: loser-symbol grep gate.** `rg -n "edgeVaultFS|depsVaultFS|jsonlIndexesLister|jsonlIndexListerFrom|vaultLuhmannLock|warnLoggerTo|osListJSONLIndexes|ExportNewOsVaultFS" internal/ cmd/` must return ZERO hits: the parallel-drafting loser symbols (R1/R2/R3 — never legally declared anywhere) and the transitional shims (`osListJSONLIndexes`, deleted by T12; `ExportNewOsVaultFS`, call sites migrated by T12 per R12 and shim deleted by T7) must not exist in the final tree. Any hit → a task landed a loser or skipped its gated deletion; fix the SITE per the owning resolution (R1/R2/R3/R12) before proceeding — never rename-in-place to dodge the grep.

- [ ] **Step 5.5: coverage stance for cmd/engram (issue AC).** Post-rework, `cmd/engram` holds ONLY the declaration-free `main.go` — a single-statement `main()` over the `cli.Primitives` literal, `targ check-thin-api`-enforced, with no testable logic. The adapter logic and the tests that used to sit beside it in cmd now live in `internal/cli`: unit tests with fake primitives plus integration `_test` files with real os/syscall funcs, all counting toward internal coverage like any other internal tests; the production literal itself is guarded by cli_test.go's end-to-end binary build. Inspect how the coverage gate treats `cmd/engram`: read the check-full output's coverage section (and `rg -n "cmd/" dev/targs.go dev/*.toml` for exclusion patterns). Record the finding + decision as a one-paragraph note in the commit body of Step 6: the expected call is (a) `cmd/engram` stays coverage-excluded as an entry point (the repo's entry-point exclusion doctrine — wiring only, no testable logic). If the tooling instead shows main.go entering coverage uncovered, that is call (b): decide deliberately between an explicit entry-point exclusion and a trivial main-wiring smoke test (the most cmd may keep), and surface the choice to the orchestrator. Do not silently leave the stance undecided — the issue AC requires a deliberate call.

- [ ] **Step 6: Commit.**

```bash
git add dev/golangci-lint.toml
git commit -m "check(#700): enforce internal purity via depguard + forbidigo

Zero carve-outs: raw I/O primitives enter from cmd/engram's
declaration-free main (check-thin-api-enforced), adapter composition is
injected in internal/cli, and real-os code under internal/ is confined
to _test files ('!$test'), so the rule needs no file exceptions. Custom
forbidigo list replaces print defaults (printing stays legal).
max-issues-per-linter=0 so findings never truncate.

AI-Used: [claude]"
```

