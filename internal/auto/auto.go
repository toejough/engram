// Package auto is automatic memory: it brings the notes of the vault that bear on a prompt to the agent without the
// agent asking (793a2b). It performs no I/O; its collaborators are injected.
package auto

import (
	"context"
	"fmt"
	"strings"
)

// Lookup finds the notes that bear on a text, best first.
type Lookup func(ctx context.Context, text string) ([]Note, error)

// Note is a note of the vault as the lookup shows it.
type Note struct {
	Path      string // the note's file name
	Situation string // when the note applies
	Body      string // what the note says, on one line
}

// Request is a prompt, as the memory mod hands it over.
type Request struct {
	Text string `json:"text"`
}

// Deliver returns the block of the notes that bear on the prompt (793d4a).
func Deliver(ctx context.Context, request Request, lookup Lookup) (string, error) {
	notes, err := lookup(ctx, request.Text)
	if err != nil {
		return "", fmt.Errorf("looking up the notes: %w", err)
	}

	return render(notes), nil
}

// unexported constants.
const (
	// marker opens a block, so that a reader of the session's record can tell what automatic memory delivered.
	marker = "[engram memory] Notes from the vault that bear on this prompt, delivered by the memory system, " +
		"not typed by the user."
)

// render writes the block: the marker line and one line for each note.
func render(notes []Note) string {
	lines := make([]string, 0, len(notes)+1)
	lines = append(lines, marker)

	for _, note := range notes {
		lines = append(lines, fmt.Sprintf("- %s (%s): %s", note.Path, note.Situation, note.Body))
	}

	return strings.Join(lines, "\n") + "\n"
}
