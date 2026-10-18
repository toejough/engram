//go:build targ && integration

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	. "github.com/onsi/gomega"
)

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
	g.Expect(outcome.Delays).To(Equal([]int{7000}), "the one timer the mod asked for, in milliseconds")
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
	g.Expect(outcome.Settle.Commands).To(Equal(1), "the commands the hook had run when it settled: the delivery")
	g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
	g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")

	reports := outcome.inputsOf("engram auto write")

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
// stand-in host whose delivery runs until its time limit and whose timers fire at once: a live session cannot be made
// to pass the bound without waiting for it.
func Test793d2a1j_Mod_ADeliveryThatDoesNotFinishWithinItsBoundIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runDeliveryDriver(t, deliveryScenario{Stalls: true})

	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Delays).To(Equal([]int{7000}), "the one timer the mod asked for, in milliseconds")
	g.Expect(outcome.Settle.Commands).To(Equal(1), "the commands the hook had run when it settled: the delivery")
	g.Expect(outcome.limitOf("engram auto deliver")).To(BeNumerically("==", 7000), "the time limit of the delivery")
	g.Expect(outcome.Logged).To(BeEmpty(), "the lines the user was shown")

	reports := outcome.inputsOf("engram auto write")

	g.Expect(reports).To(HaveLen(1), "the runs of auto write")

	if len(reports) != 1 {
		return
	}

	g.Expect(reports[0]).To(HaveKeyWithValue("session", "a-session"))
	g.Expect(reports[0]).To(HaveKeyWithValue("turn", ""))
	g.Expect(reports[0]["failure"]).To(And(ContainSubstring("the delivery for a prompt could not be done"),
		ContainSubstring("7000 ms")), "the failure")
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
// session cannot show what a stand-in records. The stand-in's judgement does not answer until the driver lets it, so a
// hook that waited for the job would not give the event's result back.
func Test793d2b1a_Mod_JudgesATurnAndGivesTheReplyToTheWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"hold": true, "exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
	})

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Held).To(BeFalse(), "the event's result was not given back while the judgement was running")
	g.Expect(outcome.Calls).To(HaveLen(3), "the judgement, the model and the writing, in that order")

	if len(outcome.Calls) != 3 {
		return
	}

	g.Expect(outcome.Calls[0].Step).To(Equal("engram auto judge"))
	g.Expect(outcome.Calls[0].Input).To(Equal(map[string]any{
		"session": "a-session", "turn": "a-turn", "text": "hello", "answer": "Understood.", "origin": "composer",
	}))
	g.Expect(outcome.Calls[1].Step).To(Equal("model"))
	g.Expect(outcome.Calls[1].Asked).To(Equal(map[string]any{
		"model": "haiku", "effort": "low", "system": "THE INSTRUCTIONS", "prompt": "THE PROMPT",
		"maxTokens": float64(1500), "timeoutMs": float64(60000),
	}), "the request, exactly as the judgement printed it")
	g.Expect(outcome.Calls[2].Step).To(Equal("engram auto write"))
	g.Expect(outcome.Calls[2].Input).To(Equal(map[string]any{
		"session": "a-session", "turn": "a-turn", "reply": "THE REPLY",
	}))
}

func Test793d2b1b_Mod_AJudgementThatPrintsNothingEndsTheCapture(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{Judge: map[string]any{"exitCode": 0, "stdout": ""}})

	g.Expect(outcome.Calls).To(HaveLen(1), "only the judgement was run")

	if len(outcome.Calls) == 1 {
		g.Expect(outcome.Calls[0].Step).To(Equal("engram auto judge"))
	}
}

