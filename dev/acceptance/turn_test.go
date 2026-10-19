//go:build network

package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855a2e1a_ANoteArrivesBeforeTheAgentDeclaresItsWorkDone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat *conversation
		note string
	)

	step(t, "Given the user has run engram update, and a vault holding a note that says a release is done only when "+
		"CHANGELOG.md has the version's entry", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayRun("touch")
		note = machine.learnsFact(releaseChangelog)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		inSession := machine.startSession("a-project")
		inSession.hasFile("CHANGELOG.md", "# Changelog\n\n## v1.3.0\n\n- the staging reset moved\n")
		chat = inSession.converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to tag v1.4.0, the prompt arrives with nothing from the vault, "+
		"and the agent has tagged and is about to declare the release done", func() {
		sent = chat.tell(askTagRelease)

		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent declared the release done")
	})

	step(t, "Then before the agent declares it done its context holds the changelog note", func() {
		arrived, delivered := sent.arrivalOf(note)

		machine.logArrival(sent, arrived)

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			machine.whatWasJudged(sent))
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")
		g.Expect(arrived.block).To(BeNumerically("<", sent.finalAt()), "the block comes before the agent's last message")

		// the capture of the turn is over before the row returns
		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})
}

func Test855a2e1b_ANoteArrivesAfterAFailureTheAgentCannotExplain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat *conversation
		note string
	)

	step(t, "Given the user has run engram update, and a vault holding a note that says make ci fails with the "+
		"message LINT_STRICT is unset unless LINT_STRICT=1 is set", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayRun("make")
		note = machine.learnsFact(makeCIStrict)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		inSession := machine.startSession("a-project")
		inSession.hasFile("Makefile", "ci:\n\t@test \"$$LINT_STRICT\" = 1 || { echo \"make ci: LINT_STRICT is unset\" "+
			">&2; exit 1; }\n\t@echo ci passed\n")
		chat = inSession.converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to get the branch ready for review, the prompt arrives with "+
		"nothing from the vault, and the agent runs make ci, which fails with that message", func() {
		sent = chat.tell(askReadyForReview)

		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
	})

	step(t, "Then before the agent's next action its context holds the make ci note", func() {
		arrived, delivered := sent.arrivalOf(note)

		machine.logArrival(sent, arrived)

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			machine.whatWasJudged(sent))
		g.Expect(sent.resultTextAt(arrived.before)).To(ContainSubstring("LINT_STRICT"),
			"the block follows the failed command's result")
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})
}

func Test855a2e1c_ANoteArrivesBeforeTheAgentStartsBuildingANewApproach(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat *conversation
		note string
	)

	step(t, "Given the user has run engram update, and a vault holding a note that says schema changes are written "+
		"as migrations with the migrate tool and never by editing schema.sql", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayEditFiles()
		note = machine.learnsFact(schemaMigrations)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		inSession := machine.startSession("a-project")
		inSession.hasFile("schema.sql", "create table orders (id integer primary key, created_at text);\n")
		chat = inSession.converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to add an index on orders.created_at, the prompt arrives with "+
		"nothing from the vault, and the agent is about to edit schema.sql", func() {
		sent = chat.tell(askAddIndex)

		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
	})

	step(t, "Then before the agent's next action its context holds the migration note", func() {
		arrived, delivered := sent.arrivalOf(note)

		machine.logArrival(sent, arrived)

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			machine.whatWasJudged(sent))
		g.Expect(sent.resultTextAt(arrived.before)).NotTo(BeEmpty(), "the block follows a tool's result")
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})
}

// unexported constants.
const (
	// askAddIndex is a prompt whose work is an edit of the schema, which the schemaMigrations note forbids (855a2e1c).
	askAddIndex = "Add this line to the end of schema.sql: create index idx_orders_created_at on orders (created_at); " +
		"Reply with one short sentence."
	// askReadyForReview is a prompt of one command, make ci, which fails (855a2e1b).
	askReadyForReview = "Run make ci by itself, with no pipe or redirect, and tell me what happened. " +
		"Reply with one short sentence."
	// askTagRelease is a prompt of one command, after which the agent declares the work done (855a2e1a).
	askTagRelease = "Tag v1.4.0 by running touch tag-v1.4.0, then tell me the release is done. " +
		"Reply with one short sentence."
)

