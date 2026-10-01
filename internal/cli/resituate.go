package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
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
	// func. Wired via vaultLockFromLocker in newResituateDeps. Guards the note
	// read-modify-write against concurrent amend/resituate/learn runs.
	Lock     func(vault string) (func(), error)
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Write    func(path string, data []byte) error
	Embedder embed.Embedder
	// Optional fields — nil skips vocab assignment and trigger check.
	LoadTermVectors func(vault string) ([]TermWithVector, error)
	ListMD          func(vault string) ([]string, error)
	LogWarning      func(format string, args ...any)
	Now             func() time.Time
	// Offers connects resituate to the parent outbox (design D5, D6, E41).
	// The zero value — no parent configured — offers nothing.
	Offers offerHooks
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
	queued, err := runResituateLocked(ctx, args, deps, stdout)
	if err != nil {
		return err
	}

	// The drain runs after the vault lock is released (design D6).
	if queued {
		deps.Offers.drain(ctx, args.Vault)
	}

	return nil
}

// unexported variables.
var (
	errResituateFrontmatter  = errors.New("resituate: note has no parseable frontmatter")
	errResituateNoteNotFound = errors.New("resituate: note not found")
	errResituateUnknownType  = errors.New("resituate: unknown note type")
)

// applyResituateVocabAssignment performs only the term-assignment part of
// applyVocabAssignmentAfterResituate, keeping the trigger check outside this
// early-return chain.
func applyResituateVocabAssignment(deps ResituateDeps, vault, notePath, content string) {
	applyVocabAssignmentCore(
		deps.LoadTermVectors, deps.Read, deps.Write, deps.LogWarning,
		vault, notePath, content, "resituate")
}

// applyVocabAssignmentAfterResituate assigns vocab terms and checks the refit
// trigger for a resituated note. Mirrors applyVocabAssignmentAfterAmend.
// Requires LoadTermVectors + Read + Write for assignment (gated inside helper).
// Requires Now to be non-nil for the trigger check (ListMD is gated inside callee).
func applyVocabAssignmentAfterResituate(deps ResituateDeps, vault, notePath, content string) {
	applyResituateVocabAssignment(deps, vault, notePath, content)

	if deps.Now == nil {
		return // trigger check needs an injected clock; wiring provides the injected clock
	}

	// Trigger check — uses ListMD dep (optional, gated inside the callee).
	checkAndPersistVocabRefitTrigger(
		vault, deps.ListMD, deps.Read, deps.Write, deps.LogWarning, deps.Now(),
	)

	warnIfPendingOffers(vault, deps.ListMD, deps.Read, deps.LogWarning)
}

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

// newResituateDeps composes RunResituate's dependencies from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newResituateDeps(d Deps) ResituateDeps {
	vfs := newVaultFS(d.FS)

	return ResituateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vfs, vault)
		},
		Read:     vfs.ReadFile,
		Write:    writeAtomicWithPathFromFS(d.FS),
		Embedder: d.Embed,
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vfs.ListMD, vfs.ReadFile)
		},
		ListMD:     vfs.ListMD,
		LogWarning: logWarningTo(d.Stderr),
		Now:        d.Now,
		Offers:     newOfferHooks(d),
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

// rerenderFact rewrites a fact note's situation: and body opener, leaving
// every other frontmatter field and the rest of the body as they were.
func rerenderFact(frontmatter, body []byte, situation string) (string, error) {
	return resituateTyped(frontmatter, body, situation, "fact",
		func(doc *factFrontmatterDoc, situation string) string {
			doc.Situation = situation

			return doc.Created
		},
		func(doc factFrontmatterDoc) string {
			return renderFactBody(factFields{
				Situation: doc.Situation, Subject: doc.Subject, Predicate: doc.Predicate, Object: doc.Object,
			})
		})
}

