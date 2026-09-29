"""
Tests for run_audit.py -- the thin pipeline runner wrapping the existing
audit stages (extract -> audit -> scorecard) for a future before/after re-run.

All stage functions are stubbed/monkeypatched: no real API calls, no real
corpus enumeration (fixture dirs only), no writes outside tmp_path.
"""

import json
import os
import sys
from pathlib import Path

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import audit_moments
import run_audit


# ---------------------------------------------------------------------------
# Helpers / fixtures
# ---------------------------------------------------------------------------


def write_config(tmp_path: Path, overrides: dict) -> Path:
    """Write a config JSON file with the given overrides and return its path."""
    cfg_path = tmp_path / "config.json"
    cfg_path.write_text(json.dumps(overrides))
    return cfg_path


@pytest.fixture
def out_dir(tmp_path):
    return tmp_path / "out"


@pytest.fixture
def base_config(tmp_path, out_dir):
    """A minimal valid config dict pointing output at a tmp dir."""
    return {
        "sample_size": 2,
        "stages": ["extract", "audit"],
        "output_dir": str(out_dir),
        "model_tiers": {"detect": "haiku", "judge": "sonnet"},
        "seed": 42,
    }


@pytest.fixture
def fixture_projects_root(tmp_path):
    """A fake ~/.claude/projects layout: main + subagent + workflow + excluded."""
    root = tmp_path / "projects"
    repo = root / "-Users-fake-repo"
    (repo / "sess1" / "subagents" / "workflows").mkdir(parents=True)
    (repo / "sess1.jsonl").write_text('{"type":"user"}\n')
    (repo / "sess1" / "subagents" / "agent-a.jsonl").write_text('{"type":"user"}\n')
    (repo / "sess1" / "subagents" / "agent-a.meta.json").write_text("{}")
    (repo / "sess1" / "subagents" / "workflows" / "wf-1.jsonl").write_text(
        '{"type":"user"}\n'
    )
    # Ephemeral eval dir -- must be excluded from enumeration
    ephemeral = root / "-private-tmp-cummatrix-real-ws"
    ephemeral.mkdir(parents=True)
    (ephemeral / "x.jsonl").write_text('{"type":"user"}\n')
    return root


# ---------------------------------------------------------------------------
# Config parsing
# ---------------------------------------------------------------------------


class TestConfigParsing:
    def test_empty_config_gets_all_defaults(self, tmp_path):
        cfg_path = write_config(tmp_path, {})
        cfg = run_audit.load_config(str(cfg_path))
        assert cfg["sample_size"] == run_audit.DEFAULT_CONFIG["sample_size"]
        assert cfg["stages"] == ["extract", "audit", "scorecard"]
        assert cfg["model_tiers"] == {"detect": "haiku", "judge": "sonnet"}
        assert isinstance(cfg["seed"], int)
        assert cfg["output_dir"]  # non-empty

    def test_partial_override_keeps_other_defaults(self, tmp_path):
        cfg_path = write_config(tmp_path, {"sample_size": 5})
        cfg = run_audit.load_config(str(cfg_path))
        assert cfg["sample_size"] == 5
        assert cfg["stages"] == ["extract", "audit", "scorecard"]

    def test_missing_config_file_fails_loud(self, tmp_path):
        with pytest.raises(FileNotFoundError):
            run_audit.load_config(str(tmp_path / "nope.json"))

    def test_unknown_key_rejected(self, tmp_path):
        cfg_path = write_config(tmp_path, {"samplesize": 5})
        with pytest.raises(ValueError, match="samplesize"):
            run_audit.load_config(str(cfg_path))

    def test_invalid_stage_rejected(self, tmp_path):
        cfg_path = write_config(tmp_path, {"stages": ["extract", "bogus"]})
        with pytest.raises(ValueError, match="bogus"):
            run_audit.load_config(str(cfg_path))

    def test_output_dir_may_not_be_frozen_results(self, tmp_path):
        cfg_path = write_config(
            tmp_path, {"output_dir": str(run_audit.FROZEN_RESULTS_DIR)}
        )
        with pytest.raises(ValueError, match="frozen"):
            run_audit.load_config(str(cfg_path))

    def test_output_dir_may_not_be_inside_frozen_results(self, tmp_path):
        cfg_path = write_config(
            tmp_path, {"output_dir": str(run_audit.FROZEN_RESULTS_DIR / "sub")}
        )
        with pytest.raises(ValueError, match="frozen"):
            run_audit.load_config(str(cfg_path))

    def test_default_output_dir_is_not_frozen_results(self, tmp_path):
        cfg_path = write_config(tmp_path, {})
        cfg = run_audit.load_config(str(cfg_path))
        resolved = Path(cfg["output_dir"]).resolve()
        assert resolved != run_audit.FROZEN_RESULTS_DIR.resolve()

    def test_model_tiers_must_match_instrument_pins(self, tmp_path):
        # audit_moments.audit_transcript hardcodes haiku detect / sonnet judge;
        # a config asking for anything else must fail loud, not silently run
        # with tiers the instrument won't use.
        cfg_path = write_config(
            tmp_path, {"model_tiers": {"detect": "sonnet", "judge": "opus"}}
        )
        with pytest.raises(ValueError, match="model_tiers"):
            run_audit.load_config(str(cfg_path))

    def test_relative_output_dir_resolves_against_audit_dir(self, tmp_path):
        cfg_path = write_config(tmp_path, {"output_dir": "results-rerun-x"})
        cfg = run_audit.load_config(str(cfg_path))
        assert Path(cfg["output_dir"]).is_absolute()
        assert Path(cfg["output_dir"]) == run_audit.AUDIT_DIR / "results-rerun-x"


