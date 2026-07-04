#!/usr/bin/env bash
# P1 — Retrieval-pollution probe (four arms, ~$0 local)
#
# Claim: EMBEDDED synthetic QA nodes WITHOUT the qa-kind exclusion pollute the
# matched set; WITH all four seam points they cause zero disturbance.
#
# Arms:
#   Arm 0  — copy vault, no QA nodes, current binary (fresh re-baseline)
#   Arm 1  — copy vault + 5 embedded QA pairs, current binary (pollution measurement)
#   Arm 2  — copy vault + 5 embedded QA pairs, probe-only PATCHED binary (exclusion gate)
#   Arm V  — value: paraphrased Q cosine vs Q notes vs content notes (D5 channel premise)
#
# Pre-registered criteria (interpret VERBATIM):
#   Arm 2 PASS = QA notes appear NOWHERE in items[] (including tag_nominations_added)
#            AND 0 top-5 disturbances vs Arm 0
#   Arm 2 FAIL = any QA surfacing or any displaced note
#   Arm V PASS = >=8/10 paraphrases rank own Q note #1 among Q notes AND above every content note
#   Arm V BORDERLINE = 6-7/10
#   Arm V FAIL = <6/10
#
# Safety: LIVE_VAULT is never written. COPY_VAULT in $WORK_DIR only.
# Committed before running per plan mandate.
#
# Usage: bash dev/eval/qa/p1_retrieval_pollution.sh [--results-dir <dir>]
set -euo pipefail

# ---------------------------------------------------------------------------
# 1. Safety setup
# ---------------------------------------------------------------------------
LIVE_VAULT="${ENGRAM_VAULT_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault}"
WORK_DIR=$(mktemp -d -t qa-probe-XXXXXX)
COPY_VAULT="$WORK_DIR/qa-pollution-probe-vault"
RESULTS_DIR="${1:-$(dirname "$0")}"
DATE="2026-07-03"

echo "=== P1 Retrieval-Pollution Probe ==="
echo "LIVE_VAULT: $LIVE_VAULT"
echo "WORK_DIR:   $WORK_DIR"
echo "COPY_VAULT: $COPY_VAULT"
echo "RESULTS_DIR: $RESULTS_DIR"
echo ""

# Sanity check: live vault exists
[ -d "$LIVE_VAULT" ] || { echo "ERROR: LIVE_VAULT missing: $LIVE_VAULT"; exit 1; }

# Copy vault (no QA nodes yet)
cp -r "$LIVE_VAULT" "$COPY_VAULT"
[ -d "$COPY_VAULT" ] || { echo "ERROR: COPY_VAULT missing after copy"; exit 1; }
echo "Copied vault: $(ls $COPY_VAULT/*.md 2>/dev/null | wc -l) .md files"

# ---------------------------------------------------------------------------
# 2. Synthetic QA pairs (5 pairs = 10 files)
#    A notes MUST carry vocab: terms from the real vault vocab set.
#    Point D (TermIndex) only fires for vocab-tagged notes — omitting them
#    makes Arm 2's Point-D pass vacuous.
# ---------------------------------------------------------------------------
echo ""
echo "--- Writing 5 synthetic QA pairs ---"

# Pair 1: eval checkpoint pattern
cat > "$COPY_VAULT/qa.${DATE}.eval-checkpoint.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.eval-checkpoint.a
---
What should an eval harness do to survive orchestrator process reaping or a session-limit window termination?

Answered by: [[qa.2026-07-03.eval-checkpoint.a]]
QEOF

cat > "$COPY_VAULT/qa.${DATE}.eval-checkpoint.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.eval-checkpoint.q
certainty: high
contributors: [159.2026-07-02.eval-runs-checkpoint-per-trial-and-survive-orchestrator]
vocab: [eval-methodology, behavioral-failure-reproduction]
---
An eval harness that runs many trials over minutes-to-hours must: (1) append each trial result to a JSONL checkpoint file immediately after the trial completes (flush per write), with resume-skip logic so a restarted run skips already-completed triples; (2) launch via nohup or disown so the process survives the orchestrator's reaping; (3) treat near-zero-cost trials as degraded builds — discard and re-run them, never pool with clean results; (4) check the account usage window before launching a long run to avoid the session-limit window.

Answers: [[qa.2026-07-03.eval-checkpoint.q]]
Contributors: [[159.2026-07-02.eval-runs-checkpoint-per-trial-and-survive-orchestrator]]
AEOF

# Pair 2: copy-vault safety
cat > "$COPY_VAULT/qa.${DATE}.copy-vault-safety.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.copy-vault-safety.a
---
How should probe scripts protect the live vault from contamination while running eval experiments?

Answered by: [[qa.2026-07-03.copy-vault-safety.a]]
QEOF

cat > "$COPY_VAULT/qa.${DATE}.copy-vault-safety.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.copy-vault-safety.q
certainty: high
contributors: [160.2026-07-03.eval-arms-escape-sandbox-via-payload-paths]
vocab: [eval-methodology, eval-fixture-design, cli-verification]
---
Probe scripts must use a copy vault only: set ENGRAM_VAULT_PATH to a temporary copy (cp -r LIVE_VAULT COPY_VAULT in a mktemp -d WORK_DIR), never the live vault path. Use set -u to catch unset vars. Verify git status of the live vault BEFORE starting and AFTER completing to confirm no writes occurred. Never give eval arms bypassPermissions when the real repo is reachable. Strip or rewrite absolute paths (like /Users/joe/repos/...) in eval payloads before injection so agents cannot follow them out of the sandbox.

Answers: [[qa.2026-07-03.copy-vault-safety.q]]
Contributors: [[160.2026-07-03.eval-arms-escape-sandbox-via-payload-paths]]
AEOF

# Pair 3: observable attribution
cat > "$COPY_VAULT/qa.${DATE}.observable-attribution.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.observable-attribution.a
---
How should contributor attribution for Q&A notes be captured to avoid confabulation?

Answered by: [[qa.2026-07-03.observable-attribution.a]]
QEOF

cat > "$COPY_VAULT/qa.${DATE}.observable-attribution.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.observable-attribution.q
certainty: high
contributors: [145.2026-06-30.recall-value-gate-not-holdable-by-wording-naming-primes]
vocab: [behavioral-failure-reproduction, vocabulary-crystallization, eval-methodology]
---
Attribution must be cite-derived: contributors are notes whose full-basename wikilinks appear in the WRITTEN ANSWER body text, not a list the agent generates by free-recall at close. An agent free-listing "which notes did you use?" confabulates significantly. The observable bar is: the answer body contains [[basename]] links; the contributor list is MACHINE-WRITTEN by extracting those links, never hand-typed by the agent. This ensures attribution is verifiable and durable — a human can read the note and check.

Answers: [[qa.2026-07-03.observable-attribution.q]]
Contributors: [[145.2026-06-30.recall-value-gate-not-holdable-by-wording-naming-primes]]
AEOF

