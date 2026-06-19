package cli

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
)

// unexported constants.
const (
	closingFrontmatterFence = 2 // fencesSeen value when the closing --- has been reached
	defaultHalfLifeDays     = 60.0
	defaultRecencyFloor     = 3
	defaultTailWeight       = 0.2
	hoursPerDay             = 24
	noteDateFormat          = "2006-01-02"
	openFrontmatterFence    = 1 // fencesSeen value while inside the frontmatter block
	turnAnchorPrefix        = "turn-"
)

// recencyParams are the tunable knobs (defaults chosen by the eval in recency_eval_test.go).
type recencyParams struct {
	halfLifeDays float64 // age at which the decay factor is 0.5
	tailWeight   float64 // extra lift for the last turn of a session (turnFrac=1)
	floor        int     // how many of the absolute-newest chunks the band guarantees
}

// applyChunkRecency returns a copy of scored with each score multiplied by its
// recency factor. turnFrac = turnN / maxTurn(source); 0 when the source has no
// turn anchors. Chunks with zero IngestedAt (legacy, not yet backfilled) are
// treated as age 0 (maximally recent) so they are not penalised.
func applyChunkRecency(
	scored []scoredChunk,
	now time.Time,
	maxTurnBySrc map[string]int,
	p recencyParams,
) []scoredChunk {
	out := make([]scoredChunk, len(scored))

	for i, s := range scored {
		ageDays := 0.0

		if !s.record.IngestedAt.IsZero() && !now.IsZero() {
			age := now.Sub(s.record.IngestedAt).Hours() / hoursPerDay
			if age > 0 {
				ageDays = age
			}
		}

		turnFrac := 0.0

		if n, ok := parseTurnN(s.record.Anchor); ok {
			if maxN := maxTurnBySrc[s.record.Source]; maxN > 0 {
				turnFrac = float64(n) / float64(maxN)
			}
		}

		out[i] = scoredChunk{
			record: s.record,
			score:  s.score * float32(recencyMultiplier(ageDays, turnFrac, p)),
		}
	}

	return out
}

// chunkNotePath returns the note-path key for a chunk record in the form
// "source#anchor", matching the key used in resolvedItem.notePath for chunk
// items. Centralising this avoids inline string concatenation scattered across
// mergeChunkSpace and newestChunkItems.
func chunkNotePath(r chunk.Record) string {
	return r.Source + "#" + r.Anchor
}

// defaultRecencyParams returns the eval-tuned recency knobs.
// Chosen cell (recorded after running TestRecencyEvalDiscriminatingHalfLife):
// halfLife=60, floor=3.
// Rationale: operators work in monthly cycles, so a short half-life would
// suppress legitimate older content. At halfLife=60d the decay is intentionally
// gentle (2wk→0.85, 1mo→0.71, 2mo→0.50), providing a soft tilt that slightly
// favours recency without drowning relevance. The band (not the re-rank) carries
// the freshness guarantee: newestChunkItems selects the floor-newest chunks by
// age regardless of absolute date, and fillRecencyBand force-inserts them even
// when the newest available content is weeks old. floor=3 ensures at least 3
// of the absolute-newest chunks always survive the cap.
func defaultRecencyParams() recencyParams {
	return recencyParams{
		halfLifeDays: defaultHalfLifeDays,
		tailWeight:   defaultTailWeight,
		floor:        defaultRecencyFloor,
	}
}