# ---------------------------------------------------------------------------
# Estimate mode arithmetic
# ---------------------------------------------------------------------------


class TestEstimateArithmetic:
    def test_compute_estimate_arithmetic(self):
        cost_records = {
            "rejudge-all-nulls.jsonl": [0.10, 0.20, 0.30],  # mean 0.20
            "relevance-cues.jsonl": [0.40],  # mean 0.40
        }
        est = run_audit.compute_estimate(
            sample_size=10,
            cost_records_by_file=cost_records,
            baseline_moments=6,
            baseline_transcripts=4,
        )
        assert est["sample_size"] == 10
        assert est["baseline"]["moments"] == 6
        assert est["baseline"]["audited_transcripts"] == 4
        assert est["baseline"]["moments_per_transcript"] == pytest.approx(1.5)
        files = est["cost_files"]
        assert files["rejudge-all-nulls.jsonl"]["n_records"] == 3
        assert files["rejudge-all-nulls.jsonl"]["mean_usd"] == pytest.approx(0.20)
        assert files["relevance-cues.jsonl"]["n_records"] == 1
        # pooled: (0.6 + 0.4) / 4 records = 0.25
        assert est["pooled_mean_usd_per_moment_call"] == pytest.approx(0.25)
        # expected moments: 10 transcripts x 1.5 = 15
        assert est["expected_moments"] == pytest.approx(15.0)
        # total: 15 x 0.25 = 3.75
        assert est["estimated_total_usd"] == pytest.approx(3.75)

    def test_compute_estimate_rejects_empty_cost_records(self):
        # An empty/missing cost basis must raise, never silently estimate $0
        # (eval inputs fail loud, no silent fallback).
        with pytest.raises(ValueError, match="cost"):
            run_audit.compute_estimate(
                sample_size=10,
                cost_records_by_file={"rejudge-all-nulls.jsonl": []},
                baseline_moments=6,
                baseline_transcripts=4,
            )

    def test_estimate_from_results_dir_reads_fixture_cost_files(self, tmp_path):
        results = tmp_path / "results"
        results.mkdir()
        # Two cost files with known cost_usd fields (one null -- must be skipped)
        (results / "rejudge-all-nulls.jsonl").write_text(
            json.dumps({"moment_id": "a#1", "cost_usd": 0.5})
            + "\n"
            + json.dumps({"moment_id": "a#2", "cost_usd": None})
            + "\n"
        )
        (results / "relevance-cues.jsonl").write_text(
            json.dumps({"moment_id": "a#1", "cost_usd": 0.3}) + "\n"
        )
        # Baseline moment corpus: 3 moments
        (results / "moments.jsonl").write_text(
            "".join(json.dumps({"moment_id": f"m#{i}"}) + "\n" for i in range(3))
        )
        # Baseline audited transcripts: 1 + 1 = 2 across the events files
        (results / "pilot-transcript-events.jsonl").write_text(
            json.dumps({"transcript_path": "/t/p1.jsonl"}) + "\n"
        )
        (results / "full-run-transcript-events.jsonl").write_text(
            json.dumps({"transcript_path": "/t/f1.jsonl"}) + "\n"
        )
        (results / "refill-transcript-events.jsonl").write_text("")

        est = run_audit.estimate_from_results_dir(sample_size=4, results_dir=results)
        assert est["baseline"]["moments"] == 3
        assert est["baseline"]["audited_transcripts"] == 2
        # per-call: (0.5 + 0.3) / 2 = 0.4; moments/transcript = 1.5
        assert est["pooled_mean_usd_per_moment_call"] == pytest.approx(0.4)
        assert est["expected_moments"] == pytest.approx(6.0)
        assert est["estimated_total_usd"] == pytest.approx(2.4)

    def test_format_estimate_prints_basis(self):
        est = run_audit.compute_estimate(
            sample_size=10,
            cost_records_by_file={
                "rejudge-all-nulls.jsonl": [0.10, 0.30],
                "relevance-cues.jsonl": [0.20],
            },
            baseline_moments=3,
            baseline_transcripts=2,
        )
        text = run_audit.format_estimate(est)
        assert "rejudge-all-nulls.jsonl" in text
        assert "relevance-cues.jsonl" in text
        assert "USD" in text or "$" in text
        # The basis (real recorded per-call averages) must be stated, not bare totals
        assert "mean" in text.lower() or "average" in text.lower()


