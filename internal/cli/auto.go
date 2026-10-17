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

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/auto"
)

// AutoDeliverArgs holds parsed flags for `engram auto deliver`.
type AutoDeliverArgs struct {
	VaultPath   string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
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
}

// AutoJudgeArgs holds parsed flags for `engram auto judge`.
type AutoJudgeArgs struct {
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
}

// AutoJudgeDeps holds injected dependencies for RunAutoJudge.
type AutoJudgeDeps struct {
	// Stdin carries the turn that ended: one JSON object with the session's ID and the turn's final text.
	Stdin io.Reader
	// ReadDir and ReadFile read the session-log directory, where Claude Code keeps the record of each session.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
}

// AutoWriteArgs holds parsed flags for `engram auto write`.
type AutoWriteArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	VaultName string `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on the note's vault: field (default \"personal\")"` //nolint:lll // single unbreakable struct-tag string
}

// AutoWriteDeps holds injected dependencies for RunAutoWrite.
type AutoWriteDeps struct {
	// Stdin carries the reply of the model: one JSON object with the session's ID and the reply.
	Stdin io.Reader
	// Learn writes a note through the learn path.
	Learn LearnRun
}

// LearnRun writes a note through the learn path, as `engram learn` does, and returns the path it prints for the note.
type LearnRun func(ctx context.Context, args LearnArgs) (string, error)

// QueryRun runs the lookup of `engram query` for the arguments it is given and prints its payload.
type QueryRun func(ctx context.Context, args QueryArgs, stdout io.Writer) error

// RunAutoDeliver reads a prompt from standard input and prints the block of the notes that bear on it (793d4a).
func RunAutoDeliver(ctx context.Context, args AutoDeliverArgs, deps AutoDeliverDeps, stdout io.Writer) error {
	var request auto.Request

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&request)
	if decodeErr != nil {
		return fmt.Errorf("auto deliver: reading the request: %w", decodeErr)
	}

	block, err := auto.Deliver(ctx, request,
		func(ctx context.Context, text string) ([]auto.Note, error) {
			return lookUpNotes(ctx, args.VaultPath, text, deps.Query)
		},
		func(session string) string {
			return heldText(deps, args.SessionsDir, session)
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

// RunAutoJudge reads a turn that ended from standard input and prints the request for the model that judges it, or
// nothing when the session's record holds no prompt (793d4g).
func RunAutoJudge(_ context.Context, args AutoJudgeArgs, deps AutoJudgeDeps, stdout io.Writer) error {
	var turn auto.TurnEnd

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&turn)
	if decodeErr != nil {
		return fmt.Errorf("auto judge: reading the turn: %w", decodeErr)
	}

	request, found := auto.Judge(recordOf(deps.ReadDir, deps.ReadFile, args.SessionsDir, turn.Session), turn.Answer)
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

// RunAutoWrite reads the reply of the model to the judgement of a turn from standard input and writes each lesson it
// holds as a note, marked as written automatically, printing the path of each note (793d4h).
func RunAutoWrite(ctx context.Context, args AutoWriteArgs, deps AutoWriteDeps, stdout io.Writer) error {
	var reply auto.Reply

	decodeErr := json.NewDecoder(deps.Stdin).Decode(&reply)
	if decodeErr != nil {
		return fmt.Errorf("auto write: reading the reply: %w", decodeErr)
	}

	// a reply that is not a list of lessons holds none
	lessons, _ := auto.ParseLessons(reply.Text)

	var failures []error

	for _, lesson := range lessons {
		path, learnErr := deps.Learn(ctx, learnArgsOfLesson(lesson, args, reply.Session))
		if learnErr != nil {
			failures = append(failures, fmt.Errorf("writing %s: %w", lesson.Slug, learnErr))

			continue
		}

		_, _ = fmt.Fprintln(stdout, path)
	}

	return errors.Join(failures...)
}

// findRecord reads the record of a session, which Claude Code keeps as <ID>.jsonl in a directory of the session-log
// directory for the project, or in the session-log directory itself.
func findRecord(
	readDir func(path string) ([]fs.DirEntry, error), readFile func(path string) ([]byte, error), dir, session string,
) []byte {
	name := session + ".jsonl"
	candidates := []string{filepath.Join(dir, name)}

	// a directory that cannot be listed has no project directories to search
	entries, _ := readDir(dir)

	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, filepath.Join(dir, entry.Name(), name))
		}
	}

	for _, candidate := range candidates {
		record, err := readFile(candidate)
		if err == nil {
			return record
		}
	}

	return nil
}

// heldText returns the text of what a session's agent holds of the vault (793d4d). A record that cannot be read holds
// nothing, so a prompt is delivered a note it may already hold rather than none.
func heldText(deps AutoDeliverDeps, dir, session string) string {
	return auto.HeldText(recordOf(deps.ReadDir, deps.ReadFile, dir, session))
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

// newAutoDeliverDeps wires RunAutoDeliver from the injected CLI capabilities — pure composition (#700).
func newAutoDeliverDeps(d Deps) AutoDeliverDeps {
	queryDeps := newQueryDeps(d)

	return AutoDeliverDeps{
		Stdin: d.Stdin,
		Query: func(ctx context.Context, args QueryArgs, stdout io.Writer) error {
			return RunQuery(ctx, args, queryDeps, stdout)
		},
		ReadDir:  d.FS.ReadDir,
		ReadFile: d.FS.ReadFile,
	}
}

// newAutoJudgeDeps wires RunAutoJudge from the injected CLI capabilities — pure composition (#700).
func newAutoJudgeDeps(d Deps) AutoJudgeDeps {
	return AutoJudgeDeps{Stdin: d.Stdin, ReadDir: d.FS.ReadDir, ReadFile: d.FS.ReadFile}
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

		_, situation := frontmatterTypeAndSituation([]byte(frontmatter))

		notes = append(notes, auto.Note{Path: item.Path, Situation: situation, Body: body})
	}

	return notes, nil
}

// recordOf returns the record of a session, or nothing. A session whose ID is not a plain file name has none: its ID
// would otherwise name a file outside the directory (793d4d). Nor does a session when there is no directory of records
// (793d4f): the file name alone would name a file of the working directory.
func recordOf(
	readDir func(path string) ([]fs.DirEntry, error), readFile func(path string) ([]byte, error), dir, session string,
) []byte {
	if dir == "" || session == "" || session != filepath.Base(session) || session == ".." {
		return nil
	}

	return findRecord(readDir, readFile, dir, session)
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
