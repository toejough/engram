package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Exported constants.
const (
	SkillOfferRefresh  SkillOfferKind = "refresh"
	SkillOfferRegister SkillOfferKind = "register"
	SkillOfferRemove   SkillOfferKind = "remove"
)

// ShippedSkill is one skill engram ships, as raw `SKILL.md` bytes. A later
// registration-wiring unit reads `agent-instructions/skills/<name>/SKILL.md`
// and supplies it here — this file never reads skill directories itself
// (skill-runbook-registration: "Skill files SHALL carry no engram-specific
// metadata").
type ShippedSkill struct {
	Name    string
	Content []byte
}

// SkillOffer is one pending registration decision produced by
// CompareSkillOffers. Hash is the hash the offer would act on if accepted —
// the skill's current content hash for Register/Refresh, or the note's
// existing skill_hash for Remove — the same value RecordSkillDeclined
// stores against Skill when the offer is declined.
type SkillOffer struct {
	Kind SkillOfferKind
	// Skill is the shipped skill's name (Register/Refresh) or the name
	// extracted from the orphaned note's slug (Remove).
	Skill string
	// Basename is the note's basename; empty for Register, since no note
	// exists yet.
	Basename string
	Hash     string
}

// SkillOfferKind identifies what a SkillOffer proposes: creating a note for
// a newly-shipped skill, refreshing a note whose skill_hash is stale, or
// removing a note whose skill is no longer shipped (skill-runbook-
// registration D4).
type SkillOfferKind string

// CompareSkillOffers computes the deterministic (sorted by skill name) list
// of registration offers: a shipped skill with no note offers Register; one
// whose note's skill_hash differs from the skill's current content hash
// offers Refresh; a skill note whose skill is absent from skills offers
// Remove; hashes matching or the acting hash already declined makes no
// offer (skill-runbook-registration: "Registration SHALL offer, not act,
// and remember declines by hash"). names is a vault ListMD-shaped listing
// of full .md filenames; readFile reads a vault-joined path. A duplicate
// skill note (found by FindSkillNote's identity rule) aborts the whole
// comparison with errDuplicateSkillNote, naming every match, and makes no
// change.
func CompareSkillOffers(
	vault string,
	skills []ShippedSkill,
	names []string,
	readFile func(string) ([]byte, error),
	declined map[string]string,
) ([]SkillOffer, error) {
	byName := groupSkillNoteCandidates(vault, names, readFile)
	shipped := make(map[string]bool, len(skills))
	offers := make([]SkillOffer, 0, len(skills))

	for _, skill := range skills {
		shipped[skill.Name] = true

		offer, hasOffer, err := compareOneShippedSkill(skill, byName[skill.Name], declined)
		if err != nil {
			return nil, err
		}

		if hasOffer {
			offers = append(offers, offer)
		}
	}

	for skillName, matches := range byName {
		if shipped[skillName] {
			continue
		}

		offer, hasOffer, err := removalOfferFor(skillName, matches, declined)
		if err != nil {
			return nil, err
		}

		if hasOffer {
			offers = append(offers, offer)
		}
	}

	sort.Slice(offers, func(i, j int) bool { return offers[i].Skill < offers[j].Skill })

	return offers, nil
}

// FindSkillNote locates skillName's runbook note among the vault's full .md
// filenames (a ListMD-shaped listing): the note whose basename ends with
// ".skill-<name>.md" and whose frontmatter is type runbook with a non-empty
// skill_hash (skill-runbook-registration D3). found is false when no note
// matches; errDuplicateSkillNote names every match when more than one does.
func FindSkillNote(
	vault, skillName string,
	names []string,
	readFile func(string) ([]byte, error),
) (basename string, hash string, found bool, err error) {
	byName := groupSkillNoteCandidates(vault, names, readFile)

	return resolveSkillNoteMatches(byName[skillName])
}

// ReadSkillRegistrations loads the decline state from skill-
// registrations.json at the vault root. A missing file is not an error —
// it degrades to an empty decline state (no skill has ever been declined).
// Any other read failure is propagated: unlike vocab.centroids.json's
// purely-derived data, a decline is user intent, so an unreadable (but
// present) file must not be silently treated as "nothing declined" and
// then re-offer everything. An unrecognized top-level field is rejected
// rather than silently dropped, since this file has no third-party writer
// whose forward-compatible fields it needs to tolerate.
func ReadSkillRegistrations(vault string, readFile func(string) ([]byte, error)) (map[string]string, error) {
	data, readErr := readFile(filepath.Join(vault, skillRegistrationsFilename))
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return map[string]string{}, nil
		}

		return nil, fmt.Errorf("skill registrations: reading %s: %w", skillRegistrationsFilename, readErr)
	}

	var doc skillRegistrationsDoc

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	decodeErr := decoder.Decode(&doc)
	if decodeErr != nil {
		return nil, fmt.Errorf("%w %s: %w", errSkillRegistrationsDecode, skillRegistrationsFilename, decodeErr)
	}

	if doc.Declined == nil {
		doc.Declined = map[string]string{}
	}

	return doc.Declined, nil
}

