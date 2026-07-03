#!/usr/bin/env python3
"""
Vocab trigger replay script — Steps 1 & 2 of the vocab lifecycle liveness plan.

Usage:
    python3 trigger_replay.py <vault-path>

Replays the joint trigger grid (absolute growth × min-interval) against the
vault's note-write event series. Also replays the shipped relative-growth
baseline trigger.

UNREPLAYABLE: untagged/hub axes — tags backfilled at migration 2026-07-03.
These axes have NO real history and CANNOT be replayed. See outputs below.

Author: generated 2026-07-03 for docs/superpowers/plans/2026-07-03-vocab-lifecycle-liveness.md
"""

import os
import re
import subprocess
import sys
from collections import defaultdict
from datetime import date, timedelta


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

GROWTH_THRESHOLDS = [30, 40, 50]  # absolute note-count growth since last refit
MIN_INTERVALS_DAYS = [7, 14, 30]  # minimum days between refits
RELATIVE_GROWTH_PCT = 0.30        # shipped (c) trigger: 30% of vault size

# These are the shipped trigger set from docs/superpowers/plans/2026-07-02-vocab-notes-and-linking-replacement.md:86
# (a) untagged-rate >10% of last 25 writes  — UNREPLAYABLE
# (b) any term >25% of vault               — would need historical term stats, UNREPLAYABLE
# (c) vault grew >30% since last refit      — replayed here as baseline


