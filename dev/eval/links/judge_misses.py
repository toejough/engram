"""P1 miss-population judge for link-value exploration.

For each of 50 replay records (25 queries × n=3,10) run a sonnet judge that
sweeps ALL 136 vault notes and reports which ones whose lesson genuinely bears
on the query task are ABSENT from the delivered top-10 notes.

Outputs:
  dev/eval/links/misses_p1.json  — [{query_id, n, missed_note, why_needed}]

Prints a labeled summary table at the end.

Usage:
  cd /Users/joe/repos/personal/engram
  python3 dev/eval/links/judge_misses.py [--workers 6]
"""
import argparse
import concurrent.futures as cf
import json
import os
import re
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
VAULT = os.path.expanduser("~/.local/share/engram/vault")
REPLAYS_PATH = os.path.join(HERE, "replays.json")
QUERIES_PATH = os.path.join(HERE, "queries.json")
OUT_PATH = os.path.join(HERE, "misses_p1.json")

MODEL = "claude-sonnet-4-6"
SITUATION_TRUNC = 140
BASENAME_TRUNC = 140


# ---------------------------------------------------------------------------
# Vault catalog
# ---------------------------------------------------------------------------

def build_catalog() -> list[dict]:
    """Read all *.md notes from vault; extract basename + situation line."""
    notes = sorted(f for f in os.listdir(VAULT) if f.endswith(".md"))
    catalog = []
    for name in notes:
        path = os.path.join(VAULT, name)
        situation = ""
        try:
            with open(path, encoding="utf-8") as fh:
                for line in fh:
                    if line.startswith("situation:"):
                        situation = line[len("situation:"):].strip().strip('"')
                        break
        except OSError:
            situation = "(unreadable)"
        if len(situation) > SITUATION_TRUNC:
            situation = situation[:SITUATION_TRUNC] + "…"
        catalog.append({"basename": name, "situation": situation})
    return catalog


def format_catalog(catalog: list[dict]) -> str:
    """Compact text rendering of the catalog for inclusion in the prompt."""
    lines = ["VAULT CATALOG (all notes):"]
    for entry in catalog:
        bn = entry["basename"]
        if len(bn) > BASENAME_TRUNC:
            bn = bn[:BASENAME_TRUNC] + "…"
        lines.append(f"  {bn}  |  {entry['situation']}")
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Claude subprocess (mirrors run.py pattern)
# ---------------------------------------------------------------------------

def _run_claude(prompt: str) -> dict:
    """Run claude -p and return parsed JSON output.

    Returns the raw JSON dict (keys: result, total_cost_usd, is_error, ...).
    Raises RuntimeError on non-retryable failure.
    """
    args = [
        "claude", "-p", prompt,
        "--output-format", "json",
        "--model", MODEL,
        "--permission-mode", "bypassPermissions",
    ]
    env = dict(os.environ)

    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        result = subprocess.run(args, capture_output=True, text=True, env=env)
        try:
            out = json.loads(result.stdout)
        except Exception:
            out = {}
        cost = out.get("total_cost_usd", 0) or 0
        is_err = out.get("is_error") or (not out)
        # transient cheap error → retry
        if is_err and cost < 0.02:
            continue
        return out
    # All retries exhausted — return last out (may be empty)
    return out  # noqa: F821


# ---------------------------------------------------------------------------
# Per-replay judge call
# ---------------------------------------------------------------------------

JUDGE_PROMPT_TMPL = """\
You are a strict relevance judge for an agent-memory recall system.

TASK DESCRIPTION (phrases an agent used to retrieve memories):
{phrases}

DELIVERED NOTES (top-10 notes already returned by the retrieval system for this query):
{delivered}

{catalog}

INSTRUCTION:
List every catalog note whose lesson genuinely BEARS ON the task these query \
phrases describe — i.e. an agent doing that task would act differently for \
having read it — and that is ABSENT from the delivered list above.

Be strict: topical adjacency is NOT enough. The note must change what the \
agent would do. Ignore chunk/doc files — only evaluate notes that appear in \
the VAULT CATALOG above.

Return STRICT JSON (no markdown fences, no other text):
[{{"missed_note": "<exact basename>", "why_needed": "<one line>"}}]

Return an empty JSON array [] if no notes are missing.
"""


