package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	originClaudeAncestor sourceOrigin = "claude-ancestor" // ancestor .claude dir
	originExplicit       sourceOrigin = "explicit"        // --transcript/--markdown/--pi-sessions
	originManualSweep    sourceOrigin = "manual-sweep"    // --sweep / SweepSpec.ExtraRoots
	originPiAncestor     sourceOrigin = "pi-ancestor"     // ancestor .pi dir
	originRepo           sourceOrigin = "repo"            // repo-markdown root (repoRootFor(env))
	originSessionLog     sourceOrigin = "session-log"     // recorded session transcripts
)

// hashedSource is one gathered source after the cheap-skip-aware hashing
// pass (Unit 3 step 1): its current content hash, and the raw bytes IF this
// run actually read them. raw is nil when the hash was reused cheaply from
// the manifest's cached FileHash without a read — the load-bearing case
// that must stay cheap for the ~79% of sources that are unchanged duplicates
// on every run.
type hashedSource struct {
	ref  sourceRef
	raw  []byte
	hash string
}

// sourceOrigin tags where a sourceRef was discovered: an explicit flag, the
// repo-markdown root, an ancestor .claude/.pi dir, a recorded session log,
// or a manual --sweep/ExtraRoots root. selectCanonical ranks hash-identical
// candidates by this tag (Unit 3 step 2) — origin cannot be recovered from
// the path alone once every gatherSources stage's output is merged, so each
// stage sets it at the point it constructs the sourceRef/SweepRoot.
type sourceOrigin string

// buildHashedSources computes each gathered source's current content hash,
// reusing the manifest's cached FileHash when cheapSkipEligible (Unit 3 step
// 1's load-bearing seeding: gatherSources never reads file bytes, and this
// must not either, for the common unchanged-duplicate case — eagerly
// hashing every source here would defeat the design the pipeline sits in).
// A swept (non-explicit) source that fails to read is skipped and reported
// via stdout, not returned as an error; an explicit source that fails to
// read returns an error immediately, matching RunIngest's historical
// per-source tolerance.
func buildHashedSources(
	sources []sourceRef,
	chunksDir string,
	deps IngestDeps,
	manifest ingestManifest,
	stdout io.Writer,
) ([]hashedSource, error) {
	hashed := make([]hashedSource, 0, len(sources))

	for _, src := range sources {
		prior, known := manifest[src.path]

		if known && cheapSkipEligible(src.path, prior, chunksDir, deps) {
			hashed = append(hashed, hashedSource{ref: src, hash: prior.FileHash})

			continue
		}

		raw, err := deps.ReadFile(src.path)
		if err != nil {
			wrapped := fmt.Errorf("ingest: reading %s: %w", src.path, err)
			if !src.explicit && errorsIsReadFailure(wrapped) {
				_, _ = fmt.Fprintf(stdout, "skip %s: %v\n", src.path, wrapped)

				continue
			}

			return nil, wrapped
		}

		hashed = append(hashed, hashedSource{ref: src, hash: hashBytes(raw), raw: raw})
	}

	return hashed, nil
}

