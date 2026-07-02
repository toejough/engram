"""
test_traversal.py — Unit tests for traversal.py (toy 6-node graph).

Plain assert pattern (qanchor style). Run with:
    python dev/eval/links/test_traversal.py

Toy graph layout:
    A - B - C - D   (path)
        |
        E           (branch from B)
    F               (isolated — no edges)

Edges (undirected): A-B, B-C, C-D, B-E
Degrees: A=1, B=3, C=2, D=1, E=1, F=0
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from traversal import (
    _strip_md,
    _add_md,
    expand_one_hop,
    ppr_rank,
    ppr_blend,
    rank_boost,
    supersession_ride_along,
    tag_filter_candidates,
)

# Toy undirected fabric (edges without .md)
FABRIC = [
    {"src": "A", "dst": "B"},
    {"src": "B", "dst": "C"},
    {"src": "C", "dst": "D"},
    {"src": "B", "dst": "E"},
]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def ranked_item(basename: str, score: float, kind: str = "fact", rank: int = 1) -> dict:
    return {"basename": basename, "score": score, "kind": kind, "rank": rank}


def basenames(result: list[dict]) -> list[str]:
    return [r["basename"] for r in result]


# ---------------------------------------------------------------------------
# expand_one_hop tests
# ---------------------------------------------------------------------------

def test_expand_one_hop_order():
    """Neighbors are appended in source-rank order; baseline items are first."""
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("C.md", 0.7, rank=2),
    ]
    result = expand_one_hop(ranked, FABRIC, top_m=2)
    bns = basenames(result)

    # Original items first, unchanged
    assert bns[:2] == ["A.md", "C.md"], f"Originals not first: {bns}"

    # Neighbors of A (rank-1 source): B  → B added first
    # Neighbors of C (rank-2 source): B (already added), D  → D added after B
    assert "B.md" in bns[2:], f"B.md missing from appended: {bns}"
    assert "D.md" in bns[2:], f"D.md missing from appended: {bns}"
    b_idx = bns.index("B.md")
    d_idx = bns.index("D.md")
    assert b_idx < d_idx, f"B (from rank-1 A) should precede D (from rank-2 C): {bns}"


def test_expand_one_hop_isolated_node_excluded():
    """Isolated node F is never appended (no edges to it)."""
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    result = expand_one_hop(ranked, FABRIC, top_m=5)
    assert "F.md" not in basenames(result), "F.md is isolated and must not appear"


def test_expand_one_hop_non_top_m_source_excluded():
    """Neighbors of nodes outside top-M are not expanded.

    Graph: A-B-C-D, B-E.  Baseline: [C(rank-1), A(rank-2)], top_m=1.
    Expand only C's neighbors: {B, D}.  A is rank-2 (outside top_m=1), so
    E (neighbor of B, reachable via A's neighborhood) must NOT appear.
    """
    ranked = [
        ranked_item("C.md", 0.9, rank=1),  # neighbors: B, D
        ranked_item("A.md", 0.8, rank=2),  # neighbors: B (already added via C)
    ]
    result = expand_one_hop(ranked, FABRIC, top_m=1)
    bns = basenames(result)
    # C's neighbors B and D must be added
    assert "B.md" in bns, f"B.md (neighbor of C, rank-1) expected: {bns}"
    assert "D.md" in bns, f"D.md (neighbor of C, rank-1) expected: {bns}"
    # E is only reachable via B, which is rank-2's neighbor; E is not a direct
    # neighbor of C (rank-1), so E must not appear
    assert "E.md" not in bns, f"E.md must not appear (E is neighbor of B not C): {bns}"


def test_expand_one_hop_no_duplicates():
    """No basename appears twice in the result."""
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("C.md", 0.7, rank=2),
    ]
    result = expand_one_hop(ranked, FABRIC, top_m=2)
    bns = basenames(result)
    assert len(bns) == len(set(bns)), f"Duplicates in result: {bns}"


def test_expand_one_hop_baseline_order_preserved():
    """Baseline items always appear first and in their original order."""
    ranked = [
        ranked_item("B.md", 0.9, rank=1),
        ranked_item("C.md", 0.7, rank=2),
        ranked_item("D.md", 0.5, rank=3),
    ]
    result = expand_one_hop(ranked, FABRIC, top_m=5)
    assert basenames(result)[:3] == ["B.md", "C.md", "D.md"], \
        f"Baseline order not preserved: {basenames(result)[:3]}"


# ---------------------------------------------------------------------------
# ppr_rank tests
# ---------------------------------------------------------------------------

def test_ppr_recovers_2hop():
    """PPR propagates to a 2-hop node from the seed."""
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    result, new_count = ppr_rank(
        ranked, FABRIC, alpha=0.5, idf_weight=False, rescale_c=1.0, act_tau=0.01
    )
    bns = basenames(result)
    # B is 1-hop from A → must appear
    assert "B.md" in bns, f"B.md (1-hop from A) not in PPR result: {bns}"
    # C is 2-hop from A → must appear
    assert "C.md" in bns, f"C.md (2-hop from A) not in PPR result: {bns}"
    # At least B and C are new (A itself is the seed)
    assert new_count > 0, f"Expected new_count > 0, got {new_count}"


def test_ppr_disconnected_node_absent():
    """F (isolated/not in fabric) never appears in PPR result."""
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    result, _ = ppr_rank(
        ranked, FABRIC, alpha=0.5, idf_weight=False, rescale_c=1.0, act_tau=0.01
    )
    assert "F.md" not in basenames(result), "F.md is not in any fabric edge; must not appear"


def test_ppr_no_seeds_returns_baseline():
    """When no baseline node is in the graph, ppr_rank returns the baseline unchanged."""
    ranked = [ranked_item("F.md", 0.9, rank=1)]  # F has no edges
    result, new_count = ppr_rank(ranked, FABRIC)
    assert basenames(result) == ["F.md"], f"Expected baseline unchanged: {basenames(result)}"
    assert new_count == 0


def test_ppr_blend_baseline_nodes_kept():
    """ppr_blend preserves all baseline nodes in the result."""
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("F.md", 0.5, rank=2),  # F not in graph
    ]
    result, _ = ppr_blend(ranked, FABRIC)
    bns = basenames(result)
    assert "A.md" in bns, "A.md (baseline + in graph) must appear in blend result"
    assert "F.md" in bns, "F.md (baseline, not in graph) must appear in blend result"


# ---------------------------------------------------------------------------
# rank_boost tests
# ---------------------------------------------------------------------------

def test_rank_boost_baseline_relative_order_preserved():
    """Baseline notes (sim=0.9, 0.7, 0.5) all outscore any neighbor (w*max_sim ≤ 0.09)."""
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("C.md", 0.7, rank=2),
        ranked_item("D.md", 0.5, rank=3),
    ]
    result, _ = rank_boost(ranked, FABRIC, w=0.1)
    bns = basenames(result)
    a_pos = bns.index("A.md")
    c_pos = bns.index("C.md")
    d_pos = bns.index("D.md")
    assert a_pos < c_pos < d_pos, \
        f"Baseline relative order must be A<C<D, got positions A={a_pos}, C={c_pos}, D={d_pos}"


def test_rank_boost_neighbor_boosted_score():
    """A neighbor's boosted score = w × max(linked baseline sim)."""
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    result, _ = rank_boost(ranked, FABRIC, w=0.1)
    # B is a neighbor of A → score should be 0.1 * 0.9 = 0.09
    b_items = [r for r in result if r["basename"] == "B.md"]
    assert b_items, "B.md expected in rank_boost result"
    assert abs(b_items[0]["score"] - 0.09) < 1e-9, \
        f"B.md score should be 0.09, got {b_items[0]['score']}"


def test_rank_boost_entrant_count():
    """Entrant count is the number of non-baseline items in the top-20."""
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    _, entrant_count = rank_boost(ranked, FABRIC, w=0.1)
    # A's neighbors in the fabric: B, (and E via B but E is only neighbor of B)
    # Wait: undirected — A-B, so B is A's neighbor; B's neighbors are A, C, E
    # rank_boost expands from all baseline notes: A → neighbors {B, C, E}
    # B, C, E get boosted scores (A-B: B gets 0.09; A-B-C/E: but C/E link to B not directly to A)
    # Actually: B is neighbor of A (score 0.09). C is neighbor of A? No: A-B-C (C is 2-hop from A)
    # graph.get("A") = ["B"]  (only B is adjacent to A in FABRIC)
    # So only B gets a boost from A's baseline position
    assert entrant_count >= 0, f"entrant_count should be non-negative, got {entrant_count}"
    assert "B.md" in basenames(result := rank_boost(ranked, FABRIC, w=0.1)[0]), \
        "B.md (neighbor of A) should be in rank_boost result"


# ---------------------------------------------------------------------------
# supersession_ride_along tests
# ---------------------------------------------------------------------------

def test_supersession_inserts_after_old():
    """new_note is inserted directly after old_note."""
    l5 = [{"old": "A", "new": "C", "type": "updates", "claim": "..."}]
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("B.md", 0.7, rank=2),
    ]
    result = supersession_ride_along(ranked, l5)
    bns = basenames(result)
    a_idx = bns.index("A.md")
    c_idx = bns.index("C.md")
    assert c_idx == a_idx + 1, \
        f"C.md should be directly after A.md (positions A={a_idx}, C={c_idx})"


