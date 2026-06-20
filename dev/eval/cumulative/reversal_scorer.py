#!/usr/bin/env python3
"""Reversal scorer: detect standards-change-over-time adherence (C4 axis)."""
import re


def detect_standard(code_str, spec):
    """Detect which standard the code follows: 'X', 'X_prime', 'both', or 'neither'."""
    has_x = bool(re.search(spec["standard_x"]["pattern"], code_str))
    has_xp = bool(re.search(spec["standard_x_prime"]["pattern"], code_str))
    if has_x and has_xp:
        return "both"
    if has_x:
        return "X"
    if has_xp:
        return "X_prime"
    return "neither"


def score_supersession(code_str, spec):
    """Score whether X' was adopted and X dropped (correct supersession)."""
    result = detect_standard(code_str, spec)
    follows_x = result in ("X", "both")
    follows_xp = result in ("X_prime", "both")
    return {
        "follows_x": follows_x,
        "follows_x_prime": follows_xp,
        "supersession_correct": follows_xp and not follows_x,
    }