# Pair 4: QA exclusion from main set
cat > "$COPY_VAULT/qa.${DATE}.qa-exclusion-seam.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.qa-exclusion-seam.a
---
Why must QA question and answer notes be excluded from the main cosine matched set in engram recall?

Answered by: [[qa.2026-07-03.qa-exclusion-seam.a]]
QEOF

cat > "$COPY_VAULT/qa.${DATE}.qa-exclusion-seam.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.qa-exclusion-seam.q
certainty: high
contributors: [153.2026-07-01.keep-concrete-token-in-idiosyncratic-notes]
vocab: [retrieval-design, memory-system-architecture, vocabulary-crystallization]
---
QA notes must be excluded from the main cosine matched set because question-shaped wording loses retrieval: embedding a question like "how to handle X?" in the main cosine space competes poorly against content notes (embeddings tuned to topic descriptions, not question forms). The QA notes' sanctioned retrieval path is a DEDICATED Q-channel (cosine of incoming ask against Q-note embeddings only, additive payload section), not competition with content notes. Without exclusion at all four seam points in the query pipeline (pre-clustering filter, matched-set floor/cap, tag-nomination gate, TermIndex builder), QA notes pollute the main matched set and displace useful content notes.

Answers: [[qa.2026-07-03.qa-exclusion-seam.q]]
Contributors: [[153.2026-07-01.keep-concrete-token-in-idiosyncratic-notes]]
AEOF

# Pair 5: usage signal design
cat > "$COPY_VAULT/qa.${DATE}.usage-signal-design.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.usage-signal-design.a
---
What mechanism should track which vault notes are most useful over time for retention and triage decisions?

Answered by: [[qa.2026-07-03.usage-signal-design.a]]
QEOF

cat > "$COPY_VAULT/qa.${DATE}.usage-signal-design.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.usage-signal-design.q
certainty: high
contributors: [99.2026-06-26.verified-benefit-ledger-memory-wins-are-idiosyncratic-capability]
vocab: [memory-system-architecture, retrieval-design, lever-tracking]
---
Contribution in-degree via Q&A nodes: each A note (answer) carries a contributors list of the vault notes whose content fed the answer. Counting how many A notes cite a given content note (InDegreeIn from vaultgraph, restricted to the QA-node set) gives a graded usage signal. Unlike binary activation (last-used date only), this counts how many times a note has been the source of a delivered answer — frequently-cited notes are proven keepers; never-cited notes are triage candidates. This signal is derived at read time from the body wikilinks, requiring no persistent counter that can drift.

Answers: [[qa.2026-07-03.usage-signal-design.q]]
Contributors: [[99.2026-06-26.verified-benefit-ledger-memory-wins-are-idiosyncratic-capability]]
AEOF

echo "Wrote 10 QA files to $COPY_VAULT"

# ---------------------------------------------------------------------------
# 3. Embed the new QA nodes (missing sidecars only)
# ---------------------------------------------------------------------------
echo ""
echo "--- Embedding new QA nodes (local model, \$0) ---"
EMBED_CMD="ENGRAM_VAULT_PATH=$COPY_VAULT engram embed apply"
echo "Command: $EMBED_CMD"
ENGRAM_VAULT_PATH="$COPY_VAULT" engram embed apply 2>&1 | tee "$WORK_DIR/embed.log"
echo "Embedding done."

# ---------------------------------------------------------------------------
# 4. Build case list (48 cases: 36 P1 + 8 P2 + 4 P3)
# ---------------------------------------------------------------------------
LINKS_DIR="$(dirname "$0")/../links"
QUERIES_JSON="$LINKS_DIR/queries.json"
MISSES_JSON="$LINKS_DIR/misses_p1.json"
BRIDGES_JSON="$LINKS_DIR/bridges_p2.json"
SUPSN_JSON="$LINKS_DIR/supersession_p3.json"

# Build the cases JSON via Python (inline, no temp file dependency)
CASES_JSON="$WORK_DIR/cases.json"
python3 - <<'PYEOF' "$QUERIES_JSON" "$MISSES_JSON" "$BRIDGES_JSON" "$SUPSN_JSON" "$CASES_JSON"
import json, sys

queries_path, misses_path, bridges_path, supsn_path, out_path = sys.argv[1:]

queries = {q["id"]: q["phrases"] for q in json.load(open(queries_path))}
misses  = json.load(open(misses_path))
bridges = json.load(open(bridges_path))
supsns  = json.load(open(supsn_path))

cases = []

# P1 cases: misses with (query_id, n) -> phrases[:n]
for m in misses:
    qid, n = m["query_id"], m["n"]
    phrases = queries.get(qid, [])[:n]
    if not phrases:
        print(f"WARN: no phrases for {qid}", flush=True)
        continue
    cases.append({"case_id": f"P1-{qid}-n{n}", "kind": "P1",
                  "phrases": phrases, "needed": m["missed_note"]})

# Bridge cases: own phrases
for b in bridges:
    cases.append({"case_id": f"P2-{b['case_id']}", "kind": "P2",
                  "phrases": b["phrases"], "needed": b["needed_note"]})

# Supersession cases: P3-4..7 (supersession_miss=True)
for s in supsns:
    if s.get("supersession_miss"):
        cases.append({"case_id": f"P3-{s['pair_id']}", "kind": "P3",
                      "phrases": s["phrases"], "needed": s["new_note"]})

print(f"Cases: {len(cases)} total ({sum(1 for c in cases if c['kind']=='P1')} P1, "
      f"{sum(1 for c in cases if c['kind']=='P2')} P2, "
      f"{sum(1 for c in cases if c['kind']=='P3')} P3)", flush=True)
json.dump(cases, open(out_path, "w"), indent=2)
PYEOF

