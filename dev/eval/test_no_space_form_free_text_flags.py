"""$0 static regression guard, adopted in place of the `writing-skills` behavioral RED/GREEN
eval (Route A task 7.10/7.10a, design.md D20 Results): the behavioral trial could not be
reproduced — agent-composed free-text values never carried a `-`-leading shape (0/18 delivered
trials, matching 0/1426 in real usage), and shim's own scenario got refused as suspected prompt
injection before ever reaching an `engram` call. The real, measured exposure is a shipped
template teaching the space-separated `--flag "value"` form directly (shim's own verbatim
`--text`/`--phrase` pass-through, D14) — which IS mechanically, cheaply, and exactly checkable by
scanning the shipped files for that literal pattern, with no model, no spawn, no API cost.

Scans every shipped `agent-instructions/skills/*/SKILL.md` (plus any `references/` subdirectory)
and every `agent-instructions/guidance/*.md` for the old, vulnerable space-separated quoted form
of any free-text flag named in design.md D15's grep list (minus the identifier-like flags --slug/
--note/--target/--chunk-source/--position/--tag/--certainty, which never carry text a user would
type starting with "-" and are deliberately left unquoted/space-form per the plan's own
precedent).
"""
import glob
import os
import re
import subprocess

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# design.md D15's free-text flag grep list, minus the identifier-like flags (--slug, --note,
# --target, --chunk-source, --position, --tag, --certainty) that are deliberately left
# unquoted/space-form — they never carry a value a user would type starting with "-".
FREE_TEXT_FLAGS = (
    "text", "phrase", "situation", "subject", "predicate", "object", "behavior", "impact",
    "action", "source", "body", "done-when", "red-flag", "trigger", "question", "answer",
    "supersedes", "contributors",
)

# Matches `--flag "value"` / `--flag 'value'` (a space, then a quote) but NOT `--flag="value"`
# (no space before the quote) and NOT a longer flag name this one is a prefix of (e.g. --sourced).
_SPACE_FORM_RES = {
    flag: re.compile(rf"--{re.escape(flag)}(?![A-Za-z0-9-])\s+[\"']")
    for flag in FREE_TEXT_FLAGS
}


def find_space_form_hits(text):
    """Returns [(flag, matched_snippet), ...] for every space-separated quoted free-text flag
    found in `text`. Empty list means clean."""
    hits = []
    for flag, pattern in _SPACE_FORM_RES.items():
        for match in pattern.finditer(text):
            snippet = text[match.start():match.start() + 60].replace("\n", "\\n")
            hits.append((flag, snippet))
    return hits


def _skill_and_guidance_paths():
    paths = []
    for skill_md in sorted(glob.glob(os.path.join(REPO, "agent-instructions", "skills", "*", "SKILL.md"))):
        paths.append(skill_md)
        skill_dir = os.path.dirname(skill_md)
        refs_dir = os.path.join(skill_dir, "references")
        if os.path.isdir(refs_dir):
            for root, _dirs, files in os.walk(refs_dir):
                for fn in files:
                    if fn.endswith(".md"):
                        paths.append(os.path.join(root, fn))
    for guidance_md in sorted(glob.glob(os.path.join(REPO, "agent-instructions", "guidance", "*.md"))):
        paths.append(guidance_md)
    return paths


def test_shipped_skills_and_guidance_have_no_space_form_free_text_flags():
    """GREEN: the current, shipped files are clean."""
    offenders = {}
    for path in _skill_and_guidance_paths():
        with open(path) as f:
            hits = find_space_form_hits(f.read())
        if hits:
            offenders[os.path.relpath(path, REPO)] = hits
    assert not offenders, (
        f"space-separated quoted free-text flag(s) found in shipped files: {offenders} — "
        'use the --flag="value" form (D17); a leading "-" in the value would otherwise be '
        "misread as a new flag (#787's defect class)"
    )


def test_scanner_would_have_caught_the_pre_fix_shim_and_skill_text():
    """RED (self-test, proves the scanner is meaningful, not just vacuously passing): the
    scanner DOES find the old space-separated form in the pre-7.3-7.8 commit's text, via
    `git show ec83792d:<path>` — the commit immediately before this change's own `=`-form
    edits landed (same RED_REF task 7.10 used)."""
    red_ref = "ec83792de2b2b0253ed73b6115d79b7e9ff16cf1"
    targets = [
        "agent-instructions/guidance/shim.md",
        "agent-instructions/skills/recall/SKILL.md",
        "agent-instructions/skills/route/SKILL.md",
        "agent-instructions/skills/curate/SKILL.md",
        "agent-instructions/skills/write-memory/SKILL.md",
    ]
    total_hits = 0
    for rel_path in targets:
        out = subprocess.run(["git", "show", f"{red_ref}:{rel_path}"], cwd=REPO,
                              capture_output=True, text=True, check=True)
        hits = find_space_form_hits(out.stdout)
        assert hits, f"expected the scanner to find the pre-fix space form in {rel_path}, found none"
        total_hits += len(hits)
    assert total_hits > 0


def test_space_form_detector_does_not_false_positive_on_equals_form():
    text = 'engram learn fact --situation="- the agent assumed X" --source="session notes"'
    assert find_space_form_hits(text) == []


def test_space_form_detector_catches_single_and_double_quoted_space_form():
    text = 'engram query --phrase "a phrase" and --source \'a source\''
    hits = find_space_form_hits(text)
    flags_found = {flag for flag, _ in hits}
    assert flags_found == {"phrase", "source"}


def test_space_form_detector_ignores_identifier_like_flags_not_in_the_free_text_list():
    text = 'engram learn fact --slug "my-slug" --chunk-source "1.2026-01-01.note#top" --note "path.md"'
    assert find_space_form_hits(text) == []


def test_space_form_detector_does_not_false_positive_on_a_longer_flag_name_prefix():
    """--supersedes must not match inside a hypothetical --supersedes-strict or similar; here
    confirm --source doesn't match inside --sourced-from (a flag that isn't real, but exercises
    the negative lookahead)."""
    text = 'some text mentioning --sourced-from "x" which is not the --source flag at all'
    hits = find_space_form_hits(text)
    assert hits == []
