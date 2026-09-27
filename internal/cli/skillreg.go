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
	"strings"

	"go.yaml.in/yaml/v3"
)

// Exported constants.
const (
	SkillOfferRefresh  SkillOfferKind = "refresh"
	SkillOfferRegister SkillOfferKind = "register"
	SkillOfferRemove   SkillOfferKind = "remove"
)

// SkillOffer is one pending registration decision produced by
// CompareSkillOffers. Hash is the hash the offer would act on if accepted —
// the skill's current content hash for Register/Refresh, or the note's
// existing skill_hash for Remove — the same value RecordSkillDeclined
// stores against Skill when the offer is declined.
type SkillOffer struct {
	Kind SkillOfferKind
	// Key is the skill key (design D3): the candidate's key for
	// Register/Refresh, or the note's key for Remove.
	Key string
	// ScopeID is the answer scope (design D6): the candidate's scope, or,
	// for a removal, the scope the note's key form names.
	ScopeID string
	// SourcePath is the candidate's resolved source file, or, for a
	// removal, the note's recorded skill_source (empty when it has none).
	SourcePath string
	// Basename is the note's basename; empty for Register, since no note
	// exists yet.
	Basename string
	Hash     string
	// EngramOwned marks an offer whose source lies under an engram-owned
	// skills root (the candidate's resolved path, or a removal's recorded
	// skill_source). The prompts show the source path of every other offer,
	// so the user sees which file they are accepting.
	EngramOwned bool
}

// SkillOfferKind identifies what a SkillOffer proposes: creating a note for
// a newly-shipped skill, refreshing a note whose skill_hash is stale, or
// removing a note whose skill is no longer shipped (skill-runbook-
// registration D4).
type SkillOfferKind string

// FindSkillNote locates skill key key's runbook note among the vault's full
// .md filenames (a ListMD-shaped listing), per groupSkillNoteCandidates'
// identity rule: a runbook note carrying a non-empty skill_hash whose
// recognized skill_key equals key (design D3). A slug never identifies a
// note. found is false when no note matches; errDuplicateSkillNote names
// every match when more than one does.
func FindSkillNote(
	vault, key string,
	names []string,
	readFile func(string) ([]byte, error),
) (basename string, hash string, found bool, err error) {
	byKey := groupSkillNoteCandidates(vault, names, readFile)

	return resolveSkillNoteMatches(byKey[key])
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
//
// Schema versions 1 and 2 read the same way (design D7): a version-1 file's
// names are the bare keys of the same skills. A schema_version above 2 is
// errSkillRegistrationsVersion, never an empty decline state.
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

	if doc.SchemaVersion > skillRegistrationsSchemaVersion {
		return nil, fmt.Errorf("%w: %s has schema_version %d, this engram reads up to %d",
			errSkillRegistrationsVersion, skillRegistrationsFilename, doc.SchemaVersion, skillRegistrationsSchemaVersion)
	}

	if doc.Declined == nil {
		doc.Declined = map[string]string{}
	}

	return doc.Declined, nil
}

// RecordSkillDeclined sets skill key key's declined hash to hash — the
// skill's current content hash for a Register/Refresh decline, or the
// note's skill_hash for a Remove decline — leaving every other entry
// untouched, and writes skill-registrations.json atomically via writeFile
// (the existing atomic-write dep, composed by the caller). Every write
// stamps schema version 2 in the unchanged {schema_version, declined}
// shape, so the first write over a version-1 file migrates it keeping every
// entry (design D7). An unreadable or too-new file is never written.
func RecordSkillDeclined(
	vault, key, hash string,
	readFile func(string) ([]byte, error),
	writeFile func(string, []byte) error,
) error {
	declined, readErr := ReadSkillRegistrations(vault, readFile)
	if readErr != nil {
		return readErr
	}

	declined[key] = hash

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
	// mirroring vocab.centroids.json's schema_version convention: version 2
	// keys declines by skill key (design D7); version 1 keyed them by skill
	// name, which equals the bare key.
	skillRegistrationsSchemaVersion = 2
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
	// errSkillRegistrationsVersion reports a skill-registrations.json
	// written by a newer engram (schema_version above 2, design D7).
	errSkillRegistrationsVersion = errors.New("skill registrations: unsupported schema version")
)