# ---------------------------------------------------------------------------
# 5. Run query against vault, parse items[] for QA notes and top-5
# ---------------------------------------------------------------------------
# Helper: run engram query with phrases against vault, return YAML output
run_arm () {
    local vault="$1"; shift
    local binary="$1"; shift
    local arm_label="$1"; shift
    local cases_json="$1"; shift
    local out_json="$1"; shift

    echo "  Running $arm_label ($("${binary}" version 2>/dev/null || echo 'current') against $vault)"

    python3 - "$vault" "$binary" "$arm_label" "$cases_json" "$out_json" <<'PYEOF'
import json, os, re, subprocess, sys

vault, binary, arm_label, cases_path, out_path = sys.argv[1:]
cases = json.load(open(cases_path))

QA_PREFIX = ("qa.", "qa.")  # both qa.<date>.*.q.md and qa.<date>.*.a.md
QA_PAT = re.compile(r'^qa\.')

results = []
for case in cases:
    cmd = [binary, "query", "--lazy-chunks"]
    for p in case["phrases"]:
        cmd += ["--phrase", p]
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    r = subprocess.run(cmd, capture_output=True, text=True, env=env)

    # Parse items[]
    items = []
    rank = 0
    cur_path = None; cur_score = None; cur_kind = None
    in_items = False
    for line in r.stdout.splitlines():
        if line == "items:":
            in_items = True; continue
        if in_items and re.match(r"^[a-z_]+:", line) and not line.startswith("  "):
            if "clusters:" in line or "metadata:" in line:
                in_items = False
                if cur_path:
                    rank += 1
                    items.append({"basename": cur_path, "score": cur_score,
                                  "kind": cur_kind, "rank": rank})
                    cur_path = cur_score = cur_kind = None
            continue
        if not in_items: continue
        pm = re.match(r"^  - path:\s*(.+?)\s*$", line)
        if pm:
            if cur_path:
                rank += 1
                items.append({"basename": cur_path, "score": cur_score,
                               "kind": cur_kind, "rank": rank})
            raw = pm.group(1).strip('"').split("#")[0]
            cur_path = os.path.basename(raw)
            cur_score = None; cur_kind = None; continue
        if cur_path:
            sm = re.match(r"^\s+score:\s*([\d.]+)\s*$", line)
            if sm: cur_score = float(sm.group(1)); continue
            km = re.match(r"^\s+kind:\s*(\S+)\s*$", line)
            if km: cur_kind = km.group(1)
    if in_items and cur_path:
        rank += 1
        items.append({"basename": cur_path, "score": cur_score,
                       "kind": cur_kind, "rank": rank})

    # Check tag_nominations_added from metadata
    tag_nominations = 0
    for line in r.stdout.splitlines():
        m = re.match(r"\s*tag_nominations_added:\s*(\d+)", line)
        if m: tag_nominations = int(m.group(1))

    qa_in_items = [it for it in items if QA_PAT.match(it["basename"])]
    top5 = [it for it in items if it["rank"] <= 5 and it["kind"] in ("fact", "feedback")]

    results.append({
        "case_id": case["case_id"],
        "kind": case["kind"],
        "needed": case["needed"],
        "qa_in_items": qa_in_items,
        "top5_notes": top5,
        "total_items": len(items),
        "tag_nominations_added": tag_nominations,
        "returncode": r.returncode,
    })
    if qa_in_items:
        print(f"  [{arm_label}] {case['case_id']}: QA nodes in items: {[x['basename'] for x in qa_in_items]}")

json.dump(results, open(out_path, "w"), indent=2)
print(f"  [{arm_label}] Done: {len(results)} cases, "
      f"{sum(1 for r in results if r['qa_in_items'])} with QA in items[]", flush=True)
PYEOF
}

# ---------------------------------------------------------------------------
# 6. Arm 0: current binary, no QA nodes (fresh re-baseline)
# ---------------------------------------------------------------------------
echo ""
echo "=== ARM 0: Re-baseline (no QA nodes, current binary) ==="
# Use a vault WITHOUT QA files = original copy (before we added QA files)
# We've already added QA files to COPY_VAULT. Create a clean copy.
COPY_VAULT_NOQA="$WORK_DIR/qa-probe-noqa-vault"
cp -r "$LIVE_VAULT" "$COPY_VAULT_NOQA"
ARM0_JSON="$WORK_DIR/arm0_results.json"
run_arm "$COPY_VAULT_NOQA" "engram" "ARM0" "$CASES_JSON" "$ARM0_JSON"

# ---------------------------------------------------------------------------
# 7. Arm 1: current binary, with QA nodes (pollution measurement)
# ---------------------------------------------------------------------------
echo ""
echo "=== ARM 1: Pollution measurement (QA nodes + current binary) ==="
ARM1_JSON="$WORK_DIR/arm1_results.json"
run_arm "$COPY_VAULT" "engram" "ARM1" "$CASES_JSON" "$ARM1_JSON"

# ---------------------------------------------------------------------------
# 8. Arm 2: patched binary (probe-only worktree, all 4 seam points)
# ---------------------------------------------------------------------------
echo ""
echo "=== ARM 2: Exclusion gate (patched binary, all 4 seam points) ==="
REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
WT_DIR="$WORK_DIR/qa-probe-worktree"
PATCHED_BINARY="$WORK_DIR/engram-qa-probe"

echo "Creating git worktree at $WT_DIR..."
git -C "$REPO_ROOT" worktree add "$WT_DIR" HEAD 2>&1

echo "Patching all 4 seam points..."
# Patch vocab.go: extend isVocabKind to also exclude qa-question and qa-answer
# This single change covers all 4 callers (A:435, A:1084, B:846, C:95) + adds Point D coverage
# Point D is in query_nominations.go:337 which also calls isVocabKind via the same function.
# Strategy: modify isVocabKind itself to include QA kinds, so all 4 callers are covered at once.
sed -i '' \
    's/return kind == typeVocab || kind == typeVocabIndex/return kind == typeVocab || kind == typeVocabIndex || kind == "qa-question" || kind == "qa-answer"/' \
    "$WT_DIR/internal/cli/vocab.go"

echo "Verifying patch..."
grep -n 'qa-question' "$WT_DIR/internal/cli/vocab.go" || { echo "ERROR: patch not applied"; exit 1; }

echo "Building patched binary..."
(cd "$WT_DIR" && go build -o "$PATCHED_BINARY" ./cmd/engram) 2>&1
[ -x "$PATCHED_BINARY" ] || { echo "ERROR: binary not built"; exit 1; }
echo "Patched binary: $PATCHED_BINARY"

echo "Running Arm 2 with patched binary..."
ARM2_JSON="$WORK_DIR/arm2_results.json"
run_arm "$COPY_VAULT" "$PATCHED_BINARY" "ARM2" "$CASES_JSON" "$ARM2_JSON"

echo "Removing worktree..."
git -C "$REPO_ROOT" worktree remove "$WT_DIR" --force 2>&1
echo "Worktree removed."

# ---------------------------------------------------------------------------
# 9. Arm V: value arm — Q-note cosine vs paraphrases (D5 channel premise)
# ---------------------------------------------------------------------------
echo ""
echo "=== ARM V: Value arm (paraphrase-to-Q-note cosine ranking) ==="

