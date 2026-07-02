"""Retrieval-channel probe (FREE — embedder only, no LLM) for the question-anchored eval.

The delivery eval (qanchor_eval.py) injected the note, so it measured the APPLICATION channel. Note
120's "question-useful" could instead be a RETRIEVAL property: a question-shaped note embedding closer
to a future question and ranking above a topic-shaped one. This probe measures exactly that.

For each pair: seed a vault with BOTH the A (topic) and B (question) note, run `engram query` with the
future question, and compare their cosine scores. If B out-scores A, question-anchoring helps retrieval
(partial note-120 vindication). If not, the retrieval channel is neutral/A -> the PARK is bulletproof
on both channels. (Prior verified note 72: retrieval is not the bottleneck -> expect neutral.)

Usage: python3 qanchor_retrieval_probe.py
"""
import json
import os
import re
import subprocess
import tempfile

from qanchor_corpus import PAIRS

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/qanchor-eval")
NOTES = json.load(open(os.path.join(ROOT, "notes.json")))


def _split(note):
    """'SITUATION: x\\nLESSON: y' -> (x, y). Falls back to (whole, whole) if unparseable."""
    m = re.search(r"SITUATION:\s*(.*?)\s*LESSON:\s*(.*)", note, re.DOTALL | re.IGNORECASE)
    if not m:
        return note.strip(), note.strip()
    return m.group(1).strip(), m.group(2).strip()


def _learn(vault, slug, situation, lesson):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(["engram", "learn", "fact", "--slug", slug, "--position", "top",
                    "--source", f"qanchor probe: {slug}", "--situation", situation,
                    "--subject", "the lesson", "--predicate", "states", "--object", lesson],
                   env=env, check=True, capture_output=True, text=True)


def _query_scores(vault, chunks, phrase, slugs):
    """Return {slug: score} for the given query phrase over the seeded vault."""
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    r = subprocess.run(["engram", "query", "--lazy-chunks", "--phrase", phrase],
                       env=env, capture_output=True, text=True)
    scores = {}
    cur = None
    for line in r.stdout.splitlines():
        pm = re.match(r"\s*-?\s*path:\s*(.+)", line)
        sm = re.match(r"\s*score:\s*([0-9.]+)", line)
        if pm:
            cur = pm.group(1).strip()
        elif sm and cur:
            for s in slugs:
                if s in cur:
                    scores[s] = float(sm.group(1))
            cur = None
    return scores


def main():
    os.makedirs(os.path.join(ROOT, "probe"), exist_ok=True)
    print(f"{'pair':18} {'score_A':>8} {'score_B':>8}  {'winner':>7}")
    bwin = awin = tie = 0
    rows = []
    for key in PAIRS:
        vault = tempfile.mkdtemp(prefix=f"probe-{key}-", dir=os.path.join(ROOT, "probe"))
        chunks = os.path.join(vault, "chunks")
        os.makedirs(chunks, exist_ok=True)
        sa, la = _split(NOTES[key]["A"])
        sb, lb = _split(NOTES[key]["B"])
        slug_a, slug_b = f"qa-{key}-a", f"qa-{key}-b"
        _learn(vault, slug_a, sa, la)
        _learn(vault, slug_b, sb, lb)
        sc = _query_scores(vault, chunks, PAIRS[key]["future_q"], [slug_a, slug_b])
        a, b = sc.get(slug_a, 0.0), sc.get(slug_b, 0.0)
        w = "B>A" if b > a else ("A>B" if a > b else "=")
        bwin += b > a
        awin += a > b
        tie += a == b
        rows.append({"pair": key, "score_A": a, "score_B": b, "winner": w})
        print(f"{key:18} {a:>8.4f} {b:>8.4f}  {w:>7}")

    print(f"\n=== RETRIEVAL CHANNEL (cosine of future_q to each anchor) ===")
    print(f"  B out-ranks A in {bwin}/{len(PAIRS)} pairs; A>B in {awin}; tie {tie}")
    ma = sum(r["score_A"] for r in rows) / len(rows)
    mb = sum(r["score_B"] for r in rows) / len(rows)
    print(f"  mean score:  A={ma:.4f}  B={mb:.4f}  (B−A = {(mb-ma):+.4f})")
    json.dump(rows, open(os.path.join(ROOT, "retrieval-probe.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