# ---------------------------------------------------------------------------
# Corpus enumeration + sampling determinism
# ---------------------------------------------------------------------------


class TestEnumerationAndSampling:
    def test_enumerate_corpus_finds_all_jsonl_conventions(self, fixture_projects_root):
        paths = run_audit.enumerate_corpus(str(fixture_projects_root))
        names = [os.path.basename(p) for p in paths]
        assert "sess1.jsonl" in names
        assert "agent-a.jsonl" in names
        assert "wf-1.jsonl" in names
        # meta.json sidecars are not transcripts
        assert "agent-a.meta.json" not in names

    def test_enumerate_corpus_excludes_ephemeral_private_tmp_dirs(
        self, fixture_projects_root
    ):
        paths = run_audit.enumerate_corpus(str(fixture_projects_root))
        assert not any("-private-tmp" in p for p in paths)

    def test_enumerate_corpus_is_sorted(self, fixture_projects_root):
        paths = run_audit.enumerate_corpus(str(fixture_projects_root))
        assert paths == sorted(paths)

    def test_sampling_deterministic_under_seed(self):
        corpus = [f"/t/{i:03d}.jsonl" for i in range(20)]
        s1 = run_audit.sample_corpus(corpus, sample_size=5, seed=99)
        s2 = run_audit.sample_corpus(corpus, sample_size=5, seed=99)
        assert s1 == s2
        assert len(s1) == 5
        assert set(s1) <= set(corpus)

    def test_sampling_differs_across_seeds(self):
        corpus = [f"/t/{i:03d}.jsonl" for i in range(20)]
        s1 = run_audit.sample_corpus(corpus, sample_size=5, seed=1)
        s2 = run_audit.sample_corpus(corpus, sample_size=5, seed=2)
        assert s1 != s2

    def test_sampling_independent_of_input_order(self):
        corpus = [f"/t/{i:03d}.jsonl" for i in range(20)]
        s1 = run_audit.sample_corpus(corpus, sample_size=5, seed=7)
        s2 = run_audit.sample_corpus(list(reversed(corpus)), sample_size=5, seed=7)
        assert s1 == s2

    def test_sample_size_larger_than_corpus_takes_all(self):
        corpus = ["/t/a.jsonl", "/t/b.jsonl"]
        s = run_audit.sample_corpus(corpus, sample_size=10, seed=1)
        assert sorted(s) == sorted(corpus)

    def test_enumerate_corpus_excludes_dev_eval_harness_dirs(self, fixture_projects_root):
        # Eval-harness trial project dirs (the audit's own session dir, pressure-test
        # fixture sessions) are not natural usage -- the named exclusion rule.
        for name in (
            "-Users-fake-repo-dev-eval-audit",
            "-Users-fake-repo--claude-worktrees-x-dev-eval-cumulative-taskBF-RED-red1",
        ):
            d = fixture_projects_root / name
            d.mkdir(parents=True)
            (d / "y.jsonl").write_text('{"type":"user"}\n')
        paths = run_audit.enumerate_corpus(str(fixture_projects_root))
        assert not any("-dev-eval-" in p for p in paths)
        # real repo transcripts still enumerated
        assert any(p.endswith("sess1.jsonl") for p in paths)
        assert "-dev-eval-" in run_audit.EXCLUDED_PROJECT_DIR_SUBSTRINGS
        assert "-private-tmp" in run_audit.EXCLUDED_PROJECT_DIR_PREFIXES


