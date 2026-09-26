package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
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

// AdoptSkillNote implements `engram register-skills --adopt <name>=<note-ref>`
// (skill-runbook-registration: "An existing runbook note SHALL be adoptable
// as a skill's note"). It resolves noteRef the same way `engram amend
// --target` does, requires the target to be a runbook note, renames it to
// slug "skill-<name>" via RenameAndRewriteReferences (same Luhmann id and
// date, inbound wikilinks rewritten, sidecar moved), replaces its body with
// the current SKILL.md and preamble, stamps skill_hash, preserves every
// other frontmatter field, and clears any pending marker — an adopted note
// is a direct field-for-field promotion, never awaiting curation (design
// D6: "the fields are kept and the note is not marked pending"). Adopting a
// note already named skill-<name> is a no-op rename (idempotent) that still
// refreshes body/hash.
//
// Decision: pending is unconditionally cleared (set false), not merely left
// alone — the spec's postcondition is "SHALL NOT be marked pending", and a
// stray pre-existing pending marker would otherwise keep an adopted note
// excluded from query results even after adoption supplies real fields.
func AdoptSkillNote(
	ctx context.Context,
	vault string,
	skill ShippedSkill,
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

	conflictErr := checkAdoptConflict(vault, skill.Name, oldBasename, deps)
	if conflictErr != nil {
		return conflictErr
	}

	newBasename, basenameErr := skillNoteBasename(oldBasename, skill.Name)
	if basenameErr != nil {
		return basenameErr
	}

	raw, renameErr := ensureSkillNoteBasename(vault, oldBasename, newBasename, raw, deps.Rename)
	if renameErr != nil {
		return renameErr
	}

	updated, renderErr := applySkillNoteBody(raw, skill, false)
	if renderErr != nil {
		return renderErr
	}

	full := filepath.Join(vault, newBasename+mdExt)

	writeErr := deps.Rename.WriteFile(full, []byte(updated))
	if writeErr != nil {
		return fmt.Errorf("register-skills: adopt: write %s: %w", newBasename, writeErr)
	}

	embedErr := writeAmendedSidecar(
		ctx, AmendDeps{Write: deps.Rename.WriteFile, Embedder: deps.Embedder}, full, updated,
	)
	if embedErr != nil {
		return embedErr
	}

	_, _ = fmt.Fprintln(stdout, full)

	return nil
}

// RefreshSkill accepts a refresh offer: it replaces the note's body with the
// current SKILL.md (preamble included), sets skill_hash to the current hash,
// sets pending: true (so curation re-checks the fields against the new
// text), and preserves every other frontmatter field — situation, done_when,
// red_flags, triggers, created, and the basename all survive untouched
// (skill-runbook-registration: "Accepting a refresh SHALL replace the body,
// keep the fields, and mark the note pending").
func RefreshSkill(
	ctx context.Context,
	vault string,
	skill ShippedSkill,
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

	updated, renderErr := applySkillNoteBody(raw, skill, true)
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

// RegisterSkill accepts a registration offer for skill: it creates the
// skill's runbook note through the normal capture path (RunLearn), with a
// fresh top-level Luhmann id, slug "skill-<name>", the preamble+SKILL.md
// body, skill_hash, and pending: true — no situation, done_when, triggers, or
// red_flags (skill-runbook-registration: "Accepting registration SHALL
// create a pending note without runbook fields"). This is the only caller
// that sets LearnArgs' unexported skipRunbookRequiredFields bypass;
// `engram learn runbook`'s own situation/done_when requiredness is
// unaffected.
func RegisterSkill(
	ctx context.Context, vault, vaultName string, skill ShippedSkill, deps LearnDeps, stdout io.Writer,
) error {
	args := LearnArgs{
		Type:                      typeRunbook,
		Slug:                      skillSlugPrefix + skill.Name,
		Vault:                     vault,
		VaultName:                 vaultName,
		Position:                  positionTop,
		Source:                    skillRegistrationSource(skill.Name),
		Body:                      skillNoteBody(skill.Name, skill.Content),
		SkillHash:                 SkillContentHash(skill.Content),
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
	// runbook note (register or refresh) carries, naming the skill file it
	// mirrors (skill-runbook-registration, learn-runbook-capture: "the body
	// SHALL begin with a one-line preamble stating it mirrors <skill path>
	// and that procedure edits belong in the skill file").
	skillNotePreambleFormat = "> Mirrors skill `agent-instructions/skills/%s/SKILL.md` — edit the procedure " +
		"there; the runbook fields on this note are authored here.\n"
)

// unexported variables.
var (
	errAdoptConflict            = errors.New("register-skills: adopt: skill already registered to a different note")
	errAdoptNoteNotFound        = errors.New("register-skills: adopt: note not found")
	errAdoptTargetNotRunbook    = errors.New("register-skills: adopt: target is not a runbook note")
	errAdoptUnparseableBasename = errors.New("register-skills: adopt: note basename has no Luhmann id/date")
	errSkillNoteNoFrontmatter   = errors.New("register-skills: note has no parseable frontmatter")
)

// applySkillNoteBody parses raw as a runbook note's frontmatter, replaces its
// body with skill's current SKILL.md (preamble included) and its skill_hash
// with skill's current content hash, sets pending, and leaves every other
// frontmatter field (situation, done_when, red_flags, triggers, created,
// source, repo, user, vault, issue, sources, tags, supersedes) exactly as
// parsed — shared by RefreshSkill (pending=true) and AdoptSkillNote
// (pending=false).
func applySkillNoteBody(raw []byte, skill ShippedSkill, pending bool) (string, error) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", errSkillNoteNoFrontmatter
	}

	var doc runbookFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", fmt.Errorf("register-skills: parsing runbook frontmatter: %w", unmarshalErr)
	}

	doc.SkillHash = SkillContentHash(skill.Content)
	doc.Pending = pending

	body := renderRunbookBody(runbookFields{
		Body:       skillNoteBody(skill.Name, skill.Content),
		Supersedes: doc.Supersedes,
	})

	return marshalFrontmatter(doc) + body, nil
}

