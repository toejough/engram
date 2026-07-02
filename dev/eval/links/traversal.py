"""
traversal.py — Pure traversal functions for the link-value PoC probe (S2).

No I/O — all functions take data structures and return transformed lists.
Fabric edge dicts use basenames WITHOUT .md extension.
Ranked-note dicts use basenames WITH .md extension.
All functions normalize internally via _strip_md / _add_md.
"""
from __future__ import annotations

from collections import defaultdict
from typing import Any


# ---------------------------------------------------------------------------
# Basename normalization helpers
# ---------------------------------------------------------------------------

def _strip_md(basename: str) -> str:
    """Remove .md suffix if present."""
    return basename[:-3] if basename.endswith(".md") else basename


def _add_md(basename: str) -> str:
    """Ensure .md suffix."""
    return basename if basename.endswith(".md") else basename + ".md"


# ---------------------------------------------------------------------------
# Chunk pinning (S2b harness correction)
# ---------------------------------------------------------------------------

def _is_chunk(item: dict) -> bool:
    """True if a ranked item is a chunk (pinned; never re-ranked by traversals)."""
    return item.get("kind") == "chunk"


def _rebuild_with_pinned_chunks(
    baseline: list[dict],
    note_sequence: list[dict],
) -> list[dict]:
    """Rebuild a ranked list with chunks pinned at their baseline positions.

    S2b harness correction (chunk-pinned): traversals may only re-order/insert
    NOTE items (kind fact/feedback). Chunk items keep their absolute baseline
    indices — the list is treated as slots: chunks are pinned, notes compete
    among themselves for the remaining slots + appended positions.

    If the note sequence exhausts before reaching a chunk's pinned index, the
    remaining chunks compact earlier (a list cannot hold gaps); with enough
    notes, every chunk stays at its exact baseline position.
    """
    chunk_positions = [
        (idx, item) for idx, item in enumerate(baseline) if _is_chunk(item)
    ]
    result: list[dict] = []
    chunk_i = 0
    note_i = 0
    pos = 0
    while chunk_i < len(chunk_positions) or note_i < len(note_sequence):
        if chunk_i < len(chunk_positions) and (
            pos >= chunk_positions[chunk_i][0] or note_i >= len(note_sequence)
        ):
            result.append(chunk_positions[chunk_i][1])
            chunk_i += 1
        else:
            result.append(note_sequence[note_i])
            note_i += 1
        pos += 1
    return result


# ---------------------------------------------------------------------------
# Graph construction
# ---------------------------------------------------------------------------

def _build_undirected_graph(fabric: list[dict]) -> dict[str, list[str]]:
    """Build undirected adjacency list from a fabric edge list.

    Supports two edge schemas:
      {'src': ..., 'dst': ...}   (L1, L2, L3, L4, L7)
      {'old': ..., 'new': ...}   (L5)

    Node names are stripped of .md. Self-loops and empty node names are skipped.
    """
    graph: dict[str, list[str]] = defaultdict(list)
    for edge in fabric:
        src_raw = edge.get("src") or edge.get("old") or ""
        dst_raw = edge.get("dst") or edge.get("new") or ""
        src = _strip_md(src_raw)
        dst = _strip_md(dst_raw)
        if not src or not dst or src == dst:
            continue
        if dst not in graph[src]:
            graph[src].append(dst)
        if src not in graph[dst]:
            graph[dst].append(src)
    return dict(graph)


# ---------------------------------------------------------------------------
# T1 / T6: one-hop payload expansion
# ---------------------------------------------------------------------------

def expand_one_hop(
    ranked: list[dict],
    fabric: list[dict],
    top_m: int = 5,
) -> list[dict]:
    """Append 1-hop neighbors of the top-M ranked notes (T1/T6 traversal).

    Ordering: original ranked list first; new neighbors appended in
    (source-rank ascending, then adjacency-list order); deduped across the whole result.
    Each appended item carries a ``via`` key naming the source note.

    Args:
        ranked: list of {basename, score, kind, rank, ...} note dicts.
        fabric: list of edge dicts without .md on node names.
        top_m: how many top ranked notes to expand neighbors from.

    Returns:
        New list (originals unchanged + neighbors appended).
    """
    graph = _build_undirected_graph(fabric)
    already: set[str] = {_strip_md(r["basename"]) for r in ranked}

    appended: list[dict] = []
    seen_new: set[str] = set()

    for item in ranked[:top_m]:
        node = _strip_md(item["basename"])
        for neighbor in graph.get(node, []):
            if neighbor not in already and neighbor not in seen_new:
                seen_new.add(neighbor)
                appended.append({
                    "basename": _add_md(neighbor),
                    "score": 0.0,
                    "kind": None,
                    "rank": None,
                    "via": item["basename"],
                })

    return list(ranked) + appended


