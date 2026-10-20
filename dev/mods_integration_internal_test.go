//go:build targ && integration

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/auto"
)

// Test793d2a1c_Mod_ADeliveryThatPrintsNothingIsNotAFailure runs the mod under node against a stand-in host that records
// every command the mod runs: a live session cannot show that a report to the writing was not made, since the offline
// run ends once the model is asked.
func Test793d2a1c_Mod_ADeliveryThatPrintsNothingIsNotAFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{Deliver: map[string]any{"exitCode": 0, "stdout": ""}})

	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")
	g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")
	g.Expect(outcome.Calls).To(HaveLen(2), "the commands the mod ran: the moment, which prints nothing, and the delivery")
}

// Test793d2a1f_Mod_TextThatIsNotABlockIsNotAFailure runs the mod under node against a stand-in host that records every
// command the mod runs, for the delivery of an engram that does not know the command.
func Test793d2a1f_Mod_TextThatIsNotABlockIsNotAFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{Deliver: map[string]any{
		"exitCode": 0, "stdout": "Unknown command: auto\nusage: engram <command> [flags]\n",
	}})

	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")
	g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")
	g.Expect(outcome.Calls).To(HaveLen(2), "the commands the mod ran: the moment, which prints nothing, and the delivery")
}

// Test793d2a1g_Mod_ASessionThatDoesNotGiveItsIDInTimeDoesNotHoldThePrompt runs the mod under node against a stand-in
// host: a live session cannot be made to stall in the call for its ID. The stand-in's timer fires at once and records
// the delay it was asked for, and its call for the ID never answers.
func Test793d2a1g_Mod_ASessionThatDoesNotGiveItsIDInTimeDoesNotHoldThePrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	node, findErr := findNode()
	g.Expect(findErr).NotTo(HaveOccurred())

	if findErr != nil {
		return
	}

	hooks := filepath.Join(repositoryRoot, "agent-instructions", "mods", "engram-memory", "hooks")
	source, readErr := os.ReadFile(filepath.Join(hooks, "register.js"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// a module under a name that node reads as one, whatever release of node it is
	module := filepath.Join(t.TempDir(), "register.mjs")
	g.Expect(os.WriteFile(module, source, 0o600)).To(Succeed())

	ctx, cancel := context.WithTimeout(t.Context(), driverTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, node, "--input-type=module", "--eval", hostDriver)
	command.Env = append(os.Environ(), "MOD="+module)

	out, runErr := command.Output()
	if runErr != nil {
		t.Fatalf("the mod did not let the prompt through (%v): %s", runErr, out)
	}

	var outcome struct {
		Passed struct {
			Text    string   `json:"text"`
			Context []string `json:"context"`
		} `json:"passed"`
		Delays []int `json:"delays"`
		Ran    int   `json:"ran"`
	}

	g.Expect(json.Unmarshal(out, &outcome)).To(Succeed(), "the driver printed %q", out)
	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Delays).To(Equal([]int{5000}), "the one timer the mod asked for, in milliseconds")
	g.Expect(outcome.Ran).To(BeZero(), "the command was run")
}

// Test793d2a1i_Mod_ACommandThatFailsIsReportedToTheWriting runs the mod under node against a stand-in host that counts
// the commands the mod ran when the hook settled: a live session shows a report made inside the hook only as a wait.
func Test793d2a1i_Mod_ACommandThatFailsIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{Deliver: map[string]any{
		"exitCode": 1, "stdout": "", "stderr": "auto deliver: looking up: the embedder is not there\nat more\n",
	}})

	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Settle.Commands).To(Equal(2), "the commands run when the hook settled: the moment, the delivery")
	g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
	g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")

	reports := outcome.reports()

	g.Expect(reports).To(HaveLen(1), "the runs of auto write")

	if len(reports) != 1 {
		return
	}

	g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
	g.Expect(reports[0]).To(HaveKeyWithValue("turn", ""))
	g.Expect(reports[0]["failure"]).To(And(ContainSubstring("the delivery for a prompt could not be done"),
		ContainSubstring("the embedder is not there")), "the failure")
}

// Test793d2a1j_Mod_ADeliveryThatDoesNotFinishWithinItsBoundIsReportedToTheWriting runs the mod under node against a
// stand-in host whose model call never settles unless its signal aborts it, and whose timers fire at once: a live
// session cannot be made to pass the bound without waiting for it.
func Test793d2a1j_Mod_ADeliveryThatDoesNotFinishWithinItsBoundIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	// the model's call that is aborted settles with no answer, or rejects: the report says the 2 minutes either way
	for name, rejects := range map[string]bool{
		"a call that settles with no answer when aborted": false,
		"a call that rejects when aborted":                true,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			outcome := runDeliveryDriver(t, deliveryScenario{
				Steps:             map[string]map[string]any{"moment": {"exitCode": 0, "stdout": aStep(1)}},
				ModelNeverSettles: true, ModelRejectsWhenAborted: rejects, FastTimers: true,
			})

			g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
			g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
			g.Expect(outcome.Delays).To(ContainElement(120000), "the timer that aborts the call, in milliseconds")
			g.Expect(outcome.modelCalls()).To(HaveLen(1), "the model was asked once")
			g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
			g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")

			reports := outcome.reports()

			g.Expect(reports).To(HaveLen(1), "the runs of auto write")

			if len(reports) != 1 {
				return
			}

			g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
			g.Expect(reports[0]).To(HaveKeyWithValue("turn", ""))
			g.Expect(reports[0]["failure"]).To(And(ContainSubstring("the delivery for a prompt could not be done"),
				ContainSubstring("2 minutes")), "the failure")
		})
	}
}

// Test793d2a1k_Mod_TheLinesTheCommandPrintsForTheUserAreShownThroughTheHostsLogAndAreNotAddedToTheContext runs the mod
// under node against a stand-in host that records the calls of the host's log: a live session shows them only as
// events of the stream, and the acceptance rows of 793a2e read those.
func Test793d2a1k_Mod_TheLinesTheCommandPrintsForTheUserAreShownThroughTheHostsLogAndAreNotAddedToTheContext(
	t *testing.T,
) {
	t.Parallel()

	g := NewWithT(t)

	block := "[engram memory] The lines below are notes.\n- a.md (s): b"
	lines := []string{"delivered the note a.md", "20:32:25 wrote the note b.md"}
	outcome := runDeliveryDriver(t, deliveryScenario{Deliver: map[string]any{
		"exitCode": 0, "stdout": block + "\n", "stderr": strings.Join(lines, "\n") + "\n",
	}})

	g.Expect(outcome.Logged).To(Equal(lines), "the lines the user was shown, in order")
	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(Equal([]string{block}), "the context beside the prompt: the block alone")
}

// Test793d2b1a_Mod_JudgesATurnAndGivesTheReplyToTheWriting runs the mod under node against a stand-in host: a live
// session cannot show what a stand-in records. The stand-in's moment does not answer until the driver lets it, so a
// hook that waited for the job would not give the event's result back.
func Test793d2b1a_Mod_JudgesATurnAndGivesTheReplyToTheWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Moment = map[string]any{"hold": true, "exitCode": 0, "stdout": momentPrinted}

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Held).To(BeFalse(), "the event's result was not given back while the moment was running")
	g.Expect(outcome.stepsAndLimits()).To(Equal([]string{
		"engram auto moment: 5000", "model: 10000", "engram auto judge: 10000", "model: 30000",
		"engram auto place: 5000", "model: 15000", "engram auto write: 20000",
	}), "the three steps, each followed by the model, and the writing, with their limits")

	if len(outcome.Calls) != 7 {
		return
	}

	g.Expect(outcome.Calls[0].Input).To(Equal(momentInput("hello", "composer")))
	g.Expect(outcome.Calls[1].Asked).To(Equal(map[string]any{
		"model": "haiku", "effort": "low", "system": "THE MOMENT", "prompt": "THE WINDOW",
		"maxTokens": float64(200), "timeoutMs": float64(10000),
	}), "the request, exactly as the moment printed it")
	g.Expect(outcome.Calls[2].Input).To(Equal(map[string]any{
		"session": "a-session", "turn": "a-turn", "text": "hello", "answer": "Understood.", "origin": "composer",
		"reply": "LEARN", "carry": map[string]any{"cue": "turn-end", "window": "THE WINDOW"},
	}), "the same fields, the reply and the carry")
	g.Expect(outcome.Calls[3].Asked).To(Equal(map[string]any{
		"model": "sonnet", "system": "THE INSTRUCTIONS", "prompt": "THE PROMPT", "maxTokens": float64(3000),
		"timeoutMs": float64(30000),
	}), "the request, exactly as the judgement printed it")
	g.Expect(outcome.Calls[4].Input).To(Equal(map[string]any{
		"session": "a-session", "turn": "a-turn", "reply": "LESSONS", "carry": map[string]any{},
	}))
	g.Expect(outcome.Calls[5].Asked).To(Equal(map[string]any{
		"model": "sonnet", "system": "THE ORDER", "prompt": "THE LESSONS", "maxTokens": float64(1500),
		"timeoutMs": float64(15000),
	}), "the request, exactly as the placement printed it")
	g.Expect(outcome.Calls[6].Input).To(Equal(map[string]any{
		"session": "a-session", "turn": "a-turn", "reply": "LESSONS", "placement": "PLACEMENT",
		"carry": map[string]any{"seen": []any{"1.2026-10-17.a-note.md"}},
	}), "the reply to the lessons, the carry of the placement and the reply to the placement")
}

func Test793d2b1b_Mod_AJudgementThatPrintsNothingEndsTheCapture(t *testing.T) {
	t.Parallel()

	t.Run("a moment that prints nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{})

		g.Expect(outcome.stepsAndLimits()).To(Equal([]string{"engram auto moment: 5000"}),
			"only the moment was run: the model was never asked")
	})

	t.Run("a judgement that prints nothing", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		scenario := printingChain()
		scenario.Judge = map[string]any{"exitCode": 0, "stdout": ""}

		outcome := runCaptureDriver(t, scenario)

		g.Expect(outcome.stepsAndLimits()).To(Equal([]string{
			"engram auto moment: 5000", "model: 10000", "engram auto judge: 10000",
		}), "the model was asked once, and the writing was never run")
	})
}

func Test793d2b1c_Mod_AStepThatFailsIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	failures := map[string]map[string]any{
		"exits with 1":                 {"exitCode": 1, "stdout": "Unknown command: auto\n"},
		"prints what is not a request": {"exitCode": 0, "stdout": "Unknown command: auto\n"},
		"is ended":                     {"rejects": "the command ran for 20000 ms"},
	}

	for _, step := range []string{"moment", "judge"} {
		for name, failure := range failures {
			t.Run("a "+step+" that "+name, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				scenario := printingChain()
				scenario.set(step, failure)

				outcome := runCaptureDriver(t, scenario)
				reports := outcome.reports()

				g.Expect(reports).To(HaveLen(1), "the report to the writing")
				g.Expect(outcome.modelCalls()).To(Equal(modelCallsBefore[step]),
					"the model was not asked for the request of the failed step")

				if len(reports) != 1 {
					return
				}

				g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
				g.Expect(reports[0]).To(HaveKeyWithValue("turn", "a-turn"))
				g.Expect(reports[0]).NotTo(HaveKey("reply"))
				g.Expect(reports[0]["failure"]).To(ContainSubstring("engram auto " + step))
			})
		}
	}
}

func Test793d2b1d_Mod_AModelThatDoesNotAnswerIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	for name, failure := range map[string]map[string]any{
		"an API error":                 {"isAnswered": false, "reason": "api-error", "status": 529},
		"a reply with no text":         {"isAnswered": false, "reason": "empty-reply"},
		"a call that the host rejects": {"rejects": "the cap is refused"},
	} {
		for call := range 2 {
			t.Run(fmt.Sprintf("%s on the call %d", name, call+1), func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				scenario := printingChain()
				scenario.Models[call] = failure

				outcome := runCaptureDriver(t, scenario)
				reports := outcome.reports()

				g.Expect(reports).To(HaveLen(1), "the report to the writing")

				if len(reports) != 1 {
					return
				}

				g.Expect(reports[0]).To(HaveKeyWithValue("turn", "a-turn"))
				g.Expect(reports[0]).NotTo(HaveKey("reply"))
				g.Expect(reports[0]["failure"]).To(ContainSubstring([]string{"moment", "judge"}[call]+": the model call"),
					"the failure names the step whose request the model was asked for")
				g.Expect(outcome.modelCalls()).To(HaveLen(call+1), "the calls after the failed one were never made")
			})
		}
	}
}

