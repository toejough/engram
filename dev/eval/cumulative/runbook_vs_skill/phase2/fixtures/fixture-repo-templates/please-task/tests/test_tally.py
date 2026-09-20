import subprocess
import sys


def run(*args):
    return subprocess.run([sys.executable, "tally.py", *args], capture_output=True, text=True)


def test_prints_report_to_stdout():
    result = run("sample.txt")
    assert result.returncode == 0
    assert result.stdout.splitlines()[0] == "the\t3"


def test_out_flag_writes_report_to_file(tmp_path):
    target = tmp_path / "report.txt"
    result = run("sample.txt", "--out", str(target))
    assert result.returncode == 0
    assert result.stdout == ""
    assert target.read_text().splitlines()[0] == "the\t3"
