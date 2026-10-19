//go:build network

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

// unexported constants.
const (
	// callSlack is how far before the end of a step a model call may start and still be the step's call: the files of a
	// step and the clock of the meter are read at different moments.
	callSlack = 100 * time.Millisecond
	// the cue of the capture of a turn
	cueTurnEnd = "turn-end"
	// measureBound is the 2 minutes of every wait of the flow of a prompt (855d2a).
	measureBound = 2 * time.Minute
	// the kinds of the lines of the meter
	meterModel = "model"
	meterTool  = "tool"
	meterTurn  = "turn"
	// the steps of the chains, in order
	stepDeliver = "deliver"
	stepJudge   = "judge"
	stepMoment  = "moment"
	stepPhrases = "phrases"
	stepPlace   = "place"
	stepRecall  = "recall"
	stepWrite   = "write"
)

// unexported variables.
var (
	haikuPrice  = price{input: 0.10e-6, output: 0.50e-6, cacheRead: 0.01e-6, cacheWrite: 0.20e-6}
	sonnetPrice = price{input: 2e-6, output: 10e-6, cacheRead: 0.10e-6, cacheWrite: 4e-6}
)

// chain is the steps of one flow, in order, with the model call that followed each step that printed a request.
type chain struct {
	steps []stepRun
	calls map[string]meterRecord // by the name of the step whose request it answered
}

// callTook is how long the model call of a step took, or zero when the step had none.
func (c chain) callTook(step string) time.Duration {
	return time.Duration(c.calls[step].Took) * time.Millisecond
}

// depth is the depth the moment judgement chose for a chain of the delivery flow, from the reply that the step after
// it was given, or "" when the chain has no step after the moment.
func (c chain) depth() string {
	if len(c.steps) < 2 {
		return ""
	}

	var reply struct {
		Depth string `json:"depth"`
	}

	_ = json.Unmarshal([]byte(c.steps[1].field("reply")), &reply)

	return reply.Depth
}

// end is when the chain's last step ended.
func (c chain) end() time.Time { return c.steps[len(c.steps)-1].end }

// start is when the chain's first step began.
func (c chain) start() time.Time { return c.steps[0].start }

// stepTook is how long a step of the chain took, or zero when the chain has none.
func (c chain) stepTook(step string) time.Duration {
	for _, run := range c.steps {
		if run.step == step {
			return run.end.Sub(run.start)
		}
	}

	return 0
}

// took is the time of the whole chain: from the start of its first step to the end of its last.
func (c chain) took() time.Duration { return c.end().Sub(c.start()) }

// chains links the machine's step runs and model calls.
func (m *machine) chains() []chain { return chainsOf(m.stepRuns(), m.meterRecords()) }

// hasMeter installs the meter mod beside the mod of automatic memory.
func (m *machine) hasMeter() {
	m.t.Helper()

	root, err := moduleRoot()
	if err != nil {
		m.t.Fatalf("finding the module: %v", err)
	}

	m.installMod("meter", filepath.Join(root, "dev", "acceptance", "testdata", "meter"))
}

// meterRecords reads what the meter mod wrote, oldest first.
func (m *machine) meterRecords() []meterRecord {
	data, _ := os.ReadFile(filepath.Join(m.root, "meter.jsonl"))

	var records []meterRecord

	for line := range strings.SplitSeq(string(data), "\n") {
		var record meterRecord

		if json.Unmarshal([]byte(line), &record) == nil && record.Kind != "" {
			records = append(records, record)
		}
	}

	return records
}

// stepRuns reads the runs of the steps that stepsAreRecorded kept and that have ended.
func (m *machine) stepRuns() []stepRun {
	inputs, _ := filepath.Glob(filepath.Join(m.root, "steps-seen", "*-*.in"))

	var runs []stepRun

	for _, name := range inputs {
		base := strings.TrimSuffix(filepath.Base(name), ".in")
		step, _, _ := strings.Cut(base, "-")
		data, readErr := os.ReadFile(name)
		started, startErr := os.Stat(name)
		ended, endErr := os.Stat(strings.TrimSuffix(name, ".in") + ".done")
		output, _ := os.ReadFile(strings.TrimSuffix(name, ".in") + ".out")

		var input map[string]any

		if readErr != nil || startErr != nil || endErr != nil || json.Unmarshal(data, &input) != nil {
			continue
		}

		runs = append(runs, stepRun{
			step: step, input: input, output: string(output), start: started.ModTime(), end: ended.ModTime(),
		})
	}

	return runs
}

