package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// SkillAcceptDeps holds the dependencies shared by the skill-registration
// accept actions that mutate an EXISTING runbook note in place — RefreshSkill
// and RemoveSkill (skill-runbook-registration: "Accepting a refresh ... SHALL
// replace the body ..."; "Accepting a removal ... SHALL remove the note").
// Pared down from AmendDeps: no chunk-source validation, vocab assignment, or
// identity re-stamping — a skill-note sync preserves "all other frontmatter"
// untouched apart from the fields each action documents.
type SkillAcceptDeps struct {
	// Lock acquires the vault lock for the read-modify-write / delete
	// (mirrors AmendDeps.Lock).
	Lock func(vault string) (func(), error)
	// Read reads a vault-joined note path.
	Read func(path string) ([]byte, error)
	// Write atomically (over)writes a vault-joined path — used for both the
	// note and its .vec.json sidecar (mirrors AmendDeps.Write).
	Write func(path string, data []byte) error
	// Remove deletes a file at path — used by RemoveSkill for the note and
	// its sidecar via discardNote (mirrors AmendDeps.Remove).
	Remove func(path string) error
	// Embedder rebuilds RefreshSkill's sidecar. Unused by RemoveSkill.
	Embedder embed.Embedder
}

// SkillAdoptDeps holds the dependencies AdoptSkillNote needs: locating the
// target note (Scan, mirroring AmendDeps/ResituateDeps' vault scan), renaming
// it and rewriting inbound wikilinks (Rename, the existing
// RenameAndRewriteReferences primitive), and rebuilding its sidecar after the
// body replace (Embedder).
type SkillAdoptDeps struct {
	Lock     func(vault string) (func(), error)
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Rename   RenameRewriteDeps
	Embedder embed.Embedder
}

// SkillNoteSource is what a skill runbook note mirrors: the source file a
// key's winning candidate was read from (design D3, D8). Register, Refresh
// and Adopt write the key-derived slug (SkillKeySlug), skill_key,
// skill_source and skill_hash from it, and the body's preamble names
// SkillSource.
type SkillNoteSource struct {
	// Key is the skill key; the note's slug derives from it.
	Key string
	// SkillSource is the `~`-relative resolved source path (absolute when it
	// lies under no home), recorded as the note's skill_source and named by
	// its preamble and register provenance (design D8).
	SkillSource string
	// Content is the source file's bytes: the note body and hash input.
	Content []byte
}

// NewSkillNoteSource builds the note source for a keyed candidate, with
// SkillSource home-relative (`~/…`) against the first of homes that
// contains it — the home as given and its resolved form, since SourcePath
// is fully resolved while the removal-eligibility check (design D5) expands
// `~` against the home as given and matches a root's path or its resolved
// path. A path under no home is kept absolute.
func NewSkillNoteSource(candidate SkillCandidate, homes ...string) SkillNoteSource {
	return SkillNoteSource{
		Key:         candidate.Key,
		SkillSource: homeRelativePath(candidate.SourcePath, homes),
		Content:     candidate.Content,
	}
}

