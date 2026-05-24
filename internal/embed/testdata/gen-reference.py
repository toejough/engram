#!/usr/bin/env -S uv run --with sentence-transformers --with numpy --script
# /// script
# requires-python = ">=3.10"
# dependencies = ["sentence-transformers>=3.0", "numpy>=1.26"]
# ///
"""Generate reference cosines for the engram query spike UAT 13 gate.

Writes parity-reference.json with 5 sentence pairs, their cosines under
the chosen model, and the model identifier. Re-run whenever the pair set
or the model changes.
"""

import json
import sys
from pathlib import Path

import numpy as np
from sentence_transformers import SentenceTransformer

# 5 pairs span clearly-similar to clearly-different so a real embedder
# produces a recognisable spread of cosines.
PAIRS = [
    # 1: near-paraphrases (expected high)
    (
        "The cat sat on the mat.",
        "A cat is sitting on the mat.",
    ),
    # 2: same topic, different surface (expected medium-high)
    (
        "Verify the current behaviour before claiming a delta.",
        "Check what the system does today before asserting how a change differs.",
    ),
    # 3: shared vocab, different topic (expected low-medium)
    (
        "The agent embeds notes on write.",
        "The agent eats notes on Wednesdays.",
    ),
    # 4: unrelated (expected near zero)
    (
        "Cosine similarity ranges from minus one to one.",
        "Pour the batter into the greased pan.",
    ),
    # 5: identical (expected exactly 1.0)
    (
        "Identical sentences embed to identical vectors.",
        "Identical sentences embed to identical vectors.",
    ),
]

MODEL_ID = "sentence-transformers/all-MiniLM-L6-v2"


def cosine(a: np.ndarray, b: np.ndarray) -> float:
    return float(np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b)))


def main(out_path: Path) -> None:
    model = SentenceTransformer(MODEL_ID)
    pairs_out = []
    last_dims = None
    for left, right in PAIRS:
        vecs = model.encode([left, right], normalize_embeddings=False)
        last_dims = int(vecs.shape[1])
        pairs_out.append(
            {
                "left": left,
                "right": right,
                "cosine": cosine(vecs[0], vecs[1]),
            }
        )
    payload = {
        "model_id": MODEL_ID,
        "dims": last_dims,
        "pairs": pairs_out,
    }
    out_path.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"wrote {out_path}", file=sys.stderr)


if __name__ == "__main__":
    out = Path(__file__).parent / "parity-reference.json"
    main(out)
