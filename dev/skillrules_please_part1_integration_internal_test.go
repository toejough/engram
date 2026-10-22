//go:build targ && integration

package dev

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// Test792a18a1a_TheSlashFormAlwaysFires reads the please skill the repository ships: the explicit slash form is an
// opt-in that fires however trivial the ask looks, because the user chose the workflow deliberately.
func Test792a18a1a_TheSlashFormAlwaysFires(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("The explicit slash form `/please <ask>` is an opt-in and always fires " +
		"regardless of how trivial the ask looks"))
	g.Expect(text).To(ContainSubstring("the user chose the workflow deliberately."))
	g.Expect(text).To(ContainSubstring("Run a fixed seven-step workflow against the user's `<ask>`."))
}

// Test792a18a1b_AHandedOverPieceOfWorkFires reads the please skill the repository ships: a natural-language phrasing
// such as "take this end-to-end" fires, and an ask that entails planning, tests or documentation fires.
func Test792a18a1b_AHandedOverPieceOfWorkFires(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("natural-language phrasings of the same intent: \"please do X\", " +
		"\"please work through X\", \"please take care of X\", \"work through X end-to-end\", " +
		"\"take this end-to-end\", \"drive X to completion\", \"see this through\"."))
	g.Expect(text).To(ContainSubstring("fire only when the user is handing over a multi-step piece of work to be " +
		"carried end-to-end."))
	g.Expect(text).To(ContainSubstring("If the ask plausibly entails any of {planning, more than one file, testing, " +
		"documentation, a commit, closing an issue}, the skill fires."))
}

// Test792a18a1c_ACasualPleaseDoesNot reads the please skill the repository ships: a casual "please" attached to a
// single trivial action, such as "please read this file", does not fire the skill, and no shipped instruction says
// that it does.
func Test792a18a1c_ACasualPleaseDoesNot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Do not fire on casual uses of \"please\" attached to a single trivial " +
		"action (\"please read this file\", \"please rename this var\")"))
	g.Expect(text).To(ContainSubstring("If the ask plausibly resolves to a single file edit or a single tool call " +
		"with no follow-up, the skill should not fire — handle it directly."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanCasualPlease)).To(BeEmpty())
}

// Test792a18a1d_AnUnclearBoundaryFires reads the please skill the repository ships: when the boundary of a
// natural-language ask is genuinely unclear, the skill fires.
func Test792a18a1d_AnUnclearBoundaryFires(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("When the boundary is genuinely unclear, fire — the workflow tolerates " +
		"small asks; the reverse is more expensive."))
}

// Test792a18b1a_ABareInvocationGetsOneQuestion reads the please skill the repository ships: a bare invocation, or a
// request with no concrete ask, gets the one question "What would you like me to work on?" through AskUserQuestion,
// and the single question is the entire turn.
func Test792a18b1a_ABareInvocationGetsOneQuestion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("`<ask>` is required. If `/please` is invoked bare, or the surrounding " +
		"natural-language request has no concrete ask, ask **one** targeted question"))
	g.Expect(text).To(ContainSubstring("\"What would you like me to work on?\""))
	g.Expect(text).To(ContainSubstring("via `AskUserQuestion` and wait."))
	g.Expect(text).To(ContainSubstring("The single question is the entire turn."))
}

// Test792a18b1b_NothingHappensMeanwhile reads the please skill the repository ships: while the single question is
// pending no task is created, no recall or learn runs, no file is read and the transcript is not opened, and no
// shipped instruction permits any of them meanwhile.
func Test792a18b1b_NothingHappensMeanwhile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Take **no other action** in the meantime"))
	g.Expect(text).To(ContainSubstring("do not create tasks"))
	g.Expect(text).To(ContainSubstring("do not run `/recall` or `/learn`"))
	g.Expect(text).To(ContainSubstring("do not read files \"to be ready\""))
	g.Expect(text).To(ContainSubstring("do not open the transcript."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanTaskCreation)).To(BeEmpty())
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanRecallOrLearn)).To(BeEmpty())
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanFileRead)).To(BeEmpty())
}