// fillRecencyBand guarantees every item in mustInclude appears in the returned
// slice (length <= limit). mustInclude is the ordered set of the floor-newest
// chunks, built by newestChunkItems. Items already present in items count as
// satisfied; missing ones are prepended, displacing the lowest-ranked items NOT
// in mustInclude, capped at limit. No-op when all mustInclude are already
// present or mustInclude is empty. Membership is by notePath.
func fillRecencyBand(items, mustInclude []resolvedItem, limit int) []resolvedItem {
	mustKey := make(map[string]bool, len(mustInclude))
	for _, r := range mustInclude {
		mustKey[r.notePath] = true
	}

	present := make(map[string]bool, len(items))
	have := 0

	for _, it := range items {
		present[it.notePath] = true
		if mustKey[it.notePath] {
			have++
		}
	}

	deficit := len(mustInclude) - have
	if deficit <= 0 {
		return items
	}

	// Never inject more than the whole budget — guards len(mustInclude) > limit,
	// where the band would otherwise prepend more items than limit allows.
	if deficit > limit {
		deficit = limit
	}

	missing := make([]resolvedItem, 0, deficit)

	for _, r := range mustInclude {
		if len(missing) >= deficit {
			break
		}

		if !present[r.notePath] {
			missing = append(missing, r)
		}
	}

	if len(missing) == 0 {
		return items
	}

	return spliceRecent(items, missing, mustKey, limit)
}

// lessByIngestedAtDesc orders two scored chunks newest-IngestedAt first. Zero
// IngestedAt (legacy, unknown recency) sorts last. Equal IngestedAt ties break
// on descending turn-N (latest turn first).
func lessByIngestedAtDesc(a, b scoredChunk) bool {
	timeA := a.record.IngestedAt
	timeB := b.record.IngestedAt

	if timeA.IsZero() && timeB.IsZero() {
		return turnNOf(a.record.Anchor) > turnNOf(b.record.Anchor)
	}

	if timeA.IsZero() {
		return false
	}

	if timeB.IsZero() {
		return true
	}

	if !timeA.Equal(timeB) {
		return timeA.After(timeB) // newer IngestedAt first
	}

	return turnNOf(a.record.Anchor) > turnNOf(b.record.Anchor)
}

// maxTurnBySource returns the highest turn ordinal seen per source.
// Sources with no turn anchors are absent from the map.
func maxTurnBySource(records []chunk.Record) map[string]int {
	maxBySource := make(map[string]int, len(records))

	for _, r := range records {
		n, ok := parseTurnN(r.Anchor)
		if !ok {
			continue
		}

		if cur, seen := maxBySource[r.Source]; !seen || n > cur {
			maxBySource[r.Source] = n
		}
	}

	return maxBySource
}

// mostRecentlyUsedNoteItems returns the n note items (kind != chunkItemKind)
// with the smallest noteAgeDays (freshest LastUsed→created), newest first — the
// note side of the combined floor band. Operates on the merged resolvedItems,
// which carry lastUsed/created (Task 2.3). Returns nil when n<=0 or now is zero.
func mostRecentlyUsedNoteItems(items []resolvedItem, now time.Time, n int) []resolvedItem {
	if n <= 0 || now.IsZero() {
		return nil
	}

	notes := make([]resolvedItem, 0, len(items))

	for _, it := range items {
		if it.kind != chunkItemKind {
			notes = append(notes, it)
		}
	}

	sort.SliceStable(notes, func(i, j int) bool {
		return noteAgeDays(notes[i].lastUsed, notes[i].created, now) <
			noteAgeDays(notes[j].lastUsed, notes[j].created, now)
	})

	if n > len(notes) {
		n = len(notes)
	}

	return notes[:n]
}

// newestChunkItems returns the n chunk items with the largest IngestedAt
// (most recently ingested first). Chunks with zero IngestedAt (legacy, not
// yet backfilled) sort last — treated as maximally old since their recency is
// unknown. Tie-breaking on equal IngestedAt uses descending turn-N (latest
// turn first). Returns nil when n<=0.
func newestChunkItems(scored []scoredChunk, n int) []resolvedItem {
	if n <= 0 {
		return nil
	}

	candidates := make([]scoredChunk, 0, len(scored))
	candidates = append(candidates, scored...)

	sort.SliceStable(candidates, func(i, j int) bool {
		return lessByIngestedAtDesc(candidates[i], candidates[j])
	})

	if n > len(candidates) {
		n = len(candidates)
	}

	out := make([]resolvedItem, 0, n)

	for _, c := range candidates[:n] {
		out = append(out, resolvedItem{
			notePath:    chunkNotePath(c.record),
			content:     c.record.Text,
			score:       c.score,
			provenances: []string{provenanceDirect},
			kind:        chunkItemKind,
		})
	}

	return out
}