// canonicalCoversDuplicateRecords is the record-level subset gate a
// ship-readiness review found missing (2026-07-26): byte-hash identity of
// a source's CURRENT content is necessary but not sufficient proof that
// evicting its index loses nothing. mergeChunkRecords is append-only
// WITHIN a source, so a source's .jsonl index file accumulates every
// chunk record from every past ingest of that path — not just its current
// content. Two sources can therefore be byte-identical RIGHT NOW (same
// FileHash, same hash group) while their index files hold entirely
// disjoint historical chunk records: deleting one on the strength of the
// other's mere EXISTENCE can silently destroy real, unique content.
//
// Two conditions must both hold for it to be safe to remove
// duplicatePath's index in favor of canonicalPath:
//  1. canonicalPath must have an index file AT ALL. rebuildIndex writes
//     nothing for zero-chunk content (ingest.go's empty-write guard), so a
//     freshly-promoted canonical that chunks to nothing never gets a
//     file — and a group must never be left with zero searchable members
//     through this path.
//  2. Every one of duplicatePath's OWN index records (by content hash)
//     must already be present among canonicalPath's. If duplicatePath has
//     no index file at all, there is nothing to lose, so it vacuously
//     covers (handled by duplicateRecordsCoveredBy).
//
// Not the ingest hot-path's production call site (a Gate B finding,
// 2026-07-26, ruled out re-reading and re-decoding canonicalPath's index
// once per duplicate in a group): reconcileDuplicate instead takes an
// already-loaded canonicalRecords/canonicalIndexExists pair, cached once
// per group by reconcileGroupCanonicalFirst, and calls
// duplicateRecordsCoveredBy directly. This function remains as the
// single-call reference for the same two-condition semantics, pinned
// directly by TestCanonicalCoversDuplicateRecords.
func canonicalCoversDuplicateRecords(chunksDir, canonicalPath, duplicatePath string, deps IngestDeps) bool {
	if !indexFileExists(chunksDir, canonicalPath, deps) {
		return false
	}

	canonicalRecords := loadPriorRecords(indexPathFor(chunksDir, canonicalPath), deps)

	return duplicateRecordsCoveredBy(chunksDir, duplicatePath, canonicalRecords, deps)
}

// canonicalLess reports whether a outranks b under selectCanonical's
// precedence order (a "less" b means a wins).
func canonicalLess(a, b sourceRef) bool {
	rankA, rankB := originRank(a.origin), originRank(b.origin)
	if rankA != rankB {
		return rankA < rankB
	}

	// Rule 3's "closest ancestor first": only meaningful within the same
	// ancestor rank, where AncestorDepth is set; zero (and thus a no-op tie)
	// for every other origin.
	if a.ancestorDepth != b.ancestorDepth {
		return a.ancestorDepth < b.ancestorDepth
	}

	sepA := strings.Count(a.path, string(filepath.Separator))
	sepB := strings.Count(b.path, string(filepath.Separator))

	if sepA != sepB {
		return sepA < sepB
	}

	return a.path < b.path
}

// canonicalUpToDate reports whether member's manifest entry already
// correctly reflects it as the indexed canonical: known, not a former
// duplicate, same hash, and its index file still present.
func canonicalUpToDate(prior manifestEntry, known bool, member hashedSource, chunksDir string, deps IngestDeps) bool {
	return known && prior.DuplicateOf == "" &&
		prior.FileHash == member.hash && indexFileExists(chunksDir, member.ref.path, deps)
}

// cheapSkipEligible reports whether source's manifest entry can be trusted
// without a read: its stat matches deps.Stat's current signature AND —
// either it is a recorded duplicate (which never has an index file, so none
// is expected) or its index file still exists (indexFileExists — the
// crash-window self-heal check, meaningful only for canonical entries).
//
// Its nil-Stat default (false, fail CLOSED — forces a read) is deliberately
// the opposite of indexFileExists's (true, fail OPEN): without deps.Stat
// there is no way to tell whether source changed at all, so guessing "skip"
// risks silently missing a real content change — the one failure mode this
// function must never allow. indexFileExists's own nil-Stat branch is dead
// here (this function already returned before reaching it), so the two
// defaults never actually collide; see indexFileExists's doc comment for
// why IT fails open. Both branches are unreachable in production —
// newIngestDeps always wires Stat (verified live, Gate B) — so this is
// purely a Stat-less-test-fixture concern.
func cheapSkipEligible(source string, prior manifestEntry, chunksDir string, deps IngestDeps) bool {
	if deps.Stat == nil {
		return false
	}

	stat, err := deps.Stat(source)
	if err != nil || stat != prior.SourceStat {
		return false
	}

	if prior.DuplicateOf != "" {
		return true
	}

	return indexFileExists(chunksDir, source, deps)
}

