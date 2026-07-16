import run_underload_repro as h


def _stub_hermetic(monkeypatch):
    """Stub the keychain/config-dir subprocess seams (matrix.build_cfg_template,
    matrix.refresh_creds) plus the paid turn (h.call_turn — the bare module-level name, the
    correct seam per Gate A) so a trial reaches build_trial_cwd and the note-194 delivery gate
    at $0 API cost, short-circuiting invalid at turn 1."""
    monkeypatch.setattr(h.matrix, "build_cfg_template", lambda *a, **k: None)
    monkeypatch.setattr(h.matrix, "refresh_creds", lambda *a, **k: None)
    monkeypatch.setattr(h, "call_turn", lambda *a, **k: {"is_error": True, "total_cost_usd": 0.0, "result": ""})


def test_guidance_flag_threads_to_trial_cwd(tmp_path, monkeypatch):
    treatment = "## TREATMENT-MARKER-recall-cue\n\nrun /recall glance first."
    gpath = tmp_path / "treat.md"; gpath.write_text(treatment)
    captured = {}
    real_build = h.build_trial_cwd
    def spy(guidance_text, marker):
        captured["g"] = guidance_text
        return real_build(guidance_text, marker)
    monkeypatch.setattr(h, "build_trial_cwd", spy)
    # make the test hermetic: run_one_trial runs matrix.build_cfg_template / matrix.refresh_creds
    # (keychain subprocess) BEFORE build_trial_cwd — stub them so the test doesn't depend on
    # ~/.claude or the keychain, and reaches the spy by ASSERTION rather than erroring out.
    _stub_hermetic(monkeypatch)
    h.main(["--guidance", str(gpath), "--fixtures", "fixture1_beacon_relay",
            "--n", "1", "--workers", "1", "--out", str(tmp_path / "o.jsonl")])
    assert "TREATMENT-MARKER-recall-cue" in captured["g"]   # threading proven on the real path


def test_headingless_guidance_still_delivered_and_gated(tmp_path, monkeypatch):
    """Regression for the note-194 treatment-delivery gate defect: the gate used to key off the
    first markdown heading and silently SKIP itself entirely when guidance_text had none. A
    heading-less guidance_text (e.g. a bare bullet, like the Task 2 draft cue) must still be
    inlined into the trial CLAUDE.md AND must still pass through the (now heading-agnostic)
    delivery gate without raising — i.e. the gate runs and finds the probe present, rather than
    never running at all."""
    treatment = "- run /recall glance before committing. HEADLESS-TREATMENT-MARKER-9f2c"
    gpath = tmp_path / "treat.md"; gpath.write_text(treatment)
    captured = {}
    real_build = h.build_trial_cwd
    def spy(guidance_text, marker):
        captured["g"] = guidance_text
        return real_build(guidance_text, marker)
    monkeypatch.setattr(h, "build_trial_cwd", spy)
    _stub_hermetic(monkeypatch)
    h.main(["--guidance", str(gpath), "--fixtures", "fixture1_beacon_relay",
            "--n", "1", "--workers", "1", "--out", str(tmp_path / "o.jsonl")])
    assert "HEADLESS-TREATMENT-MARKER-9f2c" in captured["g"]   # delivered; gate ran, did not raise


def test_guidance_probe_picks_first_nonblank_line_when_headingless():
    """Direct unit test on the delivery-gate probe helper: for guidance with no markdown heading,
    the probe must be the first non-blank line verbatim, not None (the pre-fix behavior — a None
    probe is exactly what silently disabled the delivery gate for heading-less treatments)."""
    guidance_text = "\n\n  - a bare bullet cue, no heading at all\nmore text below\n"
    assert h._guidance_probe(guidance_text) == "- a bare bullet cue, no heading at all"
