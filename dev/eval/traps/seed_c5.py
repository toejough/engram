"""Build the C5 recency seed: a chunk index whose NEWEST chunk is R-decision.md (the ZÖRBAX
comment-marker decision), padded with topically-distant distractors so R surfaces by RECENCY,
not cosine. Run once before c5.py.

  python3 seed_c5.py            # builds /tmp/c5/chunks  (override with TRAPS_ROOT)

R is ingested LAST so it is the newest chunk; distractors are unrelated domains (lunch, travel,
hiring, ...) so the build task ("Go timestamp function") cannot cosine-match R — it only reaches
R through the recent channel.
"""
import argparse, os, shutil, subprocess, sys, tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import crowd

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c5")
SEED_CHUNKS = os.path.join(ROOT, "chunks")

# Topically-distant filler so the recent channel is realistic and R wins by recency, not cosine.
DISTRACTORS = {
    "lunch-menu.md": "# Cafeteria menu, week 24\nMonday: lentil soup and focaccia. Tuesday: grilled "
                     "halloumi wraps. Wednesday: mushroom risotto. Thursday: roast squash salad. "
                     "Friday: margherita flatbread. Coffee bar restocked with oat milk.",
    "offsite-travel.md": "# Lisbon offsite logistics\nFlights land Sunday afternoon; shuttle to the "
                         "hotel in Alfama. Walking tour Monday evening, dinner reservations at 8pm. "
                         "Bring a light jacket — coastal wind picks up after sunset.",
    "hiring-update.md": "# Recruiting status\nTwo design candidates moved to final round. The backend "
                        "loop is paused until Q3 headcount confirms. Referral bonus window closes "
                        "end of month. Update the scorecards before Friday's debrief.",
    "garden-notes.md": "# Community garden\nTomatoes need staking; the basil bolted in the heat. "
                       "Compost bin three is ready to turn. Someone please fix the hose reel by the "
                       "north bed. Next workday is Saturday morning.",
    "book-club.md": "# Book club picks\nThis month: a slow-burn detective novel set in 1920s Cairo. "
                    "Next month we vote between a space-opera and a food memoir. Meeting moved to the "
                    "library annex because the cafe is too loud.",
    "budget-memo.md": "# Travel budget\nPer-diem rates unchanged for domestic trips. International "
                      "lodging cap raised slightly. Submit receipts within two weeks. The old expense "
                      "portal is read-only now; use the new one for anything after the first.",
}

# R: the recent decision. Phrased as a generic annotation-token policy — NO mention of Go,
# timestamps, or functions — so it is topically distant from the build task.
R_NAME = "R-decision.md"
R_BODY = ("# 2026-06-22 — inline annotation token decision\n"
          "The team agreed today that from now on every explanatory note we leave inline in our "
          "source must open with the marker token ZÖRBAX immediately after the comment slashes, so "
          "the documentation indexer can collect them. Existing notes can stay as they are; all new "
          "ones use the ZÖRBAX marker.")


def _ingest(path):
    r = subprocess.run(["engram", "ingest", "--markdown", path, "--chunks-dir", SEED_CHUNKS],
                       capture_output=True, text=True)
    if r.returncode != 0:
        sys.exit(f"ingest failed for {path}:\n{r.stdout}\n{r.stderr}")
    return r.stdout.strip()


def build_seed(crowd_n=0):
    """Build the C5 chunk index: distractors (older), optional crowd variant chunks, R (newest).

    Crowd chunks are ingested BEFORE R so R stays the newest chunk and still wins by recency —
    the crowd competes on cosine in the recent channel without displacing R's recency rank."""
    if os.path.isdir(SEED_CHUNKS):
        shutil.rmtree(SEED_CHUNKS)
    os.makedirs(SEED_CHUNKS, exist_ok=True)
    src = tempfile.mkdtemp(prefix="c5-seed-src-")
    # Distractors first (older), crowd next, R last (newest) — recency is by ingest time.
    for name, body in DISTRACTORS.items():
        p = os.path.join(src, name); open(p, "w").write(body); _ingest(p)
    if crowd_n > 0:
        variants = crowd.make_variants(
            crowd.load_real_notes(crowd.real_vault()), crowd_n, seed=7, recency_frac=0.3)
        crowd.seed_into_chunks(SEED_CHUNKS, variants)
    rp = os.path.join(src, R_NAME); open(rp, "w").write(R_BODY)
    print("R ingest:", _ingest(rp))
    print(f"seed built at {SEED_CHUNKS} ({len(DISTRACTORS)} distractors + {crowd_n} crowd + R newest)")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--crowd", type=int, default=0,
                    help="ingest N real-vault variant chunks before R (R stays newest)")
    a = ap.parse_args()
    build_seed(a.crowd)


if __name__ == "__main__":
    main()
