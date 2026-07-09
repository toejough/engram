# `engram recall` — subprocess-isolated recall (payload-prune production form)

**Status: ⛔ REJECTED 2026-07-08 (Joe) — do not build.** The design got too complicated, and isolation at
all nesting depths forces launching the agent from engram (`engram recall` → `claude -p`) rather than as a
plain Agent-tool subagent. Revisit only if a viable subagent route emerges (isolation via normal subagent
dispatch, or an accepted depth-limited form). The isolation *premise* remains smoke-validated
(`dev/eval/LEDGER.md#payload-prune-smoke`) and is not relitigated. See `docs/ROADMAP.md` — Track B.

*(Original status: this spec captures the design reached with Joe on 2026-06-30/07-01 so the orientation
isn't lost. It spawns parallel sub-recalls that write the vault/manifest concurrently; Track 0 shipped
2026-07-01 — the flock/atomic-write safety that concurrency needs.)*

## Why (premise, validated)

The payload-prune lever is smoke-validated (note 149;
`dev/eval/LEDGER.md#payload-prune-smoke`): carrying only the Step-3 synthesis into
the build — not the raw ~97 KB `engram query` payload — cut build_cost ~40% with zero capability loss, on
opus, n=3. The saving is mechanistic (the payload re-reads as `cache_read` every turn). The smoke proved the
*isolation premise* via a harness proxy; this is the production form.

## The core problem the mechanism must solve

To keep the payload out of the caller's context, the query + reasoning must happen in an **isolated context**
that returns only the synthesis. Claude Code has **no API to strip a tool result from a live context**, so the
payload must never enter it. An Agent-tool subagent boundary is **insufficient**: recall is invoked at
arbitrary nesting depth and often as a *leaf* subagent's first step (the "recall at decision moments"
doctrine), and subagents can't reliably spawn subagents. So the isolation boundary must be a **subprocess** the
binary controls — which reaches any depth.

## Mechanism

A new Go subcommand `engram recall`, running in the caller's process, whose heavy work happens behind a
`claude -p` boundary:

```
caller (any agent, any depth)
  │  Step 0-1: generate query phrases in its own context (needs live context; cheap, no payload)
  ▼
engram recall --mode deep|glance --phrase … --phrase …        ← Go, in caller's process
  │  1. sweep once: engram ingest --auto   (flock + atomic — REQUIRES Track 0)
  │  2. exec claude -p "<go:embed'd sub-recall prompt + phrases>"   (injected dep)
  │        └─ sub-recall (isolated context):
  │             engram query --lazy-chunks --phrase …            ← ~97 KB payload lands HERE
  │             Step 2.5 cluster judgment · crystallize (learn/amend) · Step 3 synthesis
  │             emits ONLY the synthesis nucleus as its final message
  │  3. capture result.result, print synthesis to stdout
  ▼
caller reads synthesis from stdout            ← payload never entered the caller
  │  Step 3-tail: walk its own plan against the synthesis (cheap, in-context)
```

**Interface (Joe's refinement):** the caller passes **queries**, not its ask/situation/plan. Query-gen
(recall Step 0-1) is inherently pre-retrieval and needs the caller's live context, so it stays in the caller;
the payload-heavy retrieve→judge→crystallize→synthesize is what we isolate. The subprocess returns the
**plan-independent nucleus** (the "apply these as requirements" list + crystallized lessons); the
**plan-relative walk** stays in the caller (cheap, keeps the plan private).

**Why `claude -p`, not a raw Anthropic API call:** the sub-recall is *agentic* — it runs `engram query`,
`engram show-chunk`, `engram amend`/`learn` as tools in a loop. `claude -p` provides that tool-use loop, plus
Claude Code auth (keychain), skill/prompt loading, and transcript capture for `/learn`. A raw API call would
force `engram` to implement an agent loop. (Consciously deviates from the "external API only for LLM
operations" wording — Joe approved the CLI-harness call; the agentic requirement justifies it.)

**Why this is the binary's first LLM dependency:** confirmed the query path makes zero external calls today
(only `update` execs `git`). Mechanism ported from the `$METER` harness `claude()` (`harness.py:77-101`):
`["claude","-p",<prompt>,"--output-format","json","--model",<id>,"--permission-mode","bypassPermissions"]`;
read `result.result` + `result.total_cost_usd`; rate-limit retry (15/45/120 s) + keychain cred refresh.

## Likely needs / call sites

- **Entry recall** (top agent's orient) and **leaf recall** (subagent mid-work) — identical path; the whole
  point is reaching leaves.
- **Each recall is an independent one-shot** — no session resumption. "Recall at the next decision moment" = a
  fresh `engram recall` call. (The harness `--resume` existed only to *measure* carry.)
- **Graceful degradation:** if `claude` is absent/unauthed, fall back to emitting the raw `engram query`
  payload (today's inline behavior) — no hard failure.
- **Self-contained prompt:** `go:embed` the sub-recall procedure into the binary (the recall procedure *minus*
  caller-side query-gen and the plan-walk), so it doesn't depend on the skill being installed in the `-p` env.

## Wiring (from the CLI map)

`internal/cli/recall.go`: `RecallArgs` (targ-tagged flags: `--mode`, repeated `--phrase`, `--model`, vault/
chunks), `RecallDeps` (inject the sweep, the exec-`claude` call, stdout — keeps "DI everywhere, no `os/exec` in
Run\*"), `RunRecall(ctx, args, deps, stdout)`, `newOsRecallDeps()`. Register in `internal/cli/targets.go`.
Unit test in `cli_test` with Gomega + manual deps (the `amend_test.go` pattern).

## Concurrency dependency (Track 0)

The parent `engram recall` sweeps once (flock + atomic — the Track 0 #660 fix); **sub-recalls skip the sweep**.
Sub-recall crystallization writes rely on Track 0: `learn` is already flock-safe; `amend` needs the flock
extension; sidecars need atomic writes. Without Track 0, N parallel sub-recalls corrupt the manifest/vault.

## Decisions made (veto-able)

- exec-`claude` is an injected dep (DI preserved).
- Sub-recall model defaults to the caller's `--model`, overridable — synthesis quality gates everything
  downstream, so don't cheap it out by default.
- `claude -p` over raw API (agentic requirement, above).

## Open forks (resolve at build time)

- **glance inline vs subprocess** (Joe deferred 2026-07-01 as part of splitting concurrency out). Leading
  option: deep→subprocess (the measured −40% case), glance→inline (a whole `claude -p` spawn likely costs more
  than a cheap ~3-phrase read-only glance's carry; glance-subprocess is unmeasured). Alternatives: both→
  subprocess (uniform, prunes leaf-glances, pays spawn overhead everywhere); a per-call `--isolate` flag.
- **return-path fidelity** — the synthesis must survive the subprocess→caller hop intact (note 149); validate
  when built (the smoke proxy skipped this).
- **sub-recall model/tier** — default decided above; confirm against the `route` skill (note 135).

## Validation approach (when built)

Re-confirm the −40% holds in the *real delegated topology* (the smoke measured a single-session topology, not
orchestrator + delegated subagents); trap gate GREEN; `recall_cost` `$METER`; a return-path-fidelity check.