// meterRecord is one line of the meter mod: a model call, a tool call of the main loop, or the end of a turn.
type meterRecord struct {
	Kind     string      `json:"k"`
	At       int64       `json:"at"`
	Took     int64       `json:"took"`
	Model    string      `json:"model"`
	System   int         `json:"system"`
	Prompt   int         `json:"prompt"`
	Head     string      `json:"head"`
	Answered bool        `json:"answered"`
	Usage    *tokenUsage `json:"usage"`
	Reply    string      `json:"reply"`
	Tool     string      `json:"tool"`
	Command  string      `json:"command"`
}

// price is what a token costs in USD.
type price struct {
	input, output, cacheRead, cacheWrite float64
}

// stepRun is one run of an `engram auto` step, as stepsAreRecorded keeps it.
type stepRun struct {
	step   string
	input  map[string]any
	output string
	start  time.Time // when the step was given its input
	end    time.Time // when the step exited
}

// carry is the carry the step printed in canonical form, or "" when it printed no request.
func (r stepRun) carry() string {
	var printed struct {
		Request json.RawMessage `json:"request"`
		Carry   any             `json:"carry"`
	}

	if json.Unmarshal([]byte(r.output), &printed) != nil || printed.Request == nil {
		return ""
	}

	return canonical(printed.Carry)
}

// cue is the cue the step was given.
func (r stepRun) cue() string { return r.field("cue") }

// field is a field of the step's input as text, canonical when it is not a string.
func (r stepRun) field(name string) string {
	if text, isText := r.input[name].(string); isText {
		return text
	}

	if r.input[name] == nil {
		return ""
	}

	return canonical(r.input[name])
}