// AdoptSkillNote implements `engram register-skills --adopt <key>=<note-ref>`
// (skill-runbook-registration: "An existing runbook note SHALL be adoptable
// as a skill's note"). It resolves noteRef the same way `engram amend
// --target` does, requires the target to be a runbook note, renames it to
// the slug derived from source's key (SkillKeySlug) via
// RenameAndRewriteReferences (same Luhmann id and date, inbound wikilinks
// rewritten, sidecar moved, old basename appended to aliases), replaces its body with the current source file
// and preamble, stamps skill_hash, skill_key and skill_source, preserves
// every frontmatter key it doesn't set (keys the typed runbook model does
// not define included; comments best-effort), and removes any pending
// marker — an adopted
// note is a direct field-for-field promotion, never awaiting curation
// (design D6: "the fields are kept and the note is not marked pending").
// Adopting a note already carrying the key's slug is a no-op rename
// (idempotent) that still refreshes body and fields.
//
// Decision: pending is unconditionally cleared (the key removed), not merely left
// alone — the spec's postcondition is "SHALL NOT be marked pending", and a
// stray pre-existing pending marker would otherwise keep an adopted note
// excluded from query results even after adoption supplies real fields.
func AdoptSkillNote(
	ctx context.Context,
	vault string,
	source SkillNoteSource,
	noteRef string,
	deps SkillAdoptDeps,
	stdout io.Writer,
) error {
	release, lockErr := acquireOptionalLock(deps.Lock, vault)
	if lockErr != nil {
		return fmt.Errorf("register-skills: adopt: acquiring vault lock: %w", lockErr)
	}

	defer release()

	oldBasename, raw, targetErr := resolveAdoptTarget(vault, noteRef, deps)
	if targetErr != nil {
		return targetErr
	}

	keyErr := checkAdoptTargetKey(oldBasename, raw, source.Key)
	if keyErr != nil {
		return keyErr
	}

	conflictErr := checkAdoptConflict(vault, source.Key, oldBasename, deps)
	if conflictErr != nil {
		return conflictErr
	}

	newBasename, basenameErr := skillNoteBasename(oldBasename, source.Key)
	if basenameErr != nil {
		return basenameErr
	}

	// Render before any rename or write, so a note the full frontmatter
	// decode rejects is refused untouched — never renamed and left unkeyed.
	// The render is repeated below on the post-rename content, which the
	// rename's wikilink rewrite may have changed.
	preRenderErr := checkAdoptRenders(raw, oldBasename, newBasename, source)
	if preRenderErr != nil {
		return preRenderErr
	}

	raw, rewritten, renameErr := ensureSkillNoteBasename(vault, oldBasename, newBasename, raw, deps.Rename)
	if renameErr != nil {
		return renameErr
	}

	updated, renderErr := applySkillNoteBody(raw, source, false)
	if renderErr != nil {
		return renderErr
	}

	full := filepath.Join(vault, newBasename+mdExt)

	writeErr := writeAdoptedNote(ctx, full, updated, rewritten, deps)
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintln(stdout, full)

	return nil
}

// RefreshSkill accepts a refresh offer: it replaces the note's body with the
// current source file (preamble included), sets skill_hash to the current
// hash and skill_key/skill_source to the current key and source — following
// a plugin version bump's new path — sets pending: true (so curation re-checks the fields
// against the new text), and preserves every frontmatter key it doesn't set
// (comments best-effort) — situation, done_when, red_flags, triggers,
// created, keys the typed runbook model does not define, and the basename
// all survive untouched (skill-runbook-registration: "Accepting a refresh SHALL
// replace the body, keep the fields, and mark the note pending").
func RefreshSkill(
	ctx context.Context,
	vault string,
	source SkillNoteSource,
	basename string,
	deps SkillAcceptDeps,
	stdout io.Writer,
) error {
	release, lockErr := acquireOptionalLock(deps.Lock, vault)
	if lockErr != nil {
		return fmt.Errorf("register-skills: refresh: acquiring vault lock: %w", lockErr)
	}

	defer release()

	full := filepath.Join(vault, pathOf(basename))

	raw, readErr := deps.Read(full)
	if readErr != nil {
		return fmt.Errorf("register-skills: refresh: read %s: %w", basename, readErr)
	}

	updated, renderErr := applySkillNoteBody(raw, source, true)
	if renderErr != nil {
		return renderErr
	}

	writeErr := deps.Write(full, []byte(updated))
	if writeErr != nil {
		return fmt.Errorf("register-skills: refresh: write %s: %w", basename, writeErr)
	}

	embedErr := writeAmendedSidecar(ctx, AmendDeps{Write: deps.Write, Embedder: deps.Embedder}, full, updated)
	if embedErr != nil {
		return embedErr
	}

	_, _ = fmt.Fprintln(stdout, full)

	return nil
}

