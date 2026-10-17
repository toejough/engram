//go:build network

package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unexported constants.
const (
	// askStagingReset is a prompt that the stagingReset note answers. The Zorblax staging database is made up, so
	// the agent cannot know the answer without the note.
	askStagingReset = "When is the Zorblax staging database reset?"
	// askStagingResetDay is another prompt that the stagingReset note answers.
	askStagingResetDay = "What day of the week is the Zorblax staging database reset on?"
	// claudeSettings lets the agent run engram, which the scenarios where it looks a note up itself need.
	claudeSettings = `{"permissions":{"allow":["Bash(engram:*)"]}}`
	learnSource    = "acceptance test"
	modName        = "engram-memory"
)

// unexported variables.
var (
	formatPython = feedback{
		slug:      "format-python-before-commit",
		situation: "committing a Python file",
		behavior:  "left trailing whitespace in a Python file",
		impact:    "the lint check failed",
		action:    "run black on the file before committing",
	}
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

// feedback is a note of the vault that corrects a behavior.
type feedback struct {
	slug      string
	situation string
	behavior  string
	impact    string
	action    string
}

// hasClaudeCode installs Claude Code's directory on the machine, with the one permission the scenarios need.
func (m *machine) hasClaudeCode() {
	m.t.Helper()

	mkdirAll(m.t, m.claudeDir)
	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), claudeSettings)
}

// hasMod installs the mod of this branch as `engram update` does: its files in the engram-owned root, and a link to
// them from Claude Code's skills directory.
func (m *machine) hasMod() {
	m.t.Helper()

	root, err := moduleRoot()
	if err != nil {
		m.t.Fatalf("finding the module: %v", err)
	}

	target := filepath.Join(m.claudeDir, "engram", "skills", modName)

	copyErr := copyTree(filepath.Join(root, "agent-instructions", "mods", modName), target, false)
	if copyErr != nil {
		m.t.Fatalf("installing the mod: %v", copyErr)
	}

	mkdirAll(m.t, filepath.Join(m.claudeDir, "skills"))

	linkErr := os.Symlink(target, filepath.Join(m.claudeDir, "skills", modName))
	if linkErr != nil {
		m.t.Fatalf("linking the mod: %v", linkErr)
	}
}

// hasStandInEngram puts a stand-in for engram first on the machine's PATH: it records its arguments and standard
// input, and prints the block it is given, whatever it is asked.
func (m *machine) hasStandInEngram(block string) {
	m.t.Helper()

	dir := filepath.Join(m.root, "stand-in")
	mkdirAll(m.t, dir)
	writeFile(m.t, filepath.Join(dir, "block"), block)

	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + filepath.Join(dir, "args") + "'\ncat > '" +
		filepath.Join(dir, "stdin") + "'\ncat '" + filepath.Join(dir, "block") + "'\n"
	path := filepath.Join(m.gobin, engramBinaryName)

	mkdirAll(m.t, m.gobin)
	writeFile(m.t, path, script)

	err := os.Chmod(path, 0o700)
	if err != nil {
		m.t.Fatalf("making the stand-in executable: %v", err)
	}
}

// hookFailures returns the lines of the machine's Claude Code debug logs that say the mod's hook failed and was
// skipped: a prompt then goes on as if the mod were not installed, but the user sees that it failed.
func (m *machine) hookFailures() []string {
	var failures []string

	for line := range strings.SplitSeq(m.hookLog(), "\n") {
		if strings.Contains(line, "engram-memory") && strings.Contains(line, "skipped") {
			failures = append(failures, line)
		}
	}

	return failures
}

// hookLog returns the lines of the machine's Claude Code debug logs that say what became of its modules and hooks.
func (m *machine) hookLog() string {
	logs, _ := filepath.Glob(filepath.Join(m.root, "claude-debug-*.log"))

	var lines []string

	for _, name := range logs {
		data, err := os.ReadFile(name)
		if err != nil {
			continue
		}

		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, "engram-memory") || strings.Contains(line, "hooks module") ||
				strings.Contains(line, "prompt.submit") || strings.Contains(line, "$.process.run") {
				lines = append(lines, line)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// learnsFact writes a fact into the vault and returns the file name of the note.
func (m *machine) learnsFact(note fact) string {
	m.t.Helper()

	out := m.engram("learn", "fact", "--slug", note.slug, "--position", "top", "--source", learnSource,
		"--situation", note.situation, "--subject", note.subject, "--predicate", note.predicate, "--object", note.object)
	m.mustSucceed(out, "learn fact "+note.slug)

	return m.noteNamed(note.slug)
}

// learnsFeedback writes a feedback note into the vault and returns the file name of the note.
func (m *machine) learnsFeedback(note feedback) string {
	m.t.Helper()

	out := m.engram("learn", "feedback", "--slug", note.slug, "--position", "top", "--source", learnSource,
		"--situation", note.situation, "--behavior", note.behavior, "--impact", note.impact, "--action", note.action)
	m.mustSucceed(out, "learn feedback "+note.slug)

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

// standInCalled returns the arguments and the request, the JSON object on standard input, that the stand-in engram was
// run with.
func (m *machine) standInCalled() (string, map[string]string) {
	m.t.Helper()

	args, argsErr := os.ReadFile(filepath.Join(m.root, "stand-in", "args"))
	stdin, stdinErr := os.ReadFile(filepath.Join(m.root, "stand-in", "stdin"))

	if argsErr != nil || stdinErr != nil {
		m.t.Fatalf("the stand-in engram was never run (%v, %v)\n%s", argsErr, stdinErr, m.hookLog())
	}

	request := map[string]string{}

	err := json.Unmarshal(stdin, &request)
	if err != nil {
		m.t.Fatalf("the stand-in engram's standard input is not a JSON object: %v\n%s", err, stdin)
	}

	return strings.TrimSpace(string(args)), request
}

// step logs one line of the scenario a test mirrors, and runs it: the test's output reads as the Gherkin (793b9).
func step(t *testing.T, line string, run func()) {
	t.Helper()
	t.Log(line)
	run()
}