// RecordSkillDeclined sets skillName's declined hash to hash — the skill's
// current content hash for a Register/Refresh decline, or the note's
// skill_hash for a Remove decline — leaving every other entry untouched,
// and writes skill-registrations.json atomically via writeFile (the
// existing atomic-write dep, composed by the caller).
func RecordSkillDeclined(
	vault, skillName, hash string,
	readFile func(string) ([]byte, error),
	writeFile func(string, []byte) error,
) error {
	declined, readErr := ReadSkillRegistrations(vault, readFile)
	if readErr != nil {
		return readErr
	}

	declined[skillName] = hash

	doc := skillRegistrationsDoc{SchemaVersion: skillRegistrationsSchemaVersion, Declined: declined}

	data, _ := json.Marshal(doc) //nolint:errchkjson // string-keyed string map never fails to encode (vocab_centroids.go)

	writeErr := writeFile(filepath.Join(vault, skillRegistrationsFilename), data)
	if writeErr != nil {
		return fmt.Errorf("skill registrations: writing %s: %w", skillRegistrationsFilename, writeErr)
	}

	return nil
}

// SkillContentHash returns the lowercase hex SHA-256 of a skill's SKILL.md
// bytes — the skill_hash frontmatter value (vault-note-identity spec,
// skill-runbook-registration D3).
func SkillContentHash(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}

// unexported constants.
const (
	// skillRegistrationsFilename is the vault-root decline-state file, a
	// sibling of vocab.centroids.json (vocab_centroids.go).
	skillRegistrationsFilename = "skill-registrations.json"
	// skillRegistrationsSchemaVersion versions skill-registrations.json,
	// mirroring vocab.centroids.json's schema_version convention.
	skillRegistrationsSchemaVersion = 1
	// skillSlugPrefix is the slug prefix a skill's runbook note carries:
	// basename `<luhmann>.<date>.skill-<name>.md` (skill-runbook-
	// registration D3).
	skillSlugPrefix = "skill-"
)

// unexported variables.
var (
	// errDuplicateSkillNote reports more than one runbook note matching a
	// skill's slug suffix and carrying a non-empty skill_hash
	// (skill-runbook-registration: "duplicate matches are an error naming
	// both").
	errDuplicateSkillNote = errors.New("skill-runbook-registration: duplicate skill note")
	// errSkillRegistrationsDecode reports a skill-registrations.json body
	// that fails to decode (malformed JSON or an unrecognized field — see
	// ReadSkillRegistrations).
	errSkillRegistrationsDecode = errors.New("skill registrations: decoding")
)

// skillNoteCandidate is one runbook note in the vault carrying a non-empty
// skill_hash, keyed by the skill name extracted from its "skill-<name>"
// slug (skill-runbook-registration D3).
type skillNoteCandidate struct {
	Basename string
	Hash     string
}

// skillNoteFrontmatterProbe extracts just the fields skill-note identity
// needs from a candidate note's frontmatter: its declared type and its
// skill_hash, if any.
type skillNoteFrontmatterProbe struct {
	Type      string `yaml:"type"`
	SkillHash string `yaml:"skill_hash"`
}

// skillRegistrationsDoc is the on-disk shape of skill-registrations.json —
// the vault-root decline-state file, a sibling of vocab.centroids.json
// (vocab_centroids.go), following the same schema_version convention.
//
//nolint:tagliatelle // skill-registrations JSON keys follow the vocab.centroids.json convention (snake_case)
type skillRegistrationsDoc struct {
	SchemaVersion int               `json:"schema_version"`
	Declined      map[string]string `json:"declined"`
}

// compareOneShippedSkill compares one shipped skill against its resolved
// note matches, returning the offer CompareSkillOffers should record (if
// any). Split out of CompareSkillOffers's loop to keep both functions
// within the repo's cyclomatic-complexity budget.
func compareOneShippedSkill(
	skill ShippedSkill, matches []skillNoteCandidate, declined map[string]string,
) (SkillOffer, bool, error) {
	basename, noteHash, found, resolveErr := resolveSkillNoteMatches(matches)
	if resolveErr != nil {
		return SkillOffer{}, false, resolveErr
	}

	hash := SkillContentHash(skill.Content)

	switch {
	case !found:
		if declined[skill.Name] == hash {
			return SkillOffer{}, false, nil
		}

		return SkillOffer{Kind: SkillOfferRegister, Skill: skill.Name, Hash: hash}, true, nil
	case noteHash != hash:
		if declined[skill.Name] == hash {
			return SkillOffer{}, false, nil
		}

		return SkillOffer{Kind: SkillOfferRefresh, Skill: skill.Name, Basename: basename, Hash: hash}, true, nil
	default:
		return SkillOffer{}, false, nil
	}
}

