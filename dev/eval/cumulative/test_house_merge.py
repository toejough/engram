import json
import os
import shutil

import score

CUM = os.path.dirname(os.path.abspath(__file__))


def test_house_checks_merged(tmp_path):
    # Resolve the shared house block relative to the spec's own dir (production behavior).
    shutil.copy(os.path.join(CUM, "house_gotchas.json"), tmp_path / "house_gotchas.json")
    spec_path = tmp_path / "some_spec.json"
    spec_path.write_text(json.dumps({
        "app": "some",
        "interface": "x",
        "house_checks_file": "house_gotchas.json",
        "checks": [{"name": "native1", "bucket": "native", "symptom": "s",
                    "steps": [{"argv": ["add", "a"], "assert": "exit0"}]}],
    }))
    spec = score.load_spec(str(spec_path))
    assert len(spec["checks"]) == 9  # 1 native + 8 house


def test_missing_checks_key_defaults_empty(tmp_path):
    spec_path = tmp_path / "bare_spec.json"
    spec_path.write_text(json.dumps({"app": "bare", "interface": "x"}))
    spec = score.load_spec(str(spec_path))
    assert spec["checks"] == []
