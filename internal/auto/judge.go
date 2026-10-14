package auto

import (
	_ "embed" // the prompts are text files, measured as they were
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Cue is a moment the classification named and what it is about.
type Cue struct{ Moment, Subject string }

// Decision is what the curation judgement decided for one offer.
type Decision struct {
	Outcome, Target string
	Fields          [][2]string
}

// Request is one model call the mod relays with exactly these fields.
type Request struct {
	Model     string `json:"model"`
	Effort    string `json:"effort"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"maxTokens"`
	TimeoutMs int    `json:"timeoutMs"`
}

// Tiers maps each kind of judgement to the model that answers it; a kind without a row is an error, never a default
// (793d25b1). Every judgement runs at the lowest accepted effort (793c79).
type Tiers map[string]string

func (t Tiers) captureRequest(window string, near []Note) Request {
	notes := fence(numbered(near, false))

	return t.request(judgeCapture, fill(capturePrompt, "{window}", fence(window), "{notes}", notes))
}

func (t Tiers) classifyRequest(recent string) Request {
	return t.request(judgeClassify, fill(classifyPrompt, "{recent}", fence(recent)))
}

func (t Tiers) curationRequest(offer Note, near []Note) Request {
	offered := fence("situation: " + offer.Situation + ". " + offer.Body)

	return t.request(judgeCuration, fill(curationPrompt, "{offer}", offered, "{notes}", fence(numbered(near, true))))
}

func (t Tiers) relevanceRequest(moment, text string, candidates []Note) Request {
	return t.request(judgeRelevance, fill(relevancePrompt,
		"{moment}", moment, "{text}", cutEnd(text, relevanceLimit), "{notes}", numbered(candidates, false)))
}

// request builds the call of a kind from its prompt, with the reply tokens and the deadline of its kind. The
// relevance judgement holds the agent's prompt: 5.8 s here and 2 s for the lookup are the 8 s hold (793d36b3). The
// judgement's deadline sits above its measured 95th percentile of 3.44 s (1.7 times), the lookup's above its 0.59 s
// with a cold embedder cache (3.4 times).
func (t Tiers) request(kind, prompt string) Request {
	limits := map[string]struct{ tokens, timeout int }{
		judgeClassify: {300, 4000}, judgeRelevance: {200, 5800}, judgeCapture: {1500, 40000}, judgeCuration: {700, 60000},
	}[kind]
	check(t[kind] != "", fmt.Sprintf("%v %q", errNoTier, kind))

	return Request{Model: t[kind], Effort: lowEffort, Prompt: prompt, MaxTokens: limits.tokens, TimeoutMs: limits.timeout}
}

// DefaultTiers is the tier table of the System: the cheapest capable model for every judgement (793d18, H2).
func DefaultTiers() Tiers {
	return Tiers{judgeCapture: cheapest, judgeClassify: cheapest, judgeCuration: cheapest, judgeRelevance: cheapest}
}

// unexported constants.
const (
	cheapest       = "haiku"
	jsonSpace      = " \t\r\n"
	judgeCapture   = "capture"
	judgeClassify  = "classify"
	judgeCuration  = "curation"
	judgeRelevance = "relevance"
	lowEffort      = "low"
	outcomeAbsent  = "absent"
	outcomeCovered = "covered"
	outcomeNear    = "near"
	outcomeRefused = "rejected"
	relevanceLimit = 1500
	stringLimit    = 600
)

//go:embed prompts/capture.txt
var capturePrompt string

//go:embed prompts/classify.txt
var classifyPrompt string

//go:embed prompts/curation.txt
var curationPrompt string

//go:embed prompts/relevance.txt
var relevancePrompt string

// unexported variables.
var (
	errJobKind  = errors.New("unknown job kind")
	errNoTier   = errors.New("no judgement tier for kind")
	errReply    = errors.New("judgement reply rejected")
	fencedReply = regexp.MustCompile("^```(?:json)?[ \\t]*\\r?\\n((?s:.*?))\\r?\\n```$")
	// a tag of a prompt as a reader would take it: a bracket of any shape, then white space, format and control
	// characters around the slash, then the name in any case (793b397)
	tagPattern = regexp.MustCompile(
		`(?i)[<＜﹤‹〈⟨]([` + space + `\p{Cf}\p{Cc}]*/?[` + space + `\p{Cf}\p{Cc}]*(?:window|notes|recent|offer)\b)`)
)

// reply is the answer to one request, as the model call returned it.
type reply struct {
	IsAnswered bool   `json:"isAnswered"`
	Text       string `json:"text"`
	Reason     string `json:"reason"`
}

// check fails the step with a reply that is not what was asked for (793b299), unless the condition holds.
func check(condition bool, what string) {
	if !condition {
		panic(fmt.Errorf("%w: %s", errReply, what))
	}
}

// distinctKeys says whether no object of a valid JSON document, at any depth, has the same key twice.
func distinctKeys(decoder *json.Decoder) bool {
	token, _ := decoder.Token()
	delim, nested := token.(json.Delim)
	seen := map[string]bool{}

	for nested && decoder.More() {
		if delim == '{' {
			key, _ := decoder.Token()
			name, _ := key.(string)

			if seen[name] {
				return false
			}

			seen[name] = true
		}

		if !distinctKeys(decoder) {
			return false
		}
	}

	if nested {
		_, _ = decoder.Token()
	}

	return true
}

// document reads a reply strictly (793b299): the reply is exactly one JSON document. A code fence that opens at the
// first character and closes at the last one is the only wrapper taken off; prose around the document, a second
// document, or anything else is an error, never a best guess, so that no text the parser skips can differ from the
// text a reader sees. Only the white space of JSON is trimmed (a no-break or a line-separator space is not), the text
// is valid UTF-8, and no object has a key twice (a reader that keeps the first of two would see another reply than
// the parser that keeps the last, 793b397).
func document(text string) any {
	body := strings.Trim(text, jsonSpace)
	if found := fencedReply.FindStringSubmatch(body); found != nil {
		body = strings.Trim(found[1], jsonSpace)
	}

	var value any

	one := body != "" && utf8.ValidString(body) && json.Unmarshal([]byte(body), &value) == nil
	check(one && distinctKeys(json.NewDecoder(strings.NewReader(body))), "the reply is not exactly one JSON document")

	return value
}

// fence writes `<` before the name of a tag of a prompt as an entity, so that a text cannot close or open one.
func fence(text string) string { return tagPattern.ReplaceAllString(text, "&lt;${1}") }

// fill puts the values in the places the prompt names, in one pass over the prompt: a value is data, and a placeholder
// inside it is not filled (a later placeholder would otherwise be found in an earlier value). A text that sits in a
// tag of the prompt cannot close or open the tags: the bracket of a window, notes, recent or offer tag (in any case,
// with any spacing, format or control characters in it, whatever the bracket) is written as an entity (`fence`).
func fill(prompt string, pairs ...string) string {
	return strings.NewReplacer(pairs...).Replace(prompt)
}

// judged has the mod relay a request to the model, and reads the reply it brings with the parser of its kind.
func judged[T any](e *engine, request Request, what string, parse func(string) T) T {
	answer := e.ask(request)
	if !answer.IsAnswered {
		panic(&StepError{Message: what + " judgement unanswered: " + answer.Reason, Kind: failureKind(answer.Reason)})
	}

	return parse(answer.Text)
}

// list reads a reply that is an array of at most `most` entries.
func list(text string, most int) []any {
	entries, ok := document(text).([]any)
	check(ok && len(entries) <= most, "the reply is not an array of at most "+strconv.Itoa(most))

	return entries
}

// noteFields are the free-text fields of the two kinds the System writes, in `engram learn`'s and `engram amend`'s
// names.
func noteFields(kind string) []string {
	return map[string][]string{
		"feedback": {"behavior", "impact", "action"}, "fact": {"subject", "predicate", "object"},
	}[kind]
}

// numbered lists notes as the prompts show them.
func numbered(notes []Note, withKind bool) string {
	lines := make([]string, len(notes))
	for index, note := range notes {
		kind := ""
		if withKind {
			kind = "kind: " + note.Kind + ". "
		}

		lines[index] = fmt.Sprintf("%d. %ssituation: %s. %s", index+1, kind, note.Situation, note.Body)
	}

	return orElse(strings.Join(lines, "\n"), "none")
}

// object reads a value that is an object with exactly these keys.
func object(value any, keys ...string) map[string]any {
	fields, ok := value.(map[string]any)
	missing := func(key string) bool { _, found := fields[key]; return !found }

	check(ok && len(fields) == len(keys) && !slices.ContainsFunc(keys, missing), "an object has not exactly its keys")

	return fields
}

// parseCuration reads the curation reply strictly (793b299): exactly one object with exactly the keys outcome, into
// and fields. covered and near name one of the nearest notes (`into`, its number); near also gives that note's
// fields, exactly those of its kind; absent and rejected give neither. Anything else is an error and nothing is
// settled. The redaction is applied to every field the judge wrote.
func parseCuration(text string, near []Note) Decision {
	fields := object(document(text), "fields", "into", "outcome")
	outcome, _ := fields["outcome"].(string)
	decision := Decision{Outcome: outcome}
	number, isNumber := whole(fields["into"])

	switch outcome {
	case outcomeAbsent, outcomeRefused:
		check(fields["into"] == nil && fields["fields"] == nil, outcome+" gives no note and no fields")
	case outcomeCovered, outcomeNear:
		check(isNumber && number >= 1 && number <= len(near), "into is no nearest note")

		target := near[number-1]
		decision.Target = target.Path

		if outcome == outcomeNear {
			names := noteFields(target.Kind)
			check(names != nil, "a note of this kind cannot take a near offer")

			set := object(fields["fields"], names...)
			for _, name := range names {
				decision.Fields = append(decision.Fields, [2]string{name, redact(short(set[name]))})
			}
		} else {
			check(fields["fields"] == nil, "covered gives no fields")
		}
	default:
		check(false, "unknown outcome")
	}

	return decision
}

// parseMoments reads the classification strictly (793b299): an array of at most one object per moment, each with
// exactly the keys moment (one of recall's four) and subject (a short string); anything else is an error and names no
// moment. The redaction is applied to every subject.
func parseMoments(text string) []Cue {
	entries := list(text, len(recallMoments()))
	cues := make([]Cue, 0, len(entries))

	for _, entry := range entries {
		fields := object(entry, "moment", "subject")
		moment, _ := fields["moment"].(string)
		repeated := slices.ContainsFunc(cues, func(cue Cue) bool { return cue.Moment == moment })
		check(slices.Contains(recallMoments(), moment) && !repeated, "a moment is none of the four or is repeated")

		cues = append(cues, Cue{Moment: moment, Subject: redact(short(fields["subject"]))})
	}

	return cues
}

// parseSelection reads the relevance reply: an array of distinct whole numbers, each the number of a candidate. Any
// other reply, or a number outside the candidates, is an error and nothing is delivered. It returns zero-based
// indexes.
func parseSelection(text string, count int) []int {
	picked := make([]int, 0, count)

	for _, value := range list(text, count) {
		number, ok := whole(value)
		valid := ok && number >= 1 && number <= count && !slices.Contains(picked, number-1)
		check(valid, "a number is no candidate or repeats one")

		picked = append(picked, number-1)
	}

	return picked
}

// recallMoments are the four decision moments of recall (792a19a), the phrases a prompt is looked up with.
func recallMoments() []string {
	return []string{
		"endorsing or ranking a proposed approach", "declaring work done", "a failure that cannot be explained",
		"starting to build a new approach",
	}
}

// short is a field of a judge's reply: a non-empty string of at most 600 characters.
func short(value any) string {
	text, ok := value.(string)
	check(ok && strings.TrimSpace(text) != "" && len([]rune(text)) <= stringLimit, "a field is not a short string")

	return strings.TrimSpace(text)
}

// whole reads a JSON number that is a whole number.
func whole(value any) (int, bool) {
	number, ok := value.(float64)

	return int(number), ok && number == math.Trunc(number) && math.Abs(number) < math.MaxInt32
}
