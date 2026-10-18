package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/auto"
)

// AutoDeliverArgs holds parsed flags for `engram auto deliver`.
type AutoDeliverArgs struct {
	VaultPath   string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
	RecordsDir  string `targ:"flag,name=records,desc=directory of the records of what automatic memory did (default $XDG_DATA_HOME/engram/auto)"`                              //nolint:lll // single unbreakable struct-tag string
}

// AutoDeliverDeps holds injected dependencies for RunAutoDeliver.
type AutoDeliverDeps struct {
	// Stdin carries the request: one JSON object with the prompt's text.
	Stdin io.Reader
	// Query runs the lookup of notes, the one `engram query` runs, and prints its payload.
	Query QueryRun
	// ReadDir and ReadFile read the session-log directory, where Claude Code keeps the record of each session.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
	// WriteRecord writes one file of the records of a session, and makes its directory: the mark of a record whose line
	// was shown.
	WriteRecord func(path string, data []byte) error
}

// AutoJudgeArgs holds parsed flags for `engram auto judge`: those of the delivery, because the judgement looks up
// notes and reads the record of a session as the delivery does, and the switch for the turns of headless prompts.
type AutoJudgeArgs struct {
	VaultPath   string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"`                            //nolint:lll // single unbreakable struct-tag string
	CaptureSDK  bool   `targ:"flag,name=capture-sdk,env=ENGRAM_AUTO_CAPTURE_SDK,desc=judge the turns of headless prompts (origin sdk) too; off by default so a script run with claude -p writes no note"` //nolint:lll // single unbreakable struct-tag string
	RecordsDir  string `targ:"flag,name=records,desc=directory of the records of what automatic memory did (default $XDG_DATA_HOME/engram/auto)"`                                                         //nolint:lll // single unbreakable struct-tag string
}

// AutoJudgeDeps holds injected dependencies for RunAutoJudge: those of the delivery, because the judgement looks up
// notes and reads the record of a session as the delivery does, and those of the writing of a record of a turn it
// leaves out.
type AutoJudgeDeps struct {
	// Stdin carries the turn that ended: one JSON object with the session's ID, the turn's ID, the turn's final text and
	// the origin of its prompt.
	Stdin io.Reader
	// Query runs the lookup of notes, the one `engram query` runs, and prints its payload.
	Query QueryRun
	// ReadDir and ReadFile read the session-log directory, where Claude Code keeps the record of each session.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
	// Now is the time a record is made at.
	Now func() time.Time
	// WriteRecord writes one record, a file of its own, and makes its directory.
	WriteRecord func(path string, data []byte) error
}

// AutoSweepArgs holds parsed flags for `engram auto sweep`: where the records of sessions and the chunk index are, and
// the switch for the turns of headless prompts that the judgement has too.
type AutoSweepArgs struct {
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"`                             //nolint:lll // single unbreakable struct-tag string
	ChunksDir   string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`                                                                                           //nolint:lll // single unbreakable struct-tag string
	CaptureSDK  bool   `targ:"flag,name=capture-sdk,env=ENGRAM_AUTO_CAPTURE_SDK,desc=sweep the turns of headless prompts (origin sdk) too; off by default so a script run with claude -p indexes nothing"` //nolint:lll // single unbreakable struct-tag string
}

// AutoSweepDeps holds injected dependencies for RunAutoSweep.
type AutoSweepDeps struct {
	// Stdin carries the turn that ended: one JSON object with the session's ID and the origin of its prompt.
	Stdin io.Reader
	// Ingest indexes sources in the chunk index, as `engram ingest` does.
	Ingest IngestRun
	// ReadDir lists the session-log directory, where Claude Code keeps the record of each session, and Exists reports
	// whether a file is in it.
	ReadDir func(path string) ([]fs.DirEntry, error)
	Exists  func(path string) bool
}

// AutoWriteArgs holds parsed flags for `engram auto write`.
type AutoWriteArgs struct {
	VaultPath  string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	VaultName  string `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on the note's vault: field (default \"personal\")"` //nolint:lll // single unbreakable struct-tag string
	RecordsDir string `targ:"flag,name=records,desc=directory of the records of what automatic memory did (default $XDG_DATA_HOME/engram/auto)"`    //nolint:lll // single unbreakable struct-tag string
}

