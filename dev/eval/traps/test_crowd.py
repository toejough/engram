"""TDD for the crowd generator: deterministic variants, link re-pointing, knobs, read-only source."""
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))
import crowd
import retrieval_probe as rp

SRC = [{"slug": "77.x", "luhmann": "77", "type": "fact", "situation": "s",
        "fields": {"subject": "http requests in Go", "predicate": "use", "object": "NewRequestWithContext"},
        "links": ["91.y"]},
       {"slug": "91.y", "luhmann": "91", "type": "fact", "situation": "s2",
        "fields": {"subject": "logging", "predicate": "use", "object": "slog"}, "links": []}]


def test_make_variants_count_unique_deterministic():
    a = crowd.make_variants(SRC, n=5, seed=7)
    b = crowd.make_variants(SRC, n=5, seed=7)
    assert len(a) == 5 and len({v["slug"] for v in a}) == 5
    assert [v["slug"] for v in a] == [v["slug"] for v in b]


def test_links_repoint_to_sibling_variants_or_drop():
    v = crowd.make_variants(SRC, n=4, seed=7)
    slugs = {x["slug"] for x in v}
    for x in v:
        for link in x["links"]:
            assert link in slugs                      # never dangling-outside-crowd


def test_vocab_knob_biases_toward_matching_notes():
    v = crowd.make_variants(SRC, n=10, seed=7, vocab_terms=["http"], vocab_frac=0.5)
    hits = sum(1 for x in v if "http" in x["fields"].get("subject", "").lower())
    assert hits >= 4


def test_recency_knob_marks_some_newer():
    v = crowd.make_variants(SRC, n=10, seed=7, recency_frac=0.5)
    assert 1 <= sum(1 for x in v if x["newer"]) <= 9


def test_seed_into_refuses_real_vault(tmp_path):
    import pytest
    with pytest.raises(RuntimeError):
        crowd.seed_into(crowd.real_vault(), [])


def test_rank_in_payload_found_and_absent():
    p = {"items": [{"path": "v/other.md"}, {"path": "v/target.md"}]}
    assert rp.rank_in_payload(p, "target") == {"surfaced": True, "rank": 2}
    assert rp.rank_in_payload({"items": [{"path": "v/x.md"}]}, "target") == {"surfaced": False, "rank": None}