func Test793d2b1c_Mod_AStepThatFailsIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	for name, judge := range map[string]map[string]any{
		"a judgement that exits with 1":                 {"exitCode": 1, "stdout": "Unknown command: auto\n"},
		"a judgement that prints what is not a request": {"exitCode": 0, "stdout": "Unknown command: auto\n"},
		"a judgement that is ended":                     {"rejects": "the command ran for 20000 ms"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			outcome := runCaptureDriver(t, captureScenario{Judge: judge})

			g.Expect(outcome.Calls).To(HaveLen(2), "the judgement, and the report to the writing")

			if len(outcome.Calls) != 2 {
				return
			}

			g.Expect(outcome.Calls[1].Step).To(Equal("engram auto write"))
			g.Expect(outcome.Calls[1].Input).To(HaveKeyWithValue("session", "a-session"))
			g.Expect(outcome.Calls[1].Input).To(HaveKeyWithValue("turn", "a-turn"))
			g.Expect(outcome.Calls[1].Input).NotTo(HaveKey("reply"))
			g.Expect(outcome.Calls[1].Input["failure"]).To(ContainSubstring("engram auto judge"))
		})
	}
}

func Test793d2b1d_Mod_AModelThatDoesNotAnswerIsReportedToTheWriting(t *testing.T) {
	t.Parallel()

	for name, model := range map[string]map[string]any{
		"an API error":                 {"isAnswered": false, "reason": "api-error", "status": 529},
		"a reply with no text":         {"isAnswered": false, "reason": "empty-reply"},
		"a call that the host rejects": {"rejects": "the cap is refused"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			outcome := runCaptureDriver(t, captureScenario{
				Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
				Model: model,
			})

			g.Expect(outcome.Calls).To(HaveLen(3), "the judgement, the model, and the report to the writing")

			if len(outcome.Calls) != 3 {
				return
			}

			g.Expect(outcome.Calls[2].Step).To(Equal("engram auto write"))
			g.Expect(outcome.Calls[2].Input).To(HaveKeyWithValue("turn", "a-turn"))
			g.Expect(outcome.Calls[2].Input).NotTo(HaveKey("reply"))
			g.Expect(outcome.Calls[2].Input["failure"]).To(ContainSubstring("model call"))
		})
	}
}

func Test793d2b1e_Mod_TheJudgementReceivesTheOriginOfThePromptTheTurnBeganWith(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge:  map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model:  map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Origin: "task-notification",
	})

	g.Expect(outcome.Calls).NotTo(BeEmpty(), "the judgement was run")

	if len(outcome.Calls) > 0 {
		g.Expect(outcome.Calls[0].Step).To(Equal("engram auto judge"))
		g.Expect(outcome.Calls[0].Input).To(Equal(map[string]any{
			"session": "a-session", "turn": "a-turn", "text": "hello", "answer": "Understood.", "origin": "task-notification",
		}))
	}
}

func Test793d2b1f_Mod_ATurnOfASubagentIsNotCaptured(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Agent: "a-subagent",
	})

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Calls).To(BeEmpty(), "neither the judgement, the model nor the writing was asked")
}

func Test793d2b1g_Mod_TheJudgementReceivesTheTextTheTurnBeganWith(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Text:  "fix the zorblax build",
	})

	g.Expect(outcome.Calls).NotTo(BeEmpty(), "the judgement was run")

	if len(outcome.Calls) > 0 {
		g.Expect(outcome.Calls[0].Step).To(Equal("engram auto judge"))
		g.Expect(outcome.Calls[0].Input).To(Equal(map[string]any{
			"session": "a-session", "turn": "a-turn", "text": "fix the zorblax build", "answer": "Understood.",
			"origin": "composer",
		}))
	}
}

