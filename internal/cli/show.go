package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/engram/internal/luhmann"
	"github.com/toejough/engram/internal/vaultgraph"
)

// ShowArgs holds parsed flags for `engram show`.
type ShowArgs struct {
	Ref       string `targ:"positional,required,desc=note ref: full basename | [[wikilink]] | trailing .md | or bare Luhmann id"` //nolint:lll // single struct-tag string
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
}

// ShowDeps holds injected dependencies for RunShow. The command is read-only.
type ShowDeps struct {
	Scan func(vault string) ([]vaultgraph.Note, error)
	Read func(path string) ([]byte, error)
}

// RunShow resolves a note reference to a vault note and prints its content
// plus its outbound wikilink targets, so one fetch reveals the next hop. The
// reference may be a full basename, a [[wikilink]] (brackets and an optional
// |display are tolerated), a trailing .md, or a bare Luhmann id — the bare-id
// case is a normalization pre-step layered over the basename match, since a
// bare id is not itself a resolvable wikilink target. Read-only; no writes.
func RunShow(_ context.Context, args ShowArgs, deps ShowDeps, stdout io.Writer) error {
	ref := normalizeShowRef(args.Ref)
	if ref == "" {
		return errShowEmptyRef
	}

	notes, scanErr := deps.Scan(args.VaultPath)
	if scanErr != nil {
		return fmt.Errorf("show: scan: %w", scanErr)
	}

	note, ok := resolveShowRef(notes, ref)
	if !ok {
		return fmt.Errorf("%w: %q", errShowNoteNotFound, args.Ref)
	}

	notePath := pathOf(note.Basename)

	body, readErr := deps.Read(filepath.Join(args.VaultPath, notePath))
	if readErr != nil {
		return fmt.Errorf("show: read %s: %w", notePath, readErr)
	}

	renderShow(stdout, string(body), note.Outgoing)

	return nil
}

// unexported variables.
var (
	errShowEmptyRef     = errors.New("show: empty note reference")
	errShowNoteNotFound = errors.New("show: note not found")
)

// newOsShowDeps wires RunShow to the real filesystem vault scanner and reader.
func newOsShowDeps() ShowDeps {
	fsys := &osVaultFS{}

	return ShowDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(fsys, vault)
		},
		Read: fsys.ReadFile,
	}
}

// normalizeShowRef canonicalizes a user-supplied reference to a basename or
// bare id: trims surrounding whitespace, strips [[ ]] wikilink brackets and an
// optional |display segment, then drops a trailing .md extension.
func normalizeShowRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "[[")
	ref = strings.TrimSuffix(ref, "]]")

	if pipe := strings.IndexByte(ref, '|'); pipe >= 0 {
		ref = ref[:pipe]
	}

	ref = strings.TrimSpace(ref)
	ref = strings.TrimSuffix(ref, ".md")

	return strings.TrimSpace(ref)
}

// renderShow writes the note content followed by a clearly-delimited list of
// its outbound wikilink targets (the next hops to fetch with `engram show`).
func renderShow(stdout io.Writer, content string, outbound []string) {
	_, _ = io.WriteString(stdout, content)

	if !strings.HasSuffix(content, "\n") {
		_, _ = io.WriteString(stdout, "\n")
	}

	_, _ = fmt.Fprintln(stdout, "\n# outbound links (fetch with: engram show <basename>)")

	if len(outbound) == 0 {
		_, _ = fmt.Fprintln(stdout, "(none)")

		return
	}

	for _, target := range outbound {
		_, _ = fmt.Fprintln(stdout, target)
	}
}

// resolveShowRef finds the note matching ref, preferring an exact basename
// match and falling back to a bare Luhmann id match (computed from each note's
// basename via the canonical luhmann parser, so resolution does not depend on
// Scan having populated Note.LuhmannID).
func resolveShowRef(notes []vaultgraph.Note, ref string) (vaultgraph.Note, bool) {
	for _, note := range notes {
		if note.Basename == ref {
			return note, true
		}
	}

	for _, note := range notes {
		if id, ok := luhmann.FromBasename(note.Basename); ok && id == ref {
			return note, true
		}
	}

	return vaultgraph.Note{}, false
}
