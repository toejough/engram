//go:build network

package acceptance_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test793a2a1f_ASecondUpdateStillWritesEachLessonAsOneNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a machine on which the user has run engram update once", func() {
		machine.hasRunUpdate()
	})

	step(t, "When the user runs engram update again, and corrects the agent in a prompt of a session", func() {
		machine.mustSucceed(machine.update(), "engram update")

		chat = machine.startSession("zorblax").converse()
		chat.tell(askHowToBuild)

		chat.tell(correctTheBuild)
	})

	step(t, "Then the vault holds one note for the correction", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)
		machine.awaitSettled(chat.session.id, chat.turnEnds, 2)

		g.Expect(machine.noteNames()).To(HaveLen(1), "the notes of the vault")
	})
}

func Test793a2a1g_ALessonIsNotWrittenToAVaultThatDoesNotExist(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a machine with Claude Code on which the user has run engram update, and whose configured vault no "+
		"longer exists", func() {
		machine.hasRunUpdate()
		machine.vaultIsRemoved()

		g.Expect(machine.hasVault()).To(BeFalse(), "the vault is gone")

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user corrects the agent in a prompt of a session, and the turn ends", func() {
		chat.tell(askHowToBuild)
		chat.tell(correctTheBuild)
	})

	step(t, "Then the vault still does not exist 2 minutes after the turn ends", func() {
		machine.awaitSettled(chat.session.id, chat.turnEnds, 2)

		g.Expect(machine.hasVault()).To(BeFalse(), "a vault exists at "+machine.vault)
	})
}

func Test793a2c1a_ACorrectionBecomesANote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		inSession *session
		chat      *conversation
	)

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has answered a question about how to build a project", func() {
		machine.hasRunUpdate()

		inSession = machine.startSession("zorblax")
		chat = inSession.converse()
		chat.tell(askHowToBuild)
	})

	step(t, "When the user corrects the agent's answer in a prompt, and the turn ends", func() {
		chat.tell(correctTheBuild)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the correction and says it was written "+
		"automatically, and the agent wrote nothing to the vault", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)

		g.Expect(inSession.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

func Test793a2c1b_ARequestToRememberBecomesANote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember something, and the turn ends", func() {
		chat.tell(askToRemember)
	})

	step(t, "Then within 2 minutes the vault holds a note that states it and says it was written automatically, "+
		"and the agent wrote nothing to the vault", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states what was asked to be remembered", "Sunday", "03:40")

		g.Expect(chat.session.agentRan("engram")).To(BeFalse(), "the agent ran an engram command")
	})
}

func Test793a2c1c_AnApproachTheUserConfirmedBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has done a task one way", func() {
		machine.hasRunUpdate("sort")

		inSession := machine.startSession("zorblax")
		inSession.hasFile("zorblax.txt", "pear\nfig\napple\n")
		chat = inSession.converse()

		chat.tell(askToSort)
	})

	step(t, "When the user tells the agent in a prompt that the way worked and to keep doing it, and the turn ends",
		func() {
			chat.tell(confirmTheSort)
		})

	step(t, "Then within 2 minutes the vault holds a note that states the approach and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the approach", "sort", "zorblax")
	})
}

func Test793a2c1d_TheAgentsOwnCorrectionBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may read files "+
		"but may not run engram commands", func() {
		machine.hasRunUpdate()

		inSession := machine.startSession("zorblax")
		inSession.hasFile("zorblax.conf", "listen_port = 7731\n")
		chat = inSession.converse()
	})

	step(t, "When the user asks the agent in a prompt to guess what a file says and then to read it, "+
		"and the agent says its guess was wrong and states what the file says, and the turn ends", func() {
		chat.tell(askToGuessAndRead)
	})

	step(t, "Then within 2 minutes the vault holds a note that states what the file says and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states what the file says", "7731")
	})
}

func Test793a2c1e_AnApproachThatSucceededWithEvidenceBecomesANote(t *testing.T) {
	t.Parallel()

	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may run a check script "+
		"but may not run engram commands", func() {
		machine.hasRunUpdate("sh check.sh")

		inSession := machine.startSession("zorblax")
		inSession.hasFile("check.sh", checkScript)
		chat = inSession.converse()
	})

	step(t, "When the agent, unsure whether an approach will make the check pass, tries it in a turn "+
		"and the turn shows that the check passed", func() {
		chat.tell(askToTryTheFlag)
	})

	step(t, "Then within 2 minutes the vault holds a note that states the approach and says it was written "+
		"automatically", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states the approach", "zorblax-safe")
	})
}

