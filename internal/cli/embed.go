package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// EmbedApplyArgs holds parsed flags for `engram embed apply`.
type EmbedApplyArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	All       bool   `targ:"flag,name=all,desc=re-embed every note regardless of state"`
	Missing   bool   `targ:"flag,name=missing,desc=embed only notes without sidecars (default if no mode flag)"`
	Stale     bool   `targ:"flag,name=stale,desc=re-embed notes whose body hash changed"`
	Force     bool   `targ:"flag,name=force,desc=also re-embed sidecars whose model_id differs from the binary"`
	DryRun    bool   `targ:"flag,name=dry-run,desc=report what would change without writing"`
}

// EmbedDeps holds injected dependencies for the embed commands. All
// fields are required by RunEmbedApply / RunEmbedStatus.
type EmbedDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Write    func(path string, data []byte) error
	Embedder embed.Embedder
}

// EmbedStatusArgs holds parsed flags for `engram embed status`.
type EmbedStatusArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
}

// RunEmbedApply walks the vault and (re-)embeds notes per the selection
// flags. Per-note progress lines go to stdout. The vault lock is not
// acquired here — sidecar writes don't collide with luhmann ID assignment.
func RunEmbedApply(
	ctx context.Context,
	args EmbedApplyArgs,
	deps EmbedDeps,
	stdout io.Writer,
) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed apply: scan: %w", err)
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Basename < notes[j].Basename
	})

	selection := selectStates(args)
	modelID := deps.Embedder.ModelID()
	filesystem := readerFS{read: deps.Read}

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		full := filepath.Join(args.VaultPath, notePath)

		state := embed.ComputeState(filesystem, full, modelID)

		if !selection.shouldEmbed(state) {
			continue
		}

		if args.DryRun {
			_, _ = fmt.Fprintf(stdout, "would-embed %s (%s)\n", notePath, state)

			continue
		}

		applyOne(ctx, args.VaultPath, notePath, deps, state, stdout)
	}

	return nil
}

// RunEmbedStatus emits one line per state category to stdout. The
// output shape matches the spike spec verbatim so the values are
// scriptable.
func RunEmbedStatus(
	_ context.Context,
	args EmbedStatusArgs,
	deps EmbedDeps,
	stdout io.Writer,
) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed status: scan: %w", err)
	}

	counts := tallyStates(notes, args.VaultPath, deps)

	return writeStatusReport(stdout, counts)
}

// unexported variables.
var (
	sharedEmbedder = embed.NewLazyEmbedder() //nolint:gochecknoglobals // shared lazy singleton across CLI commands
)

// applySelection captures which states the user asked to re-embed,
// derived from the flag combination on EmbedApplyArgs.
type applySelection struct {
	wantOK, wantMissing, wantStale, wantIncompat bool
}

func (a applySelection) shouldEmbed(state embed.State) bool {
	switch state {
	case embed.StateOK:
		return a.wantOK
	case embed.StateMissing:
		return a.wantMissing
	case embed.StateStale:
		return a.wantStale
	case embed.StateIncompatible:
		return a.wantIncompat
	case embed.StateBroken:
		return a.wantStale || a.wantMissing || a.wantIncompat || a.wantOK
	default:
		return false
	}
}

// osEmbedFS is the production adapter wrapping os.ReadFile/WriteFile +
// vaultgraph.ScanVault behind named methods so coverage tracking treats
// the wiring as one unit measured by integration tests, rather than as
// three anonymous closures that each need their own unit tests.
type osEmbedFS struct{}

// Read reads path via os.ReadFile.
func (osEmbedFS) Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	return data, nil
}

// Scan returns every note in vault via vaultgraph.ScanVault. The
// returned error (if any) is propagated as-is since vaultgraph is an
// internal package; wrapcheck excludes internal packages.
func (osEmbedFS) Scan(vault string) ([]vaultgraph.Note, error) {
	return vaultgraph.ScanVault(&osVaultFS{}, vault)
}