// AutoWriteDeps holds injected dependencies for RunAutoWrite.
type AutoWriteDeps struct {
	// Stdin carries the reply of the model: one JSON object with the session's ID and the reply.
	Stdin io.Reader
	// Learn writes a note through the learn path.
	Learn LearnRun
	// Now is the time a record is made at.
	Now func() time.Time
	// WriteRecord writes one record, a file of its own, and makes its directory.
	WriteRecord func(path string, data []byte) error
	// NoteExists reports whether a file is in the vault.
	NoteExists func(path string) bool
}

// IngestRun indexes sources in the chunk index, as `engram ingest` does, and prints what it indexed.
type IngestRun func(ctx context.Context, args IngestArgs, stdout io.Writer) error

// LearnRun writes a note through the learn path, as `engram learn` does, and returns the path it prints for the note.
type LearnRun func(ctx context.Context, args LearnArgs) (string, error)

// QueryRun runs the lookup of `engram query` for the arguments it is given and prints its payload.
type QueryRun func(ctx context.Context, args QueryArgs, stdout io.Writer) error

// RunAutoDeliver reads a prompt from standard input, prints the block of the notes that bear on it on standard output
// (793d4a), and prints on standard error the lines that show the user the notes it delivered and what automatic memory
// did that the user was not shown yet, marking the records whose lines it printed (793d4p).
func RunAutoDeliver(
	ctx context.Context, args AutoDeliverArgs, deps AutoDeliverDeps, stdout, stderr io.Writer,
) error {
	var request auto.Request

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&request)
	if decodeErr != nil {
		return fmt.Errorf("auto deliver: reading the request: %w", decodeErr)
	}

	if !auto.UserSent(request.Origin) {
		return nil
	}

	// the record of the session is read once: it says what the agent holds (793d4d)
	record := recordOf(deps.ReadDir, deps.ReadFile, args.SessionsDir, request.Session)

	delivery, err := auto.Deliver(ctx, request,
		func(ctx context.Context, text string) ([]auto.Note, error) {
			return lookUpNotes(ctx, args.VaultPath, text, deps.Query)
		},
		func(string) string {
			return auto.HeldText(record)
		},
		func(session string) []auto.Line {
			return auto.ActivityLines(recordsOf(deps, args.RecordsDir, session))
		})
	if err != nil {
		return fmt.Errorf("auto deliver: %w", err)
	}

	_, writeErr := io.WriteString(stdout, delivery.Block)
	if writeErr != nil {
		return fmt.Errorf("auto deliver: writing the block: %w", writeErr)
	}

	for _, line := range delivery.Shown {
		_, shownErr := fmt.Fprintln(stderr, line)
		if shownErr != nil {
			return fmt.Errorf("auto deliver: writing a line for the user: %w", shownErr)
		}
	}

	markShown(deps.WriteRecord, args.RecordsDir, request.Session, delivery.Told)

	return nil
}

// RunAutoJudge reads a turn that ended from standard input and prints the request for the model that judges it, or
// nothing when the session's record holds no prompt. It fails, printing nothing, when no vault exists at the path or
// no record of the session is found, so that the model is not asked for a turn that cannot be judged (793d4g).
func RunAutoJudge(ctx context.Context, args AutoJudgeArgs, deps AutoJudgeDeps, stdout io.Writer) error {
	var turn auto.TurnEnd

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&turn)
	if decodeErr != nil {
		return fmt.Errorf("auto judge: reading the turn: %w", decodeErr)
	}

	if !auto.Judged(turn.Origin, args.CaptureSDK) {
		// a turn of a prompt the user sends that is not judged is a headless one that the user did not ask to capture
		if auto.UserSent(turn.Origin) {
			recordLeftOut(deps, args.RecordsDir, turn)
		}

		return nil
	}

	_, vaultErr := deps.ReadDir(args.VaultPath)
	if vaultErr != nil {
		return fmt.Errorf("auto judge: no vault at %s: %w", args.VaultPath, vaultErr)
	}

	record := recordOf(deps.ReadDir, deps.ReadFile, args.SessionsDir, turn.Session)
	if record == nil {
		return fmt.Errorf("auto judge: %w %s under %s", errNoRecord, turn.Session, args.SessionsDir)
	}

	request, found, judgeErr := auto.Judge(ctx, record, turn,
		func(ctx context.Context, text string) ([]auto.Note, error) {
			return lookUpNotes(ctx, args.VaultPath, text, deps.Query)
		})
	if judgeErr != nil {
		return fmt.Errorf("auto judge: %w", judgeErr)
	}

	if !found {
		return nil
	}

	encoded, marshalErr := json.Marshal(request)
	if marshalErr != nil {
		return fmt.Errorf("auto judge: encoding the request: %w", marshalErr)
	}

	_, writeErr := fmt.Fprintln(stdout, string(encoded))
	if writeErr != nil {
		return fmt.Errorf("auto judge: writing the request: %w", writeErr)
	}

	return nil
}

