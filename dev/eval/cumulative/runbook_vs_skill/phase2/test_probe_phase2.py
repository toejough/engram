"""Unit tests for probe_phase2.py — the phase-2 conversion-parity eval harness.

Never calls claude. Covering-note removal / carrier-add tests use a synthetic vault dir (never the
operator's real vault) but read the real, committed encodings/ fixtures (safe — static repo files).
Transcript-scoring tests build synthetic event lists directly (mirroring phase-1's test_probe.py
style), matching the real Claude Code session-transcript shape probe.py's parse_transcript_events
already verified against a real transcript.
"""
import json
import os
import re

import pytest

import probe_phase2 as pp

HERE = os.path.dirname(os.path.abspath(__file__))


# ----- synthetic transcript helpers (mirrors phase-1's test_probe.py) -----

def _tool_use(name, input_, idx, tool_id="tu"):
    return {"idx": idx, "kind": "tool_use", "name": name, "input": input_, "id": f"{tool_id}{idx}"}


def _tool_result(idx, tool_use_idx, content, tool_id="tu"):
    return {"idx": idx, "kind": "tool_result", "tool_use_id": f"{tool_id}{tool_use_idx}", "content": content}


def _text_ev(text, idx):
    return {"idx": idx, "kind": "text", "text": text}


# ----- covering-note removal + carrier add (synthetic vault dir, never the real one) -----

def _make_synthetic_vault(tmp_path, extra_notes=()):
    vault = tmp_path / "vault"
    vault.mkdir()
    for base in pp.TASK_A_REMOVAL:
        (vault / f"{base}.md").write_text(f"removal note {base}")
        (vault / f"{base}.vec.json").write_text("{}")
    for base in pp.TASK_B_REMOVAL:
        (vault / f"{base}.md").write_text(f"removal note {base}")
        (vault / f"{base}.vec.json").write_text("{}")
    (vault / f"{pp.NOTE_830_BASENAME}.md").write_text("note 830 body")
    (vault / f"{pp.NOTE_830_BASENAME}.vec.json").write_text("{}")
    for base, content in extra_notes:
        (vault / f"{base}.md").write_text(content)
        (vault / f"{base}.vec.json").write_text("{}")
    return str(vault)


def test_remove_covering_notes_task_a_deletes_only_task_a_list(tmp_path):
    keep_base = "999.2026-01-01.unrelated-note"
    vault = _make_synthetic_vault(tmp_path, extra_notes=[(keep_base, "unrelated content")])
    pp.remove_covering_notes(vault, "A", "S")
    for base in pp.TASK_A_REMOVAL:
        assert not os.path.exists(os.path.join(vault, f"{base}.md"))
        assert not os.path.exists(os.path.join(vault, f"{base}.vec.json"))
    # Task B's list and note 830 are untouched by a Task A removal
    for base in pp.TASK_B_REMOVAL:
        assert os.path.exists(os.path.join(vault, f"{base}.md"))
    assert os.path.exists(os.path.join(vault, f"{pp.NOTE_830_BASENAME}.md"))
    # unrelated note survives
    assert os.path.exists(os.path.join(vault, f"{keep_base}.md"))


def test_remove_covering_notes_task_b_removes_830_except_for_arm_r(tmp_path):
    vault_non_r = _make_synthetic_vault(tmp_path)
    pp.remove_covering_notes(vault_non_r, "B", "F")
    for base in pp.TASK_B_REMOVAL:
        assert not os.path.exists(os.path.join(vault_non_r, f"{base}.md"))
    assert not os.path.exists(os.path.join(vault_non_r, f"{pp.NOTE_830_BASENAME}.md"))
    assert not os.path.exists(os.path.join(vault_non_r, f"{pp.NOTE_830_BASENAME}.vec.json"))


def test_remove_covering_notes_task_b_keeps_830_for_arm_r(tmp_path):
    vault_r = _make_synthetic_vault(tmp_path)
    pp.remove_covering_notes(vault_r, "B", "R")
    for base in pp.TASK_B_REMOVAL:
        assert not os.path.exists(os.path.join(vault_r, f"{base}.md"))
    # 830 survives for arm R — it IS the carrier
    assert os.path.exists(os.path.join(vault_r, f"{pp.NOTE_830_BASENAME}.md"))
    assert os.path.exists(os.path.join(vault_r, f"{pp.NOTE_830_BASENAME}.vec.json"))


def test_add_carrier_task_a_r_copies_encoding_note_and_returns_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    basename = pp.add_carrier(vault, "A", "R")
    assert basename == "1.2026-09-11.commit-conventional-message"
    assert os.path.exists(os.path.join(vault, basename + ".md"))
    assert os.path.exists(os.path.join(vault, basename + ".vec.json"))


def test_add_carrier_task_a_f_copies_encoding_note_and_returns_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    basename = pp.add_carrier(vault, "A", "F")
    assert basename == "1.2026-09-11.git-commit-skill-procedure"
    assert os.path.exists(os.path.join(vault, basename + ".md"))


def test_add_carrier_task_b_r_does_not_copy_but_returns_830_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    before_listing = sorted(os.listdir(vault))
    basename = pp.add_carrier(vault, "B", "R")
    assert basename == pp.NOTE_830_BASENAME
    # nothing was copied in — the listing is unchanged
    assert sorted(os.listdir(vault)) == before_listing


def test_add_carrier_s_and_rdirect_add_nothing(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    before_listing = sorted(os.listdir(vault))
    assert pp.add_carrier(vault, "A", "S") is None
    assert pp.add_carrier(vault, "A", "Rdirect") is None
    assert sorted(os.listdir(vault)) == before_listing


def test_add_carrier_n_adds_nothing_to_vault(tmp_path):
    """Arm N (no-instructions control): same background vault + covering-note removal as every
    other arm, but NO carrier — measures whether the task needs instructions at all."""
    vault = _make_synthetic_vault(tmp_path)
    before_listing = sorted(os.listdir(vault))
    assert pp.add_carrier(vault, "A", "N") is None
    assert pp.add_carrier(vault, "B", "N") is None
    assert sorted(os.listdir(vault)) == before_listing


# ----- eval-session note exclusion (luhmann >= EXCLUDE_LUHMANN_MIN), every arm and task -----

def _write_note(vault, basename, content="body"):
    with open(os.path.join(vault, basename + ".md"), "w") as f:
        f.write(content)
    with open(os.path.join(vault, basename + ".vec.json"), "w") as f:
        f.write("{}")


def test_remove_eval_session_notes_keeps_below_floor_removes_at_or_above_floor(tmp_path):
    vault = str(tmp_path / "vault")
    os.makedirs(vault)
    _write_note(vault, "100.2026-01-01.old-fact")
    _write_note(vault, "954.2026-09-08.just-below-floor")
    _write_note(vault, "955.2026-09-10.at-floor")
    _write_note(vault, "1200.2026-09-11.well-above-floor")
    _write_note(vault, "qa.2026-09-11.some-question")

    removed = pp.remove_eval_session_notes(vault, min_luhmann=955)

    assert removed == 2
    remaining_md = sorted(n for n in os.listdir(vault) if n.endswith(".md"))
    assert remaining_md == [
        "100.2026-01-01.old-fact.md",
        "954.2026-09-08.just-below-floor.md",
        "qa.2026-09-11.some-question.md",
    ]
    # sidecars removed alongside their notes
    assert not os.path.exists(os.path.join(vault, "955.2026-09-10.at-floor.vec.json"))
    assert not os.path.exists(os.path.join(vault, "1200.2026-09-11.well-above-floor.vec.json"))
    assert os.path.exists(os.path.join(vault, "954.2026-09-08.just-below-floor.vec.json"))
    assert os.path.exists(os.path.join(vault, "qa.2026-09-11.some-question.vec.json"))


def test_remove_eval_session_notes_removes_alpha_suffixed_ids_at_or_above_floor(tmp_path):
    """988a, 988a1, 988b (leading integer 988) and 955 (bare) are all >= 955 and must be removed;
    954z (leading integer 954) is below the floor and must be kept."""
    vault = str(tmp_path / "vault")
    os.makedirs(vault)
    _write_note(vault, "954z.2026-09-08.just-below-floor")
    _write_note(vault, "955.2026-09-10.at-floor")
    _write_note(vault, "988a.2026-09-12.trial-validity-gate")
    _write_note(vault, "988a1.2026-09-12.verified-means-the-gate-covers")
    _write_note(vault, "988b.2026-09-12.read-the-checker-body")

    removed = pp.remove_eval_session_notes(vault, min_luhmann=955)

    assert removed == 4
    remaining_md = sorted(n for n in os.listdir(vault) if n.endswith(".md"))
    assert remaining_md == ["954z.2026-09-08.just-below-floor.md"]


def test_remove_eval_session_notes_respects_custom_floor(tmp_path):
    vault = str(tmp_path / "vault")
    os.makedirs(vault)
    _write_note(vault, "100.2026-01-01.old-fact")
    _write_note(vault, "200.2026-01-02.also-old")
    removed = pp.remove_eval_session_notes(vault, min_luhmann=150)
    assert removed == 1
    assert os.path.exists(os.path.join(vault, "100.2026-01-01.old-fact.md"))
    assert not os.path.exists(os.path.join(vault, "200.2026-01-02.also-old.md"))


def test_leading_luhmann_number_parses_integer_prefix_and_ignores_qa():
    assert pp._leading_luhmann_number("955.2026-09-10.at-floor") == 955
    assert pp._leading_luhmann_number("1.2026-09-11.commit-conventional-message") == 1
    assert pp._leading_luhmann_number("qa.2026-09-11.some-question") is None


def test_leading_luhmann_number_parses_leading_integer_of_alpha_suffixed_ids():
    """988a, 988a1, 988b (internal/luhmann grammar: digits, then letters, then digits, ...) all
    carry the leading INTEGER component 988 — an alpha-suffixed id must not be treated as
    'not a plain integer' and skipped by the eval-session-notes floor (bug: a bare-agent
    transcript recalled note 988a, which should have been >= EXCLUDE_LUHMANN_MIN and removed)."""
    assert pp._leading_luhmann_number("988a.2026-09-12.some-slug") == 988
    assert pp._leading_luhmann_number("988a1.2026-09-12.some-slug") == 988
    assert pp._leading_luhmann_number("988b.2026-09-12.some-slug") == 988
    assert pp._leading_luhmann_number("954z.2026-09-08.just-below-floor") == 954


def test_setup_trial_vault_applies_eval_session_note_exclusion_for_every_arm_and_task(tmp_path, monkeypatch):
    """The exclusion must run inside setup_trial_vault (every arm, every task) — not just be a
    standalone function nobody calls. verify_vault_health is stubbed out: this test's fake
    REAL_VAULT has placeholder (non-real) .vec.json sidecars, so it is not meant to pass a real
    `engram embed status` check — that plumbing is covered separately by the dry-run against the
    real vault (see task-3-report.md)."""
    fake_real_vault = str(tmp_path / "real_vault")
    os.makedirs(fake_real_vault)
    _write_note(fake_real_vault, "100.2026-01-01.old-fact")
    _write_note(fake_real_vault, "960.2026-09-11.this-session-note")
    monkeypatch.setattr(pp, "REAL_VAULT", fake_real_vault)
    monkeypatch.setattr(pp, "verify_vault_health", lambda vault: {})

    for task, arm in (("A", "S"), ("A", "R"), ("B", "F"), ("B", "Rdirect")):
        trial_vault = str(tmp_path / f"trial_{task}_{arm}")
        env = {"ENGRAM_VAULT_PATH": trial_vault}
        pp.setup_trial_vault(env, task, arm)
        remaining_md = [n for n in os.listdir(trial_vault) if n.endswith(".md")]
        assert "960.2026-09-11.this-session-note.md" not in remaining_md, (task, arm)


def test_setup_trial_vault_scrubs_openspec_domain_for_opsx_tasks(tmp_path, monkeypatch):
    """opsx-propose and opsx-archive have no skill/runbook carrier of their own (bare-agent
    baseline fixtures), so their task.json removal_basenames is the ONLY thing that keeps the real
    vault's ~90 openspec/opsx notes out of the background vault. Without it a bare agent recalls
    and cites the real openspec workflow (note 996) and an 8/8 'no instructions needed' result is
    contamination, not a finding. Uses the REAL production vault as background (read-only
    copytree — note 956: reads are not the write hazard) but stubs verify_vault_health so the test
    doesn't depend on the engram binary being on PATH."""
    monkeypatch.setattr(pp, "verify_vault_health", lambda vault: {})
    pattern = re.compile(r"openspec|opsx", re.IGNORECASE)
    for task in ("opsx-propose", "opsx-archive"):
        trial_vault = str(tmp_path / f"trial_{task}")
        env = {"ENGRAM_VAULT_PATH": trial_vault}
        pp.setup_trial_vault(env, task, "N")
        offenders = []
        for name in os.listdir(trial_vault):
            if not name.endswith(".md"):
                continue
            basename = name[: -len(".md")]
            if basename.startswith("qa."):
                continue  # qa.* notes are handled elsewhere — out of scope for this removal list
            if pattern.search(basename):
                offenders.append(name)
                continue
            with open(os.path.join(trial_vault, name), encoding="utf-8", errors="replace") as f:
                if pattern.search(f.read()):
                    offenders.append(name)
        assert offenders == [], (task, offenders)


# ----- Rdirect CLAUDE.md: procedure section + marker -----

def test_build_claude_md_phase2_no_extra_matches_phase1_shape():
    marker = "RUNBOOK-VS-SKILL-PROBE2-abc12345"
    claude_md = pp.build_claude_md_phase2(marker, extra=None)
    assert f"PROBE-TOKEN: {marker}" in claude_md
    assert "## Project procedure" not in claude_md


def test_build_claude_md_phase2_rdirect_appends_procedure_section_verbatim():
    marker = "RUNBOOK-VS-SKILL-PROBE2-def67890"
    extra_text = "1. Do the thing.\n2. Verify the thing."
    claude_md = pp.build_claude_md_phase2(marker, extra=extra_text)
    assert f"PROBE-TOKEN: {marker}" in claude_md
    assert "## Project procedure" in claude_md
    assert extra_text in claude_md
    # the procedure section comes after the marker line (appended, not interleaved)
    assert claude_md.index(f"PROBE-TOKEN: {marker}") < claude_md.index("## Project procedure")


# ----- shim-only CLAUDE.md (task 2.3): shim.md verbatim + marker, no recall.md -----

def test_build_claude_md_shimonly_contains_shim_body_and_marker():
    marker = "RUNBOOK-VS-SKILL-PROBE2-shim0001"
    claude_md = pp.build_claude_md_shimonly(marker, extra=None)
    shim_source = open(pp.SHIM_GUIDANCE_PATH).read()
    assert shim_source.strip() in claude_md
    assert f"PROBE-TOKEN: {marker}" in claude_md
    # the marker line comes after the shim body (appended, not interleaved)
    assert claude_md.index(shim_source.strip()[:40]) < claude_md.index(f"PROBE-TOKEN: {marker}")


def test_build_claude_md_shimonly_excludes_recall_md_content():
    marker = "RUNBOOK-VS-SKILL-PROBE2-shim0002"
    claude_md = pp.build_claude_md_shimonly(marker, extra=None)
    recall_md = open(pp.p1.GUIDANCE_PATH).read()
    # recall.md's own cue text ("/recall glance") must NOT leak into a shim-only CLAUDE.md —
    # that's exactly the hybrid D0/D7 rule out.
    assert "/recall glance" not in claude_md
    assert "/recall glance" in recall_md  # sanity: confirms the negative assertion is meaningful


def test_build_claude_md_shimonly_appends_project_procedure_when_extra_given():
    marker = "RUNBOOK-VS-SKILL-PROBE2-shim0003"
    extra_text = "1. Do the thing.\n2. Verify the thing."
    claude_md = pp.build_claude_md_shimonly(marker, extra=extra_text)
    assert "## Project procedure" in claude_md
    assert extra_text in claude_md
    assert claude_md.index(f"PROBE-TOKEN: {marker}") < claude_md.index("## Project procedure")


def test_setup_trial_repo_shim_md_true_writes_shim_content(tmp_path):
    marker = "RUNBOOK-VS-SKILL-PROBE2-shim0004"
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "R", marker, shim_md=True)
    claude_md = open(os.path.join(repo_path, "CLAUDE.md")).read()
    assert "/recall glance" not in claude_md
    shim_source = open(pp.SHIM_GUIDANCE_PATH).read()
    assert shim_source.strip()[:40] in claude_md


def test_setup_trial_repo_shim_md_false_writes_recall_md_content_unchanged(tmp_path):
    marker = "RUNBOOK-VS-SKILL-PROBE2-shim0005"
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "R", marker, shim_md=False)
    claude_md = open(os.path.join(repo_path, "CLAUDE.md")).read()
    assert "/recall glance" in claude_md


def test_argparser_shim_md_flag_defaults_false_and_parses_true():
    ap = pp.build_argparser()
    assert ap.parse_args(["--task", "history-rewrite"]).shim_md is False
    assert ap.parse_args(["--task", "history-rewrite", "--shim-md"]).shim_md is True


# ----- --shim-only (task 4.1): thin alias for --noskills --add-recall-learn-runbooks --shim-md --

def test_argparser_shim_only_flag_defaults_false_and_parses_true():
    ap = pp.build_argparser()
    assert ap.parse_args(["--task", "history-rewrite"]).shim_only is False
    assert ap.parse_args(["--task", "history-rewrite", "--shim-only"]).shim_only is True


def test_apply_shim_only_alias_sets_all_three_underlying_flags():
    args = pp.build_argparser().parse_args(["--task", "history-rewrite", "--shim-only"])
    pp.apply_shim_only_alias(args)
    assert args.noskills is True
    assert args.add_recall_learn_runbooks is True
    assert args.shim_md is True


def test_apply_shim_only_alias_is_noop_when_flag_absent():
    args = pp.build_argparser().parse_args(["--task", "history-rewrite"])
    pp.apply_shim_only_alias(args)
    assert args.noskills is False
    assert args.add_recall_learn_runbooks is False
    assert args.shim_md is False


def test_shim_only_produces_identical_trial_setup_to_explicit_three_flags():
    """--shim-only alone must be indistinguishable, post-expansion, from passing the three
    underlying flags explicitly — the whole point of the shorthand (tasks.md 4.1 item 1)."""
    via_shorthand = pp.apply_shim_only_alias(
        pp.build_argparser().parse_args(["--task", "history-rewrite", "--shim-only"]))
    via_explicit = pp.apply_shim_only_alias(
        pp.build_argparser().parse_args(["--task", "history-rewrite", "--noskills",
                                          "--add-recall-learn-runbooks", "--shim-md"]))
    assert (via_shorthand.noskills, via_shorthand.add_recall_learn_runbooks, via_shorthand.shim_md) == (
        via_explicit.noskills, via_explicit.add_recall_learn_runbooks, via_explicit.shim_md)
    assert (via_shorthand.noskills, via_shorthand.add_recall_learn_runbooks, via_shorthand.shim_md) == (
        True, True, True)


def test_rdirect_procedure_text_task_a_reads_a_r_note_body_verbatim():
    text = pp.rdirect_procedure_text("A")
    assert "Check VCS type" in text
    assert "AI-Used: [claude]" in text
    # frontmatter fields are stripped
    assert "situation:" not in text


def test_rdirect_procedure_text_task_b_reads_real_830_note_body_verbatim():
    text = pp.rdirect_procedure_text("B")
    assert "git check-ignore" in text
    assert "situation:" not in text


# ----- S skill deployment path per task -----