func Test793d2b1e_Mod_TheJudgementReceivesTheOriginOfThePromptTheTurnBeganWith(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Origin = "task-notification"

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Calls).NotTo(BeEmpty(), "the moment was run")

	if len(outcome.Calls) > 0 {
		g.Expect(outcome.Calls[0].Step).To(Equal("engram auto moment"))
		g.Expect(outcome.Calls[0].Input).To(Equal(momentInput("hello", "task-notification")))
	}
}

func Test793d2b1f_Mod_ATurnOfASubagentIsNotCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Agent = "a-subagent"

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Calls).To(BeEmpty(), "nothing was asked: no step, no model and no writing")
}

func Test793d2b1g_Mod_TheJudgementReceivesTheTextTheTurnBeganWith(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Text = "fix the zorblax build"

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Calls).NotTo(BeEmpty(), "the moment was run")

	if len(outcome.Calls) > 0 {
		g.Expect(outcome.Calls[0].Step).To(Equal("engram auto moment"))
		g.Expect(outcome.Calls[0].Input).To(Equal(momentInput("fix the zorblax build", "composer")))
	}
}

func Test793d2b1h_Mod_ATurnTakesTheOriginAndTheTextOfItsOwnPromptWhenOthersAreWaiting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Script = []map[string]any{
		{"submit": map[string]any{"text": "first", "origin": "composer"}},
		{"start": map[string]any{"text": "first", "turnId": "t1"}},
		{"submit": map[string]any{"text": "second", "origin": "composer"}},
		{"submit": map[string]any{"text": "<task-notification>done</task-notification>", "origin": "task-notification"}},
		{"complete": map[string]any{"turnId": "t1", "answer": "done one"}},
		{"start": map[string]any{"text": "second", "turnId": "t2"}},
		{"complete": map[string]any{"turnId": "t2", "answer": "done two"}},
	}

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.inputsOf("engram auto moment")).To(Equal([]map[string]any{
		{
			"session": "a-session", "turn": "t1", "text": "first", "answer": "done one", "origin": "composer",
			"cue": "turn-end",
		},
		{
			"session": "a-session", "turn": "t2", "text": "second", "answer": "done two", "origin": "composer",
			"cue": "turn-end",
		},
	}))
}

func Test793d2b1i_Mod_ATurnThatStartsWithNoTextHasNoOrigin(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scenario := printingChain()
	scenario.Script = []map[string]any{
		{"submit": map[string]any{"text": "first", "origin": "composer"}},
		{"start": map[string]any{"text": "first", "turnId": "t1"}},
		{"complete": map[string]any{"turnId": "t1", "answer": "done one"}},
		{"start": map[string]any{"text": "", "turnId": "t2"}},
		{"complete": map[string]any{"turnId": "t2", "answer": "continued"}},
	}

	outcome := runCaptureDriver(t, scenario)
	moments := outcome.inputsOf("engram auto moment")

	g.Expect(moments).To(HaveLen(2))

	if len(moments) == 2 {
		g.Expect(moments[1]).To(Equal(map[string]any{
			"session": "a-session", "turn": "t2", "text": "", "answer": "continued", "origin": "", "cue": "turn-end",
		}))
	}
}

func Test793d2b1j_Mod_AStageThatStallsIsEndedAtItsBoundAndReported(t *testing.T) {
	t.Parallel()

	t.Run("the call for the session's ID", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		scenario := printingChain()
		scenario.Stalls = "id"

		outcome := runCaptureDriver(t, scenario)

		g.Expect(outcome.Delays).To(ContainElement(5000), "the mod waited 5 seconds for the ID")
		g.Expect(outcome.Calls).To(BeEmpty(), "and then did nothing: no record can be made without the ID")
	})

	moment, askedMoment := "engram auto moment: 5000", "model: 10000"
	judge, askedJudge := "engram auto judge: 10000", "model: 30000"
	place, askedPlace := "engram auto place: 5000", "model: 15000"
	report := "engram auto write: 15000"

	for name, stage := range map[string]struct {
		stalls   string
		want     []string
		reported string
	}{
		"the moment":                     {"moment", []string{moment, report}, "engram auto moment"},
		"the model's call of the moment": {"model1", []string{moment, askedMoment, report}, "model call"},
		"the judgement":                  {"judge", []string{moment, askedMoment, judge, report}, "engram auto judge"},
		"the model's call of the judgement": {
			"model2", []string{moment, askedMoment, judge, askedJudge, report}, "model call",
		},
		"the writing": {
			"write", []string{
				moment, askedMoment, judge, askedJudge, place, askedPlace, "engram auto write: 20000", report,
			}, "engram auto write",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			scenario := printingChain()
			scenario.Stalls = stage.stalls

			outcome := runCaptureDriver(t, scenario)

			g.Expect(outcome.stepsAndLimits()).To(Equal(stage.want), "the calls the mod made, with their time limits")

			if len(outcome.Calls) > 0 {
				last := outcome.Calls[len(outcome.Calls)-1]
				g.Expect(last.Input["failure"]).To(ContainSubstring(stage.reported), "the failure names the stage")
			}
		})
	}

	for name, stage := range map[string]struct {
		stalls string
		want   []string
		failed string
	}{
		"the placement": {
			"place", []string{moment, askedMoment, judge, askedJudge, place, "engram auto write: 20000"},
			"engram auto place",
		},
		"the model's call of the placement": {
			"model3", []string{
				moment, askedMoment, judge, askedJudge, place, askedPlace, "engram auto write: 20000",
			}, "model call",
		},
	} {
		t.Run(name+" is ended at its row and the lessons are written at the top", func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			scenario := printingChain()
			scenario.Stalls = stage.stalls

			outcome := runCaptureDriver(t, scenario)

			g.Expect(outcome.stepsAndLimits()).To(Equal(stage.want), "the calls the mod made, with their time limits")

			if len(outcome.Calls) > 0 {
				last := outcome.Calls[len(outcome.Calls)-1]
				g.Expect(last.Input["placementFailure"]).To(ContainSubstring(stage.failed), "the failure names the stage")
				g.Expect(last.Input).To(HaveKeyWithValue("reply", "LESSONS"))
				g.Expect(last.Input).NotTo(HaveKey("failure"), "the capture did not fail")
			}
		})
	}

	t.Run("the stages together", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		scenario := printingChain()
		scenario.Stalls = "write"

		outcome := runCaptureDriver(t, scenario)

		// the longest chain: the ID, the three steps and their model calls, the writing that stalls, and the report of that
		g.Expect(outcome.Delays).To(ContainElement(5000), "the bound of the ID")
		g.Expect(outcome.limits()).To(HaveLen(8))
		g.Expect(outcome.limits()).To(HaveEach(BeNumerically(">", 0)), "every stage has a limit")

		total := 5000.0
		for _, limit := range outcome.limits() {
			total += limit
		}

		g.Expect(total).To(BeNumerically("<", 120000), "the stages together, in milliseconds")
	})
}

func Test793d2b1k_Mod_ARequestThatIsNotJSONIsAFailureOfTheJudgement(t *testing.T) {
	t.Parallel()

	for _, step := range []string{"moment", "judge"} {
		t.Run("a "+step+" that prints a request that is not JSON", func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			scenario := printingChain()
			scenario.set(step, map[string]any{"exitCode": 0, "stdout": `{"request": oops` + "\n"})

			outcome := runCaptureDriver(t, scenario)
			reports := outcome.reports()

			g.Expect(reports).To(HaveLen(1), "the report to the writing")
			g.Expect(outcome.modelCalls()).To(Equal(modelCallsBefore[step]), "the model was never asked for it")

			if len(reports) == 1 {
				g.Expect(reports[0]["failure"]).To(ContainSubstring("engram auto " + step))
			}
		})
	}
}

func Test793d2b1l_Mod_TheWorkAfterATurnIsStartedFromATimerWithNoCallInFlightWhenTheEventSettles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := printingChain()
	scenario.Moment = map[string]any{"hold": true, "exitCode": 0, "stdout": momentPrinted}

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Settle.Made).To(BeZero(), "the calls the hook of the event made")
	g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
	g.Expect(outcome.inputsOf("engram auto moment")).To(HaveLen(1), "the moment was run when the timers were let go")
	g.Expect(outcome.Sweeps).To(HaveLen(1), "the sweep was run when the timers were let go")
}

func Test793d2b1m_Mod_ALongFailureIsReportedWhole(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// a line of 1000 characters: the step at its beginning, a long path, and the cause at its end
	line := "Error: auto moment: reading " + strings.Repeat("a-long-directory/", 55) + "the record: permission denied"
	outcome := runCaptureDriver(t, captureScenario{
		Moment: map[string]any{"exitCode": 1, "stdout": "", "stderr": line + "\n"},
	})

	reports := outcome.reports()

	g.Expect(reports).To(HaveLen(1), "the report to the writing")

	if len(reports) != 1 {
		return
	}

	failure, isText := reports[0]["failure"].(string)

	g.Expect(isText).To(BeTrue(), "the failure is text")
	g.Expect(utf8.RuneCountInString(failure)).To(BeNumerically(">", 1000), "the failure as reported")
	g.Expect(failure).To(Equal("engram auto moment exited with 1: "+line), "the failure, from the step to the cause")
}

func Test793d2c1a_Mod_SweepsTheConversationAfterATurnApartFromTheCapture(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := printingChain()
	scenario.Moment = map[string]any{"hold": true, "exitCode": 0, "stdout": momentPrinted}

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Held).To(BeFalse(), "the event's result was not given back while the sweep was running")
	g.Expect(outcome.Sweeps).To(HaveLen(1), "the sweep was run once")

	if len(outcome.Sweeps) != 1 {
		return
	}

	g.Expect(outcome.Sweeps[0].Step).To(Equal("engram auto sweep"))
	g.Expect(outcome.Sweeps[0].Input).To(Equal(map[string]any{"session": "a-session", "origin": "composer"}))
	g.Expect(outcome.Sweeps[0].WhileJudging).To(BeTrue(), "the sweep did not wait for the capture")
}

func Test793d2c1b_Mod_ASweepThatFailsIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	for name, sweep := range map[string]map[string]any{
		"a sweep that exits with 1": {"exitCode": 1, "stdout": "Error: no record of session\n"},
		"a sweep that is ended":     {"rejects": "the command ran for 90000 ms"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			outcome := runCaptureDriver(t, captureScenario{Sweep: sweep})

			reports := outcome.reports()

			g.Expect(reports).To(HaveLen(1), "the report to the writing")

			if len(reports) != 1 {
				return
			}

			g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
			g.Expect(reports[0]).To(HaveKeyWithValue("turn", "a-turn"))
			g.Expect(reports[0]).NotTo(HaveKey("reply"))
			g.Expect(reports[0]["failure"]).To(ContainSubstring("engram auto sweep"))
		})
	}
}

func Test793d2c1c_Mod_ASweepThatStallsIsEndedAtItsBoundAndReported(t *testing.T) {
	t.Parallel()

	t.Run("the call for the session's ID", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{Stalls: "id"})

		g.Expect(outcome.Delays).To(ContainElement(5000), "the mod waited 5 seconds for the ID")
		g.Expect(outcome.Sweeps).To(BeEmpty(), "and then did not sweep")
	})

	t.Run("the sweep", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{Stalls: "sweep"})

		g.Expect(outcome.Sweeps).To(HaveLen(2), "the sweep, and the run as ended")

		reports := outcome.stepsAndLimits()

		g.Expect(reports).To(ContainElement("engram auto write: 15000"), "the report of the failure")

		if len(outcome.Sweeps) == 2 {
			g.Expect(outcome.Sweeps[0].TimeoutMs).To(BeNumerically("==", 90000), "the bound of the sweep")
			g.Expect(outcome.Sweeps[1].TimeoutMs).To(BeNumerically("==", 5000), "the bound of the run as ended")
			g.Expect(5000+outcome.Sweeps[0].TimeoutMs+outcome.Sweeps[1].TimeoutMs+15000).To(BeNumerically("<", 120000),
				"the ID, the sweep, the run as ended and the report together, in milliseconds")
		}
	})
}

