"""Builders for the cake cross-cluster fixtures. Each note is a real engram fact note
(with a .vec.json sidecar) so `engram query` clusters them exactly as production would."""
import os
import shutil
import subprocess

# (slug, subject, predicate, object) — the predicate/object encode the means-ends shared key:
# a "needs <X>" note and a "provides <X>" note share the literal key X.
CAKE_REQ = [
    ("cake-needs-sweetness", "a cake", "needs", "sweetness"),
    ("cake-needs-texture", "a cake", "needs", "texture"),
    ("cake-needs-fluffiness", "a cake", "needs", "fluffiness"),
]
CAKE_MECH = [
    ("sugar-provides-sweetness", "sugar", "provides", "sweetness"),
    ("flour-provides-texture", "flour", "provides", "texture"),
    ("bakingsoda-provides-fluffiness", "baking soda", "provides", "fluffiness"),
]
# Unrelated cluster for the precision control — git notes share no shared key with cake notes.
GIT_NOTES = [
    ("git-rebase-before-merge", "a feature branch", "must be rebased on", "main before merge"),
    ("git-ff-only-merges", "merges", "must be", "fast-forward only"),
    ("git-never-push-unreviewed", "worktree work", "must not be", "pushed before review"),
]


def _learn(vault, slug, subj, pred, obj):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(
        ["engram", "learn", "fact", "--slug", slug, "--position", "top",
         "--source", f"cake fixture: {slug}",
         "--situation", f"{subj} {pred} {obj}",
         "--subject", subj, "--predicate", pred, "--object", obj],
        env=env, check=True, capture_output=True, text=True)


def build(kind, dst):
    if os.path.exists(dst):
        shutil.rmtree(dst)
    os.makedirs(dst)
    if kind == "cake":
        notes = CAKE_REQ + CAKE_MECH
    elif kind == "control":
        notes = CAKE_REQ + CAKE_MECH + GIT_NOTES   # cake + genuinely-unrelated git cluster
    elif kind == "analogy":
        # a tempting-but-invalid analogy pair: both "rise" but no shared provided property key
        notes = CAKE_MECH + [
            ("bread-dough-rises", "bread dough", "rises when", "yeast ferments"),
            ("stock-market-rises", "the stock market", "rises when", "demand grows"),
        ]
    elif kind == "transitive":
        notes = [
            ("joe-wants-cake", "Joe", "wants", "cake"),
            ("cake-needs-sweetness", "a cake", "needs", "sweetness"),
            ("sugar-provides-sweetness", "sugar", "provides", "sweetness"),
        ]
    else:
        raise ValueError(kind)
    for slug, subj, pred, obj in notes:
        _learn(dst, slug, subj, pred, obj)
    # Embed-on-write can silently warn-and-skip the sidecar (learn.go autoEmbedNote). Without a
    # .vec.json, `engram query` cannot cluster the note and the RED/GREEN check passes vacuously
    # (0 clusters → 0 edges). Verify every note got a sidecar; fail loud if not.
    missing = [n for n in os.listdir(dst)
               if n.endswith(".md") and not os.path.exists(os.path.join(dst, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"fixture {kind}: missing .vec.json sidecars for {missing} — "
                           f"run `engram embed apply --missing` or check the embedder")
    return sorted(n for n in os.listdir(dst) if n.endswith(".md"))