// RunAutoSweep reads a turn that ended from standard input and indexes the record of its session in the chunk index, as
// the explicit source of an ingest, so that `engram query` finds the conversation (793d4n). It prints nothing and
// indexes nothing for a prompt the judgement does not act for, and fails, naming the cause, when the input is not a
// JSON object, the session has no record or the index fails.
func RunAutoSweep(ctx context.Context, args AutoSweepArgs, deps AutoSweepDeps) error {
	var turn auto.TurnEnd

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&turn)
	if decodeErr != nil {
		return fmt.Errorf("auto sweep: reading the turn: %w", decodeErr)
	}

	if !auto.Judged(turn.Origin, args.CaptureSDK) {
		return nil
	}

	record := recordPathOf(deps.ReadDir, deps.Exists, args.SessionsDir, turn.Session)
	if record == "" {
		return fmt.Errorf("auto sweep: %w %s under %s", errNoRecord, turn.Session, args.SessionsDir)
	}

	ingestErr := deps.Ingest(ctx, IngestArgs{Transcripts: []string{record}, ChunksDir: args.ChunksDir}, io.Discard)
	if ingestErr != nil {
		return fmt.Errorf("auto sweep: indexing %s: %w", record, ingestErr)
	}

	return nil
}

// RunAutoWrite reads the reply of the model to the judgement of a turn from standard input and writes each lesson it
// holds as a note, marked as written automatically, printing the path of each note (793d4h).
func RunAutoWrite(ctx context.Context, args AutoWriteArgs, deps AutoWriteDeps, stdout io.Writer) error {
	var reply auto.Reply

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&reply)
	if decodeErr != nil {
		return fmt.Errorf("auto write: reading the reply: %w", decodeErr)
	}

	results := outcomes{base: auto.Record{At: deps.Now(), Session: reply.Session, Turn: reply.Turn}}
	failures := writeLessons(ctx, args, deps, reply, &results, stdout)

	if writeRecords(deps.WriteRecord, args.RecordsDir, results.records) {
		return nil
	}

	return errors.Join(failures...)
}

// unexported constants.
const (
	// the permissions of a record and of the directory of the records of a session
	recordDirPerm fs.FileMode = 0o750
	// recordFileExt ends the name of the file of a record.
	recordFileExt              = ".json"
	recordFilePerm fs.FileMode = 0o600
	// recordTimeFormat writes the time a record was made at in the name of its file: it sorts as the time does.
	recordTimeFormat = "20060102T150405.000000000Z"
	// toldFileExt ends the name of the mark of a record whose line was shown: the name of the record's file, then this.
	toldFileExt = ".told"
)

// unexported variables.
var (
	// errNoRecord begins the failure of a judgement or a sweep that finds no record of its session.
	errNoRecord = errors.New("no record of session")
)

// outcomes is what became of a reply so far, as the records that tell it: base holds what they share.
type outcomes struct {
	base    auto.Record
	records []auto.Record
}

// add adds the record of one outcome.
func (o *outcomes) add(kind, note, slug, reason string) {
	record := o.base
	record.Kind, record.Note, record.Slug, record.Reason = kind, note, slug, reason

	o.records = append(o.records, record)
}

// coveringNote returns the file name of the note of the vault that a lesson names as covering it, or nothing: only the
// name of a file that is in the vault covers, with the suffix .md or without it (793d4h).
func coveringNote(covered, vault string, exists func(path string) bool) string {
	name := strings.TrimSuffix(covered, ".md")
	if !isPlainName(name) {
		return ""
	}

	name += ".md"
	if exists(filepath.Join(vault, name)) {
		return name
	}

	return ""
}