func Test793d2c1d_Mod_ATurnOfASubagentIsNotSwept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := printingChain()
	scenario.Agent = "a-subagent"

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Sweeps).To(BeEmpty(), "the sweep was never run")
}

func Test793d2c1e_Mod_ASweepThatIsEndedIsRunOnceMoreAsEnded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Sweep: map[string]any{"rejects": "the command ran for 90000 ms"},
		SweepEnded: map[string]any{
			"exitCode": 1, "stdout": "",
			"stderr": "auto sweep: the sweep was ended at its time limit; run engram ingest --auto\n",
		},
	})

	g.Expect(outcome.Sweeps).To(HaveLen(2), "the sweep, and the run as ended")

	if len(outcome.Sweeps) == 2 {
		g.Expect(outcome.Sweeps[0].Input).To(Equal(map[string]any{"session": "a-session", "origin": "composer"}))
		g.Expect(outcome.Sweeps[1].Input).To(Equal(map[string]any{
			"session": "a-session", "origin": "composer", "ended": true,
		}))
		g.Expect(outcome.Sweeps[1].TimeoutMs).To(BeNumerically("==", 5000), "the bound of the run as ended")
	}

	reports := outcome.reports()

	g.Expect(reports).To(HaveLen(1), "the report to the writing")

	if len(reports) == 1 {
		g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
		g.Expect(reports[0]).To(HaveKeyWithValue("turn", "a-turn"))
		g.Expect(reports[0]["failure"]).To(And(ContainSubstring("engram auto sweep exited with 1"),
			ContainSubstring("run engram ingest --auto")), "the failure, from the run as ended")
	}
}

func Test793d2c1f_Mod_ASweepThatGaveUpBeforeIsNotReportedAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Sweep:      map[string]any{"rejects": "the command ran for 90000 ms"},
		SweepEnded: map[string]any{"exitCode": 0, "stdout": ""},
	})

	g.Expect(outcome.Sweeps).To(HaveLen(2), "the sweep, and the run as ended")
	g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")
}

// Test855b92_Mod_ADeepDeliveryWritesWhenItsDeliveryIsEmptyAndNotWhenItFails runs the mod under node against a stand-in
// host whose delivery prints nothing, or exits with 1.
func Test855b92_Mod_ADeepDeliveryWritesWhenItsDeliveryIsEmptyAndNotWhenItFails(t *testing.T) {
	t.Parallel()

	steps := map[string]map[string]any{
		"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
		"recall": {"exitCode": 0, "stdout": aDeepStep()},
	}

	t.Run("an empty delivery", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		outcome := runDeliveryDriver(t, deliveryScenario{Steps: steps, Deliver: map[string]any{"exitCode": 0, "stdout": ""}})

		g.Expect(outcome.Passed.Context).To(BeEmpty(), "no block")

		writes := outcome.reports()

		g.Expect(writes).To(HaveLen(1), "the write")

		if len(writes) == 1 {
			g.Expect(writes[0]).To(HaveKey("cue"))
		}
	})

	t.Run("a delivery that fails", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		outcome := runDeliveryDriver(t, deliveryScenario{
			Steps: steps, Deliver: map[string]any{"exitCode": 1, "stdout": "", "stderr": "auto deliver: no\n"},
		})

		for _, report := range outcome.reports() {
			g.Expect(report).NotTo(HaveKey("cue"), "only the report of the failure, no write")
		}
	})
}

// Test855d2a1a_Mod_TheStepsRunInOrderEachGivenTheReplyAndTheCarryBefore runs the mod under node against a stand-in host
// that records every command the mod runs and every request it asks the model for, in order.
func Test855d2a1a_Mod_TheStepsRunInOrderEachGivenTheReplyAndTheCarryBefore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Steps: map[string]map[string]any{
			"moment":  {"exitCode": 0, "stdout": aStep(1)},
			"phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall":  {"exitCode": 0, "stdout": aStep(3)},
		},
		Deliver: map[string]any{"exitCode": 0, "stdout": "[engram memory] The lines below are notes.\n- a.md (s): b\n"},
		Replies: []string{"reply 1", "reply 2", "reply 3"},
	})

	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(Equal([]string{"[engram memory] The lines below are notes.\n- a.md (s): b"}),
		"the block, as context")
	g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")
	g.Expect(outcome.order()).To(Equal([]string{
		"engram auto moment", "model", "engram auto phrases", "model", "engram auto recall", "model",
		"engram auto deliver",
	}), "the steps and the model's calls, in order")

	if len(outcome.Calls) != 7 {
		return
	}

	base := map[string]any{"session": "a-session", "origin": "composer", "cue": "prompt", "held": []any{}}
	input := func(extra map[string]any) map[string]any {
		fields := map[string]any{}

		maps.Copy(fields, base)
		maps.Copy(fields, extra)

		return fields
	}

	g.Expect(outcome.Calls[0].Input).To(Equal(input(map[string]any{"text": "hello"})))
	g.Expect(outcome.Calls[1].Asked).To(Equal(requestOfStep(1)), "the request, exactly as the step printed it")
	g.Expect(outcome.Calls[1].Asked).NotTo(HaveKey("timeoutMs"), "no time limit in the request")
	g.Expect(outcome.Calls[1].Signal).To(BeTrue(), "the call can be aborted")
	g.Expect(outcome.Calls[2].Input).To(Equal(input(map[string]any{"reply": "reply 1", "carry": carryOfStep(1)})))
	g.Expect(outcome.Calls[3].Asked).To(Equal(requestOfStep(2)))
	g.Expect(outcome.Calls[4].Input).To(Equal(input(map[string]any{"reply": "reply 2", "carry": carryOfStep(2)})))
	g.Expect(outcome.Calls[5].Asked).To(Equal(requestOfStep(3)))
	g.Expect(outcome.Calls[6].Input).To(Equal(input(map[string]any{
		"text": "hello", "reply": "reply 3", "carry": carryOfStep(3),
	})))
	g.Expect(outcome.limitOf("engram auto moment")).To(BeNumerically("==", 120000), "the time limit of a step")
	g.Expect(outcome.limitOf("engram auto phrases")).To(BeNumerically("==", 120000))
	g.Expect(outcome.limitOf("engram auto recall")).To(BeNumerically("==", 120000))
	g.Expect(outcome.limitOf("engram auto deliver")).To(BeNumerically("==", 120000))
	g.Expect(outcome.Delays).To(ContainElements(5000, 120000), "the bounds of the call for the ID and of a wait")
}

// Test855d2a1b_Mod_AStepThatPrintsNothingLeavesTheLinesToTheDelivery runs the mod under node against a stand-in host
// whose delivery prints a line and no block, and whose steps print nothing in turn.
func Test855d2a1b_Mod_AStepThatPrintsNothingLeavesTheLinesToTheDelivery(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		steps map[string]map[string]any
		asked int
		order []string
	}{
		"the moment prints nothing": {
			steps: map[string]map[string]any{"moment": {"exitCode": 0, "stdout": ""}},
			asked: 0, order: []string{"engram auto moment", "engram auto deliver"},
		},
		"the phrases print nothing after a moment request": {
			steps: map[string]map[string]any{
				"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": ""},
			},
			asked: 1, order: []string{
				"engram auto moment", "model", "engram auto phrases", "engram auto deliver",
			},
		},
		"the recall prints nothing after the two requests": {
			steps: map[string]map[string]any{
				"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
				"recall": {"exitCode": 0, "stdout": ""},
			},
			asked: 2, order: []string{
				"engram auto moment", "model", "engram auto phrases", "model", "engram auto recall", "engram auto deliver",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			outcome := runDeliveryDriver(t, deliveryScenario{
				Steps:   test.steps,
				Deliver: map[string]any{"exitCode": 0, "stdout": "", "stderr": "20:32:25 wrote the note b.md\n"},
				Replies: []string{"reply 1", "reply 2", "reply 3"},
			})

			g.Expect(outcome.order()).To(Equal(test.order), "the steps and the model's calls, in order")
			g.Expect(outcome.modelCalls()).To(HaveLen(test.asked), "the requests the model was asked")
			g.Expect(outcome.Logged).To(Equal([]string{"20:32:25 wrote the note b.md"}), "the line shown to the user")
			g.Expect(outcome.Passed.Context).To(BeEmpty(), "no context beside the prompt")
			g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")

			deliveries := outcome.inputsOf("engram auto deliver")

			g.Expect(deliveries).To(HaveLen(1))

			if len(deliveries) == 1 {
				g.Expect(deliveries[0]).To(HaveKeyWithValue("text", "hello"))
				g.Expect(deliveries[0]).To(HaveKeyWithValue("cue", "prompt"))
				g.Expect(deliveries[0]).NotTo(HaveKey("reply"))
				g.Expect(deliveries[0]).NotTo(HaveKey("carry"))
			}
		})
	}
}

// Test855d2a1d_Mod_OutputThatIsNotARequestOrABlockFailsTheStep runs the mod under node against a stand-in host that
// records every command the mod runs and every request it asks the model for.
func Test855d2a1d_Mod_OutputThatIsNotARequestOrABlockFailsTheStep(t *testing.T) {
	t.Parallel()

	t.Run("a moment that prints what is not a request", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		outcome := runDeliveryDriver(t, deliveryScenario{Steps: map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": "Unknown command: auto\n"},
		}})

		g.Expect(outcome.modelCalls()).To(BeEmpty(), "the model was never asked")
		g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
		g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")

		reports := outcome.reports()

		g.Expect(reports).To(HaveLen(1), "the runs of auto write")

		if len(reports) == 1 {
			g.Expect(reports[0]["failure"]).To(ContainSubstring("engram auto moment"), "the failure names the step")
		}
	})

	t.Run("a delivery that prints what is not a block", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		outcome := runDeliveryDriver(t, deliveryScenario{Deliver: map[string]any{
			"exitCode": 0, "stdout": "Unknown command: auto\n",
		}})

		g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
		g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
		g.Expect(outcome.reports()).To(BeEmpty(), "no failure was reported")
	})
}

// Test855d2a1e_Mod_AModelThatDoesNotAnswerAndAWriteThatFailsAreReported runs the mod under node against a stand-in host
// whose model gives no text, rejects or is aborted at its bound, and whose writing exits with 1 the first time: a live
// model cannot be made to fail on cue.
func Test855d2a1e_Mod_AModelThatDoesNotAnswerAndAWriteThatFailsAreReported(t *testing.T) {
	t.Parallel()

	steps := map[string]map[string]any{
		"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
		"recall": {"exitCode": 0, "stdout": aDeepStep()},
	}

	for name, fate := range map[string]string{"gives no text": "empty", "rejects the call": "rejects",
		"is aborted at 2 minutes": "aborted"} {
		for call := range 3 {
			t.Run(fmt.Sprintf("a model that %s on its call %d", name, call+1), func(t *testing.T) {
				t.Parallel()

				g := NewWithT(t)
				fates := make([]string, 3)
				fates[call] = fate

				outcome := runDeliveryDriver(t, deliveryScenario{
					Steps: steps, Deliver: map[string]any{"exitCode": 0, "stdout": deepBlock + "\n"},
					ModelFates: fates, FastTimers: true,
				})

				g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
				g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
				g.Expect(outcome.inputsOf("engram auto deliver")).To(BeEmpty(), "the delivery was never run")

				reports := outcome.reports()

				g.Expect(reports).To(HaveLen(1), "the runs of auto write")

				if len(reports) == 1 {
					g.Expect(reports[0]["failure"]).To(ContainSubstring("the model call"), "the failure names the model call")
					g.Expect(reports[0]).NotTo(HaveKey("cue"), "the report is no write of a deep delivery")
				}
			})
		}
	}

	t.Run("a write that fails", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		outcome := runDeliveryDriver(t, deliveryScenario{
			Steps: steps, Deliver: map[string]any{"exitCode": 0, "stdout": deepBlock + "\n"},
			Writes: []map[string]any{{"exitCode": 1, "stdout": "", "stderr": "auto write: the vault is not there\n"}},
		})

		g.Expect(outcome.Passed.Context).To(Equal([]string{deepBlock}), "the block, as context")

		writes := outcome.reports()

		g.Expect(writes).To(HaveLen(2), "the write, and the run that reports its failure")

		if len(writes) == 2 {
			g.Expect(writes[0]).To(HaveKey("cue"))
			g.Expect(writes[1]["failure"]).To(And(ContainSubstring("the write after a deep delivery"),
				ContainSubstring("the vault is not there")), "the failure names the write")
		}
	})
}