// request is the request the step printed: the model, the system and the prompt, or false when it printed none.
func (r stepRun) request() (request struct{ Model, System, Prompt string }, printed bool) {
	var out struct {
		Request struct {
			Model  string `json:"model"`
			System string `json:"system"`
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	if json.Unmarshal([]byte(r.output), &out) != nil || r.carry() == "" {
		return request, false
	}

	request.Model, request.System, request.Prompt = out.Request.Model, out.Request.System, out.Request.Prompt

	return request, true
}

// tokenUsage is what a model call cost in tokens, as the host reports it.
type tokenUsage struct {
	Input       int    `json:"input_tokens"`                //nolint:tagliatelle // the host spells a usage in snake_case
	Output      int    `json:"output_tokens"`               //nolint:tagliatelle // the host spells a usage in snake_case
	CacheRead   int    `json:"cache_read_input_tokens"`     //nolint:tagliatelle // the host spells a usage in snake_case
	CacheCreate int    `json:"cache_creation_input_tokens"` //nolint:tagliatelle // the host spells a usage in snake_case
	Model       string `json:"model"`
}

// cost is the cost of the tokens at a price, in USD.
func (u tokenUsage) cost(at price) float64 {
	return float64(u.Input)*at.input + float64(u.Output)*at.output + float64(u.CacheRead)*at.cacheRead +
		float64(u.CacheCreate)*at.cacheWrite
}

// plus is the sum of two usages.
func (u tokenUsage) plus(other tokenUsage) tokenUsage {
	return tokenUsage{
		Input: u.Input + other.Input, Output: u.Output + other.Output, CacheRead: u.CacheRead + other.CacheRead,
		CacheCreate: u.CacheCreate + other.CacheCreate,
	}
}

// byCarry is the step of a name that was given the carry that a step printed.
func byCarry(from stepRun, name string, ordered []stepRun) (stepRun, bool) {
	carry := from.carry()
	if carry == "" {
		return stepRun{}, false
	}

	for _, run := range ordered {
		if run.step == name && run.field("carry") == carry {
			return run, true
		}
	}

	return stepRun{}, false
}

// callsOf finds the model call of each step that printed a request: the earliest call, not yet taken, of the request's
// size that began after the step ended.
func callsOf(steps []stepRun, meter []meterRecord, used map[int]bool) map[string]meterRecord {
	calls := map[string]meterRecord{}

	for _, run := range steps {
		request, printed := run.request()
		if !printed {
			continue
		}

		for index, record := range meter {
			began := time.UnixMilli(record.At)

			if record.Kind == meterModel && !used[index] && record.System == len(request.System) &&
				record.Prompt == len(request.Prompt) && !began.Before(run.end.Add(-callSlack)) {
				used[index] = true
				calls[run.step] = record

				break
			}
		}
	}

	return calls
}

// canonical writes a decoded JSON value in a form that equal values share.
func canonical(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(data)
}

// captureOf is the steps of the capture that begins with a moment: judge and place by the carry, and the write of the
// same turn that follows them.
func captureOf(first stepRun, ordered []stepRun) []stepRun {
	linked := []stepRun{first}

	for _, name := range []string{stepJudge, stepPlace} {
		next, found := byCarry(linked[len(linked)-1], name, ordered)
		if !found {
			break
		}

		linked = append(linked, next)
	}

	last := linked[len(linked)-1]

	for _, run := range ordered {
		if run.step == stepWrite && run.field("turn") == first.field("turn") && run.field("reply") != "" &&
			!run.start.Before(last.end) {
			return append(linked, run)
		}
	}

	return linked
}

// chainsOf links the runs of the steps into the chains of the delivery flow (855d2a, 855d2c) and of the capture
// (855d2b), each in the order of its first step. A step is linked to the one before it by the carry the one before
// printed; a delivery that follows a step that printed nothing is the earliest one for the same session, cue and text
// and event that began after it; a write is linked to a capture by the turn.
func chainsOf(runs []stepRun, meter []meterRecord) []chain {
	ordered := slices.SortedFunc(slices.Values(runs), func(a, b stepRun) int { return a.start.Compare(b.start) })
	used := map[int]bool{}

	var chains []chain

	for _, first := range ordered {
		if first.step != stepMoment {
			continue
		}

		var linked []stepRun

		if first.cue() == cueTurnEnd {
			linked = captureOf(first, ordered)
		} else {
			linked = deliveryOf(first, ordered)
		}

		chains = append(chains, chain{steps: linked, calls: callsOf(linked, meter, used)})
	}

	return chains
}

// deliveryAfter is the earliest delivery for the session, cue, text and event of a chain's first step that began after
// the last step ended.
func deliveryAfter(first, last stepRun, ordered []stepRun) (stepRun, bool) {
	for _, run := range ordered {
		if run.step == stepDeliver && run.field("session") == first.field("session") && run.cue() == first.cue() &&
			run.field("text") == first.field("text") && run.field("event") == first.field("event") &&
			!run.start.Before(last.end) {
			return run, true
		}
	}

	return stepRun{}, false
}

// deliveryOf is the steps of the delivery flow that begins with a moment: phrases and recall by the carry, then the
// delivery.
func deliveryOf(first stepRun, ordered []stepRun) []stepRun {
	linked := []stepRun{first}

	for _, name := range []string{stepPhrases, stepRecall} {
		next, found := byCarry(linked[len(linked)-1], name, ordered)
		if !found {
			break
		}

		linked = append(linked, next)
	}

	last := linked[len(linked)-1]

	if next, found := byCarry(last, stepDeliver, ordered); found {
		return append(linked, next)
	}

	if next, found := deliveryAfter(first, last, ordered); found {
		return append(linked, next)
	}

	return linked
}

// longest is the greatest of the durations.
func longest(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	return slices.Max(durations)
}

// median is the middle of the durations, and the greater of the two middle ones for an even count.
func median(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := slices.Sorted(slices.Values(durations))

	return sorted[len(sorted)/2]
}

// secs writes a duration as seconds with two decimals, or "-" for none.
func secs(took time.Duration) string {
	if took == 0 {
		return "-"
	}

	return fmt.Sprintf("%.2f", took.Seconds())
}

// table writes rows as columns under a header, for the log of a measurement.
func table(header []string, rows [][]string) string {
	var out strings.Builder

	writer := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(writer, strings.Join(header, "\t"))

	for _, row := range rows {
		_, _ = fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	_ = writer.Flush()

	return out.String()
}