// fileExistsFrom returns a check that a file is there, through the injected file system.
func fileExistsFrom(d Deps) func(path string) bool {
	return func(path string) bool {
		_, err := d.FS.Stat(path)

		return err == nil
	}
}

// findRecord reads the record of a session, which Claude Code keeps as <ID>.jsonl in a directory of the session-log
// directory for the project, or in the session-log directory itself.
func findRecord(
	readDir func(path string) ([]fs.DirEntry, error), readFile func(path string) ([]byte, error), dir, session string,
) []byte {
	for _, candidate := range recordCandidates(readDir, dir, session) {
		record, err := readFile(candidate)
		if err == nil {
			return record
		}
	}

	return nil
}

// isPlainName reports whether an ID is a plain file name: it names no directory, so it cannot name a file outside the
// directory it is joined to.
func isPlainName(name string) bool {
	return name != "" && name == filepath.Base(name) && name != "." && name != ".."
}

// learnArgsOfLesson is the write of a lesson through the learn path: a note of its kind at the top of the order, whose
// source says it was written automatically.
func learnArgsOfLesson(lesson auto.Lesson, args AutoWriteArgs, session string) LearnArgs {
	return LearnArgs{
		Type:      lesson.Kind,
		Slug:      lesson.Slug,
		Vault:     args.VaultPath,
		VaultName: args.VaultName,
		Position:  positionTop,
		Source:    auto.SourceFor(session),
		Situation: lesson.Situation,
		Behavior:  lesson.Behavior,
		Impact:    lesson.Impact,
		Action:    lesson.Action,
		Subject:   lesson.Subject,
		Predicate: lesson.Predicate,
		Object:    lesson.Object,
	}
}

// lookUpNotes runs the lookup of `engram query` for a text, as its message and its only phrase, and reads the notes
// out of the payload it prints.
func lookUpNotes(ctx context.Context, vault, text string, query QueryRun) ([]auto.Note, error) {
	var payload bytes.Buffer

	args := QueryArgs{Phrases: []string{text}, Text: text, VaultPath: vault}

	queryErr := query(ctx, args, &payload)
	if queryErr != nil {
		return nil, fmt.Errorf("looking up %q: %w", text, queryErr)
	}

	return notesOf(payload.Bytes())
}

// markShown marks each record whose line was shown, by an empty file beside the record, so that its line is not shown
// again (793d4p). A mark that cannot be written is the end of it.
func markShown(write func(path string, data []byte) error, dir, session string, files []string) {
	for _, file := range files {
		_ = write(filepath.Join(dir, session, file+toldFileExt), nil)
	}
}

// newAutoDeliverDeps wires RunAutoDeliver and RunAutoJudge from the injected CLI capabilities - pure composition
// (#700).
func newAutoDeliverDeps(d Deps) AutoDeliverDeps {
	queryDeps := newQueryDeps(d)

	return AutoDeliverDeps{
		Stdin: d.Stdin,
		Query: func(ctx context.Context, args QueryArgs, stdout io.Writer) error {
			return RunQuery(ctx, args, queryDeps, stdout)
		},
		ReadDir:     d.FS.ReadDir,
		ReadFile:    d.FS.ReadFile,
		WriteRecord: recordWriterFrom(d),
	}
}

// newAutoJudgeDeps wires RunAutoJudge from the injected CLI capabilities - pure composition (#700).
func newAutoJudgeDeps(d Deps) AutoJudgeDeps {
	deliverDeps := newAutoDeliverDeps(d)

	return AutoJudgeDeps{
		Stdin:       deliverDeps.Stdin,
		Query:       deliverDeps.Query,
		ReadDir:     deliverDeps.ReadDir,
		ReadFile:    deliverDeps.ReadFile,
		Now:         d.Now,
		WriteRecord: recordWriterFrom(d),
	}
}

// newAutoSweepDeps wires RunAutoSweep from the injected CLI capabilities - pure composition (#700).
func newAutoSweepDeps(d Deps) AutoSweepDeps {
	ingestDeps := newIngestDeps(d)

	return AutoSweepDeps{
		Stdin: d.Stdin,
		Ingest: func(ctx context.Context, args IngestArgs, stdout io.Writer) error {
			return RunIngest(ctx, args, ingestDeps, stdout)
		},
		ReadDir: d.FS.ReadDir,
		Exists:  fileExistsFrom(d),
	}
}

