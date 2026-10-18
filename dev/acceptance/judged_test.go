//go:build network

package acceptance_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855a3b1a_APromptThatIsNotARecallMomentArrivesWithNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a vault holding a note that says when the staging database "+
		"is reset", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.learnsFact(stagingReset)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		chat = machine.startSession("a-project").converse()
	})

	var sent *turn

	step(t, "When the user sends a prompt that says only thanks, that settles when the staging database is reset",
		func() {
			sent = chat.tell(askThanks)
		})

	step(t, "Then the prompt arrives with nothing from the vault, and the agent answers it", func() {
		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")

		// the chain ran and found no recall moment: the moment ran, and nothing failed
		_, ran := machine.stepPrinted("moment")

		g.Expect(ran).To(BeTrue(), "the moment ran")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)

		g.Expect(machine.recordKinds(chat.session.id)).NotTo(ContainElement("failed"), "what became of the turns")
	})
}

func Test855a3b1b_ANoteWhoseWordsMatchButThatDoesNotApplyIsNotDelivered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat *conversation
		note string
	)

	step(t, "Given the user has run engram update, and a vault holding a note that says when the staging database "+
		"is reset", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		note = machine.learnsFact(stagingReset)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		chat = machine.startSession("a-project").converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to write a function that parses the staging database "+
		"connection string from a config file", func() {
		sent = chat.tell(askParseConnection)
	})

	step(t, "Then the prompt arrives with nothing from the vault, and the agent answers it", func() {
		g.Expect(sent.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(sent.reply).NotTo(BeEmpty(), "the agent answered")

		// the recall found the note and the candidate judgement left it out: the moment and the recall ran, the recall
		// held the note as a candidate, and nothing failed
		_, moment := machine.stepPrinted("moment")
		recall, recalled := machine.stepPrinted("recall")

		g.Expect(moment).To(BeTrue(), "the moment ran")
		g.Expect(recalled).To(BeTrue(), "the recall ran")
		g.Expect(recall).To(ContainSubstring(`"id":"`+note+`"`), "the recall held the note as a candidate")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)

		g.Expect(machine.recordKinds(chat.session.id)).NotTo(ContainElement("failed"),
			machine.recordsText()+"\n"+machine.deliveryAsked())
	})
}

func Test855a3b1c_OfTwoNotesThatConflictOnlyTheNewerIsDelivered(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var older, newer string

	step(t, "Given the user has run engram update, and a vault holding an older note that says the staging database "+
		"is reset every Sunday night and a newer note, linked to it by nothing, that says it is reset every Monday "+
		"night and not on Sunday", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		older = machine.learnsFact(stagingReset)
		newer = machine.learnsFact(stagingMonday)
		machine.isDated(older, olderDay)
		machine.mustSucceed(machine.update(), "engram update")
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to prepare a test run against the staging database tonight",
		func() {
			sent = machine.startSession("a-project").send(askPrepareRun)
			t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
		})

	step(t, "Then before the agent acts its context holds the newer note and not the older", func() {
		blocks := sent.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")

		if len(blocks) == 1 {
			g.Expect(blocks[0]).To(ContainSubstring(newer))
			g.Expect(blocks[0]).NotTo(ContainSubstring(older))
		}

		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")
	})
}

func Test855a3b1d_TheNotesThatArriveAreTheNotesTheRecallSkillDraws(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var stagingNote, backupNote, formatNote string

	step(t, "Given the user has run engram update, and a vault holding a note that says when the staging database "+
		"is reset, a note that says to take a backup before a database migration, and a note about formatting "+
		"Python files", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayRunEngramAndSkills()
		stagingNote = machine.learnsFact(stagingReset)
		backupNote = machine.learnsFact(migrationBackup)
		formatNote = machine.learnsFeedback(formatPython)
		machine.mustSucceed(machine.update(), "engram update")
	})

	var arrived, named *turn

	step(t, "When the user asks the agent in a prompt to plan a migration of the staging database for tonight, and "+
		"in a second session on the same vault in which automatic memory is not on asks the agent to run the recall "+
		"skill in its glance mode for the same prompt and to name the notes it drew on", func() {
		arrived = machine.startSession("a-project").send(askPlanMigration)
		t.Logf("the agent replied %q (cost $%.4f)", arrived.reply, arrived.cost)

		machine.automaticMemoryIsOff()

		named = machine.startSession("another-project").send(askToRecallAtGlance)
		t.Logf("the agent replied %q (cost $%.4f)", named.reply, named.cost)
	})

	step(t, "Then the notes that arrived with the prompt in the first session are the staging note and the backup "+
		"note, and the notes that the agent names in the second are the same two", func() {
		blocks := arrived.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")

		if len(blocks) == 1 {
			g.Expect(blocks[0]).To(And(ContainSubstring(stagingNote), ContainSubstring(backupNote)))
			g.Expect(blocks[0]).NotTo(ContainSubstring(formatNote))
		}

		g.Expect(named.reply).To(And(ContainSubstring(stagingNote), ContainSubstring(backupNote)),
			"the notes the agent names")
		g.Expect(named.reply).NotTo(ContainSubstring(formatNote), "the notes the agent names")
		g.Expect(named.blocks()).To(BeEmpty(), "automatic memory is not on in the second session")
	})
}

