"""Fabric L3: shared-rare-token edges.

For each note, tokenizes the note body (full text including situation field)
using a simple word tokenizer. Computes document frequency (DF) per token.
Links notes that share a token appearing in ≤DF_THRESHOLD notes (rare tokens).

The DF threshold is tuned so the fabric is not a flood — see the degree
distribution report. Plan target: meaningful connectivity, not a hairball.

Writes dev/eval/links/fabrics/l3.json: [{src, dst, key: token}]
Prints degree distribution and the settled threshold.
"""
import glob
import json
import math
import os
import re
import sys
from collections import defaultdict

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
OUT_PATH = os.path.join(os.path.dirname(__file__), "fabrics", "l3.json")

# Tokens to ignore — common words that appear in most notes as boilerplate
STOP_WORDS = {
    "a", "an", "the", "and", "or", "not", "is", "are", "was", "were", "be",
    "been", "being", "have", "has", "had", "do", "does", "did", "will", "would",
    "shall", "should", "may", "might", "must", "can", "could", "to", "of", "in",
    "on", "at", "by", "for", "with", "from", "as", "this", "that", "these",
    "those", "it", "its", "i", "we", "you", "he", "she", "they", "their", "our",
    "your", "my", "his", "her", "what", "which", "who", "when", "where", "how",
    "if", "than", "but", "so", "yet", "nor", "both", "either", "neither",
    "no", "any", "all", "each", "every", "some", "same", "other", "such",
    "into", "out", "up", "down", "over", "under", "between", "through",
    "about", "above", "below", "before", "after", "during", "without",
    "within", "along", "across", "among", "around", "per", "toward",
    "use", "used", "using", "also", "only", "just", "more", "less",
    "very", "much", "well", "even", "still", "already", "then", "there",
    "here", "now", "new", "one", "two", "three", "first", "last", "next",
    "most", "best", "true", "false", "null", "none", "whether",
    # YAML frontmatter boilerplate that bleeds into body text
    "information", "learned", "when", "lesson", "context", "situation",
    "subject", "predicate", "object", "type", "fact", "feedback",
    "tier", "l2", "luhmann", "created", "source", "predicate",
    "related", "action", "behavior", "impact", "session",
}

# Minimum token length (short tokens are too generic)
MIN_TOKEN_LEN = 4

_WORD_RE = re.compile(r"[a-z][a-z0-9_-]*")
_FRONTMATTER_RE = re.compile(r"^---\n.*?\n---\n", re.DOTALL)


def extract_text(content: str) -> str:
    """Return the meaningful text from a note (body after frontmatter, plus situation field)."""
    # Include full text — situation field is inside frontmatter, valuable for token matching
    return content.lower()


def tokenize(text: str) -> set[str]:
    """Return the set of meaningful lowercase word tokens from text."""
    tokens = set()
    for tok in _WORD_RE.findall(text):
        if len(tok) >= MIN_TOKEN_LEN and tok not in STOP_WORDS:
            tokens.add(tok)
    return tokens