// chunkingClass reports which of chunkSource's dispatch branches path's
// extension selects: ".jsonl" sources are stripped as transcripts
// (ReadTranscript + chunk.Transcript); everything else — only ".md" ever
// reaches this point, per gatherSources' extension filter — is chunked as
// markdown directly off the raw bytes. Two sources must never be treated
// as duplicates of each other across this boundary even when their raw
// bytes are byte-identical: chunkSource produces genuinely different,
// non-empty chunk records for each (Gate B: markdown yields the literal
// bytes, transcript yields "USER: ..."-shaped text derived from them) — a
// dedup pass keyed on hash alone would permanently delete one's distinct
// chunking. groupByHash and groupManifestByHash both partition by this
// alongside the hash for exactly that reason.
func chunkingClass(path string) string {
	if filepath.Ext(path) == jsonlExt {
		return "jsonl"
	}

	return "markdown"
}

// dropVaultCopiesOutsideVault applies Rule B: an .md file with a sibling
// .vec.json sidecar (embed/sidecar.go's SidecarPath; the vault is 1:1
// .md/.vec.json) is a vault note, and is dropped from ingestion entirely
// unless it sits under the resolved vault path — catching a drifted vault
// copy (edited since it was copied) that exact-content dedup cannot, since
// it hashes differently from the live note. An unresolved vault ("")
// disables the rule rather than treating every sidecar-paired .md as
// "outside" it. Explicit --transcript/--markdown sources are exempt, the
// same escape hatch every other automatic-exclusion rule in this pipeline
// gives deliberately-named files.
func dropVaultCopiesOutsideVault(sources []sourceRef, vault string, deps IngestDeps) []sourceRef {
	if vault == "" {
		return sources
	}

	kept := make([]sourceRef, 0, len(sources))

	for _, src := range sources {
		if isVaultCopyOutsideVault(src, vault, deps) {
			continue
		}

		kept = append(kept, src)
	}

	return kept
}

// duplicateRecordsCoveredBy reports whether every one of duplicatePath's
// own index records (if it has an index file at all — absence is
// vacuously covered, since there is then nothing to lose) has its content
// hash present in canonicalRecords, an already-loaded canonical record
// set (keyed by ContentHash, as loadPriorRecords returns). Split out from
// canonicalCoversDuplicateRecords so a caller reconciling many duplicates
// against the SAME canonical in one pass can load the canonical's records
// ONCE per hash group and reuse them across every duplicate in it, rather
// than re-reading and re-decoding the same canonical index file once per
// duplicate. Two such callers: prune_duplicates.go's
// reconcileOneDuplicateGroup (`prune --duplicates`) and ingest_dedup.go's
// reconcileDuplicate (the ingest hot path, cached by
// reconcileGroupCanonicalFirst — a Gate B finding, 2026-07-26).
func duplicateRecordsCoveredBy(
	chunksDir, duplicatePath string,
	canonicalRecords map[string]chunk.Record,
	deps IngestDeps,
) bool {
	dupData, err := deps.ReadFile(indexPathFor(chunksDir, duplicatePath))
	if err != nil {
		return true // nothing indexed for the duplicate: nothing to lose
	}

	dupRecords, decodeErr := chunk.DecodeRecords(dupData)
	if decodeErr != nil {
		return false // can't verify safety: refuse
	}

	for _, record := range dupRecords {
		if _, covered := canonicalRecords[record.ContentHash]; !covered {
			return false
		}
	}

	return true
}

// groupByHash buckets hashedSources by content hash AND chunkingClass —
// never across it (see chunkingClass). Map order does not matter: every
// group is resolved independently, and selectCanonical is itself
// order-independent within a group.
func groupByHash(hashed []hashedSource) map[string][]hashedSource {
	groups := make(map[string][]hashedSource, len(hashed))

	for _, hs := range hashed {
		key := hs.hash + "|" + chunkingClass(hs.ref.path)
		groups[key] = append(groups[key], hs)
	}

	return groups
}

