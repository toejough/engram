//go:build network

package acceptance_test

import (
	"path/filepath"
	"testing"
)

// unexported constants.
const (
	// askStagingReset is a prompt that the stagingReset note answers. The Zorblax staging database is made up, so
	// the agent cannot know the answer without the note.
	askStagingReset = "When is the Zorblax staging database reset?"
	// claudeSettings lets the agent run engram, which the scenarios where it looks a note up itself need.
	claudeSettings = `{"permissions":{"allow":["Bash(engram:*)"]}}`
	learnSource    = "acceptance test"
)

// unexported variables.
var (
	stagingReset = fact{
		slug:      "zorblax-staging-reset",
		situation: "testing against the Zorblax staging database",
		subject:   "the Zorblax staging database",
		predicate: "is reset every",
		object:    "Sunday at 03:40 UTC",
	}
)

// fact is a note of the vault that states one thing.
type fact struct {
	slug      string
	situation string
	subject   string
	predicate string
	object    string
}

// hasClaudeCode installs Claude Code's directory on the machine, with the one permission the scenarios need.
func (m *machine) hasClaudeCode() {
	m.t.Helper()

	mkdirAll(m.t, m.claudeDir)
	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), claudeSettings)
}

// learnsFact writes a fact into the vault and returns the file name of the note.
func (m *machine) learnsFact(note fact) string {
	m.t.Helper()

	out := m.engram("learn", "fact", "--slug", note.slug, "--position", "top", "--source", learnSource,
		"--situation", note.situation, "--subject", note.subject, "--predicate", note.predicate, "--object", note.object)
	m.mustSucceed(out, "learn fact "+note.slug)

	return m.noteNamed(note.slug)
}

// mustSucceed fails the test when a command exited non-zero.
func (m *machine) mustSucceed(out result, what string) {
	m.t.Helper()

	if out.exit != 0 {
		m.t.Fatalf("%s exited %d\nstdout: %s\nstderr: %s", what, out.exit, out.stdout, out.stderr)
	}
}

// noteNamed returns the file name of the vault note with a slug.
func (m *machine) noteNamed(slug string) string {
	m.t.Helper()

	matches, err := filepath.Glob(filepath.Join(m.vault, "*."+slug+".md"))
	if err != nil || len(matches) != 1 {
		m.t.Fatalf("expected one note with the slug %s in %s, found %v (%v)", slug, m.vault, matches, err)
	}

	return filepath.Base(matches[0])
}

// step logs one line of the scenario a test mirrors, and runs it: the test's output reads as the Gherkin (793b9).
func step(t *testing.T, line string, run func()) {
	t.Helper()
	t.Log(line)
	run()
}
