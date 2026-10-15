package auto

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// Counts is what the System did: the items it delivered, the notes it added and amended, the offers it curated.
type Counts struct {
	Delivered int `json:"delivered"`
	Added     int `json:"added"`
	Amended   int `json:"amended"`
	Curated   int `json:"curated"`
}

// plus is the sum of two counts.
func (c Counts) plus(o Counts) Counts {
	return Counts{c.Delivered + o.Delivered, c.Added + o.Added, c.Amended + o.Amended, c.Curated + o.Curated}
}

// Done is the end of a job: what it delivers, and the session state the mod keeps for it.
type Done struct {
	Block     string          `json:"block,omitempty"`
	Line      string          `json:"line,omitempty"`
	Notes     []Note          `json:"notes,omitempty"`
	Ledger    json.RawMessage `json:"ledger,omitempty"`
	Work      json.RawMessage `json:"work,omitempty"`
	Show      bool            `json:"show,omitempty"`
	LineShown bool            `json:"lineShown,omitempty"`
}

// Job is what the mod hands to `engram auto run`: the event it has, the record of an earlier round of the job and the
// replies of the model to its requests. A job names only the fields its kind reads.
type Job struct {
	Kind        string          `json:"kind"`
	Step        string          `json:"step"`
	Moment      string          `json:"moment"`
	Agent       string          `json:"agent"`
	Text        string          `json:"text"`
	Transcript  string          `json:"transcript"`
	LastPrompt  string          `json:"lastPrompt"`
	Args        string          `json:"args"`
	Answer      string          `json:"answer"`
	SessionID   string          `json:"sessionId"`
	Path        string          `json:"path"`
	Last        string          `json:"last"`
	ReadContext bool            `json:"readContext"`
	Route       bool            `json:"route"`
	TurnEnded   bool            `json:"turnEnded"`
	LineShown   bool            `json:"lineShown"`
	Turn        int             `json:"turn"`
	State       []record        `json:"state"`
	Reply       *reply          `json:"reply"`
	Ledger      json.RawMessage `json:"ledger"`
	Work        json.RawMessage `json:"work"`
	Response    json.RawMessage `json:"toolResponse"`
	Held        []Note          `json:"held"`
	ToolUses    []ToolCall      `json:"toolUses"`
	Messages    []Message       `json:"messages"`
	Tally       []Counts        `json:"tally"`
	Warnings    []string        `json:"warnings"`
}

