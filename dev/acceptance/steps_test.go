//go:build network

package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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
	// captureBound is how long after a turn ends the vault must hold the notes of its lessons (793a2c).
	captureBound = 2 * time.Minute
	// capturePoll is how often the vault is looked at while a note is awaited.
	capturePoll = 250 * time.Millisecond
	// claudeSettings lets the agent run engram, which the scenarios where it looks a note up itself need.
	claudeSettings = `{"permissions":{"allow":["Bash(engram:*)"]}}`
	// earlierConversation is a conversation that answers askStagingReset.
	earlierConversation = "In an earlier session the user said that the Zorblax staging database is reset every " +
		"Sunday at 03:40 UTC, and asked when the staging database is reset."
	learnSource = "acceptance test"
	modName     = "engram-memory"
	// ownTurnSettle is how long the test of a turn of the session's own waits for a record of that turn that would
	// come: three times the longest capture of the live runs (793b85).
	ownTurnSettle = 20 * time.Second
	piDirectory   = ".pi"
	// settleTime is how long a test of an absence waits, after the records of a write exist, for a second write to
	// show: the writes of one turn end within a second of each other.
	settleTime = 5 * time.Second
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

// agentMayRun gives the agent the permission to run the commands named, and not to run engram: a note that appears in
// the vault is then not the agent's.
func (m *machine) agentMayRun(commands ...string) {
	m.t.Helper()

	allowed := make([]string, 0, len(commands))
	for _, command := range commands {
		allowed = append(allowed, "Bash("+command+":*)")
	}

	settings, err := json.Marshal(map[string]any{"permissions": map[string]any{"allow": allowed}})
	if err != nil {
		m.t.Fatalf("encoding the agent's permissions: %v", err)
	}

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), string(settings))
}

// awaitAutomaticNote waits for the note of a lesson, a note that holds every word given, logs it, fails the test
// unless the note says it was written automatically, and returns the file name of the note.
func (m *machine) awaitAutomaticNote(turnEnds time.Time, what string, words ...string) string {
	m.t.Helper()

	note, took := m.awaitNote(turnEnds, what, func(note vaultNote) bool { return note.states(words...) })
	m.t.Logf("the note %s was written %s after the turn ended:\n%s", note.name, took, note.text)

	if !strings.Contains(note.source(), "written automatically") {
		m.t.Fatalf("the source of %s does not say that it was written automatically: %q", note.name, note.source())
	}

	return note.name
}

// awaitChunk waits until engram query, asked as the user asks it, returns a chunk of the conversation that holds every
// word given, and returns how long after the turn ended it did. It fails the test when no such chunk comes within the
// bound of the turn's end.
func (m *machine) awaitChunk(turnEnds time.Time, phrase string, words ...string) time.Duration {
	m.t.Helper()

	deadline := turnEnds.Add(captureBound)

	for {
		out := m.engram("query", "--text", phrase, "--phrase", phrase)
		m.mustSucceed(out, "engram query")

		if chunkHolding(out.stdout, words...) {
			return max(time.Since(turnEnds), 0)
		}

		if time.Now().After(deadline) {
			m.t.Fatalf("engram query returned no chunk holding %v within %s of the turn's end\n%s",
				words, captureBound, out.stdout)
		}

		time.Sleep(capturePoll)
	}
}

// awaitNote waits until a note of the vault satisfies match, or the bound passes, and returns the note with how long
// after the turn ended it was written. It fails the test when no note satisfies match in time.
func (m *machine) awaitNote(turnEnds time.Time, what string, match func(vaultNote) bool) (vaultNote, time.Duration) {
	m.t.Helper()

	deadline := turnEnds.Add(captureBound)

	for {
		for _, note := range m.notes() {
			if match(note) {
				return note, max(note.written.Sub(turnEnds), 0)
			}
		}

		if time.Now().After(deadline) {
			m.t.Fatalf("no note %s was in the vault within %s of the turn's end\nnotes: %v\n%s\n%s",
				what, captureBound, m.noteNames(), m.recordsText(), m.hookLog())
		}

		time.Sleep(capturePoll)
	}
}

// awaitRecord waits until automatic memory has recorded what became of a turn of a session, and returns the kind of the
// first record. A test of an absence waits for it, so that the absence is the result of the judgement and not of a
// judgement that did not finish.
func (m *machine) awaitRecord(session string, turnEnds time.Time) string {
	m.t.Helper()

	return m.awaitRecords(session, turnEnds, 1)[0]
}