// Test855d2a1f_Mod_ADeepDeliveryWritesAfterTheBlockFromATimer runs the mod under node against a stand-in host that
// counts the calls in flight when the hook settles.
func Test855d2a1f_Mod_ADeepDeliveryWritesAfterTheBlockFromATimer(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Steps: map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall": {"exitCode": 0, "stdout": aDeepStep()},
		},
		Deliver: map[string]any{"exitCode": 0, "stdout": deepBlock + "\n"},
		Replies: []string{"reply 1", "reply 2", "reply 3"},
	})

	g.Expect(outcome.Passed.Context).To(Equal([]string{deepBlock}), "the block, as context")
	g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
	g.Expect(outcome.Settle.Commands).To(Equal(7), "the calls made when the hook settled: no write among them")

	writes := outcome.inputsOf("engram auto write")

	g.Expect(writes).To(HaveLen(1), "the runs of auto write")

	if len(writes) == 1 {
		g.Expect(writes[0]).To(Equal(map[string]any{
			"session": "a-session", "cue": "prompt", "reply": "reply 3", "carry": carryOfDeepStep(),
		}), "the session's ID, the cue, the reply to the candidate request and the carry")
	}

	g.Expect(outcome.limitOf("engram auto write")).To(BeNumerically("==", 60000), "the time limit of the write")
}

// Test855d2a1g_Mod_AGlanceWritesNothing runs the mod under node against a stand-in host whose recall prints a request
// without the mark of a write. Mutation: a mod that runs the writing for every delivery fails it.
func Test855d2a1g_Mod_AGlanceWritesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Steps: map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall": {"exitCode": 0, "stdout": aStep(3)},
		},
		Deliver: map[string]any{"exitCode": 0, "stdout": deepBlock + "\n"},
	})

	g.Expect(outcome.Passed.Context).To(HaveLen(1), "the block, as context")
	g.Expect(outcome.reports()).To(BeEmpty(), "the runs of auto write")
}

// Test855d2a1h_Mod_ANoteOrPassageDeliveredEarlierIsLeftOutOfALaterBlock runs the mod under node against a stand-in host
// whose delivery prints blocks that share a note and a passage, one prompt after another: a live session cannot be made
// to print a block that names what it printed before. The passages' IDs hold spaces, and two of them begin alike.
func Test855d2a1h_Mod_ANoteOrPassageDeliveredEarlierIsLeftOutOfALaterBlock(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		framing = "[engram memory] The lines below are notes."
		note    = "- a.md (when s): the body"
		first   = "- My Docs/a.md#1 (passage): the first passage"
		second  = "- My Docs/b.md#2 (passage): the second passage"
	)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Prompts: 2,
		Delivers: []map[string]any{
			{"exitCode": 0, "stdout": framing + "\n" + note + "\n" + first + "\n"},
			{"exitCode": 0, "stdout": framing + "\n" + note + "\n" + second + "\n" + first + "\n"},
		},
	})

	g.Expect(outcome.All).To(HaveLen(2))

	if len(outcome.All) != 2 {
		return
	}

	g.Expect(outcome.All[0].Context).To(Equal([]string{framing + "\n" + note + "\n" + first}), "the first block, whole")
	g.Expect(outcome.All[1].Context).To(Equal([]string{framing + "\n" + second}),
		"the second block, less the note and the passage the first held: the passage that begins alike stays")

	deliveries := outcome.inputsOf("engram auto deliver")

	g.Expect(deliveries).To(HaveLen(2))

	if len(deliveries) == 2 {
		g.Expect(deliveries[0]["held"]).To(BeEmpty(), "the IDs held before the first")
		g.Expect(deliveries[1]["held"]).To(Equal([]any{"a.md", "My Docs/a.md#1"}), "the IDs held before the second, whole")
	}
}

// Test855d2a1i_Mod_TheLineForTheUserNamesTheNotesThatAreLeft runs the mod under node against a stand-in host whose
// delivery prints the lines for the user beside blocks that share a note.
func Test855d2a1i_Mod_TheLineForTheUserNamesTheNotesThatAreLeft(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const framing = "[engram memory] The lines below are notes."

	outcome := runDeliveryDriver(t, deliveryScenario{
		Prompts: 3,
		Delivers: []map[string]any{
			{"exitCode": 0, "stdout": framing + "\n- a.md (s): x\n", "stderr": "delivered the note a.md\n"},
			{
				"exitCode": 0, "stdout": framing + "\n- a.md (s): x\n- b.md (s): y\n- My Docs/c.md#3 (passage): z\n",
				"stderr": "delivered the notes a.md, b.md and the passage My Docs/c.md#3\n",
			},
			{
				"exitCode": 0, "stdout": framing + "\n- a.md (s): x\n",
				"stderr": "delivered the note a.md\n20:32:25 wrote the note d.md\n",
			},
		},
	})

	g.Expect(outcome.Logged).To(Equal([]string{
		"delivered the note a.md",
		"delivered the note b.md and the passage My Docs/c.md#3",
		"20:32:25 wrote the note d.md",
	}), "the lines the user was shown: none for the note a block left out, the others whole")
}

// Test855d2a1j_Mod_DeliveriesOfOneSessionRunOneAfterAnotherEachGivenWhatTheOneBeforeHeld runs the mod under node
// against a stand-in host whose delivery takes a while and prints the same block each time, for two prompts raised at
// once.
func Test855d2a1j_Mod_DeliveriesOfOneSessionRunOneAfterAnotherEachGivenWhatTheOneBeforeHeld(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	block := "[engram memory] The lines below are notes.\n- a.md (s): x\n"
	outcome := runDeliveryDriver(t, deliveryScenario{
		Prompts: 2, Concurrent: true, DeliverDelayMs: 100,
		Deliver: map[string]any{"exitCode": 0, "stdout": block, "stderr": "delivered the note a.md\n"},
	})

	g.Expect(outcome.All).To(HaveLen(2))

	if len(outcome.All) != 2 {
		return
	}

	given := 0

	for _, passed := range outcome.All {
		if len(passed.Context) > 0 {
			given++
		}
	}

	g.Expect(given).To(Equal(1), "how many of the two prompts were given the note")
	g.Expect(outcome.Logged).To(Equal([]string{"delivered the note a.md"}), "the lines the user was shown")

	deliveries := outcome.inputsOf("engram auto deliver")

	g.Expect(deliveries).To(HaveLen(2))

	if len(deliveries) == 2 {
		g.Expect(deliveries[0]["held"]).To(BeEmpty(), "the IDs held when the first delivery began")
		g.Expect(deliveries[1]["held"]).To(Equal([]any{"a.md"}), "the IDs held when the second began: the first's")
	}
}

// Test855d2a1k_Mod_TwoDeepDeliveriesOfASessionWriteOneAfterTheOther runs the mod under node against a stand-in host
// whose writing takes a while: the second write starts when the first has ended (855b29).
func Test855d2a1k_Mod_TwoDeepDeliveriesOfASessionWriteOneAfterTheOther(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Steps: map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall": {"exitCode": 0, "stdout": aDeepStep()},
		},
		Deliver: map[string]any{"exitCode": 0, "stdout": deepBlock + "\n"}, Prompts: 2, WriteDelayMs: 20,
	})

	g.Expect(outcome.WriteEvents).To(Equal([]string{"start", "end", "start", "end"}), "the writes, one after the other")
}

func Test855d2b1a_Mod_APlacementThatPrintsNothingLeavesTheLessonsToBeWrittenAtTheTop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scenario := printingChain()
	scenario.Place = map[string]any{"exitCode": 0, "stdout": ""}

	outcome := runCaptureDriver(t, scenario)

	g.Expect(outcome.modelCalls()).To(HaveLen(2), "the model was asked twice")

	writes := outcome.inputsOf("engram auto write")

	g.Expect(writes).To(Equal([]map[string]any{{"session": "a-session", "turn": "a-turn", "reply": "LESSONS"}}),
		"the reply to the lessons request, and no carry and no placement reply: the placement printed neither")
}

func Test855d2b1b_Mod_APlacementThatFailsLeavesTheLessonsToBeWrittenAtTheTopAndRecorded(t *testing.T) {
	t.Parallel()

	failures := map[string]struct {
		place  map[string]any
		models []map[string]any
		calls  int
	}{
		"a placement that exits with 1": {place: map[string]any{"exitCode": 1, "stdout": "boom\n"}, calls: 2},
		"a placement that prints what is not a request": {
			place: map[string]any{"exitCode": 0, "stdout": "Unknown command: auto\n"}, calls: 2,
		},
		"a placement that is ended": {place: map[string]any{"rejects": "the command ran for 20000 ms"}, calls: 2},
		"a model that answers the placement with an API error": {
			models: []map[string]any{nil, nil, {"isAnswered": false, "reason": "api-error", "status": 529}}, calls: 3,
		},
		"a model that rejects the placement": {
			models: []map[string]any{nil, nil, {"rejects": "the cap is refused"}}, calls: 3,
		},
	}

	for name, failure := range failures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			scenario := printingChain()

			if failure.place != nil {
				scenario.Place = failure.place
			}

			for call, model := range failure.models {
				if model != nil {
					scenario.Models[call] = model
				}
			}

			outcome := runCaptureDriver(t, scenario)

			g.Expect(outcome.modelCalls()).To(HaveLen(failure.calls), "the model calls made before the placement failed")

			writes := outcome.inputsOf("engram auto write")

			g.Expect(writes).To(HaveLen(1), "the one writing: of the lessons, and not of a failure")

			if len(writes) != 1 {
				return
			}

			g.Expect(writes[0]).To(And(HaveKeyWithValue("session", "a-session"), HaveKeyWithValue("turn", "a-turn"),
				HaveKeyWithValue("reply", "LESSONS")))
			g.Expect(writes[0]["placementFailure"]).To(ContainSubstring("place"), "the failure names the placement")
			g.Expect(writes[0]).NotTo(HaveKey("failure"), "the capture did not fail")
			g.Expect(writes[0]).NotTo(HaveKey("carry"), "no carry: the placement gave none")
			g.Expect(writes[0]).NotTo(HaveKey("placement"), "no placement reply")
		})
	}
}

// Test855d2c1a_Mod_ANoteArrivesWithTheCallAfterAFailureAndAfterABatch runs the mod under node against a stand-in host
// that raises the three events of a tool in the turn of a prompt.
func Test855d2c1a_Mod_ANoteArrivesWithTheCallAfterAFailureAndAfterABatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Delivers: []map[string]any{
			{"exitCode": 0, "stdout": noteBlock("a.md"), "stderr": "delivered the note a.md\n"},
			{"exitCode": 0, "stdout": noteBlock("b.md"), "stderr": "delivered the note b.md\n"},
			{"exitCode": 0, "stdout": noteBlock("c.md"), "stderr": "delivered the note c.md\n"},
		},
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "make ci"},
				Answer: map[string]any{"result": "ok", "context": []string{"another hook's"}}},
			failureEvent("a-session", "t1", false),
			batchEvent("a-session", "t2", "t3"),
		},
	}))

	g.Expect(outcome.Answers).To(HaveLen(3))

	if len(outcome.Answers) != 3 {
		return
	}

	g.Expect(outcome.Answers[0]).To(Equal(map[string]any{
		"result": "ok", "context": []any{"another hook's", strings.TrimSpace(noteBlock("a.md"))},
	}), "the block, one more entry of the result's context")
	g.Expect(outcome.Answers[1]).To(Equal(additionalContextOf("b.md")))
	g.Expect(outcome.Answers[2]).To(Equal(additionalContextOf("c.md")))
	g.Expect(outcome.Logged).To(
		Equal([]string{"delivered the note a.md", "delivered the note b.md", "delivered the note c.md"}),
		"the line the user was shown each time")

	moments := outcome.inputsOf("engram auto moment")

	g.Expect(moments).To(Equal([]map[string]any{
		momentOf("before-tool", []any{}, map[string]any{"tool": "Bash", "input": map[string]any{"command": "make ci"}}),
		momentOf("failure", []any{"a.md"}, map[string]any{
			"tool": "Bash", "input": map[string]any{"command": "make ci"}, "error": "exit 1",
		}),
		momentOf("batch-done", []any{"a.md", "b.md"}, map[string]any{"calls": []any{
			map[string]any{"tool": "Bash", "input": map[string]any{"command": "a"}, "response": "ok"},
			map[string]any{
				"tool": "Read", "input": map[string]any{"file_path": "b"}, "response": map[string]any{"lines": float64(3)},
			},
		}}),
	}), "the cue, the event, the text the turn began with and the session, for each")
}