def _write_transcript(path: Path, lines):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps(line) + "\n" for line in lines))


class TestExclusionRules:
    """Named re-measure exclusion rules (2026-09-28, Joe's option 1 + corpus rule)."""

    def _subagent(self, root, session, agent, prompt, description=None):
        path = root / "-Users-fake-repo" / session / "subagents" / f"agent-{agent}.jsonl"
        _write_transcript(
            path,
            [{"type": "user", "timestamp": "2026-09-10T00:00:00Z",
              "message": {"role": "user", "content": prompt}}],
        )
        if description is not None:
            (path.parent / f"agent-{agent}.meta.json").write_text(
                json.dumps({"agentType": "general-purpose", "description": description})
            )
        return str(path)

    def _main(self, root, session, entrypoint, cwd, prompt="hi"):
        path = root / "-Users-fake-repo" / f"{session}.jsonl"
        _write_transcript(
            path,
            [{"type": "user", "timestamp": "2026-09-10T00:00:00Z", "entrypoint": entrypoint,
              "cwd": cwd, "message": {"role": "user", "content": prompt}}],
        )
        return str(path)

    def test_excluded_session_matches_main_and_all_its_subagents(self, tmp_path):
        sid = "f6dfe139-52a1-4f7d-aee0-2bb16645d7ff"
        assert sid in run_audit.EXCLUDED_SESSION_IDS
        main = self._main(tmp_path, sid, "cli", "/Users/joe")
        sub = self._subagent(tmp_path, sid, "a1", "do work")
        other = self._main(tmp_path, "other-session", "cli", "/Users/joe")
        assert run_audit.exclusion_reason(main) == "excluded_session"
        assert run_audit.exclusion_reason(sub) == "excluded_session"
        assert run_audit.exclusion_reason(other) is None

    def test_pressure_test_subject_by_dispatch_description(self, tmp_path):
        for i, desc in enumerate(
            ["RED baseline: write-memory handoff", "GREEN rep 1: recall citation",
             "Pressure test: recall must refuse", "Rate-limit probe 2"]
        ):
            p = self._subagent(tmp_path, "s1", f"d{i}", "work", description=desc)
            assert run_audit.exclusion_reason(p) == "eval_subject_description", desc
        # dev work ABOUT an eval harness is natural usage, not a test subject
        for i, desc in enumerate(
            ["Build harness + run 2.1 RED baseline", "Scope task 2.1 RED baseline eval",
             "Review phase-2 Task 3 (probe_phase2)"]
        ):
            p = self._subagent(tmp_path, "s2", f"k{i}", "work", description=desc)
            assert run_audit.exclusion_reason(p) is None, desc

    def test_pressure_test_subject_by_roleplay_prompt(self, tmp_path):
        p = self._subagent(
            tmp_path, "s3", "r1", "You are role-playing as an agent executing the recall skill"
        )
        assert run_audit.exclusion_reason(p) == "eval_subject_prompt"
        p = self._subagent(tmp_path, "s3", "r2", 'You are acting as the "write-memory" skill')
        assert run_audit.exclusion_reason(p) == "eval_subject_prompt"

    def test_headless_eval_arm_main_session(self, tmp_path):
        arm = self._main(
            tmp_path, "arm1", "sdk-cli",
            "/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill",
        )
        assert run_audit.exclusion_reason(arm) == "headless_eval_arm"
        # headless session outside an engram checkout (e.g. phone-llm relay) is kept
        phone = self._main(tmp_path, "ph1", "sdk-cli", "/Users/joe/repos/personal/local-llm-optimization")
        assert run_audit.exclusion_reason(phone) is None
        # interactive session inside engram is kept
        inter = self._main(tmp_path, "in1", "cli", "/Users/joe/repos/personal/engram")
        assert run_audit.exclusion_reason(inter) is None

    def test_automated_security_review_main_session(self, tmp_path):
        prompt = "Review this change for security vulnerabilities.\n\nChanged files: x"
        review = self._main(tmp_path, "rv1", "sdk-py", "/Users/joe/repos/personal/phone-llm", prompt)
        assert run_audit.exclusion_reason(review) == "automated_security_review"
        # same entrypoint, different prompt: kept
        other = self._main(tmp_path, "rv2", "sdk-py", "/Users/joe/repos/personal/phone-llm", "hello")
        assert run_audit.exclusion_reason(other) is None
        # same prompt typed interactively: kept
        typed = self._main(tmp_path, "rv3", "cli", "/Users/joe", prompt)
        assert run_audit.exclusion_reason(typed) is None

    def test_filter_exclusions_counts_per_rule(self, tmp_path):
        keep = self._main(tmp_path, "k", "cli", "/Users/joe")
        drop = self._subagent(tmp_path, "s4", "x", "w", description="RED baseline: x")
        kept, counts = run_audit.filter_exclusions([drop, keep])
        assert kept == [keep]
        assert counts["eval_subject_description"] == 1
        assert sum(counts.values()) == 1