// unexported variables.
var (
	// makeCIStrict is a note that explains the failure of make ci (855a2e1b).
	makeCIStrict = fact{
		slug:      "zorblax-make-ci-strict",
		situation: "running make ci",
		subject:   "make ci",
		predicate: "fails with the message LINT_STRICT is unset unless",
		object:    "LINT_STRICT=1 is set",
	}
	// releaseChangelog is a note that bears on the moment a release is declared done (855a2e1a).
	releaseChangelog = fact{
		slug:      "zorblax-release-changelog",
		situation: "declaring a release done",
		subject:   "a release",
		predicate: "is done only when",
		object:    "CHANGELOG.md has the version's entry",
	}
	// schemaMigrations is a note that bears on the moment the agent starts to edit the schema (855a2e1c).
	schemaMigrations = fact{
		slug:      "zorblax-schema-migrations",
		situation: "changing the database schema",
		subject:   "schema changes",
		predicate: "are written as migrations with the migrate tool and",
		object:    "never by editing schema.sql",
	}
)

// cueTime is how long a cue's chain took.
type cueTime struct {
	cue  string
	took time.Duration
}

// agentMayEditFiles gives the agent the permission to read and edit files, and not to run engram.
func (m *machine) agentMayEditFiles() {
	m.t.Helper()

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), `{"permissions":{"allow":["Read","Edit","Write"]}}`)
}

// cueTimes returns, for each delivery of the machine, the time from the start of the moment of its cue, the last one of
// the cue that began before it, to the delivery's output.
func (m *machine) cueTimes() []cueTime {
	moments, _ := filepath.Glob(filepath.Join(m.root, "steps-seen", "moment-*.in"))
	delivered, _ := filepath.Glob(filepath.Join(m.root, "steps-seen", "deliver-*.out"))

	var times []cueTime

	for _, out := range delivered {
		cue, delivery := stepCue(strings.TrimSuffix(out, ".out") + ".in")
		finished, statErr := os.Stat(out)

		var began time.Time

		for _, moment := range moments {
			if momentCue, at := stepCue(moment); momentCue == cue && !at.After(delivery) && at.After(began) {
				began = at
			}
		}

		if statErr == nil && !began.IsZero() {
			times = append(times, cueTime{cue: cue, took: finished.ModTime().Sub(began)})
		}
	}

	return times
}

// logArrival logs the block that arrived within the turn, the row that records it, and how long after its moment began
// the cue's delivery printed it (855a2e).
func (m *machine) logArrival(sent *turn, arrived arrival) {
	m.t.Helper()

	row := sent.rows[arrived.block]

	m.t.Logf("the block arrived as row %d of the turn, after row %d, as the attachment of type %s from %s, event %s: %s",
		arrived.block, arrived.before, row.Attachment.Type, row.Attachment.HookName, row.Attachment.HookEvent,
		row.Attachment.Content)

	for _, took := range m.cueTimes() {
		m.t.Logf("the cue %s: %s from the start of its moment to the delivery's output", took.cue, took.took)
	}
}

// requestCosts returns the cost of the requests that the steps of the machine printed, by the cue of their carry.
func (m *machine) requestCosts() map[string]requestCost {
	costs := map[string]requestCost{}
	outs, _ := filepath.Glob(filepath.Join(m.root, "steps-seen", "*-*.out"))

	for _, name := range outs {
		data, readErr := os.ReadFile(name)

		var step struct {
			Request struct {
				System string `json:"system"`
				Prompt string `json:"prompt"`
			} `json:"request"`
			Carry struct {
				Cue string `json:"cue"`
			} `json:"carry"`
		}

		if readErr != nil || json.Unmarshal(data, &step) != nil || step.Request.Prompt == "" {
			continue
		}

		cost := costs[step.Carry.Cue]
		cost.calls++
		cost.chars += len(step.Request.System) + len(step.Request.Prompt)
		costs[step.Carry.Cue] = cost
	}

	return costs
}

// whatWasJudged tells, for the failure of a row, the blocks delivered within the turn, what each step was given and the
// hooks the host skipped.
func (m *machine) whatWasJudged(sent *turn) string {
	var told strings.Builder

	told.WriteString(strings.Join(sent.withinTurn(), "\n---\n"))

	for _, step := range []string{"moment", "phrases", "recall", "deliver"} {
		told.WriteString("\nthe input of " + step + ":\n" + m.stepsGiven(step))
	}

	told.WriteString("\nthe hooks skipped:\n" + strings.Join(m.hookFailures(), "\n"))

	return told.String()
}

// requestCost is the model calls the machine's steps asked for in a session and the size of their requests, by cue:
// the steps print one request for each call (855d2a), and the size is the characters of its system and prompt.
type requestCost struct {
	calls int
	chars int
}

// stepCue returns the cue a step was given, from the file that kept its input, and when the file was written.
func stepCue(name string) (string, time.Time) {
	data, readErr := os.ReadFile(name)
	info, statErr := os.Stat(name)

	var input struct {
		Cue string `json:"cue"`
	}

	if readErr != nil || statErr != nil || json.Unmarshal(data, &input) != nil {
		return "", time.Time{}
	}

	return input.Cue, info.ModTime()
}