func Test793d2b1h_Mod_ATurnTakesTheOriginAndTheTextOfItsOwnPromptWhenOthersAreWaiting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Script: []map[string]any{
			{"submit": map[string]any{"text": "first", "origin": "composer"}},
			{"start": map[string]any{"text": "first", "turnId": "t1"}},
			{"submit": map[string]any{"text": "second", "origin": "composer"}},
			{"submit": map[string]any{"text": "<task-notification>done</task-notification>", "origin": "task-notification"}},
			{"complete": map[string]any{"turnId": "t1", "answer": "done one"}},
			{"start": map[string]any{"text": "second", "turnId": "t2"}},
			{"complete": map[string]any{"turnId": "t2", "answer": "done two"}},
		},
	})

	judgements := outcome.inputsOf("engram auto judge")

	g.Expect(judgements).To(Equal([]map[string]any{
		{"session": "a-session", "turn": "t1", "text": "first", "answer": "done one", "origin": "composer"},
		{"session": "a-session", "turn": "t2", "text": "second", "answer": "done two", "origin": "composer"},
	}))
}

func Test793d2b1i_Mod_ATurnThatStartsWithNoTextHasNoOrigin(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Script: []map[string]any{
			{"submit": map[string]any{"text": "first", "origin": "composer"}},
			{"start": map[string]any{"text": "first", "turnId": "t1"}},
			{"complete": map[string]any{"turnId": "t1", "answer": "done one"}},
			{"start": map[string]any{"text": "", "turnId": "t2"}},
			{"complete": map[string]any{"turnId": "t2", "answer": "continued"}},
		},
	})

	judgements := outcome.inputsOf("engram auto judge")

	g.Expect(judgements).To(HaveLen(2))

	if len(judgements) == 2 {
		g.Expect(judgements[1]).To(Equal(map[string]any{
			"session": "a-session", "turn": "t2", "text": "", "answer": "continued", "origin": "",
		}))
	}
}

func Test793d2b1j_Mod_AStageThatStallsIsEndedAtItsBoundAndReported(t *testing.T) {
	t.Parallel()

	judged := map[string]any{"exitCode": 0, "stdout": requestPrinted}
	answered := map[string]any{"isAnswered": true, "text": "THE REPLY"}

	t.Run("the call for the session's ID", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{Judge: judged, Model: answered, Stalls: "id"})

		g.Expect(outcome.Delays).To(ContainElement(5000), "the mod waited 5 seconds for the ID")
		g.Expect(outcome.Calls).To(BeEmpty(), "and then did nothing: no record can be made without the ID")
	})

	for name, stage := range map[string]struct {
		stalls   string
		want     []string
		reported string
	}{
		"the judgement": {
			stalls: "judge", reported: "engram auto judge",
			want: []string{"engram auto judge: 15000", "engram auto write: 15000"},
		},
		"the model's call": {
			stalls: "model", reported: "model call",
			want: []string{"engram auto judge: 15000", "model: 60000", "engram auto write: 15000"},
		},
		"the writing": {
			stalls: "write", reported: "engram auto write",
			want: []string{
				"engram auto judge: 15000", "model: 60000", "engram auto write: 20000", "engram auto write: 15000",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			outcome := runCaptureDriver(t, captureScenario{Judge: judged, Model: answered, Stalls: stage.stalls})

			g.Expect(outcome.stepsAndLimits()).To(Equal(stage.want), "the calls the mod made, with their time limits")

			if len(outcome.Calls) > 0 {
				last := outcome.Calls[len(outcome.Calls)-1]
				g.Expect(last.Input["failure"]).To(ContainSubstring(stage.reported), "the failure names the stage")
			}
		})
	}

	t.Run("the stages together", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{Judge: judged, Model: answered, Stalls: "write"})

		// the longest chain: the ID, the judgement, the model, the writing that stalls, and the report of that
		g.Expect(outcome.Delays).To(ContainElement(5000), "the bound of the ID")
		g.Expect(outcome.limits()).To(HaveLen(4))
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

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": `{"model": oops` + "\n"},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
	})

	g.Expect(outcome.Calls).To(HaveLen(2), "the judgement, and the report to the writing: the model was never asked")

	if len(outcome.Calls) == 2 {
		g.Expect(outcome.Calls[1].Step).To(Equal("engram auto write"))
		g.Expect(outcome.Calls[1].Input["failure"]).To(ContainSubstring("engram auto judge"))
	}
}