// Test792a18c1a_SevenTasksInOneCall reads the please skill the repository ships: the seven steps are pushed to the
// task list in a single TaskCreate call, each is marked in progress and then completed, and the steps are listed in
// order.
func Test792a18c1a_SevenTasksInOneCall(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("push all seven steps below to the task list via `TaskCreate` in a single " +
		"call"))
	g.Expect(text).To(ContainSubstring("mark each `in_progress` when you begin it and `completed` when you finish it."))

	steps := []string{
		"1. **Capture (open)", "2. **Orient", "3. **Plan.**", "4. **Execute (TDD).**", "5. **Document.**",
		"6. **Complete.**", "7. **Capture (close)",
	}

	for index := 1; index < len(steps); index++ {
		g.Expect(pleasePart1Before(text, steps[index-1], steps[index])).To(BeTrue(), steps[index])
	}
}

// Test792a18c1b_NoStepStartsEarly reads the please skill the repository ships: each step starts only after the
// previous one is completed, so repository docs are not read while step 1's learn is in progress, and no shipped
// instruction says otherwise.
func Test792a18c1b_NoStepStartsEarly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Each step starts only after the previous step is `completed`."))
	g.Expect(text).To(ContainSubstring("Do not begin reading repo docs (step 2) while step 1's `/learn` is still " +
		"`in_progress`."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanStepOverlap)).To(BeEmpty())
}

// Test792a18c1c_AMissingMechanismIsTheOnlyNA reads the please skill the repository ships: a step is not applicable
// only when its mechanism does not exist, step 6's commit without version control is the example, and the rationale
// naming the missing mechanism goes into the task description.
func Test792a18c1c_AMissingMechanismIsTheOnlyNA(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("A step is N/A only when the *mechanism* doesn't exist in this environment"))
	g.Expect(text).To(ContainSubstring("step 6's commit when the repo is not under VCS"))
	g.Expect(text).To(ContainSubstring("When marking N/A, write the one-line rationale into the task description " +
		"naming the missing mechanism"))
	g.Expect(text).To(ContainSubstring("(e.g. \"no VCS — `git` not present\")"))
}

// Test792a18c1d_SmallIsNotNA reads the please skill the repository ships: a step that seems to have nothing to do,
// such as documenting a change that touched no documentation, is run and not marked not applicable, and no shipped
// instruction calls "nothing to document" a reason to skip.
func Test792a18c1d_SmallIsNotNA(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("\"nothing to document\""))
	g.Expect(text).To(ContainSubstring("are **not** N/A — run the step"))
	g.Expect(text).To(ContainSubstring("if it produces nothing, that's fine, and the closing `/learn` will record it."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanNothingToDocument)).To(BeEmpty())
}

// Test792a18c1e_HurryDoesNotWaive reads the please skill the repository ships: the user cannot waive steps by saying
// "no ceremony", "skip the plan" or "hurry"; the workflow, including the plan, runs.
func Test792a18c1e_HurryDoesNotWaive(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("The user said \"no ceremony / skip the plan / hurry\" and you collapsed the " +
		"workflow"))
	g.Expect(text).To(ContainSubstring("The user cannot waive steps. Run the workflow;"))
	g.Expect(text).To(ContainSubstring("3. **Plan.** Write a plan to accomplish the ask."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanWaiver)).To(BeEmpty())
}

// Test792a18c1f_ARedirectUpdatesTheList reads the please skill the repository ships: when the user redirects
// mid-workflow the task list is updated to the new shape and no open step is abandoned without an explicit close.
func Test792a18c1f_ARedirectUpdatesTheList(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("If the user interrupts and redirects mid-workflow, update the task list to " +
		"reflect the new shape"))
	g.Expect(text).To(ContainSubstring("do not abandon open steps without an explicit close."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanAbandon)).To(BeEmpty())
}

// Test792a18d1a_LearnFirst reads the please skill the repository ships: step 1, the learn skill, runs before any new
// work and is listed before the orientation step.
func Test792a18d1a_LearnFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("1. **Capture (open) — `/learn`.** Before starting new work, run the `learn` " +
		"skill to preserve anything pending from the session so far"))
	g.Expect(text).To(ContainSubstring("You're about to start working without running step 1 (`/learn`) | Stop. " +
		"Open the workflow with `/learn` first."))
	g.Expect(pleasePart1Before(text, "1. **Capture (open)", "2. **Orient")).To(BeTrue())
}

