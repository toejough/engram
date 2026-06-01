# L3 Tier — Synthesis Flow (`/learn`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. The skill-edit task (Task 2) is GATED on superpowers:writing-skills (RED→GREEN→REFACTOR with pressure tests) — no SKILL.md edit without a failing behavioral test first.

**Goal:** Wire L3 synthesis into `/learn`: after L2 facts are captured, generate *scenario*-based seeds, ensure each new L2 is discoverable from those scenarios, cluster, and update-or-create the per-cluster L3 ADRs by semantic (centroid-cosine) match — each ADR linked to its L2s.

**Architecture:** One binary task surfaces, per cluster, the nearest existing L3 note + cosine (bridging `cluster.BestMatch` from Plan 1 to the `query` payload). Then the `/learn` skill orchestrates the LLM-judgment steps (scenario seeds, L2-tweak, ADR authoring, update-vs-create) using that payload. Decisions: trigger inside `/learn`; semantic match (centroid cosine ≥ ~0.9); ADR shape is a tunable body, not load-bearing.

**Tech Stack:** Go (`internal/cli/query.go`, reusing `cluster.BestMatch` + the existing per-cluster `Centroids`); `skills/learn/SKILL.md` via writing-skills TDD. `targ test` / `targ check-full`.

**Depends on:** Plan 1 (tier field, `--tier`, `cluster.BestMatch`).

---

### Task 1: surface per-cluster nearest-L3 match in `query`

**Files:**
- Modify: `internal/cli/query.go` (where `clusterReport`/clusters are rendered into the YAML payload; reuse `clusterReport.autoK.Centroids[c]` and `cluster.BestMatch`)
- Test: `internal/cli/query_test.go`

- [ ] **Step 1: Write the failing test.** A query over a vault with an L2 cluster and an existing L3 returns, on that cluster, a `nearest_l3` block (path + cosine).

```go
func TestRunQuery_ClusterReportsNearestL3(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	payload, err := runQueryForTest(vaultWithL2ClusterAndOneL3, QueryArgs{Phrases: seedPhrases})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	g.Expect(payload.Clusters).NotTo(BeEmpty())
	g.Expect(payload.Clusters[0].NearestL3.Path).To(ContainSubstring("Permanent/"))
	g.Expect(payload.Clusters[0].NearestL3.Cosine).To(BeNumerically(">", float32(0)))
}
```

- [ ] **Step 2: Run to verify it fails.** `targ test` — FAIL (`NearestL3` absent).

- [ ] **Step 3: Implement.** For each cluster, load the vectors of all `tier: L3` notes in the vault (reuse `loadCompatibleSidecars` + the `tierLineRE` from Plan 1 to select L3 notes), call `cluster.BestMatch(centroid, l3Vectors, 0.0)` (threshold 0 so we always report the best + its cosine; the skill applies the 0.9 cut), and attach `NearestL3{Path, Cosine}` (or null when no L3 notes exist) to the cluster in the payload struct + YAML.

- [ ] **Step 4: Run to verify it passes.** `targ test` — PASS.

- [ ] **Step 5: Coverage — no L3 present** → `nearest_l3` is null/omitted; add that test.

- [ ] **Step 6: Commit.**

```bash
git add internal/cli/query.go internal/cli/query_test.go
git commit -m "feat(query): report nearest existing L3 (centroid cosine) per cluster

AI-Used: [claude]"
```

---

### Task 2: add the L3-synthesis flow to the `/learn` skill  (GATED: superpowers:writing-skills)

**Files:**
- Modify: `skills/learn/SKILL.md` (new section after the episode/fact/feedback workflow)

- [ ] **Step 1 (RED): baseline behavioral test.** Dispatch a subagent told it just finished a build and wrote L2 facts via the *current* `/learn`, asked "do the L3 synthesis." Confirm it does NOT: (a) seed by *scenarios* (it seeds by lesson keywords or skips), (b) produce a tier-L3 ADR linked to the L2s, (c) update-vs-create by semantic match. Record the exact gaps (FORM, not keyword-presence).

- [ ] **Step 2 (GREEN): write the skill section.** Add `### 6b. L3 synthesis — scenario-discoverable ADRs` to `skills/learn/SKILL.md`, stating: when this `/learn` pass wrote L2 facts, for the new facts —
  1. **Scenario seeds:** enumerate 3–6 *situations an agent could be in where this fact should surface and be considered before acting* (situational/plan-grounded — NOT phrasings of the lesson; the agent won't know it needs the lesson).
  2. **Search + ensure discoverability:** `engram query --phrase "<scenario>" …`; if the new L2 doesn't rank high, revise the L2's `situation`/framing and re-embed (`engram embed apply --stale`) until it does.
  3. **Update-or-create per cluster:** read each returned cluster's `nearest_l3` (Task 1). Cosine ≥ 0.9 → **update** that L3 (regenerate, preferring recent L2s where they diverge); else **create** a new L3. Leave other L3s alone. (One L2 may land in several clusters → several L3s.)
  4. **Write the ADR:** `engram learn fact --tier L3 --slug <topic>-adr --relation "<each-L2-luhmann>|synthesized into this standard" …` with a short ADR body (title / 2–3 line context / the standard / derived-from). Treat the ADR shape as tunable.

- [ ] **Step 3 (GREEN verify): re-run the scenario.** With the section pasted in-prompt (subagents auto-load the *installed* skill — pin the version), confirm the agent now seeds by scenarios, produces a tier-L3 ADR `--relation`-linked to its L2s, and updates vs creates per the `nearest_l3` cosine.

- [ ] **Step 4 (REFACTOR): pressure tests.** (a) Tempt it to seed by lesson keywords under time pressure — confirm it still uses scenarios. (b) Give it a near-duplicate existing L3 (cosine 0.93) — confirm it updates, not duplicates. (c) Confirm the ADR stays short. Close loopholes in the section text.

- [ ] **Step 5: Propagate + verify.** `engram update`; confirm the installed `~/.claude/skills/learn/SKILL.md` and `~/.config/opencode/...` contain `6b`.

- [ ] **Step 6: Commit.**

```bash
git add skills/learn/SKILL.md
git commit -m "feat(skills): /learn L3 synthesis — scenario-seeded ADRs over L2 clusters

AI-Used: [claude]"
```

## Self-review notes
- Covers spec: scenario seeds (not keywords), discoverability-tweak, semantic update-vs-create, ADR + links, trigger inside `/learn`.
- The binary stays pure compute (Task 1 surfaces the match); all LLM-judgment is in the skill (Task 2).