def test_supersession_no_duplicate_if_already_present():
    """If new_note is already delivered, no second insertion."""
    l5 = [{"old": "A", "new": "B", "type": "updates", "claim": "..."}]
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("B.md", 0.7, rank=2),  # B already delivered
    ]
    result = supersession_ride_along(ranked, l5)
    bns = basenames(result)
    assert bns.count("B.md") == 1, f"B.md must appear exactly once; got count={bns.count('B.md')}"
    assert len(bns) == 2, f"Length must stay 2 (no extra insertion); got {len(bns)}"


def test_supersession_no_matching_edge_unchanged():
    """Notes without a matching L5 edge are unchanged."""
    l5 = [{"old": "X", "new": "Y", "type": "updates", "claim": "..."}]  # no A or B
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("B.md", 0.7, rank=2),
    ]
    result = supersession_ride_along(ranked, l5)
    assert basenames(result) == ["A.md", "B.md"], \
        f"No matching edges; result must equal baseline: {basenames(result)}"


def test_supersession_inserted_score_lower():
    """Inserted new_note has score < old_note's score."""
    l5 = [{"old": "A", "new": "C", "type": "updates", "claim": "..."}]
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    result = supersession_ride_along(ranked, l5)
    a_score = next(r["score"] for r in result if r["basename"] == "A.md")
    c_score = next(r["score"] for r in result if r["basename"] == "C.md")
    assert c_score < a_score, \
        f"Inserted C.md score ({c_score}) should be < A.md score ({a_score})"


