"""Free, deterministic cost curve for the content-budget cap (Option 1, Task 2).

Runs `engram query` on ONE fixed 10-phrase set at a schedule of ENGRAM_CONTENT_BUDGET values and
records, per budget: payload wire bytes, a token PROXY (total content chars / 4), and the budget
report's items_with_full_content + chunks_snippeted. Zero LLM calls — characterizes the whole cost
axis for free so opus is spent only on quality verification (Task 3).

  python3 cap_cost_curve.py
"""
import json
import os
import re
import subprocess

# The exact 10-phrase set from this session's Option 1 recall — constant across all budgets.
PHRASES = [
    "implementing a content budget cap in engram query payload",
    "minimum spend for the recall benefit sweep cap",
    "iteratively reduce a parameter until tests fail find the knee",
    "query payload full content members recency channel truncation",
    "binary search a threshold against expensive eval harness",
    "engram query internal cli Go renderItems content emission",
    "C3 C4 C5 C6 warm eval arms quality signal harness",
    "check metric sensitivity before crowning cheaper arm",
    "noise floor underpowered eval gap below noise",
    "engram cost reduction option 1 cap payload content",
]
BUDGETS = [0, 60, 30, 15, 8, 4, 2]
CHARS_PER_TOKEN = 4  # stated proxy, not a real tokenizer


def run(budget):
    args = ["engram", "query"]
    for p in PHRASES:
        args += ["--phrase", p]
    env = dict(os.environ)
    env["ENGRAM_CONTENT_BUDGET"] = str(budget)
    out = subprocess.run(args, capture_output=True, text=True, env=env).stdout
    nbytes = len(out.encode())
    full = _budget_field(out, "items_with_full_content")
    snipped = _budget_field(out, "chunks_snippeted")
    # The agent reads the WHOLE payload, so wire bytes / 4 is the honest token proxy.
    return {"budget": budget, "bytes": nbytes, "tok_proxy": nbytes // CHARS_PER_TOKEN,
            "items_full": full, "snipped": snipped}


def _budget_field(payload, key):
    m = re.search(rf"^\s*{key}:\s*(\d+)\s*$", payload, re.MULTILINE)
    return int(m.group(1)) if m else None


def main():
    rows = [run(b) for b in BUDGETS]
    base = rows[0]["tok_proxy"] or 1
    print(f"{'budget':>7} {'bytes':>9} {'tok_proxy':>10} {'Δtok%':>7} {'items_full':>11} {'snipped':>8}")
    for r in rows:
        dpct = 100 * (r["tok_proxy"] - base) / base
        print(f"{r['budget']:>7} {r['bytes']:>9} {r['tok_proxy']:>10} {dpct:>6.0f}% "
              f"{str(r['items_full']):>11} {str(r['snipped']):>8}")
    json.dump(rows, open("/tmp/cap_cost_curve.json", "w"), indent=1)


if __name__ == "__main__":
    main()
