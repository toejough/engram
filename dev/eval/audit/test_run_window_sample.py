"""Tests for run_window_sample.py -- stubbed, no API calls, writes only under tmp_path."""

import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import audit_moments
import run_window_sample as rws


def _fake_split(counts):
    def split(path):
        return [
            {"location_start": i * 10 + 1, "location_end": i * 10 + 10, "window_index": i}
            for i in range(counts[path])
        ]
    return split


def test_choose_windows_deterministic_and_in_range():
    counts = {f"/t/{i}.jsonl": (i % 5) + 1 for i in range(20)}
    sample = sorted(counts)
    a = rws.choose_windows(sample, 739, split=_fake_split(counts))
    b = rws.choose_windows(sample, 739, split=_fake_split(counts))
    assert a == b
    for item in a:
        assert 0 <= item["window_index"] < counts[item["transcript_path"]]
        assert item["total_windows"] == counts[item["transcript_path"]]
    # single-window transcripts always get window 0
    assert all(i["window_index"] == 0 for i in a if i["total_windows"] == 1)


def test_audit_one_window_passes_only_chosen_window(monkeypatch):
    counts = {"/t/x.jsonl": 3}
    monkeypatch.setattr(audit_moments, "split_into_windows", _fake_split(counts))
    seen = {}

    def fake_audit(path):
        seen["windows"] = audit_moments.split_into_windows(path)
        return [{"moment_id": "x#1"}]

    monkeypatch.setattr(audit_moments, "audit_transcript", fake_audit)
    item = {"transcript_path": "/t/x.jsonl", "window_index": 1, "total_windows": 3,
            "location_start": 11, "location_end": 20}
    assert rws.audit_one_window(item) == [{"moment_id": "x#1"}]
    assert [w["location_start"] for w in seen["windows"]] == [11]
    # the module binding is restored afterwards
    assert len(audit_moments.split_into_windows("/t/x.jsonl")) == 3


def test_audit_one_window_fails_loud_on_window_drift(monkeypatch):
    monkeypatch.setattr(audit_moments, "split_into_windows", _fake_split({"/t/x.jsonl": 2}))
    item = {"transcript_path": "/t/x.jsonl", "window_index": 1, "total_windows": 2,
            "location_start": 99, "location_end": 120}
    with pytest.raises(RuntimeError, match="window drift"):
        rws.audit_one_window(item)


def test_projected_total_includes_prior_spend():
    assert rws.projected_total(17.65, 10.0, 4, 47) == pytest.approx(17.65 + 2.5 * 47)
    assert rws.projected_total(5.0, 0.0, 0, 47) == 5.0