# 10 paraphrases (2 per pair) — authored here, committed with script
PARAPHRASES_JSON="$WORK_DIR/paraphrases.json"
python3 - "$PARAPHRASES_JSON" <<'PYEOF'
import json, sys
# 10 paraphrases authored inline (2 per synthetic pair)
paraphrases = [
    # Pair 1: eval-checkpoint
    {"pair": "eval-checkpoint",
     "q_slug": "qa.2026-07-03.eval-checkpoint.q",
     "paraphrase": "How to make eval harnesses survive session-limit window terminations?",
     "id": "V01a"},
    {"pair": "eval-checkpoint",
     "q_slug": "qa.2026-07-03.eval-checkpoint.q",
     "paraphrase": "What pattern prevents losing eval run progress when the orchestrator process dies?",
     "id": "V01b"},
    # Pair 2: copy-vault-safety
    {"pair": "copy-vault-safety",
     "q_slug": "qa.2026-07-03.copy-vault-safety.q",
     "paraphrase": "How do I copy a vault safely for use in an eval without touching live data?",
     "id": "V02a"},
    {"pair": "copy-vault-safety",
     "q_slug": "qa.2026-07-03.copy-vault-safety.q",
     "paraphrase": "What shell-script guards protect the production vault during probe experiments?",
     "id": "V02b"},
    # Pair 3: observable-attribution
    {"pair": "observable-attribution",
     "q_slug": "qa.2026-07-03.observable-attribution.q",
     "paraphrase": "What citation pattern for note attribution avoids agent confabulation at close?",
     "id": "V03a"},
    {"pair": "observable-attribution",
     "q_slug": "qa.2026-07-03.observable-attribution.q",
     "paraphrase": "How to derive which notes contributed to an answer from the written text objectively?",
     "id": "V03b"},
    # Pair 4: qa-exclusion-seam
    {"pair": "qa-exclusion-seam",
     "q_slug": "qa.2026-07-03.qa-exclusion-seam.q",
     "paraphrase": "Why must QA question and answer notes be excluded from the main recall cosine set?",
     "id": "V04a"},
    {"pair": "qa-exclusion-seam",
     "q_slug": "qa.2026-07-03.qa-exclusion-seam.q",
     "paraphrase": "What happens if QA nodes compete with content notes in engram's cosine retrieval?",
     "id": "V04b"},
    # Pair 5: usage-signal-design
    {"pair": "usage-signal-design",
     "q_slug": "qa.2026-07-03.usage-signal-design.q",
     "paraphrase": "How to build a usage frequency signal from Q&A contribution links in a memory vault?",
     "id": "V05a"},
    {"pair": "usage-signal-design",
     "q_slug": "qa.2026-07-03.usage-signal-design.q",
     "paraphrase": "What accumulates which notes get cited across Q&A pairs for retention decisions over time?",
     "id": "V05b"},
]
json.dump(paraphrases, open(sys.argv[1], "w"), indent=2)
print(f"Wrote {len(paraphrases)} paraphrases")
PYEOF

ARMV_JSON="$WORK_DIR/armv_results.json"

python3 - "$COPY_VAULT" "$PARAPHRASES_JSON" "$ARMV_JSON" <<'PYEOF'
"""Arm V: For each paraphrase, run engram query and check if the target Q note
ranks #1 among Q notes AND above every content note.

Uses the CURRENT binary (no exclusion), so QA notes DO appear in items[].
Checks: target Q note cosine vs all other notes.
"""
import json, os, re, subprocess, sys

vault, para_path, out_path = sys.argv[1:]
paraphrases = json.load(open(para_path))
QA_Q_PAT = re.compile(r'^qa\..*\.q\.md$')
QA_A_PAT = re.compile(r'^qa\..*\.a\.md$')

def run_query(vault, phrase):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    r = subprocess.run(["engram", "query", "--lazy-chunks", "--phrase", phrase],
                       capture_output=True, text=True, env=env)
    items = []
    rank = 0
    cur_path = None; cur_score = None; cur_kind = None
    in_items = False
    for line in r.stdout.splitlines():
        if line == "items:":
            in_items = True; continue
        if in_items and re.match(r"^[a-z_]+:", line) and not line.startswith("  "):
            if "clusters:" in line or "metadata:" in line:
                in_items = False
                if cur_path:
                    rank += 1
                    items.append({"basename": cur_path, "score": cur_score or 0.0, "rank": rank})
                    cur_path = cur_score = cur_kind = None
            continue
        if not in_items: continue
        pm = re.match(r"^  - path:\s*(.+?)\s*$", line)
        if pm:
            if cur_path:
                rank += 1
                items.append({"basename": cur_path, "score": cur_score or 0.0, "rank": rank})
            raw = pm.group(1).strip('"').split("#")[0]
            cur_path = os.path.basename(raw)
            cur_score = None; continue
        if cur_path:
            sm = re.match(r"^\s+score:\s*([\d.]+)\s*$", line)
            if sm: cur_score = float(sm.group(1))
    if in_items and cur_path:
        rank += 1
        items.append({"basename": cur_path, "score": cur_score or 0.0, "rank": rank})
    return items

results = []
pass_count = 0
print(f"{'Para ID':>8} {'Q slug':>44} {'Q score':>8} {'top content':>10}  pass?")
print("-" * 85)

for p in paraphrases:
    target_q = p["q_slug"] + ".md"
    items = run_query(vault, p["paraphrase"])

    # Partition: Q notes, A notes, content notes
    q_notes = [it for it in items if QA_Q_PAT.match(it["basename"])]
    content_notes = [it for it in items if it.get("basename", "").endswith(".md")
                     and not QA_Q_PAT.match(it["basename"])
                     and not QA_A_PAT.match(it["basename"])]

    target_in_q = next((it for it in q_notes if it["basename"] == target_q), None)
    target_score = target_in_q["score"] if target_in_q else 0.0

    # Rank among Q notes (1 = best)
    q_sorted = sorted(q_notes, key=lambda x: x["score"], reverse=True)
    q_rank = next((i+1 for i, it in enumerate(q_sorted) if it["basename"] == target_q), None)

    # Best content note score
    top_content_score = max((it["score"] for it in content_notes), default=0.0)

    # PASS criteria: target Q is #1 among Q notes AND above every content note
    passes = (q_rank == 1 and target_score > top_content_score)
    if passes:
        pass_count += 1

    row = {
        "id": p["id"],
        "q_slug": p["q_slug"],
        "paraphrase": p["paraphrase"][:60],
        "target_score": target_score,
        "q_rank_among_q_notes": q_rank,
        "top_content_score": top_content_score,
        "passes": passes,
    }
    results.append(row)
    status = "PASS" if passes else "FAIL"
    print(f"{p['id']:>8} {p['q_slug'][-44:]:>44} {target_score:>8.4f} {top_content_score:>10.4f}  {status}")

print(f"\nArm V: {pass_count}/10 paraphrases PASS")
if pass_count >= 8:
    verdict = "PASS (>=8/10)"
elif pass_count >= 6:
    verdict = "BORDERLINE (6-7/10) — channel viable but needs larger-n check"
else:
    verdict = "FAIL (<6/10) — Q-to-Q matching too weak for D5 channel"
print(f"Arm V verdict: {verdict}")

json.dump({"results": results, "pass_count": pass_count, "verdict": verdict}, open(out_path, "w"), indent=2)
PYEOF

# ---------------------------------------------------------------------------
# 10. Compare and report
# ---------------------------------------------------------------------------
echo ""
echo "=== RESULTS COMPARISON ==="

python3 - "$ARM0_JSON" "$ARM1_JSON" "$ARM2_JSON" "$ARMV_JSON" <<'PYEOF'
import json

def load(p):
    return json.load(open(p))

arm0 = load(sys.argv[1] if False else "/dev/null") ; import sys
arm0 = load(sys.argv[1])
arm1 = load(sys.argv[2])
arm2 = load(sys.argv[3])
armv = load(sys.argv[4])

print("\n--- Arm 0 (baseline, no QA) ---")
print(f"  Cases: {len(arm0)}")
print(f"  QA in items: 0 (baseline, no QA files)")

