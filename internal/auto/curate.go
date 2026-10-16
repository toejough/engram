package auto

import (
	"regexp"
	"slices"
	"strings"
)

// unexported constants.
const (
	hashHeader = "# exchange_hash: "
	maxOffers  = 10
	nearCount  = 5
	showParts  = 3
	skillKind  = "runbook"
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

		e.attempt(stepCuration, func() { outcome = e.curateOne(name) })

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
	offer := parseShow(e.show(name))
	if !offer.pending || offer.skill || offer.Kind == skillKind {
		return "skipped"
	}

	text := Note{Situation: redact(offer.Situation), Body: redact(offer.Body)}
	near := e.nearest(offer, text)
	parse := func(reply string) Decision { return parseCuration(reply, near) }
	decision := judged(e, e.tiers.curationRequest(text, near), judgeCuration, parse)
	e.logf("curation judged %s %s %s against %d nearest note(s)", name, decision.Outcome, decision.Target, len(near))

	for _, request := range settleRequests(decision, name, offer.hash, near) {
		e.amend(request)
	}

	// a reinforcing activation (covered) is not an amendment: only a near offer changes a note's claim (793b204)
	e.count(Counts{Curated: 1, Amended: map[bool]int{true: 1}[decision.Outcome == outcomeNear]})

	return "settled"
}

// housekeepingJob is the work that follows the last job of a queue (793b202, 793b201): the chunk index catches up with
// the turn, then the pending offers are settled, then, after a top-level turn, one source of the backlog is captured
// (793e8g). A failure of one step is reported and the others still run. The mod runs the three as jobs of their own,
// each named by its step, so that a reply it cannot read is blamed on one step (`part`, 793b469); a job with no part
// does all of them.
func (e *engine) housekeepingJob() Done {
	part := e.job.Part

	check(slices.Contains([]string{"", stepIndexing, stepCuration, stepBacklog}, part),
		"the part of the housekeeping is not indexing, curation or backlog capture")

	if part == "" || part == stepIndexing {
		e.attempt(stepIndexing, func() {
			e.perform("ingest", func() (string, error) { return "", e.deps.Vault.Index(deadlineIngest) })
		})
	}

	if part == "" || part == stepCuration {
		e.attempt(stepCuration, e.curate)
	}

	if e.job.TurnEnded && (part == "" || part == stepBacklog) {
		e.attempt(stepBacklog, e.backlog)
	}

	return Done{}
}

// nearest is the existing notes an offer is judged against: for an offer that amends a note, that note; otherwise the
// vault's nearest notes by `engram query`, which leaves pending offers out. Everything the judge reads is redacted.
func (e *engine) nearest(offer shown, text Note) []Note {
	if offer.forNote != "" {
		target := parseShow(e.show(offer.forNote))

		return []Note{redacted2(offer.forNote, target.Note)}
	}

	query := QueryRequest{
		Text: cutEnd(text.Situation+". "+text.Body, textLimit), Phrases: []string{cutEnd(text.Situation, textLimit)},
		Limit: nearCount,
	}
	notes := []Note{}

	for _, found := range parseQuery(e.query(query, deadlineNearest)) {
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

	// the note as it was printed: after the exchange hash line, before the list of its links
	printed := output
	if strings.HasPrefix(printed, hashHeader) {
		_, printed, _ = strings.Cut(printed, "\n")
	}

	read, _, _ := strings.Cut(printed, "\n# outbound links")

	return shown{
		Note: Note{Kind: field("type"), Situation: field("situation"), Body: oneSpaced(body), raw: read},
		hash: first(hashLine, parts[0]), pending: pendingLine.MatchString(front), skill: skillHash.MatchString(front),
		forNote: map[bool]string{true: forNote}[noteName.MatchString(forNote)],
	}
}

// redacted2 is a note as the judge reads it: every field redacted.
func redacted2(path string, note Note) Note {
	return Note{Path: path, Kind: note.Kind, Situation: redact(note.Situation), Body: redact(note.Body), raw: note.raw}
}

// settleRequests are the amends of an outcome, in curate's order: a covered offer reinforces the note and is folded
// into it (`--discard --into` keeps the offer's identity); a near offer amends the note first, with the claim and the
// line that says where it came from, checked against the note as it was read so that a note changed since is not
// written, and is folded away after it (793b420); an absent one clears its own marker; a rejected one is discarded
// bare, never `--into`. A served offer's exchange hash is passed, as curate does, so that an offer changed since it was
// read is not settled.
func settleRequests(decision Decision, offer, hash string, near []Note) []AmendRequest {
	fold := AmendRequest{Target: offer, Discard: true, Into: decision.Target, ExpectHash: hash}

	switch decision.Outcome {
	case outcomeCovered:
		return []AmendRequest{{Target: decision.Target, Activate: true}, fold}
	case outcomeNear:
		read := ""

		if at := slices.IndexFunc(near, func(note Note) bool { return note.Path == decision.Target }); at >= 0 {
			read = near[at].raw
		}

		return []AmendRequest{{
			Target: decision.Target, Fields: decision.Fields, ExpectNote: read,
			Provenance: "amended automatically from offer " + offer,
		}, fold}
	case outcomeAbsent:
		return []AmendRequest{{Target: offer, ClearPending: true, ExpectHash: hash}}
	}

	return []AmendRequest{{Target: offer, Discard: true, ExpectHash: hash}}
}