func Test793d2b1l_Mod_TheWorkAfterATurnIsStartedFromATimerWithNoCallInFlightWhenTheEventSettles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"hold": true, "exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
	})

	g.Expect(outcome.Settle.Made).To(BeZero(), "the calls the hook of the event made")
	g.Expect(outcome.Settle.InFlight).To(BeZero(), "the calls in flight when the hook settled")
	g.Expect(outcome.inputsOf("engram auto judge")).To(HaveLen(1), "the judgement was run when the timers were let go")
	g.Expect(outcome.Sweeps).To(HaveLen(1), "the sweep was run when the timers were let go")
}

func Test793d2b1m_Mod_AFailureLongerThanItsBoundKeepsBothEnds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// a line of 1000 characters: the step at its beginning, a long path, and the cause at its end
	stderr := "Error: auto judge: reading " + strings.Repeat("a-long-directory/", 55) + "the record: permission denied\n"
	outcome := runCaptureDriver(t, captureScenario{Judge: map[string]any{"exitCode": 1, "stdout": "", "stderr": stderr}})

	reports := outcome.inputsOf("engram auto write")

	g.Expect(reports).To(HaveLen(1), "the report to the writing")

	if len(reports) != 1 {
		return
	}

	failure, isText := reports[0]["failure"].(string)

	g.Expect(isText).To(BeTrue(), "the failure is text")
	g.Expect(utf8.RuneCountInString(failure)).To(BeNumerically("<=", 300), "the failure as reported")
	g.Expect(failure).To(HavePrefix("engram auto judge exited with 1: Error: auto judge: reading a-long-directory/"),
		"the beginning of the failure, which names the step")
	g.Expect(failure).To(HaveSuffix("the record: permission denied"), "the end of the failure, which gives the cause")
}

func Test793d2c1a_Mod_SweepsTheConversationAfterATurnApartFromTheCapture(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"hold": true, "exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
	})

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Held).To(BeFalse(), "the event's result was not given back while the sweep was running")
	g.Expect(outcome.Sweeps).To(HaveLen(1), "the sweep was run once")

	if len(outcome.Sweeps) != 1 {
		return
	}

	g.Expect(outcome.Sweeps[0].Step).To(Equal("engram auto sweep"))
	g.Expect(outcome.Sweeps[0].Input).To(Equal(map[string]any{"session": "a-session", "origin": "composer"}))
	g.Expect(outcome.Sweeps[0].WhileJudging).To(BeTrue(), "the sweep did not wait for the judgement")
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

			outcome := runCaptureDriver(t, captureScenario{
				Judge: map[string]any{"exitCode": 0, "stdout": ""},
				Sweep: sweep,
			})

			reports := outcome.inputsOf("engram auto write")

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

		outcome := runCaptureDriver(t, captureScenario{
			Judge: map[string]any{"exitCode": 0, "stdout": ""}, Stalls: "id",
		})

		g.Expect(outcome.Delays).To(ContainElement(5000), "the mod waited 5 seconds for the ID")
		g.Expect(outcome.Sweeps).To(BeEmpty(), "and then did not sweep")
	})

	t.Run("the sweep", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		outcome := runCaptureDriver(t, captureScenario{
			Judge: map[string]any{"exitCode": 0, "stdout": ""}, Stalls: "sweep",
		})

		g.Expect(outcome.Sweeps).To(HaveLen(1))

		reports := outcome.stepsAndLimits()

		g.Expect(reports).To(ContainElement("engram auto write: 15000"), "the report of the failure")

		if len(outcome.Sweeps) == 1 {
			g.Expect(outcome.Sweeps[0].TimeoutMs).To(BeNumerically("==", 90000), "the bound of the sweep")
			g.Expect(5000+outcome.Sweeps[0].TimeoutMs+15000).To(BeNumerically("<", 120000),
				"the ID, the sweep and the report together, in milliseconds")
		}
	})
}

