package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestAddThenList(t *testing.T) {
	t.Parallel()

	repo := &memRepository{}
	svc := service{repo: repo}

	var buf bytes.Buffer
	if err := svc.add("hello", nil, &buf); err != nil {
		t.Fatalf("add: %v", err)
	}

	buf.Reset()
	if err := svc.list("", "", false, &buf); err != nil {
		t.Fatalf("list: %v", err)
	}

	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("list missing added note: %q", buf.String())
	}
}

func TestEditMissingIsSentinel(t *testing.T) {
	t.Parallel()

	svc := service{repo: &memRepository{}}
	err := svc.edit(99, "x", &bytes.Buffer{})

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestTagFilterCaseInsensitive(t *testing.T) {
	t.Parallel()

	repo := &memRepository{}
	svc := service{repo: repo}

	_ = svc.add("tagged", []string{"Work"}, &bytes.Buffer{})
	_ = svc.add("control", nil, &bytes.Buffer{})

	var buf bytes.Buffer
	if err := svc.list("work", "", true, &buf); err != nil {
		t.Fatalf("list: %v", err)
	}

	if got := strings.Count(buf.String(), `"id"`); got != 1 {
		t.Fatalf("want 1 match, got %d: %q", got, buf.String())
	}
}

// memRepository is an in-memory fake Repository so the core can be exercised
// without touching a real file. Its presence (a second persistence impl beside
// fileRepository) is the structural signature of fake-driven, parallel-safe tests.
type memRepository struct{ notes []Note }

func (m *memRepository) Load() ([]Note, error) { return m.notes, nil }

func (m *memRepository) Save(notes []Note) error {
	m.notes = append([]Note(nil), notes...)

	return nil
}
