package auto

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// unexported constants.
const (
	maxOffers = 10
	nearCount = 5
	showParts = 3
	skillKind = "runbook"
)

// unexported variables.
var (
	forLine  = regexp.MustCompile(`(?m)^ {2,}for: (\S+)`)
	hashLine = regexp.MustCompile(`(?m)^# exchange_hash: (\S+)`)
	// a note's file name as engram writes it; a name that could be read as a flag is never passed to a command
	noteName    = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._-]*\.md$`)
	pendingLine = regexp.MustCompile(`(?m)^pending: true$`)
	skillHash   = regexp.MustCompile(`(?m)^skill_hash:`)
)

// curate settles the pending offers, at most maxOffers a run; an offer that is skipped or fails is not counted again
// and not looked at again.
func (e *engine) curate() {
	vault := e.deps.VaultPath
	if !e.deps.Files.Exists(vault) {
		return
	}

	handled := 0

	for _, name := range e.pendingOffers(vault) {
		if handled >= maxOffers {
			break
		}

		if slices.Contains(e.work.Skipped, name) {
			continue
		}

		// one offer that fails is warned about once and does not stop the offers after it
		var outcome string

		e.attempt("curation", func() { outcome = e.curateOne(name) })

		if outcome != "skipped" {
			handled++
		}

		if outcome != "settled" {
			e.work.Skipped = append(e.work.Skipped, name)
		}
	}
}

// curateOne settles one pending offer as curate does (793b201): read it, judge it against the nearest notes, run the
// commands of the outcome. A skill registration (a runbook marked by its skill hash) is authored, never judged, so it
// is left pending for curate; a note that is not pending (settled meanwhile) is left alone.
func (e *engine) curateOne(name string) string {
	offer := parseShow(e.exec([]string{engramCmd, "show", name}, deadlineCommand))
	if !offer.pending || offer.skill || offer.Kind == skillKind {
		return "skipped"
	}

	text := Note{Situation: redact(offer.Situation), Body: redact(offer.Body)}
	near := e.nearest(offer, text)
	parse := func(reply string) Decision { return parseCuration(reply, near) }
	decision := judged(e, e.tiers.curationRequest(text, near), judgeCuration, parse)
	e.logf("curation judged %s %s %s against %d nearest note(s)", name, decision.Outcome, decision.Target, len(near))

	for _, argv := range settleArgvs(decision, name, offer.hash) {
		e.exec(argv, deadlineCommand)
	}

	// a reinforcing activation (covered) is not an amendment: only a near offer changes a note's claim (793b204)
	e.count(Counts{Curated: 1, Amended: map[bool]int{true: 1}[decision.Outcome == outcomeNear]})

	return "settled"
}

// housekeepingJob is the work that follows the last job of a queue (793b202, 793b201): the chunk index catches up with
// the turn, then the pending offers are settled, then, after a top-level turn, one source of the backlog is captured
// (793e8g). A failure of one step is reported and the others still run.
func (e *engine) housekeepingJob() Done {
	e.attempt("indexing", func() { e.exec([]string{engramCmd, "ingest", "--auto"}, deadlineIngest) })
	e.attempt("curation", e.curate)

	if e.job.TurnEnded {
		e.attempt("the backlog capture", e.backlog)
	}

	return Done{}
}

// nearest is the existing notes an offer is judged against: for an offer that amends a note, that note; otherwise the
// vault's nearest notes by `engram query`, which leaves pending offers out. Everything the judge reads is redacted.
func (e *engine) nearest(offer shown, text Note) []Note {
	if offer.forNote != "" {
		target := parseShow(e.exec([]string{engramCmd, "show", offer.forNote}, deadlineCommand))

		return []Note{redacted2(offer.forNote, target.Note)}
	}

	query := []string{
		engramCmd, "query", "--limit", strconv.Itoa(nearCount), "--text=" + cutEnd(text.Situation+". "+text.Body, textLimit),
		"--phrase=" + cutEnd(text.Situation, textLimit),
	}
	notes := []Note{}

	for _, found := range parseQuery(e.exec(query, deadlineNearest)) {
		if found.Kind != chunkKind && found.Situation != "" && len(notes) < nearCount {
			notes = append(notes, redacted2(found.Path, found))
		}
	}

	return notes
}

// pendingOffers lists the notes marked pending, the way the pending-offer bookkeeping finds them (the query excludes
// them by design).
func (e *engine) pendingOffers(vault string) []string {
	listed := must(e.step(func() (string, error) {
		names, err := e.deps.Offers.Pending(vault)

		return strings.Join(names, "\n"), err
	}))

	return slices.DeleteFunc(strings.Split(listed, "\n"), func(name string) bool {
		return !noteName.MatchString(name) || name == "README.md"
	})
}

// shown is an offer as `engram show` prints it: the exchange hash of the version shown (absent for a note with no
// xid), the front matter fields curation needs, and the note's text.
type shown struct {
	Note

	hash, forNote  string
	pending, skill bool
}

// parseShow reads `engram show`.
func parseShow(output string) shown {
	parts := append(ruleLine.Split(output, showParts), "", "")
	front := parts[1]
	field := func(name string) string {
		value := strings.TrimSpace(first(regexp.MustCompile(`(?m)^`+name+`: (.*)$`), front))

		return strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(value, `"`), "'"), `"`), "'")
	}

	body, _, _ := strings.Cut(parts[2], "\n# outbound links")
	forNote := first(forLine, front)

	return shown{
		Note: Note{Kind: field("type"), Situation: field("situation"), Body: oneSpaced(body)},
		hash: first(hashLine, parts[0]), pending: pendingLine.MatchString(front), skill: skillHash.MatchString(front),
		forNote: map[bool]string{true: forNote}[noteName.MatchString(forNote)],
	}
}

// redacted2 is a note as the judge reads it: every field redacted.
func redacted2(path string, note Note) Note {
	return Note{Path: path, Kind: note.Kind, Situation: redact(note.Situation), Body: redact(note.Body)}
}

// settleArgvs are the commands of an outcome, in curate's order: a covered offer reinforces the note and is folded
// into it (`--discard --into` keeps the offer's identity); a near offer is folded away first and its new claim then
// written into the note; an absent one clears its own marker; a rejected one is discarded bare, never `--into`. A
// served offer's exchange hash is passed, as curate does, so that an offer changed since it was read is not settled.
func settleArgvs(decision Decision, offer, hash string) [][]string {
	var guard []string
	if hash != "" {
		guard = []string{"--expect-hash=" + hash}
	}

	amend := func(target string, rest ...string) []string {
		return slices.Concat([]string{engramCmd, amendCmd, targetFlag, target}, rest, guard)
	}
	fold := amend(offer, "--discard", "--into", decision.Target)

	switch decision.Outcome {
	case outcomeCovered:
		return [][]string{{"engram", "amend", "--target", decision.Target, "--activate"}, fold}
	case outcomeNear:
		set := []string{engramCmd, amendCmd, targetFlag, decision.Target}
		for _, field := range decision.Fields {
			set = append(set, "--"+field[0]+"="+field[1])
		}

		set = append(set, "--source=amended automatically from offer "+offer)

		return [][]string{fold, set}
	case outcomeAbsent:
		return [][]string{amend(offer, "--clear-pending")}
	}

	return [][]string{amend(offer, "--discard")}
}