// Test792a18d1b_RecallIsLiteral reads the please skill the repository ships: the first action of orientation is a
// literal invocation of recall, reading files, grep and git log are not substitutes for it, and a small ask whose diff
// is right there does not excuse reading files instead of recalling.
func Test792a18d1b_RecallIsLiteral(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("The first action is **literal**: invoke `/recall`"))
	g.Expect(text).To(ContainSubstring("not \"I'll just read the file directly\""))
	g.Expect(text).To(ContainSubstring("not \"I already know this repo\""))
	g.Expect(text).To(ContainSubstring("not \"small ask, the diff is right there\""))
	g.Expect(text).To(ContainSubstring("not \"I'll just grep\"."))
	g.Expect(text).To(ContainSubstring("They are not substitutes — run `/recall` AND the file-tree tools, " +
		"never instead."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanReadingForRecall)).To(BeEmpty())
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanSubstituteForRecall)).To(BeEmpty())
}

// Test792a18d1c_OrientationLoopsOnDoubt reads the please skill the repository ships: orientation asks the user
// through AskUserQuestion and repeats the loop until understanding is solid, and does not move on while material
// doubt remains.
func Test792a18d1c_OrientationLoopsOnDoubt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Loop until the ask is understood:"))
	g.Expect(text).To(ContainSubstring("Ask the user clarifying questions about intent or any new concepts " +
		"encountered, via `AskUserQuestion`."))
	g.Expect(text).To(ContainSubstring("Repeat the recall/read/ask loop until understanding is solid."))
	g.Expect(text).To(ContainSubstring("Do not move on while material doubt remains (about the ask's meaning — or " +
		"about whether it should be done as stated)."))
}

// Test792a18e1a_AFlawIsStatedWithStakes reads the please skill the repository ships: the ask is evaluated on its
// merits during orientation, and a flaw is stated plainly, before planning, with its concrete stakes in a declarative
// sentence, the example being plaintext credentials.
func Test792a18e1a_AFlawIsStatedWithStakes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("**Evaluate the ask on its merits** during orientation"))
	g.Expect(text).To(ContainSubstring("**Challenge plainly and directly, before planning around the problem.** " +
		"State the issue and its concrete stakes in declarative sentences"))
	g.Expect(text).To(ContainSubstring("(\"this stores credentials in plaintext; anyone with file access owns every " +
		"account\")"))
	g.Expect(text).To(ContainSubstring("raise any challenge NOW"))
	g.Expect(text).To(ContainSubstring("before a plan calcifies around the problem."))
	g.Expect(pleasePart1Before(text, "2. **Orient", "3. **Plan.**")).To(BeTrue())
}

// Test792a18e1b_TheChallengeIsNotSoftened reads the please skill the repository ships: a challenge is a declarative
// statement and not a leading question, and no shipped instruction tells the agent to phrase it as one.
func Test792a18e1b_TheChallengeIsNotSoftened(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("State the issue and its concrete stakes in declarative sentences"))
	g.Expect(text).To(ContainSubstring("not softened into a leading question"))
	g.Expect(text).To(ContainSubstring("You softened a challenge into a hinting question because directness felt " +
		"rude | State it declaratively with concrete stakes"))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanSoftenedChallenge)).To(BeEmpty())
}

// Test792a18e1c_TheUsersReaffirmationIsRecordedAndHonored reads the please skill the repository ships: when the user
// reaffirms after the challenge, the agent proceeds, records the dissent in the plan as "considered X; user chose Y
// because Z", and does not relitigate it.
func Test792a18e1c_TheUsersReaffirmationIsRecordedAndHonored(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("**Resolution rule: challenge once, clearly; then commit.** If the user " +
		"weighs the challenge and reaffirms, proceed wholeheartedly"))
	g.Expect(text).To(ContainSubstring("record the dissent in the plan (\"considered X; user chose Y because Z\")."))
	g.Expect(text).To(ContainSubstring("No relitigating in later steps, no passive-aggressive hedging in the work " +
		"itself."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanRelitigation)).To(BeEmpty())
}