// skillNoteCandidate is one skill note in the vault: a runbook note with a
// "skill-" slug carrying a non-empty skill_hash and a recognized skill_key
// (design D3).
type skillNoteCandidate struct {
	Basename string
	Hash     string
	// Source is the note's recorded skill_source (`~`-relative resolved
	// path); removal eligibility (design D5) reads it for source-rooted key
	// forms.
	Source string
}

// skillNoteFrontmatterProbe extracts just the fields skill-note identity
// needs from a candidate note's frontmatter: its declared type, its
// skill_hash, and its skill_key, if any.
type skillNoteFrontmatterProbe struct {
	Type        string `yaml:"type"`
	SkillHash   string `yaml:"skill_hash"`
	SkillKey    string `yaml:"skill_key"`
	SkillSource string `yaml:"skill_source"`
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

// groupSkillNoteCandidates scans names for skill notes, grouped by their
// skill_key (design D3): runbook notes with a "skill-" slug carrying a
// non-empty skill_hash and a recognized skill_key (parseSkillKey). There is
// no slug fallback: an unkeyed note (skill_hash but no skill_key, like the
// six legacy notes D11 migrates) and a note whose skill_key is unrecognized
// are not skill notes — never matched to a key, never an alias source and
// never a removal candidate. A note failing a check (wrong type, a runbook
// with no skill_hash such as note 820, or an unreadable/unparseable file) is
// excluded the same way. Only "skill-" slugs are read: registration always
// gives a skill note the slug derived from its key (SkillKeySlug), which has
// that prefix.
func groupSkillNoteCandidates(
	vault string, names []string, readFile func(string) ([]byte, error),
) map[string][]skillNoteCandidate {
	byKey := make(map[string][]skillNoteCandidate)

	for _, name := range names {
		if !hasSkillSlug(name) {
			continue
		}

		raw, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			continue
		}

		probe, isSkillNote := skillIdentityFromFrontmatter(raw)
		if !isSkillNote || !parseSkillKey(probe.SkillKey).recognized {
			continue
		}

		byKey[probe.SkillKey] = append(byKey[probe.SkillKey], skillNoteCandidate{
			Basename: strings.TrimSuffix(name, mdExt),
			Hash:     probe.SkillHash,
			Source:   probe.SkillSource,
		})
	}

	return byKey
}

// hasSkillSlug reports whether name is a full .md filename whose trailing
// dot-segment (its slug) is a non-empty "skill-<…>" slug (basename
// "<luhmann>.<date>.skill-<…>.md").
func hasSkillSlug(name string) bool {
	if !strings.HasSuffix(name, mdExt) {
		return false
	}

	stem := strings.TrimSuffix(name, mdExt)

	lastDot := strings.LastIndexByte(stem, '.')
	if lastDot < 0 {
		return false
	}

	remainder, isSkillSlug := strings.CutPrefix(stem[lastDot+1:], skillSlugPrefix)

	return isSkillSlug && remainder != ""
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

// skillIdentityFromFrontmatter reports the note's skill_hash, skill_key
// (empty for an unkeyed note) and skill_source when raw parses as a runbook
// note's frontmatter carrying a non-empty skill_hash; ok is false for any
// other note type, a runbook with no skill_hash, or unparseable content.
func skillIdentityFromFrontmatter(raw []byte) (skillNoteFrontmatterProbe, bool) {
	frontmatter, hasFrontmatter := splitFrontmatter(raw)
	if !hasFrontmatter {
		return skillNoteFrontmatterProbe{}, false
	}

	var probe skillNoteFrontmatterProbe
	if yaml.Unmarshal(frontmatter, &probe) != nil {
		return skillNoteFrontmatterProbe{}, false
	}

	if probe.Type != typeRunbook || probe.SkillHash == "" {
		return skillNoteFrontmatterProbe{}, false
	}

	return probe, true
}
