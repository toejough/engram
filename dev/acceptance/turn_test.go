//go:build network

package acceptance_test

import (
	"path/filepath"
	"strings"
	"testing"

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

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			strings.Join(sent.withinTurn(), "\n---\n"))
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")
		g.Expect(arrived.next).To(Equal(sent.finalAt()), "the agent's next message is the one that declares it done")
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

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			strings.Join(sent.withinTurn(), "\n---\n"))
		g.Expect(sent.resultTextAt(arrived.before)).To(ContainSubstring("LINT_STRICT"),
			"the block follows the failed command's result")
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")
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

		g.Expect(delivered).To(BeTrue(), "a block that names the note was delivered within the turn: "+
			strings.Join(sent.withinTurn(), "\n---\n"))
		g.Expect(sent.resultTextAt(arrived.before)).NotTo(BeEmpty(), "the block follows a tool's result")
		g.Expect(arrived.next).To(BeNumerically(">", arrived.block), "the agent's next message follows the block")
	})
}

// unexported constants.
const (
	// askAddIndex is a prompt whose work is an edit of the schema, which the schemaMigrations note forbids (855a2e1c).
	askAddIndex = "Add this line to the end of schema.sql: create index idx_orders_created_at on orders (created_at); " +
		"Reply with one short sentence."
	// askReadyForReview is a prompt of one command, make ci, which fails (855a2e1b).
	askReadyForReview = "Run make ci and tell me what happened. Reply with one short sentence."
	// askTagRelease is a prompt of one command, after which the agent declares the work done (855a2e1a).
	askTagRelease = "Tag v1.4.0 by running touch tag-v1.4.0. Reply with one short sentence."
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

// agentMayEditFiles gives the agent the permission to read and edit files, and not to run engram.
func (m *machine) agentMayEditFiles() {
	m.t.Helper()

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"), `{"permissions":{"allow":["Read","Edit","Write"]}}`)
}