// Test792a18f1a_TheAskedTaskBegins reads the please skill the repository ships: once the ask is given, or a gate
// approves the plan, the asked task is a settled decision and begins, and building a prerequisite first is
// relitigating it.
func Test792a18f1a_TheAskedTaskBegins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Once the user gives the ask (or a gate approves the plan), the asked task " +
		"is a SETTLED decision: begin it."))
	g.Expect(text).To(ContainSubstring("Recommending you build the prerequisite / better test / \"the real " +
		"blocker\" first FEELS like diligence but IS relitigating the settled task"))
	g.Expect(text).To(ContainSubstring("You're about to recommend building a prerequisite / better test / \"the real " +
		"blocker\" instead of starting the asked task | That displacement IS relitigating the settled task. Begin " +
		"the asked task"))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanDisplacement)).To(BeEmpty())
}

// Test792a18f1b_AReversalNeedsANewFact reads the please skill the repository ships: a deviation names a genuinely new
// fact and states the reversal of direction, and a dependency-order or "we can't fully verify yet" argument is not
// new evidence.
func Test792a18f1b_AReversalNeedsANewFact(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	text := shippedText(t, pleasePart1SkillPath)

	g.Expect(text).To(ContainSubstring("Deviate only by naming a genuinely NEW fact and explicitly stating you are " +
		"reversing direction."))
	g.Expect(text).To(ContainSubstring("A dependency-order or \"we can't fully verify yet\" argument is re-weighted " +
		"old reasoning, not new evidence."))
	g.Expect(text).To(ContainSubstring("deviate only on a NEW fact, stated as a reversal."))
	g.Expect(pleasePart1Contradictions(t, pleasePart1ScanDeviation)).To(BeEmpty())
}

// TestPleasePart1FlaggedSentences shows the scanner of the absence checks finds a sentence that names what a rule
// prohibits whatever its wording around it, including one that carries a negation word, and passes only the shipped
// sentences the scan allows by their exact text.
func TestPleasePart1FlaggedSentences(t *testing.T) {
	t.Parallel()

	contradictions := map[string]struct {
		scan     pleasePart1Scan
		sentence string
	}{
		"a redirect abandons the open steps": {
			pleasePart1ScanAbandon, "On a redirect, abandon the open steps without an explicit close.",
		},
		"the diff replaces recall": {
			pleasePart1ScanReadingForRecall,
			"When the diff is right there, there is no need to invoke recall; read the files instead.",
		},
		"the challenge is phrased as a question": {
			pleasePart1ScanSoftenedChallenge, "Never state a flaw declaratively; phrase it as a leading question.",
		},
		"the files are read in the meantime": {
			pleasePart1ScanFileRead, "Do not wait for the answer; read the files in the meantime.",
		},
		"a hurry skips the plan": {
			pleasePart1ScanWaiver, "Skip the plan when the user is in a hurry; no ceremony is needed.",
		},
		"recall runs while the question is pending": {
			pleasePart1ScanRecallOrLearn, "While the question is pending, run /recall so you are prepared.",
		},
		"the task list is created while the question is pending": {
			pleasePart1ScanTaskCreation, "While the question is pending, create the task list for the work.",
		},
		"the transcript is opened while waiting": {
			pleasePart1ScanFileRead, "While waiting for the answer, open the transcript and look around.",
		},
	}

	for name, contradiction := range contradictions {
		t.Run("a contradiction is found: "+name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			text := "Nothing here applies. " + contradiction.sentence + " Nor does this."

			g.Expect(pleasePart1Flagged(text, contradiction.scan)).
				To(ConsistOf(strings.TrimSuffix(contradiction.sentence, ".")))
			g.Expect(pleasePart1Flagged(strings.ToUpper(text), contradiction.scan)).To(HaveLen(1))
		})
	}

	t.Run("a shipped sentence the scan allows is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "Nothing here applies. " + pleasePart1ShippedRedirectRule + ". Nor does this."

		g.Expect(pleasePart1Flagged(text, pleasePart1ScanAbandon)).To(BeEmpty())
	})

	t.Run("an allowed sentence with a contradiction appended is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := pleasePart1ShippedRedirectRule + ", and abandon the open steps when the work looks done."

		g.Expect(pleasePart1Flagged(text, pleasePart1ScanAbandon)).To(HaveLen(1))
	})

	t.Run("a sentence that does not name the prohibited act is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "Read the files when the task starts. Run the recall skill first."

		g.Expect(pleasePart1Flagged(text, pleasePart1ScanFileRead)).To(BeEmpty())
		g.Expect(pleasePart1Flagged(text, pleasePart1ScanRecallOrLearn)).To(BeEmpty())
	})
}