func Test855a3b1e_APassageOfAnEarlierConversationArrivesWhenItApplies(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given the user has run engram update, a vault holding no note about the production cutover, and a "+
		"searchable earlier conversation in which the user said the production cutover is frozen on the last "+
		"Friday of every month", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.learnsFeedback(formatPython)
		machine.hasConversation(freezeConversation)
		machine.mustSucceed(machine.update(), "engram update")
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to plan the staging deploys of next month so that they avoid "+
		"the production cutover freeze", func() {
		sent = machine.startSession("a-project").send(askPlanFreeze)
		t.Logf("the agent replied %q (cost $%.4f)", sent.reply, sent.cost)
	})

	step(t, "Then before the agent acts its context holds the passage of the earlier conversation that states the "+
		"freeze, and the prompt in the conversation is exactly as the user typed it", func() {
		blocks := sent.blocks()

		g.Expect(blocks).To(HaveLen(1), "blocks automatic memory delivered with the prompt")

		if len(blocks) == 1 {
			g.Expect(blocks[0]).To(ContainSubstring("(passage)"))
			g.Expect(blocks[0]).To(ContainSubstring("last Friday of every month"))
		}

		g.Expect(sent.blockAt()).To(BeNumerically("<", sent.agentActsAt()), "the block comes before the agent acts")

		text, isPrompt := sent.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askPlanFreeze))
	})
}

func Test855a3c1a_AJudgementThatNeverAnswersIsSteppedAsideAfter2Minutes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, a vault holding a note that says when the staging database is "+
		"reset, and a judgement that never answers", func() {
		machine.hasRunUpdate()
		machine.learnsFact(stagingReset)
		machine.stepNeverEnds("moment")

		chat = machine.startSession("zorblax").converse()
	})

	var first, second *turn

	step(t, "When the user sends a prompt to prepare a test run against the staging database tonight", func() {
		first = chat.tell(askPrepareRun)
		t.Logf("the prompt reached the model %s after it was sent", first.reached())

		machine.awaitRecordOfKind(chat.session.id, chat.turnEnds, "failed")
		machine.stepIsQuick()

		second = chat.tell(askSomethingElse)
	})

	step(t, "Then the prompt reaches the agent within 3 minutes, as the user typed it and with nothing from the "+
		"vault, the agent answers it, and the session shows with the next prompt a line that says the delivery could "+
		"not be done, with the reason", func() {
		g.Expect(machine.stepStarted()).To(BeTrue(), "the judgement was started")
		g.Expect(first.reached()).To(BeNumerically("<", stalledLimit), "how long the prompt waited")
		g.Expect(first.reached()).To(BeNumerically(">=", stallBound), "the prompt waited for the bound")

		text, isPrompt := first.rows[0].promptText()
		g.Expect(isPrompt).To(BeTrue())
		g.Expect(text).To(Equal(askPrepareRun))

		g.Expect(first.blocks()).To(BeEmpty(), "blocks automatic memory delivered with the prompt")
		g.Expect(first.reply).NotTo(BeEmpty(), "the agent answered")

		t.Logf("the lines shown with the next prompt: %q", second.shown)

		g.Expect(second.shown).To(ConsistOf(And(ContainSubstring("failed:"),
			ContainSubstring("the delivery for a prompt could not be done"), ContainSubstring("2 minutes"))),
			"the lines shown with the next prompt")
	})
}

// unexported constants.
const (
	// askToRecallAtGlance asks, in a session without automatic memory, for the recall skill at its glance mode for the
	// prompt of askPlanMigration, and for the notes it drew on by their file names.
	askToRecallAtGlance = "Run the recall skill in its glance mode for this prompt: \"" + askPlanMigration + "\" " +
		"Then reply with the file names of the notes the skill drew on, one to a line, and nothing else."
	// olderDay is the day the older of two conflicting notes is dated, long before the newer one.
	olderDay = "2026-01-05"
	// stallBound is the bound of a wait of the prompt's flow (855d2a).
	stallBound = 2 * time.Minute
	// stalledLimit is how long a prompt may wait when one judgement never answers (855a3c1a).
	stalledLimit = 3 * time.Minute
)

// agentMayRunEngramAndSkills lets the agent run engram and use a skill, which the recall skill needs.
func (m *machine) agentMayRunEngramAndSkills() {
	m.t.Helper()

	writeFile(m.t, filepath.Join(m.claudeDir, "settings.json"),
		`{"permissions":{"allow":["Bash(engram:*)","Skill"]}}`)
}

// automaticMemoryIsOff takes the link to the mod away, so that the next session does not load it.
func (m *machine) automaticMemoryIsOff() {
	m.t.Helper()

	err := os.Remove(filepath.Join(m.claudeDir, "skills", modName))
	if err != nil {
		m.t.Fatalf("taking the mod away: %v", err)
	}
}

// isDated sets the day a note says it was created, so that one of two notes is older than the other.
func (m *machine) isDated(note, day string) {
	m.t.Helper()

	path := filepath.Join(m.vault, note)

	data, err := os.ReadFile(path)
	if err != nil {
		m.t.Fatalf("reading %s: %v", note, err)
	}

	pattern := regexp.MustCompile(`(?m)^created: .*$`)
	if !pattern.Match(data) {
		m.t.Fatalf("%s has no created field:\n%s", note, data)
	}

	writeFile(m.t, path, string(pattern.ReplaceAll(data, []byte("created: "+day))))
}