// noteAgeDays returns a note's age in days for recency decay, preferring
// LastUsed (when it last proved useful) over created. Empty/unparseable
// stamps return 0 — treat as fresh so a malformed date never penalises.
func noteAgeDays(lastUsed, created string, now time.Time) float64 {
	stamp := lastUsed
	if stamp == "" {
		stamp = created
	}

	parsed, err := time.Parse(noteDateFormat, stamp)
	if err != nil {
		return 0
	}

	age := now.Sub(parsed).Hours() / hoursPerDay
	if age < 0 {
		age = 0
	}

	return age
}

// parseCreatedFromNote extracts the `created:` frontmatter date (YYYY-MM-DD)
// from a note's raw bytes, or "" when absent. Only the frontmatter block
// (between the opening and closing `---` fences) is scanned; body lines that
// happen to contain `created:` are ignored.
func parseCreatedFromNote(note []byte) string {
	const fence = "---"

	fencesSeen := 0

	for line := range strings.SplitSeq(string(note), "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == fence {
			fencesSeen++

			// Stop after the closing fence — everything below is body text.
			if fencesSeen == closingFrontmatterFence {
				return ""
			}

			continue
		}

		// Only match inside the frontmatter block (between the two fences).
		if fencesSeen == openFrontmatterFence {
			if rest, ok := strings.CutPrefix(trimmed, "created:"); ok {
				return strings.TrimSpace(rest)
			}
		}
	}

	return ""
}

// parseTurnN extracts the turn ordinal from a "turn-N" anchor.
// Returns (0, false) for preamble/heading anchors that carry no ordinal.
func parseTurnN(anchor string) (int, bool) {
	rest, ok := strings.CutPrefix(anchor, turnAnchorPrefix)
	if !ok {
		return 0, false
	}

	n, err := strconv.Atoi(rest)
	if err != nil || n < 0 {
		return 0, false
	}

	return n, true
}

// recencyMultiplier returns exp2(-ageDays/halfLife) * (1 + tailWeight*turnFrac).
// ageDays>=0; turnFrac in [0,1]. At age 0, turnFrac 0 it is exactly 1.0.
func recencyMultiplier(ageDays, turnFrac float64, p recencyParams) float64 {
	decay := math.Exp2(-ageDays / p.halfLifeDays)

	return decay * (1 + p.tailWeight*turnFrac)
}

// sortScoredDesc sorts in place by descending score (stable).
func sortScoredDesc(scored []scoredChunk) {
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
}

// spliceRecent prepends the missing recent items, then refills from the original
// items dropping the lowest-ranked NON-recent ones first, capped at limit.
// Two-pass fill: recent items (by recentKey) are kept ahead of non-recent ones
// even if a non-recent item had a higher pre-band score. This is the intended
// guarantee — recency-membership, not raw score, determines priority within
// the limit.
func spliceRecent(items, missing []resolvedItem, recentKey map[string]bool, limit int) []resolvedItem {
	out := make([]resolvedItem, 0, limit)
	out = append(out, missing...)

	// keep recent items from the original first, then non-recent, in original order.
	for _, item := range items {
		if len(out) >= limit {
			break
		}

		if recentKey[item.notePath] {
			out = append(out, item)
		}
	}

	for _, item := range items {
		if len(out) >= limit {
			break
		}

		if !recentKey[item.notePath] {
			out = append(out, item)
		}
	}

	return out
}

// turnNOf returns the turn ordinal for an anchor, or 0 when the anchor carries
// no ordinal (preamble/heading). A bare-int helper for tie-break comparisons.
func turnNOf(anchor string) int {
	n, _ := parseTurnN(anchor)

	return n
}