print("\n--- Arm 1 (current binary + QA nodes) ---")
qa_appearances = sum(1 for r in arm1 if r.get("qa_in_items"))
total_qa_hits = sum(len(r.get("qa_in_items", [])) for r in arm1)
print(f"  Cases with QA in items[]: {qa_appearances}/{len(arm1)}")
print(f"  Total QA item appearances: {total_qa_hits}")

# Top-5 disturbances vs Arm 0
arm0_by_case = {r["case_id"]: {it["basename"] for it in r.get("top5_notes", [])} for r in arm0}
disturbances = 0
for r1 in arm1:
    a0_top5 = arm0_by_case.get(r1["case_id"], set())
    a1_top5 = {it["basename"] for it in r1.get("top5_notes", [])}
    if a0_top5 and a0_top5 != a1_top5:
        disturbances += 1
print(f"  Top-5 disturbances vs Arm 0: {disturbances}/{len(arm1)}")

print("\n--- Arm 2 (patched binary + QA nodes) ---")
qa_in_arm2 = sum(1 for r in arm2 if r.get("qa_in_items"))
qa_hits_arm2 = sum(len(r.get("qa_in_items", [])) for r in arm2)
print(f"  Cases with QA in items[]: {qa_in_arm2}/{len(arm2)}")
print(f"  Total QA item appearances: {qa_hits_arm2}")
tag_nominations = sum(r.get("tag_nominations_added", 0) for r in arm2)
print(f"  Total tag_nominations_added (all cases): {tag_nominations}")

# Top-5 disturbances arm2 vs arm0
disturbances_arm2 = 0
for r2 in arm2:
    a0_top5 = arm0_by_case.get(r2["case_id"], set())
    a2_top5 = {it["basename"] for it in r2.get("top5_notes", [])}
    if a0_top5 and a0_top5 != a2_top5:
        disturbances_arm2 += 1
print(f"  Top-5 disturbances vs Arm 0: {disturbances_arm2}/{len(arm2)}")

print("\n--- Arm 2 PRE-REGISTERED VERDICT ---")
qa_surfaced = qa_in_arm2 > 0 or qa_hits_arm2 > 0 or tag_nominations > 0
any_disturbance = disturbances_arm2 > 0
if not qa_surfaced and not any_disturbance:
    print("  PASS: QA notes absent from items[] and tag_nominations=0; 0 top-5 disturbances")
else:
    reasons = []
    if qa_surfaced:
        reasons.append(f"QA appeared: {qa_hits_arm2} items, {tag_nominations} tag nominations")
    if any_disturbance:
        reasons.append(f"{disturbances_arm2} top-5 disturbances")
    print(f"  FAIL: {'; '.join(reasons)}")
    print("  NOTE: name the leaking seam point(s) from items[] basenames above")

print("\n--- Arm V PRE-REGISTERED VERDICT ---")
print(f"  {armv['verdict']}")
PYEOF

# ---------------------------------------------------------------------------
# 11. Write results JSON and clean up
# ---------------------------------------------------------------------------
RESULTS_JSON="$RESULTS_DIR/p1_results.json"
python3 - "$ARM0_JSON" "$ARM1_JSON" "$ARM2_JSON" "$ARMV_JSON" "$RESULTS_JSON" <<'PYEOF'
import json, sys
arm0 = json.load(open(sys.argv[1]))
arm1 = json.load(open(sys.argv[2]))
arm2 = json.load(open(sys.argv[3]))
armv = json.load(open(sys.argv[4]))
out = {"arm0": arm0, "arm1": arm1, "arm2": arm2, "armV": armv}
json.dump(out, open(sys.argv[5], "w"), indent=2)
print(f"Results written to {sys.argv[5]}")
PYEOF

echo ""
echo "=== P1 probe complete ==="
echo "WORK_DIR preserved at: $WORK_DIR (remove manually when done)"
echo ""

# Remind caller to check vault contamination
echo "IMPORTANT: verify vault contamination check:"
echo "  git -C '$LIVE_VAULT' status --porcelain"

# ---------------------------------------------------------------------------
# 12. Arm V LARGE-N: direct sidecar cosine (>=30 paraphrases, >=10 topics)
#
# Pre-registered bands (verbatim, non-negotiable):
#   PASS       >= 80% paraphrases rank own Q note #1 among Q notes AND above every content note
#   BORDERLINE 60-79%
#   FAIL       < 60%
#
# Measurement: DIRECT SIDECAR COSINE — loads every .vec.json in the copy vault,
# embeds each paraphrase via `engram embed apply` (probe embedder path, same as
# seeding above), ranks raw body_vector cosines. NEVER through `engram query` —
# after Task 2's D5' exclusion, the production binary excludes qa-question at all
# four seam points; a query-pipeline probe would return 0/30 Q-note retrievals
# and record a FALSE FAIL on the round-3 gate.
#
# Self-seeding: 10 real vault topics (new, not the 5 synthetic P1 topics).
# Uses separate WORK_DIR_LN / COPY_VAULT_LN — never writes to live vault.
# ---------------------------------------------------------------------------
echo ""
echo "=== ARM V LARGE-N: Direct sidecar cosine (>=30 paraphrases, >=10 topics) ==="

SCRIPT_DIR="$(dirname "$0")"
CORPUS_JSON="$SCRIPT_DIR/arm_v_large_n.json"
WORK_DIR_LN=$(mktemp -d -t qa-large-n-XXXXXX)
COPY_VAULT_LN="$WORK_DIR_LN/qa-large-n-vault"
export WORK_DIR_LN COPY_VAULT_LN

echo "CORPUS_JSON:  $CORPUS_JSON"
echo "WORK_DIR_LN:  $WORK_DIR_LN"
echo "COPY_VAULT_LN: $COPY_VAULT_LN"
echo ""

[ -f "$CORPUS_JSON" ] || { echo "ERROR: corpus not found: $CORPUS_JSON"; exit 1; }

# Guard 1: JSON validity + >=30 entries / >=10 topics
echo "--- Guard 1: corpus JSON validation ---"
python3 - "$CORPUS_JSON" <<'PYEOF'
import json, sys
data = json.load(open(sys.argv[1]))
n = len(data)
topics = {e["topic"] for e in data}
t = len(topics)
if n < 30:
    print(f"FAIL Guard 1: need >=30 entries, got {n}")
    sys.exit(1)
if t < 10:
    print(f"FAIL Guard 1: need >=10 distinct topics, got {t}")
    sys.exit(1)
print(f"PASS Guard 1: {n} paraphrases across {t} topics")
PYEOF

