#!/usr/bin/env python3
"""stub_engram — a fake `engram` binary for the C7 lever-recheck harness.

Why this exists: the real recall miss (a closed lever re-proposed as fresh) depends on the
disproving note being *buried* — present in the vault but NOT surfaced by the recall the agent
actually runs, because the agent's phrasing is semantically distant from the note (measured on the
real vault: note 80 = 0 hits under diagnostic phrasing, rank-1 under lever phrasing). A small toy
vault cannot reproduce that — the lone cost note always ranks #1 for any cost query. So instead of
authoring hundreds of notes to bury one, we put THIS stub on PATH ahead of the real `engram`: it
reproduces the *measured* phrasing-distance deterministically.

Behaviour, by subcommand:
  ingest --auto      -> no-op, prints a one-line tally.
  query --phrase ... -> emits a recall payload (engram's YAML shape). The BURIED note is included
                        ONLY when a phrase is lever-keyed (matches the lever's distinctive terms);
                        a general/diagnostic query gets the distractors only. Every query (its
                        phrases + whether the buried note was returned) is appended to the query log
                        ($STUB_ENGRAM_LOG) so the harness can assert, deterministically, whether the
                        skill ever issued a lever-keyed query.
  show <basename>    -> prints the note body.
  activate|embed|amend|learn ... -> no-op success (recall's write steps must not fail the run).

Config via env:
  ENGRAM_VAULT_PATH      vault dir holding the fixture notes (.md).
  STUB_ENGRAM_BURIED     basename (no .md) of the note to bury. Required for query.
  STUB_ENGRAM_LEVER_TERMS ';'-separated AND-groups; a phrase is lever-keyed if ANY group has ALL its
                          comma-separated terms present (case-insensitive). e.g.
                          "cheaper,retrieval;cheap,model;smaller,model".
  STUB_ENGRAM_LOG        path to append the JSONL query log (one object per query call).
"""
import json
import os
import sys

# Rank-1 score for the buried note on a lever-keyed query — above the distractor ladder's 0.80 top
# (the matched-note floor puts the exact-match note clearly first).
BURIED_TOP_SCORE = 0.92


def _vault_notes(vault):
    """Return {basename: (frontmatter_situation, body)} for every .md in the flat vault."""
    out = {}
    if not vault or not os.path.isdir(vault):
        return out
    for fn in sorted(os.listdir(vault)):
        if not fn.endswith(".md"):
            continue
        base = fn[:-3]
        text = open(os.path.join(vault, fn)).read()
        out[base] = text
    return out


def _phrase_is_lever_keyed(phrase, term_groups):
    low = phrase.lower()
    for group in term_groups:
        if group and all(t in low for t in group):
            return True
    return False


def _parse_phrases(argv):
    phrases = []
    i = 0
    while i < len(argv):
        if argv[i] == "--phrase" and i + 1 < len(argv):
            phrases.append(argv[i + 1])
            i += 2
        else:
            i += 1
    return phrases


def _emit_payload(items):
    """Emit a minimal but valid engram-query YAML payload the /recall skill can read."""
    lines = ["version: 1", "items:"]
    for it in items:
        lines.append(f"  - path: {it['path']}")
        lines.append(f"    kind: {it['kind']}")
        lines.append(f"    score: {it['score']}")
        lines.append("    content: |-")
        for cl in it["content"].splitlines():
            lines.append(f"      {cl}")
    # one cluster over the surfaced members; no candidate_l2s (nothing to crystallize)
    lines.append("clusters:")
    lines.append("  - id: 0")
    lines.append(f"    size: {len(items)}")
    lines.append("    candidate_l2s: []")
    lines.append("    members:")
    for it in items:
        lines.append(f"      - path: {it['path']}")
        lines.append(f"        score: {it['score']}")
    return "\n".join(lines) + "\n"


def _cmd_query(argv):
    vault = os.environ.get("ENGRAM_VAULT_PATH", "")
    buried = os.environ.get("STUB_ENGRAM_BURIED", "")
    term_groups = [
        [t.strip().lower() for t in grp.split(",") if t.strip()]
        for grp in os.environ.get("STUB_ENGRAM_LEVER_TERMS", "").split(";")
        if grp.strip()
    ]
    phrases = _parse_phrases(argv)
    notes = _vault_notes(vault)

    lever_keyed = any(_phrase_is_lever_keyed(p, term_groups) for p in phrases)
    surfaced = []
    if lever_keyed and buried in notes:
        # Measured reality (the matched-note floor — note 80, cited in the module docstring): a
        # lever-keyed query ranks the closure note #1, ABOVE every distractor. Emitting it
        # last/lowest (the old filename-ordered ladder put note 8 at the bottom at 0.38) made
        # agents honestly miss the bottom-ranked disproof — an instrument artifact, not a
        # synthesis failure.
        surfaced.append({"path": buried + ".md", "kind": "fact", "score": BURIED_TOP_SCORE,
                         "content": notes[buried]})
    score = 0.80
    for base, content in notes.items():
        if base == buried:
            continue  # keyed: already ranked #1 above; non-keyed: the measured retrieval miss
        surfaced.append({"path": base + ".md", "kind": "fact", "score": round(score, 3),
                         "content": content})
        score = max(0.30, score - 0.06)
    returned_buried = any(it["path"].startswith(buried) for it in surfaced) if buried else False

    log = os.environ.get("STUB_ENGRAM_LOG", "")
    if log:
        with open(log, "a") as fh:
            fh.write(json.dumps({"phrases": phrases, "lever_keyed": lever_keyed,
                                 "returned_buried": returned_buried}) + "\n")

    sys.stdout.write(_emit_payload(surfaced))
    return 0


def _cmd_show(argv):
    vault = os.environ.get("ENGRAM_VAULT_PATH", "")
    if not argv:
        return 0
    base = argv[0][:-3] if argv[0].endswith(".md") else argv[0]
    path = os.path.join(vault, base + ".md")
    if os.path.isfile(path):
        sys.stdout.write(open(path).read())
    return 0


def main(argv):
    if not argv:
        return 0
    cmd = argv[0]
    rest = argv[1:]
    if cmd == "query":
        return _cmd_query(rest)
    if cmd == "show":
        return _cmd_show(rest)
    if cmd == "ingest":
        sys.stdout.write("stub ingest: memory index up to date (0 chunks)\n")
        return 0
    # activate / embed / amend / learn / anything else: succeed quietly so recall's write steps
    # never fail the run under test.
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
