package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
)

// unexported constants.
const (
	minDuplicateGroupSize = 2
)

// unexported variables.
var (
	errPruneDuplicatesRemovalsFailed = errors.New("removals failed")
)

// duplicatePruneCounts tallies pruneDuplicatesLocked's outcome across every
// hash group: how many duplicates were removed, how many canonicals were
// retained, how many duplicates were refused (canonical twin unverified —
// split into the common structural zero-chunk case and the rarer anomalous
// case where a sibling's index proves the group has real content), and how
// many removal attempts failed.
type duplicatePruneCounts struct {
	removed, retained, failed           int
	refusedStructural, refusedAnomalous int
}

// anyIndexFileExists reports whether ANY path in group has an index file
// present right now. Used to classify a group whose canonical index is
// missing: if NO member (canonical or duplicate) was ever indexed, every
// member is byte-identical AND the same chunkingClass (groupManifestByHash's
// partition), so chunkSource would deterministically produce the SAME
// (empty) result for all of them — the common, structural, nothing-to-lose
// case (Gate B: ~88% of the live manifest). If some sibling DOES have an
// index file, the canonical's own missing file is the anomaly — real
// content survives elsewhere in the group, but not verifiably via the
// selected canonical, so it is worth surfacing individually rather than
// folding into the bulk summary.
func anyIndexFileExists(args PruneArgs, group []string, deps PruneDeps) bool {
	for _, path := range group {
		if deps.Exists(indexPathFor(args.ChunksDir, path)) {
			return true
		}
	}

	return false
}

// groupManifestByHash buckets live manifest entries by content hash AND
// chunkingClass (see ingest_dedup.go's chunkingClass — a .md and a .jsonl
// with identical bytes must never group as duplicates of each other),
// skipping any entry already tagged DuplicateOf by Unit 3's forward pass —
// those were never indexed, so retroactive cleanup has nothing to do for
// them; deleting their manifest entry would only force a needless re-read
// on the next ingest.
func groupManifestByHash(manifest ingestManifest) map[string][]string {
	groups := make(map[string][]string)

	for path, entry := range manifest {
		if entry.DuplicateOf != "" {
			continue
		}

		key := entry.FileHash + "|" + chunkingClass(path)
		groups[key] = append(groups[key], path)
	}

	for _, paths := range groups {
		sort.Strings(paths)
	}

	return groups
}

// pruneDuplicatesLocked implements `engram prune --duplicates`: retroactive
// cleanup for the backlog of exact-content-hash groups that accumulated
// more than one indexed member before Unit 3's ingest-time dedup existed.
// It re-derives the duplicate set from the LIVE manifest at run time —
// never from any external count — groups entries by (FileHash,
// chunkingClass) (skipping entries Unit 3's forward pass already tagged
// DuplicateOf: those were never indexed, so there is nothing to remove for
// them), and keeps exactly one member per group via Unit 3's
// selectCanonical. Every other member is evicted through
// removeDuplicateIndex, the same helper Unit 3 uses for its own
// out-of-order eviction, so deletion order (index file, then manifest
// entry) is identical here.
//
// Safety: immediately before evicting a duplicate, its retained twin's
// index file must exist RIGHT NOW; if it does not, the duplicate is
// refused rather than removed and left for a later run once the canonical
// is re-verified — this is the guarantee that no hash group can ever end
// up with zero searchable members. Refusals are reported as one bulk
// summary line for the common structural case (the whole group is a
// zero-chunk source; nothing was ever lost) and individually for the rarer
// anomalous case (a sibling's surviving index proves the group has real
// content) — see anyIndexFileExists. A removal error is reported via
// deps.LogWarning as "[FAIL] <path>: <err>", does not stop the run — every
// other removal still proceeds — but the run exits non-zero naming how
// many of how many removals failed. --dry-run performs every computation
// and reports the same counts without calling Remove or WriteFile.
func pruneDuplicatesLocked(args PruneArgs, deps PruneDeps, stdout io.Writer) error {
	manifest := ingestManifest{}

	data, err := deps.ReadFile(filepath.Join(args.ChunksDir, manifestName))
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "prune: no manifest, nothing to prune")

		return nil //nolint:nilerr // absence is the expected first-run case
	}

	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return fmt.Errorf("prune: reading manifest: %w", err)
	}

	counts := reconcileDuplicateGroups(args, manifest, deps, stdout)

	if !args.DryRun && counts.removed > 0 {
		writeErr := writePrunedManifest(args, deps, manifest)
		if writeErr != nil {
			return writeErr
		}
	}

	reportDuplicatePruneCounts(args, stdout, counts)

	if counts.failed > 0 {
		return fmt.Errorf("prune: %d of %d %w", counts.failed, counts.removed+counts.failed, errPruneDuplicatesRemovalsFailed)
	}

	return nil
}

// pruneOneDuplicate performs (or, in --dry-run, simulates) one duplicate's
// removal, given whether its canonical twin's index file was already
// verified present for the whole group. A removal error is reported via
// deps.LogWarning and returned.
func pruneOneDuplicate(
	args PruneArgs,
	path string,
	canonicalIndexExists bool,
	manifest ingestManifest,
	deps PruneDeps,
	removeDeps IngestDeps,
) (removedOne, refusedOne bool, err error) {
	if !canonicalIndexExists {
		return false, true, nil
	}

	if args.DryRun {
		return true, false, nil
	}

	rmErr := removeDuplicateIndex(args.ChunksDir, path, manifest, removeDeps)
	if rmErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("[FAIL] %s: %v", path, rmErr)
		}

		return false, false, rmErr
	}

	return true, false, nil
}

