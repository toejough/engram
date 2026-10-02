"""Unit tests for extract_queries.py's extract_phrases (pure string logic, no I/O).

#787/D16/7.15: extract_queries.py is an ongoing mining tool with no date cutoff -- a future
real `--phrase=<value>` invocation (the now-safe form #787's own fix teaches our audit
instrument, and Route A now teaches the shipped skills) would otherwise silently vanish from
its queries.json output if this regex only ever matched the space-separated form.
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import extract_queries as eq


def test_extract_phrases_space_separated_double_quoted_form():
    """Regression guard: the original, already-working form must keep matching."""
    cmd = 'engram query --lazy-chunks --phrase "first phrase" --phrase "second phrase"'
    assert eq.extract_phrases(cmd) == ["first phrase", "second phrase"]


def test_extract_phrases_space_separated_single_quoted_form():
    cmd = "engram query --lazy-chunks --phrase 'first phrase' --phrase 'second phrase'"
    assert eq.extract_phrases(cmd) == ["first phrase", "second phrase"]


def test_extract_phrases_equals_double_quoted_form_extracts_the_same_phrases():
    cmd = 'engram query --lazy-chunks --phrase="first phrase" --phrase="second phrase"'
    assert eq.extract_phrases(cmd) == ["first phrase", "second phrase"]


def test_extract_phrases_equals_single_quoted_form_extracts_the_same_phrases():
    cmd = "engram query --lazy-chunks --phrase='first phrase' --phrase='second phrase'"
    assert eq.extract_phrases(cmd) == ["first phrase", "second phrase"]


def test_extract_phrases_both_forms_mixed_in_one_command():
    """A command mixing both forms (e.g. a trial composed one `--phrase` the old way and one
    the new way) must still extract every phrase, in order."""
    cmd = 'engram query --lazy-chunks --phrase "first phrase" --phrase="second phrase"'
    assert eq.extract_phrases(cmd) == ["first phrase", "second phrase"]