# ---------------------------------------------------------------------------
# T2: Personalized PageRank (PPR)
# ---------------------------------------------------------------------------

def _run_ppr(
    seeds: dict[str, float],
    graph: dict[str, list[str]],
    alpha: float = 0.5,
    max_iter: int = 200,
    tol: float = 1e-8,
) -> dict[str, float]:
    """Power-iteration PPR; returns {node: ppr_score} for all graph nodes.

    Iteration: v^{t+1} = alpha * M^T * v^t + (1-alpha) * p
    where M is the column-stochastic random-walk matrix and p is the
    normalized personalization vector built from `seeds`.
    """
    all_nodes = sorted(graph.keys())
    if not all_nodes:
        return {}

    node_idx = {n: i for i, n in enumerate(all_nodes)}
    num_nodes = len(all_nodes)

    # Personalization vector
    p = [0.0] * num_nodes
    for node, weight in seeds.items():
        if node in node_idx:
            p[node_idx[node]] = max(weight, 0.0)
    total_p = sum(p)
    if total_p == 0.0:
        return {}
    p = [x / total_p for x in p]

    degrees = {n: len(nbrs) for n, nbrs in graph.items()}

    v = [1.0 / num_nodes] * num_nodes
    for _ in range(max_iter):
        v_new = [0.0] * num_nodes
        for j_node, nbrs in graph.items():
            j = node_idx[j_node]
            deg_j = degrees.get(j_node, 0)
            if deg_j == 0:
                continue
            contrib = alpha * v[j] / deg_j
            for nbr in nbrs:
                if nbr in node_idx:
                    v_new[node_idx[nbr]] += contrib

        diff = 0.0
        for i in range(num_nodes):
            v_new[i] += (1.0 - alpha) * p[i]
            diff += abs(v_new[i] - v[i])
        v = v_new
        if diff < tol:
            break

    return {n: v[node_idx[n]] for n in all_nodes}


def ppr_rank(
    ranked_scores: list[dict],
    fabric: list[dict],
    alpha: float = 0.5,
    idf_weight: bool = True,
    rescale_c: float = 0.4,
    act_tau: float = 0.5,
) -> tuple[list[dict], int]:
    """T2 primary: Pure PPR ranking seeded by cosine-matched notes in the fabric.

    Personalization weight = cosine_score / node_degree (when idf_weight=True) —
    the HippoRAG node-specificity / beat-4 hub suppressor.
    Activation threshold act_tau (relative to max PPR score) bounds the result.
    Scores are multiplied by rescale_c (SA-RAG anti-flood scaling).

    S2b (chunk-pinned): NOTE-ONLY re-ranking. Only fact/feedback items are
    seeded and re-ranked; chunk items keep their baseline positions. Baseline
    NOTES not activated by PPR are absent from the note sequence (HippoRAG's
    actual formula — rank ALL activated nodes, only activated nodes).

    Returns:
        (result_list, newly_activated_count) where newly_activated_count is the
        count of activated nodes not present in the input baseline.
    """
    graph = _build_undirected_graph(fabric)
    if not graph:
        return list(ranked_scores), 0

    degrees = {n: len(nbrs) for n, nbrs in graph.items()}
    baseline_notes = [r for r in ranked_scores if not _is_chunk(r)]

    seeds: dict[str, float] = {}
    for item in baseline_notes:
        node = _strip_md(item["basename"])
        if node in graph:
            raw_score = item["score"] if item["score"] is not None else 0.0
            weight = raw_score
            if idf_weight:
                deg = degrees.get(node, 1)
                weight = raw_score / deg if deg > 0 else raw_score
            seeds[node] = weight

    if not seeds:
        return list(ranked_scores), 0

    raw_ppr = _run_ppr(seeds, graph, alpha=alpha)
    if not raw_ppr:
        return list(ranked_scores), 0

    max_ppr = max(raw_ppr.values())
    if max_ppr == 0.0:
        return list(ranked_scores), 0

    threshold = act_tau * max_ppr
    activated = [
        (node, score * rescale_c)
        for node, score in raw_ppr.items()
        if score >= threshold
    ]
    activated.sort(key=lambda x: -x[1])

    baseline_set = {_strip_md(r["basename"]) for r in baseline_notes}
    note_sequence = [
        {"basename": _add_md(node), "score": ppr_score, "kind": None, "rank": None}
        for node, ppr_score in activated
    ]
    new_count = sum(1 for node, _ in activated if node not in baseline_set)
    result = _rebuild_with_pinned_chunks(ranked_scores, note_sequence)
    return result, new_count