// checkAdoptConflict errors when skillName is already registered to a
// different note than oldBasename (skill-runbook-registration: "Error if a
// different skill-<name> note already exists").
func checkAdoptConflict(vault, skillName, oldBasename string, deps SkillAdoptDeps) error {
	names, listErr := deps.Rename.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("register-skills: adopt: listing %s: %w", vault, listErr)
	}

	existingBasename, _, found, findSkillErr := FindSkillNote(vault, skillName, names, deps.Rename.ReadFile)
	if findSkillErr != nil {
		return fmt.Errorf("register-skills: adopt: %w", findSkillErr)
	}

	if found && existingBasename != oldBasename {
		return fmt.Errorf("%w: %q already registered as %q", errAdoptConflict, skillName, existingBasename)
	}

	return nil
}

// ensureSkillNoteBasename renames oldBasename to newBasename — inbound
// wikilinks rewritten, sidecar moved, via the existing
// RenameAndRewriteReferences primitive — when they differ, and returns the
// note's current raw content either way: the untouched raw when no rename
// was needed, or a fresh read at the new path otherwise. Adopting a note
// already named skill-<name> (oldBasename == newBasename) is therefore a
// no-op rename — the idempotent case.
func ensureSkillNoteBasename(
	vault, oldBasename, newBasename string, raw []byte, deps RenameRewriteDeps,
) ([]byte, error) {
	if newBasename == oldBasename {
		return raw, nil
	}

	renameErr := RenameAndRewriteReferences(deps, vault, map[string]string{oldBasename: newBasename})
	if renameErr != nil {
		return nil, fmt.Errorf("register-skills: adopt: renaming %s: %w", oldBasename, renameErr)
	}

	fresh, readErr := deps.ReadFile(filepath.Join(vault, newBasename+mdExt))
	if readErr != nil {
		return nil, fmt.Errorf("register-skills: adopt: read %s: %w", newBasename, readErr)
	}

	return fresh, nil
}

// resolveAdoptTarget scans the vault, resolves noteRef the same way `engram
// amend --target` does, and returns its basename and raw content — erroring
// when the ref doesn't resolve to any note, or resolves to a note that isn't
// type runbook (skill-runbook-registration: "Must be a runbook note; error
// otherwise").
func resolveAdoptTarget(vault, noteRef string, deps SkillAdoptDeps) (basename string, raw []byte, err error) {
	notes, scanErr := deps.Scan(vault)
	if scanErr != nil {
		return "", nil, fmt.Errorf("register-skills: adopt: scan: %w", scanErr)
	}

	relPath, findErr := findNote(notes, noteRef)
	if findErr != nil {
		return "", nil, fmt.Errorf("%w: %q", errAdoptNoteNotFound, noteRef)
	}

	basename = strings.TrimSuffix(relPath, mdExt)

	raw, readErr := deps.Rename.ReadFile(filepath.Join(vault, relPath))
	if readErr != nil {
		return "", nil, fmt.Errorf("register-skills: adopt: read %s: %w", basename, readErr)
	}

	frontmatter, hasFrontmatter := splitFrontmatter(raw)
	if !hasFrontmatter || peekNoteType(frontmatter) != typeRunbook {
		return "", nil, fmt.Errorf("%w: %q", errAdoptTargetNotRunbook, noteRef)
	}

	return basename, raw, nil
}

// skillNoteBasename computes the "skill-<name>" slug basename for oldBasename
// — same Luhmann id and date, only the slug segment changes (skill-runbook-
// registration D3/D6: adoption keeps the note's identity, it does not
// re-mint one).
func skillNoteBasename(oldBasename, skillName string) (string, error) {
	id, date, ok := idAndDateFromNoteFilename(oldBasename + mdExt)
	if !ok {
		return "", fmt.Errorf("%w: %q", errAdoptUnparseableBasename, oldBasename)
	}

	return id + "." + date + "." + skillSlugPrefix + skillName, nil
}

// skillNoteBody renders a skill runbook note's body: the one-line preamble
// naming the skill file, a blank line, then the skill's current SKILL.md
// bytes verbatim.
func skillNoteBody(skillName string, skillContent []byte) string {
	return fmt.Sprintf(skillNotePreambleFormat, skillName) + "\n" + string(skillContent)
}

// skillRegistrationSource returns the `source:` provenance text for a freshly
// registered skill note.
func skillRegistrationSource(skillName string) string {
	return fmt.Sprintf("skill registration: agent-instructions/skills/%s/SKILL.md", skillName)
}
