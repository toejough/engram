#!/bin/bash
set -e

REPO_PATH="${1:-.}"
cd "$REPO_PATH"

# Check 1: Registry entry exists with correct format
# Accept ANY name column: pressure_v2:<digits.dots><TAB><non-empty-name><TAB><YYYY-*>
if ! awk -F'\t' '/^pressure_v2:[0-9]+(\.[0-9]+)*/ && NF >= 3 && $2 != "" && $3 ~ /^[0-9]{4}-/ { found=1 } END { exit !found }' lib/sensors/registry.txt; then
  echo "FAIL: pressure_v2 registry entry not found or malformed"
  exit 1
fi

# Check 2: Migration file exists (accept either _sensor_pressure.go or _sensor_pressure_v2.go)
MIGRATION=$(ls migrations/*_sensor_pressure*.go 2>/dev/null | head -1)
if [ -z "$MIGRATION" ]; then
  echo "FAIL: Migration file for pressure_v2 not found"
  exit 1
fi

# Check 3: Migration file has init function
if ! grep -q "func init()" "$MIGRATION" || ! grep -q "registerSensor" "$MIGRATION"; then
  echo "FAIL: Migration file missing registerSensor init function"
  exit 1
fi

# Check 4: Changelog entry exists
if ! grep -q "Added sensor pressure_v2" TELEMETRY_CHANGELOG.log; then
  echo "FAIL: Changelog entry missing"
  exit 1
fi

# Check 5: Changelog has timestamp + author format
if ! grep "Added sensor pressure_v2" TELEMETRY_CHANGELOG.log | grep -q "\[[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}.*\].*Added sensor"; then
  echo "FAIL: Changelog entry missing proper timestamp format"
  exit 1
fi

# Check 6: make validate passes
if ! make validate >/dev/null 2>&1; then
  echo "FAIL: make validate failed"
  exit 1
fi

# Check 7: Registry change is STAGED
if ! git diff --cached --name-only | grep -q "lib/sensors/registry.txt"; then
  echo "FAIL: Registry change is not staged"
  exit 1
fi

echo "PASS: All end-state checks successful"
exit 0
