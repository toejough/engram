package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ResituateArgs holds parsed flags for `engram resituate`.
type ResituateArgs struct {
	Vault     string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Note      string `targ:"flag,name=note,required,desc=note ref: full basename | [[wikilink]] | trailing .md | or bare Luhmann id (required)"` //nolint:lll // single unbreakable struct-tag string
	Situation string `targ:"flag,name=situation,required,desc=the new situation to write into the note (required)"`
}

// ResituateDeps holds injected dependencies for RunResituate.
type ResituateDeps struct {
	// Lock acquires an exclusive flock on vault/.luhmann.lock and returns a release
	// func. Wired to vaultFS.Lock in newOsResituateDeps. Guards the note
	// read-modify-write against concurrent amend/resituate/learn runs.
	Lock     func(vault string) (func(), error)
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Write    func(path string, data []byte) error
	Embedder embed.Embedder
}

// RunResituate rewrites a single note's situation in both places it lives —
// the `situation:` frontmatter field and the body prose formula (for fact and
// feedback notes) — then re-embeds the note so its sidecar vector and
// content_hash track the new situation. This closes the INV-S2 divergence:
// `engram learn` is create-only, so before this command the two situation
// copies could only be kept in sync by hand.
//
// args.Vault must already be resolved by the caller via resolveVault.
func RunResituate(
	ctx context.Context,
	args ResituateArgs,
	deps ResituateDeps,
	stdout io.Writer,
) error {
	// Acquire the vault lock before any read-modify-write on the note so
	// concurrent amend/resituate/learn runs cannot produce lost updates.
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("resituate: acquiring vault lock: %w", lockErr)
	}

	defer release()

	notes, scanErr := deps.Scan(args.Vault)
	if scanErr != nil {
		return fmt.Errorf("resituate: scan: %w", scanErr)
	}

	relPath, findErr := findNote(notes, args.Note)
	if findErr != nil {
		return findErr
	}

	full := filepath.Join(args.Vault, relPath)

	raw, readErr := deps.Read(full)
	if readErr != nil {
		return fmt.Errorf("resituate: read %s: %w", relPath, readErr)
	}

	content, renderErr := resituateContent(raw, args.Situation)
	if renderErr != nil {
		return renderErr
	}

	writeErr := deps.Write(full, []byte(content))
	if writeErr != nil {
		return fmt.Errorf("resituate: write %s: %w", relPath, writeErr)
	}

	embedErr := writeResituatedSidecar(ctx, deps, full, content)
	if embedErr != nil {
		return embedErr
	}

	_, _ = fmt.Fprintln(stdout, full)

	return nil
}

// unexported variables.
var (
	errResituateFrontmatter  = errors.New("resituate: note has no parseable frontmatter")
	errResituateNoteNotFound = errors.New("resituate: note not found")
	errResituateUnknownType  = errors.New("resituate: unknown note type")
)

// findNote locates the note whose leading luhmann id OR full basename matches
// target, returning its vault-relative path. The target is normalized first
// (strips [[wikilink]] brackets, trailing .md) so all accepted ref forms work.
// The not-found error quotes the caller's original ref, not the normalized one.
// Returns errResituateNoteNotFound when nothing matches.
func findNote(notes []vaultgraph.Note, target string) (string, error) {
	original := target
	target = normalizeNoteRef(target)

	for _, note := range notes {
		if note.LuhmannID == target || note.Basename == target {
			return pathOf(note.Basename), nil
		}
	}

	return "", fmt.Errorf("%w: %q", errResituateNoteNotFound, original)
}