// RegisterSkill accepts a registration offer for source: it creates the
// runbook note through the normal capture path (RunLearn), with a fresh
// top-level Luhmann id, the slug derived from source's key, the
// preamble+source-file body, skill_hash, skill_key, skill_source, and
// pending: true — no situation, done_when, triggers, or red_flags
// (skill-runbook-registration: "Accepting registration SHALL create a
// pending note without runbook fields"). This is the only caller that sets
// LearnArgs' unexported skipRunbookRequiredFields bypass; `engram learn
// runbook`'s own situation/done_when requiredness is unaffected.
func RegisterSkill(
	ctx context.Context, vault, vaultName string, source SkillNoteSource, deps LearnDeps, stdout io.Writer,
) error {
	args := LearnArgs{
		Type:                      typeRunbook,
		Slug:                      SkillKeySlug(source.Key),
		Vault:                     vault,
		VaultName:                 vaultName,
		Position:                  positionTop,
		Source:                    skillRegistrationSourcePrefix + source.SkillSource,
		Body:                      skillNoteBody(source),
		SkillHash:                 SkillContentHash(source.Content),
		SkillKey:                  source.Key,
		SkillSource:               source.SkillSource,
		Pending:                   true,
		skipRunbookRequiredFields: true,
	}

	return RunLearn(ctx, args, deps, stdout)
}

// RemoveSkill accepts a removal offer: it deletes the skill's runbook note
// and its .vec.json sidecar under the vault lock, reusing discardNote's
// delete-note-and-sidecar mechanics (skill-runbook-registration: "Accepting
// a removal offer SHALL remove the skill note and its sidecar").
func RemoveSkill(vault, basename string, deps SkillAcceptDeps, stdout io.Writer) error {
	release, lockErr := acquireOptionalLock(deps.Lock, vault)
	if lockErr != nil {
		return fmt.Errorf("register-skills: remove: acquiring vault lock: %w", lockErr)
	}

	defer release()

	full := filepath.Join(vault, pathOf(basename))

	return discardNote(AmendDeps{Remove: deps.Remove}, full, stdout)
}

// unexported constants.
const (
	// skillNotePreambleFormat is the one-line body preamble every skill
	// runbook note (register, refresh or adopt) carries, naming the skill
	// file it mirrors with the same wording for every source, and never
	// where to edit it (design D8; learn-runbook-capture: "the body SHALL
	// begin with the one-line preamble ``> Mirrors skill `<skill path>`.``").
	skillNotePreambleFormat = "> Mirrors skill `%s`.\n"
	// skillRegistrationSourcePrefix starts a freshly registered note's
	// `source:` provenance text, followed by its skill_source.
	skillRegistrationSourcePrefix = "skill registration: "
)

// unexported variables.
var (
	// errAdoptAmbiguousRef refuses an adopt ref (a bare Luhmann id) that
	// matches more than one note.
	errAdoptAmbiguousRef = errors.New("register-skills: adopt: note ref matches more than one note")
	errAdoptConflict     = errors.New("register-skills: adopt: skill already registered to a different note")
	errAdoptNoteNotFound = errors.New("register-skills: adopt: note not found")
	// errAdoptTargetKeyed refuses an adopt whose target note is already
	// keyed to a different skill (or is claimed by another --adopt entry in
	// the same run): adopt never silently re-keys a note.
	errAdoptTargetKeyed         = errors.New("register-skills: adopt: note is already keyed to a different skill")
	errAdoptTargetNotRunbook    = errors.New("register-skills: adopt: target is not a runbook note")
	errAdoptUnparseableBasename = errors.New("register-skills: adopt: note basename has no Luhmann id/date")
	errSkillNoteNoFrontmatter   = errors.New("register-skills: note has no parseable frontmatter")
)