// rerenderFeedback rewrites a feedback note's situation: and body opener,
// leaving every other frontmatter field and the rest of the body as they
// were.
func rerenderFeedback(frontmatter, body []byte, situation string) (string, error) {
	return resituateTyped(frontmatter, body, situation, "feedback",
		func(doc *feedbackFrontmatterDoc, situation string) string {
			doc.Situation = situation

			return doc.Created
		},
		func(doc feedbackFrontmatterDoc) string {
			return renderFeedbackBody(feedbackFields{Situation: doc.Situation, Action: doc.Action})
		})
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

// resituateTyped edits a note's frontmatter as a YAML node with only its
// situation replaced (design D10 M7, D6b): every other frontmatter key,
// including keys the typed doc does not define, survives with its value —
// pending, sources, tags, supersedes, vocab_version, issue, project and the
// exchange fields among them. The typed doc is decoded from the node only
// to validate the note and to build the body opener. The body keeps
// everything after its first line; only that opener is rebuilt around the
// new situation.
//
// created: is re-emitted as the string the note holds (quoted by the
// frontmatter writer, as learn writes it), not rebuilt from the parsed date.
// It is still parsed, only to refuse a malformed note untouched:
// parseCreated is strict about the 2006-01-02 layout, so any value that
// passes is already in canonical form, and keeping the string means
// resituate can never reformat a date it did not change.
func resituateTyped[T any](
	frontmatter, body []byte,
	situation, kind string,
	resituate func(doc *T, situation string) (created string),
	opener func(doc T) string,
) (string, error) {
	mapping, parseErr := parseFrontmatterMapping(frontmatter)
	if parseErr != nil {
		return "", fmt.Errorf("resituate: parsing %s frontmatter: %w", kind, parseErr)
	}

	var doc T

	decodeErr := mapping.Decode(&doc)
	if decodeErr != nil {
		return "", fmt.Errorf("resituate: parsing %s frontmatter: %w", kind, decodeErr)
	}

	created := resituate(&doc, situation)

	_, createdErr := parseCreated(created)
	if createdErr != nil {
		return "", createdErr
	}

	setMappingValue(mapping, "situation", encodeNode(situation))
	setMappingValue(mapping, "created", encodeNode(created))

	newOpener, _, _ := strings.Cut(opener(doc), "\n")
	_, rest, _ := bytes.Cut(body, []byte("\n"))

	return marshalFrontmatter(mapping) + newOpener + "\n" + string(rest), nil
}

// runResituateLocked is RunResituate's locked section: the rewrite, the
// re-embed, and — with a parent configured — the offer's xid stamp and
// outbox entry in the same critical section (design D6). It reports
// whether an offer was queued.
func runResituateLocked(ctx context.Context, args ResituateArgs, deps ResituateDeps, stdout io.Writer) (bool, error) {
	// Acquire the vault lock before any read-modify-write on the note so
	// concurrent amend/resituate/learn runs cannot produce lost updates.
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return false, fmt.Errorf("resituate: acquiring vault lock: %w", lockErr)
	}

	defer release()

	notes, scanErr := deps.Scan(args.Vault)
	if scanErr != nil {
		return false, fmt.Errorf("resituate: scan: %w", scanErr)
	}

	relPath, findErr := findNote(notes, args.Note)
	if findErr != nil {
		return false, findErr
	}

	full := filepath.Join(args.Vault, relPath)

	raw, readErr := deps.Read(full)
	if readErr != nil {
		return false, fmt.Errorf("resituate: read %s: %w", relPath, readErr)
	}

	rendered, renderErr := resituateContent(raw, args.Situation)
	if renderErr != nil {
		return false, renderErr
	}

	staged, xid := deps.Offers.stageWrite(args.Vault, offerWrite{command: offerCmdResituate, raw: []byte(rendered)})
	content := string(staged)

	writeErr := deps.Write(full, staged)
	if writeErr != nil {
		return false, fmt.Errorf("resituate: write %s: %w", relPath, writeErr)
	}

	queued := deps.Offers.queue(args.Vault, xid)

	embedErr := writeResituatedSidecar(ctx, deps, full, content)
	if embedErr != nil {
		return queued, embedErr
	}

	applyVocabAssignmentAfterResituate(deps, args.Vault, full, content)

	_, _ = fmt.Fprintln(stdout, full)

	return queued, nil
}

// splitFrontmatter returns the YAML bytes between the leading "---\n" line
// and the closing "---\n" line (embed.SplitFrontmatter: the closing
// delimiter must be a whole line, so a value ending in "---" does not end
// the block early). Returns (nil, false) when the note has no leading
// frontmatter block.
func splitFrontmatter(raw []byte) ([]byte, bool) {
	frontmatter, _, ok := embed.SplitFrontmatter(raw)
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