func Test793a2c1f_GuessesPlansAndChatterProduceNoNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user sends a prompt that the agent answers with a guess, a plan and chatter, and the turn ends",
		func() {
			chat.tell(askForAGuessAndAPlan)
		})

	step(t, "Then 2 minutes after the turn ends the vault holds no note", func() {
		g.Expect(machine.awaitRecord(chat.session.id, chat.turnEnds)).To(Equal("nothing"), "what became of the turn")
		g.Expect(machine.noteNames()).To(BeEmpty(), "the notes of the vault")
	})
}

func Test793a2c1g_ACredentialInTheTurnIsNotInTheNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember how to deploy, with the token and the password "+
		"that the deploy needs, and the turn ends", func() {
		chat.tell(askToRememberTheDeploy)
	})

	step(t, "Then within 2 minutes the vault holds a note that states how to deploy, and no note holds the token "+
		"or the password", func() {
		machine.awaitAutomaticNote(chat.turnEnds, "that states how to deploy", "staging")

		for _, note := range machine.notes() {
			g.Expect(note.text).NotTo(ContainSubstring(deployToken), note.name)
			g.Expect(note.text).NotTo(ContainSubstring(deployPassword), note.name)
		}
	})
}

func Test793a2c1h_AThanksThatNamesNoApproachProducesNoNote(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands "+
		"and has done a task one way", func() {
		machine.hasRunUpdate("sort")

		inSession := machine.startSession("zorblax")
		inSession.hasFile("zorblax.txt", "pear\nfig\napple\n")
		chat = inSession.converse()

		chat.tell(askToSort)
	})

	step(t, "When the user thanks the agent in a prompt, saying only that it worked, and the turn ends", func() {
		chat.tell(thankForTheSort)
	})

	step(t, "Then 2 minutes after the turn ends the vault holds no note", func() {
		g.Expect(machine.awaitRecords(chat.session.id, chat.turnEnds, 2)).To(Equal([]string{"nothing", "nothing"}),
			"what became of the two turns")
		g.Expect(machine.noteNames()).To(BeEmpty(), "the notes of the vault")
	})
}

func Test793a2d1b_AutomaticMemorysWorkStartsNoSessionOfItsOwn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user corrects the agent in a prompt, and the turn ends", func() {
		chat.tell(correctTheBuild)
	})

	step(t, "Then within 2 minutes the vault holds a note for the correction, and no session but the user's exists, "+
		"so nothing was delivered to or written from another", func() {
		written := machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)

		g.Expect(machine.noteText(written)).To(ContainSubstring("(session "+chat.session.id+")"),
			"the session the note was written from")

		machine.awaitSettled(chat.session.id, chat.turnEnds, 1)

		g.Expect(machine.sessionsRecorded()).To(Equal([]string{chat.session.id}), "the sessions of the machine")
	})
}

func Test793a3a1a_AWrittenNoteIsDeliveredWithLaterPrompts(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		chat    *conversation
		written string
	)

	step(t, "Given the user has run engram update, and a session in which the user corrected the agent and the "+
		"correction became a note", func() {
		machine.hasRunUpdate()

		chat = machine.startSession("zorblax").converse()
		chat.tell(askHowToBuild)

		chat.tell(correctTheBuild)

		written = machine.awaitAutomaticNote(chat.turnEnds, "that states the correction", buildTarget)

		// the note is written when the write has ended: until it has embedded the note, a lookup finds no embedding
		kinds := machine.awaitRecords(chat.session.id, chat.turnEnds, 2)
		g.Expect(kinds[len(kinds)-1]).To(Equal("written"), "what became of the correction")
	})

	var inSession, inNewSession *turn

	step(t, "When the user sends a prompt that the note bears on, in that session and then in a new session", func() {
		inSession = chat.tell(askHowToBuild)
		inNewSession = machine.startSession("zorblax").send(askHowToBuild)
	})

	step(t, "Then each of the two prompts arrives with the note", func() {
		g.Expect(inSession.blocks()).To(ConsistOf(ContainSubstring(written)), "blocks with the prompt in that session")
		g.Expect(inNewSession.blocks()).To(ConsistOf(ContainSubstring(written)), "blocks with the prompt in a new session")
	})
}

