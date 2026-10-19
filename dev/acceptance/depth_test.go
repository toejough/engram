//go:build network

package acceptance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855a2a1a_AGlanceLeavesTheVaultsNotesAsThey(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat   *conversation
		before map[string]string
	)

	step(t, "Given the user has run engram update, and a vault holding a note that says when the staging database "+
		"is reset and a note about formatting Python files", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.learnsFact(stagingReset)
		machine.learnsFeedback(formatPython)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		before = machine.noteTexts()
		chat = machine.startSession("a-project").converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to prepare a test run against the staging database tonight, "+
		"and the agent answers", func() {
		sent = chat.tell(askPrepareRun)

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)
	})

	step(t, "Then the staging note was in the agent's context, no note was added to the vault, and every note "+
		"states what it stated before", func() {
		g.Expect(strings.Join(sent.blocks(), "\n")).To(ContainSubstring(machine.noteNamed(stagingReset.slug)),
			"the blocks delivered with the prompt")
		g.Expect(machine.stepsGiven("write")).NotTo(ContainSubstring(`"cue":"prompt"`), "the writing of a deep delivery")
		g.Expect(machine.noteTexts()).To(Equal(before), "the notes of the vault")
	})
}

func Test855a2a1b_ADeepDeliveryWritesTheLessonTheVaultLacks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, a vault holding no note about the production cutover, and a "+
		"searchable earlier conversation in which the user said the production cutover is frozen on the last "+
		"Friday of every month", func() {
		machine.hasEngram()
		machine.hasClaudeCode()
		machine.agentMayNotRunEngram()
		machine.learnsFeedback(formatPython)
		machine.hasConversation(freezeConversation)
		machine.stepsAreRecorded()
		machine.mustSucceed(machine.update(), "engram update")

		chat = machine.startSession("a-project").converse()
	})

	var sent *turn

	step(t, "When the user asks the agent in a prompt to plan the irreversible production cutover for the last "+
		"Friday of next month, and the agent answers", func() {
		sent = chat.tell(askPlanCutover)
	})

	step(t, "Then the passage of the earlier conversation was in the agent's context, and the vault holds a note "+
		"that states the freeze and says it was written automatically", func() {
		g.Expect(strings.Join(sent.blocks(), "\n")).To(ContainSubstring("(passage)"), "the blocks delivered with the prompt")

		name := machine.awaitAutomaticNote(chat.turnEnds, "that states the production cutover freeze", "cutover", "friday")

		t.Logf("the note was written %s after the block was printed", machine.noteWrittenAfterBlock(name))

		g.Expect(machine.stepsGiven("write")).To(ContainSubstring(`"cue":"prompt"`), "the writing of a deep delivery")
	})
}

// unexported constants.
const (
	// askPlanCutover is a prompt of a decision that cannot be undone, which the passage of freezeConversation applies
	// to (855a2a1b).
	askPlanCutover = "Plan the irreversible production cutover for the last Friday of next month. " +
		"Reply with one short sentence."
)

// noteTexts reads the text of each note of the machine's vault, by its file name.
func (m *machine) noteTexts() map[string]string {
	texts := map[string]string{}

	for _, note := range m.notes() {
		texts[note.name] = note.text
	}

	return texts
}

// noteWrittenAfterBlock returns how long after the delivery printed its block the note was written: the delivery's
// output file is kept when it exits.
func (m *machine) noteWrittenAfterBlock(name string) time.Duration {
	m.t.Helper()

	block, blockErr := os.Stat(filepath.Join(m.root, "steps-seen", "deliver.out"))
	if blockErr != nil {
		m.t.Fatalf("the delivery printed nothing that was kept: %v", blockErr)
	}

	for _, note := range m.notes() {
		if note.name == name {
			return note.written.Sub(block.ModTime())
		}
	}

	m.t.Fatalf("the note %s is gone", name)

	return 0
}
