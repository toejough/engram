package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/auto"
)

// AutoDeliverArgs holds parsed flags for `engram auto deliver`.
type AutoDeliverArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
}

// AutoDeliverDeps holds injected dependencies for RunAutoDeliver.
type AutoDeliverDeps struct {
	// Stdin carries the request: one JSON object with the prompt's text.
	Stdin io.Reader
	// Query is the lookup of notes, the one `engram query` uses.
	Query QueryDeps
	// ReadDir and ReadFile read the session-log directory, where Claude Code keeps the record of each session.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
}

// RunAutoDeliver reads a prompt from standard input and prints the block of the notes that bear on it (793d4a).
func RunAutoDeliver(ctx context.Context, args AutoDeliverArgs, deps AutoDeliverDeps, stdout io.Writer) error {
	var request auto.Request

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&request)
	if decodeErr != nil {
		return fmt.Errorf("auto deliver: reading the request: %w", decodeErr)
	}

	block, err := auto.Deliver(ctx, request, func(ctx context.Context, text string) ([]auto.Note, error) {
		return lookUpNotes(ctx, args.VaultPath, text, deps.Query)
	})
	if err != nil {
		return fmt.Errorf("auto deliver: %w", err)
	}

	_, writeErr := io.WriteString(stdout, block)
	if writeErr != nil {
		return fmt.Errorf("auto deliver: writing the block: %w", writeErr)
	}

	return nil
}

// findRecord reads the record of a session, which Claude Code keeps as <ID>.jsonl in a directory of the session-log
// directory for the project, or in the session-log directory itself.
func findRecord(deps AutoDeliverDeps, dir, session string) []byte {
	name := session + ".jsonl"
	candidates := []string{filepath.Join(dir, name)}

	// a directory that cannot be listed has no project directories to search
	entries, _ := deps.ReadDir(dir)

	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(dir, entry.Name(), name))
		}
	}

	for _, candidate := range candidates {
		record, err := deps.ReadFile(candidate)
		if err == nil {
			return record
		}
	}

	return nil
}

// heldText returns the text of what a session's agent holds of the vault (793d4d). A session whose ID is not a plain
// file name holds nothing: its ID would otherwise name a file outside the directory.
func heldText(deps AutoDeliverDeps, dir, session string) (string, error) {
	if session == "" || session != filepath.Base(session) || session == ".." {
		return "", nil
	}

	return auto.HeldText(findRecord(deps, dir, session)), nil
}

// lookUpNotes runs the lookup of `engram query` for a text, as its message and its only phrase, and reads the notes
// out of the payload it prints.
func lookUpNotes(ctx context.Context, vault, text string, deps QueryDeps) ([]auto.Note, error) {
	var payload bytes.Buffer

	args := QueryArgs{Phrases: []string{text}, Text: text, VaultPath: vault}

	queryErr := RunQuery(ctx, args, deps, &payload)
	if queryErr != nil {
		return nil, fmt.Errorf("looking up %q: %w", text, queryErr)
	}

	return notesOf(payload.Bytes())
}

// newAutoDeliverDeps wires RunAutoDeliver from the injected CLI capabilities — pure composition (#700).
func newAutoDeliverDeps(d Deps) AutoDeliverDeps {
	return AutoDeliverDeps{Stdin: d.Stdin, Query: newQueryDeps(d)}
}

// notesOf reads the notes out of an `engram query` payload.
func notesOf(payload []byte) ([]auto.Note, error) {
	var parsed queryPayload

	parseErr := yaml.Unmarshal(payload, &parsed)
	if parseErr != nil {
		return nil, fmt.Errorf("reading the payload: %w", parseErr)
	}

	notes := make([]auto.Note, 0, len(parsed.Items))

	for _, item := range parsed.Items {
		// an explore pick is sampled from the vocabulary, not matched to the text (793d4a)
		if slices.Equal(item.Provenances, []string{provenanceExplore}) {
			continue
		}

		frontmatter, body, found := splitFrontmatterAndBody(item.Content)
		if !found {
			body = item.Content
		}

		_, situation := frontmatterTypeAndSituation([]byte(frontmatter))

		notes = append(notes, auto.Note{Path: item.Path, Situation: situation, Body: body})
	}

	return notes, nil
}