// newOsResituateDeps wires RunResituate to the real filesystem + the bundled
// embedder.
func newOsResituateDeps() ResituateDeps {
	const perm = 0o600

	return ResituateDeps{
		Lock: (&osLearnFS{}).Lock,
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
		Read: (&osVaultFS{}).ReadFile,
		Write: func(path string, data []byte) error {
			err := atomicWriteFile(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: sharedEmbedder,
	}
}

// parseCreated parses a note's `created:` date back into a time.Time so the
// re-render preserves it rather than stamping today.
func parseCreated(created string) (time.Time, error) {
	when, err := time.Parse(dateFormat, created)
	if err != nil {
		return time.Time{}, fmt.Errorf("resituate: parsing created date %q: %w", created, err)
	}

	return when, nil
}

// peekNoteType extracts the top-level `type:` value from a frontmatter YAML
// block so we can pick the matching render path before a full unmarshal.
func peekNoteType(frontmatter []byte) string {
	var probe struct {
		Type string `yaml:"type"`
	}

	err := yaml.Unmarshal(frontmatter, &probe)
	if err != nil {
		return ""
	}

	return probe.Type
}

// relatedTail returns the related-to section of a fact/feedback body: the body
// is `formula-line\n` followed by `\nRelated to:\n...`. Cutting the first line
// and the single blank line after it yields the tail in the exact shape
// renderFactBody / renderFeedbackBody re-prepend, so an empty tail round-trips
// to `formula\n\n` and a populated tail round-trips byte-identically.
func relatedTail(body []byte) string {
	_, after, found := bytes.Cut(body, []byte("\n"))
	if !found {
		return ""
	}

	return string(bytes.TrimPrefix(after, []byte("\n")))
}

// rerenderFact rebuilds a fact note with the new situation in both the
// frontmatter and the body formula, preserving the existing related-to tail.
func rerenderFact(frontmatter, body []byte, situation string) (string, error) {
	var doc factFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", fmt.Errorf("resituate: parsing fact frontmatter: %w", unmarshalErr)
	}

	when, createdErr := parseCreated(doc.Created)
	if createdErr != nil {
		return "", createdErr
	}

	fields := factFields{
		Situation: situation,
		Subject:   doc.Subject,
		Predicate: doc.Predicate,
		Object:    doc.Object,
		Luhmann:   string(doc.Luhmann),
		Source:    doc.Source,
		Project:   doc.Project,
		Issue:     string(doc.Issue),
		Tier:      doc.Tier,
	}

	return renderFactFrontmatter(fields, when) + renderFactBody(fields) + relatedTail(body), nil
}

// rerenderFeedback rebuilds a feedback note with the new situation in both the
// frontmatter and the body formula, preserving the existing related-to tail.
func rerenderFeedback(frontmatter, body []byte, situation string) (string, error) {
	var doc feedbackFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return "", fmt.Errorf("resituate: parsing feedback frontmatter: %w", unmarshalErr)
	}

	when, createdErr := parseCreated(doc.Created)
	if createdErr != nil {
		return "", createdErr
	}

	fields := feedbackFields{
		Situation: situation,
		Behavior:  doc.Behavior,
		Impact:    doc.Impact,
		Action:    doc.Action,
		Luhmann:   string(doc.Luhmann),
		Source:    doc.Source,
		Project:   doc.Project,
		Issue:     string(doc.Issue),
		Tier:      doc.Tier,
	}

	return renderFeedbackFrontmatter(fields, when) + renderFeedbackBody(fields) + relatedTail(body), nil
}

// resituateContent re-renders raw with situation replaced. For fact and
// feedback notes both the frontmatter and the body formula carry the new
// situation; the existing related-to tail is preserved. The original `created`
// date is parsed from the note and preserved so the rewrite touches only the
// situation.
func resituateContent(raw []byte, situation string) (string, error) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", errResituateFrontmatter
	}

	noteType := peekNoteType(frontmatter)
	body := embed.ExtractBody(raw)

	switch noteType {
	case typeFact:
		return rerenderFact(frontmatter, body, situation)
	case typeFeedback:
		return rerenderFeedback(frontmatter, body, situation)
	default:
		return "", fmt.Errorf("%w: %q", errResituateUnknownType, noteType)
	}
}

// splitFrontmatter returns the YAML bytes between the leading "---\n" and the
// next "---\n" delimiter. Returns (nil, false) when the note has no leading
// frontmatter block.
func splitFrontmatter(raw []byte) ([]byte, bool) {
	delim := []byte("---\n")
	if !bytes.HasPrefix(raw, delim) {
		return nil, false
	}

	frontmatter, _, ok := bytes.Cut(raw[len(delim):], delim)
	if !ok {
		return nil, false
	}

	return frontmatter, true
}

// writeResituatedSidecar re-embeds the rewritten note and writes its sidecar.
// BuildSidecar embeds both the situation: field and the body, so the
// content_hash tracks exactly what changed. Unlike the learn-time auto-embed,
// a resituate is an explicit rewrite, so embed and write failures are surfaced
// rather than warned-and-ignored.
func writeResituatedSidecar(
	ctx context.Context,
	deps ResituateDeps,
	notePath, content string,
) error {
	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		return fmt.Errorf("resituate: embedding %s: %w", notePath, embErr)
	}

	writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("resituate: writing sidecar for %s: %w", notePath, writeErr)
	}

	return nil
}