// groupSkillNoteCandidates scans names for runbook notes carrying a
// non-empty skill_hash whose basename slug has the "skill-<name>" shape,
// grouped by the extracted skill name. A note matching the slug shape but
// failing either check (wrong type, or a runbook with no skill_hash, or an
// unreadable/unparseable file) is excluded — same non-identity as an
// unrelated note with a coincidentally matching slug.
func groupSkillNoteCandidates(
	vault string, names []string, readFile func(string) ([]byte, error),
) map[string][]skillNoteCandidate {
	byName := make(map[string][]skillNoteCandidate)

	for _, name := range names {
		skillName, isSkillSlug := skillNameFromNoteName(name)
		if !isSkillSlug {
			continue
		}

		raw, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			continue
		}

		hash, isSkillNote := skillHashFromFrontmatter(raw)
		if !isSkillNote {
			continue
		}

		byName[skillName] = append(byName[skillName], skillNoteCandidate{
			Basename: strings.TrimSuffix(name, mdExt),
			Hash:     hash,
		})
	}

	return byName
}

// removalOfferFor resolves skillName's note matches and returns a Remove
// offer when the note's skill_hash isn't already declined. Split out of
// CompareSkillOffers's loop to keep both functions within the repo's
// cyclomatic-complexity budget.
func removalOfferFor(
	skillName string, matches []skillNoteCandidate, declined map[string]string,
) (SkillOffer, bool, error) {
	basename, noteHash, found, resolveErr := resolveSkillNoteMatches(matches)
	if resolveErr != nil {
		return SkillOffer{}, false, resolveErr
	}

	if !found || declined[skillName] == noteHash {
		return SkillOffer{}, false, nil
	}

	return SkillOffer{Kind: SkillOfferRemove, Skill: skillName, Basename: basename, Hash: noteHash}, true, nil
}

// resolveSkillNoteMatches reduces a skill name's grouped note candidates to
// a single (basename, hash) pair: found is false for zero matches;
// errDuplicateSkillNote names every match's basename for more than one.
func resolveSkillNoteMatches(matches []skillNoteCandidate) (basename string, hash string, found bool, err error) {
	switch len(matches) {
	case 0:
		return "", "", false, nil
	case 1:
		return matches[0].Basename, matches[0].Hash, true, nil
	default:
		basenames := make([]string, len(matches))
		for i, match := range matches {
			basenames[i] = match.Basename
		}

		return "", "", false, fmt.Errorf("%w: %s", errDuplicateSkillNote, strings.Join(basenames, ", "))
	}
}

// skillHashFromFrontmatter reports the note's skill_hash when raw parses as
// a runbook note's frontmatter carrying a non-empty skill_hash; ok is false
// for any other note type, a runbook with no skill_hash, or unparseable
// content.
func skillHashFromFrontmatter(raw []byte) (hash string, ok bool) {
	frontmatter, hasFrontmatter := splitFrontmatter(raw)
	if !hasFrontmatter {
		return "", false
	}

	var probe skillNoteFrontmatterProbe
	if yaml.Unmarshal(frontmatter, &probe) != nil {
		return "", false
	}

	if probe.Type != typeRunbook || probe.SkillHash == "" {
		return "", false
	}

	return probe.SkillHash, true
}

// skillNameFromNoteName extracts the skill name from a full .md filename
// whose trailing dot-segment (its slug) has the shape "skill-<name>"
// (skill-runbook-registration D3: basename
// "<luhmann>.<date>.skill-<name>.md"). Returns ("", false) when name
// doesn't end in .md or its slug isn't a "skill-" slug.
func skillNameFromNoteName(name string) (string, bool) {
	if !strings.HasSuffix(name, mdExt) {
		return "", false
	}

	stem := strings.TrimSuffix(name, mdExt)

	lastDot := strings.LastIndexByte(stem, '.')
	if lastDot < 0 {
		return "", false
	}

	slug := stem[lastDot+1:]
	if !strings.HasPrefix(slug, skillSlugPrefix) {
		return "", false
	}

	skillName := strings.TrimPrefix(slug, skillSlugPrefix)
	if skillName == "" {
		return "", false
	}

	return skillName, true
}