def ppr_blend(
    ranked_scores: list[dict],
    fabric: list[dict],
    alpha: float = 0.5,
    idf_weight: bool = True,
    rescale_c: float = 0.4,
    act_tau: float = 0.5,
    sim_w: float = 1.0,
    ppr_w: float = 0.1,
) -> tuple[list[dict], int]:
    """T2 secondary blend: score = sim_w * sim + ppr_w * ppr (GAAMA anchor).

    sim = cosine score from baseline (0 for notes absent from baseline).
    ppr = rescale_c-scaled PPR score (0 for notes not activated).
    All baseline NOTES are included; activated non-baseline nodes are added.

    S2b (chunk-pinned): NOTE-ONLY re-ranking. Only fact/feedback items are
    seeded and blended; chunk items keep their baseline positions.

    Returns:
        (result_list, newly_activated_count)
    """
    graph = _build_undirected_graph(fabric)
    degrees = {n: len(nbrs) for n, nbrs in graph.items()}
    baseline_notes = [r for r in ranked_scores if not _is_chunk(r)]

    seeds: dict[str, float] = {}
    for item in baseline_notes:
        node = _strip_md(item["basename"])
        if node in graph:
            raw_score = item["score"] if item["score"] is not None else 0.0
            weight = raw_score
            if idf_weight:
                deg = degrees.get(node, 1)
                weight = raw_score / deg if deg > 0 else raw_score
            seeds[node] = weight

    scaled_ppr: dict[str, float] = {}
    new_count = 0

    if seeds and graph:
        raw_ppr = _run_ppr(seeds, graph, alpha=alpha)
        max_ppr = max(raw_ppr.values()) if raw_ppr else 0.0
        if max_ppr > 0.0:
            threshold = act_tau * max_ppr
            scaled_ppr = {
                node: score * rescale_c
                for node, score in raw_ppr.items()
                if score >= threshold
            }
            baseline_set = {_strip_md(r["basename"]) for r in baseline_notes}
            new_count = sum(1 for node in scaled_ppr if node not in baseline_set)

    sim_map = {
        _strip_md(r["basename"]): (r["score"] if r["score"] is not None else 0.0)
        for r in baseline_notes
    }

    all_nodes = set(sim_map.keys()) | set(scaled_ppr.keys())
    note_sequence = []
    for node in all_nodes:
        blend_score = sim_w * sim_map.get(node, 0.0) + ppr_w * scaled_ppr.get(node, 0.0)
        note_sequence.append(
            {"basename": _add_md(node), "score": blend_score, "kind": None, "rank": None}
        )

    note_sequence.sort(key=lambda x: -x["score"])
    result = _rebuild_with_pinned_chunks(ranked_scores, note_sequence)
    return result, new_count


# ---------------------------------------------------------------------------
# T3: neighbor rank-boost (re-rank only; edges never add payload beyond entrants)
# ---------------------------------------------------------------------------

