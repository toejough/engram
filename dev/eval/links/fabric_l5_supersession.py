"""Fabric L5: typed temporal supersession edges.

Candidate pairs: date-ordered (older, newer) note pairs meeting either:
  - body OR situation cosine ≥ 0.6 between the two notes, OR
  - one note's body text mentions the other note's basename/slug

Bound to ~150 candidate pairs by cosine rank (slug-mention pairs always included).

LLM gate (sonnet, batched): does NEWER update / narrow / refute a SPECIFIC CLAIM that
OLDER makes? Strict test: same claim-subject with a changed prescription/verdict — NOT
mere topical succession. Default DROP.

Writes:
  dev/eval/links/fabrics/l5.json  [{old, new, type, claim}]
  where type ∈ {updates, narrows, refutes}

Cost tracked and printed at completion.
Fails loud on unparseable LLM output (retries once, then error entry — no silent drops).
"""
import collections
import glob
import json
import math
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from collections import defaultdict

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
HERE = os.path.dirname(os.path.abspath(__file__))
OUT_PATH = os.path.join(HERE, "fabrics", "l5.json")

MODELS = {"sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
MODEL = "sonnet"

COSINE_THRESHOLD = 0.6    # for candidate selection
MAX_CANDIDATES = 150       # upper bound on pairs from cosine rank
BATCH_SIZE = 8             # pairs per LLM call

_FM_RE = re.compile(r"^---\n(.*?)\n---\s*\n?", re.DOTALL)
_KV_RE = re.compile(r"^([a-z_]+):\s*(.*)", re.IGNORECASE)
_DATE_RE = re.compile(r"\d{4}-\d{2}-\d{2}")


def parse_frontmatter(content: str) -> dict[str, str]:
    """Extract key-value pairs from YAML frontmatter (simple, not full YAML)."""
    fm_match = _FM_RE.match(content)
    if not fm_match:
        return {}
    result: dict[str, str] = {}
    current_key: str | None = None
    current_val_lines: list[str] = []

    for line in fm_match.group(1).splitlines():
        kv = _KV_RE.match(line)
        if kv and not line.startswith(" ") and not line.startswith("\t"):
            if current_key:
                result[current_key] = " ".join(current_val_lines).strip().strip("'\"")
            current_key = kv.group(1).lower()
            current_val_lines = [kv.group(2).strip().strip("'\"")]
        elif current_key and (line.startswith(" ") or line.startswith("\t")):
            current_val_lines.append(line.strip().strip("'\""))
        else:
            if current_key:
                result[current_key] = " ".join(current_val_lines).strip().strip("'\"")
                current_key = None
                current_val_lines = []

    if current_key:
        result[current_key] = " ".join(current_val_lines).strip().strip("'\"")

    return result


def note_date(basename: str) -> str:
    """Extract date string from note basename for ordering. Returns '' if none."""
    match = _DATE_RE.search(basename)
    return match.group() if match else ""


def note_snippet(basename: str, content: str, max_len: int = 300) -> str:
    """Compact snippet for LLM supersession evaluation."""
    fm = parse_frontmatter(content)
    situation = fm.get("situation", "(no situation)")
    note_type = fm.get("type", "")

    if note_type == "fact":
        subject = fm.get("subject", "")
        predicate = fm.get("predicate", "")
        obj = fm.get("object", "")[:max_len]
        claim_text = f"{subject} {predicate}: {obj}"
    elif note_type == "feedback":
        behavior = fm.get("behavior", "")[:150]
        impact = fm.get("impact", "")[:150]
        claim_text = f"behavior: {behavior}; impact: {impact}"
    else:
        body = _FM_RE.sub("", content).strip()
        claim_text = body[:max_len]

    date = note_date(basename)
    return (
        f"note: {basename}\n"
        f"date: {date}\n"
        f"situation: {situation}\n"
        f"claim: {claim_text[:max_len]}"
    )


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """Cosine similarity between two float vectors."""
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (norm_a * norm_b)


def load_vectors(vault_path: str) -> dict[str, dict[str, list[float]]]:
    """Load body_vector and situation_vector from each .vec.json sidecar."""
    result: dict[str, dict[str, list[float]]] = {}
    for fpath in sorted(glob.glob(os.path.join(vault_path, "*.vec.json"))):
        basename = os.path.basename(fpath)[: -len(".vec.json")]
        with open(fpath) as fh:
            data = json.load(fh)
        body = data.get("body_vector")
        sit = data.get("situation_vector")
        if body and sit:
            result[basename] = {"body": body, "situation": sit}
        else:
            print(f"  WARNING: missing vectors in {os.path.basename(fpath)}", flush=True)
    return result


def _claude(prompt: str, model: str = MODEL) -> dict:
    """One `claude -p` call with transient-failure retry."""
    wd = tempfile.mkdtemp(prefix="fabric-l5-")
    try:
        env = dict(os.environ)
        args = [
            "claude", "-p", prompt,
            "--output-format", "json",
            "--model", MODELS[model],
            "--permission-mode", "bypassPermissions",
        ]
        out: dict = {}
        for backoff in (0, 15, 45, 120):
            if backoff:
                print(f"    [retry after {backoff}s]", flush=True)
                time.sleep(backoff)
            r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
            try:
                out = json.loads(r.stdout)
            except Exception:
                out = {}
            is_err = out.get("is_error") or not out
            cost = out.get("total_cost_usd", 0) or 0
            if is_err and cost < 0.01:
                continue
            break
        return out
    finally:
        shutil.rmtree(wd, ignore_errors=True)


_SUPERSESSION_SYSTEM = """\
You are evaluating whether a NEWER memory note updates, narrows, or refutes a specific claim made by an OLDER memory note.

TYPES (strict definitions):
- updates: NEWER makes the same claim OLDER makes but with a new/changed prescription or finding —
  the subject is the same, the verdict changed (e.g., OLDER: "use X"; NEWER: "use Y instead of X").
- narrows: NEWER scopes an OLDER claim to a subset — same claim but NEWER says it only applies
  under conditions C, whereas OLDER applied broadly.
- refutes: NEWER contradicts OLDER's verdict on the same subject — one says do X, the other says don't.

STRICT REQUIREMENT: the SAME CLAIM-SUBJECT must appear in both notes with a CHANGED prescription.
Topical overlap alone is NOT enough — notes that discuss the same topic without one changing
the other's specific verdict are DROP.

Default: DROP. Only PERSIST when there is a clear, specific claim-subject in OLDER that NEWER
explicitly changes.

Respond with a JSON array — one object per pair IN ORDER:
[
  {"old": "...", "new": "...", "verdict": "PERSIST", "type": "updates"|"narrows"|"refutes",
   "claim": "<one-line claim-subject that changed>"},
  {"old": "...", "new": "...", "verdict": "DROP", "type": null, "claim": null},
  ...
]"""


def build_batch_prompt(pairs: list[tuple[str, str]], snippets: dict[str, str]) -> str:
    """Build LLM prompt for a batch of (old, new) note pairs."""
    lines = [_SUPERSESSION_SYSTEM, "", "=== PAIRS TO EVALUATE ===", ""]

    for old, new in pairs:
        lines.append(f"--- PAIR: {old} → {new} ---")
        lines.append("OLDER note:")
        for sline in snippets.get(old, f"note: {old}\n(no snippet)").splitlines():
            lines.append(f"  {sline}")
        lines.append("NEWER note:")
        for sline in snippets.get(new, f"note: {new}\n(no snippet)").splitlines():
            lines.append(f"  {sline}")
        lines.append(f'Pair: {{"old": "{old}", "new": "{new}"}}')
        lines.append("")

    return "\n".join(lines)


def parse_llm_decisions(text: str) -> list[dict]:
    """Parse JSON array from LLM output. Raises ValueError on failure."""
    text = text.strip()
    if text.startswith("```"):
        text = re.sub(r"^```[a-z]*\n?", "", text, flags=re.MULTILINE)
        text = text.rstrip("`").strip()

    start = text.find("[")
    end = text.rfind("]")
    if start == -1 or end == -1:
        raise ValueError(f"No JSON array in LLM output: {text[:200]!r}")

    return json.loads(text[start: end + 1])


def main() -> None:
    print(f"Loading vault notes from {VAULT_PATH}", flush=True)
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")
    print(f"Found {len(md_files)} notes", flush=True)

    # Load vectors
    print("Loading vectors …", flush=True)
    vectors = load_vectors(VAULT_PATH)
    notes = sorted(vectors.keys())
    print(f"Loaded {len(notes)} notes with vectors", flush=True)

    # Load note content + build snippets
    print("Building note snippets …", flush=True)
    snippets: dict[str, str] = {}
    note_body_text: dict[str, str] = {}
    for fpath in md_files:
        basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()
        snippets[basename] = note_snippet(basename, content)
        note_body_text[basename] = content  # for slug-mention detection

    # Sort notes by date for temporal ordering
    notes_by_date = sorted(notes, key=lambda b: (note_date(b), b))
    print(f"Date range: {note_date(notes_by_date[0])} → {note_date(notes_by_date[-1])}", flush=True)

    # Build candidate pairs: (older, newer) where cosine(body or situation) ≥ 0.6
    print(f"Computing candidate pairs (cosine ≥ {COSINE_THRESHOLD} or slug mention) …", flush=True)

    # Track candidates with their max cosine score for ranking
    cosine_candidates: dict[tuple[str, str], float] = {}
    slug_mention_pairs: set[tuple[str, str]] = set()

    n = len(notes_by_date)
    for i in range(n):
        for j in range(i + 1, n):
            older = notes_by_date[i]
            newer = notes_by_date[j]
            v_old = vectors[older]
            v_new = vectors[newer]

            body_cos = cosine_similarity(v_old["body"], v_new["body"])
            sit_cos = cosine_similarity(v_old["situation"], v_new["situation"])
            max_cos = max(body_cos, sit_cos)

            if max_cos >= COSINE_THRESHOLD:
                cosine_candidates[(older, newer)] = max_cos

    # Slug mention pairs: older's basename appears in newer's text or vice versa
    for i in range(n):
        for j in range(i + 1, n):
            older = notes_by_date[i]
            newer = notes_by_date[j]
            old_slug = older.split(".", 2)[-1] if "." in older else older
            new_slug = newer.split(".", 2)[-1] if "." in newer else newer
            old_body = note_body_text.get(older, "")
            new_body = note_body_text.get(newer, "")
            if old_slug in new_body or new_slug in old_body:
                slug_mention_pairs.add((older, newer))

    print(f"  Cosine ≥ {COSINE_THRESHOLD} pairs: {len(cosine_candidates)}", flush=True)
    print(f"  Slug-mention pairs: {len(slug_mention_pairs)}", flush=True)

    # Slug-mention pairs always included; cosine pairs ranked and capped at MAX_CANDIDATES
    slug_only = slug_mention_pairs - set(cosine_candidates.keys())
    cosine_sorted = sorted(cosine_candidates.items(), key=lambda kv: kv[1], reverse=True)

    cap = MAX_CANDIDATES - len(slug_only)
    cosine_capped = [pair for pair, _ in cosine_sorted[:max(0, cap)]]

    candidate_pairs: list[tuple[str, str]] = (
        list(slug_only) + cosine_capped
    )

    # Deduplicate while preserving order
    seen: set[tuple[str, str]] = set()
    deduped: list[tuple[str, str]] = []
    for pair in candidate_pairs:
        if pair not in seen:
            seen.add(pair)
            deduped.append(pair)
    candidate_pairs = deduped

    print(f"Total candidate pairs (bounded to ~{MAX_CANDIDATES}): {len(candidate_pairs)}", flush=True)

    # Batch and run LLM gate
    batches = [
        candidate_pairs[i: i + BATCH_SIZE]
        for i in range(0, len(candidate_pairs), BATCH_SIZE)
    ]
    print(f"Running LLM gate: {len(batches)} batches × ≤{BATCH_SIZE} pairs (model: {MODEL})", flush=True)

    all_decisions: list[dict] = []
    total_cost = 0.0
    error_pairs: list[tuple[str, str]] = []

    for batch_idx, batch in enumerate(batches):
        prompt = build_batch_prompt(batch, snippets)
        print(f"  Batch {batch_idx + 1}/{len(batches)} ({len(batch)} pairs) …", flush=True)
        out = _claude(prompt)
        cost = out.get("total_cost_usd", 0) or 0
        total_cost += cost
        result_text = (out.get("result") or "").strip()

        if not result_text or out.get("is_error"):
            print(f"  ERROR: empty/error response for batch {batch_idx + 1} (${cost:.3f})", flush=True)
            error_pairs.extend(batch)
            continue

        try:
            decisions = parse_llm_decisions(result_text)
        except (ValueError, json.JSONDecodeError) as exc:
            print(f"  Parse error batch {batch_idx + 1}: {exc} — retrying once", flush=True)
            time.sleep(5)
            out2 = _claude(prompt)
            cost2 = out2.get("total_cost_usd", 0) or 0
            total_cost += cost2
            cost += cost2
            result_text2 = (out2.get("result") or "").strip()
            try:
                decisions = parse_llm_decisions(result_text2)
            except (ValueError, json.JSONDecodeError) as exc2:
                print(f"  PARSE ERROR after retry batch {batch_idx + 1}: {exc2}", flush=True)
                error_pairs.extend(batch)
                continue

        all_decisions.extend(decisions)
        persists = sum(1 for d in decisions if d.get("verdict") == "PERSIST")
        print(f"    → {len(decisions)} decisions, {persists} PERSIST  ${cost:.3f}", flush=True)

    print(f"\nLLM gate complete: {len(all_decisions)} decisions, ${total_cost:.3f} total", flush=True)
    if error_pairs:
        print(f"ERROR pairs (unparseable, excluded): {len(error_pairs)}", flush=True)

    # Build surviving edges
    edges: list[dict] = []
    for dec in all_decisions:
        if dec.get("verdict") == "PERSIST":
            edges.append({
                "old": dec.get("old", ""),
                "new": dec.get("new", ""),
                "type": dec.get("type", "updates"),
                "claim": (dec.get("claim") or "").strip(),
            })

    # Degree distribution (treating each edge as bidirectional for stats)
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        degree[e["old"]] += 1
        degree[e["new"]] += 1
    degrees = sorted(degree.values(), reverse=True)
    isolated = len(md_files) - len(degree)

    print(f"\n=== L5 Fabric Stats ===", flush=True)
    print(f"  Candidate pairs evaluated:   {len(candidate_pairs)}", flush=True)
    print(f"  PERSIST edges:               {len(edges)}", flush=True)
    print(f"  Error pairs excluded:        {len(error_pairs)}", flush=True)
    print(f"  Notes with edges:            {len(degree)} / {len(md_files)}", flush=True)
    print(f"  Isolated notes:              {isolated}", flush=True)
    if degrees:
        print(f"  Degree min/med/max:          {degrees[-1]}/{degrees[len(degrees)//2]}/{degrees[0]}", flush=True)

    # Type breakdown
    by_type: dict[str, int] = defaultdict(int)
    for e in edges:
        by_type[e["type"]] += 1
    print(f"  Edge types: {dict(by_type)}", flush=True)

    print(f"\nTop-5 highest-degree notes (L5):", flush=True)
    degree_sorted = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)
    for basename, deg in degree_sorted[:5]:
        print(f"  {basename}: degree {deg}", flush=True)

    print(f"\nTotal LLM spend (L5): ${total_cost:.4f}", flush=True)

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)
    print(f"Wrote {len(edges)} edges → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