def main() -> None:
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")

    print(f"Tokenizing {len(md_files)} notes …", flush=True)

    # Build per-note token sets and document frequency
    note_tokens: dict[str, set[str]] = {}
    doc_freq: dict[str, int] = defaultdict(int)

    for fpath in md_files:
        basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()
        text = extract_text(content)
        tokens = tokenize(text)
        note_tokens[basename] = tokens
        for tok in tokens:
            doc_freq[tok] += 1

    # Distribution of document frequencies
    df_values = sorted(doc_freq.values())
    n_tokens = len(df_values)
    print(f"Total unique tokens: {n_tokens}", flush=True)

    # Print DF distribution to justify threshold choice
    df_histogram: dict[int, int] = defaultdict(int)
    for df in df_values:
        df_histogram[df] += 1

    print("\nDF distribution (df: count_of_tokens):", flush=True)
    for df in sorted(df_histogram):
        if df <= 20:
            print(f"  df={df}: {df_histogram[df]} tokens", flush=True)

    # Tokens with df > 20
    high_df = sum(count for df, count in df_histogram.items() if df > 20)
    print(f"  df>20: {high_df} tokens (filtered out)", flush=True)

    # Tune threshold: tokens in ≤4 docs are "rare" — justification:
    # - df=1 (hapax): these link no pairs at all (unique per note) → excluded implicitly
    # - df=2..4: shared by a handful of notes, likely topically related
    # - df>4: starts to include general vocabulary that would flood the graph
    # We try DF_THRESHOLD=4 and check the degree distribution.
    DF_THRESHOLD = 4

    # Build token → notes index (only rare tokens)
    rare_token_notes: dict[str, list[str]] = defaultdict(list)
    for basename, tokens in note_tokens.items():
        for tok in tokens:
            if 2 <= doc_freq[tok] <= DF_THRESHOLD:
                rare_token_notes[tok].append(basename)

    rare_count = len(rare_token_notes)
    print(f"\nRare tokens (2 ≤ df ≤ {DF_THRESHOLD}): {rare_count}", flush=True)

    # Build edges: for each rare token, create edges between all note pairs sharing it
    # Use a set to deduplicate edges (undirected, but stored as directed src<dst)
    edge_set: dict[tuple[str, str], str] = {}  # (src, dst) → token

    for tok, notes in rare_token_notes.items():
        notes_sorted = sorted(notes)
        for i in range(len(notes_sorted)):
            for j in range(i + 1, len(notes_sorted)):
                src, dst = notes_sorted[i], notes_sorted[j]
                key = (src, dst)
                if key not in edge_set:
                    edge_set[key] = tok

    edges = [{"src": src, "dst": dst, "key": tok} for (src, dst), tok in sorted(edge_set.items())]

    print(f"Edges (before threshold tune check): {len(edges)}", flush=True)

    # Check degree distribution
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        degree[e["src"]] += 1
        degree[e["dst"]] += 1

    degrees = sorted(degree.values(), reverse=True)
    if degrees:
        total_nodes = len(degree)
        print(f"\nDegree distribution (df≤{DF_THRESHOLD}):", flush=True)
        print(f"  Nodes touched: {total_nodes} / {len(md_files)}", flush=True)
        print(f"  Isolated (no edges): {len(md_files) - total_nodes}", flush=True)
        print(f"  Min degree: {degrees[-1]}", flush=True)
        print(f"  Median degree: {degrees[len(degrees)//2]}", flush=True)
        print(f"  Max degree: {degrees[0]}", flush=True)

        # If flood (max_degree > 20% of total notes), tighten to DF_THRESHOLD=3
        if degrees[0] > 0.20 * len(md_files):
            print(f"\n  Max degree {degrees[0]} > 20% of {len(md_files)} notes — tightening to df≤3", flush=True)
            DF_THRESHOLD = 3
            edge_set.clear()
            for tok, notes in rare_token_notes.items():
                if doc_freq[tok] > DF_THRESHOLD:
                    continue
                notes_sorted = sorted(notes)
                for i in range(len(notes_sorted)):
                    for j in range(i + 1, len(notes_sorted)):
                        src, dst = notes_sorted[i], notes_sorted[j]
                        key = (src, dst)
                        if key not in edge_set:
                            edge_set[key] = tok

            edges = [{"src": src, "dst": dst, "key": tok}
                     for (src, dst), tok in sorted(edge_set.items())]

            degree = defaultdict(int)
            for e in edges:
                degree[e["src"]] += 1
                degree[e["dst"]] += 1
            degrees = sorted(degree.values(), reverse=True)
            total_nodes = len(degree)
            print(f"  After tightening: {len(edges)} edges", flush=True)
            print(f"  Nodes touched: {total_nodes}", flush=True)
            print(f"  Max degree: {degrees[0] if degrees else 0}", flush=True)

    print(f"\nSettled threshold: df ≤ {DF_THRESHOLD}", flush=True)
    print(f"Final edge count: {len(edges)}", flush=True)

    # Top-5 highest-degree notes
    degree_sorted = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)
    print("\nTop-5 highest-degree notes (L3):", flush=True)
    for basename, deg in degree_sorted[:5]:
        print(f"  {basename}: degree {deg}", flush=True)

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)

    print(f"\nWrote {len(edges)} edges → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
