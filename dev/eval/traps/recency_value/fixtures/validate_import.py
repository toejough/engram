#!/usr/bin/env python3
"""Validates orders.db.json against orders.csv (same directory). Run after implementing
`import`:

    ./validate_import.py orders.db.json

Exits non-zero with a symptom message on the first mismatch — it does not state the
conversion formula, only what's wrong with the value found.
"""
import csv
import json
import os
import sys


def _load_csv_dollars(csv_path):
    dollars = {}
    with open(csv_path, newline="") as f:
        for row in csv.DictReader(f):
            dollars[row["id"]] = float(row["dollars"])
    return dollars


def validate(db_path, csv_path):
    """Return None if orders.db.json is valid, else a symptom-description string."""
    with open(db_path) as f:
        records = json.load(f)
    dollars = _load_csv_dollars(csv_path)
    if len(records) != len(dollars):
        return f"row count mismatch: db has {len(records)} record(s), orders.csv has {len(dollars)}"
    for i, rec in enumerate(records, start=1):
        rid = str(rec.get("id"))
        if rid not in dollars:
            return f"row {i}: unknown id {rid!r} — not present in orders.csv"
        amt = rec.get("amt")
        if not isinstance(amt, int) or isinstance(amt, bool):
            return f"row {i}: amt out of range — expected an integer, got {amt!r}"
        if amt != round(dollars[rid] * 1000):
            return f"row {i}: amt out of range — expected integer tenths-of-a-cent, got {dollars[rid]}"
    return None


def main():
    db_path = sys.argv[1] if len(sys.argv) > 1 else "orders.db.json"
    csv_path = os.path.join(os.path.dirname(os.path.abspath(db_path)), "orders.csv")
    if not os.path.exists(csv_path):
        csv_path = "orders.csv"
    problem = validate(db_path, csv_path)
    if problem:
        print(f"FAIL: {problem}")
        sys.exit(1)
    with open(db_path) as f:
        n = len(json.load(f))
    print(f"OK: {n} row(s) validated")
    sys.exit(0)


if __name__ == "__main__":
    main()
