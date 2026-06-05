package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// MigrateEpisodesArgs holds parsed flags for `engram migrate-episodes`.
type MigrateEpisodesArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Apply     bool   `targ:"flag,name=apply,desc=write changes (default: dry-run report only)"`
}

// MigrateEpisodesDeps holds injected dependencies for RunMigrateEpisodes.
type MigrateEpisodesDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Write    func(path string, data []byte) error
	Embedder embed.Embedder
}

// RunMigrateEpisodes rewrites every `type: episode` note into the D6 3-section
// body (## Summary / fenced ## Transcript / ## Related) and re-embeds each
// rewritten note's sidecar. Without --apply it reports what would change
// without writing. The boundary_rationale frontmatter is preserved unchanged;
// a legacy episode's ## Summary is seeded from it. The ## Related section is
// recomputed (preceding-episode links across all episodes by their ranges)
// plus any authored relations migrated to full-basename form. Idempotent: a
// re-render that equals the existing bytes is skipped (no write, no re-embed),
// so running twice changes nothing the second time.
func RunMigrateEpisodes(
	ctx context.Context,
	args MigrateEpisodesArgs,
	deps MigrateEpisodesDeps,
	stdout io.Writer,
) error {
	notes, scanErr := deps.Scan(args.VaultPath)
	if scanErr != nil {
		return fmt.Errorf("migrate-episodes: scan: %w", scanErr)
	}

	basenames := make([]string, len(notes))
	for i, note := range notes {
		basenames[i] = note.Basename
	}

	idToBasename := indexBasenamesByID(basenames)

	episodes, gatherErr := gatherEpisodeRanges(args.VaultPath, notes, deps.Read)
	if gatherErr != nil {
		return gatherErr
	}

	notesChanged := 0

	for _, note := range notes {
		changed, migrateErr := migrateOneEpisode(ctx, args, deps, note, idToBasename, episodes, stdout)
		if migrateErr != nil {
			return migrateErr
		}

		if changed {
			notesChanged++
		}
	}

	mode := "dry-run (use --apply to write)"
	if args.Apply {
		mode = "applied"
	}

	_, _ = fmt.Fprintf(stdout, "%s: %d notes\n", mode, notesChanged)

	return nil
}

// excludeBasename returns the episode ranges with any entry matching basename
// removed (so an episode never links itself during migration).
func excludeBasename(episodes []EpisodeRange, basename string) []EpisodeRange {
	out := make([]EpisodeRange, 0, len(episodes))

	for _, episode := range episodes {
		if episode.Basename == basename {
			continue
		}

		out = append(out, episode)
	}

	return out
}

// gatherEpisodeRanges reads every note's frontmatter and returns the
// EpisodeRange for those that are episodes — the input to preceding-link
// recomputation during migration.
func gatherEpisodeRanges(
	vault string,
	notes []vaultgraph.Note,
	read func(path string) ([]byte, error),
) ([]EpisodeRange, error) {
	out := make([]EpisodeRange, 0, len(notes))

	for _, note := range notes {
		full := filepath.Join(vault, pathOf(note.Basename, note.IsMOC))

		raw, readErr := read(full)
		if readErr != nil {
			return nil, fmt.Errorf("migrate-episodes: read %s: %w", note.Basename, readErr)
		}

		episodeRange, isEpisode := episodeRangeFromNote(note.Basename, raw)
		if !isEpisode {
			continue
		}

		out = append(out, episodeRange)
	}

	return out, nil
}

// migrateAuthoredRelations rewrites a list of "target|rationale" relation
// entries to full-basename "target|rationale" form, resolving bare ids via
// idToBasename. Entries already in basename form (or with no matching note) are
// left unchanged.
func migrateAuthoredRelations(relations []string, idToBasename map[string]string) []string {
	if len(relations) == 0 {
		return nil
	}

	resolved := make([]string, 0, len(relations))

	for _, relation := range relations {
		target, rationale, hasRationale := strings.Cut(relation, "|")

		if basename, found := idToBasename[strings.TrimSpace(target)]; found {
			target = basename
		}

		if hasRationale {
			resolved = append(resolved, strings.TrimSpace(target)+"|"+rationale)
		} else {
			resolved = append(resolved, strings.TrimSpace(target))
		}
	}

	return resolved
}