// indexFileExists reports whether source's per-source index file is
// present. Feeds the cheap-skip crash-window self-heal fix (Gate A: the
// window between removing an index file and its manifest entry does not
// self-heal without this check) and the eviction decision.
//
// Its nil-Stat default (true, fail OPEN — assumes presence) is the opposite
// of cheapSkipEligible's (false, fail CLOSED), on purpose, not an
// inconsistency: this function is also reached directly from
// canonicalUpToDate, a path cheapSkipEligible's own nil-Stat guard does not
// gate. Failing closed there would force a full rebuild of every
// "touched but identical" source for any Stat-less caller — confirmed live:
// manifestBackfill then stamps a fresh epoch IngestedAt (time.Unix(0, 0),
// since statOrZero yields a zero SourceStat with no Stat wired) onto a
// legacy zero-IngestedAt record that never had one, changing the encoded
// bytes on every such run and breaking TestIngestIsIdempotentByHash
// deterministically. Failing open here costs nothing a real production
// caller ever pays — newIngestDeps always wires Stat (verified live, Gate
// B) — while failing closed would cost every Stat-less test fixture a
// spurious, byte-changing rebuild for zero benefit (no missed change is
// possible: cheapSkipEligible already gated on Stat before considering this
// check from its own call site).
func indexFileExists(chunksDir, source string, deps IngestDeps) bool {
	if deps.Stat == nil {
		return true
	}

	_, err := deps.Stat(indexPathFor(chunksDir, source))

	return err == nil
}

// indexPathFor returns source's per-source .jsonl index file path.
func indexPathFor(chunksDir, source string) string {
	return filepath.Join(chunksDir, sourceSlug(source)+jsonlExt)
}

// isVaultCopyOutsideVault reports whether src is a stray vault-note copy:
// a non-explicit .md file, outside the resolved vault path, with a sibling
// .vec.json sidecar present.
func isVaultCopyOutsideVault(src sourceRef, vault string, deps IngestDeps) bool {
	if src.explicit || filepath.Ext(src.path) != mdExt {
		return false
	}

	if isUnderPathPrefix(filepath.Clean(src.path), []string{filepath.Clean(vault)}) {
		return false // it IS the canonical vault copy
	}

	if deps.Stat == nil {
		return false
	}

	_, err := deps.Stat(embed.SidecarPath(src.path))

	return err == nil
}

// originRank maps a sourceOrigin to selectCanonical's four precedence
// buckets (rule 5's tie-break handles finer distinctions within a bucket).
func originRank(origin sourceOrigin) int {
	const (
		rankExplicit = 1
		rankRepo     = 2
		rankAncestor = 3
		rankOther    = 4
	)

	switch origin {
	case originExplicit:
		return rankExplicit
	case originRepo:
		return rankRepo
	case originClaudeAncestor, originPiAncestor:
		return rankAncestor
	case originSessionLog, originManualSweep:
		return rankOther
	default:
		return rankOther
	}
}

// rawForRebuild returns the raw bytes to (re)build member's index with,
// reusing hashedSource.raw when the hashing pass already read them.
// Otherwise it reads fresh — the rare promotion case where a former
// duplicate's bytes were never read this run, or the crash-window self-heal
// case. skip reports a tolerated vanished-swept-source read failure; the
// caller treats that as a no-op, not an error.
func rawForRebuild(member hashedSource, deps IngestDeps, stdout io.Writer) (raw []byte, skip bool, err error) {
	if member.raw != nil {
		return member.raw, false, nil
	}

	raw, err = deps.ReadFile(member.ref.path)
	if err != nil {
		wrapped := fmt.Errorf("ingest: reading %s: %w", member.ref.path, err)
		if !member.ref.explicit && errorsIsReadFailure(wrapped) {
			_, _ = fmt.Fprintf(stdout, "skip %s: %v\n", member.ref.path, wrapped)

			return nil, true, nil
		}

		return nil, false, wrapped
	}

	return raw, false, nil
}