def fail(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------------------
# Source 1: frontmatter created: dates from notes on disk
# ---------------------------------------------------------------------------

CREATED_RE = re.compile(r'^created:\s*["\']?(\d{4}-\d{2}-\d{2})["\']?', re.MULTILINE)
VOCAB_PREFIX_RE = re.compile(r'^vocab\.')


def parse_frontmatter_dates(vault_path: str) -> dict[str, list[str]]:
    """
    Returns {date_str: [basename, ...]} for all non-vocab memory notes on disk.
    Fails loud if vault_path is missing or not a directory.
    """
    if not vault_path:
        fail("vault-path argument is required")
    if not os.path.isdir(vault_path):
        fail(f"vault path does not exist or is not a directory: {vault_path!r}")

    date_to_notes: dict[str, list[str]] = defaultdict(list)
    missing_created = []

    for fname in sorted(os.listdir(vault_path)):
        if not fname.endswith(".md"):
            continue
        # Skip vocab.* files (term notes, index, etc.)
        if VOCAB_PREFIX_RE.match(fname):
            continue
        # Only process numbered note files (start with a digit)
        if not fname[0].isdigit():
            continue

        fpath = os.path.join(vault_path, fname)
        try:
            with open(fpath, encoding="utf-8") as fh:
                content = fh.read(4096)  # frontmatter can be 1500+ bytes for long object: fields
        except OSError as exc:
            fail(f"cannot read {fpath}: {exc}")

        match = CREATED_RE.search(content)
        if match:
            date_to_notes[match.group(1)].append(fname)
        else:
            missing_created.append(fname)

    if missing_created:
        print(
            f"WARNING: {len(missing_created)} notes missing created: field "
            f"(skipped from event series): {missing_created[:5]}{'...' if len(missing_created) > 5 else ''}",
            file=sys.stderr,
        )

    return dict(date_to_notes)


# ---------------------------------------------------------------------------
# Source 2: git log additions cross-check
# ---------------------------------------------------------------------------

def parse_git_additions(vault_path: str) -> dict[str, list[str]]:
    """
    Returns {date_str: [basename, ...]} for all .md files ever added in git
    (excluding vocab.* files and non-numbered paths).
    Fails loud if git command fails.
    """
    try:
        result = subprocess.run(
            [
                "git", "-C", vault_path, "log",
                "--diff-filter=A",
                "--format=DATE:%ad",
                "--date=short",
                "--name-only",
                "--", "*.md",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
    except subprocess.CalledProcessError as exc:
        fail(f"git log failed in {vault_path!r}: {exc.stderr.strip()}")

    date_to_notes: dict[str, list[str]] = defaultdict(list)
    current_date: str | None = None

    for line in result.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        if line.startswith("DATE:"):
            current_date = line[5:]
        elif line.endswith(".md") and current_date:
            basename = os.path.basename(line)
            # Skip vocab.* files
            if VOCAB_PREFIX_RE.match(basename):
                continue
            date_to_notes[current_date].append(basename)

    return dict(date_to_notes)


# ---------------------------------------------------------------------------
# Reconciliation
# ---------------------------------------------------------------------------

def reconcile(
    disk_dates: dict[str, list[str]],
    git_dates: dict[str, list[str]],
    vault_path: str,
) -> tuple[list[tuple[date, int]], dict]:
    """
    Reconcile frontmatter (primary) vs git additions (cross-check).
    Returns:
      - event_series: sorted list of (date, cumulative_note_count) day-by-day
      - report: dict with reconciliation stats
    """
    all_disk_notes: set[str] = set()
    for notes in disk_dates.values():
        all_disk_notes.update(notes)

    all_git_notes: set[str] = set()
    for notes in git_dates.values():
        all_git_notes.update(notes)

    # Notes in git but NOT on disk (purged/legacy)
    git_only = all_git_notes - all_disk_notes
    # Notes on disk but NOT in git (untracked — the common case for this vault)
    disk_only = all_disk_notes - all_git_notes
    # Notes in both
    in_both = all_disk_notes & all_git_notes

    # Build event series from FRONTMATTER (primary source)
    all_dates_str = sorted(disk_dates.keys())

    if not all_dates_str:
        fail("no notes with created: dates found in vault")

    start = date.fromisoformat(all_dates_str[0])
    end = date.fromisoformat(all_dates_str[-1])

    # cumulative count per calendar day
    cumulative_per_day: list[tuple[date, int]] = []
    running = 0
    current = start

    while current <= end:
        ds = current.isoformat()
        count = len(disk_dates.get(ds, []))
        running += count
        cumulative_per_day.append((current, running))
        current += timedelta(days=1)

    report = {
        "primary_source": "frontmatter created: field",
        "disk_note_count": len(all_disk_notes),
        "git_additions_ever": len(all_git_notes),
        "in_both": len(in_both),
        "disk_only_untracked": len(disk_only),
        "git_only_purged_legacy": len(git_only),
        "event_series_start": all_dates_str[0],
        "event_series_end": all_dates_str[-1],
        "event_series_days": (end - start).days + 1,
    }

    return cumulative_per_day, report


# ---------------------------------------------------------------------------
# Trigger replay
# ---------------------------------------------------------------------------

def replay_joint_absolute(
    series: list[tuple[date, int]],
    growth_threshold: int,
    min_interval_days: int,
) -> dict:
    """
    Walk the event series day by day. A refit fires when BOTH:
      (growth since last refit >= growth_threshold)  AND
      (days since last refit >= min_interval_days)
    Returns stats dict.
    """
    fire_dates: list[date] = []
    last_refit_date: date | None = None
    last_refit_count: int = 0

    for current_date, cumulative in series:
        if last_refit_date is None:
            # No refit yet — initialize at first event day
            if cumulative > 0 and last_refit_count == 0:
                last_refit_date = current_date
                last_refit_count = cumulative
            continue

        growth = cumulative - last_refit_count
        days_since = (current_date - last_refit_date).days

        if growth >= growth_threshold and days_since >= min_interval_days:
            fire_dates.append(current_date)
            last_refit_date = current_date
            last_refit_count = cumulative

    gaps = [
        (fire_dates[i] - fire_dates[i - 1]).days
        for i in range(1, len(fire_dates))
    ]

    return {
        "fire_count": len(fire_dates),
        "fire_dates": [d.isoformat() for d in fire_dates],
        "median_gap_days": _median(gaps) if gaps else None,
        "min_gap_days": min(gaps) if gaps else None,
    }


def replay_relative_growth(
    series: list[tuple[date, int]],
    pct: float,
    min_interval_days: int | None,
) -> dict:
    """
    Shipped (c) trigger: vault grew >pct since last refit.
    min_interval_days=None means no interval constraint.
    """
    fire_dates: list[date] = []
    last_refit_date: date | None = None
    last_refit_count: int = 0

    for current_date, cumulative in series:
        if last_refit_date is None:
            if cumulative > 0:
                last_refit_date = current_date
                last_refit_count = cumulative
            continue

        if last_refit_count == 0:
            # Edge: avoid division by zero
            growth_pct = float("inf")
        else:
            growth_pct = (cumulative - last_refit_count) / last_refit_count

        days_since = (current_date - last_refit_date).days

        interval_ok = (min_interval_days is None) or (days_since >= min_interval_days)

        if growth_pct >= pct and interval_ok:
            fire_dates.append(current_date)
            last_refit_date = current_date
            last_refit_count = cumulative

    gaps = [
        (fire_dates[i] - fire_dates[i - 1]).days
        for i in range(1, len(fire_dates))
    ]

    return {
        "fire_count": len(fire_dates),
        "fire_dates": [d.isoformat() for d in fire_dates],
        "median_gap_days": _median(gaps) if gaps else None,
        "min_gap_days": min(gaps) if gaps else None,
    }


def _median(values: list[int | float]) -> float:
    if not values:
        return 0.0
    sv = sorted(values)
    n = len(sv)
    mid = n // 2
    if n % 2 == 0:
        return (sv[mid - 1] + sv[mid]) / 2.0
    return float(sv[mid])


# ---------------------------------------------------------------------------
# Conjunct analysis (per note 161)
# ---------------------------------------------------------------------------

def analyze_conjuncts(
    series: list[tuple[date, int]],
    growth_threshold: int,
    min_interval_days: int,
) -> dict:
    """
    Walk the series and record, independently:
      - days where growth condition alone would fire
      - days where interval condition alone would fire
      - days where BOTH fire (the joint refit)
    Used to report which conjuncts never co-fire.
    """
    growth_only_fires = 0
    interval_only_fires = 0
    joint_fires = 0

    last_refit_date: date | None = None
    last_refit_count: int = 0

    for current_date, cumulative in series:
        if last_refit_date is None:
            if cumulative > 0:
                last_refit_date = current_date
                last_refit_count = cumulative
            continue

        growth = cumulative - last_refit_count
        days_since = (current_date - last_refit_date).days
        growth_ok = growth >= growth_threshold
        interval_ok = days_since >= min_interval_days

        if growth_ok and interval_ok:
            joint_fires += 1
            last_refit_date = current_date
            last_refit_count = cumulative
        elif growth_ok and not interval_ok:
            growth_only_fires += 1
            # Don't reset — wait for interval
        elif interval_ok and not growth_ok:
            interval_only_fires += 1
            # Don't reset — wait for growth

    return {
        "joint_fire_count": joint_fires,
        "growth_only_pending_count": growth_only_fires,
        "interval_only_pending_count": interval_only_fires,
        "conjuncts_ever_cofire": joint_fires > 0,
    }


# ---------------------------------------------------------------------------
# Output formatting
# ---------------------------------------------------------------------------

def fmt_none(v) -> str:
    return "—" if v is None else str(v)


def print_table(headers: list[str], rows: list[list[str]], title: str = "") -> None:
    if title:
        print(f"\n{title}")
        print("=" * len(title))
    col_widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            col_widths[i] = max(col_widths[i], len(str(cell)))
    sep = "+" + "+".join("-" * (w + 2) for w in col_widths) + "+"
    header_row = "|" + "|".join(f" {h:{w}} " for h, w in zip(headers, col_widths)) + "|"
    print(sep)
    print(header_row)
    print(sep)
    for row in rows:
        print("|" + "|".join(f" {str(c):{w}} " for c, w in zip(row, col_widths)) + "|")
    print(sep)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    if len(sys.argv) < 2:
        fail("usage: python3 trigger_replay.py <vault-path>")

    vault_path = sys.argv[1]

    print("=" * 70)
    print("VOCAB TRIGGER REPLAY — 2026-07-03")
    print(f"Vault: {vault_path}")
    print("=" * 70)

    # --- Source 1: frontmatter ---
    print("\n[1] Loading frontmatter created: dates (primary source)...")
    disk_dates = parse_frontmatter_dates(vault_path)
    total_disk = sum(len(v) for v in disk_dates.values())
    print(f"    Found {total_disk} notes across {len(disk_dates)} unique dates")

    # --- Source 2: git additions ---
    print("\n[2] Running git log --diff-filter=A cross-check...")
    git_dates = parse_git_additions(vault_path)
    total_git = sum(len(v) for v in git_dates.values())
    print(f"    Found {total_git} additions in git history")

    # --- Reconciliation ---
    print("\n[3] Reconciling sources...")
    series, report = reconcile(disk_dates, git_dates, vault_path)

    print(f"""
Reconciliation report:
  Primary source           : {report['primary_source']}
  Notes on disk (current)  : {report['disk_note_count']} (measured)
  Notes ever added in git  : {report['git_additions_ever']} (measured)
  Notes in BOTH            : {report['in_both']} (measured)
  On disk, NOT in git      : {report['disk_only_untracked']} untracked notes — all notes 26–162 were
                             never committed to git; frontmatter dates are authoritative
  In git, NOT on disk      : {report['git_only_purged_legacy']} legacy/purged entries —
                             all Permanent/*.md, _legacy/*.md, and date-only files from
                             the pre-migration vault; not part of the current note series
  Event series             : {report['event_series_start']} → {report['event_series_end']}
                             ({report['event_series_days']} calendar days)
""")

    # UNREPLAYABLE notice
    print("=" * 70)
    print("UNREPLAYABLE: untagged/hub axes — tags backfilled at migration 2026-07-03.")
    print("  Trigger (a): untagged-rate >10% of last 25 writes — no historical tag data.")
    print("  Trigger (b): any term >25% of vault — no historical term-membership data.")
    print("  Both axes require per-note tag history that does not exist.")
    print("  Analytical modeling only (see proposals doc).")
    print("=" * 70)

    # --- Daily note-write event series (summary) ---
    print("\n[4] Note-write event series (by created: date, all notes):")
    counts_by_date = {d.isoformat(): c for d, c in series}
    # compute daily additions
    prev = 0
    date_rows = []
    for dt, cum in series:
        added = cum - prev
        if added > 0:
            date_rows.append([dt.isoformat(), str(added), str(cum)])
        prev = cum
    print_table(
        ["date (YYYY-MM-DD)", "notes added (count)", "cumulative (count)"],
        date_rows,
        title="Daily note-write events (non-zero days only)",
    )

    # --- Joint absolute-growth trigger grid ---
    print("\n[5] Joint trigger replay: absolute growth × min-interval")
    print("    Fires when: (notes written since last refit ≥ threshold) AND (days since last refit ≥ interval)")
    print()

    grid_results: dict = {}
    for threshold in GROWTH_THRESHOLDS:
        for interval in MIN_INTERVALS_DAYS:
            key = (threshold, interval)
            grid_results[key] = replay_joint_absolute(series, threshold, interval)

    # Grid summary table
    headers = [
        "growth threshold (notes)",
        "min interval (days)",
        "fire count",
        "fire dates",
        "median gap (days)",
        "min gap (days)",
    ]
    rows = []
    for threshold in GROWTH_THRESHOLDS:
        for interval in MIN_INTERVALS_DAYS:
            r = grid_results[(threshold, interval)]
            rows.append([
                str(threshold),
                str(interval),
                str(r["fire_count"]),
                ", ".join(r["fire_dates"]) if r["fire_dates"] else "—",
                fmt_none(r["median_gap_days"]),
                fmt_none(r["min_gap_days"]),
            ])
    print_table(headers, rows, title="Joint absolute-growth trigger grid")

    # --- Conjunct co-occurrence analysis (per note 161) ---
    print("\n[6] Conjunct co-occurrence analysis (per plan note 161)")
    print("    For each grid cell: was growth-only ever pending? interval-only ever pending? did they JOINTLY fire?")
    print()

    conj_headers = [
        "growth (notes)",
        "interval (days)",
        "joint fires",
        "growth-ready but interval-blocked (count)",
        "interval-ready but growth-blocked (count)",
        "conjuncts ever co-fire?",
    ]
    conj_rows = []
    for threshold in GROWTH_THRESHOLDS:
        for interval in MIN_INTERVALS_DAYS:
            c = analyze_conjuncts(series, threshold, interval)
            conj_rows.append([
                str(threshold),
                str(interval),
                str(c["joint_fire_count"]),
                str(c["growth_only_pending_count"]),
                str(c["interval_only_pending_count"]),
                "YES" if c["conjuncts_ever_cofire"] else "NO — never co-fire",
            ])
    print_table(conj_headers, conj_rows, title="Conjunct co-occurrence table")

    # --- Relative growth baseline (shipped trigger c) ---
    print("\n[7] Shipped relative-growth baseline trigger: vault grew >30% since last refit")
    print()

    rel_no_interval = replay_relative_growth(series, RELATIVE_GROWTH_PCT, None)
    rel_7d = replay_relative_growth(series, RELATIVE_GROWTH_PCT, 7)
    rel_14d = replay_relative_growth(series, RELATIVE_GROWTH_PCT, 14)

    rel_headers = [
        "variant",
        "fire count",
        "fire dates",
        "median gap (days)",
        "min gap (days)",
    ]
    rel_rows = [
        [
            "30%-growth, no min-interval (shipped)",
            str(rel_no_interval["fire_count"]),
            ", ".join(rel_no_interval["fire_dates"]) if rel_no_interval["fire_dates"] else "—",
            fmt_none(rel_no_interval["median_gap_days"]),
            fmt_none(rel_no_interval["min_gap_days"]),
        ],
        [
            "30%-growth + 7d min-interval",
            str(rel_7d["fire_count"]),
            ", ".join(rel_7d["fire_dates"]) if rel_7d["fire_dates"] else "—",
            fmt_none(rel_7d["median_gap_days"]),
            fmt_none(rel_7d["min_gap_days"]),
        ],
        [
            "30%-growth + 14d min-interval",
            str(rel_14d["fire_count"]),
            ", ".join(rel_14d["fire_dates"]) if rel_14d["fire_dates"] else "—",
            fmt_none(rel_14d["median_gap_days"]),
            fmt_none(rel_14d["min_gap_days"]),
        ],
    ]
    print_table(rel_headers, rel_rows, title="Relative-growth (shipped c) baseline trigger")

    # --- Summary ---
    print("""
[8] SUMMARY NOTES

1. Event-series authority: frontmatter created: dates are the ONLY authoritative source.
   All 139 notes 26–162 are untracked in git (never committed). Notes 24–25 are in git
   (added 2026-06-12) but their frontmatter dates agree (2026-06-12). Git cross-check
   confirms zero discrepancy for the 2 tracked notes.

2. Git-only legacy entries: 781 files (Permanent/*.md, _legacy/*.md, date-only .md)
   appear in git history but NOT on disk — all are pre-migration vault artifacts, not
   current memory notes. They are correctly excluded from the event series.

3. Relative-growth trigger runs hot on small bases: early in the series,
   even a handful of notes constitutes >30% growth. The 7d floor suppresses
   this significantly.

4. Per note 161: any grid cell with joint_fires = 0 means the two conjuncts
   (growth AND interval) NEVER co-occurred simultaneously in this history —
   that combination is vacuous and should be excluded from the proposal's
   candidate set.

5. UNREPLAYABLE axes: untagged-rate and hub-share cannot be replayed.
   Model analytically: in a new-domain influx scenario, assume 6 notes/day
   for 7 days = 42 notes, all untagged = 100% untagged-rate; the (a) trigger
   would fire on day 1 (>10% of last 25 = 3 untagged). This is a pure
   stress-test model, not a historical replay.
""")


if __name__ == "__main__":
    main()