def rank_boost(
    ranked_scores: list[dict],
    fabric: list[dict],
    w: float = 0.1,
) -> tuple[list[dict], int]:
    """T3: re-rank the union of baseline NOTES + their 1-hop neighbors.

    Baseline notes keep their cosine sim score.
    A non-delivered neighbor gets score = w × max(sim of its linked delivered notes).
    Entrant count = non-baseline notes that appear in the top-20 of the result.

    S2b (chunk-pinned): NOTE-ONLY re-ranking, per the plan's T3 definition
    ("re-rank buried/below-floor NOTES upward"). Chunk items keep their
    baseline positions; a boosted note may only displace other notes.

    Relative order of baseline notes is preserved among themselves as long as no
    neighbor's boosted score exceeds a baseline note's sim score.

    Returns:
        (re_ranked_list, entrant_count_at_top_20)
    """
    graph = _build_undirected_graph(fabric)
    baseline_notes = [r for r in ranked_scores if not _is_chunk(r)]
    baseline_map: dict[str, dict] = {_strip_md(r["basename"]): r for r in baseline_notes}

    neighbor_max_sim: dict[str, float] = defaultdict(float)
    for item in baseline_notes:
        node = _strip_md(item["basename"])
        sim = item["score"] if item["score"] is not None else 0.0
        for nbr in graph.get(node, []):
            if nbr not in baseline_map:
                if sim > neighbor_max_sim[nbr]:
                    neighbor_max_sim[nbr] = sim

    union = list(baseline_notes)
    for nbr, max_sim in neighbor_max_sim.items():
        union.append({
            "basename": _add_md(nbr),
            "score": w * max_sim,
            "kind": None,
            "rank": None,
        })

    union.sort(key=lambda x: -(x["score"] if x["score"] is not None else 0.0))

    result = _rebuild_with_pinned_chunks(ranked_scores, union)
    entrant_count = sum(
        1 for item in result[:20]
        if not _is_chunk(item) and _strip_md(item["basename"]) not in baseline_map
    )
    return result, entrant_count


# ---------------------------------------------------------------------------
# T5: supersession ride-along
# ---------------------------------------------------------------------------

def supersession_ride_along(
    ranked: list[dict],
    l5_fabric: list[dict],
) -> list[dict]:
    """T5: insert new_note immediately after each delivered old_note with a supersession edge.

    Uses 'old'/'new' keys from the L5 edge list. If new_note is already in
    the ranked list, no second insertion is made (dedup). The inserted item
    carries a ``via`` key referencing the old note, and its score is set to
    0.9 × old_note's score (slightly lower, so it can be identified as inserted).

    Args:
        ranked: baseline ranked list.
        l5_fabric: list of {old, new, type, claim} edges (no .md).

    Returns:
        Modified ranked list with superseders inserted after their superseded notes.
    """
    old_to_new: dict[str, str] = {}
    for edge in l5_fabric:
        old = _strip_md(edge.get("old") or edge.get("src") or "")
        new = _strip_md(edge.get("new") or edge.get("dst") or "")
        if old and new:
            old_to_new[old] = new

    added: set[str] = {_strip_md(r["basename"]) for r in ranked}
    result: list[dict] = []

    for item in ranked:
        result.append(item)
        node = _strip_md(item["basename"])
        new_node = old_to_new.get(node)
        if new_node and new_node not in added:
            added.add(new_node)
            result.append({
                "basename": _add_md(new_node),
                "score": (item["score"] * 0.9) if item["score"] is not None else 0.0,
                "kind": None,
                "rank": None,
                "via": item["basename"],
            })

    return result


# ---------------------------------------------------------------------------
# L6: tag-based candidate filter (T4-style nomination)
# ---------------------------------------------------------------------------

def tag_filter_candidates(
    ranked: list[dict],
    l6: dict,
    top_m: int = 3,
) -> tuple[list[str], int]:
    """L6: return the candidate pool of notes sharing ≥1 tag with the top-M delivered notes.

    This is a T4-style nomination: the pool is returned for candidate consideration,
    not injected into the ranked list.

    Args:
        ranked: baseline ranked list.
        l6: {'vocab': [...], 'assignments': [{'note': no-md-basename, 'tags': [...]}]}.
        top_m: number of top ranked notes whose tags seed the pool.

    Returns:
        (candidate_basenames_with_md sorted, pool_size)
    """
    assignments = l6.get("assignments", [])

    note_tags: dict[str, set[str]] = {}
    tag_notes: dict[str, set[str]] = defaultdict(set)
    for entry in assignments:
        note = _strip_md(entry["note"])
        tags = set(entry.get("tags", []))
        note_tags[note] = tags
        for tag in tags:
            tag_notes[tag].add(note)

    top_tags: set[str] = set()
    for item in ranked[:top_m]:
        node = _strip_md(item["basename"])
        top_tags |= note_tags.get(node, set())

    delivered: set[str] = {_strip_md(r["basename"]) for r in ranked}
    candidates: set[str] = set()
    for tag in top_tags:
        candidates |= tag_notes[tag]
    candidates -= delivered

    pool = sorted(_add_md(n) for n in candidates)
    return pool, len(pool)
