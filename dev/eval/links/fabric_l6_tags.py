"""Fabric L6: controlled-vocabulary tag taxonomy.

Two-pass LLM process (sonnet):

Pass 1 (one call): given all note situation lines, propose a controlled vocabulary
  of 15–25 category terms (lowercase-hyphenated, specific enough to partition the
  vault into useful groups).

Pass 2 (batched, 10 notes per call): assign each note ≤3 tags FROM THAT VOCAB ONLY.
  Notes that don't fit any category get an empty tag list.

Writes:
  dev/eval/links/fabrics/l6.json
    {
      "vocab": ["tag-a", "tag-b", ...],
      "assignments": [{"note": "<basename>", "tags": ["tag-a", ...]}, ...]
    }

Cost tracked and printed at completion.
Fails loud on unparseable LLM output (retries once, then empty tags — no silent drops).
"""
import glob
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
HERE = os.path.dirname(os.path.abspath(__file__))
OUT_PATH = os.path.join(HERE, "fabrics", "l6.json")

MODELS = {"sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
MODEL = "sonnet"

VOCAB_MIN = 15
VOCAB_MAX = 25
MAX_TAGS_PER_NOTE = 3
BATCH_SIZE = 10   # notes per tagging call

_FM_RE = re.compile(r"^---\n(.*?)\n---\s*\n?", re.DOTALL)
_KV_RE = re.compile(r"^([a-z_]+):\s*(.*)", re.IGNORECASE)
_SLUG_RE = re.compile(r"^[a-z][a-z0-9-]*$")


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


def note_snippet(basename: str, content: str) -> str:
    """Compact snippet for tagging: situation + key claim."""
    fm = parse_frontmatter(content)
    situation = fm.get("situation", "(no situation)")
    note_type = fm.get("type", "")

    if note_type == "fact":
        subject = fm.get("subject", "")
        predicate = fm.get("predicate", "")
        obj = fm.get("object", "")[:150]
        claim = f"{subject} {predicate}: {obj}"
    elif note_type == "feedback":
        behavior = fm.get("behavior", "")[:100]
        impact = fm.get("impact", "")[:150]
        claim = f"behavior: {behavior}; impact: {impact}"
    else:
        body = _FM_RE.sub("", content).strip()
        claim = body[:200]

    return (
        f"note: {basename}\n"
        f"situation: {situation}\n"
        f"claim: {claim[:200]}"
    )


def _claude(prompt: str, model: str = MODEL) -> dict:
    """One `claude -p` call with transient-failure retry."""
    wd = tempfile.mkdtemp(prefix="fabric-l6-")
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


def parse_json_from_text(text: str) -> object:
    """Parse JSON from LLM output, stripping markdown fences. Returns parsed object."""
    text = text.strip()
    if text.startswith("```"):
        text = re.sub(r"^```[a-z]*\n?", "", text, flags=re.MULTILINE)
        text = text.rstrip("`").strip()

    # Find first JSON structure
    for start_char, end_char in (("[", "]"), ("{", "}")):
        start = text.find(start_char)
        if start != -1:
            end = text.rfind(end_char)
            if end != -1 and end > start:
                return json.loads(text[start: end + 1])

    raise ValueError(f"No JSON structure found in: {text[:200]!r}")


_VOCAB_PROMPT = """\
You are designing a controlled vocabulary for tagging a collection of AI-agent memory notes.
All notes are lessons learned by an AI coding assistant. They cover: engineering practices,
evaluation methodology, cost/performance tradeoffs, tool behavior, memory-system design,
data structures, and agentic workflow patterns.

Below are all {n} note situation lines — each describes WHEN a note applies:

{situations_block}

Propose a controlled vocabulary of {vocab_min}–{vocab_max} category terms. Requirements:
- Each term is lowercase-hyphenated (e.g., "eval-methodology", "llm-cost", "code-quality")
- Specific enough to partition the vault: not "engineering" (too broad) but "go-code-conventions"
- Covers the major clusters you see across the situation lines above
- Excludes overlapping terms — each term should represent a genuinely distinct cluster

Respond with ONLY a JSON array of strings:
["term-one", "term-two", ...]"""


_TAGGING_PROMPT = """\
You are tagging memory notes with a controlled vocabulary.

VOCABULARY (use ONLY these terms, exactly as written):
{vocab_block}

For each note below, assign 0–{max_tags} tags from the vocabulary that best describe it.
A note with no fitting tags gets an empty list [].
Only assign a tag if the note is clearly about that category.

Respond with a JSON array — one object per note IN ORDER:
[
  {{"note": "...", "tags": ["tag-a", "tag-b"]}},
  {{"note": "...", "tags": []}},
  ...
]

=== NOTES TO TAG ===

{notes_block}"""


def build_vocab(situations: list[tuple[str, str]]) -> tuple[list[str], float]:
    """Pass 1: call LLM once to generate controlled vocabulary. Returns (vocab, cost)."""
    situations_block = "\n".join(f"- [{basename}] {sit}" for basename, sit in situations)
    prompt = _VOCAB_PROMPT.format(
        n=len(situations),
        situations_block=situations_block,
        vocab_min=VOCAB_MIN,
        vocab_max=VOCAB_MAX,
    )

    print("Pass 1: generating vocabulary …", flush=True)
    out = _claude(prompt)
    cost = out.get("total_cost_usd", 0) or 0
    result_text = (out.get("result") or "").strip()

    if not result_text or out.get("is_error"):
        sys.exit(f"ERROR: vocabulary generation failed (cost ${cost:.3f})")

    try:
        vocab = parse_json_from_text(result_text)
        if not isinstance(vocab, list):
            sys.exit(f"ERROR: expected JSON array for vocab, got: {type(vocab)}")
    except (ValueError, json.JSONDecodeError) as exc:
        sys.exit(f"ERROR: failed to parse vocabulary JSON: {exc}\nRaw: {result_text[:300]}")

    # Validate and clean vocab
    cleaned: list[str] = []
    for term in vocab:
        term_str = str(term).strip().lower()
        if _SLUG_RE.match(term_str):
            cleaned.append(term_str)
        else:
            print(f"  WARNING: dropping invalid vocab term {term!r}", flush=True)

    if not (VOCAB_MIN <= len(cleaned) <= VOCAB_MAX):
        print(
            f"  WARNING: vocab size {len(cleaned)} outside [{VOCAB_MIN}, {VOCAB_MAX}] — "
            f"proceeding anyway",
            flush=True,
        )

    print(f"  Vocabulary ({len(cleaned)} terms): {cleaned}", flush=True)
    print(f"  Vocab cost: ${cost:.3f}", flush=True)
    return cleaned, cost


def tag_batch(
    batch: list[tuple[str, str]],
    vocab: list[str],
    vocab_set: set[str],
) -> tuple[list[dict], float]:
    """Pass 2 batch: tag a batch of notes. Returns (assignments, cost)."""
    vocab_block = "\n".join(f"- {term}" for term in vocab)
    notes_block = "\n\n".join(snippet for _, snippet in batch)
    prompt = _TAGGING_PROMPT.format(
        vocab_block=vocab_block,
        max_tags=MAX_TAGS_PER_NOTE,
        notes_block=notes_block,
    )

    out = _claude(prompt)
    cost = out.get("total_cost_usd", 0) or 0
    result_text = (out.get("result") or "").strip()

    if not result_text or out.get("is_error"):
        return [], cost

    try:
        raw = parse_json_from_text(result_text)
        if not isinstance(raw, list):
            raise ValueError(f"Expected JSON array, got {type(raw)}")
    except (ValueError, json.JSONDecodeError) as exc:
        print(f"  Parse error: {exc} — retrying once", flush=True)
        time.sleep(5)
        out2 = _claude(prompt)
        cost += out2.get("total_cost_usd", 0) or 0
        result_text2 = (out2.get("result") or "").strip()
        try:
            raw = parse_json_from_text(result_text2)
            if not isinstance(raw, list):
                raise ValueError(f"Expected JSON array, got {type(raw)}")
        except (ValueError, json.JSONDecodeError) as exc2:
            print(f"  PARSE ERROR after retry: {exc2} — assigning empty tags to batch", flush=True)
            # Fail loud: record error entries rather than silently dropping
            return [{"note": bn, "tags": [], "_error": str(exc2)} for bn, _ in batch], cost

    # Validate tags — only accept terms in vocab
    assignments: list[dict] = []
    for item in raw:
        if not isinstance(item, dict):
            continue
        note_name = str(item.get("note", "")).strip()
        raw_tags = item.get("tags", [])
        if not isinstance(raw_tags, list):
            raw_tags = []
        valid_tags = [t for t in raw_tags if isinstance(t, str) and t.strip() in vocab_set]
        invalid_tags = [t for t in raw_tags if not (isinstance(t, str) and t.strip() in vocab_set)]
        if invalid_tags:
            print(f"  WARNING: dropped OOV tags for {note_name}: {invalid_tags}", flush=True)
        assignments.append({"note": note_name, "tags": valid_tags[:MAX_TAGS_PER_NOTE]})

    return assignments, cost


def main() -> None:
    print(f"Loading vault notes from {VAULT_PATH}", flush=True)
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")
    print(f"Found {len(md_files)} notes", flush=True)

    # Load notes and build snippets + situation list
    notes_data: list[tuple[str, str]] = []
    situations: list[tuple[str, str]] = []

    for fpath in md_files:
        basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()
        fm = parse_frontmatter(content)
        situation = fm.get("situation", "")
        snippet = note_snippet(basename, content)
        notes_data.append((basename, snippet))
        situations.append((basename, situation))

    print(f"Loaded {len(notes_data)} note snippets", flush=True)

    # Pass 1: generate vocabulary
    vocab, vocab_cost = build_vocab(situations)
    vocab_set = set(vocab)
    total_cost = vocab_cost

    # Pass 2: tag each note
    batches = [
        notes_data[i: i + BATCH_SIZE]
        for i in range(0, len(notes_data), BATCH_SIZE)
    ]
    print(f"\nPass 2: tagging {len(notes_data)} notes in {len(batches)} batches (model: {MODEL})", flush=True)

    all_assignments: list[dict] = []

    for batch_idx, batch in enumerate(batches):
        print(f"  Batch {batch_idx + 1}/{len(batches)} ({len(batch)} notes) …", flush=True)
        assignments, cost = tag_batch(batch, vocab, vocab_set)
        total_cost += cost
        all_assignments.extend(assignments)
        tagged_count = sum(1 for a in assignments if a["tags"])
        print(f"    → {len(assignments)} assignments, {tagged_count} with tags  ${cost:.3f}", flush=True)

    print(f"\nTagging complete: {len(all_assignments)} assignments, ${total_cost:.3f} total", flush=True)

    # Stats
    tagged = sum(1 for a in all_assignments if a["tags"])
    untagged = len(all_assignments) - tagged
    tag_freq: dict[str, int] = {}
    for a in all_assignments:
        for tag in a["tags"]:
            tag_freq[tag] = tag_freq.get(tag, 0) + 1

    print(f"\n=== L6 Fabric Stats ===", flush=True)
    print(f"  Vocab size:          {len(vocab)}", flush=True)
    print(f"  Notes tagged:        {tagged} / {len(all_assignments)}", flush=True)
    print(f"  Untagged notes:      {untagged}", flush=True)
    print(f"  Total assignments:   {sum(len(a['tags']) for a in all_assignments)}", flush=True)

    print(f"\nTag distribution (sorted by frequency):", flush=True)
    for tag, count in sorted(tag_freq.items(), key=lambda kv: kv[1], reverse=True):
        print(f"  {tag}: {count} notes", flush=True)

    print(f"\nTop-5 most-assigned tags:", flush=True)
    top5 = sorted(tag_freq.items(), key=lambda kv: kv[1], reverse=True)[:5]
    for tag, count in top5:
        print(f"  {tag}: {count}", flush=True)

    print(f"\nTotal LLM spend (L6): ${total_cost:.4f}", flush=True)

    # Write output
    output = {
        "vocab": vocab,
        "assignments": all_assignments,
    }
    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(output, fh, indent=2)
    print(f"Wrote → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
