"""Fabric L2: corpus-wide link-on-write with LLM JUSTIFY gate.

Per note:
  1. Top-K=8 candidates by body-vector cosine over the whole vault (exclude self).
  2. LLM gate (sonnet, ~5 source notes batched per call): propose a link ONLY with
     (a) a relation TYPE from {means-ends, causal, contradiction},
     (b) a SHARED KEY passing that type's test:
         means-ends: need term == provided effect
         causal: cause names the effect term
         contradiction: same subject+predicate opposite object
     (c) the key is a specific property/entity/effect — NOT a topic word.
     Default: DROP.
  3. Hub test: if one key would license linking a note to 3+ others, drop ALL
     edges under that key (applied post-pass over all pairs).

Emits:
  dev/eval/links/fabrics/l2_audit.txt  one line per candidate: PERSIST|DROP + key
  dev/eval/links/fabrics/l2.json       [{src, dst, type, key}]

Cost tracked per batch and printed at completion.
Fails loud on unparseable LLM output (retries once, then records error — no silent drops).
"""
import collections
import concurrent.futures as cf
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
AUDIT_PATH = os.path.join(HERE, "fabrics", "l2_audit.txt")
OUT_PATH = os.path.join(HERE, "fabrics", "l2.json")

MODELS = {"sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
MODEL = "sonnet"

TOP_K = 8             # candidates per source note
BATCH_SIZE = 5        # source notes per LLM call
HUB_THRESHOLD = 3     # edges under a single key triggers hub drop

# Frontmatter regex
_FM_RE = re.compile(r"^---\n(.*?)\n---\s*\n?", re.DOTALL)
_KV_RE = re.compile(r"^([a-z_]+):\s*(.*)", re.IGNORECASE)


def parse_frontmatter(content: str) -> dict[str, str]:
    """Extract key-value pairs from YAML frontmatter (simple, not full YAML)."""
    fm_match = _FM_RE.match(content)
    if not fm_match:
        return {}
    result: dict[str, str] = {}
    current_key = None
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


def note_snippet(content: str, max_claim_len: int = 200) -> str:
    """Compact 2-line snippet for LLM evaluation: situation + key claim."""
    fm = parse_frontmatter(content)
    situation = fm.get("situation", "(no situation)")
    note_type = fm.get("type", "")

    if note_type == "fact":
        subject = fm.get("subject", "")
        predicate = fm.get("predicate", "")
        obj = fm.get("object", "")[:max_claim_len]
        claim = f"{subject} {predicate}: {obj}"
    elif note_type == "feedback":
        behavior = fm.get("behavior", "")[:100]
        impact = fm.get("impact", "")[:max_claim_len]
        claim = f"behavior: {behavior}; impact: {impact}"
    else:
        body = _FM_RE.sub("", content).strip()
        claim = body[:max_claim_len]

    return f"situation: {situation}\nclaim: {claim[:max_claim_len]}"


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """Cosine similarity between two float vectors."""
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (norm_a * norm_b)


def load_body_vectors(vault_path: str) -> dict[str, list[float]]:
    """Load body_vector from each .vec.json sidecar."""
    vectors: dict[str, list[float]] = {}
    for fpath in sorted(glob.glob(os.path.join(vault_path, "*.vec.json"))):
        basename = os.path.basename(fpath)[: -len(".vec.json")]
        with open(fpath) as fh:
            data = json.load(fh)
        vec = data.get("body_vector")
        if vec and isinstance(vec, list):
            vectors[basename] = vec
        else:
            print(f"  WARNING: no body_vector in {os.path.basename(fpath)}", flush=True)
    return vectors


def _claude(prompt: str, model: str = MODEL) -> dict:
    """One `claude -p` call with transient-failure retry (is_error + near-zero cost → retry)."""
    wd = tempfile.mkdtemp(prefix="fabric-l2-")
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


_JUSTIFY_SYSTEM = """\
You are a memory-note link gatekeeper. Evaluate candidate links between memory notes.

LINK TYPES and their tests — propose a link ONLY when the key LITERALLY passes:
- means-ends: Note A describes a NEED (gap/requirement) that Note B PROVIDES (supplies/solves).
  KEY = the specific need/effect term that A needs and B provides.
- causal: Note A describes a CAUSE that, when present, produces the EFFECT stated in Note B
  (or vice versa). KEY = the specific cause→effect term.
- contradiction: Both notes address the SAME subject+predicate but assert OPPOSITE verdicts/objects.
  KEY = the specific subject+predicate they both address.

KEY RULES (strict):
- Must be a specific property, entity, or effect — e.g., "nil pointer before check",
  "LLM cost per call", "hub suppression threshold τ=0.8".
- NOT a topic word — e.g., "memory", "retrieval", "testing", "evaluation".

Default: DROP. Only PERSIST when TYPE is certain, KEY is specific, and KEY passes the test.

Respond with a JSON array (no markdown, no prose). One object per pair IN ORDER:
[
  {"src": "...", "dst": "...", "verdict": "PERSIST", "type": "means-ends", "key": "..."},
  {"src": "...", "dst": "...", "verdict": "DROP", "type": null, "key": null},
  ...
]"""


def build_batch_prompt(batch_sources: list[tuple[str, str]], note_snippets: dict[str, str]) -> str:
    """Build an LLM prompt for a batch of source notes and their candidates."""
    lines = [_JUSTIFY_SYSTEM, "", "=== NOTES AND PAIRS TO EVALUATE ===", ""]

    for src_basename, candidates_json in batch_sources:
        candidates = json.loads(candidates_json)
        lines.append(f"[SOURCE: {src_basename}]")
        lines.append(note_snippets.get(src_basename, "(snippet unavailable)"))
        lines.append("")

        for dst_basename in candidates:
            lines.append(f"  [CANDIDATE: {dst_basename}]")
            snippet = note_snippets.get(dst_basename, "(snippet unavailable)")
            for sline in snippet.splitlines():
                lines.append(f"  {sline}")
            lines.append("")

        lines.append(f"Pairs to evaluate for {src_basename}:")
        for dst_basename in candidates:
            lines.append(f'  {{"src": "{src_basename}", "dst": "{dst_basename}"}}')
        lines.append("")

    return "\n".join(lines)


def parse_llm_decisions(text: str) -> list[dict]:
    """Parse JSON array from LLM output. Raises ValueError on failure."""
    text = text.strip()
    # Strip markdown code fences if present
    if text.startswith("```"):
        text = re.sub(r"^```[a-z]*\n?", "", text, flags=re.MULTILINE)
        text = text.rstrip("`").strip()

    # Find JSON array
    start = text.find("[")
    end = text.rfind("]")
    if start == -1 or end == -1:
        raise ValueError(f"No JSON array found in LLM output: {text[:200]!r}")

    return json.loads(text[start: end + 1])


def main() -> None:
    print(f"Loading vault notes from {VAULT_PATH}", flush=True)
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")
    print(f"Found {len(md_files)} notes", flush=True)

    # Load body vectors
    print("Loading body vectors …", flush=True)
    body_vectors = load_body_vectors(VAULT_PATH)
    notes = sorted(body_vectors.keys())
    print(f"Loaded {len(notes)} body vectors", flush=True)

    # Load note content and build snippets
    print("Building note snippets …", flush=True)
    note_contents: dict[str, str] = {}
    note_snippets: dict[str, str] = {}
    for fpath in md_files:
        basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()
        note_contents[basename] = content
        note_snippets[basename] = note_snippet(content)

    # Top-K candidates per note by body cosine
    print(f"Computing top-{TOP_K} body-cosine candidates per note …", flush=True)
    candidates_per_note: dict[str, list[str]] = {}
    for src in notes:
        src_vec = body_vectors[src]
        scored = []
        for dst in notes:
            if dst == src:
                continue
            sim = cosine_similarity(src_vec, body_vectors[dst])
            scored.append((sim, dst))
        scored.sort(reverse=True)
        candidates_per_note[src] = [dst for _, dst in scored[:TOP_K]]

    total_pairs = sum(len(cands) for cands in candidates_per_note.values())
    print(f"Total (src, cand) pairs to evaluate: {total_pairs}", flush=True)

    # Batch sources and run LLM gate
    source_list = list(notes)
    batches = [
        source_list[i: i + BATCH_SIZE]
        for i in range(0, len(source_list), BATCH_SIZE)
    ]
    print(f"Running LLM gate: {len(batches)} batches × ≤{BATCH_SIZE} sources (model: {MODEL})", flush=True)

    all_decisions: list[dict] = []
    total_cost = 0.0
    error_pairs: list[str] = []

    for batch_idx, batch in enumerate(batches):
        batch_sources = [
            (src, json.dumps(candidates_per_note[src]))
            for src in batch
        ]
        prompt = build_batch_prompt(batch_sources, note_snippets)

        print(f"  Batch {batch_idx + 1}/{len(batches)}: {[s for s, _ in batch_sources]} …", flush=True)
        out = _claude(prompt)
        cost = out.get("total_cost_usd", 0) or 0
        total_cost += cost
        result_text = (out.get("result") or "").strip()

        if not result_text or out.get("is_error"):
            err_msg = f"BATCH {batch_idx + 1} ERROR: empty or error response (cost ${cost:.3f})"
            print(f"  {err_msg}", flush=True)
            for src in batch:
                for dst in candidates_per_note[src]:
                    error_pairs.append(f"{src} → {dst}")
            continue

        try:
            decisions = parse_llm_decisions(result_text)
        except (ValueError, json.JSONDecodeError) as exc:
            print(f"  Parse error on batch {batch_idx + 1}: {exc} — retrying once", flush=True)
            time.sleep(5)
            out2 = _claude(prompt)
            cost2 = out2.get("total_cost_usd", 0) or 0
            total_cost += cost2
            cost += cost2
            result_text2 = (out2.get("result") or "").strip()
            try:
                decisions = parse_llm_decisions(result_text2)
            except (ValueError, json.JSONDecodeError) as exc2:
                err_msg = f"BATCH {batch_idx + 1} PARSE ERROR (after retry): {exc2}"
                print(f"  {err_msg}", flush=True)
                for src in batch:
                    for dst in candidates_per_note[src]:
                        error_pairs.append(f"{src} → {dst}")
                continue

        all_decisions.extend(decisions)
        persists = sum(1 for d in decisions if d.get("verdict") == "PERSIST")
        print(f"    → {len(decisions)} decisions, {persists} PERSIST  ${cost:.3f}", flush=True)

    print(f"\nLLM gate complete: {len(all_decisions)} decisions, ${total_cost:.3f} total", flush=True)
    if error_pairs:
        print(f"ERROR pairs (unparseable, excluded): {len(error_pairs)}", flush=True)

    # Hub test: count edges per (src_or_dst, key) — if key licenses 3+ edges for any note, drop all
    key_edge_count: dict[str, int] = defaultdict(int)
    for dec in all_decisions:
        if dec.get("verdict") == "PERSIST" and dec.get("key"):
            key_edge_count[dec["key"]] += 1

    hub_keys = {key for key, count in key_edge_count.items() if count >= HUB_THRESHOLD}
    if hub_keys:
        print(f"\nHub test: {len(hub_keys)} keys with ≥{HUB_THRESHOLD} edges — dropping all edges under:", flush=True)
        for key in sorted(hub_keys):
            print(f"  key={key!r}: {key_edge_count[key]} edges", flush=True)

    # Build surviving edges
    edges: list[dict] = []
    audit_lines: list[str] = []

    for dec in all_decisions:
        src = dec.get("src", "")
        dst = dec.get("dst", "")
        verdict = dec.get("verdict", "DROP")
        rel_type = dec.get("type")
        key = dec.get("key")

        if verdict == "PERSIST" and key and key in hub_keys:
            verdict = "DROP"
            audit_lines.append(f"DROP(hub)\t{src}\t{dst}\tkey={key!r}")
        elif verdict == "PERSIST":
            edges.append({"src": src, "dst": dst, "type": rel_type, "key": key})
            audit_lines.append(f"PERSIST\t{src}\t{dst}\ttype={rel_type}\tkey={key!r}")
        else:
            audit_lines.append(f"DROP\t{src}\t{dst}\tkey={key!r}")

    for pair in error_pairs:
        audit_lines.append(f"ERROR\t{pair}")

    # Write outputs
    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(AUDIT_PATH, "w") as fh:
        fh.write("\n".join(audit_lines) + "\n")
    print(f"\nWrote {len(audit_lines)} audit lines → {AUDIT_PATH}", flush=True)

    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)
    print(f"Wrote {len(edges)} edges → {OUT_PATH}", flush=True)

    # Degree distribution
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        degree[e["src"]] += 1
        degree[e["dst"]] += 1
    degrees = sorted(degree.values(), reverse=True)
    isolated = len(md_files) - len(degree)

    print(f"\n=== L2 Fabric Stats ===", flush=True)
    print(f"  Total candidate pairs evaluated: {total_pairs}", flush=True)
    print(f"  PERSIST (pre-hub-test):   {sum(1 for d in all_decisions if d.get('verdict') == 'PERSIST')}", flush=True)
    print(f"  Hub keys dropped:         {len(hub_keys)}", flush=True)
    print(f"  Surviving edges:          {len(edges)}", flush=True)
    print(f"  Notes with edges:         {len(degree)} / {len(md_files)}", flush=True)
    print(f"  Isolated notes:           {isolated}", flush=True)
    if degrees:
        print(f"  Degree min/med/max:       {degrees[-1]}/{degrees[len(degrees)//2]}/{degrees[0]}", flush=True)

    print(f"\nTop-5 highest-degree notes (L2):", flush=True)
    degree_sorted = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)
    for basename, deg in degree_sorted[:5]:
        print(f"  {basename}: degree {deg}", flush=True)

    print(f"\nTotal LLM spend (L2): ${total_cost:.4f}", flush=True)


if __name__ == "__main__":
    main()
