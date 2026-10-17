//go:build network

package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// unexported constants.
const (
	// askStagingQuery makes the agent look the stagingReset note up itself, with a command that names no word of the
	// note's situation.
	askStagingQuery = "Run this shell command and tell me how many items it returned: " +
		"engram query --phrase 'staging database reset schedule' --limit 5"
	// askStagingReset is a prompt that the stagingReset note answers. The Zorblax staging database is made up, so
	// the agent cannot know the answer without the note.
	askStagingReset = "When is the Zorblax staging database reset?"
	// askStagingResetDay is another prompt that the stagingReset note answers.
	askStagingResetDay = "What day of the week is the Zorblax staging database reset on?"
	// claudeSettings lets the agent run engram, which the scenarios where it looks a note up itself need.
	claudeSettings = `{"permissions":{"allow":["Bash(engram:*)"]}}`
	// earlierConversation is a conversation that answers askStagingReset.
	earlierConversation = "In an earlier session the user said that the Zorblax staging database is reset every " +
		"Sunday at 03:40 UTC, and asked when the staging database is reset."
	learnSource = "acceptance test"
	modName     = "engram-memory"
	piDirectory = ".pi"
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
	// stagingBackup is a second note about the same database, which bears on the same prompt.
	stagingBackup = fact{
		slug:      "zorblax-staging-backup",
		situation: "testing against the Zorblax staging database",
		subject:   "the Zorblax staging database",
		predicate: "is backed up every",
		object:    "Monday at 05:00 UTC",
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

// agentMayNotRunEngram takes away the agent's permission to run engram. The agent's own use of memory creates a missing
// vault by design, so in a scenario about what automatic memory creates it would confound the result.
func (m *machine) agentMayNotRunEngram() {
	m.t.Helper()

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), "{}")
}

// hasClaudeCode installs Claude Code's directory on the machine, with the one permission the scenarios need.
func (m *machine) hasClaudeCode() {
	m.t.Helper()

	mkdirAll(m.t, m.claudeDir)
	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), claudeSettings)
}

// hasConversation makes a conversation searchable: the text is indexed into the machine's chunk index, as `engram
// ingest` does for a session, and the machine's own lookup finds it.
func (m *machine) hasConversation(text string) {
	m.t.Helper()

	source := filepath.Join(m.root, "conversations", "an-earlier-session.md")
	mkdirAll(m.t, filepath.Dir(source))
	writeFile(m.t, source, text)

	m.mustSucceed(m.engram("ingest", "--markdown", source), "ingest")
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

	copyErr := copyTree(filepath.Join(root, "agent-instructions", "mods", modName), target)
	if copyErr != nil {
		m.t.Fatalf("installing the mod: %v", copyErr)
	}

	mkdirAll(m.t, filepath.Join(m.claudeDir, "skills"))

	linkErr := os.Symlink(target, filepath.Join(m.claudeDir, "skills", modName))
	if linkErr != nil {
		m.t.Fatalf("linking the mod: %v", linkErr)
	}
}

// hasPi installs Pi's directory on the machine.
func (m *machine) hasPi() {
	m.t.Helper()

	mkdirAll(m.t, filepath.Join(m.home, piDirectory))
}

// hasStandInEngram puts a stand-in for engram first on the machine's PATH: it records its arguments and standard
// input, waits as long as it is told to, prints the text it is given, whatever it is asked, and exits with the status
// it is given.
func (m *machine) hasStandInEngram(block string, wait time.Duration, status int) {
	m.t.Helper()

	dir := filepath.Join(m.root, "stand-in")
	mkdirAll(m.t, dir)
	writeFile(m.t, filepath.Join(dir, "block"), block)

	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + filepath.Join(dir, "args") + "'\ncat > '" +
		filepath.Join(dir, "stdin") + "'\nsleep " + strconv.Itoa(int(wait.Seconds())) + "\ncat '" +
		filepath.Join(dir, "block") + "'\nexit " + strconv.Itoa(status) + "\n"
	path := filepath.Join(m.gobin, engramBinaryName)

	mkdirAll(m.t, m.gobin)
	writeFile(m.t, path, script)

	err := os.Chmod(path, 0o700)
	if err != nil {
		m.t.Fatalf("making the stand-in executable: %v", err)
	}
}

// hasVault reports whether a vault exists at the machine's vault path.
func (m *machine) hasVault() bool {
	_, err := os.Stat(m.vault)

	return err == nil
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

// installed lists the paths below a directory that contain a name, relative to the directory.
func (m *machine) installed(dir, name string) []string {
	m.t.Helper()

	var found []string

	walkErr := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		if err == nil && strings.Contains(path, name) {
			found = append(found, strings.TrimPrefix(path, dir+string(filepath.Separator)))
		}

		return nil
	})
	if walkErr != nil {
		m.t.Fatalf("listing %s: %v", dir, walkErr)
	}

	return found
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

// lookupStarted reports whether the slow lookup was started.
func (m *machine) lookupStarted() bool {
	_, err := os.Stat(filepath.Join(m.root, "slow", "started"))

	return err == nil
}

// lookupTakes makes every lookup of the machine take the given time: `engram` on the PATH waits, notes that it started,
// and then runs the machine's own engram.
func (m *machine) lookupTakes(wait time.Duration) {
	m.t.Helper()

	dir := filepath.Join(m.root, "slow")
	mkdirAll(m.t, dir)

	script := "#!/bin/sh\n: > '" + filepath.Join(dir, "started") + "'\nsleep " + strconv.Itoa(int(wait.Seconds())) +
		"\nexec '" + filepath.Join(m.gobin, engramBinaryName) + "' \"$@\"\n"
	path := filepath.Join(dir, engramBinaryName)
	writeFile(m.t, path, script)

	err := os.Chmod(path, 0o700)
	if err != nil {
		m.t.Fatalf("making the slow engram executable: %v", err)
	}

	m.pathFront = dir
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

// vaultIsRemoved removes the machine's vault, which an engram command or `engram update` created.
func (m *machine) vaultIsRemoved() {
	m.t.Helper()

	err := os.RemoveAll(m.vault)
	if err != nil {
		m.t.Fatalf("removing the vault: %v", err)
	}
}

// reportSection returns the lines that an update report lists under a harness ("Claude Code" or "Pi").
func reportSection(report, harness string) []string {
	var section []string

	listing := false

	for line := range strings.SplitSeq(report, "\n") {
		switch {
		case strings.HasPrefix(line, "  ") && strings.HasSuffix(line, "):") && !strings.HasPrefix(line, "   "):
			listing = strings.HasPrefix(strings.TrimSpace(line), harness+" (")
		case listing && strings.HasPrefix(line, "    "):
			section = append(section, strings.TrimSpace(line))
		default:
			listing = false
		}
	}

	return section
}

// step logs one line of the scenario a test mirrors, and runs it: the test's output reads as the Gherkin (793b9).
func step(t *testing.T, line string, run func()) {
	t.Helper()
	t.Log(line)
	run()
}