// reconcileCanonical ensures the group's winning source is indexed. A
// member that is already correctly indexed (canonicalUpToDate) costs
// nothing beyond the stat already spent building the hash registry.
// Otherwise it (re)builds the index via rawForRebuild + rebuildIndex.
func reconcileCanonical(
	ctx context.Context,
	member hashedSource,
	chunksDir string,
	deps IngestDeps,
	manifest ingestManifest,
	stdout io.Writer,
) (bool, error) {
	prior, known := manifest[member.ref.path]

	if canonicalUpToDate(prior, known, member, chunksDir, deps) {
		if member.raw == nil {
			return false, nil // fully stable: nothing was read, nothing to do
		}
		// Touched but identical content: refresh the stat so future runs
		// cheap-skip again, without rebuilding the index or re-embedding.
		manifest[member.ref.path] = manifestEntry{SourceStat: statOrZero(deps, member.ref.path), FileHash: member.hash}

		return true, nil
	}

	raw, skip, err := rawForRebuild(member, deps, stdout)
	if err != nil {
		return false, err
	}

	if skip {
		return false, nil
	}

	chunks, sourceTS, err := chunkSource(member.ref.path, raw, deps)
	if err != nil {
		return false, err
	}

	rebuilt, reused, embedded, err := rebuildIndex(
		ctx, member.ref.path, chunks, chunksDir, deps,
		ingestTimeFor(sourceTS, deps), manifestBackfill(manifest),
	)
	if err != nil {
		return false, err
	}

	manifest[member.ref.path] = manifestEntry{SourceStat: statOrZero(deps, member.ref.path), FileHash: member.hash}

	_, _ = fmt.Fprintf(stdout, "ingest %s: %d chunks (%d reused, %d embedded)\n",
		member.ref.path, rebuilt, reused, embedded)

	return true, nil
}

// reconcileDuplicate ensures a non-canonical hash-group member is recorded
// as a duplicate, never indexed. If it was indexed as canonical by a prior
// run (a higher-precedence twin now outranks it — structural change 2), its
// stale index file and manifest entry are evicted first via
// removeDuplicateIndex, then replaced with a fresh duplicate-of stub so a
// later run does not re-read it (Unit 3 step 3).
//
// Eviction is gated on canonicalIndexExists && duplicateRecordsCoveredBy,
// the record-level subset check (2026-07-26 ship-readiness finding):
// byte-hash identity of the CURRENT content is not sufficient, since
// mergeChunkRecords is append-only WITHIN a source and this member's own
// index file may hold historical records the canonical's current index
// does not. When the gate fails, the eviction — and the whole
// reconciliation for this member — is refused: the manifest entry is left
// exactly as it was (still canonical, still pointing at its own surviving
// index file) so a later run reconsiders it once the canonical catches up,
// rather than being silently marked a duplicate while its index file is
// left orphaned.
//
// canonicalIndexExists/canonicalRecords are the group's canonical index
// state, loaded at most ONCE per hash group by reconcileGroupCanonicalFirst
// and threaded in here via reconcileMember (a Gate B finding, 2026-07-26:
// this used to call canonicalCoversDuplicateRecords per duplicate, each
// call re-reading and re-decoding the same canonical .jsonl — a cost that
// recurs on every future ingest for a group that stays refused, since a
// refusal leaves DuplicateOf==""). Mirrors prune_duplicates.go's
// reconcileOneDuplicateGroup, which caches the same way.
func reconcileDuplicate(
	member hashedSource,
	canonicalPath, chunksDir string,
	deps IngestDeps,
	manifest ingestManifest,
	stdout io.Writer,
	canonicalIndexExists bool,
	canonicalRecords map[string]chunk.Record,
) (bool, error) {
	prior, known := manifest[member.ref.path]
	statNow := statOrZero(deps, member.ref.path)

	alreadyMarked := known && prior.DuplicateOf == canonicalPath &&
		prior.FileHash == member.hash && prior.SourceStat == statNow
	if alreadyMarked {
		return false, nil
	}

	if known && prior.DuplicateOf == "" {
		covered := canonicalIndexExists && duplicateRecordsCoveredBy(chunksDir, member.ref.path, canonicalRecords, deps)
		if !covered {
			_, _ = fmt.Fprintf(stdout,
				"ingest: refusing to evict %s: canonical %s does not verifiably cover its records "+
					"(needs review — run `engram prune --duplicates` once resolved)\n",
				member.ref.path, canonicalPath)

			return false, nil
		}

		err := removeDuplicateIndex(chunksDir, member.ref.path, manifest, deps)
		if err != nil {
			return false, err
		}
	}

	manifest[member.ref.path] = manifestEntry{SourceStat: statNow, FileHash: member.hash, DuplicateOf: canonicalPath}

	return true, nil
}

