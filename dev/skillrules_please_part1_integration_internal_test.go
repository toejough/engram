//go:build targ && integration

package dev

import (
	"regexp"
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
	g.Expect(pleasePart1Unnegated(t, "please read this file")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "in the meantime")).To(BeEmpty())
	g.Expect(pleasePart1Unnegated(t, "to be ready")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "reading repo docs")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "nothing to document")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "skip the plan")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "abandon")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "the diff is right there")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "leading question")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "relitigating in later steps")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "build the prerequisite")).To(BeEmpty())
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
	g.Expect(pleasePart1Unnegated(t, "fully verify yet")).To(BeEmpty())
}

// TestPleasePart1UnnegatedSentences shows the scanner of the absence checks finds a sentence that names a fragment
// without a negation, whatever its case, and passes the sentences that name it only to forbid it.
func TestPleasePart1UnnegatedSentences(t *testing.T) {
	t.Parallel()

	t.Run("a sentence that names the fragment without a negation is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "Do not fire on this. A casual Please Read This File always fires the workflow. Another sentence."

		g.Expect(pleasePart1UnnegatedSentences(text, "please read this file")).
			To(ConsistOf("A casual Please Read This File always fires the workflow"))
	})

	t.Run("a sentence that names the fragment only to forbid it is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "Do not fire on \"please read this file\". The user cannot skip the plan. Never nothing to document."

		g.Expect(pleasePart1UnnegatedSentences(text, "please read this file")).To(BeEmpty())
		g.Expect(pleasePart1UnnegatedSentences(text, "skip the plan")).To(BeEmpty())
		g.Expect(pleasePart1UnnegatedSentences(text, "nothing to document")).To(BeEmpty())
	})

	t.Run("a word that merely contains a negation does not count as one", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "If there is nothing to document, mark the step N/A."

		g.Expect(pleasePart1UnnegatedSentences(text, "nothing to document")).To(HaveLen(1))
	})
}

// unexported constants.
const (
	pleasePart1SkillPath = "agent-instructions/skills/please/SKILL.md"
)

// unexported variables.
var (
	pleasePart1NegationPattern = regexp.MustCompile(`(?i)\b(?:not|no|never|cannot|without)\b|n't\b`)
	pleasePart1SentenceEnd     = regexp.MustCompile(`[.?!]\s+`)
)

// pleasePart1Before reports whether both fragments occur in the text and the first occurs before the second.
func pleasePart1Before(text, first, second string) bool {
	firstAt := strings.Index(text, first)
	secondAt := strings.Index(text, second)

	return firstAt >= 0 && secondAt >= 0 && firstAt < secondAt
}

// pleasePart1Unnegated returns, per shipped instruction file (every SKILL.md, references markdown and guidance
// document), the sentences that name the fragment without a negation: an instruction that would contradict a
// prohibition of the please skill.
func pleasePart1Unnegated(t *testing.T, fragment string) map[string][]string {
	t.Helper()

	found := make(map[string][]string)

	for _, path := range aibInstructionPaths(t) {
		sentences := pleasePart1UnnegatedSentences(shippedText(t, path), fragment)
		if len(sentences) > 0 {
			found[path] = sentences
		}
	}

	return found
}

// pleasePart1UnnegatedSentences returns the sentences of the text that name the fragment, in any case, and carry no
// negation word.
func pleasePart1UnnegatedSentences(text, fragment string) []string {
	var unnegated []string

	lowered := strings.ToLower(fragment)

	for _, sentence := range pleasePart1SentenceEnd.Split(text, -1) {
		if strings.Contains(strings.ToLower(sentence), lowered) && !pleasePart1NegationPattern.MatchString(sentence) {
			unnegated = append(unnegated, sentence)
		}
	}

	return unnegated
}