// Response is what `engram auto run` prints: the effects of the round for the mod to apply (the debug log lines, the
// warnings to show and the counts to tally), then either the end of the job or what it needs: the model request to
// relay, a time to wait, and the state to send back with the reply.
type Response struct {
	Log      []string `json:"log,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Counts   Counts   `json:"counts"`
	Done     *Done    `json:"done,omitempty"`
	Request  *Request `json:"request,omitempty"`
	WaitMs   int      `json:"waitMs,omitempty"`
	State    []record `json:"state,omitempty"`
}

// Run reads one job from its JSON, advances it one round and returns the response as JSON. A step that fails is a
// warning in the response (793a4b); only a job that cannot be read is an error.
func Run(deps Deps, input []byte) ([]byte, error) {
	e := &engine{deps: deps, tiers: DefaultTiers(), ledger: ledger{}}
	jobs := map[string]func() Done{
		"lookup": e.lookupJob, "deep": e.deepJob, "step": e.stepJob, "tool": e.toolJob, "compact": e.compactJob,
		"capture": e.captureJob, "subagent": e.subagentJob, "housekeeping": e.housekeepingJob, "summary": e.summaryJob,
	}

	err := json.Unmarshal(input, &e.job)
	if err != nil {
		return nil, fmt.Errorf("reading the job: %w", err)
	}

	run, known := jobs[e.job.Kind]
	if !known {
		return nil, fmt.Errorf("%w: %q", errJobKind, e.job.Kind)
	}

	for _, state := range []struct {
		raw json.RawMessage
		to  any
	}{{e.job.Ledger, &e.ledger}, {e.job.Work, &e.work}} {
		if len(state.raw) > 0 {
			err = json.Unmarshal(state.raw, state.to)
		}

		if err != nil {
			return nil, fmt.Errorf("the session state cannot be read: %w", err)
		}
	}

	e.resume()

	var done Done

	if waiting := e.guard(e.job.Step, func() { done = run() }); waiting != nil {
		e.resp.Request, e.resp.WaitMs, e.resp.State = waiting.request, waiting.waitMs, e.journal
	} else {
		done.Ledger, done.Work = must(json.Marshal(e.ledger)), must(json.Marshal(e.work))
		e.resp.Done = &done
	}

	return must(json.Marshal(e.resp)), nil // a response of plain values cannot fail to encode
}

// engine is one round of one job. A job is straight-line code that fails as the JavaScript it replaces did: by panic
// with an error (a StepError of its class), which guard contains.
type engine struct {
	deps      Deps
	tiers     Tiers
	job       Job
	journal   []record
	pos       int
	replayLen int
	resp      Response
	ledger    ledger
	work      workState
}

// ask has the mod relay a request to the model, and returns the reply once the next round has it.
func (e *engine) ask(request Request) reply {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(request.Prompt)))

	rec, replayed := e.next()
	if !replayed {
		e.journal = append(e.journal, record{Hash: hash, Pending: true})
		panic(&pause{request: &request})
	}

	var answer reply

	unchanged := rec.Hash == hash && json.Unmarshal([]byte(rec.Out), &answer) == nil

	check(unchanged, "what the job asked changed between its rounds")

	return answer
}

// attempt is a guard inside a job: a pause goes on unwinding.
func (e *engine) attempt(step string, run func()) {
	if waiting := e.guard(step, run); waiting != nil {
		panic(waiting)
	}
}

// count adds to the counts of the round.
func (e *engine) count(add Counts) {
	if e.live() {
		e.resp.Counts = e.resp.Counts.plus(add)
	}
}

// guard runs a step the failure of which is contained: the event goes on as if the mod were not there, and the user
// is warned. A step that needs the mod is not a failure; its pause is returned.
func (e *engine) guard(step string, run func()) (waiting *pause) {
	defer func() {
		switch fault := recover().(type) {
		case nil:
		case *pause:
			waiting = fault
		case error:
			e.warn(step, fault)
		default:
			panic(fault)
		}
	}()

	run()

	return nil
}

// live says whether the code is past the effects an earlier round replays: what it reports is new.
func (e *engine) live() bool { return e.pos >= e.replayLen }

// logf adds a line for the mod to write to the debug log.
func (e *engine) logf(format string, args ...any) {
	if e.live() {
		e.resp.Log = append(e.resp.Log, "engram-memory: "+fmt.Sprintf(format, args...))
	}
}

// next is the record the earlier rounds made of the effect at hand, if they made one.
func (e *engine) next() (record, bool) {
	e.pos++

	if e.pos > len(e.journal) {
		return record{}, false
	}

	return e.journal[e.pos-1], true
}

// quietly reads a file that may not be there or may have grown, and is not recorded: its use is recorded.
func (e *engine) quietly(path string) string {
	text, _ := e.deps.Files.Read(path)

	return text
}

// read reads a file once: a file that is not there is an error, recorded as such.
func (e *engine) read(path string) (string, error) {
	return e.step(func() (string, error) { return e.deps.Files.Read(path) })
}

// resume takes the record of the earlier rounds, with the reply the mod brought for the last one.
func (e *engine) resume() {
	e.journal = e.job.State

	if last := len(e.journal) - 1; last >= 0 && e.journal[last].Pending {
		e.journal[last].Pending = false

		if e.job.Reply != nil {
			e.journal[last].Out = string(must(json.Marshal(e.job.Reply)))
		}
	}

	e.replayLen = len(e.journal)
}

// step performs an effect once: when the round replays it, it returns what it returned before.
func (e *engine) step(effect func() (string, error)) (string, error) {
	rec, replayed := e.next()
	if !replayed {
		out, err := effect()

		rec = record{Out: out}
		if err != nil {
			rec.Err, rec.Kind = err.Error(), kindOf(err)
		}

		e.journal = append(e.journal, rec)
	}

	if rec.Err != "" {
		return rec.Out, &StepError{Message: rec.Err, Kind: rec.Kind}
	}

	return rec.Out, nil
}

// wait has the mod wait before the next round.
func (e *engine) wait(milliseconds int) {
	if _, replayed := e.next(); !replayed {
		e.journal = append(e.journal, record{Pending: true})
		panic(&pause{waitMs: milliseconds})
	}
}

// warn adds a warning for the mod to show on the status line.
func (e *engine) warn(step string, err error) {
	e.logf("%s: %s", orElse(step, "internal"), err)

	if e.live() && step != "" {
		e.resp.Warnings = append(e.resp.Warnings, warning(step, kindOf(err)))
	}
}

// write writes a file once.
func (e *engine) write(path, text string) {
	must(e.step(func() (string, error) { return "", e.deps.Files.Write(path, text) }))
}

// pause unwinds a job that needs the mod: it has asked for a model reply or a wait.
type pause struct {
	request *Request
	waitMs  int
}

// record is what one effect of the job returned.
type record struct {
	Out     string `json:"out,omitempty"`
	Err     string `json:"err,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Pending bool   `json:"pending,omitempty"`
}

// must returns the value of a call that fails by panic.
func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

// orElse is the text, or the fallback when the text is empty.
func orElse(text, fallback string) string {
	if text == "" {
		return fallback
	}

	return text
}

// recorded performs once an effect that has a value.
func recorded[T any](e *engine, effect func() T) T {
	var value T

	out, _ := e.step(func() (string, error) {
		encoded, err := json.Marshal(effect())

		return string(encoded), err
	})
	_ = json.Unmarshal([]byte(out), &value)

	return value
}
