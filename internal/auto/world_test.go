package auto_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/toejough/engram/internal/auto"
)

// Exported constants.
const (
	EmptyQuery = "version: 1\nphrases: []\nitems: []\nclusters: []\n"
	// NoteQuery is the output of `engram query` holding one note.
	NoteQuery = `version: 1
phrases:
  - declaring work done
items:
  - path: 1.2026-10-12.note-n.md
    kind: fact
    score: 0.49
    provenances:
      - direct
    content: |+
      ---
      type: fact
      situation: declaring a Go package rename done
      object: run go build first (SENTINEL-N)
      ---

      Information learned: run go build first (SENTINEL-N).

clusters:
`
)

// Exported variables.
var (
	LessonJSON = `[{"kind":"feedback","slug":"use-targ-not-go-test","situation":"running Go tests in this repo",` +
		`"covered":null,"behavior":"ran go test directly","impact":"skips the repo gates","action":"use targ test"}]`
)

// Command is a command the job ran.
type Command struct {
	Argv     []string
	Deadline time.Duration
}

// Line is the command as one line.
func (c Command) Line() string { return strings.Join(c.Argv, " ") }

// Name is the subcommand of an engram command, else the program.
func (c Command) Name() string {
	if c.Argv[0] == "engram" && len(c.Argv) > 1 {
		return c.Argv[1]
	}

	return c.Argv[0]
}

// World is the fake world of one test.
type World struct {
	mu       sync.Mutex
	Commands []Command
	// Process answers a command by its subcommand; a command it does not know prints nothing, a query prints
	// EmptyQuery and a learn prints a note path.
	Process map[string]func(argv []string) (auto.Output, error)
	Files   map[string]string
	Writes  []string
	// Unwritable are the paths the world refuses to write.
	Unwritable map[string]bool
	Written    map[string]string
	Env        map[string]string
	Clock      time.Time
	Missing    map[string]bool
	// Model answers a request by its kind (classify, relevance, capture, curation); a function may answer unanswered.
	Model    func(kind string, request auto.Request) map[string]any
	Requests []auto.Request
	Logs     []string
	Warnings []string
	Counts   auto.Counts
	Waits    []int
	// OnWait is called with the number of waits so far when the job asks the mod to wait.
	OnWait func(waits int)
	Ledger json.RawMessage
	Work   json.RawMessage
	// Rounds is how many times the job was called.
	Rounds int
}

// NewWorld is a world with an empty vault at /vault.
func NewWorld() *World {
	return &World{
		Process: map[string]func([]string) (auto.Output, error){},
		Files:   map[string]string{}, Written: map[string]string{}, Missing: map[string]bool{}, Unwritable: map[string]bool{},
		Env: map[string]string{"HOME": "/home/t", "ENGRAM_VAULT_PATH": "/vault"}, Clock: time.UnixMilli(1_700_000_000_000),
	}
}

// Deps are the world as the job sees it.
func (w *World) Deps() auto.Deps { return auto.Deps{Runner: w, Files: w, Env: w, Clock: w} }

// Do plays the mod for one job: it calls the job, relays what it needs, and returns its end.
func (w *World) Do(t *testing.T, job map[string]any) auto.Done {
	t.Helper()

	input := maps.Clone(job)

	for range maxRounds {
		if w.Ledger != nil {
			input["ledger"] = w.Ledger
		}

		if w.Work != nil {
			input["work"] = w.Work
		}

		response := w.round(t, input)
		if response.Done != nil {
			if slices.Contains([]string{"lookup", "step", "tool", "compact"}, field[string](job, "kind")) {
				w.Ledger = response.Done.Ledger
			} else {
				w.Work = response.Done.Work
			}

			return *response.Done
		}

		input["state"], input["answers"] = response.State, w.relay(response)
	}

	t.Fatal("the job did not end")

	return auto.Done{}
}

// Exists says whether a path is in the world: the vault always is, unless the test removes it.
func (w *World) Exists(path string) bool {
	_, found := w.Files[path]

	return found || (path == w.Env["ENGRAM_VAULT_PATH"] && !w.Missing[path])
}

// Find is the commands of a subcommand.
func (w *World) Find(name string) []Command {
	found := []Command{}

	for _, command := range w.Commands {
		if command.Name() == name {
			found = append(found, command)
		}
	}

	return found
}

// Get reads a variable.
func (w *World) Get(key string) string { return w.Env[key] }

