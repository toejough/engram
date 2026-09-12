#!/bin/bash
set -euo pipefail

REPO_DIR="$1"
cd "$REPO_DIR"

# Check 1: the test file exists under tests/
if [ ! -f "tests/test_run_child.py" ]; then
  echo "FAIL: tests/test_run_child.py does not exist"
  exit 1
fi

# Check 2: the test passes against the shipped (correct) runner.py
if ! python3 -m pytest -q tests/test_run_child.py; then
  echo "FAIL: tests/test_run_child.py does not pass against the shipped runner.py"
  exit 1
fi

# Check 3: the test bites -- it must FAIL when env_extra forwarding is mutated out of runner.py.
# Source: runbook 838 step 2-3: mutate the implementation, confirm the test now fails.
SCRATCH=$(mktemp -d)
trap 'rm -rf "$SCRATCH"' EXIT
cp -r "$REPO_DIR" "$SCRATCH/repo"
rm -rf "$SCRATCH/repo/.git"

if ! grep -q "env.update(env_extra)" "$SCRATCH/repo/runner.py"; then
  echo "FAIL: mutation target (env.update(env_extra)) not found in runner.py -- cannot verify the test bites"
  exit 1
fi
sed -i.bak 's/env\.update(env_extra)/pass  # forwarding disabled/' "$SCRATCH/repo/runner.py"
rm -f "$SCRATCH/repo/runner.py.bak"

if (cd "$SCRATCH/repo" && python3 -m pytest -q tests/test_run_child.py) >/dev/null 2>&1; then
  echo "FAIL: tests/test_run_child.py still passes when env_extra forwarding is mutated out -- it does not bite"
  exit 1
fi

echo "PASS: Test-bite task end-state verified"
exit 0
