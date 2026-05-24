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
	DryRun    bool   `targ:"flag,name=dry-run,desc=report what would change, don't write"`
}

// EmbedStatusArgs holds parsed flags for `engram embed status`.
type EmbedStatusArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
}

// EmbedDeps holds injected dependencies for the embed commands. All
// fields are required by RunEmbedApply / RunEmbedStatus.
type EmbedDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Write    func(path string, data []byte) error
	Embedder embed.Embedder
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

// RunEmbedStatus emits one line per state category to stdout. The
// output shape matches the spike spec verbatim so the values are
// scriptable.
func RunEmbedStatus(_ context.Context, args EmbedStatusArgs, deps EmbedDeps, stdout io.Writer) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed status: scan: %w", err)
	}

	counts := tallyStates(notes, args.VaultPath, deps)

	return writeStatusReport(stdout, counts)
}

// stateCounts holds the per-category note counts surfaced by status.
type stateCounts struct {
	total, ok, missing, stale, incompat, broken int
}

func tallyStates(notes []vaultgraph.Note, vault string, deps EmbedDeps) stateCounts {
	counts := stateCounts{total: len(notes)}
	modelID := deps.Embedder.ModelID()
	filesystem := readerFS{read: deps.Read}

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		full := filepath.Join(vault, notePath)

		state, stateErr := embed.ComputeState(filesystem, full, modelID)
		if stateErr != nil {
			counts.broken++

			continue
		}

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

// applySelection captures which states the user asked to re-embed,
// derived from the flag combination on EmbedApplyArgs.
type applySelection struct {
	wantOK, wantMissing, wantStale, wantIncompat bool
}

func selectStates(args EmbedApplyArgs) applySelection {
	return applySelection{
		wantOK:       args.All,
		wantMissing:  args.Missing || (!args.All && !args.Stale),
		wantStale:    args.Stale || args.All,
		wantIncompat: args.Force || args.All,
	}
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

// RunEmbedApply walks the vault and (re-)embeds notes per the selection
// flags. Per-note progress lines go to stdout. The vault lock is not
// acquired here — sidecar writes don't collide with luhmann ID assignment.
func RunEmbedApply(ctx context.Context, args EmbedApplyArgs, deps EmbedDeps, stdout io.Writer) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed apply: scan: %w", err)
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Basename < notes[j].Basename
	})

	selection := selectStates(args)
	modelID := deps.Embedder.ModelID()
	dims := deps.Embedder.Dims()
	filesystem := readerFS{read: deps.Read}

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		full := filepath.Join(args.VaultPath, notePath)

		state, stateErr := embed.ComputeState(filesystem, full, modelID)
		if stateErr != nil {
			_, _ = fmt.Fprintf(stdout, "broken    %s: %v\n", notePath, stateErr)

			continue
		}

		if !selection.shouldEmbed(state) {
			continue
		}

		if args.DryRun {
			_, _ = fmt.Fprintf(stdout, "would-embed %s (%s)\n", notePath, state)

			continue
		}

		applyOne(ctx, args.VaultPath, notePath, deps, modelID, dims, state, stdout)
	}

	return nil
}

func applyOne(
	ctx context.Context,
	vault, notePath string,
	deps EmbedDeps,
	modelID string,
	dims int,
	state embed.State,
	stdout io.Writer,
) {
	full := filepath.Join(vault, notePath)

	noteBytes, readErr := deps.Read(full)
	if readErr != nil {
		_, _ = fmt.Fprintf(stdout, "skip      %s: read error: %v\n", notePath, readErr)

		return
	}

	body := embed.ExtractBody(noteBytes)

	vector, embErr := deps.Embedder.Embed(ctx, string(body))
	if embErr != nil {
		_, _ = fmt.Fprintf(stdout, "fail      %s: embed: %v\n", notePath, embErr)

		return
	}

	sidecar := embed.Sidecar{
		EmbeddingModelID: modelID,
		Dims:             dims,
		Vector:           vector,
		ContentHash:      embed.ContentHash(noteBytes),
	}

	scBytes, marshalErr := embed.MarshalSidecar(sidecar)
	if marshalErr != nil {
		_, _ = fmt.Fprintf(stdout, "fail      %s: marshal: %v\n", notePath, marshalErr)

		return
	}

	sidecarFull := filepath.Join(vault, embed.SidecarPath(notePath))

	writeErr := deps.Write(sidecarFull, scBytes)
	if writeErr != nil {
		_, _ = fmt.Fprintf(stdout, "fail      %s: write: %v\n", notePath, writeErr)

		return
	}

	_, _ = fmt.Fprintf(stdout, "embedded  %s (%s)\n", notePath, state)
}

// sharedEmbedder is the process-wide lazy embedder, so a single
// invocation that touches multiple commands (e.g., learn → auto-embed,
// embed apply → status) doesn't pay the model-unpack cost twice.
var sharedEmbedder = embed.NewLazyEmbedder()

// newOsEmbedDeps wires the production filesystem + bundled embedder for
// the embed commands.
func newOsEmbedDeps() EmbedDeps {
	const sidecarPerm = 0o600

	return EmbedDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			notes, err := vaultgraph.ScanVault(&osVaultFS{}, vault)
			if err != nil {
				return nil, fmt.Errorf("scan vault: %w", err)
			}

			return notes, nil
		},
		Read: func(path string) ([]byte, error) {
			data, err := os.ReadFile(path) //nolint:gosec // path from caller
			if err != nil {
				return nil, fmt.Errorf("read: %w", err)
			}

			return data, nil
		},
		Write: func(path string, data []byte) error {
			err := os.WriteFile(path, data, sidecarPerm) //nolint:gosec // path from caller
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			return nil
		},
		Embedder: sharedEmbedder,
	}
}