// adoptRenderInput returns the content an adopt's rename leaves for the
// note before its body is rendered: raw converted from CRLF to LF (toLF, as
// the rename converts it — design D5) with its luhmann: field rewritten to
// newBasename's id and oldBasename appended to its aliases (as renameOneNote
// does) when the adopt renames the note, or raw itself when the basename is
// unchanged and no rename runs. Rendering this before the rename covers
// exactly what the post-rename render sees, short of the inbound-wikilink
// rewrite, so a note the rename would refuse is refused untouched.
func adoptRenderInput(raw []byte, oldBasename, newBasename string) ([]byte, error) {
	lf := toLF(raw)
	if oldBasename == newBasename {
		return lf, nil
	}

	aliased, aliasErr := stampRenamedNote(string(lf), oldBasename, newBasename)
	if aliasErr != nil {
		return nil, fmt.Errorf("register-skills: adopt: recording alias on %s: %w", oldBasename, aliasErr)
	}

	return []byte(aliased), nil
}

// applySkillNoteBody edits raw's runbook frontmatter as a YAML node: it
// sets skill_hash, skill_key and skill_source from source, sets pending:
// true (pending) or removes the pending key (not pending — omitempty
// parity), and replaces the body with source's current file (preamble
// included). It preserves every frontmatter key it doesn't set — situation,
// done_when, red_flags, triggers, created, identity, supersedes, and keys
// the typed runbook note model does not define — with its value;
// comments best-effort (yaml.v3 round-trip). A note whose edited key
// carries an anchor is refused (errFrontmatterAnchoredKey), and the result
// is decoded again before it is returned (errFrontmatterUndecodable).
// Shared by RefreshSkill (pending=true) and AdoptSkillNote (pending=false).
func applySkillNoteBody(raw []byte, source SkillNoteSource, pending bool) (string, error) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", errSkillNoteNoFrontmatter
	}

	mapping, parseErr := parseFrontmatterMapping(frontmatter)
	if parseErr != nil {
		return "", fmt.Errorf("register-skills: parsing runbook frontmatter: %w", parseErr)
	}

	// The typed doc is read only, for validation and the supersedes tail.
	var doc runbookFrontmatterDoc

	decodeErr := mapping.Decode(&doc)
	if decodeErr != nil {
		return "", fmt.Errorf("register-skills: parsing runbook frontmatter: %w", decodeErr)
	}

	anchorErr := refuseAnchoredKeys(mapping, "skill_hash", "skill_key", "skill_source", "pending")
	if anchorErr != nil {
		return "", fmt.Errorf("register-skills: %w", anchorErr)
	}

	setMappingValue(mapping, "skill_hash", encodeNode(SkillContentHash(source.Content)))
	setMappingValue(mapping, "skill_key", encodeNode(source.Key))
	setMappingValue(mapping, "skill_source", encodeNode(source.SkillSource))

	if pending {
		setMappingValue(mapping, "pending", encodeNode(true))
	} else {
		deleteMappingKeys(mapping, "pending")
	}

	body := renderRunbookBody(runbookFields{
		Body:       skillNoteBody(source),
		Supersedes: doc.Supersedes,
	})

	rendered := marshalFrontmatter(mapping) + body

	verifyErr := verifyFrontmatterDecodes(rendered)
	if verifyErr != nil {
		return "", fmt.Errorf("register-skills: %w", verifyErr)
	}

	return rendered, nil
}

// checkAdoptConflict errors when key is already registered to a different
// note than oldBasename (skill-runbook-registration: "Error if a different
// note for the key already exists").
func checkAdoptConflict(vault, key, oldBasename string, deps SkillAdoptDeps) error {
	names, listErr := deps.Rename.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("register-skills: adopt: listing %s: %w", vault, listErr)
	}

	existingBasename, _, found, findSkillErr := FindSkillNote(vault, key, names, deps.Rename.ReadFile)
	if findSkillErr != nil {
		return fmt.Errorf("register-skills: adopt: %w", findSkillErr)
	}

	if found && existingBasename != oldBasename {
		return fmt.Errorf("%w: %q already registered as %q", errAdoptConflict, key, existingBasename)
	}

	return nil
}