// newAutoWriteDeps wires RunAutoWrite from the injected CLI capabilities — pure composition (#700).
func newAutoWriteDeps(d Deps) AutoWriteDeps {
	learnDeps := newLearnDeps(d)

	return AutoWriteDeps{
		Stdin: d.Stdin,
		Learn: func(ctx context.Context, args LearnArgs) (string, error) {
			var printed bytes.Buffer

			err := RunLearn(ctx, args, learnDeps, &printed)

			return strings.TrimSpace(printed.String()), err
		},
		Now:         d.Now,
		NoteExists:  fileExistsFrom(d),
		WriteRecord: recordWriterFrom(d),
	}
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

		situation, source := situationAndSourceOf([]byte(frontmatter))

		notes = append(notes, auto.Note{Path: item.Path, Situation: situation, Body: body, Source: source})
	}

	return notes, nil
}

// recordCandidates lists the files that may hold the record of a session: <ID>.jsonl in the session-log directory
// itself, and in each directory of it.
func recordCandidates(readDir func(path string) ([]fs.DirEntry, error), dir, session string) []string {
	name := session + ".jsonl"
	candidates := []string{filepath.Join(dir, name)}

	// a directory that cannot be listed has no project directories to search
	entries, _ := readDir(dir)

	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(dir, entry.Name(), name))
		}
	}

	return candidates
}

// recordLeftOut records that the judgement left a headless turn out, and why (793d4m). A record that cannot be written
// is the end of it, as it is for the writing (793d4k).
func recordLeftOut(deps AutoJudgeDeps, dir string, turn auto.TurnEnd) {
	results := outcomes{base: auto.Record{At: deps.Now(), Session: turn.Session, Turn: turn.Turn}}
	results.add(auto.RecordSkipped, "", "", auto.HeadlessSkipReason)

	_ = writeRecords(deps.WriteRecord, dir, results.records)
}

// recordOf returns the record of a session, or nothing. A session whose ID is not a plain file name has none: its ID
// would otherwise name a file outside the directory (793d4d). Nor does a session when there is no directory of records
// (793d4f): the file name alone would name a file of the working directory.
func recordOf(
	readDir func(path string) ([]fs.DirEntry, error), readFile func(path string) ([]byte, error), dir, session string,
) []byte {
	if dir == "" || !isPlainName(session) {
		return nil
	}

	return findRecord(readDir, readFile, dir, session)
}

// recordPathOf returns the path of the record of a session, or nothing, as recordOf finds it.
func recordPathOf(
	readDir func(path string) ([]fs.DirEntry, error), exists func(path string) bool, dir, session string,
) string {
	if dir == "" || !isPlainName(session) {
		return ""
	}

	for _, candidate := range recordCandidates(readDir, dir, session) {
		if exists(candidate) {
			return candidate
		}
	}

	return ""
}

// recordWriterFrom returns the writing of one record through the injected file system: a file of its own, in a
// directory that is made.
func recordWriterFrom(d Deps) func(path string, data []byte) error {
	return func(path string, data []byte) error {
		mkdirErr := d.FS.MkdirAll(filepath.Dir(path), recordDirPerm)
		if mkdirErr != nil {
			return fmt.Errorf("making the directory of the record %s: %w", path, mkdirErr)
		}

		writeErr := d.FS.WriteFileExcl(path, data, recordFilePerm)
		if writeErr != nil {
			return fmt.Errorf("writing the record %s: %w", path, writeErr)
		}

		return nil
	}
}

// recordsOf returns the records of what automatic memory did for a session, oldest first, that the user was not shown:
// those of the files of the session's directory in the directory of records, in the order of their names, which sort
// as the time they were made does (793d4k), less those with a mark beside them (793d4p). A directory that cannot be
// listed, a file that cannot be read or that is no record, and a record of another session leave nothing.
func recordsOf(deps AutoDeliverDeps, dir, session string) []auto.Record {
	if dir == "" || !isPlainName(session) {
		return nil
	}

	// a directory that cannot be listed has no records
	entries, _ := deps.ReadDir(filepath.Join(dir, session))
	shown := shownRecords(entries)
	records := make([]auto.Record, 0, len(entries))

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != recordFileExt || shown[entry.Name()] {
			continue
		}

		data, readErr := deps.ReadFile(filepath.Join(dir, session, entry.Name()))

		var record auto.Record

		if readErr == nil && json.Unmarshal(data, &record) == nil && record.Session == session {
			record.File = entry.Name()
			records = append(records, record)
		}
	}

	return records
}

