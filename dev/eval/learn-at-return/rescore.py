#!/usr/bin/env python3
"""Re-score an arm from its kept files: stream.jsonl plus the session digest (digest.py), or the
session/ JSONLs when they are still present. Dispatches on the arm id's cell to score.py (guidance cells),
lcells.py (L cells) or qrcells.py (Q/R cells), with the inputs each recorded run used.

    python3 rescore.py <arm-dir> [<arm-dir> ...]      # prints recorded vs re-scored label per arm
    python3 rescore.py --all <results-dir>            # every arm dir with a score.json; exit 1 on a mismatch
"""
import glob
import json
import os
import sys
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import digest  # noqa: E402
import lar  # noqa: E402
import lcells  # noqa: E402
import qrcells  # noqa: E402
import score  # noqa: E402


def sessions(arm_dir: str) -> Tuple[List[str], List[str], str]:
    if os.path.isfile(os.path.join(arm_dir, digest.MAIN)):
        m, s = digest.read_digest(arm_dir)
        return m, s, "digest"
    main, others = [], []
    for p in sorted(glob.glob(os.path.join(arm_dir, "session", "**", "*.jsonl"), recursive=True)):
        with open(p, errors="replace") as f:
            (others if os.sep + "subagents" + os.sep in p else main).append(f.read())
    return main, others, "session"


def _vault(arm_dir: str) -> Dict[str, str]:
    out = {}
    for p in sorted(glob.glob(os.path.join(arm_dir, "vault", "*.md"))):
        with open(p, errors="replace") as f:
            out[os.path.basename(p)] = f.read()
    return out


def rescore(arm_dir: str) -> Dict:
    with open(os.path.join(arm_dir, "score.json")) as f:
        rec = json.load(f)
    with open(os.path.join(arm_dir, "stream.jsonl")) as f:
        stream = f.readlines()
    main, others, src = sessions(arm_dir)
    cell, arm = rec["cell"], rec["arm"]
    if cell in lcells.L_CELLS:
        out = lcells.score_l(stream, main, others, cell, arm, rec.get("seeded") or [], _vault(arm_dir))
    elif cell in qrcells.CELL_SKILL:
        out = qrcells.score_qr(stream, main, others, cell, arm, sorted(_vault(arm_dir)))
    else:
        fx = lar.load_fixtures()
        with open(os.path.join(arm_dir, "unit1-report.txt")) as f:
            report = f.read()
        notes_path = os.path.join(arm_dir, "vault-notes.json")
        notes: Optional[List[str]] = None
        if os.path.isfile(notes_path):
            with open(notes_path) as f:
                notes = json.load(f)
        out = score.score_arm(stream, main, cell, arm, report, fx["cells"][cell]["lessons"], others, notes)
    return {"arm": arm_dir, "source": src, "recorded": rec["label"], "rescored": out["label"],
            "match": rec["label"] == out["label"]}


def main(argv: List[str]) -> int:
    dirs = argv
    if argv and argv[0] == "--all":
        dirs = sorted(os.path.dirname(p) for p in glob.glob(os.path.join(argv[1], "**", "score.json"), recursive=True))
    bad = 0
    for d in dirs:
        r = rescore(d)
        bad += not r["match"]
        print(f"{'ok ' if r['match'] else 'BAD'} {r['source']:7} {r['recorded']:>22} -> {r['rescored']:<22} {d}")
    print(f"arms: {len(dirs)}, mismatches: {bad}")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