// TestPleasePart1ScansFlagTheirAllowList shows each absence scan, run without its allow-list over every shipped
// instruction file, flags exactly the sentences it allows: the scan is not vacuous and no allowed sentence is stale.
func TestPleasePart1ScansFlagTheirAllowList(t *testing.T) {
	t.Parallel()

	for name, scan := range pleasePart1AllScans() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			unallowed := scan
			unallowed.allowed = nil

			found := make([]string, 0, len(scan.allowed))

			for _, sentences := range pleasePart1Contradictions(t, unallowed) {
				for _, sentence := range sentences {
					if !slices.Contains(found, sentence) {
						found = append(found, sentence)
					}
				}
			}

			g.Expect(found).To(ConsistOf(scan.allowed))
		})
	}
}

// unexported constants.
const (
	pleasePart1SkillPath = "agent-instructions/skills/please/SKILL.md"
)

// unexported variables.
var (
	pleasePart1MeantimeMarkers = pleasePart1Patterns(
		`in the meantime`, `meanwhile`, `to be ready`, `question is (?:still )?(?:pending|open|unanswered)`,
		`while (?:the|that|this|your) (?:single |one )?question`, `while (?:you )?(?:wait|await)`,
		`awaiting (?:the|their|an) (?:answer|reply|response)`,
		`until (?:the user|they|it) (?:answers|replies|responds)`, `before (?:the user|they) answer`,
	)
	pleasePart1ScanAbandon = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`redirect`, `interrupt`, `abandon`, `(?:user|they) (?:changes?|switch\w*) (?:the )?(?:ask|direction|mind)`,
			`new direction`,
		),
		allowed: []string{pleasePart1ShippedRedirectRule},
	}
	pleasePart1ScanCasualPlease = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`please read this file`, `please rename this var`, `single trivial action`, `single file edit`,
			`casual (?:uses? of )?"?please`, `\bplease\b.{0,80}\b(?:fires?|triggers?|invokes?|activates?)\b`,
			`\b(?:fires?|triggers?|invokes?|activates?)\b.{0,80}\b(?:trivial|casual|single)\b`,
		),
		allowed: []string{
			pleasePart1ShippedCasualPleaseRule, pleasePart1ShippedMultiStepTestRule, pleasePart1ShippedSlashFormRule,
		},
	}
	pleasePart1ScanDeviation = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`dependency[- ]order`, `fully verify`, `(?:can't|cannot) (?:yet )?(?:fully )?verify`,
			`\bdeviat(?:e|es|ed|ing)\b`, `(?:reversing|reverse|change|switch)\w* (?:of )?direction`, `\bpivot`,
		),
		allowed: []string{
			pleasePart1ShippedBeginAskedTaskRule, pleasePart1ShippedDependencyOrderRule,
			pleasePart1ShippedDeviateOnlyRule, pleasePart1ShippedRecallDefaultRule,
		},
	}
	pleasePart1ScanDisplacement = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`prerequisite`, `real blocker`, `better test`, `before (?:starting|beginning) the (?:asked )?task`,
			`instead of (?:starting|beginning|the asked)`,
		),
		allowed: []string{
			pleasePart1ShippedAboutToRecommendRedFlag, pleasePart1ShippedAntiDisplacementRule,
			pleasePart1ShippedRecallRecommendingRow, pleasePart1ShippedRecallSubstitutingRule,
			pleasePart1ShippedRecommendingPrerequisiteRule,
		},
	}
	pleasePart1ScanFileRead = pleasePart1Scan{
		markers: pleasePart1MeantimeMarkers,
		acts: pleasePart1Patterns(
			`\b(?:read|reads|reading|open|opens|opening|inspect|inspecting|scan|scanning)\b.{0,60}` +
				`\b(?:files?|docs?|repo|repository|code|transcript|notes?)\b`,
		),
		allowed: []string{pleasePart1ShippedNoActionMeanwhileRule},
	}
	pleasePart1ScanNothingToDocument = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`nothing to \w+`, `too (?:small|tiny|trivial|simple) to (?:bother|document|record|capture|run|do)`,
			`no memories`, `nothing relevant`,
		),
		acts:    pleasePart1Patterns(`N/A|not applicable|\bskip\w*|\bomit\w*|\bbypass`),
		allowed: []string{pleasePart1ShippedSmallIsNotNARule},
	}
	pleasePart1ScanReadingForRecall = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`read the file`, `reading (?:the )?(?:working-tree )?files?`, `already know`, `diff is right`, `\bgrep\b`,
			`git log`, `file reads?`, `working-tree`,
		),
		acts: pleasePart1Patterns(`recall`),
		allowed: []string{
			pleasePart1ShippedFileReadsDifferRule, pleasePart1ShippedOrientRecallRule,
			pleasePart1ShippedSkippedRecallRedFlag, pleasePart1ShippedSubstitutionRule,
		},
	}
	pleasePart1ScanRecallOrLearn = pleasePart1Scan{
		markers: pleasePart1MeantimeMarkers,
		acts:    pleasePart1Patterns(`\b(?:recall|learn)\b`),
		allowed: []string{pleasePart1ShippedNoActionMeanwhileRule},
	}
	pleasePart1ScanRelitigation = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`relitigat`, `re-litigat`, `reopen`, `hedg`,
			`raise (?:it|the challenge|the concern|the objection) again`, `challenge (?:it )?again`,
			`second challenge`,
		),
		acts: pleasePart1Patterns(
			`dissent|later steps|reaffirm|user chose|passive-aggressive|the challenge|challenged`,
		),
		allowed: []string{pleasePart1ShippedNoRelitigatingRule},
	}
	pleasePart1ScanSoftenedChallenge = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`leading question`, `hinting question`, `\bhints? at`, `soften`, `as a question`, `rhetorical`, `tactful`,
			`gently`, `diplomatic`,
		),
		allowed: []string{pleasePart1ShippedChallengePlainlyRule, pleasePart1ShippedSoftenedRedFlag},
	}
	pleasePart1ScanStepOverlap = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`in parallel`, `concurrently`, `simultaneously`, `at the same time`, `without waiting`, `overlap`,
			`\bbefore (?:step|the previous|the prior)\b`, `\bwhile (?:step|the (?:previous|prior|first) step)`,
			`\bstart(?:s|ed)? early\b`, `\bwhile\b.{0,60}\bin_progress\b`,
		),
		acts:    pleasePart1Patterns(`\bstep|repo docs|orient|/learn|/recall`),
		allowed: []string{pleasePart1ShippedNoEarlyOrientRule},
	}
	pleasePart1ScanSubstituteForRecall = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`substitut\w*`, `equivalen\w*`, `instead of`, `rather than`, `never instead`, `in place of`,
		),
		acts: pleasePart1Patterns(`recall`, `\b(?:read|reading|grep|git log|files?|working-tree)\b`),
		allowed: []string{
			pleasePart1ShippedNotSubstitutesRule, pleasePart1ShippedSubstitutionRule,
		},
	}
	pleasePart1ScanTaskCreation = pleasePart1Scan{
		markers: pleasePart1MeantimeMarkers,
		acts:    pleasePart1Patterns(`\btasks?\b|TaskCreate`),
		allowed: []string{pleasePart1ShippedNoActionMeanwhileRule},
	}
	pleasePart1ScanWaiver = pleasePart1Scan{
		markers: pleasePart1Patterns(
			`no ceremony`, `skip (?:the )?plan`, `\bhurr(?:y|ied)\b`,
			`(?:user|they) (?:can|may|could) (?:skip|waive|opt out)\w*`,
			`waive\w* (?:the )?(?:steps?|workflow|plan|gate)`, `skip(?:ping)? (?:the )?(?:workflow|ceremony|plan)\b`,
		),
		allowed: []string{pleasePart1ShippedCannotWaiveRule, pleasePart1ShippedHurryRedFlag},
	}
	pleasePart1SentenceEnd                    = regexp.MustCompile(`[.?!]\s+|\s\|\s`)
	pleasePart1ShippedAboutToRecommendRedFlag = "You're about to recommend building a prerequisite / better " +
		"test / \"the real blocker\" instead of starting the asked task"
	pleasePart1ShippedAntiDisplacementRule = "- **Anti-displacement (the settled-task corollary).** " +
		"\"Challenge the ask\" means stress-test the ask *in place* — it does NOT license substituting adjacent, " +
		"deferred, or \"more rigorous\" prerequisite work for what was asked"
	pleasePart1ShippedBeginAskedTaskRule = "Begin the asked task; deviate only on a NEW fact, stated as a " +
		"reversal"
	pleasePart1ShippedCannotWaiveRule  = "The user cannot waive steps"
	pleasePart1ShippedCasualPleaseRule = "Do not fire on casual uses of \"please\" attached to a single " +
		"trivial action (\"please read this file\", \"please rename this var\") — fire only when the user is " +
		"handing over a multi-step piece of work to be carried end-to-end"
	pleasePart1ShippedChallengePlainlyRule = "- **Challenge plainly and directly, before planning around the " +
		"problem.** State the issue and its concrete stakes in declarative sentences (\"this stores credentials " +
		"in plaintext; anyone with file access owns every account\") — not softened into a leading question, not " +
		"buried mid-plan, and never prefaced with reflexive praise"
	pleasePart1ShippedDependencyOrderRule = "A dependency-order or \"we can't fully verify yet\" argument is " +
		"re-weighted old reasoning, not new evidence"
	pleasePart1ShippedDeviateOnlyRule = "Deviate only by naming a genuinely NEW fact and explicitly stating " +
		"you are reversing direction"
	pleasePart1ShippedFileReadsDifferRule = "File reads, grep, and `git log` surface working-tree content; " +
		"`/recall` surfaces agent-memory vault content"
	pleasePart1ShippedHurryRedFlag = "The user said \"no ceremony / skip the plan / hurry\" and you collapsed " +
		"the workflow"
	pleasePart1ShippedMultiStepTestRule = "- **\"Multi-step\" test for ambiguous natural-language triggers.** " +
		"If the ask plausibly resolves to a single file edit or a single tool call with no follow-up, the skill " +
		"should not fire — handle it directly"
	pleasePart1ShippedNoActionMeanwhileRule = "Take **no other action** in the meantime: do not create tasks, " +
		"do not run `/recall` or `/learn`, do not read files \"to be ready\", do not open the transcript"
	pleasePart1ShippedNoEarlyOrientRule = "Do not begin reading repo docs (step 2) while step 1's `/learn` is " +
		"still `in_progress`"
	pleasePart1ShippedNoRelitigatingRule = "No relitigating in later steps, no passive-aggressive hedging in " +
		"the work itself"
	pleasePart1ShippedNotSubstitutesRule = "They are not substitutes — run `/recall` AND the file-tree tools, " +
		"never instead"
	pleasePart1ShippedOrientRecallRule = "**Orient — understand the context and the ask.** The first action " +
		"is **literal**: invoke `/recall` — not \"I'll just read the file directly\", not \"I already know this " +
		"repo\", not \"small ask, the diff is right there\", not \"I'll just grep\""
	pleasePart1ShippedRecallDefaultRule = "Default to the asked task; deviate only by (a) naming a genuinely " +
		"NEW fact and (b) stating you are reversing direction"
	pleasePart1ShippedRecallRecommendingRow = "| You're recommending a prerequisite or better test as the " +
		"first step, not the asked task"
	pleasePart1ShippedRecallSubstitutingRule = "Substituting a prerequisite, a better test, or a \"more " +
		"rigorous foundation\" as the thing to do *first* FEELS like diligence but IS relitigating the settled " +
		"task"
	pleasePart1ShippedRecommendingPrerequisiteRule = "Recommending you build the prerequisite / better test / " +
		"\"the real blocker\" first FEELS like diligence but IS relitigating the settled task — the failure mode, " +
		"not rigor"
	pleasePart1ShippedRedirectRule = "- If the user interrupts and redirects mid-workflow, update the task " +
		"list to reflect the new shape — but do not abandon open steps without an explicit close"
	pleasePart1ShippedSkippedRecallRedFlag = "You skipped `/recall` because \"I already know this\", \"the " +
		"diff is right here\", \"small ask\", \"I'll just read the file\", or \"I'll just grep\""
	pleasePart1ShippedSlashFormRule = "The explicit slash form `/please <ask>` is an opt-in and always fires " +
		"regardless of how trivial the ask looks — the user chose the workflow deliberately"
	pleasePart1ShippedSmallIsNotNARule = "\"Too small to bother\", \"nothing to document\", \"no memories " +
		"will be relevant\" are **not** N/A — run the step; if it produces nothing, that's fine, and the closing " +
		"`/learn` will record it"
	pleasePart1ShippedSoftenedRedFlag = "You softened a challenge into a hinting question because directness " +
		"felt rude"
	pleasePart1ShippedSubstitutionRule = "Reading working-tree files is **substitution**, not equivalence — " +
		"`/recall` surfaces vault memory those reads can't produce"
)