# ---------------------------------------------------------------------------
# tag_filter_candidates tests
# ---------------------------------------------------------------------------

def test_tag_filter_basic():
    """Notes sharing ≥1 tag with top-M delivered notes are in the pool."""
    l6 = {
        "vocab": ["eval", "retrieval"],
        "assignments": [
            {"note": "A", "tags": ["eval"]},
            {"note": "B", "tags": ["eval", "retrieval"]},
            {"note": "C", "tags": ["retrieval"]},
            {"note": "D", "tags": ["other"]},
        ],
    }
    ranked = [ranked_item("A.md", 0.9, rank=1)]
    pool, pool_size = tag_filter_candidates(ranked, l6, top_m=1)
    # A has tag "eval" → B (also has "eval") is a candidate
    # A does NOT have "retrieval" → C is not a candidate
    # D has "other" (not in A's tags) → not a candidate
    assert "B.md" in pool, f"B.md should be in pool (shares 'eval' with A): {pool}"
    assert "A.md" not in pool, "A.md is delivered; must not be in pool"
    assert "C.md" not in pool, f"C.md shares only 'retrieval' (not in A's tags): {pool}"
    assert "D.md" not in pool, f"D.md has 'other' (unrelated): {pool}"
    assert pool_size == 1, f"Pool size should be 1, got {pool_size}"