class TestSinceFilter:
    def test_first_timestamp_skips_untimestamped_lines(self, tmp_path):
        p = tmp_path / "t.jsonl"
        _write_transcript(
            p,
            [{"type": "summary"}, {"type": "user", "timestamp": "2026-09-08T01:02:03Z"}],
        )
        assert run_audit.first_timestamp(str(p)) == "2026-09-08T01:02:03Z"

    def test_first_timestamp_none_when_absent(self, tmp_path):
        p = tmp_path / "t.jsonl"
        _write_transcript(p, [{"type": "summary"}])
        assert run_audit.first_timestamp(str(p)) is None

    def test_filter_keeps_on_or_after_since_by_first_timestamp(self, tmp_path):
        before = tmp_path / "before.jsonl"
        on = tmp_path / "on.jsonl"
        after = tmp_path / "after.jsonl"
        undated = tmp_path / "undated.jsonl"
        # started before the cutoff, continued after it: still excluded (FIRST timestamp)
        _write_transcript(
            before,
            [
                {"timestamp": "2026-09-06T23:59:59Z"},
                {"timestamp": "2026-09-10T00:00:00Z"},
            ],
        )
        _write_transcript(on, [{"timestamp": "2026-09-07T00:00:00Z"}])
        _write_transcript(after, [{"timestamp": "2026-09-20T12:00:00Z"}])
        _write_transcript(undated, [{"type": "summary"}])
        kept, dropped = run_audit.filter_since(
            [str(before), str(on), str(after), str(undated)], "2026-09-07"
        )
        assert kept == sorted([str(on), str(after)])
        assert dropped == {"before_since": 1, "no_timestamp": 1}

    def test_config_since_defaults_to_none_and_validates(self, tmp_path):
        cfg = run_audit.load_config(str(write_config(tmp_path, {})))
        assert cfg["since"] is None
        cfg = run_audit.load_config(str(write_config(tmp_path, {"since": "2026-09-07"})))
        assert cfg["since"] == "2026-09-07"
        with pytest.raises(ValueError):
            run_audit.load_config(str(write_config(tmp_path, {"since": "Sept 7"})))

    def test_run_applies_since_before_sampling_and_records_it(
        self, monkeypatch, base_config, out_dir, tmp_path
    ):
        old = tmp_path / "c" / "old.jsonl"
        new1 = tmp_path / "c" / "new1.jsonl"
        new2 = tmp_path / "c" / "new2.jsonl"
        _write_transcript(old, [{"timestamp": "2026-09-01T00:00:00Z"}])
        _write_transcript(new1, [{"timestamp": "2026-09-08T00:00:00Z"}])
        _write_transcript(new2, [{"timestamp": "2026-09-09T00:00:00Z"}])
        make_stub_stages(monkeypatch, [str(old), str(new1), str(new2)])
        config = {**base_config, "since": "2026-09-07", "sample_size": 5}
        run_audit.run(config)
        sample = json.loads((out_dir / "sample.json").read_text())
        assert sorted(sample["sample"]) == sorted([str(new1), str(new2)])
        meta = sample["metadata"]
        assert meta["since"] == "2026-09-07"
        assert meta["corpus_size_before_since"] == 3
        assert meta["corpus_size"] == 2
        assert meta["dropped_by_since"] == {"before_since": 1, "no_timestamp": 0}