// pleasePart1Scan is one absence check: a sentence is flagged when it matches at least one marker and every act,
// unless it is one of the shipped sentences allowed by their exact text.
type pleasePart1Scan struct {
	markers []*regexp.Regexp
	acts    []*regexp.Regexp
	allowed []string
}

// pleasePart1AllScans returns every absence scan of this node by name.
func pleasePart1AllScans() map[string]pleasePart1Scan {
	return map[string]pleasePart1Scan{
		"abandon":             pleasePart1ScanAbandon,
		"casual please":       pleasePart1ScanCasualPlease,
		"deviation":           pleasePart1ScanDeviation,
		"displacement":        pleasePart1ScanDisplacement,
		"file read":           pleasePart1ScanFileRead,
		"nothing to document": pleasePart1ScanNothingToDocument,
		"reading for recall":  pleasePart1ScanReadingForRecall,
		"recall or learn":     pleasePart1ScanRecallOrLearn,
		"relitigation":        pleasePart1ScanRelitigation,
		"softened challenge":  pleasePart1ScanSoftenedChallenge,
		"step overlap":        pleasePart1ScanStepOverlap,
		"substitute":          pleasePart1ScanSubstituteForRecall,
		"task creation":       pleasePart1ScanTaskCreation,
		"waiver":              pleasePart1ScanWaiver,
	}
}

