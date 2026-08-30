## Why

Issue #739 surfaces a structural gap in the delegation-boundary memory system: the delegate-everything doctrine moves object-level work into subagent contexts, but both recall (read side) and learn (write side) live in the orchestrator's context — memory surfaced to the orchestrator never reaches dispatched subagents, and corrections discovered inside subagents are discarded when the subagent context dies. ADR-0027 (accepted 2026-08-30) identified this as the highest-value unowned gap in the memory-taxonomy map, and the runbook chain (#738) is about to make task-start recall load-bearing — for orchestrators *and* for the subagents executing the tasks. Before Phase 2 ships a fix (injection at dispatch, harvest at completion), Phase 1 measures the gap's size so the addressable ceiling for Phase 2's approach is known.

This change is deliberately measurement-only: it produces two reusable classifiers/tools (write-side correction detector, read-side dispatch-handoff auditor), runs them across all of Joe's active repos' subagent transcripts (not engram-only), and reports DERIVED vs ESTIMATE-labeled numbers plus the go/no-go gate for opening a follow-on Phase 2 change.

## What Changes

**Phase 1 produces (measurement only — no production code changes):**

- **Write-side classifier** (reusable, open-ended): A semantic classifier detecting "correction-shaped moments inside a subagent transcript that never produced a vault note." Recovers the deleted `docs/design/2026-06-28-failure-eval-material.md` methodology (detect semantically, not by word-match; over-match then prune; gate on miss-rate; keep injection-point analysis open), re-derived fresh for subagent-specific criteria, applied to a cross-repo transcript corpus, and classified per vault runbook 824 (addressable / capture-ceiling / application-or-ranking-miss) so Phase 2's reach is known before it ships.

- **Read-side auditor** (reusable, two-tier): A structural proxy script (extract recalled-note basenames from orchestrator-recall calls, substring/fuzzy-match them against subsequent dispatch prompts in the same transcript, flag as "missing" if not found) followed by LLM-calibration on a sample of the proxy's flagged-as-missing cases to confirm the proxy isn't over/under-counting paraphrased handoffs. Across a cross-repo transcript corpus, excluding fork dispatches (D3: they structurally inherit the recalled context, so counting them would deflate the gap and understate the addressable ceiling).

- **Phase 1 report**: DERIVED/ESTIMATE-labeled result numbers, per runbook 824 classification, with explicit fire-unit pinning before any ratio computation. The report states the go/no-go gate for opening Phase 2 (e.g., analogous to #734's gate: "addressable ceiling on read-side dispatch leaks is ≤ X%, unblocks Phase 2 injection mechanism with Y% reachable"). Does NOT propose Phase 2 changes or scripts.

- **dev/eval/LEDGER.md row**: One row recording the Phase 1 findings (classifier results, auditor results, fire-unit, gate decision) in the existing format, following the precedent of the "734-runbook-retrieval-probe" entry.

**Out of scope (Phase 2):**
- Editing delegate.md, learn.md, or recall.md guidance
- Editing route/please SKILL.md files
- Implementing dispatch-side injection or completion-report templates
- Any changes to production agent behavior or subagent launch mechanism

## Capabilities

### New Capabilities

- `subagent-correction-detection`: The requirement that a Phase 1 write-side classifier detects "a correction-shaped moment in a subagent transcript (feedback/rejection/self-caught error) that never produced a corresponding vault note" semantically (not by keyword/word-match), over-matches then prunes to reduce false positives, gates on miss-rate (false-negatives) rather than false-positive rate, and preserves injection-point/context open for high-value finding analysis. The classifier serves as the input to Phase 2's write-side mechanism design.

- `delegation-dispatch-handoff-audit`: The requirement that a Phase 1 read-side auditor measures the gap between "notes the orchestrator recalled" and "notes actually present in the dispatch prompt sent to a subagent." Measured via a two-tier approach: structural proxy (fast, full-corpus) + LLM-calibration sample (confidence check on paraphrasing/implicit transfer). Results inform Phase 2's read-side injection mechanism. Fork dispatches excluded from the denominator (D3) since they structurally inherit recalled context.

## Impact

- New reusable tools under `dev/eval/` (semantic correction classifier, dispatch-handoff auditor proxy script)
- `dev/eval/LEDGER.md` — new dated row recording Phase 1 findings
- No changes to production code, guidance, or subagent mechanism
- Read-only against live subagent transcripts across multiple repos (structured traversal of `~/.claude/projects/*/subagents/` paths; Pi paths analogous)
- Gates a separate Phase 2 change to be filed only once Phase 1's addressable ceiling is known
- Closes/updates #739 (Phase 1 half only); Phase 1 is timely because #738 is about to make task-start recall load-bearing for orchestrators and subagents alike, intensifying the pressure on the delegation-boundary memory gap