def judge_replay(
    replay: dict,
    queries_by_id: dict,
    catalog_text: str,
    catalog_basenames: set[str],
) -> dict:
    """Run the judge for one replay record.

    Returns a dict with keys:
      query_id, n, cost, misses: [{missed_note, why_needed}], error
    """
    query_id = replay["query_id"]
    n_val = replay["n"]
    phrases = replay.get("phrases_used", [])
    ranked_notes = replay.get("ranked_notes", [])

    # Delivered = top-10 notes by rank (ranked_notes is already note-only, sorted by rank)
    delivered_notes = ranked_notes[:10]
    delivered_basenames = {item["basename"] for item in delivered_notes}

    phrases_text = "\n".join(f"  - {p}" for p in phrases)
    delivered_text = "\n".join(
        f"  - {item['basename']}" for item in delivered_notes
    ) or "  (none)"

    prompt = JUDGE_PROMPT_TMPL.format(
        phrases=phrases_text,
        delivered=delivered_text,
        catalog=catalog_text,
    )

    raw_out = _run_claude(prompt)
    cost = raw_out.get("total_cost_usd", 0) or 0
    is_err = raw_out.get("is_error") or (not raw_out)

    if is_err:
        return {
            "query_id": query_id,
            "n": n_val,
            "cost": cost,
            "misses": [],
            "error": "claude call failed",
        }

    result_text = raw_out.get("result", "") or ""

    # Parse JSON; retry once on failure
    misses = _parse_misses(result_text)
    if misses is None:
        # Retry once
        raw_out2 = _run_claude(prompt)
        cost += raw_out2.get("total_cost_usd", 0) or 0
        result_text2 = raw_out2.get("result", "") or ""
        misses = _parse_misses(result_text2)
        if misses is None:
            return {
                "query_id": query_id,
                "n": n_val,
                "cost": cost,
                "misses": [],
                "error": f"unparseable output: {result_text[:200]!r}",
            }

    # Filter: only catalog notes that are absent from the delivered set
    filtered_misses = []
    for item in misses:
        bn = item.get("missed_note", "")
        if not bn:
            continue
        if bn not in catalog_basenames:
            # Judge hallucinated a non-existent note; skip
            continue
        if bn in delivered_basenames:
            # Judge listed a delivered note as a miss; skip
            continue
        filtered_misses.append({
            "missed_note": bn,
            "why_needed": item.get("why_needed", ""),
        })

    return {
        "query_id": query_id,
        "n": n_val,
        "cost": cost,
        "misses": filtered_misses,
        "error": None,
    }


