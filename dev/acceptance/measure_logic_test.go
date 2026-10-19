package acceptance_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
)

// unexported constants.
const (
	// callSlack is how far before the end of a step a model call may start and still be the step's call: the files of a
	// step and the clock of the meter are read at different moments.
	callSlack = 100 * time.Millisecond
	// the cue of the capture of a turn
	cueTurnEnd = "turn-end"
	// the kind of the lines of the meter that hold a model call
	meterModel = "model"
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
	// liveLimits are the limits of every live row of 793 and 855 that is not a measurement.
	liveLimits = sessionLimits{budgetUSD: "2", timeout: 5 * time.Minute}
	// measureLimits are the limits of the conversations of the measurement rows of 855d7a1 (855b112).
	measureLimits = sessionLimits{budgetUSD: "10", timeout: 30 * time.Minute}
	haikuPrice    = price{input: 0.10e-6, output: 0.50e-6, cacheRead: 0.01e-6, cacheWrite: 0.20e-6}
	sonnetPrice   = price{input: 2e-6, output: 10e-6, cacheRead: 0.10e-6, cacheWrite: 4e-6}
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

// promptRecall is what the chain of a prompt looked up: the depth chosen and the notes among its candidates.
type promptRecall struct {
	depth      string
	candidates []string
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

// all is every token the host counted: the fresh input and the output, and the input read from and written to the
// cache.
func (u tokenUsage) all() int { return u.Input + u.Output + u.CacheRead + u.CacheCreate }

// fresh is the tokens the model processed anew: the input and the output, and the input written to the cache; the
// input read from the cache is not.
func (u tokenUsage) fresh() int { return u.Input + u.Output + u.CacheCreate }

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

// wordSplitter reads the characters of a command line one at a time.
type wordSplitter struct {
	words   []string
	word    strings.Builder
	quote   rune
	started bool
	escaped bool
}

// end closes the word being read, if there is one.
func (s *wordSplitter) end() {
	if s.started {
		s.words = append(s.words, s.word.String())
		s.word.Reset()

		s.started = false
	}
}

// feed reads one character.
func (s *wordSplitter) feed(char rune) {
	switch {
	case s.escaped:
		s.word.WriteRune(char)

		s.escaped = false
	case char == '\\' && s.quote != '\'':
		s.escaped, s.started = true, true
	case s.quote != 0:
		s.feedQuoted(char)
	default:
		s.feedPlain(char)
	}
}

// feedPlain reads one character outside quotes.
func (s *wordSplitter) feedPlain(char rune) {
	switch {
	case char == '\'' || char == '"':
		s.quote, s.started = char, true
	case unicode.IsSpace(char):
		s.end()
	default:
		s.word.WriteRune(char)

		s.started = true
	}
}

// feedQuoted reads one character inside quotes.
func (s *wordSplitter) feedQuoted(char rune) {
	if char == s.quote {
		s.quote = 0

		return
	}

	s.word.WriteRune(char)
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

// candidateNotes are the file names of the notes among the candidates that a recall step printed.
func candidateNotes(output string) []string {
	var printed struct {
		Carry struct {
			Candidates []struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"candidates"`
		} `json:"carry"`
	}

	if json.Unmarshal([]byte(output), &printed) != nil {
		return nil
	}

	var notes []string

	for _, candidate := range printed.Carry.Candidates {
		if candidate.Kind == "note" {
			notes = append(notes, candidate.ID)
		}
	}

	return notes
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

// falseNone counts, among the prompts for which the recall skill drew on a note, those for which the moment judgement
// chose none.
func falseNone(chains []promptRecall, drawn [][]string) (none, withNotes int) {
	for index, notes := range drawn {
		if len(notes) == 0 || index >= len(chains) {
			continue
		}

		withNotes++

		if chains[index].depth == "none" {
			none++
		}
	}

	return none, withNotes
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

// namedNotes are the notes of the vault, by file name, that a reply names.
func namedNotes(reply string, vault []string) []string {
	var named []string

	for _, name := range vault {
		if strings.Contains(reply, name) {
			named = append(named, name)
		}
	}

	return named
}

// overlap are the items of the first list that the second holds.
func overlap(drawn, found []string) []string {
	return slices.DeleteFunc(slices.Clone(drawn), func(item string) bool { return !slices.Contains(found, item) })
}

// queryPhrases are the phrases of the engram queries that a shell command runs, each written `--phrase text` or
// `--phrase=text`.
func queryPhrases(command string) []string {
	var phrases []string

	words := shellWords(command)

	for index, word := range words {
		switch {
		case word == "--phrase" && index+1 < len(words):
			phrases = append(phrases, words[index+1])
		case strings.HasPrefix(word, "--phrase="):
			phrases = append(phrases, strings.TrimPrefix(word, "--phrase="))
		}
	}

	return phrases
}

// ratio is a share, or zero when there is nothing to share.
func ratio(part, whole int) float64 {
	if whole == 0 {
		return 0
	}

	return float64(part) / float64(whole)
}

// secs writes a duration as seconds with two decimals, or "-" for none.
func secs(took time.Duration) string {
	if took == 0 {
		return "-"
	}

	return fmt.Sprintf("%.2f", took.Seconds())
}

// shellWords splits a command line into words as a shell does for quotes and backslashes: no expansion is done.
func shellWords(text string) []string {
	var splitter wordSplitter

	for _, char := range text {
		splitter.feed(char)
	}

	splitter.end()

	return splitter.words
}

// unknownDepths are the depths that a prompt's chain arrived at which are none of none, glance and deep: a prompt with
// no chain has the depth "" and must not count as not none.
func unknownDepths(arrived []promptRecall) []string {
	var unknown []string

	for _, recall := range arrived {
		if !slices.Contains([]string{"none", "glance", "deep"}, recall.depth) {
			unknown = append(unknown, recall.depth)
		}
	}

	return unknown
}

// unmatchedRequests names each step, as "<cue> <step>", that printed a request and has no model call matched to it: a
// row that reads the time of a call as zero for such a step would pass with a recorder that saw no call.
func unmatchedRequests(chains []chain) []string {
	var unmatched []string

	for _, each := range chains {
		for _, run := range each.steps {
			_, printed := run.request()
			_, called := each.calls[run.step]

			if printed && !called {
				unmatched = append(unmatched, each.steps[0].cue()+" "+run.step)
			}
		}
	}

	return unmatched
}

// sessionLimits are what a live session may spend, and for how long it may run.
type sessionLimits struct {
	budgetUSD string
	timeout   time.Duration
}

// stepsWrapper is the script that keeps what each run of a step of the flows was given and printed. A run's files carry
// the process ID and the second it began, so a process ID that comes round again does not overwrite an earlier run.
func stepsWrapper(dir, engram string) string {
	return "#!/bin/sh\ncase \"$1 $2\" in\n\"auto deliver\"|\"auto write\"|\"auto moment\"|\"auto phrases\"|" +
		"\"auto recall\"|\"auto judge\"|\"auto place\")\nf='" + dir + "/'\"$2\"-$$\ncat > \"$f.in\"\n'" + engram +
		"' \"$@\" < \"$f.in\" > \"$f.out\"\ncode=$?\n: > \"$f.done\"\ncat \"$f.out\"\ncp \"$f.out\" '" + dir +
		"/'\"$2\".out\ncp \"$f.in\" '" + dir + "/'\"$2\".in\nexit $code\n;;\nesac\nexec '" + engram + "' \"$@\"\n"
}