// Test855d2c1d_Mod_TheSessionsIDOfACallAboutToRunComesFromTheHost runs the mod under node against a stand-in host that
// gives a session ID of its own: a tool.call event has none, and the other two events carry it.
func Test855d2c1d_Mod_TheSessionsIDOfACallAboutToRunComesFromTheHost(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		SessionID: "the-hosts-session",
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "ls"}},
			failureEvent("the-events-session", "t2", false),
		},
	}))

	moments := outcome.inputsOf("engram auto moment")

	g.Expect(moments).To(HaveLen(2))

	if len(moments) == 2 {
		g.Expect(moments[0]["session"]).To(Equal("the-hosts-session"), "the ID the host gave")
		g.Expect(moments[1]["session"]).To(Equal("the-events-session"), "the ID the event carries")
	}
}

// Test855d2c1e_Mod_ASubagentsLoopIsLeftAlone runs the mod under node against a stand-in host that raises the three
// events from a subagent's loop.
func Test855d2c1e_Mod_ASubagentsLoopIsLeftAlone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	failure := failureEvent("a-session", "t1", false)
	failure.Event["agent_id"] = "a-subagent"

	batch := batchEvent("a-session", "t2", "t3")
	batch.Event["agent_id"] = "a-subagent"

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{
				"tool": "Bash", "tool_use_id": "t0", "command": "ls", "agentId": "a-subagent",
			}, Answer: map[string]any{"result": "ok"}},
			failure, batch,
		},
	}))

	g.Expect(outcome.Calls).To(BeEmpty(), "the steps and the model's calls")
	g.Expect(outcome.Answers).To(Equal([]map[string]any{{"result": "ok"}, {}, {}}), "the result of next(e) each time")
}

// Test855d2c1f_Mod_AMomentThatIsNotOneAnswersTheEvent runs the mod under node against a stand-in host whose moment, and
// then whose recall, prints nothing.
func Test855d2c1f_Mod_AMomentThatIsNotOneAnswersTheEvent(t *testing.T) {
	t.Parallel()

	events := []eventRaised{
		{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "ls"},
			Answer: map[string]any{"result": "ok"}},
		failureEvent("a-session", "t2", false),
		batchEvent("a-session", "t3", "t4"),
	}
	want := []map[string]any{{"result": "ok"}, {}, {}}

	for name, steps := range map[string]map[string]map[string]any{
		"the moment prints nothing": {},
		"the recall prints nothing": {
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			outcome := runDeliveryDriver(t, deliveryScenario{Steps: steps, Replies: []string{"r1", "r2"}, Events: events})

			g.Expect(outcome.Answers).To(Equal(want), "the result of next(e), with no additional context")
			g.Expect(outcome.reports()).To(BeEmpty(), "the reports to the writing")
		})
	}
}

// Test855d2c1g_Mod_AFailedOrStalledStepAnswersTheEventAsItWasAndIsReported runs the mod under node against a stand-in
// host whose moment exits with 1, and whose model never settles unless its signal aborts it.
func Test855d2c1g_Mod_AFailedOrStalledStepAnswersTheEventAsItWasAndIsReported(t *testing.T) {
	t.Parallel()

	events := []eventRaised{failureEvent("a-session", "t1", false)}

	t.Run("a moment that exits with 1", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		outcome := runDeliveryDriver(t, deliveryScenario{
			Steps:      map[string]map[string]any{"moment": {"exitCode": 1, "stdout": "", "stderr": "boom\n"}},
			FastTimers: true, Events: events,
		})

		g.Expect(outcome.Answers).To(Equal([]map[string]any{{}}), "the result of next(e)")

		reports := outcome.reports()

		g.Expect(reports).To(HaveLen(1))

		if len(reports) == 1 {
			g.Expect(reports[0]["session"]).To(Equal("a-session"))
			g.Expect(reports[0]["turn"]).To(BeEmpty(), "no turn")
			g.Expect(reports[0]["failure"]).To(ContainSubstring("engram auto moment"))
		}
	})

	t.Run("a model that never answers", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
			ModelNeverSettles: true, FastTimers: true, Events: events,
		}))

		g.Expect(outcome.Answers).To(Equal([]map[string]any{{}}), "the result of next(e)")
		g.Expect(outcome.Delays).To(ContainElement(120000), "the bound of the wait")

		reports := outcome.reports()

		g.Expect(reports).To(HaveLen(1))

		if len(reports) == 1 {
			g.Expect(reports[0]["failure"]).To(ContainSubstring("the model call"))
			g.Expect(reports[0]["turn"]).To(BeEmpty(), "no turn")
		}
	})
}

// Test855d2c1h_Mod_ChainsOfOneSessionRunAtOnceAndANoteIsDeliveredOnce runs the mod under node against a stand-in host
// whose moments do not answer until both have started, and whose delivery prints the same note for both.
func Test855d2c1h_Mod_ChainsOfOneSessionRunAtOnceAndANoteIsDeliveredOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Concurrent: true, MomentBarrier: 2, DeliverDelayMs: 100,
		Deliver: map[string]any{"exitCode": 0, "stdout": noteBlock("a.md")},
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "a"},
				Answer: map[string]any{"result": "ok"}},
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t2", "command": "b"},
				Answer: map[string]any{"result": "ok"}},
		},
	}))

	g.Expect(outcome.BarrierMet).To(BeTrue(), "the moment was run for both events before either was let answer")
	g.Expect(outcome.Answers).To(HaveLen(2))

	given := 0

	for _, answer := range outcome.Answers {
		if _, has := answer["context"]; has {
			given++
		}
	}

	g.Expect(given).To(Equal(1), "how many of the two answers hold the block")
}

// Test855d2c1k_Mod_AUsersInterruptAndABatchOfFailuresJudgedAlreadyAreLeftAlone runs the mod under node against a
// stand-in host that raises an interrupt, a failure and a batch of that failure alone.
func Test855d2c1k_Mod_AUsersInterruptAndABatchOfFailuresJudgedAlreadyAreLeftAlone(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Events: []eventRaised{
			failureEvent("a-session", "t1", true),
			failureEvent("a-session", "t2", false),
			{Hook: "classic.PostToolBatch", Event: map[string]any{"session_id": "a-session", "tool_calls": []any{
				map[string]any{"tool_name": "Bash", "tool_input": map[string]any{}, "tool_use_id": "t2", "tool_response": "x"},
			}}},
		},
	}))

	moments := outcome.inputsOf("engram auto moment")

	g.Expect(moments).To(HaveLen(1), "the moments run: the second failure, once")

	if len(moments) == 1 {
		g.Expect(moments[0]["cue"]).To(Equal("failure"))
	}

	g.Expect(outcome.Answers).To(Equal([]map[string]any{{}, {}, {}}), "the result of next(e) each time")
}

// Test855d2c1l_Mod_ADeniedCallKeepsNoBlockAndItsNotesAreDeliveredLater runs the mod under node against a stand-in host
// whose hooks beneath deny a call, and then raises a prompt.
func Test855d2c1l_Mod_ADeniedCallKeepsNoBlockAndItsNotesAreDeliveredLater(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Delivers: []map[string]any{
			{"exitCode": 0, "stdout": noteBlock("a.md")}, {"exitCode": 0, "stdout": noteBlock("a.md")},
		},
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "rm"},
				Answer: map[string]any{"deny": "no"}},
			{Hook: "prompt.submit", Event: map[string]any{"text": "again", "context": []any{}, "origin": map[string]any{
				"kind": "composer"}}},
		},
	}))

	g.Expect(outcome.Answers).To(HaveLen(2))

	if len(outcome.Answers) != 2 {
		return
	}

	g.Expect(outcome.Answers[0]).To(Equal(map[string]any{"deny": "no"}), "the denial, and no block")
	g.Expect(outcome.Answers[1]["context"]).To(Equal([]any{strings.TrimSpace(noteBlock("a.md"))}),
		"the note, delivered at the later prompt")

	deliveries := outcome.inputsOf("engram auto deliver")

	g.Expect(deliveries).To(HaveLen(2))

	if len(deliveries) == 2 {
		g.Expect(deliveries[1]["held"]).To(BeEmpty(), "the denied call held nothing")
	}
}

// Test855d2c1m_Mod_ADeepDeliveryWithinATurnWritesWithItsCue runs the mod under node against a stand-in host whose
// recall prints a deep request, for a failure.
func Test855d2c1m_Mod_ADeepDeliveryWithinATurnWritesWithItsCue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{
		Steps: map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall": {"exitCode": 0, "stdout": aDeepStep()},
		},
		Replies: []string{"r1", "r2", "r3"},
		Deliver: map[string]any{"exitCode": 0, "stdout": noteBlock("a.md")},
		Events:  []eventRaised{failureEvent("a-session", "t1", false)},
	})

	g.Expect(outcome.Answers).To(Equal([]map[string]any{additionalContextOf("a.md")}), "the block, as additional context")
	g.Expect(outcome.inputsOf("engram auto write")).To(Equal([]map[string]any{{
		"session": "a-session", "cue": "failure", "reply": "r3", "carry": carryOfDeepStep(),
	}}), "the write of the deep delivery: the cue of the event, the reply to the candidates and the carry")
}

// Test855d2c1n_Mod_ACallRunsWhileItsChainDoes runs the mod under node against a stand-in host whose moment does not
// finish until the call is let run.
func Test855d2c1n_Mod_ACallRunsWhileItsChainDoes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		MomentAfterNext: true,
		Deliver:         map[string]any{"exitCode": 0, "stdout": noteBlock("a.md")},
		Events: []eventRaised{
			{Hook: "tool.call", Event: map[string]any{"tool": "Bash", "tool_use_id": "t1", "command": "ls"},
				Answer: map[string]any{"result": "ok"}},
		},
	}))

	g.Expect(outcome.NextBeforeMoment).To(BeTrue(), "the call was let run before the moment finished")
	g.Expect(outcome.Answers).To(Equal([]map[string]any{{
		"result": "ok", "context": []any{strings.TrimSpace(noteBlock("a.md"))},
	}}), "the block, once the chain was over")
}

// Test855d2c1o_Mod_ACallThatIsAMemoryCommandIsNotLookedUpFor runs the mod under node against a stand-in host that
// raises the three events of a call of the agent's own to the memory: a Bash command of engram, and the skills of it.
func Test855d2c1o_Mod_ACallThatIsAMemoryCommandIsNotLookedUpFor(t *testing.T) {
	t.Parallel()

	calls := map[string]struct {
		tool  string
		input map[string]any
	}{
		"engram query":      {"Bash", map[string]any{"command": "engram query --lazy-chunks --text=\"x\""}},
		"engram show":       {"Bash", map[string]any{"command": "engram show a-note"}},
		"engram show-chunk": {"Bash", map[string]any{"command": "engram show-chunk s#1"}},
		"engram recall":     {"Bash", map[string]any{"command": "engram recall glance"}},
		"engram learn":      {"Bash", map[string]any{"command": "cd /x && engram learn feedback --situation s"}},
		"engram amend":      {"Bash", map[string]any{"command": "engram amend a-note"}},
		"skill recall":      {"Skill", map[string]any{"skill": "recall"}},
		"skill learn":       {"Skill", map[string]any{"skill": "learn", "args": "x"}},
		"skill curate":      {"Skill", map[string]any{"skill": "curate"}},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			toolCall := map[string]any{"tool": call.tool, "tool_use_id": "t1"}

			for key, value := range call.input {
				toolCall[key] = value
			}

			failure := eventRaised{Hook: "classic.PostToolUseFailure", Event: map[string]any{
				"session_id": "a-session", "tool_name": call.tool, "tool_input": call.input, "tool_use_id": "t2",
				"error": "exit 1",
			}}
			batch := eventRaised{Hook: "classic.PostToolBatch", Event: map[string]any{
				"session_id": "a-session", "tool_calls": []any{
					map[string]any{"tool_name": call.tool, "tool_input": call.input, "tool_use_id": "t3", "tool_response": "ok"},
					map[string]any{"tool_name": call.tool, "tool_input": call.input, "tool_use_id": "t4", "tool_response": "ok"},
				},
			}}

			outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
				Deliver: map[string]any{"exitCode": 0, "stdout": noteBlock("a.md")},
				Events: []eventRaised{
					{Hook: "tool.call", Event: toolCall, Answer: map[string]any{"result": "ok"}}, failure, batch,
				},
			}))

			g.Expect(outcome.Calls).To(BeEmpty(), "the steps and the model's calls")
			g.Expect(outcome.Answers).To(Equal([]map[string]any{{"result": "ok"}, {}, {}}), "the result of next(e) each time")
			g.Expect(outcome.Logged).To(BeEmpty(), "nothing shown to the user")
		})
	}
}