// sessionsDirOf is where the records of sessions are read from when --sessions is not given (793d4f): the directory
// ENGRAM_TRANSCRIPT_DIR names when it is set, else the projects directory of CLAUDE_CONFIG_DIR when that is set, where
// Claude Code writes the records of a session started with it, else the home's .claude/projects.
func sessionsDirOf(getenv func(string) string, userHomeDir func() (string, error)) string {
	config := getenv("CLAUDE_CONFIG_DIR")
	if config != "" && getenv("ENGRAM_TRANSCRIPT_DIR") == "" {
		return filepath.Join(config, "projects")
	}

	return sessionDirFrom(getenv, userHomeDir)("")
}

// shownRecords returns the file names of the records that have a mark beside them (793d4p).
func shownRecords(entries []fs.DirEntry) map[string]bool {
	shown := make(map[string]bool, len(entries))

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == toldFileExt {
			shown[strings.TrimSuffix(entry.Name(), toldFileExt)] = true
		}
	}

	return shown
}

// situationAndSourceOf extracts the situation and the source from the front matter of a note, or nothing when the
// block does not parse.
func situationAndSourceOf(frontmatter []byte) (situation, source string) {
	var probe struct {
		Situation string `yaml:"situation"`
		Source    string `yaml:"source"`
	}

	if yaml.Unmarshal(frontmatter, &probe) != nil {
		return "", ""
	}

	return probe.Situation, probe.Source
}

// writeLessons writes the lessons of a reply through the learn path and adds what became of the reply, of each lesson
// and of each lesson it leaves out to the outcomes. A failure to get a reply holds no lessons.
func writeLessons(
	ctx context.Context, args AutoWriteArgs, deps AutoWriteDeps, reply auto.Reply, results *outcomes, stdout io.Writer,
) []error {
	if reply.Failure != "" {
		results.add(auto.RecordFailed, "", "", reply.Failure)

		return nil
	}

	lessons, skipped := auto.ParseLessons(reply.Text)

	for _, left := range skipped {
		results.add(auto.RecordSkipped, "", left.Slug, left.Reason)
	}

	if len(lessons) == 0 && len(skipped) == 0 {
		results.add(auto.RecordNothing, "", "", "")
	}

	var failures []error

	for _, lesson := range lessons {
		if covering := coveringNote(lesson.Covered, args.VaultPath, deps.NoteExists); covering != "" {
			results.add(auto.RecordSkipped, "", lesson.Slug, "the vault already states it: "+covering)

			continue
		}

		path, learnErr := deps.Learn(ctx, learnArgsOfLesson(lesson, args, reply.Session))
		if learnErr != nil {
			failures = append(failures, fmt.Errorf("writing %s: %w", lesson.Slug, learnErr))

			results.add(auto.RecordFailed, "", lesson.Slug, learnErr.Error())

			continue
		}

		_, _ = fmt.Fprintln(stdout, path)

		results.add(auto.RecordWritten, filepath.Base(path), lesson.Slug, "")
	}

	return failures
}

// writeRecords writes each record as a file of its own in the directory of its session, named for the time it was made,
// its place among the records of the reply and its kind, and reports whether every record was written. A record that
// cannot be written is left out (793d4k).
func writeRecords(write func(path string, data []byte) error, dir string, records []auto.Record) bool {
	if dir == "" {
		return len(records) == 0
	}

	written := true

	for index, record := range records {
		if !isPlainName(record.Session) {
			return false
		}

		data, marshalErr := json.Marshal(record)
		name := fmt.Sprintf("%s-%d-%s%s", record.At.UTC().Format(recordTimeFormat), index, record.Kind, recordFileExt)

		// a record that cannot be encoded or written leaves the others to be written, and the result false
		written = marshalErr == nil && write(filepath.Join(dir, record.Session, name), data) == nil && written
	}

	return written
}