// pleasePart1Before reports whether both fragments occur in the text and the first occurs before the second.
func pleasePart1Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// pleasePart1Contradictions returns, per shipped instruction file (every SKILL.md, references markdown and guidance
// document), the sentences the scan flags: sentences that name what a prohibition of the please skill forbids and are
// not one of the shipped sentences the scan allows.
func pleasePart1Contradictions(t *testing.T, scan pleasePart1Scan) map[string][]string {
	t.Helper()

	found := make(map[string][]string)

	for _, path := range aibInstructionPaths(t) {
		sentences := pleasePart1Flagged(shippedText(t, path), scan)
		if len(sentences) > 0 {
			sort.Strings(sentences)

			found[path] = sentences
		}
	}

	return found
}

// pleasePart1Flagged returns the sentences of the text that the scan flags, whatever negation words they carry.
func pleasePart1Flagged(text string, scan pleasePart1Scan) []string {
	var flagged []string

	for _, sentence := range pleasePart1SentenceEnd.Split(text, -1) {
		sentence = strings.TrimSpace(sentence)

		if pleasePart1Names(sentence, scan) && !slices.Contains(scan.allowed, sentence) {
			flagged = append(flagged, sentence)
		}
	}

	return flagged
}

// pleasePart1Names reports whether the sentence matches at least one marker of the scan and every act of it.
func pleasePart1Names(sentence string, scan pleasePart1Scan) bool {
	marked := slices.ContainsFunc(scan.markers, func(marker *regexp.Regexp) bool {
		return marker.MatchString(sentence)
	})
	if !marked {
		return false
	}

	for _, act := range scan.acts {
		if !act.MatchString(sentence) {
			return false
		}
	}

	return true
}

// pleasePart1Patterns compiles the patterns, each matched in any case.
func pleasePart1Patterns(patterns ...string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))

	for _, pattern := range patterns {
		compiled = append(compiled, regexp.MustCompile(`(?i)`+pattern))
	}

	return compiled
}