// pruneRemoveDeps adapts PruneDeps to the minimal IngestDeps shape
// removeDuplicateIndex needs (Stat, for its own index-file-exists check,
// and Remove) — consuming Unit 3's shared eviction helper rather than
// reimplementing it.
func pruneRemoveDeps(deps PruneDeps) IngestDeps {
	return IngestDeps{
		Remove: deps.Remove,
		Stat: func(path string) (SourceStat, error) {
			if deps.Exists(path) {
				return SourceStat{}, nil
			}

			return SourceStat{}, fs.ErrNotExist
		},
	}
}

// reconcileDuplicateGroups groups the live manifest by (hash, chunkingClass)
// and reconciles every group with more than one member, tallying the
// outcome.
func reconcileDuplicateGroups(
	args PruneArgs,
	manifest ingestManifest,
	deps PruneDeps,
	stdout io.Writer,
) duplicatePruneCounts {
	removeDeps := pruneRemoveDeps(deps)
	groups := groupManifestByHash(manifest)

	var counts duplicatePruneCounts

	for _, key := range sortedHashKeys(groups) {
		group := groups[key]
		if len(group) < minDuplicateGroupSize {
			continue
		}

		reconcileOneDuplicateGroup(args, group, manifest, deps, removeDeps, stdout, &counts)
	}

	return counts
}

// reconcileOneDuplicateGroup picks one hash group's canonical member via
// Unit 3's selectCanonical, then reconciles every other member against it,
// accumulating the outcome into counts. The canonical's index-file
// existence (the deletion-safety gate) and, when that gate fails, whether
// ANY group member has an index file (the structural/anomalous
// classification) are each computed once per group rather than once per
// duplicate — nothing changes either fact mid-loop, since only duplicate
// index files are ever removed here, never the canonical's.
func reconcileOneDuplicateGroup(
	args PruneArgs,
	group []string,
	manifest ingestManifest,
	deps PruneDeps,
	removeDeps IngestDeps,
	stdout io.Writer,
	counts *duplicatePruneCounts,
) {
	refs := make([]sourceRef, len(group))
	for i, path := range group {
		refs[i] = sourceRef{path: path}
	}

	canonical := selectCanonical(refs).path
	counts.retained++

	canonicalIndexExists := deps.Exists(indexPathFor(args.ChunksDir, canonical))

	var groupHasAnyIndex bool
	if !canonicalIndexExists {
		groupHasAnyIndex = anyIndexFileExists(args, group, deps)
	}

	prefix := dryRunPrefix(args.DryRun)

	for _, path := range group {
		if path == canonical {
			continue
		}

		removedOne, refusedOne, failErr := pruneOneDuplicate(args, path, canonicalIndexExists, manifest, deps, removeDeps)

		switch {
		case failErr != nil:
			counts.failed++
		case refusedOne && groupHasAnyIndex:
			counts.refusedAnomalous++

			_, _ = fmt.Fprintf(stdout, "%sprune: refusing to remove %s: canonical %s index missing "+
				"(needs review — a sibling's content survives)\n", prefix, path, canonical)
		case refusedOne:
			counts.refusedStructural++
		case removedOne:
			counts.removed++
		}
	}
}

// reportDuplicatePruneCounts prints pruneDuplicatesLocked's final tallies.
// Structural refusals (the common, zero-chunk-source case) are collapsed
// into one summary line rather than one line per item — Gate B measured a
// real run printing 47,396 near-identical "canonical index missing" lines,
// burying the handful of genuine removals and anomalies in noise.
// Anomalous refusals were already printed individually by
// reconcileOneDuplicateGroup; this only adds their count to the summary.
func reportDuplicatePruneCounts(args PruneArgs, stdout io.Writer, counts duplicatePruneCounts) {
	prefix := dryRunPrefix(args.DryRun)

	_, _ = fmt.Fprintf(stdout, "%sprune: removed %d duplicate(s), retained %d canonical(s)\n",
		prefix, counts.removed, counts.retained)

	if counts.refusedStructural > 0 {
		_, _ = fmt.Fprintf(stdout,
			"%sprune: refused %d removal(s): canonical has no index file (zero-chunk sources; nothing to lose)\n",
			prefix, counts.refusedStructural)
	}

	if counts.refusedAnomalous > 0 {
		_, _ = fmt.Fprintf(stdout,
			"%sprune: refused %d removal(s) listed above: canonical missing but sibling content still exists\n",
			prefix, counts.refusedAnomalous)
	}
}

// sortedHashKeys returns groups' keys in sorted order so --duplicates'
// per-group stdout lines (and thus its tests) are deterministic despite Go's
// randomized map iteration.
func sortedHashKeys(groups map[string][]string) []string {
	keys := make([]string, 0, len(groups))
	for hash := range groups {
		keys = append(keys, hash)
	}

	sort.Strings(keys)

	return keys
}