// checkAdoptRenders renders what an adopt would write for raw, before any
// rename or write (adoptRenderInput), so a note the rename or the full
// frontmatter decode would reject is refused untouched.
func checkAdoptRenders(raw []byte, oldBasename, newBasename string, source SkillNoteSource) error {
	renderInput, inputErr := adoptRenderInput(raw, oldBasename, newBasename)
	if inputErr != nil {
		return inputErr
	}

	_, renderErr := applySkillNoteBody(renderInput, source, false)

	return renderErr
}

// checkAdoptTargetKey refuses an adopt target (basename, raw content) whose
// skill_key is already set to a key other than key: adopt never silently
// re-keys a note. An unkeyed note, or one already keyed to key (the
// idempotent re-adopt), passes.
func checkAdoptTargetKey(basename string, raw []byte, key string) error {
	frontmatter, _ := splitFrontmatter(raw)

	var probe skillNoteFrontmatterProbe

	unmarshalErr := yaml.Unmarshal(frontmatter, &probe)
	if unmarshalErr != nil {
		return fmt.Errorf("register-skills: adopt: parsing %s frontmatter: %w", basename, unmarshalErr)
	}

	if probe.SkillKey != "" && probe.SkillKey != key {
		return fmt.Errorf("%w: %s is keyed %q, not %q", errAdoptTargetKeyed, basename, probe.SkillKey, key)
	}

	return nil
}

// ensureSkillNoteBasename renames oldBasename to newBasename — inbound
// wikilinks rewritten, sidecar moved, via the existing
// RenameAndRewriteReferences primitive — when they differ, and returns the
// note's current raw content either way: the untouched raw when no rename
// was needed, or a fresh read at the new path otherwise. Adopting a note
// already carrying the key's slug (oldBasename == newBasename) is therefore a
// no-op rename — the idempotent case. It also returns the paths of the
// notes whose references the rename rewrote (nil when no rename ran).
func ensureSkillNoteBasename(
	vault, oldBasename, newBasename string, raw []byte, deps RenameRewriteDeps,
) ([]byte, []string, error) {
	if newBasename == oldBasename {
		return raw, nil, nil
	}

	rewritten, renameErr := RenameAndRewriteReferences(deps, vault, map[string]string{oldBasename: newBasename})
	if renameErr != nil {
		return nil, nil, fmt.Errorf("register-skills: adopt: renaming %s: %w", oldBasename, renameErr)
	}

	fresh, readErr := deps.ReadFile(filepath.Join(vault, newBasename+mdExt))
	if readErr != nil {
		return nil, nil, fmt.Errorf("register-skills: adopt: read %s: %w", newBasename, readErr)
	}

	return fresh, rewritten, nil
}

// findAdoptNote resolves noteRef the way `engram amend --target` does (a
// Luhmann id or a basename, wikilink brackets allowed) but refuses a ref
// matching more than one note (errAdoptAmbiguousRef) rather than taking
// the first in listing order — an order an earlier adopt's rename can
// change.
func findAdoptNote(notes []vaultgraph.Note, noteRef string) (string, error) {
	target := normalizeNoteRef(noteRef)
	matches := make([]string, 0, 1)

	for _, note := range notes {
		if note.LuhmannID == target || note.Basename == target {
			matches = append(matches, note.Basename)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %q", errAdoptNoteNotFound, noteRef)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %q matches %s — name the note by its full basename",
			errAdoptAmbiguousRef, noteRef, strings.Join(matches, ", "))
	}
}

// homeRelativePath returns path as `~/<rel>` against the first non-empty
// home that strictly contains it, or path unchanged when none does.
func homeRelativePath(path string, homes []string) string {
	for _, home := range homes {
		if home == "" || path == home || !pathWithinRoot(path, home) {
			continue
		}

		rel, _ := filepath.Rel(home, path)

		return homeRelPrefix + rel
	}

	return path
}

