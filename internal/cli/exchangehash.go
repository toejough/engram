package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	exchangeHashPrefix = "xh1:"
)

// hashComparison is the three-way result of comparing two exchange hashes
// (design D3 versioning).
type hashComparison int

// hashComparison values.
const (
	hashUnknown hashComparison = iota
	hashEqual
	hashChanged
)

// exchangeHashFields is the offered content of a note (design D3): exactly
// the keys exchangeKeyClassification marks offered. Absent keys decode to
// empty values.
type exchangeHashFields struct {
	Type      string   `yaml:"type"`
	Situation string   `yaml:"situation"`
	Subject   string   `yaml:"subject"`
	Predicate string   `yaml:"predicate"`
	Object    string   `yaml:"object"`
	Behavior  string   `yaml:"behavior"`
	Impact    string   `yaml:"impact"`
	Action    string   `yaml:"action"`
	DoneWhen  string   `yaml:"done_when"`
	RedFlags  []string `yaml:"red_flags"`
	Triggers  []string `yaml:"triggers"`
}

// canonicalExchangeBody returns the body text an exchange hash covers:
// embed.BodyText with CRLF line endings normalized to LF and exactly one
// final newline (none for an empty body), so the same body hashes the same
// whatever line endings or final newline a file or transport gave it (ruling
// S6). A note whose leading "---" block never closes has no frontmatter, as
// everywhere else in engram, so its whole text is the body.
//
// The text after an LF frontmatter's closing line is normalized before the
// body is extracted, so a CRLF blank separator line is dropped exactly as an
// LF one is: a note with LF frontmatter and a CRLF body hashes the same as
// its LF conversion (fix-show-amend-reparent-frontmatter design D5).
func canonicalExchangeBody(raw []byte) string {
	frontmatter, rest, ok := embed.SplitFrontmatter(raw)
	if ok {
		raw = slices.Concat([]byte(fmStart), frontmatter, []byte(fmStart), toLF(rest))
	}

	body := strings.ReplaceAll(string(embed.BodyText(raw)), "\r\n", "\n")
	body = strings.TrimRight(body, "\n")

	if body == "" {
		return ""
	}

	return body + "\n"
}

// compareExchangeHashes compares two exchange hashes three ways: unknown
// when either side lacks the current version prefix, otherwise equal or
// changed.
func compareExchangeHashes(first, second string) hashComparison {
	if !strings.HasPrefix(first, exchangeHashPrefix) || !strings.HasPrefix(second, exchangeHashPrefix) {
		return hashUnknown
	}

	if first == second {
		return hashEqual
	}

	return hashChanged
}

// exchangeHash returns a note's exchange hash: the version prefix plus the
// sha256 of the canonical (sorted-key) JSON of every offered frontmatter
// field and the note's canonical body text (canonicalExchangeBody), with
// absent fields as empty strings or lists. It is the one function both the
// server and the child use, so the same file hashes the same on both sides.
// It depends on no non-offered field — identity, pending, tags, sources, supersedes, skill
// fields, or the exchange fields themselves.
func exchangeHash(raw []byte) (string, error) {
	var fields exchangeHashFields

	frontmatter, _, ok := embed.SplitFrontmatter(raw)
	if ok {
		unmarshalErr := yaml.Unmarshal(frontmatter, &fields)
		if unmarshalErr != nil {
			return "", fmt.Errorf("exchange hash: parsing frontmatter: %w", unmarshalErr)
		}
	}

	// encoding/json sorts map keys, which makes this the canonical form.
	canonical, _ := json.Marshal(map[string]any{ //nolint:errchkjson // strings and string slices always encode
		"type":      fields.Type,
		"situation": fields.Situation,
		"subject":   fields.Subject,
		"predicate": fields.Predicate,
		"object":    fields.Object,
		"behavior":  fields.Behavior,
		"impact":    fields.Impact,
		"action":    fields.Action,
		"done_when": fields.DoneWhen,
		"red_flags": nonNilStrings(fields.RedFlags),
		"triggers":  nonNilStrings(fields.Triggers),
		"body":      canonicalExchangeBody(raw),
	})
	sum := sha256.Sum256(canonical)

	return exchangeHashPrefix + hex.EncodeToString(sum[:]), nil
}

// exchangeHashChanged reports whether current differs from recorded. It is
// the check for the pull-down skip, decline matching, rejected-entry re-arm,
// and the send/apply change check, all of which treat unknown as not
// changed (design D3).
func exchangeHashChanged(recorded, current string) bool {
	return compareExchangeHashes(recorded, current) == hashChanged
}

// exchangeHashesMatch reports whether two exchange hashes are equal under the
// current version. Loop suppression and dedupe rule 2 fire only on equal
// (design D3), never on unknown.
func exchangeHashesMatch(first, second string) bool {
	return compareExchangeHashes(first, second) == hashEqual
}

// exchangeKeyClassification is the one explicit table that classifies every
// YAML key of the fact, feedback and runbook frontmatter structs as offered
// (true: hashed and sent to the parent) or not offered (false). A
// reflection test fails when a struct key is missing here, so a new field
// cannot silently fall outside the hash (design D3, r3-5).
func exchangeKeyClassification() map[string]bool {
	return map[string]bool{
		// Offered: the note's content (exchangeHashFields).
		"type":      true,
		"situation": true,
		"subject":   true,
		"predicate": true,
		"object":    true,
		"behavior":  true,
		"impact":    true,
		"action":    true,
		"done_when": true,
		"red_flags": true,
		"triggers":  true,
		// Not offered: placement, provenance, identity, bookkeeping,
		// registration, and the exchange's own fields.
		"tier":          false,
		"luhmann":       false,
		"created":       false,
		"source":        false,
		"project":       false,
		"repo":          false,
		"user":          false,
		"vault":         false,
		"pending":       false,
		"issue":         false,
		"sources":       false,
		"vocab_version": false,
		"tags":          false,
		"supersedes":    false,
		"skill_hash":    false,
		"skill_key":     false,
		"skill_source":  false,
		"xid":           false,
		"parent":        false,
		"aliases":       false,
		"offer":         false,
	}
}

// nonNilStrings returns values, or an empty list when it is nil, so an
// absent list and an empty one hash the same.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
