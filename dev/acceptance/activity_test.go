//go:build network

package acceptance_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test793a2e1a_WhatWasDeliveredAndWhatWasWrittenForThePreviousTurnIsInTheSessionByTheNextPrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		stagingNote string
		chat        *conversation
		earlier     *turn
		written     string
	)

	step(t, "Given the user has run engram update, a vault holding a note that says when the staging database is "+
		"reset, and a session in which the user asked in one prompt when it is reset and to remember that it is "+
		"backed up every Monday at 05:00 UTC", func() {
		machine.hasRunUpdate()

		stagingNote = machine.learnsFact(stagingReset)
		chat = machine.startSession("zorblax").converse()
		earlier = chat.tell(askResetAndRememberBackup)

		written = machine.awaitAutomaticNote(chat.turnEnds, "that states the backup", "Monday", "05:00")
		machine.awaitRecords(chat.session.id, chat.turnEnds, 1)
	})

	var next *turn

	step(t, "When the user sends the next prompt", func() {
		next = chat.tell(askSomethingElse)
	})

	step(t, "Then the conversation shows the staging note delivered with the earlier prompt, and the next prompt "+
		"arrives with a line that names the note written from it", func() {
		g.Expect(earlier.blocks()).To(ConsistOf(ContainSubstring(stagingNote)), "the blocks of the earlier prompt")

		g.Expect(next.blocks()).To(HaveLen(1), "the blocks of the next prompt")

		if len(next.blocks()) == 1 {
			g.Expect(linesToldBy(next.blocks()[0])).To(ConsistOf(And(ContainSubstring("wrote the note"),
				ContainSubstring(written))), "the lines that tell what automatic memory did")
		}
	})
}

// unexported constants.
const (
	// askResetAndRememberBackup asks what the stagingReset note answers, and asks to remember a second lesson about the
	// same database.
	askResetAndRememberBackup = "When is the Zorblax staging database reset? Also remember that it is backed up " +
		"every Monday at 05:00 UTC. Reply with one short sentence."
)

// linesToldBy returns the lines of a block that tell what automatic memory did: those that begin with an asterisk.
func linesToldBy(block string) []string {
	var told []string

	for line := range strings.SplitSeq(block, "\n") {
		if strings.HasPrefix(line, "* ") {
			told = append(told, line)
		}
	}

	return told
}
