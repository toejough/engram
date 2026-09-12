import subprocess
import sys


def test_greet_default():
    """Existing test: verify default greeting works."""
    result = subprocess.run([sys.executable, "cli.py"], capture_output=True, text=True)
    assert result.returncode == 0
    assert "Hello, World!" in result.stdout
