package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestInvariant_M7_MarkerMonotonic locks invariant M7: per source, the
// persisted marker never regresses across runs (marker_after >=
// marker_before) — so history is never silently re-emitted.
//
// We drive the real advance+persist primitive (advanceAndReportMarker, which
// writes through learnmarker to a real file) over a generated sequence of
// runs per source. Each run operates in the true regime: a run scans
// [from=prior_marker, now], so the last-included Mtime and `now` are both at
// or after the prior marker (this models seedFromMarkers, which seeds
// lastIncluded to the prior marker and only ever advances it forward). After
// each run we read the persisted marker back from disk and assert it did not
// move backwards. This locks the advance/persist primitive; the seeding path
// that supplies the `>= prior_marker` inputs is its complementary guarantee.
func TestInvariant_M7_MarkerMonotonic(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		sourceCount := rapid.IntRange(1, 3).Draw(rt, "sourceCount")
		runCount := rapid.IntRange(1, 8).Draw(rt, "runCount")

		tmp := t.TempDir()

		// One marker file per source; markers are independent per source.
		markerPaths := make([]string, sourceCount)
		// prevMarker tracks the last persisted marker per source for the
		// next run's `from` and the monotonicity assertion.
		prevMarker := make([]time.Time, sourceCount)

		base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

		for s := range sourceCount {
			markerPaths[s] = filepath.Join(tmp, fmt.Sprintf("marker-%d.txt", s))
			prevMarker[s] = base
		}

		for run := range runCount {
			for s := range sourceCount {
				from := prevMarker[s]

				// In the real regime, a run scans [from, now] with now >= from,
				// and the last included row's Mtime is within that window.
				nowOffset := rapid.IntRange(0, 72).Draw(rt, fmt.Sprintf("now-%d-%d", run, s))
				now := from.Add(time.Duration(nowOffset) * time.Hour)

				includedOffset := rapid.IntRange(0, nowOffset).Draw(rt,
					fmt.Sprintf("incl-%d-%d", run, s))
				lastIncluded := from.Add(time.Duration(includedOffset) * time.Hour)

				hadEntries := rapid.Bool().Draw(rt, fmt.Sprintf("had-%d-%d", run, s))
				pending := rapid.Bool().Draw(rt, fmt.Sprintf("pend-%d-%d", run, s))

				before := m7ReadMarker(t, gExpect, markerPaths[s], base)

				var stdout bytes.Buffer

				err := cli.AdvanceAndReportMarkerForTest(
					markerPaths[s], from, lastIncluded, hadEntries, pending, now, &stdout)
				gExpect.Expect(err).NotTo(HaveOccurred())

				if err != nil {
					return
				}

				after := m7ReadMarker(t, gExpect, markerPaths[s], base)

				gExpect.Expect(after.Before(before)).To(BeFalse(),
					"M7: source %d run %d regressed marker: before=%s after=%s",
					s, run, before.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano))

				prevMarker[s] = after
			}
		}
	})
}

// m7ReadMarker reads the persisted marker timestamp from disk, returning
// fallback when the marker file does not yet exist (first run).
func m7ReadMarker(t *testing.T, gExpect *WithT, path string, fallback time.Time) time.Time {
	t.Helper()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fallback
	}

	gExpect.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return fallback
	}

	parsed, parseErr := time.Parse(time.RFC3339Nano, string(data))
	gExpect.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return fallback
	}

	return parsed
}