# Capture live vault git status BEFORE large-n run
echo "--- Live vault BEFORE large-n ---"
LN_BEFORE_LINES=$(git -C "$LIVE_VAULT" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
echo "Live vault --porcelain line count: $LN_BEFORE_LINES"

# Safety: abort if copy vault already exists (guards against accidental re-run)
[ ! -d "$COPY_VAULT_LN" ] || { echo "ABORT: COPY_VAULT_LN already exists: $COPY_VAULT_LN"; exit 1; }

# Copy live vault (read-only source)
cp -r "$LIVE_VAULT" "$COPY_VAULT_LN"
[ -d "$COPY_VAULT_LN" ] || { echo "ERROR: COPY_VAULT_LN missing after copy"; exit 1; }
echo "Copied vault: $(ls "$COPY_VAULT_LN"/*.md 2>/dev/null | wc -l | tr -d ' ') .md files"

# ---------------------------------------------------------------------------
# Self-seed: write QA pairs for 10 real vault topics
# Topics derived from real vault notes 100,101,103,106,107,108,135,141,149,119.
# Slugs must match target_q_basename values in arm_v_large_n.json exactly.
# ---------------------------------------------------------------------------
echo ""
echo "--- Self-seeding 10 QA pairs (real vault topics) ---"

# Pair 1: cost-same-lever (topic from note 100)
cat > "$COPY_VAULT_LN/qa.2026-07-03.cost-same-lever.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.cost-same-lever.a
---
What is the relationship between making engram recall cheaper and enabling lighter prompts for more usage?

Answered by: [[qa.2026-07-03.cost-same-lever.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.cost-same-lever.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.cost-same-lever.q
certainty: high
contributors: [100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever]
vocab: [cost-optimization, engram-binary-ops, token-budget]
---
The two directions are the same lever: both resolve to reducing the post-fire procedure tax (~186s). Lighter prompts do not add usage (the description fires before the body loads, so shortening the body buys zero extra fires). The only live dollar lever is pruning the recall payload out of build context after Step 3. Dead/refuted directions: payload-size cap for dollars, whole-op or split haiku, cutting query phrases.

Answers: [[qa.2026-07-03.cost-same-lever.q]]
Contributors: [[100.2026-06-26.cost-and-usage-are-the-same-procedure-tax-lever]]
AEOF

# Pair 2: binary-install (topic from note 106)
cat > "$COPY_VAULT_LN/qa.2026-07-03.binary-install.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.binary-install.a
---
How should you install or rebuild the engram binary to test a code change with the real binary on PATH?

Answered by: [[qa.2026-07-03.binary-install.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.binary-install.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.binary-install.q
certainty: high
contributors: [106.2026-06-27.engram-binary-install-go-install-not-targ-build]
vocab: [cli-verification, engram-binary-ops, go-code-conventions]
---
Use `go install ./cmd/engram` to install the binary. The `use targ not go` rule covers test/lint/check — not binary install, which targ does not provide. Running targ build errors with 'unknown command: build'. A stale binary on PATH can make measurements appear unchanged after a code edit; always reinstall then re-measure.

Answers: [[qa.2026-07-03.binary-install.q]]
Contributors: [[106.2026-06-27.engram-binary-install-go-install-not-targ-build]]
AEOF

# Pair 3: lazy-retrieval-validation (topic from note 107)
cat > "$COPY_VAULT_LN/qa.2026-07-03.lazy-retrieval-validation.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.lazy-retrieval-validation.a
---
What measurements are required to validate that lazy on-demand retrieval is a net cost win over bulk payload delivery for an LLM agent?

Answered by: [[qa.2026-07-03.lazy-retrieval-validation.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.lazy-retrieval-validation.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.lazy-retrieval-validation.q
certainty: high
contributors: [107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior]
vocab: [token-budget, cost-optimization, retrieval-design]
---
Two measurements must pass: (1) SPARING — the agent's uninstructed fetch count stays well below the break-even N where N*per-fetch-cost equals bytes saved; (2) CAPABLE — a sole-source fixture proves the agent fetches the right deferred item unprompted. A deterministic payload size cut alone does not establish a net win — if the agent over-fetches or cannot select the right item, lazy is a net loss.

Answers: [[qa.2026-07-03.lazy-retrieval-validation.q]]
Contributors: [[107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior]]
AEOF

# Pair 4: crowded-vault-generalization (topic from note 103)
cat > "$COPY_VAULT_LN/qa.2026-07-03.crowded-vault-generalization.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.crowded-vault-generalization.a
---
Do engram's verified memory capability wins hold when the vault contains many distractor notes beyond toy five-note settings?

Answered by: [[qa.2026-07-03.crowded-vault-generalization.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.crowded-vault-generalization.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.crowded-vault-generalization.q
certainty: high
contributors: [103.2026-06-26.crowded-vault-wins-generalize-realistic-crowd]
vocab: [memory-system-architecture, retrieval-design, eval-methodology]
---
The four verified capability wins (C3, C4i, C5, C6) generalize to a realistic crowded vault with zero degradation under a 200-note real-vault crowd plus links. Target rank is flat from 0 to 400 real-vault distractors (Tier-1 free). Idiosyncratic notes are semantically distinctive so a realistic crowd ranks strictly below them — retrieval is robust.

Answers: [[qa.2026-07-03.crowded-vault-generalization.q]]
Contributors: [[103.2026-06-26.crowded-vault-wins-generalize-realistic-crowd]]
AEOF

# Pair 5: payload-prune-cut (topic from note 149)
cat > "$COPY_VAULT_LN/qa.2026-07-03.payload-prune-cut.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.payload-prune-cut.a
---
What is the measured cost saving from carrying only the Step-3 synthesis conclusion into a fresh build session instead of the full recall payload?

Answered by: [[qa.2026-07-03.payload-prune-cut.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.payload-prune-cut.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.payload-prune-cut.q
certainty: high
contributors: [149.2026-06-30.payload-prune-smoke-validated-40pct-build-cost-cut]
vocab: [cost-optimization, token-budget, engram-binary-ops]
---
Carrying only the Step-3 synthesis into a fresh build session (subagent-isolated recall, payload-prune) cuts build cost ~40% (~1.6 USD/app) with zero capability regression — opus n=3 smoke validated. The warm-over-cold per-round premium is recoverable. Production form: subagent-isolated recall where only the synthesis conclusion is injected into the build agent's context.

Answers: [[qa.2026-07-03.payload-prune-cut.q]]
Contributors: [[149.2026-06-30.payload-prune-smoke-validated-40pct-build-cost-cut]]
AEOF

# Pair 6: glance-recall-rung (topic from note 141)
cat > "$COPY_VAULT_LN/qa.2026-07-03.glance-recall-rung.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.glance-recall-rung.a
---
What is the cheap glance rung of the recall depth-dial and what capabilities does it deliver compared to full deep recall?

Answered by: [[qa.2026-07-03.glance-recall-rung.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.glance-recall-rung.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.glance-recall-rung.q
certainty: high
contributors: [141.2026-06-29.glance-rung-viable-at-spend-gate-and-smoke]
vocab: [cost-optimization, retrieval-design, agentic-recall-triggers]
---
The glance rung is a read-only recall tier (retrieve + recency-resolve + apply; no crystallization writes) that is validated end-to-end — glance delivers 3/4 idiosyncratic axes, fails C5, and its cost win is real on a real-scale vault. The glance rung relaxes the over-fire ceiling proportionally to its lower per-fire cost but does NOT dissolve the value gate.

Answers: [[qa.2026-07-03.glance-recall-rung.q]]
Contributors: [[141.2026-06-29.glance-rung-viable-at-spend-gate-and-smoke]]
AEOF

# Pair 7: eval-spend-estimate (topic from note 101)
cat > "$COPY_VAULT_LN/qa.2026-07-03.eval-spend-estimate.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.eval-spend-estimate.a
---
How should you estimate the dollar cost of an eval harness run before launching it?

Answered by: [[qa.2026-07-03.eval-spend-estimate.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.eval-spend-estimate.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.eval-spend-estimate.q
certainty: high
contributors: [101.2026-06-26.eval-spend-estimate-from-actual-per-cell-cost]
vocab: [cost-optimization, eval-methodology, lever-tracking]
---
Derive the estimate from the harness's actual per-cell spend: read a prior run's result JSON or RESULTS.md for a real per-cell or per-trial cost, then multiply by the fan-out (regimes, arms, trials). Guessing from intuition is off by ~50x — a warm cumulative cell is a FULL app build (~7 USD) not ~0.40 USD. Always state the realistic total before launching.

Answers: [[qa.2026-07-03.eval-spend-estimate.q]]
Contributors: [[101.2026-06-26.eval-spend-estimate-from-actual-per-cell-cost]]
AEOF

# Pair 8: model-routing-recall (topic from note 135)
cat > "$COPY_VAULT_LN/qa.2026-07-03.model-routing-recall.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.model-routing-recall.a
---
Can a cheaper model tier match opus performance on idiosyncratic reasoning tasks when engram memory is recalled?

Answered by: [[qa.2026-07-03.model-routing-recall.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.model-routing-recall.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.model-routing-recall.q
certainty: high
contributors: [135.2026-06-28.memory-democratizes-reasoning-cheap-model-matches-strong]
vocab: [model-routing, cost-optimization, memory-system-architecture]
---
Yes — sonnet with recalled memory fully matches opus across 3 idiosyncratic capability types (C3 apply-conventions 15/15, C4i recency-supersession 3/3, C6 emergent-abduction 6/6). Sonnet WITHOUT memory fails (C4i/C6 cold 0/N). Engram recall democratizes reasoning: the same idiosyncratic content that requires opus cold can be served by sonnet warm.

Answers: [[qa.2026-07-03.model-routing-recall.q]]
Contributors: [[135.2026-06-28.memory-democratizes-reasoning-cheap-model-matches-strong]]
AEOF

# Pair 9: async-not-cost-reduction (topic from note 108)
cat > "$COPY_VAULT_LN/qa.2026-07-03.async-not-cost-reduction.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.async-not-cost-reduction.a
---
Does making engram learn run asynchronously or non-blocking reduce its actual token or dollar cost?

Answered by: [[qa.2026-07-03.async-not-cost-reduction.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.async-not-cost-reduction.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.async-not-cost-reduction.q
certainty: high
contributors: [108.2026-06-27.relocation-off-critical-path-is-not-a-cost-reduction]
vocab: [cost-optimization, lever-tracking, token-budget]
---
No. Async/background execution relocates cost off the perceived critical path but spends the same tokens, the same dollars, and the same total wall-time. It is perceived-latency UX, not a cost reduction. For a tokens/cost/time goal, count only levers that reduce actual resource use on a named axis. Keep async ideas out of scope on a cost-reduction roadmap.

Answers: [[qa.2026-07-03.async-not-cost-reduction.q]]
Contributors: [[108.2026-06-27.relocation-off-critical-path-is-not-a-cost-reduction]]
AEOF

# Pair 10: retrieval-note-floor (topic from note 119)
cat > "$COPY_VAULT_LN/qa.2026-07-03.retrieval-note-floor.q.md" <<'QEOF'
---
type: qa-question
date: "2026-07-03"
answered_by: qa.2026-07-03.retrieval-note-floor.a
---
Why should crystallized vault notes receive a ranking floor above raw transcript chunks in engram's cosine retrieval?

Answered by: [[qa.2026-07-03.retrieval-note-floor.a]]
QEOF

cat > "$COPY_VAULT_LN/qa.2026-07-03.retrieval-note-floor.a.md" <<'AEOF'
---
type: qa-answer
date: "2026-07-03"
answers: qa.2026-07-03.retrieval-note-floor.q
certainty: high
contributors: [119.2026-06-28.evaluate-ranking-fix-by-knowledge-delivery-not-item-rank]
vocab: [retrieval-design, memory-system-architecture, eval-methodology]
---
Crystallized notes compress multi-turn reasoning into a single authoritative statement; raw chunks are noisy fragments from transcripts. A note floor ensures that when a relevant crystallized note exists, it outranks chunks covering the same topic. The matched-note floor (capWithNoteFloor) was shipped 2026-06-28 and improved real-path note recall@5 from 0.22 to 0.83 — up to the embedder ceiling.

Answers: [[qa.2026-07-03.retrieval-note-floor.q]]
Contributors: [[119.2026-06-28.evaluate-ranking-fix-by-knowledge-delivery-not-item-rank]]
AEOF

echo "Wrote 20 QA files (10 pairs) to $COPY_VAULT_LN"

# Embed all files in the copy vault (QA pairs + existing notes)
echo ""
echo "--- Embedding copy vault (local model, \$0) ---"
ENGRAM_VAULT_PATH="$COPY_VAULT_LN" engram embed apply 2>&1 | tee "$WORK_DIR_LN/embed_ln.log"
echo "Embedding done."

# Guard 2: every corpus target exists in seeded copy vault
echo ""
echo "--- Guard 2: corpus targets present in seeded vault ---"
python3 - "$CORPUS_JSON" "$COPY_VAULT_LN" <<'PYEOF'
import json, glob, os, sys
corpus_json, vault = sys.argv[1:]
data = json.load(open(corpus_json))
have = {os.path.basename(f)[:-3] for f in glob.glob(os.path.join(vault, 'qa.*.md'))}
missing = sorted({e['target_q_basename'] for e in data} - have)
if missing:
    print(f"FAIL Guard 2 — HARNESS BUG: corpus targets absent from seeded vault: {missing}")
    sys.exit(1)
targets = {e['target_q_basename'] for e in data}
print(f"PASS Guard 2: all {len(targets)} corpus target_q_basenames exist in seeded vault")
PYEOF

# Write paraphrase probe notes (one per corpus entry) — named ln-para-{id}.md
echo ""
echo "--- Writing paraphrase probe notes ---"
python3 - "$CORPUS_JSON" "$COPY_VAULT_LN" <<'PYEOF'
import json, os, sys
corpus_json, vault = sys.argv[1:]
data = json.load(open(corpus_json))
for entry in data:
    probe_path = os.path.join(vault, f"ln-para-{entry['id']}.md")
    content = (
        f"---\ntype: fact\ndate: \"2026-07-03\"\n---\n\n"
        f"{entry['paraphrase']}\n"
    )
    with open(probe_path, 'w') as fh:
        fh.write(content)
print(f"Wrote {len(data)} paraphrase probe notes (ln-para-*.md)")
PYEOF

# Embed paraphrase probe notes (only missing sidecars)
echo "--- Embedding paraphrase probe notes ---"
ENGRAM_VAULT_PATH="$COPY_VAULT_LN" engram embed apply 2>&1 | tail -5
echo "Para note embedding done."

# ---------------------------------------------------------------------------
# Direct sidecar cosine scorer
# Loads body_vector from every .vec.json; computes cosines.
# Q notes: qa.*.q.vec.json
# A notes: qa.*.a.vec.json (excluded from both comparison sets)
# Para notes: ln-para-*.vec.json (the probe embeddings)
# Content notes: everything else
# PASS per paraphrase: target Q body_vector cosine is #1 among Q notes
#                      AND above every content note body_vector cosine
# ---------------------------------------------------------------------------
ARMV_LN_JSON="$WORK_DIR_LN/arm_v_large_n_results.json"
ARMV_LN_OUT_JSON="$RESULTS_DIR/arm_v_large_n_results.json"

python3 - "$CORPUS_JSON" "$COPY_VAULT_LN" "$ARMV_LN_JSON" <<'PYEOF'
import json, glob, os, re, sys, math

corpus_json, vault, out_path = sys.argv[1:]
data = json.load(open(corpus_json))

QA_Q_PAT = re.compile(r'^qa\..*\.q$')
QA_A_PAT = re.compile(r'^qa\..*\.a$')
LN_PARA_PAT = re.compile(r'^ln-para-')

def load_body_vec(path):
    try:
        with open(path) as fh:
            d = json.load(fh)
        return d.get('body_vector', [])
    except Exception as e:
        return []

def cosine(a, b):
    if not a or not b or len(a) != len(b):
        return 0.0
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na == 0.0 or nb == 0.0:
        return 0.0
    return dot / (na * nb)

# Load all .vec.json sidecars
sidecars = {}
for vec_path in glob.glob(os.path.join(vault, '*.vec.json')):
    key = os.path.basename(vec_path)[:-len('.vec.json')]
    sidecars[key] = load_body_vec(vec_path)

# Partition by type
q_notes = {k: v for k, v in sidecars.items() if QA_Q_PAT.match(k)}
a_notes = {k: v for k, v in sidecars.items() if QA_A_PAT.match(k)}
para_notes = {k: v for k, v in sidecars.items() if LN_PARA_PAT.match(k)}
content_notes = {k: v for k, v in sidecars.items()
                 if not QA_Q_PAT.match(k) and not QA_A_PAT.match(k) and not LN_PARA_PAT.match(k)}

print(f"Vault sidecars: {len(q_notes)} Q-notes, {len(a_notes)} A-notes, "
      f"{len(para_notes)} para-notes, {len(content_notes)} content-notes")
print()

# Score each paraphrase
results_by_topic = {}
all_rows = []
pass_count = 0
total = 0

print(f"{'ID':>8}  {'topic':>30}  {'q_rank':>6}  {'q_score':>8}  {'top_c':>8}  pass?")
print("-" * 85)

for entry in data:
    entry_id = entry['id']
    topic = entry['topic']
    target_q = entry['target_q_basename']
    para_key = f"ln-para-{entry_id}"

    if para_key not in para_notes:
        print(f"  WARN: para sidecar missing: {para_key}")
        continue
    if target_q not in q_notes:
        print(f"  WARN: target Q sidecar missing: {target_q}")
        continue

    para_vec = para_notes[para_key]

    # Cosine of paraphrase vs all Q notes
    q_scores = {k: cosine(para_vec, v) for k, v in q_notes.items()}
    q_sorted = sorted(q_scores.items(), key=lambda x: x[1], reverse=True)
    target_score = q_scores.get(target_q, 0.0)
    q_rank = next((i + 1 for i, (k, _) in enumerate(q_sorted) if k == target_q), None)

    # Best content note cosine
    top_content_score = max((cosine(para_vec, v) for v in content_notes.values()), default=0.0)

    # PASS: target is #1 among Q notes AND above every content note
    passes = (q_rank == 1 and target_score > top_content_score)
    if passes:
        pass_count += 1
    total += 1

    status = "PASS" if passes else "FAIL"
    q_rank_str = str(q_rank) if q_rank is not None else "N/A"
    print(f"{entry_id:>8}  {topic:>30}  {q_rank_str:>6}  {target_score:>8.4f}  {top_content_score:>8.4f}  {status}")

    if topic not in results_by_topic:
        results_by_topic[topic] = {'pass': 0, 'total': 0, 'rows': []}
    results_by_topic[topic]['pass'] += int(passes)
    results_by_topic[topic]['total'] += 1
    results_by_topic[topic]['rows'].append({
        'id': entry_id,
        'q_rank': q_rank,
        'target_score': round(target_score, 4),
        'top_content_score': round(top_content_score, 4),
        'passes': passes,
    })
    all_rows.append({
        'id': entry_id,
        'topic': topic,
        'target_q_basename': target_q,
        'paraphrase': entry['paraphrase'],
        'q_rank': q_rank,
        'target_score': round(target_score, 4),
        'top_content_score': round(top_content_score, 4),
        'passes': passes,
    })

print()
rate = pass_count / total if total > 0 else 0.0
print(f"Arm V large-n: {pass_count}/{total} paraphrases PASS ({rate:.0%})")
print()

if rate >= 0.80:
    verdict = "PASS (>=80%)"
elif rate >= 0.60:
    verdict = "BORDERLINE (60-79%)"
else:
    verdict = "FAIL (<60%)"
print(f"Arm V large-n pre-registered verdict: {verdict}")

print()
print("Per-topic breakdown:")
print(f"  {'topic':>35}  {'pass/total':>10}")
print("  " + "-" * 48)
for t_name in sorted(results_by_topic.keys()):
    tc = results_by_topic[t_name]
    print(f"  {t_name:>35}  {tc['pass']}/{tc['total']:>9}")

out = {
    'pass_count': pass_count,
    'total': total,
    'rate': round(rate, 4),
    'verdict': verdict,
    'results_by_topic': {k: {'pass': v['pass'], 'total': v['total']}
                         for k, v in results_by_topic.items()},
    'rows': all_rows,
}
json.dump(out, open(out_path, 'w'), indent=2)
print(f"\nResults written to {out_path}")
PYEOF

# Copy results to RESULTS_DIR
cp "$ARMV_LN_JSON" "$ARMV_LN_OUT_JSON" 2>/dev/null || true

# Capture live vault git status AFTER large-n run
echo ""
echo "--- Live vault AFTER large-n ---"
LN_AFTER_LINES=$(git -C "$LIVE_VAULT" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
echo "Live vault --porcelain line count: $LN_AFTER_LINES"
if [ "$LN_BEFORE_LINES" = "$LN_AFTER_LINES" ]; then
    echo "PASS contamination check: live vault unchanged (line count: $LN_BEFORE_LINES)"
else
    echo "WARN: live vault line count changed ($LN_BEFORE_LINES -> $LN_AFTER_LINES)"
fi

echo ""
echo "=== ARM V LARGE-N complete ==="
echo "WORK_DIR_LN preserved at: $WORK_DIR_LN (remove manually when done)"
