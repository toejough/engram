package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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
		frontmatter, body, found := splitFrontmatterAndBody(item.Content)
		if !found {
			body = item.Content
		}

		_, situation := frontmatterTypeAndSituation([]byte(frontmatter))

		notes = append(notes, auto.Note{
			Path:      item.Path,
			Situation: situation,
			Body:      strings.Join(strings.Fields(body), " "),
		})
	}

	return notes, nil
}
