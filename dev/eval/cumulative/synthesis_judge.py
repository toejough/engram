#!/usr/bin/env python3
"""Synthesis judge: evaluate emergent-synthesis (C6 axis) probe results."""
import json, os

STUB_MODE = True


def judge_crystallization(vault_path, task_txt, expected_z_txt, stub=True):
    """Evaluate whether a new note was written capturing emergent synthesis Z.

    In stub mode: deterministic per vault path pattern.
    - vault_with_cluster → SIGNAL (cluster of related facts, Z absent → should crystallize)
    - vault_covered → MUST_NOT_FIRE (Z already explicitly stated — honesty guard)
    - otherwise → NULL
    """
    if stub:
        if "vault_with_cluster" in vault_path:
            return {
                "crystallized": True,
                "z_correct": True,
                "verdict": "SIGNAL",
                "note_path": None,
                "stub_mode": True,
            }
        elif "vault_covered" in vault_path:
            return {
                "crystallized": False,
                "z_correct": False,
                "verdict": "MUST_NOT_FIRE",
                "note_path": None,
                "stub_mode": True,
            }
        else:
            return {
                "crystallized": False,
                "z_correct": False,
                "verdict": "NULL",
                "note_path": None,
                "stub_mode": True,
            }
    # Real mode (not implemented here — requires LLM judge)
    raise NotImplementedError("Real judge mode requires LLM; use stub=True for dry runs")


def parse_synthesis_payload(payload_str):
    """Parse engram query output for clusters and candidate_l2s fields."""
    try:
        data = json.loads(payload_str)
    except Exception:
        data = {}
    return {
        "clusters": data.get("clusters", []),
        "candidate_l2s": data.get("candidate_l2s", []),
    }