func Test793a3c1a_ALessonAnEarlierNoteStatesIsNotWrittenAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		earlier string
		chat    *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that the user saved earlier and that states "+
		"a lesson, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		earlier = machine.learnsFact(stagingReset)
		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user states the same lesson to the agent in a prompt, and the turn ends", func() {
		chat.tell(askToRemember)
	})

	step(t, "Then 2 minutes after the turn ends the vault holds the earlier note only", func() {
		g.Expect(machine.awaitRecord(chat.session.id, chat.turnEnds)).To(Equal("skipped"), "what became of the turn")
		g.Expect(machine.noteNames()).To(Equal([]string{earlier}), "the notes of the vault")
	})
}

func Test793a3c1b_ALessonTheAgentSavedItselfIsNotWrittenAgain(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given the user has run engram update, and a session in which the agent may run engram commands", func() {
		machine.hasRunUpdate("engram")

		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember something and to save it with engram learn, "+
		"and the turn ends", func() {
		chat.tell(askToSaveItself)

		g.Expect(chat.session.agentRan("engram learn")).To(BeTrue(), "the agent saved the note itself")
	})

	step(t, "Then 2 minutes after the turn ends the vault holds one note for it, the one the agent saved", func() {
		g.Expect(machine.awaitRecord(chat.session.id, chat.turnEnds)).To(BeElementOf("skipped", "nothing"),
			"what became of the turn")

		notes := machine.notes()

		g.Expect(notes).To(HaveLen(1), "the notes of the vault")

		if len(notes) == 1 {
			g.Expect(notes[0].states("Sunday", "03:40")).To(BeTrue(), "the note states what was asked to be remembered")
			g.Expect(notes[0].source()).NotTo(ContainSubstring("written automatically"), "the note is the agent's")
		}
	})
}

func Test793a3c1c_ALessonNoEarlierNoteStatesIsWritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		earlier     string
		earlierText string
		chat        *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that the user saved earlier and that states "+
		"a lesson, and a session in which the agent may not run engram commands", func() {
		machine.hasRunUpdate()

		earlier = machine.learnsFact(stagingReset)
		earlierText = machine.noteText(earlier)
		chat = machine.startSession("zorblax").converse()
	})

	step(t, "When the user asks the agent in a prompt to remember a different lesson about the same subject, "+
		"and the turn ends", func() {
		chat.tell(askToRememberTheBackup)
	})

	step(t, "Then within 2 minutes the vault holds the earlier note, unchanged, and a new note for the different "+
		"lesson that says it was written automatically", func() {
		added := machine.awaitAutomaticNote(chat.turnEnds, "that states the backup", "Monday", "05:00")

		g.Expect(machine.noteNames()).To(ConsistOf(earlier, added), "the notes of the vault")
		g.Expect(machine.noteText(earlier)).To(Equal(earlierText), "the earlier note")
	})
}

func Test793a3e1a_TheAgentsOwnLookupsAndSavesKeepWorking(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var (
		staging string
		chat    *conversation
	)

	step(t, "Given the user has run engram update, a vault holding a note that says when the staging database is "+
		"reset, and a session in which the agent may run engram commands", func() {
		machine.hasRunUpdate("engram")

		staging = machine.learnsFact(stagingReset)
		chat = machine.startSession("zorblax").converse()
	})

	var (
		asked                 *turn
		saved, whenTheTurnEnd string
	)

	step(t, "When the user asks the agent in a prompt to look the staging note up with engram query and to save "+
		"another note with engram learn", func() {
		asked = chat.tell(askToLookUpAndSave)

		saved = machine.noteNamed(stagingBackup.slug)
		whenTheTurnEnd = machine.noteText(saved)
	})

	step(t, "Then the agent's lookup shows the staging note, and the vault holds the note the agent saved, as the "+
		"agent wrote it", func() {
		g.Expect(asked.toolResultsNaming(staging)).To(BeTrue(), "the agent's lookup printed the staging note")
		g.Expect(whenTheTurnEnd).To(And(ContainSubstring("Monday at 05:00 UTC"), ContainSubstring("source: agent")),
			"the note the agent saved")

		machine.awaitRecord(chat.session.id, chat.turnEnds)

		g.Expect(machine.noteText(saved)).To(Equal(whenTheTurnEnd), "the note the agent saved, after the judgement")
	})
}