# ---------------------------------------------------------------------------
# Run mode: resume-skip, manifest, halt
# ---------------------------------------------------------------------------


def make_stub_stages(monkeypatch, corpus, moments_by_path=None, audit_raises=None):
    """Monkeypatch enumerate_corpus + stage functions; record calls."""
    calls = {"extract": [], "audit": [], "scorecard": []}
    moments_by_path = moments_by_path or {}

    monkeypatch.setattr(run_audit, "enumerate_corpus", lambda root=None: list(corpus))

    def extract_stage(path):
        calls["extract"].append(path)
        return {"transcript_path": path, "dispatch_type": "main", "recall_calls": []}

    def audit_stage(path):
        calls["audit"].append(path)
        if audit_raises and path in audit_raises:
            raise audit_raises[path]
        return moments_by_path.get(
            path, [{"moment_id": f"{os.path.basename(path)}#1", "transcript_path": path}]
        )

    def scorecard_stage(moments_path, output_dir):
        calls["scorecard"].append(str(moments_path))
        return {"status": "stubbed"}

    monkeypatch.setattr(run_audit, "extract_stage", extract_stage)
    monkeypatch.setattr(run_audit, "audit_stage", audit_stage)
    monkeypatch.setattr(run_audit, "scorecard_stage", scorecard_stage)
    return calls


