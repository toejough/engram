package main

import "testing"

// A serial test with no fake/in-memory implementation — the structural opposite
// of the parallel, fake-driven tests the scorer rewards.
func TestAddAppends(t *testing.T) {
	notes = nil
	notes = append(notes, Note{ID: 1, Text: "a"})

	if len(notes) != 1 || notes[0].ID != 1 {
		t.Fatalf("unexpected notes: %v", notes)
	}
}