// unexported constants.
const (
	// askForAGuessAndAPlan makes the agent answer with a guess, a plan and chatter, none of which is a lesson.
	askForAGuessAndAPlan = "What do you think is making the Zorblax service start slowly? " +
		"Answer with a guess, then a plan, then a friendly remark. Do not read or run anything."
	// askHowToBuild is a question the agent cannot answer from anything but a guess: the project is made up.
	askHowToBuild = "How do I build the Zorblax project? Answer in one line."
	// askToGuessAndRead makes the agent state a guess, read a file whose content it cannot guess, and say that the guess
	// was wrong.
	askToGuessAndRead = "Before you read anything, say in one sentence which port the Zorblax service listens on, " +
		"guessing from its name. Then read zorblax.conf with the Read tool. Then say in one sentence which port it " +
		"really listens on, and say that your first guess was wrong."
	// askToLookUpAndSave makes the agent look the stagingReset note up with engram query and save the stagingBackup note
	// with engram learn, as two plain commands: a command with a pipe or a second command in it would be refused.
	askToLookUpAndSave = "Run these two shell commands, one after the other, each as one plain command with nothing " +
		"added to it:\n" +
		"engram query --phrase 'staging database reset schedule' --limit 5\n" +
		"engram learn fact --slug zorblax-staging-backup --position top --source agent --situation 'testing " +
		"against the Zorblax staging database' --subject 'the Zorblax staging database' --predicate " +
		"'is backed up every' --object 'Monday at 05:00 UTC'\n" +
		"Then reply with one short sentence."
	// askToRemember asks for a fact to be remembered.
	askToRemember = "Remember that the Zorblax staging database is reset every Sunday at 03:40 UTC. " +
		"Reply with one short sentence."
	// askToRememberTheBackup asks to remember a second lesson about the database that stagingReset is about.
	askToRememberTheBackup = "Remember that the Zorblax staging database is backed up every Monday at 05:00 UTC. " +
		"Reply with one short sentence."
	// askToRememberTheDeploy asks to remember a deploy whose token and password must not reach the vault.
	askToRememberTheDeploy = "Remember how to deploy to staging: run the staging deploy script with DEPLOY_TOKEN=" +
		deployToken + " set, and its password is " + deployPassword + ". Reply with one short sentence."
	// askToSaveItself asks the agent to remember a fact and to save it with engram learn, with the one plain command to
	// run: a command with a pipe or a second command in it would be refused.
	askToSaveItself = "Remember that the Zorblax staging database is reset every Sunday at 03:40 UTC, and save it " +
		"with engram learn. Run exactly this one command, with nothing added to it: engram learn fact --slug " +
		"zorblax-staging-reset --position top --source agent --situation 'testing against the Zorblax staging " +
		"database' --subject 'the Zorblax staging database' --predicate 'is reset every' --object " +
		"'Sunday at 03:40 UTC'. Then reply with one short sentence."
	// askToSort is a task the agent does one way.
	askToSort = "Print the lines of zorblax.txt in sorted order with the sort command. Reply with one short sentence."
	// askToTryTheFlag makes the agent say what it does not know, try it, and say what it saw.
	askToTryTheFlag = "You do not know whether passing --zorblax-safe makes check.sh pass: say so in one sentence. " +
		"Then run exactly this one command, with nothing added to it: sh check.sh --zorblax-safe. " +
		"Then say in one sentence whether the check passed."
	// buildTarget is the word of the correction that no other text of the conversation holds.
	buildTarget = "zorblax-build"
	// checkScript passes only when it is given the flag.
	checkScript = "#!/bin/sh\nif [ \"$1\" = \"--zorblax-safe\" ]; then echo \"CHECK PASSED\"; exit 0; fi\n" +
		"echo \"CHECK FAILED: the flag is missing\"\nexit 1\n"
	// confirmTheSort confirms the way the agent did askToSort.
	confirmTheSort = "That worked - sorting zorblax.txt with the sort command was exactly the right way. " +
		"Keep sorting Zorblax lists with the sort command. Reply with one short sentence."
	// correctTheBuild corrects whatever the agent answered to askHowToBuild.
	correctTheBuild = "No - in the Zorblax project you build with make " + buildTarget + ", never with make all. " +
		"Reply with one short sentence."
	// deployPassword and deployToken are secrets built here and never written whole in this file: a scanner of the
	// repository would take a literal for a credential.
	deployPassword = "hunter" + "2hunter2"
	deployToken    = "gh" + "p_" + "aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3aZ3"
	// thankForTheSort thanks the agent for askToSort and names no approach.
	thankForTheSort = "Thanks, that worked. Reply with one short sentence."
)