def _parse_misses(text: str) -> list[dict] | None:
    """Try to parse the judge's output as a JSON array.

    Returns the list on success, None on failure.
    Uses a bracket-depth counter to find the matching ] for the opening [,
    so trailing model reasoning (which may contain ] chars) does not confuse
    the extraction.
    """
    if not text:
        return None
    # Strip markdown fences if present
    text = text.strip()
    text = re.sub(r"^```(?:json)?\s*", "", text, flags=re.IGNORECASE)
    text = re.sub(r"\s*```$", "", text)
    text = text.strip()

    # Find start of the JSON array
    start = text.find("[")
    if start == -1:
        return None

    # Walk forward counting brackets (respecting JSON strings) to find the
    # matching ] — this handles trailing text that contains ] characters.
    depth = 0
    in_string = False
    escape_next = False
    end = -1
    for idx in range(start, len(text)):
        ch = text[idx]
        if escape_next:
            escape_next = False
            continue
        if ch == "\\" and in_string:
            escape_next = True
            continue
        if ch == '"':
            in_string = not in_string
            continue
        if in_string:
            continue
        if ch == "[":
            depth += 1
        elif ch == "]":
            depth -= 1
            if depth == 0:
                end = idx
                break

    if end == -1:
        return None

    candidate = text[start : end + 1]
    try:
        parsed = json.loads(candidate)
    except json.JSONDecodeError:
        return None

    if not isinstance(parsed, list):
        return None
    return parsed


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description="P1 miss-population judge")
    ap.add_argument("--workers", type=int, default=6, help="parallel workers")
    args = ap.parse_args()

    # Load inputs
    if not os.path.exists(REPLAYS_PATH):
        sys.exit(f"ERROR: {REPLAYS_PATH} not found — run replay.py first")
    if not os.path.exists(QUERIES_PATH):
        sys.exit(f"ERROR: {QUERIES_PATH} not found")

    with open(REPLAYS_PATH) as fh:
        replays = json.load(fh)
    with open(QUERIES_PATH) as fh:
        queries = json.load(fh)

    queries_by_id = {q["id"]: q for q in queries}

    # Build vault catalog
    print(f"Building vault catalog from {VAULT} …", flush=True)
    catalog = build_catalog()
    print(f"  {len(catalog)} notes", flush=True)
    catalog_text = format_catalog(catalog)
    catalog_basenames = {entry["basename"] for entry in catalog}

    print(
        f"Running judge on {len(replays)} replays (workers={args.workers}) …",
        flush=True,
    )

    results: list[dict] = []

    def do_one(replay: dict) -> dict:
        result = judge_replay(replay, queries_by_id, catalog_text, catalog_basenames)
        n_misses = len(result["misses"])
        err_flag = " ERROR" if result["error"] else ""
        print(
            f"  [{result['query_id']} n={result['n']}] "
            f"misses={n_misses} cost=${result['cost']:.4f}{err_flag}",
            flush=True,
        )
        return result

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {ex.submit(do_one, r): r for r in replays}
        for fut in cf.as_completed(futs):
            results.append(fut.result())

    # Sort deterministically
    results.sort(key=lambda r: (r["query_id"], r["n"]))

    # Flatten misses to output format
    flat_misses: list[dict] = []
    for result in results:
        if not result["error"]:
            for miss in result["misses"]:
                flat_misses.append({
                    "query_id": result["query_id"],
                    "n": result["n"],
                    "missed_note": miss["missed_note"],
                    "why_needed": miss["why_needed"],
                })

    with open(OUT_PATH, "w") as fh:
        json.dump(flat_misses, fh, indent=2)
    print(f"\nWrote {len(flat_misses)} miss entries → {OUT_PATH}", flush=True)

    # -----------------------------------------------------------------------
    # Summary table
    # -----------------------------------------------------------------------
    total_calls = len(replays)
    total_cost = sum(r["cost"] for r in results)
    error_count = sum(1 for r in results if r["error"])

    pairs_with_miss = sum(
        1 for r in results if r["misses"] and not r["error"]
    )

    all_missed_notes = set()
    misses_at_n3 = 0
    misses_at_n10 = 0
    miss_note_counts: dict[str, int] = {}
    for result in results:
        for miss in result["misses"]:
            bn = miss["missed_note"]
            all_missed_notes.add(bn)
            if result["n"] == 3:
                misses_at_n3 += 1
            else:
                misses_at_n10 += 1
            miss_note_counts[bn] = miss_note_counts.get(bn, 0) + 1

    top5 = sorted(miss_note_counts.items(), key=lambda x: -x[1])[:5]

    print("\n" + "=" * 70)
    print("P1 MISS-POPULATION JUDGE — SUMMARY (2026-07-02)")
    print("=" * 70)
    print(f"{'Metric':<40} {'Value':>10}")
    print("-" * 52)
    print(f"{'Judge calls (total)':<40} {total_calls:>10}")
    print(f"{'$ spent (sum total_cost_usd)':<40} {'${:.4f}'.format(total_cost):>10}")
    print(f"{'Errors (call or parse)':<40} {error_count:>10}")
    print(f"{'(query,n) pairs with ≥1 miss':<40} {pairs_with_miss:>10}")
    print(f"{'Distinct missed notes':<40} {len(all_missed_notes):>10}")
    print(f"{'Total miss entries at n=3':<40} {misses_at_n3:>10}")
    print(f"{'Total miss entries at n=10':<40} {misses_at_n10:>10}")
    print()
    print("Top-5 most-missed notes (basename | count):")
    for bn, count in top5:
        print(f"  {count:3d}x  {bn}")

    if error_count:
        print("\nErrors:")
        for r in results:
            if r["error"]:
                print(f"  [{r['query_id']} n={r['n']}] {r['error']}")

    print("=" * 70)


if __name__ == "__main__":
    main()