def test_deploy_skill_task_a_places_directory_form_skill(tmp_path):
    """Round-3 fix: Claude Code's skill listing did not discover the flat .claude/skills/commit.md
    file the way it discovered the directory-form gitignore-narrowing skill (smoke run 1
    finding), so Task A's skill is deployed as .claude/skills/commit/SKILL.md instead. Body must
    still be byte-identical to the live repo skill file (the frontmatter already carries `name:
    commit`, so no injection is needed here — see the dedicated frontmatter-injection test)."""
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "A")
    dst = os.path.join(repo_path, ".claude", "skills", "commit", "SKILL.md")
    assert os.path.isfile(dst)
    assert open(dst).read() == open(pp.TASKS["A"]["skill_src"]).read()
    # the old flat-file path must NOT exist — this harness deploys the directory form only
    assert not os.path.exists(os.path.join(repo_path, ".claude", "skills", "commit.md"))


def test_skill_body_with_name_is_noop_when_name_already_present():
    text = "---\nname: commit\ndescription: x\n---\n\n# Commit\n\nbody text\n"
    assert pp._skill_body_with_name(text, "commit") == text


def test_skill_body_with_name_injects_when_absent():
    text = "---\ndescription: x\n---\n\n# Commit\n\nbody text\n"
    result = pp._skill_body_with_name(text, "commit")
    assert "name: commit" in result
    # body content after the frontmatter is unchanged
    assert result.endswith("\n\n# Commit\n\nbody text\n")
    # only one name key was added, not a duplicate
    assert result.count("name:") == 1


def test_skill_body_with_name_leaves_non_frontmatter_text_unchanged():
    text = "no frontmatter here, just body text\n"
    assert pp._skill_body_with_name(text, "commit") == text


def test_deploy_skill_task_b_places_skill_dir(tmp_path):
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "B")
    dst = os.path.join(repo_path, ".claude", "skills", "gitignore-narrowing", "SKILL.md")
    assert os.path.isfile(dst)
    assert "name: gitignore-narrowing" in open(dst).read()


def test_deploy_skill_generic_copies_only_skill_md_not_siblings(tmp_path):
    """Generic tasks (tdd-order, route) must deploy ONLY SKILL.md, never sibling files like
    price-table.md or tests/. Each carrier form carries the SAME text, so S arm deploys exactly
    one file — this test verifies siblings are excluded."""
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    # Create a skill_src directory with SKILL.md and sibling files
    skill_src = tmp_path / "skill_src"
    skill_src.mkdir()
    (skill_src / "SKILL.md").write_text("# Test Skill\n\nBody content")
    (skill_src / "sibling.md").write_text("This should not be deployed")
    (skill_src / "tests").mkdir()
    (skill_src / "tests" / "test.txt").write_text("Test file")

    # Set up a mock task in TASKS with the skill_src
    original_tasks = pp.TASKS
    try:
        pp.TASKS["generic_test"] = {
            "skill_src": str(skill_src),
            "skill_name": "test-skill",
        }
        pp.deploy_skill(repo_path, "generic_test")

        # Verify only SKILL.md was deployed
        dst_dir = os.path.join(repo_path, ".claude", "skills", "test-skill")
        assert os.path.isdir(dst_dir)
        assert os.path.isfile(os.path.join(dst_dir, "SKILL.md"))

        # Verify siblings were NOT deployed
        assert not os.path.exists(os.path.join(dst_dir, "sibling.md"))
        assert not os.path.exists(os.path.join(dst_dir, "tests"))

        # Verify the content is correct
        assert open(os.path.join(dst_dir, "SKILL.md")).read() == "# Test Skill\n\nBody content"
    finally:
        pp.TASKS = original_tasks


def test_deploy_skill_generic_raises_error_if_skill_md_missing(tmp_path):
    """Generic tasks must raise a clear error if skill_src doesn't contain SKILL.md."""
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    # Create a skill_src directory WITHOUT SKILL.md
    skill_src = tmp_path / "skill_src_no_skill"
    skill_src.mkdir()
    (skill_src / "something.md").write_text("No SKILL.md here")

    original_tasks = pp.TASKS
    try:
        pp.TASKS["generic_test_missing"] = {
            "skill_src": str(skill_src),
            "skill_name": "broken-skill",
        }
        with pytest.raises(RuntimeError, match="does not contain SKILL.md"):
            pp.deploy_skill(repo_path, "generic_test_missing")
    finally:
        pp.TASKS = original_tasks


# ----- mutating-step regex (smoke-run-1 finding: bare '>' matched '2>&1') -----

def _mut_ev(command, idx=0):
    return {"idx": idx, "kind": "tool_use", "name": "Bash", "input": {"command": command}, "id": "tu"}


def test_mutating_step_stderr_redirect_pipe_is_not_mutating():
    """engram ingest --auto 2>&1 | tail -5 — the recall skill's own diagnostic call — must NOT be
    treated as a mutation. Smoke run 1: this false-positive fired on turn 1 of every trial,
    masking FOUND for R/F even when the carrier was genuinely in the query result."""
    ev = _mut_ev("engram ingest --auto 2>&1 | tail -5")
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_echo_redirect_to_file_is_mutating():
    ev = _mut_ev("echo x > file")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_heredoc_redirect_to_gitignore_is_mutating():
    ev = _mut_ev("cat <<EOF > .gitignore")
    assert pp.is_first_mutating_step(ev, "B") is True


def test_mutating_step_git_commit_heredoc_is_mutating():
    ev = _mut_ev("git commit -m \"$(cat <<'EOF'")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_plain_read_only_bash_not_mutating():
    for command in ("git status", "git diff --staged", "git log --oneline -5", "ls -la",
                     "cat pkg/version.go", "2>/dev/null"):
        ev = _mut_ev(command)
        assert pp.is_first_mutating_step(ev, "A") is False, command


def test_mutating_step_append_redirect_after_fd_dup_is_still_mutating():
    """A real append AFTER an fd-duplication (e.g. `2>&1 >> out.log`) must still count — only the
    fd-dup/duplication redirect itself is excluded, not every redirect in the command."""
    ev = _mut_ev("some_cmd 2>&1 >> out.log")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_engram_query_phrase_containing_git_commit_words_not_mutating():
    """Smoke-run-1 finding #2: the recall skill's own retrieval phrasing can legitimately contain
    the words 'git commit' inside a quoted --phrase argument. This exact A-R smoke transcript
    command must not be treated as a mutation."""
    ev = _mut_ev(
        'engram query --lazy-chunks \\\n'
        '  --phrase "commit a version bump following project conventions" \\\n'
        '  --phrase "git commit message conventions for this repo" \\\n'
        '  --phrase "staging files for a version release commit"'
    )
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_git_commit_heredoc_still_mutating_despite_quote_stripping():
    """The real invocation verb ('git commit') sits BEFORE the first quote character, so quote-
    stripping must not blind the detector to a genuine commit."""
    ev = _mut_ev("git commit -m \"$(cat <<'EOF'")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_git_add_inside_echoed_quotes_not_mutating():
    ev = _mut_ev('echo "git add"')
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_engram_readonly_verbs_never_mutating_even_with_redirect_looking_text():
    for command in (
        'engram query --lazy-chunks --phrase "some > thing"',
        "engram show-chunk source#anchor",
        "engram activate --note 1.2026-01-01.some-note.md",
        "engram ingest --auto",
    ):
        ev = _mut_ev(command)
        assert pp.is_first_mutating_step(ev, "A") is False, command


# ----- restated_as_plan (task 4.1): restated the runbook's steps before the first mutation -----