// Write writes data to path via os.WriteFile with 0o600 perms.
func (osEmbedFS) Write(path string, data []byte) error {
	const sidecarPerm = 0o600

	err := os.WriteFile(path, data, sidecarPerm)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// readerFS adapts EmbedDeps.Read to the embed.FS interface so we can
// reuse ComputeState's classification logic.
type readerFS struct {
	read func(string) ([]byte, error)
}

func (r readerFS) ReadFile(path string) ([]byte, error) {
	data, err := r.read(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
		}

		return nil, err
	}

	return data, nil
}

// stateCounts holds the per-category note counts surfaced by status.
type stateCounts struct {
	total, ok, missing, stale, incompat, broken int
}

func applyOne(
	ctx context.Context,
	vault, notePath string,
	deps EmbedDeps,
	state embed.State,
	stdout io.Writer,
) {
	full := filepath.Join(vault, notePath)

	noteBytes, readErr := deps.Read(full)
	if readErr != nil {
		_, _ = fmt.Fprintf(stdout, "skip      %s: read error: %v\n", notePath, readErr)

		return
	}

	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, noteBytes)
	if embErr != nil {
		_, _ = fmt.Fprintf(stdout, "fail      %s: embed: %v\n", notePath, embErr)

		return
	}

	scBytes := embed.MarshalSidecar(sidecar)
	sidecarFull := filepath.Join(vault, embed.SidecarPath(notePath))

	writeErr := deps.Write(sidecarFull, scBytes)
	if writeErr != nil {
		_, _ = fmt.Fprintf(stdout, "fail      %s: write: %v\n", notePath, writeErr)

		return
	}

	_, _ = fmt.Fprintf(stdout, "embedded  %s (%s)\n", notePath, state)
}

// newOsEmbedDeps wires the production filesystem + bundled embedder for
// the embed commands.
func newOsEmbedDeps() EmbedDeps {
	fs := &osEmbedFS{}

	return EmbedDeps{
		Scan:     fs.Scan,
		Read:     fs.Read,
		Write:    fs.Write,
		Embedder: sharedEmbedder,
	}
}

func selectStates(args EmbedApplyArgs) applySelection {
	if args.All {
		return applySelection{wantOK: true, wantMissing: true, wantStale: true, wantIncompat: true}
	}

	selection := applySelection{
		wantMissing:  args.Missing,
		wantStale:    args.Stale,
		wantIncompat: args.Force,
	}

	// No explicit mode → default to embedding missing-sidecar notes.
	if !selection.wantMissing && !selection.wantStale && !selection.wantIncompat {
		selection.wantMissing = true
	}

	return selection
}

func tallyStates(notes []vaultgraph.Note, vault string, deps EmbedDeps) stateCounts {
	counts := stateCounts{total: len(notes)}
	modelID := deps.Embedder.ModelID()
	filesystem := readerFS{read: deps.Read}

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		full := filepath.Join(vault, notePath)

		state := embed.ComputeState(filesystem, full, modelID)

		switch state {
		case embed.StateOK:
			counts.ok++
		case embed.StateMissing:
			counts.missing++
		case embed.StateStale:
			counts.stale++
		case embed.StateIncompatible:
			counts.incompat++
		case embed.StateBroken:
			counts.broken++
		}
	}

	return counts
}

func writeStatusReport(stdout io.Writer, counts stateCounts) error {
	rows := []struct {
		label string
		value int
	}{
		{"total:           ", counts.total},
		{"with-embeddings: ", counts.ok},
		{"without:         ", counts.missing},
		{"stale:           ", counts.stale},
		{"incompatible:    ", counts.incompat},
		{"broken:          ", counts.broken},
	}

	for _, row := range rows {
		_, err := fmt.Fprintf(stdout, "%s%d\n", row.label, row.value)
		if err != nil {
			return fmt.Errorf("embed status: writing output: %w", err)
		}
	}

	return nil
}