// Test855d2c1o_Mod_TheMemoryCommandsOfTheModAreThoseTheBinaryHolds: the commands the mod skips and the commands the
// binary holds the text and result of are one list, which the mod writes in its source (855b122).
func Test855d2c1o_Mod_TheMemoryCommandsOfTheModAreThoseTheBinaryHolds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	source, err := os.ReadFile(filepath.Join(repositoryRoot, "agent-instructions", "mods", "engram-memory", "hooks",
		"register.js"))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	declared := regexp.MustCompile(`const MEMORY_COMMANDS = \[([^\]]*)\]`).FindSubmatch(source)

	g.Expect(declared).NotTo(BeNil(), "the list of the memory commands in the mod")

	if declared == nil {
		return
	}

	var commands []string

	for _, word := range regexp.MustCompile(`'([^']*)'`).FindAllSubmatch(declared[1], -1) {
		commands = append(commands, string(word[1]))
	}

	g.Expect(commands).To(Equal(auto.MemoryCommands), "the list of the mod and the list of the binary")
}

// Test855d2c1p_Mod_ABatchWithAnotherCallAmongTheMemoryCommandsIsLookedUpFor runs the mod under node against a stand-in
// host that raises a batch of an engram query and a Read, and a call of the agent's that only mentions engram.
func Test855d2c1p_Mod_ABatchWithAnotherCallAmongTheMemoryCommandsIsLookedUpFor(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	query := map[string]any{"command": "engram query --text=x"}
	outcome := runDeliveryDriver(t, toolScenario(deliveryScenario{
		Delivers: []map[string]any{
			{"exitCode": 0, "stdout": noteBlock("a.md")}, {"exitCode": 0, "stdout": noteBlock("b.md")},
		},
		Events: []eventRaised{
			{Hook: "classic.PostToolBatch", Event: map[string]any{"session_id": "a-session", "tool_calls": []any{
				map[string]any{"tool_name": "Bash", "tool_input": query, "tool_use_id": "t1", "tool_response": "ok"},
				map[string]any{
					"tool_name": "Read", "tool_input": map[string]any{"file_path": "b"}, "tool_use_id": "t2",
					"tool_response": "ok",
				},
			}}},
			{Hook: "tool.call", Event: map[string]any{
				"tool": "Bash", "tool_use_id": "t3", "command": "git log --oneline  # about engram",
			}, Answer: map[string]any{"result": "ok"}},
		},
	}))

	g.Expect(outcome.Answers).To(Equal([]map[string]any{
		additionalContextOf("a.md"),
		{"result": "ok", "context": []any{strings.TrimSpace(noteBlock("b.md"))}},
	}), "the mixed batch and the other command are each looked up for")

	cues := []any{}

	for _, moment := range outcome.inputsOf("engram auto moment") {
		cues = append(cues, moment["cue"])
	}

	g.Expect(cues).To(Equal([]any{"batch-done", "before-tool"}), "the cues the steps were run for")
}

// TestMods_EveryJavaScriptModuleParses: the real mods parse under the real node, and the real node rejects a broken
// module, which the stand-in node of the unit tests cannot show (793b55).
func TestMods_EveryJavaScriptModuleParses(t *testing.T) {
	t.Parallel()

	t.Run("the mods of the source", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		modules := modulesOf(t, filepath.Join(repositoryRoot, "agent-instructions", "mods"))

		g.Expect(modules).NotTo(BeEmpty(), "the JavaScript of the mods")
		g.Expect(checkModuleSyntax(t.Context(), modules, findNode, parseModule)).To(Succeed())
	})

	t.Run("a broken module", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		err := checkModuleSyntax(t.Context(),
			map[string]string{"hooks/broken.js": "export function register( {\n"}, findNode, parseModule)

		g.Expect(err).To(MatchError(ContainSubstring("hooks/broken.js")))
		g.Expect(err).To(MatchError(ContainSubstring("SyntaxError")))
	})
}