def test_restated_as_plan_true_when_todowrite_with_two_items_precedes_mutation():
    events = [
        _tool_use("TodoWrite", {"todos": [{"content": "step 1"}, {"content": "step 2"}]}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is True


def test_restated_as_plan_true_when_assistant_text_lists_two_items_before_mutation():
    events = [
        _text_ev("Plan:\n1. Record the pre-rewrite tip.\n2. Run the rewrite.\n3. Push.", idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is True


def test_restated_as_plan_true_with_bulleted_list():
    events = [
        _text_ev("Plan:\n- record the tip\n- run the rewrite", idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is True


def test_restated_as_plan_false_when_mutation_with_no_restate_first():
    """Clean mutate-with-no-restate: the agent jumps straight to a mutating command with no
    TodoWrite and no enumerated list beforehand."""
    events = [
        _tool_use("Bash", {"command": "git status"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is False


def test_restated_as_plan_false_when_list_appears_only_after_the_mutation():
    """A plan restated AFTER the first mutating step doesn't count — the signal is specifically
    about restating BEFORE acting."""
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _text_ev("1. Did the thing.\n2. Verified the thing.", idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is False


def test_restated_as_plan_false_when_todowrite_has_only_one_item():
    events = [
        _tool_use("TodoWrite", {"todos": [{"content": "only one step"}]}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_restated_as_plan(events, "A") is False


def test_restated_as_plan_true_when_no_mutation_occurs_at_all():
    """No mutating step anywhere -> every event is 'before' it (mirrors score_found_phase2's own
    before() convention for a transcript with no mutation)."""
    events = [_text_ev("1. First.\n2. Second.", idx=0), _tool_use("Bash", {"command": "git status"}, idx=1)]
    assert pp.detect_restated_as_plan(events, "A") is True


# ----- question_stop (task 4.1): a legitimate clarity stop, never a failure (design.md D8) -----

def test_question_stop_true_when_ambiguity_text_ends_transcript_with_no_further_calls():
    """Question-stop-with-no-further-calls: the transcript simply ends right after the assistant
    poses its question — the clean, unambiguous case."""
    events = [
        _tool_use("Bash", {"command": "git status"}, idx=0),
        _text_ev("Should I use approach A or approach B here?", idx=1),
    ]
    assert pp.detect_question_stop(events, "A") is True


def test_question_stop_true_when_question_is_mid_paragraph_not_at_the_very_end():
    """Verified against a real 2.3 GREEN-round2 trial transcript: the actual clarifying question
    sat mid-paragraph ('...or is X acceptable here?'), followed by one more sentence that ends
    with a period, not '?'. The signal must not require the text BLOCK's own last character to
    be '?'."""
    events = [
        _tool_use("Bash", {"command": "git push --force-with-lease origin main"}, idx=0),
        _text_ev(
            "Two things before I call this done. First, a real gap: the reflog auto-expired. "
            "Second, is this remote a stand-in for a real GitHub-hosted repo, or is local-only "
            "the actual scope here? That determines whether a follow-up action is needed.",
            idx=1,
        ),
    ]
    assert pp.detect_question_stop(events, "A") is True


def test_question_stop_false_when_mutation_follows_the_ambiguity_text():
    """Deviate-without-stopping: the agent raises the same ambiguity but then goes on to mutate
    anyway, rather than actually waiting — never a real stop."""
    events = [
        _text_ev("I'm not sure whether to use approach A or B here.", idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    assert pp.detect_question_stop(events, "A") is False


def test_question_stop_false_when_no_ambiguity_ever_raised():
    events = [
        _tool_use("Bash", {"command": "git status"}, idx=0),
        _text_ev("Everything looks good, committing now.", idx=1),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    assert pp.detect_question_stop(events, "A") is False


def test_question_stop_true_when_ask_user_question_tool_follows_ambiguity_with_no_mutation():
    events = [
        _text_ev("Which approach would you like: A or B?", idx=0),
        _tool_use("AskUserQuestion", {"question": "A or B?"}, idx=1),
    ]
    assert pp.detect_question_stop(events, "A") is True


def test_question_stop_uses_the_last_ambiguity_mention_not_the_first():
    """An early rhetorical '?' followed by real work, then a genuine late stop, must still score
    True — the LAST ambiguity-bearing text block is what matters, not the first."""
    events = [
        _text_ev("Should I check the remote first? Let me look.", idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
        _text_ev("Actually, before I proceed: do you want me to also prune the remote?", idx=2),
    ]
    assert pp.detect_question_stop(events, "A") is True


# ----- FOUND: Arm S (skill: commit) -----

def test_found_s_true_when_commit_skill_before_first_mutation():
    events = [
        _tool_use("Skill", {"skill": "commit", "args": "commit the version bump"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is True
    assert idx == 0
    assert found_via == "skill_tool"


def test_found_s_false_when_commit_skill_after_first_mutation():
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _tool_use("Skill", {"skill": "commit", "args": "late invoke"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False
    assert idx is None
    assert found_via is None


def test_found_s_false_when_wrong_skill_name():
    events = [_tool_use("Skill", {"skill": "recall", "args": "x"}, idx=0)]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False
    assert found_via is None


def test_found_s_task_b_uses_gitignore_narrowing_skill_name():
    events = [
        _tool_use("Skill", {"skill": "gitignore-narrowing", "args": "narrow it"}, idx=0),
        _tool_use("Bash", {"command": "git add .gitignore"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("B", "S", events, carrier_basename=None)
    assert found is True
    assert found_via == "skill_tool"


# ----- FOUND: Arm R/F requires carrier basename in the query result -----

def test_found_r_true_when_query_result_contains_carrier_basename():
    carrier = "1.2026-09-11.commit-conventional-message"
    events = [
        _tool_use("Bash", {"command": "engram query --lazy-chunks --phrase \"commit conventions\""}, idx=0),
        _tool_result(1, 0, f"items:\n  - path: {carrier}.md\n    kind: runbook"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "R", events, carrier_basename=carrier)
    assert found is True
    assert idx == 0
    assert found_via == "engram_query"


def test_found_r_false_when_query_result_lacks_carrier_basename():
    carrier = "1.2026-09-11.commit-conventional-message"
    events = [
        _tool_use("Bash", {"command": "engram query --lazy-chunks --phrase \"commit conventions\""}, idx=0),
        _tool_result(1, 0, "items: []\n"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "R", events, carrier_basename=carrier)
    assert found is False
    assert idx is None
    assert found_via is None


def test_found_f_true_when_query_result_contains_carrier_basename():
    carrier = "1.2026-09-11.git-commit-skill-procedure"
    events = [
        _tool_use("Bash", {"command": "engram query --phrase x"}, idx=0),
        _tool_result(1, 0, f"{carrier}.md"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "F", events, carrier_basename=carrier)
    assert found is True
    assert found_via == "engram_query"


def test_found_rdirect_is_always_na():
    events = [_tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0)]
    found, idx, found_via = pp.score_found_phase2("A", "Rdirect", events, carrier_basename=None)
    assert found is None
    assert idx is None
    assert found_via is None


# ----- FOUND: Arm S file_read mechanism (Read/Bash-cat/Glob of SKILL.md) -----

def test_found_s_true_when_read_skill_md_before_first_mutation():
    events = [
        _tool_use("Read", {"file_path": "/foo/bar/.claude/skills/commit/SKILL.md"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is True
    assert idx == 0
    assert found_via == "file_read"


def test_found_s_true_when_bash_cat_skill_md_before_first_mutation():
    events = [
        _tool_use("Bash", {"command": "cat .claude/skills/commit/SKILL.md"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is True
    assert idx == 0
    assert found_via == "file_read"


def test_found_s_true_when_glob_skill_md_before_first_mutation():
    events = [
        _tool_use("Glob", {"pattern": "**/.claude/skills/commit/SKILL.md"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is True
    assert idx == 0
    assert found_via == "file_read"


def test_found_s_false_when_read_skill_md_after_first_mutation():
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _tool_use("Read", {"file_path": ".claude/skills/commit/SKILL.md"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False
    assert idx is None
    assert found_via is None


def test_found_s_false_when_read_wrong_skill_md():
    events = [
        _tool_use("Read", {"file_path": ".claude/skills/recall/SKILL.md"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    found, idx, found_via = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False
    assert found_via is None


# ----- steps.json evaluation: not_pattern and after -----

def test_evaluate_steps_not_pattern_rejects_git_add_dash_a():
    steps = [
        {"n": 1, "name": "stage explicit paths, not -A", "signal": "bash_regex",
         "pattern": r"git\s+add", "not_pattern": r"-A|git\s+add\s+\.$"},
    ]
    events = [
        _tool_use("Bash", {"command": "git add -A"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert k == 1
    assert all_ is True


def test_evaluate_steps_not_pattern_all_matches_disqualified_stays_false():
    steps = [
        {"n": 1, "name": "stage explicit paths, not -A", "signal": "bash_regex",
         "pattern": r"git\s+add", "not_pattern": r"-A"},
    ]
    events = [_tool_use("Bash", {"command": "git add -A"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False
    assert k == 0
    assert all_ is False


def test_evaluate_steps_after_requires_match_strictly_after_referenced_step():
    steps = [
        {"n": 1, "name": "check state", "signal": "bash_regex", "pattern": r"git\s+status"},
        {"n": 2, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add", "after": 1},
    ]
    # git add occurs BEFORE git status — step 2 must not count despite matching the pattern
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is False
    assert k == 1


def test_evaluate_steps_after_satisfied_when_match_follows_referenced_step():
    steps = [
        {"n": 1, "name": "check state", "signal": "bash_regex", "pattern": r"git\s+status"},
        {"n": 2, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add", "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git status"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True
    assert k == 2
    assert all_ is True


def test_evaluate_steps_after_same_event_later_match_position_is_satisfied():
    """Final-review finding (coordinator correction): a single compound Bash command can
    legitimately satisfy an earlier step and a later step at once (e.g.
    `git add X && git diff --cached --name-only`). Requiring a STRICTLY LATER EVENT index for
    `after` wrongly fails the later step here — the fix accepts a match in the SAME Bash event
    when this step's own match starts strictly after the referenced step's match position within
    that command string."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git add file.txt && git diff --cached --name-only"}, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True
    assert k == 2
    assert all_ is True


def test_evaluate_steps_after_same_event_earlier_match_position_not_satisfied():
    """Mirror case: when the 'verify' pattern's match position comes BEFORE the referenced step's
    match position in the SAME command, the after constraint must NOT be satisfied — same
    command, wrong order."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git diff --cached --name-only && git add file.txt"}, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is False
    assert k == 1


def test_evaluate_steps_after_same_event_scans_later_match_past_earlier_occurrence():
    """Second final-review finding: when the verify pattern occurs MULTIPLE times in one compound
    command, `after` must find a LATER occurrence, not bail out because the FIRST (leftmost)
    occurrence sits before the referenced step's match. Real shape (Task B trials B-S#0/1/2,
    B-F#1/3): an early 'git status' (unrelated to this step) precedes the 'git add' staging call,
    and a later 'git diff --cached'/'git status --short' follows it — the naive `search()`-based
    fix (round 2) picked the EARLY occurrence and wrongly failed; `finditer` must scan forward."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex",
         "pattern": r"git\s+(diff\s+--cached|status)", "after": 1},
    ]
    events = [
        _tool_use("Bash", {
            "command": ("git status --porcelain -uall && git add .gitignore scripts/build.sh "
                        "testdata/fixture.json && git diff --cached --name-only && "
                        "git status --short"),
        }, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True
    assert all_ is True


def test_evaluate_steps_after_same_event_no_later_occurrence_at_all_stays_false():
    """When the verify pattern's ONLY occurrence in the command precedes the referenced step's
    match, and no later occurrence exists anywhere in the command, the after constraint must stay
    unsatisfied — there is truly nothing to find after scanning forward."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {
            "command": "git diff --cached && git add .gitignore scripts/build.sh testdata/fixture.json",
        }, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is False
    assert k == 1


def test_evaluate_steps_after_unmatched_referenced_step_forces_dependent_false():
    """Second final-review ruling: if the referenced step never matched at all, the dependent
    'after' step must be FALSE, never vacuously True — an unmatched reference defaulting to
    min_idx=-1 would otherwise let the dependent step match ANYWHERE in the transcript, as if
    there were no ordering constraint. A 'verify' step with nothing matched to verify AFTER is
    not the procedure."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add\s+--\s"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git diff --cached --name-only"}, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False
    assert results["2"] is False
    assert k == 0


def test_evaluate_steps_repo_state_signal_uses_provided_checker():
    steps = [{"n": 1, "name": "custom repo check", "signal": "repo_state", "pattern": "always_true"}]

    def checker(pattern_name, repo_path):
        assert pattern_name == "always_true"
        assert repo_path == "/some/repo"
        return True

    results, k, all_ = pp.evaluate_steps(steps, [], repo_path="/some/repo", repo_checker=checker)
    assert results["1"] is True
    assert k == 1


# ----- tool_path signal + any_of (round 5: native Read/Edit/Write credit) -----

def test_evaluate_steps_tool_path_credits_native_read_on_matching_path():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert k == 1


def test_evaluate_steps_tool_path_false_when_wrong_tool():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Bash", {"command": "cat .gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_path_false_when_path_does_not_match():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Read", {"file_path": "/repo/README.md"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_path_matches_edit_and_write_too():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    for tool_name in ("Edit", "Write"):
        events = [_tool_use(tool_name, {"file_path": "/repo/.gitignore"}, idx=0)]
        results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
        assert results["1"] is True, tool_name


def test_evaluate_steps_any_of_satisfied_by_either_alternative():
    steps = [{
        "n": 1, "name": "inspect .gitignore",
        "any_of": [
            {"signal": "bash_regex", "pattern": r"cat\s+\.gitignore"},
            {"signal": "tool_path", "tools": ["Read"], "pattern": r"\.gitignore$"},
        ],
    }]
    # only the tool_path alternative is satisfied
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True

    # only the bash_regex alternative is satisfied
    events2 = [_tool_use("Bash", {"command": "cat .gitignore"}, idx=0)]
    results2, k2, all2_ = pp.evaluate_steps(steps, events2, repo_path="/does/not/matter")
    assert results2["1"] is True


def test_evaluate_steps_any_of_false_when_neither_alternative_matches():
    steps = [{
        "n": 1, "name": "inspect .gitignore",
        "any_of": [
            {"signal": "bash_regex", "pattern": r"cat\s+\.gitignore"},
            {"signal": "tool_path", "tools": ["Read"], "pattern": r"\.gitignore$"},
        ],
    }]
    events = [_tool_use("Bash", {"command": "git status"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_input_regex_matches_json_serialized_input():
    """tool_input_regex signal matches a regex pattern against the JSON-serialized input dict."""
    steps = [{"n": 1, "name": "dispatch with explicit model",
              "signal": "tool_input_regex", "tool": "Agent",
              "regex": r'"model"\s*:\s*"haiku"'}]
    events = [_tool_use("Agent", {"model": "haiku", "prompt": "do work"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_evaluate_steps_tool_input_regex_false_when_pattern_not_in_input():
    """tool_input_regex false when the pattern does not match the input."""
    steps = [{"n": 1, "name": "dispatch with explicit model",
              "signal": "tool_input_regex", "tool": "Agent",
              "regex": r'"model"\s*:\s*"haiku"'}]
    events = [_tool_use("Agent", {"model": "opus", "prompt": "do work"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_input_regex_false_when_wrong_tool():
    """tool_input_regex false when the tool name does not match."""
    steps = [{"n": 1, "name": "dispatch with explicit model",
              "signal": "tool_input_regex", "tool": "Agent",
              "regex": r'"model"\s*:\s*"haiku"'}]
    events = [_tool_use("Bash", {"command": "echo model=haiku"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_input_regex_case_insensitive():
    """tool_input_regex with case_insensitive flag matches regardless of case."""
    steps = [{"n": 1, "name": "check input",
              "signal": "tool_input_regex", "tool": "Agent",
              "regex": r'"MODEL"\s*:\s*"haiku"', "case_insensitive": True}]
    events = [_tool_use("Agent", {"model": "haiku", "prompt": "do work"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_evaluate_steps_tool_input_regex_after_ordering():
    """tool_input_regex respects after: ordering — must be strictly after the referenced step."""
    steps = [
        {"n": 1, "name": "recall", "signal": "bash_regex", "pattern": r"engram\s+query"},
        {"n": 2, "name": "dispatch", "signal": "tool_input_regex", "tool": "Agent",
         "regex": r'"model"\s*:\s*"haiku"', "after": 1}
    ]
    events = [
        _tool_use("Bash", {"command": "engram query"}, idx=0),
        _tool_use("Agent", {"model": "haiku", "prompt": "do work"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True


def test_real_gitignore_steps_json_step_1_credits_synthetic_native_read():
    """Round-5 finding: hand-verifying B-R showed an agent that inspects .gitignore via the
    native Read tool (not `cat .gitignore` in Bash) got no credit for step 1. The real fixture's
    step 1 is now any_of(bash_regex, tool_path) — a synthetic Read-only transcript must credit
    it."""
    steps = pp.load_steps("B")
    step1 = next(s for s in steps if s["n"] == 1)
    assert "any_of" in step1
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_commit_message_format_and_body_true_on_conforming_message(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m",
                     "feat(pkg): bump version\n\nwhy we did this\n\nAI-Used: [claude]"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is True


def test_commit_message_format_and_body_true_even_with_only_co_authored_trailer(tmp_path):
    """Round-4 ruling: the checker no longer scores the trailer at all — a body line's CONTENT
    is never inspected, only that one exists. A message whose only 'body' line is a
    Co-Authored-By trailer (the harness's own injected convention, not AI-Used) must still PASS
    this checker; trailer classification is reported separately via classify_trailer."""
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "feat(pkg): bump version\n\nCo-Authored-By: x"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is True


def test_commit_message_format_and_body_false_when_no_body_at_all(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "feat(pkg): bump version"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is False


def test_commit_message_format_and_body_false_when_subject_form_missing(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "bumped the version\n\nsome body text"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is False


# ----- classify_trailer: reported, not scored (round 4) -----

def _commit_with_message(repo, message):
    import subprocess
    os.makedirs(repo, exist_ok=True)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", message], cwd=repo, check=True, capture_output=True)


def test_classify_trailer_ai_used(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]")
    assert pp.classify_trailer(repo) == "ai_used"


def test_classify_trailer_co_authored(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>")
    assert pp.classify_trailer(repo) == "co_authored"


def test_classify_trailer_both(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(
        repo,
        "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>",
    )
    assert pp.classify_trailer(repo) == "both"


def test_classify_trailer_none(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy we made this change")
    assert pp.classify_trailer(repo) == "none"


# ----- gitignore_narrowed_to_generated checker (Task B step 3) -----

def _write_gitignore(repo, lines):
    os.makedirs(repo, exist_ok=True)
    with open(os.path.join(repo, ".gitignore"), "w") as f:
        f.write("\n".join(lines) + "\n")


def test_gitignore_narrowed_to_generated_true_when_narrowed_correctly(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is True


def test_gitignore_narrowed_to_generated_true_with_leading_globstar_form(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["**/testdata/generated", "*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is True


def test_gitignore_narrowed_to_generated_false_when_bare_testdata_still_present(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/", "testdata/generated/"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_bare_scripts_still_present(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "scripts/"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_narrowed_target_missing(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_gitignore_missing(tmp_path):
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


# ----- real steps.json files: valid JSON, evaluable, every repo_state pattern registered -----

def _step_signals(step):
    """A step's list of signal definitions — its own top-level signal, or every entry in
    `any_of`."""
    return step["any_of"] if "any_of" in step else [step]


def test_both_real_steps_json_files_are_valid_json_and_loadable():
    for task in ("A", "B"):
        steps = pp.load_steps(task)
        assert isinstance(steps, list)
        assert len(steps) > 0
        for step in steps:
            assert "n" in step
            for sig in _step_signals(step):
                assert "signal" in sig and "pattern" in sig


def test_real_steps_json_task_a_evaluates_against_a_synthetic_transcript_and_repo(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m",
                     "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]"],
                    cwd=repo, check=True, capture_output=True)
    events = [
        _tool_use("Bash", {"command": "ls -la .jj 2>/dev/null || true"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
        _tool_use("Bash", {"command": "git diff --staged"}, idx=2),
        _tool_use("Bash", {"command": "git log --oneline -5"}, idx=3),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=4),
        _tool_use("Bash", {"command": "git commit -m 'feat: bump'"}, idx=5),
        _tool_use("Bash", {"command": "git log -1"}, idx=6),
    ]
    steps = pp.load_steps("A")
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path=repo)
    assert k == len(steps)
    assert all_ is True


def test_real_steps_json_task_b_evaluates_against_a_synthetic_transcript_and_repo(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "*.o"])
    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),
        _tool_use("Bash", {"command": "git add scripts/build.sh"}, idx=3),
        _tool_use("Bash", {"command": "git diff --cached"}, idx=4),
    ]
    steps = pp.load_steps("B")
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path=repo)
    assert k == len(steps)
    assert all_ is True


def test_real_commit_steps_json_step_7_verify_requires_order_after_commit():
    """Final-review finding: Task A step 7 ('git log -1 or git status after commit') shares its
    pattern with step 2's 'git status' check. Without an 'after' constraint, an EARLY git status
    call (satisfying step 2, run before any real work) would ALSO satisfy step 7, even though no
    verification ever happened after the commit. The real fixture must carry after=6 (the commit
    step) so this can't happen — this proves the ordering constraint changes the verdict, not just
    that the field is present."""
    steps = pp.load_steps("A")
    step7 = next(s for s in steps if s["n"] == 7)
    assert step7.get("after") == 6

    def always_true_checker(_pattern, _repo):
        return True

    events_no_post_commit_verify = [
        _tool_use("Bash", {"command": "ls -la .jj 2>/dev/null || true"}, idx=0),  # step 1
        _tool_use("Bash", {"command": "git status"}, idx=1),                     # step 2
        _tool_use("Bash", {"command": "git log --oneline -5"}, idx=2),           # step 3
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=3),         # step 4
        _tool_use("Bash", {"command": "git commit -m 'feat: bump'"}, idx=4),     # step 6
    ]
    results, k, all_ = pp.evaluate_steps(steps, events_no_post_commit_verify, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["2"] is True  # the early git status still credits step 2
    assert results["7"] is False  # but must NOT also credit step 7 — no call happened after commit
    assert all_ is False

    events_with_post_commit_verify = events_no_post_commit_verify + [
        _tool_use("Bash", {"command": "git log -1"}, idx=5),
    ]
    results2, k2, all2_ = pp.evaluate_steps(steps, events_with_post_commit_verify, repo_path="/x",
                                             repo_checker=always_true_checker)
    assert results2["7"] is True
    assert all2_ is True


def test_real_gitignore_steps_json_step_6_verify_requires_order_after_staging():
    """Final-review finding: Task B step 6 ('git diff --cached or git status') shares its
    pattern with step 4's 'git status' check. Without an 'after' constraint, an EARLY git status
    call (satisfying step 4, run before staging) would ALSO satisfy step 6, even though nothing
    was ever verified after staging. The real fixture must carry after=5 (the staging step)."""
    steps = pp.load_steps("B")
    step6 = next(s for s in steps if s["n"] == 6)
    assert step6.get("after") == 5

    def always_true_checker(_pattern, _repo):
        return True

    events_no_post_stage_verify = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),                                   # step 1
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),   # step 2
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),                           # step 4
        _tool_use("Bash", {"command": "git add scripts/build.sh"}, idx=3),                         # step 5
    ]
    results, k, all_ = pp.evaluate_steps(steps, events_no_post_stage_verify, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["4"] is True  # the early git status still credits step 4
    assert results["6"] is False  # but must NOT also credit step 6 — nothing verified after staging
    assert all_ is False

    events_with_post_stage_verify = events_no_post_stage_verify + [
        _tool_use("Bash", {"command": "git diff --cached"}, idx=4),
    ]
    results2, k2, all2_ = pp.evaluate_steps(steps, events_with_post_stage_verify, repo_path="/x",
                                             repo_checker=always_true_checker)
    assert results2["6"] is True
    assert all2_ is True


def test_real_gitignore_steps_json_step_6_verify_satisfied_by_same_compound_staging_command():
    """Regression test for the reported real trial (hand-verified B-R, idx 16): a single compound
    Bash command stages AND verifies in one call —
    `git add .gitignore scripts/build.sh testdata/fixture.json && git diff --cached --name-only
    && git status --short`. Step 5 (staging) and step 6 (verify, after=5) must BOTH be credited
    from this one event — the after-constraint fix must not regress this legitimate case back to
    false."""
    steps = pp.load_steps("B")

    def always_true_checker(_pattern, _repo):
        return True

    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),
        _tool_use("Bash", {
            "command": ("git add .gitignore scripts/build.sh testdata/fixture.json "
                        "&& git diff --cached --name-only && git status --short"),
        }, idx=3),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["5"] is True
    assert results["6"] is True
    assert all_ is True


def test_real_gitignore_steps_json_step_6_finds_later_verify_past_leading_status_match():
    """Regression for the real trials B-S#0/1/2, B-F#1/3: an early 'git status' (satisfying step
    4) occurs BEFORE the staging 'git add' call within the SAME compound command, and a later
    'git diff --cached'/'git status --short' occurs after it. Step 6 (after=5) must find that
    LATER occurrence, not fail because the leftmost occurrence of its own pattern sits before
    step 5's match."""
    steps = pp.load_steps("B")

    def always_true_checker(_pattern, _repo):
        return True

    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {
            "command": ("git status --porcelain -uall && git add .gitignore scripts/build.sh "
                        "testdata/fixture.json && git diff --cached --name-only && "
                        "git status --short"),
        }, idx=2),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["4"] is True
    assert results["5"] is True
    assert results["6"] is True
    assert all_ is True


def test_real_gitignore_steps_json_step_6_false_when_step_5_never_matched():
    """Second final-review ruling on the real fixture: if step 5 (staging) never matches at all
    (e.g. the agent used `git add -A` instead of explicit paths), step 6 (after=5) must be FALSE
    even though its own verify pattern matches later in the transcript — not vacuously True."""
    steps = pp.load_steps("B")

    def always_true_checker(_pattern, _repo):
        return True

    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),
        _tool_use("Bash", {"command": "git add -A"}, idx=3),
        _tool_use("Bash", {"command": "git diff --cached --name-only"}, idx=4),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["5"] is False
    assert results["6"] is False


def test_real_gitignore_steps_json_step_5_accepts_explicit_path_terminator_form():
    """Final-review round-2 widening: `git add -- <paths>` (B-R#2's real shape) must credit
    step 5 — previously only a bare `git add <path>` (no `--`) matched, scoring a false miss on a
    legitimate explicit-path staging call."""
    steps = pp.load_steps("B")
    step5 = next(s for s in steps if s["n"] == 5)
    events = [
        _tool_use("Bash", {
            "command": "git add -- .gitignore scripts/build.sh testdata/fixture.json",
        }, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps([step5], events, repo_path="/does/not/matter")
    assert results["5"] is True


def test_every_repo_state_pattern_in_real_steps_json_is_registered():
    for task in ("A", "B"):
        for step in pp.load_steps(task):
            for sig in _step_signals(step):
                if sig["signal"] == "repo_state":
                    assert sig["pattern"] in pp.REPO_STATE_CHECKERS, (
                        f"Task {task} step {step['n']} names repo_state pattern "
                        f"{sig['pattern']!r}, which has no registered checker"
                    )


# ----- non-fatal scoring: one trial's scoring exception must not lose sibling results -----

def test_score_trial_scoring_exception_is_captured_not_raised(monkeypatch, tmp_path):
    def _boom(*args, **kwargs):
        raise ValueError("synthetic scoring failure")

    monkeypatch.setattr(pp, "score_found_phase2", _boom)
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    scored = pp._score_trial("A", "R", events=[], repo_path=repo, carrier_basename="x")
    assert scored["scoring_error"] is not None
    assert "synthetic scoring failure" in scored["scoring_error"]
    # safe defaults, not a raised exception
    assert scored["found"] is None
    assert scored["end_state"] is False
    assert scored["followed_k"] == 0


def test_score_trial_no_exception_leaves_scoring_error_none():
    scored = pp._score_trial("A", "S", events=[], repo_path="/does/not/exist", carrier_basename=None)
    assert scored["scoring_error"] is None


def test_score_trial_populates_trailer_for_task_a(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]")
    scored = pp._score_trial("A", "S", events=[], repo_path=repo, carrier_basename=None)
    assert scored["trailer"] == "ai_used"


def test_score_trial_trailer_is_na_for_task_b():
    scored = pp._score_trial("B", "S", events=[], repo_path="/does/not/exist", carrier_basename=None)
    assert scored["trailer"] == "n/a"


# ----- live scoring vs --rescore: same field set (round 6) -----

def test_live_scoring_and_rescore_emit_the_same_scored_field_set(tmp_path):
    """The live scoring path (_score_trial, feeding run_one_trial_phase2's persisted record) and
    --rescore must emit the SAME set of scored fields — round-6 finding: first_procedure_step_index
    was computed by --rescore but never by the live path, so base result files lacked it.
    `rescored_from` is the one expected rescore-only addition (provenance marking that a record
    was rescored) — everything else must match exactly.

    `valid`/`marker_seen` are computed live by run_one_trial_phase2 itself (from the marker it
    generated), not by `_score_trial` — they are added to the expected set here rather than read
    off `scored`. The truncated-projects-dir fix (a long trial repo path can make Claude Code
    truncate its own projects/<slug> dir name, so a kept record's transcript_path can be None
    even though the trial ran) taught --rescore to re-derive both fields too, by reading the
    PROBE-TOKEN back off the trial repo's own committed CLAUDE.md — see rescore_file /
    _read_repo_marker — so the full live record schema (not just _score_trial's slice of it) is
    now what --rescore must match."""
    repo_path = pp.setup_trial_repo(str(tmp_path / "trial"), "A", "R",
                                     marker="RUNBOOK-VS-SKILL-PROBE2-parity")
    carrier = "1.2026-09-11.commit-conventional-message"

    transcript_path = tmp_path / "session.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu1", "name": "Bash",
                                      "input": {"command": "engram query --phrase x"}}]}
        }),
        json.dumps({
            "type": "user", "timestamp": "2026-09-11T00:00:01.000Z",
            "message": {"content": [{"type": "tool_result", "tool_use_id": "tu1",
                                      "content": f"{carrier}.md"}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:02.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu2", "name": "Bash",
                                      "input": {"command": "git add pkg/version.go"}}]}
        }),
    ]
    with open(transcript_path, "w") as f:
        f.write("\n".join(lines) + "\n")

    # --- live path: _score_trial's scored keys (minus scoring_error, a live-only diagnostic
    # never persisted as a scored field name in the record schema itself) ---
    events = pp.p1.parse_transcript_events([str(transcript_path)])
    scored = pp._score_trial("A", "R", events, repo_path, carrier)
    live_scored_keys = ((set(scored.keys()) - {"scoring_error"})
                         | {"valid", "marker_seen", "invalid_reason", "api_error"})

    # --- rescore path: whatever keys rescore_file adds/overwrites onto a minimal provenance-only
    # record ---
    minimal_record = {"task": "A", "arm": "R", "repo_path": repo_path,
                       "transcript_path": str(transcript_path), "carrier_basename": carrier}
    results_path = tmp_path / "minimal.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(minimal_record) + "\n")
    rescored_path = tmp_path / "minimal.rescored.jsonl"
    pp.rescore_file(str(results_path), str(rescored_path))
    rescored_record = pp.load_jsonl(str(rescored_path))[0]
    rescore_added_keys = set(rescored_record.keys()) - set(minimal_record.keys())

    assert rescore_added_keys - {"rescored_from"} == live_scored_keys


def test_score_trial_load_steps_exception_is_captured_not_raised(monkeypatch, tmp_path):
    """Round-2 review: load_steps() itself must be inside the try — a missing/invalid steps.json
    must not break the 'never raises' contract either. n_steps stays at its 0 default since it
    is set only after a successful load_steps() call."""
    def _boom(task_key):
        raise FileNotFoundError("synthetic missing steps.json")

    monkeypatch.setattr(pp, "load_steps", _boom)
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    scored = pp._score_trial("A", "R", events=[], repo_path=repo, carrier_basename="x")
    assert scored["scoring_error"] is not None
    assert "synthetic missing steps.json" in scored["scoring_error"]
    assert scored["n_steps"] == 0
    assert scored["found"] is None
    assert scored["end_state"] is False


# ----- starting-state assertion (real fixture init scripts, no claude calls) -----

def test_setup_trial_repo_task_a_preserves_unstaged_and_decoy(tmp_path):
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "R", marker="RUNBOOK-VS-SKILL-PROBE2-x")
    import subprocess
    status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                             capture_output=True, text=True, check=True).stdout
    assert " M pkg/version.go" in status
    assert "?? notes/" in status


def test_setup_trial_repo_task_b_preserves_decoy_and_ignored_dirs(tmp_path):
    """Task B's fixture (init_fixture_repo.sh, fixed at commit b098afa1) commits only .gitignore
    + src/main.go; scripts/build.sh and testdata/fixture.json start ignored AND untracked (not
    in the index), and tmp.log is created as an untracked decoy after the fixture commit."""
    repo_path = pp.setup_trial_repo(str(tmp_path), "B", "F", marker="RUNBOOK-VS-SKILL-PROBE2-y")
    import subprocess
    status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                             capture_output=True, text=True, check=True).stdout
    assert "?? tmp.log" in status

    for rel_path in ("testdata/generated/big.bin", "scripts/build.sh", "testdata/fixture.json"):
        ignored = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", rel_path])
        assert ignored.returncode == 0, f"{rel_path} should be ignored"

    ls_files = subprocess.run(["git", "-C", repo_path, "ls-files"],
                               capture_output=True, text=True, check=True).stdout
    assert "scripts/build.sh" not in ls_files
    assert "testdata/fixture.json" not in ls_files


def test_setup_trial_repo_task_a_arm_s_deploys_skill_and_commits_it(tmp_path):
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "S", marker="RUNBOOK-VS-SKILL-PROBE2-z")
    assert os.path.isfile(os.path.join(repo_path, ".claude", "skills", "commit", "SKILL.md"))
    import subprocess
    log = subprocess.run(["git", "-C", repo_path, "log", "--oneline"], capture_output=True, text=True).stdout
    assert "add project config" in log


def test_setup_trial_repo_task_a_arm_n_adds_no_claude_dir_and_plain_claude_md(tmp_path):
    """Arm N (no-instructions control): same CLAUDE.md as S/R/F (guidance + cue + marker, no
    Rdirect '## Project procedure' section) and NO .claude dir at all — no skill deployed."""
    marker = "RUNBOOK-VS-SKILL-PROBE2-narm"
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "N", marker=marker)
    assert not os.path.isdir(os.path.join(repo_path, ".claude"))
    claude_md = open(os.path.join(repo_path, "CLAUDE.md")).read()
    assert f"PROBE-TOKEN: {marker}" in claude_md
    assert "## Project procedure" not in claude_md


def test_setup_trial_repo_task_b_arm_n_adds_no_claude_dir(tmp_path):
    repo_path = pp.setup_trial_repo(str(tmp_path), "B", "N", marker="RUNBOOK-VS-SKILL-PROBE2-narmb")
    assert not os.path.isdir(os.path.join(repo_path, ".claude"))


# ----- .eval/original_tip re-stamping (issue #752 regression) -----
#
# Bug: several fixtures' init_fixture_repo.sh stamp .eval/original_tip via `git rev-parse HEAD`
# BEFORE setup_trial_repo makes its own "add project config" commit on top. Every trial's real
# HEAD at hand-off is therefore one commit AHEAD of what original_tip recorded. history-rewrite's
# done_when_checks.sh Checks 5/6 verify recoverability of the EXACT recorded original_tip -- so an
# agent that faithfully follows the task (record `git rev-parse HEAD`, i.e. the harness's commit,
# then preserve that SHA's reachability) can never satisfy a check testing the SHA one commit
# further back, regardless of what it does correctly. Fix: setup_trial_repo re-stamps
# .eval/original_tip to whatever is actually HEAD once its own commit lands.

@pytest.mark.parametrize("task_key", ["history-rewrite", "tdd-order", "bisect-before-fix", "route"])
def test_setup_trial_repo_original_tip_matches_head_after_setup(tmp_path, task_key):
    """Every task whose init_fixture_repo.sh stamps .eval/original_tip must see it re-stamped, by
    setup_trial_repo, to the SHA that is actually HEAD once setup_trial_repo returns -- the true
    pre-agent-work tip a trial agent starts from -- not the fixture's pre-harness-commit parent."""
    marker = f"RUNBOOK-VS-SKILL-PROBE2-origtip-{task_key}"
    repo_path = pp.setup_trial_repo(str(tmp_path), task_key, "R", marker=marker)
    import subprocess
    head_sha = subprocess.run(["git", "-C", repo_path, "rev-parse", "HEAD"],
                               capture_output=True, text=True, check=True).stdout.strip()
    orig_tip_path = os.path.join(str(tmp_path), ".eval", "original_tip")
    assert os.path.isfile(orig_tip_path), f"{task_key} fixture should stamp .eval/original_tip"
    recorded_tip = open(orig_tip_path).read().strip()
    assert recorded_tip == head_sha


def test_setup_trial_repo_history_rewrite_original_tip_is_the_add_project_config_commit(tmp_path):
    """The recorded original_tip must be the harness's own 'add project config' commit -- the SHA
    a trial agent actually observes as HEAD at hand-off -- not its parent. Before the fix,
    original_tip pointed one commit BEHIND real HEAD, so no ref an agent created at (or descended
    from) real HEAD could ever equal the recorded tip, making done_when_checks.sh Checks 5/6
    near-unpassable regardless of the agent's actions."""
    repo_path = pp.setup_trial_repo(str(tmp_path), "history-rewrite", "R",
                                     marker="RUNBOOK-VS-SKILL-PROBE2-origtip-hr")
    import subprocess
    subject = subprocess.run(["git", "-C", repo_path, "log", "-1", "--format=%s"],
                              capture_output=True, text=True, check=True).stdout.strip()
    assert subject == "add project config"
    head_sha = subprocess.run(["git", "-C", repo_path, "rev-parse", "HEAD"],
                               capture_output=True, text=True, check=True).stdout.strip()
    orig_tip_path = os.path.join(str(tmp_path), ".eval", "original_tip")
    recorded_tip = open(orig_tip_path).read().strip()
    assert recorded_tip == head_sha


# ----- decision frame outputs: baseline_uninterpretable and shim loss -----

def _agg(n, valid_n, found_n, end_state_n, followed_all_n, end_state_given_found_n=None,
         followed_all_given_found_n=None, n_steps=6, api_error_n=0, stalled_n=0):
    return {
        "n": n, "valid_n": valid_n, "found_n": found_n, "found_given_n": found_n,
        "end_state_n": end_state_n,
        "end_state_given_found_n": end_state_given_found_n if end_state_given_found_n is not None else end_state_n,
        "followed_all_n": followed_all_n,
        "followed_all_given_found_n": (followed_all_given_found_n if followed_all_given_found_n is not None
                                        else followed_all_n),
        "recall_fired_n": 0, "followed_mean_k": float(followed_all_n), "n_steps": n_steps,
        "cost_mean": 0.10, "duration_mean": 30.0,
        "trailer_ai_used_n": 0, "trailer_co_authored_n": 0,
        "api_error_n": api_error_n, "stalled_n": stalled_n,
    }


def test_decomposition_baseline_uninterpretable_true_when_s_end_state_below_3():
    agg = {"S": _agg(5, 5, 2, 2, 2), "R": _agg(5, 5, 4, 4, 4), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["baseline_uninterpretable"] is True


def test_decomposition_baseline_uninterpretable_false_when_s_end_state_at_least_3():
    agg = {"S": _agg(5, 5, 3, 3, 3), "R": _agg(5, 5, 4, 4, 4), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["baseline_uninterpretable"] is False


def test_decomposition_shim_loss_reports_both_populations_when_they_diverge():
    """Controller ruling (final): shim_loss must report BOTH the total population (all valid R
    trials, incl. not-found) and the given_found population (only R's found=true subset) — these
    must be observably different, not aliased. R's found_n=3 < valid_n=5, and
    end_state_given_found_n(0) != end_state_n(3), by construction. Final-review correction:
    shim_loss is a percentage-point RATE difference (dict with both fractions), never a raw-count
    subtraction — see test_decomposition_shim_loss_unequal_n_equal_rates_is_zero_pp for why."""
    agg = {
        "S": _agg(5, 5, 4, 4, 4),
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    # Rdirect 4/5 (80%) vs R 3/5 (60%) -> +20.0pp
    assert frame["shim_loss_total"] == {"pp_diff": 20.0, "rdirect": "4/5", "r": "3/5"}
    # Rdirect 4/5 (80%) vs R's found=true subset 0/3 (0%) -> +80.0pp
    assert frame["shim_loss_given_found"] == {"pp_diff": 80.0, "rdirect": "4/5", "r_given_found": "0/3"}
    assert frame["shim_loss_total"] != frame["shim_loss_given_found"]


def test_decomposition_shim_loss_unequal_n_equal_rates_is_zero_pp():
    """Final-review finding: Task B's real numbers have Rdirect at valid_n=4 (one trial
    invalidated by a mid-session rate limit) at 4/4 (100%) vs R at valid_n=5 at 5/5 (100%) — raw
    counts differ (4 - 5 = -1) but the RATES are identical. The reported figure must be 0
    percentage points, not a negative count implying Rdirect lost ground."""
    agg = {
        "R": _agg(5, 5, 5, 5, 5, end_state_given_found_n=5, followed_all_given_found_n=5),
        "F": _agg(5, 5, 5, 5, 5, end_state_given_found_n=5, followed_all_given_found_n=5),
        "Rdirect": _agg(4, 4, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["shim_loss_total"] == {"pp_diff": 0.0, "rdirect": "4/4", "r": "5/5"}


def test_decomposition_note_quality_f_reports_both_populations_when_they_diverge():
    agg = {
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    # F 4/5 (80%) vs Rdirect 4/5 (80%) -> 0.0pp
    assert frame["note_quality_F_total"] == {"pp_diff": 0.0, "f": "4/5", "rdirect": "4/5"}
    # F's found=true subset 2/4 (50%) vs Rdirect 4/5 (80%) -> -30.0pp
    assert frame["note_quality_F_given_found"] == {
        "pp_diff": -30.0, "f_given_found": "2/4", "rdirect": "4/5"}
    assert frame["note_quality_F_total"] != frame["note_quality_F_given_found"]


def test_decomposition_type_effect_reports_both_populations_when_they_diverge():
    agg = {
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["type_effect"]["end_state_total"] == 3 - 4
    assert frame["type_effect"]["end_state_given_found"] == 0 - 2
    assert frame["type_effect"]["followed_all_total"] == 3 - 4
    assert frame["type_effect"]["followed_all_given_found"] == 0 - 2
    assert frame["type_effect"]["end_state_total"] != frame["type_effect"]["end_state_given_found"]
    assert (frame["type_effect"]["followed_all_total"]
            != frame["type_effect"]["followed_all_given_found"])


def test_decomposition_parity_cant_distinguish_within_one_trial():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 3, 3, 3), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["parity"]["S_vs_R"]["end_state"] == "cant_distinguish"
    assert frame["parity"]["S_vs_F"]["end_state"] == "cant_distinguish"


def test_decomposition_parity_worse_and_better_at_two_or_more():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 1, 1, 1), "F": _agg(5, 5, 1, 1, 1),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["parity"]["S_vs_R"]["end_state"] == "worse"
    agg2 = {"S": _agg(5, 5, 1, 1, 1), "R": _agg(5, 5, 5, 5, 5), "F": _agg(5, 5, 1, 1, 1),
            "Rdirect": _agg(5, 5, None, 3, 3)}
    frame2 = pp.decomposition(agg2)
    assert frame2["parity"]["S_vs_R"]["end_state"] == "better"


def test_decomposition_note_quality_given_delivery_format():
    agg = {"R": _agg(5, 5, 4, 3, 3, end_state_given_found_n=3, followed_all_given_found_n=2)}
    frame = pp.decomposition(agg)
    assert frame["note_quality_given_delivery_R"]["end_state"] == "3 of 4"
    assert frame["note_quality_given_delivery_R"]["followed_all"] == "2 of 4"


# ----- arm N (no-instructions control): decomposition ignores it; --summarize reports it -----

def test_decomposition_ignores_arm_n():
    """N is a control, not part of the decision frame — decomposition() must produce the exact
    same output whether or not N is present in agg."""
    agg_without_n = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 3, 3, 3), "F": _agg(5, 5, 3, 3, 3),
                      "Rdirect": _agg(5, 5, None, 3, 3)}
    agg_with_n = dict(agg_without_n, N=_agg(5, 5, None, 1, 1))
    assert pp.decomposition(agg_with_n) == pp.decomposition(agg_without_n)


def test_format_table_shows_n_column_and_no_instructions_baseline_line():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 3, 3, 3), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3), "N": _agg(5, 5, None, 1, 2)}
    table = pp.format_table("A", agg)
    assert "N" in table.splitlines()[1]  # header row includes the N column
    assert "no-instructions baseline: FOLLOWED-all 2/5, END-STATE 1/5" in table


def test_format_table_found_row_is_na_for_n():
    agg = {"S": _agg(5, 5, 4, 4, 4), "N": _agg(5, 5, None, 1, 2)}
    table = pp.format_table("A", agg)
    found_row = next(line for line in table.splitlines() if line.startswith("FOUND"))
    assert "n/a" in found_row


def test_format_table_omits_no_instructions_line_when_n_absent():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 3, 3, 3)}
    table = pp.format_table("A", agg)
    assert "no-instructions baseline" not in table


def test_score_found_phase2_arm_n_is_always_na():
    events = [_tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0)]
    found, idx, found_via = pp.score_found_phase2("A", "N", events, carrier_basename=None)
    assert found is None
    assert idx is None
    assert found_via is None


def test_found_method_arm_n_is_na():
    assert pp.found_method("N", None) == "n/a"
    assert pp.found_method("N", False) == "n/a"


def test_arms_includes_n_but_default_arms_does_not():
    assert "N" in pp.ARMS
    assert "N" not in pp.DEFAULT_ARMS
    assert pp.DEFAULT_ARMS == ("S", "R", "F", "Rdirect")


# ----- aggregate() -----

def test_aggregate_filters_by_task_and_arm_and_validity():
    records = [
        {"task": "A", "arm": "S", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": False, "total_cost_usd": 0.5, "duration_ms": 1000},
        {"task": "A", "arm": "S", "valid": False, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": False, "total_cost_usd": 0.5, "duration_ms": 1000},
        {"task": "B", "arm": "S", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 6, "n_steps": 6, "recall_fired": False, "total_cost_usd": 0.3, "duration_ms": 500},
        {"task": "A", "arm": "R", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": True, "total_cost_usd": 0.4, "duration_ms": 900},
    ]
    agg = pp.aggregate(records, "A", "S")
    assert agg["n"] == 2
    assert agg["valid_n"] == 1
    assert agg["found_n"] == 1
    assert agg["end_state_n"] == 1


def test_aggregate_rdirect_found_n_is_zero_when_found_is_none():
    records = [
        {"task": "A", "arm": "Rdirect", "valid": True, "found": None, "end_state": True,
         "followed_all": True, "followed_k": 7, "n_steps": 7, "recall_fired": False,
         "total_cost_usd": 0.2, "duration_ms": 800},
    ]
    agg = pp.aggregate(records, "A", "Rdirect")
    assert agg["found_n"] == 0
    assert agg["end_state_n"] == 1


# ----- --rescore also recomputes FOUND (round 3) -----

def test_rescore_recomputes_found_from_kept_transcript(tmp_path):
    """Simulates the smoke-run-1 scenario: an original record was scored found=False by the
    pre-fix mutating regex (a bare '>' falsely matched '2>&1' in the recall skill's own
    diagnostic call, making that turn 0 count as 'the first mutation'), even though the carrier
    basename genuinely appeared in the engram query result before any real mutation. Rescoring
    with the fixed detector must flip found to True."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo)

    transcript_path = tmp_path / "session.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu1", "name": "Bash",
                                      "input": {"command": "engram ingest --auto 2>&1 | tail -5"}}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:01.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu2", "name": "Bash",
                                      "input": {"command": "engram query --phrase x"}}]}
        }),
        json.dumps({
            "type": "user", "timestamp": "2026-09-11T00:00:02.000Z",
            "message": {"content": [{"type": "tool_result", "tool_use_id": "tu2",
                                      "content": "1.2026-09-11.commit-conventional-message.md"}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:03.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu3", "name": "Bash",
                                      "input": {"command": "git add pkg/version.go"}}]}
        }),
    ]
    with open(transcript_path, "w") as f:
        f.write("\n".join(lines) + "\n")

    record = {
        "task": "A", "arm": "R", "repo_path": repo, "transcript_path": str(transcript_path),
        "carrier_basename": "1.2026-09-11.commit-conventional-message",
        "found": False, "found_index": None,  # the pre-fix (buggy) scoring
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")

    rescored_path = tmp_path / "rescored.jsonl"
    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["found"] is True
    assert rescored["found_index"] == 1
    assert rescored["found_method"] == "engram query"
    assert rescored["first_procedure_step_index"] == 3  # the git add, not the 2>&1 diagnostic call
    assert rescored["rescored_from"] == str(results_path)


def test_rescore_preserves_provenance_fields_from_kept_record():
    """arm/carrier_basename/task provenance must be READ from the kept record, never re-derived —
    a missing repo_path is the only thing that should short-circuit to an error."""
    record = {"task": "A", "arm": "S", "repo_path": "/does/not/exist", "transcript_path": None,
              "carrier_basename": None}
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        results_path = os.path.join(td, "results.jsonl")
        with open(results_path, "w") as f:
            f.write(json.dumps(record) + "\n")
        rescored_path = os.path.join(td, "rescored.jsonl")
        pp.rescore_file(results_path, rescored_path)
        rescored = pp.load_jsonl(rescored_path)[0]
    assert rescored["error"] == "rescore: repo missing"
    assert rescored["arm"] == "S"


# ----- truncated-projects-dir bug: rediscover_transcript_paths + _read_repo_marker + rescore -----

def _make_truncated_session_jsonl(dir_path, filename, cwd, extra_text=""):
    """A synthetic Claude Code session jsonl: a leading 'queue-operation' record with no `cwd`
    key (verified against a real transcript — the file's literal first line often has none),
    followed by a 'user' record carrying `cwd` and, optionally, `extra_text` in its message
    content (used to plant the PROBE-TOKEN marker text so marker-validity recomputation can find
    it)."""
    os.makedirs(dir_path, exist_ok=True)
    lines = [
        json.dumps({"type": "queue-operation", "operation": "x", "sessionId": "s1",
                     "timestamp": "2026-09-12T00:00:00.000Z"}),
        json.dumps({"type": "user", "cwd": cwd, "sessionId": "s1",
                     "timestamp": "2026-09-12T00:00:01.000Z",
                     "message": {"content": [{"type": "text", "text": extra_text}]} if extra_text
                     else {"content": []}}),
    ]
    path = os.path.join(dir_path, filename)
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")
    return path


def test_rediscover_transcript_paths_finds_match_across_cfg_dirs(tmp_path):
    """The bug: a long trial repo path makes Claude Code truncate the projects/ dir name and
    append a random 6-char suffix (observed '...-trials-gitigno-t7vm24'). Which cfg-N a live
    run's worker pool assigned to this trial is never recorded, so every cfg-*/ dir under the
    trial's run_root must be tried."""
    run_root = tmp_path / "run"
    trial_dir = run_root / "trials" / "gitignore-nested-N-0"
    repo_path = str(trial_dir / "repo")
    os.makedirs(repo_path, exist_ok=True)

    # An unrelated session under a DIFFERENT cfg dir must never be picked.
    unrelated_repo = str(run_root / "trials" / "other" / "repo")
    os.makedirs(unrelated_repo, exist_ok=True)
    _make_truncated_session_jsonl(run_root / "cfg-0" / "projects" / "unrelated-name",
                                   "session.jsonl", cwd=unrelated_repo)

    full_slug = pp.p1.isolation.project_slug(repo_path)
    truncated_name = full_slug[: len(full_slug) - 10] + "-t7vm24"
    jsonl_path = _make_truncated_session_jsonl(run_root / "cfg-1" / "projects" / truncated_name,
                                                "session.jsonl", cwd=repo_path)

    found = pp.rediscover_transcript_paths(str(trial_dir), repo_path)

    assert found == [jsonl_path]


def test_rediscover_transcript_paths_returns_empty_for_falsy_trial_dir():
    assert pp.rediscover_transcript_paths(None, "/some/repo") == []
    assert pp.rediscover_transcript_paths("", "/some/repo") == []


def test_rediscover_transcript_paths_returns_empty_when_run_root_missing(tmp_path):
    trial_dir = str(tmp_path / "does-not-exist" / "trials" / "t0")
    assert pp.rediscover_transcript_paths(trial_dir, "/some/repo") == []


def test_rediscover_transcript_paths_returns_empty_when_no_cfg_dir_matches(tmp_path):
    run_root = tmp_path / "run"
    trial_dir = run_root / "trials" / "t0"
    repo_path = str(trial_dir / "repo")
    os.makedirs(repo_path, exist_ok=True)
    other_repo = str(run_root / "trials" / "other" / "repo")
    os.makedirs(other_repo, exist_ok=True)
    _make_truncated_session_jsonl(run_root / "cfg-0" / "projects" / "some-dir",
                                   "session.jsonl", cwd=other_repo)

    assert pp.rediscover_transcript_paths(str(trial_dir), repo_path) == []


def test_read_repo_marker_extracts_probe_token(tmp_path):
    repo = tmp_path / "repo"
    repo.mkdir()
    (repo / "CLAUDE.md").write_text("some guidance\n\nPROBE-TOKEN: ABC123\n")

    assert pp._read_repo_marker(str(repo)) == "ABC123"


def test_read_repo_marker_returns_none_when_claude_md_missing(tmp_path):
    repo = tmp_path / "repo"
    repo.mkdir()

    assert pp._read_repo_marker(str(repo)) is None


def test_read_repo_marker_returns_none_when_no_probe_token_line(tmp_path):
    repo = tmp_path / "repo"
    repo.mkdir()
    (repo / "CLAUDE.md").write_text("no marker here\n")

    assert pp._read_repo_marker(str(repo)) is None


def test_rescore_rediscovers_missing_transcript_path_and_flips_valid_true(tmp_path):
    """End-to-end reproduction of the first --baseline batch's bug: a kept record has
    transcript_path=None (the live run's exact-slug projects/ lookup found nothing because Claude
    Code truncated its own directory name), but the trial actually ran and its transcript exists
    under a truncated-name sibling directory that carries the marker text. --rescore must
    re-discover the transcript, recompute marker validity from the trial repo's own committed
    CLAUDE.md, and flip both `transcript_path` and `valid`/`marker_seen` accordingly."""
    run_root = tmp_path / "run"
    trial_dir = run_root / "trials" / "A-R-0"
    repo_path = str(trial_dir / "repo")
    os.makedirs(repo_path, exist_ok=True)
    marker = "RUNBOOK-VS-SKILL-BASELINE-deadbeef"
    with open(os.path.join(repo_path, "CLAUDE.md"), "w") as f:
        f.write(f"some guidance\n\nPROBE-TOKEN: {marker}\n")

    full_slug = pp.p1.isolation.project_slug(repo_path)
    truncated_name = full_slug[: len(full_slug) - 10] + "-t7vm24"
    jsonl_path = _make_truncated_session_jsonl(
        run_root / "cfg-0" / "projects" / truncated_name, "session.jsonl",
        cwd=repo_path, extra_text=f"PROBE-TOKEN: {marker}")

    record = {
        "task": "A", "arm": "R", "trial_dir": str(trial_dir), "repo_path": repo_path,
        "transcript_path": None, "carrier_basename": None,
        "valid": False, "marker_seen": False,
        "found": None, "found_index": None,
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")
    rescored_path = tmp_path / "rescored.jsonl"

    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["transcript_path"] == str(jsonl_path)
    assert rescored["marker_seen"] is True
    assert rescored["valid"] is True


def test_rescore_leaves_valid_false_when_rediscovery_finds_nothing(tmp_path):
    """No matching transcript anywhere under the run_root — rescore must not fabricate a
    validity; valid/marker_seen stay False and transcript_path stays None."""
    run_root = tmp_path / "run"
    trial_dir = run_root / "trials" / "A-R-0"
    repo_path = str(trial_dir / "repo")
    os.makedirs(repo_path, exist_ok=True)
    with open(os.path.join(repo_path, "CLAUDE.md"), "w") as f:
        f.write("some guidance\n\nPROBE-TOKEN: RUNBOOK-VS-SKILL-BASELINE-neverfound\n")
    os.makedirs(run_root / "cfg-0" / "projects", exist_ok=True)  # no matching session anywhere

    record = {
        "task": "A", "arm": "R", "trial_dir": str(trial_dir), "repo_path": repo_path,
        "transcript_path": None, "carrier_basename": None,
        "valid": False, "marker_seen": False,
        "found": None, "found_index": None,
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")
    rescored_path = tmp_path / "rescored.jsonl"

    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["transcript_path"] is None
    assert rescored["marker_seen"] is False
    assert rescored["valid"] is False


# ----- shortened run root (RUN_ROOT_SUBDIR, run_baseline's short dir name) -----

def test_run_root_subdir_is_shortened_from_phase2():
    assert pp.RUN_ROOT_SUBDIR == "p2"


# ----- task registry: fixtures/<name>/ discovery + optional task.json -----

def _write_synthetic_fixture(task_dir, with_task_json=None):
    task_dir.mkdir(parents=True)
    (task_dir / "init_fixture_repo.sh").write_text("#!/bin/bash\nmkdir -p \"$1\"\n")
    (task_dir / "done_when_checks.sh").write_text("#!/bin/bash\nexit 0\n")
    (task_dir / "task-prompt.txt").write_text("do the widget task")
    (task_dir / "steps.json").write_text("[]")
    if with_task_json is not None:
        (task_dir / "task.json").write_text(json.dumps(with_task_json))


def test_discover_tasks_finds_synthetic_fixture_dir(tmp_path):
    fixtures_dir = tmp_path / "fixtures"
    task_dir = fixtures_dir / "widget"
    _write_synthetic_fixture(task_dir)

    discovered = pp.discover_tasks(str(fixtures_dir))

    assert "widget" in discovered
    entry = discovered["widget"]
    assert entry["init_script"] == str(task_dir / "init_fixture_repo.sh")
    assert entry["done_when_script"] == str(task_dir / "done_when_checks.sh")
    assert entry["task_prompt"] == str(task_dir / "task-prompt.txt")
    assert entry["steps_json"] == str(task_dir / "steps.json")
    # no task.json present -> carrier config defaults to empty/None
    assert entry["removal_basenames"] == ()
    assert entry["carrier_r_src"] is None
    assert entry["carrier_f_src"] is None
    assert entry["skill_name"] is None
    assert entry["skill_src"] is None


def test_discover_tasks_ignores_dir_missing_a_required_file(tmp_path):
    fixtures_dir = tmp_path / "fixtures"
    incomplete = fixtures_dir / "incomplete"
    incomplete.mkdir(parents=True)
    (incomplete / "init_fixture_repo.sh").write_text("#!/bin/bash\n")
    # missing done_when_checks.sh, task-prompt.txt, steps.json

    discovered = pp.discover_tasks(str(fixtures_dir))

    assert "incomplete" not in discovered


def test_discover_tasks_ignores_non_directory_entries(tmp_path):
    fixtures_dir = tmp_path / "fixtures"
    fixtures_dir.mkdir()
    (fixtures_dir / "stray_file.txt").write_text("not a task dir")

    discovered = pp.discover_tasks(str(fixtures_dir))

    assert discovered == {}


def test_discover_tasks_reads_optional_task_json_and_resolves_paths_relative_to_here(tmp_path):
    fixtures_dir = tmp_path / "fixtures"
    task_dir = fixtures_dir / "widget"
    _write_synthetic_fixture(task_dir, with_task_json={
        "removal_basenames": ["1.note", "2.note"],
        "carrier_r_src": "../elsewhere/vault",
        "carrier_f_src": None,
        "skill_name": "widget",
        "skill_src": "../elsewhere/widget-skill",
    })

    discovered = pp.discover_tasks(str(fixtures_dir))

    entry = discovered["widget"]
    assert entry["removal_basenames"] == ("1.note", "2.note")
    assert entry["skill_name"] == "widget"
    assert entry["carrier_f_src"] is None
    assert entry["carrier_r_src"] == os.path.normpath(os.path.join(pp.HERE, "../elsewhere/vault"))
    assert entry["skill_src"] == os.path.normpath(os.path.join(pp.HERE, "../elsewhere/widget-skill"))


def test_discover_tasks_missing_fixtures_dir_returns_empty():
    assert pp.discover_tasks("/does/not/exist/anywhere") == {}


def test_task_a_task_json_round_trips_into_tasks_registry():
    a = pp.TASKS["A"]
    assert a["removal_basenames"] == pp.TASK_A_REMOVAL
    assert a["skill_name"] == "commit"
    assert a["skill_src"] == os.path.join(pp.p1.REPO, ".claude", "skills", "commit.md")
    assert a["carrier_r_src"] == os.path.join(pp.ENCODINGS_DIR, "taskA", "A-R", "vault")
    assert a["carrier_f_src"] == os.path.join(pp.ENCODINGS_DIR, "taskA", "A-F", "vault")


def test_task_b_task_json_round_trips_into_tasks_registry():
    b = pp.TASKS["B"]
    assert b["removal_basenames"] == pp.TASK_B_REMOVAL
    assert b["skill_name"] == "gitignore-narrowing"
    assert b["carrier_r_src"] is None
    assert b["carrier_f_src"] == os.path.join(pp.ENCODINGS_DIR, "taskB", "B-F", "vault")
    assert b["skill_src"] == os.path.join(pp.ENCODINGS_DIR, "taskB", "B-S", "skills", "gitignore-narrowing")


def test_task_key_aliases_resolve_commit_and_gitignore_to_a_and_b():
    assert pp.resolve_task_key("commit") == "A"
    assert pp.resolve_task_key("gitignore") == "B"
    assert pp.resolve_task_key("A") == "A"
    assert pp.resolve_task_key("B") == "B"
    assert pp.resolve_task_key("newtask") == "newtask"


def test_validate_task_key_accepts_alias_and_canonical():
    assert pp.validate_task_key("commit") == "A"
    assert pp.validate_task_key("gitignore") == "B"
    assert pp.validate_task_key("A") == "A"
    assert pp.validate_task_key("B") == "B"


def test_validate_task_key_raises_systemexit_for_unknown_task():
    with pytest.raises(SystemExit):
        pp.validate_task_key("nonexistent-task")


# ----- --baseline mode: arm N summary formatting from synthetic records -----

def test_format_baseline_summary_basic_from_synthetic_records():
    records = [
        {"valid": True, "end_state": True, "followed_all": True, "n_steps": 3,
         "followed_steps": {"1": True, "2": True, "3": True}, "total_cost_usd": 0.10},
        {"valid": True, "end_state": False, "followed_all": False, "n_steps": 3,
         "followed_steps": {"1": True, "2": False, "3": True}, "total_cost_usd": 0.20},
    ]

    summary = pp.format_baseline_summary("gitignore", "sonnet5", records)

    assert "end result 1/2" in summary
    assert "did every step 1/2" in summary
    assert "step 2: 1/2" in summary
    assert "mean cost: $0.15" in summary


def test_format_baseline_summary_excludes_invalid_trials_from_rates_and_cost():
    records = [
        {"valid": True, "end_state": True, "followed_all": True, "n_steps": 2,
         "followed_steps": {"1": True, "2": True}, "total_cost_usd": 0.10},
        {"valid": False, "end_state": False, "followed_all": False, "n_steps": 2,
         "followed_steps": {"1": False, "2": False}, "total_cost_usd": 0.05},
    ]

    summary = pp.format_baseline_summary("gitignore", "sonnet5", records)

    assert "end result 1/1" in summary
    assert "did every step 1/1" in summary
    assert "mean cost: $0.10" in summary


def test_format_baseline_summary_no_valid_trials_reports_zero_over_zero():
    records = [{"valid": False, "end_state": False, "followed_all": False, "n_steps": 0,
                "followed_steps": {}, "total_cost_usd": 0.0}]

    summary = pp.format_baseline_summary("widget", "sonnet5", records)

    assert "end result 0/0" in summary
    assert "did every step 0/0" in summary
    assert "mean cost: $0.00" in summary


def test_baseline_step_miss_counts_counts_false_and_missing_as_misses():
    records = [
        {"valid": True, "followed_steps": {"1": True, "2": False}},
        {"valid": True, "followed_steps": {"1": True}},  # step 2 missing counts as a miss
    ]

    counts = pp.baseline_step_miss_counts(records, n_steps=2)

    assert counts == {1: 0, 2: 2}


def test_baseline_step_miss_counts_ignores_invalid_trials():
    records = [
        {"valid": True, "followed_steps": {"1": True}},
        {"valid": False, "followed_steps": {"1": False}},
    ]

    counts = pp.baseline_step_miss_counts(records, n_steps=1)

    assert counts == {1: 0}


# ----- four candidate phase-2 fixtures: steps.json bash_regex positive/negative coverage -----

def _assert_bash_step(steps, n, positive_cmd, negative_cmd):
    """A step with no `after` dependency: run it in isolation and confirm the positive command
    matches while the negative command does not."""
    step = next(s for s in steps if s["n"] == n)
    pos_events = [_tool_use("Bash", {"command": positive_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps([step], pos_events, repo_path="/does/not/matter")
    assert results[str(n)] is True, f"expected step {n} to match positive command {positive_cmd!r}"
    neg_events = [_tool_use("Bash", {"command": negative_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps([step], neg_events, repo_path="/does/not/matter")
    assert results[str(n)] is False, f"expected step {n} to NOT match negative command {negative_cmd!r}"


def _assert_bash_step_all_registered(task_key):
    """Every registered repo_state pattern in this task's real steps.json must be in
    REPO_STATE_CHECKERS (mirrors test_every_repo_state_pattern_in_real_steps_json_is_registered,
    generalized to a task key beyond the original hardcoded A/B)."""
    for step in pp.load_steps(task_key):
        for sig in _step_signals(step):
            if sig["signal"] == "repo_state":
                assert sig["pattern"] in pp.REPO_STATE_CHECKERS, (
                    f"Task {task_key} step {step['n']} names repo_state pattern "
                    f"{sig['pattern']!r}, which has no registered checker"
                )


# --- gitignore-nested ---

def test_gitignore_nested_steps_json_is_valid_and_registered():
    steps = pp.load_steps("gitignore-nested")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("gitignore-nested")


def test_gitignore_nested_step1_inspect_gitignore_bash_and_negative():
    steps = pp.load_steps("gitignore-nested")
    _assert_bash_step(steps, 1, "cat .gitignore", "cat other.txt")


def test_gitignore_nested_step1_inspect_gitignore_via_native_read():
    steps = pp.load_steps("gitignore-nested")
    step1 = next(s for s in steps if s["n"] == 1)
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_gitignore_nested_step2_check_ignore_bash_and_negative():
    steps = pp.load_steps("gitignore-nested")
    _assert_bash_step(
        steps, 2,
        "git check-ignore -q internal/core/testdata/rapid/big.bin",
        "git status",
    )


def test_gitignore_nested_step4_status_bash_and_negative():
    steps = pp.load_steps("gitignore-nested")
    _assert_bash_step(steps, 4, "git status --porcelain", "git log --oneline")


def test_gitignore_nested_step5_stage_explicit_paths_bash_and_negative():
    steps = pp.load_steps("gitignore-nested")
    _assert_bash_step(
        steps, 5,
        "git add -- .gitignore internal/core/testdata/fixture.json "
        "internal/api/testdata/fixture.json testdata/fixture.json",
        "git add -A",
    )


def test_gitignore_nested_step6_verify_after_step5():
    steps = pp.load_steps("gitignore-nested")
    events = [
        _tool_use("Bash", {"command": "git add -- .gitignore testdata/fixture.json"}, idx=0),
        _tool_use("Bash", {"command": "git diff --cached --name-only"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is True


def test_gitignore_nested_step6_verify_false_when_step5_never_matched():
    steps = pp.load_steps("gitignore-nested")
    events = [_tool_use("Bash", {"command": "git diff --cached --name-only"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is False


def _write_repo_with_gitignore(repo, lines):
    import subprocess
    os.makedirs(repo, exist_ok=True)
    subprocess.run(["git", "init", "-q"], cwd=repo, check=True, capture_output=True)
    with open(os.path.join(repo, ".gitignore"), "w") as f:
        f.write("\n".join(lines) + "\n")
    for path in pp._RAPID_PATHS_NESTED + pp._FIXTURE_JSON_PATHS_NESTED:
        full = os.path.join(repo, path)
        os.makedirs(os.path.dirname(full), exist_ok=True)
        with open(full, "w") as f:
            f.write("x")


def test_gitignore_rapid_ignored_fixture_trackable_all_depths_true_with_leading_globstar(tmp_path):
    repo = str(tmp_path / "repo")
    _write_repo_with_gitignore(repo, ["**/testdata/rapid/", "*.o"])
    assert pp.default_repo_checker(
        "gitignore_rapid_ignored_fixture_trackable_all_depths", repo,
    ) is True


def test_gitignore_rapid_ignored_fixture_trackable_all_depths_false_with_middle_slash_trap(tmp_path):
    """The trap: a middle-slash pattern (`testdata/rapid/`, no leading `**/`) anchors to the
    .gitignore's own directory and stops matching nested paths — runbook 830 step 3."""
    repo = str(tmp_path / "repo")
    _write_repo_with_gitignore(repo, ["testdata/rapid/", "*.o"])
    assert pp.default_repo_checker(
        "gitignore_rapid_ignored_fixture_trackable_all_depths", repo,
    ) is False


def test_gitignore_rapid_ignored_fixture_trackable_all_depths_false_when_fixture_json_still_ignored(tmp_path):
    repo = str(tmp_path / "repo")
    _write_repo_with_gitignore(repo, ["testdata/", "*.o"])
    assert pp.default_repo_checker(
        "gitignore_rapid_ignored_fixture_trackable_all_depths", repo,
    ) is False


# --- test-bite ---

def test_test_bite_steps_json_is_valid_and_registered():
    steps = pp.load_steps("test-bite")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("test-bite")


def test_test_bite_step1_inspect_runner_bash_and_negative():
    steps = pp.load_steps("test-bite")
    _assert_bash_step(steps, 1, "cat runner.py", "cat other.py")


def test_test_bite_step1_inspect_runner_via_native_read():
    steps = pp.load_steps("test-bite")
    step1 = next(s for s in steps if s["n"] == 1)
    events = [_tool_use("Read", {"file_path": "/repo/runner.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_test_bite_step2_write_test_file_tool_path():
    steps = pp.load_steps("test-bite")
    step2 = next(s for s in steps if s["n"] == 2)
    pos_events = [_tool_use("Write", {"file_path": "/repo/tests/test_run_child.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step2], pos_events, repo_path="/does/not/matter")
    assert results["2"] is True
    neg_events = [_tool_use("Write", {"file_path": "/repo/tests/test_other.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step2], neg_events, repo_path="/does/not/matter")
    assert results["2"] is False


def test_test_bite_step3_run_pytest_bash_and_negative():
    steps = pp.load_steps("test-bite")
    _assert_bash_step(steps, 3, "pytest -q tests/test_run_child.py", "python3 -m unittest")


def _write_runner_repo(repo, mutate=False):
    os.makedirs(os.path.join(repo, "tests"), exist_ok=True)
    runner_src = (
        "import json, os, subprocess, sys\n\n"
        "def run_child(env_extra=None):\n"
        "    env = os.environ.copy()\n"
        "    if env_extra:\n"
        "        env.update(env_extra)\n"
        "    script = \"import json, os, sys; sys.stdout.write(json.dumps(dict(os.environ)))\"\n"
        "    result = subprocess.run([sys.executable, '-c', script], env=env,\n"
        "                            capture_output=True, text=True, check=True)\n"
        "    return json.loads(result.stdout)\n"
    )
    if mutate:
        runner_src = runner_src.replace("env.update(env_extra)", "pass")
    with open(os.path.join(repo, "runner.py"), "w") as f:
        f.write(runner_src)
    with open(os.path.join(repo, "conftest.py"), "w") as f:
        f.write("import os, sys\nsys.path.insert(0, os.path.dirname(__file__))\n")
    with open(os.path.join(repo, "tests", "test_run_child.py"), "w") as f:
        f.write(
            "from runner import run_child\n\n"
            "def test_env_extra_reaches_child():\n"
            "    result = run_child({'PROBE_TEST_VAR': 'expected-value'})\n"
            "    assert result.get('PROBE_TEST_VAR') == 'expected-value'\n"
        )


def test_test_bites_true_when_test_fails_on_mutated_runner(tmp_path):
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    _write_runner_repo(repo, mutate=False)
    assert pp.default_repo_checker("test_bites", repo) is True


def test_test_bites_false_when_test_still_passes_on_mutated_runner(tmp_path):
    """A test that only asserts 'no exception raised' passes both shipped and mutated code — it
    does not bite (runbook 838)."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    os.makedirs(os.path.join(repo, "tests"), exist_ok=True)
    runner_src = (
        "import os, subprocess, sys\n\n"
        "def run_child(env_extra=None):\n"
        "    env = os.environ.copy()\n"
        "    if env_extra:\n"
        "        env.update(env_extra)\n"
        "    subprocess.run([sys.executable, '-c', 'pass'], env=env, check=True)\n"
        "    return True\n"
    )
    with open(os.path.join(repo, "runner.py"), "w") as f:
        f.write(runner_src)
    with open(os.path.join(repo, "conftest.py"), "w") as f:
        f.write("import os, sys\nsys.path.insert(0, os.path.dirname(__file__))\n")
    with open(os.path.join(repo, "tests", "test_run_child.py"), "w") as f:
        f.write(
            "from runner import run_child\n\n"
            "def test_no_exception():\n"
            "    run_child({'PROBE_TEST_VAR': 'expected-value'})\n"
        )
    assert pp.default_repo_checker("test_bites", repo) is False


def test_test_bites_false_when_no_test_file():
    assert pp.default_repo_checker("test_bites", "/does/not/exist") is False


# --- history-rewrite ---

def test_history_rewrite_steps_json_is_valid_and_registered():
    steps = pp.load_steps("history-rewrite")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("history-rewrite")


def test_history_rewrite_step1_inspect_history_any_of():
    steps = pp.load_steps("history-rewrite")
    step1 = next(s for s in steps if s["n"] == 1)
    for positive_cmd in ("git log --oneline -- secrets.env", "git log --all"):
        events = [_tool_use("Bash", {"command": positive_cmd}, idx=0)]
        results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
        assert results["1"] is True, positive_cmd
    events = [_tool_use("Bash", {"command": "git status"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_history_rewrite_step2_record_pre_rewrite_tip_bash_and_negative():
    steps = pp.load_steps("history-rewrite")
    _assert_bash_step(steps, 2, "git rev-parse HEAD", "git rev-parse --short HEAD")


def test_history_rewrite_step3_run_rewrite_tool_bash_and_negative():
    steps = pp.load_steps("history-rewrite")
    _assert_bash_step(
        steps, 3,
        "git filter-branch --force --index-filter 'git rm --cached secrets.env' -- --all",
        "git rebase -i HEAD~5",
    )


def test_history_rewrite_step4_fetch_after_step3():
    steps = pp.load_steps("history-rewrite")
    events = [
        _tool_use("Bash", {"command": "git filter-branch -- --all"}, idx=0),
        _tool_use("Bash", {"command": "git fetch origin"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is True


def test_history_rewrite_step4_false_when_step3_never_matched():
    steps = pp.load_steps("history-rewrite")
    events = [_tool_use("Bash", {"command": "git fetch origin"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is False


def test_history_rewrite_step5_force_push_after_step3():
    steps = pp.load_steps("history-rewrite")
    events = [
        _tool_use("Bash", {"command": "git filter-branch -- --all"}, idx=0),
        _tool_use("Bash", {"command": "git push origin main --force-with-lease"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is True


def test_history_rewrite_step5_false_for_plain_push_without_force():
    steps = pp.load_steps("history-rewrite")
    events = [
        _tool_use("Bash", {"command": "git filter-branch -- --all"}, idx=0),
        _tool_use("Bash", {"command": "git push origin main"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is False


def test_history_rewrite_step6_verify_after_step5():
    steps = pp.load_steps("history-rewrite")
    events = [
        _tool_use("Bash", {"command": "git filter-branch -- --all"}, idx=0),
        _tool_use("Bash", {"command": "git push origin main --force-with-lease"}, idx=1),
        _tool_use("Bash", {"command": "git log --oneline -- secrets.env"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is True


def test_history_rewrite_step6_false_when_step5_never_matched():
    steps = pp.load_steps("history-rewrite")
    events = [
        _tool_use("Bash", {"command": "git filter-branch -- --all"}, idx=0),
        _tool_use("Bash", {"command": "git log --oneline -- secrets.env"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is False


# --- bisect-before-fix ---

def test_bisect_before_fix_steps_json_is_valid_and_registered():
    steps = pp.load_steps("bisect-before-fix")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("bisect-before-fix")


def test_bisect_before_fix_step1_run_gate_bash_and_negative():
    steps = pp.load_steps("bisect-before-fix")
    _assert_bash_step(steps, 1, "bash gate.sh", "cat gate.sh")


def test_bisect_before_fix_step1_bare_dot_slash_form():
    steps = pp.load_steps("bisect-before-fix")
    step1 = next(s for s in steps if s["n"] == 1)
    events = [_tool_use("Bash", {"command": "./gate.sh"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_bisect_before_fix_step2_inspect_plan_or_flagged_files_any_of():
    steps = pp.load_steps("bisect-before-fix")
    step2 = next(s for s in steps if s["n"] == 2)
    for positive_cmd in ("cat PLAN.md", "cat foo.py", "grep return bar.py"):
        events = [_tool_use("Bash", {"command": positive_cmd}, idx=0)]
        results, _, _ = pp.evaluate_steps([step2], events, repo_path="/does/not/matter")
        assert results["2"] is True, positive_cmd
    events = [_tool_use("Bash", {"command": "cat README.md"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step2], events, repo_path="/does/not/matter")
    assert results["2"] is False


def test_bisect_before_fix_step3_checkout_parent_bash_and_negative():
    steps = pp.load_steps("bisect-before-fix")
    _assert_bash_step(steps, 3, "git checkout HEAD~1", "git checkout main")


def test_bisect_before_fix_step4_rerun_gate_after_step3():
    steps = pp.load_steps("bisect-before-fix")
    events = [
        _tool_use("Bash", {"command": "git checkout HEAD~1"}, idx=0),
        _tool_use("Bash", {"command": "./gate.sh"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is True


def test_bisect_before_fix_step4_false_when_step3_never_matched():
    steps = pp.load_steps("bisect-before-fix")
    events = [_tool_use("Bash", {"command": "./gate.sh"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is False


def test_bisect_before_fix_step5_return_to_head_after_step4():
    steps = pp.load_steps("bisect-before-fix")
    events = [
        _tool_use("Bash", {"command": "git checkout HEAD~1"}, idx=0),
        _tool_use("Bash", {"command": "./gate.sh"}, idx=1),
        _tool_use("Bash", {"command": "git checkout main"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is True


def test_bisect_before_fix_step5_false_for_unrelated_branch():
    steps = pp.load_steps("bisect-before-fix")
    events = [
        _tool_use("Bash", {"command": "git checkout HEAD~1"}, idx=0),
        _tool_use("Bash", {"command": "./gate.sh"}, idx=1),
        _tool_use("Bash", {"command": "git checkout feature-branch"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is False


def test_bisect_before_fix_step6_fix_bar_py_any_of():
    steps = pp.load_steps("bisect-before-fix")
    step6 = next(s for s in steps if s["n"] == 6)
    events = [_tool_use("Bash", {"command": "sed -i '' '/print(/d' bar.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step6], events, repo_path="/does/not/matter")
    assert results["6"] is True
    events = [_tool_use("Bash", {"command": "sed -i '' 's/x/y/' foo.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step6], events, repo_path="/does/not/matter")
    assert results["6"] is False


def test_bisect_before_fix_step6_fix_bar_py_via_native_edit():
    steps = pp.load_steps("bisect-before-fix")
    step6 = next(s for s in steps if s["n"] == 6)
    events = [_tool_use("Edit", {"file_path": "/repo/bar.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step6], events, repo_path="/does/not/matter")
    assert results["6"] is True


def test_bisect_before_fix_step7_verify_gate_after_step6():
    steps = pp.load_steps("bisect-before-fix")
    events = [
        _tool_use("Bash", {"command": "sed -i '' '/print(/d' bar.py"}, idx=0),
        _tool_use("Bash", {"command": "bash gate.sh"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["7"] is True


def test_bisect_before_fix_step7_false_when_step6_never_matched():
    steps = pp.load_steps("bisect-before-fix")
    events = [_tool_use("Bash", {"command": "bash gate.sh"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["7"] is False


# --- tdd-order ---

def test_tdd_order_steps_json_is_valid_and_registered():
    steps = pp.load_steps("tdd-order")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("tdd-order")


def test_tdd_order_step1_write_test_file_via_native_edit_and_negative():
    steps = pp.load_steps("tdd-order")
    step1 = next(s for s in steps if s["n"] == 1)
    pos_events = [_tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], pos_events, repo_path="/does/not/matter")
    assert results["1"] is True
    neg_events = [_tool_use("Edit", {"file_path": "/repo/slugify.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], neg_events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_tdd_order_step1_write_test_file_bash_and_negative():
    steps = pp.load_steps("tdd-order")
    _assert_bash_step(
        steps, 1,
        "cat > test_slugify_behavior.py << 'EOF'",
        "cat > slugify.py << 'EOF'",
    )


def test_tdd_order_step2_run_pytest_after_step1():
    steps = pp.load_steps("tdd-order")
    events = [
        _tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0),
        _tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is True


def test_tdd_order_step2_false_when_step1_never_matched():
    steps = pp.load_steps("tdd-order")
    events = [_tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is False


def test_tdd_order_step3_edit_impl_after_step2_via_native_edit():
    steps = pp.load_steps("tdd-order")
    events = [
        _tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0),
        _tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=1),
        _tool_use("Edit", {"file_path": "/repo/slugify.py"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["3"] is True


def test_tdd_order_step3_does_not_credit_editing_the_test_file():
    """The implementation-edit step must not be satisfiable by editing test_slugify.py again --
    \\bslugify.py$ must not match test_slugify.py's trailing substring."""
    steps = pp.load_steps("tdd-order")
    step3 = next(s for s in steps if s["n"] == 3)
    events = [_tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step3], events, repo_path="/does/not/matter")
    assert results["3"] is False


def test_tdd_order_step4_run_pytest_after_step3():
    steps = pp.load_steps("tdd-order")
    events = [
        _tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0),
        _tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=1),
        _tool_use("Edit", {"file_path": "/repo/slugify.py"}, idx=2),
        _tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=3),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is True


def test_tdd_order_step4_false_when_step3_never_matched():
    steps = pp.load_steps("tdd-order")
    events = [
        _tool_use("Edit", {"file_path": "/repo/test_slugify.py"}, idx=0),
        _tool_use("Bash", {"command": "python3 -m pytest -q"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is False


def _git(repo, *args):
    import subprocess
    subprocess.run(["git", "-C", repo] + list(args), check=True, capture_output=True)


def _write_tdd_order_repo(tmp_path):
    """A repo laid out like the real fixture: tmp_path/repo (git repo) + tmp_path/.eval/original_tip
    (baseline tip, recorded before any of the caller's additional commits)."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text\n")
    with open(os.path.join(repo, "test_slugify.py"), "w") as f:
        f.write("from slugify import slugify\n\n\ndef test_returns_a_string():\n    assert isinstance(slugify('x'), str)\n")
    _git(repo, "init", "-q", "-b", "main")
    _git(repo, "config", "user.email", "t@example.com")
    _git(repo, "config", "user.name", "Trial")
    _git(repo, "add", "-A")
    _git(repo, "commit", "-q", "-m", "chore: initial slugify stub and existing test")
    import subprocess
    orig_tip = subprocess.run(["git", "-C", repo, "rev-parse", "HEAD"],
                               capture_output=True, text=True, check=True).stdout.strip()
    eval_dir = tmp_path / ".eval"
    eval_dir.mkdir()
    (eval_dir / "original_tip").write_text(orig_tip)
    return repo


_TDD_ORDER_PATTERN = "tdd_order_test_only_commit_precedes_impl"


def test_tdd_order_test_only_commit_precedes_impl_true_for_test_then_impl_commits(tmp_path):
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    _git(repo, "add", "test_slugify.py")
    _git(repo, "commit", "-q", "-m", "test: add slugify behavior spec (RED)")
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text.lower()\n")
    _git(repo, "add", "slugify.py")
    _git(repo, "commit", "-q", "-m", "feat: implement slugify lowercasing")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is True


def test_tdd_order_test_only_commit_precedes_impl_false_for_one_combined_commit(tmp_path):
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text.lower()\n")
    _git(repo, "add", "-A")
    _git(repo, "commit", "-q", "-m", "feat: implement slugify with test")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


def test_tdd_order_test_only_commit_precedes_impl_false_when_no_new_commits(tmp_path):
    repo = _write_tdd_order_repo(tmp_path)
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


def test_tdd_order_test_only_commit_precedes_impl_ignores_non_py_commits(tmp_path):
    """A commit that touches only a non-.py file (mirroring the harness's own CLAUDE.md-only 'add
    project config' commit) must never count toward the total."""
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "CLAUDE.md"), "w") as f:
        f.write("# project config\n")
    _git(repo, "add", "CLAUDE.md")
    _git(repo, "commit", "-q", "-m", "add project config")
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    _git(repo, "add", "test_slugify.py")
    _git(repo, "commit", "-q", "-m", "test: add slugify behavior spec (RED)")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


def test_tdd_order_test_only_commit_precedes_impl_false_when_a_later_commit_mixes_test_and_impl(tmp_path):
    """done_when_checks.sh Check 3: a relevant commit that touches BOTH a test file and an
    implementation .py file fails the signal, even though there are >= 2 relevant commits and the
    first one is test-only."""
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    _git(repo, "add", "test_slugify.py")
    _git(repo, "commit", "-q", "-m", "test: add slugify behavior spec (RED)")
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text.lower()\n")
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_more():\n    assert slugify('X') == 'x'\n")
    _git(repo, "add", "-A")
    _git(repo, "commit", "-q", "-m", "feat: implement slugify and add another test in the same commit")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


def test_tdd_order_test_only_commit_precedes_impl_false_when_first_relevant_commit_overimplements(tmp_path):
    """done_when_checks.sh Check 4 (file-classification half): the FIRST relevant commit must
    touch ONLY test file(s). A first commit that already includes implementation changes
    alongside the test fails, even if a second, later commit is impl-only."""
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text.lower()\n")
    _git(repo, "add", "-A")
    _git(repo, "commit", "-q", "-m", "feat: over-implement in the first commit")
    with open(os.path.join(repo, "slugify.py"), "a") as f:
        f.write("\n")
    _git(repo, "add", "slugify.py")
    _git(repo, "commit", "-q", "-m", "chore: touch impl again")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


def test_tdd_order_test_only_commit_precedes_impl_false_when_first_relevant_commit_is_impl_only(tmp_path):
    """done_when_checks.sh Check 4: implementation-then-test order (impl written first) fails
    even though neither commit mixes test and impl and there are >= 2 relevant commits."""
    repo = _write_tdd_order_repo(tmp_path)
    with open(os.path.join(repo, "slugify.py"), "w") as f:
        f.write("def slugify(text):\n    return text.lower()\n")
    _git(repo, "add", "slugify.py")
    _git(repo, "commit", "-q", "-m", "feat: implement slugify lowercasing first")
    with open(os.path.join(repo, "test_slugify.py"), "a") as f:
        f.write("\n\ndef test_lowercases():\n    assert slugify('Hello') == 'hello'\n")
    _git(repo, "add", "test_slugify.py")
    _git(repo, "commit", "-q", "-m", "test: add slugify behavior spec after the fact")
    assert pp.default_repo_checker(_TDD_ORDER_PATTERN, repo) is False


# --- opsx-propose ---

def test_opsx_propose_steps_json_is_valid_and_registered():
    steps = pp.load_steps("opsx-propose")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("opsx-propose")


def test_opsx_propose_step1_new_change_bash_and_negative():
    steps = pp.load_steps("opsx-propose")
    _assert_bash_step(
        steps, 1,
        "openspec new change add-rate-limiting",
        "openspec status --change add-rate-limiting --json",
    )


def test_opsx_propose_step2_status_after_step1():
    steps = pp.load_steps("opsx-propose")
    events = [
        _tool_use("Bash", {"command": "openspec new change add-rate-limiting"}, idx=0),
        _tool_use("Bash", {"command": "openspec status --change add-rate-limiting --json"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is True


def test_opsx_propose_step2_false_when_step1_never_matched():
    steps = pp.load_steps("opsx-propose")
    events = [_tool_use("Bash", {"command": "openspec status --change add-rate-limiting --json"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is False


_OPSX_NEW_CHANGE_EVENT = _tool_use("Bash", {"command": "openspec new change add-rate-limiting"}, idx=0)


def test_opsx_propose_step3_write_proposal_via_native_write():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["3"] is True

    neg_events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/design.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, neg_events, repo_path="/does/not/matter")
    assert results["3"] is False


def test_opsx_propose_step4_write_design_after_step3():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/design.md"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is True


def test_opsx_propose_step5_write_spec_delta_after_step3():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/specs/widget-api/spec.md"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is True


def test_opsx_propose_step6_write_tasks_after_step5():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/specs/widget-api/spec.md"}, idx=2),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/tasks.md"}, idx=3),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is True


def test_opsx_propose_step6_false_when_step5_never_matched():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/tasks.md"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is False


def test_opsx_propose_step7_final_status_after_step6():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/proposal.md"}, idx=1),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/specs/widget-api/spec.md"}, idx=2),
        _tool_use("Write", {"file_path": "/repo/openspec/changes/add-rate-limiting/tasks.md"}, idx=3),
        _tool_use("Bash", {"command": "openspec status --change add-rate-limiting"}, idx=4),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["7"] is True


def test_opsx_propose_step7_false_when_step6_never_matched():
    steps = pp.load_steps("opsx-propose")
    events = [
        _OPSX_NEW_CHANGE_EVENT,
        _tool_use("Bash", {"command": "openspec status --change add-rate-limiting"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["7"] is False


# --- opsx-archive ---

def test_opsx_archive_steps_json_is_valid_and_registered():
    steps = pp.load_steps("opsx-archive")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig and "pattern" in sig
    _assert_bash_step_all_registered("opsx-archive")


def test_opsx_archive_step1_discover_change_any_of():
    steps = pp.load_steps("opsx-archive")
    step1 = next(s for s in steps if s["n"] == 1)
    for positive_cmd in ("openspec list --json", "openspec status --change add-rate-limiting --json",
                         "ls openspec/changes"):
        events = [_tool_use("Bash", {"command": positive_cmd}, idx=0)]
        results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
        assert results["1"] is True, positive_cmd
    events = [_tool_use("Bash", {"command": "git status"}, idx=0)]
    results, _, _ = pp.evaluate_steps([step1], events, repo_path="/does/not/matter")
    assert results["1"] is False


_OPSX_ARCHIVE_DISCOVER_EVENT = _tool_use("Bash", {"command": "openspec list --json"}, idx=0)


def test_opsx_archive_step2_check_tasks_after_step1():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _tool_use("Bash", {"command": "cat openspec/changes/add-rate-limiting/tasks.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is True


def test_opsx_archive_step2_via_native_read():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _tool_use("Read", {"file_path": "/repo/openspec/changes/add-rate-limiting/tasks.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["2"] is True


def test_opsx_archive_step3_assess_spec_sync_after_step1():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _tool_use("Bash", {"command": "diff openspec/changes/add-rate-limiting/specs/widget-api/spec.md openspec/specs/widget-api/spec.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["3"] is True


def test_opsx_archive_step4_perform_archive_any_of():
    steps = pp.load_steps("opsx-archive")
    for positive_cmd in ("openspec archive add-rate-limiting --yes",
                         "mv openspec/changes/add-rate-limiting openspec/changes/archive/2026-09-12-add-rate-limiting"):
        events = [
            _OPSX_ARCHIVE_DISCOVER_EVENT,
            _tool_use("Bash", {"command": positive_cmd}, idx=1),
        ]
        results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
        assert results["4"] is True, positive_cmd


def test_opsx_archive_step4_false_when_step1_never_matched():
    steps = pp.load_steps("opsx-archive")
    events = [_tool_use("Bash", {"command": "openspec archive add-rate-limiting --yes"}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["4"] is False


_OPSX_ARCHIVE_PERFORM_EVENT = _tool_use("Bash", {"command": "openspec archive add-rate-limiting --yes"}, idx=1)


def test_opsx_archive_step5_verify_main_spec_after_step4():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _OPSX_ARCHIVE_PERFORM_EVENT,
        _tool_use("Bash", {"command": "cat openspec/specs/widget-api/spec.md"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is True


def test_opsx_archive_step5_false_when_step4_never_matched():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _tool_use("Bash", {"command": "cat openspec/specs/widget-api/spec.md"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["5"] is False


def test_opsx_archive_step6_validate_after_step4():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _OPSX_ARCHIVE_PERFORM_EVENT,
        _tool_use("Bash", {"command": "openspec validate --all --strict"}, idx=2),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is True


def test_opsx_archive_step6_false_when_step4_never_matched():
    steps = pp.load_steps("opsx-archive")
    events = [
        _OPSX_ARCHIVE_DISCOVER_EVENT,
        _tool_use("Bash", {"command": "openspec validate --all --strict"}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["6"] is False


# --- route ---
#
# Task 4.4 stage-2 rerun 2 (real-transcript finding, sonnet5 route-R-0, 2026-09-16): a fully
# compliant trial still scored followed_all=False because two of steps.json's real regexes were
# too strict for equally-valid command shapes the agent actually ran:
#
#   step 9 ("all three required tags"): the agent quoted each --tag value per normal shell
#   practice (`--tag "work-kind/foo"`), but the any_of patterns required the tag prefix
#   (`work-kind/`, `tier/`, `outcome/`) immediately after `--tag\s+` with no quote character
#   allowed in between, so a quoted, otherwise-correct invocation never matched any of the six
#   orderings.
#
#   step 10 ("aggregate: query route evidence then amend or learn"): the route runbook's own
#   documented "no prior aggregate exists" branch has the agent create a brand-new aggregate note
#   via `engram learn fact --slug route-evidence-<work-kind> --position top ...` — a real,
#   runbook-sanctioned path, distinct from the amend-an-existing-aggregate path the two any_of
#   alternatives were written for (a single combined query+amend/learn command, or an
#   `--target ... route-evidence` amend). Neither alternative recognized the create-new-aggregate
#   shape.
#
# Both are the same class of gap task 4.1 found in history-rewrite's step 3 (#754: the checker's
# regex not recognizing an equally-valid alternate command shape) — fixed here by widening the
# fixture's regex/any_of, not by asking the agent to type something differently.

def test_route_steps_json_is_valid_and_registered():
    """Unlike the other fixtures' _step_signals checks, route mixes bash_regex (`pattern`) and
    tool_input_regex (`regex`, on the Agent dispatch handoff) signals, so each signal is checked
    for its own signal-appropriate key rather than assuming `pattern` universally."""
    steps = pp.load_steps("route")
    assert isinstance(steps, list) and len(steps) > 0
    for step in steps:
        assert "n" in step
        for sig in _step_signals(step):
            assert "signal" in sig
            key = "pattern" if sig["signal"] in ("bash_regex", "tool_path", "repo_state") else "regex"
            assert key in sig, f"step {step['n']} signal {sig['signal']!r} missing {key!r}"
    _assert_bash_step_all_registered("route")


def test_route_step8_write_evidence_note_bash_and_negative():
    steps = pp.load_steps("route")
    _assert_bash_step(
        steps, 8,
        'engram learn fact --slug route-dispatch-cli-flag-implementation --tag work-kind/x '
        '--tag tier/cheap --tag outcome/pass',
        "engram learn fact --slug route-evidence-cli-flag-implementation",
    )


_ROUTE_EVIDENCE_WRITE_EVENT = _tool_use(
    "Bash",
    {"command": 'engram learn fact --slug route-dispatch-cli-flag-implementation --position top'},
    idx=0,
)


def test_route_step9_tags_match_with_quoted_values_real_transcript_shape():
    """Real route-R-0 transcript (task 4.4 stage-2 rerun 2): tag values are quoted per normal
    shell practice. Before the fix, no any_of alternative tolerated the quote character between
    `--tag` and the tag prefix."""
    steps = pp.load_steps("route")
    step9 = next(s for s in steps if s["n"] == 9)
    quoted_cmd = (
        'engram learn fact --slug route-dispatch-cli-flag-implementation --position top '
        '--tag "work-kind/cli-flag-implementation" --tag "tier/cheap" --tag "outcome/pass" '
        '--source "route dispatch record" --situation "routing cli-flag-implementation work" '
        '--subject "cli-flag-implementation dispatch at cheap (haiku)" --predicate "resolved as" '
        '--object "pass per review verdict"'
    )
    events = [_tool_use("Bash", {"command": quoted_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps([step9], events, repo_path="/does/not/matter")
    assert results["9"] is True


def test_route_step9_tags_still_match_unquoted_values():
    steps = pp.load_steps("route")
    step9 = next(s for s in steps if s["n"] == 9)
    unquoted_cmd = (
        "engram learn fact --slug route-dispatch-x --tag work-kind/x --tag tier/cheap "
        "--tag outcome/pass"
    )
    events = [_tool_use("Bash", {"command": unquoted_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps([step9], events, repo_path="/does/not/matter")
    assert results["9"] is True


def test_route_step9_false_when_a_required_tag_is_missing():
    steps = pp.load_steps("route")
    step9 = next(s for s in steps if s["n"] == 9)
    missing_outcome_cmd = 'engram learn fact --tag "work-kind/x" --tag "tier/cheap"'
    events = [_tool_use("Bash", {"command": missing_outcome_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps([step9], events, repo_path="/does/not/matter")
    assert results["9"] is False


def test_route_step10_create_new_aggregate_after_step8_real_transcript_shape():
    """Real route-R-0 transcript: no prior route-evidence-<work-kind> aggregate existed, so the
    agent correctly took the route runbook's own documented create-new-aggregate branch
    (`engram learn fact --slug route-evidence-<work-kind> --position top ...`) rather than
    amending an existing one. Before the fix, neither any_of alternative recognized this shape."""
    steps = pp.load_steps("route")
    create_aggregate_cmd = (
        'engram learn fact --slug route-evidence-cli-flag-implementation --position top '
        '--source "route dispatch record" '
        '--situation "routing cli-flag-implementation work: which tier the evidence supports" '
        '--subject "route evidence for cli-flag-implementation" --predicate "tallies" '
        '--object "cheap 1/1 as of 2026-09-16 -- evidence: '
        '940.2026-09-16.route-dispatch-cli-flag-implementation"'
    )
    events = [
        _ROUTE_EVIDENCE_WRITE_EVENT,
        _tool_use("Bash", {"command": create_aggregate_cmd}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["10"] is True


def test_route_step10_amend_existing_aggregate_still_matches():
    steps = pp.load_steps("route")
    amend_cmd = (
        'engram amend --target 940.2026-09-16.route-evidence-cli-flag-implementation '
        '--object "cheap 2/2 as of 2026-09-16"'
    )
    events = [
        _ROUTE_EVIDENCE_WRITE_EVENT,
        _tool_use("Bash", {"command": amend_cmd}, idx=1),
    ]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["10"] is True


def test_route_step10_false_when_step8_never_matched():
    steps = pp.load_steps("route")
    create_aggregate_cmd = 'engram learn fact --slug route-evidence-cli-flag-implementation --position top'
    events = [_tool_use("Bash", {"command": create_aggregate_cmd}, idx=0)]
    results, _, _ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["10"] is False


# ----- rate-limited stub detection (vault note 988a) -----

# Compact-JSON snippet matching the real transcript bytes verified in
# results/baseline_sonnet5_opsx-archive.rate-limited.jsonl (no whitespace around ":").
_RATE_LIMIT_STUB_TRANSCRIPT_TEXT = (
    '{"type":"assistant","message":{"content":[{"type":"text",'
    '"text":"You\'ve hit your session limit · resets 1pm (America/Detroit)"}]},'
    '"error":"rate_limit","isApiErrorMessage":true,"apiErrorStatus":429}'
)


def test_rate_limit_signal_present_via_error_field_in_raw_text():
    assert pp._rate_limit_signal_present(None, '{"error":"rate_limit"}') is True


def test_rate_limit_signal_present_via_api_error_status_in_raw_text():
    assert pp._rate_limit_signal_present(None, '{"apiErrorStatus":429}') is True


def test_rate_limit_signal_present_via_session_limit_message_text():
    assert pp._rate_limit_signal_present(None, "You've hit your session limit · resets 1pm") is True


def test_rate_limit_signal_present_via_result_dict_error_field():
    assert pp._rate_limit_signal_present({"error": "rate_limit"}, "") is True


def test_rate_limit_signal_present_via_result_dict_api_error_status():
    assert pp._rate_limit_signal_present({"apiErrorStatus": 429}, "") is True


def test_rate_limit_signal_absent_for_ordinary_transcript():
    assert pp._rate_limit_signal_present({"total_cost_usd": 0.5}, '{"type":"assistant"}') is False


def test_is_rate_limited_stub_true_for_real_stub_shape():
    """The real stub shape (vault note 988a / results/*.rate-limited.jsonl): 1 turn, $0.00, the
    rate-limit signal present."""
    assert pp.is_rate_limited_stub(1, 0.0, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is True


def test_is_rate_limited_stub_true_when_num_turns_missing():
    """num_turns=None (never recorded) is treated as 0 turns — still a stub when cost is zero and
    the signal is present."""
    assert pp.is_rate_limited_stub(None, 0.0, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is True


def test_is_rate_limited_stub_false_when_real_work_was_done():
    """The real completed shape (results/baseline_sonnet5_opsx-propose.rate-limited.jsonl, lines
    1-2): 25 turns, $0.667 spent — the rate-limit signal shows up near the transcript's tail (the
    account limit was tripped on a final wrap-up turn) but substantial real work already happened.
    Must NOT be classified as a stub."""
    assert pp.is_rate_limited_stub(25, 0.667, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is False


def test_is_rate_limited_stub_false_when_no_rate_limit_signal():
    """Zero turns/cost alone, with no rate-limit signal anywhere, is out of scope (task item 5) —
    never classified as a rate-limit stub."""
    assert pp.is_rate_limited_stub(1, 0.0, None, '{"type":"assistant","message":{}}') is False


def test_is_rate_limited_stub_false_for_zero_cost_but_multiple_turns():
    assert pp.is_rate_limited_stub(3, 0.0, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is False


# ----- classify_validity (shared by live scoring and --rescore) -----

def test_classify_validity_rate_limited_overrides_marker_seen_true():
    """A rate-limited stub's trial repo carries the marker (its CLAUDE.md was committed before
    claude was ever spawned) — marker_seen=True must NOT make it valid."""
    valid, reason, api_error = pp.classify_validity(True, 1, 0.0, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT)
    assert valid is False
    assert reason == "rate_limit"


def test_classify_validity_no_marker_when_not_rate_limited():
    valid, reason, api_error = pp.classify_validity(False, 5, 0.3, None, '{"type":"assistant"}')
    assert valid is False
    assert reason == "no_marker"


def test_classify_validity_valid_when_marker_seen_and_not_rate_limited():
    valid, reason, api_error = pp.classify_validity(True, 5, 0.3, None, '{"type":"assistant"}')
    assert valid is True
    assert reason is None


def test_classify_validity_rate_limit_truncated_for_real_work_then_signal():
    """Mirrors the opsx-propose real-completed-looking records (vault note 988a.2026-09-12
    follow-up): marker seen, substantial real work (25 turns, $0.667), but the rate-limit signal
    is present near the tail — the account limit cut the session off mid-task on a later turn.
    Not a zero-work stub (is_rate_limited_stub is False here), but still not a genuine completed
    trial: must be invalid with a DISTINCT reason from the zero-work stub's 'rate_limit', so a
    truncation is never conflated with (or silently absorbed into) a real pass/fail verdict."""
    valid, reason, api_error = pp.classify_validity(True, 25, 0.667, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT)
    assert valid is False
    assert reason == "rate_limit_truncated"
    assert pp.is_rate_limited_stub(25, 0.667, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is False


def test_classify_validity_rate_limit_truncated_distinct_from_zero_work_stub_reason():
    stub_valid, stub_reason, stub_error = pp.classify_validity(True, 1, 0.0, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT)
    truncated_valid, truncated_reason, truncated_error = pp.classify_validity(
        True, 18, 0.4975, None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT)
    assert stub_valid is False and truncated_valid is False
    assert stub_reason == "rate_limit"
    assert truncated_reason == "rate_limit_truncated"
    assert stub_reason != truncated_reason


# Real false positive (found rescoring results/baseline_sonnet5_opsx-propose.jsonl records #0 and
# #7 -- both 30+ turns, real cost, end_state=True, followed_all=True): a trial's background vault
# includes real memory notes, and vault note 988a's own body NARRATES a past rate-limit incident
# in prose -- "a session that hit the account's 5-hour limit ('You've hit your session limit ·
# resets 1pm', transcript error rate_limit / apiErrorStatus 429, 1 turn, $0.00)". When an agent's
# `engram query` tool_result surfaces that note during a genuinely successful trial, the raw
# transcript text contains "hit your session limit" with NEITHER `"isApiErrorMessage":true` NOR
# `"apiErrorStatus":429` in the real compact-JSON form (the prose spells it "apiErrorStatus 429",
# no quotes/colon) -- this must never be classified as a truncation.
_RECALLED_NOTE_PROSE_ABOUT_RATE_LIMIT = (
    '{"type":"user","message":{"content":[{"type":"tool_result","content":'
    '"situation: scoring headless claude -p eval trials; behavior: the trial validity gate keyed '
    "only on the CLAUDE.md marker being seen; a session that hit the account's 5-hour limit "
    "('You\\u2019ve hit your session limit \\u00b7 resets 1pm', transcript error rate_limit / "
    'apiErrorStatus 429, 1 turn, $0.00) still shows the marker"}]}}'
)


def test_classify_validity_not_truncated_when_signal_is_only_recalled_note_prose():
    """Guards the exact false positive found while rescoring the good opsx-propose relaunch: a
    fully-completed trial (many turns, real cost, marker seen) whose transcript merely QUOTES a
    vault note's prose about a past rate-limit incident must stay valid — never
    'rate_limit_truncated'."""
    valid, reason, api_error = pp.classify_validity(True, 32, 0.84, None, _RECALLED_NOTE_PROSE_ABOUT_RATE_LIMIT)
    assert valid is True
    assert reason is None


def test_rate_limit_truncation_signal_absent_for_recalled_note_prose_alone():
    assert pp._rate_limit_truncation_signal_present(None, _RECALLED_NOTE_PROSE_ABOUT_RATE_LIMIT) is False
    # Sanity: the loose, turn/cost-gated helper DOES fire on this text (why the stricter check
    # exists in the first place — a truncation check has no turn/cost gate to fall back on).
    assert pp._rate_limit_signal_present(None, _RECALLED_NOTE_PROSE_ABOUT_RATE_LIMIT) is True


def test_rate_limit_truncation_signal_present_for_real_system_record():
    assert pp._rate_limit_truncation_signal_present(None, _RATE_LIMIT_STUB_TRANSCRIPT_TEXT) is True


# ----- format_baseline_summary: rate-limited suffix (task item 3) -----

def _rl_record(valid, invalid_reason=None, end_state=False, followed_all=False, n_steps=6,
               total_cost_usd=0.1, stalled_asking=False):
    return {
        "valid": valid, "invalid_reason": invalid_reason, "end_state": end_state,
        "followed_all": followed_all, "followed_steps": {}, "n_steps": n_steps,
        "total_cost_usd": total_cost_usd, "stalled_asking": stalled_asking,
    }


def test_format_baseline_summary_appends_rate_limited_suffix_when_present():
    records = (
        [_rl_record(True, end_state=True, followed_all=True) for _ in range(2)]
        + [_rl_record(False, invalid_reason="rate_limit") for _ in range(6)]
    )
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "end result 2/2, did every step 2/2 (api-errors: 6)" in first_line


def test_format_baseline_summary_omits_rate_limited_suffix_when_zero():
    records = [_rl_record(True, end_state=True, followed_all=True) for _ in range(3)]
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "api-errors" not in first_line


def test_format_baseline_summary_rate_limited_trials_excluded_from_denominator():
    """The rate-limited trials are invalid — they must not count toward the valid_n denominator
    (they are surfaced only via the suffix count)."""
    records = (
        [_rl_record(True, end_state=True, followed_all=True)]
        + [_rl_record(False, invalid_reason="rate_limit") for _ in range(8)]
    )
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "end result 1/1, did every step 1/1 (api-errors: 8)" in first_line


def test_format_baseline_summary_counts_rate_limit_truncated_in_same_suffix():
    """A mid-task session-limit truncation (invalid_reason == 'rate_limit_truncated') counts in
    the SAME '(api-errors: K)' suffix as a zero-work stub ('rate_limit') — the suffix reports
    total rate-limit-related invalidations, not just the stub subset; nothing else about the
    summary format changes."""
    records = (
        [_rl_record(True, end_state=True, followed_all=True)]
        + [_rl_record(False, invalid_reason="rate_limit") for _ in range(6)]
        + [_rl_record(False, invalid_reason="rate_limit_truncated") for _ in range(2)]
    )
    summary = pp.format_baseline_summary("opsx-propose", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "end result 1/1, did every step 1/1 (api-errors: 8)" in first_line


# ----- aggregate()/format_table(): rate-limited suffix in --summarize (task item 3) -----

def test_aggregate_reports_rate_limited_n():
    records = [
        {"task": "A", "arm": "R", "valid": True, "invalid_reason": None, "found": True,
         "end_state": True, "followed_all": True, "followed_k": 6, "n_steps": 6,
         "recall_fired": False, "total_cost_usd": 0.3, "duration_ms": 500},
        {"task": "A", "arm": "R", "valid": False, "invalid_reason": "rate_limit", "found": None,
         "end_state": False, "followed_all": False, "followed_k": 0, "n_steps": 0,
         "recall_fired": False, "total_cost_usd": 0.0, "duration_ms": 100},
    ]
    agg = pp.aggregate(records, "A", "R")
    assert agg["valid_n"] == 1
    assert agg["api_error_n"] == 1


def test_aggregate_counts_rate_limit_truncated_alongside_stub_in_rate_limited_n():
    records = [
        {"task": "A", "arm": "N", "valid": False, "invalid_reason": "rate_limit", "found": None,
         "end_state": False, "followed_all": False, "followed_k": 0, "n_steps": 0,
         "recall_fired": False, "total_cost_usd": 0.0, "duration_ms": 100},
        {"task": "A", "arm": "N", "valid": False, "invalid_reason": "rate_limit_truncated",
         "found": None, "end_state": False, "followed_all": False, "followed_k": 0, "n_steps": 0,
         "recall_fired": False, "total_cost_usd": 0.667, "duration_ms": 900000},
    ]
    agg = pp.aggregate(records, "A", "N")
    assert agg["valid_n"] == 0
    assert agg["api_error_n"] == 2


def test_format_table_valid_row_appends_rate_limited_suffix_when_present():
    agg = {"S": _agg(9, 1, 1, 1, 1)}
    agg["S"]["api_error_n"] = 8
    table = pp.format_table("A", agg)
    valid_row = next(line for line in table.splitlines() if line.startswith("valid (n)"))
    assert "1/9 (api-errors: 8)" in valid_row


def test_format_table_valid_row_omits_suffix_when_no_rate_limited_trials():
    agg = {"S": _agg(5, 5, 4, 4, 4)}
    table = pp.format_table("A", agg)
    valid_row = next(line for line in table.splitlines() if line.startswith("valid (n)"))
    assert "api-errors" not in valid_row


# ----- stalled_asking aggregation in format_baseline_summary -----

def test_format_baseline_summary_appends_stalled_suffix_when_present():
    records = (
        [_rl_record(True, end_state=True, followed_all=True) for _ in range(3)]
        + [_rl_record(True, end_state=True, followed_all=True, stalled_asking=True) for _ in range(2)]
    )
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "end result 5/5, did every step 5/5 (stalled: 2)" in first_line


def test_format_baseline_summary_omits_stalled_suffix_when_zero():
    records = [_rl_record(True, end_state=True, followed_all=True) for _ in range(3)]
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "stalled" not in first_line


def test_format_baseline_summary_stalled_and_rate_limited_both_shown():
    """When both rate-limited and stalled trials are present, both suffixes should appear."""
    records = (
        [_rl_record(True, end_state=True, followed_all=True) for _ in range(2)]
        + [_rl_record(True, end_state=True, followed_all=True, stalled_asking=True) for _ in range(2)]
        + [_rl_record(False, invalid_reason="rate_limit") for _ in range(1)]
    )
    summary = pp.format_baseline_summary("opsx-archive", "sonnet5", records)
    first_line = summary.splitlines()[0]
    assert "end result 4/4, did every step 4/4 (api-errors: 1) (stalled: 2)" in first_line


# ----- aggregate() stalled_n computation -----

def test_aggregate_reports_stalled_n():
    records = [
        {"task": "A", "arm": "R", "valid": True, "invalid_reason": None, "found": True,
         "end_state": True, "followed_all": True, "followed_k": 6, "n_steps": 6,
         "recall_fired": False, "stalled_asking": False, "total_cost_usd": 0.3, "duration_ms": 500},
        {"task": "A", "arm": "R", "valid": True, "invalid_reason": None, "found": True,
         "end_state": True, "followed_all": True, "followed_k": 6, "n_steps": 6,
         "recall_fired": False, "stalled_asking": True, "total_cost_usd": 0.3, "duration_ms": 500},
    ]
    agg = pp.aggregate(records, "A", "R")
    assert agg["valid_n"] == 2
    assert agg["stalled_n"] == 1


def test_aggregate_stalled_only_counts_valid_records():
    """Stalled should only count records where valid=True; rate-limited (invalid) records
    should not be counted."""
    records = [
        {"task": "A", "arm": "N", "valid": True, "invalid_reason": None, "found": None,
         "end_state": False, "followed_all": False, "followed_k": 0, "n_steps": 0,
         "recall_fired": False, "stalled_asking": True, "total_cost_usd": 0.1, "duration_ms": 100},
        {"task": "A", "arm": "N", "valid": False, "invalid_reason": "rate_limit", "found": None,
         "end_state": False, "followed_all": False, "followed_k": 0, "n_steps": 0,
         "recall_fired": False, "stalled_asking": True, "total_cost_usd": 0.0, "duration_ms": 100},
    ]
    agg = pp.aggregate(records, "A", "N")
    assert agg["valid_n"] == 1
    assert agg["stalled_n"] == 1


# ----- aggregate() question_stop_n / scoreable_n (task 4.1 item 4, design.md D8) -----

def _rec(arm="R", found=True, end_state=True, followed_all=True, followed_k=6, n_steps=6,
         restated_as_plan=True, question_stop=False, valid=True, invalid_reason=None):
    return {
        "task": "A", "arm": arm, "valid": valid, "invalid_reason": invalid_reason,
        "found": found, "end_state": end_state, "followed_all": followed_all,
        "followed_k": followed_k, "n_steps": n_steps, "recall_fired": False,
        "stalled_asking": False, "restated_as_plan": restated_as_plan,
        "question_stop": question_stop, "total_cost_usd": 0.3, "duration_ms": 500,
    }


def test_aggregate_reports_question_stop_n():
    records = [_rec(question_stop=False), _rec(question_stop=True)]
    agg = pp.aggregate(records, "A", "R")
    assert agg["valid_n"] == 2
    assert agg["question_stop_n"] == 1


def test_aggregate_question_stop_excludes_trial_from_found_and_end_state_and_followed_all():
    """A question_stop=True trial with found/end_state/followed_all all False (it stopped
    instead of finishing) must NOT drag those rates down — it is excluded from scoreable_n
    entirely, per design.md D8 (tasks.md 4.1 item 4: never counted as a failure)."""
    records = [
        _rec(found=True, end_state=True, followed_all=True, question_stop=False),
        _rec(found=False, end_state=False, followed_all=False, question_stop=True),
    ]
    agg = pp.aggregate(records, "A", "R")
    assert agg["valid_n"] == 2
    assert agg["scoreable_n"] == 1
    assert agg["found_n"] == 1
    assert agg["end_state_n"] == 1
    assert agg["followed_all_n"] == 1


def test_aggregate_reports_restated_as_plan_n_excluding_question_stop_trials():
    records = [
        _rec(restated_as_plan=True, question_stop=False),
        _rec(restated_as_plan=False, question_stop=True),
    ]
    agg = pp.aggregate(records, "A", "R")
    assert agg["restated_as_plan_n"] == 1
    assert agg["scoreable_n"] == 1


def test_aggregate_scoreable_n_equals_valid_n_when_no_question_stops():
    """Backward-compatible default: with no question_stop trial at all, scoreable_n and valid_n
    are identical, so every pre-existing found/end_state/followed_all number is unchanged."""
    records = [_rec(), _rec(), _rec()]
    agg = pp.aggregate(records, "A", "R")
    assert agg["scoreable_n"] == agg["valid_n"] == 3


# ----- decomposition()/format_table() use scoreable_n, not valid_n, for these ratios -----

def test_decomposition_shim_rate_uses_scoreable_n_not_valid_n():
    agg = {"R": _agg(5, 5, 3, 3, 3)}
    agg["R"]["scoreable_n"] = 4  # one of the 5 valid trials was a question-stop, excluded
    agg["R"]["question_stop_n"] = 1
    frame = pp.decomposition(agg)
    assert frame["shim_rate_R"] == "3/4"


def test_decomposition_falls_back_to_valid_n_when_scoreable_n_absent():
    """A hand-built agg dict that predates the question_stop exclusion (no scoreable_n key) must
    behave exactly as before — _scoreable_n falls back to valid_n."""
    agg = {"R": _agg(5, 5, 3, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["shim_rate_R"] == "3/5"


def test_format_table_found_row_uses_scoreable_n_when_present():
    agg = {"R": _agg(5, 5, 3, 3, 3)}
    agg["R"]["scoreable_n"] = 4
    table = pp.format_table("A", agg)
    found_row = next(line for line in table.splitlines() if line.startswith("FOUND"))
    assert "3/4" in found_row


def test_format_table_valid_row_appends_question_stop_suffix_when_present():
    agg = {"S": _agg(5, 5, 4, 4, 4)}
    agg["S"]["question_stop_n"] = 2
    table = pp.format_table("A", agg)
    valid_row = next(line for line in table.splitlines() if line.startswith("valid (n)"))
    assert "5/5 (question-stops: 2)" in valid_row


def test_format_table_shows_restated_as_plan_row():
    agg = {"R": _agg(5, 5, 3, 3, 3)}
    agg["R"]["restated_as_plan_n"] = 2
    table = pp.format_table("A", agg)
    restated_row = next(line for line in table.splitlines() if line.startswith("restated as plan"))
    assert "2/5" in restated_row


# ----- format_baseline_summary()/baseline_step_miss_counts() exclude question-stop trials -----

def test_format_baseline_summary_excludes_question_stop_from_rates():
    records = [
        {"valid": True, "invalid_reason": None, "end_state": True, "followed_all": True,
         "n_steps": 2, "followed_steps": {"1": True, "2": True}, "question_stop": False,
         "total_cost_usd": 0.1},
        {"valid": True, "invalid_reason": None, "end_state": False, "followed_all": False,
         "n_steps": 2, "followed_steps": {}, "question_stop": True, "total_cost_usd": 0.1},
    ]
    summary = pp.format_baseline_summary("history-rewrite", "sonnet", records)
    first_line = summary.splitlines()[0]
    assert "end result 1/1" in first_line
    assert "did every step 1/1" in first_line
    assert "question-stops: 1" in first_line


def test_baseline_step_miss_counts_excludes_question_stop_trials():
    records = [
        {"valid": True, "question_stop": False, "followed_steps": {"1": False}},
        {"valid": True, "question_stop": True, "followed_steps": {"1": False}},
    ]
    counts = pp.baseline_step_miss_counts(records, 1)
    assert counts[1] == 1  # only the non-question-stop trial's miss is counted


# ----- format_table valid row with stalled suffix -----

def test_format_table_valid_row_appends_stalled_suffix_when_present():
    agg = {"S": _agg(5, 5, 4, 4, 4, stalled_n=2)}
    table = pp.format_table("A", agg)
    valid_row = next(line for line in table.splitlines() if line.startswith("valid (n)"))
    assert "5/5 (stalled: 2)" in valid_row


def test_format_table_valid_row_appends_both_suffixes_when_both_present():
    agg = {"S": _agg(10, 8, 4, 4, 4, api_error_n=2, stalled_n=1)}
    table = pp.format_table("A", agg)
    valid_row = next(line for line in table.splitlines() if line.startswith("valid (n)"))
    assert "8/10 (api-errors: 2, stalled: 1)" in valid_row


# ----- --rescore reclassifies rate-limited stubs retroactively (task item 4) -----

def _stub_transcript_lines(session_id="rl-session"):
    return [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-12T16:06:36.440Z",
            "message": {"content": [
                {"type": "text", "text": "You've hit your session limit · resets 1pm (America/Detroit)"}
            ]},
            "error": "rate_limit", "isApiErrorMessage": True, "apiErrorStatus": 429,
            "sessionId": session_id,
        }),
    ]


def test_rescore_reclassifies_rate_limited_stub_as_invalid(tmp_path):
    """Reproduces the real stub shape: marker_seen would be True (CLAUDE.md carries the marker
    regardless), num_turns=1, total_cost_usd=0.0, and the transcript shows the rate-limit
    signal — --rescore must flip valid to False and set invalid_reason to 'rate_limit'."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    marker = "RUNBOOK-VS-SKILL-BASELINE-stub1"
    with open(os.path.join(repo, "CLAUDE.md"), "w") as f:
        f.write(f"some guidance\n\nPROBE-TOKEN: {marker}\n")

    transcript_path = tmp_path / "session.jsonl"
    with open(transcript_path, "w") as f:
        f.write("\n".join(_stub_transcript_lines()) + "\n")

    record = {
        "task": "opsx-archive", "arm": "N", "repo_path": repo,
        "transcript_path": str(transcript_path), "carrier_basename": None,
        "valid": True, "marker_seen": True, "invalid_reason": None,
        "num_turns": 1, "total_cost_usd": 0.0,
        "found": None, "found_index": None,
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")
    rescored_path = tmp_path / "rescored.jsonl"

    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["valid"] is False
    assert rescored["invalid_reason"] == "rate_limit"
    # cost/turns are kept as observed, not zeroed out or dropped by rescore.
    assert rescored["num_turns"] == 1
    assert rescored["total_cost_usd"] == 0.0


def test_rescore_marks_truncated_real_work_trial_invalid_with_distinct_reason(tmp_path):
    """A trial that did substantial real work (many turns, real cost) but whose transcript carries
    the rate-limit text near its tail (the account limit cut the session off mid-task on a later
    turn, not on turn 1) must be marked invalid with invalid_reason 'rate_limit_truncated' —
    distinct from the zero-work stub's 'rate_limit' — never left valid: a session cut off
    mid-task never got the chance to finish, so its FOLLOWED/END-STATE would misrepresent the
    trial as a genuine failure (vault note 988a.2026-09-12 follow-up: the first two records of
    results/baseline_sonnet5_opsx-propose.rate-limited.jsonl were exactly this shape)."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    marker = "RUNBOOK-VS-SKILL-BASELINE-real1"
    with open(os.path.join(repo, "CLAUDE.md"), "w") as f:
        f.write(f"some guidance\n\nPROBE-TOKEN: {marker}\n")

    transcript_path = tmp_path / "session.jsonl"
    lines = [
        json.dumps({
            "type": "user", "timestamp": "2026-09-12T16:00:00.000Z",
            "message": {"content": [{"type": "text", "text": f"PROBE-TOKEN: {marker}"}]},
        }),
    ] + _stub_transcript_lines(session_id="real-session")
    with open(transcript_path, "w") as f:
        f.write("\n".join(lines) + "\n")

    record = {
        "task": "opsx-propose", "arm": "N", "repo_path": repo,
        "transcript_path": str(transcript_path), "carrier_basename": None,
        "valid": True, "marker_seen": True, "invalid_reason": None,
        "num_turns": 25, "total_cost_usd": 0.667,
        "found": None, "found_index": None,
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")
    rescored_path = tmp_path / "rescored.jsonl"

    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["valid"] is False
    assert rescored["invalid_reason"] == "rate_limit_truncated"
    # cost/turns are kept as observed, not zeroed out or dropped by rescore.
    assert rescored["num_turns"] == 25
    assert rescored["total_cost_usd"] == 0.667


# ----- env passing to check_end_state_phase2 -----

def test_check_end_state_phase2_passes_env_to_subprocess(tmp_path):
    """check_end_state_phase2 with an env parameter passes it to the subprocess.run call."""
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path, exist_ok=True)

    # Create a dummy checks script that echoes the ENGRAM_VAULT_PATH env var
    checks_script = tmp_path / "checks.sh"
    checks_script.write_text("#!/bin/bash\nif [ -z \"$ENGRAM_VAULT_PATH\" ]; then echo 'FAIL: ENGRAM_VAULT_PATH not set'; exit 1; fi\nexit 0\n")
    checks_script.chmod(0o755)

    # Create a mock TASKS entry with this script
    mock_task_key = "test_route_task"
    original_tasks = pp.TASKS
    try:
        pp.TASKS[mock_task_key] = {
            "done_when_script": str(checks_script),
        }

        # Test without env: should fail
        success, output = pp.check_end_state_phase2(mock_task_key, repo_path, env=None)
        assert success is False
        assert "ENGRAM_VAULT_PATH not set" in output

        # Test with env containing ENGRAM_VAULT_PATH: should pass
        test_env = os.environ.copy()
        test_env["ENGRAM_VAULT_PATH"] = str(tmp_path / "vault")
        success, output = pp.check_end_state_phase2(mock_task_key, repo_path, env=test_env)
        assert success is True
    finally:
        pp.TASKS = original_tasks


# ----- detect_stalled_asking -----

def test_detect_stalled_asking_true_when_last_assistant_ends_with_question(tmp_path):
    transcript_path = tmp_path / "transcript.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "text", "text": "Should I proceed with this change?"}]}
        }),
    ]
    transcript_path.write_text("\n".join(lines) + "\n")

    result = pp.detect_stalled_asking([str(transcript_path)])
    assert result is True


def test_detect_stalled_asking_false_when_last_assistant_ends_with_period(tmp_path):
    transcript_path = tmp_path / "transcript.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "text", "text": "I have completed the work."}]}
        }),
    ]
    transcript_path.write_text("\n".join(lines) + "\n")

    result = pp.detect_stalled_asking([str(transcript_path)])
    assert result is False


def test_detect_stalled_asking_false_when_no_transcripts():
    result = pp.detect_stalled_asking([])
    assert result is False


def test_detect_stalled_asking_finds_last_assistant_among_multiple_messages(tmp_path):
    transcript_path = tmp_path / "transcript.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "text", "text": "First message?"}]}
        }),
        json.dumps({
            "type": "user", "timestamp": "2026-09-11T00:00:01.000Z",
            "message": {"content": [{"type": "text", "text": "Do it"}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:02.000Z",
            "message": {"content": [{"type": "text", "text": "Done."}]}
        }),
    ]
    transcript_path.write_text("\n".join(lines) + "\n")

    result = pp.detect_stalled_asking([str(transcript_path)])
    assert result is False  # Last assistant message ends with period


# ----- fixture regression tests -----

def test_fixture_task_json_removal_basenames_not_clobbered():
    """Regression test: task.json files with carrier keys must have non-empty removal_basenames.

    When a fixture has carrier keys (skill_src, carrier_r_src, carrier_f_src), the removal_basenames
    array holds the list of notes to exclude from the reference corpus. This array must not be
    empty (a regression from commit 17d49a32 where carrier wiring rewrote the file and clobbered
    it to []). Additionally, if the real vault exists at /Users/joe/.local/share/engram/vault,
    every basename in removal_basenames must exist as <basename>.md in that vault (skip this
    validation if the vault path is absent, to allow CI environments without the vault).
    """
    fixtures_dir = os.path.join(HERE, "fixtures")
    assert os.path.isdir(fixtures_dir), f"fixtures directory not found at {fixtures_dir}"

    real_vault = "/Users/joe/.local/share/engram/vault"
    vault_exists = os.path.isdir(real_vault)

    for task_name in os.listdir(fixtures_dir):
        task_dir = os.path.join(fixtures_dir, task_name)
        if not os.path.isdir(task_dir):
            continue

        task_json_path = os.path.join(task_dir, "task.json")
        if not os.path.exists(task_json_path):
            continue

        with open(task_json_path, "r") as f:
            task_config = json.load(f)

        # Check if this fixture has carrier keys
        has_carrier_keys = any(
            key in task_config
            for key in ["skill_src", "carrier_r_src", "carrier_f_src"]
        )

        if has_carrier_keys:
            # Carrier fixtures must have non-empty removal_basenames
            removal_basenames = task_config.get("removal_basenames", [])
            assert isinstance(removal_basenames, list), (
                f"{task_name}/task.json: removal_basenames must be a list, "
                f"got {type(removal_basenames).__name__}"
            )
            assert len(removal_basenames) > 0, (
                f"{task_name}/task.json has carrier keys but removal_basenames is empty"
            )

            # If the real vault exists, every basename must exist as a note
            if vault_exists:
                for basename in removal_basenames:
                    note_path = os.path.join(real_vault, f"{basename}.md")
                    assert os.path.exists(note_path), (
                        f"{task_name}/task.json removal_basenames includes '{basename}' "
                        f"but {note_path} does not exist in the vault"
                    )