// resolveAdoptTarget scans the vault, resolves noteRef the same way `engram
// amend --target` does, and returns its basename and raw content, converted
// from CRLF to LF — erroring
// when the ref doesn't resolve to any note, or resolves to a note that isn't
// type runbook (skill-runbook-registration: "Must be a runbook note; error
// otherwise").
func resolveAdoptTarget(vault, noteRef string, deps SkillAdoptDeps) (basename string, raw []byte, err error) {
	notes, scanErr := deps.Scan(vault)
	if scanErr != nil {
		return "", nil, fmt.Errorf("register-skills: adopt: scan: %w", scanErr)
	}

	basename, findErr := findAdoptNote(notes, noteRef)
	if findErr != nil {
		return "", nil, findErr
	}

	relPath := pathOf(basename)

	raw, readErr := deps.Rename.ReadFile(filepath.Join(vault, relPath))
	if readErr != nil {
		return "", nil, fmt.Errorf("register-skills: adopt: read %s: %w", basename, readErr)
	}

	// Adopt always rewrites the note, so a CRLF note is converted to LF here
	// and every later step sees LF text (design D5).
	raw = toLF(raw)

	frontmatter, hasFrontmatter := splitFrontmatter(raw)
	if !hasFrontmatter || peekNoteType(frontmatter) != typeRunbook {
		return "", nil, fmt.Errorf("%w: %q", errAdoptTargetNotRunbook, noteRef)
	}

	return basename, raw, nil
}

// skillNoteBasename computes the key-derived slug basename (SkillKeySlug)
// for oldBasename — same Luhmann id and date, only the slug segment changes
// (skill-runbook-registration D3/D6: adoption keeps the note's identity, it
// does not re-mint one).
func skillNoteBasename(oldBasename, key string) (string, error) {
	id, date, ok := idAndDateFromNoteFilename(oldBasename + mdExt)
	if !ok {
		return "", fmt.Errorf("%w: %q", errAdoptUnparseableBasename, oldBasename)
	}

	return id + "." + date + "." + SkillKeySlug(key), nil
}

// skillNoteBody renders a skill runbook note's body: the one-line preamble
// naming the mirrored file (SkillSource), a blank line, then the source
// file's current bytes verbatim.
func skillNoteBody(source SkillNoteSource) string {
	return fmt.Sprintf(skillNotePreambleFormat, source.SkillSource) + "\n" + string(source.Content)
}

// writeAdoptedNote writes the adopted note's updated content at full,
// rebuilds its sidecar, and rebuilds the sidecar of every referrer the adopt
// rename rewrote (rewritten) — otherwise those referrers' content_hash no
// longer matches their rewritten bodies and `engram embed status` reports
// them stale.
func writeAdoptedNote(ctx context.Context, full, updated string, rewritten []string, deps SkillAdoptDeps) error {
	writeErr := deps.Rename.WriteFile(full, []byte(updated))
	if writeErr != nil {
		return fmt.Errorf("register-skills: adopt: write %s: %w",
			strings.TrimSuffix(filepath.Base(full), mdExt), writeErr)
	}

	embedErr := writeAmendedSidecar(
		ctx, AmendDeps{Write: deps.Rename.WriteFile, Embedder: deps.Embedder}, full, updated,
	)
	if embedErr != nil {
		return embedErr
	}

	// The adopted note's own sidecar was just rebuilt from its final body.
	referrers := slices.DeleteFunc(rewritten, func(path string) bool { return path == full })

	referrerErr := RebuildNoteSidecars(ctx, deps.Rename, deps.Embedder, referrers)
	if referrerErr != nil {
		return fmt.Errorf("register-skills: adopt: rebuilding referrer sidecars: %w", referrerErr)
	}

	return nil
}
