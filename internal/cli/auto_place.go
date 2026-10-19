package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/toejough/engram/internal/auto"
	"github.com/toejough/engram/internal/luhmann"
)

// AutoPlaceArgs holds parsed flags for `engram auto place`.
type AutoPlaceArgs struct {
	VaultPath   string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
}

// AutoPlaceDeps holds injected dependencies for RunAutoPlace.
type AutoPlaceDeps struct {
	// Stdin carries the reply of the lessons judgement and its carry: one JSON object.
	Stdin io.Reader
	// ReadDir and ReadFile read the vault and the session-log directory.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
}

// RunAutoPlace reads the reply of the lessons judgement from standard input and prints the request for the model that
// places the lessons against the notes the session has seen, or nothing when the reply holds no whole lesson or the
// session has seen no note (855d4h). It looks nothing up: the rest of the vault is not searched.
func RunAutoPlace(_ context.Context, args AutoPlaceArgs, deps AutoPlaceDeps, stdout io.Writer) error {
	var input auto.PlaceInput

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&input)
	if decodeErr != nil {
		return fmt.Errorf("auto place: reading the lessons: %w", decodeErr)
	}

	lessons, lessonsErr := auto.LessonsOf(input.Reply)
	if lessonsErr != nil {
		return fmt.Errorf("auto place: %w", lessonsErr)
	}

	if len(lessons) == 0 {
		return nil
	}

	vault := vaultNotes(deps.ReadDir, deps.ReadFile, args.VaultPath)

	lessons = auto.Uncovered(lessons, vault)
	if len(lessons) == 0 {
		return nil
	}

	held := auto.HeldText(recordOf(deps.ReadDir, deps.ReadFile, args.SessionsDir, input.Session))

	step, found := auto.Place(lessons, auto.SeenNotes(vault, held, input.Session))
	if !found {
		return nil
	}

	return printStep(stdout, "auto place", step)
}

// newAutoPlaceDeps wires RunAutoPlace from the injected CLI capabilities - pure composition (#700).
func newAutoPlaceDeps(d Deps) AutoPlaceDeps {
	return AutoPlaceDeps{Stdin: d.Stdin, ReadDir: d.FS.ReadDir, ReadFile: d.FS.ReadFile}
}

// vaultNotes reads the notes of a vault: the files named for a Luhmann ID, which the vault's own README is not. A vault
// that cannot be listed holds no note the session has seen: the writing reports why it cannot write there.
func vaultNotes(
	readDir func(path string) ([]fs.DirEntry, error), readFile func(path string) ([]byte, error), vault string,
) []auto.Note {
	// a vault that cannot be listed has no notes
	entries, _ := readDir(vault)
	notes := make([]auto.Note, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()

		if _, isNote := luhmann.FromBasename(strings.TrimSuffix(name, ".md")); entry.IsDir() || !isNote ||
			!strings.HasSuffix(name, ".md") {
			continue
		}

		data, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			continue
		}

		fields, body := noteOf(queryItem{Content: string(data)})
		notes = append(notes, auto.Note{Path: name, Situation: fields.Situation, Body: body, Source: fields.Source,
			Created: fields.Created, Type: fields.Type})
	}

	return notes
}