// Names are the subcommands of the commands the job ran, in order.
func (w *World) Names() []string {
	names := make([]string, 0, len(w.Commands))
	for _, command := range w.Commands {
		names = append(names, command.Name())
	}

	return names
}

// Now is the time of the world.
func (w *World) Now() time.Time { return w.Clock }

// Read reads a file of the world.
func (w *World) Read(path string) (string, error) {
	text, found := w.Files[path]
	if !found || w.Missing[path] {
		return "", fmt.Errorf("open %s: no such file or directory", path)
	}

	return text, nil
}

// Run runs the command as the engram program would.
func (w *World) Run(argv []string, deadline time.Duration) (auto.Output, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	command := Command{Argv: argv, Deadline: deadline}
	w.Commands = append(w.Commands, command)

	if answer, known := w.Process[command.Name()]; known {
		return answer(argv)
	}

	switch command.Name() {
	case "query":
		return auto.Output{Stdout: EmptyQuery}, nil
	case "learn":
		return auto.Output{Stdout: "/vault/9.2026-10-12.written.md\n"}, nil
	}

	return auto.Output{}, nil
}

// Walk lists the files below a root.
func (w *World) Walk(root string) ([]string, error) {
	if w.Missing[root+"/walk"] {
		return nil, errors.New("open " + root + ": permission denied")
	}

	var files []string

	for path := range w.Files {
		if strings.HasPrefix(path, root+"/") {
			files = append(files, path)
		}
	}

	slices.Sort(files)

	return files, nil
}

// Write writes a file of the world.
func (w *World) Write(path, text string) error {
	if w.Unwritable[path] {
		return errors.New("open " + path + ": permission denied")
	}

	w.Writes = append(w.Writes, path)
	w.Written[path] = text
	w.Files[path] = text

	return nil
}

func (w *World) model(request auto.Request) map[string]any {
	if w.Model == nil {
		return Answer("[]")
	}

	return w.Model(kindOf(request.Prompt), request)
}

// relay waits as long as the job asks, and brings the replies of the model to its requests.
func (w *World) relay(response auto.Response) []map[string]any {
	if response.WaitMs > 0 {
		w.Waits = append(w.Waits, response.WaitMs)

		if w.OnWait != nil {
			w.OnWait(len(w.Waits))
		}

		w.Clock = w.Clock.Add(time.Duration(response.WaitMs) * time.Millisecond)
	}

	answers := make([]map[string]any, 0, len(response.Requests))

	for _, request := range response.Requests {
		w.Requests = append(w.Requests, request)
		answers = append(answers, w.model(request))
	}

	return answers
}

// round calls the job once, and tallies the effects of the round.
func (w *World) round(t *testing.T, input map[string]any) auto.Response {
	t.Helper()

	output, err := auto.Run(w.Deps(), asJSON(input))
	if err != nil {
		t.Fatalf("the job failed: %v", err)
	}

	w.Rounds++

	var response auto.Response

	err = json.Unmarshal(output, &response)
	if err != nil {
		t.Fatalf("the response cannot be read: %v", err)
	}

	w.Logs, w.Warnings = append(w.Logs, response.Log...), append(w.Warnings, response.Warnings...)
	w.Counts.Delivered += response.Counts.Delivered
	w.Counts.Added += response.Counts.Added
	w.Counts.Amended += response.Counts.Amended
	w.Counts.Curated += response.Counts.Curated

	return response
}

// Answer is a reply of the model.
func Answer(text string) map[string]any {
	return map[string]any{"isAnswered": true, "text": text, "usage": map[string]any{}}
}

// Replies answers each kind with a fixed text.
func Replies(texts map[string]string) func(string, auto.Request) map[string]any {
	return func(kind string, _ auto.Request) map[string]any {
		if text, found := texts[kind]; found {
			return Answer(text)
		}

		return Answer("[]")
	}
}

// Unanswered is a model call that was cut or refused.
func Unanswered(reason string) map[string]any {
	return map[string]any{"isAnswered": false, "reason": reason}
}

// unexported constants.
const (
	maxRounds = 200
)

// kindOf tells the kind of a request from its prompt.
func kindOf(prompt string) string {
	switch {
	case strings.Contains(prompt, "You watch an AI coding agent"):
		return "classify"
	case strings.Contains(prompt, "which vault notes bear"):
		return "relevance"
	case strings.Contains(prompt, "keep the long-term memory"):
		return "capture"
	}

	return "curation"
}