// migrateEpisodeSidecar re-embeds a rewritten episode and writes its sidecar,
// mirroring the resituate re-embed path. Failures are surfaced (an explicit
// rewrite, not a best-effort learn-time embed).
func migrateEpisodeSidecar(ctx context.Context, deps MigrateEpisodesDeps, notePath, content string) error {
	if deps.Embedder == nil {
		return nil
	}

	vector, embErr := deps.Embedder.Embed(ctx, string(embed.Text([]byte(content))))
	if embErr != nil {
		return fmt.Errorf("migrate-episodes: embedding %s: %w", notePath, embErr)
	}

	sidecar := embed.Sidecar{
		EmbeddingModelID: deps.Embedder.ModelID(),
		Dims:             deps.Embedder.Dims(),
		Vector:           vector,
		ContentHash:      embed.ContentHash([]byte(content)),
	}

	writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("migrate-episodes: writing sidecar for %s: %w", notePath, writeErr)
	}

	return nil
}

// migrateOneEpisode re-renders a single note if it is an episode, writing and
// re-embedding only when the rendered bytes differ from the existing bytes.
// Returns whether the note changed. Non-episode notes are no-ops.
func migrateOneEpisode(
	ctx context.Context,
	args MigrateEpisodesArgs,
	deps MigrateEpisodesDeps,
	note vaultgraph.Note,
	idToBasename map[string]string,
	episodes []EpisodeRange,
	stdout io.Writer,
) (bool, error) {
	relPath := pathOf(note.Basename, note.IsMOC)
	full := filepath.Join(args.VaultPath, relPath)

	raw, readErr := deps.Read(full)
	if readErr != nil {
		return false, fmt.Errorf("migrate-episodes: read %s: %w", relPath, readErr)
	}

	newContent, isEpisode, renderErr := rerenderEpisodeForMigration(note.Basename, raw, idToBasename, episodes)
	if renderErr != nil {
		return false, fmt.Errorf("migrate-episodes: %s: %w", relPath, renderErr)
	}

	if !isEpisode || newContent == string(raw) {
		return false, nil
	}

	verb := "would-rewrite"

	if args.Apply {
		writeErr := deps.Write(full, []byte(newContent))
		if writeErr != nil {
			return false, fmt.Errorf("migrate-episodes: write %s: %w", relPath, writeErr)
		}

		embedErr := migrateEpisodeSidecar(ctx, deps, full, newContent)
		if embedErr != nil {
			return false, embedErr
		}

		verb = "rewrote"
	}

	_, _ = fmt.Fprintf(stdout, "%s %s\n", verb, relPath)

	return true, nil
}

// newOsMigrateEpisodesDeps wires RunMigrateEpisodes to the real filesystem and
// the bundled embedder.
func newOsMigrateEpisodesDeps() MigrateEpisodesDeps {
	const perm = 0o600

	return MigrateEpisodesDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
		Read: (&osVaultFS{}).ReadFile,
		Write: func(path string, data []byte) error {
			err := os.WriteFile(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: sharedEmbedder,
	}
}

// rerenderEpisodeForMigration parses a note's existing body (legacy verbatim or
// already-migrated 3-section), rebuilds it in the D6 format, and returns the
// full rendered note. ok=false for non-episode notes. The same parser handles
// both shapes so re-running converges to a fixed point.
func rerenderEpisodeForMigration(
	basename string,
	raw []byte,
	idToBasename map[string]string,
	episodes []EpisodeRange,
) (string, bool, error) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", false, nil
	}

	if peekNoteType(frontmatter) != typeEpisode {
		return "", false, nil
	}

	var doc episodeFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", false, fmt.Errorf("parsing episode frontmatter: %w", unmarshalErr)
	}

	when, createdErr := parseCreated(doc.Created)
	if createdErr != nil {
		return "", false, createdErr
	}

	body := embed.ExtractBody(raw)
	parsed := parseEpisodeBody(string(body))

	summary := parsed.summary
	if strings.TrimSpace(summary) == "" {
		summary = doc.BoundaryRationale
	}

	relations := migrateAuthoredRelations(parsed.relations, idToBasename)
	preceding := computePrecedingLinks(excludeBasename(episodes, basename), doc.Provenance.TranscriptRange.Start)

	fields := episodeFields{
		Situation:         doc.Situation,
		Summary:           summary,
		BoundaryRationale: doc.BoundaryRationale,
		TranscriptText:    parsed.transcript,
		Sessions:          doc.Provenance.Sessions,
		TranscriptFiles:   doc.Provenance.TranscriptFiles,
		TranscriptStart:   doc.Provenance.TranscriptRange.Start,
		TranscriptEnd:     doc.Provenance.TranscriptRange.End,
		Luhmann:           string(doc.Luhmann),
		Source:            doc.Source,
		Project:           doc.Project,
		Issue:             string(doc.Issue),
		Tier:              doc.Tier,
		Relations:         relations,
		Preceding:         preceding,
	}

	return renderEpisodeFrontmatter(fields, when) + renderEpisodeBody(fields), true, nil
}