// unexported constants.
const (
	// captureDriver runs the hooks of a turn of the module named by MOD against a host that behaves as SCENARIO says: a
	// prompt of an origin is raised and the turn starts (unless a subagent ran the turn), and the turn completes, or the
	// steps of its script are done in order. Its moment, judgement and placement each answer once the driver lets them
	// (hold), exit with a code and print a text, or reject (a step not given prints nothing), and its model answers
	// each call with the next of the results, or with the one result, or rejects. It records the order and the input of
	// what the module asks of it, except the commands of the flow of a prompt and the sweep, and prints what the hook of
	// the end of the turn returned, whether that hook was still waiting when the driver gave up on it, the calls, the
	// sweeps, which it records apart with whether the capture was still held when each was run, and the calls the hook
	// made and left in flight when it settled. The mod's timers run once the hook has settled.
	captureDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const scenario = JSON.parse(process.env.SCENARIO)
const calls = []
const sweeps = []
let letCaptureGo
let captureReleased = false
const gate = new Promise((resolve) => { letCaptureGo = resolve })
function releaseCapture() {
  captureReleased = true
  letCaptureGo()
}

// a stage that stalls: the timers of the mod fire at once and are recorded, and the driver waits with the real one
const wait = globalThis.setTimeout
const delays = []
if (scenario.stalls) {
  globalThis.setTimeout = (callback, delay) => { delays.push(delay); return wait(callback, 0) }
}

// the capture and the sweep are over when the calls have not changed for 50 ms
async function settled() {
  for (let last = -1, stable = 0, tick = 0; tick < 300 && stable < 5; tick++) {
    await new Promise((resolve) => wait(resolve, 10))
    const total = calls.length + sweeps.length
    stable = total === last ? stable + 1 : 0
    last = total
  }
}

// the host ends a command at the time limit it was given, and the model's call at the limit of its request; the
// first run of the writing is the one that stalls, and the nth call of the model when the scenario says
let writes = 0
let asked = 0
const host = {
  session: { id: () => (scenario.stalls === 'id' ? new Promise(() => {}) : Promise.resolve('a-session')) },
  process: {
    run: async (argv, init) => {
      const input = JSON.parse(init.stdin)
      const step = argv[2]
      if ((step === 'moment' && input.cue !== 'turn-end') || ['phrases', 'recall', 'deliver'].includes(step)) {
        return { exitCode: 0, stdout: '' }
      }
      if (step === 'sweep') {
        sweeps.push({ step: argv.join(' '), input, timeoutMs: init.timeoutMs, whileJudging: !captureReleased })
        if (input.ended && scenario.sweepEnded) return scenario.sweepEnded
        if (scenario.stalls === 'sweep') throw new Error('the command ran too long')
        if (scenario.sweep?.rejects) throw new Error(scenario.sweep.rejects)
        return scenario.sweep ?? { exitCode: 0, stdout: '' }
      }
      calls.push({ step: argv.join(' '), input, timeoutMs: init.timeoutMs })
      const ranTooLong = step === scenario.stalls && (step !== 'write' || ++writes === 1)
      if (ranTooLong) throw new Error('the command ran too long')
      const answer = scenario[step] ?? { exitCode: 0, stdout: '' }
      if (answer.rejects) throw new Error(answer.rejects)
      if (answer.hold) {
        await gate
        return { exitCode: answer.exitCode, stdout: answer.stdout }
      }
      return answer
    },
  },
  model: {
    complete: async (request) => {
      calls.push({ step: 'model', asked: request })
      asked++
      if (scenario.stalls === 'model' + asked) return { isAnswered: false, reason: 'aborted' }
      const result = scenario.models?.[asked - 1] ?? scenario.model
      if (result.rejects) throw new Error(result.rejects)
      return result
    },
  },
  ui: { log: () => {} },
}

// the host holds the next event of a mod until the calls its event left in flight settle, so the calls are counted when
// they are made and while they are in flight; its timers run only when the driver lets them
let made = 0
let inFlight = 0
function counted(call) {
  return (...args) => {
    made++
    inFlight++
    const done = () => { inFlight-- }
    const running = call(...args)
    Promise.resolve(running).then(done, done)
    return running
  }
}
host.session.id = counted(host.session.id)
host.process.run = counted(host.process.run)
host.model.complete = counted(host.model.complete)
const timers = []
host.clock = { after: (_milliseconds, run) => { timers.push(run) } }
function fireTimers() {
  for (const run of timers.splice(0)) void run()
}

if (scenario.script) {
  for (const step of scenario.script) {
    if (step.submit) {
      const raised = { text: step.submit.text, context: [], origin: { kind: step.submit.origin }, wait: false }
      await handlers['prompt.submit'](host, raised, async (event) => event)
    }
    if (step.start) {
      await handlers['turn.start'](host, step.start, async (event) => ({ turnId: event.turnId }))
    }
    if (step.complete) {
      const completed = { reason: 'answer', ...step.complete }
      await handlers['turn.complete'](host, completed, async (event) => ({ text: event.answer }))
      fireTimers()
    }
  }
  releaseCapture()
  await settled()
  console.log(JSON.stringify({ returned: {}, held: false, calls, sweeps, delays }))
  process.exit(0)
}

const agent = scenario.agent ? { agentId: scenario.agent } : {}
const origin = { kind: scenario.origin || 'composer' }
const text = scenario.text || 'hello'
if (!scenario.agent) {
  await handlers['prompt.submit'](host, { text, context: [], origin, wait: false }, async (event) => event)
}
if (handlers['turn.start']) {
  const started = { text, turnId: 'a-turn', ...agent }
  await handlers['turn.start'](host, started, async (event) => ({ turnId: event.turnId }))
}

const waited = Symbol('waited')
const madeBefore = made
const hook = handlers['turn.complete'](host, { answer: 'Understood.', turnId: 'a-turn', reason: 'answer', ...agent },
  async (event) => ({ text: event.answer }))
let giveUp
const gaveUp = new Promise((resolve) => { giveUp = wait(() => resolve(waited), 500) })
const returned = await Promise.race([hook, gaveUp])
clearTimeout(giveUp)
const held = returned === waited
const settle = { made: made - madeBefore, inFlight }

fireTimers()

// a capture that is held lets the sweep start while it runs
for (let tick = 0; scenario.moment?.hold && tick < 50 && sweeps.length === 0; tick++) {
  await new Promise((resolve) => wait(resolve, 10))
}

releaseCapture()

await settled()

console.log(JSON.stringify({ returned: held ? {} : returned, held, calls, sweeps, delays, settle }))
`
	// deepBlock is the block the delivery prints in the tests of the write after a deep delivery.
	deepBlock = "[engram memory] The lines.\n- a.md (s): b"
	// deliveryDriver runs the prompt.submit hook of the module named by MOD against a host that behaves as SCENARIO says:
	// each step of the chain exits with a code and prints a text (a step that is not named prints nothing), or runs
	// until its time limit; its model answers with the next of the replies, or never settles unless its signal aborts
	// it. The timers of the module are recorded with the delay they were asked for, and fire at once when the scenario
	// says so; the host counts the commands the module has run and the calls in flight when the hook settles, runs the
	// timers of the module only then, and prints what the hook let through, the commands and the requests the module
	// made in order with their input and time limit, the delays the timers were asked for, the count, and the lines the
	// module asked the host to log, in order.
	deliveryDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const scenario = JSON.parse(process.env.SCENARIO)
const calls = []
const delays = []
const wait = globalThis.setTimeout
// the timers of the module are recorded, and fire at once when the scenario says so
globalThis.setTimeout = (callback, delay, ...rest) => {
  delays.push(delay)
  return scenario.fastTimers ? wait(callback, 0) : wait(callback, delay, ...rest)
}

// the host holds the next event of a mod until the calls its event left in flight settle
let inFlight = 0
function counted(call) {
  return (...args) => {
    inFlight++
    const done = () => { inFlight-- }
    const running = call(...args)
    Promise.resolve(running).then(done, done)
    return running
  }
}
const timers = []
const logged = []
let replied = 0
let nth = 0
let delivered = 0
let written = 0
let setup = false
let started = 0
let barrierMet = false
let nextSeen = false
let nextBeforeMoment = false
let nextRaised
const nextRaisedPromise = new Promise((resolve) => { nextRaised = resolve })
const arrivals = []
const writeEvents = []
const host = {
  session: { id: counted(async () => scenario.sessionId ?? 'a-session') },
  process: {
    run: counted(async (argv, init) => {
      const step = argv[2]
      if (setup) return { exitCode: 0, stdout: '' }
      calls.push({ step: argv.join(' '), input: JSON.parse(init.stdin), timeoutMs: init.timeoutMs })
      if (step === 'moment' && scenario.momentBarrier) {
        // the moments wait for each other: a mod that queues its chains never starts the second, and the wait ends
        started++
        await Promise.race([
          new Promise((resolve) => {
            arrivals.push(resolve)
            if (started >= scenario.momentBarrier) arrivals.forEach((go) => go())
          }),
          new Promise((resolve) => wait(resolve, 2000)),
        ])
        barrierMet = started >= scenario.momentBarrier
      }
      if (step === 'moment' && scenario.momentAfterNext) {
        // the moment does not finish until the call is let run: a mod that runs its chain first never gets there, and
        // the wait ends
        await Promise.race([nextRaisedPromise, new Promise((resolve) => wait(resolve, 2000))])
        nextBeforeMoment = nextSeen
      }
      if (step === 'write') {
        const mine = written++
        writeEvents.push('start')
        if (scenario.writeDelayMs) await new Promise((resolve) => wait(resolve, scenario.writeDelayMs))
        writeEvents.push('end')
        return scenario.writes?.[mine] ?? { exitCode: 0, stdout: '' }
      }
      if (step === scenario.stalls) {
        await new Promise((resolve) => wait(resolve, 5))
        throw new Error('the command ran too long')
      }
      if (step === 'deliver') {
        if (scenario.deliverDelayMs) await new Promise((resolve) => wait(resolve, scenario.deliverDelayMs))
        return scenario.delivers?.[delivered++] ?? scenario.deliver ?? { exitCode: 0, stdout: '' }
      }
      return scenario.steps?.[step] ?? { exitCode: 0, stdout: '' }
    }),
  },
  model: {
    complete: counted(async (asked, options) => {
      calls.push({ step: 'model', asked, signal: options?.signal !== undefined })
      const fate = scenario.modelFates?.[nth++]
      if (fate === 'empty') return { isAnswered: false, reason: 'empty-reply' }
      if (fate === 'rejects') throw new Error('the model failed')
      if (scenario.modelNeverSettles || fate === 'aborted') {
        return new Promise((resolve, reject) => {
          options.signal.addEventListener('abort', () => (scenario.modelRejectsWhenAborted
            ? reject(new Error('aborted'))
            : resolve({ isAnswered: false, reason: 'aborted' })))
        })
      }
      return { isAnswered: true, text: scenario.replies?.[replied++] ?? 'a reply' }
    }),
  },
  clock: { after: (_milliseconds, run) => { timers.push(run) } },
  ui: { log: (text) => { logged.push(text) } },
}

const submit = () => handlers['prompt.submit'](host, { text: 'hello', context: [], origin: { kind: 'composer' } },
  async (event) => event)
const prompts = scenario.events ? 0 : (scenario.prompts ?? 1)
let all = []
const answers = []
if (scenario.events) {
  // the turn of a prompt is begun, with no step run, and then the events of its tools are raised
  setup = true
  await submit()
  await handlers['turn.start'](host, { turnId: 'a-turn', text: 'hello' }, async (event) => event)
  setup = false
  // an event no hook of the mod handles is answered by the hooks beneath it
  const raise = (item) => {
    const next = async (passed) => {
      nextSeen = true
      nextRaised()
      return item.answer ?? (item.hook === 'prompt.submit' ? passed : {})
    }
    return handlers[item.hook] ? handlers[item.hook](host, item.event, next) : next()
  }
  if (scenario.concurrent) {
    answers.push(...await Promise.all(scenario.events.map(raise)))
  } else {
    for (const item of scenario.events) answers.push(await raise(item))
  }
} else if (scenario.concurrent) {
  all = await Promise.all(Array.from({ length: prompts }, submit))
} else {
  for (let index = 0; index < prompts; index++) all.push(await submit())
}
const passed = all[0]
const settle = { commands: calls.length, inFlight }

for (const run of timers.splice(0)) void run()

// the report is over when the commands have not changed for 50 ms
for (let last = -1, stable = 0, tick = 0; tick < 300 && stable < 5; tick++) {
  await new Promise((resolve) => wait(resolve, 10))
  stable = calls.length === last ? stable + 1 : 0
  last = calls.length
}

console.log(JSON.stringify({
  passed, all, calls, delays, settle, logged, writeEvents, answers, barrierMet, nextBeforeMoment,
}))
process.exit(0)
`
	// driverTimeout is how long the mod may take under the stand-in host before the test gives it up.
	driverTimeout = 20 * time.Second
	// hostDriver runs the prompt.submit hook of the module named by MOD against a host whose call for the session's ID
	// never answers and whose timers fire at once, and prints what the hook let through, the delays the timers were
	// asked for and how many commands the mod ran.
	hostDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const delays = []
globalThis.setTimeout = (callback, delay) => { delays.push(delay); callback(); return 0 }

const ran = []
const host = {
  session: { id: () => new Promise(() => {}) },
  process: { run: async (...args) => { ran.push(args); return { exitCode: 0, stdout: '' } } },
}

let passed = null
await handlers['prompt.submit'](host, { text: 'hello', context: [] }, async (event) => { passed = event })
console.log(JSON.stringify({ passed, delays, ran: ran.length }))
`
	// judgePrinted is what the judgement prints for a turn that is a learn moment: the request for the model that
	// judges it, and the carry, as one JSON object.
	judgePrinted = `{"request":{"model":"sonnet","system":"THE INSTRUCTIONS","prompt":"THE PROMPT","maxTokens":3000,` +
		`"timeoutMs":30000},"carry":{}}` + "\n"
	// momentPrinted is what the moment prints for a turn that ended: the request for the model that judges whether it
	// is a learn moment, and the carry, as one JSON object.
	momentPrinted = `{"request":{"model":"haiku","effort":"low","system":"THE MOMENT","prompt":"THE WINDOW",` +
		`"maxTokens":200,"timeoutMs":10000},"carry":{"cue":"turn-end","window":"THE WINDOW"}}` + "\n"
	// placePrinted is what the placement prints for lessons and the notes the session has seen: the request for the
	// model that places them, and the carry that names the notes, as one JSON object.
	placePrinted = `{"request":{"model":"sonnet","system":"THE ORDER","prompt":"THE LESSONS","maxTokens":1500,` +
		`"timeoutMs":15000},"carry":{"seen":["1.2026-10-17.a-note.md"]}}` + "\n"
)

// unexported variables.
var (
	modelCallsBefore = map[string][]map[string]any{
		"moment": nil,
		"judge":  {requestOfMoment()},
		"place":  {requestOfMoment(), requestOfJudgement()},
	}
)

// captureOutcome is what captureDriver printed.
type captureOutcome struct {
	Returned struct {
		Text string `json:"text"`
	} `json:"returned"`
	Held   bool  `json:"held"`
	Delays []int `json:"delays"`
	Calls  []struct {
		Step      string         `json:"step"`
		Input     map[string]any `json:"input"`
		Asked     map[string]any `json:"asked"`
		TimeoutMs float64        `json:"timeoutMs"`
	} `json:"calls"`
	Sweeps []struct {
		Step         string         `json:"step"`
		Input        map[string]any `json:"input"`
		TimeoutMs    float64        `json:"timeoutMs"`
		WhileJudging bool           `json:"whileJudging"`
	} `json:"sweeps"`
	Settle struct {
		Made     int `json:"made"`
		InFlight int `json:"inFlight"`
	} `json:"settle"`
}

// inputsOf returns the inputs of the calls that ran a step, in the order they were made.
func (o captureOutcome) inputsOf(step string) []map[string]any {
	var inputs []map[string]any

	for _, call := range o.Calls {
		if call.Step == step {
			inputs = append(inputs, call.Input)
		}
	}

	return inputs
}

// limits returns the time limit of each call the mod made, in order: that of a command, and that of the request of the
// model's call.
func (o captureOutcome) limits() []float64 {
	limits := make([]float64, 0, len(o.Calls))

	for _, call := range o.Calls {
		if call.Step == "model" {
			limit, _ := call.Asked["timeoutMs"].(float64)
			limits = append(limits, limit)
		} else {
			limits = append(limits, call.TimeoutMs)
		}
	}

	return limits
}

// modelCalls returns the requests the model was asked for, in order.
func (o captureOutcome) modelCalls() []map[string]any {
	var asked []map[string]any

	for _, call := range o.Calls {
		if call.Step == "model" {
			asked = append(asked, call.Asked)
		}
	}

	return asked
}

// reports returns the inputs of the runs of the writing, the reports of a failure, in the order they were made.
func (o captureOutcome) reports() []map[string]any {
	return o.inputsOf("engram auto write")
}

// stepsAndLimits returns each call the mod made, in order, with its time limit: "<step>: <milliseconds>".
func (o captureOutcome) stepsAndLimits() []string {
	steps := make([]string, 0, len(o.Calls))

	for index, limit := range o.limits() {
		steps = append(steps, fmt.Sprintf("%s: %.0f", o.Calls[index].Step, limit))
	}

	return steps
}

// captureScenario is how the stand-in host of captureDriver behaves: its moment, judgement and placement, its model,
// the result of each call of it when they differ, and its sweep, the origin of the prompt the turn begins with
// (composer when empty), the text of the prompt (hello when empty), the subagent that ran the turn, if one did, and a
// script of prompts raised, turns started and turns completed, in the order given, which stands in place of the one
// prompt and the one turn. A step the scenario does not give prints nothing.
type captureScenario struct {
	Moment map[string]any `json:"moment,omitempty"`
	Judge  map[string]any `json:"judge,omitempty"`
	Place  map[string]any `json:"place,omitempty"`
	Model  map[string]any `json:"model,omitempty"`
	// Models are the results of the calls of the model, in order; a call beyond them has the result of Model.
	Models []map[string]any `json:"models,omitempty"`
	Sweep  map[string]any   `json:"sweep,omitempty"`
	// SweepEnded is what the sweep run as ended exits with and prints
	SweepEnded map[string]any   `json:"sweepEnded,omitempty"`
	Origin     string           `json:"origin,omitempty"`
	Agent      string           `json:"agent,omitempty"`
	Text       string           `json:"text,omitempty"`
	Script     []map[string]any `json:"script,omitempty"`
	// Stalls names the stage that stalls: id, moment, model1, judge, model2, place, model3, write or sweep
	Stalls string `json:"stalls,omitempty"`
}

// set gives the step of the capture the result it is to have: the moment, the judgement or the placement.
func (s *captureScenario) set(step string, result map[string]any) {
	switch step {
	case "moment":
		s.Moment = result
	case "judge":
		s.Judge = result
	case "place":
		s.Place = result
	}
}

// deliveryOutcome is what deliveryDriver printed.
type deliveryOutcome struct {
	Passed struct {
		Text    string   `json:"text"`
		Context []string `json:"context"`
	} `json:"passed"`
	// All is what the hook let through for each prompt raised, in the order they were raised.
	All []struct {
		Text    string   `json:"text"`
		Context []string `json:"context"`
	} `json:"all"`
	Calls []struct {
		Step      string         `json:"step"`
		Input     map[string]any `json:"input"`
		Asked     map[string]any `json:"asked"`
		Signal    bool           `json:"signal"`
		TimeoutMs float64        `json:"timeoutMs"`
	} `json:"calls"`
	Delays []int `json:"delays"`
	Settle struct {
		Commands int `json:"commands"`
		InFlight int `json:"inFlight"`
	} `json:"settle"`
	Logged []string `json:"logged"`
	// WriteEvents is the start and the end of each run of the writing, in the order they happened.
	WriteEvents []string `json:"writeEvents"`
	// Answers is what the hook of each event of the scenario answered, in the order the events are listed.
	Answers []map[string]any `json:"answers"`
	// BarrierMet is whether the moments of the scenario's barrier were all run before any was let answer.
	BarrierMet bool `json:"barrierMet"`
	// NextBeforeMoment is whether the call was let run (next(e)) before the moment of the scenario finished.
	NextBeforeMoment bool `json:"nextBeforeMoment"`
}

// inputsOf returns the inputs of the runs of a step, in the order they were run.
func (o deliveryOutcome) inputsOf(step string) []map[string]any {
	var inputs []map[string]any

	for _, call := range o.Calls {
		if call.Step == step {
			inputs = append(inputs, call.Input)
		}
	}

	return inputs
}

// limitOf returns the time limit the first command that ran a step was given.
func (o deliveryOutcome) limitOf(step string) float64 {
	for _, call := range o.Calls {
		if call.Step == step {
			return call.TimeoutMs
		}
	}

	return 0
}

// modelCalls returns the requests the module asked the model for, in order.
func (o deliveryOutcome) modelCalls() []map[string]any {
	var asked []map[string]any

	for _, call := range o.Calls {
		if call.Step == "model" {
			asked = append(asked, call.Asked)
		}
	}

	return asked
}

// order returns the commands and the model's calls of the chain in the order they were made: the reports to the writing
// are left out.
func (o deliveryOutcome) order() []string {
	var steps []string

	for _, call := range o.Calls {
		if call.Step != "engram auto write" {
			steps = append(steps, call.Step)
		}
	}

	return steps
}

// reports returns the inputs of the runs of the writing, the reports of a failure, in the order they were run.
func (o deliveryOutcome) reports() []map[string]any {
	return o.inputsOf("engram auto write")
}

// deliveryScenario is how the stand-in host of deliveryDriver behaves: what each step of the chain exits with and
// prints, what the delivery does, the replies of the model, the step that runs until its time limit, that the model
// never settles unless its signal aborts it, and that the timers of the module fire at once.
type deliveryScenario struct {
	Deliver map[string]any `json:"deliver,omitempty"`
	// Delivers is what each run of the delivery exits with and prints, in order, in place of Deliver.
	Delivers []map[string]any `json:"delivers,omitempty"`
	// DeliverDelayMs is how long each run of the delivery takes.
	DeliverDelayMs int `json:"deliverDelayMs,omitempty"`
	// Prompts is how many prompts are raised (one when zero), one after another, or all at once when Concurrent.
	Prompts    int  `json:"prompts,omitempty"`
	Concurrent bool `json:"concurrent,omitempty"`
	// ModelRejectsWhenAborted makes the call of ModelNeverSettles reject, not settle, when its signal aborts it.
	ModelRejectsWhenAborted bool                      `json:"modelRejectsWhenAborted,omitempty"`
	Steps                   map[string]map[string]any `json:"steps,omitempty"`
	Replies                 []string                  `json:"replies,omitempty"`
	Stalls                  string                    `json:"stalls,omitempty"`
	ModelNeverSettles       bool                      `json:"modelNeverSettles,omitempty"`
	FastTimers              bool                      `json:"fastTimers,omitempty"`
	// ModelFates is what becomes of each call of the model, in order: empty (it gives no text), rejects, aborted (it
	// settles only when its signal aborts it) or, when the list is short or the entry empty, an answer.
	ModelFates []string `json:"modelFates,omitempty"`
	// Writes is what each run of the writing exits with and prints, in order, in place of an exit with 0.
	Writes []map[string]any `json:"writes,omitempty"`
	// WriteDelayMs is how long each run of the writing takes.
	WriteDelayMs int `json:"writeDelayMs,omitempty"`
	// Events are the events of tool calls raised in the turn of a prompt, instead of prompts: in order, or all at once
	// when Concurrent.
	Events []eventRaised `json:"events,omitempty"`
	// SessionID is the session's ID the host gives, a-session when empty.
	SessionID string `json:"sessionId,omitempty"`
	// MomentBarrier is how many runs of the moment wait for each other before any is let answer.
	MomentBarrier int `json:"momentBarrier,omitempty"`
	// MomentAfterNext makes the moment wait until an event has been let run, up to 2 seconds.
	MomentAfterNext bool `json:"momentAfterNext,omitempty"`
}

// eventRaised is an event the stand-in host raises: the hook, the event, and what the hooks beneath answer.
type eventRaised struct {
	Hook   string         `json:"hook"`
	Event  map[string]any `json:"event"`
	Answer map[string]any `json:"answer,omitempty"`
}

// aDeepStep is what the recall prints at deep depth, the third of a chain: the request for the model, the carry and the
// mark that a write follows the delivery.
func aDeepStep() string {
	const n = 3

	return fmt.Sprintf(`{"request":{"model":"sonnet","system":"S%d","prompt":"P%d","maxTokens":100},`+
		`"carry":{"cue":"prompt","depth":"deep","window":"W%d"},"write":true}`+"\n", n, n, n)
}

// aStep is what a step prints for a judgement, the nth of a chain: the request for the model and the carry.
func aStep(n int) string {
	return fmt.Sprintf(`{"request":{"model":"sonnet","system":"S%d","prompt":"P%d","maxTokens":100},`+
		`"carry":{"cue":"prompt","depth":"glance","window":"W%d"}}`+"\n", n, n, n)
}

// additionalContextOf is the answer of a classic event that carries the block of a note as additional context.
func additionalContextOf(name string) map[string]any {
	return map[string]any{"additionalContext": []any{strings.TrimSpace(noteBlock(name))}}
}

// batchEvent is a classic.PostToolBatch event of two calls of the session.
func batchEvent(session, first, second string) eventRaised {
	return eventRaised{Hook: "classic.PostToolBatch", Event: map[string]any{
		"session_id": session, "tool_calls": []any{
			map[string]any{
				"tool_name": "Bash", "tool_input": map[string]any{"command": "a"}, "tool_use_id": first,
				"tool_response": "ok",
			},
			map[string]any{
				"tool_name": "Read", "tool_input": map[string]any{"file_path": "b"}, "tool_use_id": second,
				"tool_response": map[string]any{"lines": 3},
			},
		},
	}}
}

// carryOfDeepStep is the carry of aDeepStep as the driver decodes it.
func carryOfDeepStep() map[string]any {
	return map[string]any{"cue": "prompt", "depth": "deep", "window": "W3"}
}

// carryOfStep is the carry of aStep as the driver decodes it.
func carryOfStep(n int) map[string]any {
	return map[string]any{"cue": "prompt", "depth": "glance", "window": fmt.Sprintf("W%d", n)}
}

// failureEvent is a classic.PostToolUseFailure event of the session for a Bash call.
func failureEvent(session, call string, interrupt bool) eventRaised {
	event := map[string]any{
		"session_id": session, "tool_name": "Bash", "tool_input": map[string]any{"command": "make ci"},
		"tool_use_id": call, "error": "exit 1",
	}

	if interrupt {
		event["is_interrupt"] = true
	}

	return eventRaised{Hook: "classic.PostToolUseFailure", Event: event}
}

// modulesOf reads every JavaScript file below a directory, by its path relative to the directory.
func modulesOf(t *testing.T, dir string) map[string]string {
	t.Helper()

	modules := map[string]string{}

	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".js") {
			return err
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		relative, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		modules[relative] = string(source)

		return nil
	})
	if walkErr != nil {
		t.Fatalf("reading the JavaScript of %s: %v", dir, walkErr)
	}

	return modules
}

// momentInput is what the moment is given for a turn of the stand-in's session, that began with a text and an origin.
func momentInput(text, origin string) map[string]any {
	return map[string]any{
		"session": "a-session", "turn": "a-turn", "text": text, "answer": "Understood.", "origin": origin,
		"cue": "turn-end",
	}
}

// momentOf is what the moment is given for an event of the turn of the prompt "hello".
func momentOf(cue string, held []any, event map[string]any) map[string]any {
	return map[string]any{
		"session": "a-session", "origin": "composer", "cue": cue, "held": held, "text": "hello", "event": event,
	}
}

// noteBlock is a block of one note, as the delivery prints it.
func noteBlock(name string) string {
	return "[engram memory] The lines below are notes.\n- " + name + " (s): x\n"
}

// printingChain is a capture whose moment, judgement and placement each print a request, and whose model answers each
// in turn, with LEARN, LESSONS and PLACEMENT.
func printingChain() captureScenario {
	return captureScenario{
		Moment: map[string]any{"exitCode": 0, "stdout": momentPrinted},
		Judge:  map[string]any{"exitCode": 0, "stdout": judgePrinted},
		Place:  map[string]any{"exitCode": 0, "stdout": placePrinted},
		Models: []map[string]any{
			{"isAnswered": true, "text": "LEARN"}, {"isAnswered": true, "text": "LESSONS"},
			{"isAnswered": true, "text": "PLACEMENT"},
		},
	}
}

// requestOfJudgement is the request of judgePrinted as the driver decodes it.
func requestOfJudgement() map[string]any {
	return map[string]any{
		"model": "sonnet", "system": "THE INSTRUCTIONS", "prompt": "THE PROMPT", "maxTokens": float64(3000),
		"timeoutMs": float64(30000),
	}
}

// requestOfMoment is the request of momentPrinted as the driver decodes it.
func requestOfMoment() map[string]any {
	return map[string]any{
		"model": "haiku", "effort": "low", "system": "THE MOMENT", "prompt": "THE WINDOW", "maxTokens": float64(200),
		"timeoutMs": float64(10000),
	}
}

// requestOfStep is the request of aStep as the driver decodes it.
func requestOfStep(n int) map[string]any {
	return map[string]any{
		"model": "sonnet", "system": fmt.Sprintf("S%d", n), "prompt": fmt.Sprintf("P%d", n), "maxTokens": float64(100),
	}
}

// runCaptureDriver runs the capture driver against the mod of the source, and reads what it printed.
func runCaptureDriver(t *testing.T, scenario captureScenario) captureOutcome {
	t.Helper()

	var outcome captureOutcome

	runDriver(t, captureDriver, scenario, &outcome)

	return outcome
}

// runDeliveryDriver runs the delivery driver against the mod of the source, and reads what it printed.
func runDeliveryDriver(t *testing.T, scenario deliveryScenario) deliveryOutcome {
	t.Helper()

	var outcome deliveryOutcome

	runDriver(t, deliveryDriver, scenario, &outcome)

	return outcome
}

// runDriver runs a driver under node against the mod of the source, with a scenario, and reads what it printed into
// the outcome.
func runDriver(t *testing.T, driver string, scenario, outcome any) {
	t.Helper()

	encoded, marshalErr := json.Marshal(scenario)
	if marshalErr != nil {
		t.Fatalf("encoding the scenario: %v", marshalErr)
	}

	node, findErr := findNode()
	if findErr != nil {
		t.Fatalf("finding node: %v", findErr)
	}

	source, readErr := os.ReadFile(filepath.Join(repositoryRoot, "agent-instructions", "mods", "engram-memory", "hooks",
		"register.js"))
	if readErr != nil {
		t.Fatalf("reading the mod: %v", readErr)
	}

	// a module under a name that node reads as one, whatever release of node it is
	module := filepath.Join(t.TempDir(), "register.mjs")

	writeErr := os.WriteFile(module, source, 0o600)
	if writeErr != nil {
		t.Fatalf("writing the mod: %v", writeErr)
	}

	ctx, cancel := context.WithTimeout(t.Context(), driverTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, node, "--input-type=module", "--eval", driver)
	command.Env = append(os.Environ(), "MOD="+module, "SCENARIO="+string(encoded))

	out, runErr := command.CombinedOutput()
	if runErr != nil {
		t.Fatalf("the mod did not run under the stand-in host (%v): %s", runErr, out)
	}

	unmarshalErr := json.Unmarshal(out, outcome)
	if unmarshalErr != nil {
		t.Fatalf("the driver printed %q: %v", out, unmarshalErr)
	}
}

// toolScenario gives a scenario the steps that print a request each, so that a chain runs to its delivery, unless it
// has steps of its own.
func toolScenario(scenario deliveryScenario) deliveryScenario {
	if scenario.Steps == nil {
		scenario.Steps = map[string]map[string]any{
			"moment": {"exitCode": 0, "stdout": aStep(1)}, "phrases": {"exitCode": 0, "stdout": aStep(2)},
			"recall": {"exitCode": 0, "stdout": aStep(3)},
		}
	}

	return scenario
}
