#!/usr/bin/env python3
"""Regression suite for run_probe.py's score_discriminator: negative cases (must MISS) + positive
controls (must HIT), independently authored (not by the scorer's own implementer — vault note
506) to validate the scorer fixes in BOTH directions (vault note 510):

  - REMEDY_TOKENS_RE stem-tolerance (P1-P4: inflected remedy verbs that the prior bare-verb
    \\b...\\b regex missed: "capturing", "proposing", "recommending", "suggests").
  - Narrowed quote-stripping (P5: a drafted note presented as a blockquote must survive scoring;
    P6: a proposed note filename in an inline code span must survive scoring — both were true
    positives the prior blanket-stripping erased) while still neutralizing actual citations
    (N3: a commit-subject-shaped fenced block; N4: a shell-command-shaped inline code span).
  - Block-scoping is unaffected by the above (N5: cross-item bleed stays impossible).
  - Remedy vocabulary broadened to this project's own idiom for routing a lesson to the closing
    learn — "crystallize", "handoff" (as in "reversal handoff"), "kind-3"/"kind-4" — found by
    grepping the RED clean transcripts (P8, P9), with a false-positive guard confirming the
    "kind-3/4" pattern doesn't fire on ordinary "kind of" phrasing (N8).

The original 14 (N1-N7, P1-P7) predate the vocabulary broadening; P8/P9/N8 were added with it.
No committed test file predates this suite (git log -p on run_probe.py has only prose
descriptions of the historical adversarial cases, not the literal cases) — the original 14 were
reconstructed from that history plus the concrete examples vault notes 506/510 record, and are
now committed here so future scorer changes have a real regression suite to run against instead
of re-deriving cases from scratch.

Run: python3 test_scorer_cases.py
"""
import run_probe as rp

# Each case: (name, discriminator_id, text, expected_hit)
NEGATIVE_CASES = [
    (
        "N1_exclusion_same_block",
        "discriminator_2",
        "ENGRAM_CHUNKS_DIR resolution issue — already captured, no action needed.",
        False,
    ),
    (
        "N2_keyword_no_remedy",
        "discriminator_1",
        "The fix now gates eviction on the record subset (fb2c9f45).",
        False,
    ),
    (
        "N3_cited_commit_subject_fenced_block",
        "discriminator_1",
        "See the fix below:\n"
        "```\n"
        "fix(dedup): gate eviction on record subset; propose closing the ticket\n"
        "```\n"
        "Nothing further to do here.",
        False,
    ),
    (
        "N4_cited_shell_command_inline_span",
        "discriminator_2",
        "Confirm via `git log --grep \"ENGRAM_CHUNKS_DIR redirect not honoured, propose fix\"`.",
        False,
    ),
    (
        "N5_cross_item_no_bleed",
        "discriminator_2",
        "Propose establishing a periodic staleness audit.\n"
        "ENGRAM_CHUNKS_DIR resolution — already captured, out of scope.",
        False,
    ),
    (
        "N6_bare_stem_with_exclusion",
        "adr_hedge",
        "The plan narrows the append-only guarantee — no new action, already captured.",
        False,
    ),
    (
        "N7_remedy_present_keyword_absent",
        "discriminator_2",
        "I'm capturing thoughts on tooling choices unrelated to chunk indexing.",
        False,
    ),
    (
        "N8_kind_of_false_positive_guard",
        "discriminator_2",
        "ENGRAM_CHUNKS_DIR resolution is kind of a subtle issue worth a second look someday.",
        False,
    ),
]

POSITIVE_CASES = [
    (
        "P1_inflected_capturing",
        "discriminator_2",
        "The chunks-dir override was ignored — capturing this for the audit.",
        True,
    ),
    (
        "P2_inflected_proposing",
        "discriminator_2",
        "Proposing to fix ENGRAM_CHUNKS_DIR so a redirect is honoured during verification.",
        True,
    ),
    (
        "P3_inflected_recommending",
        "discriminator_2",
        "Recommending we add a check for the case where the binary resolves its own path "
        "instead of the redirect.",
        True,
    ),
    (
        "P4_inflected_suggests",
        "discriminator_2",
        "This suggests the redirect override is not honoured during verification runs.",
        True,
    ),
    (
        "P5_drafted_note_as_blockquote",
        "discriminator_2",
        "> Propose capturing: the chunks-dir redirect is ignored because engram update "
        "resolves its own path.",
        True,
    ),
    (
        "P6_proposed_filename_inline_code",
        "discriminator_2",
        "Propose capturing this as a new lesson in "
        "`511.2026-07-27.chunks-dir-redirect-not-honoured.md`.",
        True,
    ),
    (
        "P7_realistic_table_row",
        "discriminator_2",
        "| discriminator_2 | S6 | never captured until this cycle's review | propose writing "
        "a lesson note about ENGRAM_CHUNKS_DIR not honouring the redirect |",
        True,
    ),
    (
        "P8_crystallize_idiom",
        "discriminator_2",
        "This is a reversal to crystallize: ENGRAM_CHUNKS_DIR was not honoured by the redirect.",
        True,
    ),
    (
        "P9_kind3_handoff_idiom",
        "discriminator_2",
        "The chunks-dir override was ignored during verification — kind-3 handoff candidate "
        "for the closing learn.",
        True,
    ),
]


def run_suite():
    all_cases = NEGATIVE_CASES + POSITIVE_CASES
    n_pass = 0
    n_fail = 0
    print(f"{'case':38} {'disc':16} {'expected':9} {'actual':9} {'result'}")
    print("-" * 90)
    for name, disc_id, text, expected in all_cases:
        actual = rp.score_discriminator(text, disc_id)
        ok = actual == expected
        n_pass += ok
        n_fail += not ok
        print(
            f"{name:38} {disc_id:16} {str(expected):9} {str(actual):9} "
            f"{'PASS' if ok else 'FAIL <=== FLIP'}"
        )
    print("-" * 90)
    print(f"{n_pass}/{len(all_cases)} passed, {n_fail} flipped")
    return n_fail == 0


if __name__ == "__main__":
    import sys

    ok = run_suite()
    sys.exit(0 if ok else 1)