func Test793d2c1d_Mod_ATurnOfASubagentIsNotSwept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureScenario{
		Judge: map[string]any{"exitCode": 0, "stdout": requestPrinted},
		Model: map[string]any{"isAnswered": true, "text": "THE REPLY"},
		Agent: "a-subagent",
	})

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Sweeps).To(BeEmpty(), "the sweep was never run")
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
	// steps of its script are done in order. Its
	// judgement answers once the driver lets it (hold), exits with a code and prints a text, or rejects, and its model
	// answers with a result or rejects. It records the order and the input of what the module asks of it, except the
	// delivery and the sweep, and prints what the hook of the end of the turn returned, whether that hook was still
	// waiting when the driver gave up on it, the calls, the sweeps, which it records apart with whether the judgement
	// was still running when each was run, and the calls the hook made and left in flight when it settled. The mod's
	// timers run once the hook has settled.
	captureDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const scenario = JSON.parse(process.env.SCENARIO)
const calls = []
const sweeps = []
let letJudgementAnswer
let judgementReleased = false
const judgement = new Promise((resolve) => { letJudgementAnswer = resolve })
function releaseJudgement(result) {
  judgementReleased = true
  letJudgementAnswer(result)
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
// first run of the writing is the one that stalls
let writes = 0
const host = {
  session: { id: () => (scenario.stalls === 'id' ? new Promise(() => {}) : Promise.resolve('a-session')) },
  process: {
    run: async (argv, init) => {
      if (argv[2] === 'deliver') return { exitCode: 0, stdout: '' }
      if (argv[2] === 'sweep') {
        sweeps.push({ step: argv.join(' '), input: JSON.parse(init.stdin), timeoutMs: init.timeoutMs,
          whileJudging: !judgementReleased })
        if (scenario.stalls === 'sweep') throw new Error('the command ran too long')
        if (scenario.sweep?.rejects) throw new Error(scenario.sweep.rejects)
        return scenario.sweep ?? { exitCode: 0, stdout: '' }
      }
      calls.push({ step: argv.join(' '), input: JSON.parse(init.stdin), timeoutMs: init.timeoutMs })
      const ranTooLong = argv[2] === scenario.stalls && (argv[2] !== 'write' || ++writes === 1)
      if (ranTooLong) throw new Error('the command ran too long')
      if (argv[2] !== 'judge') return { exitCode: 0, stdout: '' }
      if (scenario.judge.rejects) throw new Error(scenario.judge.rejects)
      return scenario.judge.hold ? judgement : scenario.judge
    },
  },
  model: {
    complete: async (asked) => {
      calls.push({ step: 'model', asked })
      if (scenario.stalls === 'model') return { isAnswered: false, reason: 'aborted' }
      if (scenario.model.rejects) throw new Error(scenario.model.rejects)
      return scenario.model
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
  releaseJudgement({ exitCode: scenario.judge.exitCode, stdout: scenario.judge.stdout })
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

// a judgement that is held lets the sweep start while it runs
for (let tick = 0; scenario.judge.hold && tick < 50 && sweeps.length === 0; tick++) {
  await new Promise((resolve) => wait(resolve, 10))
}

releaseJudgement({ exitCode: scenario.judge.exitCode, stdout: scenario.judge.stdout })

await settled()

console.log(JSON.stringify({ returned: held ? {} : returned, held, calls, sweeps, delays, settle }))
`
	// deliveryDriver runs the prompt.submit hook of the module named by MOD against a host that behaves as SCENARIO says:
	// its delivery exits with a code and prints a text, or runs until its time limit, when the timers of the module
	// fire at once and are recorded. The host counts the commands the module has run and the calls in flight when the
	// hook settles, runs the timers of the module only then, and prints what the hook let through, the commands the
	// module ran in order with their input and time limit, the delays the timers were asked for, the count, and the lines
	// the module asked the host to log, in order.
	deliveryDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const scenario = JSON.parse(process.env.SCENARIO)
const calls = []
const delays = []
const wait = globalThis.setTimeout
if (scenario.stalls) {
  globalThis.setTimeout = (callback, delay) => { delays.push(delay); return wait(callback, 0) }
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
const host = {
  session: { id: counted(async () => 'a-session') },
  process: {
    run: counted(async (argv, init) => {
      calls.push({ step: argv.join(' '), input: JSON.parse(init.stdin), timeoutMs: init.timeoutMs })
      if (argv[2] !== 'deliver') return { exitCode: 0, stdout: '' }
      if (!scenario.stalls) return scenario.deliver
      await new Promise((resolve) => wait(resolve, 5))
      throw new Error('the command ran too long')
    }),
  },
  clock: { after: (_milliseconds, run) => { timers.push(run) } },
  ui: { log: (text) => { logged.push(text) } },
}

let passed = null
await handlers['prompt.submit'](host, { text: 'hello', context: [], origin: { kind: 'composer' } },
  async (event) => { passed = event })
const settle = { commands: calls.length, inFlight }

for (const run of timers.splice(0)) void run()

// the report is over when the commands have not changed for 50 ms
for (let last = -1, stable = 0, tick = 0; tick < 300 && stable < 5; tick++) {
  await new Promise((resolve) => wait(resolve, 10))
  stable = calls.length === last ? stable + 1 : 0
  last = calls.length
}

console.log(JSON.stringify({ passed, calls, delays, settle, logged }))
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
	// requestPrinted is what a judgement prints for a turn: the request for the model, as one JSON object.
	requestPrinted = `{"model":"haiku","effort":"low","system":"THE INSTRUCTIONS","prompt":"THE PROMPT",` +
		`"maxTokens":1500,"timeoutMs":60000}` + "\n"
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

// stepsAndLimits returns each call the mod made, in order, with its time limit: "<step>: <milliseconds>".
func (o captureOutcome) stepsAndLimits() []string {
	steps := make([]string, 0, len(o.Calls))

	for index, limit := range o.limits() {
		steps = append(steps, fmt.Sprintf("%s: %.0f", o.Calls[index].Step, limit))
	}

	return steps
}

// captureScenario is how the stand-in host of captureDriver behaves: its judgement, its model and its sweep, the origin
// of the prompt the turn begins with (composer when empty), the text of the prompt (hello when empty), the subagent
// that ran the turn, if one did, and a script of prompts raised, turns started and turns completed, in the order given,
// which stands in place of the one prompt and the one turn.
type captureScenario struct {
	Judge  map[string]any   `json:"judge"`
	Model  map[string]any   `json:"model"`
	Sweep  map[string]any   `json:"sweep,omitempty"`
	Origin string           `json:"origin,omitempty"`
	Agent  string           `json:"agent,omitempty"`
	Text   string           `json:"text,omitempty"`
	Script []map[string]any `json:"script,omitempty"`
	// Stalls names the stage that stalls: id, judge, model, write or sweep
	Stalls string `json:"stalls,omitempty"`
}

// deliveryOutcome is what deliveryDriver printed.
type deliveryOutcome struct {
	Passed struct {
		Text    string   `json:"text"`
		Context []string `json:"context"`
	} `json:"passed"`
	Calls []struct {
		Step      string         `json:"step"`
		Input     map[string]any `json:"input"`
		TimeoutMs float64        `json:"timeoutMs"`
	} `json:"calls"`
	Delays []int `json:"delays"`
	Settle struct {
		Commands int `json:"commands"`
		InFlight int `json:"inFlight"`
	} `json:"settle"`
	Logged []string `json:"logged"`
}

// inputsOf returns the inputs of the commands that ran a step, in the order they were run.
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

// deliveryScenario is how the stand-in host of deliveryDriver behaves: what the delivery exits with and prints, or that
// it runs until its time limit.
type deliveryScenario struct {
	Deliver map[string]any `json:"deliver,omitempty"`
	Stalls  bool           `json:"stalls,omitempty"`
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