// awaitRecordOfKind waits until automatic memory has made a record of the kind for a turn of a session. A write has
// ended when its records exist: they are made last.
func (m *machine) awaitRecordOfKind(session string, turnEnds time.Time, kind string) {
	m.t.Helper()

	deadline := turnEnds.Add(captureBound)

	for !slices.Contains(m.recordKinds(session), kind) {
		if time.Now().After(deadline) {
			m.t.Fatalf("automatic memory made no record of the kind %s within %s of the last turn's end: %v\n%s", kind,
				captureBound, m.recordKinds(session), m.hookLog())
		}

		time.Sleep(capturePoll)
	}
}

// awaitRecords waits until automatic memory has made count records of the turns of a session, and returns the kinds of
// the records, oldest first. A write has ended when its records exist: they are made last.
func (m *machine) awaitRecords(session string, turnEnds time.Time, count int) []string {
	m.t.Helper()

	deadline := turnEnds.Add(captureBound)

	for {
		if kinds := m.recordKinds(session); len(kinds) >= count {
			return kinds
		}

		if time.Now().After(deadline) {
			m.t.Fatalf("automatic memory made fewer than %d records of the turns within %s of the last one's end\n%s",
				count, captureBound, m.hookLog())
		}

		time.Sleep(capturePoll)
	}
}

// awaitSettled waits until automatic memory has made count records of the turns of a session and then for the settling
// time, so that a second note written for the same turn would be in the vault when the test looks.
func (m *machine) awaitSettled(session string, turnEnds time.Time, count int) {
	m.t.Helper()

	m.awaitRecords(session, turnEnds, count)
	time.Sleep(settleTime)
}

// chunksCannotBeWritten takes the right to create a file in the machine's chunk index directory away; reading it still
// works.
func (m *machine) chunksCannotBeWritten() {
	m.t.Helper()

	mkdirAll(m.t, m.chunks)

	err := os.Chmod(m.chunks, 0o500)
	if err != nil {
		m.t.Fatalf("making the chunk index read-only: %v", err)
	}

	m.t.Cleanup(func() { _ = os.Chmod(m.chunks, dirPerm) })
}

// deliveryIsQuick takes away the file that keeps the delivery of deliveryTakes slow.
func (m *machine) deliveryIsQuick() {
	m.t.Helper()

	err := os.Remove(filepath.Join(m.root, "slow-delivery", "slow"))
	if err != nil {
		m.t.Fatalf("making the delivery quick: %v", err)
	}
}

// deliveryStarted reports whether the slow delivery of deliveryTakes was started.
func (m *machine) deliveryStarted() bool {
	_, err := os.Stat(filepath.Join(m.root, "slow-delivery", "started"))

	return err == nil
}