// reconcileMember brings one hash-group candidate's manifest state (and, for
// the canonical member, its index file) in line with the group's canonical
// decision. canonicalIndexExists/canonicalRecords are only consumed by the
// duplicate branch (reconcileDuplicate's eviction gate) — reconcileCanonical
// ignores them — but are threaded through here uniformly so
// reconcileGroupCanonicalFirst has one call shape for every member.
func reconcileMember(
	ctx context.Context,
	member hashedSource,
	canonicalPath, chunksDir string,
	deps IngestDeps,
	manifest ingestManifest,
	stdout io.Writer,
	canonicalIndexExists bool,
	canonicalRecords map[string]chunk.Record,
) (bool, error) {
	if member.ref.path == canonicalPath {
		return reconcileCanonical(ctx, member, chunksDir, deps, manifest, stdout)
	}

	return reconcileDuplicate(
		member, canonicalPath, chunksDir, deps, manifest, stdout, canonicalIndexExists, canonicalRecords,
	)
}

// removeDuplicateIndex evicts one source's stale index file and manifest
// entry — index file first, then manifest entry (the reverse order would
// strand an unreferenced index file nothing cleans up). Shared verbatim by
// Unit 3's out-of-order-arrival eviction (structural change 2) and Unit 4's
// retroactive `prune --duplicates` cleanup, rather than each reimplementing
// it.
func removeDuplicateIndex(chunksDir, source string, manifest ingestManifest, deps IngestDeps) error {
	if deps.Remove != nil && indexFileExists(chunksDir, source, deps) {
		indexPath := indexPathFor(chunksDir, source)

		err := deps.Remove(indexPath)
		if err != nil {
			return fmt.Errorf("ingest: removing duplicate index %s: %w", indexPath, err)
		}
	}

	delete(manifest, source)

	return nil
}

// selectCanonical picks the highest-precedence sourceRef among a group of
// hash-identical candidates. Pure: same candidates in any order yield the
// same winner (TestSelectCanonicalIsOrderIndependent). Precedence, first
// match wins:
//  1. explicit sources (--transcript/--markdown/--pi-sessions)
//  2. the repo-markdown root
//  3. an ancestor .claude/.pi dir, closest ancestor first
//  4. anything else (session logs, manual --sweep, extra roots)
//  5. tie-break: fewest path separators, then byte-wise lexicographic path
//
// Panics on an empty slice — callers only ever invoke this on a non-empty
// hash group (a group exists because at least one source hashed into it).
func selectCanonical(candidates []sourceRef) sourceRef {
	best := candidates[0]

	for _, cand := range candidates[1:] {
		if canonicalLess(cand, best) {
			best = cand
		}
	}

	return best
}

// sourceRefsFromRoot lists one SweepRoot's .md/.jsonl files as sourceRefs,
// tagging each with the root's Origin/AncestorDepth — the single body
// shared by piSessionSources' per-flag roots and sweptSources' pre-resolved
// --auto/--sweep roots (previously duplicated ListSources->filter->append
// bodies; Gate B flagged it). skipChunksDir additionally drops any path
// under chunksDir — a swept root containing the index must not self-ingest
// it; explicit --pi-sessions dirs never do, so that caller passes false.
func sourceRefsFromRoot(
	root SweepRoot,
	deps IngestDeps,
	chunksDir string,
	skipChunksDir bool,
) ([]sourceRef, error) {
	found, err := deps.ListSources(root)
	if err != nil {
		return nil, fmt.Errorf("ingest: sweeping %s: %w", root.Path, err)
	}

	var chunksPrefix string
	if skipChunksDir {
		chunksPrefix = filepath.Clean(chunksDir) + string(filepath.Separator)
	}

	var sources []sourceRef

	for _, path := range found {
		if chunksPrefix != "" && strings.HasPrefix(filepath.Clean(path), chunksPrefix) {
			continue
		}

		ext := filepath.Ext(path)
		if ext != ".md" && ext != jsonlExt {
			continue
		}

		sources = append(sources, sourceRef{
			path:          path,
			explicit:      root.Origin == originExplicit,
			origin:        root.Origin,
			ancestorDepth: root.AncestorDepth,
		})
	}

	return sources, nil
}
