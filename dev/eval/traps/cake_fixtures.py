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


def _basename(vault, slug):
    for n in os.listdir(vault):
        if n.endswith(f".{slug}.md"):
            return n[:-3]   # drop .md
    raise RuntimeError(f"no note for slug {slug} in {vault}")


def _amend(vault, target_id, rel_basename, typed):
    # `engram amend --relation` renders a body "Related to:" wikilink, which
    # ParseWikilinks parses into Note.Outgoing (the graph BFS traverses these).
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(["engram", "amend", "--target", target_id,
                    "--relation", f"{rel_basename}|{typed}"],
                   env=env, check=True, capture_output=True, text=True)


def _link_transitive_chain(vault, mid_slug, end_slug):
    # joe-wants-cake --(causal: cake)--> <mid> --(means-ends: sweetness)--> <end>
    mid = _basename(vault, mid_slug)
    end = _basename(vault, end_slug)
    _amend(vault, "1", mid, "causal: cake")
    _amend(vault, "2", end, "means-ends: sweetness")


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
    elif kind == "transitive_blind":
        # Neutral slugs: the answer ("sugar") lives ONLY in the bridge note's
        # subject/body, never in its basename — so using it REQUIRES reading the note.
        notes = [
            ("joe-wants-cake", "Joe", "wants", "cake"),
            ("requirement-one", "a cake", "needs", "sweetness"),
            ("supplier-two", "sugar", "provides", "sweetness"),
        ]
    elif kind == "crossdomain":
        # The bridge B (flood/road vocabulary) shares only the DATE "March 16th" with note A
        # — never the query's "birthday party" vocabulary. So no phrase an agent generates
        # about the party cosine-reaches B; only the A->B wikilink (shared date) does.
        notes = [
            ("joe-party-date", "Joe's birthday party", "happens on", "March 16th at the community center"),
            ("flood-advisory", "March 16th", "coincides with",
             "the seasonal river flood that closes Bridge Road all day"),
            # topical distractors so cosine has party-relevant alternatives to surface
            ("party-catering", "a birthday party", "usually includes", "catering and a cake"),
            ("party-guests", "a birthday party", "needs", "a guest list and invitations"),
        ]
    else:
        raise ValueError(kind)
    for slug, subj, pred, obj in notes:
        _learn(dst, slug, subj, pred, obj)
    if kind == "transitive":
        _link_transitive_chain(dst, "cake-needs-sweetness", "sugar-provides-sweetness")
    elif kind == "transitive_blind":
        _link_transitive_chain(dst, "requirement-one", "supplier-two")
    elif kind == "crossdomain":
        # A (joe-party-date, luhmann 1) --[shared date]--> B (flood-advisory)
        _amend(dst, "1", _basename(dst, "flood-advisory"), "causal: March 16th date")
    # Embed-on-write can silently warn-and-skip the sidecar (learn.go autoEmbedNote). Without a
    # .vec.json, `engram query` cannot cluster the note and the RED/GREEN check passes vacuously
    # (0 clusters → 0 edges). Verify every note got a sidecar; fail loud if not.
    missing = [n for n in os.listdir(dst)
               if n.endswith(".md") and not os.path.exists(os.path.join(dst, n[:-3] + ".vec.json"))]
    if missing:
        raise RuntimeError(f"fixture {kind}: missing .vec.json sidecars for {missing} — "
                           f"run `engram embed apply --missing` or check the embedder")
    return sorted(n for n in os.listdir(dst) if n.endswith(".md"))