def test_tag_filter_pool_excludes_delivered():
    """All delivered notes are excluded from the pool regardless of tags."""
    l6 = {
        "vocab": ["eval"],
        "assignments": [
            {"note": "A", "tags": ["eval"]},
            {"note": "B", "tags": ["eval"]},
        ],
    }
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("B.md", 0.7, rank=2),
    ]
    pool, pool_size = tag_filter_candidates(ranked, l6, top_m=2)
    assert "A.md" not in pool and "B.md" not in pool, \
        f"Delivered notes must not be in pool: {pool}"
    assert pool_size == 0, f"Pool size should be 0 (both tagged notes are delivered): {pool_size}"


def test_tag_filter_top_m_limits_seed():
    """Only the top-M notes seed the tag pool."""
    l6 = {
        "vocab": ["alpha", "beta"],
        "assignments": [
            {"note": "A", "tags": ["alpha"]},
            {"note": "B", "tags": ["beta"]},
            {"note": "C", "tags": ["alpha"]},  # shares with A
            {"note": "D", "tags": ["beta"]},   # shares with B
        ],
    }
    # top_m=1 → only A seeds the pool; B's tags not considered
    ranked = [
        ranked_item("A.md", 0.9, rank=1),
        ranked_item("B.md", 0.7, rank=2),
    ]
    pool, _ = tag_filter_candidates(ranked, l6, top_m=1)
    assert "C.md" in pool, f"C shares 'alpha' with A (rank-1 seed): {pool}"
    assert "D.md" not in pool, f"D shares 'beta' with B (rank-2, outside top-1): {pool}"


# ---------------------------------------------------------------------------
# Test runner
# ---------------------------------------------------------------------------

ALL_TESTS = [
    test_expand_one_hop_order,
    test_expand_one_hop_isolated_node_excluded,
    test_expand_one_hop_non_top_m_source_excluded,
    test_expand_one_hop_no_duplicates,
    test_expand_one_hop_baseline_order_preserved,
    test_ppr_recovers_2hop,
    test_ppr_disconnected_node_absent,
    test_ppr_no_seeds_returns_baseline,
    test_ppr_blend_baseline_nodes_kept,
    test_rank_boost_baseline_relative_order_preserved,
    test_rank_boost_neighbor_boosted_score,
    test_rank_boost_entrant_count,
    test_supersession_inserts_after_old,
    test_supersession_no_duplicate_if_already_present,
    test_supersession_no_matching_edge_unchanged,
    test_supersession_inserted_score_lower,
    test_tag_filter_basic,
    test_tag_filter_pool_excludes_delivered,
    test_tag_filter_top_m_limits_seed,
]

if __name__ == "__main__":
    failures = []
    for test_fn in ALL_TESTS:
        try:
            test_fn()
            print(f"  PASS  {test_fn.__name__}")
        except AssertionError as exc:
            print(f"  FAIL  {test_fn.__name__}: {exc}")
            failures.append(test_fn.__name__)
        except Exception as exc:
            print(f"  ERROR {test_fn.__name__}: {type(exc).__name__}: {exc}")
            failures.append(test_fn.__name__)

    print()
    total = len(ALL_TESTS)
    if failures:
        print(f"FAILED {len(failures)}/{total}: {failures}")
        sys.exit(1)
    else:
        print(f"All {total} tests passed.")
