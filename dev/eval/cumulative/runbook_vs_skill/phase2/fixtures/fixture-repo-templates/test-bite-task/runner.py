"""A small subprocess runner: run_child spawns a child python process and is supposed to
forward env_extra into that child's environment."""
import json
import os
import subprocess
import sys


def run_child(env_extra=None):
    """Spawn a child python process and return its os.environ as a dict, forwarding env_extra
    into the child's environment on top of the current process environment."""
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    script = "import json, os, sys; sys.stdout.write(json.dumps(dict(os.environ)))"
    result = subprocess.run(
        [sys.executable, "-c", script],
        env=env,
        capture_output=True,
        text=True,
        check=True,
    )
    return json.loads(result.stdout)
