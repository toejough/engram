# L3 Tier — Validation Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. This plan builds a harness (scripts + scoring), not library code; "tests" here are smoke-runs of each script on a tiny fixture before the full chain.

**Goal:** The accumulation chain that measures whether **L3** (vs L1, vs L2) accumulates usefully across apps and survives app-order permutation — using real headless builds and skill-based `/recall` + `/learn` with the new tier machinery.

**Architecture:** Per (layer, order) chain, a fresh vault accumulates across three stages (cold → +1 prior → +2 prior). Each stage is a **headless `claude -p`** that (1) `/recall`s **tier-capped** (`--tier <arm>`), (2) builds the app, (3) `/learn`s **only the arm's tier** (L1→episodes; L2→facts, no L3; L3→facts + the Plan-2 L3 synthesis). Three cyclic orders of {todo, bookmarks, contacts} put each app in each position once. A separate scorer agent grades each build by pattern; synthesis reports the cold→+1→+2 curve per layer.

**Tech Stack:** bash drivers reusing `dev/eval/run-layer-arm.sh` + the provisioned cfg (`dev/eval/.layer-run/cfg`, recall+learn skills + keychain cred); `engram --tier`; agent-based scoring. Builds run headless (cred provisioning is the user's `security find-generic-password` step or the existing valid cfg).

**Depends on:** Plans 1 + 2 (tier field, `--tier` cap, the `/learn` L3 synthesis).

---

### Task 1: tier-isolated per-stage build+learn driver

**Files:**
- Create: `dev/eval/run-chain-stage.sh`

- [ ] **Step 1: Write the driver.** `run-chain-stage.sh <layer L1|L2|L3> <app> <vault> <workdir> <learn?yes|no>`:
  - Builds a prompt: "Build `<app spec>` in `<workdir>`. STEP 1: consult memory with the recall skill, but query engram **tier-capped**: `engram query --tier <layer> --phrase …` — apply what surfaces. STEP 2: build, make `go test` pass. STEP 3 (if learn=yes): use the learn skill but capture **only `<layer>`** — L1: write an episode of this build; L2: write facts (one per convention); L3: write facts AND run the §6b L3 synthesis (scenario-seeded ADRs)."
  - Runs headless `claude -p` (reuse `run-layer-arm.sh`'s invocation: `CLAUDE_CONFIG_DIR`, `ENGRAM_VAULT_PATH=<vault>`, engram on PATH, `--permission-mode bypassPermissions`, JSON output).
  - Saves result JSON + workspace + session id keyed by `<layer>-<app>-<vault-hash>`.

- [ ] **Step 2: Smoke-test** on one stage (L2, todo, empty vault): `bash -n` then a single real run; confirm it builds + writes only L2 facts (`engram query --tier L2` on the vault returns the facts; `--tier L3` returns nothing).

- [ ] **Step 3: Commit.**
```bash
git add dev/eval/run-chain-stage.sh
git commit -m "feat(eval): tier-isolated per-stage build+learn driver for the accumulation chain

AI-Used: [claude]"
```

---

### Task 2: chain orchestrator (3 layers × 3 orders × 3 stages)

**Files:**
- Create: `dev/eval/run-accumulation-chain.py`

- [ ] **Step 1: Write the orchestrator.** For `layer in [L1,L2,L3]` × `order in [[todo,bm,contacts],[bm,contacts,todo],[contacts,todo,bm]]`: a fresh vault `/tmp/chain-<layer>-o<i>/vault`; run 3 stages sequentially via `run-chain-stage.sh` (stage 3 = `learn=no`), threading the SAME vault so it accumulates; record per-stage {app, priorCount, workdir, session}. Chains are independent → run them with a small process pool (cap ~6) but **stages within a chain strictly sequential** (vault accumulation).

- [ ] **Step 2: Smoke-test** one full chain (L2, order 1) end-to-end at small scale; confirm the vault grows across stages and stage-2 recall surfaces stage-1's memory.

- [ ] **Step 3: Commit.**
```bash
git add dev/eval/run-accumulation-chain.py
git commit -m "feat(eval): accumulation-chain orchestrator (3 layers x 3 orders x 3 stages)

AI-Used: [claude]"
```

---

### Task 3: scoring + accumulation-curve synthesis

**Files:**
- Create: `dev/eval/score-chain.py`

- [ ] **Step 1: Write the scorer + synthesis.** A separate **agent** scorer (name-agnostic, judges the *pattern*) grades each build's workspace against the 10-item architecture rubric (`dev/eval/testdata/layer-vaults/contacts-rubric.md`, generalized to all three apps). Aggregate per layer the mean arch score at priorCount 0 / 1 / 2 (the **cold → +1 → +2 curve**), and per-order so order-dependence is visible. Report turns/cost/time per stage too.

- [ ] **Step 2: Run the full chain + score.** Execute `run-accumulation-chain.py`, then `score-chain.py`. Headline: does the curve *rise* cold→+1→+2 for L3 (accumulation helping), and is it order-stable?

- [ ] **Step 3: Record results** in `docs/superpowers/specs/2026-05-31-l3-synthesis-tier-design.md` under a Results section. Commit the harness (NOT the `/tmp` runs or the cred cfg — those stay gitignored under `dev/eval/.layer-run/`).
```bash
git add dev/eval/score-chain.py docs/superpowers/specs/2026-05-31-l3-synthesis-tier-design.md
git commit -m "feat(eval): chain scoring + accumulation-curve synthesis; record L3 results

AI-Used: [claude]"
```

## Self-review notes
- Tier isolation is enforced two ways: recall via `--tier <layer>` (Plan 1) and learn instructed to capture only that layer.
- The cyclic-order design gives each app a cold/+1/+2 reading, controlling order with 9 chains instead of all 18 permutations.
- Honest caveat to carry into Results: builds are headless clean-room agents; scoring is agent-judged by pattern (avoids the earlier regex name-bias).