class TestRunMode:
    def test_resume_skips_transcripts_already_recorded(
        self, monkeypatch, base_config, out_dir
    ):
        corpus = ["/t/a.jsonl", "/t/b.jsonl"]
        calls = make_stub_stages(monkeypatch, corpus)
        # Pre-record transcript a as done (the established driver convention:
        # done-set is read from the per-transcript output file).
        out_dir.mkdir(parents=True)
        events_path = out_dir / "transcript-events.jsonl"
        events_path.write_text(
            json.dumps({"transcript_path": "/t/a.jsonl", "dispatch_type": "main"}) + "\n"
        )
        # Pin the sample so it covers both transcripts deterministically.
        (out_dir / "sample.json").write_text(
            json.dumps({"metadata": {"seed": 42}, "sample": corpus})
        )

        run_audit.run(base_config)

        assert calls["extract"] == ["/t/b.jsonl"]
        assert calls["audit"] == ["/t/b.jsonl"]
        # Events file now records both transcripts, a exactly once.
        recorded = [
            json.loads(l)["transcript_path"]
            for l in events_path.read_text().splitlines()
            if l.strip()
        ]
        assert recorded == ["/t/a.jsonl", "/t/b.jsonl"]

    def test_run_writes_sample_and_incremental_outputs(
        self, monkeypatch, base_config, out_dir
    ):
        corpus = ["/t/a.jsonl", "/t/b.jsonl", "/t/c.jsonl"]
        moments = {
            "/t/a.jsonl": [
                {"moment_id": "a.jsonl#1", "transcript_path": "/t/a.jsonl"},
                {"moment_id": "a.jsonl#2", "transcript_path": "/t/a.jsonl"},
            ],
        }
        make_stub_stages(monkeypatch, corpus, moments_by_path=moments)
        base_config["sample_size"] = 3

        run_audit.run(base_config)

        sample = json.loads((out_dir / "sample.json").read_text())
        assert sorted(sample["sample"]) == sorted(corpus)
        assert sample["metadata"]["seed"] == 42
        moment_lines = [
            json.loads(l)
            for l in (out_dir / "moments.jsonl").read_text().splitlines()
            if l.strip()
        ]
        ids = [m["moment_id"] for m in moment_lines]
        assert "a.jsonl#1" in ids and "a.jsonl#2" in ids

    def test_existing_sample_reused_without_enumeration(
        self, monkeypatch, base_config, out_dir
    ):
        # On restart the persisted sample is the sample -- the live corpus may
        # have changed between invocations, so re-enumerating would break resume.
        corpus = ["/t/a.jsonl"]
        calls = make_stub_stages(monkeypatch, corpus)

        def boom(root=None):
            raise AssertionError("enumerate_corpus must not run when sample.json exists")

        monkeypatch.setattr(run_audit, "enumerate_corpus", boom)
        out_dir.mkdir(parents=True)
        (out_dir / "sample.json").write_text(
            json.dumps({"metadata": {"seed": 42}, "sample": corpus})
        )

        run_audit.run(base_config)
        assert calls["audit"] == ["/t/a.jsonl"]

    def test_manifest_content(self, monkeypatch, base_config, out_dir):
        corpus = ["/t/a.jsonl", "/t/b.jsonl"]
        moments = {
            "/t/a.jsonl": [
                {
                    "moment_id": "a.jsonl#1",
                    "transcript_path": "/t/a.jsonl",
                    "cost_usd": 0.25,
                }
            ],
            "/t/b.jsonl": [
                {
                    "moment_id": "b.jsonl#1",
                    "transcript_path": "/t/b.jsonl",
                    "cost_usd": 0.5,
                }
            ],
        }
        make_stub_stages(monkeypatch, corpus, moments_by_path=moments)
        base_config["stages"] = ["extract", "audit", "scorecard"]

        run_audit.run(base_config)

        manifest = json.loads((out_dir / "run-manifest.json").read_text())
        assert manifest["config"]["sample_size"] == base_config["sample_size"]
        assert manifest["config"]["seed"] == 42
        assert manifest["instrument_version"] == getattr(
            audit_moments, "INSTRUMENT_VERSION", None
        )
        assert "started_at" in manifest and "finished_at" in manifest
        assert manifest["per_stage_costs"]["audit_usd"] == pytest.approx(0.75)
        outputs = manifest["outputs"]
        assert outputs["moments"] == str(out_dir / "moments.jsonl")
        assert outputs["transcript_events"] == str(out_dir / "transcript-events.jsonl")
        assert outputs["sample"] == str(out_dir / "sample.json")
        assert manifest["status"] == "completed"
        assert manifest["scorecard"] == {"status": "stubbed"}

    def test_exhausted_retries_halts_immediately(
        self, monkeypatch, base_config, out_dir
    ):
        corpus = ["/t/a.jsonl", "/t/b.jsonl", "/t/c.jsonl"]
        calls = make_stub_stages(
            monkeypatch,
            corpus,
            audit_raises={
                "/t/b.jsonl": audit_moments.ExhaustedRetriesError("api down")
            },
        )
        base_config["sample_size"] = 3
        # Pin sample order so b is hit second.
        out_dir.mkdir(parents=True)
        (out_dir / "sample.json").write_text(
            json.dumps({"metadata": {"seed": 42}, "sample": corpus})
        )

        with pytest.raises(audit_moments.ExhaustedRetriesError):
            run_audit.run(base_config)

        # c must never have been attempted -- immediate halt, not log-and-continue.
        assert calls["audit"] == ["/t/a.jsonl", "/t/b.jsonl"]
        # a's completed work is preserved on disk for resume.
        recorded = [
            json.loads(l)["transcript_path"]
            for l in (out_dir / "transcript-events.jsonl").read_text().splitlines()
            if l.strip()
        ]
        assert recorded == ["/t/a.jsonl"]
        # Manifest records the halt.
        manifest = json.loads((out_dir / "run-manifest.json").read_text())
        assert manifest["status"] == "halted_exhausted_retries"

    def test_stages_respected_extract_only(self, monkeypatch, base_config, out_dir):
        corpus = ["/t/a.jsonl"]
        calls = make_stub_stages(monkeypatch, corpus)
        base_config["stages"] = ["extract"]
        base_config["sample_size"] = 1

        run_audit.run(base_config)

        assert calls["extract"] == ["/t/a.jsonl"]
        assert calls["audit"] == []
        assert calls["scorecard"] == []


# ---------------------------------------------------------------------------
# Scorecard stage wrapper (real one -- no subprocess in tests)
# ---------------------------------------------------------------------------


class TestScorecardStage:
    def test_scorecard_stage_refuses_pinned_baseline_mismatch(
        self, monkeypatch, tmp_path
    ):
        # build_scorecard.py pins SRC/OUT to the frozen results/ baseline.
        # For any re-run output dir the wrapper must refuse to invoke it
        # (invoking would re-read the frozen moments and write into results/),
        # and must say so loudly instead of silently "succeeding".
        def no_subprocess(*args, **kwargs):
            raise AssertionError("subprocess must not run for a pinned-path mismatch")

        monkeypatch.setattr(run_audit.subprocess, "run", no_subprocess)
        result = run_audit.scorecard_stage(
            moments_path=tmp_path / "moments.jsonl", output_dir=tmp_path
        )
        assert result["status"] == "skipped_pinned_baseline"
        assert "build_scorecard.py" in result["reason"]