// deliveryTakes makes the delivery take the given time for as long as the file slow is in the wrapper's directory:
// engram on the PATH notes that the delivery started and waits, and runs the machine's own engram for anything else.
// The wait replaces the process, so that ending the delivery ends all of it.
func (m *machine) deliveryTakes(wait time.Duration) {
	m.t.Helper()

	dir := filepath.Join(m.root, "slow-delivery")
	mkdirAll(m.t, dir)
	writeFile(m.t, filepath.Join(dir, "slow"), "")

	m.wrapsEngram(dir, "#!/bin/sh\ncase \"$1 $2\" in\n\"auto deliver\")\nif [ -e '"+filepath.Join(dir, "slow")+
		"' ]; then\n: > '"+filepath.Join(dir, "started")+"'\nexec sleep "+strconv.Itoa(int(wait.Seconds()))+
		"\nfi\n;;\nesac\nexec '"+filepath.Join(m.gobin, engramBinaryName)+"' \"$@\"\n")
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

// hasRunUpdate is a machine with Claude Code on which the user has run engram update, and on which the agent may run
// the commands named and not engram.
func (m *machine) hasRunUpdate(agentMayRun ...string) {
	m.t.Helper()

	m.hasEngram()
	m.hasClaudeCode()
	m.agentMayRun(agentMayRun...)
	m.mustSucceed(m.update(), "engram update")
}

// hasStandInEngram puts a stand-in for engram first on the machine's PATH: for the delivery it records its arguments
// and standard input, waits as long as it is told to, prints the text it is given and exits with the status it is
// given; for the writing it notes that it was run; for any other command it reads its input and exits with 0, so that
// what the mod reports or does after a turn replaces nothing the delivery recorded.
func (m *machine) hasStandInEngram(block string, wait time.Duration, status int) {
	m.t.Helper()

	dir := filepath.Join(m.root, "stand-in")
	mkdirAll(m.t, dir)
	writeFile(m.t, filepath.Join(dir, "block"), block)

	script := "#!/bin/sh\ncase \"$1 $2\" in\n\"auto deliver\")\nprintf '%s\\n' \"$*\" > '" + filepath.Join(dir, "args") +
		"'\ncat > '" + filepath.Join(dir, "stdin") + "'\nsleep " + strconv.Itoa(int(wait.Seconds())) + "\ncat '" +
		filepath.Join(dir, "block") + "'\nexit " + strconv.Itoa(status) + "\n;;\n\"auto write\")\ncat > '" +
		filepath.Join(dir, "write") + "'\nexit 0\n;;\nesac\ncat > /dev/null\nexit 0\n"
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

// headlessIsNotCaptured leaves the switch for the capture of headless prompts out of the machine's environment, as a
// user's own is until the user asks (793d4m).
func (m *machine) headlessIsNotCaptured() {
	m.t.Helper()

	m.headlessLeftOut = true
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

// jobStarted reports whether a job of automatic memory after a turn (judge, write or sweep) was started: the slow
// engram of jobsTake notes each one.
func (m *machine) jobStarted(job string) bool {
	_, err := os.Stat(filepath.Join(m.root, "jobs", "started-"+job))

	return err == nil
}

// jobsTake makes the work after a turn take the given time: engram on the PATH notes that the judgement, the writing or
// the sweep started and then waits, and runs the machine's own engram for anything else. The wait replaces the process,
// so that ending the job ends all of it.
func (m *machine) jobsTake(wait time.Duration) {
	m.t.Helper()

	dir := filepath.Join(m.root, "jobs")

	m.wrapsEngram(dir, "#!/bin/sh\ncase \"$1 $2\" in\n\"auto judge\"|\"auto write\"|\"auto sweep\")\n: > '"+dir+
		"/started-'\"$2\"\nexec sleep "+strconv.Itoa(int(wait.Seconds()))+"\n;;\nesac\nexec '"+
		filepath.Join(m.gobin, engramBinaryName)+"' \"$@\"\n")
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

	m.wrapsEngram(dir, "#!/bin/sh\n: > '"+filepath.Join(dir, "started")+"'\nsleep "+strconv.Itoa(int(wait.Seconds()))+
		"\nexec '"+filepath.Join(m.gobin, engramBinaryName)+"' \"$@\"\n")
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

// noteNames lists the file names of the notes of the machine's vault.
func (m *machine) noteNames() []string {
	notes := m.notes()
	names := make([]string, 0, len(notes))

	for _, note := range notes {
		names = append(names, note.name)
	}

	return names
}

// noteText returns the text of the note of the machine's vault with a file name.
func (m *machine) noteText(name string) string {
	m.t.Helper()

	data, err := os.ReadFile(filepath.Join(m.vault, name))
	if err != nil {
		m.t.Fatalf("reading the note %s: %v", name, err)
	}

	return string(data)
}

// notes reads the notes of the machine's vault: the files named for a Luhmann ID, which the vault's own README is not.
func (m *machine) notes() []vaultNote {
	matches, err := filepath.Glob(filepath.Join(m.vault, "[0-9]*.md"))
	if err != nil {
		m.t.Fatalf("listing the notes of %s: %v", m.vault, err)
	}

	notes := make([]vaultNote, 0, len(matches))

	for _, name := range matches {
		data, readErr := os.ReadFile(name)
		info, statErr := os.Stat(name)

		if readErr == nil && statErr == nil {
			notes = append(notes, vaultNote{name: filepath.Base(name), text: string(data), written: info.ModTime()})
		}
	}

	return notes
}

// recordKinds returns the kinds of the records that automatic memory made for a session, oldest first: the records are
// the JSON files of the session's directory in the engram data directory, named for the time they were made.
func (m *machine) recordKinds(session string) []string {
	files, err := filepath.Glob(filepath.Join(m.home, ".local", "share", "engram", "auto", session, "*.json"))
	if err != nil {
		m.t.Fatalf("listing the records of %s: %v", session, err)
	}

	slices.Sort(files)

	kinds := make([]string, 0, len(files))

	for _, name := range files {
		data, readErr := os.ReadFile(name)

		var record struct {
			Kind string `json:"kind"`
		}

		if readErr == nil && json.Unmarshal(data, &record) == nil {
			kinds = append(kinds, record.Kind)
		}
	}

	return kinds
}

// recordsText returns the records that automatic memory made on the machine, a file after another, for the message of a
// test that waited for a note or a record and found none: what became of the turns is the cause.
func (m *machine) recordsText() string {
	files, _ := filepath.Glob(filepath.Join(m.home, ".local", "share", "engram", "auto", "*", "*.json"))

	slices.Sort(files)

	lines := make([]string, 0, len(files)+1)
	lines = append(lines, "records:")

	for _, name := range files {
		data, err := os.ReadFile(name)
		if err == nil {
			lines = append(lines, filepath.Base(name)+": "+strings.TrimSpace(string(data)))
		}
	}

	return strings.Join(lines, "\n")
}

// sessionsRecorded returns the IDs of the sessions that Claude Code holds a record of on the machine, sorted: the files
// below the directory of the records are named for a session.
func (m *machine) sessionsRecorded() []string {
	m.t.Helper()

	var ids []string

	walkErr := filepath.WalkDir(m.transcripts, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(path, ".jsonl") {
			ids = append(ids, strings.TrimSuffix(filepath.Base(path), ".jsonl"))
		}

		return nil
	})
	if walkErr != nil {
		m.t.Fatalf("listing the records of sessions in %s: %v", m.transcripts, walkErr)
	}

	slices.Sort(ids)

	return ids
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

// standInReported reports whether the stand-in engram was run as the writing: the mod reported a failure to it.
func (m *machine) standInReported() bool {
	_, err := os.Stat(filepath.Join(m.root, "stand-in", "write"))

	return err == nil
}

// vaultCannotBeWritten takes the right to create a file in the machine's vault away; reading it still works.
func (m *machine) vaultCannotBeWritten() {
	m.t.Helper()

	err := os.Chmod(m.vault, 0o500)
	if err != nil {
		m.t.Fatalf("making the vault read-only: %v", err)
	}

	m.t.Cleanup(func() { _ = os.Chmod(m.vault, dirPerm) })
}

// vaultIsRemoved removes the machine's vault, which an engram command or `engram update` created.
func (m *machine) vaultIsRemoved() {
	m.t.Helper()

	err := os.RemoveAll(m.vault)
	if err != nil {
		m.t.Fatalf("removing the vault: %v", err)
	}
}

// wrapsEngram puts a script named engram in a directory that comes first on the machine's PATH, so that it stands
// between the mod and the machine's own engram.
func (m *machine) wrapsEngram(dir, script string) {
	m.t.Helper()

	mkdirAll(m.t, dir)

	path := filepath.Join(dir, engramBinaryName)
	writeFile(m.t, path, script)

	err := os.Chmod(path, 0o700)
	if err != nil {
		m.t.Fatalf("making the engram in %s executable: %v", dir, err)
	}

	m.pathFront = dir
}

// writeTakes makes the writing of the lessons take the given time: engram on the PATH notes that the writing started
// and waits, and then runs the machine's own engram, which writes what it was given.
func (m *machine) writeTakes(wait time.Duration) {
	m.t.Helper()

	dir := filepath.Join(m.root, "slow-write")

	m.wrapsEngram(dir, "#!/bin/sh\ncase \"$1 $2\" in\n\"auto write\")\n: > '"+dir+"/started'\nsleep "+
		strconv.Itoa(int(wait.Seconds()))+"\n;;\nesac\nexec '"+filepath.Join(m.gobin, engramBinaryName)+
		"' \"$@\"\n")
}

// vaultNote is a note of the vault as its file holds it.
type vaultNote struct {
	name    string
	text    string
	written time.Time // when the file was last written
}

// source returns the value of the line of the note's front matter that says where the note came from.
func (n vaultNote) source() string {
	for line := range strings.SplitSeq(n.text, "\n") {
		if value, found := strings.CutPrefix(line, "source: "); found {
			return value
		}
	}

	return ""
}

// states reports whether the note's text holds every word given, whatever their case.
func (n vaultNote) states(words ...string) bool {
	lower := strings.ToLower(n.text)

	for _, word := range words {
		if !strings.Contains(lower, strings.ToLower(word)) {
			return false
		}
	}

	return true
}

// chunkHolding reports whether a payload of engram query has an item of the kind chunk that holds every word, whatever
// their case: the payload lists one item at each line that begins "  - path: ".
func chunkHolding(payload string, words ...string) bool {
	for item := range strings.SplitSeq(payload, "\n  - path: ") {
		lower := strings.ToLower(item)

		if strings.Contains(item, "kind: chunk") && !slices.ContainsFunc(words, func(word string) bool {
			return !strings.Contains(lower, strings.ToLower(word))
		}) {
			return true
		}
	}

	return false
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
